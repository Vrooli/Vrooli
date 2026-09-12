package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/vrooli/envkit-go"
)

var (
	fakeAgentOnce sync.Once
	fakeAgentPath string
	fakeAgentErr  error
)

// BuildFakeAgent builds the test-only corpus replay executable once for the
// current test binary. It is never wired into a production launch path.
func BuildFakeAgent(t testing.TB) string {
	t.Helper()
	fakeAgentOnce.Do(func() {
		_, file, _, ok := runtime.Caller(0)
		if !ok {
			fakeAgentErr = os.ErrNotExist
			return
		}
		apiRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
		dir, err := os.MkdirTemp("", "agent-manager-fake-agent-")
		if err != nil {
			fakeAgentErr = err
			return
		}
		fakeAgentPath = filepath.Join(dir, "fake-agent")
		cmd := exec.Command("go", "build", "-o", fakeAgentPath, "./cmd/fake-agent")
		cmd.Dir = apiRoot
		cmd.Env = envkit.Toolchain(envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, nil), envkit.ToolchainOptions{})
		fakeAgentErr = cmd.Run()
	})
	if fakeAgentErr != nil {
		t.Fatalf("build fake agent: %v", fakeAgentErr)
	}
	return fakeAgentPath
}
