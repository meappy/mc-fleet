// Package resolve turns a spec.Mod into a concrete ModInfo (URL + checksum
// + transitive deps). Mirrors the Python tool's resolver semantics.
package resolve

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const modrinthBase = "https://api.modrinth.com/v2"

// ModInfo is a fully-resolved mod ready for download + install.
type ModInfo struct {
	Slug          string
	ProjectID     string // dedup key together with Source
	VersionNumber string // sentinel value
	DownloadURL   string
	SHA512        string // empty when source=url (we can't verify those)
	SHA1          string
	ClientSide    string // "required" | "optional" | "unsupported" | "unknown"
	ServerSide    string
	Dependencies  []ModDep // required deps only, by project_id
	Source        string   // "modrinth" | "curseforge" | "url"
	Filename      string   // optional override
}

// Key returns the dedup identity for a ModInfo.
func (m ModInfo) Key() string { return m.Source + ":" + m.ProjectID }

// ModDep is a transitive dep reference.
type ModDep struct {
	ProjectID string
	Type      string // "required" | "optional" | "incompatible" | "embedded"
}

type modrinthProject struct {
	ID         string `json:"id"`
	Slug       string `json:"slug"`
	ClientSide string `json:"client_side"`
	ServerSide string `json:"server_side"`
}

type modrinthVersion struct {
	VersionNumber string `json:"version_number"`
	VersionType   string `json:"version_type"`
	Files         []struct {
		URL     string            `json:"url"`
		Primary bool              `json:"primary"`
		Hashes  map[string]string `json:"hashes"`
	} `json:"files"`
	Dependencies []struct {
		ProjectID    string `json:"project_id"`
		DependencyType string `json:"dependency_type"`
	} `json:"dependencies"`
}

// httpClient is shared so connections are reused. 20s is generous for
// Modrinth which is normally <300ms.
var httpClient = &http.Client{Timeout: 20 * time.Second}

func httpGetJSON(reqURL string, out any) error {
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "mc-fleet/1 (https://github.com/meappy/mc-fleet)")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("HTTP 404 (slug or project missing)")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// Modrinth resolves a slug to a concrete ModInfo.
//
// If `pinned` is non-empty, requires an exact version_number match.
// If `channel` is non-empty (release|beta|alpha), restricts to that channel.
// Otherwise returns Modrinth's "latest matching" entry.
func Modrinth(slug, mcVersion, loader, pinned, channel string) (*ModInfo, error) {
	var proj modrinthProject
	if err := httpGetJSON(modrinthBase+"/project/"+url.PathEscape(slug), &proj); err != nil {
		return nil, fmt.Errorf("modrinth %q: %w", slug, err)
	}
	q := url.Values{}
	q.Set("loaders", fmt.Sprintf(`["%s"]`, loader))
	q.Set("game_versions", fmt.Sprintf(`["%s"]`, mcVersion))
	var versions []modrinthVersion
	if err := httpGetJSON(modrinthBase+"/project/"+url.PathEscape(slug)+"/version?"+q.Encode(), &versions); err != nil {
		return nil, fmt.Errorf("modrinth %q versions: %w", slug, err)
	}
	if len(versions) == 0 {
		return nil, fmt.Errorf("modrinth %q: no %s build for MC %s", slug, loader, mcVersion)
	}

	var match *modrinthVersion
	switch {
	case pinned != "":
		for i := range versions {
			if versions[i].VersionNumber == pinned {
				match = &versions[i]
				break
			}
		}
		if match == nil {
			// Build a short "available" hint
			n := len(versions)
			if n > 5 {
				n = 5
			}
			recent := make([]string, n)
			for i := 0; i < n; i++ {
				recent[i] = versions[i].VersionNumber
			}
			return nil, fmt.Errorf("modrinth %q: pinned version %q not found (recent: %v)", slug, pinned, recent)
		}
	case channel != "":
		for i := range versions {
			if versions[i].VersionType == channel {
				match = &versions[i]
				break
			}
		}
		if match == nil {
			return nil, fmt.Errorf("modrinth %q: no %q-channel build for MC %s + %s", slug, channel, mcVersion, loader)
		}
	default:
		match = &versions[0]
	}

	// Prefer the primary file; fall back to the first if no primary flagged.
	var fileURL, sha512, sha1 string
	for i := range match.Files {
		f := match.Files[i]
		if f.Primary || (i == 0 && fileURL == "") {
			fileURL = f.URL
			sha512 = f.Hashes["sha512"]
			sha1 = f.Hashes["sha1"]
			if f.Primary {
				break
			}
		}
	}
	if fileURL == "" {
		return nil, fmt.Errorf("modrinth %q: no downloadable file in version %q", slug, match.VersionNumber)
	}

	deps := make([]ModDep, 0, len(match.Dependencies))
	for _, d := range match.Dependencies {
		if d.DependencyType == "required" && d.ProjectID != "" {
			deps = append(deps, ModDep{ProjectID: d.ProjectID, Type: d.DependencyType})
		}
	}

	return &ModInfo{
		Slug:          proj.Slug,
		ProjectID:     proj.ID,
		VersionNumber: match.VersionNumber,
		DownloadURL:   fileURL,
		SHA512:        sha512,
		SHA1:          sha1,
		ClientSide:    proj.ClientSide,
		ServerSide:    proj.ServerSide,
		Dependencies:  deps,
		Source:        "modrinth",
	}, nil
}

// ModrinthByProjectID is a thin lookup used during transitive dep resolution,
// when we only have a project_id and need to walk deps.
func ModrinthByProjectID(projectID, mcVersion, loader string) (*ModInfo, error) {
	var proj modrinthProject
	if err := httpGetJSON(modrinthBase+"/project/"+url.PathEscape(projectID), &proj); err != nil {
		return nil, fmt.Errorf("modrinth dep %q: %w", projectID, err)
	}
	return Modrinth(proj.Slug, mcVersion, loader, "", "")
}
