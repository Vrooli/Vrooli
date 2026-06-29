// Package codecs — codex.go is the [Codec] implementation for OpenAI's
// Codex CLI (the `codex` binary invoked with `exec --json`).
//
// What this codec owns:
//   - CLI args + env shape (BuildArgs, BuildContinueArgs, BuildEnv)
//   - The Codex --json stream decoder (DecodeStreamLine, ParseTranscriptLine)
//   - Per-run state: thread_id (session id), captured run model
//   - Cost-event construction with optional pricing-service lookup
//   - Session-expiry detection from stderr
//
// The previous CodexRunner type and its dual JSON-stream/wrapper paths
// were consolidated here. The wrapper-fallback path (`resource-codex run …`
// with no event parsing) was deleted: it had no production caller after
// codex CLI direct-invocation became the default, and emitted unstructured
// log events without cost or message classification.
package codecs

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/fallback"

	"github.com/google/uuid"
)

// =============================================================================
// Constants
// =============================================================================

// CodexCLICommand is the binary name resolved on the host PATH.
const CodexCLICommand = "codex"

// CodexResourceCommand is the legacy Vrooli wrapper name kept around for
// transition-period process detection in the reconciler/terminator.
const CodexResourceCommand = "resource-codex"

const codexTagEnvKey = "CODEX_AGENT_TAG"

// The pricing seam ([PricingService], [PricingCostRequest],
// [PricingCostCalculation], [buildCostEvent]) lives in pricing.go — it is
// runner-agnostic; codex is currently its only consumer.

// =============================================================================
// Codec
// =============================================================================

// Codex is the [Codec] implementation for the codex CLI.
type Codex struct {
	baseCodec
	pricingService PricingService
	ollama         *ollamaLister
}

// CodexOption configures a Codex codec.
type CodexOption func(*Codex)

// WithPricingService injects a pricing service used when building cost
// events. Optional — when omitted, cost events carry token counts without
// dollar amounts. The pricing seam is runner-agnostic ([PricingService] in
// pricing.go); codex is currently its only consumer.
func WithPricingService(svc PricingService) CodexOption {
	return func(c *Codex) { c.pricingService = svc }
}

// codexBase is the identity shared by NewCodex and NewCodexForTest.
func codexBase() baseCodec {
	return baseCodec{
		runnerType:     domain.RunnerTypeCodex,
		binaryDesc:     "codex CLI",
		installHint:    "Run: vrooli resource install codex",
		tagEnvKey:      codexTagEnvKey,
		continuePrefix: "codex",
		labels: Labels{
			StartMessage:         "Codex execution started",
			EndMessage:           "Codex execution completed",
			ContinueStartMessage: "Codex continuation started",
			ContinueEndMessage:   "Codex continuation completed",
		},
	}
}

// NewCodex resolves the `codex` binary on PATH and returns a codec ready
// to be wrapped in [core.NewRunner]. Returns a codec with Available=false
// when the binary is missing so the runner registry can register a stub.
func NewCodex(opts ...CodexOption) (*Codex, error) {
	c := &Codex{
		baseCodec: resolveBinary(codexBase(), CodexCLICommand),
		ollama:    newOllamaLister(),
	}
	c.newParser = c.NewTranscriptParser
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// NewCodexForTest returns a Codex codec with a fake binary path and
// Available=false. Used by codec tests that exercise BuildArgs / decode
// paths without launching a real process.
func NewCodexForTest() *Codex {
	c := &Codex{baseCodec: testBase(codexBase(), "/fake/codex", "test codex codec")}
	c.newParser = c.NewTranscriptParser
	return c
}

// Capabilities satisfies [Codec]. SupportedModels is the curated cloud list
// plus any locally-pulled Ollama models discovered via the cached lister
// (codex reaches them natively through `--oss --local-provider ollama`).
func (c *Codex) Capabilities() runner.Capabilities {
	models := []string{
		"gpt-5.5",
		"gpt-5.4",
		"gpt-5.4-mini",
		"gpt-5.3-codex",
		"gpt-5.3-codex-spark",
		"gpt-5.2",
	}
	models = append(models, c.ollama.list()...)

	return runner.Capabilities{
		SupportsMessages:         true,
		SupportsToolEvents:       true,
		SupportsCostTracking:     true,
		SupportsStreaming:        true, // codec only supports JSON-stream path
		SupportsCancellation:     true,
		SupportsContinuation:     true, // `codex exec resume <thread_id>`
		SupportsImageAttachments: true, // `codex exec -i/--image <FILE>`
		MaxTurns:                 0,
		SupportedModels:          models,
		SupportedFeatures:        []string{},
		AllowedExtraFlags:        []string{"--verbose"},
	}
}

// BuildEnv satisfies [Codec]. Codex reads model + limits from environment
// variables for backwards compatibility with the resource-codex wrapper.
// Binary resolution, availability, probing, labels, type, tag-key and
// continuation tags are provided by the embedded [baseCodec].
func (c *Codex) BuildEnv(tag string, extras map[string]string) []string {
	return standardBuildEnv(codexTagEnvKey, tag, extras, "CODEX_NON_INTERACTIVE=true")
}

// BuildPrompt satisfies [Codec]. Codex receives image attachments via the
// `-i/--image` flag (added in BuildArgs), not embedded in the prompt text, so
// the prompt is passed through unchanged.
func (c *Codex) BuildPrompt(prompt string, _ []runner.Attachment) string {
	return prompt
}

// BuildArgs satisfies [Codec]. Captures req.GetConfig().Model on state so
// cost events emitted from DecodeStreamLine can label themselves. An
// `ollama/`-prefixed model routes codex to its local OSS provider via
// `--oss --local-provider ollama`; image attachments are passed with `-i`.
func (c *Codex) BuildArgs(state State, req runner.ExecuteRequest) []string {
	cfg := req.GetConfig()
	model := strings.TrimSpace(cfg.Model)
	if s, ok := state.(*codexState); ok {
		s.runModel = model
	}

	args := []string{
		"exec",
		"--json",
		"--skip-git-repo-check",
	}

	// Network access policy determines Codex's internal sandbox mode.
	// SandboxConfig.Mode (overlayfs file isolation / bwrap-based agent
	// containment) is a separate concern handled at the orchestration
	// layer; this codec only sees the resolved RunConfig.
	switch cfg.NetworkAccess.Effective() {
	case domain.NetworkAccessNone:
		// `--full-auto` is deprecated on codex ≥0.135.0; the supported
		// equivalent for non-interactive exec is an explicit sandbox policy.
		args = append(args, "--sandbox", "workspace-write")
	default:
		args = append(args, "--dangerously-bypass-approvals-and-sandbox")
	}

	bareModel, isOllama := splitOllamaModel(model)
	if isOllama {
		args = append(args, "--oss", "--local-provider", "ollama")
	}
	if model != "" {
		args = append(args, "-m", bareModel)
	}
	if req.WorkingDir != "" {
		args = append(args, "-C", req.WorkingDir)
	}
	args = appendAttachmentFlags(args, "-i", req.Attachments)
	if extras, ok := cfg.ExtraFlags[domain.RunnerTypeCodex]; ok {
		args = append(args, extras...)
	}

	args = append(args, "-") // read prompt from stdin
	return args
}

// BuildContinueArgs satisfies [Codec]. `codex exec resume` carries the
// session-id positional argument and accepts an optional follow-up prompt
// inline (rather than via stdin).
func (c *Codex) BuildContinueArgs(state State, req runner.ContinueRequest) []string {
	model := strings.TrimSpace(req.GetConfig().Model)
	if s, ok := state.(*codexState); ok {
		s.runModel = model
	}
	args := []string{
		"exec", "resume",
		"--json",
		"--skip-git-repo-check",
		// `--full-auto` is deprecated on codex ≥0.135.0; use the explicit
		// sandbox policy instead (see BuildArgs).
		"--sandbox", "workspace-write",
	}
	if _, isOllama := splitOllamaModel(model); isOllama {
		args = append(args, "--oss", "--local-provider", "ollama")
	}
	args = appendAttachmentFlags(args, "-i", req.Attachments)
	args = append(args, req.SessionID)
	if strings.TrimSpace(req.Prompt) != "" {
		args = append(args, req.Prompt)
	}
	return args
}

// =============================================================================
// State
// =============================================================================

// codexState carries per-run mutable state through the stream decode loop.
type codexState struct {
	threadID string // captured from stream events; used as SessionID
	runModel string // captured at BuildArgs time for cost event labelling
}

func (s *codexState) SessionID() string { return s.threadID }

// NewState satisfies [Codec].
func (c *Codex) NewState() State { return &codexState{} }

// =============================================================================
// Stream-event types
// =============================================================================

// CodexStreamEvent represents a single event from Codex's --json output.
type CodexStreamEvent struct {
	Type     string          `json:"type"`
	ThreadID string          `json:"thread_id,omitempty"`
	Item     *CodexItem      `json:"item,omitempty"`
	Usage    *CodexUsage     `json:"usage,omitempty"`
	Error    *CodexError     `json:"error,omitempty"`
	Tool     *CodexToolEvent `json:"tool,omitempty"`
}

// CodexItem represents an item in the Codex stream.
type CodexItem struct {
	ID               string            `json:"id"`
	Type             string            `json:"type"`
	Text             string            `json:"text,omitempty"`
	Name             string            `json:"name,omitempty"`
	Input            json.RawMessage   `json:"input,omitempty"`
	Output           string            `json:"output,omitempty"`
	ExitCode         *int              `json:"exit_code,omitempty"`
	Command          string            `json:"command,omitempty"`
	AggregatedOutput string            `json:"aggregated_output,omitempty"`
	Changes          []CodexFileChange `json:"changes,omitempty"`
	Status           string            `json:"status,omitempty"`
}

// CodexFileChange represents a file modification in Codex's file_change event.
type CodexFileChange struct {
	Path string `json:"path"`
	Kind string `json:"kind,omitempty"`
}

// CodexUsage represents token usage in turn.completed events.
type CodexUsage struct {
	InputTokens       int `json:"input_tokens"`
	CachedInputTokens int `json:"cached_input_tokens,omitempty"`
	OutputTokens      int `json:"output_tokens"`
}

// CodexError represents an error in the stream.
type CodexError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// CodexToolEvent represents a tool-related event emitted at the top level.
type CodexToolEvent struct {
	Name   string          `json:"name"`
	Input  json.RawMessage `json:"input,omitempty"`
	Output string          `json:"output,omitempty"`
}

// =============================================================================
// Stream decoding
// =============================================================================

// DecodeStreamLine satisfies [Codec].
func (c *Codex) DecodeStreamLine(state State, runID uuid.UUID, line string) ([]*domain.RunEvent, error) {
	s, ok := state.(*codexState)
	if !ok {
		return nil, fmt.Errorf("codex: invalid state type %T", state)
	}

	streamEvent, ok := decodeCodexStreamEvent(line)
	if !ok {
		return nil, nil
	}

	// Capture thread_id for session continuation. Only the first one wins
	// — subsequent events on the same run carry the same id.
	if streamEvent.ThreadID != "" && s.threadID == "" {
		s.threadID = streamEvent.ThreadID
	}

	return c.parseCodexEvents(s, runID, streamEvent), nil
}

// ClassifyTerminalError satisfies [Codec]. Codex's failure shapes
// distinguish two distinct "thread not found" cases:
//
//   - On resume, the session id no longer maps to a live thread: a
//     plain "thread … not found" without rollout-writer context means
//     the session is genuinely gone → ErrCodeRunnerSessionExpired.
//   - During a live run, codex's rollout-recorder goroutine drops the
//     thread mid-stream. Codex emits this in two surface forms across
//     versions — the function-name form `record_rollout_items` and the
//     human-readable form `failed to record rollout items` (the latter
//     is what current codex prints; both are treated as the same
//     condition). The session id is still valid but the rollout state
//     is unrecoverable → ErrCodeRunnerSessionStateLost.
//
// Returning nil from this method (no recognised pattern) lets
// core.Runner fall back to ErrCodeRunnerExecution as before.
func (c *Codex) ClassifyTerminalError(stderr string, exitCode int) *domain.RunnerError {
	if !strings.Contains(stderr, "thread") || !strings.Contains(stderr, "not found") {
		return nil
	}
	cause := strings.TrimSpace(stderr)
	if strings.Contains(stderr, "record_rollout_items") || strings.Contains(stderr, "record rollout items") {
		return domain.NewRunnerSessionStateLostError(c.Type(), errors.New(cause))
	}
	return domain.NewRunnerSessionExpiredError(c.Type(), errors.New(cause))
}

// Classify satisfies [Codec]. Codex classification order:
//
//  1. Thread-not-found stderr — distinguishes ReasonSessionStateLost
//     (rollout-recorder dropped the thread mid-stream; mirrors
//     [ClassifyTerminalError]) from ReasonSessionExpired (session id
//     no longer maps to a live thread).
//  2. Residual TextClassifier — covers unknown/deprecated model,
//     auth, quota, network, etc.
//
// Returns nil only when stderr is empty and exitCode == 0.
func (c *Codex) Classify(stderr string, exitCode int) *fallback.ClassifiedError {
	if stderr == "" && exitCode == 0 {
		return nil
	}
	if strings.Contains(stderr, "thread") && strings.Contains(stderr, "not found") {
		cause := strings.TrimSpace(stderr)
		if strings.Contains(stderr, "record_rollout_items") || strings.Contains(stderr, "record rollout items") {
			return fallback.New(fallback.ReasonSessionStateLost, cause, nil)
		}
		return fallback.New(fallback.ReasonSessionExpired, cause, nil)
	}
	return fallback.NewTextClassifier().Classify(fallback.ClassifyInput{
		RunnerType: string(c.Type()),
		Stderr:     stderr,
		ExitCode:   exitCode,
	})
}

// UpdateMetrics satisfies [Codec].
func (c *Codex) UpdateMetrics(event *domain.RunEvent, metrics *runner.ExecutionMetrics, lastAssistant *string) {
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
	case *domain.CostEventData:
		// Token breakdown + cost from CostEventData. Codex emits one
		// cost event per turn, so we accumulate.
		metrics.TokensInput += data.InputTokens
		metrics.TokensOutput += data.OutputTokens
		metrics.CacheReadTokens += data.CacheReadTokens
		metrics.CacheCreationTokens += data.CacheCreationTokens
		metrics.CostEstimateUSD += data.TotalCostUSD
	case *domain.MetricEventData:
		// Legacy fallback for any MetricEventData that might still come.
		if data.Name == "tokens" {
			metrics.TokensOutput = int(data.Value)
		}
	}
}

// NewTranscriptParser satisfies [Codec]. Single-line parsing is provided by
// the embedded [baseCodec.ParseTranscriptLine], which delegates here.
func (c *Codex) NewTranscriptParser() runner.TranscriptParser {
	return &codexTranscriptParser{codec: c, state: &codexState{}}
}

type codexTranscriptParser struct {
	codec *Codex
	state *codexState
}

func (p *codexTranscriptParser) ParseTranscriptLine(runID uuid.UUID, line string) runner.TranscriptParseResult {
	streamEvent, ok := decodeCodexStreamEvent(line)
	if !ok {
		return runner.TranscriptParseResult{}
	}

	if streamEvent.ThreadID != "" {
		p.state.threadID = streamEvent.ThreadID
	}
	events := p.codec.parseCodexEvents(p.state, runID, streamEvent)

	result := runner.TranscriptParseResult{
		Events:    events,
		SessionID: streamEvent.ThreadID,
	}

	switch streamEvent.Type {
	case "turn.completed":
		result.Terminal = &runner.TranscriptTerminal{Success: true, ExitCode: 0}
	case "error":
		if streamEvent.Error != nil {
			result.Terminal = &runner.TranscriptTerminal{
				Success:      false,
				ExitCode:     1,
				ErrorMessage: runner.StripANSI(streamEvent.Error.Message),
			}
		}
	}
	return result
}

// =============================================================================
// Stream parser
// =============================================================================

// decodeCodexStreamEvent unwraps a single stdout/transcript line into a
// *CodexStreamEvent. Returns ok=false for blank or non-JSON lines.
func decodeCodexStreamEvent(line string) (*CodexStreamEvent, bool) {
	line = strings.TrimSpace(line)
	// Some Codex builds prefix events with `data:` (SSE-style).
	if strings.HasPrefix(line, "data:") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	}
	if line == "" || line[0] != '{' {
		return nil, false
	}
	var ev CodexStreamEvent
	if err := json.Unmarshal([]byte(line), &ev); err != nil {
		return nil, false
	}
	return &ev, true
}

// parseCodexEvents dispatches a parsed CodexStreamEvent to the
// per-event-type handlers and returns zero or more domain events.
func (c *Codex) parseCodexEvents(state *codexState, runID uuid.UUID, streamEvent *CodexStreamEvent) []*domain.RunEvent {
	events := []*domain.RunEvent{}

	// Top-level tool payloads (some Codex builds emit these outside item.completed).
	if streamEvent.Tool != nil && streamEvent.Item == nil {
		toolName := streamEvent.Tool.Name
		var input map[string]interface{}
		if streamEvent.Tool.Input != nil {
			_ = json.Unmarshal(streamEvent.Tool.Input, &input)
		}
		if len(input) > 0 {
			events = append(events, domain.NewToolCallEvent(runID, toolName, "", input))
		}
		if streamEvent.Tool.Output != "" {
			events = append(events, domain.NewToolResultEvent(runID, toolName, "", runner.StripANSI(streamEvent.Tool.Output), nil))
		}
		if len(events) > 0 {
			return events
		}
	}

	if strings.HasPrefix(streamEvent.Type, "item.") && streamEvent.Item != nil {
		return parseCodexItemEvents(runID, streamEvent.Item)
	}

	switch streamEvent.Type {
	case "thread.started":
		return []*domain.RunEvent{domain.NewLogEvent(runID, "debug", "Thread started: "+streamEvent.ThreadID)}

	case "turn.started":
		return []*domain.RunEvent{domain.NewLogEvent(runID, "debug", "Turn started")}

	case "turn.completed":
		if streamEvent.Usage != nil {
			return []*domain.RunEvent{buildCostEvent(runID, domain.RunnerTypeCodex, c.pricingService, state.runModel, usageTokens{
				InputTokens:     streamEvent.Usage.InputTokens,
				OutputTokens:    streamEvent.Usage.OutputTokens,
				CacheReadTokens: streamEvent.Usage.CachedInputTokens,
			})}
		}

	case "error":
		if streamEvent.Error != nil {
			return []*domain.RunEvent{domain.NewErrorEvent(
				runID,
				runner.StripANSI(streamEvent.Error.Code),
				runner.StripANSI(streamEvent.Error.Message),
				false,
			)}
		}
	}

	return nil
}

// parseCodexItemEvents handles item.* events (item.completed, item.started,
// etc.). Codex emits items for messages, reasoning, file_changes, tool
// calls/results, and command_executions.
func parseCodexItemEvents(runID uuid.UUID, item *CodexItem) []*domain.RunEvent {
	if item == nil {
		return nil
	}
	switch item.Type {
	case "agent_message":
		if item.Text != "" {
			return []*domain.RunEvent{domain.NewMessageEvent(runID, "assistant", runner.StripANSI(item.Text))}
		}
	case "reasoning":
		if item.Text != "" {
			return []*domain.RunEvent{domain.NewLogEvent(runID, "debug", "Reasoning: "+runner.StripANSI(item.Text))}
		}
	case "file_change":
		if len(item.Changes) > 0 {
			input := map[string]interface{}{"status": item.Status}
			files := make([]map[string]string, 0, len(item.Changes))
			for _, ch := range item.Changes {
				files = append(files, map[string]string{"path": ch.Path, "kind": ch.Kind})
			}
			input["files"] = files
			return []*domain.RunEvent{domain.NewToolCallEvent(runID, "file_change", "", input)}
		}
	case "tool_call":
		var input map[string]interface{}
		if item.Input != nil {
			_ = json.Unmarshal(item.Input, &input)
		}
		return []*domain.RunEvent{domain.NewToolCallEvent(runID, item.Name, "", input)}
	case "tool_result":
		var input map[string]interface{}
		if item.Input != nil {
			_ = json.Unmarshal(item.Input, &input)
		}
		events := []*domain.RunEvent{}
		if len(input) > 0 {
			events = append(events, domain.NewToolCallEvent(runID, item.Name, "", input))
		}
		events = append(events, domain.NewToolResultEvent(
			runID, item.Name, "",
			runner.StripANSI(item.Output), nil,
		))
		return events
	case "command_execution":
		return parseCodexCommandExecution(runID, item)
	}
	return nil
}

// parseCodexCommandExecution extracts shell commands as bash tool events.
func parseCodexCommandExecution(runID uuid.UUID, item *CodexItem) []*domain.RunEvent {
	const toolName = "bash"
	isTerminal := item.Status == "completed" ||
		item.Status == "failed" ||
		item.Status == "error" ||
		item.Status == "cancelled" ||
		item.Status == "timed_out"

	if isTerminal {
		events := make([]*domain.RunEvent, 0, 2)
		// For non-success terminal states, emit tool_call + tool_result
		// so failed commands retain command/status context. Successful
		// command completion only emits tool_result for backward-compat.
		if item.Command != "" && item.Status != "completed" {
			input := map[string]interface{}{
				"command":     item.Command,
				"status":      item.Status,
				"runner_tool": "command_execution",
			}
			events = append(events, domain.NewToolCallEvent(runID, toolName, "", input))
		}

		var errMsg error
		if item.ExitCode != nil && *item.ExitCode != 0 {
			errMsg = fmt.Errorf("command exited with code %d", *item.ExitCode)
		} else if item.Status != "completed" {
			errMsg = fmt.Errorf("command status: %s", item.Status)
		}

		output := item.AggregatedOutput
		if strings.TrimSpace(output) == "" {
			output = item.Output
		}
		events = append(events, domain.NewToolResultEvent(
			runID, toolName, "",
			runner.StripANSI(output), errMsg,
		))
		return events
	}

	if item.Command != "" {
		input := map[string]interface{}{
			"command":     item.Command,
			"status":      item.Status,
			"runner_tool": "command_execution",
		}
		return []*domain.RunEvent{domain.NewToolCallEvent(runID, toolName, "", input)}
	}
	return nil
}

// Compile-time interface checks.
var (
	_ Codec = (*Codex)(nil)
	_ State = (*codexState)(nil)
)
