package domain

import (
	"fmt"
	"strings"
	"time"
)

// QualificationUsageState separates usage the provider actually reported from
// usage it did not. Unknown usage is reserved conservatively and never treated
// as zero.
type QualificationUsageState string

const (
	QualificationUsageMeasured        QualificationUsageState = "measured"
	QualificationUsageUnknownReserved QualificationUsageState = "unknown-reserved"
)

// QualificationUsage records what one bounded live qualification run could and
// could not observe. It never invents a token or dollar value the provider did
// not report; an unreported value stays zero with ReservedUnknown set and the
// State marked unknown-reserved.
type QualificationUsage struct {
	State           QualificationUsageState `json:"state,omitempty"`
	InputTokens     int64                   `json:"inputTokens,omitempty"`
	OutputTokens    int64                   `json:"outputTokens,omitempty"`
	CostUSD         float64                 `json:"costUsd,omitempty"`
	ReservedUnknown bool                    `json:"reservedUnknown,omitempty"`
}

// QualificationReceipt is the reusable, route-keyed evidence record produced by
// one bounded live qualification run. It binds the route and the
// requested/owner-resolved configuration to the arguments the runner actually
// passed, the provider's own acknowledgment when the provider exposes one, the
// source and runtime identity, the operation identity, the accepted output, and
// the observed or truthfully-unknown usage.
//
// A receipt is evidence, never a launch. An empty layer stays empty rather than
// being backfilled, so a reader can tell "not observed" from "observed and
// equal". Dependent delegated work is admitted only when this receipt exists and
// its effective identities match the requested identity exactly.
type QualificationReceipt struct {
	// Route identifies the selected economical route the receipt qualifies
	// (for example a role reference such as "code.flatrate"). It is required:
	// a receipt without a route identity is not a qualification.
	Route string `json:"route,omitempty"`

	// Requested side is the caller's request taken verbatim from the admission
	// record. Empty means the caller did not request the field.
	RequestedRunner  string `json:"requestedRunner,omitempty"`
	RequestedModel   string `json:"requestedModel,omitempty"`
	RequestedRoleRef string `json:"requestedRoleRef,omitempty"`
	RequestedEffort  string `json:"requestedEffort,omitempty"`

	// Effective side is the owner-resolved configuration that actually ran.
	EffectiveRunner string `json:"effectiveRunner,omitempty"`
	EffectiveModel  string `json:"effectiveModel,omitempty"`
	EffectiveEffort string `json:"effectiveEffort,omitempty"`

	// Passed layer: the runner-native control arguments the selected codec
	// emitted, plus the diagnostics explaining any refused translation.
	PassedControlArgs      []string `json:"passedControlArgs,omitempty"`
	TranslationDiagnostics []string `json:"translationDiagnostics,omitempty"`

	// ProviderAcknowledgment is the provider's own confirmation of the
	// effective runner/model/effort, when the provider exposes one.
	ProviderAcknowledgment []string `json:"providerAcknowledgment,omitempty"`

	// Source and runtime identity. RuntimeVersion is only observable at a live
	// launch and is required for a usable receipt.
	CatalogDigest  string `json:"catalogDigest,omitempty"`
	PolicyDigest   string `json:"policyDigest,omitempty"`
	PolicyPath     string `json:"policyPath,omitempty"`
	RuntimeVersion string `json:"runtimeVersion,omitempty"`

	// Operation identity ties the receipt to the durable run/operation it came
	// from so a fresh coordinator can reconcile and reuse it.
	RunID       string `json:"runId,omitempty"`
	OperationID string `json:"operationId,omitempty"`

	// AcceptedOutput is true only when the bounded run produced identifiable
	// accepted output, independent of exit code or wrapper success.
	AcceptedOutput bool `json:"acceptedOutput,omitempty"`

	Usage       QualificationUsage `json:"usage,omitempty"`
	Limitations []string           `json:"limitations,omitempty"`
	CapturedAt  time.Time          `json:"capturedAt,omitempty"`
}

// DependentDelegationRequest is the route/configuration identity a piece of
// dependent delegated work is about to use.
type DependentDelegationRequest struct {
	Runner string
	Model  string
	Effort string
}

// QualificationEvidence is the live-observed evidence one bounded
// qualification run supplies to complete a receipt. Every field is taken from
// owner-observed runtime evidence; a zero field means "not observed" and is
// never inferred from the requested or effective layers. ProviderAcknowledgment
// may stay empty when the provider exposes no acknowledgment, but
// RuntimeVersion and AcceptedOutput are required for the receipt to qualify a
// dependent route.
type QualificationEvidence struct {
	Route                  string
	RuntimeVersion         string
	ProviderAcknowledgment []string
	AcceptedOutput         bool
	Usage                  QualificationUsage
	Limitations            []string
	RunID                  string
	OperationID            string
	CapturedAt             time.Time
}

// CaptureQualificationReceipt derives a receipt from an admission record and
// completes it with the observed live evidence in one call. Deriving and
// completing together keeps the route and the requested/effective/passed
// layers bound to the admission that produced them, so a caller cannot pair a
// receipt with a configuration it did not run. It copies the evidence slices
// rather than aliasing them and leaves any unobserved field empty.
func CaptureQualificationReceipt(admission *RunAdmission, evidence QualificationEvidence) *QualificationReceipt {
	receipt := NewQualificationReceipt(evidence.Route, admission)
	if receipt == nil {
		return nil
	}
	receipt.RuntimeVersion = strings.TrimSpace(evidence.RuntimeVersion)
	receipt.ProviderAcknowledgment = append([]string(nil), evidence.ProviderAcknowledgment...)
	receipt.AcceptedOutput = evidence.AcceptedOutput
	receipt.Usage = evidence.Usage
	receipt.Limitations = append([]string(nil), evidence.Limitations...)
	receipt.RunID = strings.TrimSpace(evidence.RunID)
	receipt.OperationID = strings.TrimSpace(evidence.OperationID)
	receipt.CapturedAt = evidence.CapturedAt
	return receipt
}

// MissingIdentities returns the ordered names of the identities a receipt must
// carry before it can qualify dependent delegation. It is empty only for a
// receipt that records accepted output, a route, a runtime identity, and an
// effective runner and model. AdmitDependentDelegation and reporting share this
// single completeness source so a partial receipt can never be reported as
// complete.
func (r *QualificationReceipt) MissingIdentities() []string {
	if r == nil {
		return []string{"receipt"}
	}
	var missing []string
	if !r.AcceptedOutput {
		missing = append(missing, "acceptedOutput")
	}
	if strings.TrimSpace(r.Route) == "" {
		missing = append(missing, "route")
	}
	if strings.TrimSpace(r.RuntimeVersion) == "" {
		missing = append(missing, "runtimeVersion")
	}
	if strings.TrimSpace(r.EffectiveRunner) == "" {
		missing = append(missing, "effectiveRunner")
	}
	if strings.TrimSpace(r.EffectiveModel) == "" {
		missing = append(missing, "effectiveModel")
	}
	return missing
}

// NewQualificationReceipt derives a receipt from an admission record, copying
// the requested/effective/passed layers verbatim. It intentionally leaves the
// live-only fields (runtime version, provider acknowledgment, accepted output,
// usage, limitations) for the caller to fill from the bounded run, so the
// caller cannot accidentally claim a live observation the owner did not make.
func NewQualificationReceipt(route string, admission *RunAdmission) *QualificationReceipt {
	if admission == nil {
		return nil
	}
	return &QualificationReceipt{
		Route:                  strings.TrimSpace(route),
		RequestedRunner:        admission.RequestedRunner,
		RequestedModel:         admission.RequestedModel,
		RequestedRoleRef:       admission.RequestedRoleRef,
		RequestedEffort:        admission.RequestedEffort,
		EffectiveRunner:        admission.EffectiveRunner,
		EffectiveModel:         admission.EffectiveModel,
		EffectiveEffort:        admission.EffectiveEffort,
		PassedControlArgs:      append([]string(nil), admission.PassedControlArgs...),
		TranslationDiagnostics: append([]string(nil), admission.TranslationDiagnostics...),
		ProviderAcknowledgment: append([]string(nil), admission.ProviderAcknowledgment...),
		CatalogDigest:          admission.CatalogDigest,
		PolicyDigest:           admission.PolicyDigest,
		PolicyPath:             admission.PolicyPath,
		RuntimeVersion:         admission.RuntimeVersion,
	}
}

// AdmitDependentDelegation is the configuration-sensitive prerequisite gate. It
// returns nil only when a live qualification receipt exists, records accepted
// output and a runtime identity, and its effective runner/model/effort match the
// requested identity exactly. Any missing or mismatched identity keeps dependent
// delegation closed. It is a pure predicate over owner state so orchestration,
// the CLI and review all share the same decision.
func AdmitDependentDelegation(receipt *QualificationReceipt, req DependentDelegationRequest) error {
	if receipt == nil {
		return NewValidationError("qualification", "dependent delegation is closed: no live qualification receipt")
	}
	if missing := receipt.MissingIdentities(); len(missing) > 0 {
		return NewValidationError("qualification", "dependent delegation is closed: qualification receipt missing "+strings.Join(missing, ", "))
	}
	if receipt.EffectiveRunner != strings.TrimSpace(req.Runner) {
		return NewValidationError("runner", fmt.Sprintf("dependent delegation is closed: receipt runner %q does not match requested %q", receipt.EffectiveRunner, req.Runner))
	}
	if receipt.EffectiveModel != strings.TrimSpace(req.Model) {
		return NewValidationError("model", fmt.Sprintf("dependent delegation is closed: receipt model %q does not match requested %q", receipt.EffectiveModel, req.Model))
	}
	if strings.TrimSpace(receipt.EffectiveEffort) != strings.TrimSpace(req.Effort) {
		return NewValidationError("effort", fmt.Sprintf("dependent delegation is closed: receipt effort %q does not match requested %q", receipt.EffectiveEffort, req.Effort))
	}
	return nil
}
