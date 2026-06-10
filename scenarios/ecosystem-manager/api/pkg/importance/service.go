package importance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type ScenarioReader interface {
	ListScenarios(ctx context.Context) ([]ScenarioFact, error)
}

type CentralityReader interface {
	Centrality(ctx context.Context) ([]CentralityMetric, error)
}

type RecencyReader interface {
	RecentActivity(ctx context.Context) (map[string]int, error)
}

type Service struct {
	cfg        Config
	scenarios  ScenarioReader
	centrality CentralityReader
	recency    RecencyReader
	now        func() time.Time

	mu       sync.Mutex
	cached   Report
	cachedAt time.Time
}

type ServiceConfig struct {
	Config     Config
	Scenarios  ScenarioReader
	Centrality CentralityReader
	Recency    RecencyReader
	Now        func() time.Time
}

func NewService(cfg ServiceConfig) *Service {
	if cfg.Config == (Config{}) {
		cfg.Config = DefaultConfig()
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Service{
		cfg:        cfg.Config,
		scenarios:  cfg.Scenarios,
		centrality: cfg.Centrality,
		recency:    cfg.Recency,
		now:        cfg.Now,
	}
}

func NewDefaultService(projectRoot string) *Service {
	httpClient := &http.Client{Timeout: 2 * time.Second}
	return NewService(ServiceConfig{
		Scenarios: FileScenarioReader{ScenariosDir: filepath.Join(projectRoot, "scenarios")},
		Centrality: HTTPDependencyAnalyzerReader{
			BaseURL:    firstEnv("SCENARIO_DEPENDENCY_ANALYZER_API_BASE", "SCENARIO_DEPENDENCY_ANALYZER_API_URL"),
			HTTPClient: httpClient,
		},
		Recency: HTTPSwarmOperationsReader{
			BaseURL:    firstEnv("SWARM_MANAGER_API_BASE", "SWARM_MANAGER_API_URL"),
			HTTPClient: httpClient,
			Window:     24 * time.Hour,
		},
	})
}

func (s *Service) Report(ctx context.Context, refresh bool) (Report, error) {
	if s == nil {
		return Report{}, fmt.Errorf("importance service is not initialized")
	}
	now := s.now().UTC()
	if !refresh {
		s.mu.Lock()
		if !s.cachedAt.IsZero() && now.Sub(s.cachedAt) < s.cfg.CacheTTL {
			cached := s.cached
			s.mu.Unlock()
			return cached, nil
		}
		s.mu.Unlock()
	}

	facts, err := s.scenarios.ListScenarios(ctx)
	if err != nil {
		return Report{}, fmt.Errorf("list scenarios: %w", err)
	}

	degraded := []string{}
	centrality := []CentralityMetric{}
	if s.centrality != nil {
		centrality, err = s.centrality.Centrality(ctx)
		if err != nil {
			degraded = append(degraded, "centrality:"+err.Error())
		}
	} else {
		degraded = append(degraded, "centrality:not_configured")
	}

	recency := map[string]int{}
	if s.recency != nil {
		recency, err = s.recency.RecentActivity(ctx)
		if err != nil {
			degraded = append(degraded, "recency:"+err.Error())
		}
	} else {
		degraded = append(degraded, "recency:not_configured")
	}

	report := Compose(facts, centrality, recency, s.cfg, degraded)
	report.GeneratedAt = now

	s.mu.Lock()
	s.cached = report
	s.cachedAt = now
	s.mu.Unlock()

	return report, nil
}

type FileScenarioReader struct {
	ScenariosDir string
}

func (r FileScenarioReader) ListScenarios(context.Context) ([]ScenarioFact, error) {
	if strings.TrimSpace(r.ScenariosDir) == "" {
		return nil, fmt.Errorf("scenarios dir is empty")
	}
	entries, err := os.ReadDir(r.ScenariosDir)
	if err != nil {
		return nil, err
	}
	out := make([]ScenarioFact, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		fact := ScenarioFact{Name: name}
		servicePath := filepath.Join(r.ScenariosDir, name, ".vrooli", "service.json")
		if data, err := os.ReadFile(servicePath); err == nil {
			fact.SystemRequired = parseSystemRequired(data)
		}
		out = append(out, fact)
	}
	return out, nil
}

func parseSystemRequired(data []byte) bool {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return false
	}
	if v, ok := raw["system_required"].(bool); ok {
		return v
	}
	if service, ok := raw["service"].(map[string]any); ok {
		if v, ok := service["system_required"].(bool); ok {
			return v
		}
	}
	return false
}

type HTTPDependencyAnalyzerReader struct {
	BaseURL    string
	HTTPClient *http.Client
}

func (r HTTPDependencyAnalyzerReader) Centrality(ctx context.Context) ([]CentralityMetric, error) {
	base := strings.TrimRight(strings.TrimSpace(r.BaseURL), "/")
	if base == "" {
		return nil, fmt.Errorf("not_configured")
	}
	var payload struct {
		Nodes []CentralityMetric `json:"nodes"`
	}
	if err := getJSON(ctx, r.HTTPClient, base+"/api/v1/graph/centrality", &payload); err != nil {
		return nil, err
	}
	return payload.Nodes, nil
}

type HTTPSwarmOperationsReader struct {
	BaseURL    string
	HTTPClient *http.Client
	Window     time.Duration
}

func (r HTTPSwarmOperationsReader) RecentActivity(ctx context.Context) (map[string]int, error) {
	base := strings.TrimRight(strings.TrimSpace(r.BaseURL), "/")
	if base == "" {
		return nil, fmt.Errorf("not_configured")
	}
	window := r.Window
	if window <= 0 {
		window = 24 * time.Hour
	}
	endpoint := base + "/api/v1/operations?window=" + url.QueryEscape(formatPTDuration(window))
	var payload struct {
		Activities       []operationActivity `json:"activities"`
		RecentlyFinished []operationActivity `json:"recently_finished"`
	}
	if err := getJSON(ctx, r.HTTPClient, endpoint, &payload); err != nil {
		return nil, err
	}
	counts := map[string]int{}
	for _, row := range append(payload.Activities, payload.RecentlyFinished...) {
		name := row.scenarioName()
		if name != "" {
			counts[name]++
		}
	}
	return counts, nil
}

type operationActivity struct {
	OwnerType  string `json:"owner_type"`
	OwnerKind  string `json:"owner_kind"`
	OwnerName  string `json:"owner_name"`
	OwnerTitle string `json:"owner_title"`
}

func (a operationActivity) scenarioName() string {
	if strings.EqualFold(a.OwnerType, "scenario") || strings.EqualFold(a.OwnerKind, "scenario") {
		return strings.TrimSpace(a.OwnerName)
	}
	return ""
}

func getJSON(ctx context.Context, client *http.Client, endpoint string, dest any) error {
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(dest)
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

func firstEnv(names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return ""
}
