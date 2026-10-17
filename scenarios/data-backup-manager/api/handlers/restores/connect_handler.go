package restores

import (
	"context"
	"log"

	internalrestores "data-backup-manager/internal/restores"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	restoresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/restores"
)

// Deps wires the seams the Connect restores handler needs.
type Deps struct {
	Service internalrestores.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the restores Connect-RPC handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) RestoreTarget(ctx context.Context, req *connect.Request[restoresv1.RestoreTargetRequest]) (*connect.Response[restoresv1.RestoreTargetResponse], error) {
	rec, err := h.deps.Service.RestoreTarget(ctx,
		req.Msg.TargetId, req.Msg.DestinationId, req.Msg.SnapshotId, req.Msg.Location)
	if err != nil {
		return nil, h.translate("RestoreTarget", err)
	}
	return connect.NewResponse(&restoresv1.RestoreTargetResponse{Restore: restoreToProto(rec)}), nil
}

func (h *connectHandler) VerifyTarget(ctx context.Context, req *connect.Request[restoresv1.VerifyTargetRequest]) (*connect.Response[restoresv1.VerifyTargetResponse], error) {
	rec, err := h.deps.Service.VerifyTarget(ctx,
		req.Msg.TargetId, req.Msg.DestinationId, req.Msg.SnapshotId)
	if err != nil {
		return nil, h.translate("VerifyTarget", err)
	}
	return connect.NewResponse(&restoresv1.VerifyTargetResponse{Restore: restoreToProto(rec)}), nil
}

func (h *connectHandler) GetRestore(ctx context.Context, req *connect.Request[restoresv1.GetRestoreRequest]) (*connect.Response[restoresv1.GetRestoreResponse], error) {
	rec, err := h.deps.Service.GetRestore(ctx, req.Msg.Id)
	if err != nil {
		return nil, h.translate("GetRestore", err)
	}
	return connect.NewResponse(&restoresv1.GetRestoreResponse{Restore: restoreToProto(rec)}), nil
}

func (h *connectHandler) ListRestores(ctx context.Context, req *connect.Request[restoresv1.ListRestoresRequest]) (*connect.Response[restoresv1.ListRestoresResponse], error) {
	list, err := h.deps.Service.ListRestores(ctx, req.Msg.TargetId, int(req.Msg.PageSize))
	if err != nil {
		return nil, h.translate("ListRestores", err)
	}
	resp := &restoresv1.ListRestoresResponse{Restores: make([]*restoresv1.Restore, 0, len(list))}
	for _, r := range list {
		resp.Restores = append(resp.Restores, restoreToProto(r))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) translate(op string, err error) error {
	connectErr := internalrestores.ToConnectError(err)
	if connect.CodeOf(connectErr) == connect.CodeInternal {
		h.deps.Logger.Printf("restores.%s: %v", op, err)
	}
	return connectErr
}

func restoreToProto(r internalrestores.Restore) *restoresv1.Restore {
	pr := &restoresv1.Restore{
		Id:            r.ID,
		TargetId:      r.TargetID,
		DestinationId: r.DestinationID,
		SnapshotId:    r.SnapshotID,
		Mode:          restoreModeToProto(r.Mode),
		Status:        restoreStatusToProto(r.Status),
		Location:      r.Location,
		Checksum:      r.Checksum,
		Error:         r.Error,
	}
	if !r.LastVerifiedAt.IsZero() {
		pr.LastVerifiedAt = timestamppb.New(r.LastVerifiedAt)
	}
	if !r.RequestedAt.IsZero() {
		pr.RequestedAt = timestamppb.New(r.RequestedAt)
	}
	if !r.FinishedAt.IsZero() {
		pr.FinishedAt = timestamppb.New(r.FinishedAt)
	}
	return pr
}

func restoreModeToProto(m internalrestores.RestoreMode) restoresv1.RestoreMode {
	switch m {
	case internalrestores.ModeRestore:
		return restoresv1.RestoreMode_RESTORE_MODE_RESTORE
	case internalrestores.ModeVerify:
		return restoresv1.RestoreMode_RESTORE_MODE_VERIFY
	default:
		return restoresv1.RestoreMode_RESTORE_MODE_UNSPECIFIED
	}
}

func restoreStatusToProto(s internalrestores.RestoreStatus) restoresv1.RestoreStatus {
	switch s {
	case internalrestores.RestoreRequested:
		return restoresv1.RestoreStatus_RESTORE_STATUS_REQUESTED
	case internalrestores.RestoreRestoring:
		return restoresv1.RestoreStatus_RESTORE_STATUS_RESTORING
	case internalrestores.RestoreVerifying:
		return restoresv1.RestoreStatus_RESTORE_STATUS_VERIFYING
	case internalrestores.RestoreVerified:
		return restoresv1.RestoreStatus_RESTORE_STATUS_VERIFIED
	case internalrestores.RestoreRestored:
		return restoresv1.RestoreStatus_RESTORE_STATUS_RESTORED
	case internalrestores.RestoreFailed:
		return restoresv1.RestoreStatus_RESTORE_STATUS_FAILED
	default:
		return restoresv1.RestoreStatus_RESTORE_STATUS_UNSPECIFIED
	}
}
