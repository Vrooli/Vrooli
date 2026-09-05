package lifecycle

import (
	"fmt"
	"time"
)

// Typed lifecycle progress (plan Phase 2). Start/stop/restart orchestration
// emits ProgressEvent values instead of printing text; sinks consume them as
// data. The text renderer below reproduces the historical progressf lines
// byte-for-byte for humans; the registry sink (operation record) and test
// capture sinks consume the same stream. There is intentionally no parallel
// print path — if it isn't an event, it doesn't reach the console.

// ProgressEventKind names one observable lifecycle transition. The set is
// closed: sinks are entitled to exhaustively switch on it.
type ProgressEventKind string

const (
	// EventOperationStarted marks the beginning of a start/restart body for
	// the operation's scenario (after any restart-triggered stop).
	EventOperationStarted ProgressEventKind = "operation_started"
	// EventOperationCompleted marks a terminal success. AlreadyRunning
	// distinguishes the reuse fast-path from a full start.
	EventOperationCompleted ProgressEventKind = "operation_completed"
	// EventOperationFailed marks a terminal failure at Step.
	EventOperationFailed ProgressEventKind = "operation_failed"
	// EventStopStarted marks the beginning of a stop (standalone or the
	// leading step of a restart).
	EventStopStarted ProgressEventKind = "stop_started"
	// EventPhaseStarted marks a lifecycle phase (setup|develop) beginning.
	EventPhaseStarted ProgressEventKind = "phase_started"
	// EventPhaseCompleted marks a lifecycle phase finishing successfully.
	EventPhaseCompleted ProgressEventKind = "phase_completed"
	// EventHealthWaiting marks the health gate beginning.
	EventHealthWaiting ProgressEventKind = "health_waiting"
	// EventDependencyStarting marks a scenario dependency (Index of Total)
	// requiring a (re)start for Reason.
	EventDependencyStarting ProgressEventKind = "dependency_starting"
	// EventDependencyReused marks a scenario dependency reused as-is.
	// AfterLockWait distinguishes reuse discovered while waiting on the
	// dependency's lifecycle lock.
	EventDependencyReused ProgressEventKind = "dependency_reused"
	// EventDependencyStalePolicy marks a running-but-stale dependency being
	// kept per its freshness policy (Policy: reuse_running|rebuild_only).
	EventDependencyStalePolicy ProgressEventKind = "dependency_stale_policy"
	// EventResourceStarting marks a resource dependency requiring a start.
	EventResourceStarting ProgressEventKind = "resource_starting"
	// EventResourceReused marks a resource dependency reused as-is.
	EventResourceReused ProgressEventKind = "resource_reused"
	// EventResourceEnsureConfig marks the resource `ensure` verb running with
	// scenario-declared config.
	EventResourceEnsureConfig ProgressEventKind = "resource_ensure_config"
)

// ProgressEvent is one typed lifecycle transition. Scenario is always the
// scenario the event is about; for dependency/resource events it is the
// PARENT scenario being started and Dependency names the dep. At is stamped
// from the runner's injected clock at publish time.
type ProgressEvent struct {
	At             time.Time
	Kind           ProgressEventKind
	Scenario       string
	Operation      string // start|restart|stop (top-level verb)
	Phase          string // setup|develop for phase events
	Dependency     string // scenario-dependency or resource name
	Reason         string // human reason for a (re)start decision
	Policy         string // freshness policy for stale-policy events
	Verdict        string // health verdict for completion events
	AlreadyRunning bool
	AfterLockWait  bool
	Index          int // 1-based dependency position (dependency events)
	Total          int // dependency count (dependency events)
	Err            error
}

// ProgressSink consumes lifecycle progress events. Implementations must be
// cheap and non-blocking: they run inline on the orchestration path.
type ProgressSink interface {
	Publish(ProgressEvent)
}

// publish stamps and fans an event out to every configured sink. The default
// sink set is exactly the text renderer; WithProgressSink appends more.
func (r *Runner) publish(ev ProgressEvent) {
	ev.At = r.runtimeDeps().now()
	r.sinksMu.RLock()
	sinks := append([]ProgressSink(nil), r.sinks...)
	r.sinksMu.RUnlock()
	if len(sinks) == 0 {
		r.Publish(ev) // default renderer via the Runner's own sink impl
		return
	}
	for _, sink := range sinks {
		sink.Publish(ev)
	}
}

// WithProgressSink registers an additional progress sink alongside the text
// renderer and returns the runner for chaining.
func (r *Runner) WithProgressSink(sink ProgressSink) *Runner {
	r.sinksMu.Lock()
	defer r.sinksMu.Unlock()
	if r.sinks == nil {
		r.sinks = []ProgressSink{r}
	}
	r.sinks = append(r.sinks, sink)
	return r
}

// Publish implements ProgressSink on the Runner itself: the built-in text
// renderer. It reproduces the historical progressf lines byte-for-byte and
// keeps their verbosity gating: lines are the primary in-flight heartbeat at
// Quiet and Normal (where the structured info stream is suppressed on TTYs)
// and are suppressed at Verbose, where the slog debug stream plus raw tool
// stdout already give a running picture. Written without color codes or
// carriage returns so the output stays CI- and log-capture-safe.
func (r *Runner) Publish(ev ProgressEvent) {
	if r.Verbosity == VerbosityVerbose || r.Out == nil {
		return
	}
	if line := renderProgressLine(ev); line != "" {
		fmt.Fprint(r.Out, line)
	}
}

// renderProgressLine maps an event to its historical console line ("" for
// events with no human line). Characterization tests assert these bytes.
func renderProgressLine(ev ProgressEvent) string {
	switch ev.Kind {
	case EventOperationStarted:
		return fmt.Sprintf("starting %s...\n", ev.Scenario)
	case EventOperationCompleted:
		if ev.AlreadyRunning {
			return fmt.Sprintf("%s is already running\n", ev.Scenario)
		}
		return ""
	case EventStopStarted:
		return fmt.Sprintf("stopping %s...\n", ev.Scenario)
	case EventPhaseStarted:
		return fmt.Sprintf("running %s phase for %s...\n", ev.Phase, ev.Scenario)
	case EventHealthWaiting:
		return fmt.Sprintf("waiting for %s to become healthy...\n", ev.Scenario)
	case EventDependencyStarting:
		return fmt.Sprintf("%s: starting dependency %s (%s)\n", ev.Scenario, ev.Dependency, ev.Reason)
	case EventDependencyReused:
		if ev.AfterLockWait {
			return fmt.Sprintf("%s: dependency %s became ready while another invocation held its lifecycle lock; reusing existing process\n", ev.Scenario, ev.Dependency)
		}
		return fmt.Sprintf("%s: dependency %s already running; reusing existing process\n", ev.Scenario, ev.Dependency)
	case EventDependencyStalePolicy:
		switch ev.Policy {
		case "reuse_running":
			return fmt.Sprintf("%s: stale but reused per freshness_policy=reuse_running (%s)\n", ev.Dependency, ev.Reason)
		case "rebuild_only":
			return fmt.Sprintf("%s: rebuilding stale dependency without restart per freshness_policy=rebuild_only (%s)\n", ev.Dependency, ev.Reason)
		}
		return ""
	case EventResourceStarting:
		return fmt.Sprintf("%s: starting resource dependency %s (%s)\n", ev.Scenario, ev.Dependency, ev.Reason)
	case EventResourceReused:
		return fmt.Sprintf("%s: resource dependency %s already running; reusing existing service\n", ev.Scenario, ev.Dependency)
	case EventResourceEnsureConfig:
		return fmt.Sprintf("%s: ensuring %s dependency config\n", ev.Scenario, ev.Dependency)
	}
	return ""
}
