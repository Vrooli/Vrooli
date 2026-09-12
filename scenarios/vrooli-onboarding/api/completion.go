package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	readinessdomain "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/readiness"
)

// completionBlocker names one reason configuration is not complete. It carries
// metadata only and never a credential value.
type completionBlocker = readinessdomain.CompletionBlocker

// completionAssessment separates the two questions the flow has always
// conflated: what stops configuration from being complete, and what is merely
// less than ideal. A blocker is a required item that is unresolved. A degraded
// gap is an optional item that is unresolved; it stops completion only until
// the operator acknowledges that exact set.
type completionAssessment struct {
	Blockers       []completionBlocker
	Degraded       []completionBlocker
	DegradedDigest string
}

const (
	provisionInWizardRemediation = "Provide this credential on the onboarding credentials step; the value goes straight to the credential authority."
	applyHostRemediation         = "Apply the selection in onboarding; run `vrooli setup --sudo-mode=ask` if the item needs host privilege."
	recoveryRemediation          = "Provide the missing required credential, then export a fresh recovery bundle."
	retryApplyRemediation        = "Resolve the reported host condition, then apply the selection again."
)

// assessCompletion decides what stops the flow from reporting completion.
//
// It reads the readiness verdict the operator sees, plus the apply run when one
// exists, so the answer cannot differ from what the wizard displays. Passing a
// nil run asks the same question before any apply has been started.
func assessCompletion(readiness readinessResponse, run *applyRun) completionAssessment {
	// Readiness may contribute blockers from diagnostics that do not have a
	// credential or host row of their own, such as an unresolved required
	// consumer binding. Preserve those blockers while deriving the normal
	// credential/host/recovery verdicts from the same response.
	assessment := completionAssessment{
		Blockers: append([]completionBlocker(nil), readiness.Blockers...),
		Degraded: []completionBlocker{},
	}
	contextualCredentialAddresses := contextualCredentialAddressSet(readiness.Credentials)
	credentialStatusPending := false
	for _, credential := range readiness.Credentials {
		// A derived or generated value is written by its declaring component.
		// The operator cannot supply it, so its absence is never a reason to
		// withhold completion from them.
		if credential.Status == "deferred" {
			// Optional credentials remain visible in the inventory, but their
			// provider is deliberately not contacted during the required-first
			// readiness pass. They must not become degraded work merely because
			// they were intentionally deferred.
			continue
		}
		if credential.Status == "pending" {
			credentialStatusPending = true
			continue
		}
		if credential.Status == "configured" && credentialEvidenceRequiresExercise(credential) && credential.EvidenceStatus != "verified" {
			blocker := completionBlocker{
				Kind:        "credential",
				Name:        credential.LogicalID + ":" + credential.Field + ":verification",
				Reason:      "the credential is stored but its owning provider has not supplied the evidence required for readiness",
				Remediation: "verify this credential through its owning provider, then retry readiness",
			}
			if credential.Required {
				assessment.Blockers = append(assessment.Blockers, blocker)
			} else {
				assessment.Degraded = append(assessment.Degraded, blocker)
			}
			continue
		}
		if credential.Status == "configured" || !operatorSuppliedCredential(credential) {
			continue
		}
		blocker := completionBlocker{
			Kind:        "credential",
			Name:        credential.LogicalID + ":" + credential.Field,
			Reason:      credentialGapReason(credential),
			Remediation: credentialGapRemediation(credential),
		}
		if credential.Required {
			assessment.Blockers = append(assessment.Blockers, blocker)
			continue
		}
		assessment.Degraded = append(assessment.Degraded, blocker)
	}
	if credentialStatusPending {
		assessment.Blockers = append(assessment.Blockers, completionBlocker{
			Kind:        "readiness",
			Name:        "credential-status",
			Reason:      "one or more credential status probes are still pending",
			Remediation: "retry readiness before applying the selection",
		})
	}
	for _, host := range readiness.Hosts {
		if host.Status == "pending" {
			assessment.Blockers = append(assessment.Blockers, completionBlocker{
				Kind:        "readiness",
				Name:        host.Name,
				Reason:      host.Kind + " status is still being checked",
				Remediation: firstNonEmptyString(host.Remediation, "retry readiness before applying the selection"),
			})
			continue
		}
		if host.Status != "missing" && host.Status != "unsupported" {
			continue
		}
		blocker := completionBlocker{
			Kind:        "host",
			Name:        host.Name,
			Reason:      host.Kind + " is " + host.Status + " on this host",
			Remediation: firstNonEmptyString(host.Remediation, applyHostRemediation),
		}
		if host.Required {
			assessment.Blockers = append(assessment.Blockers, blocker)
			continue
		}
		assessment.Degraded = append(assessment.Degraded, blocker)
	}
	// A required credential that is declared but absent is a recovery gap as
	// well as a credential gap. It is reported under both kinds because the two
	// answers have different owners: the credential step can fix the first, and
	// only a fresh recovery export closes the second.
	for _, address := range readiness.Recovery.RequiredAbsent {
		// Recovery diagnosis intentionally includes the global managed
		// population. Only an address in this contextual closure can block the
		// current onboarding decision; unrelated managed identities remain
		// visible in the doctor/recovery surface without becoming setup work.
		if _, contextual := contextualCredentialAddresses[address]; !contextual {
			continue
		}
		assessment.Blockers = append(assessment.Blockers, completionBlocker{
			Kind:        "recovery",
			Name:        address,
			Reason:      "a required credential is declared and absent, so recovery cannot cover it",
			Remediation: recoveryRemediation,
		})
	}
	if run != nil {
		for _, item := range run.Items {
			if item.Outcome == "applied" || item.Outcome == "already_satisfied" || item.Outcome == "not_applicable" {
				continue
			}
			assessment.Blockers = append(assessment.Blockers, completionBlocker{
				Kind:        "apply",
				Name:        item.Kind + ":" + item.Name,
				Reason:      "the apply item reported " + item.Outcome,
				Remediation: firstNonEmptyString(item.Remediation, retryApplyRemediation),
			})
		}
	}
	sortBlockers(assessment.Blockers)
	sortBlockers(assessment.Degraded)
	assessment.DegradedDigest = degradedDigest(assessment.Degraded)
	return assessment
}

// credentialEvidenceRequiresExercise interprets the small, provider-neutral
// policy vocabulary. A descriptor may still expose unverified storage state
// for advisory policies, but release-required policies cannot satisfy
// onboarding completion from storage presence alone.
func credentialEvidenceRequiresExercise(credential credentialReadiness) bool {
	switch strings.ToLower(strings.TrimSpace(credential.EvidencePolicy)) {
	case "required", "verify-required", "operation-required", "release-required", "always":
		return true
	default:
		return false
	}
}

// operatorSuppliedCredential reports whether a person has to provide this
// value. It mirrors credentialspec.Descriptor.OperatorSupplied for the
// readiness projection, which carries the provisioning kind as a plain string.
func operatorSuppliedCredential(credential credentialReadiness) bool {
	switch strings.TrimSpace(credential.Provisioning) {
	case "derived", "generated":
		return false
	default:
		return true
	}
}

func credentialGapReason(credential credentialReadiness) string {
	if credential.Status == "unsupported" {
		return "the credential backend could not answer for this address"
	}
	return "the credential is declared and not configured"
}

func credentialGapRemediation(credential credentialReadiness) string {
	switch credential.Status {
	case "unsupported":
		return "Retry credential verification; if the condition persists, run `vrooli credentials doctor` and resolve the reported provider condition."
	case credentialStatusPending:
		return "Retry readiness after the credential authority finishes checking this address."
	default:
		return provisionInWizardRemediation
	}
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func sortBlockers(items []completionBlocker) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Kind == items[j].Kind {
			return items[i].Name < items[j].Name
		}
		return items[i].Kind < items[j].Kind
	})
}

// degradedDigest names the exact set of degraded items an acknowledgement
// applies to. An acknowledgement carrying a different digest does not authorise
// completion, so accepting one gap can never silently authorise another.
func degradedDigest(items []completionBlocker) string {
	if len(items) == 0 {
		return ""
	}
	hash := sha256.New()
	for _, item := range items {
		_, _ = hash.Write([]byte(item.Kind + "|" + item.Name + "\n"))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func degradedAcknowledgementMatches(state OperatorState, digest string) bool {
	if digest == "" {
		return true
	}
	if state.Completion == nil || state.Completion.DegradedAcknowledgement == nil {
		return false
	}
	return state.Completion.DegradedAcknowledgement.ReadinessDigest == digest
}

// configurationMayComplete is the single predicate that decides whether the
// completion marker may be written.
//
// The predicate lives here, not at a Next button, because three surfaces can
// complete configuration — the UI, the CLI wizard, and a direct API caller —
// and only the marker write is common to all three.
func configurationMayComplete(assessment completionAssessment, state OperatorState) bool {
	if len(assessment.Blockers) > 0 {
		return false
	}
	return degradedAcknowledgementMatches(state, assessment.DegradedDigest)
}

type degradedAcknowledgementRequest struct {
	ReadinessDigest string `json:"readiness_digest"`
}

func acknowledgeDegradedReadiness(ctx context.Context, digest string) (readinessdomain.AcknowledgeResult, error) {
	// Acknowledgement is part of the completion contract, so it must evaluate
	// the same optional credential set that apply uses. The normal readiness
	// view may defer optional provider probes for fast rendering; accepting a
	// digest cannot use that cheaper view or it could reject a digest emitted by
	// the apply run as stale.
	readiness, err := buildReadinessResponseForApply(ctx)
	if err != nil {
		return readinessdomain.AcknowledgeResult{}, err
	}
	if readiness.DegradedDigest != digest {
		return readinessdomain.AcknowledgeResult{}, fmt.Errorf("the acknowledged degraded set is not the current one")
	}
	if _, err := operatorStateService().RecordDegradedAcknowledgement(ctx, digest, operatorStateNow()); err != nil {
		return readinessdomain.AcknowledgeResult{}, err
	}
	return readinessdomain.AcknowledgeResult{Status: "acknowledged", ReadinessDigest: digest, Degraded: readiness.Degraded}, nil
}

func (s *Server) handleV2DegradedAcknowledgement(w http.ResponseWriter, r *http.Request) {
	var request degradedAcknowledgementRequest
	if !decodeJSONBody(w, r, &request) {
		return
	}
	digest := strings.TrimSpace(request.ReadinessDigest)
	if digest == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "readiness_digest is required"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	readiness, err := buildReadinessResponseForApply(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	// The digest must name the gap that exists now. Accepting a stale digest
	// would let an acknowledgement recorded for one gap authorise completion
	// over a different one.
	if readiness.DegradedDigest != digest {
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":                   "the acknowledged degraded set is not the current one",
			"current_degraded_digest": readiness.DegradedDigest,
			"degraded":                readiness.Degraded,
		})
		return
	}
	if _, err := operatorStateService().RecordDegradedAcknowledgement(ctx, digest, operatorStateNow()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"status":           "acknowledged",
		"readiness_digest": digest,
		"degraded":         readiness.Degraded,
	})
}
