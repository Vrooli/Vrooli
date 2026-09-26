package readiness

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vrooli-bridge/internal/auth"
	"vrooli-bridge/internal/hostbroker"
	"vrooli-bridge/internal/onboard"
	internalreadiness "vrooli-bridge/internal/readiness"

	"github.com/gorilla/mux"
	_ "modernc.org/sqlite"
)

type readyPinger struct{ err error }

func (p readyPinger) PingContext(context.Context) error { return p.err }

type stubOnboard struct {
	onboard.Service
	ops    []onboard.Op
	events []onboard.StepEvent
}

type stubBroker struct {
	result   hostbroker.Result
	requests []hostbroker.Request
	err      error
}

func (s *stubBroker) Call(_ context.Context, request hostbroker.Request) (hostbroker.Result, error) {
	s.requests = append(s.requests, request)
	return s.result, s.err
}

func TestConfigureEndpointPersistsAndRejectsUnsafeURL(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(internalreadiness.Schema()); err != nil {
		t.Fatal(err)
	}
	store := internalreadiness.NewStore(db, internalreadiness.Endpoint{URL: "http://192.168.1.173:18767", Mode: "lan", Source: "derived"})
	router := mux.NewRouter()
	Module(readyPinger{}, stubOnboard{}, store, true).Mount(router)
	request := httptest.NewRequest(http.MethodPut, "/api/v1/readiness/endpoint", strings.NewReader(`{"endpoint":"https://bridge.example.test","reachability_mode":"manual"}`))
	request = request.WithContext(auth.WithIdentity(request.Context(), auth.Identity{OwnerID: "owner"}))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("configure status=%d body=%s", response.Code, response.Body.String())
	}
	selected, err := store.Resolve(context.Background())
	if err != nil || selected.URL != "https://bridge.example.test" || selected.Mode != "manual" {
		t.Fatalf("saved=%#v err=%v", selected, err)
	}
	unsafe := httptest.NewRequest(http.MethodPut, "/api/v1/readiness/endpoint", strings.NewReader(`{"endpoint":"http://127.0.0.1:18767","reachability_mode":"lan"}`))
	unsafe = unsafe.WithContext(auth.WithIdentity(unsafe.Context(), auth.Identity{OwnerID: "owner"}))
	blocked := httptest.NewRecorder()
	router.ServeHTTP(blocked, unsafe)
	if blocked.Code != http.StatusBadRequest {
		t.Fatalf("unsafe status=%d", blocked.Code)
	}
}

func TestLegacySSHConnectionSourceDoesNotProduceSelfIPFirewallRule(t *testing.T) {
	got := remediationCandidateIP("192.168.1.176", "192.168.1.173", "http://192.168.1.173:18767")
	if got != "192.168.1.176" {
		t.Fatalf("candidate remediation IP = %q", got)
	}
}

func (s stubOnboard) ListOps(context.Context, onboard.ListFilter) ([]onboard.Op, error) {
	return s.ops, nil
}

func (s stubOnboard) GetOp(context.Context, string) (onboard.Op, []onboard.StepEvent, error) {
	return s.ops[0], s.events, nil
}

func TestReadinessIsOwnerGatedAndReportsDurableAdmission(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(internalreadiness.Schema()); err != nil {
		t.Fatal(err)
	}
	store := internalreadiness.NewStore(db, internalreadiness.Endpoint{URL: "http://192.168.1.173:18767", Mode: "lan", Source: "configured"})
	broker := &stubBroker{result: hostbroker.Result{Version: "v1", Status: "verified", Evidence: hostbroker.Evidence{Available: true, Active: true}}}
	mod := Module(readyPinger{}, stubOnboard{ops: []onboard.Op{{ID: "op", Host: "mini", ControlPlaneURL: "http://192.168.1.173:18767", ReachabilityMode: "lan", State: onboard.StateFailed, FailureReason: onboard.FailureControlPlaneUnreachable}}, events: []onboard.StepEvent{{StepID: onboard.StepAdmission, Detail: "endpoint http://192.168.1.173:18767; candidate source 192.168.1.176; curl-exit-28"}}}, store, true, broker)
	router := mux.NewRouter()
	mod.Mount(router)
	unauth := httptest.NewRecorder()
	router.ServeHTTP(unauth, httptest.NewRequest(http.MethodGet, "/api/v1/readiness", nil))
	if unauth.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status=%d", unauth.Code)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/readiness", nil)
	request = request.WithContext(auth.WithIdentity(request.Context(), auth.Identity{OwnerID: "owner"}))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	for _, expected := range []string{"\"status\":\"candidate_blocked\"", "\"endpoint\":\"http://192.168.1.173:18767\"", "\"listener\":true", "\"auth\":true", "\"storage\":true", "\"keys\":true", "\"host\":\"mini\"", "\"broker_available\":true"} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Fatalf("body %q missing %q", response.Body.String(), expected)
		}
	}
}

func TestReadinessDoesNotMislabelNonFirewallOnboardingFailureAsCandidateBlocked(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(internalreadiness.Schema()); err != nil {
		t.Fatal(err)
	}
	store := internalreadiness.NewStore(db, internalreadiness.Endpoint{URL: "http://192.168.1.173:18767", Mode: "lan", Source: "configured"})
	router := mux.NewRouter()
	Module(readyPinger{}, stubOnboard{ops: []onboard.Op{{ID: "op", Host: "mini", ControlPlaneURL: "http://192.168.1.173:18767", State: onboard.StateFailed, FailureReason: onboard.FailureEndpointNameUnresolvable}}}, store, true).Mount(router)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/readiness", nil).WithContext(auth.WithIdentity(context.Background(), auth.Identity{OwnerID: "owner"}))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), `"status":"candidate_blocked"`) {
		t.Fatalf("non-firewall admission failure was mislabeled: %s", response.Body.String())
	}
}

func TestFirewallMutationRequiresOwnerAndConfirmation(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(internalreadiness.Schema()); err != nil {
		t.Fatal(err)
	}
	broker := &stubBroker{result: hostbroker.Result{Version: "v1", Status: "changed", Changed: true}}
	onboardStub := stubOnboard{ops: []onboard.Op{{ID: "op", Host: "192.168.1.176", ControlPlaneURL: "http://192.168.1.173:18767", State: onboard.StateFailed, FailureReason: onboard.FailureControlPlaneUnreachable}}, events: []onboard.StepEvent{{StepID: onboard.StepAdmission, Detail: "candidate source 192.168.1.176"}}}
	router := mux.NewRouter()
	Module(readyPinger{}, onboardStub, internalreadiness.NewStore(db, internalreadiness.Endpoint{}), true, broker).Mount(router)
	noConfirm := httptest.NewRequest(http.MethodPost, "/api/v1/readiness/firewall", strings.NewReader(`{"action":"allow","candidate_ip":"192.168.1.176"}`)).WithContext(auth.WithIdentity(context.Background(), auth.Identity{OwnerID: "owner"}))
	blocked := httptest.NewRecorder()
	router.ServeHTTP(blocked, noConfirm)
	if blocked.Code != http.StatusBadRequest || len(broker.requests) != 0 {
		t.Fatalf("blocked=%d requests=%d", blocked.Code, len(broker.requests))
	}
	confirmed := httptest.NewRequest(http.MethodPost, "/api/v1/readiness/firewall", strings.NewReader(`{"action":"allow","confirm":true,"candidate_ip":"192.168.1.176"}`)).WithContext(auth.WithIdentity(context.Background(), auth.Identity{OwnerID: "owner"}))
	ok := httptest.NewRecorder()
	router.ServeHTTP(ok, confirmed)
	if ok.Code != http.StatusOK || len(broker.requests) != 1 || broker.requests[0].Action != "bridge.ufw.allow" {
		t.Fatalf("status=%d requests=%#v", ok.Code, broker.requests)
	}
	wrong := httptest.NewRequest(http.MethodPost, "/api/v1/readiness/firewall", strings.NewReader(`{"action":"allow","confirm":true,"candidate_ip":"192.168.1.177"}`)).WithContext(auth.WithIdentity(context.Background(), auth.Identity{OwnerID: "owner"}))
	wrongResponse := httptest.NewRecorder()
	router.ServeHTTP(wrongResponse, wrong)
	if wrongResponse.Code != http.StatusBadRequest || len(broker.requests) != 1 {
		t.Fatalf("wrong=%d requests=%d", wrongResponse.Code, len(broker.requests))
	}
	preview := httptest.NewRequest(http.MethodPost, "/api/v1/readiness/firewall", strings.NewReader(`{"action":"preview","candidate_ip":"192.168.1.176"}`)).WithContext(auth.WithIdentity(context.Background(), auth.Identity{OwnerID: "owner"}))
	previewResponse := httptest.NewRecorder()
	router.ServeHTTP(previewResponse, preview)
	if previewResponse.Code != http.StatusOK || len(broker.requests) != 1 || !strings.Contains(previewResponse.Body.String(), `"status":"preview"`) {
		t.Fatalf("preview=%d body=%s requests=%d", previewResponse.Code, previewResponse.Body.String(), len(broker.requests))
	}
}
