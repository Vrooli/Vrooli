package discovery

import (
	"context"
	"os/exec"
	"time"
)

type commandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type defaultCommandRunner struct{}

func (defaultCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

var execRunner commandRunner = defaultCommandRunner{}

// commandTimeout bounds each `vrooli` discovery sweep. `vrooli scenario list
// --json` walks every scenario and can take ~10s on a full tree, so the budget
// must clear that with headroom — an under-budget timeout silently empties the
// create-task picker (results are cached for cacheTTL afterward).
var commandTimeout = 30 * time.Second
