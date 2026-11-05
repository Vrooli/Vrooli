// Package importance provides the optional network enrichment for scoring
// results. It is deliberately outside the core scoring math: failures and
// timeouts omit the enrichment instead of failing GetScore.
package importance

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

const (
	defaultTimeout             = time.Second
	defaultCentralityWeight    = 0.45
	defaultCoreProximityWeight = 0.30
	defaultRecencyWeight       = 0.25
	defaultSystemRequiredFloor = 0.90
	defaultNeutralScore        = 0.50
)

// Summary is the optional score enrichment shown to operators.
type Summary struct {
	Score          float64
	SystemRequired bool
	Components     Components
	Signals        Signals
	Degraded       []string
}

type Components struct {
	Centrality    float64
	CoreProximity float64
	Recency       float64
}

type Signals struct {
	DirectReverseDependencyCount     int
	TransitiveReverseDependencyCount int
	RequiredReverseDependencyCount   int
	RequiredEdgeWeightedScore        float64
	DistanceToCoreSeed               int
	NearestCoreSeed                  string
	RecentActivityCount              int
}

// HTTPDoer is the small seam needed by tests.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// URLResolver resolves a scenario API URL.
type URLResolver interface {
	ResolveScenarioURLDefault(context.Context, string) (string, error)
}

// Config controls optional source lookup.
type Config struct {
	HTTPClient HTTPDoer
	Resolver   URLResolver
	Timeout    time.Duration
	Window     time.Duration
}

// Service fetches optional importance sources.
type Service struct {
	httpClient HTTPDoer
	resolver   URLResolver
	timeout    time.Duration
	window     time.Duration
}

// New returns the production reader. Source base URLs may be supplied via
// SCENARIO_DEPENDENCY_ANALYZER_API_BASE / _URL and SWARM_MANAGER_API_BASE /
// _URL; otherwise api-core discovery is attempted inside the request budget.
func New(cfg Config) *Service {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	window := cfg.Window
	if window <= 0 {
		window = 24 * time.Hour
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	resolver := cfg.Resolver
	if resolver == nil {
		resolver = discovery.NewResolver(discovery.ResolverConfig{})
	}
	return &Service{
		httpClient: client,
		resolver:   resolver,
		timeout:    timeout,
		window:     window,
	}
}

// Fetch returns nil when every optional source misses. Partial source data
// still returns a Summary with neutral defaults and degraded notes.
func (s *Service) Fetch(ctx context.Context, scenario string, systemRequired bool) *Summary {
	if s == nil {
		return nil
	}
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return nil
	}
	timeout := s.timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	centrality, centralityOK, centralityErr := s.fetchCentrality(ctx, scenario)
	recency, recencyOK, recencyErr := s.fetchRecency(ctx, scenario)
	if !centralityOK && !recencyOK {
		return nil
	}

	degraded := []string{}
	if !centralityOK {
		degraded = append(degraded, "centrality:"+shortReason(centralityErr))
	}
	if !recencyOK {
		degraded = append(degraded, "recency:"+shortReason(recencyErr))
	}

	return compose(scenario, systemRequired, centrality, centralityOK, recency, recencyOK, degraded)
}

type centralityMetric struct {
	Scenario                         string  `json:"scenario"`
	DirectReverseDependencyCount     int     `json:"direct_reverse_dependency_count"`
	TransitiveReverseDependencyCount int     `json:"transitive_reverse_dependency_count"`
	RequiredReverseDependencyCount   int     `json:"required_reverse_dependency_count"`
	RequiredEdgeWeightedScore        float64 `json:"required_edge_weighted_score"`
	DistanceToCoreSeed               int     `json:"distance_to_core_seed"`
	NearestCoreSeed                  string  `json:"nearest_core_seed"`
}

func (s *Service) fetchCentrality(ctx context.Context, scenario string) (centralityMetric, bool, error) {
	base, err := s.baseURL(ctx, "scenario-dependency-analyzer",
		"SCENARIO_DEPENDENCY_ANALYZER_API_BASE", "SCENARIO_DEPENDENCY_ANALYZER_API_URL")
	if err != nil {
		return centralityMetric{}, false, err
	}
	var payload struct {
		Nodes []centralityMetric `json:"nodes"`
	}
	if err := s.getJSON(ctx, strings.TrimRight(base, "/")+"/api/v1/graph/centrality", &payload); err != nil {
		return centralityMetric{}, false, err
	}
	for _, node := range payload.Nodes {
		if strings.TrimSpace(node.Scenario) == scenario {
			return node, true, nil
		}
	}
	return centralityMetric{}, false, fmt.Errorf("scenario_not_found")
}

type operationActivity struct {
	OwnerType string `json:"owner_type"`
	OwnerKind string `json:"owner_kind"`
	OwnerName string `json:"owner_name"`
}

func (s *Service) fetchRecency(ctx context.Context, scenario string) (int, bool, error) {
	base, err := s.baseURL(ctx, "swarm-manager", "SWARM_MANAGER_API_BASE", "SWARM_MANAGER_API_URL")
	if err != nil {
		return 0, false, err
	}
	window := s.window
	if window <= 0 {
		window = 24 * time.Hour
	}
	endpoint := strings.TrimRight(base, "/") + "/api/v1/operations?window=" + url.QueryEscape(formatPTDuration(window))
	var payload struct {
		Activities       []operationActivity `json:"activities"`
		RecentlyFinished []operationActivity `json:"recently_finished"`
	}
	if err := s.getJSON(ctx, endpoint, &payload); err != nil {
		return 0, false, err
	}
	count := 0
	for _, row := range append(payload.Activities, payload.RecentlyFinished...) {
		if row.scenarioName() == scenario {
			count++
		}
	}
	return count, true, nil
}

func (a operationActivity) scenarioName() string {
	if strings.EqualFold(a.OwnerType, "scenario") || strings.EqualFold(a.OwnerKind, "scenario") {
		return strings.TrimSpace(a.OwnerName)
	}
	return ""
}

func (s *Service) baseURL(ctx context.Context, scenario string, envNames ...string) (string, error) {
	for _, name := range envNames {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value, nil
		}
	}
	if s.resolver == nil {
		return "", fmt.Errorf("not_configured")
	}
	return s.resolver.ResolveScenarioURLDefault(ctx, scenario)
}

func (s *Service) getJSON(ctx context.Context, endpoint string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http_status_%d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}

func compose(_ string, systemRequired bool, metric centralityMetric, hasCentrality bool, recentCount int, hasRecency bool, degraded []string) *Summary {
	centralityScore := defaultNeutralScore
	coreScore := defaultNeutralScore
	if hasCentrality {
		centralityScore = normalizeSingleCentrality(metric.RequiredEdgeWeightedScore)
		coreScore = normalizeCoreProximity(metric.DistanceToCoreSeed)
	}
	recencyScore := defaultNeutralScore
	if hasRecency {
		recencyScore = normalizeRecency(recentCount)
	}

	totalWeight := defaultCentralityWeight + defaultCoreProximityWeight + defaultRecencyWeight
	combined := ((centralityScore * defaultCentralityWeight) +
		(coreScore * defaultCoreProximityWeight) +
		(recencyScore * defaultRecencyWeight)) / totalWeight
	if systemRequired && combined < defaultSystemRequiredFloor {
		combined = defaultSystemRequiredFloor
	}

	return &Summary{
		Score:          round4(clamp01(combined)),
		SystemRequired: systemRequired,
		Components: Components{
			Centrality:    round4(centralityScore),
			CoreProximity: round4(coreScore),
			Recency:       round4(recencyScore),
		},
		Signals: Signals{
			DirectReverseDependencyCount:     metric.DirectReverseDependencyCount,
			TransitiveReverseDependencyCount: metric.TransitiveReverseDependencyCount,
			RequiredReverseDependencyCount:   metric.RequiredReverseDependencyCount,
			RequiredEdgeWeightedScore:        metric.RequiredEdgeWeightedScore,
			DistanceToCoreSeed:               metric.DistanceToCoreSeed,
			NearestCoreSeed:                  metric.NearestCoreSeed,
			RecentActivityCount:              recentCount,
		},
		Degraded: degraded,
	}
}

func normalizeSingleCentrality(value float64) float64 {
	if value <= 0 {
		return 0
	}
	if value >= 10 {
		return 1
	}
	return value / 10
}

func normalizeCoreProximity(distance int) float64 {
	if distance < 0 {
		return defaultNeutralScore
	}
	return 1.0 / float64(distance+1)
}

func normalizeRecency(count int) float64 {
	if count <= 0 {
		return 0
	}
	if count >= 5 {
		return 1
	}
	return float64(count) / 5.0
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func round4(value float64) float64 {
	return math.Round(value*10000) / 10000
}

func shortReason(err error) string {
	if err == nil {
		return "unavailable"
	}
	text := strings.TrimSpace(err.Error())
	if text == "" {
		return "unavailable"
	}
	if len(text) > 80 {
		return text[:80]
	}
	return text
}

func formatPTDuration(d time.Duration) string {
	seconds := int(d.Seconds())
	if seconds%3600 == 0 {
		return fmt.Sprintf("PT%dH", seconds/3600)
	}
	if seconds%60 == 0 {
		return fmt.Sprintf("PT%dM", seconds/60)
	}
	return fmt.Sprintf("PT%dS", seconds)
}
