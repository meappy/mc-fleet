// Package reconcile applies a spec to a machine's mods folders: resolves each
// mod via Modrinth/CurseForge/URL, downloads any out-of-date jars, drops the
// `.<slug>.version` sentinel, and rewrites servers.dat / launcher_profiles.json.
package reconcile

import (
	"fmt"

	"github.com/meappy/mc-fleet/internal/launcher"
	"github.com/meappy/mc-fleet/internal/spec"
)

// Opts tunes a Run.
type Opts struct {
	DryRun bool
	Clean  bool // remove jars on disk that aren't in the spec
}

// Run iterates each spec.servers[*] and spec.client_profiles[*], reconciles
// the corresponding per-profile mods folder on `m`. Real impl in commit 4.
func Run(s *spec.Spec, m launcher.Machine, _ Opts) error {
	return fmt.Errorf("reconcile.Run: not implemented yet (commit 4)")
}
