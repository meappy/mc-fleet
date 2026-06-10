// Command meappy-mc is a thin, brand-locked wrapper around mc-fleet that bakes
// in the Meappy spec URL and runs `apply` (sync + launch) with no arguments
// required. It exists so users can double-click a single artefact and have
// Minecraft come up with the right mods.
//
// Power users can still run the underlying tool directly: `mc-fleet sync`,
// `mc-fleet apply --remote https://other.example/spec.json`, etc.
package main

import (
	"fmt"
	"os"

	"github.com/meappy/mc-fleet/internal/launcher"
	"github.com/meappy/mc-fleet/internal/notify"
	"github.com/meappy/mc-fleet/internal/reconcile"
	"github.com/meappy/mc-fleet/internal/spec"
)

const meappyRemote = "https://minecraft.meappy.me/fleet-spec.json"

var Version = "dev"

func main() {
	notify.Start("Meappy Minecraft", "syncing mods…")

	s, err := spec.Load(meappyRemote, "")
	if err != nil {
		notify.Error(fmt.Sprintf("failed to fetch spec: %v", err))
		os.Exit(1)
	}

	m := launcher.LocalMachine("")
	if err := reconcile.Run(s, m, reconcile.Opts{}); err != nil {
		notify.Error(fmt.Sprintf("sync failed: %v", err))
		os.Exit(1)
	}

	notify.Progress("launching Minecraft…")
	if err := launcher.OpenMinecraft(); err != nil {
		notify.Error(fmt.Sprintf("failed to launch: %v", err))
		os.Exit(1)
	}
	notify.Done("Meappy Minecraft ready")
}
