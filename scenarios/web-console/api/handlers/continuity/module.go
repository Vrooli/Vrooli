// Package continuity exposes the typed local conversation integrity and
// reconciliation contract used by the CLI and UI.
package continuity

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	continuityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/continuity"
	continuityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/continuity/continuity_v1connect"
	model "web-console/internal/continuity"
	"web-console/internal/module"
)

type Service interface {
	Integrity(context.Context) (model.IntegrityReport, error)
	Reconcile(context.Context, bool, string, string, string, int, int) ([]model.ReconcileItem, int, int, model.Receipt, string, int, bool, error)
	Catalog(context.Context, string, int) ([]model.CatalogRecord, bool, error)
	Search(context.Context, string, string, string, string, string, string, string, string, string, int) ([]model.SearchMatch, bool, int64, int64, error)
	Receipt(context.Context, string) (model.Receipt, error)
	Rollback(context.Context, string, string) (model.Receipt, error)
	Publish(context.Context, int) (model.PublicationReport, error)
}

func (h *handler) ListCatalog(ctx context.Context, req *connect.Request[continuityv1.ListCatalogRequest]) (*connect.Response[continuityv1.ListCatalogResponse], error) {
	records, truncated, err := h.service.Catalog(ctx, req.Msg.GetLifecycleState(), int(req.Msg.GetLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*continuityv1.CatalogRecord, 0, len(records))
	for _, r := range records {
		r = model.PublicRecord(r)
		aliases := make([]*continuityv1.CatalogAlias, 0, len(r.Aliases))
		for _, a := range r.Aliases {
			aliases = append(aliases, &continuityv1.CatalogAlias{Kind: a.Kind, Value: a.Value})
		}
		out = append(out, &continuityv1.CatalogRecord{SessionId: r.SessionID, LifecycleState: string(r.LifecycleState), Backend: r.Backend, AgentType: r.AgentType, AgentSessionId: r.AgentSessionID, AgentHomeRef: r.AgentHomeRef, RolloutRef: r.RolloutRef, OriginalTitle: r.OriginalTitle, CurrentTitle: r.CurrentTitle, TopicSummary: r.TopicSummary, Cwd: r.CWD, CreatedAt: r.CreatedAt.UTC().Format(time.RFC3339Nano), LastActivityAt: r.LastActivityAt.UTC().Format(time.RFC3339Nano), SourceFingerprint: r.SourceFingerprint, Aliases: aliases})
	}
	return connect.NewResponse(&continuityv1.ListCatalogResponse{Records: out, Truncated: truncated}), nil
}

func (h *handler) Search(ctx context.Context, req *connect.Request[continuityv1.SearchRequest]) (*connect.Response[continuityv1.SearchResponse], error) {
	matches, truncated, total, distinct, err := h.service.Search(ctx, req.Msg.GetQuery(), req.Msg.GetLifecycleState(), req.Msg.GetAgentType(), req.Msg.GetCwd(), req.Msg.GetCreatedAfter(), req.Msg.GetSessionId(), req.Msg.GetAgentSessionId(), req.Msg.GetTitle(), req.Msg.GetTopicSummary(), int(req.Msg.GetLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	out := make([]*continuityv1.SearchMatch, 0, len(matches))
	for _, m := range matches {
		out = append(out, &continuityv1.SearchMatch{EventId: safeUTF8(m.EventID), SessionId: safeUTF8(m.SessionID), Sequence: m.Sequence, Role: safeUTF8(m.Role), CreatedAt: safeUTF8(m.CreatedAt), Excerpt: safeUTF8(m.Excerpt), LifecycleState: safeUTF8(m.LifecycleState)})
	}
	return connect.NewResponse(&continuityv1.SearchResponse{Matches: out, Truncated: truncated, TotalMatches: total, DistinctSessions: distinct}), nil
}

func (h *handler) GetReceipt(ctx context.Context, req *connect.Request[continuityv1.GetReceiptRequest]) (*connect.Response[continuityv1.GetReceiptResponse], error) {
	if req.Msg.GetOperationId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("operation_id is required"))
	}
	r, err := h.service.Receipt(ctx, req.Msg.GetOperationId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	out := &continuityv1.GetReceiptResponse{OperationId: r.OperationID, SessionId: r.SessionID, Command: r.Command, FromState: string(r.FromState), ToState: string(r.ToState), Status: r.Status, ErrorCode: r.ErrorCode, CreatedAt: r.CreatedAt.UTC().Format(time.RFC3339Nano)}
	if !r.CompletedAt.IsZero() {
		out.CompletedAt = r.CompletedAt.UTC().Format(time.RFC3339Nano)
	}
	return connect.NewResponse(out), nil
}

func (h *handler) Rollback(ctx context.Context, req *connect.Request[continuityv1.RollbackRequest]) (*connect.Response[continuityv1.RollbackResponse], error) {
	receipt, err := h.service.Rollback(ctx, req.Msg.GetManifestHash(), req.Msg.GetOperationId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&continuityv1.RollbackResponse{ManifestHash: req.Msg.GetManifestHash(), ReceiptId: receipt.OperationID, ReceiptStatus: receipt.Status}), nil
}

func (h *handler) Publish(ctx context.Context, req *connect.Request[continuityv1.PublishRequest]) (*connect.Response[continuityv1.PublishResponse], error) {
	report, err := h.service.Publish(ctx, int(req.Msg.GetLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewResponse(&continuityv1.PublishResponse{Attempted: int32(report.Attempted), Published: int32(report.Published), Failed: int32(report.Failed), Next: int32(report.Next)}), nil
}

type ModuleDeps struct {
	Service Service
	Logger  *log.Logger
}

type handler struct {
	service Service
	logger  *log.Logger
}

// safeUTF8 keeps historical transcript evidence inspectable even when a native
// provider wrote malformed bytes. Protobuf strings must be valid UTF-8; the
// replacement rune preserves the surrounding excerpt without dropping the
// durable record or failing the entire response.
func safeUTF8(value string) string {
	return strings.ToValidUTF8(value, "\uFFFD")
}

func Module(deps ModuleDeps) module.Module {
	if deps.Logger == nil {
		deps.Logger = log.Default()
	}
	path, h := continuityconnect.NewContinuityServiceHandler(&handler{service: deps.Service, logger: deps.Logger})
	return module.Module{
		Name: "continuity",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h})
		},
		Endpoints: Endpoints,
	}
}

func (h *handler) Integrity(ctx context.Context, _ *connect.Request[continuityv1.IntegrityRequest]) (*connect.Response[continuityv1.IntegrityResponse], error) {
	report, err := h.service.Integrity(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&continuityv1.IntegrityResponse{
		Sessions: report.Sessions, ConversationSessions: report.ConversationSessions,
		ConversationEvents: report.ConversationEvents, Checkpoints: report.Checkpoints,
		WorkspacePanes: report.WorkspacePanes, OrphanConversations: report.OrphanConversations,
		OrphanCheckpoints: report.OrphanCheckpoints, OrphanWorkspacePanes: report.OrphanWorkspacePanes,
		UncatalogedConversations: report.UncatalogedConversations,
		Generation:               report.Generation,
		EventContentHash:         report.EventContentHash,
	}), nil
}

func (h *handler) Reconcile(ctx context.Context, req *connect.Request[continuityv1.ReconcileRequest]) (*connect.Response[continuityv1.ReconcileResponse], error) {
	items, observations, mutations, receipt, manifestHash, nextOffset, complete, err := h.service.Reconcile(ctx, req.Msg.GetApply(), req.Msg.GetGeneration(), req.Msg.GetOperationId(), req.Msg.GetManifestHash(), int(req.Msg.GetOffset()), int(req.Msg.GetBatchSize()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*continuityv1.ReconcileItem, 0, len(items))
	for _, item := range items {
		item = model.PublicReconcileItem(item)
		out = append(out, &continuityv1.ReconcileItem{Action: string(item.Action), SessionId: item.Record.SessionID, Reason: item.Reason, LifecycleState: string(item.Record.LifecycleState), SourceFingerprint: item.Record.SourceFingerprint})
	}
	return connect.NewResponse(&continuityv1.ReconcileResponse{Applied: req.Msg.GetApply(), Observations: int32(observations), Mutations: int32(mutations), Items: out, ReceiptId: receipt.OperationID, ReceiptStatus: receipt.Status, ManifestHash: manifestHash, NextOffset: int32(nextOffset), Complete: complete}), nil
}
