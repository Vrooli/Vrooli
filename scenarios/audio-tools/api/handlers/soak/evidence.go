//go:build audio_lab

package soak

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"

	"audio-tools/internal/acceptance"
	"audio-tools/internal/conformance"
	intexp "audio-tools/internal/experiment"
	trustfloor "audio-tools/internal/qualification"
	"audio-tools/internal/soak"
)

const requiredRealtimeTrustRuns = 3

func persistQualificationEvidence(ctx context.Context, service *intexp.Service, opt soak.Options, result soak.Result, runErr error, artifactRef string) error {
	machine := baseMachineEvidence(opt, result)
	deviceProfile := "D2"
	if result.Run.Lane == "realtime" {
		deviceProfile = "D1"
	}
	claims, err := acceptanceClaimsForRun(ctx, service, opt, result.Run)
	if err != nil {
		return fmt.Errorf("resolve realtime acceptance cohort: %w", err)
	}
	claims = addMobileRecoveryClaim(claims, opt, result, runErr)
	if err := addAcceptanceEvidence(machine, result.Run, deviceProfile, claims); err != nil {
		return err
	}
	addMobileEmulationEvidence(machine, opt, result)
	if err := persistBrowserEvidence(ctx, service, opt, result, runErr, artifactRef, machine, claims); err != nil {
		return err
	}
	if err := persistFaultEvidence(ctx, service, opt, result, runErr, artifactRef); err != nil {
		return err
	}
	if result.Run.Lane != "realtime" {
		return nil
	}
	return persistDeviceEvidence(ctx, service, opt, result, runErr, artifactRef, machine)
}

func baseMachineEvidence(opt soak.Options, result soak.Result) map[string]any {
	product := "dictation-studio"
	if strings.EqualFold(strings.TrimSpace(opt.Surface), "swarm-manager") {
		product = "swarm-manager-quick-capture"
	}
	machine := map[string]any{
		"os":                   runtime.GOOS,
		"arch":                 runtime.GOARCH,
		"run_id":               result.Run.RunID,
		"lane":                 result.Run.Lane,
		"shape":                result.Run.Shape,
		"surface":              opt.Surface,
		"product":              product,
		"browser":              "playwright-driver",
		"browser_product_path": true,
	}
	return machine
}

func addMobileRecoveryClaim(claims map[string]any, opt soak.Options, result soak.Result, runErr error) map[string]any {
	if opt.MobileRecovery && result.MobileRecoveryPassed && runErr == nil {
		if claims == nil {
			claims = map[string]any{}
		}
		claims["mobile-recovery"] = map[string]any{"in_band": true, "negative_controls_passed": true}
	}
	return claims
}

func addAcceptanceEvidence(machine map[string]any, run conformance.Run, deviceProfile string, claims map[string]any) error {
	if claims != nil {
		acceptanceMachine, acceptanceErr := acceptance.MachineJSON(run, deviceProfile, acceptance.Cohort, acceptance.Policy, claims)
		if acceptanceErr != nil {
			return fmt.Errorf("encode acceptance evidence: %w", acceptanceErr)
		}
		var acceptance map[string]any
		if err := json.Unmarshal(acceptanceMachine, &acceptance); err != nil {
			return fmt.Errorf("decode acceptance evidence: %w", err)
		}
		for key, value := range acceptance {
			machine[key] = value
		}
	}
	return nil
}

func addMobileEmulationEvidence(machine map[string]any, opt soak.Options, result soak.Result) {
	if opt.MobileRecovery {
		machine["mobile_emulation"] = map[string]any{"viewport_width": 390, "viewport_height": 844, "native": false, "diagnostic_export": result.MobileRecoveryPassed}
	}
}

func persistBrowserEvidence(ctx context.Context, service *intexp.Service, opt soak.Options, result soak.Result, runErr error, artifactRef string, machine map[string]any, claims map[string]any) error {
	machineJSON, err := json.Marshal(machine)
	if err != nil {
		return fmt.Errorf("encode machine evidence: %w", err)
	}
	// Browser-product-path evidence is target-aware: its Passed bit must match
	// the independent acceptance claims carried in machine_json. Recognition
	// quality is a separate conformance assertion and must not erase proof that
	// the transport/provider obligations were exercised and passed.
	passed := acceptanceClaimsPassed(claims, runErr)
	if _, err := service.RecordQualificationEvidence(ctx, intexp.QualificationEvidence{
		EngineID: opt.EngineID, ModelID: opt.ModelID, Strategy: opt.Strategy, PolicyProfile: opt.Policy,
		ProviderTier: result.Run.Cell.ProviderTier,
		Kind:         trustfloor.QualificationBrowserProductPath, Passed: passed, ArtifactRef: artifactRef,
		Notes:       "automated BAS browser product-path qualification; Passed is scoped to the target-aware acceptance claims in machine_json",
		MachineJSON: machineJSON,
	}); err != nil {
		return fmt.Errorf("record browser_product_path: %w", err)
	}
	return nil
}

func persistFaultEvidence(ctx context.Context, service *intexp.Service, opt soak.Options, result soak.Result, runErr error, artifactRef string) error {
	if fault := strings.TrimSpace(opt.Fault); fault != "" {
		// The fault matrix is a provider-neutral promotion gate. Keep one
		// durable row per injected profile so the rubric can distinguish an
		// observed, passing fault from a browser-path run that merely happened
		// to return an artifact. The run document is the source of truth for
		// the assertion verdict; machine JSON contains metadata only.
		verdict := result.Run.Evaluate()
		faultMachine, marshalErr := json.Marshal(map[string]any{
			"run_id":              result.Run.RunID,
			"lane":                result.Run.Lane,
			"shape":               result.Run.Shape,
			"fault_profile":       fault,
			"fault_observed":      len(result.Run.Assertions) > 0,
			"fault_run_qualified": verdict.Qualified && runErr == nil,
		})
		if marshalErr != nil {
			return fmt.Errorf("encode fault evidence: %w", marshalErr)
		}
		if _, err := service.RecordQualificationEvidence(ctx, intexp.QualificationEvidence{
			EngineID: opt.EngineID, ModelID: opt.ModelID, Strategy: opt.Strategy, PolicyProfile: opt.Policy,
			Kind: trustfloor.QualificationFault, FaultProfile: fault,
			Passed: qualificationRunPassed(result.Run, runErr), ArtifactRef: artifactRef,
			Notes:       "automated BAS deterministic fault-matrix qualification; pass requires every fault assertion in the conformance run",
			MachineJSON: faultMachine,
		}); err != nil {
			return fmt.Errorf("record fault %q: %w", fault, err)
		}
	}
	return nil
}

func persistDeviceEvidence(ctx context.Context, service *intexp.Service, opt soak.Options, result soak.Result, runErr error, artifactRef string, machine map[string]any) error {
	device := result.AudioDeviceEvidence
	devicePassed := deviceEvidencePassed(result.Run, runErr, device)
	deviceMachine := deviceMachineEvidence(machine, device)
	machineJSON, err := json.Marshal(deviceMachine)
	if err != nil {
		return fmt.Errorf("encode device evidence: %w", err)
	}
	if _, err := service.RecordQualificationEvidence(ctx, intexp.QualificationEvidence{
		EngineID: opt.EngineID, ModelID: opt.ModelID, Strategy: opt.Strategy, PolicyProfile: opt.Policy,
		Kind: trustfloor.QualificationDevice, Passed: devicePassed, ArtifactRef: artifactRef,
		Notes: deviceEvidenceNotes(device), MachineJSON: machineJSON,
	}); err != nil {
		return fmt.Errorf("record device: %w", err)
	}
	return nil
}

func deviceEvidencePassed(run conformance.Run, runErr error, device *soak.AudioDeviceEvidence) bool {
	return qualificationRunPassed(run, runErr) && device != nil && device.Enumerated &&
		strings.Contains(device.SelectedLabel, "Vrooli_Qualification_Microphone") && device.SampleRate > 0 && device.ChannelCount > 0
}

func deviceMachineEvidence(machine map[string]any, device *soak.AudioDeviceEvidence) map[string]any {
	if device != nil {
		machine["audio_device_evidence"] = device
		machine["device_path"] = "pipewire-host-device"
	}
	// Device evidence is a separate qualification kind. Do not copy the
	// browser acceptance envelope into it: a failed device row would otherwise
	// look like a contradictory negative acceptance receipt to the board.
	deviceMachine := make(map[string]any, len(machine))
	for key, value := range machine {
		deviceMachine[key] = value
	}
	for _, key := range []string{"target_revision", "method", "cohort", "build", "device_profile", "policy_revision", "corpus_revision", "acceptance"} {
		delete(deviceMachine, key)
	}
	return deviceMachine
}

func deviceEvidenceNotes(device *soak.AudioDeviceEvidence) string {
	notes := "automated OS capture-device and browser audio-stack evidence; this proves the PipeWire device path, not any particular microphone's analog front end"
	if device == nil {
		notes += "; BAS returned no device evidence"
	}
	return notes
}

func qualificationRunPassed(run conformance.Run, runErr error) bool {
	return runErr == nil && run.Evaluate().Qualified
}

func acceptanceClaimsForRun(ctx context.Context, service *intexp.Service, opt soak.Options, run conformance.Run) (map[string]any, error) {
	claims := acceptance.ClaimsForRealtimeRun(run)
	if len(claims) == 0 {
		return nil, nil
	}
	history, err := qualificationHistory(ctx, service, opt)
	if err != nil {
		return nil, err
	}
	priorTrustRuns := priorPassingTrustRuns(history)
	parityHistory, err := parityQualificationHistory(ctx, service, opt)
	if err != nil {
		return nil, err
	}
	if providerParityEligible(run, "D1", parityHistory) {
		claims["provider-parity"] = map[string]any{"in_band": true, "negative_controls_passed": true, "attempts": run.Turns}
	}
	return claimsAfterTrustCohort(claims, priorTrustRuns), nil
}

func qualificationHistory(ctx context.Context, service *intexp.Service, opt soak.Options) ([]intexp.QualificationEvidence, error) {
	return service.ListQualificationEvidence(ctx, intexp.QualificationEvidenceFilter{
		EngineID: opt.EngineID, ModelID: opt.ModelID, Strategy: opt.Strategy, PolicyProfile: opt.Policy,
	})
}

func parityQualificationHistory(ctx context.Context, service *intexp.Service, opt soak.Options) ([]intexp.QualificationEvidence, error) {
	// Corpus-quality producers intentionally have no product policy profile;
	// they qualify the held-out recognition cell independently of the browser's
	// default policy. Keep the exact cell filter for the repeated product-trust
	// gate, but use an engine/model-only view for the composite parity gate.
	return service.ListQualificationEvidence(ctx, intexp.QualificationEvidenceFilter{
		EngineID: opt.EngineID, ModelID: opt.ModelID,
	})
}

func priorPassingTrustRuns(history []intexp.QualificationEvidence) int {
	priorTrustRuns := 0
	seenRuns := make(map[string]struct{})
	for _, item := range history {
		runID, ok := eligiblePriorTrustRun(item)
		if !ok {
			continue
		}
		if _, duplicate := seenRuns[runID]; duplicate {
			continue
		}
		seenRuns[runID] = struct{}{}
		priorTrustRuns++
	}
	return priorTrustRuns
}

func eligiblePriorTrustRun(item intexp.QualificationEvidence) (string, bool) {
	if !item.Passed || item.Kind != trustfloor.QualificationBrowserProductPath {
		return "", false
	}
	var envelope map[string]any
	if err := json.Unmarshal(item.MachineJSON, &envelope); err != nil || !acceptanceEnvelopeMatches(envelope) {
		return "", false
	}
	runID, _ := envelope["run_id"].(string)
	if strings.TrimSpace(runID) == "" {
		return "", false
	}
	acceptanceMap, _ := envelope["acceptance"].(map[string]any)
	// Older receipts may have been persisted before the trust claim was
	// eligible. Count them when their owned run artifact independently proves
	// the transport cohort; otherwise repetition can never reach its threshold.
	if !claimInBand(acceptanceMap["dictation-trust"]) || !claimInBand(acceptanceMap["provider-evidence"]) {
		if !realtimeTransportArtifactPassed(item.ArtifactRef) {
			return "", false
		}
	}
	return runID, true
}

func acceptanceEnvelopeMatches(envelope map[string]any) bool {
	return envelope["target_revision"] == acceptance.TargetRevision && envelope["method"] == acceptance.Method &&
		envelope["cohort"] == acceptance.Cohort && envelope["policy_revision"] == acceptance.Policy
}

// providerParityEligible is deliberately a composite owner gate. A stable
// engine must have a complete long-form product-path run, paced interactive
// evidence, a passing speaker-policy receipt, and a passing held-out corpus
// receipt for the same provider cell. Inventory availability and a single
// successful browser click are not parity evidence.
func providerParityEligible(run conformance.Run, deviceProfile string, history []intexp.QualificationEvidence) bool {
	if !parityRunEligible(run) || !providerParityAssertionsPassed(run) {
		return false
	}
	return providerParityReceiptsMatch(run, deviceProfile, history)
}

func parityRunEligible(run conformance.Run) bool {
	return run.Lane == conformance.LaneRealtime && run.DurationMs >= 60*60*1000 && run.Turns >= 30
}

func providerParityAssertionsPassed(run conformance.Run) bool {
	return conformance.PassedAssertions(run,
		"interval_accounting_exactly_once", "capture_signal_observed",
		"browser_retention_bounded", "server_retention_bounded",
		"capture_callback_processing_budget", "zero_duplicate_committed_segments",
		"zero_silent_terminal_outcomes", "provider_cell_identity",
		"wire_rate_within_budget", "capture_ready_latency", "first_partial_latency_stable",
		"committed_text_lag_stable", "continuous_interim_text_visible",
		"word_error_rate_stable")
}

func providerParityReceiptsMatch(run conformance.Run, deviceProfile string, history []intexp.QualificationEvidence) bool {
	speakerPolicy, corpusQuality := false, false
	for _, item := range history {
		acceptanceMap, ok := providerParityAcceptance(item, run, deviceProfile)
		if !ok {
			continue
		}
		if claimInBand(acceptanceMap["speaker-policy"]) {
			speakerPolicy = true
		}
		if claimInBand(acceptanceMap["corpus-quality"]) {
			corpusQuality = true
		}
	}
	return speakerPolicy && corpusQuality
}

func providerParityAcceptance(item intexp.QualificationEvidence, run conformance.Run, deviceProfile string) (map[string]any, bool) {
	if !item.Passed || item.EngineID != run.Cell.EngineID || item.ModelID != run.Cell.ModelID ||
		(item.Strategy != "" && item.Strategy != run.Cell.Strategy) {
		return nil, false
	}
	var envelope map[string]any
	if err := json.Unmarshal(item.MachineJSON, &envelope); err != nil || !parityEnvelopeMatches(envelope, run, deviceProfile) {
		return nil, false
	}
	acceptanceMap, ok := envelope["acceptance"].(map[string]any)
	return acceptanceMap, ok
}

func parityEnvelopeMatches(envelope map[string]any, run conformance.Run, deviceProfile string) bool {
	if !acceptanceEnvelopeMatches(envelope) || envelope["corpus_revision"] != acceptance.Cohort ||
		envelope["build"] != run.Code.Server || envelope["device_profile"] != deviceProfile {
		return false
	}
	providerCell, _ := envelope["provider_cell"].(map[string]any)
	return len(providerCell) > 0 && providerCell["engine_id"] == run.Cell.EngineID &&
		providerCell["model_id"] == run.Cell.ModelID && providerCell["strategy"] == run.Cell.Strategy
}

func realtimeTransportArtifactPassed(artifactRef string) bool {
	if strings.TrimSpace(artifactRef) == "" {
		return false
	}
	// #nosec G304 -- artifactRef is returned by PersistEvidence immediately before this verifier and is not request-controlled.
	payload, err := os.ReadFile(artifactRef)
	if err != nil {
		return false
	}
	var run conformance.Run
	if err := json.Unmarshal(payload, &run); err != nil {
		return false
	}
	if run.Lane != conformance.LaneRealtime || run.DurationMs < 60*60*1000 || strings.TrimSpace(run.Fault) != "" {
		return false
	}
	return conformance.PassedAssertions(run,
		"interval_accounting_exactly_once", "capture_signal_observed",
		"browser_retention_bounded", "server_retention_bounded",
		"capture_callback_processing_budget", "zero_duplicate_committed_segments",
		"zero_silent_terminal_outcomes", "provider_cell_identity",
		"wire_rate_within_budget",
	)
}

func claimsAfterTrustCohort(claims map[string]any, priorTrustRuns int) map[string]any {
	if len(claims) == 0 {
		return nil
	}
	out := make(map[string]any, len(claims))
	for name, claim := range claims {
		if (name == "dictation-trust" || name == "provider-evidence") && priorTrustRuns+1 < requiredRealtimeTrustRuns {
			continue
		}
		out[name] = claim
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func claimInBand(raw any) bool {
	claim, ok := raw.(map[string]any)
	return ok && claim["in_band"] == true && claim["negative_controls_passed"] == true
}

func acceptanceClaimsPassed(claims map[string]any, runErr error) bool {
	if runErr != nil {
		return false
	}
	if len(claims) == 0 {
		return false
	}
	for _, raw := range claims {
		if !claimInBand(raw) {
			return false
		}
	}
	return true
}
