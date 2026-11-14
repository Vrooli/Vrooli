package seeds

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestSeedCancellationContainsRunningDescendantsWithHeldOrRedirectedOutput(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("manager-owned operation integration requires Linux")
	}
	for _, redirect := range []bool{false, true} {
		t.Run(fmt.Sprint(redirect), func(t *testing.T) {
			root := t.TempDir()
			dir := filepath.Join(root, "bas", "seeds")
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			output := ""
			if redirect {
				output = " >/dev/null 2>&1"
			}
			script := "sleep 120" + output + " &\nprintf '%s' \"$!\" > child.pid\nwait\n"
			if err := os.WriteFile(filepath.Join(dir, "seed.sh"), []byte(script), 0600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())
			manager := NewManager(root, root, root, nil)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			done := make(chan error, 1)
			go func() { _, err := manager.Apply(ctx); done <- err }()
			var pid string
			deadline := time.After(5 * time.Second)
			for pid == "" {
				select {
				case err := <-done:
					t.Fatalf("seed exited before descendant witness: %v", err)
				case <-deadline:
					t.Fatal("descendant did not start")
				default:
				}
				raw, err := os.ReadFile(filepath.Join(root, "child.pid"))
				if err == nil {
					pid = strings.TrimSpace(string(raw))
				}
				if pid == "" {
					time.Sleep(5 * time.Millisecond)
				}
			}
			statPath := "/proc/" + pid + "/stat"
			before, err := os.ReadFile(statPath)
			if err != nil {
				t.Fatalf("child was not alive before timeout: %v", err)
			}
			started := time.Now()
			cancel()
			select {
			case err := <-done:
				if err == nil {
					t.Fatal("cancelled seed reported success")
				}
			case <-time.After(8 * time.Second):
				t.Fatal("descendant output held seed cancellation beyond bound")
			}
			if elapsed := time.Since(started); elapsed > 8*time.Second {
				t.Fatalf("cancellation took %s", elapsed)
			}
			after, err := os.ReadFile(statPath)
			if err == nil && string(after) == string(before) {
				t.Fatal("exact running descendant survived seed cancellation")
			}
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				t.Fatal(err)
			}
			if err == nil {
				fields := strings.Fields(string(after))
				if len(fields) < 3 || (fields[2] != "Z" && fields[2] != "X") {
					t.Fatalf("descendant remains live after owner cleanup: %s", after)
				}
			}
		})
	}
}
