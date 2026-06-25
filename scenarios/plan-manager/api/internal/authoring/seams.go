package authoring

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	internalplans "plan-manager/internal/plans"
)

// PlanWriter is the write seam onto the plans SSOT. Finalize maps the session's
// sections into a plans.Plan and persists it through this seam. Production wraps
// the plans domain Service (a CreatePlan adapter in the handler module, mirroring
// validation's planAdapter); tests inject a fake. authoring never owns the plan
// record — it only composes and writes it.
type PlanWriter interface {
	CreatePlan(ctx context.Context, p internalplans.Plan) (internalplans.Plan, error)
}

// AnchorAutofiller captures the regression anchor for a plan-in-progress. The
// production impl shells git-control-tower (LookPath-guarded, timeout-bounded);
// a nil seam or an error degrades the regression-anchor section to "left for the
// author" (AutofillResult.Degraded=true) — never a fabricated anchor.
type AnchorAutofiller interface {
	// Anchor returns the prose to fill the regression-anchor section for the
	// given plan title/slug. An error degrades that section honestly.
	Anchor(ctx context.Context, title, slug string) (string, error)
}

// RequiredReadingSource discovers the required-reading set via prompt-manager
// plan-skill-discovery. A nil seam or an error degrades the required-reading
// section honestly.
type RequiredReadingSource interface {
	// RequiredReading returns the prose to fill the required-reading section for
	// the given plan title/intent. An error degrades that section honestly.
	RequiredReading(ctx context.Context, title string) (string, error)
}

// ReferenceExtractor extracts code references via code-facts. A nil seam or an
// error degrades the references section honestly.
type ReferenceExtractor interface {
	// References returns the prose to fill the references section for the given
	// plan title/scope. An error degrades that section honestly.
	References(ctx context.Context, title, scope string) (string, error)
}

// CommandRunner is the exec seam shared by the production autofill sources (the
// live dispatch path to git-control-tower / prompt-manager / code-facts).
// Production wires execRunner (LookPath-guarded, timeout-bounded); tests inject a
// fake. A nil runner means "no live dispatch" — the source degrades honestly.
type CommandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

// DefaultRunner returns the production CommandRunner (LookPath-guarded,
// timeout-bounded). Wired in the handler module; tests inject a fake instead.
func DefaultRunner() CommandRunner { return execRunner }

// autofillTimeout bounds a single autofill dispatch. Generous (a discovery or
// anchor capture can shell out) but finite so a hung command cannot block the
// wizard forever.
const autofillTimeout = 2 * time.Minute

// execRunner is the production CommandRunner. It guards against running an
// arbitrary binary by requiring the command to be on PATH, bounds the call with
// a timeout, and returns combined output. Never fabricates results: a failure is
// a real error the caller turns into a degraded autofill honestly.
func execRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	if _, err := exec.LookPath(name); err != nil {
		return nil, fmt.Errorf("command %q not found on PATH: %w", name, err)
	}
	ctx, cancel := context.WithTimeout(ctx, autofillTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("run %s %s: %w", name, strings.Join(args, " "), err)
	}
	return out, nil
}

// --- production autofill sources (CommandRunner-backed, all degradable) ---

// cmdAnchorAutofiller captures the regression anchor by shelling git-control-tower
// through the CommandRunner seam. A nil runner or a command failure degrades.
type cmdAnchorAutofiller struct{ run CommandRunner }

// NewCommandAnchorAutofiller wires the production AnchorAutofiller over the given
// CommandRunner (git-control-tower). A nil runner yields a source that always
// degrades (honest, never a false fill).
func NewCommandAnchorAutofiller(run CommandRunner) AnchorAutofiller {
	return cmdAnchorAutofiller{run: run}
}

func (a cmdAnchorAutofiller) Anchor(ctx context.Context, title, slug string) (string, error) {
	if a.run == nil {
		return "", fmt.Errorf("git-control-tower unavailable")
	}
	out, err := a.run(ctx, "git-control-tower", "baseline", "show", "--json")
	if err != nil {
		return "", err
	}
	captured := strings.TrimSpace(string(out))
	if captured == "" {
		return "", fmt.Errorf("git-control-tower returned no baseline")
	}
	return captured, nil
}

// cmdRequiredReadingSource discovers required reading via prompt-manager
// plan-skill-discovery through the CommandRunner seam.
type cmdRequiredReadingSource struct{ run CommandRunner }

// NewCommandRequiredReadingSource wires the production RequiredReadingSource over
// the given CommandRunner (prompt-manager). A nil runner always degrades.
func NewCommandRequiredReadingSource(run CommandRunner) RequiredReadingSource {
	return cmdRequiredReadingSource{run: run}
}

func (s cmdRequiredReadingSource) RequiredReading(ctx context.Context, title string) (string, error) {
	if s.run == nil {
		return "", fmt.Errorf("prompt-manager unavailable")
	}
	out, err := s.run(ctx, "prompt-manager", "discover", title, "--type", "skill,doc")
	if err != nil {
		return "", err
	}
	discovered := strings.TrimSpace(string(out))
	if discovered == "" {
		return "", fmt.Errorf("prompt-manager returned no required reading")
	}
	return discovered, nil
}

// cmdReferenceExtractor extracts code references via code-facts through the
// CommandRunner seam.
type cmdReferenceExtractor struct{ run CommandRunner }

// NewCommandReferenceExtractor wires the production ReferenceExtractor over the
// given CommandRunner (code-facts). A nil runner always degrades.
func NewCommandReferenceExtractor(run CommandRunner) ReferenceExtractor {
	return cmdReferenceExtractor{run: run}
}

func (e cmdReferenceExtractor) References(ctx context.Context, title, scope string) (string, error) {
	if e.run == nil {
		return "", fmt.Errorf("code-facts unavailable")
	}
	query := strings.TrimSpace(title + " " + scope)
	out, err := e.run(ctx, "code-facts", "search", query, "--json")
	if err != nil {
		return "", err
	}
	refs := strings.TrimSpace(string(out))
	if refs == "" {
		return "", fmt.Errorf("code-facts returned no references")
	}
	return refs, nil
}
