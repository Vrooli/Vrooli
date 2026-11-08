//go:build linux

package maintenance

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/config"
)

func TestOrphanCurrentHostPinnedStopProbe(t *testing.T) {
	if os.Getenv("VROOLI_ORPHAN_NATIVE_PROBE") != "1" {
		t.Skip("set VROOLI_ORPHAN_NATIVE_PROBE=1 for an isolated native orphan fixture")
	}
	root, home := t.TempDir(), t.TempDir()
	image := filepath.Join(root, "scenarios", "orphan-fixture", "api", "server")
	data, err := os.ReadFile("/bin/sleep")
	if err != nil {
		t.Fatal(err)
	}
	if err := config.WriteOwnedFileDurable(image, data, 0o700); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(image, "60")
	cmd.Env = []string{"HOME=" + home, "PATH=/usr/bin:/bin"}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	scope := fmt.Sprintf("outage-orphan-probe-%d", time.Now().UnixNano())
	ref, _, err := platform.AdoptIntoScope(platform.AdoptSpec{PID: cmd.Process.Pid, Scope: scope, Slice: platform.ServicesSlice, Description: "Isolated maintenance orphan regression fixture"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	entry, ok := readProcessEntry(cmd.Process.Pid)
	if !ok || entry.Cgroup != ref.Path || insideLifecycleOwnedScope(entry) {
		t.Fatalf("fixture scope not isolated: %+v", entry)
	}
	if err := NewController(root, home).stopOrphan(ctx, cmd.Process.Pid); err != nil {
		t.Fatal(err)
	}
	if members, err := platform.ScopeProcesses(ref); (err != nil && !errors.Is(err, os.ErrNotExist)) || len(members) != 0 {
		t.Fatalf("fixture native scope still has members or is unreadable: %v %v", members, err)
	}
	t.Logf("pinned fixture PID%d stopped; exact native scope %s empty; no serving process mutated", cmd.Process.Pid, scope)
}
