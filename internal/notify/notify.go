// Package notify shows native progress + completion notifications without
// opening a terminal window. macOS uses NSUserNotification (via beeep),
// Windows uses toast, Linux uses libnotify.
//
// Phase-1 stub: stdout fallback. Native impls land in commit 6.
package notify

import "fmt"

// Start kicks off a job (title + first status). Idempotent across multiple
// invocations of the same job within one process.
func Start(title, status string) {
	fmt.Printf("── %s ──\n  %s\n", title, status)
}

// Progress updates the current job's status line.
func Progress(status string) {
	fmt.Printf("  %s\n", status)
}

// Done marks the current job as completed.
func Done(status string) {
	fmt.Printf("✅ %s\n", status)
}

// Error marks the current job as failed.
func Error(status string) {
	fmt.Printf("❌ %s\n", status)
}
