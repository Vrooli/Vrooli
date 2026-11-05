package exports_service

import (
	"os/exec"
	"path/filepath"
	"runtime"
)

// osOpener invokes the host OS's file-manager via os/exec. It is the
// default SystemOpener installed by Module when no override is supplied.
//
// Linux falls back to opening the containing directory for Reveal because
// most Linux file managers do not support selecting a specific file via
// the freedesktop xdg-open contract.
type osOpener struct{}

func (osOpener) Reveal(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", "-R", path)
	case "windows":
		cmd = exec.Command("explorer", "/select,", path)
	default:
		cmd = exec.Command("xdg-open", filepath.Dir(path))
	}
	return cmd.Run()
}

func (osOpener) OpenFolder(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("explorer", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Run()
}
