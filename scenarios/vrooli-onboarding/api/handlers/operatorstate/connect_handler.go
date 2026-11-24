package operatorstate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/vrooli/internal/operatorstate"
	operatorstatev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/operatorstate"
	operatorstatev1connect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/operatorstate/operatorstatev1connect"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/operatorstateapi"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/permissions"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type connectHandler struct{ service operatorstateapi.Service }

func NewConnectHandler(service operatorstateapi.Service) operatorstatev1connect.OperatorStateServiceHandler {
	return &connectHandler{service: service}
}

func (h *connectHandler) GetOperatorState(ctx context.Context, _ *connect.Request[operatorstatev1.GetOperatorStateRequest]) (*connect.Response[operatorstatev1.GetOperatorStateResponse], error) {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok || !principal.IsVerified() {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("verified onboarding principal required"))
	}
	registry, err := h.service.PermissionRegistry()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	result, err := registry.Read(ctx, "onboarding/operator-state", permissions.ReadRequest{ActorID: principal.Subject, Owner: "vrooli-onboarding"})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get operator state: %w", err))
	}
	var document operatorstate.Document
	if err := json.Unmarshal(result.Value, &document); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("decode operator state permission: %w", err))
	}
	state, err := documentToProto(document)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encode operator state: %w", err))
	}
	return connect.NewResponse(&operatorstatev1.GetOperatorStateResponse{State: state}), nil
}

func (h *connectHandler) PatchOperatorState(ctx context.Context, req *connect.Request[operatorstatev1.PatchOperatorStateRequest]) (*connect.Response[operatorstatev1.PatchOperatorStateResponse], error) {
	if req.Msg.GetState() == nil || req.Msg.GetUpdateMask() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("state and update_mask are required"))
	}
	stateJSON, err := (&protojson.MarshalOptions{UseProtoNames: true}).Marshal(req.Msg.GetState())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("encode operator state patch: %w", err))
	}
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok || !principal.IsVerified() {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("verified onboarding principal required"))
	}
	payload, err := json.Marshal(operatorstateapi.PermissionPatch{State: stateJSON, UpdateMask: req.Msg.GetUpdateMask().GetPaths()})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	registry, err := h.service.PermissionRegistry()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	requestID := req.Msg.GetRequestId()
	if requestID == "" {
		legacyInput := append(append([]byte(principal.Subject), 0), payload...)
		legacyDigest := sha256.Sum256(legacyInput)
		requestID = "legacy:" + hex.EncodeToString(legacyDigest[:])
	}
	if len(requestID) > 256 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request_id exceeds 256 characters"))
	}
	requestDigest := sha256.Sum256(payload)
	requestDigestHex := hex.EncodeToString(requestDigest[:])
	expectedRevision := req.Msg.GetExpectedRevision()
	if expectedRevision == "" {
		prior, found, findErr := h.service.FindMutationReceipt(ctx, "vrooli-onboarding", "onboarding/operator-state", requestID)
		if findErr != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("find operator state request receipt: %w", findErr))
		}
		if found {
			if prior.ActorID != principal.Subject || prior.Target != "local" || prior.Subject != "operator-state-document:local" || prior.RequestDigest != requestDigestHex {
				return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("request_id is already bound to a different operator-state request"))
			}
			expectedRevision = prior.ExpectedRevision
		} else {
			current, readErr := registry.Read(ctx, "onboarding/operator-state", permissions.ReadRequest{ActorID: principal.Subject, Owner: "vrooli-onboarding"})
			if readErr != nil {
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("read current operator state revision: %w", readErr))
			}
			expectedRevision = current.Revision
		}
	}
	receipt, err := registry.Write(ctx, "onboarding/operator-state", permissions.WriteRequest{ActorID: principal.Subject, Owner: "vrooli-onboarding", ExpectedRevision: expectedRevision, IdempotencyKey: requestID, Value: payload})
	if err != nil {
		if errors.Is(err, operatorstate.ErrRevisionConflict) {
			return nil, connect.NewError(connect.CodeAborted, err)
		}
		if errors.Is(err, operatorstate.ErrMutationReceiptConflict) {
			return nil, connect.NewError(connect.CodeAlreadyExists, err)
		}
		if code := connect.CodeOf(err); code != connect.CodeUnknown {
			return nil, connect.NewError(code, err)
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	readback, err := registry.Read(ctx, "onboarding/operator-state", permissions.ReadRequest{ActorID: principal.Subject, Owner: "vrooli-onboarding"})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("read patched operator state: %w", err))
	}
	var document operatorstate.Document
	if err := json.Unmarshal(readback.Value, &document); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("decode patched operator state: %w", err))
	}
	state, err := documentToProto(document)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encode patched operator state: %w", err))
	}
	return connect.NewResponse(&operatorstatev1.PatchOperatorStateResponse{State: state, Receipt: receiptToProto(receipt)}), nil
}

func receiptToProto(receipt permissions.Receipt) *operatorstatev1.OperatorStateMutationReceipt {
	return &operatorstatev1.OperatorStateMutationReceipt{Id: receipt.ID, Owner: receipt.Owner, PermissionId: receipt.PermissionID, ActorId: receipt.ActorID, Target: "local", Subject: "operator-state-document:local", RequestId: receipt.IdempotencyKey, ExpectedRevision: receipt.ExpectedRevision, ResultRevision: receipt.ResultRevision, RequestDigest: receipt.RequestDigest, DecisionRef: receipt.DecisionRef, Outcome: receipt.Outcome, CreatedAt: timestamppb.New(receipt.At)}
}

func documentToProto(document operatorstate.Document) (*operatorstatev1.OperatorState, error) {
	data, err := operatorstateapi.MarshalDocument(document)
	if err != nil {
		return nil, err
	}
	state := new(operatorstatev1.OperatorState)
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, state); err != nil {
		return nil, err
	}
	return state, nil
}
