//go:build audio_lab

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
	"strings"
	"time"

	intexp "audio-tools/internal/experiment"
	"audio-tools/internal/modulekit"
	"audio-tools/internal/soak"
	"audio-tools/internal/stt/session"

	"github.com/gorilla/mux"
)

type Deps struct {
	Ledgers             *session.Registry
	Experiments         *intexp.Service
	TestIsolationActive func() bool
}

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

func serve(w http.ResponseWriter, r *http.Request, d Deps) {
	opt, ok := decodeSoakOptions(w, r)
	if !ok {
		return
	}
	ownerCtx, opt, ok := prepareManagedSoak(w, r, d, opt)
	if !ok {
		return
	}
	executeManagedSoak(ownerCtx, w, r, d, opt)
}

func decodeSoakOptions(w http.ResponseWriter, r *http.Request) (soak.Options, bool) {
	var opt soak.Options
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&opt); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("decode soak options: %v", err))
		return soak.Options{}, false
	}
	return opt, true
}

func prepareManagedSoak(w http.ResponseWriter, r *http.Request, d Deps, opt soak.Options) (context.Context, soak.Options, bool) {
	identityToken, okIdentity := exactlyOneHeader(r, "X-Agent-Identity-Token")
	grantReceipt, okGrant := exactlyOneHeader(r, "X-Vrooli-Browser-Grant-Receipt")
	testLeaseReceipt, okLease := exactlyOneHeader(r, "X-Test-Genie-Lease-Receipt")
	if !okIdentity || !okGrant || !okLease {
		writeError(w, http.StatusUnauthorized, "soak requires one run identity, browser grant, and Test Genie lease receipt")
		return nil, soak.Options{}, false
	}
	if !verifySoakLease(r.Context(), w, testLeaseReceipt) {
		return nil, soak.Options{}, false
	}
	brokerURL, opt, ok := configureManagedSoak(w, opt)
	if !ok {
		return nil, soak.Options{}, false
	}
	ownerCtx := soak.WithBrowserOwnerCredentials(r.Context(), identityToken, grantReceipt, testLeaseReceipt)
	ownerCtx = soak.WithManagedBrowserOwner(ownerCtx, brokerURL)
	// A normal product-path run may be observed without a temporary database
	// lease. Deterministic fault injection remains double-gated by the same
	// test-genie isolation seam used by the WebSocket transport. The explicit
	// harness gate exists for the standalone accelerated matrix; it is never
	// enabled by a normal deployment.
	faultHarnessEnabled := strings.EqualFold(strings.TrimSpace(os.Getenv("VROOLI_AUDIO_SOAK_FAULTS")), "1")
	if strings.TrimSpace(opt.Fault) != "" && (d.TestIsolationActive == nil || !d.TestIsolationActive()) && !faultHarnessEnabled {
		writeError(w, http.StatusPreconditionFailed, "faulted soak runs require an active test-genie isolation lease")
		return nil, soak.Options{}, false
	}
	return ownerCtx, opt, true
}

func verifySoakLease(ctx context.Context, w http.ResponseWriter, receipt string) bool {
	testGenieURL := firstNonEmpty(os.Getenv("TEST_GENIE_API_URL"), os.Getenv("TEST_GENIE_URL"))
	if testGenieURL == "" {
		writeError(w, http.StatusFailedDependency, "Test Genie lease verifier is not configured")
		return false
	}
	if err := VerifyAudioToolsDemandLease(ctx, testGenieURL, receipt, &http.Client{Timeout: 5 * time.Second}); err != nil {
		writeError(w, http.StatusForbidden, "Test Genie did not verify an active Audio Tools lease")
		return false
	}
	return true
}

func configureManagedSoak(w http.ResponseWriter, opt soak.Options) (string, soak.Options, bool) {
	brokerURL := firstNonEmpty(os.Getenv("BROWSER_AUTOMATION_STUDIO_API_URL"), os.Getenv("BAS_API_URL"))
	if brokerURL == "" {
		writeError(w, http.StatusFailedDependency, "BAS Agent Access endpoint is not configured")
		return "", soak.Options{}, false
	}
	brokerURL = strings.TrimRight(brokerURL, "/")
	if !strings.HasSuffix(brokerURL, "/api/v1") {
		brokerURL += "/api/v1"
	}
	uiURL := firstNonEmpty(os.Getenv("SOAK_UI_URL"), os.Getenv("AUDIO_TOOLS_UI_URL"), os.Getenv("UI_BASE_URL"))
	if uiURL == "" {
		writeError(w, http.StatusFailedDependency, "server-selected Audio Tools UI URL is not configured")
		return "", soak.Options{}, false
	}
	if requestedSurface := strings.TrimSpace(opt.Surface); requestedSurface != "" && requestedSurface != "audio-tools" {
		writeError(w, http.StatusBadRequest, "the managed Audio Tools soak only supports its owner-selected product surface")
		return "", soak.Options{}, false
	}
	fixture, err := soak.FixedDictationFixture(os.Getenv("VROOLI_ROOT"))
	if err != nil {
		writeError(w, http.StatusFailedDependency, "fixed Audio Tools qualification fixture is unavailable")
		return "", soak.Options{}, false
	}
	opt.UIURL = strings.TrimRight(uiURL, "/")
	opt.Surface = "audio-tools"
	opt.Fixture = fixture
	return brokerURL, opt, true
}

func executeManagedSoak(ownerCtx context.Context, w http.ResponseWriter, r *http.Request, d Deps, opt soak.Options) {
	// The qualification is server-owned. BAS owns admission and capacity for
	// isolated browser sessions; unrelated foreign sessions must not serialize
	// this scenario's work. A CLI disconnect, PTY timeout, or orchestration
	// cancellation must not close the browser lease halfway through a valid
	// long-form capture. RunTimeout still bounds the detached work, and scenario
	// stop remains the operator-level abort mechanism.
	runCtx, cancel := context.WithTimeout(context.WithoutCancel(ownerCtx), soak.RunTimeout(opt))
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

func exactlyOneHeader(r *http.Request, name string) (string, bool) {
	values := r.Header.Values(name)
	if len(values) != 1 || strings.TrimSpace(values[0]) == "" {
		return "", false
	}
	return strings.TrimSpace(values[0]), true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
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
