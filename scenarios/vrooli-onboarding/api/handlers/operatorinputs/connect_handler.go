package operatorinputs

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/vrooli/internal/operatorcapability"
	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/setup/v1"
	operatorinputsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/operatorinputs"
	operatorinputsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/operatorinputs/operatorinputsv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	internaloperatorinputs "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/operatorinputs"
)

type connectHandler struct {
	service internaloperatorinputs.Service
}

func NewConnectHandler(service internaloperatorinputs.Service) *connectHandler {
	return &connectHandler{service: service}
}

func (h *connectHandler) ListOperatorInputs(ctx context.Context, _ *connect.Request[operatorinputsv1.ListOperatorInputsRequest]) (*connect.Response[operatorinputsv1.ListOperatorInputsResponse], error) {
	queue, err := h.service.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("read operator input queue: %w", err))
	}
	requests := make([]*setupv1.OperatorInputRequest, 0, len(queue.Requests))
	for _, request := range queue.Requests {
		requests = append(requests, requestToProto(request))
	}
	return connect.NewResponse(&operatorinputsv1.ListOperatorInputsResponse{
		Version:   int32(queue.Version),
		UpdatedAt: timestamppb.New(queue.UpdatedAt),
		Requests:  requests,
	}), nil
}

func (h *connectHandler) ResolveOperatorInputs(ctx context.Context, req *connect.Request[operatorinputsv1.ResolveOperatorInputsRequest]) (*connect.Response[operatorinputsv1.ResolveOperatorInputsResponse], error) {
	if req == nil || req.Msg == nil || len(req.Msg.GetAnswers()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, internaloperatorinputs.ErrInvalidAnswer)
	}
	answers := make([]operatorcapability.Answer, 0, len(req.Msg.GetAnswers()))
	for _, answer := range req.Msg.GetAnswers() {
		if answer == nil || strings.TrimSpace(answer.GetRequestId()) == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, internaloperatorinputs.ErrInvalidAnswer)
		}
		answers = append(answers, operatorcapability.Answer{RequestID: answer.GetRequestId(), Value: answer.GetValue(), Declined: answer.GetDeclined()})
	}
	if err := h.service.Resolve(ctx, req.Msg.GetExpectedRevision(), answers); err != nil {
		code := connect.CodeInternal
		switch {
		case errors.Is(err, internaloperatorinputs.ErrInvalidAnswer):
			code = connect.CodeInvalidArgument
		case errors.Is(err, internaloperatorinputs.ErrFailedPrecondition):
			code = connect.CodeFailedPrecondition
		case errors.Is(err, internaloperatorinputs.ErrRevisionConflict):
			code = connect.CodeAborted
		}
		return nil, connect.NewError(code, err)
	}
	outcomes := make([]*operatorinputsv1.AnswerOutcome, 0, len(answers))
	for _, answer := range answers {
		status := operatorinputsv1.AnswerOutcomeStatus_ANSWER_OUTCOME_STATUS_APPLIED
		if answer.Declined {
			status = operatorinputsv1.AnswerOutcomeStatus_ANSWER_OUTCOME_STATUS_DECLINED
		}
		outcomes = append(outcomes, &operatorinputsv1.AnswerOutcome{RequestId: answer.RequestID, Status: status})
	}
	return connect.NewResponse(&operatorinputsv1.ResolveOperatorInputsResponse{Outcomes: outcomes}), nil
}

func requestToProto(request operatorcapability.Request) *setupv1.OperatorInputRequest {
	candidates := make([]*setupv1.Candidate, 0, len(request.Candidates))
	for _, candidate := range request.Candidates {
		candidates = append(candidates, &setupv1.Candidate{
			Id: candidate.ID, Kind: candidate.Kind, Label: candidate.Label, Location: candidate.Location,
			StableIdentity: candidate.StableIdentity, DeviceIdentity: candidate.DeviceIdentity,
			Writable: candidate.Writable, Status: candidate.Status, Risk: candidate.Risk,
			Remediation: candidate.Remediation, Metadata: candidate.Metadata,
		})
	}
	return &setupv1.OperatorInputRequest{
		Id: request.ID, Kind: inputKindToProto(request.Kind), ContractVersion: request.ContractVersion,
		Owner: request.Owner, CapabilityId: request.CapabilityID, ActionId: request.ActionID,
		InputId: request.InputID, Title: request.Title, Description: request.Description,
		DefaultValue: request.Default, Options: request.Options, Candidates: candidates,
		Remediation: request.Remediation, Unblocks: request.Unblocks, Validation: request.Validation,
		Required: request.Required, Declinable: request.Declinable, Decision: request.Decision,
		CredentialLogicalId: request.CredentialLogicalID, CredentialField: request.CredentialField, Provider: request.Provider, RequirementGroup: request.RequirementGroup,
		ConsumerRefs: request.ConsumerRefs, CompanionSettings: request.CompanionSettings, AcquisitionRef: request.AcquisitionRef, VerificationRef: request.VerificationRef,
		RecoveryRef: request.RecoveryRef, HelpRef: request.HelpRef, EvidencePolicy: request.EvidencePolicy,
	}
}

func inputKindToProto(kind operatorcapability.InputKind) setupv1.OperatorInputKind {
	switch kind {
	case operatorcapability.KindSecret:
		return setupv1.OperatorInputKind_OPERATOR_INPUT_KIND_SECRET
	case operatorcapability.KindChoice:
		return setupv1.OperatorInputKind_OPERATOR_INPUT_KIND_CHOICE
	case operatorcapability.KindConfirm:
		return setupv1.OperatorInputKind_OPERATOR_INPUT_KIND_CONFIRM
	case operatorcapability.KindPath:
		return setupv1.OperatorInputKind_OPERATOR_INPUT_KIND_PATH
	case operatorcapability.KindEnum:
		return setupv1.OperatorInputKind_OPERATOR_INPUT_KIND_ENUM
	case operatorcapability.KindBoolean:
		return setupv1.OperatorInputKind_OPERATOR_INPUT_KIND_BOOLEAN
	case operatorcapability.KindDuration:
		return setupv1.OperatorInputKind_OPERATOR_INPUT_KIND_DURATION
	case operatorcapability.KindConfirmation:
		return setupv1.OperatorInputKind_OPERATOR_INPUT_KIND_CONFIRMATION
	default:
		return setupv1.OperatorInputKind_OPERATOR_INPUT_KIND_UNSPECIFIED
	}
}

var _ operatorinputsconnect.OperatorInputsServiceHandler = (*connectHandler)(nil)
