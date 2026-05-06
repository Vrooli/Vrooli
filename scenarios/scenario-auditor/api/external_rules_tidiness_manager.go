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
	"github.com/vrooli/api-core/discovery"

	rulespkg "scenario-auditor/rules"
)

// tidinessManagerProvider delegates type-safety checks to tidiness-manager.
type tidinessManagerProvider struct {
	client     *http.Client
	ruleLookup map[string]rulespkg.Rule
}

func newTidinessManagerProvider() externalRuleProvider {
	p := &tidinessManagerProvider{
		client:     &http.Client{Timeout: 2 * time.Minute},
		ruleLookup: make(map[string]rulespkg.Rule),
	}
	for _, rule := range p.Rules() {
		p.ruleLookup[rule.ID] = rule
	}
	return p
}

func (p *tidinessManagerProvider) ID() string {
	return "tidiness-manager"
}

func (p *tidinessManagerProvider) Name() string {
	return "Tidiness Manager"
}

func (p *tidinessManagerProvider) Rules() []rulespkg.Rule {
	return []rulespkg.Rule{
		{
			ID:          "TS_CONFIG_STRICT",
			Name:        "TypeScript strict mode with protective comments",
			Description: "Ensures tsconfig.json has strict: true, noUncheckedIndexedAccess: true, and protective comment blocks that prevent future agents from weakening type safety.",
			Category:    "typescript",
			Severity:    "high",
			Enabled:     true,
			Standard:    "type-safety",
		},
		{
			ID:          "ESLINT_SAFETY_RULES",
			Name:        "ESLint safety-critical rules configured",
			Description: "Ensures ESLint config has required safety rules (react-hooks, no-non-null-assertion, no-unsafe-*, import/no-cycle) with protective comment blocks.",
			Category:    "typescript",
			Severity:    "high",
			Enabled:     true,
			Standard:    "type-safety",
		},
		{
			ID:          "TS_DANGEROUS_PATTERNS",
			Name:        "No dangerous TypeScript patterns",
			Description: "Detects per-file as any, as Type, @ts-ignore, and non-null assertions that bypass TypeScript safety.",
			Category:    "typescript",
			Severity:    "medium",
			Enabled:     true,
			Standard:    "type-safety",
		},
		{
			ID:          "ESLINT_TYPED_CONFIG",
			Name:        "ESLint typed configuration complete",
			Description: "Ensures ESLint uses strict typed linting, parserOptions.project, and the TypeScript import resolver when import/no-cycle is enabled.",
			Category:    "typescript",
			Severity:    "high",
			Enabled:     true,
			Standard:    "type-safety",
		},
		{
			ID:          "NODE_BUILD_TYPECHECK",
			Name:        "UI build performs type checking",
			Description: "Ensures ui/package.json build scripts run TypeScript type checking before bundling.",
			Category:    "typescript",
			Severity:    "high",
			Enabled:     true,
			Standard:    "quality-gates",
		},
		{
			ID:          "TESTING_CONFIG_LINT_STRICT",
			Name:        "Lint strict mode enabled in testing.json",
			Description: "Ensures .vrooli/testing.json enables strict lint failure behavior for applicable languages.",
			Category:    "testing",
			Severity:    "high",
			Enabled:     true,
			Standard:    "quality-gates",
		},
		{
			ID:          "GO_MOD_PRESENT_FOR_API_OR_CLI",
			Name:        "Go modules present for Go targets",
			Description: "Ensures api/ or cli/ directories with Go code contain go.mod so tooling can run consistently.",
			Category:    "go",
			Severity:    "high",
			Enabled:     true,
			Standard:    "go-quality",
		},
		{
			ID:          "GO_LINT_CONFIG_PRESENT",
			Name:        "golangci-lint config present for Go targets",
			Description: "Ensures Go targets with source files include .golangci.yml/.golangci.yaml.",
			Category:    "go",
			Severity:    "high",
			Enabled:     true,
			Standard:    "go-quality",
		},
		{
			ID:          "GO_LINT_REQUIRED_LINTERS",
			Name:        "golangci-lint baseline complete",
			Description: "Ensures Go lint config enables the required baseline linters (gofumpt, govet, typecheck, staticcheck, errcheck, ineffassign, unused).",
			Category:    "go",
			Severity:    "high",
			Enabled:     true,
			Standard:    "go-quality",
		},
		{
			ID:          "MAKEFILE_QUALITY_GATES",
			Name:        "Makefile quality targets wired",
			Description: "Ensures scenario Makefiles expose real fmt/lint targets for UI and Go code instead of placeholders.",
			Category:    "build",
			Severity:    "medium",
			Enabled:     true,
			Standard:    "quality-gates",
		},
	}
}

type tidinessManagerScanRequest struct {
	ScenarioName    string `json:"scenario_name"`
	IncludePatterns bool   `json:"include_patterns"`
}

type tidinessManagerScanResponse struct {
	Scenario                     string                         `json:"scenario"`
	TSConfigFound                bool                           `json:"tsconfig_found"`
	TSConfigStrict               bool                           `json:"tsconfig_strict"`
	TSConfigNoUnchecked          bool                           `json:"tsconfig_no_unchecked_indexed_access"`
	TSConfigHasProtectiveComment bool                           `json:"tsconfig_has_protective_comment"`
	ESLintConfigFound            bool                           `json:"eslint_config_found"`
	ESLintHasHeaderComment       bool                           `json:"eslint_has_header_comment"`
	ESLintHasPerRuleComments     bool                           `json:"eslint_has_per_rule_comments"`
	ESLintSafetyRules            []tidinessManagerRuleStatus    `json:"eslint_safety_rules"`
	PatternSummary               *tidinessManagerPatternSummary `json:"pattern_summary,omitempty"`
	Violations                   []tidinessManagerViolation     `json:"violations"`
}

type tidinessManagerRuleStatus struct {
	Rule     string `json:"rule"`
	MinLevel string `json:"min_level"`
	Found    bool   `json:"found"`
	Level    string `json:"level,omitempty"`
}

type tidinessManagerPatternSummary struct {
	TotalFiles            int `json:"total_files"`
	AsAnyCount            int `json:"as_any_count"`
	AsTypeAssertionCount  int `json:"as_type_assertion_count"`
	TsIgnoreCount         int `json:"ts_ignore_count"`
	NonNullAssertionCount int `json:"non_null_assertion_count"`
	TopFiles              []struct {
		FilePath              string `json:"file_path"`
		AsAnyCount            int    `json:"as_any_count,omitempty"`
		AsTypeAssertionCount  int    `json:"as_type_assertion_count,omitempty"`
		TsIgnoreCount         int    `json:"ts_ignore_count,omitempty"`
		NonNullAssertionCount int    `json:"non_null_assertion_count,omitempty"`
		Total                 int    `json:"total"`
	} `json:"top_files,omitempty"`
}

type tidinessManagerViolation struct {
	RuleID      string `json:"rule_id"`
	Severity    string `json:"severity"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Remediation string `json:"remediation,omitempty"`
	FilePath    string `json:"file_path,omitempty"`
}

func (p *tidinessManagerProvider) Run(ctx context.Context, target standardsScanTarget, ruleIDs []string) ([]StandardsViolation, error) {
	cleaned := strings.TrimSpace(target.Name)
	if cleaned == "" {
		return nil, nil
	}

	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, "tidiness-manager")
	if err != nil {
		return nil, err
	}

	// Determine if we need pattern detection
	requested := map[string]struct{}{}
	for _, id := range ruleIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			requested[id] = struct{}{}
		}
	}

	includePatterns := false
	if len(requested) == 0 {
		includePatterns = true
	} else if _, ok := requested["TS_DANGEROUS_PATTERNS"]; ok {
		includePatterns = true
	}

	payload, _ := json.Marshal(tidinessManagerScanRequest{
		ScenarioName:    cleaned,
		IncludePatterns: includePatterns,
	})
	endpoint := fmt.Sprintf("%s/api/v1/scan/type-safety", baseURL)
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
		return nil, fmt.Errorf("tidiness-manager responded with %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var parsed tidinessManagerScanResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	now := time.Now().Format(time.RFC3339)
	var violations []StandardsViolation

	for _, v := range parsed.Violations {
		if len(requested) > 0 {
			if _, ok := requested[v.RuleID]; !ok {
				continue
			}
		}
		meta := p.ruleLookup[v.RuleID]
		recommendation := v.Remediation
		if recommendation == "" {
			recommendation = "Review the violation and apply the suggested remediation."
		}
		violations = append(violations, StandardsViolation{
			ID:             uuid.New().String(),
			ScenarioName:   cleaned,
			Type:           v.RuleID,
			Severity:       v.Severity,
			Title:          v.Title,
			Description:    v.Description,
			FilePath:       v.FilePath,
			Recommendation: recommendation,
			Standard:       meta.Standard,
			DiscoveredAt:   now,
		})
	}

	// Add pattern summary violation if applicable
	if includePatterns && parsed.PatternSummary != nil {
		ps := parsed.PatternSummary
		totalPatterns := ps.AsAnyCount + ps.AsTypeAssertionCount + ps.TsIgnoreCount + ps.NonNullAssertionCount
		if totalPatterns > 0 {
			if len(requested) == 0 || func() bool { _, ok := requested["TS_DANGEROUS_PATTERNS"]; return ok }() {
				recommendation := "Replace dangerous patterns with safe alternatives:\n" +
					"  • as any / as Type → Use type guards: if (typeof x === 'string') { x.trim() }\n" +
					"  • @ts-ignore → Fix the underlying type error with proper types\n" +
					"  • Non-null assertion (!) → Use optional chaining (?.) or nullish coalescing (??)\n\n" +
					"Run `pnpm lint` in the ui/ directory to find specific locations."

				if len(ps.TopFiles) > 0 {
					var fileBreakdown strings.Builder
					fileBreakdown.WriteString("\n\nTop files to fix:\n")
					for _, f := range ps.TopFiles {
						fileBreakdown.WriteString(fmt.Sprintf("  %s: %d patterns", f.FilePath, f.Total))
						var details []string
						if f.AsAnyCount > 0 {
							details = append(details, fmt.Sprintf("%d as-any", f.AsAnyCount))
						}
						if f.AsTypeAssertionCount > 0 {
							details = append(details, fmt.Sprintf("%d as-type", f.AsTypeAssertionCount))
						}
						if f.TsIgnoreCount > 0 {
							details = append(details, fmt.Sprintf("%d ts-ignore", f.TsIgnoreCount))
						}
						if f.NonNullAssertionCount > 0 {
							details = append(details, fmt.Sprintf("%d non-null", f.NonNullAssertionCount))
						}
						if len(details) > 0 {
							fileBreakdown.WriteString(fmt.Sprintf(" (%s)", strings.Join(details, ", ")))
						}
						fileBreakdown.WriteString("\n")
					}
					recommendation += fileBreakdown.String()
				}

				violations = append(violations, StandardsViolation{
					ID:           uuid.New().String(),
					ScenarioName: cleaned,
					Type:         "TS_DANGEROUS_PATTERNS",
					Severity:     "medium",
					Title:        "Dangerous TypeScript patterns detected",
					Description: fmt.Sprintf(
						"Found %d dangerous patterns across %d files: %d as-any, %d as-type, %d ts-ignore, %d non-null assertions",
						totalPatterns, ps.TotalFiles, ps.AsAnyCount, ps.AsTypeAssertionCount, ps.TsIgnoreCount, ps.NonNullAssertionCount,
					),
					Recommendation: recommendation,
					Standard:       "type-safety",
					DiscoveredAt:   now,
				})
			}
		}
	}

	return violations, nil
}

type tidinessManagerFixRequest struct {
	ScenarioName string `json:"scenario_name"`
}

type tidinessManagerFixResponse = tidinessManagerScanResponse

func (p *tidinessManagerProvider) Fix(ctx context.Context, scenarioNames []string, ruleIDs []string, dryRun bool) ([]ExternalFixResult, error) {
	// Only TS_CONFIG_STRICT is fixable
	fixable := false
	for _, id := range ruleIDs {
		if strings.TrimSpace(id) == "TS_CONFIG_STRICT" {
			fixable = true
			break
		}
	}
	if !fixable {
		return nil, fmt.Errorf("only TS_CONFIG_STRICT is auto-fixable")
	}

	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, "tidiness-manager")
	if err != nil {
		return nil, err
	}

	var results []ExternalFixResult
	for _, name := range scenarioNames {
		cleaned := strings.TrimSpace(name)
		if cleaned == "" {
			continue
		}

		if dryRun {
			// For dry run, just scan and report what would change
			results = append(results, ExternalFixResult{
				ScenarioName: cleaned,
				RuleID:       "TS_CONFIG_STRICT",
				Fixed:        false,
				Changes: []ExternalFixChange{
					{Type: "dry_run", Detail: "Would add strict: true, noUncheckedIndexedAccess: true, and protective comments to tsconfig.json"},
				},
			})
			continue
		}

		payload, _ := json.Marshal(tidinessManagerFixRequest{ScenarioName: cleaned})
		endpoint := fmt.Sprintf("%s/api/v1/scan/type-safety/fix", baseURL)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
		if err != nil {
			results = append(results, ExternalFixResult{
				ScenarioName: cleaned,
				RuleID:       "TS_CONFIG_STRICT",
				Fixed:        false,
				Error:        err.Error(),
			})
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := p.client.Do(req)
		if err != nil {
			results = append(results, ExternalFixResult{
				ScenarioName: cleaned,
				RuleID:       "TS_CONFIG_STRICT",
				Fixed:        false,
				Error:        err.Error(),
			})
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode >= http.StatusBadRequest {
			b, _ := io.ReadAll(resp.Body)
			results = append(results, ExternalFixResult{
				ScenarioName: cleaned,
				RuleID:       "TS_CONFIG_STRICT",
				Fixed:        false,
				Error:        fmt.Sprintf("tidiness-manager fix responded with %d: %s", resp.StatusCode, strings.TrimSpace(string(b))),
			})
			continue
		}

		var parsed tidinessManagerFixResponse
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			results = append(results, ExternalFixResult{
				ScenarioName: cleaned,
				RuleID:       "TS_CONFIG_STRICT",
				Fixed:        false,
				Error:        err.Error(),
			})
			continue
		}

		// Check if violations still exist after fix
		hasViolations := false
		for _, v := range parsed.Violations {
			if v.RuleID == "TS_CONFIG_STRICT" {
				hasViolations = true
				break
			}
		}

		changes := []ExternalFixChange{
			{Type: "config_update", Detail: "Updated tsconfig.json with strict: true, noUncheckedIndexedAccess: true, and protective comment block"},
		}

		results = append(results, ExternalFixResult{
			ScenarioName: cleaned,
			RuleID:       "TS_CONFIG_STRICT",
			Fixed:        !hasViolations,
			FilePath:     "ui/tsconfig.json",
			Changes:      changes,
		})
	}

	return results, nil
}
