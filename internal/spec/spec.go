// Package spec parses fleet-spec.json (the public mod-list contract).
//
// Wire format mirrors the Python reference at tools/fleet/fleet-spec.json.
// Mod entries accept both bare-string and object forms — see Mod.UnmarshalJSON
// for the parsing rules.
package spec

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Spec is the top-level fleet-spec.json shape.
type Spec struct {
	SchemaVersion  int               `json:"schema_version"`
	Servers        map[string]Server `json:"servers"`
	ClientProfiles []ClientProfile   `json:"client_profiles,omitempty"`
}

// Server is one named multiplayer server + its mod list.
type Server struct {
	MinecraftVersion string `json:"minecraft_version"`
	Loader           string `json:"loader"`
	Address          string `json:"address,omitempty"`
	Icon             string `json:"icon,omitempty"`
	Mods             []Mod  `json:"mods"`
}

// ClientProfile is a non-server-attached profile (vanilla / Fabric utility).
type ClientProfile struct {
	Name             string `json:"name"`
	MinecraftVersion string `json:"minecraft_version"`
	Loader           string `json:"loader"`
	LoaderVersion    string `json:"loader_version,omitempty"`
	Icon             string `json:"icon,omitempty"`
	Mods             []Mod  `json:"mods,omitempty"`
}

// Mod is one entry in `mods`. JSON forms accepted:
//
//   - Bare string: "dmr" / "jei:beta" / "soph-backpacks:1.21.1-3.25.53.1822"
//   - Object: {"slug":"x", "url":"...", "id":123, "source":"curseforge",
//     "side":"client|server|both", "channel":"release|beta|alpha",
//     "version":"...", "filename":"..."}
type Mod struct {
	Slug     string `json:"slug,omitempty"`
	Channel  string `json:"channel,omitempty"`
	Version  string `json:"version,omitempty"`
	Source   string `json:"source,omitempty"`
	ID       int    `json:"id,omitempty"`
	URL      string `json:"url,omitempty"`
	Filename string `json:"filename,omitempty"`
	Side     string `json:"side,omitempty"` // "both" (default) | "client" | "server"
}

var channelTokens = map[string]bool{"release": true, "beta": true, "alpha": true}

// UnmarshalJSON accepts both string and object forms.
func (m *Mod) UnmarshalJSON(b []byte) error {
	// String form?
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		slug, rest, hasRest := strings.Cut(s, ":")
		m.Slug = slug
		if hasRest && rest != "" {
			if channelTokens[rest] {
				m.Channel = rest
			} else {
				m.Version = rest
			}
		}
		return nil
	}
	// Object form — alias avoids infinite recursion into UnmarshalJSON.
	type rawMod Mod
	var raw rawMod
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*m = Mod(raw)
	return nil
}

// effective side, normalising the empty/zero default to "both".
func (m Mod) EffectiveSide() string {
	if m.Side == "" {
		return "both"
	}
	return m.Side
}

// EffectiveSource normalises empty → "modrinth", auto-promotes to "url" if URL is set,
// keeps "curseforge" explicit when caller set it.
func (m Mod) EffectiveSource() string {
	if m.Source != "" {
		return m.Source
	}
	if m.URL != "" {
		return "url"
	}
	return "modrinth"
}

// Load fetches a Spec from `remote` URL or, if `local` is non-empty, that path.
// Local wins when both are set (so callers can override the public mirror).
func Load(remote, local string) (*Spec, error) {
	var body []byte
	var err error
	switch {
	case local != "":
		body, err = os.ReadFile(local)
		if err != nil {
			return nil, fmt.Errorf("read spec %s: %w", local, err)
		}
	case remote != "":
		body, err = httpGet(remote)
		if err != nil {
			return nil, fmt.Errorf("fetch spec %s: %w", remote, err)
		}
	default:
		return nil, fmt.Errorf("spec.Load: pass --remote URL or --spec PATH")
	}
	var s Spec
	if err := json.Unmarshal(body, &s); err != nil {
		return nil, fmt.Errorf("parse spec: %w", err)
	}
	if s.SchemaVersion != 0 && s.SchemaVersion != 1 {
		return nil, fmt.Errorf("unsupported schema_version %d (want 1)", s.SchemaVersion)
	}
	return &s, nil
}

func httpGet(url string) ([]byte, error) {
	c := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "mc-fleet/1 (https://github.com/meappy/mc-fleet)")
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
