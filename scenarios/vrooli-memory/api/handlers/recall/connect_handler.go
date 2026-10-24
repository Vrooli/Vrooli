package recall

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	journal "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal"
	journalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal/journal_v1connect"

	learningpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/learning"
	"vrooli-memory/internal/learning"

	"connectrpc.com/connect"

	sourcev1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/recall"
	sourceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/recall/recall_v1connect"
	memoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/recall"
	memoryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/recall/recall_v1connect"

	"vrooli-memory/internal/ledgerclient"
)

type connectHandler struct {
	client  sourceconnect.RecallServiceClient
	logger  *log.Logger
	journal journalconnect.JournalServiceClient
}

func NewConnectHandler(client sourceconnect.RecallServiceClient, logger *log.Logger, journals ...journalconnect.JournalServiceClient) *connectHandler {
	if logger == nil {
		logger = log.Default()
	}
	h := &connectHandler{client: client, logger: logger}
	if len(journals) > 0 {
		h.journal = journals[0]
	}
	return h
}

func (h *connectHandler) Recall(ctx context.Context, in *connect.Request[memoryv1.RecallRequest]) (*connect.Response[memoryv1.RecallResponse], error) {
	req := connect.NewRequest(&sourcev1.RecallRequest{})
	if err := ledgerclient.TranslateWithScope(in.Msg, req.Msg, in.Msg.GetScope()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	ledgerclient.ForwardHeaders(in.Header(), req.Header())
	resp, err := h.client.Recall(ctx, req)
	if err != nil {
		return nil, ledgerclient.RPCError("recall", err)
	}
	out := &memoryv1.RecallResponse{}
	if err := ledgerclient.Translate(resp.Msg, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	h.enrichLearning(ctx, in.Msg.GetScope(), in.Header(), out.Hits)
	return connect.NewResponse(out), nil
}

func (h *connectHandler) Wake(ctx context.Context, in *connect.Request[memoryv1.WakeRequest]) (*connect.Response[memoryv1.WakeResponse], error) {
	req := connect.NewRequest(&sourcev1.WakeRequest{})
	if err := ledgerclient.TranslateWithScope(in.Msg, req.Msg, in.Msg.GetScope()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	ledgerclient.ForwardHeaders(in.Header(), req.Header())
	resp, err := h.client.Wake(ctx, req)
	if err != nil {
		return nil, ledgerclient.RPCError("wake", err)
	}
	out := &memoryv1.WakeResponse{}
	if err := ledgerclient.Translate(resp.Msg, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) Zoom(ctx context.Context, in *connect.Request[memoryv1.ZoomRequest]) (*connect.Response[memoryv1.ZoomResponse], error) {
	req := connect.NewRequest(&sourcev1.ZoomRequest{})
	if err := ledgerclient.TranslateWithScope(in.Msg, req.Msg, in.Msg.GetScope()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp, err := h.client.Zoom(ctx, req)
	if err != nil {
		return nil, ledgerclient.RPCError("zoom", err)
	}
	out := &memoryv1.ZoomResponse{}
	if err := ledgerclient.Translate(resp.Msg, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) ListSiblingEvents(ctx context.Context, in *connect.Request[memoryv1.ListSiblingEventsRequest]) (*connect.Response[memoryv1.ListSiblingEventsResponse], error) {
	req := connect.NewRequest(&sourcev1.ListSiblingEventsRequest{})
	if err := ledgerclient.TranslateWithScope(in.Msg, req.Msg, in.Msg.GetScope()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp, err := h.client.ListSiblingEvents(ctx, req)
	if err != nil {
		return nil, ledgerclient.RPCError("list sibling events", err)
	}
	out := &memoryv1.ListSiblingEventsResponse{}
	if err := ledgerclient.Translate(resp.Msg, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

var _ memoryconnect.RecallServiceHandler = (*connectHandler)(nil)

// enrichLearning joins recalled note IDs to bounded authoritative attempt evidence.
// Partial scans carry reliability=false; they never turn missing evidence into zero.
func (h *connectHandler) enrichLearning(ctx context.Context, scope string, headers http.Header, hits []*memoryv1.RecallHit) {
	wanted := false
	for _, hit := range hits {
		if strings.Contains(hit.Text, "preference/v1 ") || strings.Contains(hit.Text, "avoid/v1 ") || strings.Contains(hit.Text, "learning-observation/v1 ") || strings.Contains(hit.Text, "learning-attempt/v1 ") {
			wanted = true
		}
	}
	if !wanted {
		return
	}
	reliable := false
	for _, hit := range hits {
		hit.EvidenceReliable = &reliable
	}
	if h.journal == nil {
		return
	}
	var entries []*journal.Entry
	cursor := ""
	for page := 0; page < 10; page++ {
		req := connect.NewRequest(&journal.ListEntriesRequest{Scope: scope, Limit: 100, Cursor: cursor})
		ledgerclient.ForwardHeaders(headers, req.Header())
		res, err := h.journal.ListEntries(ctx, req)
		if err != nil {
			return
		}
		entries = append(entries, res.Msg.Entries...)
		next := res.Msg.NextCursor
		if next == "" {
			reliable = true
			break
		}
		if next == cursor {
			return
		}
		cursor = next
	}
	for _, hit := range hits {
		hit.EvidenceReliable = nil
	}
	applyLearningEvidence(hits, entries, reliable)
}

func applyLearningEvidence(hits []*memoryv1.RecallHit, entries []*journal.Entry, reliable bool) {
	byID := map[string]*memoryv1.RecallHit{}
	for _, hit := range hits {
		hit.EvidenceReliable = nil
		hit.Supported = 0
		hit.Contradicted = 0
		byID[hit.EntryId] = hit
	}
	attemptsByID := map[string]*learningpb.Attempt{}
	for _, entry := range entries {
		if !strings.Contains(entry.Body, "learning-attempt/v1 ") {
			continue
		}
		if attempt, err := learning.Decode(entry.Body); err == nil {
			attemptsByID[attempt.AttemptId] = attempt
		}
	}
	noteOrigins := map[string]*learningpb.Attempt{}
	for _, entry := range entries {
		hit := byID[entry.Id]
		if hit == nil {
			continue
		}
		if observation, err := learning.DecodeObservation(entry.Body); err == nil {
			if attempt := attemptsByID[observation.AttemptId]; attempt != nil {
				noteOrigins[hit.EntryId] = attempt
				root := attempt
				for depth := 0; depth < 4 && root.GetParentAttemptId() != ""; depth++ {
					parent := attemptsByID[root.GetParentAttemptId()]
					if parent == nil {
						break
					}
					root = parent
				}
				hit.ContextKey = root.ContextKey
				hit.Operation = attempt.Operation
				hit.Provenance = attempt.Provenance
			}
		}
	}
	// Latest later-run observation replaces the earlier tentative verdict for that attempt.
	feedback := map[string]string{}
	feedbackTime := map[string]string{}
	for _, entry := range entries {
		if hit := byID[entry.Id]; hit != nil && entry.CreatedAt != nil {
			hit.CreatedAt = entry.CreatedAt.AsTime().UTC().Format(time.RFC3339Nano)
		}
		if !strings.HasPrefix(entry.Body, "learning-observation/v1 ") {
			continue
		}
		observation, err := learning.DecodeObservation(entry.Body)
		if err != nil {
			reliable = false
			continue
		}
		if observation.Provenance == "test" || (observation.Disposition != "supported" && observation.Disposition != "contradicted") {
			continue
		}
		if observation.ObservedAt >= feedbackTime[observation.AttemptId] {
			feedbackTime[observation.AttemptId] = observation.ObservedAt
			feedback[observation.AttemptId] = observation.Disposition
		}
	}
	// A note's originating outcome matters even before anyone adopts that note.
	for id, attempt := range noteOrigins {
		hit := byID[id]
		verdict := feedback[attempt.AttemptId]
		if verdict == "contradicted" || (verdict == "" && attempt.Outcome == "failed") {
			hit.Contradicted++
		} else if verdict == "supported" || (verdict == "" && attempt.Outcome == "verified_success") {
			hit.Supported++
		} else {
			unknown := false
			hit.EvidenceReliable = &unknown
		}
	}
	seen := map[string]bool{}
	for _, entry := range entries {
		if !strings.Contains(entry.Body, "learning-attempt/v1 ") {
			continue
		}
		attempt, err := learning.Decode(entry.Body)
		if err != nil {
			reliable = false
			continue
		}
		if seen[attempt.AttemptId] || attempt.Provenance == "test" {
			continue
		}
		seen[attempt.AttemptId] = true
		for _, use := range attempt.Advice {
			hit := byID[use.EntryId]
			if hit == nil || use.Decision != "applied" {
				continue
			}
			verdict := use.Verdict
			if later := feedback[attempt.AttemptId]; later != "" {
				verdict = later
			}
			if verdict == "supported" {
				hit.Supported++
			}
			if verdict == "contradicted" {
				hit.Contradicted++
			}
		}
	}
	for _, hit := range hits {
		value := reliable && (hit.EvidenceReliable == nil || hit.GetEvidenceReliable())
		hit.EvidenceReliable = &value
	}
}
