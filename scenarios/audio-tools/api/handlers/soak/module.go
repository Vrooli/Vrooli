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
	"sync/atomic"

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

// qualificationActive is process-local because the soak owns the one shared
// BAS/browser and PipeWire qualification lane. The CLI performs a capacity
// preflight for operator feedback, but admission must also be guarded here so
// direct API callers cannot start concurrent server-owned runs between that
// check and the POST.
var qualificationActive atomic.Bool

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
	if !qualificationActive.CompareAndSwap(false, true) {
		writeError(w, http.StatusConflict, "a server-owned audio qualification is already running; wait for it to finish before starting another")
		return
	}
	defer qualificationActive.Store(false)
	// The qualification is server-owned. A CLI disconnect, PTY timeout, or
	// orchestration cancellation must not close the browser lease halfway
	// through a valid long-form capture. RunTimeout still bounds the detached
	// work, and scenario stop remains the operator-level abort mechanism.
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
	machineJSON, err := json.Marshal(machine)
	if err != nil {
		return fmt.Errorf("encode machine evidence: %w", err)
	}
	// Browser-product-path evidence answers whether the browser/ledger/product
	// route completed. Recognition quality is a separate conformance assertion
	// and must not erase proof that this route was exercised.
	// A completed HTTP request is not the same thing as a qualified run. The
	// driver returns a conformance document even when an assertion fails so
	// callers can inspect the evidence; persisted trust-floor rows must mirror
	// that verdict and never turn a partial run into a passing product/device
	// record.
	passed := qualificationRunPassed(result.Run, runErr)
	if _, err := service.RecordQualificationEvidence(ctx, intexp.QualificationEvidence{
		EngineID: opt.EngineID, ModelID: opt.ModelID, Strategy: opt.Strategy, PolicyProfile: opt.Policy,
		Kind: trustfloor.QualificationBrowserProductPath, Passed: passed, ArtifactRef: artifactRef,
		Notes:       "automated BAS browser product-path qualification; pass requires the complete conformance run",
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
	machineJSON, err = json.Marshal(machine)
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
