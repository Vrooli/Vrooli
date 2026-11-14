//go:build !windows

package hostinventory

import (
	"context"
	"testing"
	"time"
)

func TestOSCommandRunnerCancelsChildProcessGroup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	started := time.Now()
	_, err := (osCommandRunner{}).Run(ctx, "sh", "-c", "sleep 30 & wait")
	if err == nil {
		t.Fatal("Run returned nil for a canceled process group")
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("Run waited %s after context cancellation", elapsed)
	}
}
