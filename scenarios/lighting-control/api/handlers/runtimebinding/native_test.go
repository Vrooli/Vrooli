package runtimebinding

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/lightingbinding"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
)

func nativeIdentity() lightingbinding.Identity {
	return lightingbinding.Identity{Scenario: "lighting-control", Variant: "live", RuntimeInstanceID: "instance-one", Generation: "api-one", PID: os.Getpid()}
}
func nativeRequest(id string, revision uint64, binding *lightingbinding.Binding) lightingbinding.Request {
	return lightingbinding.Request{Identity: nativeIdentity(), OperationID: id, ExpectedRevision: revision, Binding: binding}
}
func nativeCall(t *testing.T, handler http.Handler, request lightingbinding.Request) (int, lightingbinding.Receipt) {
	t.Helper()
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/apply", bytes.NewReader(body)))
	var receipt lightingbinding.Receipt
	if response.Code == 200 {
		if err := json.Unmarshal(response.Body.Bytes(), &receipt); err != nil {
			t.Fatal(err)
		}
	}
	return response.Code, receipt
}

func TestNativeBindingRecoveryReplayAndLostAcknowledgment(t *testing.T) {
	service := &fakeService{}
	handler := newNativeHandler(nativeIdentity(), service)
	binding := &lightingbinding.Binding{Address: "[::1]:6742", Incarnation: "backend-one"}
	request := nativeRequest("bind-one", 0, binding)
	code, receipt := nativeCall(t, handler, request)
	if code != 200 || receipt.Error != "" || receipt.State.Binding == nil || receipt.State.Revision != 1 {
		t.Fatalf("bind receipt %#v, code %d", receipt, code)
	}
	// Simulate losing the response. The retained request returns the same receipt
	// without applying effective policy a second time.
	code, retry := nativeCall(t, handler, request)
	if code != 200 || retry.OperationID != receipt.OperationID || service.binds != 1 || service.reconciles != 1 {
		t.Fatalf("lost ACK replayed effects: %#v %+v", retry, service)
	}
	code, receipt = nativeCall(t, handler, nativeRequest("quiesce", 1, nil))
	if code != 200 || receipt.State.Binding != nil || service.quiesces != 1 {
		t.Fatalf("quiesce: %#v %+v", receipt, service)
	}
	if code, _ := nativeCall(t, handler, request); code != 409 {
		t.Fatalf("old bind revived after quiesce: HTTP %d", code)
	}
	replacement := *binding
	replacement.Incarnation = "backend-two"
	code, receipt = nativeCall(t, handler, nativeRequest("recover", 2, &replacement))
	if code != 200 || receipt.State.Binding.Incarnation != "backend-two" || service.binds != 2 {
		t.Fatalf("generation recovery: %#v %+v", receipt, service)
	}
	stale := nativeRequest("stale", 3, binding)
	stale.Identity.Generation = "old-api"
	if code, _ := nativeCall(t, handler, stale); code != 409 || service.binds != 2 {
		t.Fatal("stale scenario generation performed work")
	}
}

func TestNativeBindingRejectsRemoteAndConflictingRetries(t *testing.T) {
	service := &fakeService{}
	handler := newNativeHandler(nativeIdentity(), service)
	for _, address := range []string{"localhost:6742", "192.0.2.1:6742", "127.0.0.1:0", "[::1%lo]:6742"} {
		if code, _ := nativeCall(t, handler, nativeRequest("bad", 0, &lightingbinding.Binding{Address: address, Incarnation: "backend"})); code != 400 {
			t.Fatalf("address %q HTTP %d", address, code)
		}
	}
	request := nativeRequest("same-id", 0, &lightingbinding.Binding{Address: "127.0.0.1:6742", Incarnation: "one"})
	nativeCall(t, handler, request)
	request.Binding = &lightingbinding.Binding{Address: "127.0.0.1:6742", Incarnation: "two"}
	if code, _ := nativeCall(t, handler, request); code != 409 || service.binds != 1 {
		t.Fatal("conflicting retry performed work")
	}
}

type failingNativeService struct{ fakeService }

func (s *failingNativeService) Reconcile(context.Context) ([]*controlv1.ReconcileZoneResult, error) {
	s.reconciles++
	return nil, errors.New("fixture SDK unavailable")
}
func TestNativeReconcileFailureQuiescesAndRetainsErrorReceipt(t *testing.T) {
	service := &failingNativeService{}
	handler := newNativeHandler(nativeIdentity(), service)
	request := nativeRequest("failure", 0, &lightingbinding.Binding{Address: "127.0.0.1:6742", Incarnation: "one"})
	_, receipt := nativeCall(t, handler, request)
	if receipt.Error == "" || receipt.State.Binding != nil || service.quiesces != 1 {
		t.Fatalf("failure did not fence backend: %#v %+v", receipt, service)
	}
	nativeCall(t, handler, request)
	if service.binds != 1 || service.quiesces != 1 {
		t.Fatal("failed uncertain receipt replayed SDK effects")
	}
}

type drainingNativeService struct {
	fakeService
	entered, release chan struct{}
	once             sync.Once
}

func (s *drainingNativeService) Quiesce(ctx context.Context) error {
	s.once.Do(func() { close(s.entered) })
	select {
	case <-s.release:
		s.quiesces++
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func TestNativeQuiesceAcknowledgmentWaitsForWriterDrain(t *testing.T) {
	service := &drainingNativeService{entered: make(chan struct{}), release: make(chan struct{})}
	handler := newNativeHandler(nativeIdentity(), service)
	done := make(chan struct{})
	go func() {
		defer close(done)
		body, _ := json.Marshal(nativeRequest("drain", 0, nil))
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/v1/apply", bytes.NewReader(body)))
	}()
	<-service.entered
	select {
	case <-done:
		t.Fatal("quiesce acknowledged before drain")
	default:
	}
	close(service.release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("drain did not acknowledge")
	}
}
func TestNativeBindingOptOutDoesNoWork(t *testing.T) {
	service := &fakeService{}
	stop, err := StartNativeFromEnv(context.Background(), service, func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	stop()
	if *service != (fakeService{}) {
		t.Fatalf("opt-out performed work: %+v", service)
	}
}

func TestNativeBindingShadowLaunchDoesNoWork(t *testing.T) {
	service := &fakeService{}
	stop, err := StartNativeFromEnv(context.Background(), service, func(key string) string {
		switch key {
		case "VROOLI_RUNTIME_DIAGNOSTICS_DIR":
			return "/must-not-be-created"
		case "VROOLI_SCENARIO":
			return "lighting-control"
		case "VROOLI_VARIANT":
			return "shadow"
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	stop()
	if *service != (fakeService{}) {
		t.Fatal("shadow launch performed backend work")
	}
}

func TestNativeBindingNewScenarioGenerationReappliesRetainedBackend(t *testing.T) {
	service := &fakeService{}
	current := nativeIdentity()
	current.Generation = "api-restarted"
	handler := newNativeHandler(current, service)
	binding := &lightingbinding.Binding{Address: "127.0.0.1:6742", Incarnation: "same-owned-backend"}
	stale := nativeRequest("old-root-request", 0, binding)
	if code, _ := nativeCall(t, handler, stale); code != 409 || service.binds != 0 {
		t.Fatal("old generation was adopted")
	}
	recovered := nativeRequest("fresh-root-recovery", 0, binding)
	recovered.Identity = current
	if code, receipt := nativeCall(t, handler, recovered); code != 200 || receipt.Error != "" || service.binds != 1 || service.reconciles != 1 {
		t.Fatalf("fresh owner did not restore effective policy: %d %#v %+v", code, receipt, service)
	}
}

type zoneNativeService struct {
	fakeService
	results []*controlv1.ReconcileZoneResult
}

func (s *zoneNativeService) Reconcile(context.Context) ([]*controlv1.ReconcileZoneResult, error) {
	s.reconciles++
	return s.results, nil
}
func TestNativeReceiptPreservesPerZoneFailureAndPendingEvidence(t *testing.T) {
	for _, test := range []struct {
		name            string
		result          *controlv1.ZoneApplyResult
		failed, pending bool
	}{
		{name: "failed SDK command", result: &controlv1.ZoneApplyResult{Operation: controlv1.OperationState_OPERATION_STATE_FAILED, Reason: "SDK transport failed"}, failed: true},
		{name: "unacknowledged local write", result: &controlv1.ZoneApplyResult{Operation: controlv1.OperationState_OPERATION_STATE_COMMAND_SENT, LocalWriteCompleted: true}, failed: true},
		{name: "negative mode readback", result: &controlv1.ZoneApplyResult{Operation: controlv1.OperationState_OPERATION_STATE_BACKEND_ACKNOWLEDGED, LocalWriteCompleted: true, BackendAcknowledged: true, CachedModeReadback: controlv1.CachedModeReadback_CACHED_MODE_READBACK_MISMATCH}, failed: true},
		{name: "unsupported", result: &controlv1.ZoneApplyResult{Operation: controlv1.OperationState_OPERATION_STATE_UNSUPPORTED}, pending: true},
		{name: "disconnected controller", result: &controlv1.ZoneApplyResult{Operation: controlv1.OperationState_OPERATION_STATE_DEFERRED, Reason: "controller is not present"}, pending: true},
		{name: "acknowledged command", result: &controlv1.ZoneApplyResult{Operation: controlv1.OperationState_OPERATION_STATE_BACKEND_ACKNOWLEDGED, LocalWriteCompleted: true, BackendAcknowledged: true, CachedModeReadback: controlv1.CachedModeReadback_CACHED_MODE_READBACK_MATCHES}},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := &zoneNativeService{results: []*controlv1.ReconcileZoneResult{{ControllerId: "controller", ZoneId: "zone", Result: test.result}}}
			handler := newNativeHandler(nativeIdentity(), service)
			_, receipt := nativeCall(t, handler, nativeRequest("zone-result", 0, &lightingbinding.Binding{Address: "127.0.0.1:6742", Incarnation: "one"}))
			if (receipt.Error != "") != test.failed || receipt.State.ReconcilePending != test.pending || len(receipt.Zones) != 1 || receipt.Zones[0].Operation != int32(test.result.Operation) {
				t.Fatalf("truthful receipt %#v", receipt)
			}
			if test.failed && (receipt.State.Binding != nil || service.quiesces != 1) {
				t.Fatal("failed command established a healthy binding")
			}
			if !test.failed && (receipt.State.Binding == nil || service.quiesces != 0) {
				t.Fatal("pending/acknowledged state lost its binding")
			}
		})
	}
}

func TestNativeUnsupportedZoneKeepsSupportedBackendBound(t *testing.T) {
	service := &zoneNativeService{results: []*controlv1.ReconcileZoneResult{
		{ControllerId: "supported", ZoneId: "zone", Result: &controlv1.ZoneApplyResult{Operation: controlv1.OperationState_OPERATION_STATE_BACKEND_ACKNOWLEDGED, LocalWriteCompleted: true, BackendAcknowledged: true}},
		{ControllerId: "unsupported", ZoneId: "zone", Result: &controlv1.ZoneApplyResult{Operation: controlv1.OperationState_OPERATION_STATE_UNSUPPORTED, Reason: "device capability changed"}},
	}}
	handler := newNativeHandler(nativeIdentity(), service)
	_, receipt := nativeCall(t, handler, nativeRequest("mixed-capabilities", 0, &lightingbinding.Binding{Address: "127.0.0.1:6742", Incarnation: "one"}))
	if receipt.Error != "" || receipt.State.Binding == nil || !receipt.State.ReconcilePending || len(receipt.Zones) != 2 || service.quiesces != 0 {
		t.Fatalf("unsupported zone disabled supported backend: %#v %+v", receipt, service)
	}
}
