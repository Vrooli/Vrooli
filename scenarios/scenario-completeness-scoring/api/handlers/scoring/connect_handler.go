package scoring

import (
	"context"
	"errors"
	"log"
	"math"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	scoringv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-completeness-scoring/v1/scoring"

	internalscoring "scenario-completeness-scoring/internal/scoring"
)

// Scorer is the seam the Connect handler calls; production wires
// *internalscoring.Service, tests substitute a stub.
type Scorer interface {
	GetScore(scenario string) (internalscoring.Result, error)
}

// Deps wires the Connect scoring handler.
type Deps struct {
	Scorer Scorer
	Logger *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the ScoreService implementation.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) GetScore(ctx context.Context, req *connect.Request[scoringv1.GetScoreRequest]) (*connect.Response[scoringv1.GetScoreResponse], error) {
	result, err := h.deps.Scorer.GetScore(req.Msg.GetScenario())
	if err != nil {
		if errors.Is(err, internalscoring.ErrUnknownScenario) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		h.deps.Logger.Printf("scoring.GetScore: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(resultToProto(result)), nil
}

// resultToProto converts the domain result to the wire contract. Purely
// mechanical; all semantics live in internal/scoring.
func resultToProto(r internalscoring.Result) *scoringv1.GetScoreResponse {
	out := &scoringv1.GetScoreResponse{
		Scenario: r.Scenario,
		Category: r.Category,
		Maturity: &scoringv1.MaturityHeadline{
			WorkingRung:      r.Maturity.WorkingRung,
			LadderClean:      r.Maturity.LadderClean,
			SatisfiedThrough: r.Maturity.SatisfiedThrough,
			BuildPassing:     r.Maturity.BuildPassing,
		},
		Composite: &scoringv1.CompositeScore{
			Score:               boundedInt32(r.Composite.Score),
			Classification:      r.Composite.Classification,
			ClassificationLabel: r.Composite.ClassificationLabel,
		},
		Freshness: &scoringv1.FreshnessBlock{
			CurrentDigest:    r.Freshness.Digest,
			DigestError:      r.Freshness.DigestErr,
			SuggestedCommand: r.Freshness.SuggestedCommand,
		},
		CalculatedAt: timestamppb.New(r.CalculatedAt),
	}

	for _, d := range r.Maturity.Dimensions {
		out.Maturity.Dimensions = append(out.Maturity.Dimensions, &scoringv1.DimensionCount{
			Dimension:   d.Dimension,
			ErrorPlus:   boundedInt32(d.ErrorPlus),
			Total:       boundedInt32(d.Total),
			Approximate: d.Approximate,
		})
	}

	for _, g := range r.Composite.Groups {
		pg := &scoringv1.ScoreGroup{
			Id:    g.ID,
			Label: g.Label,
			Score: g.Score,
			Max:   g.Max,
		}
		for _, m := range g.Metrics {
			pg.Metrics = append(pg.Metrics, &scoringv1.MetricLine{
				Id:        m.ID,
				Label:     m.Label,
				Observed:  m.Observed,
				Points:    m.Points,
				MaxPoints: m.MaxPoints,
				Threshold: m.Threshold,
			})
		}
		out.Composite.Groups = append(out.Composite.Groups, pg)
	}

	for _, p := range r.Freshness.Phases {
		pf := &scoringv1.PhaseFreshness{
			Phase:      p.Phase,
			Verdict:    p.Verdict,
			LastRunId:  p.LastRunID,
			LastDigest: p.LastDigest,
			LastStatus: p.LastStatus,
		}
		if !p.LastRunAt.IsZero() {
			pf.LastRunAt = timestamppb.New(p.LastRunAt)
		}
		out.Freshness.Phases = append(out.Freshness.Phases, pf)
	}

	for _, rec := range r.Recommends {
		out.Recommendations = append(out.Recommendations, &scoringv1.Recommendation{
			Priority:     rec.Priority,
			Description:  rec.Description,
			ImpactPoints: rec.Impact,
		})
	}

	for _, ap := range r.ActionPlan {
		out.ActionPlan = append(out.ActionPlan, &scoringv1.ActionPhase{
			Title:           ap.Title,
			Actions:         ap.Actions,
			EstimatedPoints: ap.EstimatedPoints,
		})
	}

	for _, d := range r.Degradations {
		out.Degradations = append(out.Degradations, &scoringv1.CollectorDegradation{
			Collector: d.Collector,
			State:     d.State,
			Reason:    d.Reason,
		})
	}

	if r.Importance != nil {
		out.Importance = &scoringv1.ImportanceSummary{
			Score:          r.Importance.Score,
			SystemRequired: r.Importance.SystemRequired,
			Components: &scoringv1.ImportanceComponents{
				Centrality:    r.Importance.Components.Centrality,
				CoreProximity: r.Importance.Components.CoreProximity,
				Recency:       r.Importance.Components.Recency,
			},
			Signals: &scoringv1.ImportanceSignals{
				DirectReverseDependencyCount:     boundedInt32(r.Importance.Signals.DirectReverseDependencyCount),
				TransitiveReverseDependencyCount: boundedInt32(r.Importance.Signals.TransitiveReverseDependencyCount),
				RequiredReverseDependencyCount:   boundedInt32(r.Importance.Signals.RequiredReverseDependencyCount),
				RequiredEdgeWeightedScore:        r.Importance.Signals.RequiredEdgeWeightedScore,
				DistanceToCoreSeed:               boundedInt32(r.Importance.Signals.DistanceToCoreSeed),
				NearestCoreSeed:                  r.Importance.Signals.NearestCoreSeed,
				RecentActivityCount:              boundedInt32(r.Importance.Signals.RecentActivityCount),
			},
			Degraded: append([]string(nil), r.Importance.Degraded...),
		}
	}

	return out
}

func boundedInt32(value int) int32 {
	if value > math.MaxInt32 {
		return math.MaxInt32
	}
	if value < math.MinInt32 {
		return math.MinInt32
	}
	return int32(value)
}
