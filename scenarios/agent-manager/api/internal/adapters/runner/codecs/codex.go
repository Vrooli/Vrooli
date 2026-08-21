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
	"math"
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

// ToolCapabilityMap declares Codex's native tool vocabulary at the codec
// boundary; runsignal consumes the same closed capability labels.
func (c *Codex) ToolCapabilityMap() map[string]string {
	return map[string]string{"bash": "shell", "command_execution": "shell", "apply_patch": "file-edit", "file_change": "file-edit", "read": "file-read", "grep": "search", "wait": "wait", "update_plan": "plan", "task": "delegate"}
}

// Codex exposes no per-launch native tool allowlist.
var codexToolTranslations = map[domain.CanonicalTool]string{}

// CodexOption configures a Codex codec.
type CodexOption func(*Codex)

// WithPricingService injects a pricing service used when building cost
// events. Optional — when omitted, cost events carry token counts without
// dollar amounts. The pricing seam is runner-agnostic ([PricingService] in
// pricing.go); codex is currently its only consumer.
func WithPricingService(svc PricingService) CodexOption {
	return func(c *Codex) { c.pricingService = svc }
}

// HasChargeSource reports whether production composition supplied the pricing
// dependency required by Codex's token-only stream.
func (c *Codex) HasChargeSource() bool { return c.pricingService != nil }

// codexBase is the identity shared by NewCodex and NewCodexForTest.
func codexBase() baseCodec {
	return baseCodec{
		runnerType:     domain.RunnerTypeCodex,
		binaryDesc:     "codex CLI",
		installHint:    "Run: vrooli resource install codex",
		tagEnvKey:      codexTagEnvKey,
		continuePrefix: "codex",
		goalStatus:     codexGoalStatus,
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

// NewCodexForTestWithBinary is a test-only constructor for process replay.
func NewCodexForTestWithBinary(path string) *Codex {
	c := NewCodexForTest()
	c.binaryPath, c.available = path, true
	return c
}

// Capabilities satisfies [Codec]. It reports only locally-pulled Ollama models
// discovered via the cached lister; resource role resolution owns concrete
// coding-agent model selection.
func (c *Codex) Capabilities() runner.Capabilities {
	return runner.Capabilities{
		SpawnCapabilities: []runner.SpawnCapability{
			{ExecutionMode: "codec_pipe", SandboxModes: []string{"protected", "tracking", "off"}},
			{ExecutionMode: "interactive", SandboxModes: []string{"tracking", "off"}, NativeObjective: true},
		},
		SupportsMessages:         true,
		SupportsToolEvents:       true,
		SupportsCostTracking:     true,
		SupportsStreaming:        true, // codec only supports JSON-stream path
		SupportsCancellation:     true,
		SupportsContinuation:     true, // `codex exec resume <thread_id>`
		SupportsWarmIteration:    true,
		SupportsImageAttachments: true,  // `codex exec -i/--image <FILE>`
		SupportsToolRestriction:  false, // Codex has no per-launch allowlist for its native tools.
		ToolRestrictionMappings:  canonicalToolMappings(codexToolTranslations),
		SupportsEffort:           true,
		// Codex config documents minimal, low, medium, high, and xhigh. The
		// portable scale has no minimal level, so max must not be claimed here.
		EffortMappings:        map[string]string{"low": "model_reasoning_effort=low", "medium": "model_reasoning_effort=medium", "high": "model_reasoning_effort=high", "xhigh": "model_reasoning_effort=xhigh"},
		MaxTurns:              0,
		SupportedModels:       c.ollama.list(),
		SupportsRunnerDefault: true,
		DynamicModelPrefixes:  []string{ollamaModelPrefix},
		SupportedFeatures:     []string{},
		AllowedExtraFlags:     []string{"--verbose", "-c"},
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

func (c *Codex) ControlArgs(cfg *domain.RunConfig) ([]string, error) { return codexControlArgs(cfg) }

// BuildArgs satisfies [Codec]. Captures req.GetConfig().Model on state so
// cost events emitted from DecodeStreamLine can label themselves. An
// `ollama/`-prefixed model routes codex to its local OSS provider via
// `--oss --local-provider ollama`; image attachments are passed with `-i`.
func (c *Codex) BuildArgs(state State, req runner.ExecuteRequest) []string {
	cfg := req.GetConfig()
	model := strings.TrimSpace(cfg.Model)
	if s, ok := state.(*codexState); ok {
		s.runModel = model
		s.billing = cfg.Billing
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

	// Runner-native model, effort, and tool controls are centralized so the
	// interactive path cannot drift from codec-pipe execution.
	controlArgs, _ := c.ControlArgs(cfg)
	args = append(args, controlArgs...)
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
		s.billing = req.GetConfig().Billing
	}
	args := []string{
		"exec", "resume",
		"--json",
		"--skip-git-repo-check",
	}
	controlArgs, _ := c.ControlArgs(req.GetConfig())
	args = append(args, controlArgs...)
	// `codex exec resume` has no working-directory flag. It resumes from the
	// session recorded in CODEX_HOME; the launcher supplies the working dir.
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
	threadID         string // captured from stream events; used as SessionID
	runModel         string // captured at BuildArgs time for cost event labelling
	billing          domain.BillingSnapshot
	turn             int
	lastMessageID    string
	lastMessage      string
	lastMessageEvent *domain.RunEvent
	lastRolloutUsage *CodexUsage
	retainUser       bool
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
	ID     string          `json:"id,omitempty"`
	CallID string          `json:"call_id,omitempty"`
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
		if data.Role == "assistant" && !data.EvidenceOnly {
			*lastAssistant = data.Content
			metrics.TurnsUsed++
		}
	case *domain.ToolCallEventData:
		metrics.ToolCallCount++
	case *domain.UsageEventData:
		// Codex emits one usage event per turn, so we accumulate it.
		metrics.TokensInput += data.InputTokens
		metrics.TokensOutput += data.OutputTokens
		metrics.CacheReadTokens += data.CacheReadTokens
		metrics.CacheCreationTokens += data.CacheCreationTokens
	case *domain.ChargeEventData:
		if data.AmountMicroUSD != nil {
			metrics.CostEstimateUSD += float64(*data.AmountMicroUSD) / 1_000_000
		}
	case *domain.MetricEventData:
		// Legacy fallback for any MetricEventData that might still come.
		if data.Name == "tokens" {
			metrics.TokensOutput = int(data.Value)
		}
	}
}

// ExtractCommand extends the shared explicit-field extraction for Codex's
// interactive wrapper tool. Imported Codex sessions can record the command
// as JavaScript source passed to tools.exec_command rather than as a native
// command field. The wrapper and its named cmd/command property are both
// required; arbitrary prose is never interpreted as a command.
func (c *Codex) ExtractCommand(input map[string]any) runner.CommandExtraction {
	extracted := c.baseCodec.ExtractCommand(input)
	if extracted.Command != "" || len(extracted.Args) > 0 || extracted.Reason == "" {
		return extracted
	}
	raw, ok := input["input"].(string)
	if !ok {
		return runner.CommandExtraction{Reason: "codex tool input has no command-bearing wrapper"}
	}
	if !strings.Contains(raw, "exec_command") {
		return runner.CommandExtraction{Reason: "codex tool input has no command-bearing wrapper"}
	}
	for _, key := range []string{`"cmd"`, `"command"`, "cmd", "command"} {
		for at := strings.Index(raw, key); at >= 0; {
			rest := raw[at+len(key):]
			colon := strings.IndexByte(rest, ':')
			if colon < 0 {
				break
			}
			value := strings.TrimSpace(rest[colon+1:])
			var command string
			if err := json.NewDecoder(strings.NewReader(value)).Decode(&command); err == nil && strings.TrimSpace(command) != "" {
				return runner.CommandExtraction{Command: strings.TrimSpace(command), Reason: "codex exec_command wrapper"}
			}
			next := strings.Index(rest[len(key):], key)
			if next < 0 {
				break
			}
			at += len(key) + next
		}
	}
	return runner.CommandExtraction{Reason: "codex exec_command wrapper has no command argument"}
}

// NewTranscriptParser satisfies [Codec]. Single-line parsing is provided by
// the embedded [baseCodec.ParseTranscriptLine], which delegates here.
func (c *Codex) NewTranscriptParser() runner.TranscriptParser {
	return &codexTranscriptParser{codec: c, state: &codexState{}}
}

func (p *codexTranscriptParser) SetTranscriptModel(model string) {
	p.state.runModel = strings.TrimSpace(model)
	if p.state.runModel == "" {
		p.state.runModel = "unknown"
	}
}

func (p *codexTranscriptParser) SetTranscriptRetention(retain bool) { p.state.retainUser = retain }

type codexTranscriptParser struct {
	codec *Codex
	state *codexState
}

func (p *codexTranscriptParser) ParseTranscriptLine(runID uuid.UUID, line string) runner.TranscriptParseResult {
	// On-disk interactive rollout dialect: every line wraps as
	// {"type":"event_msg"|"response_item"|"session_meta"|"turn_context",
	// "payload":{…}}, foreign to the flat `exec --json` decoder. Detect and
	// handle it before the stdout path (design §3). A non-rollout line falls
	// through to the exec-json decoder below, so pipe-mode replay is unchanged.
	if res, ok := p.parseRolloutLine(runID, line); ok {
		return res
	}

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
		Timestamp: transcriptLineTimestamp(line),
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
// On-disk rollout dialect (interactive runs)
// =============================================================================

// codexRolloutLine is the outer envelope of a codex on-disk rollout record
// ($CODEX_HOME/sessions/YYYY/MM/DD/rollout-*.jsonl). Every line is one of a
// small set of wrapper types carrying a typed payload.
type codexRolloutLine struct {
	Type    string              `json:"type"`
	Payload codexRolloutPayload `json:"payload"`
}

// codexRolloutPayload is the union of the payload fields this codec maps.
// Unused fields for a given payload type stay zero.
type codexRolloutPayload struct {
	Type string `json:"type"`
	// session_meta
	ID string `json:"id"`
	// agent_message / user_message
	Message string `json:"message"`
	// function_call / custom_tool_call
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // function_call: JSON-encoded args
	Input     string `json:"input"`     // custom_tool_call: raw tool input
	CallID    string `json:"call_id"`
	// function_call_output / custom_tool_call_output
	Output string `json:"output"`
	// turn_aborted
	Reason string               `json:"reason"`
	Info   *codexTokenCountInfo `json:"info"`
}

type codexTokenCountInfo struct {
	LastTokenUsage  *CodexUsage `json:"last_token_usage"`
	TotalTokenUsage *CodexUsage `json:"total_token_usage"`
}

// rolloutWrapperTypes are the outer `type` values that mark a line as the
// on-disk rollout dialect rather than the flat exec-json stdout dialect.
var rolloutWrapperTypes = map[string]bool{
	"event_msg":     true,
	"response_item": true,
	"session_meta":  true,
	"turn_context":  true,
}

// parseRolloutLine maps one on-disk rollout record to the same domain events
// the exec-json decoder emits. ok=false means the line is not a rollout record
// (blank, non-JSON, or a flat exec-json line) and the caller should fall back
// to the stdout decoder. The mapping mirrors parseCodexEvents:
//   - event_msg.agent_message           → assistant MessageEvent
//   - event_msg.user_message            → suppressed (orchestrator owns the prompt)
//   - response_item.function_call       → ToolCallEvent
//   - response_item.custom_tool_call    → ToolCallEvent
//   - response_item.*_output            → ToolResultEvent
//   - event_msg.task_complete/turn_completed → terminal success
//   - event_msg.turn_aborted            → terminal (not success)
//   - event_msg.error                   → error event + terminal failure
//   - session_meta.id                   → SessionID (thread id)
func (p *codexTranscriptParser) parseRolloutLine(runID uuid.UUID, line string) (runner.TranscriptParseResult, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || trimmed[0] != '{' {
		return runner.TranscriptParseResult{}, false
	}
	var rl codexRolloutLine
	if err := json.Unmarshal([]byte(trimmed), &rl); err != nil {
		return runner.TranscriptParseResult{}, false
	}
	if !rolloutWrapperTypes[rl.Type] {
		return runner.TranscriptParseResult{}, false
	}

	result := runner.TranscriptParseResult{Timestamp: transcriptLineTimestamp(line)}
	pl := rl.Payload

	switch rl.Type {
	case "session_meta":
		if pl.ID != "" {
			p.state.threadID = pl.ID
			result.SessionID = pl.ID
		}
		return result, true

	case "turn_context":
		p.state.turn++
		return result, true

	case "response_item":
		result.Events = parseCodexRolloutItem(runID, pl)
		return result, true

	case "event_msg":
		switch pl.Type {
		case "token_count":
			if pl.Info != nil && (pl.Info.TotalTokenUsage != nil || pl.Info.LastTokenUsage != nil) {
				if p.state.turn == 0 {
					p.state.turn = 1
				}
				usage := pl.Info.LastTokenUsage
				cumulative := pl.Info.TotalTokenUsage != nil
				if cumulative {
					usage = pl.Info.TotalTokenUsage
				}
				delta := rolloutUsageDelta(p.state, usage, cumulative)
				if delta.InputTokens > 0 || delta.OutputTokens > 0 || delta.CacheReadTokens > 0 {
					result.Events = markUsageTurn(buildCostEvents(runID, domain.RunnerTypeCodex, p.codec.pricingService, p.state.runModel, delta), p.state.turn)
				}
			}
		case "agent_message":
			if text := runner.StripANSI(strings.TrimSpace(pl.Message)); text != "" {
				p.state.lastMessage = text
				p.state.lastMessageID = pl.ID
				messageEvent := newCodexMessageEvent(runID, p.state, text, pl.ID, "event_msg.agent_message", false, "")
				p.state.lastMessageEvent = messageEvent
				result.Events = []*domain.RunEvent{messageEvent}
			}
		case "user_message":
			if p.state.retainUser && strings.TrimSpace(pl.Message) != "" {
				result.Events = []*domain.RunEvent{domain.NewProviderMessageEvent(runID, "user", runner.StripANSI(pl.Message), domain.MessageEventData{ProviderOrigin: "codex", ProviderEventType: "event_msg.user_message", RawEvidenceRef: "codex:event_msg.user_message"})}
			}
		case "task_complete", "turn_completed":
			if p.state.lastMessageEvent != nil {
				markProviderMessageTerminal(p.state.lastMessageEvent, pl.Type, "event_msg."+pl.Type, "codex:event_msg."+pl.Type)
				result.Events = append(result.Events, newProviderTerminalEvidence(runID, p.state.lastMessageEvent, pl.Type, "event_msg."+pl.Type, "codex:event_msg."+pl.Type))
			}
			result.Terminal = &runner.TranscriptTerminal{Success: true, ExitCode: 0}
		case "turn_aborted":
			msg := "turn aborted"
			if pl.Reason != "" {
				msg = "turn aborted: " + pl.Reason
			}
			result.Terminal = &runner.TranscriptTerminal{Success: false, ExitCode: 1, ErrorMessage: msg}
		case "error":
			if msg := runner.StripANSI(strings.TrimSpace(pl.Message)); msg != "" {
				result.Events = []*domain.RunEvent{domain.NewErrorEvent(runID, "execution_error", msg, false)}
				result.Terminal = &runner.TranscriptTerminal{Success: false, ExitCode: 1, ErrorMessage: msg}
			}
		}
		return result, true
	}
	return result, true
}

// rolloutUsageDelta converts Codex's cumulative usage snapshots to increments
// before they become durable usage events. total_token_usage is authoritative
// for archived rollouts; its input count includes cached input, so that subset
// is split out before persistence. A reset starts a new cumulative series.
func rolloutUsageDelta(state *codexState, current *CodexUsage, cumulative bool) usageTokens {
	if state == nil || current == nil {
		return usageTokens{}
	}
	previous := state.lastRolloutUsage
	state.lastRolloutUsage = &CodexUsage{InputTokens: current.InputTokens, CachedInputTokens: current.CachedInputTokens, OutputTokens: current.OutputTokens}
	if previous == nil || current.InputTokens < previous.InputTokens || current.CachedInputTokens < previous.CachedInputTokens || current.OutputTokens < previous.OutputTokens {
		return codexUsageDelta(current.InputTokens, current.OutputTokens, current.CachedInputTokens, cumulative)
	}
	return codexUsageDelta(current.InputTokens-previous.InputTokens, current.OutputTokens-previous.OutputTokens, current.CachedInputTokens-previous.CachedInputTokens, cumulative)
}

func codexUsageDelta(input, output, cached int, cumulative bool) usageTokens {
	if cumulative && input >= cached {
		input -= cached
	}
	return usageTokens{InputTokens: input, OutputTokens: output, CacheReadTokens: cached}
}

// parseCodexRolloutItem maps a response_item payload to tool events. Both the
// generic function_call/output and MCP-style custom_tool_call/output shapes
// carry a call_id, a name (on the call side), and a string output.
func parseCodexRolloutItem(runID uuid.UUID, pl codexRolloutPayload) []*domain.RunEvent {
	switch pl.Type {
	case "function_call":
		var input map[string]interface{}
		if pl.Arguments != "" {
			_ = json.Unmarshal([]byte(pl.Arguments), &input)
		}
		return []*domain.RunEvent{domain.NewToolCallEvent(runID, pl.Name, pl.CallID, input)}
	case "custom_tool_call":
		input := map[string]interface{}{}
		if pl.Input != "" {
			input["input"] = pl.Input
		}
		return []*domain.RunEvent{domain.NewToolCallEvent(runID, pl.Name, pl.CallID, input)}
	case "function_call_output", "custom_tool_call_output":
		return []*domain.RunEvent{domain.NewToolResultEvent(
			runID, "", pl.CallID, runner.StripANSI(pl.Output), nil,
		)}
	}
	// message / reasoning and other response items carry no distinct
	// domain event here (assistant text arrives via event_msg.agent_message).
	return nil
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
		toolCallID := streamEvent.Tool.CallID
		if toolCallID == "" {
			toolCallID = streamEvent.Tool.ID
		}
		var input map[string]interface{}
		if streamEvent.Tool.Input != nil {
			_ = json.Unmarshal(streamEvent.Tool.Input, &input)
		}
		if len(input) > 0 {
			events = append(events, domain.NewToolCallEvent(runID, toolName, toolCallID, input))
		}
		if streamEvent.Tool.Output != "" {
			events = append(events, domain.NewToolResultEvent(runID, toolName, toolCallID, runner.StripANSI(streamEvent.Tool.Output), nil))
		}
		if len(events) > 0 {
			return events
		}
	}

	if strings.HasPrefix(streamEvent.Type, "item.") && streamEvent.Item != nil {
		return parseCodexItemEvents(state, runID, streamEvent.Item)
	}

	switch streamEvent.Type {
	case "thread.started":
		return []*domain.RunEvent{domain.NewLogEvent(runID, "debug", "Thread started: "+streamEvent.ThreadID)}

	case "turn.started":
		state.turn++
		return []*domain.RunEvent{domain.NewLogEvent(runID, "debug", "Turn started")}

	case "turn.completed":
		if state.turn == 0 {
			state.turn = 1
		}
		if state.lastMessageEvent != nil {
			markProviderMessageTerminal(state.lastMessageEvent, "turn_completed", "turn.completed", "codex:turn.completed")
			events = append(events, newProviderTerminalEvidence(runID, state.lastMessageEvent, "turn_completed", "turn.completed", "codex:turn.completed"))
		}
		if streamEvent.Usage != nil {
			events = append(events, markUsageTurn(buildCostEvents(runID, domain.RunnerTypeCodex, c.pricingService, state.runModel, usageTokens{
				InputTokens:     streamEvent.Usage.InputTokens,
				OutputTokens:    streamEvent.Usage.OutputTokens,
				CacheReadTokens: streamEvent.Usage.CachedInputTokens,
			}, state.billing), state.turn)...)
		}
		return events

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
func parseCodexItemEvents(state *codexState, runID uuid.UUID, item *CodexItem) []*domain.RunEvent {
	if item == nil {
		return nil
	}
	switch item.Type {
	case "agent_message":
		if item.Text != "" {
			text := runner.StripANSI(item.Text)
			state.lastMessage = text
			state.lastMessageID = item.ID
			messageEvent := newCodexMessageEvent(runID, state, text, item.ID, "item.completed", false, "")
			state.lastMessageEvent = messageEvent
			return []*domain.RunEvent{messageEvent}
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
		return []*domain.RunEvent{domain.NewToolCallEvent(runID, item.Name, item.ID, input)}
	case "tool_result":
		var input map[string]interface{}
		if item.Input != nil {
			_ = json.Unmarshal(item.Input, &input)
		}
		events := []*domain.RunEvent{}
		if len(input) > 0 {
			events = append(events, domain.NewToolCallEvent(runID, item.Name, item.ID, input))
		}
		result := domain.NewToolResultEvent(
			runID, item.Name, item.ID,
			runner.StripANSI(item.Output), nil,
		)
		if data, ok := result.Data.(*domain.ToolResultEventData); ok {
			data.ExitCode, data.DurationMS = codexNestedResultMetadata(item.Output)
		}
		events = append(events, result)
		return events
	case "command_execution":
		return parseCodexCommandExecution(runID, item)
	}
	return nil
}

func newCodexMessageEvent(runID uuid.UUID, state *codexState, content, messageID, eventType string, terminal bool, reason string) *domain.RunEvent {
	turnID := ""
	conversationID := ""
	if state != nil {
		conversationID = state.threadID
		if state.turn > 0 {
			turnID = fmt.Sprintf("%s:turn-%d", conversationID, state.turn)
		}
	}
	return domain.NewProviderMessageEvent(runID, "assistant", content, domain.MessageEventData{
		MessageID:         messageID,
		ConversationID:    conversationID,
		TurnID:            turnID,
		ProviderOrigin:    "codex",
		CompletionReason:  reason,
		Terminal:          terminal,
		ProviderEventType: eventType,
		RawEvidenceRef:    "codex:" + eventType,
	})
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
			events = append(events, domain.NewToolCallEvent(runID, toolName, item.ID, input))
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
		result := domain.NewToolResultEvent(
			runID, toolName, item.ID,
			runner.StripANSI(output), errMsg,
		)
		if data, ok := result.Data.(*domain.ToolResultEventData); ok {
			data.ExitCode = item.ExitCode
		}
		events = append(events, result)
		return events
	}

	if item.Command != "" {
		input := map[string]interface{}{
			"command":     item.Command,
			"status":      item.Status,
			"runner_tool": "command_execution",
		}
		return []*domain.RunEvent{domain.NewToolCallEvent(runID, toolName, item.ID, input)}
	}
	return nil
}

// codexNestedResultMetadata handles the interactive tool-result envelope. It
// intentionally extracts only typed process facts; output text remains in the
// event log and is never copied into the analytical read model.
func codexNestedResultMetadata(output string) (*int, *int64) {
	var envelope struct {
		Metadata struct {
			ExitCode        *int     `json:"exit_code"`
			DurationSeconds *float64 `json:"duration_seconds"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil || (envelope.Metadata.ExitCode == nil && envelope.Metadata.DurationSeconds == nil) {
		return nil, nil
	}
	var duration *int64
	if envelope.Metadata.DurationSeconds != nil {
		value := int64(math.Round(*envelope.Metadata.DurationSeconds * 1000))
		duration = &value
	}
	return envelope.Metadata.ExitCode, duration
}

// Compile-time interface checks.
var (
	_ Codec = (*Codex)(nil)
	_ State = (*codexState)(nil)
)
