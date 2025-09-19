package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"
)

const (
	maxIssuesForPrompt = 12
)

var (
	fixInstructionsOnce sync.Once
	fixInstructions     string
	fixInstructionsErr  error
)

type standardsFixPromptData struct {
	Scenario             string
	ScenarioPath         string
	TotalViolations      int
	ListedViolations     []StandardsViolation
	AdditionalViolations int
	SeverityCounts       map[string]int
	IssueIDs             []string
	Instructions         string
	Timestamp            string
}

type vulnerabilityFixPromptData struct {
	Scenario           string
	ScenarioPath       string
	TotalFindings      int
	ListedFindings     []StoredVulnerability
	AdditionalFindings int
	SeverityCounts     map[string]int
	IssueIDs           []string
	Instructions       string
	Timestamp          string
}

func loadFixInstructions() (string, error) {
	fixInstructionsOnce.Do(func() {
		path := filepath.Join(getScenarioRoot(), "prompts", "fix-generation.txt")
		data, err := os.ReadFile(path)
		if err != nil {
			fixInstructionsErr = err
			return
		}
		fixInstructions = string(data)
	})
	return fixInstructions, fixInstructionsErr
}

func buildStandardsFixPrompt(scenarioName, scenarioPath string, violations []StandardsViolation, issueIDs []string) (string, string, map[string]string, error) {
	if len(violations) == 0 {
		return "", "", nil, fmt.Errorf("no standards violations provided for prompt generation")
	}

	instructions, err := loadFixInstructions()
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to load fix instructions: %w", err)
	}

	listed := violations
	additional := 0
	if len(violations) > maxIssuesForPrompt {
		listed = violations[:maxIssuesForPrompt]
		additional = len(violations) - maxIssuesForPrompt
	}

	severityCounts := map[string]int{
		"critical": 0,
		"high":     0,
		"medium":   0,
		"low":      0,
	}
	for _, v := range violations {
		key := strings.ToLower(v.Severity)
		if _, ok := severityCounts[key]; ok {
			severityCounts[key]++
		}
	}

	ids := issueIDs
	if len(ids) == 0 {
		ids = make([]string, 0, len(violations))
		for _, v := range violations {
			if v.ID != "" {
				ids = append(ids, v.ID)
			}
		}
	}

	data := standardsFixPromptData{
		Scenario:             scenarioName,
		ScenarioPath:         scenarioPath,
		TotalViolations:      len(violations),
		ListedViolations:     listed,
		AdditionalViolations: additional,
		SeverityCounts:       severityCounts,
		IssueIDs:             ids,
		Instructions:         instructions,
		Timestamp:            time.Now().Format(time.RFC3339),
	}

	templatePath := filepath.Join(getScenarioRoot(), "prompts", "standards-fix.tmpl")
	tplBytes, err := os.ReadFile(templatePath)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to load standards fix template: %w", err)
	}

	tpl, err := template.New("standards-fix").Parse(string(tplBytes))
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to parse standards fix template: %w", err)
	}

	var buffer bytes.Buffer
	if err := tpl.Execute(&buffer, data); err != nil {
		return "", "", nil, fmt.Errorf("failed to render standards fix prompt: %w", err)
	}

	metadata := map[string]string{
		"scenario":      scenarioName,
		"scenario_path": scenarioPath,
		"issue_count":   strconv.Itoa(len(violations)),
		"issue_kind":    "standards",
	}
	if len(ids) > 0 {
		metadata["issue_ids"] = strings.Join(ids, ",")
	}
	if additional > 0 {
		metadata["issue_list_truncated"] = strconv.Itoa(additional)
	}

	return buffer.String(), "Fix Standards Violations", metadata, nil
}

func buildVulnerabilityFixPrompt(scenarioName, scenarioPath string, findings []StoredVulnerability, issueIDs []string) (string, string, map[string]string, error) {
	if len(findings) == 0 {
		return "", "", nil, fmt.Errorf("no vulnerabilities provided for prompt generation")
	}

	instructions, err := loadFixInstructions()
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to load fix instructions: %w", err)
	}

	sorted := make([]StoredVulnerability, len(findings))
	copy(sorted, findings)
	sort.Slice(sorted, func(i, j int) bool {
		sevI := strings.ToLower(sorted[i].Severity)
		sevJ := strings.ToLower(sorted[j].Severity)
		severityOrder := map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3}
		if severityOrder[sevI] == severityOrder[sevJ] {
			return sorted[i].Title < sorted[j].Title
		}
		return severityOrder[sevI] < severityOrder[sevJ]
	})

	listed := sorted
	additional := 0
	if len(sorted) > maxIssuesForPrompt {
		listed = sorted[:maxIssuesForPrompt]
		additional = len(sorted) - maxIssuesForPrompt
	}

	severityCounts := map[string]int{
		"critical": 0,
		"high":     0,
		"medium":   0,
		"low":      0,
	}
	for _, v := range sorted {
		key := strings.ToLower(v.Severity)
		if _, ok := severityCounts[key]; ok {
			severityCounts[key]++
		}
	}

	ids := issueIDs
	if len(ids) == 0 {
		ids = make([]string, 0, len(sorted))
		for _, v := range sorted {
			if v.ID != "" {
				ids = append(ids, v.ID)
			}
		}
	}

	data := vulnerabilityFixPromptData{
		Scenario:           scenarioName,
		ScenarioPath:       scenarioPath,
		TotalFindings:      len(sorted),
		ListedFindings:     listed,
		AdditionalFindings: additional,
		SeverityCounts:     severityCounts,
		IssueIDs:           ids,
		Instructions:       instructions,
		Timestamp:          time.Now().Format(time.RFC3339),
	}

	templatePath := filepath.Join(getScenarioRoot(), "prompts", "vulnerability-fix.tmpl")
	tplBytes, err := os.ReadFile(templatePath)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to load vulnerability fix template: %w", err)
	}

	tpl, err := template.New("vulnerability-fix").Parse(string(tplBytes))
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to parse vulnerability fix template: %w", err)
	}

	var buffer bytes.Buffer
	if err := tpl.Execute(&buffer, data); err != nil {
		return "", "", nil, fmt.Errorf("failed to render vulnerability fix prompt: %w", err)
	}

	metadata := map[string]string{
		"scenario":      scenarioName,
		"scenario_path": scenarioPath,
		"issue_count":   strconv.Itoa(len(sorted)),
		"issue_kind":    "vulnerabilities",
	}
	if len(ids) > 0 {
		metadata["issue_ids"] = strings.Join(ids, ",")
	}
	if additional > 0 {
		metadata["issue_list_truncated"] = strconv.Itoa(additional)
	}

	return buffer.String(), "Fix Vulnerabilities", metadata, nil
}
