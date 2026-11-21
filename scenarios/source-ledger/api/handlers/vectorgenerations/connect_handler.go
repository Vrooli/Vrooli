package vectorgenerations

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strings"

	"connectrpc.com/connect"
	vectorstore "source-ledger/internal/vector"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	vectorv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/vector"
)

const (
	writeCapability = "source-ledger:write"
	readCapability  = "source-ledger:read"
	operationTarget = "source-ledger.sqlite"
)

type lifecycle interface {
	Stage(context.Context, string) (vectorstore.SQLiteGeneration, error)
	CatchUp(context.Context, string, string) (vectorstore.SQLiteGeneration, error)
	Validate(context.Context, string) (vectorstore.GenerationValidation, error)
	Promote(context.Context, string, string) (vectorstore.SQLiteGeneration, error)
	Rollback(context.Context, string) (vectorstore.SQLiteGeneration, error)
	Operation(context.Context, string) (vectorstore.GenerationOperation, error)
	ResumeOperation(context.Context, string) (vectorstore.GenerationOperation, vectorstore.SQLiteGeneration, error)
	Active(context.Context) (vectorstore.SQLiteGeneration, bool, error)
	Load(context.Context, string) (vectorstore.SQLiteGeneration, error)
}

type connectHandler struct {
	service lifecycle
	logger  *log.Logger
}

func NewConnectHandler(service lifecycle, logger *log.Logger) *connectHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &connectHandler{service: service, logger: logger}
}

func (h *connectHandler) StageGeneration(ctx context.Context, req *connect.Request[vectorv1.StageGenerationRequest]) (*connect.Response[vectorv1.GenerationControlResponse], error) {
	if err := authorize(ctx, "vector_generation.stage", writeCapability); err != nil {
		return nil, err
	}
	operationID, err := stableID(req.Msg.GetOperationId(), "operation_id")
	if err != nil {
		return nil, err
	}
	generation, err := h.service.Stage(ctx, operationID)
	if err != nil {
		return nil, connectFailure(err)
	}
	return h.withOperation(ctx, generation, operationID)
}

func (h *connectHandler) CatchUpGeneration(ctx context.Context, req *connect.Request[vectorv1.CatchUpGenerationRequest]) (*connect.Response[vectorv1.GenerationControlResponse], error) {
	if err := authorize(ctx, "vector_generation.catch_up", writeCapability); err != nil {
		return nil, err
	}
	operationID, err := stableID(req.Msg.GetOperationId(), "operation_id")
	if err != nil {
		return nil, err
	}
	generationID, err := stableID(req.Msg.GetGenerationId(), "generation_id")
	if err != nil {
		return nil, err
	}
	generation, err := h.service.CatchUp(ctx, operationID, generationID)
	if err != nil {
		return nil, connectFailure(err)
	}
	return h.withOperation(ctx, generation, operationID)
}

func (h *connectHandler) ValidateGeneration(ctx context.Context, req *connect.Request[vectorv1.ValidateGenerationRequest]) (*connect.Response[vectorv1.GenerationControlResponse], error) {
	if err := authorize(ctx, "vector_generation.validate", writeCapability); err != nil {
		return nil, err
	}
	generationID, err := stableID(req.Msg.GetGenerationId(), "generation_id")
	if err != nil {
		return nil, err
	}
	validation, err := h.service.Validate(ctx, generationID)
	if err != nil {
		return nil, connectFailure(err)
	}
	generation, err := h.service.Load(ctx, generationID)
	if err != nil {
		return nil, connectFailure(err)
	}
	return connect.NewResponse(&vectorv1.GenerationControlResponse{Generation: generationProto(generation), Validation: validationProto(validation)}), nil
}

func (h *connectHandler) PromoteGeneration(ctx context.Context, req *connect.Request[vectorv1.PromoteGenerationRequest]) (*connect.Response[vectorv1.GenerationControlResponse], error) {
	if err := authorize(ctx, "vector_generation.promote", writeCapability); err != nil {
		return nil, err
	}
	operationID, err := stableID(req.Msg.GetOperationId(), "operation_id")
	if err != nil {
		return nil, err
	}
	generationID, err := stableID(req.Msg.GetGenerationId(), "generation_id")
	if err != nil {
		return nil, err
	}
	generation, err := h.service.Promote(ctx, operationID, generationID)
	if err != nil {
		return nil, connectFailure(err)
	}
	return h.withOperation(ctx, generation, operationID)
}

func (h *connectHandler) RollbackGeneration(ctx context.Context, req *connect.Request[vectorv1.RollbackGenerationRequest]) (*connect.Response[vectorv1.GenerationControlResponse], error) {
	if err := authorize(ctx, "vector_generation.rollback", writeCapability); err != nil {
		return nil, err
	}
	operationID, err := stableID(req.Msg.GetOperationId(), "operation_id")
	if err != nil {
		return nil, err
	}
	generation, err := h.service.Rollback(ctx, operationID)
	if err != nil {
		return nil, connectFailure(err)
	}
	return h.withOperation(ctx, generation, operationID)
}

func (h *connectHandler) GetGenerationOperation(ctx context.Context, req *connect.Request[vectorv1.GetGenerationOperationRequest]) (*connect.Response[vectorv1.GenerationControlResponse], error) {
	if err := authorize(ctx, "vector_generation.status", readCapability); err != nil {
		return nil, err
	}
	operationID, err := stableID(req.Msg.GetOperationId(), "operation_id")
	if err != nil {
		return nil, err
	}
	operation, err := h.service.Operation(ctx, operationID)
	if err != nil {
		return nil, connectFailure(err)
	}
	response := &vectorv1.GenerationControlResponse{Operation: operationProto(operation)}
	if operation.GenerationID != "all-eligible" {
		generation, err := h.service.Load(ctx, operation.GenerationID)
		if err != nil {
			return nil, connectFailure(err)
		}
		response.Generation = generationProto(generation)
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) ResumeGenerationOperation(ctx context.Context, req *connect.Request[vectorv1.ResumeGenerationOperationRequest]) (*connect.Response[vectorv1.GenerationControlResponse], error) {
	if err := authorize(ctx, "vector_generation.resume", writeCapability); err != nil {
		return nil, err
	}
	operationID, err := stableID(req.Msg.GetOperationId(), "operation_id")
	if err != nil {
		return nil, err
	}
	operation, generation, err := h.service.ResumeOperation(ctx, operationID)
	if err != nil {
		return nil, connectFailure(err)
	}
	response := &vectorv1.GenerationControlResponse{Operation: operationProto(operation)}
	if generation.ID != "" {
		response.Generation = generationProto(generation)
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) GetActiveGeneration(ctx context.Context, _ *connect.Request[vectorv1.GetActiveGenerationRequest]) (*connect.Response[vectorv1.GenerationControlResponse], error) {
	if err := authorize(ctx, "vector_generation.active", readCapability); err != nil {
		return nil, err
	}
	generation, found, err := h.service.Active(ctx)
	if err != nil {
		return nil, connectFailure(err)
	}
	response := &vectorv1.GenerationControlResponse{ActiveFound: found}
	if found {
		response.Generation = generationProto(generation)
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) withOperation(ctx context.Context, generation vectorstore.SQLiteGeneration, operationID string) (*connect.Response[vectorv1.GenerationControlResponse], error) {
	operation, err := h.service.Operation(ctx, operationID)
	if err != nil {
		return nil, connectFailure(err)
	}
	return connect.NewResponse(&vectorv1.GenerationControlResponse{Generation: generationProto(generation), Operation: operationProto(operation)}), nil
}

func authorize(ctx context.Context, operation, capability string) error {
	if _, err := authn.RequireOperation(ctx, capability, operation, operationTarget); err != nil {
		if principal, ok := identity.PrincipalFromContext(ctx); !ok || !principal.IsVerified() {
			return connect.NewError(connect.CodeUnauthenticated, err)
		}
		return connect.NewError(connect.CodePermissionDenied, err)
	}
	return nil
}

func stableID(value, field string) (string, error) {
	if value == "" || len(value) > 128 || strings.TrimSpace(value) != value {
		return "", connect.NewError(connect.CodeInvalidArgument, errors.New(field+" must be a stable non-empty ID of at most 128 bytes"))
	}
	if strings.IndexFunc(value, unstableIDChar) >= 0 {
		return "", connect.NewError(connect.CodeInvalidArgument, errors.New(field+" contains an unsupported character"))
	}
	return value, nil
}

// unstableIDChar reports a character outside ASCII letters, digits and "-_.:".
func unstableIDChar(char rune) bool {
	return !(char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || strings.ContainsRune("-_.:", char))
}

func connectFailure(err error) error {
	var code connect.Code
	switch {
	case errors.Is(err, sql.ErrNoRows):
		code = connect.CodeNotFound
	case errors.Is(err, vectorstore.ErrGenerationDiskAdmissionUnavailable):
		code = connect.CodeUnavailable
	case errors.Is(err, vectorstore.ErrGenerationOperationConflict), errors.Is(err, vectorstore.ErrGenerationCursorConflict):
		code = connect.CodeAborted
	case errors.Is(err, vectorstore.ErrGenerationCatchUpRequired), errors.Is(err, vectorstore.ErrGenerationNotReady):
		code = connect.CodeFailedPrecondition
	case errors.Is(err, context.Canceled):
		code = connect.CodeCanceled
	case errors.Is(err, context.DeadlineExceeded):
		code = connect.CodeDeadlineExceeded
	default:
		code = connect.CodeInternal
	}
	return connect.NewError(code, err)
}

func generationProto(generation vectorstore.SQLiteGeneration) *vectorv1.SQLiteGeneration {
	rollbackExpires := ""
	if !generation.RollbackExpires.IsZero() {
		rollbackExpires = generation.RollbackExpires.UTC().Format("2006-01-02T15:04:05.999999999Z07:00")
	}
	return &vectorv1.SQLiteGeneration{
		GenerationId: generation.ID, RecipeId: generation.RecipeID, Dimensions: int32(generation.Dimensions),
		State: generation.State, SnapshotId: generation.SnapshotID, SnapshotWatermark: generation.SnapshotWatermark,
		CatchupWatermark: generation.CatchupWatermark, SnapshotCursor: generation.SnapshotCursor,
		PredecessorId: generation.PredecessorID, RollbackExpiresAt: rollbackExpires,
		Protected: generation.Protected, Version: generation.Version,
	}
}

func operationProto(operation vectorstore.GenerationOperation) *vectorv1.GenerationOperation {
	return &vectorv1.GenerationOperation{
		OperationId: operation.OperationID, Operation: operation.Operation,
		GenerationId: operation.GenerationID, RecipeId: operation.RecipeID,
		State: operation.State, ReceiptId: operation.ReceiptID,
		PeakBytes: operation.PeakBytes, StartedAt: operation.StartedAt,
		CompletedAt: operation.CompletedAt, Error: operation.Error,
	}
}

func validationProto(validation vectorstore.GenerationValidation) *vectorv1.GenerationValidation {
	return &vectorv1.GenerationValidation{
		GenerationId: validation.GenerationID, State: validation.State, RecipeId: validation.RecipeID,
		Dimensions: int32(validation.Dimensions), ReadyForPromotion: validation.ReadyForPromotion,
		ExpectedPoints: validation.ExpectedPoints, ActualPoints: validation.ActualPoints,
		MissingPoints: validation.MissingPoints, OrphanedPoints: validation.OrphanedPoints,
		InvalidPoints: validation.InvalidPoints, PendingChanges: validation.PendingChanges,
		CatchupWatermark: validation.CatchupWatermark, CurrentWatermark: validation.CurrentWatermark,
	}
}
