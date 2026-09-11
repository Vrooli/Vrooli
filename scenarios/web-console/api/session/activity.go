package session

import (
	"sync"
	"time"

	"web-console/internal/backend"
)

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#session-activity-contract

// ActivityState is the projection of an agent session between messages.
type ActivityState string

const (
	ActivityUnknown ActivityState = "unknown"
	ActivityWorking ActivityState = "working"
	ActivityIdle    ActivityState = "idle"
	ActivityWaiting ActivityState = "waiting"
)

// ActivitySource names the evidence behind an Activity.
type ActivitySource string

const (
	SourceScreen       ActivitySource = "screen"
	SourceOutputClock  ActivitySource = "output_clock"
	SourceHook         ActivitySource = "hook"
	SourceHarnessEvent ActivitySource = "harness_event"
)

// Activity is what a session's agent is doing, as far as Web Console can tell.
type Activity struct {
	State          ActivityState
	Source         ActivitySource
	Confidence     float32
	Since          time.Time
	LastOutputAt   time.Time
	Prompt         *backend.PendingPrompt
	Harness        string
	HarnessVersion string
	// Answerable is true when Messages may answer Prompt (see ActivityConfig.Answering).
	Answerable bool
}

// Hook kinds a harness hook can report (see OnHook).
const (
	HookWaiting = "waiting" // a permission or question prompt is up
	HookIdle    = "idle"    // the turn ended; the agent is at its prompt
	HookOutput  = "output"  // activity worth noting, no state change
)

const (
	activityDebounce        = 150 * time.Millisecond
	activityQuietWindow     = 1500 * time.Millisecond
	activityHeartbeat       = 15 * time.Second
	activityEchoWindow      = 300 * time.Millisecond
	activityConfidenceFloor = 0.6

	confidenceHarnessEvent = 0.95
	confidenceHook         = 0.9
	confidenceOutput       = 0.7
	confidenceClockOnly    = 0.5
)

// ActivityClock abstracts time so the detector's timers can be driven in tests.
type ActivityClock interface {
	Now() time.Time
	AfterFunc(d time.Duration, f func()) (stop func() bool)
}

type realActivityClock struct{}

func (realActivityClock) Now() time.Time { return time.Now() }

func (realActivityClock) AfterFunc(d time.Duration, f func()) func() bool {
	return time.AfterFunc(d, f).Stop
}

// ActivityConfig wires a detector to its session and to the server.
type ActivityConfig struct {
	SessionID string
	// Screen reads the session's emulator. It takes the emulator's own lock,
	// so the detector never calls it from the PTY read path.
	Screen func() backend.ScreenView
	// Harness resolves the agent harness and its screen detector at each
	// evaluation; a session's agent type can become known after it starts.
	Harness func() (name string, detector backend.PromptDetector)
	// Publish receives each change worth pushing (and the working heartbeat).
	Publish func(Activity)
	// Answering reports whether Messages may answer the prompt a waiting
	// activity shows, and the harness version that decided it. Nil keeps
	// every prompt read-only.
	Answering func(harness string, prompt backend.PendingPrompt) (version string, answerable bool)
	Clock     ActivityClock
}

// ActivityDetector derives a session's Activity from output frames, its
// screen, harness hooks, and harness events. It is event-driven: output
// schedules one debounced screen read and one quiet-window check; nothing
// polls. Evidence ranks harness event > hook > screen prompt box > screen
// glyph > output clock.
type ActivityDetector struct {
	cfg   ActivityConfig
	clock ActivityClock

	mu            sync.Mutex
	current       Activity
	published     Activity
	hasPublished  bool
	lastFrame     time.Time
	lastInput     time.Time
	hookWaiting   bool
	hookAt        time.Time
	harnessPrompt *backend.PendingPrompt
	harnessState  ActivityState
	stopEval      func() bool
	stopQuiet     func() bool
	stopBeat      func() bool
	closed        bool
}

// NewActivityDetector returns a detector in the unknown state.
func NewActivityDetector(cfg ActivityConfig) *ActivityDetector {
	clock := cfg.Clock
	if clock == nil {
		clock = realActivityClock{}
	}
	if cfg.Harness == nil {
		cfg.Harness = func() (string, backend.PromptDetector) { return "", nil }
	}
	return &ActivityDetector{
		cfg:     cfg,
		clock:   clock,
		current: Activity{State: ActivityUnknown, Source: SourceOutputClock, Since: clock.Now()},
	}
}

// Current returns the activity as reported: below the confidence floor any
// state reads as unknown.
func (d *ActivityDetector) Current() Activity {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.answeringLocked(reportedActivity(d.current))
}

// OnFrame records terminal output. It runs on the PTY read path with the
// emulator locked, so it only records time and schedules work.
func (d *ActivityDetector) OnFrame() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return
	}
	now := d.clock.Now()
	d.lastFrame = now
	d.current.LastOutputAt = now
	echo := !d.lastInput.IsZero() && now.Sub(d.lastInput) < activityEchoWindow
	// Output makes an idle or unknown session working. It never replaces
	// stronger evidence for a session already working (the screen's
	// in-progress marker) with the output clock's weaker reading.
	if !echo && d.harnessPrompt == nil && d.harnessState == "" &&
		d.current.State != ActivityWaiting && d.current.State != ActivityWorking {
		d.setLocked(ActivityWorking, SourceOutputClock, confidenceOutput, nil)
	}
	d.scheduleEvalLocked()
	d.scheduleQuietLocked()
}

// OnInput records that the user's input reached the terminal. Output within
// the echo window after it is the echo, not the agent working.
func (d *ActivityDetector) OnInput() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return
	}
	d.lastInput = d.clock.Now()
	d.scheduleEvalLocked()
}

// OnHook applies a harness hook: HookWaiting, HookIdle, or HookOutput.
func (d *ActivityDetector) OnHook(kind string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return
	}
	now := d.clock.Now()
	switch kind {
	case HookWaiting:
		d.hookWaiting = true
		d.hookAt = now
		prompt := d.current.Prompt
		if prompt == nil {
			prompt = &backend.PendingPrompt{Kind: "unknown"}
		}
		d.setLocked(ActivityWaiting, SourceHook, confidenceHook, prompt)
		d.scheduleEvalLocked() // the screen fills in what is being asked
	case HookIdle:
		d.hookWaiting = false
		d.setLocked(ActivityIdle, SourceHook, confidenceHook, nil)
	case HookOutput:
		d.current.LastOutputAt = now
	}
}

// OnHarnessPrompt applies a harness event that raised a prompt (OpenCode
// permission.asked). It holds until OnHarnessPromptResolved.
func (d *ActivityDetector) OnHarnessPrompt(prompt backend.PendingPrompt) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return
	}
	d.harnessPrompt = &prompt
	d.setLocked(ActivityWaiting, SourceHarnessEvent, confidenceHarnessEvent, d.harnessPrompt)
}

// OnHarnessPromptResolved clears a harness prompt; the agent resumes.
func (d *ActivityDetector) OnHarnessPromptResolved() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed || d.harnessPrompt == nil {
		return
	}
	d.harnessPrompt = nil
	d.setLocked(ActivityWorking, SourceHarnessEvent, confidenceHarnessEvent, nil)
}

// OnHarnessState applies a harness-reported working or idle state (OpenCode
// session status). It ranks above the screen until a prompt appears.
func (d *ActivityDetector) OnHarnessState(state ActivityState) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed || (state != ActivityWorking && state != ActivityIdle) {
		return
	}
	d.harnessState = state
	if d.harnessPrompt == nil {
		d.setLocked(state, SourceHarnessEvent, confidenceHarnessEvent, nil)
	}
}

// Close stops the detector's timers; later events are ignored.
func (d *ActivityDetector) Close() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.closed = true
	for _, stop := range []func() bool{d.stopEval, d.stopQuiet, d.stopBeat} {
		if stop != nil {
			stop()
		}
	}
}

// scheduleEvalLocked reads the screen at most once per activityDebounce. It
// never postpones a pending evaluation: an agent's spinner redraws faster than
// the window, and a trailing debounce reset by every frame would never read
// the screen while the agent works.
func (d *ActivityDetector) scheduleEvalLocked() {
	if d.stopEval != nil {
		return
	}
	d.stopEval = d.clock.AfterFunc(activityDebounce, func() {
		d.mu.Lock()
		d.stopEval = nil
		d.mu.Unlock()
		d.evaluate(false)
	})
}

func (d *ActivityDetector) scheduleQuietLocked() {
	if d.stopQuiet != nil {
		d.stopQuiet()
	}
	d.stopQuiet = d.clock.AfterFunc(activityQuietWindow, func() { d.evaluate(true) })
}

// evaluate reads the screen once (outside the detector lock: the read takes
// the emulator lock) and applies the evidence in rank order.
func (d *ActivityDetector) evaluate(quiet bool) {
	name, detector := d.cfg.Harness()
	var analysis backend.PromptAnalysis
	if detector != nil && d.cfg.Screen != nil {
		if view := d.cfg.Screen(); view != nil {
			analysis = detector.Analyze(view)
		}
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return
	}
	d.current.Harness = name
	now := d.clock.Now()
	outputSinceHook := d.lastFrame.After(d.hookAt)
	recentOutput := !d.lastFrame.IsZero() && now.Sub(d.lastFrame) < activityQuietWindow
	echoOnly := !d.lastInput.IsZero() && !d.lastFrame.IsZero() && d.lastFrame.Sub(d.lastInput) < activityEchoWindow

	switch {
	case d.harnessPrompt != nil:
		d.setLocked(ActivityWaiting, SourceHarnessEvent, confidenceHarnessEvent, d.harnessPrompt)
	case analysis.Prompt != nil:
		confidence := analysis.Confidence
		if d.hookWaiting && confidence < confidenceHook {
			confidence = confidenceHook
		}
		d.setLocked(ActivityWaiting, SourceScreen, confidence, analysis.Prompt)
	case d.hookWaiting && !outputSinceHook:
		d.setLocked(ActivityWaiting, SourceHook, confidenceHook, d.current.Prompt)
	case d.harnessState != "":
		d.setLocked(d.harnessState, SourceHarnessEvent, confidenceHarnessEvent, nil)
	case analysis.Working:
		d.setLocked(ActivityWorking, SourceScreen, analysis.Confidence, nil)
	case analysis.AwaitingInput && (quiet || echoOnly || !recentOutput):
		d.setLocked(ActivityIdle, SourceScreen, analysis.Confidence, nil)
	case recentOutput && !quiet && !echoOnly:
		d.setLocked(ActivityWorking, SourceOutputClock, confidenceOutput, nil)
	case quiet:
		d.setLocked(ActivityUnknown, SourceOutputClock, confidenceClockOnly, nil)
	}
	if d.hookWaiting && analysis.Prompt == nil && outputSinceHook {
		d.hookWaiting = false
	}
}

// setLocked moves to a state and publishes when the reported state or the
// prompt changed.
func (d *ActivityDetector) setLocked(state ActivityState, source ActivitySource, confidence float32, prompt *backend.PendingPrompt) {
	if state != d.current.State {
		d.current.Since = d.clock.Now()
	}
	d.current.State = state
	d.current.Source = source
	d.current.Confidence = confidence
	d.current.Prompt = prompt
	d.publishLocked(false)
	if state == ActivityWorking {
		d.ensureHeartbeatLocked()
	}
}

func (d *ActivityDetector) publishLocked(force bool) {
	reported := d.answeringLocked(reportedActivity(d.current))
	changed := !d.hasPublished ||
		reported.State != d.published.State ||
		!samePrompt(reported.Prompt, d.published.Prompt)
	if !changed && !force {
		return
	}
	d.published = reported
	d.hasPublished = true
	if d.cfg.Publish != nil {
		d.cfg.Publish(reported)
	}
}

// ensureHeartbeatLocked republishes every activityHeartbeat while working so
// the working strip's "last output" stays fresh without per-frame traffic.
func (d *ActivityDetector) ensureHeartbeatLocked() {
	if d.stopBeat != nil {
		return
	}
	var beat func()
	beat = func() {
		d.mu.Lock()
		defer d.mu.Unlock()
		d.stopBeat = nil
		if d.closed || d.current.State != ActivityWorking {
			return
		}
		d.publishLocked(true)
		d.stopBeat = d.clock.AfterFunc(activityHeartbeat, beat)
	}
	d.stopBeat = d.clock.AfterFunc(activityHeartbeat, beat)
}

func reportedActivity(a Activity) Activity {
	if a.State != ActivityUnknown && a.Confidence < activityConfidenceFloor {
		a.State = ActivityUnknown
		a.Prompt = nil
	}
	return a
}

// answeringLocked asks the host answering policy about the prompt a waiting
// activity reports.
func (d *ActivityDetector) answeringLocked(a Activity) Activity {
	if a.State != ActivityWaiting || a.Prompt == nil || d.cfg.Answering == nil {
		return a
	}
	a.HarnessVersion, a.Answerable = d.cfg.Answering(a.Harness, *a.Prompt)
	return a
}

func samePrompt(a, b *backend.PendingPrompt) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.Kind != b.Kind || a.Text != b.Text || len(a.Options) != len(b.Options) {
		return false
	}
	for i := range a.Options {
		if a.Options[i] != b.Options[i] {
			return false
		}
	}
	return true
}
