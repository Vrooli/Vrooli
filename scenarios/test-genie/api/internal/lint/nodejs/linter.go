package nodejs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"test-genie/internal/shared"
)

// Severity indicates the severity of a lint issue.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Issue represents a single lint or type error finding.
type Issue struct {
	File     string   `json:"file"`
	Line     int      `json:"line"`
	Column   int      `json:"column,omitempty"`
	Message  string   `json:"message"`
	Severity Severity `json:"severity"`
	Rule     string   `json:"rule,omitempty"`
	Source   string   `json:"source"`
}

// Result holds the result of linting Node.js code.
type Result struct {
	Language     string               `json:"language"`
	Success      bool                 `json:"success"`
	Issues       []Issue              `json:"issues,omitempty"`
	TypeErrors   int                  `json:"typeErrors"`
	LintWarnings int                  `json:"lintWarnings"`
	ToolsUsed    []string             `json:"toolsUsed"`
	Skipped      bool                 `json:"skipped"`
	SkipReason   string               `json:"skipReason,omitempty"`
	Observations []shared.Observation `json:"observations,omitempty"`
}

// Config holds configuration for Node.js linting.
type Config struct {
	Dir           string
	CommandLookup shared.LookupFunc
}

// Linter performs Node.js linting.
type Linter struct {
	config    Config
	logWriter io.Writer
}

// Option configures a Linter.
type Option func(*Linter)

// New creates a new Node.js linter.
func New(config Config, opts ...Option) *Linter {
	l := &Linter{
		config:    config,
		logWriter: io.Discard,
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// WithLogger sets the log writer.
func WithLogger(w io.Writer) Option {
	return func(l *Linter) {
		l.logWriter = w
	}
}

// Lint runs Node.js linting and returns the result.
func (l *Linter) Lint(ctx context.Context) *Result {
	result := &Result{
		Language: "nodejs",
	}

	// Check if directory exists
	if _, err := os.Stat(l.config.Dir); os.IsNotExist(err) {
		result.Skipped = true
		result.SkipReason = "ui/ directory not found"
		result.Success = true
		result.Observations = append(result.Observations,
			shared.NewSkipObservation("Node: ui/ directory not found"))
		return result
	}

	// Check if package.json exists
	packagePath := filepath.Join(l.config.Dir, "package.json")
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		result.Skipped = true
		result.SkipReason = "no package.json found"
		result.Success = true
		result.Observations = append(result.Observations,
			shared.NewSkipObservation("Node: no package.json found in ui/"))
		return result
	}

	var allIssues []Issue

	// Run TypeScript type checking if tsconfig.json exists
	if l.hasTsConfig() {
		tscResult := l.runTsc(ctx)
		allIssues = append(allIssues, tscResult.Issues...)
		result.TypeErrors += tscResult.TypeErrors
		result.ToolsUsed = append(result.ToolsUsed, tscResult.ToolsUsed...)
		result.Observations = append(result.Observations, tscResult.Observations...)
	}

	// Run ESLint if config exists
	if l.hasEslintConfig() {
		eslintResult := l.runEslint(ctx)
		allIssues = append(allIssues, eslintResult.Issues...)
		result.LintWarnings += eslintResult.LintWarnings
		result.ToolsUsed = append(result.ToolsUsed, eslintResult.ToolsUsed...)
		result.Observations = append(result.Observations, eslintResult.Observations...)
	}

	// If no tools were run
	if len(result.ToolsUsed) == 0 {
		result.Skipped = true
		result.SkipReason = "no TypeScript or ESLint configuration found"
		result.Success = true
		result.Observations = append(result.Observations,
			shared.NewSkipObservation("Node: no tsconfig.json or ESLint config found"))
		return result
	}

	result.Issues = allIssues
	result.Success = result.TypeErrors == 0 // Type errors fail, lint warnings don't

	return result
}

func (l *Linter) hasTsConfig() bool {
	tsConfigPath := filepath.Join(l.config.Dir, "tsconfig.json")
	_, err := os.Stat(tsConfigPath)
	return err == nil
}

func (l *Linter) hasEslintConfig() bool {
	// Check for various ESLint config files
	configs := []string{
		".eslintrc",
		".eslintrc.js",
		".eslintrc.cjs",
		".eslintrc.json",
		".eslintrc.yml",
		".eslintrc.yaml",
		"eslint.config.js",
		"eslint.config.mjs",
		"eslint.config.cjs",
	}
	for _, config := range configs {
		if _, err := os.Stat(filepath.Join(l.config.Dir, config)); err == nil {
			return true
		}
	}
	return false
}

func (l *Linter) hasCommand(name string) bool {
	if l.config.CommandLookup != nil {
		_, err := l.config.CommandLookup(name)
		return err == nil
	}
	_, err := exec.LookPath(name)
	return err == nil
}

func (l *Linter) runTsc(ctx context.Context) *Result {
	result := &Result{
		Language:  "nodejs",
		ToolsUsed: []string{"tsc"},
	}

	// Find tsc - prefer local node_modules
	tscPath := l.findTsc()
	if tscPath == "" {
		result.Observations = append(result.Observations,
			shared.NewWarningObservation("Node: tsc not found, skipping type checking"))
		return result
	}

	cmd := exec.CommandContext(ctx, tscPath, "--noEmit")
	cmd.Dir = l.config.Dir

	output, err := cmd.CombinedOutput()

	if err != nil {
		// tsc exits with non-zero if there are errors
		issues := l.parseTscOutput(string(output))
		result.Issues = issues
		result.TypeErrors = len(issues)

		if len(issues) > 0 {
			result.Observations = append(result.Observations,
				shared.NewErrorObservation(fmt.Sprintf("Node: tsc found %d type error(s)", len(issues))))
		} else {
			result.Observations = append(result.Observations,
				shared.NewErrorObservation(fmt.Sprintf("tsc failed: %v", err)))
		}
		return result
	}

	result.Success = true
	result.Observations = append(result.Observations,
		shared.NewSuccessObservation("Node: tsc found no type errors"))
	return result
}

func (l *Linter) findTsc() string {
	// Try local node_modules first
	localTsc := filepath.Join(l.config.Dir, "node_modules", ".bin", "tsc")
	if _, err := os.Stat(localTsc); err == nil {
		return localTsc
	}
	// Try global
	if l.hasCommand("tsc") {
		return "tsc"
	}
	return ""
}

func (l *Linter) parseTscOutput(output string) []Issue {
	var issues []Issue
	// tsc output format: file(line,col): error TSxxxx: message
	re := regexp.MustCompile(`^(.+)\((\d+),(\d+)\):\s+error\s+(TS\d+):\s+(.+)$`)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		matches := re.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		lineNum, _ := strconv.Atoi(matches[2])
		colNum, _ := strconv.Atoi(matches[3])

		issues = append(issues, Issue{
			File:     matches[1],
			Line:     lineNum,
			Column:   colNum,
			Message:  matches[5],
			Severity: SeverityError,
			Rule:     matches[4],
			Source:   "tsc",
		})
	}

	return issues
}

func (l *Linter) runEslint(ctx context.Context) *Result {
	result := &Result{
		Language:  "nodejs",
		ToolsUsed: []string{"eslint"},
	}

	// Find eslint - prefer local node_modules
	eslintPath := l.findEslint()
	if eslintPath == "" {
		result.Observations = append(result.Observations,
			shared.NewWarningObservation("Node: eslint not found, skipping linting"))
		return result
	}

	cmd := exec.CommandContext(ctx, eslintPath, ".", "--format", "json")
	cmd.Dir = l.config.Dir

	output, err := cmd.Output()

	// eslint exits with 1 if there are issues
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				// Issues found, parse them
				if len(output) == 0 {
					output = exitErr.Stderr
				}
			} else if exitErr.ExitCode() == 2 {
				// Configuration error
				result.Observations = append(result.Observations,
					shared.NewErrorObservation(fmt.Sprintf("ESLint configuration error: %s", string(exitErr.Stderr))))
				return result
			}
		}
	}

	issues := l.parseEslintOutput(output)
	result.Issues = issues
	result.LintWarnings = len(issues)
	result.Success = true // ESLint issues are warnings, not failures

	if len(issues) == 0 {
		result.Observations = append(result.Observations,
			shared.NewSuccessObservation("Node: eslint found no issues"))
	} else {
		result.Observations = append(result.Observations,
			shared.NewWarningObservation(fmt.Sprintf("Node: eslint found %d issue(s)", len(issues))))
	}

	return result
}

func (l *Linter) findEslint() string {
	// Try local node_modules first
	localEslint := filepath.Join(l.config.Dir, "node_modules", ".bin", "eslint")
	if _, err := os.Stat(localEslint); err == nil {
		return localEslint
	}
	// Try global
	if l.hasCommand("eslint") {
		return "eslint"
	}
	return ""
}

type eslintOutput []eslintFileResult

type eslintFileResult struct {
	FilePath string          `json:"filePath"`
	Messages []eslintMessage `json:"messages"`
}

type eslintMessage struct {
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Message  string `json:"message"`
	RuleId   string `json:"ruleId"`
	Severity int    `json:"severity"` // 1 = warning, 2 = error
}

func (l *Linter) parseEslintOutput(output []byte) []Issue {
	var parsed eslintOutput
	if err := json.Unmarshal(output, &parsed); err != nil {
		shared.LogWarn(l.logWriter, "failed to parse eslint JSON: %v", err)
		return nil
	}

	var issues []Issue
	for _, file := range parsed {
		for _, msg := range file.Messages {
			severity := SeverityWarning
			if msg.Severity == 2 {
				severity = SeverityError
			}
			issues = append(issues, Issue{
				File:     file.FilePath,
				Line:     msg.Line,
				Column:   msg.Column,
				Message:  msg.Message,
				Severity: severity,
				Rule:     msg.RuleId,
				Source:   "eslint",
			})
		}
	}

	return issues
}
