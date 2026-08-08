package rules

import (
	"connectrpc.com/connect"
	"context"
	sourcev1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/rules"
	sourceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/rules/rules_v1connect"
	memoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/rules"
	memoryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/rules/rules_v1connect"
	"google.golang.org/protobuf/proto"
	"log"
	"vrooli-memory/internal/ledgerclient"
)

type connectHandler struct {
	client sourceconnect.ClassificationRulesServiceClient
	logger *log.Logger
}

func NewConnectHandler(client sourceconnect.ClassificationRulesServiceClient, logger *log.Logger) *connectHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &connectHandler{client: client, logger: logger}
}
func (h *connectHandler) ListRules(ctx context.Context, in *connect.Request[memoryv1.ListRulesRequest]) (*connect.Response[memoryv1.ListRulesResponse], error) {
	return proxy(ctx, in, &sourcev1.ListRulesRequest{}, &memoryv1.ListRulesResponse{}, h.client.ListRules, "list rules")
}
func (h *connectHandler) CreateRule(ctx context.Context, in *connect.Request[memoryv1.CreateRuleRequest]) (*connect.Response[memoryv1.CreateRuleResponse], error) {
	return proxy(ctx, in, &sourcev1.CreateRuleRequest{}, &memoryv1.CreateRuleResponse{}, h.client.CreateRule, "create rule")
}
func (h *connectHandler) DryRunRule(ctx context.Context, in *connect.Request[memoryv1.DryRunRuleRequest]) (*connect.Response[memoryv1.DryRunRuleResponse], error) {
	return proxy(ctx, in, &sourcev1.DryRunRuleRequest{}, &memoryv1.DryRunRuleResponse{}, h.client.DryRunRule, "dry run rule")
}
func (h *connectHandler) EnableRule(ctx context.Context, in *connect.Request[memoryv1.EnableRuleRequest]) (*connect.Response[memoryv1.EnableRuleResponse], error) {
	return proxy(ctx, in, &sourcev1.EnableRuleRequest{}, &memoryv1.EnableRuleResponse{}, h.client.EnableRule, "enable rule")
}
func (h *connectHandler) RevertRule(ctx context.Context, in *connect.Request[memoryv1.RevertRuleRequest]) (*connect.Response[memoryv1.RevertRuleResponse], error) {
	return proxy(ctx, in, &sourcev1.RevertRuleRequest{}, &memoryv1.RevertRuleResponse{}, h.client.RevertRule, "revert rule")
}
func (h *connectHandler) RefacetCorpus(ctx context.Context, in *connect.Request[memoryv1.RefacetCorpusRequest]) (*connect.Response[memoryv1.RefacetCorpusResponse], error) {
	return proxy(ctx, in, &sourcev1.RefacetCorpusRequest{}, &memoryv1.RefacetCorpusResponse{}, h.client.RefacetCorpus, "refacet corpus")
}
func (h *connectHandler) MeasureDistribution(ctx context.Context, in *connect.Request[memoryv1.MeasureDistributionRequest]) (*connect.Response[memoryv1.MeasureDistributionResponse], error) {
	return proxy(ctx, in, &sourcev1.MeasureDistributionRequest{}, &memoryv1.MeasureDistributionResponse{}, h.client.MeasureDistribution, "measure distribution")
}

func proxy[MI, SI, MO, SO any](ctx context.Context, in *connect.Request[MI], src *SI, out *MO, invoke func(context.Context, *connect.Request[SI]) (*connect.Response[SO], error), op string) (*connect.Response[MO], error) {
	req := connect.NewRequest(src)
	if err := ledgerclient.TranslateWithScope(any(in.Msg).(proto.Message), any(req.Msg).(proto.Message), "agent-memory"); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	ledgerclient.ForwardHeaders(in.Header(), req.Header())
	resp, err := invoke(ctx, req)
	if err != nil {
		return nil, ledgerclient.RPCError(op, err)
	}
	if err := ledgerclient.Translate(any(resp.Msg).(proto.Message), any(out).(proto.Message)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

var _ memoryconnect.ClassificationRulesServiceHandler = (*connectHandler)(nil)
