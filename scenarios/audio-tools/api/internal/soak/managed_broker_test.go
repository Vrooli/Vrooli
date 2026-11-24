//go:build audio_lab

package soak

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"audio-tools/internal/conformance"
)

func TestManagedSoakTaskStaysOnOneBASBrokerSession(t *testing.T) {
	type requestRecord struct {
		Path      string
		Identity  string
		Grant     string
		TestGenie string
		Body      map[string]any
	}
	var requests []requestRecord
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var body map[string]any
		if r.Body != nil {
			data, _ := io.ReadAll(r.Body)
			if len(data) > 0 {
				_ = json.Unmarshal(data, &body)
			}
		}
		requests = append(requests, requestRecord{Path: r.URL.Path, Identity: r.Header.Get("X-Agent-Identity-Token"), Grant: r.Header.Get("X-Vrooli-Browser-Grant-Receipt"), TestGenie: r.Header.Get("X-Test-Genie-Lease-Receipt"), Body: body})
		switch r.URL.Path {
		case "/browser/sessions":
			_, _ = io.WriteString(w, `{"session_id":"bas-session-1","audio_device_evidence":{"enumerated":true,"selectedLabel":"PipeWire","sampleRate":16000,"channelCount":1}}`)
		case "/browser/sessions/bas-session-1/act":
			_, _ = io.WriteString(w, `{"session_id":"bas-session-1","outcome":{"success":true}}`)
		case "/browser/sessions/bas-session-1/audio/diagnostics":
			_, _ = io.WriteString(w, `{"session_id":"bas-session-1","diagnostics":{"diag":{"state":"recording","capturedSequence":1},"final":"private transcript","interim":"quick","coverage_visible":true,"export_available":true,"diagnostic_metadata_only":true}}`)
		default:
			_, _ = io.WriteString(w, `{"completed":true}`)
		}
	}))
	defer server.Close()
	d := &driver{
		client: server.Client(),
		opt:    Options{Lane: conformance.LaneRealtime, UIURL: "https://dictation.example.test", Surface: "audio-tools", Strategy: "product"},
		owner:  &browserOwnerCredentials{identityToken: "signed-run-token", grantReceipt: "run-grant", testLease: "tglease_opaque", brokerURL: server.URL},
	}
	ctx := context.Background()
	started, err := d.start(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if started.SessionID != "bas-session-1" || started.AudioDeviceEvidence == nil || !started.AudioDeviceEvidence.Enumerated {
		t.Fatalf("BAS open result=%#v", started)
	}
	if err := d.step(ctx, started.SessionID, 1, "start", map[string]any{"click": map[string]any{"selector": "[data-testid=ready]"}}); err != nil {
		t.Fatal(err)
	}
	var observation pageObservation
	if err := d.evaluate(ctx, started.SessionID, 2, &observation); err != nil {
		t.Fatal(err)
	}
	if observation.Transcript != "private transcript" || !observation.CoverageVisible || !observation.ExportAvailable {
		t.Fatalf("fixed diagnostic response=%#v", observation)
	}
	for _, suffix := range []string{"restart", "stop"} {
		if err := d.postBroker(ctx, "/browser/sessions/"+started.SessionID+"/audio/"+suffix, map[string]any{}, nil); err != nil {
			t.Fatal(err)
		}
	}
	if err := d.close(ctx, started); err != nil {
		t.Fatal(err)
	}
	if len(requests) != 6 {
		t.Fatalf("broker call count=%d, want open+act+diagnostics+restart+stop+close: %#v", len(requests), requests)
	}
	for _, request := range requests {
		if request.Identity != "signed-run-token" || request.Grant != "run-grant" || request.TestGenie != "" {
			t.Fatalf("authority headers were not scoped to BAS: %#v", request)
		}
		if request.Path == "" || request.Path[:min(len(request.Path), 9)] == "/session/" {
			t.Fatalf("managed task opened a direct driver path: %#v", request)
		}
	}
	open := requests[0].Body
	if open["audio_mode"] != "pipewire_qualification" || open["reuse_mode"] != "fresh" || open["start_url"] != "https://dictation.example.test/dictation-studio?stt_engine_id=&stt_strategy=product" {
		t.Fatalf("managed open did not bind fresh same-session audio task: %#v", open)
	}
	if _, clientSelectedProfile := open["permission_ref"]; clientSelectedProfile {
		t.Fatalf("managed open unexpectedly selected a client profile: %#v", open)
	}
}

func TestAcceleratedManagedStartRequestsFixedFakeMicrophoneOwnerMode(t *testing.T) {
	var request map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/browser/sessions" || r.Method != http.MethodPost {
			t.Errorf("unexpected managed start request: %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode managed start: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"session_id":"accelerated-session"}`)
	}))
	defer server.Close()
	d := &driver{
		client: server.Client(),
		opt:    Options{Lane: conformance.LaneAccelerated, UIURL: "https://audio-tools.example.test", Fixture: "server-fixed-wav"},
		owner:  &browserOwnerCredentials{identityToken: "run", grantReceipt: "grant", testLease: "lease", brokerURL: server.URL},
	}
	started, err := d.start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if started.SessionID != "accelerated-session" || request["audio_mode"] != "fake_microphone" {
		t.Fatalf("accelerated start=%#v request=%#v", started, request)
	}
	if _, ok := request["fixture"]; ok {
		t.Fatalf("caller fixture path escaped to BAS: %#v", request)
	}
}

func TestManagedRunWithEvidenceDrivesWholeTaskThroughOneBrokerSession(t *testing.T) {
	type requestRecord struct {
		Path      string
		Identity  string
		Grant     string
		TestGenie string
		Body      map[string]any
	}
	var mu sync.Mutex
	var requests []requestRecord
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var body map[string]any
		if r.Body != nil {
			data, _ := io.ReadAll(r.Body)
			if len(data) > 0 {
				_ = json.Unmarshal(data, &body)
			}
		}
		mu.Lock()
		requests = append(requests, requestRecord{Path: r.URL.Path, Identity: r.Header.Get("X-Agent-Identity-Token"), Grant: r.Header.Get("X-Vrooli-Browser-Grant-Receipt"), TestGenie: r.Header.Get("X-Test-Genie-Lease-Receipt"), Body: body})
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/browser/sessions":
			_, _ = io.WriteString(w, `{"session_id":"bas-session-full-task","audio_device_evidence":{"enumerated":true,"selectedLabel":"PipeWire","sampleRate":16000,"channelCount":1}}`)
		case "/browser/sessions/bas-session-full-task/audio/diagnostics":
			_, _ = io.WriteString(w, `{"session_id":"bas-session-full-task","diagnostics":{"diag":{"schemaVersion":1,"sessionId":"bas-session-full-task","state":"completed","capturedSequence":1,"capturedSamples":16000,"captureSampleRate":16000,"sentSequence":1,"processedSequence":1,"retainedBytes":0,"signalObserved":true,"captureStartedAtMs":1,"firstCaptureAtMs":2,"lastCaptureAtMs":3,"firstSignalAtMs":2,"firstSentAtMs":3,"firstPartialAtMs":4,"firstPartialLatencyMs":3,"committedTextLagMs":2,"providerId":"engine-qualified","modelId":"model-qualified","doneSent":true,"terminalReason":"final","statusCodes":[],"errorCodes":[]},"final":"private transcription words","interim":"private interim words","coverage_visible":true,"export_available":true,"diagnostic_metadata_only":true}}`)
		default:
			_, _ = io.WriteString(w, `{"session_id":"bas-session-full-task","outcome":{"success":true,"extracted_data":{}}}`)
		}
	}))
	defer server.Close()

	root, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := FixedDictationFixture(root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithBrowserOwnerCredentials(context.Background(), "opaque-run-identity", "opaque-am-browser-grant", "opaque-test-genie-lease")
	ctx = WithManagedBrowserOwner(ctx, server.URL)
	result, runErr := RunWithEvidence(ctx, Options{
		UIURL:     "https://audio-tools.example.test",
		Fixture:   fixture,
		Lane:      conformance.LaneRealtime,
		Profile:   "continuous",
		Turns:     1,
		FeedMS:    10,
		Reference: "private transcription words",
		EngineID:  "engine-qualified",
		ModelID:   "model-qualified",
		Strategy:  "product",
		Policy:    "default",
	}, nil)
	if runErr != nil {
		t.Fatalf("managed full task returned an execution error: %v", runErr)
	}
	if result.AudioDeviceEvidence == nil || !result.AudioDeviceEvidence.Enumerated || result.AudioDeviceEvidence.SelectedLabel != "PipeWire" {
		t.Fatalf("metadata-only PipeWire receipt missing: %#v", result.AudioDeviceEvidence)
	}
	if result.Run.Evaluate().Qualified {
		t.Fatal("fake owner diagnostics without a live ledger must not qualify the run")
	}
	var evidence strings.Builder
	if err := result.Run.WriteJSON(&evidence); err != nil {
		t.Fatal(err)
	}
	for _, private := range []string{
		"private transcription words",
		"private interim words",
		"opaque-run-identity",
		"opaque-am-browser-grant",
		"opaque-test-genie-lease",
	} {
		if strings.Contains(evidence.String(), private) {
			t.Fatalf("conformance evidence persisted request-private data (%q): %s", private, evidence.String())
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if len(requests) < 8 {
		t.Fatalf("managed task made too few broker requests: %#v", requests)
	}
	if requests[0].Path != "/browser/sessions" || requests[0].Body["audio_mode"] != "pipewire_qualification" {
		t.Fatalf("managed task did not open the qualified fresh BAS session: %#v", requests[0])
	}
	seen := map[string]bool{}
	for _, request := range requests {
		seen[request.Path] = true
		if request.Identity != "opaque-run-identity" || request.Grant != "opaque-am-browser-grant" || request.TestGenie != "" {
			t.Fatalf("authority was not confined to BAS owner headers: %#v", request)
		}
		if strings.HasPrefix(request.Path, "/session/") {
			t.Fatalf("managed task bypassed the Broker: %#v", request)
		}
	}
	for _, required := range []string{
		"/browser/sessions/bas-session-full-task/act",
		"/browser/sessions/bas-session-full-task/audio/diagnostics",
		"/browser/sessions/bas-session-full-task/audio/restart",
		"/browser/sessions/bas-session-full-task/audio/stop",
		"/browser/sessions/bas-session-full-task/close",
	} {
		if !seen[required] {
			t.Errorf("managed task omitted same-session operation %q; paths=%v", required, seen)
		}
	}
}

func TestManagedRunClosesBrokerSessionWhenOwnerActionIsDenied(t *testing.T) {
	var mu sync.Mutex
	var paths []string
	closeCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/browser/sessions":
			_, _ = io.WriteString(w, `{"session_id":"cleanup-session","audio_device_evidence":{"enumerated":true,"selectedLabel":"PipeWire","sampleRate":16000,"channelCount":1}}`)
		case "/browser/sessions/cleanup-session/act":
			w.WriteHeader(http.StatusForbidden)
			_, _ = io.WriteString(w, `{"error":"scope_denied"}`)
		case "/browser/sessions/cleanup-session/close":
			mu.Lock()
			closeCount++
			mu.Unlock()
			_, _ = io.WriteString(w, `{"closed":true}`)
		default:
			t.Errorf("unexpected Broker route: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	root, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := FixedDictationFixture(root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithBrowserOwnerCredentials(context.Background(), "run-identity", "browser-grant", "test-genie-lease")
	ctx = WithManagedBrowserOwner(ctx, server.URL)
	_, runErr := RunWithEvidence(ctx, Options{
		UIURL: "https://audio-tools.example.test", Fixture: fixture,
		Lane: conformance.LaneRealtime, Profile: "continuous", Turns: 1, FeedMS: 10,
		EngineID: "engine-qualified", ModelID: "model-qualified", Strategy: "product", Policy: "default",
	}, nil)
	if runErr == nil || !strings.Contains(runErr.Error(), "HTTP 403") {
		t.Fatalf("owner action denial error = %v", runErr)
	}
	mu.Lock()
	defer mu.Unlock()
	if closeCount != 1 {
		t.Fatalf("session close attempts = %d, want 1; paths=%v", closeCount, paths)
	}
	if len(paths) != 3 || paths[0] != "/browser/sessions" || paths[1] != "/browser/sessions/cleanup-session/act" || paths[2] != "/browser/sessions/cleanup-session/close" {
		t.Fatalf("denied task path order = %v", paths)
	}
}

func TestManagedStartClosesUnqualifiedPipeWireSession(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/browser/sessions" {
			_, _ = io.WriteString(w, `{"session_id":"unqualified-session"}`)
			return
		}
		if r.URL.Path != "/browser/sessions/unqualified-session/close" {
			t.Errorf("unexpected Broker route: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = io.WriteString(w, `{"closed":true}`)
	}))
	defer server.Close()
	d := &driver{
		client: server.Client(),
		opt:    Options{Lane: conformance.LaneRealtime, UIURL: "https://audio-tools.example.test"},
		owner:  &browserOwnerCredentials{identityToken: "run", grantReceipt: "grant", testLease: "lease", brokerURL: server.URL},
	}
	if _, err := d.start(context.Background()); err == nil || !strings.Contains(err.Error(), "PipeWire-qualified") {
		t.Fatalf("unqualified session error = %v", err)
	}
	if len(paths) != 2 || paths[0] != "/browser/sessions" || paths[1] != "/browser/sessions/unqualified-session/close" {
		t.Fatalf("unqualified open cleanup order = %v", paths)
	}
}
