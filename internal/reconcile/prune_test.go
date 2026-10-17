package reconcile

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/meappy/mc-fleet/internal/resolve"
)

// seedModsDir builds a mods folder resembling a real one that has drifted:
// two mods still in the spec, several jars that are not, the sentinels that
// belong to them, and some files prune must never touch.
func seedModsDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := []string{
		"dmr.jar", ".dmr.version",
		"jei.jar", ".jei.version",
		// dropped from the spec — these are what prune exists for
		"xaero-zoomout.jar", ".xaero-zoomout.version",
		"clockwork_mod.jar", ".clockwork_mod.version",
		"ars-nouveau.jar",
		// both halves of a renamed slug
		"alexs-mobs(1.21.1).jar",
		// a sentinel orphaned by a jar deleted by hand earlier
		".knight-lib.version",
		// must survive: not a jar, not a sentinel
		"options.txt",
		"notes.md",
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "shaderpacks"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func remaining(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range entries {
		got = append(got, e.Name())
	}
	sort.Strings(got)
	return got
}

func resolvedFixture() []*resolve.ModInfo {
	return []*resolve.ModInfo{{Slug: "dmr"}, {Slug: "jei"}}
}

func TestPruneUnknown_RemovesOnlyStaleJarsAndSentinels(t *testing.T) {
	dir := seedModsDir(t)

	if err := pruneUnknown(dir, resolvedFixture(), false); err != nil {
		t.Fatalf("pruneUnknown: %v", err)
	}

	want := []string{
		".dmr.version", ".jei.version",
		"dmr.jar", "jei.jar",
		"notes.md", "options.txt", "shaderpacks",
	}
	got := remaining(t, dir)
	if len(got) != len(want) {
		t.Fatalf("remaining = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("remaining = %v, want %v", got, want)
		}
	}
}

// A dry run must report without touching anything — the flag is the only
// safety net between a bad spec fetch and a wiped mods folder.
func TestPruneUnknown_DryRunDeletesNothing(t *testing.T) {
	dir := seedModsDir(t)
	before := remaining(t, dir)

	if err := pruneUnknown(dir, resolvedFixture(), true); err != nil {
		t.Fatalf("pruneUnknown: %v", err)
	}

	after := remaining(t, dir)
	if len(before) != len(after) {
		t.Fatalf("dry run changed the folder: %v -> %v", before, after)
	}
	for i := range before {
		if before[i] != after[i] {
			t.Fatalf("dry run changed the folder: %v -> %v", before, after)
		}
	}
}

// Every jar being stale is legitimate (an emptied profile), but it is also
// what a mis-parsed spec looks like, so pin the behaviour deliberately.
func TestPruneUnknown_EmptyResolvedClearsAllJars(t *testing.T) {
	dir := seedModsDir(t)

	if err := pruneUnknown(dir, nil, false); err != nil {
		t.Fatalf("pruneUnknown: %v", err)
	}

	for _, name := range remaining(t, dir) {
		if filepath.Ext(name) == ".jar" {
			t.Fatalf("jar %s survived an empty spec", name)
		}
	}
}
