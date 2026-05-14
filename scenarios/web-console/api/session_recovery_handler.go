package main

// copyCodexHome lives in package main because it depends on the per-session
// CODEX_HOME layout (sessionCodexHome in pty.go). It is injected into the
// sessions handler's Adapter via the CopyCodexHome field.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// copyCodexHome rsyncs (or copies via tar fallback) the per-session
// CODEX_HOME from the orphan to the fresh pane. Bounded: codex homes
// contain symlinks to global config + per-session rollouts, typically
// < 50MB.
func copyCodexHome(oldID, newID string) error {
	src := sessionCodexHome(oldID)
	dst := sessionCodexHome(newID)
	if _, err := os.Stat(src); err != nil {
		return nil
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return fmt.Errorf("mkdirall %s: %w", dst, err)
	}
	if path, err := exec.LookPath("rsync"); err == nil {
		out, err := exec.Command(path, "-a", "--", src+"/", dst+"/").CombinedOutput()
		if err != nil {
			return fmt.Errorf("rsync %s -> %s: %v: %s", src, dst, err, string(out))
		}
		return nil
	}
	if path, err := exec.LookPath("cp"); err == nil {
		out, err := exec.Command(path, "-a", filepath.Join(src, "."), dst).CombinedOutput()
		if err != nil {
			return fmt.Errorf("cp -a %s -> %s: %v: %s", src, dst, err, string(out))
		}
		return nil
	}
	return fmt.Errorf("neither rsync nor cp available")
}
