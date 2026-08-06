package measures

import (
	"context"
	"net/http"
	"strconv"

	measures "github.com/vrooli/measures-go"
	"program-runtime/internal/clock"
)

// Handler exposes the scenario-owned measure declarations. Collectors are
// supplied by the owning services so measure reads cannot drift from the
// source-of-truth state. The zero-value fallback keeps isolated handler tests
// deterministic when no service wiring is needed.
func Handler(clk clock.Clock, collectors ...func() int) (http.Handler, error) {
	reg := measures.NewRegistry(measures.WithClock(clk.Now))
	for i, declaration := range declarations() {
		decl := declaration
		if err := reg.Register(decl, func(context.Context, measures.MeasureRequest) (measures.MeasureResult, error) {
			value := 0
			if i < len(collectors) && collectors[i] != nil {
				value = collectors[i]()
			}
			return measures.MeasureResult{Value: strconv.Itoa(value), Provenance: measures.Provenance{ExecutedQuery: "live in-memory program-runtime owner"}}, nil
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
	}
}
