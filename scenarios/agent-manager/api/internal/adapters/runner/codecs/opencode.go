// Package codecs — opencode.go is the [Codec] implementation for the
// OpenCode CLI (the `resource-opencode` wrapper invoked with
// `--format json`).
//
// What this codec owns:
//   - CLI args + env shape (BuildArgs, BuildContinueArgs, BuildEnv)
//   - The OpenCode JSON stream decoder (DecodeStreamLine, ParseTranscriptLine)
//   - Per-run state: captured session id, terminal step_finish marker
//   - step_finish handling — emits cost + message events, signals early
//     termination via OnEarlyTerminate when reason is terminal
//   - Log-file fallback for error messages when the wrapper exits with
//     just an exit-status string (PostClassify hook)
package codecs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/fallback"

	"github.com/google/uuid"

	repocontract "github.com/vrooli/repo-contract-go"
)

// =============================================================================
// Constants
// =============================================================================

// OpenCodeResourceCommand is the Vrooli wrapper binary name resolved on
// the host PATH.
const OpenCodeResourceCommand = "resource-opencode"

const opencodeTagEnvKey = "OPENCODE_AGENT_TAG"

// =============================================================================
// Codec
// =============================================================================

// OpenCode is the [Codec] implementation for the resource-opencode wrapper.
type OpenCode struct {
	binaryPath  string
	available   bool
	message     string
	installHint string
}

// NewOpenCode resolves the `resource-opencode` binary on PATH and runs a
// quick health-check via `status --format json`. Returns a codec with
// Available=false (rather than an error) when the binary is missing or
// unhealthy so the runner registry can register a stub instead.
func NewOpenCode() (*OpenCode, error) {
	binaryPath, err := exec.LookPath(OpenCodeResourceCommand)
	if err != nil {
		return &OpenCode{
			available:   false,
			message:     "resource-opencode not found in PATH",
			installHint: "Run: vrooli resource install opencode",
		}, nil
	}

	c := &OpenCode{
		binaryPath: binaryPath,
		available:  true,
		message:    "resource-opencode available",
	}

	// Quick health check via `status --format json`. Mirrors the
	// behaviour of the previous OpenCodeRunner constructor.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binaryPath, "status", "--format", "json")
	output, statusErr := cmd.Output()
	if statusErr != nil {
		c.available = false
		c.message = fmt.Sprintf("resource-opencode status check failed: %v", statusErr)
		c.installHint = "Run: resource-opencode manage install"
		return c, nil
	}

	var statusData map[string]interface{}
	if err := json.Unmarshal(output, &statusData); err == nil {
		if healthy, ok := statusData["healthy"].(bool); ok && !healthy {
			c.available = false
			if msg, ok := statusData["message"].(string); ok {
				c.message = msg
			} else {
				c.message = "resource-opencode is not healthy"
			}
			c.installHint = "Run: resource-opencode manage install"
		}
	}
	return c, nil
}

// NewOpenCodeForTest returns an OpenCode codec with a fake binary path
// and Available=false. Used by codec tests that exercise BuildArgs /
// decode paths without launching a real process.
func NewOpenCodeForTest() *OpenCode {
	return &OpenCode{
		binaryPath: "/fake/opencode",
		available:  false,
		message:    "test opencode codec",
	}
}

// Type satisfies [Codec].
func (c *OpenCode) Type() domain.RunnerType { return domain.RunnerTypeOpenCode }

// Capabilities satisfies [Codec].
func (c *OpenCode) Capabilities() runner.Capabilities {
	return runner.Capabilities{
		SupportsMessages:         true,
		SupportsToolEvents:       true,
		SupportsCostTracking:     false,
		SupportsStreaming:        false,
		SupportsCancellation:     true,
		SupportsContinuation:     true, // `--session <id>`
		SupportsImageAttachments: false,
		MaxTurns:                 0,
		SupportedModels: []string{
			"anthropic/claude-sonnet-4-5",
			"anthropic/claude-opus-4-5",
			"openai/gpt-4o",
			"openai/o4-mini",
			"google/gemini-2.0-flash",
			"deepseek/deepseek-chat",
		},
		SupportedFeatures: []string{},
		AllowedExtraFlags: []string{"--verbose"},
	}
}

// BinaryPath satisfies [Codec].
func (c *OpenCode) BinaryPath() string { return c.binaryPath }

// BinaryDescription satisfies [Codec].
func (c *OpenCode) BinaryDescription() string { return "resource-opencode wrapper" }

// TagEnvKey satisfies [Codec].
func (c *OpenCode) TagEnvKey() string { return opencodeTagEnvKey }

// Available satisfies [Codec].
func (c *OpenCode) Available(ctx context.Context) (bool, string) {
	if !c.available {
		msg := c.message
		if c.installHint != "" {
			msg += ". " + c.installHint
		}
		return false, msg
	}
	if _, err := os.Stat(c.binaryPath); os.IsNotExist(err) {
		return false, "resource-opencode binary not found. Run: vrooli resource install opencode"
	}
	return true, "resource-opencode is available"
}

// ProbeModel satisfies [Codec]. Lightweight by design — a deep probe
// would burn vendor quota; runtime classification surfaces a real
// "model is gone" failure on first use.
func (c *OpenCode) ProbeModel(ctx context.Context, modelID string) error {
	if available, msg := c.Available(ctx); !available {
		return fmt.Errorf("opencode unavailable: %s", msg)
	}
	return nil
}

// Labels satisfies [Codec].
func (c *OpenCode) Labels() Labels {
	return Labels{
		StartMessage:         "OpenCode execution started",
		EndMessage:           "OpenCode execution completed",
		ContinueStartMessage: "OpenCode continuation started",
		ContinueEndMessage:   "OpenCode continuation completed",
	}
}

// ContinueTag satisfies [Codec].
func (c *OpenCode) ContinueTag(req runner.ContinueRequest) string {
	return fmt.Sprintf("opencode-continue-%s", req.RunID.String()[:8])
}

// BuildEnv satisfies [Codec]. Mirrors the resource-opencode env shape.
func (c *OpenCode) BuildEnv(tag string, extras map[string]string) []string {
	env := runner.SanitizedBaseEnv()
	env = append(env, fmt.Sprintf("%s=%s", opencodeTagEnvKey, tag))
	env = append(env, "OPENCODE_NON_INTERACTIVE=true")
	return runner.AppendEnvMap(env, extras)
}

// BuildPrompt satisfies [Codec]. OpenCode passes the prompt as a CLI
// argument (in BuildArgs); stdin is unused.
func (c *OpenCode) BuildPrompt(_ string, _ []runner.Attachment) string { return "" }

// BuildArgs satisfies [Codec]. The prompt is on the command line so the
// caller MUST close stdin — core.Runner does that automatically when the
// codec returns "" from BuildPrompt.
func (c *OpenCode) BuildArgs(_ State, req runner.ExecuteRequest) []string {
	args := []string{
		"run", // resource-opencode subcommand
		"run", // opencode subcommand
		req.EffectivePrompt(),
		"--format", "json",
	}
	cfg := req.GetConfig()
	if cfg.Model != "" {
		args = append(args, "--model", cfg.Model)
	}
	if extras, ok := cfg.ExtraFlags[domain.RunnerTypeOpenCode]; ok {
		args = append(args, extras...)
	}
	return args
}

// BuildContinueArgs satisfies [Codec].
func (c *OpenCode) BuildContinueArgs(_ State, req runner.ContinueRequest) []string {
	return []string{
		"run", "run",
		req.Prompt,
		"--session", req.SessionID,
		"--format", "json",
	}
}

// =============================================================================
// State
// =============================================================================

// opencodeState carries per-run mutable state.
type opencodeState struct {
	sessionID   string
	stepTermina bool // set by step_finish parsing when reason is terminal
}

func (s *opencodeState) SessionID() string { return s.sessionID }

// NewState satisfies [Codec].
func (c *OpenCode) NewState() State { return &opencodeState{} }

// =============================================================================
// Stream-event types
// =============================================================================

// OpenCodeStreamEvent represents a single event from OpenCode's JSON
// output. Format: {"type":"...", "timestamp":..., "sessionID":"...", "part":{...}}.
type OpenCodeStreamEvent struct {
	Type      string         `json:"type"`
	Timestamp int64          `json:"timestamp,omitempty"`
	SessionID string         `json:"sessionID,omitempty"`
	Part      *OpenCodePart  `json:"part,omitempty"`
	Error     *OpenCodeError `json:"error,omitempty"`
}

// OpenCodePart is the body of a streaming event.
type OpenCodePart struct {
	ID        string          `json:"id,omitempty"`
	SessionID string          `json:"sessionID,omitempty"`
	MessageID string          `json:"messageID,omitempty"`
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	Reason    string          `json:"reason,omitempty"`
	Snapshot  string          `json:"snapshot,omitempty"`
	Cost      float64         `json:"cost,omitempty"`
	Tokens    *OpenCodeTokens `json:"tokens,omitempty"`
	Name      string          `json:"name,omitempty"`  // legacy tool name field
	Input     json.RawMessage `json:"input,omitempty"` // legacy tool input field
	Output    string          `json:"output,omitempty"`
	IsError   bool            `json:"isError,omitempty"`
	Time      *OpenCodeTime   `json:"time,omitempty"`
	// New fields for the actual OpenCode tool_use format.
	Tool   string         `json:"tool,omitempty"`
	CallID string         `json:"callID,omitempty"`
	State  *OpenCodeState `json:"state,omitempty"`
}

// OpenCodeState is the body of a tool_use event's "state" field.
type OpenCodeState struct {
	Status   string                 `json:"status,omitempty"`
	Input    map[string]interface{} `json:"input,omitempty"`
	Output   string                 `json:"output,omitempty"`
	Title    string                 `json:"title,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// OpenCodeTokens carries token counts in step_finish events.
type OpenCodeTokens struct {
	Input     int            `json:"input"`
	Output    int            `json:"output"`
	Reasoning int            `json:"reasoning,omitempty"`
	Cache     *OpenCodeCache `json:"cache,omitempty"`
}

// OpenCodeCache carries cache-token counts.
type OpenCodeCache struct {
	Read  int `json:"read"`
	Write int `json:"write"`
}

// OpenCodeTime carries timing metadata.
type OpenCodeTime struct {
	Start int64 `json:"start,omitempty"`
	End   int64 `json:"end,omitempty"`
}

// OpenCodeError carries error metadata.
type OpenCodeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
}

// =============================================================================
// Stream decoding
// =============================================================================

// DecodeStreamLine satisfies [Codec]. Returns zero or more events.
// Captures session_id and terminal-step-finish markers on state.
func (c *OpenCode) DecodeStreamLine(state State, runID uuid.UUID, line string) ([]*domain.RunEvent, error) {
	s, ok := state.(*opencodeState)
	if !ok {
		return nil, fmt.Errorf("opencode: invalid state type %T", state)
	}

	line = strings.TrimSpace(line)
	if line == "" {
		return nil, nil
	}
	if line[0] != '{' && line[0] != '[' {
		// Non-JSON startup output — silently skip.
		return nil, nil
	}

	if line[0] == '[' {
		var streamEvents []OpenCodeStreamEvent
		if err := json.Unmarshal([]byte(line), &streamEvents); err != nil {
			return nil, domain.NewInternalError("invalid opencode JSON", err)
		}
		events := []*domain.RunEvent{}
		for _, ev := range streamEvents {
			c.captureSessionID(s, &ev)
			parsed, err := c.parseOpenCodeStreamEvent(s, runID, ev)
			if err != nil {
				return events, err
			}
			events = append(events, parsed...)
		}
		return events, nil
	}

	var streamEvent OpenCodeStreamEvent
	if err := json.Unmarshal([]byte(line), &streamEvent); err != nil {
		return nil, domain.NewInternalError("invalid opencode JSON", err)
	}
	c.captureSessionID(s, &streamEvent)
	return c.parseOpenCodeStreamEvent(s, runID, streamEvent)
}

func (c *OpenCode) captureSessionID(s *opencodeState, ev *OpenCodeStreamEvent) {
	if s.sessionID != "" {
		return
	}
	if ev.SessionID != "" {
		s.sessionID = ev.SessionID
		return
	}
	if ev.Part != nil && ev.Part.SessionID != "" {
		s.sessionID = ev.Part.SessionID
	}
}

// OnEarlyTerminate satisfies [Codec]. Reads the terminal-step-finish
// flag stashed by DecodeStreamLine and signals the scanner loop to exit.
func (c *OpenCode) OnEarlyTerminate(state State, _ string) bool {
	s, ok := state.(*opencodeState)
	if !ok {
		return false
	}
	return s.stepTermina
}

// PostClassify satisfies [Codec]. When the wrapper exits with only an
// exit-status string (typical for opencode subprocess crashes), tail
// the latest log file and substitute the most recent error message.
func (c *OpenCode) PostClassify(_ State, result *runner.ExecuteResult) {
	if result == nil {
		return
	}
	if result.Success {
		return
	}
	msg := strings.TrimSpace(result.ErrorMessage)
	if msg != "" && !strings.Contains(msg, "exit status") {
		return
	}
	if fallback := resolveOpenCodeLogError(); fallback != "" {
		if msg == "" {
			result.ErrorMessage = fallback
		} else {
			result.ErrorMessage = fallback
		}
	}
}

// ClassifyTerminalError satisfies [Codec]. OpenCode signals session
// expiry via stderr that mentions "session" and one of "not found" /
// "expired" / "invalid". OpenCode does not currently surface a state-
// lost shape distinct from expiry, so all matches map to
// ErrCodeRunnerSessionExpired.
func (c *OpenCode) ClassifyTerminalError(stderr string, exitCode int) *domain.RunnerError {
	if !strings.Contains(stderr, "session") {
		return nil
	}
	if !strings.Contains(stderr, "not found") &&
		!strings.Contains(stderr, "expired") &&
		!strings.Contains(stderr, "invalid") {
		return nil
	}
	return domain.NewRunnerSessionExpiredError(c.Type(), errors.New(strings.TrimSpace(stderr)))
}

// Classify satisfies [Codec]. OpenCode classification order:
//
//  1. Session expiry stderr — ReasonSessionExpired (mirrors
//     [ClassifyTerminalError]; OpenCode does not currently surface a
//     state-lost shape distinct from expiry).
//  2. Residual TextClassifier — covers model, auth, network, etc.
//
// Returns nil only when stderr is empty and exitCode == 0.
func (c *OpenCode) Classify(stderr string, exitCode int) *fallback.ClassifiedError {
	if stderr == "" && exitCode == 0 {
		return nil
	}
	if strings.Contains(stderr, "session") &&
		(strings.Contains(stderr, "not found") ||
			strings.Contains(stderr, "expired") ||
			strings.Contains(stderr, "invalid")) {
		return fallback.New(fallback.ReasonSessionExpired, strings.TrimSpace(stderr), nil)
	}
	return fallback.NewTextClassifier().Classify(fallback.ClassifyInput{
		RunnerType: string(c.Type()),
		Stderr:     stderr,
		ExitCode:   exitCode,
	})
}

// UpdateMetrics satisfies [Codec].
func (c *OpenCode) UpdateMetrics(event *domain.RunEvent, metrics *runner.ExecutionMetrics, lastAssistant *string) {
	if event == nil {
		return
	}
	switch data := event.Data.(type) {
	case *domain.MessageEventData:
		if data.Role == "assistant" {
			*lastAssistant = data.Content
			metrics.TurnsUsed++
		}
	case *domain.ToolCallEventData:
		metrics.ToolCallCount++
	case *domain.ToolResultEventData:
		// OpenCode bundles tool call + result in one event with
		// status="completed"; count the result as a tool call.
		metrics.ToolCallCount++
	case *domain.MetricEventData:
		if data.Name == "tokens" {
			totalTokens := int(data.Value)
			if totalTokens > metrics.TokensInput+metrics.TokensOutput {
				metrics.TokensOutput = totalTokens - metrics.TokensInput
			}
		} else if data.Name == "cost" {
			metrics.CostEstimateUSD = data.Value
		}
	case *domain.CostEventData:
		metrics.TokensInput = data.InputTokens
		metrics.TokensOutput = data.OutputTokens
		metrics.CacheReadTokens = data.CacheReadTokens
		metrics.CacheCreationTokens = data.CacheCreationTokens
		metrics.CostEstimateUSD = data.TotalCostUSD
	}
}

// ParseTranscriptLine satisfies [Codec] for single-line transcript parsing.
// Multi-line replay uses NewTranscriptParser to retain runner state across
// the transcript stream.
func (c *OpenCode) ParseTranscriptLine(runID uuid.UUID, line string) runner.TranscriptParseResult {
	parser := c.NewTranscriptParser()
	return parser.ParseTranscriptLine(runID, line)
}

// NewTranscriptParser satisfies [Codec].
func (c *OpenCode) NewTranscriptParser() runner.TranscriptParser {
	return &opencodeTranscriptParser{codec: c, state: &opencodeState{}}
}

type opencodeTranscriptParser struct {
	codec *OpenCode
	state *opencodeState
}

func (p *opencodeTranscriptParser) ParseTranscriptLine(runID uuid.UUID, line string) runner.TranscriptParseResult {
	events, err := p.codec.DecodeStreamLine(p.state, runID, line)
	result := runner.TranscriptParseResult{
		Events:    events,
		SessionID: p.state.sessionID,
		Err:       err,
	}

	var streamEvent OpenCodeStreamEvent
	if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &streamEvent); err == nil &&
		streamEvent.Part != nil && streamEvent.Type == "step_finish" {
		if isTerminalStepFinish(streamEvent.Part) {
			terminal := &runner.TranscriptTerminal{Success: true, ExitCode: 0}
			reason := strings.ToLower(strings.TrimSpace(streamEvent.Part.Reason))
			switch reason {
			case "error":
				terminal.Success = false
				terminal.ExitCode = 1
				terminal.ErrorMessage = runner.StripANSI(streamEvent.Part.Output)
				if terminal.ErrorMessage == "" {
					terminal.ErrorMessage = "opencode reported terminal error"
				}
			case "cancelled", "canceled":
				terminal.Success = false
				terminal.ExitCode = 130
				terminal.ErrorMessage = "opencode session cancelled"
			}
			result.Terminal = terminal
		}
	}
	return result
}

// =============================================================================
// Stream parser
// =============================================================================

// parseOpenCodeStreamEvent dispatches a parsed OpenCodeStreamEvent to the
// per-event-type handlers and returns zero or more domain events.
func (c *OpenCode) parseOpenCodeStreamEvent(state *opencodeState, runID uuid.UUID, streamEvent OpenCodeStreamEvent) ([]*domain.RunEvent, error) {
	if streamEvent.Error != nil {
		code := streamEvent.Error.Code
		if code == "" {
			code = streamEvent.Error.Type
		}
		if code == "" {
			code = "execution_error"
		}
		return []*domain.RunEvent{domain.NewErrorEvent(runID, code, runner.StripANSI(streamEvent.Error.Message), false)}, nil
	}

	switch streamEvent.Type {
	case "step_start":
		return []*domain.RunEvent{domain.NewLogEvent(runID, "info", "OpenCode step started")}, nil

	case "text":
		if streamEvent.Part != nil && streamEvent.Part.Text != "" {
			return []*domain.RunEvent{domain.NewMessageEvent(runID, "assistant", runner.StripANSI(streamEvent.Part.Text))}, nil
		}

	case "tool_call", "tool_use", "tool-call":
		if streamEvent.Part != nil {
			return parseOpenCodeToolUse(runID, streamEvent.Part), nil
		}

	case "tool_result", "tool-result":
		if streamEvent.Part != nil {
			return parseOpenCodeToolResult(runID, streamEvent.Part), nil
		}

	case "step_finish":
		if streamEvent.Part != nil {
			return c.handleStepFinish(state, runID, streamEvent.Part), nil
		}

	case "error":
		if streamEvent.Part != nil && streamEvent.Part.IsError {
			return []*domain.RunEvent{domain.NewErrorEvent(runID, "execution_error", runner.StripANSI(streamEvent.Part.Output), false)}, nil
		}

	case "user_message":
		if streamEvent.Part != nil && streamEvent.Part.Text != "" {
			return []*domain.RunEvent{domain.NewMessageEvent(runID, "user", runner.StripANSI(streamEvent.Part.Text))}, nil
		}

	case "thinking":
		if streamEvent.Part != nil && streamEvent.Part.Text != "" {
			return []*domain.RunEvent{domain.NewLogEvent(runID, "debug", fmt.Sprintf("Thinking: %s", runner.StripANSI(streamEvent.Part.Text)))}, nil
		}

	case "assistant", "response", "message", "assistant_message":
		if streamEvent.Part != nil {
			text := runner.StripANSI(streamEvent.Part.Text)
			if text == "" {
				text = runner.StripANSI(streamEvent.Part.Output)
			}
			if text != "" {
				return []*domain.RunEvent{domain.NewMessageEvent(runID, "assistant", text)}, nil
			}
		}

	case "content", "content_block":
		if streamEvent.Part != nil && streamEvent.Part.Text != "" {
			role := "assistant"
			if streamEvent.Part.Type == "user" {
				role = "user"
			}
			return []*domain.RunEvent{domain.NewMessageEvent(runID, role, runner.StripANSI(streamEvent.Part.Text))}, nil
		}
	}

	// Part.Type as secondary classification: OpenCode often signals tool
	// invocations via part.type="tool" with a generic top-level type.
	if streamEvent.Part != nil && streamEvent.Part.Type != "" {
		switch streamEvent.Part.Type {
		case "text", "assistant":
			if streamEvent.Part.Text != "" {
				return []*domain.RunEvent{domain.NewMessageEvent(runID, "assistant", runner.StripANSI(streamEvent.Part.Text))}, nil
			}
		case "tool", "tool-call", "tool_call", "tool_use":
			return parseOpenCodeToolUse(runID, streamEvent.Part), nil
		case "tool-result", "tool_result":
			return parseOpenCodeToolResult(runID, streamEvent.Part), nil
		}
	}

	return []*domain.RunEvent{domain.NewLogEvent(
		runID, "debug", fmt.Sprintf("OpenCode event [%s]", streamEvent.Type),
	)}, nil
}

// parseOpenCodeToolUse extracts a tool_call (and possibly a bundled
// tool_result when state.status == completed) from a part.
func parseOpenCodeToolUse(runID uuid.UUID, part *OpenCodePart) []*domain.RunEvent {
	toolName := part.Tool
	if toolName == "" {
		toolName = part.Name
	}
	if toolName == "" {
		toolName = "unknown_tool"
	}

	// Inputs come from State.Input first (current OpenCode), with a
	// legacy fallback to the part-level Input field.
	input := make(map[string]interface{})
	if part.State != nil && part.State.Input != nil {
		input = part.State.Input
	} else if part.Input != nil {
		_ = json.Unmarshal(part.Input, &input)
	}

	// OpenCode sometimes bundles call + result in one event with
	// status=="completed".
	if part.State != nil && part.State.Status == "completed" {
		events := []*domain.RunEvent{domain.NewToolCallEvent(runID, toolName, "", input)}
		output := runner.StripANSI(part.State.Output)
		if output == "" {
			output = runner.StripANSI(part.Output)
		}
		var errMsg error
		if part.IsError {
			errMsg = fmt.Errorf("%s", output)
		}
		events = append(events, domain.NewToolResultEvent(runID, toolName, part.CallID, output, errMsg))
		return events
	}
	return []*domain.RunEvent{domain.NewToolCallEvent(runID, toolName, "", input)}
}

// parseOpenCodeToolResult extracts a tool_result (and possibly a
// preceding tool_call when state.input is non-nil) from a part.
func parseOpenCodeToolResult(runID uuid.UUID, part *OpenCodePart) []*domain.RunEvent {
	toolName := part.Tool
	if toolName == "" {
		toolName = part.Name
	}

	output := runner.StripANSI(part.Output)
	if output == "" && part.State != nil {
		output = runner.StripANSI(part.State.Output)
	}

	events := []*domain.RunEvent{}
	if part.State != nil && part.State.Input != nil {
		events = append(events, domain.NewToolCallEvent(runID, toolName, "", part.State.Input))
	}
	if part.IsError {
		events = append(events, domain.NewToolResultEvent(runID, toolName, part.CallID, "", fmt.Errorf("%s", output)))
		return events
	}
	events = append(events, domain.NewToolResultEvent(runID, toolName, part.CallID, output, nil))
	return events
}

// handleStepFinish builds the cost + assistant-message events from a
// step_finish part and stashes the terminal flag on state.
func (c *OpenCode) handleStepFinish(state *opencodeState, runID uuid.UUID, part *OpenCodePart) []*domain.RunEvent {
	events := []*domain.RunEvent{}

	if costEvent := buildOpenCodeCostEvent(runID, part); costEvent != nil {
		events = append(events, costEvent)
	}
	if msgEvent := extractOpenCodeAssistantMessage(runID, part); msgEvent != nil {
		events = append(events, msgEvent)
	}

	if isTerminalStepFinish(part) {
		state.stepTermina = true
	}
	return events
}

func buildOpenCodeCostEvent(runID uuid.UUID, part *OpenCodePart) *domain.RunEvent {
	var inputTokens, outputTokens, cacheRead, cacheWrite int
	if part.Tokens != nil {
		inputTokens = part.Tokens.Input
		outputTokens = part.Tokens.Output
		if part.Tokens.Cache != nil {
			cacheRead = part.Tokens.Cache.Read
			cacheWrite = part.Tokens.Cache.Write
		}
	}
	return &domain.RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: domain.EventTypeMetric,
		Timestamp: time.Now(),
		Data: &domain.CostEventData{
			InputTokens:         inputTokens,
			OutputTokens:        outputTokens,
			CacheCreationTokens: cacheWrite,
			CacheReadTokens:     cacheRead,
			TotalCostUSD:        part.Cost,
			CostSource:          domain.CostSourceRunnerReported,
			PricingProvider:     "opencode",
		},
	}
}

// extractOpenCodeAssistantMessage tries Text → Output → Snapshot (when
// not a hash) for the assistant's final message in a step_finish.
func extractOpenCodeAssistantMessage(runID uuid.UUID, part *OpenCodePart) *domain.RunEvent {
	if part.Text != "" {
		return domain.NewMessageEvent(runID, "assistant", runner.StripANSI(part.Text))
	}
	if part.Output != "" {
		return domain.NewMessageEvent(runID, "assistant", runner.StripANSI(part.Output))
	}
	if part.Snapshot != "" && !isLikelyHash(part.Snapshot) {
		return domain.NewMessageEvent(runID, "assistant", runner.StripANSI(part.Snapshot))
	}
	return nil
}

// isTerminalStepFinish reports whether a step_finish reason should end
// the run.
func isTerminalStepFinish(part *OpenCodePart) bool {
	if part == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(part.Reason)) {
	case "stop", "length", "error", "cancelled", "canceled":
		return true
	default:
		return false
	}
}

// isLikelyHash detects 40-char or 64-char hex strings (git or sha256
// digests) so we don't accidentally treat them as prose snapshots.
func isLikelyHash(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	if len(trimmed) != 40 && len(trimmed) != 64 {
		return false
	}
	for _, r := range trimmed {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

// =============================================================================
// Log-file fallback for missing error messages
// =============================================================================

// resolveOpenCodeLogError tails the most recent OpenCode log file and
// extracts the most recent "message":"..." line. Used by PostClassify
// when the wrapper exits with just an exit-status string.
func resolveOpenCodeLogError() string {
	logDir := openCodeLogDir()
	if logDir == "" {
		return ""
	}
	latest, err := newestFile(logDir, "*.log")
	if err != nil || latest == "" {
		return ""
	}
	tail, err := tailFile(latest, 64*1024)
	if err != nil || tail == "" {
		return ""
	}
	return extractErrorMessage(tail)
}

func openCodeLogDir() string {
	if base := strings.TrimSpace(os.Getenv("OPENCODE_XDG_DATA_HOME")); base != "" {
		return filepath.Join(base, "opencode", "log")
	}
	if base := strings.TrimSpace(os.Getenv("OPENCODE_DATA_DIR")); base != "" {
		return filepath.Join(base, "xdg-data", "opencode", "log")
	}
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		return ""
	}
	return filepath.Join(root, "data", "opencode", "xdg-data", "opencode", "log")
}

func newestFile(dir, pattern string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil || len(matches) == 0 {
		return "", err
	}
	sort.Slice(matches, func(i, j int) bool {
		infoI, errI := os.Stat(matches[i])
		infoJ, errJ := os.Stat(matches[j])
		if errI != nil || errJ != nil {
			return matches[i] > matches[j]
		}
		return infoI.ModTime().After(infoJ.ModTime())
	})
	return matches[0], nil
}

func tailFile(path string, maxBytes int64) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return "", err
	}
	size := stat.Size()
	if size <= 0 {
		return "", nil
	}
	offset := size - maxBytes
	if offset < 0 {
		offset = 0
	}
	buf := make([]byte, size-offset)
	if _, err := file.ReadAt(buf, offset); err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return string(buf), nil
}

func extractErrorMessage(logs string) string {
	const messageKey = `"message":"`
	idx := strings.LastIndex(logs, messageKey)
	if idx == -1 {
		return ""
	}
	start := idx + len(messageKey)
	end := start
	for end < len(logs) {
		if logs[end] == '"' && logs[end-1] != '\\' {
			break
		}
		end++
	}
	if end <= start || end >= len(logs) {
		return ""
	}
	raw := `"` + logs[start:end] + `"`
	var decoded string
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return ""
	}
	return strings.TrimSpace(decoded)
}

// Compile-time interface checks.
var (
	_ Codec = (*OpenCode)(nil)
	_ State = (*opencodeState)(nil)
)
