package measures

// DefaultConfidenceThreshold is the conservative default auto-execution
// threshold (θ): a read-only measure is auto-executed only when the minimum
// param-resolution confidence is at least this high. 0.8 admits deterministic
// time-window resolution (1.0) and manifest-default fallback (0.8) but withholds
// auto-execution behind a confirmation when any param leaned on a lower-
// confidence constrained/best-effort LLM extraction — i.e. we auto-answer only
// when we are sure of every parameter. Phase 3/4 expose it as a tunable config
// lever (per the plan's "θ default conservative").
const DefaultConfidenceThreshold = 0.8

// GateAction is the decision the auto-execution gate reaches for a resolved
// measure. It is the single place the read-vs-confirm contract is encoded.
type GateAction int

const (
	// GateNeedsParams: a required param could not be resolved. The measure is
	// not executed; the caller must ask for the missing params (never guess).
	GateNeedsParams GateAction = iota
	// GateConfirm: params are fully resolved but execution is withheld pending
	// explicit user confirmation. Reached when the measure is write/destructive
	// (NEVER auto-run, even at full confidence) or when confidence < θ.
	GateConfirm
	// GateExecute: a read-only, run-eligible measure resolved every required
	// param at confidence ≥ θ — safe to auto-execute and return the answer.
	GateExecute
)

// GateDecision is the gate's verdict plus a human-readable reason (surfaced in
// provenance / --explain).
type GateDecision struct {
	Action GateAction
	// Reason explains the verdict (e.g. "write effect requires confirmation").
	Reason string
}

// Execute reports whether the decision permits auto-execution.
func (d GateDecision) Execute() bool { return d.Action == GateExecute }

// Gate decides whether a resolved measure may be auto-executed, keyed on the
// EXISTING, enforced governance signals (effect + run_eligible) plus the
// confidence threshold θ. The ordering is deliberate and safety-first:
//
//  1. missing required params  → GateNeedsParams (ask, don't guess);
//  2. not auto-executable      → GateConfirm (write/destructive or not
//     run-eligible ALWAYS confirm, even with complete params at full
//     confidence — Non-goal §12);
//  3. confidence below θ       → GateConfirm (we resolved everything but are
//     not sure enough to run unattended);
//  4. otherwise                → GateExecute.
//
// It is a pure function of the declaration, the resolution result, and θ, so it
// is identically enforceable by the provider, by cli-health static checks, and
// by tests.
func Gate(decl MeasureDeclaration, res ResolveResult, threshold float64) GateDecision {
	if !res.Complete() {
		return GateDecision{Action: GateNeedsParams, Reason: "required parameter(s) unresolved: " + joinNeeds(res.Needs)}
	}
	if !decl.AutoExecutable() {
		return GateDecision{Action: GateConfirm, Reason: confirmReason(decl)}
	}
	if res.Confidence < threshold {
		return GateDecision{Action: GateConfirm, Reason: "resolution confidence below auto-execute threshold"}
	}
	return GateDecision{Action: GateExecute, Reason: "read-only measure, all params resolved at high confidence"}
}

// confirmReason states why a fully-resolved measure still requires confirmation.
func confirmReason(decl MeasureDeclaration) string {
	switch {
	case decl.Effect == EffectWrite:
		return "write effect requires confirmation (never auto-executed)"
	case decl.Effect == EffectDestructive:
		return "destructive effect requires confirmation (never auto-executed)"
	case !decl.RunEligible:
		return "measure is not run-eligible; requires confirmation"
	default:
		return "measure is not auto-executable; requires confirmation"
	}
}

// joinNeeds renders the unresolved-param list for the gate reason.
func joinNeeds(needs []string) string {
	out := ""
	for i, n := range needs {
		if i > 0 {
			out += ", "
		}
		out += n
	}
	return out
}
