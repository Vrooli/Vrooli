// Package clipboard provides cross-platform clipboard operations.
package clipboard

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// Copy copies text to the system clipboard.
// Returns an error message describing why it failed, or empty string on success.
func Copy(text string) string {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		// Try xclip first, then xsel, then wl-copy (Wayland)
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.Command("wl-copy")
		} else {
			return "No clipboard tool found (install xclip, xsel, or wl-copy)"
		}
	case "windows":
		cmd = exec.Command("clip")
	default:
		return fmt.Sprintf("Clipboard not supported on %s", runtime.GOOS)
	}

	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Sprintf("Failed to copy to clipboard: %v", err)
	}

	return ""
}

// IsAvailable checks if clipboard functionality is available on this system.
func IsAvailable() bool {
	switch runtime.GOOS {
	case "darwin":
		_, err := exec.LookPath("pbcopy")
		return err == nil
	case "linux":
		if _, err := exec.LookPath("xclip"); err == nil {
			return true
		}
		if _, err := exec.LookPath("xsel"); err == nil {
			return true
		}
		if _, err := exec.LookPath("wl-copy"); err == nil {
			return true
		}
		return false
	case "windows":
		_, err := exec.LookPath("clip")
		return err == nil
	default:
		return false
	}
}

// ToolName returns the name of the clipboard tool that would be used.
func ToolName() string {
	switch runtime.GOOS {
	case "darwin":
		return "pbcopy"
	case "linux":
		if _, err := exec.LookPath("xclip"); err == nil {
			return "xclip"
		}
		if _, err := exec.LookPath("xsel"); err == nil {
			return "xsel"
		}
		if _, err := exec.LookPath("wl-copy"); err == nil {
			return "wl-copy"
		}
		return "none"
	case "windows":
		return "clip"
	default:
		return "none"
	}
}
