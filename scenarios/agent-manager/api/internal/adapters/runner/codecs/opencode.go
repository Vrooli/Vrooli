// Package codecs — opencode.go is the [Codec] implementation for the
// OpenCode CLI (the raw `opencode` binary invoked with
// `run … --format json --print-logs`).
//
// What this codec owns:
//   - CLI args + env shape (BuildArgs, BuildContinueArgs, BuildEnv)
//   - The OpenCode JSON stream decoder (DecodeStreamLine, ParseTranscriptLine)
//   - Per-run state: captured session id, terminal step_finish marker
//   - step_finish handling — emits cost + message events, signals early
//     termination via OnEarlyTerminate when reason is terminal
//   - Log-file fallback for error messages when opencode exits with
//     just an exit-status string (PostClassify hook)
//
// The previous resource-opencode wrapper path (`resource-opencode run run …`)
// was deleted: the wrapper was refactored into a thin Go CLI wiring only
// lifecycle + permissions, so its `run`/`status` subcommands no longer
// exist. This codec now invokes the upstream binary directly, mirroring
// the codex and claude-code codecs.
package codecs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/fallback"

	"github.com/google/uuid"
)

// =============================================================================
// Constants
// =============================================================================

// OpenCodeCLICommand is the binary name resolved on the host PATH.
const OpenCodeCLICommand = "opencode"

// OpenCodeResourceCommand is the legacy Vrooli wrapper name kept around
// for transition-period process detection in the reconciler/terminator.
const OpenCodeResourceCommand = "resource-opencode"

const opencodeTagEnvKey = "OPENCODE_AGENT_TAG"

// =============================================================================
// Codec
// =============================================================================

// OpenCode is the [Codec] implementation for the opencode CLI.
type OpenCode struct {
	baseCodec
	ollama *ollamaLister
}

// opencodeBase is the identity shared by NewOpenCode and NewOpenCodeForTest.
func opencodeBase() baseCodec {
	return baseCodec{
		runnerType:     domain.RunnerTypeOpenCode,
		binaryDesc:     "opencode CLI",
		installHint:    "Run: vrooli resource install opencode",
		tagEnvKey:      opencodeTagEnvKey,
		continuePrefix: "opencode",
		labels: Labels{
			StartMessage:         "OpenCode execution started",
			EndMessage:           "OpenCode execution completed",
			ContinueStartMessage: "OpenCode continuation started",
			ContinueEndMessage:   "OpenCode continuation completed",
		},
	}
}

// NewOpenCode resolves the `opencode` binary on PATH and returns a codec
// ready to be wrapped in [core.NewRunner]. Returns a codec with
// Available=false (rather than an error) when the binary is missing so the
// runner registry can register a stub instead. A deep health-check is
// intentionally avoided — mirrors NewCodex (LookPath only); the
// authoritative "binary is broken" signal comes from runtime
// classification on the first real invocation.
func NewOpenCode() (*OpenCode, error) {
	c := &OpenCode{
		baseCodec: resolveBinary(opencodeBase(), OpenCodeCLICommand),
		ollama:    newOllamaLister(),
	}
	c.newParser = c.NewTranscriptParser
	return c, nil
}

// NewOpenCodeForTest returns an OpenCode codec with a fake binary path
// and Available=false. Used by codec tests that exercise BuildArgs /
// decode paths without launching a real process.
func NewOpenCodeForTest() *OpenCode {
	c := &OpenCode{baseCodec: testBase(opencodeBase(), "/fake/opencode", "test opencode codec")}
	c.newParser = c.NewTranscriptParser
	return c
}

// Capabilities satisfies [Codec]. It reports only locally-pulled Ollama models
// discovered through OpenCode's first-class provider block; resource role
// resolution owns concrete coding-agent model selection.
func (c *OpenCode) Capabilities() runner.Capabilities {
	return runner.Capabilities{
		SupportsMessages:         true,
		SupportsToolEvents:       true,
		SupportsCostTracking:     true, // step_finish tokens + cost parsed in handleStepFinish
		SupportsStreaming:        true, // JSON event stream via `run --format json`
		SupportsCancellation:     true,
		SupportsContinuation:     true, // `--session <id>`
		SupportsImageAttachments: true, // `opencode run -f/--file <FILE>`
		MaxTurns:                 0,
		SupportedModels:          c.ollama.list(),
		SupportsRunnerDefault:    true,
		DynamicModelPrefixes:     []string{ollamaModelPrefix},
		SupportedFeatures:        []string{},
		AllowedExtraFlags:        []string{"--verbose"},
	}
}

// ProbeModel satisfies [Codec]. Lightweight by design — it never makes a
// billable vendor call. Beyond the availability check it does a quota-free
// LOCAL validation: cloud models are confirmed against opencode's catalog
// cache and Ollama models against the locally-pulled tag list, so an
// undeclared/dead model surfaces a clear message instead of opencode's
// opaque ProviderModelNotFoundError. Any uncertain case degrades to nil.
func (c *OpenCode) ProbeModel(ctx context.Context, modelID string) error {
	if available, msg := c.Available(ctx); !available {
		return fmt.Errorf("opencode unavailable: %s", msg)
	}
	var ollamaModels []string
	if c.ollama != nil {
		ollamaModels = c.ollama.list()
	}
	return validateOpenCodeModel(modelID, ollamaModels, opencodeCatalogPath())
}

// opencodeCatalogPath returns the path to opencode's local model catalog
// cache (populated by the opencode binary from models.dev). Empty when the
// user cache dir cannot be resolved.
func opencodeCatalogPath() string {
	dir, err := os.UserCacheDir()
	if err != nil || dir == "" {
		return ""
	}
	return filepath.Join(dir, "opencode", "models.json")
}

// validateOpenCodeModel reports an error ONLY when it can positively confirm
// the model id is unusable. Every uncertain case — empty model, missing
// catalog cache, unreachable Ollama daemon, unknown provider — degrades to
// nil so preflight never false-rejects a run. ollamaModels is the list of
// locally-pulled models in "ollama/<tag>" form; catalogPath points at the
// opencode cloud-catalog cache.
func validateOpenCodeModel(modelID string, ollamaModels []string, catalogPath string) error {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil // runner-default sentinel — accept
	}
	// Local Ollama models resolve through opencode's first-class ollama
	// provider, so the tag must actually be pulled on this host.
	if bare, isOllama := strings.CutPrefix(modelID, ollamaModelPrefix); isOllama {
		if len(ollamaModels) == 0 {
			return nil // daemon unreachable / cold cache — cannot confirm
		}
		for _, m := range ollamaModels {
			if m == ollamaModelPrefix+bare {
				return nil
			}
		}
		return fmt.Errorf("ollama model %q is not pulled locally — run `ollama pull %s` or choose an installed model", modelID, bare)
	}
	// Cloud models are "<provider>/<rest>"; confirm against the catalog.
	provider, rest, ok := strings.Cut(modelID, "/")
	if !ok || provider == "" || rest == "" || catalogPath == "" {
		return nil
	}
	data, err := os.ReadFile(catalogPath)
	if err != nil {
		return nil // no catalog cache — cannot confirm
	}
	var catalog map[string]struct {
		Models map[string]json.RawMessage `json:"models"`
	}
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil
	}
	prov, ok := catalog[provider]
	if !ok {
		return nil // unknown provider — cannot confirm absence
	}
	if _, ok := prov.Models[rest]; ok {
		return nil
	}
	return fmt.Errorf("model %q is not available from provider %q in the opencode catalog — opencode would fail with ProviderModelNotFoundError; check the model id (e.g. \"openrouter/<vendor>/<model>\") or refresh the catalog", modelID, provider)
}

// BuildEnv satisfies [Codec]. The raw opencode binary reads its config
// from the default XDG location (~/.config/opencode/opencode.json) and its
// auth from ~/.local/share/opencode/auth.json — the same locations the
// resource-opencode `permissions` adapter and the resource install path
// write to. We deliberately do NOT override XDG_CONFIG_HOME/OPENCODE_CONFIG
// here so the managed config/auth is the single source of truth.
func (c *OpenCode) BuildEnv(tag string, extras map[string]string) []string {
	return standardBuildEnv(opencodeTagEnvKey, tag, extras, "OPENCODE_NON_INTERACTIVE=true")
}

// BuildPrompt satisfies [Codec]. OpenCode passes the prompt as a CLI
// argument (in BuildArgs) and image attachments via `-f` flags; stdin is
// unused.
func (c *OpenCode) BuildPrompt(_ string, _ []runner.Attachment) string { return "" }

// BuildArgs satisfies [Codec]. The prompt is on the command line so the
// caller MUST close stdin — core.Runner does that automatically when the
// codec returns "" from BuildPrompt. Image attachments are passed via
// `-f/--file` (one per attachment).
func (c *OpenCode) BuildArgs(state State, req runner.ExecuteRequest) []string {
	args := []string{
		"run",
		req.EffectivePrompt(),
		"--format", "json",
		"--print-logs", // surface logs to stderr for the PostClassify fallback
	}
	// `opencode run` attaches to a shared OpenCode server that resolves its
	// own project directory and ignores the launched process's cwd. Without
	// --dir the session executes in the server's directory (often the repo
	// root), so an in-place run scoped to one path can write files into a
	// different tree entirely. Pin the session to the run's WorkingDir.
	if dir := strings.TrimSpace(req.WorkingDir); dir != "" {
		args = append(args, "--dir", dir)
		if s, ok := state.(*opencodeState); ok {
			s.workingDir = dir
		}
	}
	cfg := req.GetConfig()
	if cfg.Model != "" {
		args = append(args, "-m", cfg.Model)
	}
	args = appendAttachmentFlags(args, "-f", req.Attachments)
	if extras, ok := cfg.ExtraFlags[domain.RunnerTypeOpenCode]; ok {
		args = append(args, extras...)
	}
	return args
}

// BuildContinueArgs satisfies [Codec]. `opencode run --session <id>`
// resumes an existing session with a follow-up prompt and any image
// attachments via `-f/--file`.
func (c *OpenCode) BuildContinueArgs(state State, req runner.ContinueRequest) []string {
	args := []string{
		"run",
		req.Prompt,
		"--session", req.SessionID,
		"--format", "json",
		"--print-logs",
	}
	// Pin the resumed session to the run's WorkingDir — see BuildArgs.
	if dir := strings.TrimSpace(req.WorkingDir); dir != "" {
		args = append(args, "--dir", dir)
		if s, ok := state.(*opencodeState); ok {
			s.workingDir = dir
		}
	}
	args = appendAttachmentFlags(args, "-f", req.Attachments)
	return args
}

// =============================================================================
// State
// =============================================================================

// opencodeState carries per-run mutable state.
type opencodeState struct {
	sessionID   string
	stepTermina bool // set by step_finish parsing when reason is terminal
	// workingDir is the run's pinned --dir (absolute). Stashed by
	// BuildArgs/BuildContinueArgs so the stream decoder can reject tool
	// results whose target path resolves outside it (defense-in-depth
	// against a model fabricating an absolute path — see toolTargetOutsideDir).
	// Empty (e.g. transcript replay, no resolved dir) disables the guard.
	workingDir string
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

// openCodeNoOpExitCode is the synthetic exit code stamped on a run that
// exited cleanly but executed zero tool calls (a no-op). The process really
// exited 0; this marks the reclassified failure so it is not mistaken for a
// genuine non-zero process exit.
const openCodeNoOpExitCode = 1

// openCodeNoOpErrorMessage explains a zero-tool-call no-op run, adding a
// model/template hint when the agent emitted a tool call as plain text.
func openCodeNoOpErrorMessage(summary *domain.RunSummary) string {
	base := "opencode run made no tool calls — the agent took no action (no files read or changed), so the run did no work"
	if summary != nil && looksLikeUnexecutedToolCall(summary.Description) {
		return base + "; the model emitted a tool call as text instead of executing it. " +
			"This happens with Ollama models whose template does not return structured tool_calls; " +
			"use a tool-calling-capable model such as gemma4:12b, llama3.1, llama3.2, or mistral, or an OpenRouter model"
	}
	return base
}

// openCodeNoEffectErrorMessage explains a run that issued tool calls but had
// none complete successfully — the agent attempted actions that did not land
// (e.g. a write to a non-existent/wrong directory). Distinct from the
// zero-tool-call no-op: here the model DID try to act, it just failed to.
func openCodeNoEffectErrorMessage(toolCalls int, summary *domain.RunSummary) string {
	base := fmt.Sprintf("opencode run made %d tool call(s) but none completed successfully — "+
		"the agent's actions did not land (e.g. a write to a non-existent or wrong directory), "+
		"so the run did no effective work", toolCalls)
	if summary != nil && looksLikeUnexecutedToolCall(summary.Description) {
		return base + "; the model emitted a tool call as text instead of executing it"
	}
	return base
}

// looksLikeUnexecutedToolCall reports whether text is (just) a tool call
// rendered as JSON — an object carrying a function name and arguments,
// optionally wrapped in a ``` fence. A well-behaved run never leaves a raw
// tool-call blob as its final assistant message, so this is a strong signal
// the model narrated a tool call instead of invoking it.
func looksLikeUnexecutedToolCall(text string) bool {
	t := strings.TrimSpace(text)
	if strings.HasPrefix(t, "```") {
		if i := strings.IndexByte(t, '\n'); i >= 0 {
			t = t[i+1:]
		}
		t = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(t), "```"))
	}
	if !strings.HasPrefix(t, "{") {
		return false
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal([]byte(t), &obj); err != nil {
		return false
	}
	if _, hasName := obj["name"]; !hasName {
		return false
	}
	_, hasArgs := obj["arguments"]
	if !hasArgs {
		_, hasArgs = obj["parameters"]
	}
	return hasArgs
}

// PostClassify satisfies [Codec]. When the wrapper exits with only an
// exit-status string (typical for opencode subprocess crashes), tail
// the latest log file and substitute the most recent error message.
func (c *OpenCode) PostClassify(_ State, result *runner.ExecuteResult) {
	if result == nil {
		return
	}
	if result.Success {
		// A clean exit that produced no successful tool result did no
		// observable work. For an agentic coding runner this is a no-op,
		// not a success — even reading a file is a tool call with a result.
		// Two shapes both slip past a bare exit-code check:
		//   - zero tool calls:        the agent took no action at all.
		//   - calls but no successes: the agent attempted actions that never
		//     landed — e.g. a write to a hallucinated/non-existent directory,
		//     or (with some Ollama model templates) a tool call narrated as
		//     message text that opencode never executed.
		// Reclassify so the run reports as failed rather than as a silent
		// false success.
		if result.Metrics.SuccessfulToolResults == 0 {
			result.Success = false
			result.ExitCode = openCodeNoOpExitCode
			if result.Metrics.ToolCallCount == 0 {
				result.ErrorMessage = openCodeNoOpErrorMessage(result.Summary)
			} else {
				result.ErrorMessage = openCodeNoEffectErrorMessage(result.Metrics.ToolCallCount, result.Summary)
			}
		}
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
		// A result that reported success is proof the agent's action
		// actually landed (file read/written/edited). PostClassify reads
		// this to tell real work from a silent no-op.
		if data.Success {
			metrics.SuccessfulToolResults++
		}
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

// NewTranscriptParser satisfies [Codec]. Single-line parsing is provided by
// the embedded [baseCodec.ParseTranscriptLine], which delegates here.
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
			return parseOpenCodeToolUse(runID, streamEvent.Part, state.workingDir), nil
		}

	case "tool_result", "tool-result":
		if streamEvent.Part != nil {
			return parseOpenCodeToolResult(runID, streamEvent.Part, state.workingDir), nil
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
			return parseOpenCodeToolUse(runID, streamEvent.Part, state.workingDir), nil
		case "tool-result", "tool_result":
			return parseOpenCodeToolResult(runID, streamEvent.Part, state.workingDir), nil
		}
	}

	return []*domain.RunEvent{domain.NewLogEvent(
		runID, "debug", fmt.Sprintf("OpenCode event [%s]", streamEvent.Type),
	)}, nil
}

// parseOpenCodeToolUse extracts a tool_call (and possibly a bundled
// tool_result when state.status == completed) from a part. workingDir is the
// run's pinned --dir; a completed write/edit whose target resolves outside it
// is downgraded to an error result (see toolTargetOutsideDir).
func parseOpenCodeToolUse(runID uuid.UUID, part *OpenCodePart, workingDir string) []*domain.RunEvent {
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
		} else if guardErr := toolTargetOutsideDir(toolName, input, workingDir); guardErr != nil {
			errMsg = guardErr
		}
		events = append(events, domain.NewToolResultEvent(runID, toolName, part.CallID, output, errMsg))
		return events
	}
	return []*domain.RunEvent{domain.NewToolCallEvent(runID, toolName, "", input)}
}

// toolTargetOutsideDir reports an error when a mutating tool's target path
// resolves outside the run's pinned working directory. It is the
// defense-in-depth half of the grounding fix: even with the directory in the
// model's context, a weak model can still emit a write to a fabricated
// absolute path. Such a result did not land useful work inside the run's
// scope, so it must NOT count as a successful tool result.
//
// Only mutating tools (write/edit/patch) are guarded — reads and shell
// commands legitimately touch paths elsewhere. An empty workingDir, a missing
// path, or a relative path (resolves under the dir by definition) all pass.
func toolTargetOutsideDir(toolName string, input map[string]interface{}, workingDir string) error {
	dir := strings.TrimSpace(workingDir)
	if dir == "" || input == nil {
		return nil
	}
	if !isMutatingOpenCodeTool(toolName) {
		return nil
	}
	target := openCodeToolTargetPath(input)
	if target == "" || !filepath.IsAbs(target) {
		return nil // no target, or relative → resolves under the dir
	}
	rel, err := filepath.Rel(dir, filepath.Clean(target))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("tool %q targeted %q outside the run's working directory %q — "+
			"the write did not land in scope and is rejected as ineffective", toolName, target, dir)
	}
	return nil
}

// isMutatingOpenCodeTool reports whether an opencode tool name writes to the
// filesystem (and thus has a guardable target path). Matched
// case-insensitively against the file-mutating tool family.
func isMutatingOpenCodeTool(toolName string) bool {
	switch strings.ToLower(strings.TrimSpace(toolName)) {
	case "write", "edit", "patch", "multiedit", "apply_patch", "applypatch":
		return true
	default:
		return false
	}
}

// openCodeToolTargetPath extracts the filesystem target from a tool input.
// opencode's file tools key it as "filePath"; tolerate "path"/"file" too.
func openCodeToolTargetPath(input map[string]interface{}) string {
	for _, key := range []string{"filePath", "path", "file"} {
		if v, ok := input[key]; ok {
			if s, ok := v.(string); ok {
				if s = strings.TrimSpace(s); s != "" {
					return s
				}
			}
		}
	}
	return ""
}

// parseOpenCodeToolResult extracts a tool_result (and possibly a
// preceding tool_call when state.input is non-nil) from a part.
func parseOpenCodeToolResult(runID uuid.UUID, part *OpenCodePart, workingDir string) []*domain.RunEvent {
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
	if part.State != nil {
		if guardErr := toolTargetOutsideDir(toolName, part.State.Input, workingDir); guardErr != nil {
			events = append(events, domain.NewToolResultEvent(runID, toolName, part.CallID, "", guardErr))
			return events
		}
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

// openCodeLogDir resolves the directory raw opencode writes its log files
// to. opencode honours XDG_DATA_HOME and otherwise defaults to
// ~/.local/share/opencode/log — the same default-XDG location BuildEnv
// relies on (no vrooli-scoped override).
func openCodeLogDir() string {
	if base := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); base != "" {
		return filepath.Join(base, "opencode", "log")
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".local", "share", "opencode", "log")
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
