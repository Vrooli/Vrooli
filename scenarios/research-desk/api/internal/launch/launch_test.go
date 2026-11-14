package launch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/schedule"

	"research-desk/internal/home"
	"research-desk/internal/skillscripts"
	"research-desk/internal/testutil"
	"research-desk/internal/writer"
)

// The launcher is tested at its Web Console seam with a fake Codex that
// behaves as the probe measured: an input row "› " at the cursor, "ready"
// after the stance, a goal row after /goal, and an app-server process that
// outlives the window. The service runs over the excerpts in
// internal/testutil's archive.

const (
	prepared = "local-models-w1-20261111"   // real prepared packet, topic local-model-performance
	handRun  = "programme-three-20261111"   // real hand-launched charter, three topics
	other    = "standing-teams-w1-20261110" // real prepared packet, one topic
)

// fakeCodex is one Web Console session running Codex.
type fakeCodex struct {
	mu      sync.Mutex
	created []SessionSpec
	input   []string // "paste:…", "type:…", "key:Enter"
	pending string
	screen  Screen
	goals   []home.Goal
	procs   []home.Process
	// choiceAfterStance makes Codex show a numbered choice instead of "ready".
	choiceAfterStance bool
	// swallowEnters is how many Enters Codex takes as part of a typing burst
	// (typed text only; a paste shows a placeholder and submits).
	swallowEnters int
	pasted        bool
	// resumeTo is the status /goal resume leaves.
	resumeTo string
	// keepProcs keeps processes alive after a signal.
	keepProcs bool
	signalled []int
	// shown is the thread the terminal shows; shownIn overrides it per
	// session. turnRunning is a running turn.
	shown       string
	shownIn     map[string]string
	turnRunning bool
	// goalsErr makes goals_1.sqlite unreadable.
	goalsErr error
}

func newCodex() *fakeCodex {
	return &fakeCodex{resumeTo: "active", shown: "thread-1", screen: idle("OpenAI Codex"), procs: []home.Process{
		{PID: 41, CodexHome: "x", WebConsoleSession: "sess-1", InTerminal: true},
		{PID: 42, CodexHome: "x", AppServer: true, WebConsoleSession: "sess-1", InTerminal: true},
	}}
}

var target = Target{Session: "sess-1", Thread: "thread-1"}

func idle(text string) Screen {
	lines := strings.Split(text+"\n› ", "\n")
	return Screen{Text: strings.Join(lines, "\n"), CursorRow: len(lines) - 1}
}

func (f *fakeCodex) Create(_ context.Context, spec SessionSpec) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.created = append(f.created, spec)
	f.screen = idle("OpenAI Codex (v0.130)")
	return "sess-1", nil
}

func (f *fakeCodex) Screen(context.Context, string) (Screen, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.screen, nil
}

func (f *fakeCodex) Type(_ context.Context, _, text string, paste bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	kind := "type:"
	if paste {
		kind = "paste:"
		f.screen = idle(f.screen.Text + "\n› [Pasted Content]")
	} else {
		f.screen = Screen{Text: f.screen.Text + "\n› " + text, CursorRow: strings.Count(f.screen.Text, "\n") + 1}
	}
	f.input = append(f.input, kind+text)
	f.pending, f.pasted = text, paste
	return nil
}

func (f *fakeCodex) Keys(_ context.Context, _ string, keys ...Key) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, k := range keys {
		name := k.Name
		if k.Ctrl {
			name = "ctrl+" + name
		}
		f.input = append(f.input, "key:"+name)
		if k != Enter {
			continue
		}
		if f.swallowEnters > 0 && !f.pasted {
			f.swallowEnters--
			continue
		}
		text := f.pending
		f.pending = ""
		switch {
		case text == "/goal pause":
			f.setStatus("paused")
			f.screen = idle("• goal paused")
		case text == "/goal resume":
			f.setStatus(f.resumeTo)
			f.screen = idle("• goal resumed")
		case strings.HasPrefix(text, "/goal "):
			f.goals = append(f.goals, home.Goal{SessionID: "sess-1", ThreadID: "thread-1", Objective: strings.TrimPrefix(text, "/goal "), Status: "active", Updated: time.Now()})
			f.screen = idle("• goal set")
		case f.choiceAfterStance:
			f.screen = Screen{Text: "Allow network access?\n› 1. Yes\n  2. No", CursorRow: 1}
		default:
			f.screen = idle("• ready")
		}
	}
	return nil
}

func (f *fakeCodex) setStatus(status string) {
	for i := range f.goals {
		f.goals[i].Status, f.goals[i].Updated = status, time.Now()
	}
}

func (f *fakeCodex) WaitIdle(context.Context, string, time.Duration, time.Duration) (bool, error) {
	return false, nil
}

func (f *fakeCodex) SessionGoals(context.Context, string) ([]home.Goal, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.goals), f.goalsErr
}

func (f *fakeCodex) ThreadView(sessionID, _ string) (home.ThreadView, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	shown, ok := f.shownIn[sessionID]
	if !ok {
		shown = f.shown
	}
	return home.ThreadView{Shown: shown, TurnRunning: f.turnRunning}, nil
}

func (f *fakeCodex) SessionProcesses(string) ([]home.Process, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.procs), nil
}

func (f *fakeCodex) signal(pid int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.signalled = append(f.signalled, pid)
	if !f.keepProcs {
		f.procs = slices.DeleteFunc(f.procs, func(p home.Process) bool { return p.PID == pid })
	}
	return nil
}

func (f *fakeCodex) typed() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.input)
}

func launcher(codex *fakeCodex) *Launcher {
	return &Launcher{
		Terminal: codex, Sessions: codex, WorkingDir: "/repo", Signal: codex.signal, Poll: time.Millisecond,
		Patience: Patience{Settle: time.Millisecond, Composer: 50 * time.Millisecond, Ready: 50 * time.Millisecond, Goal: 50 * time.Millisecond, Stop: 50 * time.Millisecond},
	}
}

var settings = Settings{Model: "gpt-5.5", Effort: "high"}

func TestALaunchTypesOnlyTheStanceAndTheGoalAndVerifiesTheGoalRow(t *testing.T) {
	codex := newCodex()
	var steps []string
	var thread string
	err := launcher(codex).Run(context.Background(), SessionSpec{IdempotencyKey: "c#1", Label: "research · c"}, Texts{Stance: "Stance.\n\nReply \"ready\" and wait.\n", Kernel: "Run the commission."}, settings, func(step, session, th string) error {
		steps = append(steps, step)
		if th != "" {
			thread = th
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{StepCreated, StepComposerReady, StepStanceSent, StepReadySeen, StepGoalSent, StepGoalVerified}; !slices.Equal(steps, want) {
		t.Fatalf("steps = %v, want %v", steps, want)
	}
	if thread != "thread-1" {
		t.Fatalf("thread = %q, want the goal row's thread", thread)
	}
	want := []string{"paste:Stance.\n\nReply \"ready\" and wait.", "key:Enter", "type:/goal Run the commission.", "key:Enter"}
	if got := codex.typed(); !slices.Equal(got, want) {
		t.Fatalf("input = %q, want %q", got, want)
	}

	// An Enter that Codex takes as part of the typing burst is pressed once more.
	burst := newCodex()
	burst.swallowEnters = 1
	if err := launcher(burst).Run(context.Background(), SessionSpec{}, Texts{Stance: "Stance.", Kernel: "Kernel."}, settings, func(string, string, string) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if got := burst.typed(); !slices.Equal(got, []string{"paste:Stance.", "key:Enter", "type:/goal Kernel.", "key:Enter", "key:Enter"}) {
		t.Fatalf("input = %q", got)
	}
	stuck := newCodex()
	stuck.swallowEnters = 2
	err = launcher(stuck).Run(context.Background(), SessionSpec{}, Texts{Stance: "Stance.", Kernel: "Kernel."}, settings, func(string, string, string) error { return nil })
	var stepErr *StepError
	if !errors.As(err, &stepErr) || stepErr.Step != StepGoalSent || !strings.Contains(err.Error(), "after two Enters") {
		t.Fatalf("err = %v", err)
	}
	spec := codex.created[0]
	if spec.WorkingDir != "/repo" || spec.IdempotencyKey != "c#1" || !strings.Contains(spec.Command, "vrooli-agent-launcher --agent codex -- --yolo -m gpt-5.5 -c model_reasoning_effort=high") || !strings.Contains(spec.Command, "exec codex --yolo") {
		t.Fatalf("session spec = %+v", spec)
	}
}

func TestALaunchStopsAtAChoiceAndSaysWhatToDo(t *testing.T) {
	codex := newCodex()
	codex.choiceAfterStance = true
	var steps []string
	err := launcher(codex).Run(context.Background(), SessionSpec{}, Texts{Stance: "Stance.", Kernel: "Kernel."}, settings, func(step, _, _ string) error {
		steps = append(steps, step)
		return nil
	})
	var stepErr *StepError
	if !errors.As(err, &stepErr) || stepErr.Step != StepReadySeen || !strings.Contains(err.Error(), "answer it in Web Console") {
		t.Fatalf("err = %v, want a ready_seen failure naming the choice", err)
	}
	if !slices.Equal(steps, []string{StepCreated, StepComposerReady, StepStanceSent}) {
		t.Fatalf("steps = %v", steps)
	}
	for _, in := range codex.typed() {
		if strings.Contains(in, "/goal") {
			t.Fatalf("the goal was typed after a failed step: %q", codex.typed())
		}
	}
}

// [REQ:RD-P0-001]
func TestALaunchRefusesBadTextsAndSettingsBeforeCreatingASession(t *testing.T) {
	for name, tc := range map[string]struct {
		texts    Texts
		settings Settings
	}{
		"two-line kernel": {Texts{Stance: "s", Kernel: "one\ntwo"}, settings},
		"empty stance":    {Texts{Stance: " ", Kernel: "k"}, settings},
		"shell in model":  {Texts{Stance: "s", Kernel: "k"}, Settings{Model: "gpt; rm", Effort: "high"}},
	} {
		codex := newCodex()
		err := launcher(codex).Run(context.Background(), SessionSpec{}, tc.texts, tc.settings, func(string, string, string) error { return nil })
		if err == nil || len(codex.created) != 0 {
			t.Fatalf("%s: err = %v, sessions created = %d", name, err, len(codex.created))
		}
	}
}

// [REQ:RD-P0-003]
func TestStopPausesTheGoalQuitsCodexAndEndsOnlyTheSessionsProcesses(t *testing.T) {
	codex := newCodex()
	codex.goals = []home.Goal{{ThreadID: "thread-1", Objective: "k", Status: "active"}}
	// A worker that inherited the session's Codex home but left its terminal,
	// and another session's process with the same home, are not the session's.
	codex.procs = append(codex.procs,
		home.Process{PID: 43, CodexHome: "x", WebConsoleSession: "sess-1"},
		home.Process{PID: 44, CodexHome: "x", WebConsoleSession: "sess-2", InTerminal: true})
	stopped, err := launcher(codex).Stop(context.Background(), target, true)
	if err != nil {
		t.Fatal(err)
	}
	if !stopped.Window || stopped.Ended != 2 || !strings.HasPrefix(stopped.Outcome(), "window stopped; 2 Codex process(es) of session "+target.Session+" ended; none still running") {
		t.Fatalf("the stop says what it found and left: %+v %q", stopped, stopped.Outcome())
	}
	want := []string{"type:/goal pause", "key:Enter", "key:Escape", "key:ctrl+c", "key:ctrl+c"}
	if got := codex.typed(); !slices.Equal(got, want) {
		t.Fatalf("input = %q, want %q", got, want)
	}
	if !slices.Equal(codex.signalled, []int{41, 42}) || codex.goals[0].Status != "paused" {
		t.Fatalf("signalled %v, goal %s", codex.signalled, codex.goals[0].Status)
	}

	stubborn := newCodex()
	stubborn.keepProcs = true
	_, err = launcher(stubborn).Stop(context.Background(), target, true)
	if err == nil || !strings.Contains(err.Error(), "41, 42") || !slices.Equal(stubborn.signalled, []int{41, 42}) {
		t.Fatalf("err = %v, signalled %v; want each process signalled once and named", err, stubborn.signalled)
	}

	// A closed window is not typed into; its leftover processes still end.
	closed := newCodex()
	closed.screen = Screen{Text: "$ ", CursorRow: 0}
	gone, err := launcher(closed).Stop(context.Background(), target, false)
	if err != nil || len(closed.typed()) != 0 || len(closed.signalled) != 2 || !strings.HasPrefix(gone.Outcome(), "no window found; 2 Codex process(es)") {
		t.Fatalf("err = %v, typed %q, signalled %v, outcome %q", err, closed.typed(), closed.signalled, gone.Outcome())
	}
}

// [REQ:RD-P0-012]
// A control reads and types for the commission's own thread, and types
// nothing unless the terminal shows that thread at an idle input row; only
// /goal pause and /goal resume may go in during a running turn (measured).
func TestControlsReachTheCommissionsOwnThreadOrNothing(t *testing.T) {
	codex := newCodex()
	codex.screen = idle("• ready")
	l := launcher(codex)
	ctx := context.Background()
	// A newer goal in another thread of the same Codex home is not this one's.
	codex.goals = []home.Goal{
		{ThreadID: "thread-1", Status: "paused", Updated: time.Now().Add(-time.Hour)},
		{ThreadID: "thread-9", Status: "active", Updated: time.Now()},
	}
	if status, err := l.GoalStatus(ctx, target); err != nil || status != "paused" {
		t.Fatalf("status = %q, %v; want the commission thread's own goal", status, err)
	}
	if _, err := l.GoalStatus(ctx, Target{Session: "sess-1", Thread: "thread-5"}); err == nil {
		t.Fatal("a thread without a goal row read as a status")
	}

	for name, setup := range map[string]func(){
		"another thread shown": func() { codex.shown = "thread-9" },
		"not at the input row": func() { codex.screen = Screen{Text: "Working (esc to interrupt)", CursorRow: 0} },
	} {
		codex.shown, codex.turnRunning, codex.screen = "thread-1", false, idle("• ready")
		setup()
		for _, act := range []func() error{
			func() error { _, err := l.Pause(ctx, target); return err },
			func() error { _, err := l.Resume(ctx, target); return err },
			func() error { return l.ResendStance(ctx, target, "Stance.") },
			func() error { _, err := l.Stop(ctx, target, true); return err },
		} {
			if err := act(); !errors.Is(err, ErrNotReady) || !strings.Contains(err.Error(), "OPERATOR.md") {
				t.Fatalf("%s: err = %v, want not ready with the manual path", name, err)
			}
		}
		if len(codex.typed()) != 0 || len(codex.signalled) != 0 {
			t.Fatalf("%s: typed %q, signalled %v", name, codex.typed(), codex.signalled)
		}
	}

	// During a running turn Codex applies /goal pause and /goal resume at once
	// and the turn goes on; a stance or a stop would queue or cut it.
	for _, working := range []Screen{idle("1\n2\n3"), idle("Working (esc to interrupt)\n3")} {
		codex.shown, codex.turnRunning, codex.screen = "thread-1", true, working
		codex.goals = []home.Goal{{ThreadID: "thread-1", Status: "active", Updated: time.Now()}}
		if status, err := l.Pause(ctx, target); err != nil || status != "paused" {
			t.Fatalf("pause during a turn: %q, %v", status, err)
		}
		codex.screen = working
		if status, err := l.Resume(ctx, target); err != nil || status != "active" {
			t.Fatalf("resume during a turn: %q, %v", status, err)
		}
		codex.screen = working
		if err := l.ResendStance(ctx, target, "Stance."); !errors.Is(err, ErrNotReady) {
			t.Fatalf("stance during a turn: %v", err)
		}
		if _, err := l.Stop(ctx, target, true); !errors.Is(err, ErrNotReady) {
			t.Fatalf("stop during a turn: %v", err)
		}
	}
	typed := slices.DeleteFunc(codex.typed(), func(in string) bool { return in == "key:Enter" })
	if want := []string{"type:/goal pause", "type:/goal resume", "type:/goal pause", "type:/goal resume"}; !slices.Equal(typed, want) {
		t.Fatalf("typed %q, want only the goal commands", typed)
	}
	// Another thread shown still refuses them.
	codex.shown = "thread-9"
	if _, err := l.Pause(ctx, target); !errors.Is(err, ErrNotReady) {
		t.Fatalf("pause with another thread shown: %v", err)
	}
}

// reader overrides derived state for the commissions a test names; the real
// reader derives the rest from the copied folders.
type reader struct {
	*home.Reader
	states   map[string]home.State
	sessions map[string]string
	// tweaks change any other derived field of a commission.
	tweaks map[string]func(*home.Commission)
}

func (r *reader) Snapshot(ctx context.Context, opts home.Options) (*home.Snapshot, error) {
	snap, err := r.Reader.Snapshot(ctx, opts)
	if err != nil {
		return nil, err
	}
	for _, c := range snap.Commissions {
		if s, ok := r.states[c.ID]; ok {
			c.State = s
		}
		if id, ok := r.sessions[c.ID]; ok {
			c.Session = &home.Match{Kind: home.MatchExact, SessionID: id, ThreadID: "thread-1", WindowOpen: true}
		}
		if tweak, ok := r.tweaks[c.ID]; ok {
			tweak(c)
		}
	}
	return snap, nil
}

type packets struct {
	built []map[string]any
}

func (p *packets) Preview(_ context.Context, spec []byte) (*skillscripts.PacketResult, error) {
	return &skillscripts.PacketResult{OK: true}, nil
}

func (p *packets) Build(_ context.Context, spec []byte) (*skillscripts.PacketResult, error) {
	var m map[string]any
	if err := json.Unmarshal(spec, &m); err != nil {
		return nil, err
	}
	p.built = append(p.built, m)
	return &skillscripts.PacketResult{OK: true, CommissionID: "programme-five-20261112", Written: true}, nil
}

func (p *packets) Check(_ context.Context, id string) (*skillscripts.PacketResult, error) {
	return &skillscripts.PacketResult{OK: true, CommissionID: id}, nil
}

type rig struct {
	service  *Service
	codex    *fakeCodex
	reader   *reader
	packets  *packets
	research string
	writer   *writer.Writer
}

func newRig(t *testing.T) *rig {
	t.Helper()
	h := testutil.NewHome(t)
	research := h.Research
	clock := schedule.NewFake(time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC))
	r := &reader{
		Reader:   home.NewReader(home.Config{ResearchDir: research, CodexDir: h.Codex}, home.Deps{Clock: clock, Procs: home.FixedProcs{}}),
		states:   map[string]home.State{},
		sessions: map[string]string{},
		tweaks:   map[string]func(*home.Commission){},
	}
	w := writer.New(research, h.State, testutil.EntryChars)
	t.Cleanup(func() { w.Close() })
	codex := newCodex()
	limits := &skillscripts.Limits{}
	limits.Defaults.SupervisorModel, limits.Defaults.SupervisorEffort = "gpt-5.5", "xhigh"
	p := &packets{}
	return &rig{
		service: &Service{
			Reader: r, Writer: w, Packets: p, Launcher: launcher(codex), Clock: clock, Logger: log.New(io.Discard, "", 0),
			Limits: func() (*skillscripts.Limits, error) { return limits, nil },
		},
		codex: codex, reader: r, packets: p, research: research, writer: w,
	}
}

func (r *rig) launchFile(t *testing.T, id string) *home.LaunchRecord {
	t.Helper()
	record, err := home.ReadLaunch(filepath.Join(r.research, "cohorts", id, home.LaunchFile))
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func (r *rig) wait(t *testing.T, id string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for r.service.isRunning(id) {
		if time.Now().After(deadline) {
			t.Fatal("the launch did not finish")
		}
		time.Sleep(time.Millisecond)
	}
}

// [REQ:RD-P0-001]
func TestAPreparedPacketLaunchesAndEveryStepIsRecorded(t *testing.T) {
	r := newRig(t)
	first, err := r.service.Launch(context.Background(), prepared, Choices{})
	if err != nil {
		t.Fatal(err)
	}
	if first.LaunchedBy != "desk" || first.IdempotencyKey != prepared+"#1" {
		t.Fatalf("first record = %+v", first)
	}
	r.wait(t, prepared)
	record := r.launchFile(t, prepared)
	var steps []string
	for _, s := range record.Steps {
		steps = append(steps, s.Step)
	}
	if !slices.Equal(steps, stepOrder) || record.Failed != nil || record.WebConsoleSessionID != "sess-1" || record.ThreadID != "thread-1" {
		t.Fatalf("LAUNCH.json = %+v", record)
	}
	kernel, _ := os.ReadFile(filepath.Join(r.research, "cohorts", prepared, home.KernelFile))
	if got := r.codex.typed(); got[2] != "type:/goal "+strings.TrimSpace(string(kernel)) {
		t.Fatalf("goal typed = %q, want KERNEL.txt", got[2])
	}
	if !strings.Contains(r.codex.created[0].Command, "-m gpt-5.5 -c model_reasoning_effort=xhigh") {
		t.Fatalf("command %q does not use the limits.json defaults", r.codex.created[0].Command)
	}

	r.reader.states[handRun] = home.StateStopped
	if _, err := r.service.Launch(context.Background(), handRun, Choices{AssertNeverStarted: true}); !errors.Is(err, home.ErrRefused) || !strings.Contains(err.Error(), handRun+" is stopped, not prepared; relaunch it as a new commission") {
		t.Fatalf("relaunching a commission that already ran: err = %v, want refused", err)
	}
}

// [REQ:RD-P0-002]
// A launch is blocked only by a commission that may be running; each
// blocker carries its choices and the desk's recommendation, and the
// request's choice clears it.
func TestBlockersAreFewAndResolvedInTheRequest(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	now := r.service.Clock.Now()
	spec := []byte(`{"topics":["niche-book-production"],"allowance_hours":8}`)
	verdict := func(p Preflight) Verdict {
		t.Helper()
		for _, v := range p.Verdicts {
			if v.CommissionID == handRun {
				return v
			}
		}
		t.Fatalf("no verdict for %s: %+v", handRun, p.Verdicts)
		return Verdict{}
	}

	// Live: blocks, both choices, nothing pre-selected.
	r.reader.states[handRun] = home.StateLive
	p, err := r.service.PreflightSpec(ctx, spec, Choices{})
	if err != nil {
		t.Fatal(err)
	}
	v := verdict(p)
	if p.Launchable() || !v.Blocks || !slices.Equal(v.Choices, []Resolution{Supersede, RunAlongside}) || v.Recommended != "" || !strings.Contains(v.RecommendedReason, "live") {
		t.Fatalf("a live commission blocks with both choices and no recommendation: %+v %q", v, p.Blockers)
	}
	for _, ch := range []Choices{{Resolutions: map[string]Resolution{handRun: RunAlongside}}, {Resolutions: map[string]Resolution{handRun: Supersede}}} {
		if p, _ = r.service.PreflightSpec(ctx, spec, ch); !p.Launchable() || verdict(p).Chosen != ch.Resolutions[handRun] {
			t.Fatalf("the request's choice clears the blocker: %+v %q", verdict(p), p.Blockers)
		}
	}
	alongside := []byte(`{"topics":["niche-book-production"],"run_alongside":["` + handRun + `"]}`)
	if p, _ = r.service.PreflightSpec(ctx, alongside, Choices{}); !p.Launchable() || verdict(p).Chosen != RunAlongside {
		t.Fatalf("the spec's run_alongside is a choice: %q", p.Blockers)
	}

	// Paused with its allowance ended: supersede is recommended.
	r.reader.states[handRun] = home.StatePaused
	r.reader.tweaks[handRun] = func(c *home.Commission) {
		c.AllowanceEnd = home.Time{Value: now.Add(-time.Hour), Evidence: home.Evidence{Known: true}}
	}
	if v = verdict(must(r.service.PreflightSpec(ctx, spec, Choices{}))); !v.Blocks || v.Recommended != Supersede {
		t.Fatalf("paused with its allowance ended: %+v", v)
	}
	r.reader.tweaks[handRun] = func(c *home.Commission) {
		c.AllowanceEnd = home.Time{Value: now.Add(time.Hour), Evidence: home.Evidence{Known: true}}
	}
	if v = verdict(must(r.service.PreflightSpec(ctx, spec, Choices{}))); !v.Blocks || v.Recommended != "" {
		t.Fatalf("paused with allowance left: %+v", v)
	}

	// Unknown: blocks only with an open session or a write in 24 hours.
	r.reader.states[handRun] = home.StateUnknown
	for name, tc := range map[string]struct {
		tweak  func(*home.Commission)
		blocks bool
	}{
		"quiet": {func(c *home.Commission) {
			c.Session, c.LastWrite = nil, home.Time{Value: now.Add(-25 * time.Hour), Evidence: home.Evidence{Known: true}}
		}, false},
		"recent write": {func(c *home.Commission) {
			c.Session, c.LastWrite = nil, home.Time{Value: now.Add(-time.Hour), Evidence: home.Evidence{Known: true}}
		}, true},
		"unknown write": {func(c *home.Commission) { c.Session, c.LastWrite = nil, home.Time{Evidence: home.Unknown("cut")} }, true},
		"open session": {func(c *home.Commission) {
			c.Session = &home.Match{SessionID: "sess-1", WindowOpen: true}
			c.LastWrite = home.Time{Value: now.Add(-25 * time.Hour), Evidence: home.Evidence{Known: true}}
		}, true},
	} {
		r.reader.tweaks[handRun] = tc.tweak
		v = verdict(must(r.service.PreflightSpec(ctx, spec, Choices{})))
		if v.Blocks != tc.blocks || v.Quiet == tc.blocks {
			t.Errorf("%s unknown commission: blocks %v quiet %v, want blocks %v: %s", name, v.Blocks, v.Quiet, tc.blocks, v.Message)
		}
	}

	// Unknown with a recent write but no session or goal found anywhere:
	// nothing to stop, so run alongside is recommended and supersede stays
	// offered. With a session found, the operator chooses.
	r.reader.tweaks[handRun] = func(c *home.Commission) {
		c.Session, c.LastWrite = nil, home.Time{Value: now.Add(-time.Hour), Evidence: home.Evidence{Known: true}}
		c.NeverStarted = home.NeverStarted{Assertable: true, Reason: "no session or goal names this charter"}
	}
	if v = verdict(must(r.service.PreflightSpec(ctx, spec, Choices{}))); !v.Blocks || v.Recommended != RunAlongside || !slices.Contains(v.Choices, Supersede) || !strings.Contains(v.RecommendedReason, "nothing to stop") {
		t.Fatalf("an unknown commission with nothing found running: %+v", v)
	}
	r.reader.tweaks[handRun] = func(c *home.Commission) {
		c.Session = &home.Match{SessionID: "sess-1"}
		c.NeverStarted = home.NeverStarted{Reason: "session sess-1 names this charter"}
	}
	if v = verdict(must(r.service.PreflightSpec(ctx, spec, Choices{}))); !v.Blocks || v.Recommended != "" {
		t.Fatalf("an unknown commission with a session found: %+v", v)
	}

	// A stopped one never blocks and offers its checkpoint.
	delete(r.reader.tweaks, handRun)
	r.reader.states[handRun] = home.StateStopped
	p, _ = r.service.PreflightSpec(ctx, []byte(`{"topics":["ai-video-production-and-funnels"]}`), Choices{})
	if !p.Launchable() || len(p.Verdicts) != 1 || !strings.HasSuffix(p.Verdicts[0].ReadFirst, "ai-video-production-and-funnels/runs/"+handRun+"/CHECKPOINT.md") {
		t.Fatalf("a stopped commission should offer its checkpoint as read_first: %+v", p.Verdicts)
	}
}

// commandText is command-line text in a sentence the operator reads.
var commandText = regexp.MustCompile("`|research-desk [a-z]|vrooli [a-z]|packet\\.py|(^|\\s)--[a-z]")

// Blocker lines and verdict messages say what is wrong and what the
// operator can choose, in every state, and never name a command.
func TestBlockerAndVerdictTextNamesNoCommand(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	spec := []byte(`{"topics":["niche-book-production"],"allowance_hours":8}`)
	seen := 0
	for _, state := range []home.State{home.StateLive, home.StatePaused, home.StateLimited, home.StateUnknown, home.StateStopped, home.StateEnded, home.StatePrepared} {
		r.reader.states[handRun] = state
		for _, p := range []Preflight{must(r.service.PreflightSpec(ctx, spec, Choices{})), must(r.service.PreflightCommission(ctx, handRun, Choices{}))} {
			lines := slices.Clone(p.Blockers)
			for _, v := range p.Verdicts {
				lines = append(lines, v.Message, v.RecommendedReason)
			}
			for _, line := range lines {
				seen++
				if commandText.MatchString(line) {
					t.Errorf("%s: %q names a command", state, line)
				}
			}
		}
	}
	if seen < 10 {
		t.Fatalf("only %d lines were checked", seen)
	}
}

func must(p Preflight, err error) Preflight {
	if err != nil {
		panic(err)
	}
	return p
}

// sharing makes handRun a live commission on the prepared packet's topic,
// in session sess-1.
func (r *rig) sharing() {
	r.reader.states[handRun] = home.StateLive
	r.reader.sessions[handRun] = "sess-1"
	r.reader.tweaks[handRun] = func(c *home.Commission) {
		c.Topics = append(c.Topics, home.TopicRef{Topic: "local-model-performance"})
	}
	r.codex.goals = []home.Goal{{SessionID: "sess-1", ThreadID: "thread-1", Objective: "old", Status: "active"}}
}

// Superseding stops the old supervisor through its own thread
// first, then launches; it records both sides.
func TestSupersedingStopsTheOldSupervisorFirstAndRecordsBothSides(t *testing.T) {
	r := newRig(t)
	r.sharing()
	ctx := context.Background()
	if _, err := r.service.Launch(ctx, prepared, Choices{}); !errors.Is(err, home.ErrRefused) || !strings.Contains(err.Error(), "choose supersede or run alongside") {
		t.Fatalf("an unresolved blocker: %v", err)
	}
	first, err := r.service.Launch(ctx, prepared, Choices{Resolutions: map[string]Resolution{handRun: Supersede}})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(first.Supersedes, []string{handRun}) {
		t.Fatalf("the new record names what it supersedes: %+v", first)
	}
	r.wait(t, prepared)
	typed := r.codex.typed()
	stop := []string{"type:/goal pause", "key:Enter", "key:Escape", "key:ctrl+c", "key:ctrl+c"}
	if len(typed) < len(stop)+1 || !slices.Equal(typed[:len(stop)], stop) || !strings.HasPrefix(typed[len(stop)], "paste:") {
		t.Fatalf("the old supervisor is stopped before the stance is sent: %q", typed)
	}
	old := r.launchFile(t, handRun)
	if old.SupersededBy != prepared || old.Discarded == nil || !strings.Contains(old.Discarded.Reason, "superseded by "+prepared) || old.Controls[len(old.Controls)-1].Action != ActSupersede || old.WebConsoleSessionID != "sess-1" {
		t.Fatalf("the old LAUNCH.json records the supersession: %+v", old)
	}
	if got := old.Controls[len(old.Controls)-1].Outcome; !strings.HasPrefix(got, "window stopped;") || !strings.HasSuffix(got, "none still running") {
		t.Fatalf("the old LAUNCH.json says whether a window was found and that nothing still runs: %q", got)
	}
	if rec := r.launchFile(t, prepared); rec.Failed != nil || !slices.Equal(rec.Supersedes, []string{handRun}) || len(rec.Steps) != len(stepOrder) {
		t.Fatalf("the new LAUNCH.json: %+v", rec)
	}
}

func TestDelegatedLaunchChecksEverySupersededTargetBeforeEffects(t *testing.T) {
	r := newRig(t)
	r.sharing()
	p := identity.Principal{Kind: identity.ActorAgent, Subject: "research-requester", Verified: true, Scopes: []string{"research-desk:write"}, Grant: &identity.OperationGrant{
		ID: "approved", Approver: "operator", ExpiresAt: time.Now().Add(time.Hour),
		Operations: []string{"research.launch", "research.control"}, Targets: []string{"research-desk:commission/" + prepared},
	}}
	ctx := identity.WithPrincipal(context.Background(), p)
	ch := Choices{Resolutions: map[string]Resolution{handRun: Supersede}}
	_, err := r.service.Launch(ctx, prepared, ch)
	if !errors.Is(err, authn.ErrOperation) || !strings.Contains(err.Error(), "research-desk:commission/"+handRun) {
		t.Fatalf("unapproved supersession must request its own target: %v", err)
	}
	if len(r.codex.typed()) != 0 {
		t.Fatal("typed before all target grants passed")
	}
	if _, err := os.Stat(filepath.Join(r.research, "cohorts", prepared, "LAUNCH.json")); !os.IsNotExist(err) {
		t.Fatalf("launch record written before approval: %v", err)
	}
	p.Grant.Targets = append(p.Grant.Targets, "research-desk:commission/"+handRun)
	ctx = identity.WithPrincipal(context.Background(), p)
	if _, err := r.service.Launch(ctx, prepared, ch); err != nil {
		t.Fatal(err)
	}
	r.wait(t, prepared)
	if r.launchFile(t, prepared).Failed != nil {
		t.Fatalf("approved launch failed: %+v", r.launchFile(t, prepared).Failed)
	}
}

// [REQ:RD-P0-002]
// Superseding with two paused supervisors (programme-three and programme-four on the
// real home): both are stopped through their own sessions before anything
// launches and both record superseded_by; if either cannot be stopped the
// whole launch is refused, naming it, and nothing is typed.
func TestSupersedingTwoStopsBothOrNeither(t *testing.T) {
	r := newRig(t)
	r.sharing()
	r.reader.states[handRun] = home.StatePaused
	r.reader.states[other] = home.StatePaused
	r.reader.sessions[other] = "sess-2"
	r.reader.tweaks[other] = func(c *home.Commission) {
		c.Topics = append(c.Topics, home.TopicRef{Topic: "local-model-performance"})
		c.Session.ThreadID = "thread-2"
	}
	r.codex.goals = []home.Goal{
		{SessionID: "sess-1", ThreadID: "thread-1", Objective: "old", Status: "paused"},
		{SessionID: "sess-2", ThreadID: "thread-2", Objective: "older", Status: "paused"},
	}
	r.codex.procs = append(r.codex.procs,
		home.Process{PID: 51, CodexHome: "y", WebConsoleSession: "sess-2", InTerminal: true},
		home.Process{PID: 52, CodexHome: "y", AppServer: true, WebConsoleSession: "sess-2", InTerminal: true})
	both := Choices{Resolutions: map[string]Resolution{handRun: Supersede, other: Supersede}}
	ctx := context.Background()

	r.codex.shownIn = map[string]string{"sess-2": "thread-9"}
	_, err := r.service.Launch(ctx, prepared, both)
	if !errors.Is(err, home.ErrRefused) || !strings.Contains(err.Error(), "cannot stop "+other) || !strings.Contains(err.Error(), "nothing was launched") {
		t.Fatalf("one supervisor cannot be stopped: err = %v", err)
	}
	if len(r.codex.typed()) != 0 || len(r.codex.created) != 0 || len(r.codex.signalled) != 0 {
		t.Fatalf("typed %q, created %d, signalled %v", r.codex.typed(), len(r.codex.created), r.codex.signalled)
	}

	r.codex.shownIn["sess-2"] = "thread-2"
	rec, err := r.service.Launch(ctx, prepared, both)
	if err != nil {
		t.Fatal(err)
	}
	r.wait(t, prepared)
	if got := r.launchFile(t, prepared); got.Failed != nil || len(got.Supersedes) != 2 || !slices.Contains(got.Supersedes, handRun) || !slices.Contains(got.Supersedes, other) || len(got.Steps) != len(stepOrder) {
		t.Fatalf("the new LAUNCH.json: %+v (first record %+v)", got, rec)
	}
	for _, id := range []string{handRun, other} {
		if old := r.launchFile(t, id); old.SupersededBy != prepared || old.Discarded == nil || !strings.HasSuffix(old.Controls[len(old.Controls)-1].Outcome, "none still running") {
			t.Fatalf("%s LAUNCH.json: %+v", id, old)
		}
	}
	for _, pid := range []int{41, 51} {
		if !slices.Contains(r.codex.signalled, pid) {
			t.Fatalf("supervisor %d was not stopped: signalled %v", pid, r.codex.signalled)
		}
	}
}

func TestASupersessionThatCannotStopRefusesBeforeAnythingIsLaunched(t *testing.T) {
	r := newRig(t)
	r.sharing()
	r.codex.shown = "thread-9" // the terminal shows another thread
	_, err := r.service.Launch(context.Background(), prepared, Choices{Resolutions: map[string]Resolution{handRun: Supersede}})
	if !errors.Is(err, home.ErrRefused) || !strings.Contains(err.Error(), "cannot stop "+handRun) || !strings.Contains(err.Error(), "nothing was launched") {
		t.Fatalf("err = %v", err)
	}
	if len(r.codex.typed()) != 0 || len(r.codex.created) != 0 || len(r.codex.signalled) != 0 {
		t.Fatalf("typed %q, created %d, signalled %v", r.codex.typed(), len(r.codex.created), r.codex.signalled)
	}
	for _, id := range []string{prepared, handRun} {
		if _, err := os.Stat(filepath.Join(r.research, "cohorts", id, home.LaunchFile)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("%s LAUNCH.json written: %v", id, err)
		}
	}

	// Run alongside leaves the old one alone.
	r.codex.shown = "thread-1"
	if _, err := r.service.Launch(context.Background(), prepared, Choices{Resolutions: map[string]Resolution{handRun: RunAlongside}}); err != nil {
		t.Fatal(err)
	}
	r.wait(t, prepared)
	if rec := r.launchFile(t, prepared); !slices.Equal(rec.RunAlongside, []string{handRun}) || len(rec.Supersedes) != 0 {
		t.Fatalf("run alongside is recorded: %+v", rec)
	}
	if len(r.codex.signalled) != 0 {
		t.Fatalf("running alongside stopped %v", r.codex.signalled)
	}
}

// An unknown commission is launched on the operator's assertion only
// when the desk found no session or goal for it; the assertion is recorded.
func TestAnUnknownCommissionLaunchesOnTheOperatorsAssertion(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	r.reader.states[prepared] = home.StateUnknown
	r.reader.tweaks[prepared] = func(c *home.Commission) {
		c.NeverStarted = home.NeverStarted{Reason: "a goal in x/goals_1.sqlite (thread t2, active) names this charter"}
	}
	if _, err := r.service.Launch(ctx, prepared, Choices{AssertNeverStarted: true}); !errors.Is(err, home.ErrRefused) || !strings.Contains(err.Error(), "names this charter") {
		t.Fatalf("a goal found refuses the assertion: %v", err)
	}
	const why = "no session or goal names this charter in /codex or 2 other Codex homes"
	r.reader.tweaks[prepared] = func(c *home.Commission) { c.NeverStarted = home.NeverStarted{Assertable: true, Reason: why} }
	p, err := r.service.PreflightCommission(ctx, prepared, Choices{})
	if err != nil || p.Launchable() || !p.NeverStarted.Assertable || !strings.Contains(p.Blockers[0], "launch it only if you know it was never started") {
		t.Fatalf("without the assertion: %+v %v", p, err)
	}
	if _, err := r.service.Launch(ctx, prepared, Choices{}); !errors.Is(err, home.ErrRefused) {
		t.Fatalf("launch without the assertion: %v", err)
	}
	if _, err := r.service.Launch(ctx, prepared, Choices{AssertNeverStarted: true}); err != nil {
		t.Fatal(err)
	}
	r.wait(t, prepared)
	rec := r.launchFile(t, prepared)
	if rec.NeverStarted == nil || rec.NeverStarted.Evidence != why || !rec.NeverStarted.At.Equal(r.service.Clock.Now().UTC()) || rec.Failed != nil {
		t.Fatalf("the assertion is recorded with what the desk found: %+v", rec)
	}
}

// Suggested stance quotes are the operator's words or nothing: a desk-built
// PACKET.json first, else the stance of a commission that started, never a
// hand-made folder that never started; workstreams carry their titles.
func TestDefaultsForALaunchNeedNoTyping(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	topic := []string{"standing-teams-work-routes"}
	s, err := r.service.Suggest(ctx, topic)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Quotes) != 0 || !strings.Contains(s.QuotesReason, "no commission on these topics has started") || !strings.Contains(s.QuotesReason, "type the stance quotes") {
		t.Fatalf("a hand-made packet that never started is not a source: %+v (%s)", s.Quotes, s.QuotesReason)
	}
	r.reader.tweaks[other] = func(c *home.Commission) {
		c.Started = home.Time{Value: time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC), Evidence: home.Evidence{Known: true, Source: "session"}}
	}
	s, _ = r.service.Suggest(ctx, topic)
	if len(s.Quotes) != 1 || !strings.HasPrefix(s.Quotes[0].Text, "at least for now things are still meant to flow through swarm manager") || s.Quotes[0].CommissionID != other || !strings.HasSuffix(s.Quotes[0].Source, other+"/"+home.StanceFile) || strings.HasPrefix(s.Quotes[0].Text, `"`) || strings.HasPrefix(s.Quotes[0].Text, "“") {
		t.Fatalf("once started, the quoted lines of its STANCE-MESSAGE.txt, unquoted: %+v (%s)", s.Quotes, s.QuotesReason)
	}
	delete(r.reader.tweaks, other)
	packet := `{"schema":"research-desk.packet/v1","commission_id":"` + other + `","built_at":"2026-11-10T09:00:00Z","spec":{"topics":["standing-teams-work-routes"],"stance_quotes":["\"Be bold.\"","Land it … all of it."]}}`
	if err := os.WriteFile(filepath.Join(r.research, "cohorts", other, home.PacketFile), []byte(packet), 0o644); err != nil {
		t.Fatal(err)
	}
	s, _ = r.service.Suggest(ctx, topic)
	if len(s.Quotes) != 2 || s.Quotes[0].Text != "Be bold." || s.Quotes[1].Text != "Land it … all of it." || !strings.HasSuffix(s.Quotes[0].Source, home.PacketFile) {
		t.Fatalf("a desk-built PACKET.json's stance_quotes come first, ellipses as written: %+v (%s)", s.Quotes, s.QuotesReason)
	}
	if len(s.Topics) != 1 || s.Topics[0].Topic != "standing-teams-work-routes" {
		t.Fatalf("one workstream list per topic: %+v", s.Topics)
	}
	s, _ = r.service.Suggest(ctx, []string{"no-such-topic"})
	if len(s.Quotes) != 0 || !strings.Contains(s.QuotesReason, "type the stance quotes") || s.Topics[0].Evidence.Known {
		t.Fatalf("no commission on the topics: %+v", s)
	}
}

// [REQ:RD-P0-010]
func TestAnUnfinishedDeskLaunchIsMarkedInterruptedAtTheNextStep(t *testing.T) {
	r := newRig(t)
	at := time.Date(2026, 11, 12, 8, 0, 0, 0, time.UTC)
	cut := &home.LaunchRecord{Schema: home.LaunchSchema, CommissionID: prepared, LaunchedBy: "desk", Steps: []home.LaunchStep{{Step: StepCreated, At: at}, {Step: StepComposerReady, At: at}}}
	hand := &home.LaunchRecord{Schema: home.LaunchSchema, CommissionID: other, LaunchedBy: "hand"}
	for id, rec := range map[string]*home.LaunchRecord{prepared: cut, other: hand} {
		if _, err := r.writer.WriteLaunch(id, rec); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.service.MarkInterrupted(context.Background()); err != nil {
		t.Fatal(err)
	}
	if f := r.launchFile(t, prepared).Failed; f == nil || *f != (home.LaunchFailure{Step: StepStanceSent, Reason: "interrupted"}) {
		t.Fatalf("failed = %+v", f)
	}
	if r.launchFile(t, other).Failed != nil {
		t.Fatal("a hand launch was marked interrupted")
	}
}

// [REQ:RD-P0-012]
func TestControlsActOnlyOnAnOpenWindowAndAreRecorded(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	if _, err := r.service.Control(ctx, prepared, ActPause, ""); !errors.Is(err, home.ErrRefused) {
		t.Fatalf("pause of a prepared commission: err = %v, want refused", err)
	}
	if _, err := r.service.Control(ctx, handRun, ActDiscard, " "); !errors.Is(err, home.ErrRefused) {
		t.Fatalf("discard without a reason: err = %v, want refused", err)
	}
	if len(r.codex.typed()) != 0 {
		t.Fatalf("a refused control typed %q", r.codex.typed())
	}

	r.reader.states[handRun] = home.StateLive
	r.reader.sessions[handRun] = "sess-1"
	r.codex.goals = []home.Goal{{ThreadID: "thread-1", Objective: "k", Status: "active"}}
	// Codex restarts a completed goal on /goal resume, so resume needs a paused one.
	if _, err := r.service.Control(ctx, handRun, ActResume, ""); !errors.Is(err, home.ErrRefused) || len(r.codex.typed()) != 0 {
		t.Fatalf("resume of an active goal: err = %v, typed %q", err, r.codex.typed())
	}
	out, err := r.service.Control(ctx, handRun, ActPause, "")
	if err != nil {
		t.Fatal(err)
	}
	if out.GoalStatus != "paused" || !slices.Equal(r.codex.typed(), []string{"type:/goal pause", "key:Enter"}) {
		t.Fatalf("goal %q after typing %q", out.GoalStatus, r.codex.typed())
	}
	record := r.launchFile(t, handRun)
	if record.LaunchedBy != "hand" || record.WebConsoleSessionID != "sess-1" || len(record.Controls) != 1 || record.Controls[0].Action != ActPause {
		t.Fatalf("LAUNCH.json = %+v", record)
	}

	r.reader.states[handRun] = home.StateStopped
	before := len(r.codex.typed())
	if _, err := r.service.Control(ctx, handRun, ActDiscard, "Bad start: wrong topics."); err != nil {
		t.Fatal(err)
	}
	if len(r.codex.typed()) != before {
		t.Fatal("discarding a stopped commission typed into its session")
	}
	if d := r.launchFile(t, handRun).Discarded; d == nil || d.Reason != "Bad start: wrong topics." {
		t.Fatalf("discarded = %+v", d)
	}
	if _, err := r.service.Control(ctx, handRun, ActStop, ""); !errors.Is(err, home.ErrRefused) {
		t.Fatalf("control after discard: err = %v, want refused", err)
	}
}

// [REQ:RD-P0-003]
func TestARelaunchIsANewCommissionFromTheOldSpec(t *testing.T) {
	r := newRig(t)
	if _, err := r.service.Relaunch(context.Background(), handRun, nil); !errors.Is(err, home.ErrRefused) || !strings.Contains(err.Error(), "no PACKET.json") {
		t.Fatalf("relaunch without PACKET.json: err = %v", err)
	}
	packet := `{"schema":"research-desk.packet/v1","commission_id":"` + other + `","spec":{"topics":["standing-teams-work-routes"],"allowance_hours":6,"supersedes":null}}`
	if err := os.WriteFile(filepath.Join(r.research, "cohorts", other, home.PacketFile), []byte(packet), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := r.service.Relaunch(context.Background(), other, nil)
	if err != nil {
		t.Fatal(err)
	}
	spec := r.packets.built[0]
	if fmt.Sprint(spec["supersedes"]) != "["+other+"]" || spec["carry_baseline_from"] != nil || spec["allowance_hours"] != 6.0 || p.Packet.CommissionID != "programme-five-20261112" {
		t.Fatalf("built spec = %v", spec)
	}
}

// [REQ:RD-P0-003]
// A bad start that never wrote BASELINE.json is relaunched from the same
// packet through the real packet.py; the new commission records its own
// baseline. One that has a baseline carries it.
func TestABadStartRelaunchesThroughPacketPyWithOrWithoutABaseline(t *testing.T) {
	r := newRig(t)
	r.service.Packets = skillscripts.Packets{Paths: skillscripts.Paths{Skills: "../../../skills"}, Runner: skillscripts.NewPython(), ResearchDir: r.research}
	ctx := context.Background()
	first, err := r.service.Build(ctx, []byte(`{"topics":["niche-book-production"],"allowance_hours":6,"slug":"bad-start","stance_quotes":["Make research easier to manage."]}`), nil)
	if err != nil || first.Packet == nil || !first.Packet.Written {
		t.Fatalf("build: %+v, %v", first.Packet, err)
	}
	bad := first.Packet.CommissionID
	p, err := r.service.Relaunch(ctx, bad, nil)
	if err != nil || p.Packet == nil || !p.Packet.Written {
		t.Fatalf("relaunch without a baseline: %+v, %v", p.Packet, err)
	}
	next := p.Packet.CommissionID
	record, _, err := home.ReadPacket(filepath.Join(r.research, "cohorts", next, home.PacketFile))
	var spec map[string]any
	if err == nil {
		err = json.Unmarshal(record.Spec, &spec)
	}
	if err != nil || fmt.Sprint(spec["supersedes"]) != "["+bad+"]" || spec["carry_baseline_from"] != nil {
		t.Fatalf("relaunched spec = %v, %v", spec, err)
	}
	if _, err := os.Stat(filepath.Join(r.research, "cohorts", next, home.BaselineFile)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a baseline appeared from nowhere: %v", err)
	}

	if err := os.WriteFile(filepath.Join(r.research, "cohorts", next, home.BaselineFile), []byte(`{"measured":"before"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err = r.service.Relaunch(ctx, next, nil)
	if err != nil || p.Packet == nil || !p.Packet.Written {
		t.Fatalf("relaunch with a baseline: %+v, %v", p.Packet, err)
	}
	carried, err := os.ReadFile(filepath.Join(r.research, "cohorts", p.Packet.CommissionID, home.BaselineFile))
	if err != nil || string(carried) != `{"measured":"before"}` {
		t.Fatalf("baseline not carried: %q, %v", carried, err)
	}
}

// [REQ:RD-P0-002]
// The operator's choices at Build go into the spec, so the charter that
// packet.py renders names every commission it supersedes and each one it runs
// alongside; the request wins over the spec.
func TestBuildWritesTheChoicesIntoTheCharter(t *testing.T) {
	r := newRig(t)
	r.service.Packets = skillscripts.Packets{Paths: skillscripts.Paths{Skills: "../../../skills"}, Runner: skillscripts.NewPython(), ResearchDir: r.research}
	ctx := context.Background()
	spec := []byte(`{"topics":["niche-book-production"],"allowance_hours":6,"slug":"choices","stance_quotes":["Make research easier to manage."],"run_alongside":["` + other + `"]}`)
	charter := func(choices map[string]Resolution) string {
		t.Helper()
		p, err := r.service.Build(ctx, spec, choices)
		if err != nil || p.Packet == nil || !p.Packet.Written {
			t.Fatalf("build %v: %+v, %v", choices, p.Packet, err)
		}
		text, err := os.ReadFile(filepath.Join(r.research, "cohorts", p.Packet.CommissionID, home.CharterFile))
		if err != nil {
			t.Fatal(err)
		}
		return string(text)
	}
	if text := charter(map[string]Resolution{other: Supersede, handRun: RunAlongside}); !strings.Contains(text, "Supersedes `"+other+"`.") || !strings.Contains(text, "Runs alongside `"+handRun+"` by the operator's choice") {
		t.Fatalf("the charter does not name the choices (the request wins over the spec's run_alongside):\n%s", text)
	}
	text := charter(map[string]Resolution{other: Supersede, handRun: Supersede})
	if !strings.Contains(text, "Supersedes `"+other+"` and `"+handRun+"`.") && !strings.Contains(text, "Supersedes `"+handRun+"` and `"+other+"`.") {
		t.Fatalf("two supersedes are not both named in the charter:\n%s", text)
	}
	if strings.Contains(text, "Runs alongside") {
		t.Fatalf("a superseded commission is still run alongside:\n%s", text)
	}
}

func TestTheLaunchTestReportsEachStatusAndStopsItsSession(t *testing.T) {
	r := newRig(t)
	// The fake Codex completes the goal when it is resumed, as the kernel asks.
	r.codex.resumeTo = "complete"
	started, err := r.service.TestLaunch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !started.Running || started.ReportPath == "" {
		t.Fatalf("started = %+v", started)
	}
	if _, err := r.service.TestLaunch(context.Background()); !errors.Is(err, home.ErrRefused) {
		t.Fatalf("a second test while one runs: err = %v, want refused", err)
	}
	r.wait(t, testClaim)
	report, err := r.service.LatestTest()
	if err != nil {
		t.Fatal(err)
	}
	if report.FailedReason != "" || report.StatusAfterGoal != "active" || report.StatusAfterPause != "paused" || report.StatusAfterResume != "complete" || report.StatusFinal != "complete" || !report.Stopped {
		t.Fatalf("report = %+v", report)
	}
	if report.KernelChars < 550 || report.KernelChars > 600 || !strings.HasSuffix(r.codex.created[0].Label, "(stopped; keep)") {
		t.Fatalf("kernel %d chars, label %q", report.KernelChars, r.codex.created[0].Label)
	}
	if report.ReportPath != started.ReportPath || report.Running || len(report.Steps) != len(stepOrder) {
		t.Fatalf("latest = %+v, started at %s", report, started.ReportPath)
	}
	entries, _ := os.ReadDir(filepath.Join(r.research, "cohorts"))
	for _, e := range entries {
		if _, err := os.Stat(filepath.Join(r.research, "cohorts", e.Name(), home.LaunchFile)); err == nil {
			t.Fatalf("the launch test wrote LAUNCH.json in %s", e.Name())
		}
	}
}

// A goal database that cannot be read leaves the goal's status unknown;
// pause and resume, which need it, are refused with the reason and type
// nothing.
func TestAnUnreadableGoalStatusIsUnknownAndRefusesPauseAndResume(t *testing.T) {
	r := newRig(t)
	r.reader.states[handRun] = home.StateLive
	r.reader.sessions[handRun] = "sess-1"
	r.codex.goalsErr = errors.New("database is locked")
	for _, action := range []string{ActPause, ActResume} {
		_, err := r.service.Control(context.Background(), handRun, action, "")
		if !errors.Is(err, home.ErrRefused) || !strings.Contains(err.Error(), "unknown") || !strings.Contains(err.Error(), "database is locked") {
			t.Errorf("%s: err = %v, want refused with the unknown status's reason", action, err)
		}
	}
	if len(r.codex.typed()) != 0 {
		t.Fatalf("typed %q", r.codex.typed())
	}
	if got := known(r.service.Launcher.GoalStatus(context.Background(), target)); !strings.HasPrefix(got, "unknown: ") {
		t.Errorf("the launch test reports %q, not a blank status", got)
	}
}

// Launching an existing packet shows the start of its own STATE.md verbatim,
// capped at 12 lines or 800 characters, with its path; it never blocks.
func TestPreflightShowsTheStartOfThePacketsOwnState(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	r.reader.states[prepared] = home.StatePrepared // rewriting STATE.md would make it unknown
	path := filepath.Join(r.research, "cohorts", prepared, home.StateFile)
	real, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	excerpt := func() *StateExcerpt {
		t.Helper()
		p, err := r.service.PreflightCommission(ctx, prepared, Choices{})
		if err != nil || p.PacketState == nil || p.PacketState.Path != path {
			t.Fatalf("an existing packet carries its STATE.md excerpt: %+v %v", p, err)
		}
		if !p.Launchable() {
			t.Fatalf("the excerpt never blocks: %v", p.Blockers)
		}
		return p.PacketState
	}
	if e := excerpt(); e.Text == "" || !strings.HasPrefix(string(real), e.Text) || strings.Count(e.Text, "\n") > ExcerptLines || len([]rune(e.Text)) > ExcerptChars || e.Truncated != (len(e.Text) < len(real)) {
		t.Fatalf("the real STATE.md's start, verbatim: %+v", e)
	}
	lines := strings.Repeat("Do not start the research yet.\n", 30)
	if err := os.WriteFile(path, []byte(lines), 0o644); err != nil {
		t.Fatal(err)
	}
	if e := excerpt(); e.Text != strings.Repeat("Do not start the research yet.\n", ExcerptLines) || !e.Truncated {
		t.Fatalf("first 12 lines: %+v", e)
	}
	if err := os.WriteFile(path, []byte(strings.Repeat("é", 1000)), 0o644); err != nil {
		t.Fatal(err)
	}
	if e := excerpt(); e.Text != strings.Repeat("é", ExcerptChars) || !e.Truncated {
		t.Fatalf("first 800 characters, not bytes: %d runes", len([]rune(e.Text)))
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if e := excerpt(); e.Text != "" || e.Reason != "no "+home.StateFile {
		t.Fatalf("an absent STATE.md is said: %+v", e)
	}
	spec := []byte(`{"topics":["local-model-performance"]}`)
	if p, err := r.service.PreflightSpec(ctx, spec, Choices{}); err != nil || p.PacketState != nil {
		t.Fatalf("a spec has no packet of its own yet: %+v %v", p.PacketState, err)
	}
}
