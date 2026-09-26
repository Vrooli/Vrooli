package rules

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	rulesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/rules"
	rulesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/rules/rules_v1connect"
)

type handlers struct {
	client rulesconnect.ClassificationRulesServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, base := cliapp.NewConnectHTTPClientWithTimeout(core, 0)
	return &handlers{client: rulesconnect.NewClassificationRulesServiceClient(httpClient, base)}
}

func scope(ctx cliapp.OperationContext) string {
	if value := ctx.Flag("scope"); value != "" {
		return value
	}
	return "agent-memory"
}

func (h *handlers) listCall(ctx cliapp.OperationContext) (*rulesv1.ListRulesResponse, error) {
	response, err := h.client.ListRules(context.Background(), connect.NewRequest(&rulesv1.ListRulesRequest{Scope: scope(ctx)}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list source-ledger classification rules", err, nil)
	}
	return response.Msg, nil
}
func (h *handlers) listReport(ctx cliapp.OperationContext, msg *rulesv1.ListRulesResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.GetRules()))
	for _, rule := range msg.GetRules() {
		results = append(results, fmt.Sprintf("%s priority=%d facet=%s kind=%s enabled=%t", rule.GetId(), rule.GetPriority(), rule.GetFacetId(), rule.GetKind(), rule.GetEnabled()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Scope %s has %d classification rule(s).", scope(ctx), len(results))}, Results: results}
}

func (h *handlers) createCall(ctx cliapp.OperationContext) (*rulesv1.CreateRuleResponse, error) {
	rule := &rulesv1.Rule{Id: ctx.Flag("id"), Scope: scope(ctx), Priority: int32Value(ctx.Flag("priority")), FacetId: ctx.Flag("facet"), SourceRuntime: ctx.Flag("source-runtime"), Kind: ctx.Flag("kind"), KindGlob: ctx.Flag("kind-glob"), SourcePathGlob: ctx.Flag("source-path-glob"), BodyPattern: ctx.Flag("body-pattern")}
	response, err := h.client.CreateRule(context.Background(), connect.NewRequest(&rulesv1.CreateRuleRequest{Rule: rule}))
	if err != nil {
		return nil, cliapp.WrapAPIError("create source-ledger classification rule", err, nil)
	}
	return response.Msg, nil
}
func (h *handlers) createReport(ctx cliapp.OperationContext, msg *rulesv1.CreateRuleResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Created rule %s in scope %s for facet %s.", msg.GetRule().GetId(), scope(ctx), msg.GetRule().GetFacetId())}}
}

func (h *handlers) deleteCall(ctx cliapp.OperationContext) (*rulesv1.DeleteRuleResponse, error) {
	response, err := h.client.DeleteRule(context.Background(), connect.NewRequest(&rulesv1.DeleteRuleRequest{Scope: scope(ctx), RuleId: ctx.Flag("rule-id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("delete source-ledger classification rule", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) deleteReport(ctx cliapp.OperationContext, _ *rulesv1.DeleteRuleResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Deleted disabled rule %s from scope %s.", ctx.Flag("rule-id"), scope(ctx))}}
}

func (h *handlers) dryRunCall(ctx cliapp.OperationContext) (*rulesv1.DryRunRuleResponse, error) {
	response, err := h.client.DryRunRule(context.Background(), connect.NewRequest(&rulesv1.DryRunRuleRequest{Scope: scope(ctx), RuleId: ctx.Flag("rule-id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("dry-run source-ledger classification rule", err, nil)
	}
	return response.Msg, nil
}
func (h *handlers) dryRunReport(ctx cliapp.OperationContext, msg *rulesv1.DryRunRuleResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Dry-run rule %s in scope %s: %d match(es), fingerprint=%s.", msg.GetRuleId(), scope(ctx), msg.GetMatchCount(), msg.GetCorpusFingerprint())}, Results: msg.GetSamples()}
}

func (h *handlers) enableCall(ctx cliapp.OperationContext) (*rulesv1.EnableRuleResponse, error) {
	response, err := h.client.EnableRule(context.Background(), connect.NewRequest(&rulesv1.EnableRuleRequest{Scope: scope(ctx), RuleId: ctx.Flag("rule-id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("enable source-ledger classification rule", err, nil)
	}
	return response.Msg, nil
}
func (h *handlers) enableReport(ctx cliapp.OperationContext, _ *rulesv1.EnableRuleResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Enabled rule %s in scope %s.", ctx.Flag("rule-id"), scope(ctx))}}
}

func (h *handlers) revertCall(ctx cliapp.OperationContext) (*rulesv1.RevertRuleResponse, error) {
	response, err := h.client.RevertRule(context.Background(), connect.NewRequest(&rulesv1.RevertRuleRequest{Scope: scope(ctx), RuleId: ctx.Flag("rule-id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("revert source-ledger classification rule", err, nil)
	}
	return response.Msg, nil
}
func (h *handlers) revertReport(ctx cliapp.OperationContext, msg *rulesv1.RevertRuleResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Reverted rule %s in scope %s; restored=%d.", ctx.Flag("rule-id"), scope(ctx), msg.GetRestoredCount())}}
}

func (h *handlers) refacetCall(ctx cliapp.OperationContext) (*rulesv1.RefacetCorpusResponse, error) {
	response, err := h.client.RefacetCorpus(context.Background(), connect.NewRequest(&rulesv1.RefacetCorpusRequest{Scope: scope(ctx), AfterEntryId: ctx.Flag("after-entry-id"), Limit: int32Value(ctx.Flag("limit"))}))
	if err != nil {
		return nil, cliapp.WrapAPIError("refacet source-ledger corpus", err, nil)
	}
	return response.Msg, nil
}
func (h *handlers) refacetReport(ctx cliapp.OperationContext, msg *rulesv1.RefacetCorpusResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Re-faceted scope %s: total=%d assigned=%d rule-assigned=%d classified=%d failed=%d complete=%t next=%s.", scope(ctx), msg.GetTotal(), msg.GetAssigned(), msg.GetRuleAssigned(), msg.GetClassified(), msg.GetFailed(), msg.GetComplete(), msg.GetNextEntryId())}}
}

func (h *handlers) measureCall(ctx cliapp.OperationContext) (*rulesv1.MeasureDistributionResponse, error) {
	response, err := h.client.MeasureDistribution(context.Background(), connect.NewRequest(&rulesv1.MeasureDistributionRequest{Scope: scope(ctx)}))
	if err != nil {
		return nil, cliapp.WrapAPIError("measure source-ledger classification distribution", err, nil)
	}
	return response.Msg, nil
}
func (h *handlers) measureReport(ctx cliapp.OperationContext, msg *rulesv1.MeasureDistributionResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Scope %s distribution: total=%d rule-matched=%d classifier-tail=%d within-ceiling=%t.", scope(ctx), msg.GetTotal(), msg.GetRuleMatched(), msg.GetClassifierTail(), msg.GetWithinCeiling())}, Results: []string{fmt.Sprintf("max-tail-facet=%s max-tail-percent=%.2f", msg.GetMaxTailFacet(), msg.GetMaxTailPercent())}}
}

func int32Value(raw string) int32 { var value int32; _, _ = fmt.Sscan(raw, &value); return value }
