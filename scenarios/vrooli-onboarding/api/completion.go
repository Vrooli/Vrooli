package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sort"
	"strings"
	"time"
)

// completionBlocker names one reason configuration is not complete. It carries
// metadata only and never a credential value.
type completionBlocker struct {
	// Kind is one of credential, host, recovery, or apply.
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	Reason      string `json:"reason"`
	Remediation string `json:"remediation"`
}

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
	assessment := completionAssessment{Blockers: []completionBlocker{}, Degraded: []completionBlocker{}}
	for _, credential := range readiness.Credentials {
		// A derived or generated value is written by its declaring component.
		// The operator cannot supply it, so its absence is never a reason to
		// withhold completion from them.
		if credential.Status == "configured" || !operatorSuppliedCredential(credential) {
			continue
		}
		blocker := completionBlocker{
			Kind:        "credential",
			Name:        credential.LogicalID + ":" + credential.Field,
			Reason:      credentialGapReason(credential),
			Remediation: provisionInWizardRemediation,
		}
		if credential.Required {
			assessment.Blockers = append(assessment.Blockers, blocker)
			continue
		}
		assessment.Degraded = append(assessment.Degraded, blocker)
	}
	for _, host := range readiness.Hosts {
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
	readiness, err := buildReadinessResponse(ctx)
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
