package learning

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	source "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal"
	scopes "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/scopes"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/learning"
	"vrooli-memory/internal/learning"
	"vrooli-memory/internal/ledgerclient"
)

type Handler struct {
	client *ledgerclient.Client
	now    func() time.Time
}

func NewHandler(client *ledgerclient.Client) *Handler { return &Handler{client: client, now: time.Now} }
func (h *Handler) scope(ctx context.Context, scope string, headers http.Header) error {
	if strings.TrimSpace(scope) == "" || len(scope) > 128 {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("explicit scope required (at most 128 bytes)"))
	}
	req := connect.NewRequest(&scopes.ListScopesRequest{Scope: scope})
	ledgerclient.ForwardHeaders(headers, req.Header())
	res, err := h.client.Scopes.ListScopes(ctx, req)
	if err != nil {
		return ledgerclient.RPCError("check learning scope", err)
	}
	for _, s := range res.Msg.Scopes {
		if s.Id == scope {
			return nil
		}
	}
	return connect.NewError(connect.CodeNotFound, fmt.Errorf("learning scope is not provisioned"))
}

func (h *Handler) RecordAttempt(ctx context.Context, in *connect.Request[pb.RecordAttemptRequest]) (*connect.Response[pb.RecordAttemptResponse], error) {
	body, err := learning.Encode(in.Msg.Attempt)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	end, _ := time.Parse(time.RFC3339Nano, in.Msg.Attempt.FinishedAt)
	if end.After(h.now()) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("attempt has not finished"))
	}
	if err = h.scope(ctx, in.Msg.Scope, in.Header()); err != nil {
		return nil, err
	}
	// Validate recalled IDs against the exact scope. Evidence references remain
	// owner-verifiable references, not an assertion by this service of their truth.
	for _, u := range in.Msg.Attempt.Advice {
		req := connect.NewRequest(&source.GetEntryRequest{Scope: in.Msg.Scope, Id: u.EntryId})
		ledgerclient.ForwardHeaders(in.Header(), req.Header())
		entry, e := h.client.Journal.GetEntry(ctx, req)
		if e != nil {
			return nil, ledgerclient.RPCError("resolve advice entry", e)
		}
		start, _ := time.Parse(time.RFC3339Nano, in.Msg.Attempt.StartedAt)
		if entry.Msg.Entry == nil || entry.Msg.Entry.CreatedAt == nil || entry.Msg.Entry.CreatedAt.AsTime().After(start) {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("advice must exist before the attempt starts"))
		}
	}
	req := connect.NewRequest(&source.AppendEntryRequest{Scope: in.Msg.Scope, Kind: "task-record", Body: body, ImportProvenance: &source.ImportProvenance{Runtime: "vrooli-memory.learning/v1", SourceLocator: in.Msg.Attempt.AttemptId, ContentHash: "immutable-attempt"}})
	ledgerclient.ForwardHeaders(in.Header(), req.Header())
	res, err := h.client.Journal.AppendEntry(ctx, req)
	if err != nil {
		return nil, ledgerclient.RPCError("record learning attempt", err)
	}
	if res.Msg.Entry == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("missing journal entry"))
	}
	if res.Msg.Entry.Body != body {
		return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("attempt_id already records a different outcome; use a new attempt ID"))
	}
	return connect.NewResponse(&pb.RecordAttemptResponse{EntryId: res.Msg.Entry.Id, Existing: res.Msg.Existing}), nil
}

func (h *Handler) RecordObservation(ctx context.Context, in *connect.Request[pb.RecordObservationRequest]) (*connect.Response[pb.RecordObservationResponse], error) {
	if err := h.scope(ctx, in.Msg.Scope, in.Header()); err != nil {
		return nil, err
	}
	o := in.Msg.Observation
	if err := learning.ValidateObservation(o); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	get := connect.NewRequest(&source.GetEntryRequest{Scope: in.Msg.Scope, Id: o.AttemptId})
	ledgerclient.ForwardHeaders(in.Header(), get.Header())
	entry, err := h.client.Journal.GetEntry(ctx, get)
	if err != nil {
		return nil, ledgerclient.RPCError("resolve observation attempt", err)
	}
	if entry.Msg.Entry == nil || entry.Msg.Entry.Kind != "task-record" {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("attempt_id is not a task record in this scope"))
	}
	attempt, err := learning.Decode(entry.Msg.Entry.Body)
	if err != nil || attempt.AttemptId != o.AttemptId {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("attempt_id does not identify a valid attempt"))
	}
	body, err := learning.EncodeObservation(o)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	req := connect.NewRequest(&source.AppendEntryRequest{Scope: in.Msg.Scope, Kind: "attempt-observation", Body: body, ImportProvenance: &source.ImportProvenance{Runtime: "vrooli-memory.learning/v1", SourceLocator: o.ObservationId, ContentHash: "immutable-observation"}})
	ledgerclient.ForwardHeaders(in.Header(), req.Header())
	res, err := h.client.Journal.AppendEntry(ctx, req)
	if err != nil {
		return nil, ledgerclient.RPCError("record learning observation", err)
	}
	if res.Msg.Entry == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("missing journal entry"))
	}
	if res.Msg.Entry.Body != body {
		return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("observation_id already records a different observation"))
	}
	return connect.NewResponse(&pb.RecordObservationResponse{EntryId: res.Msg.Entry.Id, Existing: res.Msg.Existing}), nil
}

func (h *Handler) MeasureLearning(ctx context.Context, in *connect.Request[pb.MeasureLearningRequest]) (*connect.Response[pb.MeasureLearningResponse], error) {
	now := h.now().UTC()
	to := now
	from := now.Add(-7 * 24 * time.Hour)
	var err error
	if in.Msg.To != "" {
		to, err = time.Parse(time.RFC3339Nano, in.Msg.To)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("to must be RFC3339"))
		}
		from = to.Add(-7 * 24 * time.Hour)
	}
	if in.Msg.From != "" {
		from, err = time.Parse(time.RFC3339Nano, in.Msg.From)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("from must be RFC3339"))
		}
	}
	if !from.Before(to) || to.After(now) || to.Sub(from) > 90*24*time.Hour {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("window must be ordered, historical, and at most 90 days"))
	}
	if err = h.scope(ctx, in.Msg.Scope, in.Header()); err != nil {
		return nil, err
	}
	const pageSize = 100
	const maxPages = 10
	var entries []*source.Entry
	cursor := ""
	truncated := false
	seen := map[string]bool{}
	for page := 0; page < maxPages; page++ {
		req := connect.NewRequest(&source.ListEntriesRequest{Scope: in.Msg.Scope, Limit: pageSize, Cursor: cursor})
		ledgerclient.ForwardHeaders(in.Header(), req.Header())
		res, e := h.client.Journal.ListEntries(ctx, req)
		if e != nil {
			return nil, ledgerclient.RPCError("measure learning", e)
		}
		entries = append(entries, res.Msg.Entries...)
		next := res.Msg.NextCursor
		if next == "" {
			break
		}
		if seen[next] {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("journal cursor repeated"))
		}
		seen[next] = true
		cursor = next
		if page == maxPages-1 {
			truncated = true
		}
	}
	return connect.NewResponse(learning.Measure(entries, in.Msg.Scope, from, to, in.Msg.Operation, in.Msg.ContextKey, truncated)), nil
}
