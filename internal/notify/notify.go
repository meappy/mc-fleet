// Package notify routes progress + completion through native OS notifications
// when available, falling back to stdout. This lets the binary run without a
// terminal window — critical for the public meappy-mc.app double-click path
// on macOS, and for Windows users (no terminal, no Cygwin needed).
//
// macOS uses NSUserNotificationCenter via beeep.
// Windows uses toast notifications via beeep.
// Linux uses libnotify/notify-send via beeep.
//
// All four entry points are safe to call when stdout isn't a TTY — we just
// suppress the stdout echo in that case.
package notify

import (
	"fmt"
	"os"

	"github.com/gen2brain/beeep"
	"golang.org/x/term"
)

var (
	currentTitle = "mc-fleet"
	echoStdout   = term.IsTerminal(int(os.Stdout.Fd()))
)

// Start kicks off a job. `title` is the app name shown in the notification;
// `status` is the first-line progress message.
func Start(title, status string) {
	currentTitle = title
	_ = beeep.Notify(title, status, "")
	if echoStdout {
		fmt.Printf("── %s ──\n  %s\n", title, status)
	}
}

// Progress updates the current job's status. Intentionally NOT firing a
// notification — that would spam the user with toasts. Stdout only.
func Progress(status string) {
	if echoStdout {
		fmt.Printf("  %s\n", status)
	}
}

// Done marks the job as completed with a success notification.
func Done(status string) {
	_ = beeep.Notify(currentTitle, status, "")
	if echoStdout {
		fmt.Printf("✅ %s\n", status)
	}
}

// Error marks the job as failed and fires an alert-style notification (more
// likely to break through Do Not Disturb on macOS).
func Error(status string) {
	_ = beeep.Alert(currentTitle, status, "")
	if echoStdout {
		fmt.Fprintf(os.Stderr, "❌ %s\n", status)
	}
}
