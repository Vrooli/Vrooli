//go:build unix

package source

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
)

func TestRecordingRefusesFIFOWithoutWaitingForWriter(t *testing.T) {
	fifo := filepath.Join(t.TempDir(), "source.webm")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	for _, freeze := range []bool{false, true} {
		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		done := make(chan error, 1)
		go func() {
			if freeze {
				_, err := FreezeVideoSource(ctx, &VideoSource{Path: fifo}, t.TempDir())
				done <- err
			} else {
				_, err := ResolveVideoSourceContext(ctx, []executionwriter.ArtifactData{{ArtifactType: "video", Payload: map[string]any{"path": fifo}}}, nil)
				done <- err
			}
		}()
		select {
		case err := <-done:
			if err == nil {
				t.Fatal("accepted named pipe")
			}
		case <-time.After(250 * time.Millisecond):
			t.Fatal("recording open blocked waiting for FIFO writer")
		}
		cancel()
	}
	if _, err := openRegularRecording(t.Context(), filepath.Dir(fifo)); err == nil {
		t.Fatal("accepted directory")
	}
	if _, err := openRegularRecording(t.Context(), "/dev/null"); err == nil {
		t.Fatal("accepted character device")
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(fifo, link); err != nil {
		t.Fatal(err)
	}
	if _, err := openRegularRecording(t.Context(), link); err == nil {
		t.Fatal("accepted symlink to FIFO")
	}
}
