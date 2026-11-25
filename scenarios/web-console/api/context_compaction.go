package main

// Idle-session compaction and per-session context watching. Web Console types
// `/compact` into a live Claude or Codex session only when the authoritative
// activity signal says the session is idle, nobody is composing, and no goal
// is active. Nothing here persists a harness setting: it never types `/model`
// or `/autocompact`, which save themselves as defaults, and watcher settings
// live only in this process. See docs/configuration/usage-modes.md.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/cli-core/agentcatalog/harness"
	"github.com/vrooli/cli-core/agenttranscript"
	"web-console/internal/sessionstore"
	"web-console/session"
	"web-console/terminal"
)

// compactMinTokens: below this a summary saves nothing. A variable only so the
// live check, whose throwaway session is far smaller, can lower it.
var compactMinTokens int64 = 20_000

const (
	// compactQuiet is how long a session must have shown no terminal output,
	// which includes keystrokes echoing into a draft, before text is typed into it.
	compactQuiet = 15 * time.Second
	// compactVerifyTimeout bounds the wait for the harness to record a compaction.
	compactVerifyTimeout = 150 * time.Second
	compactWatchInterval = 15 * time.Second
	// compactCooldown keeps a threshold watcher from compacting one session twice in a row.
	compactCooldown = 5 * time.Minute
	// compactActivityFloor is the activity confidence below which a state is not trusted.
	compactActivityFloor = 0.6
	compactHistoryLen    = 12
	maxPinnedStanceBytes = 4096
	compactStanceWait    = 30 * time.Second
)

// Skip reasons. Transient reasons can clear by themselves, so a pending
// "compact when idle" stays armed for them; the others refuse it outright.
const (
	skipBusy       = "busy: the session is working on a turn"
	skipWaiting    = "waiting for input: the session shows a prompt"
	skipUnknown    = "activity unknown: the session's state is not certain enough to type into"
	skipRecent     = "recent terminal activity: someone may be typing"
	skipDraft      = "the composer holds text that may be unsent (only a dimmed placeholder counts as empty)"
	skipComposer   = "composer not recognised: it could not be confirmed empty"
	skipExited     = "the agent has exited to a shell"
	skipProcess    = "the pane's current process could not be read"
	skipGoal       = "an active goal: compacting could clear or derail it"
	skipSmall      = "context too small to gain from a compaction"
	skipNoReading  = "context size unknown: no transcript usage was found"
	skipNoSession  = "session not found or not live"
	skipHarness    = "not a Claude or Codex session"
	skipInProgress = "a compaction of this session is already in progress"
)

func transientSkip(reason string) bool {
	switch reason {
	case skipBusy, skipWaiting, skipUnknown, skipRecent, skipDraft, skipInProgress:
		return true
	}
	return false
}

type compactSession struct {
	ID, Label, Harness, Transcript string
	Model                          string
	ModelSource                    string
	UsedTokens                     int64
	UsedTokensFound                bool
	UsedSource                     string
	ModelWindowTokens              int64
	ModelWindowSource              string
	CompactionPointTokens          int64
	CompactionPointSource          string
}

// compactDriver is the part of Web Console the manager acts through.
type compactDriver interface {
	Sessions(ctx context.Context) []compactSession
	Activity(id string) (session.Activity, bool)
	ScreenLines(id string) []string
	// Composer is the prompt box read from the styled screen.
	Composer(id string) composerState
	// CurrentCommand is the pane's foreground process now, not what the
	// session was launched as.
	CurrentCommand(id string) (string, error)
	// TypeText types text as keystrokes (never bracketed paste) and Enter submits it.
	TypeText(id, text string) error
	PressEnter(id string) error
}

type compactEvent struct {
	At           time.Time `json:"at"`
	Trigger      string    `json:"trigger"` // manual | armed | threshold | bulk
	Outcome      string    `json:"outcome"` // compacted | skipped | failed
	Reason       string    `json:"reason,omitempty"`
	TokensBefore int64     `json:"tokensBefore,omitempty"`
	// TokensAfter is the request size of the first response after the
	// compaction; it stays zero until the session has answered again.
	TokensAfter      int64 `json:"tokensAfter,omitempty"`
	PinnedStanceSent bool  `json:"pinnedStanceSent,omitempty"`
}

type compactWatch struct {
	Armed           bool  `json:"armed"`
	ThresholdTokens int64 `json:"thresholdTokens"`
	lastRun         time.Time
}

type compactCandidate struct {
	SessionID       string               `json:"sessionId"`
	Label           string               `json:"label"`
	Harness         string               `json:"harness"`
	State           string               `json:"state"`
	ContextTokens   int64                `json:"contextTokens"`
	ContextBudget   sessionContextBudget `json:"contextBudget"`
	GoalActive      bool                 `json:"goalActive"`
	Eligible        bool                 `json:"eligible"`
	SkipReason      string               `json:"skipReason,omitempty"`
	Armed           bool                 `json:"armed"`
	ThresholdTokens int64                `json:"thresholdTokens"`
	Last            *compactEvent        `json:"last,omitempty"`
}

// sessionContextBudget is the single provenance-carrying record behind the
// live meter. Missing values stay nil and carry an explicit unknown state; a
// zero is never treated as an unlimited model window.
type sessionContextBudget struct {
	Model                  string   `json:"model,omitempty"`
	ModelState             string   `json:"modelState"`
	ModelSource            string   `json:"modelSource"`
	UsedTokens             *int64   `json:"usedTokens"`
	UsedSource             string   `json:"usedSource"`
	ModelWindowTokens      *int64   `json:"modelWindowTokens"`
	ModelWindowSource      string   `json:"modelWindowSource"`
	CompactionPointTokens  *int64   `json:"compactionPointTokens"`
	CompactionPointSource  string   `json:"compactionPointSource"`
	TokensUntilCompaction  *int64   `json:"tokensUntilCompaction"`
	ForecastState          string   `json:"forecastState"`
	TokensPerSecond        *float64 `json:"tokensPerSecond"`
	SecondsUntilCompaction *float64 `json:"secondsUntilCompaction"`
	ProgressPercent        *float64 `json:"progressPercent"`
}

type contextUsageSample struct {
	at     time.Time
	used   int64
	model  string
	window int64
	point  int64
}

func contextBudgetFor(cs compactSession, used int64, hasUsage bool) sessionContextBudget {
	out := sessionContextBudget{
		Model: cs.Model, ModelState: "unknown", ModelSource: "unknown", UsedSource: "unknown",
		ModelWindowSource: "unknown", CompactionPointSource: "unknown", ForecastState: "unknown",
	}
	if strings.TrimSpace(cs.Model) != "" {
		out.ModelState = "reported"
		if strings.TrimSpace(cs.ModelSource) != "" {
			out.ModelSource = cs.ModelSource
		}
	}
	if hasUsage && used > 0 {
		out.UsedTokens = &used
		out.UsedSource = strings.TrimSpace(cs.UsedSource)
		if out.UsedSource == "" {
			out.UsedSource = "native transcript"
		}
	}
	if cs.ModelWindowTokens > 0 && supportedWindowSource(cs.ModelWindowSource) {
		window := cs.ModelWindowTokens
		out.ModelWindowTokens = &window
		out.ModelWindowSource = strings.TrimSpace(cs.ModelWindowSource)
	}
	if cs.CompactionPointTokens > 0 && strings.TrimSpace(cs.CompactionPointSource) != "" {
		point := cs.CompactionPointTokens
		out.CompactionPointTokens = &point
		out.CompactionPointSource = cs.CompactionPointSource
	}
	if out.UsedTokens != nil && out.ModelWindowTokens != nil {
		percent := float64(*out.UsedTokens) * 100 / float64(*out.ModelWindowTokens)
		out.ProgressPercent = &percent
	}
	if out.UsedTokens != nil && out.CompactionPointTokens != nil {
		remaining := max(*out.CompactionPointTokens-*out.UsedTokens, 0)
		out.TokensUntilCompaction = &remaining
		out.ForecastState = "insufficient_history"
	}
	return out
}

func supportedWindowSource(source string) bool {
	switch strings.TrimSpace(source) {
	case harness.WindowSourceClaudeStatusline, harness.WindowSourceCodexModelsCache:
		return true
	default:
		return false
	}
}

func (m *compactionManager) withContextForecast(sessionID string, budget sessionContextBudget, used int64, hasUsage bool, now time.Time) sessionContextBudget {
	if !hasUsage || used <= 0 || budget.CompactionPointTokens == nil || budget.TokensUntilCompaction == nil {
		m.mu.Lock()
		delete(m.usageSamples, sessionID)
		m.mu.Unlock()
		return budget
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	previous, found := m.usageSamples[sessionID]
	window, point := int64(0), int64(0)
	if budget.ModelWindowTokens != nil {
		window = *budget.ModelWindowTokens
	}
	if budget.CompactionPointTokens != nil {
		point = *budget.CompactionPointTokens
	}
	m.usageSamples[sessionID] = contextUsageSample{at: now, used: used, model: budget.Model, window: window, point: point}
	if !found {
		return budget
	}
	if previous.model != budget.Model || previous.window != window || previous.point != point {
		return budget
	}
	elapsed := now.Sub(previous.at).Seconds()
	delta := used - previous.used
	if elapsed < 5 || elapsed > 5*60 || delta <= 0 {
		return budget
	}
	rate := float64(delta) / elapsed
	seconds := float64(*budget.TokensUntilCompaction) / rate
	budget.TokensPerSecond = &rate
	budget.SecondsUntilCompaction = &seconds
	budget.ForecastState = "projected"
	return budget
}

type compactionManager struct {
	drv   compactDriver
	now   func() time.Time
	sleep func(time.Duration)

	mu            sync.Mutex
	watches       map[string]*compactWatch
	history       map[string][]compactEvent
	inflight      map[string]bool
	goals         map[string]goalScan
	pinnedStances map[string]string
	usageSamples  map[string]contextUsageSample
	running       bool
}

func newCompactionManager(drv compactDriver) *compactionManager {
	return &compactionManager{
		drv: drv, now: time.Now, sleep: time.Sleep,
		watches: map[string]*compactWatch{}, history: map[string][]compactEvent{}, inflight: map[string]bool{}, goals: map[string]goalScan{}, pinnedStances: map[string]string{}, usageSamples: map[string]contextUsageSample{},
	}
}

// ---- reading the transcript ------------------------------------------------

type goalScan struct {
	offset  int64
	tracker agenttranscript.GoalTracker
}

type contextReading struct {
	Tokens     int64
	Model      string
	GoalActive bool
	Found      bool
}

// readTranscriptContext reads the newest request size and whether a goal is
// active. The goal state is scanned incrementally (a goal event can be old), so
// each call reads only what was appended since the last one. A read that fails
// part way is returned as an error, so the rest of the transcript is never
// assumed to hold no goal.
func (m *compactionManager) readTranscriptContext(cs compactSession) (contextReading, error) {
	var out contextReading
	if cs.Transcript == "" {
		if cs.UsedTokensFound && cs.UsedTokens > 0 {
			out.Tokens, out.Found = cs.UsedTokens, true
		}
		return out, nil
	}
	file, err := os.Open(cs.Transcript)
	if err != nil {
		return out, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return out, err
	}
	out.Tokens, out.Model, out.Found = lastContextUsage(file, info.Size(), cs.Harness)
	m.mu.Lock()
	scan := m.goals[cs.Transcript]
	m.mu.Unlock()
	if scan.offset > info.Size() {
		scan = goalScan{} // the file was replaced
	}
	if _, err := file.Seek(scan.offset, io.SeekStart); err != nil {
		return out, err
	}
	consumed, err := scan.tracker.Read(file, cs.Harness)
	scan.offset += consumed
	m.mu.Lock()
	m.goals[cs.Transcript] = scan
	m.mu.Unlock()
	out.GoalActive = scan.tracker.Active()
	return out, err
}

// lastContextUsage finds the newest native context count and model in the last
// 2 MB of a transcript.
func lastContextUsage(file *os.File, size int64, harness string) (int64, string, bool) {
	const window = 2 << 20
	start := max(size-window, 0)
	buf := make([]byte, size-start)
	if _, err := file.ReadAt(buf, start); err != nil && err != io.EOF {
		return 0, "", false
	}
	return agenttranscript.LastContextUsage(buf, harness)
}

// compactionRecorded reports whether the transcript gained a compaction record
// after offset: Claude's compact_boundary or Codex's compacted entry.
func compactionRecorded(path string, offset int64, harness string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return false
	}
	data, err := io.ReadAll(io.LimitReader(file, 64<<20))
	if err != nil {
		return false
	}
	return agenttranscript.CompactionRecorded(data, harness)
}

// ---- decisions -------------------------------------------------------------

// composerState is what the prompt box holds. Found is false when no prompt
// marker is on screen. Text is the operator-visible text after the marker; it
// is empty only when the box is blank or shows nothing but dimmed placeholder
// text. Styling is the only evidence that text is a placeholder: wording is
// never trusted, because an operator's draft can begin with any words.
type composerState struct {
	Found bool
	Text  string
}

func isComposerMarker(r rune) bool { return r == '>' || r == '›' || r == '❯' }

func isComposerPad(r rune) bool { return r == 0 || r == ' ' || r == '\t' || r == '│' || r == '|' }

// readComposer reads the last prompt-marker row on screen, which is the
// composer for both harnesses.
func readComposer(cells [][]terminal.Cell) composerState {
	for y := len(cells) - 1; y >= 0; y-- {
		row := cells[y]
		i := 0
		for i < len(row) && isComposerPad(row[i].Rune) {
			i++
		}
		if i >= len(row) || !isComposerMarker(row[i].Rune) {
			continue
		}
		start, end := i+1, len(row)
		for end > start && isComposerPad(row[end-1].Rune) {
			end--
		}
		for start < end && isComposerPad(row[start].Rune) {
			start++
		}
		if start >= end {
			return composerState{Found: true}
		}
		placeholder, first := true, true
		var text strings.Builder
		for _, cell := range row[start:end] {
			if cell.Rune != 0 {
				text.WriteRune(cell.Rune)
			}
			if cell.Rune == 0 || cell.Rune == ' ' {
				continue
			}
			// A block cursor can sit on the placeholder's first character.
			if !cell.SGR.Faint && !(first && cell.SGR.Inverse) {
				placeholder = false
			}
			first = false
		}
		if placeholder {
			return composerState{Found: true}
		}
		return composerState{Found: true, Text: strings.TrimSpace(text.String())}
	}
	return composerState{}
}

var shellCommands = map[string]bool{
	"bash": true, "zsh": true, "sh": true, "dash": true, "fish": true, "ksh": true, "tcsh": true, "csh": true,
	"nu": true, "pwsh": true, "login": true, "-bash": true, "-zsh": true, "-sh": true, "-fish": true,
}

type sessionReading struct {
	state    string
	conf     float32
	quiet    bool
	prompt   bool
	composer composerState
	command  string
	cmdErr   error
	tokens   int64
	found    bool
	goal     bool
	readErr  error
	harness  string
	model    string
	hasState bool
}

func (m *compactionManager) inspect(cs compactSession) sessionReading {
	r := sessionReading{harness: cs.Harness}
	if act, ok := m.drv.Activity(cs.ID); ok {
		r.hasState = true
		r.state, r.conf, r.prompt = string(act.State), act.Confidence, act.Prompt != nil
		r.quiet = act.LastOutputAt.IsZero() || m.now().Sub(act.LastOutputAt) >= compactQuiet
	}
	reading, err := m.readTranscriptContext(cs)
	r.readErr, r.tokens, r.found, r.goal, r.model = err, reading.Tokens, reading.Found, reading.GoalActive, reading.Model
	r.composer = m.drv.Composer(cs.ID)
	r.command, r.cmdErr = m.drv.CurrentCommand(cs.ID)
	return r
}

// skipReason is why a session must not be compacted now, or "" when it may be.
// Order matters: permanent reasons are reported before transient ones.
func skipReason(r sessionReading) string {
	switch {
	case r.harness != "claude" && r.harness != "codex":
		return skipHarness
	case !r.hasState:
		return skipNoSession
	case r.cmdErr != nil:
		return skipProcess
	case shellCommands[filepath.Base(strings.TrimSpace(r.command))]:
		return skipExited // the stored agent type is only how the session started
	case r.goal:
		return skipGoal
	case r.readErr != nil || !r.found:
		return skipNoReading
	case r.tokens < compactMinTokens:
		return skipSmall
	case r.state == string(session.ActivityWorking):
		return skipBusy
	case r.state == string(session.ActivityWaiting) || r.prompt:
		return skipWaiting
	case r.state != string(session.ActivityIdle) || r.conf < compactActivityFloor:
		return skipUnknown
	case !r.quiet:
		return skipRecent
	case !r.composer.Found:
		return skipComposer
	case r.composer.Text != "":
		return skipDraft
	}
	return ""
}

func (m *compactionManager) find(ctx context.Context, id string) (compactSession, bool) {
	for _, cs := range m.drv.Sessions(ctx) {
		if cs.ID == id {
			return cs, true
		}
	}
	return compactSession{}, false
}

// Candidates lists every live supported session with its context record, state,
// and whether, and if not why not, Web Console can compact it now.
func (m *compactionManager) Candidates(ctx context.Context) []compactCandidate {
	sessions := m.drv.Sessions(ctx)
	out := make([]compactCandidate, 0, len(sessions))
	active := make(map[string]struct{}, len(sessions))
	for _, cs := range sessions {
		active[cs.ID] = struct{}{}
		r := m.inspect(cs)
		reason := skipReason(r)
		m.mu.Lock()
		if m.inflight[cs.ID] {
			reason = skipInProgress
		}
		watch := m.watches[cs.ID]
		var last *compactEvent
		if h := m.history[cs.ID]; len(h) > 0 {
			event := h[len(h)-1]
			last = &event
		}
		m.mu.Unlock()
		budgetSession := cs
		if r.model != "" && r.model != cs.Model {
			budgetSession.Model = r.model
			budgetSession.ModelSource = "native transcript usage"
			if resolver, ok := m.drv.(interface {
				modelWindow(context.Context, string, string) (int64, string)
			}); ok {
				budgetSession.ModelWindowTokens, budgetSession.ModelWindowSource = resolver.modelWindow(ctx, cs.Harness, r.model)
			}
		}
		budget := contextBudgetFor(budgetSession, r.tokens, r.found)
		forecastNow := time.Now()
		if m.now != nil {
			forecastNow = m.now()
		}
		budget = m.withContextForecast(cs.ID, budget, r.tokens, r.found, forecastNow)
		c := compactCandidate{
			SessionID: cs.ID, Label: cs.Label, Harness: cs.Harness, State: r.state, ContextTokens: r.tokens,
			ContextBudget: budget,
			GoalActive:    r.goal, Eligible: reason == "", SkipReason: reason, Last: last,
		}
		if watch != nil {
			c.Armed, c.ThresholdTokens = watch.Armed, watch.ThresholdTokens
		}
		out = append(out, c)
	}
	m.mu.Lock()
	for id := range m.usageSamples {
		if _, ok := active[id]; !ok {
			delete(m.usageSamples, id)
		}
	}
	m.mu.Unlock()
	sort.Slice(out, func(i, j int) bool { return out[i].ContextTokens > out[j].ContextTokens })
	return out
}

func (m *compactionManager) record(id string, event compactEvent) compactEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	h := append(m.history[id], event)
	if len(h) > compactHistoryLen {
		h = h[len(h)-compactHistoryLen:]
	}
	m.history[id] = h
	return event
}

// Compact types /compact into one session if, and only if, it is eligible at
// this moment. It re-reads the session first: a candidate list can be stale.
func (m *compactionManager) Compact(ctx context.Context, id, trigger string) compactEvent {
	event := compactEvent{At: m.now(), Trigger: trigger}
	cs, ok := m.find(ctx, id)
	if !ok {
		event.Outcome, event.Reason = "skipped", skipNoSession
		return m.record(id, event)
	}
	m.mu.Lock()
	if m.inflight[id] {
		m.mu.Unlock()
		event.Outcome, event.Reason = "skipped", skipInProgress
		return event
	}
	m.inflight[id] = true
	m.mu.Unlock()
	defer func() { m.mu.Lock(); delete(m.inflight, id); m.mu.Unlock() }()

	r := m.inspect(cs)
	event.TokensBefore = r.tokens
	if reason := skipReason(r); reason != "" {
		event.Outcome, event.Reason = "skipped", reason
		return m.record(id, event)
	}
	var offset int64
	if info, err := os.Stat(cs.Transcript); err == nil {
		offset = info.Size()
	}
	if err := m.drv.TypeText(id, "/compact"); err != nil {
		event.Outcome, event.Reason = "failed", "could not type into the session: "+err.Error()
		return m.record(id, event)
	}
	m.sleep(400 * time.Millisecond)
	if err := m.drv.PressEnter(id); err != nil {
		event.Outcome, event.Reason = "failed", "could not submit /compact: "+err.Error()
		return m.record(id, event)
	}
	deadline := m.now().Add(compactVerifyTimeout)
	pressed := 1
	for m.now().Before(deadline) {
		if ctx.Err() != nil {
			event.Outcome, event.Reason = "failed", "the request ended before the compaction was confirmed; check the session"
			return m.record(id, event)
		}
		m.sleep(1500 * time.Millisecond)
		if compactionRecorded(cs.Transcript, offset, cs.Harness) {
			event.Outcome = "compacted"
			event.TokensAfter, _ = tokensAfter(cs, offset)
			if stance, pinned := m.PinnedStance(id); pinned {
				if err := m.sendPinnedStance(ctx, cs, stance); err != nil {
					event.Reason = "compaction was recorded; pinned stance remains queued: " + err.Error()
				} else {
					event.PinnedStanceSent = true
					m.clearPinnedStance(id, stance)
				}
			}
			return m.record(id, event)
		}
		// Some narrow panes keep the command in the composer after one Enter.
		if pressed < 2 && strings.Contains(strings.Join(m.drv.ScreenLines(id), "\n"), "/compact") && m.drv.Composer(id).Text == "/compact" {
			if err := m.drv.PressEnter(id); err == nil {
				pressed++
			}
		}
	}
	event.Outcome, event.Reason = "failed", "/compact was sent but no compaction was recorded within the wait; check the session"
	return m.record(id, event)
}

// tokensAfter is the request size of the first response written after offset,
// which is when a compaction's effect on the context becomes measurable.
func tokensAfter(cs compactSession, offset int64) (int64, bool) {
	file, err := os.Open(cs.Transcript)
	if err != nil {
		return 0, false
	}
	defer file.Close()
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return 0, false
	}
	data, err := io.ReadAll(io.LimitReader(file, 16<<20))
	if err != nil {
		return 0, false
	}
	return agenttranscript.LastContextTokens(data, cs.Harness)
}

// compactBulkMax bounds one bulk request. Sessions run in parallel, so the
// request lasts about one verification wait however many are named, not one
// wait per session.
const compactBulkMax = 16

// CompactIdle acts on the named sessions only, each re-checked, and returns one
// result per session in the order asked. A busy session or one with an active
// goal is skipped with the reason stated, never waited for. At most
// compactBulkMax sessions are taken; the rest are reported as not started.
func (m *compactionManager) CompactIdle(ctx context.Context, ids []string) []compactResult {
	var unique []string
	seen := map[string]bool{}
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	results := make([]compactResult, len(unique))
	var wg sync.WaitGroup
	for i, id := range unique {
		if i >= compactBulkMax {
			results[i] = compactResult{SessionID: id, compactEvent: compactEvent{
				At: m.now(), Trigger: "bulk", Outcome: "skipped",
				Reason: fmt.Sprintf("not started: at most %d sessions are compacted per request", compactBulkMax),
			}}
			continue
		}
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			results[i] = compactResult{SessionID: id, compactEvent: m.Compact(ctx, id, "bulk")}
		}(i, id)
	}
	wg.Wait()
	return results
}

type compactResult struct {
	SessionID string `json:"sessionId"`
	compactEvent
}

// Arm requests one compaction as soon as the session is idle. An immediate
// attempt is made; only a transient skip leaves it armed.
func (m *compactionManager) Arm(ctx context.Context, id string) (compactEvent, bool) {
	event := m.Compact(ctx, id, "manual")
	if event.Outcome == "skipped" && transientSkip(event.Reason) && event.Reason != skipInProgress {
		m.mu.Lock()
		w := m.watches[id]
		if w == nil {
			w = &compactWatch{}
			m.watches[id] = w
		}
		w.Armed = true
		m.mu.Unlock()
		m.ensureRunning()
		return event, true
	}
	return event, false
}

// SetThreshold watches one session: when its context passes tokens and it is
// idle it is compacted. Zero turns the watcher off. Off by default.
func (m *compactionManager) SetThreshold(id string, tokens int64) {
	m.mu.Lock()
	w := m.watches[id]
	if w == nil {
		w = &compactWatch{}
		m.watches[id] = w
	}
	w.ThresholdTokens = tokens
	m.dropIfIdleLocked(id)
	m.mu.Unlock()
	if tokens > 0 {
		m.ensureRunning()
	}
}

func (m *compactionManager) Cancel(id string) {
	m.mu.Lock()
	if w := m.watches[id]; w != nil {
		w.Armed = false
	}
	m.dropIfIdleLocked(id)
	m.mu.Unlock()
}

func (m *compactionManager) dropIfIdleLocked(id string) {
	if w := m.watches[id]; w != nil && !w.Armed && w.ThresholdTokens <= 0 {
		delete(m.watches, id)
	}
}

func (m *compactionManager) Status(id string) (compactWatch, []compactEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var w compactWatch
	if cur := m.watches[id]; cur != nil {
		w = *cur
	}
	return w, append([]compactEvent(nil), m.history[id]...)
}

func (m *compactionManager) SetPinnedStance(id, stance string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if stance == "" {
		delete(m.pinnedStances, id)
		return
	}
	m.pinnedStances[id] = stance
}

func (m *compactionManager) PinnedStance(id string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	stance, ok := m.pinnedStances[id]
	return stance, ok
}

func (m *compactionManager) clearPinnedStance(id, expected string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.pinnedStances[id] == expected {
		delete(m.pinnedStances, id)
	}
}

// sendPinnedStance waits for a clear, quiet prompt owned by the same harness
// after compaction. It sends the operator's exact one-line text only then.
func (m *compactionManager) sendPinnedStance(ctx context.Context, cs compactSession, stance string) error {
	deadline := m.now().Add(compactStanceWait)
	expectedCommand := map[string]string{"claude": "claude", "codex": "codex"}[cs.Harness]
	if expectedCommand == "" {
		return fmt.Errorf("the harness has no verified compaction input path")
	}
	for m.now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return err
		}
		activity, ok := m.drv.Activity(cs.ID)
		command, commandErr := m.drv.CurrentCommand(cs.ID)
		fields := strings.Fields(strings.TrimSpace(command))
		baseCommand := ""
		if len(fields) > 0 {
			baseCommand = filepath.Base(fields[0])
		}
		if commandErr == nil && shellCommands[baseCommand] {
			return fmt.Errorf("session exited to a shell before the stance could be sent")
		}
		composer := m.drv.Composer(cs.ID)
		if ok && commandErr == nil && baseCommand == expectedCommand && activity.State == session.ActivityIdle && activity.Confidence >= compactActivityFloor && activity.Prompt == nil && (activity.LastOutputAt.IsZero() || m.now().Sub(activity.LastOutputAt) >= compactQuiet) && composer.Found && composer.Text == "" {
			if err := m.drv.TypeText(cs.ID, stance); err != nil {
				return fmt.Errorf("could not type the pinned stance: %w", err)
			}
			m.sleep(400 * time.Millisecond)
			if err := m.drv.PressEnter(cs.ID); err != nil {
				return fmt.Errorf("could not submit the pinned stance: %w", err)
			}
			return nil
		}
		m.sleep(1500 * time.Millisecond)
	}
	return fmt.Errorf("no verified idle prompt appeared before the stance delivery wait expired")
}

func (m *compactionManager) ensureRunning() {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	m.running = true
	m.mu.Unlock()
	go func() {
		for {
			time.Sleep(compactWatchInterval)
			if !m.tick(context.Background()) {
				return
			}
		}
	}()
}

// tick evaluates every watched session once and reports whether any watch remains.
func (m *compactionManager) tick(ctx context.Context) bool {
	m.mu.Lock()
	ids := make([]string, 0, len(m.watches))
	for id := range m.watches {
		ids = append(ids, id)
	}
	m.mu.Unlock()
	live := map[string]compactSession{}
	for _, cs := range m.drv.Sessions(ctx) {
		live[cs.ID] = cs
	}
	for _, id := range ids {
		cs, ok := live[id]
		m.mu.Lock()
		w := m.watches[id]
		if w == nil {
			m.mu.Unlock()
			continue
		}
		if !ok { // the session ended
			delete(m.watches, id)
			m.mu.Unlock()
			continue
		}
		armed, threshold, last := w.Armed, w.ThresholdTokens, w.lastRun
		m.mu.Unlock()
		r := m.inspect(cs)
		trigger := ""
		switch {
		case armed:
			trigger = "armed"
		case threshold > 0 && r.tokens >= threshold && m.now().Sub(last) >= compactCooldown:
			trigger = "threshold"
		}
		if trigger == "" || skipReason(r) != "" {
			continue // not due, or not idle yet: stay armed and look again
		}
		event := m.Compact(ctx, id, trigger)
		m.mu.Lock()
		if w := m.watches[id]; w != nil {
			w.lastRun = m.now()
			if trigger == "armed" && event.Outcome != "skipped" {
				w.Armed = false
			}
			m.dropIfIdleLocked(id)
		}
		m.mu.Unlock()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.watches) == 0 {
		m.running = false
		return false
	}
	return true
}

// ---- the Server-backed driver ---------------------------------------------

type serverCompactDriver struct{ s *Server }

func (d serverCompactDriver) Sessions(ctx context.Context) []compactSession {
	if d.s.sessionStore == nil || d.s.sessions == nil {
		return nil
	}
	rows, err := d.s.sessionStore.List(ctx)
	if err != nil {
		return nil
	}
	home, _ := os.UserHomeDir()
	var out []compactSession
	for _, row := range rows {
		if row.Status != sessionstore.StatusLive {
			continue
		}
		if _, live := d.s.sessions.Get(row.ID); !live {
			continue
		}
		cs := compactSession{ID: row.ID, Label: row.DisplayLabel}
		if cs.Label == "" {
			cs.Label = row.ID[:min(8, len(row.ID))]
		}
		switch row.AgentType {
		case sessionstore.AgentClaude:
			cs.Harness = "claude"
			if row.AgentSessionID != "" && row.CWD != "" {
				cs.Transcript = claudeTranscriptPath(home, row.CWD, row.AgentSessionID)
			}
		case sessionstore.AgentCodex:
			cs.Harness = "codex"
			cs.Transcript = codexRollout(row, sessionCodexSessionsDir(row.ID))
		case sessionstore.AgentGrok:
			cs.Harness = agenttranscript.HarnessGrok
			if row.AgentSessionID != "" {
				matches, _ := filepath.Glob(filepath.Join(sessionGrokSessionsDir(row.ID), "*", row.AgentSessionID, "updates.jsonl"))
				if len(matches) > 0 {
					cs.Transcript = matches[len(matches)-1]
				}
			}
		case sessionstore.AgentOpenCode:
			cs.Harness = agenttranscript.HarnessOpenCode
		default:
			continue
		}
		if descriptor, err := sessionstore.ParseLaunchDescriptor(row.LaunchDescriptorJSON); err == nil {
			cs.Model = strings.TrimSpace(descriptor.Model)
			if cs.Model != "" {
				cs.ModelSource = "session launch descriptor"
			}
			if descriptor.ContextLimitTokens > 0 {
				cs.CompactionPointTokens = int64(descriptor.ContextLimitTokens)
				cs.CompactionPointSource = "session launch descriptor"
			}
		}
		if row.AgentType == sessionstore.AgentOpenCode && row.AgentSessionID != "" && d.s.opencodeWatcher != nil {
			if usage, ok := d.s.opencodeWatcher.ContextUsage(row.AgentSessionID); ok && usage.Found {
				cs.UsedTokens, cs.UsedTokensFound = usage.Tokens, true
				cs.UsedSource = "OpenCode native message usage"
				if usage.Model != "" {
					cs.Model, cs.ModelSource = usage.Model, "OpenCode native message"
				}
			}
		}
		if cs.Model != "" {
			cs.ModelWindowTokens, cs.ModelWindowSource = d.modelWindow(ctx, cs.Harness, cs.Model)
		}
		out = append(out, cs)
	}
	return out
}

func (d serverCompactDriver) modelWindow(ctx context.Context, agentID, model string) (int64, string) {
	choices := d.s.GetRoleModelChoices(ctx, agentID, "local", "")
	for _, choice := range choices.Choices {
		providerModel := strings.Trim(strings.TrimSpace(choice.Provider)+"/"+strings.TrimSpace(choice.Model), "/")
		if choice.Model != model && choice.LaunchModel != model && providerModel != model {
			continue
		}
		if choice.ContextWindowTokens <= 0 || strings.TrimSpace(choice.ContextWindowSource) == "" {
			return 0, "unknown"
		}
		return choice.ContextWindowTokens, choice.ContextWindowSource
	}
	return 0, "unknown"
}

// codexRollout is the session's newest rollout file.
func codexRollout(row sessionstore.Metadata, dir string) string {
	if row.LastRolloutPath != "" {
		if _, err := os.Stat(row.LastRolloutPath); err == nil {
			return row.LastRolloutPath
		}
	}
	var newest string
	var newestAt time.Time
	_ = filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasPrefix(entry.Name(), "rollout-") || !strings.HasSuffix(entry.Name(), ".jsonl") {
			return nil
		}
		if info, ierr := entry.Info(); ierr == nil && info.ModTime().After(newestAt) {
			newest, newestAt = path, info.ModTime()
		}
		return nil
	})
	return newest
}

func (d serverCompactDriver) Activity(id string) (session.Activity, bool) {
	if detector := d.s.activityDetector(id); detector != nil {
		return detector.Current(), true
	}
	return session.Activity{}, false
}

func (d serverCompactDriver) ScreenLines(id string) []string {
	sess, ok := d.s.sessions.Get(id)
	if !ok || sess == nil {
		return nil
	}
	_, text := sess.ScreenWithText(false)
	return strings.Split(text, "\n")
}

func (d serverCompactDriver) Composer(id string) composerState {
	sess, ok := d.s.sessions.Get(id)
	if !ok || sess == nil {
		return composerState{}
	}
	view, _ := sess.ScreenWithText(false)
	return readComposer(view.Cells)
}

// CurrentCommand asks tmux what the pane is running now. Any failure is
// reported, never guessed: an unknown process is not typed into.
func (d serverCompactDriver) CurrentCommand(id string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := tmuxPaneCurrentCommand(ctx, id)
	if err != nil {
		return "", err
	}
	command := strings.TrimSpace(string(out))
	if command == "" {
		return "", fmt.Errorf("tmux reported no current command")
	}
	return command, nil
}

func (d serverCompactDriver) TypeText(id, text string) error {
	sess, ok := d.s.sessions.Get(id)
	if !ok || sess == nil {
		return fmt.Errorf("session is not live")
	}
	return sess.SendInput(session.InputText(text).WithSource("context-compaction"))
}

func (d serverCompactDriver) PressEnter(id string) error {
	sess, ok := d.s.sessions.Get(id)
	if !ok || sess == nil {
		return fmt.Errorf("session is not live")
	}
	return sess.SendInput(session.InputRaw([]byte("\r")).WithSource("context-compaction"))
}

// ---- HTTP -----------------------------------------------------------------

func (s *Server) compactionMgr() *compactionManager {
	s.compactionOnce.Do(func() {
		if s.compaction == nil {
			s.compaction = newCompactionManager(serverCompactDriver{s: s})
		}
	})
	return s.compaction
}

func (s *Server) mountContextCompaction() {
	s.router.HandleFunc("/api/v1/context-compaction/candidates", s.handleCompactionCandidates).Methods(http.MethodGet)
	s.router.HandleFunc("/api/v1/context-compaction/compact-idle", s.handleCompactIdle).Methods(http.MethodPost)
	s.router.HandleFunc("/api/v1/sessions/{id}/context-compaction", s.handleSessionCompactionGet).Methods(http.MethodGet)
	s.router.HandleFunc("/api/v1/sessions/{id}/context-compaction", s.handleSessionCompactionPost).Methods(http.MethodPost)
}

func (s *Server) handleCompactionCandidates(w http.ResponseWriter, r *http.Request) {
	writeUsageModeJSON(w, http.StatusOK, map[string]any{"candidates": s.compactionMgr().Candidates(r.Context())})
}

func (s *Server) handleCompactIdle(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireUsageModeOperator(w, r); !ok {
		return
	}
	var req struct {
		SessionIDs []string `json:"sessionIds"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&req); err != nil || len(req.SessionIDs) == 0 || len(req.SessionIDs) > compactBulkMax {
		writeUsageModeError(w, http.StatusBadRequest, fmt.Sprintf("name 1 to %d sessions to compact, taken from the candidate list", compactBulkMax))
		return
	}
	writeUsageModeJSON(w, http.StatusOK, map[string]any{"results": s.compactionMgr().CompactIdle(r.Context(), req.SessionIDs)})
}

func (s *Server) handleSessionCompactionGet(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	mgr := s.compactionMgr()
	watch, history := mgr.Status(id)
	stance, pinned := mgr.PinnedStance(id)
	writeUsageModeJSON(w, http.StatusOK, map[string]any{"watch": watch, "history": history, "pinnedStance": pinned, "pinnedStanceText": stance, "pinnedStanceMaxBytes": maxPinnedStanceBytes})
}

func (s *Server) handleSessionCompactionPost(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireUsageModeOperator(w, r); !ok {
		return
	}
	id := mux.Vars(r)["id"]
	var req struct {
		Action          string `json:"action"`
		ThresholdTokens int64  `json:"thresholdTokens"`
		Stance          string `json:"stance"`
	}
	// JSON escaping can expand quotes and backslashes in a valid max-size
	// stance, so bound the envelope above the decoded UTF-8 payload limit.
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2*maxPinnedStanceBytes+1024)).Decode(&req); err != nil {
		writeUsageModeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	mgr := s.compactionMgr()
	response := map[string]any{}
	switch req.Action {
	case "compact_when_idle":
		event, armed := mgr.Arm(r.Context(), id)
		response["result"], response["armed"] = event, armed
	case "set_threshold":
		if req.ThresholdTokens != 0 && req.ThresholdTokens < compactMinTokens {
			writeUsageModeError(w, http.StatusBadRequest, fmt.Sprintf("a threshold under %d tokens would compact sessions that gain nothing", compactMinTokens))
			return
		}
		mgr.SetThreshold(id, req.ThresholdTokens)
	case "cancel":
		mgr.Cancel(id)
	case "pin_stance":
		if _, ok := mgr.find(r.Context(), id); !ok {
			writeUsageModeError(w, http.StatusNotFound, "session not found or not live")
			return
		}
		if strings.TrimSpace(req.Stance) == "" || len(req.Stance) > maxPinnedStanceBytes || strings.ContainsAny(req.Stance, "\r\n") {
			writeUsageModeError(w, http.StatusBadRequest, "pin one non-empty line of stance text up to 4096 bytes")
			return
		}
		mgr.SetPinnedStance(id, req.Stance)
	case "unpin_stance":
		mgr.SetPinnedStance(id, "")
	default:
		writeUsageModeError(w, http.StatusBadRequest, "unknown action")
		return
	}
	watch, history := mgr.Status(id)
	response["watch"], response["history"] = watch, history
	stance, pinned := mgr.PinnedStance(id)
	response["pinnedStance"], response["pinnedStanceText"] = pinned, stance
	response["pinnedStanceMaxBytes"] = maxPinnedStanceBytes
	writeUsageModeJSON(w, http.StatusOK, response)
}
