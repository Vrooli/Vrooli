package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// Violation represents a rule violation
type Violation struct {
	RuleID         string `json:"rule_id"`
	Severity       string `json:"severity"`
	Title          string `json:"title,omitempty"`
	Message        string `json:"message"`
	File           string `json:"file,omitempty"`
	FilePath       string `json:"file_path,omitempty"`
	Line           int    `json:"line,omitempty"`
	CodeSnippet    string `json:"code_snippet,omitempty"`
	Recommendation string `json:"recommendation,omitempty"`
	Standard       string `json:"standard,omitempty"`
	Category       string `json:"category,omitempty"`
}

// RuleInfo represents metadata extracted from a rule file
type RuleInfo struct {
	ID             string                   `json:"id"`
	Name           string                   `json:"name"`
	Description    string                   `json:"description"`
	Reason         string                   `json:"reason"`
	Category       string                   `json:"category"`
	Severity       string                   `json:"severity"`
	Standard       string                   `json:"standard"`
	FilePath       string                   `json:"file_path"`
	Enabled        bool                     `json:"enabled"`
	Targets        []string                 `json:"targets"`
	Implementation RuleImplementationStatus `json:"implementation"`
	executor       ruleExecutor             `json:"-"`
}

type RuleExecutionInfo struct {
	Signature string              `json:"signature"`
	Arguments []RuleArgumentInfo  `json:"arguments"`
	CallFlow  []RuleExecutionCall `json:"call_flow"`
	Notes     []string            `json:"notes"`
}

type RuleArgumentInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

type RuleExecutionCall struct {
	Source      string `json:"source"`
	Description string `json:"description"`
	Reference   string `json:"reference"`
}

// Check executes the underlying rule implementation if one is available.
func (r RuleInfo) Check(content string, filepath string, scenario string) ([]Violation, error) {
	if r.executor == nil {
		if !r.Implementation.Valid {
			return nil, fmt.Errorf("rule implementation unavailable: %s", r.Implementation.Error)
		}
		return nil, fmt.Errorf("rule execution not configured for %s", r.ID)
	}

	return r.executor.Execute(content, filepath, scenario)
}

// LoadRulesFromFiles scans the rules directory and extracts rule metadata
func LoadRulesFromFiles() (map[string]RuleInfo, error) {
	rules := make(map[string]RuleInfo)

	// Locate the Vrooli repository root so we can find rule directories even when
	// the service is launched from a different working directory or with a
	// different HOME (a common occurrence inside managed runtimes).
	vrooliRoot, err := resolveVrooliRoot()
	if err != nil {
		return nil, err
	}

	apiRulesDir := filepath.Join(vrooliRoot, "scenarios", "scenario-auditor", "api", "rules")
	legacyRulesDir := filepath.Join(vrooliRoot, "scenarios", "scenario-auditor", "rules")

	var rulesDirs []string
	for _, candidate := range []string{apiRulesDir, legacyRulesDir} {
		if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
			rulesDirs = append(rulesDirs, candidate)
		}
	}

	if len(rulesDirs) == 0 {
		return nil, fmt.Errorf("no rule directories found under %s", vrooliRoot)
	}

	// Walk through all subdirectories
	categories := []string{"api", "cli", "config", "test", "ui", "makefile", "structure"}

	for _, category := range categories {
		var categoryDir string
		for _, base := range rulesDirs {
			dir := filepath.Join(base, category)
			if info, err := os.Stat(dir); err == nil && info.IsDir() {
				categoryDir = dir
				break
			}
		}

		if categoryDir == "" {
			continue
		}

		// Read all rule files in the category
		files, err := ioutil.ReadDir(categoryDir)
		if err != nil {
			continue
		}

		for _, file := range files {
			name := file.Name()
			if strings.HasSuffix(name, "_test.go") {
				continue
			}

			ext := filepath.Ext(name)
			if ext != ".go" {
				continue
			}
			if name == "types.go" {
				continue
			}

			filePath := filepath.Join(categoryDir, name)
			rule, err := extractRuleMetadata(filePath, category)
			if err != nil {
				continue
			}

			executor, status := compileGoRule(&rule)
			rule.executor = executor
			rule.Implementation = status

			if rule.ID != "" {
				rules[rule.ID] = rule
			}
		}
	}

	return rules, nil
}

// extractRuleMetadata parses a rule file and extracts metadata from comments
func extractRuleMetadata(filePath string, defaultCategory string) (RuleInfo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return RuleInfo{}, err
	}
	defer file.Close()

	rule := RuleInfo{
		Category: defaultCategory,
		FilePath: filePath,
		Enabled:  true,
	}

	scanner := bufio.NewScanner(file)
	inComment := false
	commentLines := []string{}

	// Read the file to find the metadata comment block
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "/*") {
			inComment = true
			continue
		}

		if inComment {
			if strings.HasPrefix(line, "*/") {
				inComment = false
				// Parse the collected comment lines
				rule = parseCommentBlock(commentLines, rule)
				break
			}
			commentLines = append(commentLines, line)
		}

		// Stop after we've processed the first comment block
		if !inComment && len(commentLines) > 0 {
			break
		}
	}

	// Generate ID from filename if not set
	if rule.ID == "" {
		baseName := strings.TrimSuffix(filepath.Base(filePath), ".go")
		rule.ID = baseName
	}

	return rule, nil
}

// parseCommentBlock parses the metadata from comment lines
func parseCommentBlock(lines []string, rule RuleInfo) RuleInfo {
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse key: value pairs
		if strings.HasPrefix(line, "Rule:") {
			rule.Name = strings.TrimSpace(strings.TrimPrefix(line, "Rule:"))
		} else if strings.HasPrefix(line, "Description:") {
			rule.Description = strings.TrimSpace(strings.TrimPrefix(line, "Description:"))
		} else if strings.HasPrefix(line, "Reason:") {
			rule.Reason = strings.TrimSpace(strings.TrimPrefix(line, "Reason:"))
		} else if strings.HasPrefix(line, "Category:") {
			rule.Category = strings.TrimSpace(strings.TrimPrefix(line, "Category:"))
		} else if strings.HasPrefix(line, "Severity:") {
			rule.Severity = strings.TrimSpace(strings.TrimPrefix(line, "Severity:"))
		} else if strings.HasPrefix(line, "Standard:") {
			rule.Standard = strings.TrimSpace(strings.TrimPrefix(line, "Standard:"))
		} else if strings.HasPrefix(line, "Targets:") {
			targets := strings.Split(strings.TrimSpace(strings.TrimPrefix(line, "Targets:")), ",")
			for _, target := range targets {
				t := strings.TrimSpace(target)
				if t != "" {
					rule.Targets = append(rule.Targets, strings.ToLower(t))
				}
			}
		}
	}

	return rule
}

// GetRuleCategories returns all available rule categories
func GetRuleCategories() map[string]RuleCategory {
	return map[string]RuleCategory{
		"api": {
			ID:          "api",
			Name:        "API Standards",
			Description: "Rules for API design, implementation, and resource management",
		},
		"cli": {
			ID:          "cli",
			Name:        "CLI & Structure",
			Description: "Rules for command-line interfaces and code structure",
		},
		"config": {
			ID:          "config",
			Name:        "Configuration",
			Description: "Rules for configuration management and environment variables",
		},
		"test": {
			ID:          "test",
			Name:        "Testing",
			Description: "Rules for test coverage and quality",
		},
		"ui": {
			ID:          "ui",
			Name:        "User Interface",
			Description: "Rules for UI components and frontend code",
		},
	}
}

// resolveVrooliRoot attempts to discover the repository root so rule files can
// be located regardless of the process working directory.
func resolveVrooliRoot() (string, error) {
	candidates := []string{}
	for _, env := range []string{"VROOLI_ROOT", "APP_ROOT"} {
		if val := strings.TrimSpace(os.Getenv(env)); val != "" {
			candidates = append(candidates, val)
		}
	}

	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, wd)
	}

	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Dir(exe))
	}

	seen := make(map[string]struct{})
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}

		path := candidate
		for {
			if _, alreadyChecked := seen[path]; alreadyChecked {
				break
			}
			seen[path] = struct{}{}

			probe := filepath.Join(path, "scenarios", "scenario-auditor")
			if info, err := os.Stat(probe); err == nil && info.IsDir() {
				return path, nil
			}

			parent := filepath.Dir(path)
			if parent == path {
				break
			}
			path = parent
		}
	}

	if home := strings.TrimSpace(os.Getenv("HOME")); home != "" {
		fallback := filepath.Join(home, "Vrooli")
		if info, err := os.Stat(fallback); err == nil && info.IsDir() {
			return fallback, nil
		}
	}

	return "", fmt.Errorf("unable to locate Vrooli root; set VROOLI_ROOT")
}

func buildRuleExecutionInfo(rule RuleInfo) RuleExecutionInfo {
	_ = rule // reserved for future rule-specific detail

	return RuleExecutionInfo{
		Signature: "Check(content string, filepath string, scenario string) ([]Violation, error)",
		Arguments: []RuleArgumentInfo{
			{
				Name:        "content",
				Type:        "string",
				Description: "Raw source being evaluated. For embedded tests and the playground this is exactly the `<input>` snippet; for repository scans it is the file contents read from disk.",
			},
			{
				Name:        "filepath",
				Type:        "string",
				Description: "Path hint for the content. Test executions synthesize names like `test_<case>.go`, while repository scans pass the relative file path.",
			},
			{
				Name:        "scenario",
				Type:        "string",
				Description: "Scenario identifier when the rule runs as part of a scan. Rules can use this to tailor validation or messages per scenario; it is empty for ad-hoc playground runs.",
			},
		},
		CallFlow: []RuleExecutionCall{
			{
				Source:      "RuleInfo.Check",
				Description: "Dispatches to a dynamically loaded rule executor and forwards `content`/`filepath`.",
				Reference:   "scenarios/scenario-auditor/api/rule_loader.go:43",
			},
			{
				Source:      "TestRunner.RunTest",
				Description: "Executes `<test-case>` snippets and invokes the rule with the snippet text and a synthetic filename.",
				Reference:   "scenarios/scenario-auditor/api/test_runner.go:247",
			},
			{
				Source:      "TestRunner.runTestDirect",
				Description: "Fallback path when Judge0 is unavailable; still dispatches to the same `Check` signature.",
				Reference:   "scenarios/scenario-auditor/api/test_runner.go:599",
			},
		},
		Notes: []string{
			"Go rule implementations are interpreted at runtime, eliminating manual registry updates.",
			"Rules receive plain text and should perform their own parsing or scanning without relying on additional context.",
			"The auditor merges runtime violations when available, but the static `Check` output alone must accurately describe issues.",
		},
	}
}

// ConvertRuleInfoToRule converts RuleInfo to the Rule struct used by the API
func ConvertRuleInfoToRule(info RuleInfo) Rule {
	return Rule{
		ID:             info.ID,
		Name:           info.Name,
		Description:    info.Description,
		Category:       info.Category,
		Severity:       info.Severity,
		Enabled:        info.Enabled,
		Standard:       info.Standard,
		Targets:        info.Targets,
		Implementation: info.Implementation,
	}
}
