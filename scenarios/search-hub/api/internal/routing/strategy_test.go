package routing

import (
	"os"
	"testing"
	"time"
)

func TestValidateStrategiesRejectsUnknownStageKindAtLoadTime(t *testing.T) {
	raw := []byte(`{
        "active_strategy":"bad",
        "router_factors":{"max_fanout_width":6,"widen_threshold":0.45,"per_provider_timeout":"4s","concurrency":8,"query_budget":"25s","zero_yield_minimum_routes":5,"demotion_window":"15m"},
        "strategies":[{"name":"bad","stages":[{"kind":"neural_magic"}]}]
    }`)
	if _, _, _, err := ValidateStrategies(raw); err == nil {
		t.Fatal("unknown stage kind was accepted")
	}
}

func TestLoadActiveStrategyRejectsUnknownStageKindBeforeServing(t *testing.T) {
	raw := []byte(`{"active_strategy":"bad","router_factors":{"max_fanout_width":6,"widen_threshold":0.45,"per_provider_timeout":"4s","concurrency":8,"query_budget":"25s","zero_yield_minimum_routes":5,"demotion_window":"15m"},"strategies":[{"name":"bad","stages":[{"kind":"not-a-stage"}]}]}`)
	path := t.TempDir() + "/strategies.json"
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write strategy fixture: %v", err)
	}
	t.Setenv("SEARCH_HUB_RETRIEVAL_STRATEGY_PATH", path)
	if _, _, err := LoadActiveStrategy(); err == nil {
		t.Fatal("LoadActiveStrategy accepted an unknown stage kind")
	}
}

func TestLoadActiveStrategyUsesEmbeddedData(t *testing.T) {
	strategy, factors, err := LoadActiveStrategy()
	if err != nil {
		t.Fatalf("LoadActiveStrategy() error = %v", err)
	}
	if strategy.Name != "lexical-cross-encoder" {
		t.Fatalf("active strategy = %q, want lexical-cross-encoder", strategy.Name)
	}
	if len(strategy.Stages) != 2 || strategy.Stages[0].Kind != StageLexical || strategy.Stages[1].Kind != StageCrossEncoder {
		t.Fatalf("active stages = %+v, want lexical then cross_encoder", strategy.Stages)
	}
	if factors.MaxFanoutWidth != 6 || factors.Concurrency != 8 {
		t.Fatalf("router factors = %+v", factors)
	}
	if factors.PerProviderTimeout != 4*time.Second || factors.QueryBudget != 25*time.Second || factors.DemotionWindow != 15*time.Minute {
		t.Fatalf("duration factors = %+v", factors)
	}
}

func TestValidateStrategiesRejectsDuplicateAndMissingActiveRows(t *testing.T) {
	base := `"router_factors":{"max_fanout_width":6,"widen_threshold":0.45,"per_provider_timeout":"4s","concurrency":8,"query_budget":"25s","zero_yield_minimum_routes":5,"demotion_window":"15m"}`
	for name, raw := range map[string]string{
		"duplicate":      `{"active_strategy":"one",` + base + `,"strategies":[{"name":"one","stages":[{"kind":"lexical"}]},{"name":"one","stages":[{"kind":"llm"}]}]}`,
		"missing active": `{"active_strategy":"missing",` + base + `,"strategies":[{"name":"one","stages":[{"kind":"lexical"}]}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, _, err := ValidateStrategies([]byte(raw)); err == nil {
				t.Fatal("invalid strategy document was accepted")
			}
		})
	}
}

func TestRouterUsesInjectedStrategyFactors(t *testing.T) {
	strategy := RetrievalStrategy{Name: "test", Stages: []RetrievalStage{{Kind: StageLexical}}}
	factors := RouterFactorValues{
		MaxFanoutWidth: 2, WidenThreshold: 0.75, PerProviderTimeout: time.Second,
		Concurrency: 2, QueryBudget: 3 * time.Second, ZeroYieldMinimumRoutes: 2,
		DemotionWindow: 2 * time.Minute,
	}
	router := NewRouter(Deps{Strategy: &strategy, RouterFactors: &factors})
	if router.RetrievalStrategy().Name != "test" {
		t.Fatalf("strategy = %+v", router.RetrievalStrategy())
	}
	if got := router.RouterFactors(); got != factors {
		t.Fatalf("router factors = %+v, want %+v", got, factors)
	}
}
