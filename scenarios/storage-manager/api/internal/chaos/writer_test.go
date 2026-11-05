package chaos

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type fakeFile struct{ io.Writer }

func (fakeFile) Close() error { return nil }

func TestRunRejectsUnboundedInputs(t *testing.T) {
	if _, err := Run(context.Background(), Config{Root: "relative", BytesPerHour: 1, Duration: time.Second}); err == nil {
		t.Fatal("relative root was accepted")
	}
	if _, err := Run(context.Background(), Config{Root: t.TempDir(), BytesPerHour: maxRate + 1, Duration: time.Second}); err == nil {
		t.Fatal("excessive rate was accepted")
	}
	if _, err := Run(context.Background(), Config{Root: t.TempDir(), BytesPerHour: 1, Duration: maxDuration + time.Second}); err == nil {
		t.Fatal("excessive duration was accepted")
	}
}

func TestRunWritesWithinGovernedRootAndStopsAtDeadline(t *testing.T) {
	root := t.TempDir()
	now := time.Unix(100, 0)
	steps := 0
	written := int64(0)
	report, err := Run(context.Background(), Config{
		Root:         root,
		BytesPerHour: int64(chunkSize) * 3600,
		Duration:     2 * time.Millisecond,
		Now: func() time.Time {
			steps++
			return now.Add(time.Duration(steps) * time.Millisecond)
		},
		Sleep: func(context.Context, time.Duration) error { return nil },
		OpenFile: func(path string, flags int, mode os.FileMode) (io.WriteCloser, error) {
			if filepath.Dir(path) != root {
				t.Fatalf("writer escaped root: %s", path)
			}
			return fakeFile{Writer: countingWriter{count: &written}}, nil
		},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.BytesWritten != written || report.BytesWritten == 0 {
		t.Fatalf("report bytes=%d writer bytes=%d", report.BytesWritten, written)
	}
	if report.Path != filepath.Join(root, ".vrooli-chaos-writer") {
		t.Fatalf("path=%q", report.Path)
	}
}

type countingWriter struct{ count *int64 }

func (w countingWriter) Write(p []byte) (int, error) {
	*w.count += int64(len(p))
	return len(p), nil
}
