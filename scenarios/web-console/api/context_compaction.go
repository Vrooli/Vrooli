package main

// Idle-session compaction and per-session context watching. Web Console types
// `/compact` into a live Claude or Codex session only when the authoritative
// activity signal says the session is idle, nobody is composing, and no goal
// is active. Nothing here persists a harness setting: it never types `/model`
// or `/autocompact`, which save themselves as defaults, and watcher settings
// live only in this process. See docs/configuration/usage-modes.md.

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"web-console/internal/sessionstore"
	"web-console/session"
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
)

// Skip reasons. Transient reasons can clear by themselves, so a pending
// "compact when idle" stays armed for them; the others refuse it outright.
const (
	skipBusy       = "busy: the session is working on a turn"
	skipWaiting    = "waiting for input: the session shows a prompt"
	skipUnknown    = "activity unknown: the session's state is not certain enough to type into"
	skipRecent     = "recent terminal activity: someone may be typing"
	skipDraft      = "the composer holds unsent text"
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
}

// compactDriver is the part of Web Console the manager acts through.
type compactDriver interface {
	Sessions(ctx context.Context) []compactSession
	Activity(id string) (session.Activity, bool)
	ScreenLines(id string) []string
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
	TokensAfter int64 `json:"tokensAfter,omitempty"`
}

type compactWatch struct {
	Armed           bool  `json:"armed"`
	ThresholdTokens int64 `json:"thresholdTokens"`
	lastRun         time.Time
}

type compactCandidate struct {
	SessionID       string        `json:"sessionId"`
	Label           string        `json:"label"`
	Harness         string        `json:"harness"`
	State           string        `json:"state"`
	ContextTokens   int64         `json:"contextTokens"`
	GoalActive      bool          `json:"goalActive"`
	Eligible        bool          `json:"eligible"`
	SkipReason      string        `json:"skipReason,omitempty"`
	Armed           bool          `json:"armed"`
	ThresholdTokens int64         `json:"thresholdTokens"`
	Last            *compactEvent `json:"last,omitempty"`
}

type compactionManager struct {
	drv   compactDriver
	now   func() time.Time
	sleep func(time.Duration)

	mu       sync.Mutex
	watches  map[string]*compactWatch
	history  map[string][]compactEvent
	inflight map[string]bool
	goals    map[string]goalScan
	running  bool
}

func newCompactionManager(drv compactDriver) *compactionManager {
	return &compactionManager{drv: drv, now: time.Now, sleep: time.Sleep,
		watches: map[string]*compactWatch{}, history: map[string][]compactEvent{}, inflight: map[string]bool{}, goals: map[string]goalScan{}}
}

// ---- reading the transcript ------------------------------------------------

type goalScan struct {
	offset int64
	active bool
}

type contextReading struct {
	Tokens     int64
	GoalActive bool
	Found      bool
}

// readTranscriptContext reads the newest request size and whether a goal is
// active. The goal state is scanned incrementally (a goal event can be old), so
// each call reads only what was appended since the last one.
func (m *compactionManager) readTranscriptContext(cs compactSession) (contextReading, error) {
	var out contextReading
	if cs.Transcript == "" {
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
	out.Tokens, out.Found = lastContextTokens(file, info.Size(), cs.Harness)
	m.mu.Lock()
	scan := m.goals[cs.Transcript]
	m.mu.Unlock()
	if scan.offset > info.Size() {
		scan = goalScan{} // the file was replaced
	}
	if _, err := file.Seek(scan.offset, io.SeekStart); err != nil {
		return out, err
	}
	reader := bufio.NewReaderSize(file, 1<<20)
	offset := scan.offset
	for {
		line, err := reader.ReadBytes('\n')
		if err == io.EOF && !bytes.HasSuffix(line, []byte("\n")) {
			break // a partial last line is read again next time
		}
		if len(line) > 0 {
			offset += int64(len(line))
			if active, ok := goalActiveInLine(line, cs.Harness); ok {
				scan.active = active
			}
		}
		if err != nil {
			break
		}
	}
	scan.offset = offset
	m.mu.Lock()
	m.goals[cs.Transcript] = scan
	m.mu.Unlock()
	out.GoalActive = scan.active
	return out, nil
}

// goalActiveInLine reports a goal change recorded on one transcript line.
// Codex records thread_goal_updated; Claude records goal_status. Any Codex
// status other than paused or finished counts as active, so an unfamiliar
// status keeps a session from being compacted.
func goalActiveInLine(line []byte, harness string) (active, ok bool) {
	switch harness {
	case "codex":
		if !bytes.Contains(line, []byte(`"thread_goal_updated"`)) {
			return false, false
		}
		var event struct {
			Payload struct {
				Type string `json:"type"`
				Goal struct {
					Status string `json:"status"`
				} `json:"goal"`
			} `json:"payload"`
		}
		if json.Unmarshal(line, &event) != nil || event.Payload.Type != "thread_goal_updated" {
			return false, false
		}
		switch strings.ToLower(event.Payload.Goal.Status) {
		case "paused", "complete", "completed", "cleared", "abandoned":
			return false, true
		}
		return true, true
	case "claude":
		if !bytes.Contains(line, []byte(`"goal_status"`)) {
			return false, false
		}
		var value any
		if json.Unmarshal(line, &value) != nil {
			return false, false
		}
		return claudeGoalActive(value)
	}
	return false, false
}

func claudeGoalActive(value any) (active, ok bool) {
	switch v := value.(type) {
	case []any:
		for _, child := range v {
			if a, found := claudeGoalActive(child); found {
				return a, true
			}
		}
	case map[string]any:
		if v["type"] == "goal_status" {
			met, _ := v["met"].(bool)
			return !met, true
		}
		for _, child := range v {
			if a, found := claudeGoalActive(child); found {
				return a, true
			}
		}
	}
	return false, false
}

// lastContextTokens finds the request size of the newest response in the last
// 2 MB of a transcript: Claude's assistant usage (input plus cache reads and
// writes) or Codex's last_token_usage input.
func lastContextTokens(file *os.File, size int64, harness string) (int64, bool) {
	const window = 2 << 20
	start := max(size-window, 0)
	buf := make([]byte, size-start)
	if _, err := file.ReadAt(buf, start); err != nil && err != io.EOF {
		return 0, false
	}
	return lastTokensIn(buf, harness)
}

func lastTokensIn(buf []byte, harness string) (int64, bool) {
	lines := bytes.Split(buf, []byte("\n"))
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
		switch harness {
		case "claude":
			if !bytes.Contains(line, []byte(`"usage"`)) || !bytes.Contains(line, []byte(`"assistant"`)) {
				continue
			}
			var entry struct {
				Type        string `json:"type"`
				IsSidechain bool   `json:"isSidechain"`
				Message     struct {
					Usage struct {
						Input  int64 `json:"input_tokens"`
						Read   int64 `json:"cache_read_input_tokens"`
						Create int64 `json:"cache_creation_input_tokens"`
					} `json:"usage"`
				} `json:"message"`
			}
			if json.Unmarshal(line, &entry) != nil || entry.Type != "assistant" || entry.IsSidechain {
				continue
			}
			if total := entry.Message.Usage.Input + entry.Message.Usage.Read + entry.Message.Usage.Create; total > 0 {
				return total, true
			}
		case "codex":
			if !bytes.Contains(line, []byte(`"token_count"`)) {
				continue
			}
			var entry struct {
				Payload struct {
					Type string `json:"type"`
					Info struct {
						Last struct {
							Input int64 `json:"input_tokens"`
						} `json:"last_token_usage"`
					} `json:"info"`
				} `json:"payload"`
			}
			if json.Unmarshal(line, &entry) == nil && entry.Payload.Type == "token_count" && entry.Payload.Info.Last.Input > 0 {
				return entry.Payload.Info.Last.Input, true
			}
		}
	}
	return 0, false
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
	if harness == "claude" {
		return bytes.Contains(data, []byte(`"compact_boundary"`))
	}
	return bytes.Contains(data, []byte(`"type":"compacted"`))
}

// ---- decisions -------------------------------------------------------------

var composerLine = regexp.MustCompile(`^[\s│|]*[>›❯]\s?(.*?)[\s│|]*$`)

// composerPlaceholder matches the hint text an empty composer shows.
var composerPlaceholder = regexp.MustCompile(`(?i)^(try ["']|ask codex|improve documentation|write tests|explain |summari[sz]e |find and fix|implement |use /skills|\? for shortcuts|type your)`)

// composerDraft returns the unsent text in the composer, or "". The last
// prompt-marker line on screen is the composer for both harnesses.
func composerDraft(lines []string) string {
	for i := len(lines) - 1; i >= 0; i-- {
		if m := composerLine.FindStringSubmatch(lines[i]); m != nil {
			text := strings.TrimSpace(m[1])
			if text == "" || composerPlaceholder.MatchString(text) {
				return ""
			}
			return text
		}
	}
	return ""
}

type sessionReading struct {
	state    string
	conf     float32
	quiet    bool
	prompt   bool
	draft    string
	tokens   int64
	found    bool
	goal     bool
	readErr  error
	harness  string
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
	r.readErr, r.tokens, r.found, r.goal = err, reading.Tokens, reading.Found, reading.GoalActive
	r.draft = composerDraft(m.drv.ScreenLines(cs.ID))
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
	case r.draft != "":
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

// Candidates lists every live Claude and Codex session with its size, state and
// whether, and if not why not, it could be compacted now.
func (m *compactionManager) Candidates(ctx context.Context) []compactCandidate {
	sessions := m.drv.Sessions(ctx)
	out := make([]compactCandidate, 0, len(sessions))
	for _, cs := range sessions {
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
		c := compactCandidate{SessionID: cs.ID, Label: cs.Label, Harness: cs.Harness, State: r.state, ContextTokens: r.tokens,
			GoalActive: r.goal, Eligible: reason == "", SkipReason: reason, Last: last}
		if watch != nil {
			c.Armed, c.ThresholdTokens = watch.Armed, watch.ThresholdTokens
		}
		out = append(out, c)
	}
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
		m.sleep(1500 * time.Millisecond)
		if compactionRecorded(cs.Transcript, offset, cs.Harness) {
			event.Outcome = "compacted"
			event.TokensAfter, _ = tokensAfter(cs, offset)
			return m.record(id, event)
		}
		// Some narrow panes keep the command in the composer after one Enter.
		if pressed < 2 && strings.Contains(strings.Join(m.drv.ScreenLines(id), "\n"), "/compact") && composerDraft(m.drv.ScreenLines(id)) == "/compact" {
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
	return lastTokensIn(data, cs.Harness)
}

// CompactIdle acts on the named sessions only, each re-checked, and returns one
// result per session. A busy session or one with an active goal is skipped with
// the reason stated, never waited for.
func (m *compactionManager) CompactIdle(ctx context.Context, ids []string) []compactResult {
	results := make([]compactResult, 0, len(ids))
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		results = append(results, compactResult{SessionID: id, compactEvent: m.Compact(ctx, id, "bulk")})
	}
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
		default:
			continue
		}
		out = append(out, cs)
	}
	return out
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
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&req); err != nil || len(req.SessionIDs) == 0 || len(req.SessionIDs) > 64 {
		writeUsageModeError(w, http.StatusBadRequest, "name the sessions to compact, taken from the candidate list")
		return
	}
	writeUsageModeJSON(w, http.StatusOK, map[string]any{"results": s.compactionMgr().CompactIdle(r.Context(), req.SessionIDs)})
}

func (s *Server) handleSessionCompactionGet(w http.ResponseWriter, r *http.Request) {
	watch, history := s.compactionMgr().Status(mux.Vars(r)["id"])
	writeUsageModeJSON(w, http.StatusOK, map[string]any{"watch": watch, "history": history})
}

func (s *Server) handleSessionCompactionPost(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireUsageModeOperator(w, r); !ok {
		return
	}
	id := mux.Vars(r)["id"]
	var req struct {
		Action          string `json:"action"`
		ThresholdTokens int64  `json:"thresholdTokens"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&req); err != nil {
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
	default:
		writeUsageModeError(w, http.StatusBadRequest, "unknown action")
		return
	}
	watch, history := mgr.Status(id)
	response["watch"], response["history"] = watch, history
	writeUsageModeJSON(w, http.StatusOK, response)
}
