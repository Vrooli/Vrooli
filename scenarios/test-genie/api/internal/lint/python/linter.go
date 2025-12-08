package python

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

// Result holds the result of linting Python code.
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

// Config holds configuration for Python linting.
type Config struct {
	Dir           string
	CommandLookup shared.LookupFunc
}

// Linter performs Python linting.
type Linter struct {
	config    Config
	logWriter io.Writer
}

// Option configures a Linter.
type Option func(*Linter)

// New creates a new Python linter.
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

// Lint runs Python linting and returns the result.
func (l *Linter) Lint(ctx context.Context) *Result {
	result := &Result{
		Language: "python",
	}

	// Check if there are Python files
	if !l.hasPythonFiles() {
		result.Skipped = true
		result.SkipReason = "no Python files found"
		result.Success = true
		result.Observations = append(result.Observations,
			shared.NewSkipObservation("Python: no .py files found"))
		return result
	}

	var allIssues []Issue

	// Run linting (ruff or flake8)
	lintResult := l.runLinter(ctx)
	allIssues = append(allIssues, lintResult.Issues...)
	result.LintWarnings += lintResult.LintWarnings
	result.ToolsUsed = append(result.ToolsUsed, lintResult.ToolsUsed...)
	result.Observations = append(result.Observations, lintResult.Observations...)

	// Run type checking if mypy is configured
	if l.hasMypyConfig() {
		mypyResult := l.runMypy(ctx)
		allIssues = append(allIssues, mypyResult.Issues...)
		result.TypeErrors += mypyResult.TypeErrors
		result.ToolsUsed = append(result.ToolsUsed, mypyResult.ToolsUsed...)
		result.Observations = append(result.Observations, mypyResult.Observations...)
	}

	// If no tools were run
	if len(result.ToolsUsed) == 0 {
		result.Skipped = true
		result.SkipReason = "no Python linting tools available"
		result.Success = true
		result.Observations = append(result.Observations,
			shared.NewSkipObservation("Python: no linting tools available (install ruff, flake8, or mypy)"))
		return result
	}

	result.Issues = allIssues
	result.Success = result.TypeErrors == 0 // Type errors fail, lint warnings don't

	return result
}

func (l *Linter) hasPythonFiles() bool {
	found := false
	filepath.WalkDir(l.config.Dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			// Skip common non-source directories
			name := d.Name()
			if name == "node_modules" || name == ".git" || name == "__pycache__" || name == ".venv" || name == "venv" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".py" {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func (l *Linter) hasMypyConfig() bool {
	// Check for mypy.ini
	if _, err := os.Stat(filepath.Join(l.config.Dir, "mypy.ini")); err == nil {
		return true
	}
	// Check pyproject.toml for [tool.mypy] section
	pyprojectPath := filepath.Join(l.config.Dir, "pyproject.toml")
	if data, err := os.ReadFile(pyprojectPath); err == nil {
		if strings.Contains(string(data), "[tool.mypy]") {
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

func (l *Linter) runLinter(ctx context.Context) *Result {
	// Try ruff first (faster, modern)
	if l.hasCommand("ruff") {
		return l.runRuff(ctx)
	}

	// Fallback to flake8
	if l.hasCommand("flake8") {
		shared.LogWarn(l.logWriter, "ruff not found, falling back to flake8")
		return l.runFlake8(ctx)
	}

	// No linter available
	result := &Result{Language: "python"}
	result.Observations = append(result.Observations,
		shared.NewWarningObservation("Python: no linter available (install ruff or flake8)"))
	return result
}

func (l *Linter) runRuff(ctx context.Context) *Result {
	result := &Result{
		Language:  "python",
		ToolsUsed: []string{"ruff"},
	}

	cmd := exec.CommandContext(ctx, "ruff", "check", "--output-format", "json", ".")
	cmd.Dir = l.config.Dir

	output, err := cmd.Output()

	// ruff exits with 1 if there are issues
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			if len(output) == 0 {
				output = exitErr.Stderr
			}
		} else {
			result.Observations = append(result.Observations,
				shared.NewErrorObservation(fmt.Sprintf("ruff failed: %v", err)))
			return result
		}
	}

	issues := l.parseRuffOutput(output)
	result.Issues = issues
	result.LintWarnings = len(issues)
	result.Success = true

	if len(issues) == 0 {
		result.Observations = append(result.Observations,
			shared.NewSuccessObservation("Python: ruff found no issues"))
	} else {
		logPythonIssues(l.logWriter, issues, 20)
		result.Observations = append(result.Observations,
			shared.NewWarningObservation(fmt.Sprintf("Python: ruff found %d issue(s)", len(issues))))
	}

	return result
}

type ruffIssue struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Filename string `json:"filename"`
	Location struct {
		Row    int `json:"row"`
		Column int `json:"column"`
	} `json:"location"`
}

func (l *Linter) parseRuffOutput(output []byte) []Issue {
	var parsed []ruffIssue
	if err := json.Unmarshal(output, &parsed); err != nil {
		shared.LogWarn(l.logWriter, "failed to parse ruff JSON: %v", err)
		return nil
	}

	var issues []Issue
	for _, i := range parsed {
		issues = append(issues, Issue{
			File:     i.Filename,
			Line:     i.Location.Row,
			Column:   i.Location.Column,
			Message:  i.Message,
			Severity: SeverityWarning,
			Rule:     i.Code,
			Source:   "ruff",
		})
	}
	return issues
}

func (l *Linter) runFlake8(ctx context.Context) *Result {
	result := &Result{
		Language:  "python",
		ToolsUsed: []string{"flake8"},
	}

	cmd := exec.CommandContext(ctx, "flake8", "--format=%(path)s:%(row)d:%(col)d: %(code)s %(text)s", ".")
	cmd.Dir = l.config.Dir

	output, err := cmd.CombinedOutput()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// Issues found
		} else {
			result.Observations = append(result.Observations,
				shared.NewErrorObservation(fmt.Sprintf("flake8 failed: %v", err)))
			return result
		}
	}

	issues := l.parseFlake8Output(string(output))
	result.Issues = issues
	result.LintWarnings = len(issues)
	result.Success = true

	if len(issues) == 0 {
		result.Observations = append(result.Observations,
			shared.NewSuccessObservation("Python: flake8 found no issues"))
	} else {
		logPythonIssues(l.logWriter, issues, 20)
		result.Observations = append(result.Observations,
			shared.NewWarningObservation(fmt.Sprintf("Python: flake8 found %d issue(s)", len(issues))))
	}

	return result
}

func (l *Linter) parseFlake8Output(output string) []Issue {
	var issues []Issue
	// Format: path:row:col: code message
	re := regexp.MustCompile(`^(.+):(\d+):(\d+): (\w+) (.+)$`)

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
			Severity: SeverityWarning,
			Rule:     matches[4],
			Source:   "flake8",
		})
	}
	return issues
}

func (l *Linter) runMypy(ctx context.Context) *Result {
	result := &Result{
		Language:  "python",
		ToolsUsed: []string{"mypy"},
	}

	if !l.hasCommand("mypy") {
		result.Observations = append(result.Observations,
			shared.NewWarningObservation("Python: mypy not found, skipping type checking"))
		return result
	}

	cmd := exec.CommandContext(ctx, "mypy", ".")
	cmd.Dir = l.config.Dir

	output, err := cmd.CombinedOutput()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// Type errors found
			issues := l.parseMypyOutput(string(output))
			result.Issues = issues
			result.TypeErrors = len(issues)
			logPythonIssues(l.logWriter, issues, 20)
			result.Observations = append(result.Observations,
				shared.NewErrorObservation(fmt.Sprintf("Python: mypy found %d type error(s)", len(issues))))
			return result
		}
		result.Observations = append(result.Observations,
			shared.NewErrorObservation(fmt.Sprintf("mypy failed: %v", err)))
		return result
	}

	result.Success = true
	result.Observations = append(result.Observations,
		shared.NewSuccessObservation("Python: mypy found no type errors"))
	return result
}

func (l *Linter) parseMypyOutput(output string) []Issue {
	var issues []Issue
	// mypy output format: file:line: error: message
	re := regexp.MustCompile(`^(.+):(\d+): error: (.+)$`)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		matches := re.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		lineNum, _ := strconv.Atoi(matches[2])

		issues = append(issues, Issue{
			File:     matches[1],
			Line:     lineNum,
			Message:  matches[3],
			Severity: SeverityError,
			Source:   "mypy",
		})
	}
	return issues
}

func logPythonIssues(w io.Writer, issues []Issue, limit int) {
	if w == nil || len(issues) == 0 {
		return
	}
	if limit <= 0 || limit > len(issues) {
		limit = len(issues)
	}
	for idx, issue := range issues {
		if idx >= limit {
			break
		}
		location := issue.File
		if issue.Line > 0 {
			location = fmt.Sprintf("%s:%d", location, issue.Line)
			if issue.Column > 0 {
				location = fmt.Sprintf("%s:%d", location, issue.Column)
			}
		}
		message := issue.Message
		if issue.Rule != "" {
			message = fmt.Sprintf("%s [%s]", message, issue.Rule)
		}
		entry := fmt.Sprintf("%s: %s", location, message)
		if issue.Severity == SeverityWarning {
			shared.LogWarn(w, "%s", entry)
			continue
		}
		shared.LogError(w, "%s", entry)
	}
	if len(issues) > limit {
		shared.LogInfo(w, "... %d more issue(s) not shown", len(issues)-limit)
	}
}
