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
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
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
		sessionLoss:    codexSessionLoss,
		labels: Labels{
			StartMessage:         "Codex execution started",
			ContinueStartMessage: "Codex continuation started",
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
	return codingAgentCapabilities(runner.Capabilities{
		SpawnCapabilities: []runner.SpawnCapability{
			{ExecutionMode: "codec_pipe", SandboxModes: []string{"protected", "tracking", "off"}},
			{ExecutionMode: "interactive", SandboxModes: []string{"tracking", "off"}, NativeObjective: true},
		},
		SupportsToolEvents:       true,
		SupportsCostTracking:     true,
		SupportsImageAttachments: true, // `codex exec -i/--image <FILE>`
		// `codex app-server` thread/compact/start (codex_compact.go).
		SupportsSessionCompaction: true,
		SupportsToolRestriction:   false, // Codex has no per-launch allowlist for its native tools.
		ToolRestrictionMappings:   canonicalToolMappings(codexToolTranslations),
		SupportsEffort:            true,
		// Codex config documents minimal, low, medium, high, and xhigh. The
		// portable scale has no minimal level, so max must not be claimed here.
		EffortMappings:       map[string]string{"low": "model_reasoning_effort=low", "medium": "model_reasoning_effort=medium", "high": "model_reasoning_effort=high", "xhigh": "model_reasoning_effort=xhigh"},
		SupportedModels:      c.ollama.list(),
		DynamicModelPrefixes: []string{ollamaModelPrefix},
		SupportedFeatures:    []string{},
		// Raw -c overrides can replace typed model, effort and sandbox
		// controls. Only the codec may generate those configuration flags.
		AllowedExtraFlags: []string{"--verbose"},
	})
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

	// Runner-native model, effort, and tool controls are centralized so the
	// interactive path cannot drift from codec-pipe execution. This includes
	// the network-derived approval/sandbox policy translated by ControlArgs.
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
		// A resumed session's first counter covers earlier turns.
		s.SetTranscriptSession(req.SessionID, false)
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
	rolloutTurnID    string
	lastMessageID    string
	lastMessage      string
	lastMessageEvent *domain.RunEvent
	// lastCumulative is the session's latest raw cumulative counter this
	// parser observed; sessionFromStart says the parser observed the session
	// from its first report (session_meta or a fresh exec), so a report with
	// no provider amount and no prior counter is entirely new usage.
	lastCumulative   *domain.UsageCounters
	sessionFromStart bool
	// turnUsageObserved records that the current turn reported usage, so its
	// closing boundary carries a receipt; turnAmount sums the turn's explicit
	// amounts for that receipt.
	turnUsageObserved bool
	turnAmount        domain.UsageCounters
	retainUser        bool
	// Goal evidence survives Consume segments with this parser, but cannot be
	// shared across runs or inferred from a request without its accepted result.
	goal codexGoalContext
}

func (s *codexState) SessionID() string { return s.threadID }

// NewState satisfies [Codec]. A fresh `codex exec` owns its session from the
// first report; a continuation names the resumed session through
// SetTranscriptSession, which clears that.
func (c *Codex) NewState() State { return &codexState{sessionFromStart: true} }

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
	InputTokens           int `json:"input_tokens"`
	CachedInputTokens     int `json:"cached_input_tokens,omitempty"`
	CacheWriteInputTokens int `json:"cache_write_input_tokens,omitempty"`
	OutputTokens          int `json:"output_tokens"`
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
	// Some native Codex builds put the same event_msg rollout records on the
	// codec pipe as on disk. Reuse the rollout parser against the live state so
	// quota metadata is observed without reading CODEX_HOME or credentials.
	if result, rollout := (&codexTranscriptParser{codec: c, state: s}).parseRolloutLine(runID, line); rollout {
		return result.Events, nil
	}

	streamEvent, ok := decodeCodexStreamEvent(runID, line)
	if !ok {
		return nil, nil
	}

	// Capture thread_id for session continuation. A continued provider may
	// switch to a new native session; never difference its counters against the
	// previous session's baseline.
	s.observeSession(streamEvent.ThreadID)

	return c.parseCodexEvents(s, runID, streamEvent), nil
}

// codexSessionLoss distinguishes Codex's two "thread not found" cases:
//
//   - On resume, the session id no longer maps to a live thread: a plain
//     "thread … not found" without rollout-writer context means the session is
//     genuinely gone → ReasonSessionExpired.
//   - During a live run, codex's rollout-recorder goroutine drops the thread
//     mid-stream. Codex emits this in two surface forms across versions — the
//     function-name form `record_rollout_items` and the human-readable form
//     `failed to record rollout items` (both are the same condition). The
//     session id is still valid but the rollout state is unrecoverable →
//     ReasonSessionStateLost.
func codexSessionLoss(stderr string) fallback.Reason {
	if !strings.Contains(stderr, "thread") || !strings.Contains(stderr, "not found") {
		return ""
	}
	if strings.Contains(stderr, "record_rollout_items") || strings.Contains(stderr, "record rollout items") {
		return fallback.ReasonSessionStateLost
	}
	return fallback.ReasonSessionExpired
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
		usage := data.Counters()
		before := int64(runner.TotalTokens(*metrics))
		if data.Cumulative != nil {
			// The invocation's usage is its latest counter minus the counter
			// before its first report, as in the run's usage account; a closing
			// receipt that repeats the turn's reports adds nothing.
			before := counterDifference(*data.Cumulative, usage)
			if metrics.UsageBaseline == nil || before.Total() < metrics.UsageBaseline.Total() {
				metrics.UsageBaseline = &before
			}
			if total := counterDifference(*data.Cumulative, *metrics.UsageBaseline); total.Total() > int64(runner.TotalTokens(*metrics)) {
				metrics.TokensInput, metrics.TokensOutput = total.InputTokens, total.OutputTokens
				metrics.CacheReadTokens, metrics.CacheCreationTokens = total.CacheReadTokens, total.CacheCreationTokens
			}
		} else {
			metrics.TokensInput += usage.InputTokens
			metrics.TokensOutput += usage.OutputTokens
			metrics.CacheReadTokens += usage.CacheReadTokens
			metrics.CacheCreationTokens += usage.CacheCreationTokens
		}
		if data.Turns > 0 {
			// Commentary messages within one provider turn are not turns.
			metrics.TurnsUsed = data.Turns
		}
		// A receipt's attached charge is counted when the receipt adds usage;
		// a closing receipt that repeats the turn's reports adds nothing.
		if data.Charge != nil && data.Charge.AmountMicroUSD != nil && int64(runner.TotalTokens(*metrics)) > before {
			metrics.CostEstimateUSD += float64(*data.Charge.AmountMicroUSD) / 1_000_000
		}
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

// SetTranscriptSession satisfies [runner.TranscriptSessionSetter].
func (s *codexState) SetTranscriptSession(sessionID string, fromStart bool) {
	s.observeSession(sessionID)
	s.sessionFromStart = fromStart
}

func (s *codexState) observeSession(sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if s == nil || sessionID == "" {
		return
	}
	if s.threadID != "" && sessionID != s.threadID {
		// A provider-confirmed session switch starts an independent counter
		// series; never difference it against the previous session's totals.
		s.lastCumulative = nil
	}
	s.threadID = sessionID
}

func (p *codexTranscriptParser) SetTranscriptSession(sessionID string, fromStart bool) {
	p.state.SetTranscriptSession(sessionID, fromStart)
}

func (p *codexTranscriptParser) SetTranscriptModel(model string) {
	p.state.runModel = strings.TrimSpace(model)
	if p.state.runModel == "" {
		p.state.runModel = "unknown"
	}
}

func (p *codexTranscriptParser) SetTranscriptRetention(retain bool) { p.state.retainUser = retain }

func (p *codexTranscriptParser) SetTranscriptBilling(billing domain.BillingSnapshot) {
	p.state.billing = billing
}

// SetGoalContext pins native goal reads to the current run's session and goal.
// Interactive tails begin at a persisted cursor, so their parser does not
// necessarily see the session/objective prefix from the original invocation.
func (p *codexTranscriptParser) SetGoalContext(runID uuid.UUID, sessionID, objective string) {
	p.state.goal = codexGoalContext{runID: runID, threadID: strings.TrimSpace(sessionID), expectedThreadID: strings.TrimSpace(sessionID), expectedObjective: strings.TrimSpace(objective)}
}

type codexTranscriptParser struct {
	codec *Codex
	state *codexState
}

func (p *codexTranscriptParser) ParseTranscriptLine(runID uuid.UUID, line string) runner.TranscriptParseResult {
	// Native updates or correlated accepted tool results are goal evidence.
	// Requests, quoted examples and unmatched outputs never terminate a goal.
	goal, hasGoal := p.state.goal.parse(runID, line)
	attachGoal := func(res runner.TranscriptParseResult) runner.TranscriptParseResult {
		if hasGoal {
			marker := goal
			res.Goal = &marker
		}
		return res
	}

	// On-disk interactive rollout dialect: every line wraps as
	// {"type":"event_msg"|"response_item"|"session_meta"|"turn_context",
	// "payload":{…}}, foreign to the flat `exec --json` decoder. Detect and
	// handle it before the stdout path (design §3). A non-rollout line falls
	// through to the exec-json decoder below, so pipe-mode replay is unchanged.
	if res, ok := p.parseRolloutLine(runID, line); ok {
		return attachGoal(res)
	}

	streamEvent, ok := decodeCodexStreamEvent(runID, line)
	if !ok {
		return attachGoal(runner.TranscriptParseResult{})
	}

	p.state.observeSession(streamEvent.ThreadID)
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
	return attachGoal(result)
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
	// response_item message
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
	// agent_message / user_message
	Message string `json:"message"`
	// task_started/task_complete
	TurnID string `json:"turn_id"`
	// function_call / custom_tool_call
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // function_call: JSON-encoded args
	Input     string `json:"input"`     // custom_tool_call: raw tool input
	CallID    string `json:"call_id"`
	// function_call_output / custom_tool_call_output
	Output json.RawMessage `json:"output"`
	// turn_context
	Model string `json:"model"`
	// turn_aborted
	Reason string               `json:"reason"`
	Info   *codexTokenCountInfo `json:"info"`
	// Native quota metadata is a sibling of info in token_count frames. It is
	// present even when info is null (for example at session start).
	RateLimits codexRateLimits `json:"rate_limits"`
}

// codexRateLimits keeps the window-shaped entries of a token_count frame's
// rate_limits and skips the rest. Newer Codex CLIs add scalar siblings such as
// limit_id, plan_type and credits; a strict map type would fail the whole frame
// and silently drop its usage.
type codexRateLimits map[string]CodexRateLimitWindow

func (m *codexRateLimits) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	out := codexRateLimits{}
	for key, value := range raw {
		trimmed := bytes.TrimSpace(value)
		if len(trimmed) == 0 || trimmed[0] != '{' {
			continue
		}
		var window CodexRateLimitWindow
		if json.Unmarshal(trimmed, &window) == nil && (window.UsedPercent != nil || window.WindowMinutes != 0) {
			out[key] = window
		}
	}
	*m = out
	return nil
}

type codexTokenCountInfo struct {
	LastTokenUsage  *CodexUsage `json:"last_token_usage"`
	TotalTokenUsage *CodexUsage `json:"total_token_usage"`
}

// CodexRateLimitWindow is the bounded, metadata-only native quota shape. The
// CLI reports a percentage/window/reset rather than an absolute token quota;
// keeping these as optional fields avoids inventing a ceiling.
type CodexRateLimitWindow struct {
	UsedPercent   *float64 `json:"used_percent,omitempty"`
	WindowMinutes int64    `json:"window_minutes,omitempty"`
	ResetAt       int64    `json:"reset_at,omitempty"`
	ResetsAt      int64    `json:"resets_at,omitempty"`
}

const codexProvider = "openai"

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
	// Recognize the dialect before decoding variable payload content. A known
	// wrapper with unsupported content must never fall into flat stdout parsing.
	var envelope struct {
		Type string `json:"type"`
	}
	if json.Unmarshal([]byte(trimmed), &envelope) != nil || !rolloutWrapperTypes[envelope.Type] {
		return runner.TranscriptParseResult{}, false
	}
	var rl codexRolloutLine
	if err := json.Unmarshal([]byte(trimmed), &rl); err != nil {
		warning := domain.NewLogEvent(runID, "warning", "Unsupported Codex rollout payload: "+err.Error())
		return runner.TranscriptParseResult{Events: []*domain.RunEvent{warning}, Timestamp: transcriptLineTimestamp(line)}, true
	}

	result := runner.TranscriptParseResult{Timestamp: transcriptLineTimestamp(line)}
	pl := rl.Payload
	if pl.TurnID != "" {
		p.state.rolloutTurnID = pl.TurnID
	}

	switch rl.Type {
	case "session_meta":
		if pl.ID != "" {
			p.state.observeSession(pl.ID)
			p.state.sessionFromStart = true
			result.SessionID = pl.ID
		}
		return result, true

	case "turn_context":
		// The turn's own model is the truth; a configured model may be absent
		// (imported sessions) or overridden in the session.
		if model := strings.TrimSpace(pl.Model); model != "" {
			p.state.runModel = model
		}
		p.state.turn++
		result.ProviderTurnStarted = true
		return result, true

	case "response_item":
		if pl.Type == "message" {
			role := strings.ToLower(strings.TrimSpace(pl.Role))
			result.UserTurnStarted = role == "user" || role == "operator"
		}
		result.Events = parseCodexRolloutItem(runID, pl, p.state)
		return result, true

	case "event_msg":
		switch pl.Type {
		case "task_started":
			p.state.turnUsageObserved, p.state.turnAmount = false, domain.UsageCounters{}
			if p.state.turn == 0 {
				p.state.turn = 1
			}
		case "token_count":
			if pl.Info != nil && (pl.Info.TotalTokenUsage != nil || pl.Info.LastTokenUsage != nil) {
				if p.state.turn == 0 {
					p.state.turn = 1
				}
				// Quota refreshes repeat the previous frame's counters; only a
				// changed counter is a new usage report.
				if usage, fresh := p.state.usageReport(runID, p.codec.pricingService, pl.Info.TotalTokenUsage, pl.Info.LastTokenUsage); fresh {
					// The report happened at the line's time, not at parse time:
					// an indexer back-fill reads it days later.
					for _, event := range usage {
						if !result.Timestamp.IsZero() {
							event.Timestamp = result.Timestamp
						}
					}
					result.Events = usage
					p.state.turnUsageObserved = true
					p.state.turnAmount = counterSum(p.state.turnAmount, usage[0].Data.(*domain.UsageEventData).Counters())
				}
			}
			result.Events = append(result.Events, codexQuotaEvents(runID, result.Timestamp, pl.RateLimits)...)
		case "agent_message":
			if text := runner.StripANSI(strings.TrimSpace(pl.Message)); text != "" {
				p.state.lastMessage = text
				p.state.lastMessageID = pl.ID
				messageEvent := newCodexMessageEvent(runID, p.state, text, pl.ID, "event_msg.agent_message", false, "")
				p.state.lastMessageEvent = messageEvent
				result.Events = []*domain.RunEvent{messageEvent}
			}
		case "user_message":
			result.UserTurnStarted = true
			if p.state.retainUser && strings.TrimSpace(pl.Message) != "" {
				result.Events = []*domain.RunEvent{domain.NewProviderMessageEvent(runID, "user", runner.StripANSI(pl.Message), domain.MessageEventData{ProviderOrigin: "codex", ProviderEventType: "event_msg.user_message", RawEvidenceRef: "codex:event_msg.user_message"})}
			}
		case "task_complete", "turn_completed":
			result.Events = append(result.Events, p.rolloutTerminalReceipt(runID, result.Timestamp)...)
			if p.state.lastMessageEvent != nil {
				markProviderMessageTerminal(p.state.lastMessageEvent, pl.Type, "event_msg."+pl.Type, "codex:event_msg."+pl.Type)
				result.Events = append(result.Events, newProviderTerminalEvidence(runID, p.state.lastMessageEvent, pl.Type, "event_msg."+pl.Type, "codex:event_msg."+pl.Type))
			}
			result.Terminal = &runner.TranscriptTerminal{Success: true, ExitCode: 0}
		case "turn_aborted":
			result.Events = append(result.Events, p.rolloutTerminalReceipt(runID, result.Timestamp)...)
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

// codexQuotaEvents converts only the native rate-limit metadata into typed
// observations. A Codex token_count frame has no absolute quota ceiling, so
// this intentionally emits percentage/window/reset metadata only.
func codexQuotaEvents(runID uuid.UUID, observedAt time.Time, rateLimits map[string]CodexRateLimitWindow) []*domain.RunEvent {
	if len(rateLimits) == 0 {
		return nil
	}
	capacity := len(rateLimits)
	if capacity > 64 {
		capacity = 64
	}
	result := make([]*domain.RunEvent, 0, capacity)
	for pool, limit := range rateLimits {
		if len(result) >= 64 {
			break
		}
		pool = strings.TrimSpace(pool)
		if pool == "" || len(pool) > 64 || limit.UsedPercent == nil || *limit.UsedPercent < 0 || *limit.UsedPercent > 100 || limit.WindowMinutes <= 0 {
			continue
		}
		resetEpoch := limit.ResetAt
		if resetEpoch == 0 {
			resetEpoch = limit.ResetsAt
		}
		var reset *time.Time
		if resetEpoch > 0 {
			value := time.Unix(resetEpoch, 0).UTC()
			reset = &value
		}
		event := domain.NewRateLimitEvent(runID, fmt.Sprintf("%dm", limit.WindowMinutes), "Codex provider quota observation", reset, 0)
		event.Timestamp = observedAt
		data, ok := event.Data.(*domain.RateLimitEventData)
		if !ok {
			continue
		}
		percent := *limit.UsedPercent
		data.Provider = codexProvider
		data.Pool = pool
		data.UsedPercent = &percent
		data.WindowMinutes = limit.WindowMinutes
		data.Provenance = "codex:event_msg.token_count.rate_limits"
		result = append(result, event)
	}
	return result
}

// Both explicit successful and aborted turn boundaries close provider usage.
// The receipt carries the turn's summed amount and the session's latest
// cumulative counter, so it stands alone when it is the only retained report
// (transcript recovery) and adds nothing when the turn's reports are retained:
// the accounting credits each counter value once. Missing token evidence
// remains unknown; a terminal marker alone cannot create zero.
func (p *codexTranscriptParser) rolloutTerminalReceipt(runID uuid.UUID, at time.Time) []*domain.RunEvent {
	if !p.state.turnUsageObserved || p.state.lastCumulative == nil {
		return nil
	}
	cumulative := *p.state.lastCumulative
	events := markUsageTurn(buildCostEvents(runID, domain.RunnerTypeCodex, p.codec.pricingService, p.state.runModel, usageTokens(p.state.turnAmount), p.state.billing), p.state.turn)
	usage := events[0].Data.(*domain.UsageEventData)
	usage.ReconciliationAuthority, usage.Turns = true, 1
	usage.SessionID, usage.Cumulative = p.state.threadID, &cumulative
	events[0].Timestamp = at
	return attachCharge(events)
}

// attachCharge moves a receipt's charge onto its usage event, so the charge
// has one representation: an authoritative receipt carries it and no separate
// charge event repeats it (F-W1-2). events[0] is the usage event.
func attachCharge(events []*domain.RunEvent) []*domain.RunEvent {
	usage, ok := events[0].Data.(*domain.UsageEventData)
	if !ok {
		return events
	}
	out := events[:1]
	for _, event := range events[1:] {
		if charge, isCharge := event.Data.(*domain.ChargeEventData); isCharge {
			usage.Charge = charge
			continue
		}
		out = append(out, event)
	}
	return out
}

// closingReport repeats the session's latest counter with no amount, so a
// turn boundary can close the invocation without adding usage.
func (s *codexState) closingReport(runID uuid.UUID, pricing PricingService) []*domain.RunEvent {
	events := markUsageTurn(buildCostEvents(runID, domain.RunnerTypeCodex, pricing, s.runModel, usageTokens{}, s.billing), s.turn)
	if usage, ok := events[0].Data.(*domain.UsageEventData); ok && s.lastCumulative != nil {
		cumulative := *s.lastCumulative
		usage.SessionID, usage.Cumulative = s.threadID, &cumulative
	}
	return events
}

// usageReport turns one provider counter report into usage events carrying
// the raw session-cumulative counter and the provider's explicit amount for
// this report. The amount is last_token_usage when Codex supplies it, the
// difference from this parser's previous counter, the whole counter when the
// parser saw the session start, and otherwise zero (unknown) — never the
// counter of a session observed mid-way. fresh is false when the counter
// repeats the previous report. A report without a cumulative counter (older
// rollouts) carries only its amount.
func (s *codexState) usageReport(runID uuid.UUID, pricing PricingService, total, last *CodexUsage) ([]*domain.RunEvent, bool) {
	var amount domain.UsageCounters
	var cumulative *domain.UsageCounters
	switch {
	case total != nil:
		value := codexCounters(total)
		if s.lastCumulative != nil && *s.lastCumulative == value {
			return nil, false
		}
		switch {
		case last != nil:
			amount = codexCounters(last)
		case s.lastCumulative != nil:
			amount = counterDifference(value, *s.lastCumulative)
		case s.sessionFromStart:
			amount = value
		}
		s.lastCumulative, cumulative = &value, &value
	case last != nil:
		amount = codexCounters(last)
	default:
		return nil, false
	}
	events := markUsageTurn(buildCostEvents(runID, domain.RunnerTypeCodex, pricing, s.runModel, usageTokens(amount), s.billing), s.turn)
	if usage, ok := events[0].Data.(*domain.UsageEventData); ok {
		usage.SessionID, usage.Cumulative = s.threadID, cumulative
	}
	return events, true
}

// codexCounters splits Codex's usage into disjoint components. Codex input
// includes cached reads (and cache writes, when reported), so those subsets
// are removed from input before persistence and consumed exactly once.
func codexCounters(u *CodexUsage) domain.UsageCounters {
	input := u.InputTokens
	if input >= u.CachedInputTokens+u.CacheWriteInputTokens {
		input -= u.CachedInputTokens + u.CacheWriteInputTokens
	}
	return domain.UsageCounters{InputTokens: input, OutputTokens: u.OutputTokens, CacheReadTokens: u.CachedInputTokens, CacheCreationTokens: u.CacheWriteInputTokens}
}

func counterSum(a, b domain.UsageCounters) domain.UsageCounters {
	return domain.UsageCounters{InputTokens: a.InputTokens + b.InputTokens, OutputTokens: a.OutputTokens + b.OutputTokens, CacheReadTokens: a.CacheReadTokens + b.CacheReadTokens, CacheCreationTokens: a.CacheCreationTokens + b.CacheCreationTokens}
}

func counterDifference(current, previous domain.UsageCounters) domain.UsageCounters {
	return domain.UsageCounters{
		InputTokens:         max(0, current.InputTokens-previous.InputTokens),
		OutputTokens:        max(0, current.OutputTokens-previous.OutputTokens),
		CacheReadTokens:     max(0, current.CacheReadTokens-previous.CacheReadTokens),
		CacheCreationTokens: max(0, current.CacheCreationTokens-previous.CacheCreationTokens),
	}
}

// parseCodexRolloutItem maps a response_item payload to tool events. Both the
// generic function_call/output and MCP-style custom_tool_call/output shapes
// carry a call_id, a name (on the call side), and a string output.
func parseCodexRolloutItem(runID uuid.UUID, pl codexRolloutPayload, state *codexState) []*domain.RunEvent {
	if pl.Type == "message" {
		text := codexRolloutMessageText(pl.Content)
		if text == "" {
			return nil
		}
		text = runner.StripANSI(strings.TrimSpace(text))
		switch strings.ToLower(strings.TrimSpace(pl.Role)) {
		case "assistant":
			message := newCodexMessageEvent(runID, state, text, pl.ID, "response_item.message", false, "")
			state.lastMessage = text
			state.lastMessageID = pl.ID
			state.lastMessageEvent = message
			return []*domain.RunEvent{message}
		case "user", "operator":
			if state != nil && state.retainUser {
				return []*domain.RunEvent{domain.NewProviderMessageEvent(runID, "user", text, domain.MessageEventData{ProviderOrigin: "codex", ProviderEventType: "response_item.message"})}
			}
		}
		return nil
	}
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
		return codexRolloutToolResult(runID, pl)
	}
	// message / reasoning and other response items carry no distinct
	// domain event here (assistant text arrives via event_msg.agent_message).
	return nil
}

// inlineImageData matches the base64 body of an inline image data URI.
var inlineImageData = regexp.MustCompile(`(data:[A-Za-z0-9.+/-]+;base64,)[A-Za-z0-9+/=]+`)

// stripInlineImageData drops inline image bytes from stored tool output: a
// screenshot is megabytes of base64 with no recall value, and the indexer
// counts images from the source transcript, not from stored events (R-1).
func stripInlineImageData(raw string) string {
	if !strings.Contains(raw, ";base64,") {
		return raw
	}
	return inlineImageData.ReplaceAllString(raw, "${1}<omitted>")
}

// codexRolloutToolResult preserves native strings and ordered typed text blocks.
// A returned tool payload establishes evidence, not a successful process exit.
func codexRolloutToolResult(runID uuid.UUID, pl codexRolloutPayload) []*domain.RunEvent {
	data := &domain.ToolResultEventData{ToolCallID: pl.CallID, RawOutputJSON: stripInlineImageData(string(pl.Output))}
	known := false
	data.OutcomeKnown = &known
	unsupported := false
	conflictingMetadata := false
	conflictingDuration := false
	var text string
	if json.Unmarshal(pl.Output, &text) == nil && len(pl.Output) > 0 && string(bytes.TrimSpace(pl.Output)) != "null" {
		data.Output = runner.StripANSI(text)
		data.ExitCode, data.DurationMS, unsupported = codexNestedResultMetadata(text)
	} else {
		var blocks []json.RawMessage
		if json.Unmarshal(pl.Output, &blocks) != nil || string(bytes.TrimSpace(pl.Output)) == "null" {
			unsupported = true
		} else {
			for _, raw := range blocks {
				var block struct {
					Type string  `json:"type"`
					Text *string `json:"text"`
				}
				output := domain.ToolOutputBlock{RawJSON: stripInlineImageData(string(raw))}
				if json.Unmarshal(raw, &block) == nil {
					output.Type = block.Type
					if (block.Type == "input_text" || block.Type == "output_text" || block.Type == "text") && block.Text != nil {
						output.Text = runner.StripANSI(*block.Text)
						// Typed metadata is parsed only from a complete text block, never
						// recursively extracted or synthesized from concatenated fragments.
						exit, duration, invalid := codexNestedResultMetadata(*block.Text)
						unsupported = unsupported || invalid
						if exit != nil {
							if data.ExitCode != nil && *data.ExitCode != *exit {
								conflictingMetadata = true
							}
							data.ExitCode = exit
						}
						if duration != nil {
							if data.DurationMS != nil && *data.DurationMS != *duration {
								conflictingDuration = true
							}
							data.DurationMS = duration
						}
					} else {
						unsupported = true
					}
				} else {
					unsupported = true
				}
				data.OutputBlocks = append(data.OutputBlocks, output)
			}
		}
	}
	if !domain.SupportedToolExitCode(data.ExitCode) {
		conflictingMetadata = true
	}
	if conflictingMetadata {
		data.ExitCode = nil
		unsupported = true
	}
	if conflictingDuration {
		data.DurationMS = nil
		unsupported = true
	}
	if data.ExitCode != nil {
		known = true
		data.Success = *data.ExitCode == 0
		if !data.Success {
			data.Error = fmt.Sprintf("command exited with code %d", *data.ExitCode)
		}
	}
	event := &domain.RunEvent{ID: uuid.New(), RunID: runID, EventType: domain.EventTypeToolResult, Timestamp: time.Now(), Data: data}
	events := []*domain.RunEvent{event}
	if unsupported {
		events = append(events, domain.NewLogEvent(runID, "warning", "Unsupported Codex tool output content retained for call "+pl.CallID))
	}
	return events
}

// codexRolloutMessageText extracts the text-bearing content items used by
// Codex response_item messages. Developer/system messages are filtered by the
// caller; content itself is intentionally reduced to text, never attachments
// or tool metadata.
func codexRolloutMessageText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return ""
	}
	var collect func(any)
	parts := []string{}
	collect = func(item any) {
		switch value := item.(type) {
		case string:
			if strings.TrimSpace(value) != "" {
				parts = append(parts, value)
			}
		case []any:
			for _, child := range value {
				collect(child)
			}
		case map[string]any:
			if text, ok := value["text"].(string); ok && strings.TrimSpace(text) != "" {
				parts = append(parts, text)
				return
			}
		}
	}
	collect(value)
	return strings.Join(parts, "\n")
}

// =============================================================================
// Stream parser
// =============================================================================

// decodeCodexStreamEvent unwraps a single stdout/transcript line into a
// *CodexStreamEvent. Returns ok=false for blank or non-JSON lines. runID is the
// run being decoded, used only to attribute a recorded parse failure; a nil
// uuid is acceptable for call sites without a run (behavior is unchanged
// either way — the drop is observed, never altered).
func decodeCodexStreamEvent(runID uuid.UUID, line string) (*CodexStreamEvent, bool) {
	line = strings.TrimSpace(line)
	// Some Codex builds prefix events with `data:` (SSE-style).
	if strings.HasPrefix(line, "data:") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	}
	if line == "" || line[0] != '{' {
		// Blank or non-JSON (banner/startup) — an expected, non-drift skip.
		return nil, false
	}
	var ev CodexStreamEvent
	if err := json.Unmarshal([]byte(line), &ev); err != nil {
		// The line opened with '{' yet would not parse: real format drift.
		// Behavior unchanged (still ok=false); record so it is observable (W1).
		recordParseFailure(runID, siteCodexStreamJSON, err.Error(), line)
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
			// exec usage is the thread's cumulative counter. A repeated counter
			// still closes this invocation, with no amount of its own.
			usageEvents, fresh := state.usageReport(runID, c.pricingService, streamEvent.Usage, nil)
			if !fresh {
				usageEvents = state.closingReport(runID, c.pricingService)
			}
			if usage, ok := usageEvents[0].Data.(*domain.UsageEventData); ok {
				// exec --json reports this completed invocation's final
				// usage here. A process stop alone has no such authority.
				usage.ReconciliationAuthority = true
				usage.Turns = state.turn
			}
			events = append(events, attachCharge(usageEvents)...)
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
		var diagnostic *domain.RunEvent
		if data, ok := result.Data.(*domain.ToolResultEventData); ok {
			var invalid bool
			data.ExitCode, data.DurationMS, invalid = codexNestedResultMetadata(item.Output)
			if !domain.SupportedToolExitCode(data.ExitCode) {
				data.ExitCode = nil
				invalid = true
			}
			if data.ExitCode != nil {
				known := true
				data.OutcomeKnown = &known
				data.Success = *data.ExitCode == 0
				if !data.Success {
					data.Error = fmt.Sprintf("command exited with code %d", *data.ExitCode)
				}
			} else if invalid {
				known := false
				data.OutcomeKnown = &known
				data.Success = false
			}
			if invalid {
				diagnostic = domain.NewLogEvent(runID, "warning", "Unsupported Codex tool result metadata retained for call "+item.ID)
			}
		}
		events = append(events, result)
		if diagnostic != nil {
			events = append(events, diagnostic)
		}
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
			turnID = state.rolloutTurnID
			if turnID == "" {
				turnID = fmt.Sprintf("%s:turn-%d", conversationID, state.turn)
			}
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
func codexNestedResultMetadata(output string) (*int, *int64, bool) {
	var envelope map[string]json.RawMessage
	if json.Unmarshal([]byte(output), &envelope) != nil {
		return nil, nil, false
	}
	raw, present := envelope["metadata"]
	if !present {
		return nil, nil, false
	}
	var metadata map[string]json.RawMessage
	if json.Unmarshal(raw, &metadata) != nil || metadata == nil {
		return nil, nil, true
	}
	var exit *int
	var duration *int64
	invalid := false
	if raw, ok := metadata["exit_code"]; ok && string(bytes.TrimSpace(raw)) != "null" {
		var value int
		if json.Unmarshal(raw, &value) != nil {
			invalid = true
		} else {
			exit = &value
		}
	}
	if raw, ok := metadata["duration_seconds"]; ok && string(bytes.TrimSpace(raw)) != "null" {
		var seconds float64
		if json.Unmarshal(raw, &seconds) != nil {
			invalid = true
		} else {
			rounded := math.Round(seconds * 1000)
			if seconds < 0 || math.IsNaN(rounded) || math.IsInf(rounded, 0) || rounded >= float64(math.MaxInt64) {
				invalid = true
			} else {
				value := int64(rounded)
				duration = &value
			}
		}
	}
	return exit, duration, invalid
}

// Compile-time interface checks.
var (
	_ Codec = (*Codex)(nil)
	_ State = (*codexState)(nil)
)
