package dependencyhealth

import (
	"context"
	"os"
	"os/exec"

	"github.com/vrooli/envkit-go"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// seam: surfaceDiscoverer isolates Code Facts surface discovery from health orchestration.
type surfaceDiscoverer interface {
	Discover(ctx context.Context, scenario, scenarioDir, repoRoot string, useCache bool) (surfaceInventory, error)
}

// seam: commandRunner isolates host command execution for deterministic health tests.
type commandRunner interface {
	Run(ctx context.Context, dir string, name string, args ...string) (string, error)
}

// seam: runtimeStatusFetcher isolates Vrooli runtime status checks from dependency health.
type runtimeStatusFetcher interface {
	ResourceStatuses(ctx context.Context) (*cliv1.ResourceStatusesResponse, error)
	ScenarioStatus(ctx context.Context, name string) (*cliv1.ScenarioStatusSingle, error)
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, dir string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, envkit.Env{"GOWORK=off"})
	out, err := cmd.CombinedOutput()
	return string(out), err
}
