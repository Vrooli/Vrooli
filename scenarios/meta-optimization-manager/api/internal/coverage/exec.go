package coverage

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// CommandRunner is the seam for invoking another scenario's CLI (the live read
// path). Production wires execRunner (os/exec); tests inject a fake returning
// canned JSON so no subprocess or live owner is needed. Keeping this a single
// narrow func type means both the SpaceReader and the NumeratorJoiner share one
// fake in tests.
type CommandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

// defaultRunTimeout bounds a single live owner read so one slow/hung owner CLI
// degrades the affected projection rather than stalling the whole scoreboard.
// test-genie health can take >10s while aggregating its fleet ledger, so keep
// this long enough for the real Validate numerator but still finite.
const defaultRunTimeout = 30 * time.Second

// execRunner is the production CommandRunner: it runs the named command with the
// given args and returns stdout. A non-zero exit or a missing binary surfaces as
// an error so the caller degrades gracefully (available=false), never panics.
func execRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	if _, err := exec.LookPath(name); err != nil {
		return nil, fmt.Errorf("command %q not found on PATH: %w", name, err)
	}
	ctx, cancel := context.WithTimeout(ctx, defaultRunTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("run %s %s: %w", name, strings.Join(args, " "), err)
	}
	return out, nil
}
