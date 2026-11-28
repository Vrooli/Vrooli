package aigateway

import (
	"encoding/json"
	"math"
	"strings"
)

// MarkAttemptFailed rewrites the execution evidence a provider returned when the
// engine could not publish its output. The gateway may have reported success,
// but a job that failed after the call has to record that outcome, otherwise the
// receipt says a result was produced that no caller can fetch.
func MarkAttemptFailed(meta map[string]string) map[string]string {
	raw := strings.TrimSpace(meta[MetaExecutionAttempt])
	if raw == "" {
		return meta
	}
	var attempt Attempt
	if err := json.Unmarshal([]byte(raw), &attempt); err != nil {
		return meta
	}
	attempt.Outcome = "failed"
	encoded, err := json.Marshal(attempt)
	if err != nil {
		return meta
	}
	copyMeta := make(map[string]string, len(meta))
	for key, value := range meta {
		copyMeta[key] = value
	}
	copyMeta[MetaExecutionAttempt] = string(encoded)
	return copyMeta
}

// EvidenceVariationFromMeta decodes one variation's execution evidence. An
// unrecognised outcome is recorded as unknown rather than trusted, and a cost is
// only reported as known when the gateway reported a finite, non-negative value
// for a call whose outcome is actually known.
func EvidenceVariationFromMeta(index int, meta map[string]string) (EvidenceVariation, bool) {
	raw := strings.TrimSpace(meta[MetaExecutionAttempt])
	if raw == "" {
		return EvidenceVariation{}, false
	}
	var attempt Attempt
	if err := json.Unmarshal([]byte(raw), &attempt); err != nil {
		return EvidenceVariation{}, false
	}
	switch attempt.Outcome {
	case "succeeded", "failed", "outcome_unknown", "not_started":
	default:
		attempt.Outcome = "outcome_unknown"
	}
	entry := EvidenceVariation{
		Index: index, Outcome: attempt.Outcome,
		ExecutionID: strings.TrimSpace(attempt.ExecutionID), Status: strings.TrimSpace(attempt.Status),
		CostState: "unknown",
	}
	if attempt.CostState == "known" && attempt.CostUSD != nil && !math.IsNaN(*attempt.CostUSD) && !math.IsInf(*attempt.CostUSD, 0) && *attempt.CostUSD >= 0 && attempt.Outcome != "outcome_unknown" {
		cost := *attempt.CostUSD
		entry.CostState = "known"
		entry.CostUSD = &cost
	}
	return entry, true
}
