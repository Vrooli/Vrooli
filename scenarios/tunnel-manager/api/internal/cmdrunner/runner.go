// Package cmdrunner declares the seam for executing local control-plane
// commands (resource operations, including cloudflared restarts). Production wires
// cmdrunner.Default, which shells out via exec.CommandContext; tests
// substitute testutil/mocks.FakeCmdRunner to assert the exact argv and
// stub output without touching the host.
//
// The tunnel, config, and recovery domains all actuate the host through
// this one boundary — querying and changing managed-resource state — so it is
// declared once here rather than reinvented per domain.
package cmdrunner

import (
	"context"
	"os/exec"
)

// Runner executes a system command and returns its combined output.
// The signature mirrors os/exec so Default is a thin adapter and fakes
// stay trivial.
type Runner func(ctx context.Context, name string, args ...string) ([]byte, error)

// Default runs a command via exec.CommandContext and returns its
// combined stdout+stderr. It is the production Runner.
func Default(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}
