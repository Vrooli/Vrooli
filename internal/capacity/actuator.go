package capacity

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ApplyExecutor is the injectable seam over running an adopter's degrade verb
// (§8.8). Production shells the owner's resource CLI; unit tests provide a fake
// so the actuator never execs/dockers in a test run. owner is the claim owner id
// (e.g. "whisper"); verb+argv come from the claim's DegradeProfile.Apply with
// the "{label}" placeholder substituted for the chosen step.
type ApplyExecutor interface {
	Apply(ctx context.Context, owner, verb string, argv []string) error
}

// ActuationOutcome records what the actuator did (or deliberately skipped) for
// one escalation action — honest accounting, no silent caps.
type ActuationOutcome struct {
	ClaimID    string `json:"claim_id"`
	OwnerID    string `json:"owner_id"`
	Action     string `json:"action"`
	ToStep     string `json:"to_step,omitempty"`
	Applied    bool   `json:"applied"`
	Skipped    bool   `json:"skipped,omitempty"`
	FreedBytes int64  `json:"frees_bytes,omitempty"`
	Reason     string `json:"reason,omitempty"`
	Err        string `json:"error,omitempty"`
}

// ActuationResult is the full set of outcomes for an escalation plan.
type ActuationResult struct {
	Outcomes   []ActuationOutcome `json:"outcomes"`
	FreedBytes int64              `json:"freed_bytes"`
}

// Actuate executes an escalation plan in enforce mode (§8.8). PlanEscalation
// stays a pure planner; Actuate is the orchestration layer it always lacked. For
// each action:
//
//   - request-degrade: resolve the target's DegradeProfile.Apply, substitute the
//     step label, run the owner's verb through the ApplyExecutor. On SUCCESS
//     record status=degraded + the new amount; on FAILURE leave the claim
//     unchanged and surface a warn (never strand a resource off-GPU). An
//     already-at-target claim is a no-op (idempotent); a target degraded within
//     policy.DegradeDebounce is skipped (anti-thrash).
//   - preempt: only when policy.PreemptEnabled — records status=preempted (the
//     last rung; degrade is always preferred). Auto-stop of the owner stays out
//     of V1 default scope, so preempt records intent in the ledger.
//   - warn: recorded as a skip (PlanEscalation emits this when a reclaim is
//     possible but preempt is disabled).
//
// Actuate is best-effort per action: one failing actuation never aborts the
// others, and the error surfaces in that outcome, not as a returned error. A
// returned error is reserved for store-level failures that make the result
// untrustworthy.
func Actuate(ctx context.Context, plan EscalationPlan, store ClaimRepository, exec ApplyExecutor, policy Policy, now time.Time) (ActuationResult, error) {
	var result ActuationResult
	for _, action := range plan.Actions {
		outcome := ActuationOutcome{
			ClaimID: action.ClaimID,
			OwnerID: action.OwnerID,
			Action:  action.Action,
			ToStep:  action.ToStep,
			Reason:  action.Reason,
		}
		switch action.Action {
		case ActionRequestDegrade:
			applied, err := actuateDegrade(ctx, store, exec, policy, now, action, &outcome)
			if err != nil {
				return result, err
			}
			if applied {
				result.FreedBytes += outcome.FreedBytes
			}
		case ActionPreempt:
			if !policy.PreemptEnabled {
				outcome.Skipped = true
				outcome.Reason = "preempt requested but policy.preempt_enabled is false"
				break
			}
			preempted, err := store.PreemptClaim(ctx, action.ClaimID, action.Reason)
			if err != nil {
				outcome.Err = err.Error()
				break
			}
			outcome.Applied = true
			outcome.FreedBytes = preempted.AmountBytes
			result.FreedBytes += preempted.AmountBytes
		default: // ActionWarn or unknown
			outcome.Skipped = true
		}
		result.Outcomes = append(result.Outcomes, outcome)
	}
	return result, nil
}

// actuateDegrade applies a single request-degrade action. It returns whether the
// degrade was applied (so the caller can sum freed bytes) and an error only on a
// store-level failure that should abort the whole actuation.
func actuateDegrade(ctx context.Context, store ClaimRepository, exec ApplyExecutor, policy Policy, now time.Time, action EscalationAction, outcome *ActuationOutcome) (bool, error) {
	claim, err := store.GetClaim(ctx, action.ClaimID)
	if err != nil {
		outcome.Err = err.Error()
		return false, nil // a vanished target is not fatal; the plan was stale
	}
	if claim.DegradeProfile == nil {
		outcome.Skipped = true
		outcome.Reason = "target has no degrade profile"
		return false, nil
	}
	targetAmount, ok := stepAmount(claim.DegradeProfile, action.ToStep)
	if !ok {
		outcome.Skipped = true
		outcome.Reason = fmt.Sprintf("step %q not in target profile", action.ToStep)
		return false, nil
	}
	// Idempotent: already at or below the requested step — nothing to do.
	if claim.AmountBytes <= targetAmount {
		outcome.Skipped = true
		outcome.Reason = fmt.Sprintf("already at or below step %q (%s)", action.ToStep, humanBytes(claim.AmountBytes))
		return false, nil
	}
	// Debounce: do not re-degrade a target that was just degraded (anti-thrash).
	if claim.Status == StatusDegraded && policy.DegradeDebounce > 0 && now.Sub(claim.UpdatedAt) < policy.DegradeDebounce {
		outcome.Skipped = true
		outcome.Reason = fmt.Sprintf("debounced: last degraded %s ago (< %s)", now.Sub(claim.UpdatedAt).Round(time.Second), policy.DegradeDebounce)
		return false, nil
	}

	verb := strings.TrimSpace(claim.DegradeProfile.Apply.Verb)
	argv := substituteLabel(claim.DegradeProfile.Apply.Argv, action.ToStep)
	if exec == nil {
		exec = DefaultExecutor()
	}
	if verb == "" {
		outcome.Err = "target degrade profile declares no apply verb"
		return false, nil
	}
	if applyErr := exec.Apply(ctx, claim.OwnerID, verb, argv); applyErr != nil {
		// Never strand the resource: leave the claim as-is and warn.
		outcome.Err = applyErr.Error()
		outcome.Reason = "actuator failed; claim left unchanged"
		return false, nil
	}
	degraded, err := store.DegradeClaim(ctx, claim.ClaimID, claim.Generation, action.ToStep, targetAmount)
	if err != nil {
		// The adopter resized but the ledger write lost a race; surface it, but the
		// resource itself did step down so this is not a "strand". Report and move on.
		outcome.Err = "adopter resized but ledger update failed: " + err.Error()
		return false, nil
	}
	outcome.Applied = true
	outcome.FreedBytes = claim.AmountBytes - degraded.AmountBytes
	return true, nil
}

// stepAmount returns the byte amount for a profile step label.
func stepAmount(profile *DegradeProfile, label string) (int64, bool) {
	if profile == nil {
		return 0, false
	}
	for _, st := range profile.Steps {
		if st.Label == label {
			return st.AmountBytes, true
		}
	}
	return 0, false
}

// substituteLabel returns a copy of argv with every "{label}" token replaced by
// the chosen step label.
func substituteLabel(argv []string, label string) []string {
	out := make([]string, len(argv))
	for i, a := range argv {
		out[i] = strings.ReplaceAll(a, "{label}", label)
	}
	return out
}

// EnforceReclaim plans and (in enforce mode) actuates the escalation needed to
// realize a verdict that depends on reclaiming idle lower-priority capacity. It
// is a no-op when the verdict needs no reclaim. In advisory/off it returns the
// plan WITHOUT actuating (the broker logs what it WOULD do); only enforce=on
// runs the actuator.
func EnforceReclaim(ctx context.Context, store ClaimRepository, requesterPriority int, verdict Verdict, ledger []CapacityClaim, exec ApplyExecutor, policy Policy, enforce string, now time.Time) (EscalationPlan, ActuationResult, error) {
	if verdict.ReclaimBytes <= 0 {
		return EscalationPlan{Satisfied: true}, ActuationResult{}, nil
	}
	plan := PlanEscalation(requesterPriority, verdict.ReclaimBytes, ledger, policy, now)
	if enforce != EnforceOn {
		return plan, ActuationResult{}, nil
	}
	res, err := Actuate(ctx, plan, store, exec, policy, now)
	return plan, res, err
}

// CmdExecutor is the production ApplyExecutor: it shells the owner's resource CLI
// (`resource-<owner> <verb> <argv...>` by convention). Both the command resolver
// and the runner are injectable so the convention can be overridden and tests
// stay hermetic.
type CmdExecutor struct {
	// CommandFor resolves the executable name for an owner. Default:
	// "resource-<owner>".
	CommandFor func(owner string) string
	// RunFn runs the resolved command. Default: exec.CommandContext capturing
	// combined output into the error on failure.
	RunFn func(ctx context.Context, name string, args ...string) error
}

// DefaultExecutor returns the production CmdExecutor.
func DefaultExecutor() CmdExecutor { return CmdExecutor{} }

// Apply runs the owner's degrade verb.
func (e CmdExecutor) Apply(ctx context.Context, owner, verb string, argv []string) error {
	name := "resource-" + owner
	if e.CommandFor != nil {
		name = e.CommandFor(owner)
	}
	args := append([]string{verb}, argv...)
	if e.RunFn != nil {
		return e.RunFn(ctx, name, args...)
	}
	cmd := exec.CommandContext(ctx, name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		trimmed := strings.TrimSpace(out.String())
		if trimmed != "" {
			return fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, trimmed)
		}
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}
