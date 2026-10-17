package lifecycle

import (
	"errors"
	"os"
	"strings"
	"testing"

	platform "github.com/vrooli/platform-go"
)

func TestRunPhaseDetailedRejectsConcurrentLifecycleOperation(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixture(t, root, "alpha")
	runner := newLifecycleRunnerForTest(t, root, home, nil)
	seedScenarioLockFile(t, home, "alpha", "424242\n")

	originalLock := lockFileFn
	t.Cleanup(func() { lockFileFn = originalLock })
	lockFileFn = func(_ *os.File, nonBlocking bool) (func(), error) {
		if nonBlocking {
			return nil, platform.ErrLockUnavailable
		}
		return func() {}, nil
	}

	_, err := runner.RunPhaseDetailed("alpha", "setup", PhaseOptions{})
	if !errors.Is(err, ErrScenarioBusy) {
		t.Fatalf("error = %v, want ErrScenarioBusy", err)
	}
	if !strings.Contains(err.Error(), "phase \"setup\"") || !strings.Contains(err.Error(), "pid 424242") {
		t.Fatalf("error = %v, want phase and lock owner details", err)
	}
}
