package rules

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"
	rulesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/rules"
	internalfacets "vrooli-memory/internal/facets"
)

type connectHandler struct {
	service *internalfacets.Service
	logger  *log.Logger
}

func NewConnectHandler(service *internalfacets.Service, logger *log.Logger) *connectHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &connectHandler{service: service, logger: logger}
}

func (h *connectHandler) ListRules(ctx context.Context, req *connect.Request[rulesv1.ListRulesRequest]) (*connect.Response[rulesv1.ListRulesResponse], error) {
	rules, err := h.service.ListRules(ctx, req.Msg.GetScope())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &rulesv1.ListRulesResponse{}
	for _, rule := range rules {
		response.Rules = append(response.Rules, ruleProto(rule))
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) CreateRule(ctx context.Context, req *connect.Request[rulesv1.CreateRuleRequest]) (*connect.Response[rulesv1.CreateRuleResponse], error) {
	in := req.Msg.GetRule()
	if in == nil || in.GetFacetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("facet_id is required"))
	}
	rule, err := h.service.CreateRule(ctx, internalfacets.Rule{ID: in.GetId(), Scope: in.GetScope(), Priority: int(in.GetPriority()), FacetID: in.GetFacetId(), SourceRuntime: in.GetSourceRuntime(), Kind: in.GetKind(), SourcePathGlob: in.GetSourcePathGlob(), BodyPattern: in.GetBodyPattern()})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&rulesv1.CreateRuleResponse{Rule: ruleProto(rule)}), nil
}

func (h *connectHandler) DryRunRule(ctx context.Context, req *connect.Request[rulesv1.DryRunRuleRequest]) (*connect.Response[rulesv1.DryRunRuleResponse], error) {
	dry, err := h.service.DryRunRule(ctx, req.Msg.GetRuleId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&rulesv1.DryRunRuleResponse{RuleId: dry.RuleID, CorpusFingerprint: dry.CorpusFingerprint, MatchCount: int32(dry.MatchCount), Samples: dry.Samples}), nil
}

func (h *connectHandler) EnableRule(ctx context.Context, req *connect.Request[rulesv1.EnableRuleRequest]) (*connect.Response[rulesv1.EnableRuleResponse], error) {
	if err := h.service.EnableRule(ctx, req.Msg.GetRuleId()); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&rulesv1.EnableRuleResponse{}), nil
}

func (h *connectHandler) RevertRule(ctx context.Context, req *connect.Request[rulesv1.RevertRuleRequest]) (*connect.Response[rulesv1.RevertRuleResponse], error) {
	count, err := h.service.RevertRule(ctx, req.Msg.GetRuleId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&rulesv1.RevertRuleResponse{RestoredCount: int32(count)}), nil
}

func (h *connectHandler) RefacetCorpus(ctx context.Context, req *connect.Request[rulesv1.RefacetCorpusRequest]) (*connect.Response[rulesv1.RefacetCorpusResponse], error) {
	result, err := h.service.RefacetCorpus(ctx, req.Msg.GetScope())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&rulesv1.RefacetCorpusResponse{
		Total:        int32(result.Total),
		Assigned:     int32(result.Assigned),
		RuleAssigned: int32(result.RuleAssigned),
		Classified:   int32(result.Classified),
		Failed:       int32(result.Failed),
	}), nil
}

func ruleProto(rule internalfacets.Rule) *rulesv1.Rule {
	return &rulesv1.Rule{Id: rule.ID, Scope: rule.Scope, Priority: int32(rule.Priority), FacetId: rule.FacetID, SourceRuntime: rule.SourceRuntime, Kind: rule.Kind, SourcePathGlob: rule.SourcePathGlob, BodyPattern: rule.BodyPattern, Enabled: rule.Enabled}
}
