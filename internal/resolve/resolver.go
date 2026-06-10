package resolve

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/meappy/mc-fleet/internal/spec"
)

// One resolves a single spec.Mod (without walking deps). URL entries return
// synthetic ModInfo whose Key uses the URL itself (so changing the URL is the
// re-download sentinel, matching the Python tool).
func One(m spec.Mod, mcVersion, loader string) (*ModInfo, error) {
	src := m.EffectiveSource()
	switch src {
	case "url":
		if m.URL == "" {
			return nil, fmt.Errorf("url mod %q: missing url", m.Slug)
		}
		slug := m.Slug
		if slug == "" {
			slug = filenameStem(m.URL)
		}
		return &ModInfo{
			Slug:          slug,
			ProjectID:     "url:" + m.URL,
			VersionNumber: m.URL, // sentinel = the URL itself
			DownloadURL:   m.URL,
			ClientSide:    "unknown",
			ServerSide:    "unknown",
			Source:        "url",
			Filename:      m.Filename,
		}, nil
	case "curseforge":
		// Public-tool path: CurseForge needs an API key. Surface a clear error
		// so the user knows what they're missing.
		return nil, fmt.Errorf("mod %q: CurseForge source needs an API key (not supported in mc-fleet yet — use a direct URL or modrinth instead)", m.Slug)
	default:
		if m.Slug == "" {
			return nil, fmt.Errorf("modrinth mod: missing slug")
		}
		return Modrinth(m.Slug, mcVersion, loader, m.Version, m.Channel)
	}
}

// WithDeps walks required transitive deps and returns parent-first BFS order,
// dedup'd by Source:ProjectID. URL entries have no transitive deps.
func WithDeps(m spec.Mod, mcVersion, loader string) ([]*ModInfo, error) {
	out := []*ModInfo{}
	seen := map[string]bool{}
	root, err := One(m, mcVersion, loader)
	if err != nil {
		return nil, err
	}
	out = append(out, root)
	seen[root.Key()] = true

	if root.Source == "url" {
		return out, nil
	}

	queue := append([]ModDep{}, root.Dependencies...)
	for len(queue) > 0 {
		dep := queue[0]
		queue = queue[1:]
		key := "modrinth:" + dep.ProjectID
		if seen[key] {
			continue
		}
		seen[key] = true
		info, err := ModrinthByProjectID(dep.ProjectID, mcVersion, loader)
		if err != nil {
			// Soft-fail on a single missing dep — surface a warning but keep going.
			return nil, fmt.Errorf("resolving transitive dep %q of %q: %w", dep.ProjectID, root.Slug, err)
		}
		out = append(out, info)
		queue = append(queue, info.Dependencies...)
	}
	return out, nil
}

// filenameStem strips path + extension from a URL, returning a slug-like name.
func filenameStem(u string) string {
	base := filepath.Base(u)
	if i := strings.Index(base, "?"); i > 0 {
		base = base[:i]
	}
	if i := strings.LastIndex(base, "."); i > 0 {
		base = base[:i]
	}
	return base
}
