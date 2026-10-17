package validationbroker

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	"google.golang.org/protobuf/types/known/durationpb"
	"test-genie/internal/storage/sqliteutil"
)

type fakeAborter struct{ calls atomic.Int32 }

type recoveringProducer struct {
	calls   atomic.Int32
	entered chan struct{}
	release chan struct{}
}

func (p *recoveringProducer) ExecuteValidation(ctx context.Context, receipt *validationv1.ValidationReceipt, intent *validationv1.ValidationIntent, transition TransitionFunc) error {
	if p.calls.Add(1) == 1 {
		close(p.entered)
		<-p.release
		return errEvidencePending
	}
	_, err := transition(ctx, receipt.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED, nil)
	return err
}

func TestServiceReattachesObserverAndDeduplicatesRecoveryDrivers(t *testing.T) {
	service := NewService(NewRepository(testsqllite(t)), nil)
	producer := &recoveringProducer{entered: make(chan struct{}), release: make(chan struct{})}
	service.SetProducer(producer)
	service.reattachDelay = time.Millisecond
	receipt := createThroughService(t, service, "recover-observer")
	<-producer.entered
	for i := 0; i < 2; i++ {
		count, err := service.Recover(context.Background())
		if err != nil || count != 0 {
			t.Fatalf("duplicate recovery: started=%d err=%v", count, err)
		}
	}
	close(producer.release)
	response, err := service.WaitValidation(context.Background(), connect.NewRequest(&validationv1.WaitValidationRequest{ReceiptId: receipt.GetReceiptId(), WaitId: "recovery-test", Timeout: durationpb.New(time.Second)}))
	if err != nil || response.Msg.GetReceipt().GetState() != validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED {
		t.Fatalf("reattachment failed: response=%v err=%v", response, err)
	}
	if producer.calls.Load() != 2 {
		t.Fatalf("producer calls=%d, want one original and one reattachment", producer.calls.Load())
	}
	history, err := service.repo.History(context.Background(), receipt.GetReceiptId())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, event := range history {
		if event.To == validationv1.ReceiptState_RECEIPT_STATE_FAILED {
			t.Fatal("observation error was recorded as a producer failure")
		}
		found = found || event.Receipt.GetRetry().GetRetryAt().IsValid()
	}
	if !found {
		t.Fatal("reattachment schedule was not durable")
	}
}

func (f *fakeAborter) AbortValidation(context.Context, *validationv1.ValidationReceipt, string, string) error {
	f.calls.Add(1)
	return nil
}

func createThroughService(t *testing.T, service *Service, key string) *validationv1.ValidationReceipt {
	t.Helper()
	response, err := service.CreateValidation(context.Background(), connect.NewRequest(&validationv1.CreateValidationRequest{Intent: validIntent("plan-manager", key)}))
	if err != nil {
		t.Fatal(err)
	}
	return response.Msg.GetReceipt()
}

func TestWaitWakesOnDurableTransitionWithoutPolling(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-RECEIPT-P0]
	service := NewService(NewRepository(testsqllite(t)), nil)
	receipt := createThroughService(t, service, "wait-transition")
	result := make(chan *validationv1.WaitValidationResponse, 1)
	errs := make(chan error, 1)
	go func() {
		response, err := service.WaitValidation(context.Background(), connect.NewRequest(&validationv1.WaitValidationRequest{ReceiptId: receipt.GetReceiptId(), WaitId: "observer-1", AfterRevision: receipt.GetRevision(), Timeout: durationpb.New(time.Second)}))
		if response != nil {
			result <- response.Msg
		}
		errs <- err
	}()
	waitForRegistration(t, service, receipt.GetReceiptId(), "observer-1")
	updated, err := service.Transition(context.Background(), receipt.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_RUNNING, nil)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-errs:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("wait did not wake")
	}
	response := <-result
	if response.GetTimedOut() || response.GetWaitCancelled() || response.GetReceipt().GetRevision() != updated.GetRevision() {
		t.Fatalf("wait response = %#v", response)
	}
}

func TestWaitWithoutRevisionCursorBlocksThroughNonterminalProgress(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-RECEIPT-P0]
	service := NewService(NewRepository(testsqllite(t)), nil)
	receipt := createThroughService(t, service, "wait-terminal")
	result := make(chan *validationv1.WaitValidationResponse, 1)
	errs := make(chan error, 1)
	go func() {
		response, err := service.WaitValidation(context.Background(), connect.NewRequest(&validationv1.WaitValidationRequest{ReceiptId: receipt.GetReceiptId(), WaitId: "terminal-observer", Timeout: durationpb.New(time.Second)}))
		if response != nil {
			result <- response.Msg
		}
		errs <- err
	}()
	waitForRegistration(t, service, receipt.GetReceiptId(), "terminal-observer")
	if _, err := service.Transition(context.Background(), receipt.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_RUNNING, nil); err != nil {
		t.Fatal(err)
	}
	select {
	case response := <-result:
		t.Fatalf("terminal wait returned on nonterminal progress: %#v", response)
	case <-time.After(25 * time.Millisecond):
	}
	if _, err := service.Transition(context.Background(), receipt.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED, nil); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-errs:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("terminal wait did not wake")
	}
	if response := <-result; response.GetReceipt().GetState() != validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED || response.GetTimedOut() || response.GetWaitCancelled() {
		t.Fatalf("terminal wait response = %#v", response)
	}
}

func TestCancelWaitDetachesObserverWithoutCancellingWork(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-RECEIPT-P0]
	service := NewService(NewRepository(testsqllite(t)), nil)
	receipt := createThroughService(t, service, "cancel-wait")
	result := make(chan *validationv1.WaitValidationResponse, 1)
	go func() {
		response, _ := service.WaitValidation(context.Background(), connect.NewRequest(&validationv1.WaitValidationRequest{ReceiptId: receipt.GetReceiptId(), WaitId: "observer-2", AfterRevision: receipt.GetRevision(), Timeout: durationpb.New(time.Second)}))
		result <- response.Msg
	}()
	waitForRegistration(t, service, receipt.GetReceiptId(), "observer-2")
	cancelled, err := service.CancelValidationWait(context.Background(), connect.NewRequest(&validationv1.CancelValidationWaitRequest{ReceiptId: receipt.GetReceiptId(), WaitId: "observer-2"}))
	if err != nil || !cancelled.Msg.GetCancelled() {
		t.Fatalf("cancel wait = %#v err=%v", cancelled, err)
	}
	response := <-result
	if !response.GetWaitCancelled() || response.GetReceipt().GetState() != validationv1.ReceiptState_RECEIPT_STATE_ADMITTED {
		t.Fatalf("cancelled wait changed work: %#v", response)
	}
	stored, err := service.repo.Get(context.Background(), receipt.GetReceiptId())
	if err != nil || stored.GetState() != validationv1.ReceiptState_RECEIPT_STATE_ADMITTED {
		t.Fatalf("stored work = %#v err=%v", stored, err)
	}
}

func TestAbortWorkUsesActuatorAndTerminalizesReceipt(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-RECEIPT-P0]
	aborter := &fakeAborter{}
	service := NewService(NewRepository(testsqllite(t)), aborter)
	receipt := createThroughService(t, service, "abort-work")
	response, err := service.AbortValidationWork(context.Background(), connect.NewRequest(&validationv1.AbortValidationWorkRequest{ReceiptId: receipt.GetReceiptId(), Reason: "operator requested stop", RequestedBy: "operator"}))
	if err != nil {
		t.Fatal(err)
	}
	if aborter.calls.Load() != 1 || response.Msg.GetReceipt().GetState() != validationv1.ReceiptState_RECEIPT_STATE_CANCELLED || response.Msg.GetReceipt().GetReasonCode() != validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_ABORTED {
		t.Fatalf("abort response = %#v calls=%d", response.Msg, aborter.calls.Load())
	}
}

func TestHistoricalShadowReadDoesNotRequireRetiredWriters(t *testing.T) {
	db := testsqllite(t)
	service := NewService(NewRepository(db), nil)
	receipt := createThroughService(t, service, "historical-shadow-read")
	observedAt := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	_, err := db.Exec(`INSERT INTO validation_shadow_comparisons
        (comparison_id, source_kind, source_id, receipt_id, receipt_revision, legacy_state, receipt_state, matched, reason_code, legacy_evidence_count, receipt_evidence_count, observed_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"shadow:historical-1", "retired-owner", "legacy-1", receipt.GetReceiptId(), receipt.GetRevision(), "passed", receipt.GetState(), true, "state_equivalent", 1, 1, sqliteutil.FormatTimestamp(observedAt))
	if err != nil {
		t.Fatal(err)
	}
	response, err := service.ListValidationShadows(context.Background(), connect.NewRequest(&validationv1.ListValidationShadowsRequest{PageSize: 10}))
	if err != nil {
		t.Fatal(err)
	}
	comparisons := response.Msg.GetComparisons()
	if len(comparisons) != 1 || comparisons[0].GetComparisonId() != "shadow:historical-1" || !comparisons[0].GetMatched() || comparisons[0].GetObservedAt().AsTime() != observedAt {
		t.Fatalf("historical shadow = %#v", comparisons)
	}
}

func TestWaitReturnsCurrentStateAtBoundedDeadline(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-RECEIPT-P0]
	service := NewService(NewRepository(testsqllite(t)), nil)
	receipt := createThroughService(t, service, "wait-timeout")
	started := time.Now()
	response, err := service.WaitValidation(context.Background(), connect.NewRequest(&validationv1.WaitValidationRequest{ReceiptId: receipt.GetReceiptId(), WaitId: "observer-3", AfterRevision: receipt.GetRevision(), Timeout: durationpb.New(20 * time.Millisecond)}))
	if err != nil {
		t.Fatal(err)
	}
	if !response.Msg.GetTimedOut() || time.Since(started) > 500*time.Millisecond {
		t.Fatalf("bounded wait = %#v duration=%s", response.Msg, time.Since(started))
	}
}

func TestProducerTerminalTransitionCompletesAttachedReceipts(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-RECEIPT-P0]
	service := NewService(NewRepository(testsqllite(t)), nil)
	producer := createThroughService(t, service, "producer")
	attached := createThroughService(t, service, "attached")
	if attached.GetState() != validationv1.ReceiptState_RECEIPT_STATE_ATTACHED || attached.GetLineageId() != producer.GetLineageId() {
		t.Fatalf("attached admission = %#v", attached)
	}
	if _, err := service.Transition(context.Background(), producer.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_RUNNING, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Transition(context.Background(), producer.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED, func(receipt *validationv1.ValidationReceipt) error {
		receipt.AchievedStrength = validationv1.ValidationStrength_VALIDATION_STRENGTH_TARGETED
		receipt.Evidence = []*validationv1.EvidenceReference{{EvidenceId: "run-1", Kind: "test-genie-run"}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	stored, err := service.repo.Get(context.Background(), attached.GetReceiptId())
	if err != nil {
		t.Fatal(err)
	}
	if stored.GetState() != validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED || stored.GetAchievedStrength() != validationv1.ValidationStrength_VALIDATION_STRENGTH_TARGETED || len(stored.GetEvidence()) != 1 {
		t.Fatalf("attached terminal receipt = %#v", stored)
	}
}

func waitForRegistration(t *testing.T, service *Service, receiptID, waitID string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		service.mu.Lock()
		_, ok := service.waiters[receiptID][waitID]
		service.mu.Unlock()
		if ok {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("waiter did not register")
}
