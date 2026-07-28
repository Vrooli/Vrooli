package journal

import (
	"context"
	"database/sql"
	"log"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	journalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/journal"
	internaljournal "vrooli-memory/internal/journal"
)

type connectHandler struct {
	service *internaljournal.Service
	logger  *log.Logger
}

func NewConnectHandler(service *internaljournal.Service, logger *log.Logger) *connectHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &connectHandler{service: service, logger: logger}
}

func (h *connectHandler) AppendEntry(ctx context.Context, req *connect.Request[journalv1.AppendEntryRequest]) (*connect.Response[journalv1.AppendEntryResponse], error) {
	if strings.TrimSpace(req.Msg.GetBody()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errRequired("body"))
	}
	created, err := h.service.Append(ctx, entryFromAppend(req.Msg))
	if err != nil {
		h.logger.Printf("journal.AppendEntry: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&journalv1.AppendEntryResponse{Entry: entryToProto(created)}), nil
}

func (h *connectHandler) GetEntry(ctx context.Context, req *connect.Request[journalv1.GetEntryRequest]) (*connect.Response[journalv1.GetEntryResponse], error) {
	if strings.TrimSpace(req.Msg.GetId()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errRequired("id"))
	}
	entry, err := h.service.Get(ctx, req.Msg.GetId())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		h.logger.Printf("journal.GetEntry: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&journalv1.GetEntryResponse{Entry: entryToProto(entry)}), nil
}

func (h *connectHandler) ListEntries(ctx context.Context, req *connect.Request[journalv1.ListEntriesRequest]) (*connect.Response[journalv1.ListEntriesResponse], error) {
	limit := int(req.Msg.GetLimit())
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errRequired("limit must be at most 500"))
	}
	entries, err := h.service.List(ctx, limit)
	if err != nil {
		h.logger.Printf("journal.ListEntries: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &journalv1.ListEntriesResponse{Entries: make([]*journalv1.Entry, 0, len(entries))}
	for _, entry := range entries {
		if req.Msg.GetFacetId() == "" || entry.FacetID == req.Msg.GetFacetId() {
			resp.Entries = append(resp.Entries, entryToProto(entry))
		}
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ProcessClassificationRetries(ctx context.Context, req *connect.Request[journalv1.ProcessClassificationRetriesRequest]) (*connect.Response[journalv1.ProcessClassificationRetriesResponse], error) {
	limit := int(req.Msg.GetLimit())
	if limit > 500 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errRequired("limit must be at most 500"))
	}
	result, err := h.service.ProcessClassificationRetries(ctx, limit)
	if err != nil {
		h.logger.Printf("journal.ProcessClassificationRetries: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&journalv1.ProcessClassificationRetriesResponse{Processed: int32(result.Processed), Deferred: int32(result.Deferred), AlreadyResolved: int32(result.AlreadyResolved)}), nil
}

func (h *connectHandler) ProcessEmbeddingRetries(ctx context.Context, req *connect.Request[journalv1.ProcessEmbeddingRetriesRequest]) (*connect.Response[journalv1.ProcessEmbeddingRetriesResponse], error) {
	limit := int(req.Msg.GetLimit())
	if limit > 500 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errRequired("limit must be at most 500"))
	}
	result, err := h.service.ProcessEmbeddingRetries(ctx, limit)
	if err != nil {
		h.logger.Printf("journal.ProcessEmbeddingRetries: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&journalv1.ProcessEmbeddingRetriesResponse{Processed: int32(result.Processed), Deferred: int32(result.Deferred), AlreadyResolved: int32(result.AlreadyResolved)}), nil
}

type errRequired string

func (e errRequired) Error() string { return string(e) + " is required" }

func entryFromAppend(in *journalv1.AppendEntryRequest) internaljournal.Entry {
	return internaljournal.Entry{Body: in.GetBody(), FacetID: in.GetFacetId(), Kind: in.GetKind(), Attribution: attributionFromProto(in.GetAttribution()), Correlation: correlationFromProto(in.GetCorrelation()), ImportKey: importKey(in.GetImportProvenance())}
}

func importKey(p *journalv1.ImportProvenance) string {
	if p == nil {
		return ""
	}
	return p.GetRuntime() + ":" + p.GetSourceLocator() + ":" + p.GetContentHash()
}

func attributionFromProto(a *journalv1.Attribution) internaljournal.Attribution {
	if a == nil {
		return internaljournal.Attribution{}
	}
	return internaljournal.Attribution{ActorID: a.GetActorId(), ActorKind: a.GetActorKind(), SourceRuntime: a.GetSourceRuntime()}
}

func correlationFromProto(c *journalv1.Correlation) internaljournal.Correlation {
	if c == nil {
		return internaljournal.Correlation{}
	}
	return internaljournal.Correlation{RunID: c.GetRunId(), WorkflowExecutionID: c.GetWorkflowExecutionId(), ActorKind: c.GetActorKind()}
}

func entryToProto(e internaljournal.Entry) *journalv1.Entry {
	out := &journalv1.Entry{Id: e.ID, Body: e.Body, FacetId: e.FacetID, Kind: e.Kind, Attribution: &journalv1.Attribution{ActorId: e.Attribution.ActorID, ActorKind: e.Attribution.ActorKind, SourceRuntime: e.Attribution.SourceRuntime}, Correlation: &journalv1.Correlation{RunId: e.Correlation.RunID, WorkflowExecutionId: e.Correlation.WorkflowExecutionID, ActorKind: e.Correlation.ActorKind}, CreatedAt: timestamppb.New(e.CreatedAt)}
	for _, f := range e.FacetTexts {
		out.FacetTexts = append(out.FacetTexts, &journalv1.FacetText{Kind: f.Kind, Text: f.Text, EmbeddingRef: f.EmbeddingRef})
	}
	return out
}
