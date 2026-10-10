package agentsession

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Liveness states of an owner.
const (
	// OwnerAlive is an owner its authority confirmed present.
	OwnerAlive = "alive"
	// OwnerGone is an owner its authority confirmed absent: the run is
	// terminal or unknown to Agent Manager, the web-console session is
	// archived or absent, the app's process is gone.
	OwnerGone = "gone"
	// OwnerUnknown is an owner that could not be asked or did not answer.
	// It is never treated as gone.
	OwnerUnknown = "unknown"
	// OwnerNotApplicable is a terminal or unknown owner: there is no
	// authority to ask, and such a session is never orphaned automatically.
	OwnerNotApplicable = "not-applicable"
)

// Liveness is one answer about an owner.
type Liveness struct {
	State     string    `json:"state"`
	Detail    string    `json:"detail,omitempty"`
	CheckedAt time.Time `json:"checked_at"`
	// Origin and WebConsoleOwner are what web-console reported for a live
	// session; they are folded into the owner's evidence so they survive the
	// session's archival.
	Origin          string `json:"-"`
	WebConsoleOwner string `json:"-"`
}

// Alive renders the state as the JSON tri-state the list reports: true,
// false, or null when unknown or not applicable.
func (l Liveness) Alive() *bool {
	switch l.State {
	case OwnerAlive:
		v := true
		return &v
	case OwnerGone:
		v := false
		return &v
	default:
		return nil
	}
}

// ErrNotFound is what a probe returns when the owner's authority answered
// and does not know the owner.
var ErrNotFound = errors.New("agentsession: owner not found")

// ErrProcessGone is what a process probe returns for a pid that is not
// running.
var ErrProcessGone = errors.New("agentsession: process not running")

// WebConsoleSession is what web-console reports about a live session.
type WebConsoleSession struct {
	Origin string
	Owner  string
}

// Probes ask each owner's authority. Every probe is read-only. A nil probe
// leaves that kind unknown.
type Probes struct {
	// RunStatus returns an Agent Manager run's status (e.g.
	// RUN_STATUS_RUNNING), or ErrNotFound when Agent Manager answered that
	// it has no such run.
	RunStatus func(ctx context.Context, runID string) (string, error)
	// WebConsoleSession returns a live web-console session, or ErrNotFound
	// when the instance that owns stateRoot answered that it has no live
	// session with that id (archived, dismissed or never created). A probe
	// that cannot reach the instance owning stateRoot returns another error.
	WebConsoleSession func(ctx context.Context, stateRoot, sessionID string) (WebConsoleSession, error)
	// ProcessStartTime returns a pid's start stamp, or ErrProcessGone.
	ProcessStartTime func(pid int) (string, error)
	// DesktopAppProcesses lists live processes outside the agent slice that
	// carry dataDir as their VROOLI_DATA: the app's runtime and its services.
	DesktopAppProcesses func(dataDir string) ([]int, error)
}

// terminalRunStatuses are the Agent Manager run statuses after which a run
// owns nothing. RUN_STATUS_UNKNOWN is terminal to Agent Manager but is not
// confirmation of anything, so it is left out; PARKED and NEEDS_REVIEW are
// resting runs that resume.
var terminalRunStatuses = map[string]bool{
	"RUN_STATUS_COMPLETE":  true,
	"RUN_STATUS_FAILED":    true,
	"RUN_STATUS_CANCELLED": true,
	"COMPLETE":             true,
	"FAILED":               true,
	"CANCELLED":            true,
}

// Check asks an owner's authority whether it is still there.
func Check(ctx context.Context, owner Owner, probes Probes, now time.Time) Liveness {
	out := Liveness{State: OwnerUnknown, CheckedAt: now.UTC()}
	switch owner.Kind {
	case KindAgentManagerRun:
		return checkRun(ctx, owner, probes, out)
	case KindWebConsoleSession:
		return checkWebConsole(ctx, owner, probes, out)
	case KindDesktopApp:
		return checkDesktopApp(owner, probes, out)
	default:
		out.State = OwnerNotApplicable
		out.Detail = "no owner authority: a " + owner.Kind + " session is never orphaned automatically"
		return out
	}
}

func checkRun(ctx context.Context, owner Owner, probes Probes, out Liveness) Liveness {
	if owner.Ref == "" {
		out.Detail = "no run id in the session's environment"
		if owner.Evidence.AgentTag != "" {
			out.Detail += " (tag " + owner.Evidence.AgentTag + " names many runs, so it is not asked)"
		}
		return out
	}
	if probes.RunStatus == nil {
		out.Detail = "agent manager probe unavailable"
		return out
	}
	status, err := probes.RunStatus(ctx, owner.Ref)
	switch {
	case errors.Is(err, ErrNotFound):
		out.State, out.Detail = OwnerGone, "agent manager has no run "+owner.Ref
	case err != nil:
		out.Detail = "agent manager did not answer: " + err.Error()
	case terminalRunStatuses[strings.ToUpper(strings.TrimSpace(status))]:
		out.State, out.Detail = OwnerGone, "run "+owner.Ref+" is "+status
	case strings.Contains(strings.ToUpper(status), "UNKNOWN"):
		out.Detail = "agent manager reports run " + owner.Ref + " as " + status + ", which confirms nothing"
	default:
		out.State, out.Detail = OwnerAlive, "run "+owner.Ref+" is "+status
	}
	return out
}

func checkWebConsole(ctx context.Context, owner Owner, probes Probes, out Liveness) Liveness {
	if owner.Ref == "" {
		out.Detail = "no web-console session id"
		return out
	}
	if probes.WebConsoleSession == nil {
		out.Detail = "web-console probe unavailable"
		return out
	}
	session, err := probes.WebConsoleSession(ctx, owner.Evidence.WebConsoleStateRoot, owner.Ref)
	switch {
	case errors.Is(err, ErrNotFound):
		out.State, out.Detail = OwnerGone, "web-console has no live session "+owner.Ref+" (archived, dismissed or absent)"
	case err != nil:
		out.Detail = "web-console did not answer: " + err.Error()
	default:
		out.State = OwnerAlive
		out.Origin, out.WebConsoleOwner = session.Origin, session.Owner
		out.Detail = fmt.Sprintf("web-console session %s is live (origin %s, owner %q)", owner.Ref, nonEmpty(session.Origin, OriginUnspecified), session.Owner)
	}
	return out
}

func checkDesktopApp(owner Owner, probes Probes, out Liveness) Liveness {
	if owner.Evidence.AppPID > 0 && owner.Evidence.AppStartTime != "" {
		if probes.ProcessStartTime == nil {
			out.Detail = "process probe unavailable"
			return out
		}
		start, err := probes.ProcessStartTime(owner.Evidence.AppPID)
		switch {
		case errors.Is(err, ErrProcessGone):
			out.State, out.Detail = OwnerGone, fmt.Sprintf("desktop app runtime pid %d has exited", owner.Evidence.AppPID)
		case err != nil:
			out.Detail = fmt.Sprintf("desktop app runtime pid %d unreadable: %v", owner.Evidence.AppPID, err)
		case start != owner.Evidence.AppStartTime:
			out.State, out.Detail = OwnerGone, fmt.Sprintf("desktop app runtime pid %d has exited (pid reused by a process started at %s)", owner.Evidence.AppPID, start)
		default:
			out.State, out.Detail = OwnerAlive, fmt.Sprintf("desktop app runtime pid %d is running", owner.Evidence.AppPID)
		}
		return out
	}
	if owner.Ref == "" {
		out.Detail = "no desktop app data directory in the session's environment"
		return out
	}
	if probes.DesktopAppProcesses == nil {
		out.Detail = "process probe unavailable"
		return out
	}
	pids, err := probes.DesktopAppProcesses(owner.Ref)
	switch {
	case err != nil:
		out.Detail = "desktop app processes unreadable: " + err.Error()
	case len(pids) == 0:
		out.State, out.Detail = OwnerGone, "no process outside the agent slice carries VROOLI_DATA="+owner.Ref
	default:
		out.State, out.Detail = OwnerAlive, fmt.Sprintf("%d app process(es) carry VROOLI_DATA=%s (first pid %d)", len(pids), owner.Ref, pids[0])
	}
	return out
}

// WithObservedOrigin folds a live web-console session's reported origin and
// owner into the owner's evidence. It reports whether anything changed.
func WithObservedOrigin(owner Owner, liveness Liveness) (Owner, bool) {
	if owner.Kind != KindWebConsoleSession || liveness.State != OwnerAlive {
		return owner, false
	}
	origin := nonEmpty(liveness.Origin, OriginUnspecified)
	if owner.Evidence.WebConsoleOrigin == origin && owner.Evidence.WebConsoleOwner == liveness.WebConsoleOwner {
		return owner, false
	}
	owner.Evidence.WebConsoleOrigin = origin
	owner.Evidence.WebConsoleOwner = liveness.WebConsoleOwner
	return owner, true
}

// HostWebConsoleRoot reports whether a web-console state root belongs to the
// host's own (live) web-console instance: under the Vrooli runtime state
// directory and not a variant's private root. A desktop build of web-console
// keeps its sessions under its app data directory and a variant under
// variants/<name>; neither is answered by the host instance, so a session
// there must stay unknown rather than look archived.
func HostWebConsoleRoot(stateRoot, runtimeStateDir string) bool {
	stateRoot, runtimeStateDir = cleanPath(stateRoot), cleanPath(runtimeStateDir)
	if stateRoot == "" || runtimeStateDir == "" {
		return false
	}
	rel, err := filepath.Rel(runtimeStateDir, stateRoot)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return false
	}
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if part == "variants" {
			return false
		}
	}
	return true
}

func nonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}
