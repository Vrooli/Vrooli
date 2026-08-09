package audit

import (
	"context"
	"log"

	"vrooli-bridge/internal/audit"
	"vrooli-bridge/internal/auth"

	"connectrpc.com/connect"

	"google.golang.org/protobuf/types/known/timestamppb"

	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/audit"
)

// Deps wires the seams the Connect audit handler needs. The handler holds only
// the READ seam (audit.Reader) — records are written by the dispatch/provision
// domains through the separate Sink seam, so there is no wire path here that can
// create or mutate a record.
type Deps struct {
	Reader audit.Reader
	Logger *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the handler, defaulting the logger.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

// ListAuditRecords returns audit records newest-first, optionally filtered by
// node or run. Owner-gated, read-only.
func (h *connectHandler) ListAuditRecords(ctx context.Context, req *connect.Request[auditv1.ListAuditRecordsRequest]) (*connect.Response[auditv1.ListAuditRecordsResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	records, err := h.deps.Reader.List(ctx, audit.ListFilter{
		NodeID: req.Msg.NodeId,
		RunID:  req.Msg.RunId,
		Limit:  int(req.Msg.Limit),
	})
	if err != nil {
		h.deps.Logger.Printf("audit.ListAuditRecords: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &auditv1.ListAuditRecordsResponse{Records: make([]*auditv1.AuditRecord, 0, len(records))}
	for _, r := range records {
		resp.Records = append(resp.Records, recordToProto(r))
	}
	return connect.NewResponse(resp), nil
}

// recordToProto translates a domain audit Record into its wire shape.
func recordToProto(r audit.Record) *auditv1.AuditRecord {
	return &auditv1.AuditRecord{
		Id:         r.ID,
		Action:     actionToProto(r.Action),
		Actor:      r.Actor,
		NodeId:     r.NodeID,
		Scenario:   r.Scenario,
		Verb:       r.Verb,
		Args:       append([]string(nil), r.Args...),
		Outcome:    outcomeToProto(r.Outcome),
		Detail:     r.Detail,
		RunId:      r.RunID,
		RecordedAt: timestamppb.New(r.RecordedAt),
	}
}

func actionToProto(a audit.Action) auditv1.AuditAction {
	switch a {
	case audit.ActionDispatch:
		return auditv1.AuditAction_AUDIT_ACTION_DISPATCH
	case audit.ActionProvision:
		return auditv1.AuditAction_AUDIT_ACTION_PROVISION
	case audit.ActionBreakGlass:
		return auditv1.AuditAction_AUDIT_ACTION_BREAK_GLASS
	default:
		return auditv1.AuditAction_AUDIT_ACTION_UNSPECIFIED
	}
}

func outcomeToProto(o audit.Outcome) auditv1.AuditOutcome {
	switch o {
	case audit.OutcomeAccepted:
		return auditv1.AuditOutcome_AUDIT_OUTCOME_ACCEPTED
	case audit.OutcomeRejected:
		return auditv1.AuditOutcome_AUDIT_OUTCOME_REJECTED
	case audit.OutcomeCompleted:
		return auditv1.AuditOutcome_AUDIT_OUTCOME_COMPLETED
	case audit.OutcomeFailed:
		return auditv1.AuditOutcome_AUDIT_OUTCOME_FAILED
	default:
		return auditv1.AuditOutcome_AUDIT_OUTCOME_UNSPECIFIED
	}
}
