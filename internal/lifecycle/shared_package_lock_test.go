package lifecycle

import (
	"strings"
	"sync"
	"testing"
	"time"
)

type synchronizedBuffer struct {
	mu sync.Mutex
	b  strings.Builder
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
