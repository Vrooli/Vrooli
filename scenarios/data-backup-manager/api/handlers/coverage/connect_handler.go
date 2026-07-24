package coverage

import (
	"context"
	"log"

	"data-backup-manager/internal/coverage"
	"data-backup-manager/internal/sources"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	coveragev1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/coverage"
	sourcesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/sources"
)

// Deps wires the seams the Connect coverage handler needs.
type Deps struct {
	Service coverage.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the coverage Connect-RPC handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) GetCoverageReport(ctx context.Context, _ *connect.Request[coveragev1.GetCoverageReportRequest]) (*connect.Response[coveragev1.GetCoverageReportResponse], error) {
	report, err := h.deps.Service.Report(ctx)
	if err != nil {
		return nil, h.translate("GetCoverageReport", err)
	}
	return connect.NewResponse(&coveragev1.GetCoverageReportResponse{Report: reportToProto(report)}), nil
}

func (h *connectHandler) AcceptDefaultTargets(ctx context.Context, req *connect.Request[coveragev1.AcceptDefaultTargetsRequest]) (*connect.Response[coveragev1.AcceptDefaultTargetsResponse], error) {
	result, err := h.deps.Service.AcceptDefaults(ctx, coverage.AcceptOptions{
		IncludeSensitive: req.Msg.IncludeSensitive,
		DryRun:           req.Msg.DryRun,
	})
	if err != nil {
		return nil, h.translate("AcceptDefaultTargets", err)
	}
	return connect.NewResponse(acceptToProto(result)), nil
}

// translate maps a domain error to a Connect error, logging only internal ones.
func (h *connectHandler) translate(op string, err error) error {
	connectErr := coverage.ToConnectError(err)
	if connect.CodeOf(connectErr) == connect.CodeInternal {
		h.deps.Logger.Printf("coverage.%s: %v", op, err)
	}
	return connectErr
}

func reportToProto(r coverage.Report) *coveragev1.CoverageReport {
	out := &coveragev1.CoverageReport{
		Summary:            summaryToProto(r.Summary),
		RegisteredTargets:  make([]*coveragev1.RegisteredTarget, 0, len(r.Registered)),
		RecommendedTargets: make([]*coveragev1.SuggestedTarget, 0, len(r.Recommended)),
		SensitiveTargets:   make([]*coveragev1.SuggestedTarget, 0, len(r.Sensitive)),
	}
	for _, t := range r.Registered {
		out.RegisteredTargets = append(out.RegisteredTargets, registeredToProto(t))
	}
	for _, s := range r.Recommended {
		out.RecommendedTargets = append(out.RecommendedTargets, suggestionToProto(s))
	}
	for _, s := range r.Sensitive {
		out.SensitiveTargets = append(out.SensitiveTargets, suggestionToProto(s))
	}
	return out
}

func summaryToProto(s coverage.Summary) *coveragev1.CoverageSummary {
	return &coveragev1.CoverageSummary{
		RegisteredCount:               int32(s.RegisteredCount),
		RecommendedCount:              int32(s.RecommendedCount),
		SensitiveCount:                int32(s.SensitiveCount),
		PlannedCount:                  int32(s.PlannedCount),
		BackedUpCount:                 int32(s.BackedUpCount),
		VerifiedCount:                 int32(s.VerifiedCount),
		DefaultCoverageComplete:       s.DefaultCoverageComplete,
		HasSensitiveUnreviewed:        s.HasSensitiveUnreviewed,
		HasUnplannedRegisteredTargets: s.HasUnplannedRegisteredTargets,
		HasUnverifiedTargets:          s.HasUnverifiedTargets,
	}
}

func registeredToProto(t coverage.RegisteredTarget) *coveragev1.RegisteredTarget {
	out := &coveragev1.RegisteredTarget{
		Id:         t.ID,
		Owner:      t.Owner,
		Name:       t.Name,
		SourceKind: kindToProto(t.SourceKind),
		Locator:    t.Locator,
		Planned:    t.Planned,
	}
	if !t.LastSuccessAt.IsZero() {
		out.LastSuccessAt = timestamppb.New(t.LastSuccessAt)
	}
	if !t.LastVerifiedAt.IsZero() {
		out.LastVerifiedAt = timestamppb.New(t.LastVerifiedAt)
	}
	return out
}

func suggestionToProto(s coverage.Suggestion) *coveragev1.SuggestedTarget {
	return &coveragev1.SuggestedTarget{
		Id:          s.ID,
		Owner:       s.Owner,
		Name:        s.Name,
		SourceKind:  kindToProto(s.SourceKind),
		Locator:     s.Locator,
		Rationale:   s.Rationale,
		ApproxBytes: s.ApproxBytes,
		Sensitive:   s.Sensitive,
		Warning:     s.Warning,
	}
}

func acceptToProto(r coverage.AcceptResult) *coveragev1.AcceptDefaultTargetsResponse {
	out := &coveragev1.AcceptDefaultTargetsResponse{
		Accepted:         make([]*coveragev1.AcceptedTarget, 0, len(r.Accepted)),
		SkippedSensitive: make([]*coveragev1.SuggestedTarget, 0, len(r.SkippedSensitive)),
		Failed:           make([]*coveragev1.AcceptError, 0, len(r.Failed)),
		DryRun:           r.DryRun,
	}
	for _, a := range r.Accepted {
		out.Accepted = append(out.Accepted, &coveragev1.AcceptedTarget{
			TargetId:     a.TargetID,
			SuggestionId: a.SuggestionID,
			Owner:        a.Owner,
			Name:         a.Name,
			SourceKind:   kindToProto(a.SourceKind),
			Locator:      a.Locator,
			Sensitive:    a.Sensitive,
		})
	}
	for _, s := range r.SkippedSensitive {
		out.SkippedSensitive = append(out.SkippedSensitive, suggestionToProto(s))
	}
	for _, f := range r.Failed {
		out.Failed = append(out.Failed, &coveragev1.AcceptError{
			SuggestionId: f.SuggestionID,
			Owner:        f.Owner,
			Name:         f.Name,
			Message:      f.Message,
		})
	}
	return out
}

// kindToProto translates the domain SourceKind to the proto enum so domain code
// never imports the generated enum (mirrors the targets/discovery handlers).
func kindToProto(k sources.SourceKind) sourcesv1.SourceKind {
	switch k {
	case sources.KindFilesystem:
		return sourcesv1.SourceKind_SOURCE_KIND_FILESYSTEM
	case sources.KindSQLite:
		return sourcesv1.SourceKind_SOURCE_KIND_SQLITE
	case sources.KindPostgres:
		return sourcesv1.SourceKind_SOURCE_KIND_POSTGRES
	case sources.KindRedis:
		return sourcesv1.SourceKind_SOURCE_KIND_REDIS
	case sources.KindQdrant:
		return sourcesv1.SourceKind_SOURCE_KIND_QDRANT
	case sources.KindObjectStorage:
		return sourcesv1.SourceKind_SOURCE_KIND_OBJECT_STORAGE
	default:
		return sourcesv1.SourceKind_SOURCE_KIND_UNSPECIFIED
	}
}
