package acceptance

import (
	"encoding/json"
	"fmt"
	"strings"

	"audio-tools/internal/conformance"
)

const (
	TargetRevision = "portable-voice-v1"
	Method         = "audio-tools-acceptance-v1"
	Cohort         = "voice-pilot-en-v1"
	Policy         = "policy-v1"
)

// MachineJSON builds the owner-backed, transcript-free envelope joined by
// audio-tools.setpoint-read. Claims are derived from conformance evidence.
func MachineJSON(run conformance.Run, deviceProfile, corpusRevision, policyRevision string, claims map[string]any) ([]byte, error) {
	if strings.TrimSpace(run.Code.Server) == "" {
		return nil, fmt.Errorf("acceptance receipt requires the server build fingerprint")
	}
	if strings.TrimSpace(run.Cell.EngineID) == "" || strings.TrimSpace(run.Cell.ModelID) == "" || strings.TrimSpace(run.Cell.Strategy) == "" {
		return nil, fmt.Errorf("acceptance receipt requires a complete provider cell")
	}
	if strings.TrimSpace(deviceProfile) == "" || strings.TrimSpace(corpusRevision) == "" || strings.TrimSpace(policyRevision) == "" {
		return nil, fmt.Errorf("acceptance receipt requires device, corpus, and policy identities")
	}
	if claims == nil {
		claims = map[string]any{}
	}
	envelope := map[string]any{
		"target_revision": TargetRevision, "method": Method, "cohort": Cohort,
		"build": run.Code.Server, "device_profile": deviceProfile,
		"policy_revision": policyRevision, "corpus_revision": corpusRevision,
		"run_id": run.RunID, "lane": run.Lane, "acceptance": claims,
	}
	return json.Marshal(envelope)
}

// MachineJSONForCell builds the same target envelope for a non-browser
// qualification cell. The claim is still joined only when the persisted
// artifact is passed, so a failed quality receipt remains visible negative
// evidence rather than becoming acceptance through its payload alone.
func MachineJSONForCell(build, deviceProfile, corpusRevision, policyRevision, engineID, modelID, strategy string, claims map[string]any) ([]byte, error) {
	if strings.TrimSpace(build) == "" || strings.TrimSpace(engineID) == "" || strings.TrimSpace(modelID) == "" || strings.TrimSpace(strategy) == "" {
		return nil, fmt.Errorf("cell acceptance receipt requires build and complete provider identity")
	}
	if strings.TrimSpace(deviceProfile) == "" || strings.TrimSpace(corpusRevision) == "" || strings.TrimSpace(policyRevision) == "" {
		return nil, fmt.Errorf("cell acceptance receipt requires device, corpus, and policy identities")
	}
	if claims == nil {
		claims = map[string]any{}
	}
	return json.Marshal(map[string]any{
		"target_revision": TargetRevision, "method": Method, "cohort": Cohort,
		"build": build, "device_profile": deviceProfile,
		"policy_revision": policyRevision, "corpus_revision": corpusRevision,
		"provider_cell": map[string]string{"engine_id": engineID, "model_id": modelID, "strategy": strategy},
		"acceptance":    claims,
	})
}

// ClaimsForRealtimeRun returns independent claims supported by a complete
// real-time 60-minute conformance run. Recognition quality is deliberately
// not part of the transport/provider-evidence gate: the evidence inventory
// treats recognition mistakes separately from interval loss and recovery.
// Accelerated or partial runs earn no claims.
func ClaimsForRealtimeRun(run conformance.Run) map[string]any {
	if run.Lane != conformance.LaneRealtime || run.DurationMs < 60*60*1000 || strings.TrimSpace(run.Fault) != "" {
		return nil
	}
	claims := map[string]any{}
	transportAssertions := []string{
		"interval_accounting_exactly_once", "capture_signal_observed",
		"browser_retention_bounded", "server_retention_bounded",
		"per_frame_cost_constant", "zero_duplicate_committed_segments",
		"zero_silent_terminal_outcomes", "provider_cell_identity",
		"wire_rate_within_budget",
	}
	if passedAssertions(run, transportAssertions...) {
		claims["dictation-trust"] = map[string]any{"in_band": true, "negative_controls_passed": true}
		claims["provider-evidence"] = map[string]any{"in_band": true, "negative_controls_passed": true}
	}
	// The realtime conformance assertions compute timing over observed turns,
	// but the adopted latency gate also requires at least 30 paced attempts for
	// each route/profile cohort. Do not let a long single-turn soak claim the
	// latency row.
	if run.Turns >= 30 && passedAssertions(run, "first_partial_latency_stable", "committed_text_lag_stable", "continuous_interim_text_visible") {
		claims["interactive-latency"] = map[string]any{"in_band": true, "negative_controls_passed": true, "attempts": run.Turns}
	}
	if len(claims) == 0 {
		return nil
	}
	return claims
}

func passedAssertions(run conformance.Run, names ...string) bool {
	byName := make(map[string]conformance.Assertion, len(run.Assertions))
	for _, assertion := range run.Assertions {
		byName[assertion.Name] = assertion
	}
	for _, name := range names {
		assertion, ok := byName[name]
		if !ok || assertion.Outcome != conformance.OutcomePassed {
			return false
		}
	}
	return true
}
