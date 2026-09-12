package audits

import (
	"context"
	"log"

	internalaudits "data-backup-manager/internal/audits"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	auditsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/audits"
)

// Deps wires the seams the Connect audits handler needs.
type Deps struct {
	Service internalaudits.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the audits Connect-RPC handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) RunSnapshotAudit(ctx context.Context, req *connect.Request[auditsv1.RunSnapshotAuditRequest]) (*connect.Response[auditsv1.RunSnapshotAuditResponse], error) {
	rec, err := h.deps.Service.RunSnapshotAudit(ctx,
		req.Msg.TargetId, req.Msg.DestinationId, req.Msg.SnapshotId,
		req.Msg.IncludeContentHash, req.Msg.IncludeSqliteChecks)
	if err != nil {
		return nil, h.translate("RunSnapshotAudit", err)
	}
	return connect.NewResponse(&auditsv1.RunSnapshotAuditResponse{Audit: auditToProto(rec)}), nil
}

func (h *connectHandler) GetAudit(ctx context.Context, req *connect.Request[auditsv1.GetAuditRequest]) (*connect.Response[auditsv1.GetAuditResponse], error) {
	rec, err := h.deps.Service.GetAudit(ctx, req.Msg.Id)
	if err != nil {
		return nil, h.translate("GetAudit", err)
	}
	return connect.NewResponse(&auditsv1.GetAuditResponse{Audit: auditToProto(rec)}), nil
}

func (h *connectHandler) ListAudits(ctx context.Context, req *connect.Request[auditsv1.ListAuditsRequest]) (*connect.Response[auditsv1.ListAuditsResponse], error) {
	list, err := h.deps.Service.ListAudits(ctx, req.Msg.TargetId, int(req.Msg.PageSize))
	if err != nil {
		return nil, h.translate("ListAudits", err)
	}
	resp := &auditsv1.ListAuditsResponse{Audits: make([]*auditsv1.Audit, 0, len(list))}
	for _, a := range list {
		resp.Audits = append(resp.Audits, auditToProto(a))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) translate(op string, err error) error {
	connectErr := internalaudits.ToConnectError(err)
	if connect.CodeOf(connectErr) == connect.CodeInternal {
		h.deps.Logger.Printf("audits.%s: %v", op, err)
	}
	return connectErr
}

func auditToProto(a internalaudits.Audit) *auditsv1.Audit {
	pa := &auditsv1.Audit{
		Id:                  a.ID,
		TargetId:            a.TargetID,
		DestinationId:       a.DestinationID,
		SnapshotId:          a.SnapshotID,
		Status:              auditStatusToProto(a.Status),
		IncludeContentHash:  a.IncludeContentHash,
		IncludeSqliteChecks: a.IncludeSQLiteCheck,
		Restorable:          a.Restorable,
		Live:                inventoryToProto(a.Live),
		Snapshot:            inventoryToProto(a.Snapshot),
		Comparison:          comparisonToProto(a.Comparison),
		Error:               a.Error,
	}
	if !a.SnapshotTime.IsZero() {
		pa.SnapshotTime = timestamppb.New(a.SnapshotTime)
	}
	if !a.RequestedAt.IsZero() {
		pa.RequestedAt = timestamppb.New(a.RequestedAt)
	}
	if !a.FinishedAt.IsZero() {
		pa.FinishedAt = timestamppb.New(a.FinishedAt)
	}
	return pa
}

func inventoryToProto(inv *internalaudits.InventorySummary) *auditsv1.InventorySummary {
	if inv == nil {
		return nil
	}
	out := &auditsv1.InventorySummary{
		Files:             inv.Files,
		Directories:       inv.Directories,
		Symlinks:          inv.Symlinks,
		Other:             inv.Other,
		RegularBytes:      inv.RegularBytes,
		PathListSha256:    inv.PathListSHA256,
		TreeContentSha256: inv.TreeContentSHA,
		UnreadablePaths:   inv.UnreadablePaths,
	}
	if !inv.CapturedAt.IsZero() {
		out.CapturedAt = timestamppb.New(inv.CapturedAt)
	}
	for _, s := range inv.SQLite {
		out.Sqlite = append(out.Sqlite, &auditsv1.SqliteInventory{
			Path:            s.Path,
			IntegrityStatus: s.IntegrityStatus,
			PageCount:       s.PageCount,
			PageSize:        s.PageSize,
			SchemaSha256:    s.SchemaSHA256,
			TableCount:      s.TableCount,
		})
	}
	return out
}

func comparisonToProto(c *internalaudits.AuditComparison) *auditsv1.AuditComparison {
	if c == nil {
		return nil
	}
	return &auditsv1.AuditComparison{
		Matches:               c.Matches,
		Mismatches:            c.Mismatches,
		LiveNewerThanSnapshot: c.LiveNewerThanSnapshot,
	}
}

func auditStatusToProto(s internalaudits.AuditStatus) auditsv1.AuditStatus {
	switch s {
	case internalaudits.AuditRequested:
		return auditsv1.AuditStatus_AUDIT_STATUS_REQUESTED
	case internalaudits.AuditRunning:
		return auditsv1.AuditStatus_AUDIT_STATUS_RUNNING
	case internalaudits.AuditCompleted:
		return auditsv1.AuditStatus_AUDIT_STATUS_COMPLETED
	case internalaudits.AuditFailed:
		return auditsv1.AuditStatus_AUDIT_STATUS_FAILED
	default:
		return auditsv1.AuditStatus_AUDIT_STATUS_UNSPECIFIED
	}
}
