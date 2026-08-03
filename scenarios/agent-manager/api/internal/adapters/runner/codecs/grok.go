// Package codecs — grok.go is the [Codec] implementation for xAI's Grok Build
// CLI (the `grok` binary invoked headlessly with
// `-p <prompt> --output-format streaming-json`).
//
// What this codec owns:
//   - CLI args + env shape (BuildArgs, BuildContinueArgs, BuildEnv)
//   - The Grok streaming-json decoder (DecodeStreamLine, NewTranscriptParser)
//   - Per-run state: accumulated assistant text, captured session id
//   - Session-expiry detection from stderr (ClassifyTerminalError / Classify)
//
// Binary resolution, availability, probing, labels, type, tag-key,
// continuation tags, the no-op PostClassify and the drain-to-EOF
// OnEarlyTerminate are all provided by the embedded [baseCodec].
//
// Capability honesty: Grok's headless stdout (verified against a captured
// trace, codecs/testdata/grok_trace.jsonl) surfaces ONLY reasoning
// ("thought"), assistant text ("text"), a terminal "end" event carrying the
// session id, and "error" events. It does NOT surface tool call/result events
// or token-usage/cost — even when a tool actually runs. Capabilities() reflects
// exactly that: messages + streaming + continuation, but no tool events and no
// cost tracking. (See the grok-runner plan §6a / R3.)
package codecs

import (
	"encoding/json"
	"errors"
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

// GrokCLICommand is the binary name resolved on the host PATH (installed by
// the `grok` resource at ~/.local/bin/grok).
const GrokCLICommand = "grok"

// grokTagEnvKey is the per-run tag env var the reconciler reads from
// /proc/<pid>/environ. Distinct from the GROK_AGENT=1 self-detection sentinel
// grok injects into its own tool subprocesses (see D7 + the grok resource's
// agent-detection signals).
const grokTagEnvKey = "GROK_AGENT_TAG"

// =============================================================================
// Codec
// =============================================================================

// Grok is the [Codec] implementation for the grok CLI.
type Grok struct {
	baseCodec
}

func (c *Grok) ToolCapabilityMap() map[string]string {
	return map[string]string{"bash": "shell", "read": "file-read", "write": "file-edit", "search": "search", "wait": "wait", "task": "delegate", "web": "network"}
}

// Grok documents --allow/--deny as the Claude Code allowedTools and
// disallowedTools equivalents. Keep the shared vocabulary at the codec seam.
var grokToolTranslations = map[domain.CanonicalTool]string{
	domain.CanonicalToolRead: "Read", domain.CanonicalToolWrite: "Write", domain.CanonicalToolEdit: "Edit",
	domain.CanonicalToolGlob: "Glob", domain.CanonicalToolGrep: "Grep", domain.CanonicalToolShell: "Bash",
	domain.CanonicalToolWebSearch: "WebSearch", domain.CanonicalToolWebFetch: "WebFetch",
}

// grokBase is the identity shared by NewGrok and NewGrokForTest.
func grokBase() baseCodec {
	return baseCodec{
		runnerType:     domain.RunnerTypeGrok,
		binaryDesc:     "grok CLI",
		installHint:    "Run: vrooli resource install grok",
		tagEnvKey:      grokTagEnvKey,
		continuePrefix: "grok",
		goalStatus:     func(line string) (string, bool, bool) { return declaredGoalStatus(line, "goal_status", "goal") },
		labels: Labels{
			StartMessage:         "Grok execution started",
			EndMessage:           "Grok execution completed",
			ContinueStartMessage: "Grok continuation started",
			ContinueEndMessage:   "Grok continuation completed",
		},
	}
}

// NewGrok resolves the `grok` binary on PATH and returns a codec ready to be
// wrapped in [core.NewRunner]. Returns a codec with Available=false (rather
// than an error) when the binary is missing so the runner registry can
// register a stub instead.
func NewGrok() (*Grok, error) {
	c := &Grok{baseCodec: resolveBinary(grokBase(), GrokCLICommand)}
	c.newParser = c.NewTranscriptParser
	return c, nil
}

// NewGrokForTest returns a Grok codec with a fake binary path and
// Available=false. Used by codec tests that exercise BuildArgs / decode paths
// without launching a real process.
func NewGrokForTest() *Grok {
	c := &Grok{baseCodec: testBase(grokBase(), "/fake/grok", "test grok codec")}
	c.newParser = c.NewTranscriptParser
	return c
}

// NewGrokForTestWithBinary is a test-only constructor for process replay.
func NewGrokForTestWithBinary(path string) *Grok {
	c := NewGrokForTest()
	c.binaryPath, c.available = path, true
	return c
}

// Capabilities satisfies [Codec]. Every bool is gated on the captured trace
// (R3 — no vaporware): grok headless surfaces assistant text and a session id
// but no tool events and no token/cost data.
func (c *Grok) Capabilities() runner.Capabilities {
	return runner.Capabilities{
		SupportsMessages:         true,  // "text" events carry assistant output
		SupportsToolEvents:       false, // headless stdout never surfaces tool calls/results
		SupportsCostTracking:     false, // no usage/cost in the stream
		SupportsStreaming:        true,  // streaming-json delta events
		SupportsCancellation:     true,  // process-kill cancellation like peers
		SupportsContinuation:     true,  // `grok --resume <session-id>` (trace-proven)
		SupportsImageAttachments: false, // no headless image-attachment flag
		SupportsToolRestriction:  true,
		ToolRestrictionMappings:  canonicalToolMappings(grokToolTranslations),
		SupportsEffort:           true,
		EffortMappings:           map[string]string{"low": "low", "medium": "medium", "high": "high", "xhigh": "xhigh", "max": "max"},
		MaxTurns:                 0, // unlimited (configurable via --max-turns)
		SupportsRunnerDefault:    true,
		SupportedFeatures:        []string{},
		AllowedExtraFlags:        nil,
	}
}

// BuildEnv satisfies [Codec]. The tag is written to GROK_AGENT_TAG; codec
// extras are merged on top.
func (c *Grok) BuildEnv(tag string, extras map[string]string) []string {
	return standardBuildEnv(grokTagEnvKey, tag, extras)
}

// BuildPrompt satisfies [Codec]. Grok takes the prompt as the `-p` CLI value
// (added in BuildArgs), not via stdin, so the launcher closes stdin.
func (c *Grok) BuildPrompt(_ string, _ []runner.Attachment) string { return "" }

func (c *Grok) ControlArgs(cfg *domain.RunConfig) ([]string, error) { return grokControlArgs(cfg) }

// BuildArgs satisfies [Codec]. Headless single-turn invocation:
// `grok -p <prompt> --output-format streaming-json [-m model] [--max-turns N]
// [--always-approve] [--cwd dir]`.
//
// Permission posture: SkipPermissionPrompt maps to grok's --always-approve
// flag and the per-invocation --allow/--deny controls carry canonical tool
// intent without mutating Grok's shared configuration. NetworkAccess is not
// mapped (D5): grok's only network knob is the operator-defined --sandbox
// profile, not a per-invocation toggle.
func (c *Grok) BuildArgs(_ State, req runner.ExecuteRequest) []string {
	cfg := req.GetConfig()

	args := []string{
		"-p", req.EffectivePrompt(),
		"--output-format", "streaming-json",
	}
	if cfg.MaxTurns > 0 {
		args = append(args, "--max-turns", strconv.Itoa(cfg.MaxTurns))
	} else {
		args = append(args, "--max-turns", "30")
	}

	// Centralize portable control translation with interactive launches.
	controlArgs, _ := c.ControlArgs(cfg)
	args = append(args, controlArgs...)
	if cfg.SkipPermissionPrompt {
		args = append(args, "--always-approve")
	}
	// Pin the session to the run's working directory. grok can attach to a
	// shared leader process (config use_leader / --leader) whose cwd differs
	// from this process, so set --cwd explicitly even though the launcher
	// already sets the process cwd (mirrors opencode's --dir).
	if dir := strings.TrimSpace(req.WorkingDir); dir != "" {
		args = append(args, "--cwd", dir)
	}
	if extras, ok := cfg.ExtraFlags[domain.RunnerTypeGrok]; ok {
		args = append(args, extras...)
	}
	return args
}

// BuildContinueArgs satisfies [Codec]. `grok -p <prompt> --resume <session-id>`
// resumes an existing session (trace-proven: the same session id is reused and
// prior context is retained).
func (c *Grok) BuildContinueArgs(_ State, req runner.ContinueRequest) []string {
	cfg := req.GetConfig()

	args := []string{
		"-p", req.Prompt,
		"--resume", req.SessionID,
		"--output-format", "streaming-json",
	}
	if cfg.MaxTurns > 0 {
		args = append(args, "--max-turns", strconv.Itoa(cfg.MaxTurns))
	} else {
		args = append(args, "--max-turns", "30")
	}
	controlArgs, _ := c.ControlArgs(cfg)
	args = append(args, controlArgs...)
	if cfg.SkipPermissionPrompt {
		args = append(args, "--always-approve")
	}
	if dir := strings.TrimSpace(req.WorkingDir); dir != "" {
		args = append(args, "--cwd", dir)
	}
	if extras, ok := cfg.ExtraFlags[domain.RunnerTypeGrok]; ok {
		args = append(args, extras...)
	}
	return args
}

// =============================================================================
// State
// =============================================================================

// grokState carries per-run mutable state through the stream decode loop.
type grokState struct {
	textBuffer    strings.Builder // accumulates "text" delta payloads
	lastAssistant string
	sessionID     string
	gotEnd        bool
	retainUser    bool
}

func (s *grokState) SessionID() string { return s.sessionID }

// NewState satisfies [Codec].
func (c *Grok) NewState() State { return &grokState{} }

// =============================================================================
// Stream-event types
// =============================================================================

// GrokStreamEvent is a single line from grok's streaming-json output. The
// stream is a flat sequence of delta events; "thought"/"text" carry a Data
// string fragment, "end" carries the terminal metadata, "error" carries a
// Message.
type GrokStreamEvent struct {
	Type       string `json:"type"`
	Data       string `json:"data,omitempty"`
	StopReason string `json:"stopReason,omitempty"`
	SessionID  string `json:"sessionId,omitempty"`
	RequestID  string `json:"requestId,omitempty"`
	Message    string `json:"message,omitempty"`
}

// =============================================================================
// Stream decoding
// =============================================================================

// DecodeStreamLine satisfies [Codec]. Returns zero or more events. Reasoning
// ("thought") deltas are accumulated-and-dropped (per-token debug events would
// flood the stream); assistant text is accumulated and flushed as one message
// on the terminal "end" event.
func (c *Grok) DecodeStreamLine(state State, runID uuid.UUID, line string) ([]*domain.RunEvent, error) {
	s, ok := state.(*grokState)
	if !ok {
		return nil, domain.NewInternalError("grok: invalid state type", nil)
	}
	return c.decode(s, runID, line), nil
}

// decode is the shared decode body used by both the live stream and transcript
// replay so they stay byte-identical.
func (c *Grok) decode(s *grokState, runID uuid.UUID, line string) []*domain.RunEvent {
	ev, ok := decodeGrokStreamEvent(line)
	if !ok {
		return nil
	}

	if ev.SessionID != "" && s.sessionID == "" {
		s.sessionID = ev.SessionID
	}

	switch ev.Type {
	case "thought":
		// Reasoning delta — accumulated nowhere durable; intentionally not
		// emitted (token-level granularity would flood the event stream).
		return nil

	case "text":
		if ev.Data != "" {
			s.textBuffer.WriteString(ev.Data)
		}
		return nil

	case "end":
		s.gotEnd = true
		return flushGrokMessage(runID, s, ev)

	case "error":
		if msg := strings.TrimSpace(ev.Message); msg != "" {
			return []*domain.RunEvent{domain.NewErrorEvent(runID, "execution_error", runner.StripANSI(msg), false)}
		}
	}
	return nil
}

// flushGrokMessage emits the accumulated assistant text as one message event.
func flushGrokMessage(runID uuid.UUID, s *grokState, ev *GrokStreamEvent) []*domain.RunEvent {
	text := runner.StripANSI(strings.TrimSpace(s.textBuffer.String()))
	s.textBuffer.Reset()
	if text == "" || text == s.lastAssistant {
		return nil
	}
	s.lastAssistant = text
	return []*domain.RunEvent{domain.NewProviderMessageEvent(runID, "assistant", text, domain.MessageEventData{
		MessageID:         ev.RequestID,
		ConversationID:    ev.SessionID,
		ProviderOrigin:    "grok",
		CompletionReason:  ev.StopReason,
		Terminal:          true,
		ProviderEventType: ev.Type,
		RawEvidenceRef:    "grok:" + ev.Type,
	})}
}

// decodeGrokStreamEvent unwraps a single stdout/transcript line. Returns
// ok=false for blank or non-JSON lines (grok may print non-JSON banners to
// stdout before the stream begins).
func decodeGrokStreamEvent(line string) (*GrokStreamEvent, bool) {
	line = strings.TrimSpace(line)
	if line == "" || line[0] != '{' {
		return nil, false
	}
	var ev GrokStreamEvent
	if err := json.Unmarshal([]byte(line), &ev); err != nil {
		return nil, false
	}
	return &ev, true
}

// UpdateMetrics satisfies [Codec]. Grok emits no tool or cost events, so only
// the assistant-message turn counter is tracked.
func (c *Grok) UpdateMetrics(event *domain.RunEvent, metrics *runner.ExecutionMetrics, lastAssistant *string) {
	if event == nil {
		return
	}
	if data, ok := event.Data.(*domain.MessageEventData); ok && data.Role == "assistant" {
		*lastAssistant = data.Content
		metrics.TurnsUsed++
	}
}

// =============================================================================
// Transcript replay
// =============================================================================

// NewTranscriptParser satisfies [Codec]. Single-line parsing is provided by
// the embedded [baseCodec.ParseTranscriptLine], which delegates here.
func (c *Grok) NewTranscriptParser() runner.TranscriptParser {
	return &grokTranscriptParser{codec: c, state: &grokState{}}
}

func (p *grokTranscriptParser) SetTranscriptRetention(retain bool) { p.state.retainUser = retain }

type grokTranscriptParser struct {
	codec *Grok
	state *grokState
}

func (p *grokTranscriptParser) ParseTranscriptLine(runID uuid.UUID, line string) runner.TranscriptParseResult {
	// On-disk interactive dialect: grok writes its session transcript
	// ($GROK_HOME/sessions/<cwd>/<session>/updates.jsonl) as ACP JSON-RPC
	// session/update notifications, foreign to the flat streaming-json
	// stdout decoder (design §3). Detect and handle it before the stdout
	// path; a non-ACP line falls through so pipe-mode replay is unchanged.
	if res, ok := p.parseACPLine(runID, line); ok {
		return res
	}

	events := p.codec.decode(p.state, runID, line)
	result := runner.TranscriptParseResult{
		Events:    events,
		SessionID: p.state.sessionID,
		Timestamp: transcriptLineTimestamp(line),
	}

	if ev, ok := decodeGrokStreamEvent(line); ok {
		switch ev.Type {
		case "end":
			result.Terminal = &runner.TranscriptTerminal{Success: true, ExitCode: 0}
		case "error":
			result.Terminal = &runner.TranscriptTerminal{
				Success:      false,
				ExitCode:     1,
				ErrorMessage: runner.StripANSI(strings.TrimSpace(ev.Message)),
			}
		}
	}
	return result
}

// =============================================================================
// On-disk ACP dialect (interactive runs)
// =============================================================================

// grokACPLine is one ACP JSON-RPC notification from grok's on-disk
// updates.jsonl. The turn-completion marker arrives under the vendor-namespaced
// method "_x.ai/session/update"; ordinary updates use "session/update" — both
// are matched by the "session/update" suffix.
type grokACPLine struct {
	Method string        `json:"method"`
	Params grokACPParams `json:"params"`
}

type grokACPParams struct {
	SessionID string        `json:"sessionId"`
	Update    grokACPUpdate `json:"update"`
}

// grokACPUpdate is the union of the ACP session/update variants this codec
// maps. Unused fields for a given sessionUpdate stay zero.
type grokACPUpdate struct {
	SessionUpdate string            `json:"sessionUpdate"`
	Content       json.RawMessage   `json:"content"` // object (message chunk) or array (tool update)
	StopReason    string            `json:"stop_reason"`
	ToolCallID    string            `json:"toolCallId"`
	Title         string            `json:"title"`
	Status        string            `json:"status"`
	RawInput      json.RawMessage   `json:"rawInput"`
	RawOutput     *grokACPRawOutput `json:"rawOutput"`
}

type grokACPRawOutput struct {
	OutputForPrompt string `json:"output_for_prompt"`
}

// parseACPLine maps one on-disk ACP notification to domain events. ok=false
// means the line is not an ACP notification (blank, non-JSON, or a flat
// streaming-json stdout line) and the caller should fall back to the stdout
// decoder. Unlike grok's headless stdout (which surfaces no tool events), the
// on-disk ACP stream carries tool_call / tool_call_update records, so
// interactive runs recover richer tool events than pipe runs. Assistant text
// chunks are accumulated and flushed as one message on turn_completed, matching
// the pipe codec's flush-on-end behavior.
func (p *grokTranscriptParser) parseACPLine(runID uuid.UUID, line string) (runner.TranscriptParseResult, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || trimmed[0] != '{' {
		return runner.TranscriptParseResult{}, false
	}
	var al grokACPLine
	if err := json.Unmarshal([]byte(trimmed), &al); err != nil {
		return runner.TranscriptParseResult{}, false
	}
	if !strings.HasSuffix(al.Method, "session/update") {
		return runner.TranscriptParseResult{}, false
	}

	if al.Params.SessionID != "" && p.state.sessionID == "" {
		p.state.sessionID = al.Params.SessionID
	}
	result := runner.TranscriptParseResult{SessionID: p.state.sessionID}

	u := al.Params.Update
	switch u.SessionUpdate {
	case "agent_message_chunk":
		if text := grokACPContentText(u.Content); text != "" {
			p.state.textBuffer.WriteString(text)
		}
	case "agent_thought_chunk":
		// reasoning is intentionally not emitted.
	case "user_message_chunk":
		if p.state.retainUser {
			if text := grokACPContentText(u.Content); text != "" {
				result.Events = []*domain.RunEvent{domain.NewMessageEvent(runID, "user", runner.StripANSI(text))}
			}
		}
	case "tool_call":
		var input map[string]interface{}
		if len(u.RawInput) > 0 {
			_ = json.Unmarshal(u.RawInput, &input)
		}
		result.Events = []*domain.RunEvent{domain.NewToolCallEvent(runID, u.Title, u.ToolCallID, input)}
	case "tool_call_update":
		// Only the terminal states emit a result; in_progress/detail
		// updates would flood the stream with duplicates.
		if u.Status == "completed" || u.Status == "failed" {
			output := ""
			if u.RawOutput != nil {
				output = u.RawOutput.OutputForPrompt
			}
			if output == "" {
				output = grokACPContentText(u.Content)
			}
			var err error
			if u.Status == "failed" {
				err = errors.New("tool call failed")
			}
			result.Events = []*domain.RunEvent{domain.NewToolResultEvent(
				runID, "", u.ToolCallID, runner.StripANSI(output), err,
			)}
		}
	case "turn_completed":
		result.Events = flushGrokMessage(runID, p.state, &GrokStreamEvent{Type: "turn_completed", StopReason: "turn_completed", SessionID: p.state.sessionID})
		result.Terminal = &runner.TranscriptTerminal{Success: true, ExitCode: 0}
	}
	return result, true
}

// grokACPContentText extracts the assistant text from an ACP content field,
// which is either the message-chunk object {"type":"text","text":"…"} or the
// tool-update array [{"type":"content","content":{"type":"text","text":"…"}}].
func grokACPContentText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	// message-chunk object form
	var obj struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil && obj.Text != "" {
		return obj.Text
	}
	// tool-update array form
	var arr []struct {
		Content struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &arr); err == nil {
		var b strings.Builder
		for _, it := range arr {
			b.WriteString(it.Content.Text)
		}
		return b.String()
	}
	return ""
}

// =============================================================================
// Result classification
// =============================================================================

// ClassifyTerminalError satisfies [Codec]. Grok's recognised typed-failure
// shape is a resume against a session that no longer exists: stderr reads
// "Failed to restore session from remote: … 404 Not Found" (trace-proven). All
// other failures fall through to ErrCodeRunnerExecution.
func (c *Grok) ClassifyTerminalError(stderr string, _ int) *domain.RunnerError {
	if isGrokSessionExpired(stderr) {
		return domain.NewRunnerSessionExpiredError(c.Type(), errors.New(strings.TrimSpace(stderr)))
	}
	return nil
}

// Classify satisfies [Codec]. Order: session-expiry stderr → ReasonSessionExpired
// (mirrors [ClassifyTerminalError]); everything else → residual TextClassifier
// (unknown/bad model, auth, quota, network). Returns nil only when stderr is
// empty and exitCode == 0.
func (c *Grok) Classify(stderr string, exitCode int) *fallback.ClassifiedError {
	if stderr == "" && exitCode == 0 {
		return nil
	}
	if isGrokSessionExpired(stderr) {
		return fallback.New(fallback.ReasonSessionExpired, strings.TrimSpace(stderr), nil)
	}
	return fallback.NewTextClassifier().Classify(fallback.ClassifyInput{
		RunnerType: string(c.Type()),
		Stderr:     stderr,
		ExitCode:   exitCode,
	})
}

// isGrokSessionExpired matches grok's "session gone on resume" stderr shape.
func isGrokSessionExpired(stderr string) bool {
	s := strings.ToLower(stderr)
	if !strings.Contains(s, "session") {
		return false
	}
	return strings.Contains(s, "not found") ||
		strings.Contains(s, "restore session") ||
		strings.Contains(s, "expired")
}

// Compile-time interface checks.
var (
	_ Codec = (*Grok)(nil)
	_ State = (*grokState)(nil)
)
