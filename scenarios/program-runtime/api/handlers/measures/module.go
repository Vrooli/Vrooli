package measures

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/vrooli/api-core/schedule"
	measures "github.com/vrooli/measures-go"
)

// Handler exposes the scenario-owned measure declarations. Collectors are
// supplied by the owning services so measure reads cannot drift from the
// source-of-truth state. The zero-value fallback keeps isolated handler tests
// deterministic when no service wiring is needed.
func Handler(clk schedule.Clock, collectors ...func() int) (http.Handler, error) {
	if len(collectors) != len(declarations()) {
		return nil, fmt.Errorf("measures: expected %d collectors, got %d", len(declarations()), len(collectors))
	}
	byName := make(map[string]func() int, len(collectors))
	for i, declaration := range declarations() {
		byName[declaration.Name] = collectors[i]
	}
	return HandlerWithCollectors(clk, byName)
}

// HandlerWithCollectors registers collectors by declaration name. A missing
// collector is an error instead of silently becoming a fabricated zero.
func HandlerWithCollectors(clk schedule.Clock, collectors map[string]func() int) (http.Handler, error) {
	reg := measures.NewRegistry(measures.WithClock(clk.Now))
	for _, declaration := range declarations() {
		decl := declaration
		collector, ok := collectors[decl.Name]
		if !ok || collector == nil {
			return nil, fmt.Errorf("measures: no collector registered for %s", decl.Name)
		}
		if err := reg.Register(decl, func(context.Context, measures.MeasureRequest) (measures.MeasureResult, error) {
			return measures.MeasureResult{Value: strconv.Itoa(collector()), Provenance: measures.Provenance{ExecutedQuery: "live in-memory program-runtime owner"}}, nil
		}); err != nil {
			return nil, err
		}
	}
	return reg.Handler(), nil
}

func declarations() []measures.MeasureDeclaration {
	return []measures.MeasureDeclaration{
		{Name: "sessions.list", Domain: "sessions", Intent: "How many program sessions are live.", Questions: []string{"how many program sessions are live"}, Result: measures.Result{Kind: measures.ResultScalar, ValueField: "count", Unit: "sessions", SummaryTemplate: "{count} live sessions"}, Effect: measures.EffectRead, RunEligible: true, Service: "SessionService", Method: "ListSessions"},
		{Name: "programs.mine", Domain: "programs", Intent: "How many recurring program failure shapes are recorded.", Questions: []string{"how many recurring program failure shapes"}, Result: measures.Result{Kind: measures.ResultScalar, ValueField: "count", Unit: "failure_shapes", SummaryTemplate: "{count} recurring failure shapes"}, Effect: measures.EffectRead, RunEligible: true, Service: "ProgramService", Method: "MineFailures"},
		{Name: "session-delegations.list", Domain: "session_delegations", Intent: "How many delegated session executions are retained.", Questions: []string{"how many delegated session executions are retained", "count session delegations"}, Result: measures.Result{Kind: measures.ResultScalar, ValueField: "count", Unit: "delegations", SummaryTemplate: "{count} retained session delegations"}, Effect: measures.EffectRead, RunEligible: true, Service: "SessionService", Method: "ListDelegations"},
		{Name: "bindings.invocations", Domain: "bindings", Intent: "How many verified receipt-backed exercise invocations were observed in the condition window.", Questions: []string{"how many binding exercise invocations were observed", "how many receipt-backed binding invocations"}, Result: measures.Result{Kind: measures.ResultScalar, ValueField: "count", Unit: "invocations", SummaryTemplate: "{count} receipt-backed binding invocations"}, Effect: measures.EffectRead, RunEligible: true, Service: "BindingConditionService", Method: "GetBindingCondition"},
		{Name: "bindings.exercise-unattributed", Domain: "bindings", Intent: "How many exercise receipts lacked verified caller attribution in the condition window.", Questions: []string{"how many binding exercise receipts are unattributed", "how many receipt invocations lack verified callers"}, Result: measures.Result{Kind: measures.ResultScalar, ValueField: "count", Unit: "receipts", SummaryTemplate: "{count} unattributed exercise receipts"}, Effect: measures.EffectRead, RunEligible: true, Service: "BindingConditionService", Method: "GetBindingCondition"},
		{Name: "bindings.failure-rate", Domain: "bindings", Intent: "What percentage of binding invocations failed in the condition window.", Questions: []string{"what is the binding failure rate"}, Result: measures.Result{Kind: measures.ResultScalar, ValueField: "percent", Unit: "percent", SummaryTemplate: "{percent}% binding failure rate"}, Effect: measures.EffectRead, RunEligible: true, Service: "BindingConditionService", Method: "GetBindingCondition"},
		{Name: "bindings.dormant", Domain: "bindings", Intent: "How many governed bindings were not invoked in the condition window.", Questions: []string{"how many bindings are dormant"}, Result: measures.Result{Kind: measures.ResultScalar, ValueField: "count", Unit: "bindings", SummaryTemplate: "{count} dormant bindings"}, Effect: measures.EffectRead, RunEligible: true, Service: "BindingConditionService", Method: "GetBindingCondition"},
		{Name: "bindings.degraded-sustained", Domain: "bindings", Intent: "How many bindings have remained degraded across the sustained condition window.", Questions: []string{"how many bindings are degraded sustained"}, Result: measures.Result{Kind: measures.ResultScalar, ValueField: "count", Unit: "bindings", SummaryTemplate: "{count} sustained degraded bindings"}, Effect: measures.EffectRead, RunEligible: true, Service: "BindingConditionService", Method: "GetBindingCondition"},
		{Name: "discovery.library-usage", Domain: "discovery", Intent: "How many typed discovery results selected a current library program.", Questions: []string{"how many library programs did discovery use"}, Result: measures.Result{Kind: measures.ResultScalar, ValueField: "count", Unit: "library_discoveries", SummaryTemplate: "{count} library-backed discoveries"}, Effect: measures.EffectRead, RunEligible: true, Service: "BindingRegistryService", Method: "ResolveIntent"},
		{Name: "discovery.null-verdict-rate", Domain: "discovery", Intent: "What percentage of typed discovery calls returned a null verdict.", Questions: []string{"what is the discovery null verdict rate"}, Result: measures.Result{Kind: measures.ResultScalar, ValueField: "percent", Unit: "percent", SummaryTemplate: "{percent}% discovery null-verdict rate"}, Effect: measures.EffectRead, RunEligible: true, Service: "BindingRegistryService", Method: "ResolveIntent"},
	}
}
