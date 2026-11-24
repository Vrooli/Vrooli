package bridgeadmin

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/nodereach"
)

type fakeDispatcher struct {
	calls     []nodereach.DispatchRequest
	response  nodereach.DispatchResponse
	responses []nodereach.DispatchResponse
	errs      []error
}

func (f *fakeDispatcher) Dispatch(_ context.Context, r nodereach.DispatchRequest) (nodereach.DispatchResponse, error) {
	f.calls = append(f.calls, r)
	i := len(f.calls) - 1
	if i < len(f.errs) && f.errs[i] != nil {
		return nodereach.DispatchResponse{}, f.errs[i]
	}
	if i < len(f.responses) {
		return f.responses[i], nil
	}
	return f.response, nil
}

type memoryStore struct{ receipts map[string]Receipt }

func (m *memoryStore) Begin(_ context.Context, r Receipt) (Receipt, bool, error) {
	if old, ok := m.receipts[r.Key]; ok {
		return old, true, nil
	}
	m.receipts[r.Key] = r
	return r, false, nil
}
func (m *memoryStore) Finish(_ context.Context, id, status, run, detail string, replay bool) error {
	r := m.receipts[id]
	r.Status = status
	r.RunID = run
	r.Detail = detail
	r.Replayed = replay
	m.receipts[id] = r
	return nil
}
func (m *memoryStore) Get(_ context.Context, key string) (Receipt, error) {
	r, ok := m.receipts[key]
	if !ok {
		return Receipt{}, errors.New("not found")
	}
	return r, nil
}
func TestOutOfScopeIsReceiptedBeforeAnyBridgeEffect(t *testing.T) {
	d := &fakeDispatcher{}
	m := &memoryStore{receipts: map[string]Receipt{}}
	s := New(d, m, map[string]map[string]bool{"scenario-to-cloud": {"health": true}})
	_, err := s.Submit(context.Background(), "admin@example.test", Request{RequestID: "req-deny", Scenario: "scenario-to-desktop", Operation: "deploy", NodeID: "node-1", Tier: "local"})
	if !errors.Is(err, ErrOutOfScope) {
		t.Fatalf("err=%v", err)
	}
	if len(d.calls) != 0 {
		t.Fatalf("out-of-scope request reached Bridge: %+v", d.calls)
	}
	for _, got := range m.receipts {
		if got.Status != "refused" {
			t.Fatalf("receipt status=%q", got.Status)
		}
	}
}
func TestTimedSecondScenarioUsesSameScopedBoundaryAndReplayKey(t *testing.T) {
	d := &fakeDispatcher{response: nodereach.DispatchResponse{RunID: "run-1"}}
	m := &memoryStore{receipts: map[string]Receipt{}}
	s := New(d, m, map[string]map[string]bool{"scenario-to-cloud": {"health": true}, "scenario-to-desktop": {"health": true}})
	start := time.Now()
	for i, scenario := range []string{"scenario-to-cloud", "scenario-to-desktop"} {
		_, err := s.Submit(context.Background(), "admin@example.test", Request{RequestID: string(rune('a' + i)), Scenario: scenario, Operation: "health", NodeID: "test-target", Tier: "local"})
		if err != nil {
			t.Fatal(err)
		}
	}
	elapsed := time.Since(start)
	if elapsed > 2*time.Second {
		t.Fatalf("two-scenario walkthrough exceeded 2s: %s", elapsed)
	}
	if len(d.calls) != 2 {
		t.Fatalf("dispatch calls=%d", len(d.calls))
	}
	if d.calls[0].RequestID == d.calls[1].RequestID {
		t.Fatal("caller requests were not independently namespaced")
	}
	if d.calls[0].Scenario == d.calls[1].Scenario {
		t.Fatal("second scenario was not exercised")
	}
}

func TestAnyScenarioGeneralizationDelegatesManifestAndGrantAdmissionToBridge(t *testing.T) {
	d := &fakeDispatcher{response: nodereach.DispatchResponse{RunID: "bridge-run"}}
	m := &memoryStore{receipts: map[string]Receipt{}}
	// nil policy is the production mode: LPBS retains caller identity, tier,
	// idempotency and receipts; Bridge owns scenario manifest and node-grant
	// admission for all declared scenarios and operations.
	s := New(d, m, nil)
	receipt, err := s.Submit(context.Background(), "admin@example.test", Request{RequestID: "any-scenario", Scenario: "audio-tools", Operation: "audio render", NodeID: "test-device-1", Tier: "test-device", Args: []string{"--format", "wav"}})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != "accepted" || len(d.calls) != 1 {
		t.Fatalf("receipt=%+v dispatches=%d", receipt, len(d.calls))
	}
	got := d.calls[0]
	if got.Scenario != "audio-tools" || got.Verb != "audio render" || !reflect.DeepEqual(got.Args, []string{"--format", "wav"}) {
		t.Fatalf("typed Bridge request=%+v", got)
	}
	if got.DeploymentTier != "test-device" {
		t.Fatalf("Bridge request tier=%q", got.DeploymentTier)
	}
}

func TestBridgeTierMismatchIsDurablyRefusedBeforeTargetEffect(t *testing.T) {
	d := &fakeDispatcher{errs: []error{&nodereach.Error{Kind: nodereach.ErrTransport, Err: connect.NewError(connect.CodeFailedPrecondition, errors.New("registered tier mismatch"))}}}
	m := &memoryStore{receipts: map[string]Receipt{}}
	s := New(d, m, map[string]map[string]bool{"scenario-to-cloud": {"health": true}})
	receipt, err := s.Submit(context.Background(), "admin@example.test", Request{RequestID: "tier-mismatch", Scenario: "scenario-to-cloud", Operation: "health", NodeID: "node-1", Tier: "local"})
	if !errors.Is(err, ErrBridgeRefused) {
		t.Fatalf("err=%v", err)
	}
	if receipt.Status != "refused" || len(d.calls) != 1 || d.calls[0].DeploymentTier != "local" {
		t.Fatalf("receipt=%+v dispatches=%d", receipt, len(d.calls))
	}
}

func TestAmbiguousRetryReusesBridgeIdempotencyKeyAndReplayReceipt(t *testing.T) {
	d := &fakeDispatcher{
		errs:      []error{errors.New("response timed out after possible Bridge acceptance")},
		responses: []nodereach.DispatchResponse{{}, {RunID: "bridge-run-1", Replayed: true}},
	}
	m := &memoryStore{receipts: map[string]Receipt{}}
	s := New(d, m, nil)
	request := Request{RequestID: "ambiguous-1", Scenario: "audio-tools", Operation: "audio render", NodeID: "node-vps", Tier: "vps"}
	first, err := s.Submit(context.Background(), "admin@example.test", request)
	if err == nil || first.Status != "uncertain" {
		t.Fatalf("first receipt=%+v err=%v", first, err)
	}
	second, err := s.Submit(context.Background(), "admin@example.test", request)
	if err != nil {
		t.Fatal(err)
	}
	if second.Status != "replayed" || second.RunID != "bridge-run-1" {
		t.Fatalf("retry receipt=%+v", second)
	}
	if len(d.calls) != 2 || d.calls[0].RequestID != d.calls[1].RequestID || d.calls[0].RequestID == "" {
		t.Fatalf("retry did not reuse Bridge idempotency key: %+v", d.calls)
	}
}

func TestHandlerMapsBridgeTierMismatchToForbidden(t *testing.T) {
	d := &fakeDispatcher{errs: []error{&nodereach.Error{Kind: nodereach.ErrTransport, Err: connect.NewError(connect.CodeFailedPrecondition, errors.New("registered tier mismatch"))}}}
	m := &memoryStore{receipts: map[string]Receipt{}}
	s := New(d, m, map[string]map[string]bool{"scenario-to-cloud": {"health": true}})
	h := Handler(s, func(*http.Request) (string, bool) { return "admin@example.test", true })
	req := httptest.NewRequest(http.MethodPost, "/dispatch", strings.NewReader(`{"request_id":"mismatch-http","scenario":"scenario-to-cloud","operation":"health","node_id":"node-1","tier":"local"}`))
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if len(d.calls) != 1 {
		t.Fatalf("expected Bridge registry preflight call, got %+v", d.calls)
	}
}
