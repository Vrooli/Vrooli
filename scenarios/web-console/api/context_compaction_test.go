package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/cli-core/usagemode"
	"web-console/internal/sessionstore"
	"web-console/session"
)

// fakeDriver stands in for live sessions. Typing "/compact" then Enter appends
// the compaction record the real harness would write.
type fakeDriver struct {
	sessions []compactSession
	activity map[string]session.Activity
	screen   map[string][]string
	typed    []string // "<id>:<text>" for every TypeText, "<id>:<enter>" for every Enter
	onEnter  func(id string)
}

func (f *fakeDriver) Sessions(context.Context) []compactSession { return f.sessions }
func (f *fakeDriver) Activity(id string) (session.Activity, bool) {
	a, ok := f.activity[id]
	return a, ok
}
func (f *fakeDriver) ScreenLines(id string) []string { return f.screen[id] }
func (f *fakeDriver) TypeText(id, text string) error {
	f.typed = append(f.typed, id+":"+text)
	return nil
}
func (f *fakeDriver) PressEnter(id string) error {
	f.typed = append(f.typed, id+":<enter>")
	if f.onEnter != nil {
		f.onEnter(id)
	}
	return nil
}

func idle(quietFor time.Duration) session.Activity {
	return session.Activity{State: session.ActivityIdle, Confidence: 0.9, LastOutputAt: time.Now().Add(-quietFor)}
}

func appendLine(t *testing.T, path, line string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := fmt.Fprintln(f, line); err != nil {
		t.Fatal(err)
	}
}

func codexUsage(tokens int) string {
	return fmt.Sprintf(`{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":%d}}}}`, tokens)
}

func codexGoal(status string) string {
	return fmt.Sprintf(`{"type":"event_msg","payload":{"type":"thread_goal_updated","goal":{"objective":"ship it","status":%q}}}`, status)
}

func claudeUsage(input, read, create int) string {
	return fmt.Sprintf(`{"type":"assistant","isSidechain":false,"message":{"usage":{"input_tokens":%d,"cache_read_input_tokens":%d,"cache_creation_input_tokens":%d}}}`, input, read, create)
}

type compactionFixture struct {
	m   *compactionManager
	drv *fakeDriver
	dir string
}

func newCompactionFixture(t *testing.T) *compactionFixture {
	t.Helper()
	drv := &fakeDriver{activity: map[string]session.Activity{}, screen: map[string][]string{}}
	m := newCompactionManager(drv)
	m.sleep = func(time.Duration) {}
	return &compactionFixture{m: m, drv: drv, dir: t.TempDir()}
}

// add registers a live session whose transcript already holds the given lines.
func (f *compactionFixture) add(t *testing.T, id, harness string, act session.Activity, lines ...string) string {
	t.Helper()
	path := filepath.Join(f.dir, id+".jsonl")
	for _, line := range lines {
		appendLine(t, path, line)
	}
	f.drv.sessions = append(f.drv.sessions, compactSession{ID: id, Label: id, Harness: harness, Transcript: path})
	f.drv.activity[id] = act
	f.drv.screen[id] = []string{"", "> ", "  ? for shortcuts"}
	marker := `{"type":"system","subtype":"compact_boundary"}`
	if harness == "codex" {
		marker = `{"type":"compacted","payload":{}}`
	}
	prev := f.drv.onEnter
	f.drv.onEnter = func(done string) {
		if prev != nil {
			prev(done)
		}
		if done == id {
			appendLine(t, path, marker)
		}
	}
	return path
}

func (f *compactionFixture) onlyCompactTyped(t *testing.T, id string) {
	t.Helper()
	for _, typed := range f.drv.typed {
		if strings.HasPrefix(typed, id+":") && typed != id+":/compact" && typed != id+":<enter>" {
			t.Fatalf("typed %q into %s: only /compact and Enter may ever be typed (never /model or /autocompact, which persist as defaults)", typed, id)
		}
	}
}

func TestIdleSessionIsCompactedWithOnlySlashCompact(t *testing.T) {
	f := newCompactionFixture(t)
	for _, tc := range []struct{ id, harness, usage string }{{"claude1", "claude", claudeUsage(0, 100000, 20000)}, {"codex1", "codex", codexUsage(120000)}} {
		f.add(t, tc.id, tc.harness, idle(time.Minute), tc.usage)
		event := f.m.Compact(context.Background(), tc.id, "manual")
		if event.Outcome != "compacted" || event.TokensBefore != 120000 || event.TokensAfter != 0 {
			t.Fatalf("%s: %+v", tc.id, event)
		}
		if got := f.drv.typed[len(f.drv.typed)-2:]; got[0] != tc.id+":/compact" || got[1] != tc.id+":<enter>" {
			t.Fatalf("%s typed %v", tc.id, got)
		}
		f.onlyCompactTyped(t, tc.id)
	}
}

func TestBusyWaitingUnknownRecentAndDraftSessionsAreNeverTypedInto(t *testing.T) {
	cases := map[string]struct {
		act    session.Activity
		screen []string
		reason string
	}{
		"busy":    {session.Activity{State: session.ActivityWorking, Confidence: 0.95, LastOutputAt: time.Now().Add(-time.Hour)}, nil, skipBusy},
		"waiting": {session.Activity{State: session.ActivityWaiting, Confidence: 0.95, LastOutputAt: time.Now().Add(-time.Hour)}, nil, skipWaiting},
		"prompt":  {session.Activity{State: session.ActivityIdle, Confidence: 0.95, LastOutputAt: time.Now().Add(-time.Hour), Prompt: nil}, nil, ""},
		"unknown": {session.Activity{State: session.ActivityUnknown, Confidence: 0.9, LastOutputAt: time.Now().Add(-time.Hour)}, nil, skipUnknown},
		"unsure":  {session.Activity{State: session.ActivityIdle, Confidence: 0.5, LastOutputAt: time.Now().Add(-time.Hour)}, nil, skipUnknown},
		"typing":  {idle(2 * time.Second), nil, skipRecent},
		"draft":   {idle(time.Minute), []string{"", "> refactor the parser to", ""}, skipDraft},
	}
	for name, tc := range cases {
		if tc.reason == "" {
			continue
		}
		t.Run(name, func(t *testing.T) {
			f := newCompactionFixture(t)
			path := f.add(t, "s1", "codex", tc.act, codexUsage(150000))
			if tc.screen != nil {
				f.drv.screen["s1"] = tc.screen
			}
			event := f.m.Compact(context.Background(), "s1", "manual")
			if event.Outcome != "skipped" || event.Reason != tc.reason {
				t.Fatalf("event = %+v, want skipped: %s", event, tc.reason)
			}
			if len(f.drv.typed) != 0 {
				t.Fatalf("typed %v into a session that must be left alone", f.drv.typed)
			}
			if compactionRecorded(path, 0, "codex") {
				t.Fatal("the transcript shows a compaction")
			}
		})
	}
}

func TestSessionWithAnActiveGoalIsNeverCompactedButAPausedOneMayBe(t *testing.T) {
	for _, harness := range []string{"codex", "claude"} {
		active := codexGoal("active")
		paused := codexGoal("paused")
		usage := codexUsage(150000)
		if harness == "claude" {
			active = `{"type":"system","goal_status":{"type":"goal_status","condition":"tests pass","met":false}}`
			paused = `{"type":"system","goal_status":{"type":"goal_status","condition":"tests pass","met":true}}`
			usage = claudeUsage(5, 140000, 10000)
		}
		f := newCompactionFixture(t)
		f.add(t, "g1", harness, idle(time.Minute), active, usage)
		event := f.m.Compact(context.Background(), "g1", "bulk")
		if event.Outcome != "skipped" || event.Reason != skipGoal || len(f.drv.typed) != 0 {
			t.Fatalf("%s active goal: %+v typed=%v", harness, event, f.drv.typed)
		}
		// The goal pauses (or is met): the scan is incremental and must notice.
		appendLine(t, f.drv.sessions[0].Transcript, paused)
		if event = f.m.Compact(context.Background(), "g1", "bulk"); event.Outcome != "compacted" {
			t.Fatalf("%s paused goal: %+v", harness, event)
		}
	}
}

func TestUnknownGoalStatusCountsAsActive(t *testing.T) {
	f := newCompactionFixture(t)
	f.add(t, "g1", "codex", idle(time.Minute), codexGoal("pursuing"), codexUsage(150000))
	if event := f.m.Compact(context.Background(), "g1", "bulk"); event.Reason != skipGoal {
		t.Fatalf("an unfamiliar goal status must block compaction: %+v", event)
	}
}

func TestSmallOrUnmeasuredContextIsSkipped(t *testing.T) {
	f := newCompactionFixture(t)
	f.add(t, "small", "codex", idle(time.Minute), codexUsage(5000))
	f.add(t, "blank", "claude", idle(time.Minute), `{"type":"user"}`)
	if e := f.m.Compact(context.Background(), "small", "bulk"); e.Reason != skipSmall {
		t.Fatalf("small: %+v", e)
	}
	if e := f.m.Compact(context.Background(), "blank", "bulk"); e.Reason != skipNoReading {
		t.Fatalf("blank: %+v", e)
	}
	if len(f.drv.typed) != 0 {
		t.Fatalf("typed %v", f.drv.typed)
	}
}

func TestCompactIdleActsOnNamedSessionsAndStatesEveryResult(t *testing.T) {
	f := newCompactionFixture(t)
	f.add(t, "idle1", "codex", idle(time.Minute), codexUsage(200000))
	f.add(t, "busy1", "claude", session.Activity{State: session.ActivityWorking, Confidence: 0.95, LastOutputAt: time.Now()}, claudeUsage(1, 180000, 1000))
	f.add(t, "goal1", "codex", idle(time.Minute), codexGoal("active"), codexUsage(250000))
	f.add(t, "bystander", "codex", idle(time.Minute), codexUsage(300000)) // not named: must be left alone

	candidates := f.m.Candidates(context.Background())
	if len(candidates) != 4 || candidates[0].SessionID != "bystander" || candidates[0].ContextTokens != 300000 {
		t.Fatalf("candidates must list sizes, largest first: %+v", candidates)
	}
	byID := map[string]compactCandidate{}
	for _, c := range candidates {
		byID[c.SessionID] = c
	}
	if !byID["idle1"].Eligible || byID["busy1"].Eligible || byID["busy1"].SkipReason != skipBusy || byID["goal1"].SkipReason != skipGoal || !byID["goal1"].GoalActive {
		t.Fatalf("candidate reasons: %+v", byID)
	}
	if len(f.drv.typed) != 0 {
		t.Fatal("listing candidates must not type anything")
	}

	results := f.m.CompactIdle(context.Background(), []string{"idle1", "busy1", "goal1", "ghost", "idle1"})
	got := map[string]string{}
	for _, r := range results {
		got[r.SessionID] = r.Outcome + ": " + r.Reason
	}
	want := map[string]string{"idle1": "compacted: ", "busy1": "skipped: " + skipBusy, "goal1": "skipped: " + skipGoal, "ghost": "skipped: " + skipNoSession}
	if len(results) != 4 {
		t.Fatalf("one result per distinct named session, got %v", results)
	}
	for id, w := range want {
		if got[id] != w {
			t.Errorf("%s = %q, want %q", id, got[id], w)
		}
	}
	for _, typed := range f.drv.typed {
		if !strings.HasPrefix(typed, "idle1:") {
			t.Fatalf("typed %q into a session that was busy, had a goal or was not named", typed)
		}
	}
}

func TestThresholdWatcherIsOffByDefaultAndCompactsOnlyAtIdlePastTheThreshold(t *testing.T) {
	f := newCompactionFixture(t)
	f.add(t, "w1", "codex", idle(time.Minute), codexUsage(180000))
	if f.m.tick(context.Background()) || len(f.drv.typed) != 0 {
		t.Fatal("with no watch set nothing may be typed")
	}

	f.m.SetThreshold("w1", 200000)
	f.m.tick(context.Background())
	if len(f.drv.typed) != 0 {
		t.Fatalf("below the threshold nothing happens: %v", f.drv.typed)
	}

	path := f.drv.sessions[0].Transcript
	appendLine(t, path, codexUsage(220000))
	f.drv.activity["w1"] = session.Activity{State: session.ActivityWorking, Confidence: 0.95, LastOutputAt: time.Now()}
	f.m.tick(context.Background())
	if len(f.drv.typed) != 0 {
		t.Fatalf("past the threshold but busy: nothing may be typed, got %v", f.drv.typed)
	}

	f.drv.activity["w1"] = idle(time.Minute)
	f.m.tick(context.Background())
	if len(f.drv.typed) != 2 {
		t.Fatalf("idle and past the threshold must compact once: %v", f.drv.typed)
	}
	_, history := f.m.Status("w1")
	if len(history) != 1 || history[0].Trigger != "threshold" || history[0].Outcome != "compacted" || history[0].TokensBefore != 220000 {
		t.Fatalf("the watcher must show what happened: %+v", history)
	}

	appendLine(t, path, codexUsage(230000)) // regrown at once: the cooldown holds it back
	f.m.tick(context.Background())
	if len(f.drv.typed) != 2 {
		t.Fatalf("cooldown ignored: %v", f.drv.typed)
	}

	f.m.SetThreshold("w1", 0)
	if watch, _ := f.m.Status("w1"); watch.ThresholdTokens != 0 || watch.Armed {
		t.Fatalf("watch not cleared: %+v", watch)
	}
}

func TestCompactWhenIdleWaitsOutABusyTurnThenFiresOnce(t *testing.T) {
	f := newCompactionFixture(t)
	f.add(t, "a1", "claude", session.Activity{State: session.ActivityWorking, Confidence: 0.95, LastOutputAt: time.Now()}, claudeUsage(1, 150000, 0))
	event, armed := f.m.Arm(context.Background(), "a1")
	if !armed || event.Reason != skipBusy || len(f.drv.typed) != 0 {
		t.Fatalf("a busy session stays armed and untouched: armed=%v %+v typed=%v", armed, event, f.drv.typed)
	}
	f.m.tick(context.Background())
	if len(f.drv.typed) != 0 {
		t.Fatal("still busy")
	}
	f.drv.activity["a1"] = idle(time.Minute)
	f.m.tick(context.Background())
	if len(f.drv.typed) != 2 {
		t.Fatalf("idle now: expected one /compact, got %v", f.drv.typed)
	}
	if watch, _ := f.m.Status("a1"); watch.Armed {
		t.Fatal("a one-shot arm must be spent after it fires")
	}
	f.m.tick(context.Background())
	if len(f.drv.typed) != 2 {
		t.Fatalf("fired twice: %v", f.drv.typed)
	}
}

func TestCompactWhenIdleDoesNotArmForAGoalSession(t *testing.T) {
	f := newCompactionFixture(t)
	f.add(t, "g1", "codex", idle(time.Minute), codexGoal("active"), codexUsage(200000))
	event, armed := f.m.Arm(context.Background(), "g1")
	if armed || event.Reason != skipGoal {
		t.Fatalf("a goal is not a wait-it-out reason: armed=%v %+v", armed, event)
	}
}

func TestContextTokensComeFromTheNewestResponse(t *testing.T) {
	f := newCompactionFixture(t)
	f.add(t, "c1", "claude", idle(time.Minute), claudeUsage(10, 1000, 0), `{"type":"assistant","isSidechain":true,"message":{"usage":{"input_tokens":900000}}}`, claudeUsage(20, 60000, 5000))
	f.add(t, "x1", "codex", idle(time.Minute), codexUsage(40000), codexUsage(90000))
	got := map[string]int64{}
	for _, c := range f.m.Candidates(context.Background()) {
		got[c.SessionID] = c.ContextTokens
	}
	if got["c1"] != 65020 || got["x1"] != 90000 {
		t.Fatalf("sizes = %v (sub-agent usage must not count)", got)
	}
}

func TestComposerDraftIgnoresPlaceholders(t *testing.T) {
	for screen, want := range map[string]string{
		"> ":                           "",
		`> Try "fix lint errors"`:      "",
		"│ > hello world     │":        "hello world",
		"› Ask Codex to do anything":   "",
		"› rename the helper function": "rename the helper function",
	} {
		if got := composerDraft([]string{"output", screen, ""}); got != want {
			t.Errorf("composerDraft(%q) = %q, want %q", screen, got, want)
		}
	}
}

// ---- HTTP ----

func compactionServer(t *testing.T) (*Server, *compactionFixture) {
	t.Helper()
	f := newCompactionFixture(t)
	s := &Server{router: mux.NewRouter(), compaction: f.m}
	s.mountContextCompaction()
	return s, f
}

func TestCompactionControlsRefuseAgentsAndAnonymousCallers(t *testing.T) {
	s, f := compactionServer(t)
	f.add(t, "s1", "codex", idle(time.Minute), codexUsage(150000))
	for _, kind := range []identity.ActorKind{"", identity.ActorAgent, identity.ActorService} {
		for _, call := range []struct{ path, body string }{
			{"/api/v1/context-compaction/compact-idle", `{"sessionIds":["s1"]}`},
			{"/api/v1/sessions/s1/context-compaction", `{"action":"compact_when_idle"}`},
			{"/api/v1/sessions/s1/context-compaction", `{"action":"set_threshold","thresholdTokens":200000}`},
		} {
			if rec := usageModeCall(t, s, http.MethodPost, call.path, call.body, kind); rec.Code != http.StatusForbidden {
				t.Errorf("kind %q %s = %d, want 403", kind, call.path, rec.Code)
			}
		}
	}
	if len(f.drv.typed) != 0 {
		t.Fatalf("a refused caller reached the session: %v", f.drv.typed)
	}
	if watch, _ := f.m.Status("s1"); watch.ThresholdTokens != 0 || watch.Armed {
		t.Fatal("a refused caller changed a watch")
	}
}

func TestOperatorCompactsThroughTheAPI(t *testing.T) {
	s, f := compactionServer(t)
	f.add(t, "s1", "codex", idle(time.Minute), codexUsage(150000))
	f.add(t, "s2", "codex", session.Activity{State: session.ActivityWorking, Confidence: 0.9, LastOutputAt: time.Now()}, codexUsage(150000))

	list := usageModeCall(t, s, http.MethodGet, "/api/v1/context-compaction/candidates", "", "")
	var listed struct {
		Candidates []compactCandidate `json:"candidates"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &listed); err != nil || len(listed.Candidates) != 2 {
		t.Fatalf("list: %v %s", err, list.Body.String())
	}

	if rec := usageModeCall(t, s, http.MethodPost, "/api/v1/context-compaction/compact-idle", `{"sessionIds":[]}`, identity.ActorHuman); rec.Code != http.StatusBadRequest {
		t.Fatalf("an empty selection must be refused, not mean 'everything': %d", rec.Code)
	}
	rec := usageModeCall(t, s, http.MethodPost, "/api/v1/context-compaction/compact-idle", `{"sessionIds":["s1","s2"]}`, identity.ActorHuman)
	var out struct {
		Results []struct {
			SessionID string `json:"sessionId"`
			Outcome   string `json:"outcome"`
			Reason    string `json:"reason"`
		} `json:"results"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || len(out.Results) != 2 || out.Results[0].Outcome != "compacted" || out.Results[1].Outcome != "skipped" || out.Results[1].Reason != skipBusy {
		t.Fatalf("results: %v %s", err, rec.Body.String())
	}

	rec = usageModeCall(t, s, http.MethodPost, "/api/v1/sessions/s2/context-compaction", `{"action":"set_threshold","thresholdTokens":1000}`, identity.ActorHuman)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("a useless threshold must be refused: %d", rec.Code)
	}
	rec = usageModeCall(t, s, http.MethodPost, "/api/v1/sessions/s2/context-compaction", `{"action":"set_threshold","thresholdTokens":300000}`, identity.ActorHuman)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"thresholdTokens":300000`) {
		t.Fatalf("set threshold: %d %s", rec.Code, rec.Body.String())
	}
	usageModeCall(t, s, http.MethodPost, "/api/v1/sessions/s2/context-compaction", `{"action":"cancel"}`, identity.ActorHuman)
	if watch, _ := f.m.Status("s2"); watch.ThresholdTokens != 300000 {
		// cancel only disarms the one-shot; the threshold is cleared by setting it to 0.
		t.Fatalf("cancel must not clear a threshold: %+v", watch)
	}
}

func TestManagedCodexLaunchCarriesAModeContextLimitForThatLaunchOnly(t *testing.T) {
	usageModeFixture(t)
	if args := strings.Join(codexLaunchArgs(sessionstore.LaunchDescriptor{Agent: "codex"}), " "); strings.Contains(args, "auto_compact") {
		t.Fatalf("no mode, no limit: %s", args)
	}
	expires := time.Now().Add(time.Hour)
	mode := usagemode.Mode{ID: "um-ctx", StartedAt: time.Now().Add(-time.Minute), Until: usagemode.Until{Kind: usagemode.UntilTime}, ExpiresAt: &expires,
		Runners: map[string]usagemode.RunnerSettings{"codex": {ContextLimitTokens: 180000}}}
	if err := usagemode.Write(usagemode.File{Modes: []usagemode.Mode{mode}}); err != nil {
		t.Fatal(err)
	}
	if args := strings.Join(codexLaunchArgs(sessionstore.LaunchDescriptor{Agent: "codex"}), " "); !strings.Contains(args, "-c model_auto_compact_token_limit=180000") {
		t.Fatalf("mode limit missing: %s", args)
	}
}
