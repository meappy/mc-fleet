// Package spec parses fleet-spec.json (the public mod-list contract).
//
// Wire format is the same as the Python reference at
// tools/fleet/fleet-spec.json — see ../../../../README.md for the schema.
package spec

import "fmt"

// Spec is the top-level fleet-spec.json shape.
type Spec struct {
	SchemaVersion int                     `json:"schema_version"`
	Servers       map[string]Server       `json:"servers"`
	ClientProfiles []ClientProfile        `json:"client_profiles,omitempty"`
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

// Mod is one entry in `mods`. Accepts both bare-string and object forms via
// custom UnmarshalJSON — see commit 3 for the real implementation.
type Mod struct {
	Slug     string `json:"slug,omitempty"`
	Channel  string `json:"channel,omitempty"`
	Version  string `json:"version,omitempty"`
	Source   string `json:"source,omitempty"` // "modrinth" | "curseforge" | "url"
	ID       int    `json:"id,omitempty"`     // CurseForge numeric ID
	URL      string `json:"url,omitempty"`
	Filename string `json:"filename,omitempty"`
	Side     string `json:"side,omitempty"` // "both" | "client" | "server"
}

// Load fetches a spec from `remote` URL (or `local` path if non-empty). The
// real implementation arrives in commit 3.
func Load(remote, local string) (*Spec, error) {
	return nil, fmt.Errorf("spec.Load: not implemented yet (commit 3 lands HTTP fetch + parse)")
}
