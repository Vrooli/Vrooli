package routing

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	aisearch "github.com/vrooli/ai-go/search"
)

// StageKind is deliberately closed. A strategy is a small, reviewable
// pipeline description, not a plug-in runtime. Adding a stage requires code,
// tests, and a taxonomy decision.
type StageKind string

const (
	StageLexical      StageKind = "lexical"
	StageEmbedding    StageKind = "embedding"
	StageCrossEncoder StageKind = "cross_encoder"
	StageLLM          StageKind = "llm"
)

var stageKinds = map[StageKind]struct{}{
	StageLexical: {}, StageEmbedding: {}, StageCrossEncoder: {}, StageLLM: {},
}

// RetrievalStage is one ordered rung in a retrieval strategy. Parameters are
// intentionally untyped JSON values at this boundary: each closed stage owns
// interpretation of its small parameter vocabulary when it becomes executable
// in a later strategy phase.
type RetrievalStage struct {
	Kind   StageKind              `json:"kind"`
	Params map[string]interface{} `json:"params,omitempty"`
}

// RetrievalStrategy names one comparable pipeline. The phase-10 active
// strategy records the current behavior; later phases may make the same
// records executable arms without changing the data contract.
type RetrievalStrategy struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Stages      []RetrievalStage `json:"stages"`
}

// RouterFactorValues is the typed runtime projection of the router factor
// rows. It is loaded once at startup and passed through Deps, so the hot query
// path never consults environment variables or scattered constants.
type RouterFactorValues struct {
	MaxFanoutWidth         int
	WidenThreshold         float64
	PerProviderTimeout     time.Duration
	Concurrency            int
	QueryBudget            time.Duration
	ZeroYieldMinimumRoutes int64
	DemotionWindow         time.Duration
}

func defaultRouterFactorValues() RouterFactorValues {
	return RouterFactorValues{
		MaxFanoutWidth:         aisearch.DefaultRouterMaxFanoutWidth,
		WidenThreshold:         aisearch.DefaultRouterWidenThreshold,
		PerProviderTimeout:     aisearch.DefaultRouterPerProviderTimeout,
		Concurrency:            aisearch.DefaultRouterConcurrency,
		QueryBudget:            aisearch.DefaultRouterQueryBudget,
		ZeroYieldMinimumRoutes: aisearch.DefaultRouterZeroYieldMinimumRoutes,
		DemotionWindow:         aisearch.DefaultRouterDemotionWindow,
	}
}

// strategyDocument is the on-disk data contract. Router factors are shared by
// all named strategies in this phase so a benchmark compares pipeline shape
// without silently changing the runtime budget between arms.
type strategyDocument struct {
	ActiveStrategy string              `json:"active_strategy"`
	RouterFactors  routerFactorsData   `json:"router_factors"`
	Strategies     []RetrievalStrategy `json:"strategies"`
}

type routerFactorsData struct {
	MaxFanoutWidth         int     `json:"max_fanout_width"`
	WidenThreshold         float64 `json:"widen_threshold"`
	PerProviderTimeout     string  `json:"per_provider_timeout"`
	Concurrency            int     `json:"concurrency"`
	QueryBudget            string  `json:"query_budget"`
	ZeroYieldMinimumRoutes int64   `json:"zero_yield_minimum_routes"`
	DemotionWindow         string  `json:"demotion_window"`
}

func (d routerFactorsData) values() (RouterFactorValues, error) {
	defaults := defaultRouterFactorValues()
	values := defaults
	values.MaxFanoutWidth = d.MaxFanoutWidth
	values.WidenThreshold = d.WidenThreshold
	values.Concurrency = d.Concurrency
	values.ZeroYieldMinimumRoutes = d.ZeroYieldMinimumRoutes
	var err error
	if values.PerProviderTimeout, err = parseFactorDuration("per_provider_timeout", d.PerProviderTimeout); err != nil {
		return RouterFactorValues{}, err
	}
	if values.QueryBudget, err = parseFactorDuration("query_budget", d.QueryBudget); err != nil {
		return RouterFactorValues{}, err
	}
	if values.DemotionWindow, err = parseFactorDuration("demotion_window", d.DemotionWindow); err != nil {
		return RouterFactorValues{}, err
	}
	if values.MaxFanoutWidth < 1 || values.MaxFanoutWidth > 128 {
		return RouterFactorValues{}, fmt.Errorf("router factor max_fanout_width=%d is outside [1,128]", values.MaxFanoutWidth)
	}
	if values.WidenThreshold < 0 || values.WidenThreshold > 1 {
		return RouterFactorValues{}, fmt.Errorf("router factor widen_threshold=%g is outside [0,1]", values.WidenThreshold)
	}
	if values.Concurrency < 1 || values.Concurrency > 128 {
		return RouterFactorValues{}, fmt.Errorf("router factor concurrency=%d is outside [1,128]", values.Concurrency)
	}
	if values.ZeroYieldMinimumRoutes < 1 || values.ZeroYieldMinimumRoutes > 1000 {
		return RouterFactorValues{}, fmt.Errorf("router factor zero_yield_minimum_routes=%d is outside [1,1000]", values.ZeroYieldMinimumRoutes)
	}
	return values, nil
}

func parseFactorDuration(key, raw string) (time.Duration, error) {
	value, err := time.ParseDuration(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("router factor %s must be a positive duration, got %q", key, raw)
	}
	return value, nil
}

// Validate enforces the closed strategy vocabulary before a strategy can be
// selected. Unknown stage kinds therefore fail startup, never a query.
func (s RetrievalStrategy) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("retrieval strategy name is required")
	}
	if len(s.Stages) == 0 {
		return fmt.Errorf("retrieval strategy %q must contain at least one stage", s.Name)
	}
	for i, stage := range s.Stages {
		if _, ok := stageKinds[stage.Kind]; !ok {
			return fmt.Errorf("retrieval strategy %q stage %d has unknown kind %q (expected lexical, embedding, cross_encoder, or llm)", s.Name, i, stage.Kind)
		}
	}
	return nil
}

// ValidateStrategies parses and validates the full strategy document. It is
// exported for focused tests and for future strategy inspection commands.
func ValidateStrategies(raw []byte) (RetrievalStrategy, RouterFactorValues, []RetrievalStrategy, error) {
	var doc strategyDocument
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&doc); err != nil {
		return RetrievalStrategy{}, RouterFactorValues{}, nil, fmt.Errorf("decode retrieval strategies: %w", err)
	}
	factors, err := doc.RouterFactors.values()
	if err != nil {
		return RetrievalStrategy{}, RouterFactorValues{}, nil, err
	}
	byName := make(map[string]RetrievalStrategy, len(doc.Strategies))
	for _, strategy := range doc.Strategies {
		if err := strategy.Validate(); err != nil {
			return RetrievalStrategy{}, RouterFactorValues{}, nil, err
		}
		if _, exists := byName[strategy.Name]; exists {
			return RetrievalStrategy{}, RouterFactorValues{}, nil, fmt.Errorf("duplicate retrieval strategy %q", strategy.Name)
		}
		byName[strategy.Name] = strategy
	}
	activeName := strings.TrimSpace(doc.ActiveStrategy)
	active, ok := byName[activeName]
	if !ok {
		return RetrievalStrategy{}, RouterFactorValues{}, nil, fmt.Errorf("active retrieval strategy %q is not defined", activeName)
	}
	strategies := make([]RetrievalStrategy, 0, len(byName))
	for _, strategy := range byName {
		strategies = append(strategies, strategy)
	}
	sort.Slice(strategies, func(i, j int) bool { return strategies[i].Name < strategies[j].Name })
	return active, factors, strategies, nil
}

//go:embed strategies.json
var embeddedStrategies embed.FS

// LoadActiveStrategy loads the scenario-owned strategy record. An explicit
// path is supported for controlled experiments; normal operation uses the
// embedded, reviewable data file and has no working-directory dependency.
func LoadActiveStrategy() (RetrievalStrategy, RouterFactorValues, error) {
	raw, err := readStrategyData()
	if err != nil {
		return RetrievalStrategy{}, RouterFactorValues{}, fmt.Errorf("read retrieval strategy data: %w", err)
	}
	active, factors, _, err := ValidateStrategies(raw)
	if err != nil {
		return RetrievalStrategy{}, RouterFactorValues{}, err
	}
	return active, factors, nil
}

// LoadStrategyCatalog returns every validated strategy row from the same data
// source as LoadActiveStrategy. Keeping this read beside the active-row loader
// prevents compare from silently evaluating a different document than runtime.
func LoadStrategyCatalog() ([]RetrievalStrategy, error) {
	raw, err := readStrategyData()
	if err != nil {
		return nil, fmt.Errorf("read retrieval strategy data: %w", err)
	}
	return StrategiesFromData(raw)
}

func readStrategyData() ([]byte, error) {
	if path := strings.TrimSpace(os.Getenv("SEARCH_HUB_RETRIEVAL_STRATEGY_PATH")); path != "" {
		return os.ReadFile(path)
	}
	return embeddedStrategies.ReadFile("strategies.json")
}

// StrategiesFromData returns all validated rows for inspection and benchmark
// tooling without exposing the mutable document representation.
func StrategiesFromData(raw []byte) ([]RetrievalStrategy, error) {
	_, _, strategies, err := ValidateStrategies(raw)
	return strategies, err
}
