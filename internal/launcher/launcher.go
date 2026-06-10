// Package launcher knows how to find the Minecraft launcher directory on
// each OS, locate per-profile mods folders, and spawn Minecraft.app /
// Minecraft.exe / minecraft-launcher.
package launcher

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Machine identifies the local install. Public users only need the OS-derived
// default (hostname); admin SSH-fanout flows pass an explicit name.
type Machine struct {
	Name      string
	LauncherDir string // absolute path to ~/Library/Application Support/minecraft (or platform equiv)
}

// LocalMachine constructs a Machine for the current OS user. name="" → hostname
// (or "local" if hostname fetch fails).
func LocalMachine(name string) Machine {
	if name == "" {
		if h, err := os.Hostname(); err == nil && h != "" {
			name = h
		} else {
			name = "local"
		}
	}
	return Machine{Name: name, LauncherDir: defaultLauncherDir()}
}

func defaultLauncherDir() string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "minecraft")
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), ".minecraft")
	default: // linux et al
		return filepath.Join(home, ".minecraft")
	}
}

// ProfileModsDir returns the per-profile mods/ path for a given profile name.
func (m Machine) ProfileModsDir(profileName string) string {
	return filepath.Join(m.LauncherDir, "profiles", profileName, "mods")
}

// OpenMinecraft launches the Minecraft launcher. Real impl in commit 6.
func OpenMinecraft() error {
	return fmt.Errorf("launcher.OpenMinecraft: not implemented yet (commit 6)")
}
