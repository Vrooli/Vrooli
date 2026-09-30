package backlog

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"swarm-manager/internal/identity"
	"swarm-manager/internal/repairbudget"

	"github.com/gorilla/mux"
)

func newEffortControlRepairService(t *testing.T, effortID string) (*EffortControlService, *repairbudget.Ledger) {
	t.Helper()
	store := NewFileEffortControlStore(t.TempDir())
	source := &stubEffortPolicySource{
		bindings:   map[string]identity.PolicyBinding{effortID: {Source: "efforts/" + effortID + "/effort.json", Digest: "sha256:one"}},
		candidates: map[string]identity.CandidatePolicyBinding{effortID: testEffortCandidate("opencode-go/model-one")},
	}
	ledger := repairbudget.New(effortID, filepath.Join(t.TempDir(), "repair-ledger.json"))
	service := NewEffortControlService(store, source).WithRepairLedger(ledger)
	if _, err := service.Admit(testEffortControlRequest(effortID, 1)); err != nil {
		t.Fatalf("admit: %v", err)
	}
	return service, ledger
}

func TestBeginRepairUsesAdmittedEffortLimits(t *testing.T) {
	service, _ := newEffortControlRepairService(t, "aquila-one")
	event := repairbudget.Event{
		AttemptID:   "wake143-repair-1",
		Fingerprint: "swarm-manager/unit-phase-flake",
		Component:   "swarm-manager",
		Kind:        repairbudget.KindRepair,
	}
	admission, err := service.BeginRepair("aquila-one", event)
	if err != nil {
		t.Fatalf("begin repair: %v", err)
	}
	if !admission.Charged || admission.Totals.Effort != 1 {
		t.Fatalf("expected one cumulative charge, got %+v", admission)
	}
	replay, err := service.BeginRepair("aquila-one", event)
	if err != nil {
		t.Fatalf("replay repair: %v", err)
	}
	if replay.Charged || !replay.AlreadyRecorded {
		t.Fatalf("expected idempotent replay, got %+v", replay)
	}
	if _, err := service.FinishRepair("aquila-one", event.AttemptID, "failed", "evidence/x.md"); err != nil {
		t.Fatalf("finish repair: %v", err)
	}
	totals, err := service.RepairTotals("aquila-one")
	if err != nil {
		t.Fatalf("totals: %v", err)
	}
	if totals.Effort != 1 {
		t.Fatalf("cumulative totals changed after finish: %+v", totals)
	}
}

func TestBeginRepairRefusesUnknownEffort(t *testing.T) {
	service, _ := newEffortControlRepairService(t, "aquila-one")
	_, err := service.BeginRepair("aquila-two", repairbudget.Event{
		AttemptID:   "a1",
		Fingerprint: "fp",
		Component:   "swarm-manager",
		Kind:        repairbudget.KindRepair,
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not-found for an unknown effort, got %v", err)
	}
}

func TestRepairAccountingMustBeConfigured(t *testing.T) {
	store := NewFileEffortControlStore(t.TempDir())
	source := &stubEffortPolicySource{}
	service := NewEffortControlService(store, source)
	if _, err := service.BeginRepair("aquila-one", repairbudget.Event{}); !errors.Is(err, ErrRepairAccountingUnavailable) {
		t.Fatalf("expected unavailable accounting refusal, got %v", err)
	}
}

func newEffortControlRepairHandler(t *testing.T) *Handler {
	t.Helper()
	source := &stubEffortPolicySource{
		bindings:   map[string]identity.PolicyBinding{"aquila-one": {Source: "efforts/aquila-one/effort.json", Digest: "sha256:one"}},
		candidates: map[string]identity.CandidatePolicyBinding{"aquila-one": testEffortCandidate("opencode-go/model-one")},
	}
	h := NewHandler(t.TempDir(), t.TempDir())
	service := NewEffortControlService(NewFileEffortControlStore(t.TempDir()), source).
		WithRepairLedgerRoot(t.TempDir())
	h.SetEffortControlService(service)
	if recorder := doEffortJSON(t, h.AdmitEffort, http.MethodPost, "aquila-one", testEffortControlRequest("aquila-one", 1)); recorder.Code != http.StatusOK {
		t.Fatalf("admit: %d %s", recorder.Code, recorder.Body.String())
	}
	return h
}

func doEffortRepairFinishJSON(t *testing.T, handler http.HandlerFunc, effortID, attemptID string, body any) *httptest.ResponseRecorder {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/efforts/"+effortID+"/repairs/"+attemptID+"/finish", bytes.NewReader(encoded))
	req = mux.SetURLVars(req, map[string]string{"effortID": effortID, "attemptID": attemptID})
	recorder := httptest.NewRecorder()
	handler(recorder, req)
	return recorder
}

func TestEffortRepairReserveFinishTotalsOverHTTP(t *testing.T) {
	h := newEffortControlRepairHandler(t)
	reserve := doEffortJSON(t, h.BeginEffortRepair, http.MethodPost, "aquila-one", effortRepairReserveRequest{
		AttemptID:   "wake144-repair-1",
		Fingerprint: "swarm-manager/phase2-rest-surface",
		Component:   "swarm-manager",
		Kind:        repairbudget.KindRepair,
	})
	if reserve.Code != http.StatusOK {
		t.Fatalf("reserve: %d %s", reserve.Code, reserve.Body.String())
	}
	var reserved effortRepairReserveResponse
	if err := json.Unmarshal(reserve.Body.Bytes(), &reserved); err != nil {
		t.Fatal(err)
	}
	if !reserved.Admission.Charged || reserved.Admission.Totals.Effort != 1 {
		t.Fatalf("expected one cumulative charge, got %+v", reserved.Admission)
	}

	replay := doEffortJSON(t, h.BeginEffortRepair, http.MethodPost, "aquila-one", effortRepairReserveRequest{
		AttemptID:   "wake144-repair-1",
		Fingerprint: "swarm-manager/phase2-rest-surface",
		Component:   "swarm-manager",
		Kind:        repairbudget.KindRepair,
	})
	if replay.Code != http.StatusOK {
		t.Fatalf("replay reserve: %d %s", replay.Code, replay.Body.String())
	}

	finish := doEffortRepairFinishJSON(t, h.FinishEffortRepair, "aquila-one", "wake144-repair-1", effortRepairFinishRequest{Outcome: "verified_success"})
	if finish.Code != http.StatusOK {
		t.Fatalf("finish: %d %s", finish.Code, finish.Body.String())
	}

	totals := doEffortJSON(t, h.GetEffortRepairs, http.MethodGet, "aquila-one", nil)
	if totals.Code != http.StatusOK {
		t.Fatalf("totals: %d %s", totals.Code, totals.Body.String())
	}
	var summary effortRepairTotalsResponse
	if err := json.Unmarshal(totals.Body.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if summary.Totals.Effort != 1 {
		t.Fatalf("finish must not recharge; got %+v", summary.Totals)
	}
}

func TestEffortRepairLimitCrossingRefusedOverHTTP(t *testing.T) {
	h := newEffortControlRepairHandler(t)
	event := func(attemptID string) effortRepairReserveRequest {
		return effortRepairReserveRequest{
			AttemptID:   attemptID,
			Fingerprint: "swarm-manager/repeated-failure",
			Component:   "swarm-manager",
			Kind:        repairbudget.KindRepair,
			NewEvidence: true,
		}
	}
	if rec := doEffortJSON(t, h.BeginEffortRepair, http.MethodPost, "aquila-one", event("r1")); rec.Code != http.StatusOK {
		t.Fatalf("first reservation: %d %s", rec.Code, rec.Body.String())
	}
	if rec := doEffortJSON(t, h.BeginEffortRepair, http.MethodPost, "aquila-one", event("r2")); rec.Code != http.StatusOK {
		t.Fatalf("second reservation: %d %s", rec.Code, rec.Body.String())
	}
	third := doEffortJSON(t, h.BeginEffortRepair, http.MethodPost, "aquila-one", event("r3"))
	if third.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 at the fingerprint limit, got %d %s", third.Code, third.Body.String())
	}
}

func TestEffortRepairUnknownAttemptRefusedOverHTTP(t *testing.T) {
	h := newEffortControlRepairHandler(t)
	recorder := doEffortRepairFinishJSON(t, h.FinishEffortRepair, "aquila-one", "never-reserved", effortRepairFinishRequest{Outcome: "done"})
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for an unreserved attempt, got %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestEffortRepairClassSeparatesPlannedFromDiversionOverHTTP(t *testing.T) {
	h := newEffortControlRepairHandler(t)
	reserve := func(attemptID, fingerprint, class string) *httptest.ResponseRecorder {
		return doEffortJSON(t, h.BeginEffortRepair, http.MethodPost, "aquila-one", effortRepairReserveRequest{
			AttemptID:   attemptID,
			Fingerprint: fingerprint,
			Component:   "swarm-manager",
			Kind:        repairbudget.KindRepair,
			Class:       class,
		})
	}

	planned := reserve("wake150-planned-1", "swarm-manager/planned-infra", repairbudget.ClassPlanned)
	if planned.Code != http.StatusOK {
		t.Fatalf("planned reserve: %d %s", planned.Code, planned.Body.String())
	}
	var plannedBody effortRepairReserveResponse
	if err := json.Unmarshal(planned.Body.Bytes(), &plannedBody); err != nil {
		t.Fatal(err)
	}
	if plannedBody.Admission.Totals.Planned != 1 || plannedBody.Admission.Totals.Effort != 0 {
		t.Fatalf("planned work must not consume the diversion bound: %+v", plannedBody.Admission.Totals)
	}

	// Relabelling the planned fingerprint as diversion is refused.
	if rec := reserve("wake150-diversion-swap", "swarm-manager/planned-infra", repairbudget.ClassDiversion); rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 for a class swap, got %d %s", rec.Code, rec.Body.String())
	}

	diversion := reserve("wake150-diversion-1", "swarm-manager/unplanned-repair", repairbudget.ClassDiversion)
	if diversion.Code != http.StatusOK {
		t.Fatalf("diversion reserve: %d %s", diversion.Code, diversion.Body.String())
	}
	if rec := reserve("wake150-planned-swap", "swarm-manager/unplanned-repair", repairbudget.ClassPlanned); rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 for a reverse class swap, got %d %s", rec.Code, rec.Body.String())
	}

	totals := doEffortJSON(t, h.GetEffortRepairs, http.MethodGet, "aquila-one", nil)
	var summary effortRepairTotalsResponse
	if err := json.Unmarshal(totals.Body.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if summary.Totals.Planned != 1 || summary.Totals.Effort != 1 {
		t.Fatalf("expected 1 planned / 1 diversion, got %+v", summary.Totals)
	}
}

func doEffortRepairActionJSON(t *testing.T, handler http.HandlerFunc, method, effortID, attemptID, suffix string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(encoded)
	}
	path := "/api/v1/efforts/" + effortID + "/repairs/" + attemptID + suffix
	req := httptest.NewRequest(method, path, reader)
	req = mux.SetURLVars(req, map[string]string{"effortID": effortID, "attemptID": attemptID})
	recorder := httptest.NewRecorder()
	handler(recorder, req)
	return recorder
}

func reserveEffortRepair(t *testing.T, h *Handler, attemptID, fingerprint string) {
	t.Helper()
	rec := doEffortJSON(t, h.BeginEffortRepair, http.MethodPost, "aquila-one", effortRepairReserveRequest{
		AttemptID:   attemptID,
		Fingerprint: fingerprint,
		Component:   "swarm-manager",
		Kind:        repairbudget.KindRepair,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("reserve %s: %d %s", attemptID, rec.Code, rec.Body.String())
	}
}

func TestEffortRepairDispatchUncertainReconcileFallbackOverHTTP(t *testing.T) {
	h := newEffortControlRepairHandler(t)
	reserveEffortRepair(t, h, "wake145-op-1", "swarm-manager/uncertain-dispatch")

	if rec := doEffortRepairActionJSON(t, h.AcknowledgeEffortRepairDispatch, http.MethodPost, "aquila-one", "wake145-op-1", "/dispatch", effortRepairRouteRequest{OwnerWork: "swarm-manager/api", OwnerRun: "run-1"}); rec.Code != http.StatusOK {
		t.Fatalf("dispatch ack: %d %s", rec.Code, rec.Body.String())
	}
	// A second, different executor is a duplicate dispatch.
	if rec := doEffortRepairActionJSON(t, h.AcknowledgeEffortRepairDispatch, http.MethodPost, "aquila-one", "wake145-op-1", "/dispatch", effortRepairRouteRequest{OwnerWork: "swarm-manager/cli", OwnerRun: "run-2"}); rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 for a duplicate executor, got %d %s", rec.Code, rec.Body.String())
	}
	if rec := doEffortRepairActionJSON(t, h.MarkEffortRepairUncertain, http.MethodPost, "aquila-one", "wake145-op-1", "/uncertain", effortRepairReasonRequest{Reason: "start response lost"}); rec.Code != http.StatusOK {
		t.Fatalf("uncertain: %d %s", rec.Code, rec.Body.String())
	}
	// Replacement stays refused while uncertain.
	if rec := doEffortRepairActionJSON(t, h.FallbackEffortRepairDispatch, http.MethodPost, "aquila-one", "wake145-op-1", "/fallback", effortRepairRouteRequest{OwnerWork: "swarm-manager/cli", OwnerRun: "run-3"}); rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 while uncertain, got %d %s", rec.Code, rec.Body.String())
	}
	reconcile := doEffortRepairActionJSON(t, h.ReconcileEffortRepairDispatch, http.MethodPost, "aquila-one", "wake145-op-1", "/reconcile", effortRepairReconcileRequest{OwnerActive: false, Reason: "owner has no run"})
	if reconcile.Code != http.StatusOK {
		t.Fatalf("reconcile: %d %s", reconcile.Code, reconcile.Body.String())
	}
	var state effortRepairDispatchResponse
	if err := json.Unmarshal(reconcile.Body.Bytes(), &state); err != nil {
		t.Fatal(err)
	}
	if state.Dispatch.Status != repairbudget.DispatchReconciled || !state.Dispatch.ReplacementAllowed {
		t.Fatalf("expected a reconciled, replaceable operation, got %+v", state.Dispatch)
	}
	if rec := doEffortRepairActionJSON(t, h.FallbackEffortRepairDispatch, http.MethodPost, "aquila-one", "wake145-op-1", "/fallback", effortRepairRouteRequest{OwnerWork: "swarm-manager/cli", OwnerRun: "run-3"}); rec.Code != http.StatusOK {
		t.Fatalf("fallback: %d %s", rec.Code, rec.Body.String())
	}
	// Fallback debits the same remaining allowance (no second charge).
	totals := doEffortJSON(t, h.GetEffortRepairs, http.MethodGet, "aquila-one", nil)
	var summary effortRepairTotalsResponse
	if err := json.Unmarshal(totals.Body.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if summary.Totals.Effort != 1 {
		t.Fatalf("fallback must not recharge, got %+v", summary.Totals)
	}
	dispatched := doEffortRepairActionJSON(t, h.GetEffortRepairDispatch, http.MethodGet, "aquila-one", "wake145-op-1", "", nil)
	if dispatched.Code != http.StatusOK {
		t.Fatalf("get dispatch: %d %s", dispatched.Code, dispatched.Body.String())
	}
	if err := json.Unmarshal(dispatched.Body.Bytes(), &state); err != nil {
		t.Fatal(err)
	}
	if state.Dispatch.Status != repairbudget.DispatchActive || state.Dispatch.OwnerRun != "run-3" {
		t.Fatalf("expected fallback executor active, got %+v", state.Dispatch)
	}
}

func TestEffortRepairCancelFencesDelayedStartOverHTTP(t *testing.T) {
	h := newEffortControlRepairHandler(t)
	reserveEffortRepair(t, h, "wake145-op-2", "swarm-manager/cancel-fence")
	if rec := doEffortRepairActionJSON(t, h.CancelEffortRepairDispatch, http.MethodPost, "aquila-one", "wake145-op-2", "/cancel", effortRepairReasonRequest{Reason: "cancelled before owner start"}); rec.Code != http.StatusOK {
		t.Fatalf("cancel: %d %s", rec.Code, rec.Body.String())
	}
	if rec := doEffortRepairActionJSON(t, h.AcknowledgeEffortRepairDispatch, http.MethodPost, "aquila-one", "wake145-op-2", "/dispatch", effortRepairRouteRequest{OwnerWork: "swarm-manager/api", OwnerRun: "run-late"}); rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 for a delayed start, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestEffortRepairLedgersIsolatedPerEffort(t *testing.T) {
	source := &stubEffortPolicySource{
		bindings: map[string]identity.PolicyBinding{
			"effort-a": {Source: "efforts/effort-a/effort.json", Digest: "sha256:a"},
			"effort-b": {Source: "efforts/effort-b/effort.json", Digest: "sha256:b"},
		},
		candidates: map[string]identity.CandidatePolicyBinding{
			"effort-a": testEffortCandidate("opencode-go/model-a"),
			"effort-b": testEffortCandidate("opencode-go/model-b"),
		},
	}
	service := NewEffortControlService(NewFileEffortControlStore(t.TempDir()), source).
		WithRepairLedgerRoot(t.TempDir())
	for _, id := range []string{"effort-a", "effort-b"} {
		if _, err := service.Admit(testEffortControlRequest(id, 1)); err != nil {
			t.Fatalf("admit %s: %v", id, err)
		}
	}
	event := func(attemptID string) repairbudget.Event {
		return repairbudget.Event{AttemptID: attemptID, Fingerprint: "fp", Component: "swarm-manager", Kind: repairbudget.KindRepair}
	}
	if _, err := service.BeginRepair("effort-a", event("a1")); err != nil {
		t.Fatalf("begin effort-a: %v", err)
	}
	totalsB, err := service.RepairTotals("effort-b")
	if err != nil {
		t.Fatalf("totals effort-b: %v", err)
	}
	if totalsB.Effort != 0 {
		t.Fatalf("effort-b must not inherit effort-a investment: %+v", totalsB)
	}
}

func TestBeginRepairBindsEffortFractionToAggregateWallAllowance(t *testing.T) {
	store := NewFileEffortControlStore(t.TempDir())
	const effortID = "aquila-bounded"
	source := &stubEffortPolicySource{
		bindings:   map[string]identity.PolicyBinding{effortID: {Source: "efforts/" + effortID + "/effort.json", Digest: "sha256:one"}},
		candidates: map[string]identity.CandidatePolicyBinding{effortID: testEffortCandidate("opencode-go/model-one")},
	}
	control := testEffortControlRequest(effortID, 1)
	control.RepairLimits.ComponentActiveMinutes = 1000
	control.RepairLimits.PerFingerprint = 5
	control.RepairLimits.PerComponent = 5
	control.RepairLimits.PerEffort = 5
	control.RepairLimits.EffortFraction = 0.2
	control.AggregateLimits.MaxWallSeconds = 600 // 10 minutes at 20% = 2 diversion minutes
	service := NewEffortControlService(store, source)
	if _, err := service.Admit(control); err != nil {
		t.Fatalf("admit: %v", err)
	}
	now := time.Date(2026, 9, 13, 18, 0, 0, 0, time.UTC)
	ledger := repairbudget.New(effortID, filepath.Join(t.TempDir(), "repair-ledger.json"))
	ledger.SetClock(func() time.Time { return now })
	service.WithRepairLedger(ledger)

	if _, err := service.BeginRepair(effortID, repairbudget.Event{
		AttemptID: "a1", Fingerprint: "fp-1", Component: "swarm-manager", Kind: repairbudget.KindRepair,
	}); err != nil {
		t.Fatalf("first repair: %v", err)
	}
	now = now.Add(3 * time.Minute)
	if _, err := service.BeginRepair(effortID, repairbudget.Event{
		AttemptID: "a2", Fingerprint: "fp-2", Component: "prompt-manager", Kind: repairbudget.KindRepair,
	}); !errors.Is(err, repairbudget.ErrBudgetExhausted) {
		t.Fatalf("expected the aggregate wall allowance to bound diversion time, got %v", err)
	}
}
