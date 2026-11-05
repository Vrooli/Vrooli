// Package soak exposes the leased, out-of-band browser product-path driver.
// The endpoint returns exactly one conformance.Run document; it does not
// persist audio, transcripts, screenshots, or a second evidence format.
package soak

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"audio-tools/internal/acceptance"
	"audio-tools/internal/conformance"
	intexp "audio-tools/internal/experiment"
	"audio-tools/internal/modulekit"
	trustfloor "audio-tools/internal/qualification"
	"audio-tools/internal/soak"
	"audio-tools/internal/stt/session"

	"github.com/gorilla/mux"
)

type Deps struct {
	Ledgers             *session.Registry
	Experiments         *intexp.Service
	TestIsolationActive func() bool
}

const requiredRealtimeTrustRuns = 3

func Module(d Deps) modulekit.Module {
	return modulekit.Module{
		Name: "soak",
		Mount: func(r *mux.Router) {
			r.HandleFunc("/api/v1/validation/soak", func(w http.ResponseWriter, r *http.Request) {
				serve(w, r, d)
			}).Methods(http.MethodPost)
			r.HandleFunc("/api/v1/validation/soak/virtual-corpus", serveVirtualCorpus).Methods(http.MethodGet)
		},
		Endpoints: Endpoints,
	}
}

// serveVirtualCorpus is intentionally a fixed, test-only corpus surface. The
// browser accelerated lane must obtain its samples through an explicit
// qualification route; it must never accept an arbitrary local path from a
// page query string. The fixture is the same canonical speech sample used by
// the realtime BAS lane, and its reference is carried as immutable metadata
// for the independent quality assertion.
func serveVirtualCorpus(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("clip") != "quick-brown-fox" {
		writeError(w, http.StatusNotFound, "unknown virtual corpus clip")
		return
	}
	roots := make([]string, 0, 6)
	if root := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); root != "" {
		roots = append(roots, filepath.Join(root, "scenarios", "audio-tools"))
	}
	for _, name := range []string{"SCENARIO_ROOT", "BUNDLE_ROOT"} {
		if root := strings.TrimSpace(os.Getenv(name)); root != "" {
			roots = append(roots, root)
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		for dir := cwd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
			roots = append(roots, dir)
		}
	}
	candidates := make([]string, 0, len(roots))
	for _, root := range roots {
		candidates = append(candidates, filepath.Join(root, "bas", "fixtures", "dictation-reference.wav"))
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		w.Header().Set("Content-Type", "audio/wav")
		w.Header().Set("X-Vrooli-Corpus-Id", "quick-brown-fox")
		w.Header().Set("X-Vrooli-Corpus-Reference", "the quick brown fox jumps.")
		http.ServeFile(w, r, path)
		return
	}
	writeError(w, http.StatusNotFound, "virtual corpus fixture is unavailable")
}

func serve(w http.ResponseWriter, r *http.Request, d Deps) {
	var opt soak.Options
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10))
	if err := decoder.Decode(&opt); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("decode soak options: %v", err))
		return
	}
	// A normal product-path run may be observed without a temporary database
	// lease. Deterministic fault injection remains double-gated by the same
	// test-genie isolation seam used by the WebSocket transport. The explicit
	// harness gate exists for the standalone accelerated matrix; it is never
	// enabled by a normal deployment.
	faultHarnessEnabled := strings.EqualFold(strings.TrimSpace(os.Getenv("VROOLI_AUDIO_SOAK_FAULTS")), "1")
	if strings.TrimSpace(opt.Fault) != "" && (d.TestIsolationActive == nil || !d.TestIsolationActive()) && !faultHarnessEnabled {
		writeError(w, http.StatusPreconditionFailed, "faulted soak runs require an active test-genie isolation lease")
		return
	}
	// The qualification is server-owned. BAS owns admission and capacity for
	// isolated browser sessions; unrelated foreign sessions must not serialize
	// this scenario's work. A CLI disconnect, PTY timeout, or orchestration
	// cancellation must not close the browser lease halfway through a valid
	// long-form capture. RunTimeout still bounds the detached work, and scenario
	// stop remains the operator-level abort mechanism.
	runCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), soak.RunTimeout(opt))
	defer cancel()
	result, runErr := soak.RunWithEvidence(runCtx, opt, d.Ledgers)
	artifactRef, persistErr := soak.PersistEvidence(result.Run)
	if persistErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("persist soak evidence: %v", persistErr))
		return
	}
	if d.Experiments == nil {
		writeError(w, http.StatusFailedDependency, "persist soak qualification evidence: experiment service is unavailable")
		return
	}
	if err := persistQualificationEvidence(context.WithoutCancel(r.Context()), d.Experiments, opt, result, runErr, artifactRef); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("persist soak qualification evidence: %v", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := result.Run.WriteJSON(w); err != nil {
		return
	}
	if runErr != nil {
		return
	}
}

func persistQualificationEvidence(ctx context.Context, service *intexp.Service, opt soak.Options, result soak.Result, runErr error, artifactRef string) error {
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
	// The generic browser artifact remains useful for diagnostics. Add the
	// target-aware envelope only when the run itself proves the narrow claim;
	// this keeps the board join owner-backed without promoting virtual replay,
	// short runs, or failed conformance documents.
	deviceProfile := "D2"
	if result.Run.Lane == "realtime" {
		deviceProfile = "D1"
	}
	claims, claimsErr := acceptanceClaimsForRun(ctx, service, opt, result.Run)
	if claimsErr != nil {
		return fmt.Errorf("resolve realtime acceptance cohort: %w", claimsErr)
	}
	if opt.MobileRecovery && result.MobileRecoveryPassed && runErr == nil {
		if claims == nil {
			claims = map[string]any{}
		}
		claims["mobile-recovery"] = map[string]any{"in_band": true, "negative_controls_passed": true}
	}
	if claims != nil {
		acceptanceMachine, acceptanceErr := acceptance.MachineJSON(result.Run, deviceProfile, acceptance.Cohort, acceptance.Policy, claims)
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
	if opt.MobileRecovery {
		machine["mobile_emulation"] = map[string]any{"viewport_width": 390, "viewport_height": 844, "native": false, "diagnostic_export": result.MobileRecoveryPassed}
	}
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
		Kind: trustfloor.QualificationBrowserProductPath, Passed: passed, ArtifactRef: artifactRef,
		Notes:       "automated BAS browser product-path qualification; Passed is scoped to the target-aware acceptance claims in machine_json",
		MachineJSON: machineJSON,
	}); err != nil {
		return fmt.Errorf("record browser_product_path: %w", err)
	}
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
	if result.Run.Lane != "realtime" {
		return nil
	}
	device := result.AudioDeviceEvidence
	devicePassed := qualificationRunPassed(result.Run, runErr) && device != nil && device.Enumerated && strings.Contains(device.SelectedLabel, "Vrooli_Qualification_Microphone") && device.SampleRate > 0 && device.ChannelCount > 0
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
	machineJSON, err = json.Marshal(deviceMachine)
	if err != nil {
		return fmt.Errorf("encode device evidence: %w", err)
	}
	notes := "automated OS capture-device and browser audio-stack evidence; this proves the PipeWire device path, not any particular microphone's analog front end"
	if device == nil {
		notes += "; BAS returned no device evidence"
	}
	if _, err := service.RecordQualificationEvidence(ctx, intexp.QualificationEvidence{
		EngineID: opt.EngineID, ModelID: opt.ModelID, Strategy: opt.Strategy, PolicyProfile: opt.Policy,
		Kind: trustfloor.QualificationDevice, Passed: devicePassed, ArtifactRef: artifactRef,
		Notes: notes, MachineJSON: machineJSON,
	}); err != nil {
		return fmt.Errorf("record device: %w", err)
	}
	return nil
}

func qualificationRunPassed(run conformance.Run, runErr error) bool {
	return runErr == nil && run.Evaluate().Qualified
}

func acceptanceClaimsForRun(ctx context.Context, service *intexp.Service, opt soak.Options, run conformance.Run) (map[string]any, error) {
	claims := acceptance.ClaimsForRealtimeRun(run)
	if len(claims) == 0 {
		return nil, nil
	}
	filter := intexp.QualificationEvidenceFilter{EngineID: opt.EngineID, ModelID: opt.ModelID, Strategy: opt.Strategy, PolicyProfile: opt.Policy}
	history, err := service.ListQualificationEvidence(ctx, filter)
	if err != nil {
		return nil, err
	}
	// Corpus-quality producers intentionally have no product policy profile;
	// they qualify the held-out recognition cell independently of the browser's
	// default policy. Keep the exact cell filter for the repeated product-trust
	// gate, but use an engine/model-only view for the composite parity gate.
	parityHistory, err := service.ListQualificationEvidence(ctx, intexp.QualificationEvidenceFilter{
		EngineID: opt.EngineID, ModelID: opt.ModelID,
	})
	if err != nil {
		return nil, err
	}
	priorTrustRuns := 0
	seenRuns := make(map[string]struct{})
	for _, item := range history {
		if !item.Passed || item.Kind != trustfloor.QualificationBrowserProductPath {
			continue
		}
		var envelope map[string]any
		if err := json.Unmarshal(item.MachineJSON, &envelope); err != nil {
			continue
		}
		if envelope["target_revision"] != acceptance.TargetRevision || envelope["method"] != acceptance.Method ||
			envelope["cohort"] != acceptance.Cohort || envelope["policy_revision"] != acceptance.Policy {
			continue
		}
		runID, _ := envelope["run_id"].(string)
		if strings.TrimSpace(runID) == "" {
			continue
		}
		acceptanceMap, _ := envelope["acceptance"].(map[string]any)
		// Older receipts may have been persisted before the trust claim was
		// eligible. Count them when their owned run artifact independently
		// proves the transport cohort; otherwise the claim gate deadlocks at
		// the first receipt and can never reach the required repetition count.
		if !claimInBand(acceptanceMap["dictation-trust"]) || !claimInBand(acceptanceMap["provider-evidence"]) {
			if !realtimeTransportArtifactPassed(item.ArtifactRef) {
				continue
			}
		}
		if _, duplicate := seenRuns[runID]; duplicate {
			continue
		}
		seenRuns[runID] = struct{}{}
		priorTrustRuns++
	}
	if providerParityEligible(run, parityHistory) {
		claims["provider-parity"] = map[string]any{"in_band": true, "negative_controls_passed": true, "attempts": run.Turns}
	}
	return claimsAfterTrustCohort(claims, priorTrustRuns), nil
}

// providerParityEligible is deliberately a composite owner gate. A stable
// engine must have a complete long-form product-path run, paced interactive
// evidence, a passing speaker-policy receipt, and a passing held-out corpus
// receipt for the same provider cell. Inventory availability and a single
// successful browser click are not parity evidence.
func providerParityEligible(run conformance.Run, history []intexp.QualificationEvidence) bool {
	if run.Lane != conformance.LaneRealtime || run.DurationMs < 60*60*1000 || run.Turns < 30 {
		return false
	}
	if !transportAssertionsPassed(run,
		"interval_accounting_exactly_once", "capture_signal_observed",
		"browser_retention_bounded", "server_retention_bounded",
		"per_frame_cost_constant", "zero_duplicate_committed_segments",
		"zero_silent_terminal_outcomes", "provider_cell_identity",
		"wire_rate_within_budget", "first_partial_latency_stable",
		"committed_text_lag_stable", "continuous_interim_text_visible",
		"word_error_rate_stable") {
		return false
	}
	speakerPolicy, corpusQuality := false, false
	for _, item := range history {
		if !item.Passed || item.EngineID != run.Cell.EngineID || item.ModelID != run.Cell.ModelID {
			continue
		}
		var envelope map[string]any
		if err := json.Unmarshal(item.MachineJSON, &envelope); err != nil {
			continue
		}
		if envelope["target_revision"] != acceptance.TargetRevision || envelope["method"] != acceptance.Method ||
			envelope["cohort"] != acceptance.Cohort || envelope["policy_revision"] != acceptance.Policy ||
			envelope["corpus_revision"] != acceptance.Cohort {
			continue
		}
		providerCell, _ := envelope["provider_cell"].(map[string]any)
		if len(providerCell) > 0 && (providerCell["engine_id"] != run.Cell.EngineID || providerCell["model_id"] != run.Cell.ModelID) {
			continue
		}
		acceptanceMap, _ := envelope["acceptance"].(map[string]any)
		if claimInBand(acceptanceMap["speaker-policy"]) {
			speakerPolicy = true
		}
		if claimInBand(acceptanceMap["corpus-quality"]) {
			corpusQuality = true
		}
	}
	return speakerPolicy && corpusQuality
}

func realtimeTransportArtifactPassed(artifactRef string) bool {
	if strings.TrimSpace(artifactRef) == "" {
		return false
	}
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
	return transportAssertionsPassed(run,
		"interval_accounting_exactly_once", "capture_signal_observed",
		"browser_retention_bounded", "server_retention_bounded",
		"per_frame_cost_constant", "zero_duplicate_committed_segments",
		"zero_silent_terminal_outcomes", "provider_cell_identity",
		"wire_rate_within_budget",
	)
}

func transportAssertionsPassed(run conformance.Run, names ...string) bool {
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

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": strings.TrimSpace(message)})
}

var Endpoints = []modulekit.EndpointDescriptor{
	{
		ID: "validation.soak", Path: "/api/v1/validation/soak", Method: "POST",
		Summary:     "Drive Dictation Studio through the real browser capture and ledger path",
		Description: "Leased test-only endpoint. Returns one conformance.Run document and never stores audio or transcript payloads.",
		Category:    "validation",
		RESTException: &modulekit.RESTException{
			Reason: modulekit.RESTReasonOpsProbe,
			Note:   "Out-of-band qualification coordination endpoint; the request and response are owned JSON transport for the existing conformance.Run schema.",
			ProtoPayloads: &modulekit.RESTProtoPayloads{
				Request:  modulekit.RESTPayload{Transport: "json", Conformance: "none"},
				Response: modulekit.RESTPayload{Transport: "json", Conformance: "none"},
				Error:    modulekit.RESTPayload{Transport: "json", Conformance: "none"},
			},
		},
	},
}
