// Command mc-fleet syncs a Minecraft client's mods folder against a published
// fleet-spec.json and (optionally) launches Minecraft. Cross-platform: macOS,
// Windows, Linux. The same binary powers both the public single-machine
// "sync + launch" flow and the admin "apply across the fleet via SSH" flow —
// admin features only activate when a machines config is present.
package main

import (
	"fmt"
	"os"

	"github.com/meappy/mc-fleet/internal/launcher"
	"github.com/meappy/mc-fleet/internal/reconcile"
	"github.com/meappy/mc-fleet/internal/spec"
	"github.com/spf13/cobra"
)

// Version is set by goreleaser at build time via -ldflags.
var Version = "dev"

// Default public spec URL. Override with --spec or --remote.
const defaultRemote = "https://minecraft.meappy.me/fleet-spec.json"

func main() {
	root := &cobra.Command{
		Use:          "mc-fleet",
		Short:        "Sync Minecraft mods against a published spec; launch the game.",
		SilenceUsage: true,
		Version:      Version,
	}

	root.AddCommand(newSyncCmd())
	root.AddCommand(newLaunchCmd())
	root.AddCommand(newApplyCmd())
	root.AddCommand(newVersionCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func newSyncCmd() *cobra.Command {
	var (
		remote     string
		specPath   string
		machineKey string
		dryRun     bool
		clean      bool
	)
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Pull spec, reconcile mods folder; do NOT launch Minecraft",
		RunE: func(_ *cobra.Command, _ []string) error {
			s, err := spec.Load(remote, specPath)
			if err != nil {
				return err
			}
			m := launcher.LocalMachine(machineKey)
			return reconcile.Run(s, m, reconcile.Opts{DryRun: dryRun, Clean: clean})
		},
	}
	cmd.Flags().StringVar(&remote, "remote", defaultRemote, "URL to fetch fleet-spec.json from")
	cmd.Flags().StringVar(&specPath, "spec", "", "local path to fleet-spec.json (overrides --remote)")
	cmd.Flags().StringVar(&machineKey, "machine", "", "machine name (defaults to hostname or 'local')")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print actions without writing")
	cmd.Flags().BoolVar(&clean, "clean", true, "delete jars that are not in the spec")
	return cmd
}

func newLaunchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "launch",
		Short: "Launch Minecraft.app / Minecraft.exe without syncing",
		RunE: func(_ *cobra.Command, _ []string) error {
			return launcher.OpenMinecraft()
		},
	}
	return cmd
}

func newApplyCmd() *cobra.Command {
	var (
		remote      string
		specPath    string
		machineKey  string
		dryRun      bool
		clean       bool
		launchAfter bool
	)
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "sync + launch in one call (default behaviour of meappy-mc)",
		RunE: func(_ *cobra.Command, _ []string) error {
			s, err := spec.Load(remote, specPath)
			if err != nil {
				return err
			}
			m := launcher.LocalMachine(machineKey)
			if err := reconcile.Run(s, m, reconcile.Opts{DryRun: dryRun, Clean: clean}); err != nil {
				return err
			}
			if launchAfter && !dryRun {
				return launcher.OpenMinecraft()
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&remote, "remote", defaultRemote, "URL to fetch fleet-spec.json from")
	cmd.Flags().StringVar(&specPath, "spec", "", "local path to fleet-spec.json (overrides --remote)")
	cmd.Flags().StringVar(&machineKey, "machine", "", "machine name (defaults to hostname or 'local')")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print actions without writing")
	cmd.Flags().BoolVar(&clean, "clean", true, "delete jars that are not in the spec")
	cmd.Flags().BoolVar(&launchAfter, "launch", true, "launch Minecraft after the sync succeeds")
	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println("mc-fleet", Version)
		},
	}
}
