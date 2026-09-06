package validationbroker

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation/validation_v1connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const maximumWait = 30 * time.Minute

type waitSignal int

const (
	waitChanged waitSignal = iota + 1
	waitCancelled
)

type WorkAborter interface {
	AbortValidation(context.Context, *validationv1.ValidationReceipt, string, string) error
}

type TransitionFunc func(context.Context, string, validationv1.ReceiptState, func(*validationv1.ValidationReceipt) error) (*validationv1.ValidationReceipt, error)

// WorkProducer owns server-lifetime validation execution. ExecuteValidation is
// invoked only for the one receipt admitted as the lineage producer.
type WorkProducer interface {
	ExecuteValidation(context.Context, *validationv1.ValidationReceipt, *validationv1.ValidationIntent, TransitionFunc) error
}

// Service is the receipt observation and control surface. Producer adapters
// call Transition after durable child progress; all waiting is notification-
// driven and a client context never owns producer work.
type Service struct {
	validationconnect.UnimplementedValidationServiceHandler
	repo          ReceiptRepository
	aborter       WorkAborter
	producer      WorkProducer
	identity      IdentityResolver
	shadows       ShadowRepository
	mu            sync.Mutex
	waiters       map[string]map[string]chan waitSignal
	drivers       map[string]bool
	reattachDelay time.Duration
}

type ShadowRepository interface {
	ListShadowComparisons(context.Context, int) ([]ShadowComparison, error)
}

func NewService(repo ReceiptRepository, aborter WorkAborter) *Service {
	service := &Service{repo: repo, aborter: aborter, waiters: map[string]map[string]chan waitSignal{}, drivers: map[string]bool{}, reattachDelay: 30 * time.Second}
	service.shadows, _ = repo.(ShadowRepository)
	return service
}

func (s *Service) SetProducer(producer WorkProducer) {
	s.producer = producer
}

func (s *Service) SetIdentityResolver(identity IdentityResolver) {
	s.identity = identity
}

func (s *Service) CreateValidation(ctx context.Context, req *connect.Request[validationv1.CreateValidationRequest]) (*connect.Response[validationv1.CreateValidationResponse], error) {
	intent := req.Msg.GetIntent()
	if intent != nil {
		intent = proto.Clone(intent).(*validationv1.ValidationIntent)
	}
	if s.identity != nil && intent != nil {
		expected := intent.GetExpectedIdentity()
		resolved, resolveErr := s.identity.Resolve(ctx, intent)
		if resolveErr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%w: resolve content inputs: %v", ErrInvalidIntent, resolveErr))
		}
		if expected != nil && strings.TrimSpace(expected.GetIdentity()) != "" && expected.GetIdentity() != resolved.GetIdentity() {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("expected content identity %s but resolved %s", expected.GetIdentity(), resolved.GetIdentity()))
		}
		intent.ExpectedIdentity = resolved
	}
	intent, err := normalizeIntent(intent)
	if err != nil {
		return nil, connectError(err)
	}
	admission, err := s.repo.Admit(ctx, intent)
	if err != nil {
		return nil, connectError(err)
	}
	if admission.Kind == AdmissionNew && s.producer != nil {
		queued, transitionErr := s.Transition(context.WithoutCancel(ctx), admission.Receipt.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_QUEUED, nil)
		if transitionErr != nil {
			return nil, connectError(transitionErr)
		}
		admission.Receipt = queued
		s.startProducer(admission.Receipt, intent)
	}
	return connect.NewResponse(&validationv1.CreateValidationResponse{Receipt: admission.Receipt}), nil
}

// Recovery and admission may race within the owner process. Register before
// launching so they cannot attach two drivers to the same durable child.
func (s *Service) startProducer(receipt *validationv1.ValidationReceipt, intent *validationv1.ValidationIntent) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := receipt.GetReceiptId()
	if s.drivers[id] {
		return false
	}
	s.drivers[id] = true
	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.drivers, id)
			s.mu.Unlock()
		}()
		s.driveProducer(receipt, intent)
	}()
	return true
}

func (s *Service) driveProducer(receipt *validationv1.ValidationReceipt, intent *validationv1.ValidationIntent) {
	receiptID := receipt.GetReceiptId()
	var err error
	for {
		if terminal(receipt.GetState()) {
			return
		}
		if retryAt := receipt.GetRetry().GetRetryAt(); retryAt.IsValid() {
			if delay := time.Until(retryAt.AsTime()); delay > 0 {
				timer := time.NewTimer(delay)
				<-timer.C
			}
		}
		// Abort may have won while this driver was waiting to reattach.
		receipt, err = s.repo.Get(context.Background(), receiptID)
		if err != nil || terminal(receipt.GetState()) {
			return
		}
		err = s.producer.ExecuteValidation(context.Background(), receipt, intent, s.Transition)
		if !errors.Is(err, errEvidencePending) {
			break
		}
		detail := err.Error()
		receipt, err = s.Transition(context.Background(), receiptID, validationv1.ReceiptState_RECEIPT_STATE_RUNNING, func(value *validationv1.ValidationReceipt) error {
			value.Detail = detail
			value.ReasonCode = validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_PROVIDER_UNAVAILABLE
			value.Retry = &validationv1.RetryDisposition{Kind: validationv1.RetryKind_RETRY_KIND_SCHEDULED, RetryAt: timestamppb.New(time.Now().Add(s.reattachDelay)), ReasonCode: validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_PROVIDER_UNAVAILABLE}
			return nil
		})
		if err != nil {
			return
		}
	}
	if err == nil {
		return
	}
	receipt, getErr := s.repo.Get(context.Background(), receiptID)
	if getErr != nil || terminal(receipt.GetState()) {
		return
	}
	_, _ = s.Transition(context.Background(), receiptID, validationv1.ReceiptState_RECEIPT_STATE_FAILED, func(value *validationv1.ValidationReceipt) error {
		value.ReasonCode = validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_PROVIDER_UNAVAILABLE
		value.Detail = err.Error()
		return nil
	})
}

// Recover resumes every durable producer that was non-terminal when this
// service instance started. The producer adapter reattaches recorded children;
// it never starts a replacement for an already-recorded operation.
func (s *Service) Recover(ctx context.Context) (int, error) {
	if s.producer == nil {
		return 0, nil
	}
	active, err := s.repo.ActiveProducers(ctx)
	if err != nil {
		return 0, err
	}
	started := 0
	for _, item := range active {
		intent, err := normalizeIntent(item.Intent)
		if err != nil {
			return 0, err
		}
		if s.startProducer(item.Receipt, intent) {
			started++
		}
	}
	return started, nil
}

func (s *Service) GetValidation(ctx context.Context, req *connect.Request[validationv1.GetValidationRequest]) (*connect.Response[validationv1.GetValidationResponse], error) {
	receipt, err := s.repo.Get(ctx, req.Msg.GetReceiptId())
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&validationv1.GetValidationResponse{Receipt: receipt}), nil
}

func (s *Service) WaitValidation(ctx context.Context, req *connect.Request[validationv1.WaitValidationRequest]) (*connect.Response[validationv1.WaitValidationResponse], error) {
	receiptID := strings.TrimSpace(req.Msg.GetReceiptId())
	waitID := strings.TrimSpace(req.Msg.GetWaitId())
	if receiptID == "" || waitID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("receipt_id and wait_id are required"))
	}
	receipt, err := s.repo.Get(ctx, receiptID)
	if err != nil {
		return nil, connectError(err)
	}
	afterRevision := req.Msg.GetAfterRevision()
	if terminal(receipt.GetState()) || (afterRevision > 0 && receipt.GetRevision() > afterRevision) {
		return connect.NewResponse(&validationv1.WaitValidationResponse{Receipt: receipt}), nil
	}
	signal, err := s.registerWaiter(receiptID, waitID)
	if err != nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, err)
	}
	defer s.removeWaiter(receiptID, waitID)
	// Close the read/register race without polling.
	receipt, err = s.repo.Get(ctx, receiptID)
	if err != nil {
		return nil, connectError(err)
	}
	if terminal(receipt.GetState()) || (afterRevision > 0 && receipt.GetRevision() > afterRevision) {
		return connect.NewResponse(&validationv1.WaitValidationResponse{Receipt: receipt}), nil
	}
	waitFor := 30 * time.Second
	if req.Msg.GetTimeout() != nil && req.Msg.GetTimeout().AsDuration() > 0 {
		waitFor = req.Msg.GetTimeout().AsDuration()
	}
	if waitFor > maximumWait {
		waitFor = maximumWait
	}
	timer := time.NewTimer(waitFor)
	defer timer.Stop()
	for {
		response := &validationv1.WaitValidationResponse{}
		select {
		case kind := <-signal:
			response.WaitCancelled = kind == waitCancelled
		case <-timer.C:
			response.TimedOut = true
		case <-ctx.Done():
			return nil, connect.NewError(connect.CodeCanceled, ctx.Err())
		}
		receipt, err = s.repo.Get(context.WithoutCancel(ctx), receiptID)
		if err != nil {
			return nil, connectError(err)
		}
		response.Receipt = receipt
		if response.GetWaitCancelled() || response.GetTimedOut() || terminal(receipt.GetState()) || (afterRevision > 0 && receipt.GetRevision() > afterRevision) {
			return connect.NewResponse(response), nil
		}
		// With no revision cursor, WaitValidation is the terminal await used by
		// generated operator actions. Nonterminal notifications are progress,
		// not a reason to make the caller poll or recursively reattach.
	}
}

func (s *Service) ListValidations(ctx context.Context, req *connect.Request[validationv1.ListValidationsRequest]) (*connect.Response[validationv1.ListValidationsResponse], error) {
	offset, err := ParsePageToken(req.Msg.GetPageToken())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	receipts, next, err := s.repo.List(ctx, ListFilter{CallerScenario: strings.TrimSpace(req.Msg.GetCallerScenario()), CallerExecutionID: strings.TrimSpace(req.Msg.GetCallerExecutionId()), PlanID: strings.TrimSpace(req.Msg.GetPlanId()), State: req.Msg.GetState(), Limit: int(req.Msg.GetPageSize()), Offset: offset})
	if err != nil {
		return nil, connectError(err)
	}
	nextToken := ""
	if next > 0 {
		nextToken = strconv.Itoa(next)
	}
	return connect.NewResponse(&validationv1.ListValidationsResponse{Receipts: receipts, NextPageToken: nextToken}), nil
}

func (s *Service) CancelValidationWait(ctx context.Context, req *connect.Request[validationv1.CancelValidationWaitRequest]) (*connect.Response[validationv1.CancelValidationWaitResponse], error) {
	receipt, err := s.repo.Get(ctx, req.Msg.GetReceiptId())
	if err != nil {
		return nil, connectError(err)
	}
	cancelled := s.signalWaiter(req.Msg.GetReceiptId(), req.Msg.GetWaitId(), waitCancelled)
	return connect.NewResponse(&validationv1.CancelValidationWaitResponse{Cancelled: cancelled, Receipt: receipt}), nil
}

func (s *Service) AbortValidationWork(ctx context.Context, req *connect.Request[validationv1.AbortValidationWorkRequest]) (*connect.Response[validationv1.AbortValidationWorkResponse], error) {
	if strings.TrimSpace(req.Msg.GetReason()) == "" || strings.TrimSpace(req.Msg.GetRequestedBy()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("reason and requested_by are required"))
	}
	receipt, err := s.repo.Get(ctx, req.Msg.GetReceiptId())
	if err != nil {
		return nil, connectError(err)
	}
	if terminal(receipt.GetState()) {
		return connect.NewResponse(&validationv1.AbortValidationWorkResponse{Receipt: receipt}), nil
	}
	if s.aborter != nil {
		if err := s.aborter.AbortValidation(ctx, receipt, req.Msg.GetReason(), req.Msg.GetRequestedBy()); err != nil {
			return nil, connect.NewError(connect.CodeUnavailable, err)
		}
	}
	receipt, err = s.repo.Get(ctx, req.Msg.GetReceiptId())
	if err != nil {
		return nil, connectError(err)
	}
	if terminal(receipt.GetState()) {
		return connect.NewResponse(&validationv1.AbortValidationWorkResponse{Receipt: receipt}), nil
	}
	receipt, err = s.Transition(ctx, receipt.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_CANCELLED, func(value *validationv1.ValidationReceipt) error {
		value.ReasonCode = validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_ABORTED
		value.Detail = fmt.Sprintf("work aborted by %s: %s", strings.TrimSpace(req.Msg.GetRequestedBy()), strings.TrimSpace(req.Msg.GetReason()))
		return nil
	})
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&validationv1.AbortValidationWorkResponse{Receipt: receipt}), nil
}

func (s *Service) ExplainValidation(ctx context.Context, req *connect.Request[validationv1.ExplainValidationRequest]) (*connect.Response[validationv1.ExplainValidationResponse], error) {
	receipt, err := s.repo.Get(ctx, req.Msg.GetReceiptId())
	if err != nil {
		return nil, connectError(err)
	}
	decisions := []string{fmt.Sprintf("receipt state is %s at revision %d", receipt.GetState(), receipt.GetRevision())}
	if compatibility := receipt.GetCompatibility(); compatibility != nil {
		decisions = append(decisions, fmt.Sprintf("admission compatibility is %s", compatibility.GetKind()))
	}
	next := []string{"wait for the receipt owner"}
	if terminal(receipt.GetState()) {
		next = []string{"consume typed evidence and reason code"}
	}
	return connect.NewResponse(&validationv1.ExplainValidationResponse{Receipt: receipt, Decisions: decisions, NextActions: next}), nil
}

func (s *Service) ListValidationShadows(ctx context.Context, req *connect.Request[validationv1.ListValidationShadowsRequest]) (*connect.Response[validationv1.ListValidationShadowsResponse], error) {
	if s.shadows == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("shadow comparison repository is unavailable"))
	}
	items, err := s.shadows.ListShadowComparisons(ctx, int(req.Msg.GetPageSize()))
	if err != nil {
		return nil, connectError(err)
	}
	comparisons := make([]*validationv1.ValidationShadowComparison, 0, len(items))
	for _, item := range items {
		comparisons = append(comparisons, shadowComparisonToProto(item))
	}
	return connect.NewResponse(&validationv1.ListValidationShadowsResponse{Comparisons: comparisons}), nil
}

func shadowComparisonToProto(item ShadowComparison) *validationv1.ValidationShadowComparison {
	return &validationv1.ValidationShadowComparison{
		ComparisonId: item.ComparisonID, SourceKind: item.SourceKind, SourceId: item.SourceID, ReceiptId: item.ReceiptID,
		ReceiptRevision: item.ReceiptRevision, LegacyState: item.LegacyState, ReceiptState: item.ReceiptState, Matched: item.Matched,
		ReasonCode: item.ReasonCode, LegacyEvidenceCount: uint32(item.LegacyEvidenceCount), ReceiptEvidenceCount: uint32(item.ReceiptEvidenceCount), ObservedAt: timestamppb.New(item.ObservedAt),
	}
}

func (s *Service) Transition(ctx context.Context, receiptID string, next validationv1.ReceiptState, mutate func(*validationv1.ValidationReceipt) error) (*validationv1.ValidationReceipt, error) {
	receipt, err := s.repo.Transition(ctx, receiptID, next, mutate)
	if err == nil {
		s.notify(receiptID)
		if terminal(receipt.GetState()) {
			attached, propagateErr := s.repo.PropagateTerminal(ctx, receipt)
			if propagateErr != nil {
				return nil, propagateErr
			}
			for _, attachedID := range attached {
				s.notify(attachedID)
			}
		}
	}
	return receipt, err
}

func (s *Service) registerWaiter(receiptID, waitID string) (<-chan waitSignal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.waiters[receiptID] == nil {
		s.waiters[receiptID] = map[string]chan waitSignal{}
	}
	if _, exists := s.waiters[receiptID][waitID]; exists {
		return nil, fmt.Errorf("wait %q is already attached", waitID)
	}
	ch := make(chan waitSignal, 1)
	s.waiters[receiptID][waitID] = ch
	return ch, nil
}

func (s *Service) removeWaiter(receiptID, waitID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.waiters[receiptID], waitID)
	if len(s.waiters[receiptID]) == 0 {
		delete(s.waiters, receiptID)
	}
}

func (s *Service) signalWaiter(receiptID, waitID string, signal waitSignal) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	waiter := s.waiters[strings.TrimSpace(receiptID)][strings.TrimSpace(waitID)]
	if waiter == nil {
		return false
	}
	select {
	case waiter <- signal:
	default:
	}
	return true
}

func (s *Service) notify(receiptID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, waiter := range s.waiters[receiptID] {
		select {
		case waiter <- waitChanged:
		default:
		}
	}
}

func connectError(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, ErrInvalidIntent), errors.Is(err, ErrIdempotencyKey), errors.Is(err, ErrInvalidTransition):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, ErrConcurrentUpdate):
		return connect.NewError(connect.CodeAborted, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
