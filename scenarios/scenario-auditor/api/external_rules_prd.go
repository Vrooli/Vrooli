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

type prdControlTowerProvider struct {
	client     *http.Client
	ruleLookup map[string]rulespkg.Rule
}

func newPRDControlTowerProvider() externalRuleProvider {
	provider := &prdControlTowerProvider{
		client:     &http.Client{Timeout: 30 * time.Second},
		ruleLookup: make(map[string]rulespkg.Rule),
	}
	for _, rule := range provider.Rules() {
		provider.ruleLookup[rule.ID] = rule
	}
	return provider
}

func init() {
	registerExternalProvider(newPRDControlTowerProvider())
}

func (p *prdControlTowerProvider) ID() string {
	return "prd-control-tower"
}

func (p *prdControlTowerProvider) Name() string {
	return "PRD Control Tower"
}

func (p *prdControlTowerProvider) Rules() []rulespkg.Rule {
	return []rulespkg.Rule{
		{ID: "prd_missing_prd", Name: "PRD.md present", Description: "Ensure PRD.md exists", Category: "documentation", Severity: "critical", Enabled: true, Standard: "prd-governance"},
		{ID: "prd_missing_requirements", Name: "Requirements registry present", Description: "Ensure requirements/index.json exists", Category: "documentation", Severity: "high", Enabled: true, Standard: "prd-governance"},
		{ID: "prd_template_sections", Name: "PRD template sections", Description: "Required template sections are present", Category: "documentation", Severity: "high", Enabled: true, Standard: "prd-template"},
		{ID: "prd_template_unexpected_sections", Name: "Unexpected PRD sections", Description: "Flag unexpected sections", Category: "documentation", Severity: "low", Enabled: true, Standard: "prd-template"},
		{ID: "prd_template_content", Name: "PRD content quality", Description: "Validate PRD content details", Category: "documentation", Severity: "medium", Enabled: true, Standard: "prd-template"},
		{ID: "prd_operational_target_linkage", Name: "Operational target linkage", Description: "P0/P1 targets link to requirements", Category: "documentation", Severity: "high", Enabled: true, Standard: "prd-linkage"},
		{ID: "prd_requirements_without_targets", Name: "Requirement linkage", Description: "Requirements link to operational targets", Category: "documentation", Severity: "medium", Enabled: true, Standard: "prd-linkage"},
		{ID: "prd_prd_ref_integrity", Name: "PRD reference integrity", Description: "Requirements reference valid PRD content", Category: "documentation", Severity: "medium", Enabled: true, Standard: "prd-linkage"},
	}
}

type prdStandardsResponse struct {
	EntityName string                  `json:"entity_name"`
	Violations []prdStandardsViolation `json:"violations"`
}

type prdStandardsViolation struct {
	RuleID         string         `json:"rule_id"`
	Severity       string         `json:"severity"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	FilePath       string         `json:"file_path"`
	Recommendation string         `json:"recommendation"`
	Metadata       map[string]any `json:"metadata"`
}

func (p *prdControlTowerProvider) Run(ctx context.Context, scenarioName string, ruleIDs []string) ([]StandardsViolation, error) {
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

	port, err := resolveScenarioPortViaCLI(ctx, "prd-control-tower", "API_PORT")
	if err != nil {
		return nil, err
	}
	if port <= 0 {
		return nil, fmt.Errorf("prd-control-tower API port not found")
	}

	endpoint := fmt.Sprintf("http://localhost:%d/api/v1/quality/scenario/%s/standards?use_cache=false", port, url.PathEscape(cleaned))
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
		return nil, fmt.Errorf("prd-control-tower responded with %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	var parsed prdStandardsResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	var violations []StandardsViolation
	for _, violation := range parsed.Violations {
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
