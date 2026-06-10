package spec

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUnmarshalMod_BareString(t *testing.T) {
	cases := []struct {
		in   string
		want Mod
	}{
		{`"dmr"`, Mod{Slug: "dmr"}},
		{`"jei:beta"`, Mod{Slug: "jei", Channel: "beta"}},
		{`"cc-tweaked:alpha"`, Mod{Slug: "cc-tweaked", Channel: "alpha"}},
		{`"sophisticated-backpacks:1.21.1-3.25.53.1822"`, Mod{
			Slug:    "sophisticated-backpacks",
			Version: "1.21.1-3.25.53.1822",
		}},
	}
	for _, c := range cases {
		var got Mod
		if err := json.Unmarshal([]byte(c.in), &got); err != nil {
			t.Fatalf("Unmarshal(%s): %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("Unmarshal(%s) = %+v, want %+v", c.in, got, c.want)
		}
	}
}

func TestUnmarshalMod_ObjectForm(t *testing.T) {
	in := `{"slug":"mouse-tweaks","side":"client"}`
	want := Mod{Slug: "mouse-tweaks", Side: "client"}
	var got Mod
	if err := json.Unmarshal([]byte(in), &got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestUnmarshalMod_UrlForm(t *testing.T) {
	in := `{"slug":"nfm-patched","url":"https://minecraft.meappy.me/nfm-patched-v4.jar"}`
	var got Mod
	if err := json.Unmarshal([]byte(in), &got); err != nil {
		t.Fatal(err)
	}
	if got.URL != "https://minecraft.meappy.me/nfm-patched-v4.jar" {
		t.Errorf("URL not parsed: %+v", got)
	}
	if got.EffectiveSource() != "url" {
		t.Errorf("EffectiveSource() = %q, want url", got.EffectiveSource())
	}
}

func TestUnmarshalMod_CurseforgeForm(t *testing.T) {
	in := `{"source":"curseforge","id":897116,"filename":"refurbished-furniture.jar"}`
	var got Mod
	if err := json.Unmarshal([]byte(in), &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != 897116 || got.Source != "curseforge" || got.Filename != "refurbished-furniture.jar" {
		t.Errorf("got %+v", got)
	}
}

func TestEffectiveSide(t *testing.T) {
	if (Mod{}).EffectiveSide() != "both" {
		t.Error("default side should be both")
	}
	if (Mod{Side: "client"}).EffectiveSide() != "client" {
		t.Error("explicit side should pass through")
	}
}

func TestLoad_RoundTripsLiveSpec(t *testing.T) {
	// Use the in-repo spec as the test fixture (deterministic + no network).
	s, err := Load("", "../../../fleet/fleet-spec.json")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if s.SchemaVersion != 1 {
		t.Fatalf("schema_version = %d, want 1", s.SchemaVersion)
	}
	dragons, ok := s.Servers["dragons"]
	if !ok {
		t.Fatalf("servers.dragons missing")
	}
	if dragons.MinecraftVersion != "1.21.1" {
		t.Errorf("dragons.mc_version = %q", dragons.MinecraftVersion)
	}
	if dragons.Loader != "neoforge" {
		t.Errorf("dragons.loader = %q", dragons.Loader)
	}
	if len(dragons.Mods) < 10 {
		t.Errorf("dragons.mods len = %d (expected at least 10)", len(dragons.Mods))
	}
	// Spot-check a few entries
	wantSlugs := []string{"dmr", "explorers-compass"}
	for _, slug := range wantSlugs {
		found := false
		for _, m := range dragons.Mods {
			if m.Slug == slug {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("dragons.mods missing %q (have %s)", slug, joinSlugs(dragons.Mods))
		}
	}
	// Mouse-tweaks should parse as side=client
	for _, m := range dragons.Mods {
		if m.Slug == "mouse-tweaks" && m.EffectiveSide() != "client" {
			t.Errorf("mouse-tweaks side = %q, want client", m.EffectiveSide())
		}
	}
}

func joinSlugs(mods []Mod) string {
	out := make([]string, 0, len(mods))
	for _, m := range mods {
		if m.Slug != "" {
			out = append(out, m.Slug)
		}
	}
	return strings.Join(out, ", ")
}
