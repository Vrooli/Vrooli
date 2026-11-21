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
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation/validation_v1connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"test-genie/internal/execution"
	sharedruns "test-genie/internal/shared/runs"
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

type EvidenceWorkAborter interface {
	AbortEvidenceValidation(context.Context, *validationv1.ValidationReceipt, *validationv1.ValidationIntent, string, string) error
}

type TransitionFunc func(context.Context, string, validationv1.ReceiptState, func(*validationv1.ValidationReceipt) error) (*validationv1.ValidationReceipt, error)

// WorkProducer owns server-lifetime validation execution. ExecuteValidation is
// invoked only for the one receipt admitted as the lineage producer.
type WorkProducer interface {
	ExecuteValidation(context.Context, *validationv1.ValidationReceipt, *validationv1.ValidationIntent, TransitionFunc) error
}

type EvidenceDeclarationResolver interface {
	ResolveEvidenceProducer(context.Context, string, string, string) (*validationv1.PinnedEvidenceProducer, error)
}

// RetainedEvidenceAdmission owns exact catalog verification and existing run
// pin leases. A missing implementation must fail closed for bound evidence.
type RetainedEvidenceAdmission interface {
	Verify(context.Context, *scenariovalidationv1.RetainedEvidenceSet) error
	Pin(context.Context, string, *scenariovalidationv1.RetainedEvidenceSet) error
	Release(context.Context, string, *scenariovalidationv1.RetainedEvidenceSet) error
}

// InvalidRetainedEvidenceError marks a bundle defect local to one validation
// receipt. Recovery can fail that receipt while continuing unrelated work.
type InvalidRetainedEvidenceError struct{ Cause error }

func (e *InvalidRetainedEvidenceError) Error() string { return e.Cause.Error() }
func (e *InvalidRetainedEvidenceError) Unwrap() error { return e.Cause }

func invalidRetainedEvidence(format string, args ...any) error {
	return &InvalidRetainedEvidenceError{Cause: fmt.Errorf(format, args...)}
}

func IsInvalidRetainedEvidence(err error) bool {
	var invalid *InvalidRetainedEvidenceError
	return errors.As(err, &invalid)
}

// Service is the receipt observation and control surface. Producer adapters
// call Transition after durable child progress; all waiting is notification-
// driven and a client context never owns producer work.
type Service struct {
	validationconnect.UnimplementedValidationServiceHandler
	repo                 ReceiptRepository
	aborter              WorkAborter
	producer             WorkProducer
	identity             IdentityResolver
	evidenceDeclarations EvidenceDeclarationResolver
	retainedEvidence     RetainedEvidenceAdmission
	shadows              ShadowRepository
	mu                   sync.Mutex
	waiters              map[string]map[string]chan waitSignal
	drivers              map[string]bool
	reattachDelay        time.Duration
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

func (s *Service) SetEvidenceDeclarationResolver(resolver EvidenceDeclarationResolver) {
	s.evidenceDeclarations = resolver
}

func (s *Service) SetRetainedEvidenceAdmission(admission RetainedEvidenceAdmission) {
	s.retainedEvidence = admission
}

// ResolveSourceIdentity is the read-only owner seam for workflows that need
// to bind evidence after an effectful candidate handoff. It deliberately
// accepts only content roots and targets, never a caller-supplied identity;
// the resolver therefore computes the identity from the current repository
// state at the point of the call.
func (s *Service) ResolveSourceIdentity(ctx context.Context, req *connect.Request[validationv1.ResolveSourceIdentityRequest]) (*connect.Response[validationv1.ResolveSourceIdentityResponse], error) {
	if req == nil || req.Msg == nil || len(req.Msg.GetContentInputs()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("content inputs are required"))
	}
	request := req.Msg
	if s.identity == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("source identity resolver is unavailable"))
	}
	intent := &validationv1.ValidationIntent{
		SchemaVersion: ReceiptSchemaVersion,
		Purpose:       validationv1.ValidationPurpose_VALIDATION_PURPOSE_EVIDENCE_PRODUCTION,
		Targets:       append([]*commonv1.ValidationTarget(nil), request.GetTargets()...),
		ContentInputs: append([]*validationv1.ContentInputRoot(nil), request.GetContentInputs()...),
	}
	identity, err := s.identity.Resolve(ctx, intent)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("resolve source identity: %w", err))
	}
	return connect.NewResponse(&validationv1.ResolveSourceIdentityResponse{Identity: identity}), nil
}

func (s *Service) verifyRetainedEvidenceProducers(ctx context.Context, intent *validationv1.ValidationIntent) error {
	for _, set := range intent.GetRetainedEvidenceSets() {
		producer, err := s.repo.Get(ctx, set.GetProducerReceiptId())
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return invalidRetainedEvidence("retained producer receipt %q is missing", set.GetProducerReceiptId())
			}
			return fmt.Errorf("resolve retained producer receipt: %w", err)
		}
		producerIntent, err := s.repo.GetIntent(ctx, set.GetProducerReceiptId())
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return invalidRetainedEvidence("retained producer intent %q is missing", set.GetProducerReceiptId())
			}
			return fmt.Errorf("resolve retained producer intent: %w", err)
		}
		pin := producerIntent.GetPinnedEvidenceProducer()
		if producer.GetState() != validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED || pin == nil || pin.GetProducer() != set.GetProducer() {
			return invalidRetainedEvidence("retained evidence does not reference the exact successful producer receipt")
		}
		if producer.GetProducedEvidenceSet() == nil || !proto.Equal(producer.GetProducedEvidenceSet(), set) {
			return invalidRetainedEvidence("retained evidence must be passed unchanged from its producer receipt")
		}
		if len(producerIntent.GetTargets()) != 1 || producerIntent.GetTargets()[0].GetId() != set.GetTarget() {
			return invalidRetainedEvidence("retained evidence producer target changed")
		}
		if producer.GetAdmittedIdentity().GetIdentity() != set.GetCandidateIdentity() {
			return invalidRetainedEvidence("retained evidence candidate identity differs from its producer admission")
		}
		child := childByID(producer, evidenceProducerChildID(pin))
		if child == nil || child.GetState() != validationv1.ChildOperationState_CHILD_OPERATION_STATE_SUCCEEDED || child.GetOperationId() != set.GetRunId() {
			return invalidRetainedEvidence("retained evidence run is not the successful child of its producer receipt")
		}
	}
	return nil
}

func (s *Service) CreateValidation(ctx context.Context, req *connect.Request[validationv1.CreateValidationRequest]) (*connect.Response[validationv1.CreateValidationResponse], error) {
	intent := req.Msg.GetIntent()
	if intent != nil {
		intent = proto.Clone(intent).(*validationv1.ValidationIntent)
	}
	if intent != nil && (intent.GetPurpose() == validationv1.ValidationPurpose_VALIDATION_PURPOSE_EVIDENCE_PRODUCTION || intent.GetPinnedEvidenceProducer() != nil) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("evidence production requires CreateEvidenceProduction"))
	}
	if intent != nil && len(intent.GetRetainedEvidenceSets()) > 0 {
		if s.retainedEvidence == nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("retained evidence catalog admission is unavailable"))
		}
		if err := s.verifyRetainedEvidenceProducers(ctx, intent); err != nil {
			if IsInvalidRetainedEvidence(err) {
				return nil, connect.NewError(connect.CodeFailedPrecondition, err)
			}
			return nil, connectError(err)
		}
		for _, set := range intent.GetRetainedEvidenceSets() {
			if err := s.retainedEvidence.Verify(ctx, set); err != nil {
				return nil, connect.NewError(connect.CodeFailedPrecondition, err)
			}
		}
	}
	receipt, err := s.createValidation(ctx, intent)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&validationv1.CreateValidationResponse{Receipt: receipt}), nil
}

func (s *Service) CreateEvidenceProduction(ctx context.Context, req *connect.Request[validationv1.CreateEvidenceProductionRequest]) (*connect.Response[validationv1.CreateEvidenceProductionResponse], error) {
	request := req.Msg
	if request == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("evidence production request is required"))
	}
	// Resolve a durable key before touching the mutable provider declaration or
	// host containment. Replays compare only the original typed request.
	if request.GetIdempotencyKey() != "" && request.GetCallerScenario() != "" {
		receipt, replayErr := s.repo.FindEvidenceProductionReplay(ctx, request)
		if replayErr != nil {
			return nil, connectError(replayErr)
		}
		if receipt != nil {
			return connect.NewResponse(&validationv1.CreateEvidenceProductionResponse{Receipt: receipt}), nil
		}
	}
	if s.evidenceDeclarations == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("evidence producer declarations are unavailable"))
	}
	if request == nil || request.GetExpectedCandidateIdentity() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("expected candidate identity is required"))
	}
	pinned, err := s.evidenceDeclarations.ResolveEvidenceProducer(ctx, request.GetProvider(), request.GetProducer(), request.GetCandidateScenario())
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if err := execution.CheckEvidenceProducerContainment(ctx); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if s.identity == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("producer source identity resolver is unavailable"))
	}
	sourceIdentity, err := s.identity.Resolve(ctx, producerSourceIntent(pinned))
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("resolve provider source identity: %w", err))
	}
	pinned.SourceIdentity = sourceIdentity.GetIdentity()
	intent := &validationv1.ValidationIntent{
		SchemaVersion: ReceiptSchemaVersion, IdempotencyKey: request.GetIdempotencyKey(), CallerScenario: request.GetCallerScenario(), CallerExecutionId: request.GetCallerExecutionId(), PlanId: request.GetPlanId(),
		Targets:                []*commonv1.ValidationTarget{{Kind: commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO, Id: request.GetCandidateScenario()}},
		Purpose:                validationv1.ValidationPurpose_VALIDATION_PURPOSE_EVIDENCE_PRODUCTION,
		RequiredStrength:       validationv1.ValidationStrength_VALIDATION_STRENGTH_TARGETED,
		ExpectedIdentity:       proto.Clone(request.GetExpectedCandidateIdentity()).(*validationv1.SourceIdentity),
		ContentInputs:          []*validationv1.ContentInputRoot{{Name: "candidate", Root: "scenarios/" + request.GetCandidateScenario(), Selections: []*validationv1.InputSelection{{Glob: "**", Required: true}}}},
		EvidencePolicy:         &validationv1.EvidencePolicy{},
		ReusePolicy:            &validationv1.ReusePolicy{Mode: validationv1.ReuseMode_REUSE_MODE_NEVER},
		ConcurrencyPolicy:      &validationv1.ConcurrencyPolicy{Mode: validationv1.ConcurrencyMode_CONCURRENCY_MODE_EXCLUSIVE},
		DeadlinePolicy:         &validationv1.DeadlinePolicy{MaximumAttempts: 1},
		PinnedEvidenceProducer: pinned,
	}
	receipt, err := s.createValidation(ctx, intent)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&validationv1.CreateEvidenceProductionResponse{Receipt: receipt}), nil
}

func (s *Service) createValidation(ctx context.Context, intent *validationv1.ValidationIntent) (*validationv1.ValidationReceipt, error) {
	if s.identity != nil && intent != nil {
		expected := intent.GetExpectedIdentity()
		resolved, resolveErr := s.identity.Resolve(ctx, intent)
		if resolveErr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%w: resolve content inputs: %v", ErrInvalidIntent, resolveErr))
		}
		if err := matchExpectedIdentity(expected, resolved); err != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
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
	if len(intent.GetRetainedEvidenceSets()) > 0 && !terminal(admission.Receipt.GetState()) {
		for _, set := range intent.GetRetainedEvidenceSets() {
			if err := s.retainedEvidence.Pin(context.WithoutCancel(ctx), admission.Receipt.GetReceiptId(), set); err != nil {
				if admission.Kind == AdmissionNew {
					_, _ = s.Transition(context.WithoutCancel(ctx), admission.Receipt.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_FAILED, func(r *validationv1.ValidationReceipt) error {
						r.ReasonCode = validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_PROVIDER_UNAVAILABLE
						r.Detail = "retained evidence pin failed: " + err.Error()
						return nil
					})
				}
				return nil, connect.NewError(connect.CodeFailedPrecondition, err)
			}
		}
	}
	if admission.Kind == AdmissionNew && s.producer != nil {
		queued, transitionErr := s.Transition(context.WithoutCancel(ctx), admission.Receipt.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_QUEUED, nil)
		if transitionErr != nil {
			return nil, connectError(transitionErr)
		}
		admission.Receipt = queued
		s.startProducer(admission.Receipt, intent)
	}
	return admission.Receipt, nil
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
			if len(item.Intent.GetRetainedEvidenceSets()) > 0 && errors.Is(err, ErrInvalidIntent) {
				if failErr := s.failRecoveredRetainedEvidence(ctx, item.Receipt.GetReceiptId(), err); failErr != nil {
					return 0, failErr
				}
				continue
			}
			return 0, err
		}
		if len(intent.GetRetainedEvidenceSets()) > 0 {
			if s.retainedEvidence == nil {
				return 0, errors.New("retained evidence lease recovery is unavailable")
			}
			if err := s.verifyRetainedEvidenceProducers(ctx, intent); err != nil {
				if !IsInvalidRetainedEvidence(err) {
					return 0, err
				}
				if failErr := s.failRecoveredRetainedEvidence(ctx, item.Receipt.GetReceiptId(), err); failErr != nil {
					return 0, failErr
				}
				continue
			}
			invalidated := false
			for _, set := range intent.GetRetainedEvidenceSets() {
				if err := s.retainedEvidence.Verify(ctx, set); err != nil {
					if !IsInvalidRetainedEvidence(err) {
						return 0, err
					}
					if failErr := s.failRecoveredRetainedEvidence(ctx, item.Receipt.GetReceiptId(), err); failErr != nil {
						return 0, failErr
					}
					invalidated = true
					break
				}
				if err := s.retainedEvidence.Pin(ctx, item.Receipt.GetReceiptId(), set); err != nil {
					if !invalidRetainedLeaseError(err) {
						return 0, err
					}
					if failErr := s.failRecoveredRetainedEvidence(ctx, item.Receipt.GetReceiptId(), err); failErr != nil {
						return 0, failErr
					}
					invalidated = true
					break
				}
			}
			if invalidated {
				continue
			}
		}
		if s.startProducer(item.Receipt, intent) {
			started++
		}
	}
	return started, nil
}

func invalidRetainedLeaseError(err error) bool {
	code := connect.CodeOf(err)
	return IsInvalidRetainedEvidence(err) || code == connect.CodeInvalidArgument || code == connect.CodeNotFound || code == connect.CodeFailedPrecondition
}

func (s *Service) failRecoveredRetainedEvidence(ctx context.Context, receiptID string, cause error) error {
	_, err := s.Transition(context.WithoutCancel(ctx), receiptID, validationv1.ReceiptState_RECEIPT_STATE_FAILED, func(receipt *validationv1.ValidationReceipt) error {
		receipt.ReasonCode = validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_REQUIRED_EVIDENCE_MISSING
		receipt.Detail = "retained evidence admission failed during recovery: " + cause.Error()
		return nil
	})
	return err
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
		intent, intentErr := s.repo.GetIntent(ctx, receipt.GetReceiptId())
		if intentErr != nil {
			return nil, connectError(intentErr)
		}
		var abortErr error
		if intent.GetPurpose() == validationv1.ValidationPurpose_VALIDATION_PURPOSE_EVIDENCE_PRODUCTION {
			if evidenceAborter, ok := s.aborter.(EvidenceWorkAborter); ok {
				abortErr = evidenceAborter.AbortEvidenceValidation(ctx, receipt, intent, req.Msg.GetReason(), req.Msg.GetRequestedBy())
			} else {
				abortErr = errors.New("pinned evidence producer abort is unavailable")
			}
		} else {
			abortErr = s.aborter.AbortValidation(ctx, receipt, req.Msg.GetReason(), req.Msg.GetRequestedBy())
		}
		if abortErr != nil {
			return nil, connect.NewError(connect.CodeUnavailable, abortErr)
		}
	}
	settledChildren := proto.Clone(receipt).(*validationv1.ValidationReceipt)
	receipt, err = s.repo.Get(ctx, req.Msg.GetReceiptId())
	if err != nil {
		return nil, connectError(err)
	}
	if terminal(receipt.GetState()) {
		return connect.NewResponse(&validationv1.AbortValidationWorkResponse{Receipt: receipt}), nil
	}
	receipt, err = s.Transition(ctx, receipt.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_CANCELLED, func(value *validationv1.ValidationReceipt) error {
		mergeAbortedChildOutcomes(value, settledChildren)
		value.ReasonCode = validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_ABORTED
		value.Detail = fmt.Sprintf("work aborted by %s: %s", strings.TrimSpace(req.Msg.GetRequestedBy()), strings.TrimSpace(req.Msg.GetReason()))
		return nil
	})
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&validationv1.AbortValidationWorkResponse{Receipt: receipt}), nil
}

func mergeAbortedChildOutcomes(current, settled *validationv1.ValidationReceipt) {
	for _, outcome := range settled.GetChildren() {
		if outcome.GetState() != validationv1.ChildOperationState_CHILD_OPERATION_STATE_SUCCEEDED && outcome.GetState() != validationv1.ChildOperationState_CHILD_OPERATION_STATE_FAILED && outcome.GetState() != validationv1.ChildOperationState_CHILD_OPERATION_STATE_CANCELLED {
			continue
		}
		for _, child := range current.GetChildren() {
			if child.GetChildId() != outcome.GetChildId() {
				continue
			}
			if child.GetState() == validationv1.ChildOperationState_CHILD_OPERATION_STATE_SUCCEEDED || child.GetState() == validationv1.ChildOperationState_CHILD_OPERATION_STATE_FAILED || child.GetState() == validationv1.ChildOperationState_CHILD_OPERATION_STATE_CANCELLED {
				break
			}
			child.State = outcome.GetState()
			child.Detail = outcome.GetDetail()
			child.ReasonCode = outcome.GetReasonCode()
			break
		}
	}
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
			var cleanupErr error
			if s.retainedEvidence != nil {
				intent, intentErr := s.repo.GetIntent(ctx, receiptID)
				if intentErr != nil {
					cleanupErr = fmt.Errorf("load retained evidence intent for terminal cleanup: %w", intentErr)
				} else {
					for _, set := range intent.GetRetainedEvidenceSets() {
						var releaseErr error
						for attempt := 0; attempt < 2; attempt++ {
							releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
							releaseErr = s.retainedEvidence.Release(releaseCtx, receiptID, set)
							cancel()
							if releaseErr == nil {
								break
							}
						}
						if releaseErr != nil {
							cleanupErr = errors.Join(cleanupErr, fmt.Errorf("release retained evidence pin after bounded retry; the existing %s run-pin lease remains bounded by expiry: %w", sharedruns.DefaultPinLeaseTTL, releaseErr))
						}
					}
				}
			}
			attached, propagateErr := s.repo.PropagateTerminal(ctx, receipt)
			for _, attachedID := range attached {
				s.notify(attachedID)
			}
			if propagateErr != nil {
				cleanupErr = errors.Join(cleanupErr, fmt.Errorf("propagate terminal receipt: %w", propagateErr))
			}
			if cleanupErr != nil {
				return receipt, cleanupErr
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
