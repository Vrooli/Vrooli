package agentaccess

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/automation/session"
	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
)

const testAppMonitorURL = "https://monitor.example.test/apps/workspace"

type appMonitorDriverSession struct {
	id, owner, lease string
	mu               sync.Mutex
	url              string
}

type appMonitorMetricBarrier struct {
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

type appMonitorFakeDriver struct {
	mu              sync.Mutex
	sessions        map[string]*appMonitorDriverSession
	starts          atomic.Int32
	metricReads     atomic.Int32
	traceStarts     atomic.Int32
	traceFinish     atomic.Int32
	navigationReads atomic.Int32
	metricBarrier   *appMonitorMetricBarrier
}

func (d *appMonitorFakeDriver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.URL.Path == "/session/start" {
		var request driver.CreateSessionRequest
		if json.NewDecoder(r.Body).Decode(&request) != nil {
			http.Error(w, "invalid create", http.StatusBadRequest)
			return
		}
		n := d.starts.Add(1)
		id := fmt.Sprintf("app-monitor-session-%d", n)
		state := &appMonitorDriverSession{id: id, owner: request.ExecutionID, lease: fmt.Sprintf("app-monitor-lease-%d", n), url: "about:blank"}
		d.mu.Lock()
		d.sessions[id] = state
		d.mu.Unlock()
		_ = json.NewEncoder(w).Encode(driver.CreateSessionResponse{SessionID: id, LeaseID: state.lease, ActivePageID: "page-" + id})
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 || parts[0] != "session" {
		http.NotFound(w, r)
		return
	}
	d.mu.Lock()
	state := d.sessions[parts[1]]
	d.mu.Unlock()
	if state == nil {
		http.NotFound(w, r)
		return
	}
	checkLease := func() bool {
		return r.Header.Get(driver.SessionExecutionHeader) == state.owner && r.Header.Get(driver.SessionLeaseHeader) == state.lease
	}
	switch strings.Join(parts[2:], "/") {
	case "record/navigate":
		var request struct {
			URL         string `json:"url"`
			ExecutionID string `json:"execution_id"`
			LeaseID     string `json:"lease_id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&request)
		if request.ExecutionID != state.owner || request.LeaseID != state.lease {
			http.Error(w, "wrong lease", http.StatusForbidden)
			return
		}
		state.mu.Lock()
		state.url = request.URL
		state.mu.Unlock()
		_ = json.NewEncoder(w).Encode(driver.NavigateResponse{DriverPageID: "page-" + state.id, URL: request.URL})
	case "record/navigation-state":
		if r.URL.Query().Get("execution_id") != state.owner || r.URL.Query().Get("lease_id") != state.lease {
			http.Error(w, "wrong lease", http.StatusForbidden)
			return
		}
		d.navigationReads.Add(1)
		state.mu.Lock()
		currentURL := state.url
		state.mu.Unlock()
		_ = json.NewEncoder(w).Encode(driver.NavigationStateResponse{SessionID: state.id, URL: currentURL, Title: "App Monitor"})
	case "app-monitor-metrics":
		if !checkLease() {
			http.Error(w, "wrong lease", http.StatusForbidden)
			return
		}
		d.metricReads.Add(1)
		d.mu.Lock()
		barrier := d.metricBarrier
		d.mu.Unlock()
		if barrier != nil {
			barrier.once.Do(func() { close(barrier.entered) })
			<-barrier.release
		}
		state.mu.Lock()
		currentURL := state.url
		state.mu.Unlock()
		_ = json.NewEncoder(w).Encode(driver.AppMonitorMetricsResponse{
			Contract: driver.AppMonitorMetricsContract, SessionID: state.id, PageID: "page-" + state.id,
			URL: currentURL, ObservedAt: time.Now().UTC().Format(time.RFC3339Nano),
			IframeCount: 8, PaneCount: 8, TabCardCount: 4, ProfilerMarkCount: 2, WorkspaceStorageBytes: 256,
		})
	case "app-monitor-capture/start":
		if !checkLease() {
			http.Error(w, "wrong lease", http.StatusForbidden)
			return
		}
		d.traceStarts.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{"contract": "app-monitor.preview-workspace.trace.v1", "sessionId": state.id, "captureId": captureIDFromBody(r), "status": "started"})
	case "app-monitor-capture/finish":
		if !checkLease() {
			http.Error(w, "wrong lease", http.StatusForbidden)
			return
		}
		d.traceFinish.Add(1)
		captureID := captureIDFromBody(r)
		trace := `{"traceEvents":[]}`
		digest := sha256.Sum256([]byte(trace))
		_ = json.NewEncoder(w).Encode(driver.AppMonitorTraceArtifact{Contract: "app-monitor.preview-workspace.trace.v1", SessionID: state.id, PageID: "page-" + state.id, CaptureID: captureID, SHA256: hex.EncodeToString(digest[:]), ByteLength: len(trace), TraceJSON: trace})
	case "close":
		_ = json.NewEncoder(w).Encode(driver.CloseSessionResponse{Success: true})
	default:
		http.NotFound(w, r)
	}
}

func TestBrokerAppMonitorCaptureWithholdsReadAfterMidCaptureRevocation(t *testing.T) {
	broker, owner, ctxA, _, _, _ := newAppMonitorCaptureHarness(t)
	info, err := broker.Open(ctxA, OpenRequest{Backend: string(sessionpermission.BackendManaged), ReuseMode: "fresh", StartURL: testAppMonitorURL})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = broker.Close(ctxA, info.SessionID) })
	begin, err := broker.CaptureAppMonitor(ctxA, info.SessionID, "", AppMonitorCaptureBegin)
	if err != nil {
		t.Fatal(err)
	}
	barrier := &appMonitorMetricBarrier{entered: make(chan struct{}), release: make(chan struct{})}
	owner.mu.Lock()
	owner.metricBarrier = barrier
	owner.mu.Unlock()
	type captureResult struct {
		response AppMonitorCaptureResponse
		err      error
	}
	result := make(chan captureResult, 1)
	go func() {
		response, captureErr := broker.CaptureAppMonitor(ctxA, info.SessionID, begin.Binding.CaptureID, AppMonitorCaptureAfterMount)
		result <- captureResult{response: response, err: captureErr}
	}()
	<-barrier.entered

	profiles := broker.profiles.(*crossRunProfileResolver)
	profiles.mu.Lock()
	profile := profiles.rows["profile-app-monitor"]
	profile.AppMonitorMetrics = false
	profiles.rows["profile-app-monitor"] = profile
	profiles.mu.Unlock()
	close(barrier.release)
	completed := <-result
	if completed.err == nil || completed.response.Metrics != nil {
		t.Fatalf("mid-capture revoke released metrics: response=%+v err=%v", completed.response, completed.err)
	}
	if owner.metricReads.Load() != 1 || owner.traceStarts.Load() != 1 {
		t.Fatalf("unexpected driver effects after revoke: metrics=%d traceStarts=%d", owner.metricReads.Load(), owner.traceStarts.Load())
	}
}

func captureIDFromBody(r *http.Request) string {
	var body struct {
		CaptureID string `json:"capture_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	return body.CaptureID
}

func newAppMonitorCaptureHarness(t *testing.T) (*Broker, *appMonitorFakeDriver, context.Context, context.Context, string, string) {
	t.Helper()
	serverOwner := &appMonitorFakeDriver{sessions: make(map[string]*appMonitorDriverSession)}
	server := httptest.NewServer(serverOwner)
	t.Cleanup(server.Close)
	client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatal(err)
	}
	manager := session.NewManagerWithClient(client)
	profile := sessionpermission.Profile{
		Effect: sessionpermission.EffectAct, Approval: sessionpermission.ApprovalNone,
		ID: "profile-app-monitor", View: true, Control: true, ProfileMode: "fresh", ProfileID: "profile-app-monitor",
		URLAllow: []string{"https://monitor.example.test/*"}, MaxActions: 100, MaxDuration: time.Minute,
		AppMonitorMetrics: true, AppMonitorTrace: true, AppMonitorTraceMaxDurationMS: 30_000, AppMonitorTraceMaxBytes: 1 << 20,
	}
	policy, err := sessionpermission.New(profile, nil)
	if err != nil {
		t.Fatal(err)
	}
	profiles := &crossRunProfileResolver{rows: map[string]sessionpermission.Profile{"profile-app-monitor": profile}, policies: map[string]sessionpermission.Authorizer{"profile-app-monitor": policy}}
	broker, err := NewBroker(manager, profiles, crossRunOpenAuthorizer{}, crossRunCallerResolver{})
	if err != nil {
		t.Fatal(err)
	}
	broker.SetTrustedOpenContextResolver(crossRunOpenResolver{})
	broker.SetEvidenceRecorder(identityEvidenceRecorder{})
	broker.SetCapabilityReporter(crossRunCapabilityReporter{{
		Backend: sessionpermission.BackendManaged, Qualified: true, EvidenceID: "synthetic-app-monitor",
		Capabilities: []sessionpermission.Capability{sessionpermission.CapabilityObserve, sessionpermission.CapabilityControl, sessionpermission.CapabilityAppMonitorMetrics, sessionpermission.CapabilityAppMonitorTrace},
	}})
	callerA := "run:00000000-0000-4000-8000-000000000001"
	callerB := "run:00000000-0000-4000-8000-000000000002"
	ctxA := WithBrowserAgentCredentials(crossRunContext(callerA, "profile-app-monitor"), "signed:"+callerA, "grant:"+callerA)
	ctxB := WithBrowserAgentCredentials(crossRunContext(callerB, "profile-app-monitor"), "signed:"+callerB, "grant:"+callerB)
	return broker, serverOwner, ctxA, ctxB, callerA, callerB
}

func TestBrokerAppMonitorCaptureBindsSingleSessionAndPhases(t *testing.T) {
	broker, owner, ctxA, ctxB, callerA, _ := newAppMonitorCaptureHarness(t)
	info, err := broker.Open(ctxA, OpenRequest{Backend: string(sessionpermission.BackendManaged), ReuseMode: "fresh", StartURL: testAppMonitorURL})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = broker.Close(ctxA, info.SessionID) })
	baseNavigationReads := owner.navigationReads.Load()
	if _, err := broker.CaptureAppMonitor(ctxB, info.SessionID, "", AppMonitorCaptureBegin); err == nil {
		t.Fatal("foreign caller began a capture on run A's session")
	}
	if owner.navigationReads.Load() != baseNavigationReads || owner.metricReads.Load() != 0 || owner.traceStarts.Load() != 0 {
		t.Fatalf("foreign caller reached driver: nav=%d metrics=%d trace-start=%d", owner.navigationReads.Load()-baseNavigationReads, owner.metricReads.Load(), owner.traceStarts.Load())
	}

	begin, err := broker.CaptureAppMonitor(ctxA, info.SessionID, "", AppMonitorCaptureBegin)
	if err != nil {
		t.Fatal(err)
	}
	if begin.Binding.CaptureID == "" || begin.Binding.CallerID != callerA || begin.Binding.RunID != "00000000-0000-4000-8000-000000000001" || begin.Binding.ProfileRevision == "" || begin.Trace != nil || begin.Metrics != nil {
		t.Fatalf("begin binding leaked/unbound fields: %+v", begin)
	}
	for _, phase := range []AppMonitorCapturePhase{AppMonitorCaptureAfterMount, AppMonitorCaptureAfterScroll, AppMonitorCaptureAfterPicker} {
		response, err := broker.CaptureAppMonitor(ctxA, info.SessionID, begin.Binding.CaptureID, phase)
		if err != nil {
			t.Fatalf("phase %s: %v", phase, err)
		}
		if response.Binding.CaptureID != begin.Binding.CaptureID || response.Binding.SessionID != info.SessionID || response.Binding.LeaseID != begin.Binding.LeaseID || response.Metrics == nil || response.Trace != nil {
			t.Fatalf("phase %s returned an invalid binding or extra trace data: %+v", phase, response)
		}
	}
	finish, err := broker.CaptureAppMonitor(ctxA, info.SessionID, begin.Binding.CaptureID, AppMonitorCaptureFinish)
	if err != nil {
		t.Fatal(err)
	}
	if finish.Trace == nil || finish.Trace.CaptureID != begin.Binding.CaptureID || finish.Metrics != nil || owner.metricReads.Load() != 3 || owner.traceStarts.Load() != 1 || owner.traceFinish.Load() != 1 {
		t.Fatalf("finish result or fixed owner calls invalid: response=%+v reads=%d starts=%d finishes=%d", finish, owner.metricReads.Load(), owner.traceStarts.Load(), owner.traceFinish.Load())
	}
	if _, err := broker.CaptureAppMonitor(ctxA, info.SessionID, begin.Binding.CaptureID, AppMonitorCaptureAfterPicker); err == nil {
		t.Fatal("replayed capture phase was accepted")
	}
}
