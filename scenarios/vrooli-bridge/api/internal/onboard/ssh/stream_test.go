package ssh

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A persisted ControlMaster keeps the ssh client's stdout and stderr open
// after the client is killed. RunStreaming must still return promptly when its
// context is cancelled; otherwise a hung remote step (2026-09-15: a Mac's
// setup waiting on a dialog) holds an onboarding op forever and cancel never
// lands.
func TestRunStreamingReturnsOnCancelWhileAnInheritedPipeStaysOpen(t *testing.T) {
	dir := t.TempDir()
	// The background child inherits stdout/stderr and outlives the client,
	// exactly like a ControlPersist master.
	script := "#!/bin/sh\n(sleep 20) &\necho started\nsleep 20\n"
	if err := os.WriteFile(filepath.Join(dir, "ssh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := NewService(t.TempDir()).RunStreaming(ctx, ConnectionConfig{Host: "node.test", Port: 22, User: "operator"}, "true", StreamOptions{
			OnStdoutLine: func(line string) {
				if line == "started" {
					cancel()
				}
			},
		})
		done <- err
	}()

	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("RunStreaming did not return after cancel while an inherited pipe stayed open")
	}
}
