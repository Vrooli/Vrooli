package lifecycle

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAcquireSharedPackageLockHasBoundedWait(t *testing.T) {
	oldTimeout, oldPoll := sharedPackageLockWaitTimeout, sharedPackageLockPollInterval
	sharedPackageLockWaitTimeout, sharedPackageLockPollInterval = 40*time.Millisecond, 5*time.Millisecond
	t.Cleanup(func() { sharedPackageLockWaitTimeout, sharedPackageLockPollInterval = oldTimeout, oldPoll })

	home := t.TempDir()
	root := t.TempDir()
	release, err := acquireSharedPackageLock(home, "@vrooli/stalled", root, nil)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	defer release()

	started := time.Now()
	_, err = acquireSharedPackageLockContext(context.Background(), home, "@vrooli/stalled", root, nil)
	if err == nil {
		t.Fatal("second acquire unexpectedly succeeded")
	}
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("bounded lock wait took %s", elapsed)
	}
	if !strings.Contains(err.Error(), "await: condition not met") {
		t.Fatalf("error = %v, want bounded await diagnostic", err)
	}
}

type synchronizedBuffer struct {
	mu sync.Mutex
	b  strings.Builder
}

func TestAcquireSharedPackageLockUsesGeneratorLockForProto(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()

	release, err := acquireSharedPackageLock(home, "proto", root, nil)
	if err != nil {
		t.Fatalf("acquire Proto lock: %v", err)
	}
	defer release()

	lockPath := filepath.Join(home, ".vrooli", "locks", "proto-generation.lock")
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("stat unified Proto lock %s: %v", lockPath, err)
	}

	legacyPath := sharedPackageLockPath(filepath.Join(home, scenarioLockDirName), root)
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("legacy per-root lock %s unexpectedly exists, stat error=%v", legacyPath, err)
	}
}

func (w *synchronizedBuffer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.b.Write(p)
}

func (w *synchronizedBuffer) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.b.String()
}

func TestAcquireSharedPackageLockSerializesProvisioningAndReportsWait(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	var log synchronizedBuffer

	releaseFirst, err := acquireSharedPackageLock(home, "@vrooli/example", root, &log)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}

	secondAcquired := make(chan struct{})
	secondDone := make(chan error, 1)
	go func() {
		releaseSecond, acquireErr := acquireSharedPackageLock(home, "@vrooli/example", root, &log)
		if acquireErr == nil {
			close(secondAcquired)
			releaseSecond()
		}
		secondDone <- acquireErr
	}()

	select {
	case <-secondAcquired:
		t.Fatal("second provisioning acquired the lock before the first released it")
	case <-time.After(350 * time.Millisecond):
	}

	releaseFirst()
	select {
	case err := <-secondDone:
		if err != nil {
			t.Fatalf("second acquire: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("second provisioning did not acquire after release")
	}

	output := log.String()
	if !strings.Contains(output, "event=waiting") || !strings.Contains(output, "event=acquired") {
		t.Fatalf("lock diagnostics = %q, want waiting and acquired events", output)
	}
	if !strings.Contains(output, `package="@vrooli/example"`) || !strings.Contains(output, root) {
		t.Fatalf("lock diagnostics = %q, want package and root", output)
	}
}
