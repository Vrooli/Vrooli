package execplan

import (
	"strconv"

	"scenario-to-cloud/apierrors"
)

// Facts are the current material observations a reviewed plan is checked
// against at admission and again at effect boundaries.
type Facts struct {
	DeploymentRevision  uint64
	TargetEnrollment    uint64
	ReleaseDigest       string
	ClosureDigest       string
	ConfigurationDigest string
	DataSchemaVersion   string
	PrivilegeSet        string
}

// FactsOf derives the material facts from a freshly compiled plan and the
// inputs it was compiled from.
func FactsOf(plan *Plan, in CompileInputs) Facts {
	return Facts{
		DeploymentRevision:  in.Observations.DeploymentRevision,
		TargetEnrollment:    in.Deployment.Target.EnrollmentGeneration,
		ReleaseDigest:       plan.ReleaseDigest,
		ClosureDigest:       plan.ClosureDigest,
		ConfigurationDigest: plan.ConfigurationDigest,
		DataSchemaVersion:   in.Observations.DataSchemaVersion,
		PrivilegeSet:        PrivilegeSet(plan.Actions, in.Closure),
	}
}

// StaleReason is one difference between a reviewed plan and the present.
type StaleReason struct {
	Kind     string `json:"kind"`
	Expected string `json:"expected"`
	Observed string `json:"observed"`
	Material bool   `json:"material"`
}

// Harmless reason kinds. These never invalidate a review.
const (
	ReasonPresentation = "presentation"
	ReasonTimestamp    = "timestamp"
)

// Validate compares the preconditions recorded in plan with current facts.
// Every precondition kind is material; a mismatch means the plan is stale
// and must be recompiled and reviewed again.
func Validate(plan *Plan, current Facts) []StaleReason {
	if plan == nil {
		return nil
	}
	observed := map[string]string{
		PreconditionDeploymentRevision:  strconv.FormatUint(current.DeploymentRevision, 10),
		PreconditionTargetEnrollment:    strconv.FormatUint(current.TargetEnrollment, 10),
		PreconditionReleaseDigest:       current.ReleaseDigest,
		PreconditionClosureDigest:       current.ClosureDigest,
		PreconditionConfigurationDigest: current.ConfigurationDigest,
		PreconditionDataSchema:          current.DataSchemaVersion,
		PreconditionPrivilegeSet:        current.PrivilegeSet,
	}
	var reasons []StaleReason
	for _, pre := range plan.Preconditions {
		got, known := observed[pre.Kind]
		if !known {
			continue
		}
		if got != pre.Value {
			reasons = append(reasons, StaleReason{Kind: pre.Kind, Expected: pre.Value, Observed: got, Material: true})
		}
	}
	return reasons
}

// Compare classifies the differences between a reviewed plan and a
// recompiled one. Precondition and target differences are material;
// presentation-only differences are harmless.
func Compare(reviewed, recompiled *Plan) []StaleReason {
	if reviewed == nil || recompiled == nil {
		return nil
	}
	var reasons []StaleReason
	for _, pre := range reviewed.Preconditions {
		got, ok := recompiled.Precondition(pre.Kind)
		if !ok || got != pre.Value {
			reasons = append(reasons, StaleReason{Kind: pre.Kind, Expected: pre.Value, Observed: got, Material: true})
		}
	}
	if reviewed.Target != recompiled.Target {
		reasons = append(reasons, StaleReason{Kind: "target", Expected: targetLabel(reviewed.Target), Observed: targetLabel(recompiled.Target), Material: true})
	}
	if reviewed.Presentation != recompiled.Presentation {
		reasons = append(reasons, StaleReason{Kind: ReasonPresentation, Expected: reviewed.Presentation.Title, Observed: recompiled.Presentation.Title, Material: false})
	}
	return reasons
}

// Material reports whether any reason invalidates the review.
func Material(reasons []StaleReason) bool {
	for _, reason := range reasons {
		if reason.Material {
			return true
		}
	}
	return false
}

// StaleError is the typed plan_stale failure for material reasons.
func StaleError(reasons []StaleReason) *apierrors.Error {
	material := make([]StaleReason, 0, len(reasons))
	for _, reason := range reasons {
		if reason.Material {
			material = append(material, reason)
		}
	}
	return apierrors.New(apierrors.CodePlanStale, "Material preconditions changed since the plan was reviewed; recompile and review again").
		WithDetail("reasons", material).
		WithNextAction(apierrors.NextAction{Owner: "scenario-to-cloud", Kind: "replan", Reference: "plan", Label: "Compile a fresh plan and review it"})
}

// DigestMismatchError is the typed plan_digest_mismatch failure.
func DigestMismatchError(reviewed, current string) *apierrors.Error {
	return apierrors.New(apierrors.CodePlanDigestMismatch, "The reviewed plan digest does not match the plan compiled now").
		WithDetail("reviewed_plan_digest", reviewed).
		WithDetail("current_plan_digest", current).
		WithNextAction(apierrors.NextAction{Owner: "scenario-to-cloud", Kind: "replan", Reference: "plan", Label: "Compile a fresh plan and review it"})
}

// NeedsInputError is the typed needs_input failure carrying the handoff.
func NeedsInputError(handoff *Handoff) *apierrors.Error {
	err := apierrors.New(apierrors.CodeNeedsInput, "The plan needs operator input before it can be applied")
	if handoff != nil {
		err = err.WithDetail("missing", handoff.Missing).
			WithNextAction(apierrors.NextAction{Owner: handoff.Owner, Kind: handoff.Kind, Reference: handoff.Reference, Label: "Resume onboarding to supply the missing inputs"})
	}
	return err
}
