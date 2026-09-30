package main

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

const packageGovernanceRuleID = "PACKAGE_GOVERNANCE_SCENARIO_ADOPTION"

var structureHealthTimeout = 2 * time.Minute

var (
	resolveStructureHealthURL = discovery.ResolveScenarioURLDefault
	structureHealthHTTPClient = http.DefaultClient
)

type packageGovernanceIssue struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Path     string `json:"path"`
}

func RunPackageGovernanceScenarioAdoption(ctx context.Context, repoRoot, scenarioName string) (result RuleResult) {
	start := time.Now()
	result = RuleResult{
		RuleID:    packageGovernanceRuleID,
		StartedAt: start,
	}
	defer func() {
		result.FinishedAt = time.Now()
		result.Passed = !hasActionableFindings(result.Findings)
	}()

	issues, err := runStructureHealthProjectValidation(ctx, repoRoot)
	if err != nil {
		result.Findings = append(result.Findings, Finding{
			Level:   "error",
			Message: fmt.Sprintf("package governance validation failed: %v", err),
			Evidence: []Evidence{
				{Type: "scenario", Ref: "structure-health"},
			},
		})
		return result
	}

	filter := strings.TrimSpace(scenarioName)
	for _, issue := range issues {
		scenario, ok := scenarioFromIssuePath(repoRoot, issue.Path)
		if !ok {
			continue
		}
		if filter != "" && scenario != filter {
			continue
		}
		result.Findings = append(result.Findings, Finding{
			Level:        mapIssueSeverity(issue.Severity),
			Message:      fmt.Sprintf("%s: %s", scenario, issue.Message),
			ScenarioName: scenario,
			Evidence:     packageIssueEvidence(issue),
		})
	}

	return result
}

func packageIssueEvidence(issue packageGovernanceIssue) []Evidence {
	evidence := []Evidence{{Type: "scenario", Ref: "structure-health"}}
	if strings.TrimSpace(issue.Path) != "" {
		evidence = append(evidence, Evidence{Type: "file", Ref: issue.Path})
	}
	if strings.TrimSpace(issue.Code) != "" {
		evidence = append(evidence, Evidence{
			Type:   "note",
			Detail: packageGovernanceRecommendation(issue.Code),
		})
	}
	return evidence
}

func packageGovernanceRecommendation(code string) string {
	switch strings.TrimSpace(code) {
	case "package-no-workspace-deps":
		return "Replace workspace protocol shared-package references with a supported isolated adoption mode such as file: or a governed Go replace directive."
	case "package-no-unauthorized-postinstall":
		return "Move shared-package hydration into vrooli-managed lifecycle/setup behavior and remove package propagation from scenario-local postinstall scripts."
	case "package-adoption-supported":
		return "Only consume packages that declare scenario adoption support for this consumer class."
	case "package-adoption-mode-valid":
		return "Use one of the adoption modes explicitly declared in the package manifest."
	case "package-go-module-replace-required":
		return "Add the required local replace directive for the governed Go package so the consumer stays workspace-independent."
	default:
		return "Review the Structure Health rule catalog and align the package boundary with its remediation."
	}
}

func mapIssueSeverity(level string) string {
	switch strings.TrimSpace(strings.ToLower(level)) {
	case "warning", "warn":
		return "warn"
	case "info":
		return "info"
	default:
		return "error"
	}
}

func scenarioFromIssuePath(repoRoot, issuePath string) (string, bool) {
	clean := filepath.Clean(strings.TrimSpace(issuePath))
	if clean == "." || clean == "" {
		return "", false
	}
	scenariosRoot := filepath.Join(repoRoot, "scenarios")
	rel, err := filepath.Rel(scenariosRoot, clean)
	if err != nil {
		return "", false
	}
	rel = filepath.ToSlash(rel)
	if rel == "." || strings.HasPrefix(rel, "..") {
		return "", false
	}
	parts := strings.Split(rel, "/")
	if len(parts) == 0 || !isScenarioDir(parts[0]) {
		return "", false
	}
	return parts[0], true
}

func runStructureHealthProjectValidation(ctx context.Context, repoRoot string) ([]packageGovernanceIssue, error) {
	validationCtx, cancel := context.WithTimeout(ctx, structureHealthTimeout)
	defer cancel()
	baseURL, err := resolveStructureHealthURL(validationCtx, "structure-health")
	if err != nil {
		return nil, fmt.Errorf("resolve structure-health: %w", err)
	}
	if strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf("structure-health returned an empty base URL")
	}
	client := scenariovalidationconnect.NewScenarioValidationServiceClient(structureHealthHTTPClient, strings.TrimRight(baseURL, "/"))
	response, err := client.ValidateTarget(validationCtx, connect.NewRequest(&scenariovalidationv1.ValidateTargetRequest{
		Target: &commonv1.ValidationTarget{
			Kind: commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PROJECT,
			Id:   "repo",
			Root: repoRoot,
		},
		Path: repoRoot,
	}))
	if err != nil {
		if validationCtx.Err() != nil {
			return nil, fmt.Errorf("structure-health validation timed out after %s: %w", structureHealthTimeout, validationCtx.Err())
		}
		return nil, err
	}
	if response == nil || response.Msg == nil || response.Msg.GetAssessment() == nil {
		return nil, fmt.Errorf("structure-health returned no assessment")
	}
	issues := make([]packageGovernanceIssue, 0, len(response.Msg.GetAssessment().GetFindings()))
	for _, finding := range response.Msg.GetAssessment().GetFindings() {
		if finding == nil {
			continue
		}
		location := filepath.FromSlash(strings.TrimSpace(finding.GetLocation()))
		if location != "" && !filepath.IsAbs(location) {
			location = filepath.Join(repoRoot, location)
		}
		issues = append(issues, packageGovernanceIssue{
			Severity: finding.GetSeverity(),
			Code:     finding.GetCode(),
			Message:  finding.GetMessage(),
			Path:     location,
		})
	}
	return issues, nil
}
