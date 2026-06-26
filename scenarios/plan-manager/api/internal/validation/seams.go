package validation

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	planmodel "plan-manager/internal/planmodel"
)

// PlanSource is the read seam onto the plans SSOT. Production wraps the plans
// domain Service; tests inject a fake. validation only reads plans — it never
// mutates them.
type PlanSource interface {
	GetPlan(ctx context.Context, idOrSlug string) (planmodel.Plan, error)
}

// ReferenceResolver resolves a single code/req/doc reference against code-facts.
// Production wires a code-facts-backed resolver; tests inject a fake. A nil
// resolver (or one returning an error) degrades the reference to UNRESOLVED —
// recorded but unresolved, never a false "resolved".
type ReferenceResolver interface {
	Resolve(ctx context.Context, ref planmodel.Reference) (planmodel.Reference, error)
}

// StalenessComputer computes the staleness tier + change factor of a resolved
// reference. Production wires the filesystem existence floor plus git-sourced
// per-reference drift refinement; tests inject a fake. A nil computer (or an
// error) degrades staleness to UNKNOWN.
type StalenessComputer interface {
	Compute(ctx context.Context, ref planmodel.Reference) (planmodel.StalenessTier, float64, error)
}

// ResultStore persists validation results and reads back the last-known result
// per plan/phase. RunValidation (the explicit, agent-in-the-loop check) writes;
// the execution context server reads the LAST STORED result so status/next never
// shell a subprocess. A nil store means "no persistence" — RunValidation still
// returns its live result, but nothing is cached for later cheap reads.
type ResultStore interface {
	SaveResult(ctx context.Context, r Result) error
	LastResult(ctx context.Context, planID, phaseID string) (Result, bool, error)
}

// CommandRunner is the exec seam for running baseline/check commands (the live
// dispatch path to git-control-tower / the diff oracle). Production wires
// execRunner (LookPath-guarded, timeout-bounded); tests inject a fake. A nil
// runner means "no live dispatch" — RunValidation yields an honest UNKNOWN
// verdict rather than fabricating a pass.
type CommandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

// DefaultRunner returns the production CommandRunner (LookPath-guarded,
// timeout-bounded). Wired in the handler module; tests inject a fake instead.
func DefaultRunner() CommandRunner { return execRunner }

// validationTimeout bounds a single baseline/check dispatch. Generous so a
// legitimately long baseline isn't killed, but finite so a hung command cannot
// block forever.
const validationTimeout = 10 * time.Minute

// ErrToolNotFound is returned by a CommandRunner when the command is not on PATH.
// The caller treats this as UNKNOWN (the check could not be performed) rather
// than FAIL (the check ran and reported a problem) — a host without
// git-control-tower installed must not make every plan look like it regressed.
var ErrToolNotFound = errors.New("command not found on PATH")

// CommandExitError reports that a command ran to completion but exited non-zero.
// Code carries the process exit code so the caller can distinguish a real
// regression (git-control-tower baseline diff exit 1) from a "not comparable"
// result (exit 2, which is actionable but not a regression → UNKNOWN).
type CommandExitError struct {
	Code   int
	Output []byte
	Err    error
}

func (e CommandExitError) Error() string {
	return fmt.Sprintf("command exited %d: %v", e.Code, e.Err)
}

func (e CommandExitError) Unwrap() error { return e.Err }

// execRunner is the production CommandRunner. It guards against running an
// arbitrary binary by requiring the command to be on PATH, bounds the call with
// a timeout, and returns combined output. Never fabricates results: a tool-absent
// miss yields ErrToolNotFound (→ UNKNOWN) and a non-zero exit yields a
// CommandExitError carrying the code (→ FAIL/UNKNOWN by exit code), so the caller
// degrades honestly instead of conflating "not installed" with "regressed".
func execRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	if _, err := exec.LookPath(name); err != nil {
		return nil, fmt.Errorf("command %q: %w", name, ErrToolNotFound)
	}
	ctx, cancel := context.WithTimeout(ctx, validationTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return out, CommandExitError{Code: exitErr.ExitCode(), Output: out, Err: err}
		}
		return out, fmt.Errorf("run %s %s: %w", name, strings.Join(args, " "), err)
	}
	return out, nil
}
