package cliutil

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"os/exec"
	"strconv"
	"strings"
)

// Every coding-agent session is born inside a ceiling. On Linux the ceiling
// is a transient scope under vrooli-agents.slice, the user-manager slice the
// agent_session_containment safeguard converges; the scope inherits the
// slice's memory ceiling and carries its own task and CPU share so one
// session cannot take the whole slice. Off Linux the same numbers apply per
// session (rlimit shim, Job Object).
//
// The primitive lives in platform-go. cli-core does not import it: 244
// modules replace cli-core, and a new import here would need a go.mod change
// in every one of them (the 2026-08 api-core incident). The binaries that
// launch agents (cmd/vrooli, cmd/vrooli-agent-launcher) register a
// SessionContainer instead; without one a launch is uncontained and says so.
// Containment never blocks a launch.
const (
	// AgentSlice is the user-manager slice every session scope lives under.
	AgentSlice = "vrooli-agents.slice"
	// agentScopePrefix names a session scope: vrooli-agent-<id>.
	agentScopePrefix = "vrooli-agent-"
	// Defaults mirror the safeguard's D3 config; used when the slice cannot
	// be read so a host without the safeguard still gets a ceiling.
	defaultAgentCPUWeight  = 50
	defaultAgentMemoryHigh = "50%"
	defaultAgentMemoryMax  = "60%"
	defaultAgentTasksMax   = 4096
	maxScopeNameLength     = 200

	ContainmentSourceSlice    = "slice"
	ContainmentSourceDefaults = "defaults"
	// ContainmentMethodNone is reported when no ceiling was applied.
	ContainmentMethodNone = "none"
)

// SessionContainment is the ceiling a session is born inside, in platform-go's
// vocabulary (systemd syntax for memory: "50%", "8G").
type SessionContainment struct {
	Slice      string
	CPUWeight  int
	MemoryHigh string
	MemoryMax  string
	TasksMax   int
}

// ContainedSession is what a container reports back: the scope ref string
// ("cgroup:/…", "pgid:N", "job:N", "none") and the method used.
type ContainedSession struct {
	Scope  string
	Method string
}

// SessionProcess is the process a container starts and waits for.
type SessionProcess struct {
	Path   string
	Args   []string
	Env    []string
	Dir    string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// SessionContainer applies the ceiling. Run starts the process inside the
// scope and waits for it, returning the session and the process's exit error;
// a container that could not set the ceiling up returns ErrUncontained-shaped
// errors from Run BEFORE starting, so the launcher can fall back. ContainSelf
// moves the calling process into the scope for the exec-replace branch.
type SessionContainer interface {
	Run(ctx context.Context, scope string, containment SessionContainment, process SessionProcess) (ContainedSession, error)
	ContainSelf(scope string, containment SessionContainment) (ContainedSession, error)
}

// DefaultSessionContainer is registered by the binaries that carry platform-go.
var DefaultSessionContainer SessionContainer

// RegisterSessionContainer installs the platform implementation.
func RegisterSessionContainer(container SessionContainer) { DefaultSessionContainer = container }

// agentContainmentFn is the seam tests replace; production reads the slice.
var agentContainmentFn = readAgentContainment

// readAgentContainment reads the ceilings the live slice carries so the
// session scope agrees with what the safeguard converged. When the slice is
// absent or unreadable the D3 defaults apply to the scope directly.
func readAgentContainment() (SessionContainment, string) {
	defaults := SessionContainment{
		CPUWeight: defaultAgentCPUWeight, MemoryHigh: defaultAgentMemoryHigh, MemoryMax: defaultAgentMemoryMax,
		TasksMax: defaultAgentTasksMax, Slice: AgentSlice,
	}
	systemctl, err := exec.LookPath("systemctl")
	if err != nil {
		return defaults, ContainmentSourceDefaults
	}
	output, err := exec.Command(systemctl, "--user", "show", AgentSlice, "-p", "ActiveState", "-p", "CPUWeight", "-p", "TasksMax").Output()
	if err != nil {
		return defaults, ContainmentSourceDefaults
	}
	values := map[string]string{}
	for _, line := range strings.Split(string(output), "\n") {
		if key, value, ok := strings.Cut(strings.TrimSpace(line), "="); ok {
			values[key] = value
		}
	}
	if values["ActiveState"] != "active" {
		return defaults, ContainmentSourceDefaults
	}
	// The slice bounds memory for every session together; the scope only
	// needs a task ceiling and a share so one session cannot take the slice.
	fromSlice := SessionContainment{Slice: AgentSlice}
	if weight, err := strconv.Atoi(values["CPUWeight"]); err == nil && weight > 0 {
		fromSlice.CPUWeight = weight
	}
	if tasks, err := strconv.Atoi(values["TasksMax"]); err == nil && tasks > 0 {
		fromSlice.TasksMax = tasks
	}
	return fromSlice, ContainmentSourceSlice
}

// agentScopeName mints the scope unit name for a session from the run id,
// else the harness session id, else a launcher-minted id, folded to the
// characters a systemd unit name accepts.
func agentScopeName(runID, sessionID string) string {
	id := strings.TrimSpace(runID)
	if id == "" {
		id = strings.TrimSpace(sessionID)
	}
	if id == "" {
		var raw [8]byte
		if _, err := rand.Read(raw[:]); err == nil {
			id = hex.EncodeToString(raw[:])
		}
	}
	var b strings.Builder
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.', r == ':':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	name := agentScopePrefix + b.String()
	if len(name) > maxScopeNameLength {
		name = name[:maxScopeNameLength]
	}
	return name
}

// containmentReport is what a launch records about its ceiling.
type containmentReport struct {
	Scope   string
	Method  string
	Source  string
	Failure string
}

// UncontainedError is returned by a SessionContainer's Run when the ceiling
// could not be set up and the process was NOT started; the launcher then
// runs the process uncontained and records why.
type UncontainedError struct{ Err error }

func (e *UncontainedError) Error() string { return "session uncontained: " + e.Err.Error() }
func (e *UncontainedError) Unwrap() error { return e.Err }

// runContainedChild is the native spawn branch: the child starts inside its
// scope through the registered container. Without a container, or when the
// container cannot set the ceiling up, the child still runs and the report
// says so.
func runContainedChild(request AgentLaunchRequest, argv0, scope string, containment SessionContainment, source string, report *containmentReport) ChildRunner {
	return func(ctx context.Context, path string, args, environment []string, stdin io.Reader, stdout, stderr io.Writer) error {
		report.Source = source
		report.Method = ContainmentMethodNone
		container := DefaultSessionContainer
		if container == nil {
			report.Failure = "no session container registered in this binary"
			return runNativeChild(request, argv0)(ctx, path, args, environment, stdin, stdout, stderr)
		}
		session, err := container.Run(ctx, scope, containment, SessionProcess{Path: path, Args: args, Env: environment, Dir: request.WorkingDir, Stdin: stdin, Stdout: stdout, Stderr: stderr})
		var uncontained *UncontainedError
		if errors.As(err, &uncontained) {
			report.Failure = uncontained.Err.Error()
			log.Printf("agent launch uncontained agent=%s scope=%s: %v", request.Agent, scope, uncontained.Err)
			return runNativeChild(request, argv0)(ctx, path, args, environment, stdin, stdout, stderr)
		}
		report.Scope, report.Method = session.Scope, session.Method
		return err
	}
}

// containSelf is the exec-replace branch: the launcher moves its own pid
// into the scope, then the exec that follows keeps the ceiling.
func containSelf(request AgentLaunchRequest, scope string, containment SessionContainment, source string, report *containmentReport) {
	report.Source = source
	report.Method = ContainmentMethodNone
	if DefaultSessionContainer == nil {
		report.Failure = "no session container registered in this binary"
		return
	}
	session, err := DefaultSessionContainer.ContainSelf(scope, containment)
	report.Scope, report.Method = session.Scope, session.Method
	if err != nil {
		report.Failure = err.Error()
		log.Printf("agent launch uncontained agent=%s scope=%s: %v", request.Agent, scope, err)
	}
}
