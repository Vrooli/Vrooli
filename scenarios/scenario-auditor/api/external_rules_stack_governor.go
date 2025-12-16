package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	rulespkg "scenario-auditor/rules"
)

// stackGovernorProvider delegates stack governance checks to scenario-stack-governor.
// scenario-auditor acts as the hub; scenario-stack-governor is the source of truth for these rules.
type stackGovernorProvider struct {
	client     *http.Client
	ruleLookup map[string]rulespkg.Rule
}

func newStackGovernorProvider() externalRuleProvider {
	p := &stackGovernorProvider{
		client:     &http.Client{Timeout: 5 * time.Minute},
		ruleLookup: make(map[string]rulespkg.Rule),
	}
	for _, rule := range p.Rules() {
		p.ruleLookup[rule.ID] = rule
	}
	return p
}

func init() {
	registerExternalProvider(newStackGovernorProvider())
}

func (p *stackGovernorProvider) ID() string {
	return "scenario-stack-governor"
}

func (p *stackGovernorProvider) Name() string {
	return "Scenario Stack Governor"
}

func (p *stackGovernorProvider) Rules() []rulespkg.Rule {
	return []rulespkg.Rule{
		{
			ID:          "GO_CLI_WORKSPACE_INDEPENDENCE",
			Name:        "Go CLI builds without workspace mode",
			Description: "Ensures Go-based scenario CLIs build with GOWORK=off (no dependency on repo-level go.work) and flags missing non-transitive replace/wiring.",
			Category:    "go",
			Severity:    "high",
			Enabled:     true,
			Standard:    "stack-governance",
		},
		{
			ID:          "REACT_VITE_UI_INSTALLS_DEPENDENCIES",
			Name:        "React/Vite UI installs dependencies correctly",
			Description: "Ensures lifecycle setup installs ui dependencies deterministically (pnpm install --ignore-workspace) to avoid vite not found during build-ui.",
			Category:    "typescript",
			Severity:    "high",
			Enabled:     true,
			Standard:    "stack-governance",
		},
	}
}

type stackGovernorRunRequest struct {
	ScenarioName string `json:"scenario_name,omitempty"`
}

type stackGovernorRunResponse struct {
	RepoRoot string                    `json:"repo_root"`
	Results  []stackGovernorRuleResult `json:"results"`
}

type stackGovernorRuleResult struct {
	RuleID   string                 `json:"rule_id"`
	Passed   bool                   `json:"passed"`
	Findings []stackGovernorFinding `json:"findings"`
}

type stackGovernorFinding struct {
	Level    string                  `json:"level"`
	Message  string                  `json:"message"`
	Evidence []stackGovernorEvidence `json:"evidence"`
}

type stackGovernorEvidence struct {
	Type   string `json:"type"`
	Ref    string `json:"ref,omitempty"`
	Detail string `json:"detail,omitempty"`
}

func (p *stackGovernorProvider) Run(ctx context.Context, scenarioName string, ruleIDs []string) ([]StandardsViolation, error) {
	cleaned := strings.TrimSpace(scenarioName)
	if cleaned == "" {
		return nil, nil
	}

	port, err := resolveScenarioPortViaCLI(ctx, "scenario-stack-governor", "API_PORT")
	if err != nil {
		return nil, err
	}
	if port <= 0 {
		return nil, fmt.Errorf("scenario-stack-governor API port not found")
	}

	payload, _ := json.Marshal(stackGovernorRunRequest{ScenarioName: cleaned})
	endpoint := fmt.Sprintf("http://localhost:%d/api/v1/run", port)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("scenario-stack-governor responded with %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var parsed stackGovernorRunResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	requested := map[string]struct{}{}
	for _, id := range ruleIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		requested[id] = struct{}{}
	}

	now := time.Now().Format(time.RFC3339)
	var violations []StandardsViolation
	for _, res := range parsed.Results {
		if len(requested) > 0 {
			if _, ok := requested[res.RuleID]; !ok {
				continue
			}
		}
		if res.Passed {
			continue
		}
		meta := p.ruleLookup[res.RuleID]
		for _, f := range res.Findings {
			severity := mapGovernorLevelToSeverity(f.Level)
			filePath, recommendation := pickViolationDetails(f.Evidence)
			if recommendation == "" {
				recommendation = "Review the rule output and apply the suggested remediation."
			}

			violations = append(violations, StandardsViolation{
				ID:             uuid.New().String(),
				ScenarioName:   cleaned,
				Type:           res.RuleID,
				Severity:       severity,
				Title:          meta.Name,
				Description:    f.Message,
				FilePath:       filePath,
				LineNumber:     0,
				Recommendation: recommendation,
				Standard:       meta.Standard,
				DiscoveredAt:   now,
				Metadata: map[string]any{
					"provider_repo_root": parsed.RepoRoot,
					"evidence":           f.Evidence,
				},
			})
		}
	}

	return violations, nil
}

func mapGovernorLevelToSeverity(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "error":
		return "high"
	case "warn", "warning":
		return "medium"
	default:
		return "low"
	}
}

func pickViolationDetails(evidence []stackGovernorEvidence) (filePath string, recommendation string) {
	for _, ev := range evidence {
		if ev.Type == "file" && ev.Ref != "" {
			filePath = ev.Ref
			break
		}
	}
	if filePath == "" {
		for _, ev := range evidence {
			if ev.Type == "path" && ev.Ref != "" {
				filePath = ev.Ref
				break
			}
		}
	}

	for _, ev := range evidence {
		if ev.Type != "note" {
			continue
		}
		t := strings.TrimSpace(ev.Detail)
		if t == "" {
			continue
		}
		recommendation = t
		break
	}
	return filePath, recommendation
}
