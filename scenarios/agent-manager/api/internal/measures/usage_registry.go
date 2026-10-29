package measures

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	measurelib "github.com/vrooli/measures-go"
	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// usageMeasure serves one usage procedure through the measures-go registry
// (F-W4-4). The registry executes the Connect procedure itself, so the
// registry and the typed RPC share one compute path, and flattens the rows
// named by rows (a path of repeated message fields) into table fields.
type usageMeasure struct {
	name, method, intent string
	questions            []string
	params               []string
	rows                 string
	valueField, unit     string
	call                 func(context.Context, map[string]string) (proto.Message, error)
}

func (h *Handler) usageMeasures() []usageMeasure {
	return []usageMeasure{
		{
			UsageForecast, "UsageForecast", "Current provider quota windows with burn rates and projected exhaustion.",
			[]string{"will I run out of quota this week", "agent usage forecast"},
			[]string{"provider"},
			"windows", "pool", "percent",
			func(ctx context.Context, p map[string]string) (proto.Message, error) {
				return msg(h.UsageForecast(ctx, connect.NewRequest(&measurepb.UsageForecastRequest{Provider: p["provider"]})))
			},
		},
		{
			UsageForecastBacktest, "UsageForecastBacktest", "Forecast error against the linear baseline on completed quota windows.",
			[]string{"how accurate is the usage forecast"},
			[]string{"provider"},
			"summaries", "population", "points",
			func(ctx context.Context, p map[string]string) (proto.Message, error) {
				return msg(h.UsageForecastBacktest(ctx, connect.NewRequest(&measurepb.UsageForecastBacktestRequest{Provider: p["provider"]})))
			},
		},
		{
			UsageBurners, "UsageBurners", "Each consumer's share of the current quota windows' observed increase.",
			[]string{"what is burning my quota", "agent usage burners by session"},
			[]string{"provider", "group_by", "since", "run_id"},
			"windows.burners", "label", "points",
			func(ctx context.Context, p map[string]string) (proto.Message, error) {
				return msg(h.UsageBurners(ctx, connect.NewRequest(&measurepb.UsageBurnersRequest{Provider: p["provider"], GroupBy: p["group_by"], Since: p["since"], RunId: p["run_id"]})))
			},
		},
		{
			UsageRecommendations, "UsageRecommendations", "Changes grouped by target with a combined quota saving, confidence and evidence.",
			[]string{"how do I use less agent quota", "agent usage recommendations"},
			[]string{"provider", "run_id", "pool", "window"},
			"groups", "target_label", "combined_saved_points",
			func(ctx context.Context, p map[string]string) (proto.Message, error) {
				return msg(h.UsageRecommendations(ctx, connect.NewRequest(&measurepb.UsageRecommendationsRequest{Provider: p["provider"], RunId: p["run_id"], Pool: p["pool"], Window: p["window"]})))
			},
		},
		{
			UsageFriction, "UsageFriction", "Recurring friction problems in plain words with owner, count, cost and status.",
			[]string{"what keeps going wrong for agents", "recurring agent friction"},
			[]string{"provider", "status"},
			"problems", "description", "points",
			func(ctx context.Context, p map[string]string) (proto.Message, error) {
				return msg(h.UsageFriction(ctx, connect.NewRequest(&measurepb.UsageFrictionRequest{Provider: p["provider"], Status: p["status"]})))
			},
		},
		{
			UsageCoverage, "UsageCoverage", "Captured share of on-disk agent transcripts by root.",
			[]string{"is all agent usage captured", "transcript capture coverage"},
			[]string{"since"},
			"roots", "kind", "bytes",
			func(ctx context.Context, p map[string]string) (proto.Message, error) {
				return msg(h.UsageCoverage(ctx, connect.NewRequest(&measurepb.UsageCoverageRequest{Since: p["since"]})))
			},
		},
		{
			UsageCompare, "UsageCompare", "Quota points per successful run of two cohorts.",
			[]string{"did this change reduce quota per success"},
			[]string{"provider", "left_cohort", "right_cohort", "left_run_ids", "right_run_ids", "change_binding"},
			"sides", "side", "points",
			func(ctx context.Context, p map[string]string) (proto.Message, error) {
				selector := func(cohort, runIDs string) *measurepb.CohortSelector {
					return &measurepb.CohortSelector{Cohort: cohort, RunIds: splitList(runIDs)}
				}
				return msg(h.UsageCompare(ctx, connect.NewRequest(&measurepb.UsageCompareRequest{
					Provider: p["provider"], ChangeBinding: p["change_binding"],
					Left: selector(p["left_cohort"], p["left_run_ids"]), Right: selector(p["right_cohort"], p["right_run_ids"]),
				})))
			},
		},
	}
}

func msg[T any, M interface {
	*T
	proto.Message
}](response *connect.Response[T], err error) (proto.Message, error) {
	if err != nil {
		return nil, err
	}
	return M(response.Msg), nil
}

func splitList(value string) []string {
	var out []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

func (h *Handler) registerUsageMeasures(registry *measurelib.Registry) error {
	for _, m := range h.usageMeasures() {
		m := m
		params := map[string]measurelib.Param{}
		for _, name := range m.params {
			params[name] = measurelib.Param{Name: name, Type: "string"}
		}
		declaration := measurelib.MeasureDeclaration{
			Name: m.name, Scenario: "agent-manager", Domain: "usage", Intent: m.intent, Questions: m.questions, Params: params,
			Result: measurelib.Result{Kind: measurelib.ResultTable, ValueField: m.valueField, Unit: m.unit, SummaryTemplate: strings.ReplaceAll(m.name, "_", " ")},
			Effect: measurelib.EffectRead, RunEligible: true, Service: "MeasuresService", Method: m.method,
		}
		if err := registry.Register(declaration, func(ctx context.Context, request measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
			response, err := m.call(ctx, request.Params)
			if err != nil {
				return measurelib.MeasureResult{}, err
			}
			result := measurelib.MeasureResult{Fields: tableRows(response, m.rows)}
			if withProvenance, ok := response.(interface {
				GetProvenance() *measurepb.MeasureProvenance
			}); ok {
				result.Provenance = measurelib.Provenance{ExecutedQuery: withProvenance.GetProvenance().GetExecutedQuery()}
			}
			return result, nil
		}); err != nil {
			return fmt.Errorf("register %s: %w", m.name, err)
		}
	}
	return nil
}

// tableRows flattens a response into rows. path names nested repeated
// message fields ("windows.burners": one row per burner, carrying its
// window's fields prefixed "windows."); "sides" is the left/right pair of a
// comparison.
func tableRows(response proto.Message, path string) []map[string]string {
	message := response.ProtoReflect()
	if path == "sides" {
		var rows []map[string]string
		for _, side := range []string{"left", "right"} {
			field := message.Descriptor().Fields().ByName(protoreflect.Name(side))
			row := protoFields(message.Get(field).Message(), "")
			row["side"] = side
			rows = append(rows, row)
		}
		return rows
	}
	return nestedRows(message, strings.Split(path, "."), map[string]string{}, "")
}

func nestedRows(message protoreflect.Message, path []string, inherited map[string]string, prefix string) []map[string]string {
	field := message.Descriptor().Fields().ByName(protoreflect.Name(path[0]))
	if field == nil || !field.IsList() || field.Message() == nil {
		return nil
	}
	var rows []map[string]string
	list := message.Get(field).List()
	for i := 0; i < list.Len(); i++ {
		element := list.Get(i).Message()
		if len(path) > 1 {
			parent := map[string]string{}
			for key, value := range inherited {
				parent[key] = value
			}
			for key, value := range protoFields(element, prefix+path[0]+".") {
				parent[key] = value
			}
			rows = append(rows, nestedRows(element, path[1:], parent, prefix+path[0]+".")...)
			continue
		}
		row := protoFields(element, "")
		for key, value := range inherited {
			row[key] = value
		}
		rows = append(rows, row)
	}
	return rows
}

// protoFields renders a message's scalar fields (repeated scalars joined by
// commas, singular messages flattened with a "name." prefix); repeated
// messages are left out.
func protoFields(message protoreflect.Message, prefix string) map[string]string {
	out := map[string]string{}
	message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		name := prefix + string(field.Name())
		switch {
		case field.IsList() && field.Message() != nil:
		case field.IsList():
			items := make([]string, value.List().Len())
			for i := range items {
				items[i] = scalar(field, value.List().Get(i))
			}
			out[name] = strings.Join(items, ",")
		case field.IsMap():
		case field.Message() != nil:
			for key, nested := range protoFields(value.Message(), name+".") {
				out[key] = nested
			}
		default:
			out[name] = scalar(field, value)
		}
		return true
	})
	return out
}

func scalar(field protoreflect.FieldDescriptor, value protoreflect.Value) string {
	switch field.Kind() {
	case protoreflect.DoubleKind, protoreflect.FloatKind:
		return strconv.FormatFloat(value.Float(), 'f', -1, 64)
	case protoreflect.EnumKind:
		if enum := field.Enum().Values().ByNumber(value.Enum()); enum != nil {
			return string(enum.Name())
		}
	}
	return value.String()
}
