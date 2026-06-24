package trials

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// CommandRunner is the seam for invoking agent-manager / workspace-sandbox (the
// live dispatch path). Production wires execRunner; tests inject a fake. A nil
// runner means "no live dispatch" — RunTask yields an honest VerdictError rather
// than fabricating a pass.
type CommandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

// dispatchTimeout bounds a single trial dispatch. Trials are expensive and
// operator-invoked; a generous ceiling keeps a hung agent from blocking forever
// without prematurely killing a legitimately long local-model run.
const dispatchTimeout = 10 * time.Minute

func execRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	if _, err := exec.LookPath(name); err != nil {
		return nil, fmt.Errorf("command %q not found on PATH: %w", name, err)
	}
	ctx, cancel := context.WithTimeout(ctx, dispatchTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).Output()
	if err != nil {
		return nil, fmt.Errorf("run %s %s: %w", name, strings.Join(args, " "), err)
	}
	return out, nil
}
