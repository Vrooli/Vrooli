// Package signals is the Connect-RPC surface for the signals domain.
// Translates between proto wire types and internal/signals domain
// types.
package signals

import (
	"context"
	"errors"
	"strings"

	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/signals"
	"architecture-cartographer/internal/signals/boundaries"

	"connectrpc.com/connect"
	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/graph"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/shared"
	signalsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/signals"
	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/signals/signals_v1connect"
)

// Handler implements signals_v1connect.SignalsServiceHandler.
type Handler struct {
	signals_v1connect.UnimplementedSignalsServiceHandler
	svc signals.Service
}

// NewHandler constructs the Connect handler.
func NewHandler(svc signals.Service) *Handler { return &Handler{svc: svc} }

var _ signals_v1connect.SignalsServiceHandler = (*Handler)(nil)

func (h *Handler) ScoreChunk(ctx context.Context, req *connect.Request[signalsv1.ScoreChunkRequest]) (*connect.Response[signalsv1.ScoreChunkResponse], error) {
	in, err := scoreInputFromProto(req.Msg.GetScenario(), req.Msg.GetChunk(), req.Msg.GetFileId(), req.Msg.GetRepoPath())
	if err != nil {
		return nil, err
	}
	v, sErr := h.svc.ScoreChunk(ctx, in)
	if sErr != nil {
		return nil, connect.NewError(signals.ErrorToConnectCode(sErr), sErr)
	}
	return connect.NewResponse(&signalsv1.ScoreChunkResponse{Verdict: verdictToProto(v)}), nil
}

func (h *Handler) ExplainVerdict(ctx context.Context, req *connect.Request[signalsv1.ExplainVerdictRequest]) (*connect.Response[signalsv1.ExplainVerdictResponse], error) {
	in, err := scoreInputFromProto(req.Msg.GetScenario(), req.Msg.GetChunk(), req.Msg.GetFileId(), req.Msg.GetRepoPath())
	if err != nil {
		return nil, err
	}
	v, sErr := h.svc.ExplainVerdict(ctx, in)
	if sErr != nil {
		return nil, connect.NewError(signals.ErrorToConnectCode(sErr), sErr)
	}
	return connect.NewResponse(&signalsv1.ExplainVerdictResponse{Verdict: verdictToProto(v)}), nil
}

func (h *Handler) ListSignals(ctx context.Context, req *connect.Request[signalsv1.ListSignalsRequest]) (*connect.Response[signalsv1.ListSignalsResponse], error) {
	descs, err := h.svc.ListSignals(ctx, strings.TrimSpace(req.Msg.GetScenario()))
	if err != nil {
		return nil, connect.NewError(signals.ErrorToConnectCode(err), err)
	}
	out := &signalsv1.ListSignalsResponse{}
	for _, d := range descs {
		out.Signals = append(out.Signals, &signalsv1.SignalDescriptor{
			Name:           d.Name,
			DefaultWeight:  d.DefaultWeight,
			Stability:      d.Stability,
			Description:    d.Description,
			Disabled:       d.Disabled,
			DisabledReason: d.DisabledReason,
		})
	}
	return connect.NewResponse(out), nil
}

func (h *Handler) BoundaryHealth(ctx context.Context, req *connect.Request[signalsv1.BoundaryHealthRequest]) (*connect.Response[signalsv1.BoundaryHealthResponse], error) {
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	rep, err := h.svc.BoundaryHealth(ctx, scenario)
	if err != nil {
		return nil, connect.NewError(signals.ErrorToConnectCode(err), err)
	}
	out := &signalsv1.BoundaryHealthResponse{
		Scenario:     rep.Scenario,
		TotalDomains: int32(rep.TotalDomains),
	}
	for _, dc := range rep.Domains {
		out.Domains = append(out.Domains, domainCouplingToProto(dc))
	}
	return connect.NewResponse(out), nil
}

func domainCouplingToProto(dc boundaries.DomainCoupling) *signalsv1.DomainCoupling {
	out := &signalsv1.DomainCoupling{
		Domain:       dc.Domain,
		Archetype:    dc.Archetype,
		Efferent:     int32(dc.Efferent),
		Afferent:     int32(dc.Afferent),
		Instability:  dc.Instability,
		FanOut:       dc.FanOut,
		DependsOn:    append([]string(nil), dc.DependsOn...),
		DependedBy:   append([]string(nil), dc.DependedBy...),
		StableKernel: dc.StableKernel,
		HealthScore:  dc.HealthScore,
	}
	for _, s := range dc.Smells {
		out.Smells = append(out.Smells, &signalsv1.CouplingSmell{
			Kind:     s.Kind,
			Severity: couplingSeverityToProto(s.Severity),
			Message:  s.Message,
		})
	}
	return out
}

func couplingSeverityToProto(s boundaries.Severity) signalsv1.CouplingSeverity {
	switch s {
	case boundaries.SeverityWarn:
		return signalsv1.CouplingSeverity_COUPLING_SEVERITY_WARN
	case boundaries.SeverityInfo:
		return signalsv1.CouplingSeverity_COUPLING_SEVERITY_INFO
	default:
		return signalsv1.CouplingSeverity_COUPLING_SEVERITY_UNSPECIFIED
	}
}

func scoreInputFromProto(scenario string, chunk *graphv1.Chunk, fileID, repoPath string) (signals.ScoreInput, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return signals.ScoreInput{}, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	if chunk == nil && strings.TrimSpace(fileID) == "" && strings.TrimSpace(repoPath) == "" {
		return signals.ScoreInput{}, connect.NewError(connect.CodeInvalidArgument, errors.New("chunk, file_id, or repo_path is required"))
	}
	in := signals.ScoreInput{Scenario: scenario, FileID: fileID, RepoPath: repoPath}
	if chunk != nil {
		in.Chunk = graph.Chunk{
			ID:            chunk.GetId(),
			FileID:        chunk.GetFileId(),
			Path:          chunk.GetPath(),
			CurrentDomain: chunk.GetCurrentDomain(),
		}
	}
	return in, nil
}

// -------------------------- proto<->domain --------------------------

func verdictToProto(v signals.Verdict) *sharedv1.Verdict {
	out := &sharedv1.Verdict{
		ChunkId:        v.ChunkID,
		ChunkPath:      v.ChunkPath,
		Tier:           tierToProto(v.Tier),
		TopDomain:      v.TopDomain,
		TopValue:       v.TopValue,
		RunnerUpDomain: v.RunnerUpDomain,
		RunnerUpValue:  v.RunnerUpValue,
		Tied:           v.Tied,
	}
	for _, s := range v.Scores {
		ps := &sharedv1.Score{
			Signal: s.Signal,
			Domain: s.Domain,
			Value:  s.Value,
			Reason: s.Reason,
		}
		for _, e := range s.Evidence {
			ps.Evidence = append(ps.Evidence, &sharedv1.Evidence{
				Kind:    e.Kind,
				Summary: e.Summary,
				Locator: e.Locator,
				Weight:  e.Weight,
			})
		}
		out.Scores = append(out.Scores, ps)
	}
	for _, d := range v.DomainValues {
		out.DomainValues = append(out.DomainValues, &sharedv1.DomainValue{
			Domain: d.Domain,
			Value:  d.Value,
		})
	}
	for _, a := range v.Abstentions {
		pa := &sharedv1.Abstention{
			Signal: a.Signal,
			Reason: a.Reason,
		}
		for _, e := range a.Evidence {
			pa.Evidence = append(pa.Evidence, &sharedv1.Evidence{
				Kind:    e.Kind,
				Summary: e.Summary,
				Locator: e.Locator,
				Weight:  e.Weight,
			})
		}
		out.Abstentions = append(out.Abstentions, pa)
	}
	return out
}

func tierToProto(t signals.Tier) sharedv1.Tier {
	switch t {
	case signals.TierAutoPlace:
		return sharedv1.Tier_TIER_AUTO_PLACE
	case signals.TierSuggest:
		return sharedv1.Tier_TIER_SUGGEST
	case signals.TierConflict:
		return sharedv1.Tier_TIER_CONFLICT
	default:
		return sharedv1.Tier_TIER_UNSPECIFIED
	}
}
