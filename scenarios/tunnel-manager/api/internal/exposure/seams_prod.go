package exposure

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"tunnel-manager/internal/cmdrunner"
)

// FilePortResolver resolves a scenario's fixed UI port from its
// service.json (<root>/<scenario>/.vrooli/service.json, ports.ui.port).
// It is the production PortResolver; tests use a fake.
type FilePortResolver struct {
	Root string
}

// NewFilePortResolver constructs a resolver rooted at the scenarios dir.
func NewFilePortResolver(root string) *FilePortResolver {
	return &FilePortResolver{Root: root}
}

var _ PortResolver = (*FilePortResolver)(nil)

// serviceJSON is the minimal slice of service.json this domain reads. A
// fixed UI port lives at ports.ui.port; ranged/dynamic scenarios omit it.
type serviceJSON struct {
	Ports map[string]struct {
		Port int `json:"port"`
	} `json:"ports"`
}

func (r *FilePortResolver) UIPort(_ context.Context, scenario string) (int, error) {
	path := filepath.Join(r.Root, scenario, ".vrooli", "service.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, ErrPortUnresolved{Scenario: scenario, Reason: fmt.Sprintf("service.json not readable: %v", err)}
	}
	var svc serviceJSON
	if err := json.Unmarshal(data, &svc); err != nil {
		return 0, ErrPortUnresolved{Scenario: scenario, Reason: fmt.Sprintf("service.json parse: %v", err)}
	}
	ui, ok := svc.Ports["ui"]
	if !ok || ui.Port == 0 {
		return 0, ErrPortUnresolved{Scenario: scenario, Reason: "no fixed UI port declared"}
	}
	return ui.Port, nil
}

// CLIRunner ensures a scenario is running by shelling `vrooli scenario
// start <scenario>` through the cmdrunner seam. It is the production
// Runner; tests use a fake. The lifecycle system makes start idempotent
// (a running scenario is a no-op), so re-exposing is safe.
type CLIRunner struct {
	Runner cmdrunner.Runner
}

// NewCLIRunner constructs the production Runner.
func NewCLIRunner(runner cmdrunner.Runner) *CLIRunner {
	return &CLIRunner{Runner: runner}
}

var _ Runner = (*CLIRunner)(nil)

func (r *CLIRunner) EnsureRunning(ctx context.Context, scenario string) error {
	if _, err := r.Runner(ctx, "vrooli", "scenario", "start", scenario); err != nil {
		return fmt.Errorf("ensure %q running: %w", scenario, err)
	}
	return nil
}
