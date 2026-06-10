// Package launcher knows how to find the Minecraft launcher directory on
// each OS, locate per-profile mods folders, and spawn Minecraft.app /
// Minecraft.exe / minecraft-launcher.
package launcher

import (
	"fmt"
	"os"
	"os/exec"
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

// OpenMinecraft launches the Minecraft launcher (the official Microsoft
// launcher; the user picks the profile + clicks Play). Best-effort across
// platforms, with sensible fallbacks if the URI handler isn't registered.
func OpenMinecraft() error {
	switch runtime.GOOS {
	case "darwin":
		return runCmd("open", "-a", "Minecraft")
	case "windows":
		// Modern Minecraft launcher registers the minecraft:// URI scheme.
		// `start` is a cmd builtin, so route through cmd /c.
		if err := runCmd("cmd", "/c", "start", "minecraft://"); err == nil {
			return nil
		}
		// Fallback: Microsoft Store install location.
		if p := findFirstExisting(
			filepath.Join(os.Getenv("PROGRAMFILES"), "Minecraft Launcher", "MinecraftLauncher.exe"),
			filepath.Join(os.Getenv("PROGRAMFILES(X86)"), "Minecraft Launcher", "MinecraftLauncher.exe"),
		); p != "" {
			return runCmd(p)
		}
		return fmt.Errorf("could not locate Minecraft launcher (install from minecraft.net/download)")
	default: // linux, freebsd, ...
		// Try the binary first, then fall back to xdg-open with the URI handler.
		if _, err := exec.LookPath("minecraft-launcher"); err == nil {
			return runCmd("minecraft-launcher")
		}
		return runCmd("xdg-open", "minecraft://")
	}
}

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Start() // launcher runs detached — we don't wait
}

func findFirstExisting(paths ...string) string {
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
