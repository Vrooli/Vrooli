package findings

import (
	"context"
	"errors"
	"log"

	"web-search/internal/findingindex"
	"web-search/internal/findings"

	"github.com/vrooli/api-core/schedule"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	findingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-search/v1/findings"
)

// Searcher is the semantic read path the SearchFindings RPC depends on. The
// production impl is *findingindex.Service; tests inject a fake.
type Searcher interface {
	Search(ctx context.Context, query string, limit int) ([]findingindex.Hit, string, error)
}

// Surfacer is the usage-telemetry seam (OT-P2-001): SearchFindings enqueues the
// ids it surfaced for asynchronous counting. The production impl is
// *findings.UsageRecorder; a nil Surfacer disables telemetry. The contract is
// fire-and-forget — the call MUST NOT block or fail the search response.
type Surfacer interface {
	Surfaced(ids []string)
}

// GCRunner runs the OT-P2-003 store-consistency pass. The production impl is
// *findings.GCService; a nil GC makes RunGC return Unavailable.
type GCRunner interface {
	Run(ctx context.Context, dryRun bool) (findings.GCReport, error)
}

// Deps wires the seams the Connect findings handler needs.
type Deps struct {
	Service  findings.Service
	Searcher Searcher
	Surfacer Surfacer
	GC       GCRunner
	// Clock anchors the CountFindings measure's relative time-window resolution.
	Clock  schedule.Clock
	Logger *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect findings handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	if d.Clock == nil {
		d.Clock = schedule.System()
	}
	return &connectHandler{deps: d}
}

// recordSurfaced enqueues the ids of the surfaced hits for async usage counting.
// No-op when no Surfacer is wired. Best-effort: it never blocks the response.
func (h *connectHandler) recordSurfaced(hits []*findingsv1.FindingHit) {
	if h.deps.Surfacer == nil || len(hits) == 0 {
		return
	}
	ids := make([]string, 0, len(hits))
	for _, hit := range hits {
		if hit.GetFinding() != nil {
			ids = append(ids, hit.GetFinding().GetId())
		}
	}
	h.deps.Surfacer.Surfaced(ids)
}

func (h *connectHandler) logIfInternal(op string, err, connectErr error) {
	if connect.CodeOf(connectErr) == connect.CodeInternal {
		h.deps.Logger.Printf("findings.%s: %v", op, err)
	}
}

func (h *connectHandler) ListFindings(ctx context.Context, req *connect.Request[findingsv1.ListFindingsRequest]) (*connect.Response[findingsv1.ListFindingsResponse], error) {
	results, err := h.deps.Service.List(ctx, findings.ListFilter{
		Status:          statusFromProto(req.Msg.GetStatus()),
		IncludeArchived: req.Msg.GetIncludeArchived(),
		Limit:           int(req.Msg.GetLimit()),
	})
	if err != nil {
		connectErr := findings.ToConnectError(err)
		h.logIfInternal("ListFindings", err, connectErr)
		return nil, connectErr
	}
	resp := &findingsv1.ListFindingsResponse{Findings: make([]*findingsv1.Finding, 0, len(results))}
	for _, f := range results {
		resp.Findings = append(resp.Findings, domainToProto(f))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetFinding(ctx context.Context, req *connect.Request[findingsv1.GetFindingRequest]) (*connect.Response[findingsv1.GetFindingResponse], error) {
	got, err := h.deps.Service.Get(ctx, req.Msg.GetId())
	if err != nil {
		connectErr := findings.ToConnectError(err)
		h.logIfInternal("GetFinding", err, connectErr)
		return nil, connectErr
	}
	return connect.NewResponse(&findingsv1.GetFindingResponse{Finding: domainToProto(got)}), nil
}

func (h *connectHandler) AddFinding(ctx context.Context, req *connect.Request[findingsv1.AddFindingRequest]) (*connect.Response[findingsv1.AddFindingResponse], error) {
	cites := make([]findings.NewCitation, 0, len(req.Msg.GetCitations()))
	for _, c := range req.Msg.GetCitations() {
		cites = append(cites, findings.NewCitation{URL: c.GetUrl(), Title: c.GetTitle()})
	}
	created, err := h.deps.Service.Add(ctx, findings.NewFinding{
		Claim:      req.Msg.GetClaim(),
		Confidence: req.Msg.GetConfidence(),
		Query:      req.Msg.GetQuery(),
		Source:     sourceFromProto(req.Msg.GetSource()),
		BriefID:    req.Msg.GetBriefId(),
		Citations:  cites,
	})
	if err != nil {
		connectErr := findings.ToConnectError(err)
		h.logIfInternal("AddFinding", err, connectErr)
		return nil, connectErr
	}
	return connect.NewResponse(&findingsv1.AddFindingResponse{Finding: domainToProto(created)}), nil
}

func (h *connectHandler) EditFinding(ctx context.Context, req *connect.Request[findingsv1.EditFindingRequest]) (*connect.Response[findingsv1.EditFindingResponse], error) {
	edited, err := h.deps.Service.Edit(ctx, req.Msg.GetId(), findings.EditInput{
		Claim:      req.Msg.GetClaim(),
		Confidence: req.Msg.GetConfidence(),
	})
	if err != nil {
		connectErr := findings.ToConnectError(err)
		h.logIfInternal("EditFinding", err, connectErr)
		return nil, connectErr
	}
	return connect.NewResponse(&findingsv1.EditFindingResponse{Finding: domainToProto(edited)}), nil
}

func (h *connectHandler) SupersedeFinding(ctx context.Context, req *connect.Request[findingsv1.SupersedeFindingRequest]) (*connect.Response[findingsv1.SupersedeFindingResponse], error) {
	f, err := h.deps.Service.Supersede(ctx, req.Msg.GetId(), req.Msg.GetReplacement(), req.Msg.GetReason())
	if err != nil {
		connectErr := findings.ToConnectError(err)
		h.logIfInternal("SupersedeFinding", err, connectErr)
		return nil, connectErr
	}
	return connect.NewResponse(&findingsv1.SupersedeFindingResponse{Finding: domainToProto(f)}), nil
}

func (h *connectHandler) FlagFinding(ctx context.Context, req *connect.Request[findingsv1.FlagFindingRequest]) (*connect.Response[findingsv1.FlagFindingResponse], error) {
	f, err := h.deps.Service.Flag(ctx, req.Msg.GetId(), req.Msg.GetReason())
	if err != nil {
		connectErr := findings.ToConnectError(err)
		h.logIfInternal("FlagFinding", err, connectErr)
		return nil, connectErr
	}
	return connect.NewResponse(&findingsv1.FlagFindingResponse{Finding: domainToProto(f)}), nil
}

func (h *connectHandler) ListDisputes(ctx context.Context, req *connect.Request[findingsv1.ListDisputesRequest]) (*connect.Response[findingsv1.ListDisputesResponse], error) {
	results, err := h.deps.Service.List(ctx, findings.ListFilter{
		Status: findings.StatusDisputed,
		Limit:  int(req.Msg.GetLimit()),
	})
	if err != nil {
		connectErr := findings.ToConnectError(err)
		h.logIfInternal("ListDisputes", err, connectErr)
		return nil, connectErr
	}
	resp := &findingsv1.ListDisputesResponse{Findings: make([]*findingsv1.Finding, 0, len(results))}
	for _, f := range results {
		resp.Findings = append(resp.Findings, domainToProto(f))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ResolveDispute(ctx context.Context, req *connect.Request[findingsv1.ResolveDisputeRequest]) (*connect.Response[findingsv1.ResolveDisputeResponse], error) {
	f, err := h.deps.Service.ResolveDispute(ctx, req.Msg.GetId(), req.Msg.GetResolution(), req.Msg.GetReplacement(), req.Msg.GetReason())
	if err != nil {
		connectErr := findings.ToConnectError(err)
		h.logIfInternal("ResolveDispute", err, connectErr)
		return nil, connectErr
	}
	return connect.NewResponse(&findingsv1.ResolveDisputeResponse{Finding: domainToProto(f)}), nil
}

func (h *connectHandler) PruneFindings(ctx context.Context, req *connect.Request[findingsv1.PruneFindingsRequest]) (*connect.Response[findingsv1.PruneFindingsResponse], error) {
	ids, err := h.deps.Service.Prune(ctx, req.Msg.GetDryRun())
	if err != nil {
		connectErr := findings.ToConnectError(err)
		h.logIfInternal("PruneFindings", err, connectErr)
		return nil, connectErr
	}
	return connect.NewResponse(&findingsv1.PruneFindingsResponse{
		Pruned:     int32(len(ids)),
		FindingIds: ids,
	}), nil
}

// SearchFindings runs the semantic read path, projecting hit ids back to full
// findings loaded from SQLite (so citations + timestamps are exact). Superseded
// findings are not in the index; when include_archived is set a SQL fallback
// appends matching superseded findings as weak, zero-score hits.
func (h *connectHandler) SearchFindings(ctx context.Context, req *connect.Request[findingsv1.SearchFindingsRequest]) (*connect.Response[findingsv1.SearchFindingsResponse], error) {
	limit := int(req.Msg.GetLimit())
	if limit <= 0 {
		limit = 10
	}
	resp := &findingsv1.SearchFindingsResponse{Method: "text"}
	if h.deps.Searcher != nil {
		hits, method, err := h.deps.Searcher.Search(ctx, req.Msg.GetQuery(), limit)
		if err != nil {
			h.deps.Logger.Printf("findings.SearchFindings: %v", err)
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		resp.Method = method
		ids := make([]string, 0, len(hits))
		for _, hit := range hits {
			ids = append(ids, hit.FindingID)
		}
		byID, err := h.deps.Service.GetMany(ctx, ids)
		if err != nil {
			h.deps.Logger.Printf("findings.SearchFindings load: %v", err)
			return nil, findings.ToConnectError(err)
		}
		for _, hit := range hits {
			f, ok := byID[hit.FindingID]
			if !ok {
				continue
			}
			// Defensive contract enforcement: superseded findings are excluded
			// from the default read path even if the index returns one (it
			// should not — only active + disputed are indexed).
			if f.Status == findings.StatusSuperseded && !req.Msg.GetIncludeArchived() {
				continue
			}
			resp.Hits = append(resp.Hits, &findingsv1.FindingHit{
				Finding: domainToProto(f),
				Score:   hit.Score,
				Weak:    hit.Weak,
			})
		}
		// Usage telemetry (OT-P2-001): record the findings this search surfaced.
		// Fire-and-forget — never blocks or fails the response. Superseded/weak
		// archived appends below are NOT counted as surfacings (they are a
		// fallback, not a semantic match).
		h.recordSurfaced(resp.Hits)
	}
	if req.Msg.GetIncludeArchived() {
		archived, err := h.deps.Service.SearchArchivedLike(ctx, req.Msg.GetQuery(), limit)
		if err != nil {
			h.deps.Logger.Printf("findings.SearchFindings archived: %v", err)
			return nil, findings.ToConnectError(err)
		}
		for _, f := range archived {
			resp.Hits = append(resp.Hits, &findingsv1.FindingHit{
				Finding: domainToProto(f),
				Score:   0,
				Weak:    true,
			})
		}
	}
	return connect.NewResponse(resp), nil
}

// CountFindings answers the findings.count measure: it resolves the request's
// canonical TimeWindow to a [from, to) range (defaulting to this_week) and
// returns the count of findings captured in it.
func (h *connectHandler) CountFindings(ctx context.Context, req *connect.Request[findingsv1.CountFindingsRequest]) (*connect.Response[findingsv1.CountFindingsResponse], error) {
	rng, err := resolveCountWindow(req.Msg.GetWindow(), h.deps.Clock.Now())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	n, err := h.deps.Service.CountInWindow(ctx, rng.From, rng.To)
	if err != nil {
		h.deps.Logger.Printf("findings.CountFindings: %v", err)
		return nil, findings.ToConnectError(err)
	}
	return connect.NewResponse(&findingsv1.CountFindingsResponse{Count: int64(n)}), nil
}

// ListEffectiveness returns findings paired with their usage telemetry and the
// blended effective score (age-decayed confidence × usage factor), computed on
// read against the handler clock (OT-P2-001).
func (h *connectHandler) ListEffectiveness(ctx context.Context, req *connect.Request[findingsv1.ListEffectivenessRequest]) (*connect.Response[findingsv1.ListEffectivenessResponse], error) {
	limit := int(req.Msg.GetLimit())
	pairs, err := h.deps.Service.ListEffectiveness(ctx, req.Msg.GetIncludeDisputed(), limit)
	if err != nil {
		h.deps.Logger.Printf("findings.ListEffectiveness: %v", err)
		return nil, findings.ToConnectError(err)
	}
	now := h.deps.Clock.Now()
	resp := &findingsv1.ListEffectivenessResponse{}
	for _, p := range pairs {
		item := &findingsv1.FindingEffectiveness{
			Finding:             domainToProto(p.Finding),
			SurfacedCount:       int32(p.Usage.SurfacedCount),
			UsedCount:           int32(p.Usage.UsedCount),
			EffectiveConfidence: findings.EffectiveConfidence(p.Finding, now),
			UsageFactor:         findings.UsageFactor(p.Usage, p.Finding, now),
			EffectiveScore:      findings.EffectiveScore(p.Finding, p.Usage, now),
		}
		if !p.Usage.LastSurfacedAt.IsZero() {
			item.LastSurfacedAt = timestamppb.New(p.Usage.LastSurfacedAt)
		}
		resp.Items = append(resp.Items, item)
	}
	return connect.NewResponse(resp), nil
}

// RecordUsage records an explicit "used" signal for a finding.
func (h *connectHandler) RecordUsage(ctx context.Context, req *connect.Request[findingsv1.RecordUsageRequest]) (*connect.Response[findingsv1.RecordUsageResponse], error) {
	f, err := h.deps.Service.RecordUsage(ctx, req.Msg.GetId())
	if err != nil {
		connectErr := findings.ToConnectError(err)
		h.logIfInternal("RecordUsage", err, connectErr)
		return nil, connectErr
	}
	return connect.NewResponse(&findingsv1.RecordUsageResponse{Finding: domainToProto(f)}), nil
}

// RunGC runs the OT-P2-003 store-consistency pass (or, with dry_run, reports the
// candidates without mutating).
func (h *connectHandler) RunGC(ctx context.Context, req *connect.Request[findingsv1.RunGCRequest]) (*connect.Response[findingsv1.RunGCResponse], error) {
	if h.deps.GC == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("findings GC not configured"))
	}
	report, err := h.deps.GC.Run(ctx, req.Msg.GetDryRun())
	if err != nil {
		h.deps.Logger.Printf("findings.RunGC: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&findingsv1.RunGCResponse{
		DryRun:                report.DryRun,
		SupersededDecayed:     report.SupersededDecayed,
		ColdArchiveCandidates: report.ColdArchiveCandidates,
		StaleDisputes:         report.StaleDisputes,
		Orphans:               report.Orphans,
	}), nil
}
