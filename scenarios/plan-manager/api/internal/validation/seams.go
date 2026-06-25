package validation

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	internalplans "plan-manager/internal/plans"
)

// PlanSource is the read seam onto the plans SSOT. Production wraps the plans
// domain Service; tests inject a fake. validation only reads plans — it never
// mutates them.
type PlanSource interface {
	GetPlan(ctx context.Context, idOrSlug string) (internalplans.Plan, error)
}

// ReferenceResolver resolves a single code/req/doc reference against code-facts.
// Production wires a code-facts-backed resolver; tests inject a fake. A nil
// resolver (or one returning an error) degrades the reference to UNRESOLVED —
// recorded but unresolved, never a false "resolved".
type ReferenceResolver interface {
	Resolve(ctx context.Context, ref internalplans.Reference) (internalplans.Reference, error)
}

// StalenessComputer computes the staleness tier + change factor of a resolved
// reference. Production wires the freshness engine + code-facts; tests inject a
// fake. A nil computer (or an error) degrades staleness to UNKNOWN.
type StalenessComputer interface {
	Compute(ctx context.Context, ref internalplans.Reference) (internalplans.StalenessTier, float64, error)
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

// execRunner is the production CommandRunner. It guards against running an
// arbitrary binary by requiring the command to be on PATH, bounds the call with
// a timeout, and returns combined output. Never fabricates results: a failure is
// a real error the caller turns into UNKNOWN/FAIL honestly.
func execRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	if _, err := exec.LookPath(name); err != nil {
		return nil, fmt.Errorf("command %q not found on PATH: %w", name, err)
	}
	ctx, cancel := context.WithTimeout(ctx, validationTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("run %s %s: %w", name, strings.Join(args, " "), err)
	}
	return out, nil
}
