package closure

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// AnalyzedDependency is one edge the analyzer reports for a scenario.
type AnalyzedDependency struct {
	Name     string
	Required bool
	Enabled  bool
}

// AnalyzerResult is the analyzer's reading of one scenario.
type AnalyzerResult struct {
	Resources   []AnalyzedDependency
	Scenarios   []AnalyzedDependency
	Tool        string
	Fingerprint string
}

// AnalyzerClient reads scenario-dependency-analyzer results. A nil client
// means the closure is derived from declarations only, which the closure
// records in Sources.AnalyzerUsed.
type AnalyzerClient interface {
	Analyze(ctx context.Context, scenarioID string) (AnalyzerResult, error)
}

// HTTPAnalyzer calls the analyzer's read-only /api/v1/analyze/{scenario} API.
type HTTPAnalyzer struct {
	ResolveBaseURL func(ctx context.Context) (string, error)
	Client         *http.Client
}

// NewHTTPAnalyzer builds a client over a base-URL resolver (service discovery).
func NewHTTPAnalyzer(resolve func(ctx context.Context) (string, error)) *HTTPAnalyzer {
	return &HTTPAnalyzer{ResolveBaseURL: resolve, Client: &http.Client{Timeout: 60 * time.Second}}
}

const analyzerTool = "scenario-dependency-analyzer"

// Analyze implements AnalyzerClient.
func (a *HTTPAnalyzer) Analyze(ctx context.Context, scenarioID string) (AnalyzerResult, error) {
	if a == nil || a.ResolveBaseURL == nil {
		return AnalyzerResult{}, fmt.Errorf("analyzer base URL resolver is not configured")
	}
	if !validComponentID(scenarioID) {
		return AnalyzerResult{}, fmt.Errorf("invalid scenario id %q", scenarioID)
	}
	baseURL, err := a.ResolveBaseURL(ctx)
	if err != nil {
		return AnalyzerResult{}, fmt.Errorf("analyzer not available: %w", err)
	}
	url := strings.TrimSuffix(baseURL, "/") + "/api/v1/analyze/" + scenarioID
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return AnalyzerResult{}, fmt.Errorf("create analyzer request: %w", err)
	}
	client := a.Client
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return AnalyzerResult{}, fmt.Errorf("analyzer request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return AnalyzerResult{}, fmt.Errorf("analyzer returned status %d", resp.StatusCode)
	}
	var body struct {
		Resources        []analyzerEdge `json:"resources"`
		Scenarios        []analyzerEdge `json:"scenarios"`
		DeploymentReport struct {
			Provenance struct {
				InputDigest string `json:"input_digest"`
			} `json:"provenance"`
		} `json:"deployment_report"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return AnalyzerResult{}, fmt.Errorf("parse analyzer response: %w", err)
	}
	result := AnalyzerResult{Tool: analyzerTool, Fingerprint: body.DeploymentReport.Provenance.InputDigest}
	for _, edge := range body.Resources {
		result.Resources = append(result.Resources, edge.dependency())
	}
	for _, edge := range body.Scenarios {
		result.Scenarios = append(result.Scenarios, edge.dependency())
	}
	return result, nil
}

type analyzerEdge struct {
	DependencyName string `json:"dependency_name"`
	Required       bool   `json:"required"`
	Configuration  struct {
		Enabled bool `json:"enabled"`
	} `json:"configuration"`
}

func (e analyzerEdge) dependency() AnalyzedDependency {
	return AnalyzedDependency{Name: e.DependencyName, Required: e.Required, Enabled: e.Configuration.Enabled}
}
