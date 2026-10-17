// Package reconcile applies a spec to a machine's per-profile mods folders.
package reconcile

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/meappy/mc-fleet/internal/launcher"
	"github.com/meappy/mc-fleet/internal/notify"
	"github.com/meappy/mc-fleet/internal/resolve"
	"github.com/meappy/mc-fleet/internal/spec"
)

// Opts tunes a Run.
type Opts struct {
	DryRun bool
	Clean  bool // remove jars on disk that aren't in the spec
}

// Run iterates each spec.servers + client_profiles entry and reconciles its
// per-profile mods folder on the local machine.
func Run(s *spec.Spec, m launcher.Machine, opts Opts) error {
	if s == nil {
		return fmt.Errorf("nil spec")
	}
	// Iterate servers (sorted by key for deterministic output).
	for key, srv := range s.Servers {
		profile := ProfileNameForServer(key, srv)
		if err := reconcileProfile(m, profile, srv.MinecraftVersion, srv.Loader,
			clientMods(srv.Mods), opts); err != nil {
			return fmt.Errorf("server %q: %w", key, err)
		}
	}
	for _, cp := range s.ClientProfiles {
		if cp.Loader == "vanilla" {
			continue // nothing to install
		}
		if err := reconcileProfile(m, cp.Name, cp.MinecraftVersion, cp.Loader,
			clientMods(cp.Mods), opts); err != nil {
			return fmt.Errorf("profile %q: %w", cp.Name, err)
		}
	}
	return nil
}

// clientMods filters out side="server" entries (server-only mods we don't
// install on a client). side="both" and side="client" pass through.
func clientMods(in []spec.Mod) []spec.Mod {
	out := make([]spec.Mod, 0, len(in))
	for _, m := range in {
		if m.EffectiveSide() == "server" {
			continue
		}
		out = append(out, m)
	}
	return out
}

// ProfileNameForServer mirrors the Python tool's naming so launcher profiles
// stay continuous across the migration.
func ProfileNameForServer(key string, s spec.Server) string {
	switch key {
	case "dragons":
		return s.MinecraftVersion + "-dragons"
	case "survival":
		return s.MinecraftVersion + "-" + s.Loader
	}
	return s.MinecraftVersion + "-" + key
}

func reconcileProfile(m launcher.Machine, profile, mcVersion, loader string,
	mods []spec.Mod, opts Opts) error {

	notify.Progress(fmt.Sprintf("profile %s (%s / %s)", profile, mcVersion, loader))

	// Pass 1: resolve every top-level entry without walking deps so explicit
	// user pins always beat transitive resolutions. Matches the Python tool's
	// two-pass fix (commit 7d559b0).
	resolved := []*resolve.ModInfo{}
	seen := map[string]bool{}
	for _, mod := range mods {
		info, err := resolve.One(mod, mcVersion, loader)
		if err != nil {
			if isSkippable(mod, err) {
				notify.Progress(fmt.Sprintf("  skip %s — %v", displayName(mod), err))
				continue
			}
			return fmt.Errorf("resolve %q: %w", displayName(mod), err)
		}
		if seen[info.Key()] {
			continue
		}
		if err := guardClientSide(info); err != nil {
			return err
		}
		resolved = append(resolved, info)
		seen[info.Key()] = true
	}

	// Pass 2: walk transitive deps. Skip anything pass 1 already covered.
	for _, mod := range mods {
		if mod.EffectiveSource() == "url" || mod.EffectiveSource() == "curseforge" {
			continue
		}
		chain, err := resolve.WithDeps(mod, mcVersion, loader)
		if err != nil {
			return fmt.Errorf("deps for %q: %w", displayName(mod), err)
		}
		for _, info := range chain {
			if seen[info.Key()] {
				continue
			}
			if err := guardClientSide(info); err != nil {
				return err
			}
			resolved = append(resolved, info)
			seen[info.Key()] = true
		}
	}

	// Diff against disk + apply
	modsDir := m.ProfileModsDir(profile)
	if err := os.MkdirAll(modsDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", modsDir, err)
	}

	for _, info := range resolved {
		jar := filepath.Join(modsDir, info.Slug+".jar")
		sentinel := filepath.Join(modsDir, "."+info.Slug+".version")
		current := readVersion(sentinel)
		if current == info.VersionNumber && fileExists(jar) {
			continue // up to date
		}
		if opts.DryRun {
			notify.Progress(fmt.Sprintf("  would install %s %s", info.Slug, info.VersionNumber))
			continue
		}
		notify.Progress(fmt.Sprintf("  downloading %s %s", info.Slug, info.VersionNumber))
		if err := downloadVerifyWrite(info.DownloadURL, info.SHA512, jar); err != nil {
			return fmt.Errorf("download %s: %w", info.Slug, err)
		}
		if err := os.WriteFile(sentinel, []byte(info.VersionNumber), 0o644); err != nil {
			return fmt.Errorf("write sentinel %s: %w", sentinel, err)
		}
	}

	if opts.Clean {
		if err := pruneUnknown(modsDir, resolved, opts.DryRun); err != nil {
			return err
		}
	}
	return nil
}

// pruneUnknown deletes jars in modsDir that the spec no longer lists, along
// with their version sentinels and any sentinel orphaned by a jar that has
// already gone.
//
// This is the other half of "diff against disk + apply" and it was missing:
// Opts.Clean existed and was documented, but nothing ever read it, so a mod
// dropped from the spec stayed on every Mac for ever. That is not cosmetic.
// xaero-zoomout was removed from the spec *because it crash-loops the client*;
// because nothing deleted it, it sat on a Mac for 11 days and then red-screened
// it — its mixin no longer matches Xaero's Minimap 26.5.0, so the minimap
// cannot construct and mod loading aborts. Stale clockwork_mod + knight-lib
// jars locked a player out of the dragons server entirely ("Channel
// [clockwork:main] missing on the server side, but required on the client!"),
// and a renamed Modrinth slug leaves both halves of the same mod installed at
// once.
//
// Only *.jar and .*.version entries are considered. Anything else in the
// folder — subdirectories, shaderpacks, stray notes — is left untouched, and a
// jar is only ever removed when the resolve step above succeeded for every
// mod, so a failed lookup can never be mistaken for "no longer in the spec".
func pruneUnknown(modsDir string, resolved []*resolve.ModInfo, dryRun bool) error {
	keepJar := make(map[string]bool, len(resolved))
	keepSentinel := make(map[string]bool, len(resolved))
	for _, info := range resolved {
		keepJar[info.Slug+".jar"] = true
		keepSentinel["."+info.Slug+".version"] = true
	}

	entries, err := os.ReadDir(modsDir)
	if err != nil {
		return fmt.Errorf("read %s: %w", modsDir, err)
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		isJar := filepath.Ext(name) == ".jar"
		isSentinel := strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".version")
		if !isJar && !isSentinel {
			continue
		}
		if isJar && keepJar[name] {
			continue
		}
		if isSentinel && keepSentinel[name] {
			continue
		}
		if dryRun {
			notify.Progress(fmt.Sprintf("  would remove %s", name))
			continue
		}
		if isJar {
			notify.Progress(fmt.Sprintf("  removing %s (not in spec)", name))
		}
		if err := os.Remove(filepath.Join(modsDir, name)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", name, err)
		}
	}
	return nil
}

// isSkippable returns true for soft errors we should warn about and continue.
// Currently: CurseForge entries (need API key, public tool doesn't carry one).
func isSkippable(m spec.Mod, _ error) bool {
	return m.EffectiveSource() == "curseforge"
}

// guardClientSide refuses to install a server-only mod on a client.
func guardClientSide(info *resolve.ModInfo) error {
	if info.ClientSide == "unsupported" {
		return fmt.Errorf("mod %q: Modrinth marks client_side=unsupported (server-only); don't list it as a client mod", info.Slug)
	}
	return nil
}

func displayName(m spec.Mod) string {
	if m.Slug != "" {
		return m.Slug
	}
	if m.URL != "" {
		return m.URL
	}
	return fmt.Sprintf("source=%s id=%d", m.Source, m.ID)
}

func readVersion(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

var dlClient = &http.Client{Timeout: 5 * time.Minute}

// downloadVerifyWrite fetches `url` and writes it to `dest` atomically (via
// a `.partial` temp file). If `expectedSha512` is non-empty, the download's
// SHA-512 must match before the temp file is renamed into place.
func downloadVerifyWrite(url, expectedSha512, dest string) error {
	tmp := dest + ".partial"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "mc-fleet/1")
	resp, err := dlClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	h := sha512.New()
	if _, err := io.Copy(io.MultiWriter(f, h), resp.Body); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if expectedSha512 != "" {
		got := hex.EncodeToString(h.Sum(nil))
		if got != expectedSha512 {
			os.Remove(tmp)
			return fmt.Errorf("SHA-512 mismatch: got %s want %s", got, expectedSha512)
		}
	}
	return os.Rename(tmp, dest)
}
