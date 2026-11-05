package lifecycle

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// O18: a lifecycle run's log survives the loss of the logs root (disk
// pressure recovery reclaims it), and only the last runs are kept.
func TestLifecycleRunLogsAreRetainedOutsideTheLogsRoot(t *testing.T) {
	home := t.TempDir()
	runner := &Runner{Home: home, Out: io.Discard, Err: io.Discard}
	failure := errors.New("vite build: exit status 1")
	for i := 0; i < lifecycleRunLogKeep+3; i++ {
		_, _ = runner.runWithLifecycleLog(startLifecycleLogContext("demo", "restart", "setup"), func(logWriter, childWriter io.Writer) error {
			_, _ = io.WriteString(childWriter, "build output line\n")
			return failure
		})
		time.Sleep(2 * time.Millisecond)
	}
	logsRoot := filepath.Join(home, ".vrooli", "logs")
	if err := os.Remove(filepath.Join(logsRoot, "demo.log")); err != nil {
		t.Fatalf("simulate pressure reclaim: %v", err)
	}

	dir, err := LifecycleRunLogsDir(home, "demo")
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(dir, logsRoot) {
		t.Fatalf("retained run logs must live outside the reclaimable logs root: %s", dir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("retained run logs: %v", err)
	}
	if len(entries) != lifecycleRunLogKeep {
		t.Fatalf("retained %d runs, want %d", len(entries), lifecycleRunLogKeep)
	}
	data, err := os.ReadFile(filepath.Join(dir, entries[len(entries)-1].Name()))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"build output line", "status: failed", "vite build: exit status 1", "operation: restart", "phase: setup"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("retained run log lacks %q:\n%s", want, data)
		}
	}
}

// One run is bounded: its head and tail are kept and the middle dropped.
func TestRetainedRunLogKeepsHeadAndTail(t *testing.T) {
	home := t.TempDir()
	log := openRetainedRunLog(home, lifecycleLogContext{Scenario: "demo", Operation: "start", Phase: "setup"}, "run-1")
	if log == nil {
		t.Fatal("retained log not opened")
	}
	chunk := strings.Repeat("x", 64<<10)
	_, _ = log.Write([]byte("FIRST LINE\n"))
	for written := 0; written < lifecycleRunLogHeadBytes+3*lifecycleRunLogTailBytes; written += len(chunk) {
		_, _ = log.Write([]byte(chunk))
	}
	_, _ = log.Write([]byte("\nLAST LINE\n"))
	log.close()
	data, err := os.ReadFile(log.path)
	if err != nil {
		t.Fatal(err)
	}
	if int64(len(data)) > lifecycleRunLogHeadBytes+lifecycleRunLogTailBytes+4096 {
		t.Fatalf("retained run log is %d bytes, beyond its bound", len(data))
	}
	for _, want := range []string{"FIRST LINE", "LAST LINE", "bytes of this run were omitted"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("retained run log lacks %q", want)
		}
	}
}
