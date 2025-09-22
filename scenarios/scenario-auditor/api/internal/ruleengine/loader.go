package ruleengine

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	rulespkg "scenario-auditor/rules"
)

// Options configures the rule loader/service.
type Options struct {
	RuleDirs   []string
	ModuleRoot string
}

// Loader loads rule metadata and compiles executors on demand.
type Loader struct {
	opts Options
}

// NewLoader constructs a loader with the provided options.
func NewLoader(opts Options) (*Loader, error) {
	if len(opts.RuleDirs) == 0 {
		return nil, fmt.Errorf("ruleengine: no rule directories provided")
	}
	cleaned := make([]string, 0, len(opts.RuleDirs))
	seen := make(map[string]struct{})
	for _, dir := range opts.RuleDirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		if _, ok := seen[dir]; ok {
			continue
		}
		cleaned = append(cleaned, dir)
		seen[dir] = struct{}{}
	}
	if len(cleaned) == 0 {
		return nil, fmt.Errorf("ruleengine: provided rule directories were empty")
	}
	opts.RuleDirs = cleaned
	return &Loader{opts: opts}, nil
}

// Load scans all configured rule directories, parses metadata, and compiles each rule.
func (l *Loader) Load() (map[string]Info, error) {
	rules := make(map[string]Info)

	categories := []string{"api", "cli", "config", "test", "ui", "makefile", "structure"}

	err := l.walkRuleFiles(func(info Info) error {
		exec, status := compileGoRule(&info, l.opts.ModuleRoot)
		info = info.WithExecutor(exec, status)
		rules[info.Rule.ID] = info
		return nil
	}, categories...)
	if err != nil {
		return nil, err
	}

	if len(rules) == 0 {
		return nil, fmt.Errorf("ruleengine: no rules discovered; check rule directories")
	}

	return rules, nil
}

// walkRuleFiles iterates all rule files in the configured directories and invokes fn for each discovered rule.
func (l *Loader) walkRuleFiles(fn func(Info) error, categories ...string) error {
	for _, category := range categories {
		dir := l.firstExistingDir(category)
		if dir == "" {
			continue
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || name == "types.go" {
				continue
			}

			filePath := filepath.Join(dir, name)
			info, err := extractRuleMetadata(filePath, category)
			if err != nil {
				// Skip unreadable rule but continue scanning others
				continue
			}

			if err := fn(info); err != nil {
				return err
			}
		}
	}
	return nil
}

func (l *Loader) firstExistingDir(category string) string {
	for _, root := range l.opts.RuleDirs {
		candidate := filepath.Join(root, category)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return ""
}

func extractRuleMetadata(filePath string, defaultCategory string) (Info, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return Info{}, err
	}
	defer file.Close()

	info := Info{
		Rule: rulespkg.Rule{
			ID:       strings.TrimSuffix(filepath.Base(filePath), ".go"),
			Category: defaultCategory,
			Enabled:  true,
		},
		FilePath: filePath,
		Targets:  []string{},
	}

	scanner := bufio.NewScanner(file)
	inComment := false
	commentLines := []string{}

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "/*") {
			inComment = true
			continue
		}

		if inComment {
			if strings.HasPrefix(strings.TrimSpace(line), "*/") {
				inComment = false
				info = parseCommentBlock(commentLines, info)
				break
			}
			commentLines = append(commentLines, line)
		}

		if !inComment && len(commentLines) > 0 {
			break
		}
	}

	// If comment block didn't set an ID (rare), default to filename
	if strings.TrimSpace(info.Rule.ID) == "" {
		info.Rule.ID = strings.TrimSuffix(filepath.Base(filePath), ".go")
	}

	return info, nil
}

func parseCommentBlock(lines []string, info Info) Info {
	for _, line := range lines {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "Rule:"):
			info.Rule.Name = strings.TrimSpace(strings.TrimPrefix(line, "Rule:"))
		case strings.HasPrefix(line, "Description:"):
			info.Rule.Description = strings.TrimSpace(strings.TrimPrefix(line, "Description:"))
		case strings.HasPrefix(line, "Reason:"):
			info.Reason = strings.TrimSpace(strings.TrimPrefix(line, "Reason:"))
		case strings.HasPrefix(line, "Category:"):
			category := strings.TrimSpace(strings.TrimPrefix(line, "Category:"))
			if category != "" {
				info.Rule.Category = category
			}
		case strings.HasPrefix(line, "Severity:"):
			severity := strings.TrimSpace(strings.TrimPrefix(line, "Severity:"))
			if severity != "" {
				info.Rule.Severity = severity
			}
		case strings.HasPrefix(line, "Standard:"):
			info.Rule.Standard = strings.TrimSpace(strings.TrimPrefix(line, "Standard:"))
		case strings.HasPrefix(line, "Targets:"):
			targets := strings.Split(strings.TrimSpace(strings.TrimPrefix(line, "Targets:")), ",")
			info.Targets = info.Targets[:0]
			for _, target := range targets {
				t := strings.ToLower(strings.TrimSpace(target))
				if t != "" {
					info.Targets = append(info.Targets, t)
				}
			}
		}
	}

	return info
}

// BuildExecutionInfo documents the invocation contract for a rule's Check function.
func BuildExecutionInfo(rule Info) ExecutionInfo {
	return ExecutionInfo{
		Signature: "Check(content []byte, filepath string, scenario string) ([]Violation, error)",
		Arguments: []ArgumentInfo{
			{
				Name:        "content",
				Type:        "[]byte",
				Description: "Raw source being evaluated (test snippets or scenario file contents).",
			},
			{
				Name:        "filepath",
				Type:        "string",
				Description: "Path hint for the content. Tests use synthetic names; scans pass relative paths.",
			},
			{
				Name:        "scenario",
				Type:        "string",
				Description: "Scenario identifier when available (may be empty for ad-hoc tests).",
			},
		},
		CallFlow: []ExecutionCall{
			{
				Source:      "Info.Check",
				Description: "Dispatches to a dynamically loaded rule executor and forwards arguments.",
				Reference:   "internal/ruleengine/types.go",
			},
			{
				Source:      "TestHarness.RunTest",
				Description: "Executes <test-case> snippets via the same Check entrypoint.",
				Reference:   "internal/ruleengine/testrunner.go",
			},
		},
		Notes: []string{
			"Rule implementations are interpreted at runtime; no manual registration required.",
			"Rules receive plain text and should perform their own parsing.",
			"Return either []rules.Violation or an error; nil means the content passed.",
		},
	}
}

// DiscoverRuleDirs attempts to find rule directories under the provided repository root.
func DiscoverRuleDirs(repoRoot string) ([]string, error) {
	candidates := []string{
		filepath.Join(repoRoot, "scenarios", "scenario-auditor", "api", "rules"),
		filepath.Join(repoRoot, "scenarios", "scenario-auditor", "rules"),
	}

	dirs := make([]string, 0, len(candidates))
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			dirs = append(dirs, c)
		}
	}

	if len(dirs) == 0 {
		return nil, fmt.Errorf("ruleengine: no rule directories found under %s", repoRoot)
	}

	sort.Strings(dirs)
	return dirs, nil
}

// DiscoverRepoRoot walks up from the provided starting points to locate the repository root containing scenarios/scenario-auditor.
func DiscoverRepoRoot(startPoints ...string) (string, error) {
	queue := []string{}
	seen := make(map[string]struct{})

	for _, p := range startPoints {
		if p == "" {
			continue
		}
		queue = append(queue, p)
	}

	if wd, err := os.Getwd(); err == nil {
		queue = append(queue, wd)
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if _, ok := seen[current]; ok {
			continue
		}
		seen[current] = struct{}{}

		probe := filepath.Join(current, "scenarios", "scenario-auditor")
		if info, err := os.Stat(probe); err == nil && info.IsDir() {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent != current {
			queue = append(queue, parent)
		}
	}

	if home := strings.TrimSpace(os.Getenv("HOME")); home != "" {
		fallback := filepath.Join(home, "Vrooli")
		if info, err := os.Stat(fallback); err == nil && info.IsDir() {
			return fallback, nil
		}
	}

	return "", fmt.Errorf("ruleengine: unable to locate Vrooli root; set VROOLI_ROOT")
}

// RuleFiles returns a list of rule files for tooling/CLI usage.
func (l *Loader) RuleFiles() ([]string, error) {
	files := []string{}
	err := l.walkRuleFiles(func(info Info) error {
		files = append(files, info.FilePath)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

// Walk walks all rule files with a custom handler.
func (l *Loader) Walk(fn func(Info) error) error {
	return l.walkRuleFiles(fn)
}
