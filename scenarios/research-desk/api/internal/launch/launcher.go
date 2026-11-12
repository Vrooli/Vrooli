// Package launch starts a supervisor from a commission packet and controls
// it afterwards (SYSTEM.md §5–§6). It types into a Web Console session only
// on an operator tap and verifies every step from Codex's own files.
package launch

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"syscall"
	"time"

	"research-desk/internal/home"
)

// Steps of an automatic launch, in order (FILE-CONTRACT §3).
const (
	StepCreated       = "created"
	StepComposerReady = "composer_ready"
	StepStanceSent    = "stance_sent"
	StepReadySeen     = "ready_seen"
	StepGoalSent      = "goal_sent"
	StepGoalVerified  = "goal_verified"
)

// Sessions reads Codex's own records of one session.
type Sessions interface {
	SessionGoals(ctx context.Context, sessionID string) ([]home.Goal, error)
	SessionProcesses(sessionID string) ([]home.Process, error)
	ThreadView(sessionID, threadID string) (home.ThreadView, error)
}

// Target is one commission's conversation: the Web Console session and the
// Codex thread inside it. Controls read and type for this thread only.
type Target struct {
	Session string
	Thread  string
}

// ErrNotReady is a control refused because the session is not showing the
// commission's thread at an idle input row. Nothing was typed.
var ErrNotReady = errors.New("not ready")

// Launcher drives one Codex session through Web Console.
type Launcher struct {
	Terminal Terminal
	Sessions Sessions
	// WorkingDir is where the supervisor's shell starts (the repository).
	WorkingDir string
	// Signal ends one process; nil sends SIGTERM.
	Signal func(pid int) error
	// Poll is the pause between file reads; zero means one second.
	Poll time.Duration
	// Patience bounds each wait; zero uses the defaults below.
	Patience Patience
}

// Patience bounds each wait of a launch or control.
type Patience struct {
	Composer time.Duration // Codex shows its input row
	Ready    time.Duration // Codex replies "ready" to the stance
	Goal     time.Duration // the goal row appears
	Stop     time.Duration // the session's processes end
	// Settle is the pause after typing and after Enter before the screen is
	// read. Codex treats keys that arrive in one burst as a paste, so an Enter
	// right behind the text inserts a newline instead of submitting.
	Settle time.Duration
}

func (l *Launcher) patience() Patience {
	p := l.Patience
	if p.Composer == 0 {
		p.Composer = 3 * time.Minute
	}
	if p.Ready == 0 {
		p.Ready = 5 * time.Minute
	}
	if p.Goal == 0 {
		p.Goal = 2 * time.Minute
	}
	if p.Stop == 0 {
		p.Stop = 15 * time.Second
	}
	if p.Settle == 0 {
		p.Settle = time.Second
	}
	return p
}

func (l *Launcher) poll() time.Duration {
	if l.Poll == 0 {
		return time.Second
	}
	return l.Poll
}

// Texts are the two messages a launch sends.
type Texts struct {
	Stance string
	Kernel string
}

// Settings are the supervisor's model and reasoning effort.
type Settings struct {
	Model  string
	Effort string
}

var settingValue = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

// Command is the governed launcher with the commission's settings, falling
// back to plain codex as Web Console's own Codex profile does.
func Command(s Settings) (string, error) {
	if !settingValue.MatchString(s.Model) || !settingValue.MatchString(s.Effort) {
		return "", fmt.Errorf("model %q and effort %q must be values from limits.json", s.Model, s.Effort)
	}
	args := "--yolo -m " + s.Model + " -c model_reasoning_effort=" + s.Effort
	return "if command -v vrooli-agent-launcher >/dev/null 2>&1; then exec vrooli-agent-launcher --agent codex -- " + args + "; fi; exec codex " + args, nil
}

// Progress records one completed step; an error stops the launch.
type Progress func(step, sessionID, threadID string) error

// StepError is a launch that stopped at a step.
type StepError struct {
	Step string
	Err  error
}

func (e *StepError) Error() string { return e.Step + ": " + e.Err.Error() }
func (e *StepError) Unwrap() error { return e.Err }

// failure splits a launch error into the step it stopped at and the reason.
func failure(err error) (step, reason string) {
	var stepErr *StepError
	if errors.As(err, &stepErr) {
		return stepErr.Step, stepErr.Err.Error()
	}
	return StepCreated, err.Error()
}

// Run creates the session, sends the stance, waits for "ready", sets the goal
// and verifies it from goals_1.sqlite. Nothing else is typed.
func (l *Launcher) Run(ctx context.Context, spec SessionSpec, texts Texts, settings Settings, progress Progress) error {
	stance := strings.TrimSpace(texts.Stance)
	kernel := strings.TrimSpace(texts.Kernel)
	if stance == "" || kernel == "" || strings.ContainsAny(kernel, "\r\n") {
		return &StepError{StepCreated, errors.New("the stance must not be empty and the kernel must be one non-empty line")}
	}
	command, err := Command(settings)
	if err != nil {
		return &StepError{StepCreated, err}
	}
	spec.Command = command
	if spec.WorkingDir == "" {
		spec.WorkingDir = l.WorkingDir
	}
	p := l.patience()
	id, err := l.Terminal.Create(ctx, spec)
	if err != nil {
		return &StepError{StepCreated, err}
	}
	steps := []struct {
		name string
		do   func() (string, error)
	}{
		{StepCreated, func() (string, error) { return "", nil }},
		{StepComposerReady, func() (string, error) { return "", l.awaitComposer(ctx, id, p.Composer, false) }},
		{StepStanceSent, func() (string, error) { return "", l.enter(ctx, id, stance, true) }},
		{StepReadySeen, func() (string, error) { return "", l.awaitComposer(ctx, id, p.Ready, true) }},
		{StepGoalSent, func() (string, error) { return "", l.enter(ctx, id, "/goal "+kernel, false) }},
		{StepGoalVerified, func() (string, error) { return l.awaitGoal(ctx, id, kernel, p.Goal) }},
	}
	for _, step := range steps {
		thread, err := step.do()
		if err == nil {
			err = progress(step.name, id, thread)
		}
		if err != nil {
			return &StepError{step.name, err}
		}
	}
	return nil
}

// enter types text, lets the terminal settle, presses Enter and checks the
// input row let go of the text; one more Enter covers an Enter Codex took as
// part of the typing burst (measured on the first launch test).
func (l *Launcher) enter(ctx context.Context, id, text string, paste bool) error {
	if err := l.Terminal.Type(ctx, id, text, paste); err != nil {
		return err
	}
	if err := l.settle(ctx, id); err != nil {
		return err
	}
	for range 2 {
		if err := l.Terminal.Keys(ctx, id, Enter); err != nil {
			return err
		}
		if err := l.settle(ctx, id); err != nil {
			return err
		}
		screen, err := l.Terminal.Screen(ctx, id)
		if err != nil {
			return err
		}
		if !holds(screen, text) {
			return nil
		}
	}
	return fmt.Errorf("Codex kept %q in its input row after two Enters; see the session in Web Console", excerpt(text))
}

// settle waits the settle time, then for quiet output.
func (l *Launcher) settle(ctx context.Context, id string) error {
	if err := sleep(ctx, l.patience().Settle); err != nil {
		return err
	}
	_, err := l.Terminal.WaitIdle(ctx, id, time.Second, 5*time.Second)
	return err
}

// awaitComposer waits until Codex shows its input row (and, when ready is
// set, has replied "ready").
func (l *Launcher) awaitComposer(ctx context.Context, id string, within time.Duration, ready bool) error {
	deadline := time.Now().Add(within)
	var last Screen
	var choice string
	for {
		exited, err := l.Terminal.WaitIdle(ctx, id, 2*time.Second, 10*time.Second)
		if err != nil {
			return err
		}
		if exited {
			return errors.New("the session's terminal exited; Codex did not start (see the session in Web Console)")
		}
		if last, err = l.Terminal.Screen(ctx, id); err != nil {
			return err
		}
		var ok bool
		ok, choice = composer(last)
		if ok && (!ready || repliedReady(last)) {
			return nil
		}
		if time.Now().After(deadline) {
			break
		}
		if err := sleep(ctx, l.poll()); err != nil {
			return err
		}
	}
	switch {
	case choice != "":
		return fmt.Errorf("Codex is showing a choice (%q); answer it in Web Console, discard this commission and relaunch", choice)
	case ready:
		return fmt.Errorf("Codex did not reply \"ready\" within %s; last line: %q", within, lastLine(last))
	default:
		return fmt.Errorf("Codex did not show its input row within %s; last line: %q", within, lastLine(last))
	}
}

// awaitGoal waits for the goal row whose objective is the kernel.
func (l *Launcher) awaitGoal(ctx context.Context, id, kernel string, within time.Duration) (string, error) {
	deadline := time.Now().Add(within)
	for {
		goals, err := l.Sessions.SessionGoals(ctx, id)
		if err != nil {
			return "", fmt.Errorf("read the session's goals_1.sqlite: %w", err)
		}
		for _, g := range goals {
			if strings.TrimSpace(g.Objective) == kernel && (g.Status == "active" || g.Status == "complete") {
				return g.ThreadID, nil
			}
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("no goal with the kernel as its objective appeared within %s (%d goal row(s) in the session)", within, len(goals))
		}
		if err := sleep(ctx, l.poll()); err != nil {
			return "", err
		}
	}
}

// GoalStatus is the status of the target thread's goal row. A goal database
// that cannot be read, or a thread without a goal row, is an error: the
// status is unknown, never "".
func (l *Launcher) GoalStatus(ctx context.Context, t Target) (string, error) {
	goals, err := l.Sessions.SessionGoals(ctx, t.Session)
	if err != nil {
		return "", fmt.Errorf("read goals_1.sqlite of session %s: %w", t.Session, err)
	}
	for _, g := range goals {
		if g.ThreadID == t.Thread {
			return g.Status, nil
		}
	}
	return "", fmt.Errorf("session %s has no goal row for thread %s", t.Session, t.Thread)
}

// manual is what the operator can do instead of a refused control.
const manual = "do it by hand in the session in Web Console, or write a wind-down in OPERATOR.md, which the scheduler reads at its next unit"

// Ready checks, before anything is typed, that the session's terminal shows
// the target thread at an idle input row: Codex's transcripts name it as the
// shown thread with no turn running, and the screen shows the input row.
func (l *Launcher) Ready(ctx context.Context, t Target) error {
	return l.ready(ctx, t, false)
}

// ready is Ready; duringTurn accepts a running turn. Only /goal pause and
// /goal resume use it: typed with Enter during a turn, Codex applies them at
// once without interrupting the turn and records no chat message (measured,
// SYSTEM.md §6). Anything else typed then would queue as a message.
func (l *Launcher) ready(ctx context.Context, t Target, duringTurn bool) error {
	if t.Session == "" || t.Thread == "" {
		return fmt.Errorf("%w: the commission's Codex thread is not known; %s", ErrNotReady, manual)
	}
	view, err := l.Sessions.ThreadView(t.Session, t.Thread)
	switch {
	case err != nil:
		return fmt.Errorf("%w: cannot tell what session %s shows (%v); %s", ErrNotReady, t.Session, err, manual)
	case view.Shown != t.Thread:
		return fmt.Errorf("%w: session %s now shows thread %s, not the commission's thread %s; %s", ErrNotReady, t.Session, view.Shown, t.Thread, manual)
	case view.TurnRunning && !duringTurn:
		return fmt.Errorf("%w: a turn is running in thread %s; try again when it ends, or %s", ErrNotReady, t.Thread, manual)
	}
	screen, err := l.Terminal.Screen(ctx, t.Session)
	if err != nil {
		return fmt.Errorf("%w: cannot read the session's screen (%v); %s", ErrNotReady, err, manual)
	}
	if ok, choice := inputRow(screen, duringTurn); !ok {
		if choice != "" {
			return fmt.Errorf("%w: Codex is showing a choice (%q); %s", ErrNotReady, choice, manual)
		}
		return fmt.Errorf("%w: Codex is not at its input row (last line %q); %s", ErrNotReady, lastLine(screen), manual)
	}
	return nil
}

// AwaitReady waits until Ready passes or within ends.
func (l *Launcher) AwaitReady(ctx context.Context, t Target, within time.Duration) error {
	deadline := time.Now().Add(within)
	for {
		err := l.Ready(ctx, t)
		if err == nil || time.Now().After(deadline) {
			return err
		}
		if err := sleep(ctx, l.poll()); err != nil {
			return err
		}
	}
}

// Pause types /goal pause and returns the goal status read back from
// goals_1.sqlite once it is paused (or after 15 s). A running turn goes on to
// its end; no further goal turn starts.
func (l *Launcher) Pause(ctx context.Context, t Target) (string, error) {
	return l.goalCommand(ctx, t, "pause", func(s string) bool { return s == "paused" })
}

// Resume types /goal resume and returns the status once it is no longer
// paused: active, or complete when the goal finishes at once.
func (l *Launcher) Resume(ctx context.Context, t Target) (string, error) {
	return l.goalCommand(ctx, t, "resume", func(s string) bool { return s != "" && s != "paused" })
}

func (l *Launcher) goalCommand(ctx context.Context, t Target, sub string, done func(string) bool) (string, error) {
	if err := l.ready(ctx, t, true); err != nil {
		return "", err
	}
	if err := l.enter(ctx, t.Session, "/goal "+sub, false); err != nil {
		return "", err
	}
	return l.AwaitStatus(ctx, t, done, 15*time.Second)
}

// AwaitStatus reads the goal status until done accepts it or within passes,
// and returns the last status read.
func (l *Launcher) AwaitStatus(ctx context.Context, t Target, done func(string) bool, within time.Duration) (string, error) {
	deadline := time.Now().Add(within)
	for {
		status, err := l.GoalStatus(ctx, t)
		if err != nil || done(status) || time.Now().After(deadline) {
			return status, err
		}
		if err := sleep(ctx, l.poll()); err != nil {
			return status, err
		}
	}
}

// ResendStance pastes the stance again and submits it.
func (l *Launcher) ResendStance(ctx context.Context, t Target, stance string) error {
	if err := l.Ready(ctx, t); err != nil {
		return err
	}
	return l.enter(ctx, t.Session, strings.TrimSpace(stance), true)
}

// Stop ends a commission's session. When its window is open it first checks
// Ready, pauses an active goal and quits Codex; then it ends the remaining
// Codex processes of that session only (the app-server outlives the window).
// A process that merely shares the Codex home is left alone. The session row
// and its files stay. The result says what was found and ended.
func (l *Launcher) Stop(ctx context.Context, t Target, windowOpen bool) (Stopped, error) {
	out := Stopped{Session: t.Session}
	if strings.TrimSpace(t.Session) == "" {
		return out, errors.New("no session to stop")
	}
	if windowOpen {
		if err := l.Ready(ctx, t); err != nil {
			return out, err
		}
		if status, err := l.GoalStatus(ctx, t); err == nil && status == "active" {
			if _, err := l.Pause(ctx, t); err != nil {
				return out, fmt.Errorf("pause the goal before quitting: %w", err)
			}
		}
		for _, keys := range [][]Key{{Escape}, {CtrlC}, {CtrlC}} {
			if err := l.Terminal.Keys(ctx, t.Session, keys...); err != nil {
				return out, err
			}
			if _, err := l.Terminal.WaitIdle(ctx, t.Session, time.Second, 5*time.Second); err != nil {
				return out, err
			}
		}
		out.Window = true
	}
	signal := l.Signal
	if signal == nil {
		signal = func(pid int) error { return syscall.Kill(pid, syscall.SIGTERM) }
	}
	deadline := time.Now().Add(l.patience().Stop)
	signalled := map[int]bool{}
	for {
		procs, err := l.Sessions.SessionProcesses(t.Session)
		if err != nil {
			return out, fmt.Errorf("check the session's Codex processes: %w", err)
		}
		var pids []string
		for _, p := range procs {
			if !p.OfSession(t.Session) {
				continue
			}
			pids = append(pids, fmt.Sprint(p.PID))
			if !signalled[p.PID] {
				signalled[p.PID] = true
				if err := signal(p.PID); err != nil && !errors.Is(err, syscall.ESRCH) {
					return out, fmt.Errorf("end Codex process %d: %w", p.PID, err)
				}
			}
		}
		out.Ended = len(signalled)
		if len(pids) == 0 {
			return out, nil
		}
		if time.Now().After(deadline) {
			return out, fmt.Errorf("Codex processes %s of this session are still running; end them from Web Console", strings.Join(pids, ", "))
		}
		if err := sleep(ctx, l.poll()); err != nil {
			return out, err
		}
	}
}

// Stopped is what a stop found and ended in one session.
type Stopped struct {
	Session string
	// Window is true when a supervisor window was open and was quit.
	Window bool
	// Ended counts the session's Codex processes that were signalled.
	Ended int
}

// Outcome says in plain words what a successful stop left: whether a
// window was found, and that nothing of the session is still running.
func (s Stopped) Outcome() string {
	found := "no window found"
	if s.Window {
		found = "window stopped"
	}
	return fmt.Sprintf("%s; %d Codex process(es) of session %s ended; none still running", found, s.Ended, s.Session)
}

func sleep(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}
