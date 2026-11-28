package execution

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"

	"swarm-manager/internal/workflowcontract"
)

// writeOffFixture returns a service whose store holds one goal execution that
// reserved an allowance and is stuck cancelling with unknown usage, like the
// qualification leftovers that motivated the write-off.
func writeOffFixture(t *testing.T) (*Service, backlogItem, Record, func() Record) {
	t.Helper()
	item, prior, newRecord := goalGrantFixture(t)
	root := t.TempDir()
	mustWriteBacklogItem(t, root, item.Kind, item.Name, map[string]any{"name": item.Name, "title": "Goal item", "status": "in_progress"})
	service := NewService(ServiceConfig{DataRoot: root, StorePath: filepath.Join(root, ".vrooli", "execution-runs.json"), TransitionRegistry: testTransitionRegistry(t)})
	prior.Status = StatusCancelling
	prior.Cancellation = &CancellationStanding{RequestID: "cancel-prior", RequestedAt: nowRFC3339(), AcknowledgedAt: nowRFC3339()}
	if err := service.store.Save([]Record{prior}); err != nil {
		t.Fatal(err)
	}
	return service, item, prior, newRecord
}

func storedRecord(t *testing.T, service *Service, id string) Record {
	t.Helper()
	records, idx, err := service.loadRecordLocked(id)
	if err != nil {
		t.Fatal(err)
	}
	return records[idx]
}

// Unknown usage must never be written off on assumption: every precondition
// failure leaves the cancellation exactly as it was.
func TestWriteOffCancellationRefusesWithoutProof(t *testing.T) {
	cases := map[string]struct {
		reader        stubGoalRunReader
		mutate        func(*Record)
		actor, reason string
	}{
		"live run":       {reader: stubGoalRunReader{terminal: false}, actor: "operator", reason: "unpriced model"},
		"unreadable run": {reader: stubGoalRunReader{usageErr: errors.New("agent-manager unavailable")}, actor: "operator", reason: "unpriced model"},
		"no reservation": {reader: stubGoalRunReader{terminal: true}, mutate: func(r *Record) { r.WorkflowGrant = nil }, actor: "operator", reason: "unpriced model"},
		"blank actor":    {reader: stubGoalRunReader{terminal: true}, actor: " ", reason: "unpriced model"},
		"blank reason":   {reader: stubGoalRunReader{terminal: true}, actor: "operator", reason: ""},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			service, _, prior, _ := writeOffFixture(t)
			if tc.mutate != nil {
				tc.mutate(&prior)
				if err := service.store.Save([]Record{prior}); err != nil {
					t.Fatal(err)
				}
			}
			service.goalRunReader = tc.reader
			if _, err := service.WriteOffCancellation(t.Context(), prior.ExecutionID, tc.actor, tc.reason); err == nil {
				t.Fatal("write-off succeeded without its preconditions")
			}
			after := storedRecord(t, service, prior.ExecutionID)
			if after.Status != prior.Status || after.SettledUsage != nil || (after.Cancellation != nil && after.Cancellation.WrittenOffAt != "") {
				t.Fatalf("refused write-off changed the record: %+v", after)
			}
		})
	}
}

func TestWriteOffCancellationRecoversInterruptedReservation(t *testing.T) {
	service, _, prior, _ := writeOffFixture(t)
	prior.Status = StatusInterrupted
	prior.Cancellation = nil
	if err := service.store.Save([]Record{prior}); err != nil {
		t.Fatal(err)
	}
	service.goalRunReader = stubGoalRunReader{terminal: true}
	if _, err := service.WriteOffCancellation(t.Context(), prior.ExecutionID, "operator", "owner terminated without terminal accounting"); err != nil {
		t.Fatalf("interrupted reservation should be recoverable: %v", err)
	}
	if got := storedRecord(t, service, prior.ExecutionID); got.Status != StatusCanceled || got.SettledUsage == nil {
		t.Fatalf("interrupted reservation was not written off: %+v", got)
	}
}

// A write-off charges the whole reservation, so the item's allowance treats
// the unknown usage as the most it could have been.
func TestWriteOffCancellationChargesTheFullReservation(t *testing.T) {
	service, item, prior, newRecord := writeOffFixture(t)
	service.goalRunReader = stubGoalRunReader{usage: &workflowcontract.Usage{Tokens: 12}, terminal: true}
	grant := prior.WorkflowGrant

	written, err := service.WriteOffCancellation(t.Context(), prior.ExecutionID, "operator", "model unpriced; charge can never be measured")
	if err != nil {
		t.Fatal(err)
	}
	usage := written.SettledUsage
	if written.Status != StatusCanceled || usage == nil || !usage.TokensKnown || !usage.ChargeMeasured ||
		usage.Tokens != grant.MaxTokens || usage.Turns != int64(grant.MaxTurns) || usage.WallSeconds != grant.MaxWallTimeSeconds ||
		usage.ChargeMicroUSD != grant.MaxChargeMicroUSD || usage.Children != int64(grant.MaxChildren) ||
		usage.NodeAttempts != int64(grant.MaxNodeAttempts) || usage.Retries != int64(grant.MaxRetries) || usage.Slices != int64(prior.MaxSlices) {
		t.Fatalf("write-off did not charge exactly the reservation: grant=%+v usage=%+v", grant, usage)
	}
	if c := written.Cancellation; c.WriteOffActor != "operator" || c.WriteOffReason == "" || c.WrittenOffAt == "" || c.SettledAt == "" ||
		!strings.Contains(written.FailureReason, "full reservation is charged as used") {
		t.Fatalf("write-off was not recorded: %+v %q", c, written.FailureReason)
	}

	next := newRecord()
	err = (&Service{}).prepareExecutionGrantLocked(t.Context(), []Record{written}, &next, item)
	if err == nil || !strings.Contains(err.Error(), "exhausted") {
		t.Fatalf("the written-off reservation was not counted as consumed: %v %+v", err, next.WorkflowGrant)
	}
}

// The automatic path stays strict: a reserved cancellation with unknown usage
// remains cancelling however long it waits.
func TestAutomaticCancelKeepsReservedUnknownUsagePending(t *testing.T) {
	service, _, prior, _ := writeOffFixture(t)
	service.goalRunReader = stubGoalRunReader{usage: &workflowcontract.Usage{Tokens: 12}, terminal: true}
	record, err := service.Cancel(t.Context(), prior.ExecutionID)
	if err != nil || record.Status != StatusCancelling || record.SettledUsage != nil || !strings.Contains(record.FailureReason, "reservation retained") {
		t.Fatalf("automatic cancellation released a reservation on unknown usage: %+v %v", record, err)
	}
}

// Agent Manager refuses to stop a run that already ended. That refusal must
// count as the stop acknowledgement, not strand the cancellation, while a
// failed stop on a live run still does.
func TestGoalCancellationTreatsStopOfEndedRunAsAcknowledged(t *testing.T) {
	service, _, prior, _ := writeOffFixture(t)
	prior.Cancellation.AcknowledgedAt = ""
	if err := service.store.Save([]Record{prior}); err != nil {
		t.Fatal(err)
	}
	stopper := &stubStopper{err: errors.New("status 409: STATE_TERMINAL")}
	service.stopper = stopper

	service.goalRunReader = stubGoalRunReader{terminal: false}
	pending, err := service.Cancel(t.Context(), prior.ExecutionID)
	if err != nil || pending.Status != StatusCancelling || pending.Cancellation.AcknowledgedAt != "" || !strings.Contains(pending.FailureReason, "goal run stop unavailable") {
		t.Fatalf("a failed stop on a live run was treated as acknowledged: %+v %v", pending, err)
	}

	service.goalRunReader = stubGoalRunReader{usage: completeGoalUsage(), terminal: true}
	settled, err := service.Cancel(t.Context(), prior.ExecutionID)
	if err != nil || settled.Status != StatusCanceled || settled.Cancellation.AcknowledgedAt == "" || settled.SettledUsage == nil || settled.SettledUsage.Tokens != completeGoalUsage().Tokens {
		t.Fatalf("stop refused for an ended run stranded the cancellation: %+v %v", settled, err)
	}
	if stopper.stopCalls != 2 {
		t.Fatalf("stop attempts = %d, want 2", stopper.stopCalls)
	}
}

// A cancelled goal execution has no workflow correlation. The maintenance
// cycle must still reconcile it (restore backlog status, stamp ReconciledAt)
// instead of aborting every cycle on the missing correlation.
func TestReconcileCancellationWithoutWorkflowCorrelation(t *testing.T) {
	service, _, prior, _ := writeOffFixture(t)
	service.goalRunReader = stubGoalRunReader{terminal: true}
	if _, err := service.WriteOffCancellation(t.Context(), prior.ExecutionID, "operator", "unknown usage"); err != nil {
		t.Fatal(err)
	}
	if err := service.reconcilePendingCancellations(t.Context()); err != nil {
		t.Fatalf("a cancellation without a workflow correlation aborted reconciliation: %v", err)
	}
	if after := storedRecord(t, service, prior.ExecutionID); after.Cancellation.ReconciledAt == "" {
		t.Fatalf("cancellation was not reconciled: %+v", after.Cancellation)
	}
}

func TestCancelHandlerWriteOffBody(t *testing.T) {
	service, _, prior, _ := writeOffFixture(t)
	handler := NewHandlerFromService(service)
	post := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/execution/"+prior.ExecutionID+"/cancel", strings.NewReader(body))
		req = mux.SetURLVars(req, map[string]string{"execution_id": prior.ExecutionID})
		rec := httptest.NewRecorder()
		handler.Cancel(rec, req)
		return rec
	}

	service.goalRunReader = stubGoalRunReader{terminal: false}
	if rec := post(`{"write_off":true,"actor":"operator","reason":"unpriced"}`); rec.Code != http.StatusConflict {
		t.Fatalf("live run write-off status = %d: %s", rec.Code, rec.Body.String())
	}
	if rec := post(`{"write_off":true,"actor":"operator"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("missing reason status = %d: %s", rec.Code, rec.Body.String())
	}
	if rec := post(`{not json`); rec.Code != http.StatusBadRequest {
		t.Fatalf("malformed body status = %d: %s", rec.Code, rec.Body.String())
	}
	service.goalRunReader = stubGoalRunReader{terminal: true}
	if rec := post(`{"write_off":true,"actor":"operator","reason":"unpriced"}`); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "written off") {
		t.Fatalf("write-off status = %d: %s", rec.Code, rec.Body.String())
	}
}
