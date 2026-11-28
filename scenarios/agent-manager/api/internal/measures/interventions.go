package measures

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"agent-manager/internal/interventions"

	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"

	measurelib "github.com/vrooli/measures-go"
)

// UsageInterventions is operator interventions per agent-hour by category.
const UsageInterventions = "usage.interventions"

// InterventionSource summarizes classified operator messages over a window.
type InterventionSource interface {
	Summarize(ctx context.Context, version string, gate interventions.Gate, from, to time.Time) (interventions.Summary, error)
	Rows(ctx context.Context, version string, gate interventions.Gate, from, to time.Time, runID string, limit int) ([]interventions.Row, error)
}

// SetInterventions supplies the operator-intervention classifications.
func (h *Handler) SetInterventions(source InterventionSource) { h.interventions = source }

var interventionsDefinition = Definition{
	ID: UsageInterventions, Counts: "operator messages that followed agent work, classified by the decision model",
	Numerator: "messages per category the proven gate admits", Denominator: "agent-hours: ten-minute run slots with a provider request, divided by six",
	SourceTable: "operator_interventions, transcript_usage",
	Limitation:  "a category is shown only once its model met precision 0.8 and recall 0.7 on the labelled hold-out; unshown categories report no rate (cannot tell), and coverage is the classified share",
}

// registerInterventions declares the measure. It is registry-only: no typed
// RPC exists yet, so Service and Method are empty.
func (h *Handler) registerInterventions(registry *measurelib.Registry) error {
	return registry.Register(measurelib.MeasureDeclaration{
		Name: UsageInterventions, Scenario: "agent-manager", Domain: "usage",
		Intent:    "Operator corrections per agent-hour by category (early stop, false inability, wrong claim, quality miss, too dense, instruction ignored), with classification coverage and the agent work before each correction.",
		Questions: []string{"how often do I have to correct agents", "operator interventions per agent hour", "why do agents need correcting"},
		Params:    map[string]measurelib.Param{"window": {Name: "window", Type: measurelib.ParamTypeTimeWindow, Default: string(measurelib.TokenThisWeek)}},
		Result:    measurelib.Result{Kind: measurelib.ResultTable, ValueField: "per_agent_hour", Unit: "interventions per agent-hour", SummaryTemplate: "operator interventions ({window})"},
		Effect:    measurelib.EffectRead, RunEligible: true,
	}, func(ctx context.Context, request measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
		if h.interventions == nil {
			return measurelib.MeasureResult{}, fmt.Errorf("operator intervention classifications are unavailable")
		}
		window, err := h.resolveWindow(request, measurelib.TokenThisWeek)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		summary, err := h.interventions.Summarize(ctx, interventions.ClassifierVersion, interventions.ProvenGate, window.From, window.To)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		return measurelib.MeasureResult{Fields: interventionRows(summary), Provenance: measurelib.Provenance{ExecutedQuery: fmt.Sprintf(
			"SELECT classifications FROM operator_interventions WHERE classifier_version = %q AND occurred_at >= %q AND occurred_at < %q; agent-hours FROM transcript_usage",
			interventions.ClassifierVersion, window.From.UTC().Format(time.RFC3339Nano), window.To.UTC().Format(time.RFC3339Nano))}}, nil
	})
}

// interventionRows is one row per friction category, the pooled "operator
// friction, category unproven" bucket, then a coverage row. An unshown
// category's or bucket's rate is empty, never zero: the classifier is not proven
// for it, so silence would be a claim it cannot make.
func interventionRows(summary interventions.Summary) []map[string]string {
	count := func(v int64) string { return strconv.FormatInt(v, 10) }
	rows := make([]map[string]string, 0, len(summary.Categories)+2)
	for _, category := range summary.Categories {
		if !category.Category.Friction() {
			continue
		}
		shown := interventions.ProvenGate.Proven(category.Category)
		row := map[string]string{
			"category": string(category.Category), "owner": category.Category.Owner(), "shown": strconv.FormatBool(shown),
			"classified": count(category.Classified), "interventions": "", "per_agent_hour": "", "repeats": "", "median_work_tokens": "",
		}
		if shown {
			row["interventions"], row["repeats"], row["median_work_tokens"] = count(category.Shown), count(category.Repeats), count(category.MedianWorkTokens)
			if summary.AgentHours > 0 {
				row["per_agent_hour"] = strconv.FormatFloat(float64(category.Shown)/summary.AgentHours, 'f', 4, 64)
			}
		}
		rows = append(rows, row)
	}
	pooledShown := interventions.ProvenGate.Proven(interventions.PooledFriction)
	pooled := map[string]string{
		"category": string(interventions.PooledFriction), "owner": "", "shown": strconv.FormatBool(pooledShown),
		"classified": "", "interventions": "", "per_agent_hour": "", "repeats": "", "median_work_tokens": "",
	}
	if pooledShown {
		pooled["interventions"], pooled["repeats"] = count(summary.Pooled.Shown), count(summary.Pooled.Repeats)
		if summary.AgentHours > 0 {
			pooled["per_agent_hour"] = strconv.FormatFloat(float64(summary.Pooled.Shown)/summary.AgentHours, 'f', 4, 64)
		}
	}
	rows = append(rows, pooled)
	coverage := ""
	if value := summary.Coverage(); value != nil {
		coverage = strconv.FormatFloat(*value, 'f', 4, 64)
	}
	rows = append(rows, map[string]string{
		"category": "all", "classified": count(summary.Classified), "unclassified": count(summary.Unclassified),
		"coverage": coverage, "agent_hours": strconv.FormatFloat(summary.AgentHours, 'f', 2, 64),
	})
	return rows
}

// interventionsSection is the operator interventions of [from, to) that
// interventions.ProvenGate admits: counts per category with their owner, rate
// per agent-hour, the agent work before them, and examples. When none is
// admitted it says why and gives the classification coverage, so an empty list
// is never read as "no friction".
func interventionsSection(ctx context.Context, source InterventionSource, gate interventions.Gate, from, to time.Time, limit int) (*measurepb.UsageInterventions, error) {
	summary, err := source.Summarize(ctx, interventions.ClassifierVersion, gate, from, to)
	if err != nil {
		return nil, err
	}
	out := &measurepb.UsageInterventions{
		OperatorMessages: summary.Classified + summary.Unclassified, Classified: summary.Classified, AgentHours: summary.AgentHours,
	}
	if coverage := summary.Coverage(); coverage != nil {
		out.Coverage = *coverage
	}
	shown := int64(0)
	for _, category := range summary.Categories {
		if category.Shown == 0 {
			continue
		}
		shown += category.Shown
		row := &measurepb.UsageInterventionCategory{
			Category: string(category.Category), Owner: category.Category.Owner(), Interventions: category.Shown, Repeats: category.Repeats,
			WorkRequests: category.Work.Requests, WorkToolCalls: category.Work.ToolCalls, WorkTokens: category.Work.Tokens,
		}
		if summary.AgentHours > 0 {
			row.PerAgentHour = float64(category.Shown) / summary.AgentHours
		}
		out.Categories = append(out.Categories, row)
	}
	if pooled := summary.Pooled; pooled.Shown > 0 {
		shown += pooled.Shown
		row := &measurepb.UsageInterventionCategory{
			Category: string(interventions.PooledFriction), Interventions: pooled.Shown, Repeats: pooled.Repeats, Pooled: true,
			WorkRequests: pooled.Work.Requests, WorkToolCalls: pooled.Work.ToolCalls, WorkTokens: pooled.Work.Tokens,
		}
		if summary.AgentHours > 0 {
			row.PerAgentHour = float64(pooled.Shown) / summary.AgentHours
		}
		for _, hint := range pooled.Hints {
			row.Hints = append(row.Hints, &measurepb.UsageInterventionHint{Category: string(hint.Category), Owner: hint.Category.Owner(), Count: hint.Count})
		}
		out.Categories = append(out.Categories, row)
	}
	out.Shown = shown
	switch {
	case shown > 0:
		rows, err := source.Rows(ctx, interventions.ClassifierVersion, gate, from, to, "", limit)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			example := &measurepb.UsageInterventionExample{
				EventId: r.EventID, RunId: r.RunID, OccurredAt: r.OccurredAt, Category: string(r.Category), Owner: r.Category.Owner(), Text: r.Text, Pooled: r.Pooled,
				WorkRequests: r.Work.Requests, WorkToolCalls: r.Work.ToolCalls, WorkTokens: r.Work.Tokens, RepeatOfEventId: r.RepeatOf,
			}
			if r.Probability != nil {
				example.Probability = *r.Probability
			}
			out.Examples = append(out.Examples, example)
		}
	case summary.Classified+summary.Unclassified == 0:
		out.UnshownReason = "no operator messages were classified in this window"
	case !provenAny(gate):
		out.UnshownReason = "no category has passed the accuracy gate yet, so the classified messages are not shown"
	default:
		out.UnshownReason = "no classified message in this window falls in a category that has passed the accuracy gate"
	}
	return out, nil
}

// provenAny reports whether the gate proves any category at all.
func provenAny(gate interventions.Gate) bool {
	if gate.Proven(interventions.PooledFriction) {
		return true
	}
	for _, category := range interventions.Categories() {
		if gate.Proven(category) {
			return true
		}
	}
	return false
}
