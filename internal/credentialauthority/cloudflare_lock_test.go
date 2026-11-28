package credentialauthority

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestCloudflareConnectorLockSerializesProcesses(t *testing.T) {
	if os.Getenv("VROOLI_TEST_CONNECTOR_LOCK_CHILD") == "1" {
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()
		release, err := AcquireCloudflareConnectorLock(ctx)
		if errors.Is(err, context.DeadlineExceeded) {
			os.Exit(42)
		}
		if err != nil {
			t.Fatal(err)
		}
		release()
		return
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SUDO_USER", "")
	release, err := AcquireCloudflareConnectorLock(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	run := func() error {
		cmd := exec.Command(os.Args[0], "-test.run=^TestCloudflareConnectorLockSerializesProcesses$")
		cmd.Env = append(os.Environ(), "VROOLI_TEST_CONNECTOR_LOCK_CHILD=1")
		return cmd.Run()
	}
	var exit *exec.ExitError
	if err := run(); !errors.As(err, &exit) || exit.ExitCode() != 42 {
		t.Fatalf("child must wait for held lock: %v", err)
	}
	release()
	if err := run(); err != nil {
		t.Fatalf("child must acquire released lock: %v", err)
	}
}
