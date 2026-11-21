package coverage

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// CommandRunner is the seam for invoking another scenario's CLI. Since the
// numerator reads moved to typed API↔API calls (numeratorclient.go), the only
// remaining CLI shell-out is the SpaceReader's denominator read of the owner's
// `space --projection <p> --json` verb. Production wires execRunner (os/exec);
// tests inject a fake returning canned JSON so no subprocess or live owner is
// needed.
type CommandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

// spaceReadTimeout bounds a single denominator (space-verb) read so one slow or
// hung owner CLI degrades the affected projection rather than stalling the whole
// scoreboard. The space verb is a cheap doc projection (it has a fast file-parse
// fallback), so a short bound is correct; the numerator deadline lives with the
// typed joiner (numeratorDeadline).
const spaceReadTimeout = 5 * time.Second

// execRunner is the production CommandRunner: it runs the named command with the
// given args and returns stdout. A non-zero exit or a missing binary surfaces as
// an error so the caller degrades gracefully (the SpaceReader falls back to a
// direct doc parse), never panics.
func execRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	if _, err := exec.LookPath(name); err != nil {
		return nil, fmt.Errorf("command %q not found on PATH: %w", name, err)
	}
	ctx, cancel := context.WithTimeout(ctx, spaceReadTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("run %s %s: %w", name, strings.Join(args, " "), err)
	}
	return out, nil
}
