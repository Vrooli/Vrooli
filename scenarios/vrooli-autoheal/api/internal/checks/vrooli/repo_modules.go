package vrooli

// Repo-root Go module consistency.
//
// This check exists because of a detection gap, not a recovery gap. On
// 2026-09-01 a new import inside packages/api-core pulled two transitive
// dependencies that consumer modules did not carry in their go.sum. Nothing in
// the repo noticed. The first signal was a scenario failing to start, and
// because a failed build makes `vrooli scenario restart` a one-way door, the
// ecosystem drained to 118 of 120 scenarios stopped before anyone looked.
//
// langrecover.RecoverRepoRootGo was written for exactly this failure, with a
// doc comment naming it -- but no check ever produced a result that could
// select it, so the recovery path was unreachable in practice. This check is
// the missing half: it asks, on a schedule, whether the repo-root module still
// resolves, and it maps a positive drift signature onto that recovery.
//
// Cost note: it runs `go -C <root> list -e ./...`, which loads and resolves the
// package graph without compiling it. That is enough to surface "missing
// go.sum entry" and "updates to go.mod needed", and it is far cheaper than a
// full build, so the check can run often enough to catch drift early.

import (
	"context"
	"strings"
	"time"

	langrecover "vrooli-autoheal-langrecover"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

const (
	repoModulesCheckID = "vrooli-repo-root-modules"

	// recommendedActionRecoverRepoRootGo names the recovery that repairs the
	// repo-root module graph.
	recommendedActionRecoverRepoRootGo = "recover-repo-root-go"

	// rootCauseRepoRootModuleDrift marks a repo-root module graph that no
	// longer resolves.
	rootCauseRepoRootModuleDrift = "repo-root-module-drift"
)

// repoModulesProbeTimeout bounds the module resolution probe.
const repoModulesProbeTimeout = 4 * time.Minute

// RepoModulesCheck verifies that the repo-root Go module graph still resolves.
type RepoModulesCheck struct {
	executor checks.CommandExecutor
	interval int
	// resolveRoot locates the repo root; overridable for tests.
	resolveRoot func() string
}

// RepoModulesCheckOption configures a RepoModulesCheck.
type RepoModulesCheckOption func(*RepoModulesCheck)

// WithRepoModulesExecutor sets the command executor (for testing).
func WithRepoModulesExecutor(executor checks.CommandExecutor) RepoModulesCheckOption {
	return func(c *RepoModulesCheck) { c.executor = executor }
}

// WithRepoModulesRoot overrides repo-root resolution (for testing).
func WithRepoModulesRoot(resolve func() string) RepoModulesCheckOption {
	return func(c *RepoModulesCheck) { c.resolveRoot = resolve }
}

// NewRepoModulesCheck creates the repo-root module consistency check.
func NewRepoModulesCheck(opts ...RepoModulesCheckOption) *RepoModulesCheck {
	c := &RepoModulesCheck{
		executor:    checks.DefaultExecutor,
		interval:    900,
		resolveRoot: resolveVrooliRoot,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *RepoModulesCheck) ID() string    { return repoModulesCheckID }
func (c *RepoModulesCheck) Title() string { return "Repo Root Go Modules" }
func (c *RepoModulesCheck) Description() string {
	return "Verifies the repo-root Go module graph still resolves (go.mod/go.sum consistency)"
}

func (c *RepoModulesCheck) Importance() string {
	return "A shared-package import change can leave consumer modules without the go.sum entries " +
		"they need. Every affected build then fails, which turns scenario restart into a one-way door."
}
func (c *RepoModulesCheck) Category() checks.Category  { return checks.CategoryScenario }
func (c *RepoModulesCheck) IntervalSeconds() int       { return c.interval }
func (c *RepoModulesCheck) Platforms() []platform.Type { return nil }
func (c *RepoModulesCheck) IsCritical() bool           { return true }

// Run resolves the repo-root package graph and classifies any drift.
func (c *RepoModulesCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{CheckID: c.ID(), Details: make(map[string]interface{})}

	root := c.resolveRoot()
	if root == "" {
		result.Status = checks.StatusWarning
		result.Message = "repo root could not be resolved; module consistency not checked"
		// Nothing was measured, so healing would be guesswork.
		result.Details["autoHealEligible"] = false
		return result
	}
	result.Details["repoRoot"] = root

	probeCtx, cancel := context.WithTimeout(ctx, repoModulesProbeTimeout)
	defer cancel()

	output, err := c.executor.CombinedOutput(probeCtx, "go", "-C", root, "list", "-e", "./...")
	outputText := string(output)

	signature := langrecover.DetectGoSignature(outputText)
	if signature != langrecover.GoSignatureNone {
		// Critical: while this is true, nothing that depends on the module
		// graph can build, and every restart is destructive.
		result.Status = checks.StatusCritical
		result.Message = "repo-root Go module graph does not resolve"
		result.Details["rootCause"] = rootCauseRepoRootModuleDrift
		result.Details["recommendedAction"] = recommendedActionRecoverRepoRootGo
		result.Details["signature"] = goSignatureName(signature)
		result.Details["output"] = truncateForDetails(outputText)
		return result
	}

	if err != nil {
		// The probe failed for a reason this check cannot classify. Report it,
		// but do not invite a dependency recovery that may be irrelevant.
		result.Status = checks.StatusWarning
		result.Message = "repo-root module probe failed"
		result.Details["error"] = err.Error()
		result.Details["output"] = truncateForDetails(outputText)
		result.Details["autoHealEligible"] = false
		return result
	}

	result.Status = checks.StatusOK
	result.Message = "repo-root Go module graph resolves"
	return result
}

// goSignatureName renders a signature for the incident record.
func goSignatureName(sig langrecover.GoSignature) string {
	switch sig {
	case langrecover.GoSignatureMissingSum:
		return "missing-go-sum-entry"
	case langrecover.GoSignatureMissingModule:
		return "missing-module"
	default:
		return "none"
	}
}

// repoModulesOutputCap bounds stored probe output; `go list` over a large repo
// can emit a great deal on failure.
const repoModulesOutputCap = 8000

func truncateForDetails(value string) string {
	if len(value) <= repoModulesOutputCap {
		return value
	}
	return value[:repoModulesOutputCap] + "\n[output truncated]"
}

// RecoveryActions offers the repo-root module repair.
func (c *RepoModulesCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	available := false
	if lastResult != nil && lastResult.Details != nil {
		available = lastResult.Details["rootCause"] == rootCauseRepoRootModuleDrift
	}
	return []checks.RecoveryAction{
		{
			ID:          recommendedActionRecoverRepoRootGo,
			Name:        "Repair repo-root Go modules",
			Description: "Runs the minimum-viable Go module recovery (download, escalating to tidy) at the repo root",
			// Gated on an actual drift signature: this rewrites go.mod and
			// go.sum, so it must never run speculatively.
			Available: available,
		},
	}
}

// ExecuteAction runs the repo-root module recovery.
func (c *RepoModulesCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	start := time.Now()
	if actionID != recommendedActionRecoverRepoRootGo {
		return checks.ActionResult{
			ActionID: actionID,
			CheckID:  c.ID(),
			Success:  false,
			Message:  "unknown action: " + actionID,
		}
	}

	root := c.resolveRoot()
	if root == "" {
		return checks.ActionResult{
			ActionID: actionID,
			CheckID:  c.ID(),
			Success:  false,
			Message:  "repo root could not be resolved",
			Duration: time.Since(start),
		}
	}

	// Re-derive the signature from a fresh probe rather than trusting a stale
	// result: the drift may already have been repaired by hand.
	probeCtx, cancel := context.WithTimeout(ctx, repoModulesProbeTimeout)
	output, _ := c.executor.CombinedOutput(probeCtx, "go", "-C", root, "list", "-e", "./...")
	cancel()

	decision := langrecover.DecideRepoRoot(string(output), root)
	if !decision.Has() {
		return checks.ActionResult{
			ActionID: actionID,
			CheckID:  c.ID(),
			Success:  true,
			Message:  "no module drift signature present; nothing to repair",
			Duration: time.Since(start),
		}
	}

	result, err := langrecover.RecoverRepoRootGo(ctx, recoveryRunner(), root, decision.GoSig)

	actionOutput := result.Output
	if len(result.ModifiedPaths) > 0 {
		actionOutput = "modified: " + strings.Join(result.ModifiedPaths, ", ") + "\n" + actionOutput
	}

	// An unattended heal that moves a dependency version must say so loudly.
	// Warning is the right channel: the action succeeded, but it changed what
	// gets compiled, and only a human can judge whether that was wanted.
	warning := ""
	if changed := langrecover.ChangedVersionDeltas(result.VersionDeltas); len(changed) > 0 {
		warning = "recovery changed selected module versions: " + langrecover.FormatVersionDeltas(changed)
	}

	action := checks.ActionResult{
		ActionID: actionID,
		CheckID:  c.ID(),
		Output:   actionOutput,
		Warning:  warning,
		Duration: time.Since(start),
	}
	switch {
	case err != nil:
		action.Success, action.Message, action.Error = false, "repo-root recovery failed", err.Error()
	case result.Err != nil:
		action.Success, action.Message, action.Error = false, "repo-root recovery command failed", result.Err.Error()
	default:
		action.Success = true
		action.Message = "repo-root Go module graph repaired via " + result.Command
	}
	return action
}
