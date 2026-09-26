package convergence

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// CommandRunner is the seam for the soft upstream reads (scenario-auditor /
// test-genie clean-on-all-tools). Production wires execRunner; tests inject a
// fake. A nil runner means "no live read" — the affected signal degrades to its
// honest unknown value rather than failing.
type CommandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

const defaultRunTimeout = 10 * time.Second

func execRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	if _, err := exec.LookPath(name); err != nil {
		return nil, fmt.Errorf("command %q not found on PATH: %w", name, err)
	}
	ctx, cancel := context.WithTimeout(ctx, defaultRunTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).Output()
	if err != nil {
		return nil, fmt.Errorf("run %s %s: %w", name, strings.Join(args, " "), err)
	}
	return out, nil
}
