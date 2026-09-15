package runner

import (
	"errors"
	"os/exec"
	"testing"
)

type codedExitError struct{ code int }

func (e codedExitError) Error() string { return "remote process exited" }
func (e codedExitError) ExitCode() int { return e.code }

func TestExtractExitCodeHandlesLocalAndRemoteLaunchFailures(t *testing.T) {
	if code, ok := ExtractExitCode(nil); !ok || code != 0 {
		t.Fatalf("nil error = %d, %v", code, ok)
	}
	remote := codedExitError{code: 17}
	if code, ok := ExtractExitCode(remote); !ok || code != 17 {
		t.Fatalf("remote exit = %d, %v", code, ok)
	}
	if code, ok := ExtractExitCode(errors.New("transport failed")); ok || code != -1 {
		t.Fatalf("transport error = %d, %v", code, ok)
	}
	cmd := exec.Command("sh", "-c", "exit 9")
	if err := cmd.Run(); err == nil {
		t.Fatal("expected local command failure")
	} else if code, ok := ExtractExitCode(err); !ok || code != 9 {
		t.Fatalf("local exit = %d, %v (err=%v)", code, ok, err)
	}
}
