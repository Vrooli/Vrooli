package rules

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	rulesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/rules"
	rulesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/rules/rules_v1connect"
)

type handlers struct {
	client rulesconnect.ClassificationRulesServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	http, base := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: rulesconnect.NewClassificationRulesServiceClient(http, base)}
}

func (h *handlers) listCall(ctx cliapp.OperationContext) (*rulesv1.ListRulesResponse, error) {
	r, err := h.client.ListRules(context.Background(), connect.NewRequest(&rulesv1.ListRulesRequest{Scope: ctx.Flag("scope")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list classification rules", err, nil)
	}
	return r.Msg, nil
}
func (h *handlers) listReport(_ cliapp.OperationContext, m *rulesv1.ListRulesResponse) cliapp.ListReport {
	results := make([]string, 0, len(m.GetRules()))
	for _, rule := range m.GetRules() {
		results = append(results, fmt.Sprintf("%s priority=%d facet=%s enabled=%t", rule.GetId(), rule.GetPriority(), rule.GetFacetId(), rule.GetEnabled()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d classification rule(s).", len(results))}, Results: results}
}
func (h *handlers) createCall(ctx cliapp.OperationContext) (*rulesv1.CreateRuleResponse, error) {
	priority := int64(100)
	var err error
	if ctx.Flag("priority") != "" {
		priority, err = strconv.ParseInt(ctx.Flag("priority"), 10, 32)
	}
	if err != nil {
		return nil, fmt.Errorf("priority must be an integer")
	}
	rule := &rulesv1.Rule{Id: ctx.Flag("id"), Scope: ctx.Flag("scope"), Priority: int32(priority), FacetId: ctx.Flag("facet"), SourceRuntime: ctx.Flag("source-runtime"), Kind: ctx.Flag("kind"), SourcePathGlob: ctx.Flag("source-path"), BodyPattern: ctx.Flag("body-pattern")}
	r, err := h.client.CreateRule(context.Background(), connect.NewRequest(&rulesv1.CreateRuleRequest{Rule: rule}))
	if err != nil {
		return nil, cliapp.WrapAPIError("create classification rule", err, nil)
	}
	return r.Msg, nil
}
func (h *handlers) createReport(_ cliapp.OperationContext, m *rulesv1.CreateRuleResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Created rule %s.", m.GetRule().GetId())}}
}
func (h *handlers) dryRunCall(ctx cliapp.OperationContext) (*rulesv1.DryRunRuleResponse, error) {
	r, err := h.client.DryRunRule(context.Background(), connect.NewRequest(&rulesv1.DryRunRuleRequest{Scope: ctx.Flag("scope"), RuleId: ctx.Positional("rule-id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("dry-run classification rule", err, nil)
	}
	return r.Msg, nil
}
func (h *handlers) dryRunReport(_ cliapp.OperationContext, m *rulesv1.DryRunRuleResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Rule %s matched %d entr(y/ies).", m.GetRuleId(), m.GetMatchCount())}, Results: m.GetSamples()}
}
func (h *handlers) enableCall(ctx cliapp.OperationContext) (*rulesv1.EnableRuleResponse, error) {
	r, err := h.client.EnableRule(context.Background(), connect.NewRequest(&rulesv1.EnableRuleRequest{Scope: ctx.Flag("scope"), RuleId: ctx.Positional("rule-id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("enable classification rule", err, nil)
	}
	return r.Msg, nil
}
func (h *handlers) enableReport(_ cliapp.OperationContext, _ *rulesv1.EnableRuleResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Classification rule enabled."}}
}
func (h *handlers) revertCall(ctx cliapp.OperationContext) (*rulesv1.RevertRuleResponse, error) {
	r, err := h.client.RevertRule(context.Background(), connect.NewRequest(&rulesv1.RevertRuleRequest{Scope: ctx.Flag("scope"), RuleId: ctx.Positional("rule-id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("revert classification rule", err, nil)
	}
	return r.Msg, nil
}
func (h *handlers) revertReport(_ cliapp.OperationContext, m *rulesv1.RevertRuleResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Appended %d prior facet assignment(s).", m.GetRestoredCount())}}
}
func (h *handlers) refacetCall(ctx cliapp.OperationContext) (*rulesv1.RefacetCorpusResponse, error) {
	batchSize := 200
	if raw := ctx.Flag("batch-size"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 || parsed > 500 {
			return nil, fmt.Errorf("batch-size must be an integer from 1 to 500")
		}
		batchSize = parsed
	}
	total := &rulesv1.RefacetCorpusResponse{}
	after := ""
	for {
		r, err := h.client.RefacetCorpus(context.Background(), connect.NewRequest(&rulesv1.RefacetCorpusRequest{Scope: ctx.Flag("scope"), AfterEntryId: after, Limit: int32(batchSize)}))
		if err != nil {
			return nil, cliapp.WrapAPIError("re-facet corpus", err, nil)
		}
		batch := r.Msg
		total.Total += batch.Total
		total.Assigned += batch.Assigned
		total.RuleAssigned += batch.RuleAssigned
		total.Classified += batch.Classified
		total.Failed += batch.Failed
		if batch.Complete {
			return total, nil
		}
		if batch.NextEntryId == "" {
			return nil, fmt.Errorf("re-facet corpus returned an incomplete batch without a continuation cursor")
		}
		after = batch.NextEntryId
	}
}
func (h *handlers) refacetReport(_ cliapp.OperationContext, m *rulesv1.RefacetCorpusResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Re-faceted %d/%d entries (%d rule, %d classifier, %d failed).", m.GetAssigned(), m.GetTotal(), m.GetRuleAssigned(), m.GetClassified(), m.GetFailed())}}
}
func (h *handlers) measureDistributionCall(ctx cliapp.OperationContext) (*rulesv1.MeasureDistributionResponse, error) {
	r, err := h.client.MeasureDistribution(context.Background(), connect.NewRequest(&rulesv1.MeasureDistributionRequest{Scope: ctx.Flag("scope")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("measure classification distribution", err, nil)
	}
	return r.Msg, nil
}
func (h *handlers) measureDistributionReport(_ cliapp.OperationContext, m *rulesv1.MeasureDistributionResponse) cliapp.ListReport {
	results := make([]string, 0, len(m.GetRuleCoverage())+len(m.GetClassifierTailByFacet()))
	ruleIDs := make([]string, 0, len(m.GetRuleCoverage()))
	for id := range m.GetRuleCoverage() {
		ruleIDs = append(ruleIDs, id)
	}
	sort.Strings(ruleIDs)
	for _, id := range ruleIDs {
		results = append(results, fmt.Sprintf("rule %s: %d match(es)", id, m.GetRuleCoverage()[id]))
	}
	facets := make([]string, 0, len(m.GetClassifierTailByFacet()))
	for id := range m.GetClassifierTailByFacet() {
		facets = append(facets, id)
	}
	sort.Strings(facets)
	for _, id := range facets {
		results = append(results, fmt.Sprintf("classifier-tail %s: %d", id, m.GetClassifierTailByFacet()[id]))
	}
	status := "PASS"
	if !m.GetWithinCeiling() {
		status = "FAIL"
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%s: %d total; %d deterministic-rule match(es); %d classifier-tail entr(y/ies); max tail %s %.2f%% (ceiling %.2f%%).", status, m.GetTotal(), m.GetRuleMatched(), m.GetClassifierTail(), m.GetMaxTailFacet(), m.GetMaxTailPercent()*100, m.GetCeilingPercent()*100)}, Results: results}
}
