package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	rulespkg "scenario-auditor/rules"
)

// testGenieProvider implements externalRuleProvider for test-genie structure validation.
// This delegates structure validation to test-genie, making it the single source of truth
// for scenario structure requirements. This avoids drift between scenario-auditor and
// test-genie's structure phase.
type testGenieProvider struct {
	client     *http.Client
	ruleLookup map[string]rulespkg.Rule
}

func newTestGenieProvider() externalRuleProvider {
	provider := &testGenieProvider{
		client:     &http.Client{Timeout: 30 * time.Second},
		ruleLookup: make(map[string]rulespkg.Rule),
	}
	for _, rule := range provider.Rules() {
		provider.ruleLookup[rule.ID] = rule
	}
	return provider
}

func init() {
	registerExternalProvider(newTestGenieProvider())
}

func (p *testGenieProvider) ID() string {
	return "test-genie"
}

func (p *testGenieProvider) Name() string {
	return "Test Genie"
}

// Rules returns the structure validation rules managed by test-genie.
// These rule IDs must match what test-genie's standards endpoint returns.
func (p *testGenieProvider) Rules() []rulespkg.Rule {
	return []rulespkg.Rule{
		// BAS/Playbooks structure (required when ui/ exists)
		{ID: "structure_bas", Name: "BAS playbooks structure", Description: "bas/ directory with cases/, flows/, actions/, seeds/ subdirectories (required when ui/ exists)", Category: "structure", Severity: "high", Enabled: true, Standard: "scenario-structure"},

		// Core structure rules
		{ID: "structure_required_dirs", Name: "Required directories", Description: "Required scenario directories exist", Category: "structure", Severity: "high", Enabled: true, Standard: "scenario-structure"},
		{ID: "structure_required_files", Name: "Required files", Description: "Required scenario files exist", Category: "structure", Severity: "high", Enabled: true, Standard: "scenario-structure"},
		{ID: "structure_service_json", Name: "Service manifest", Description: ".vrooli/service.json is valid", Category: "structure", Severity: "high", Enabled: true, Standard: "scenario-structure"},
		{ID: "structure_cli", Name: "CLI structure", Description: "CLI entry point is properly configured", Category: "structure", Severity: "medium", Enabled: true, Standard: "scenario-structure"},
		{ID: "structure_general", Name: "General structure", Description: "General structure validation", Category: "structure", Severity: "medium", Enabled: true, Standard: "scenario-structure"},
	}
}

// testGenieStandardsResponse matches the response format from test-genie's
// /api/v1/quality/scenario/{name}/structure endpoint.
type testGenieStandardsResponse struct {
	EntityName  string                      `json:"entity_name"`
	ValidatedAt string                      `json:"validated_at"`
	Success     bool                        `json:"success"`
	Violations  []testGenieStructureViolation `json:"violations"`
	Summary     string                      `json:"summary"`
}

type testGenieStructureViolation struct {
	RuleID         string         `json:"rule_id"`
	Severity       string         `json:"severity"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	FilePath       string         `json:"file_path"`
	Recommendation string         `json:"recommendation"`
	Metadata       map[string]any `json:"metadata"`
}

func (p *testGenieProvider) Run(ctx context.Context, scenarioName string, ruleIDs []string) ([]StandardsViolation, error) {
	cleaned := strings.TrimSpace(scenarioName)
	if cleaned == "" {
		return nil, nil
	}

	requested := make(map[string]struct{})
	for _, id := range ruleIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		requested[id] = struct{}{}
	}

	port, err := resolveScenarioPortViaCLI(ctx, "test-genie", "API_PORT")
	if err != nil {
		return nil, err
	}
	if port <= 0 {
		return nil, fmt.Errorf("test-genie API port not found")
	}

	endpoint := fmt.Sprintf("http://localhost:%d/api/v1/quality/scenario/%s/structure", port, url.PathEscape(cleaned))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		payload, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("test-genie responded with %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	var parsed testGenieStandardsResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	var violations []StandardsViolation
	for _, violation := range parsed.Violations {
		// Filter by requested rule IDs if specified
		if len(requested) > 0 {
			if _, ok := requested[violation.RuleID]; !ok {
				continue
			}
		}

		ruleMeta := p.ruleLookup[violation.RuleID]
		violations = append(violations, StandardsViolation{
			ID:             uuid.New().String(),
			ScenarioName:   cleaned,
			Type:           violation.RuleID,
			Severity:       strings.ToLower(violation.Severity),
			Title:          violation.Title,
			Description:    violation.Description,
			FilePath:       violation.FilePath,
			Recommendation: violation.Recommendation,
			Standard:       ruleMeta.Standard,
			DiscoveredAt:   time.Now().Format(time.RFC3339),
			Metadata:       violation.Metadata,
		})
	}

	return violations, nil
}
