package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const packageGovernanceRuleID = "PACKAGE_GOVERNANCE_SCENARIO_ADOPTION"

type packageAuditCLIResponse struct {
	Success bool `json:"success"`
	Audit   struct {
		Validation struct {
			Issues []packageGovernanceIssue `json:"issues"`
		} `json:"validation"`
		Issues []packageGovernanceIssue `json:"issues"`
	} `json:"audit"`
}

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

	report, stderr, err := runPackageAuditCLI(ctx, repoRoot)
	if err != nil {
		result.Findings = append(result.Findings, Finding{
			Level:   "error",
			Message: fmt.Sprintf("package governance audit failed: %v", err),
			Evidence: []Evidence{
				{Type: "command", Ref: packageAuditCommandRef()},
				{Type: "note", Detail: strings.TrimSpace(stderr)},
			},
		})
		return result
	}

	filter := strings.TrimSpace(scenarioName)
	for _, issue := range report.Audit.Issues {
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
	evidence := []Evidence{
		{Type: "command", Ref: packageAuditCommandRef()},
	}
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
		return "Replace workspace-star shared-package references with a supported isolated adoption mode such as file: or a governed Go replace directive."
	case "package-no-unauthorized-postinstall":
		return "Move shared-package hydration into vrooli-managed lifecycle/setup behavior and remove package propagation from scenario-local postinstall scripts."
	case "package-adoption-supported":
		return "Only consume packages that declare scenario adoption support for this consumer class."
	case "package-adoption-mode-valid":
		return "Use one of the adoption modes explicitly declared in the package manifest."
	default:
		return "Review the governed package manifest and align the scenario's package adoption with `vrooli package audit`."
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

func packageAuditCommandRef() string {
	return filepath.Base(packageGovernanceBinary()) + " --json --no-stale-check package audit --all"
}

func runPackageAuditCLI(ctx context.Context, repoRoot string) (packageAuditCLIResponse, string, error) {
	command := exec.CommandContext(ctx, packageGovernanceBinary(), "--json", "--no-stale-check", "package", "audit", "--all")
	command.Dir = repoRoot
	command.Env = append(os.Environ(), "VROOLI_SOURCE_ROOT="+repoRoot)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		return packageAuditCLIResponse{}, stderr.String(), err
	}

	var response packageAuditCLIResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		return packageAuditCLIResponse{}, stderr.String(), fmt.Errorf("decode package audit response: %w", err)
	}
	if !response.Success {
		return packageAuditCLIResponse{}, stderr.String(), fmt.Errorf("package audit returned success=false")
	}
	return response, stderr.String(), nil
}

func packageGovernanceBinary() string {
	if path := strings.TrimSpace(os.Getenv("VROOLI_BIN")); path != "" {
		return path
	}
	return "vrooli"
}
