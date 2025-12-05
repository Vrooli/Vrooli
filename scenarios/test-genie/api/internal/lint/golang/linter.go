package golang

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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

// Result holds the result of linting Go code.
type Result struct {
	Language     string              `json:"language"`
	Success      bool                `json:"success"`
	Issues       []Issue             `json:"issues,omitempty"`
	TypeErrors   int                 `json:"typeErrors"`
	LintWarnings int                 `json:"lintWarnings"`
	ToolsUsed    []string            `json:"toolsUsed"`
	Skipped      bool                `json:"skipped"`
	SkipReason   string              `json:"skipReason,omitempty"`
	Observations []shared.Observation `json:"observations,omitempty"`
}

// Config holds configuration for Go linting.
type Config struct {
	Dir           string
	CommandLookup shared.LookupFunc
}

// Linter performs Go linting.
type Linter struct {
	config    Config
	logWriter io.Writer
}

// Option configures a Linter.
type Option func(*Linter)

// New creates a new Go linter.
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

// Lint runs Go linting and returns the result.
func (l *Linter) Lint(ctx context.Context) *Result {
	result := &Result{
		Language: "go",
	}

	// Check if directory exists
	if _, err := os.Stat(l.config.Dir); os.IsNotExist(err) {
		result.Skipped = true
		result.SkipReason = "api/ directory not found"
		result.Success = true
		result.Observations = append(result.Observations,
			shared.NewSkipObservation("Go: api/ directory not found"))
		return result
	}

	// Check if go.mod exists
	goModPath := filepath.Join(l.config.Dir, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		result.Skipped = true
		result.SkipReason = "no go.mod found"
		result.Success = true
		result.Observations = append(result.Observations,
			shared.NewSkipObservation("Go: no go.mod found in api/"))
		return result
	}

	// Try golangci-lint first
	if l.hasCommand("golangci-lint") {
		return l.runGolangciLint(ctx, result)
	}

	// Fallback to go vet
	shared.LogWarn(l.logWriter, "golangci-lint not found, falling back to go vet")
	result.Observations = append(result.Observations,
		shared.NewWarningObservation("golangci-lint not available, using go vet"))

	if l.hasCommand("go") {
		return l.runGoVet(ctx, result)
	}

	// Neither available
	result.Skipped = true
	result.SkipReason = "no Go linting tools available"
	result.Success = true
	result.Observations = append(result.Observations,
		shared.NewSkipObservation("Go: no linting tools available (install golangci-lint or go)"))
	return result
}

func (l *Linter) hasCommand(name string) bool {
	if l.config.CommandLookup != nil {
		_, err := l.config.CommandLookup(name)
		return err == nil
	}
	_, err := exec.LookPath(name)
	return err == nil
}

func (l *Linter) runGolangciLint(ctx context.Context, result *Result) *Result {
	result.ToolsUsed = append(result.ToolsUsed, "golangci-lint")

	cmd := exec.CommandContext(ctx, "golangci-lint", "run", "--out-format", "json", "./...")
	cmd.Dir = l.config.Dir

	output, err := cmd.Output()

	// golangci-lint exits with 1 if there are issues, which is not an error for us
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit code 1 means issues found, which we handle below
			if exitErr.ExitCode() != 1 {
				result.Success = false
				result.Observations = append(result.Observations,
					shared.NewErrorObservation(fmt.Sprintf("golangci-lint failed: %v", err)))
				return result
			}
			// Use stderr output if stdout is empty
			if len(output) == 0 {
				output = exitErr.Stderr
			}
		} else {
			result.Success = false
			result.Observations = append(result.Observations,
				shared.NewErrorObservation(fmt.Sprintf("golangci-lint failed: %v", err)))
			return result
		}
	}

	// Parse JSON output
	issues, typeErrors := l.parseGolangciLintOutput(output)
	result.Issues = issues
	result.TypeErrors = typeErrors
	result.LintWarnings = len(issues) - typeErrors
	result.Success = typeErrors == 0

	if len(issues) == 0 {
		result.Observations = append(result.Observations,
			shared.NewSuccessObservation("Go: golangci-lint found no issues"))
	} else {
		result.Observations = append(result.Observations,
			shared.NewInfoObservation(fmt.Sprintf("Go: %d issue(s) found by golangci-lint", len(issues))))
	}

	return result
}

func (l *Linter) runGoVet(ctx context.Context, result *Result) *Result {
	result.ToolsUsed = append(result.ToolsUsed, "go vet")

	cmd := exec.CommandContext(ctx, "go", "vet", "./...")
	cmd.Dir = l.config.Dir

	output, err := cmd.CombinedOutput()

	if err != nil {
		// go vet exits with 1 if there are issues
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// Parse the output for issues
			issues := l.parseGoVetOutput(string(output))
			result.Issues = issues
			result.TypeErrors = len(issues) // go vet issues are considered type-level errors
			result.Success = false
			result.Observations = append(result.Observations,
				shared.NewErrorObservation(fmt.Sprintf("Go: go vet found %d issue(s)", len(issues))))
			return result
		}
		result.Success = false
		result.Observations = append(result.Observations,
			shared.NewErrorObservation(fmt.Sprintf("go vet failed: %v", err)))
		return result
	}

	result.Success = true
	result.Observations = append(result.Observations,
		shared.NewSuccessObservation("Go: go vet found no issues"))
	return result
}

// golangciLintOutput represents the JSON output from golangci-lint.
type golangciLintOutput struct {
	Issues []golangciLintIssue `json:"Issues"`
}

type golangciLintIssue struct {
	FromLinter string `json:"FromLinter"`
	Text       string `json:"Text"`
	Severity   string `json:"Severity"`
	Pos        struct {
		Filename string `json:"Filename"`
		Line     int    `json:"Line"`
		Column   int    `json:"Column"`
	} `json:"Pos"`
}

func (l *Linter) parseGolangciLintOutput(output []byte) ([]Issue, int) {
	var parsed golangciLintOutput
	if err := json.Unmarshal(output, &parsed); err != nil {
		shared.LogWarn(l.logWriter, "failed to parse golangci-lint JSON: %v", err)
		return nil, 0
	}

	var issues []Issue
	typeErrors := 0

	for _, i := range parsed.Issues {
		severity := SeverityWarning
		// Type-related linters produce errors
		typeRelatedLinters := []string{"typecheck", "govet", "staticcheck"}
		for _, tl := range typeRelatedLinters {
			if i.FromLinter == tl {
				severity = SeverityError
				typeErrors++
				break
			}
		}

		issues = append(issues, Issue{
			File:     i.Pos.Filename,
			Line:     i.Pos.Line,
			Column:   i.Pos.Column,
			Message:  i.Text,
			Severity: severity,
			Rule:     i.FromLinter,
			Source:   "golangci-lint",
		})
	}

	return issues, typeErrors
}

func (l *Linter) parseGoVetOutput(output string) []Issue {
	var issues []Issue
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// go vet output format: file.go:line:col: message
		// or: file.go:line: message
		parts := strings.SplitN(line, ":", 4)
		if len(parts) < 3 {
			continue
		}

		issue := Issue{
			File:     parts[0],
			Severity: SeverityError,
			Source:   "go vet",
		}

		// Parse line number
		if _, err := fmt.Sscanf(parts[1], "%d", &issue.Line); err != nil {
			continue
		}

		// Check if column is present
		if len(parts) == 4 {
			fmt.Sscanf(parts[2], "%d", &issue.Column)
			issue.Message = strings.TrimSpace(parts[3])
		} else {
			issue.Message = strings.TrimSpace(parts[2])
		}

		issues = append(issues, issue)
	}

	return issues
}
