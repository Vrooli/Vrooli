// Package agents provides agent lifecycle management.
// This file contains the execution builder pattern for assembling agent execution
// configurations, extracted from handlers to localize change.
package agents

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"test-genie/internal/shared"
)

// ExecutionConfig holds all configuration needed to execute an agent.
// This separates "what to execute" from "how to execute it".
type ExecutionConfig struct {
	// Command and arguments
	Command    string   // Base command (e.g., "claude")
	Args       []string // Command arguments
	WorkingDir string   // Working directory for execution

	// Environment
	Environment map[string]string

	// Resource limits
	MaxMemoryMB   int
	MaxCPUPercent int
	TimeoutSecs   int

	// Security
	NetworkEnabled bool
	AllowedTools   []string

	// Agent metadata
	AgentID  string
	Scenario string
	Model    string
	Prompt   string
}

// ExecutionBuilderInput contains all inputs needed to build an execution config.
type ExecutionBuilderInput struct {
	// Required fields
	AgentID  string
	Scenario string
	Model    string
	Prompt   string

	// Paths
	RepoRoot    string
	APIEndpoint string

	// Optional execution parameters (defaults used if zero)
	TimeoutSecs    int
	MaxTurns       int
	MaxFiles       int
	MaxBytes       int64
	NetworkEnabled bool

	// Tool configuration
	AllowedTools    []string
	SystemPreamble  string
	SystemPreamble2 string

	// Output configuration
	OutputFormat string // "stream-json", "text", etc.

	// Container/execution settings
	MaxMemoryMB   int
	MaxCPUPercent int
}

// ExecutionBuilderResult holds the outcome of building an execution config.
type ExecutionBuilderResult struct {
	Config  ExecutionConfig
	Errors  []string
	Success bool
}

// ExecutionBuilder constructs agent execution configurations.
// It uses the builder pattern to allow step-by-step construction with validation.
type ExecutionBuilder struct {
	defaults Config // Agent defaults from config
	errors   []string
	config   ExecutionConfig
}

// NewExecutionBuilder creates a new execution builder with the given defaults.
func NewExecutionBuilder(defaults Config) *ExecutionBuilder {
	return &ExecutionBuilder{
		defaults: defaults,
		errors:   make([]string, 0),
		config: ExecutionConfig{
			Environment: make(map[string]string),
		},
	}
}

// WithInput populates the builder from a structured input.
// This is the primary entry point for building configurations.
func (b *ExecutionBuilder) WithInput(input ExecutionBuilderInput) *ExecutionBuilder {
	// Validate required fields
	if input.AgentID == "" {
		b.errors = append(b.errors, "agentID is required")
	}
	if input.Scenario == "" {
		b.errors = append(b.errors, "scenario is required")
	}
	if input.Model == "" {
		b.errors = append(b.errors, "model is required")
	}
	if input.Prompt == "" {
		b.errors = append(b.errors, "prompt is required")
	}
	if input.RepoRoot == "" {
		b.errors = append(b.errors, "repoRoot is required")
	}
	if input.APIEndpoint == "" {
		b.errors = append(b.errors, "apiEndpoint is required")
	}

	// Store metadata
	b.config.AgentID = input.AgentID
	b.config.Scenario = input.Scenario
	b.config.Model = input.Model
	b.config.Prompt = input.Prompt

	// Set working directory
	b.config.WorkingDir = filepath.Join(input.RepoRoot, "scenarios", input.Scenario)

	// Apply defaults for optional parameters
	timeout := input.TimeoutSecs
	if timeout <= 0 {
		timeout = b.defaults.DefaultTimeoutSeconds
	}
	b.config.TimeoutSecs = timeout

	maxTurns := input.MaxTurns
	if maxTurns <= 0 {
		maxTurns = b.defaults.DefaultMaxTurns
	}

	maxFiles := input.MaxFiles
	if maxFiles <= 0 {
		maxFiles = b.defaults.DefaultMaxFilesChanged
	}

	maxBytes := input.MaxBytes
	if maxBytes <= 0 {
		maxBytes = b.defaults.DefaultMaxBytesWritten
	}

	b.config.NetworkEnabled = input.NetworkEnabled
	b.config.MaxMemoryMB = input.MaxMemoryMB
	b.config.MaxCPUPercent = input.MaxCPUPercent
	b.config.AllowedTools = input.AllowedTools

	// Build command arguments
	b.buildClaudeArgs(input, maxTurns, maxFiles, maxBytes)

	return b
}

// buildClaudeArgs constructs the Claude CLI arguments.
func (b *ExecutionBuilder) buildClaudeArgs(input ExecutionBuilderInput, maxTurns, maxFiles int, maxBytes int64) {
	b.config.Command = "claude"

	args := []string{
		"--print",                        // Non-interactive mode
		"--dangerously-skip-permissions", // We enforce permissions server-side
	}

	// Model
	args = append(args, "--model", input.Model)

	// Output format
	outputFormat := input.OutputFormat
	if outputFormat == "" {
		outputFormat = "stream-json"
	}
	args = append(args, "--output-format", outputFormat)

	// Timeout
	if b.config.TimeoutSecs > 0 {
		args = append(args, "--max-thinking-budget", strconv.Itoa(b.config.TimeoutSecs*1000))
	}

	// Max turns
	if maxTurns > 0 {
		args = append(args, "--max-turns", strconv.Itoa(maxTurns))
	}

	// Allowed tools (if specified)
	if len(input.AllowedTools) > 0 {
		args = append(args, "--allowed-tools", strings.Join(input.AllowedTools, ","))
	}

	// System preambles
	if input.SystemPreamble != "" {
		args = append(args, "--system", input.SystemPreamble)
	}
	if input.SystemPreamble2 != "" {
		args = append(args, "--system2", input.SystemPreamble2)
	}

	// The prompt itself goes last
	args = append(args, input.Prompt)

	b.config.Args = args

	// Set up environment variables
	b.config.Environment["TEST_GENIE_AGENT_ID"] = input.AgentID
	b.config.Environment["TEST_GENIE_API_ENDPOINT"] = input.APIEndpoint
	b.config.Environment["TEST_GENIE_SCENARIO"] = input.Scenario
	b.config.Environment["TEST_GENIE_MAX_FILES"] = strconv.Itoa(maxFiles)
	b.config.Environment["TEST_GENIE_MAX_BYTES"] = strconv.FormatInt(maxBytes, 10)
}

// Build finalizes and returns the execution configuration.
func (b *ExecutionBuilder) Build() ExecutionBuilderResult {
	return ExecutionBuilderResult{
		Config:  b.config,
		Errors:  b.errors,
		Success: len(b.errors) == 0,
	}
}

// --- Decision functions for execution building ---

// ExecutionParameterDecision describes what value to use for an execution parameter.
type ExecutionParameterDecision struct {
	Value         interface{}
	Source        string // "request", "default", "clamped"
	WasClamped    bool
	OriginalValue interface{}
}

// DecideTimeoutSeconds determines the timeout to use for an agent execution.
// This is the central decision point for timeout selection.
//
// Decision criteria:
//   - If request specifies a value > 0, use it (clamped to bounds)
//   - Otherwise use the default from config
//   - Always clamp to [60, 3600] seconds
func DecideTimeoutSeconds(requestValue int, defaultValue int) ExecutionParameterDecision {
	minTimeout := 60
	maxTimeout := 3600

	if requestValue > 0 {
		clamped := shared.ClampInt(requestValue, minTimeout, maxTimeout)
		return ExecutionParameterDecision{
			Value:         clamped,
			Source:        "request",
			WasClamped:    clamped != requestValue,
			OriginalValue: requestValue,
		}
	}

	return ExecutionParameterDecision{
		Value:  defaultValue,
		Source: "default",
	}
}

// DecideMaxTurns determines the max turns to use for an agent execution.
// Decision: request > 0 uses request (clamped), otherwise default.
func DecideMaxTurns(requestValue int, defaultValue int) ExecutionParameterDecision {
	minTurns := 5
	maxTurns := 200

	if requestValue > 0 {
		clamped := shared.ClampInt(requestValue, minTurns, maxTurns)
		return ExecutionParameterDecision{
			Value:         clamped,
			Source:        "request",
			WasClamped:    clamped != requestValue,
			OriginalValue: requestValue,
		}
	}

	return ExecutionParameterDecision{
		Value:  defaultValue,
		Source: "default",
	}
}

// DecideMaxFilesChanged determines the max files limit for an agent execution.
func DecideMaxFilesChanged(requestValue int, defaultValue int) ExecutionParameterDecision {
	minFiles := 1
	maxFiles := 500

	if requestValue > 0 {
		clamped := shared.ClampInt(requestValue, minFiles, maxFiles)
		return ExecutionParameterDecision{
			Value:         clamped,
			Source:        "request",
			WasClamped:    clamped != requestValue,
			OriginalValue: requestValue,
		}
	}

	return ExecutionParameterDecision{
		Value:  defaultValue,
		Source: "default",
	}
}

// DecideMaxBytesWritten determines the max bytes limit for an agent execution.
func DecideMaxBytesWritten(requestValue int64, defaultValue int64) ExecutionParameterDecision {
	minBytes := int64(1024)      // 1KB
	maxBytes := int64(104857600) // 100MB

	if requestValue > 0 {
		clamped := shared.ClampInt64(requestValue, minBytes, maxBytes)
		return ExecutionParameterDecision{
			Value:         clamped,
			Source:        "request",
			WasClamped:    clamped != requestValue,
			OriginalValue: requestValue,
		}
	}

	return ExecutionParameterDecision{
		Value:  defaultValue,
		Source: "default",
	}
}

// FormatCommandForDisplay returns a display-safe version of the command.
// Truncates long prompts and masks sensitive environment variables.
func (c ExecutionConfig) FormatCommandForDisplay() string {
	displayArgs := make([]string, len(c.Args))
	copy(displayArgs, c.Args)

	// Truncate the last arg (prompt) if too long
	if len(displayArgs) > 0 {
		lastIdx := len(displayArgs) - 1
		if len(displayArgs[lastIdx]) > 100 {
			displayArgs[lastIdx] = displayArgs[lastIdx][:100] + "..."
		}
	}

	return fmt.Sprintf("%s %s", c.Command, strings.Join(displayArgs, " "))
}
