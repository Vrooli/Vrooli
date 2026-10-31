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
	"strconv"
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
	// subagentThread says session_meta named this rollout a sub-agent thread:
	// its user-role text is the parent agent's, not the operator's.
	subagentThread bool
	// forkPrefix says the rollout is still in the parent history a fork
	// copied (see parseRolloutLine).
	forkPrefix bool
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

// CodexItem represents an item in the Codex stream, and the rollout's
// event_msg item_completed CommandExecution item, which has the same fields.
type CodexItem struct {
	ID               string            `json:"id"`
	Type             string            `json:"type"`
	Text             string            `json:"text,omitempty"`
	Name             string            `json:"name,omitempty"`
	Input            json.RawMessage   `json:"input,omitempty"`
	Output           string            `json:"output,omitempty"`
	ExitCode         *int              `json:"exit_code,omitempty"`
	Command          codexCommand      `json:"command,omitempty"`
	AggregatedOutput string            `json:"aggregated_output,omitempty"`
	Changes          []CodexFileChange `json:"changes,omitempty"`
	Status           string            `json:"status,omitempty"`
	Cwd              string            `json:"cwd,omitempty"`
	Source           string            `json:"source,omitempty"`
	Duration         *codexDuration    `json:"duration,omitempty"`
}

// codexCommand is a command execution's command: shell text on the exec
// stream, argv (["/bin/bash", "-lc", "<script>"]) in rollout items.
type codexCommand []string

func (c *codexCommand) UnmarshalJSON(data []byte) error {
	var text string
	if json.Unmarshal(data, &text) == nil {
		*c = nil
		if strings.TrimSpace(text) != "" {
			*c = codexCommand{text}
		}
		return nil
	}
	var argv []string
	if json.Unmarshal(data, &argv) != nil {
		*c = nil // an unknown shape is no command, not a failed line
		return nil
	}
	*c = argv
	return nil
}

// codexDuration is the rollout's {secs, nanos} duration.
type codexDuration struct {
	Secs  int64 `json:"secs"`
	Nanos int64 `json:"nanos"`
}

func (d *codexDuration) milliseconds() *int64 {
	if d == nil || d.Secs < 0 || d.Nanos < 0 {
		return nil
	}
	ms := d.Secs*1000 + d.Nanos/1_000_000
	return &ms
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

// codexNestedCommandTool names the result of a command that an exec call ran
// (a rollout CommandExecution item). It has no tool call of its own: the
// model called exec, and runsignal joins the result to that call.
const codexNestedCommandTool = "command_execution"

// codexNestedOutputLimit bounds the output kept on a nested command result.
// The agent saw the command's output through the exec call's own result,
// which is stored in full; this excerpt keeps the first and last lines, where
// errors are, without storing every output twice.
const codexNestedOutputLimit = 2048

// codexCommandRecord is a nested command result's RawOutputJSON: the native
// item's scalar fields, with its output bodies replaced by their size.
type codexCommandRecord struct {
	Type        string   `json:"type"`
	Command     []string `json:"command"`
	Cwd         string   `json:"cwd,omitempty"`
	Status      string   `json:"status"`
	ExitCode    *int     `json:"exit_code"`
	Source      string   `json:"source,omitempty"`
	OutputBytes int      `json:"output_bytes"`
}

// parseCodexCommandExecution maps one command execution to tool events. On the
// exec stream (type command_execution) the command is the model's own tool
// call: item.started emits the call and the terminal item its result, with the
// call repeated for a failure so its command survives a missed start. In a
// rollout (type CommandExecution) the command ran inside an exec call, so it
// is a result only, carrying the command record, exit code and duration.
func parseCodexCommandExecution(runID uuid.UUID, item *CodexItem) []*domain.RunEvent {
	const toolName = "bash"
	nested := item.Type == "CommandExecution"
	command := ""
	if len(item.Command) > 0 {
		command = item.Command[len(item.Command)-1]
		if !nested {
			command = strings.Join(item.Command, " ")
		}
	}
	callInput := func() map[string]interface{} {
		return map[string]interface{}{"command": command, "status": item.Status, "runner_tool": "command_execution"}
	}
	switch item.Status {
	case "completed", "failed", "error", "cancelled", "timed_out":
	default:
		if command != "" && !nested {
			return []*domain.RunEvent{domain.NewToolCallEvent(runID, toolName, item.ID, callInput())}
		}
		return nil
	}
	events := make([]*domain.RunEvent, 0, 2)
	if command != "" && item.Status != "completed" && !nested {
		events = append(events, domain.NewToolCallEvent(runID, toolName, item.ID, callInput()))
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
	printed := len(output)
	output = runner.StripANSI(output)
	name := toolName
	if nested {
		name = codexNestedCommandTool
		output = boundedExcerpt(output, codexNestedOutputLimit)
	}
	result := domain.NewToolResultEvent(runID, name, item.ID, output, errMsg)
	data := result.Data.(*domain.ToolResultEventData)
	data.ExitCode = item.ExitCode
	if nested {
		if !domain.SupportedToolExitCode(data.ExitCode) {
			data.ExitCode = nil
		}
		data.DurationMS = item.Duration.milliseconds()
		known := true
		data.OutcomeKnown = &known
		record, _ := json.Marshal(codexCommandRecord{Type: item.Type, Command: item.Command, Cwd: item.Cwd, Status: item.Status, ExitCode: data.ExitCode, Source: item.Source, OutputBytes: printed})
		data.RawOutputJSON = string(record)
	}
	return append(events, result)
}

// boundedExcerpt keeps text up to limit bytes, else its head and tail.
func boundedExcerpt(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	half := limit / 2
	return strings.ToValidUTF8(text[:half], "") + "\n[…]\n" + strings.ToValidUTF8(text[len(text)-half:], "")
}

// codexTextExitCodePattern matches the exit line Codex's exec tools print at the
// start of a line ("Process exited with code 1", "Exit code: 1").
var codexTextExitCodePattern = regexp.MustCompile(`(?m)^(?:Process exited with code|Exit code:?)\s*(-?\d{1,4})\s*$`)

// codexTextExitCode reads the exit code from a plain-text tool result, nil when
// the output carries none.
func codexTextExitCode(output string) *int {
	match := codexTextExitCodePattern.FindStringSubmatch(output)
	if match == nil {
		return nil
	}
	code, err := strconv.Atoi(match[1])
	if err != nil {
		return nil
	}
	return &code
}

// codexNestedResultMetadata handles the interactive tool-result envelope. It
// intentionally extracts only typed process facts; output text remains in the
// event log and is never copied into the analytical read model.
func codexNestedResultMetadata(output string) (*int, *int64, bool) {
	var envelope map[string]json.RawMessage
	if json.Unmarshal([]byte(output), &envelope) != nil {
		return codexTextExitCode(output), nil, false
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
