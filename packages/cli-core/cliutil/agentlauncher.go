package cliutil

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/vrooli/envkit-go"
)

const (
	// AgentManagerLauncherBaseEnv is the explicit endpoint override used by
	// the standalone launcher. It is intentionally separate from generic API
	// base variables so a child agent cannot redirect unrelated CLI traffic.
	AgentManagerLauncherBaseEnv = "AGENT_MANAGER_API_BASE"
	// AgentManagerLauncherTimeoutEnv is an optional operator tuning knob for
	// unusually busy local stores. The default remains deliberately short.
	AgentManagerLauncherTimeoutEnv = "AGENT_MANAGER_ATTACH_TIMEOUT"

	// AgentManagerIdentityTokenEnv is the only environment key the launcher
	// adds to a child process after a successful attach.
	AgentManagerIdentityTokenEnv = EnvIdentityToken

	// HumanOwnerDelegationEnv carries a short-lived, session-scoped human-owner
	// delegation that Web Console injects into a verified-human terminal session
	// (minted by scenario-authenticator). The launcher does not mint or verify
	// it; it only forwards the inherited value on attach so agent-manager can
	// verify it and attribute the attached run to the operator. It is generic to
	// every coding agent and is scrubbed-on-adopt like the identity token so a
	// nested launch never re-forwards a stale delegation for a different session.
	HumanOwnerDelegationEnv = "WC_HUMAN_OWNER_DELEGATION"

	// HumanOwnerDelegationHeader carries the delegation on the attach request.
	// agent-manager reads it, verifies the JWT against the scenario-authenticator
	// root, and attributes the attached run only on success. Shared so the
	// launcher (sets) and agent-manager (reads) cannot drift.
	HumanOwnerDelegationHeader = "X-Human-Owner-Delegation"

	// WebConsoleSessionIDHeader carries the web-console session id on the attach
	// request. The human-owner delegation's wc_session_id claim is the web-console
	// session id, NOT the harness/coding-agent session id, so agent-manager binds
	// the delegation against this value. Not a secret (it is also in the child
	// env as WC_WEB_CONSOLE_SESSION_ID for attribution); it only correlates the
	// delegation to the session that minted it so a delegation cannot be replayed
	// onto a different session's attach. Shared so launcher (sets) and
	// agent-manager (reads/binds) cannot drift.
	WebConsoleSessionIDHeader = "X-Wc-Session-Id"

	// AgentSessionEnv marks an environment that is already inside a governed
	// coding-agent session. A launch reaches the agent through more than one
	// launcher stage — the operator's `vrooli agent launch` execs the shim,
	// and the shim is this same binary — and exec keeps the pid, so without a
	// marker every stage records its own lease and mints its own scope for
	// one session. On 2026-09-04 that produced two leases 36 ms apart for
	// each claude session, under two spellings of the agent name, and left
	// the first stage's scope named in a lease after systemd had reaped the
	// empty cgroup: a session `vrooli agent list` showed and the kernel had
	// never heard of.
	AgentSessionEnv = "VROOLI_AGENT_SESSION"

	defaultAgentManagerLauncherBase = "http://127.0.0.1:18800"

	// defaultAgentManagerAttachTimeout bounds the attribution call only.
	//
	// The budget can afford to be generous because of which failure it actually
	// bounds. When agent-manager is not running, the connection is refused
	// immediately and none of this timeout is spent; it is only consumed when
	// agent-manager is up but slow, which is exactly the case worth waiting for.
	// Against that, a coding agent takes seconds to reach its first prompt, so a
	// ceiling in this range is imperceptible.
	//
	// It was 300ms, which was under the real cost: attaching to a local
	// agent-manager measured 350-440ms across repeated samples on a developer
	// host under load, so every attach timed out and every agent ran
	// unattributed. Nothing failed loudly, because attribution is deliberately
	// fail-open — the shims installed correctly and simply never carried an
	// identity. Operators tune with AGENT_MANAGER_ATTACH_TIMEOUT.
	defaultAgentManagerAttachTimeout = 2 * time.Second
)

// AgentLaunchRequest describes one native coding-agent invocation. Args are
// passed as typed argv tokens; this package never accepts or constructs a
// shell command string.
type AgentLaunchRequest struct {
	Agent string
	// RequireNativeIdentity opts into protected startup: live broker refusal or
	// unavailable custody stops before attachment, lease and child execution.
	RequireNativeIdentity bool
	TaskID                string
	Args                  []string
	Environment           []string
	APIBase               string
	WorkingDir            string
	// Claims are advisory paths this session says it will edit; a live
	// session holding an overlapping claim is named at launch and the
	// launch continues (D8: advisory in this release).
	Claims []string

	// AttachTimeout bounds only the optional attribution request. It never
	// bounds or cancels the coding agent itself.
	AttachTimeout time.Duration

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	// Test seams keep fallback behavior deterministic without starting a real
	// coding agent or requiring a running Agent Manager.
	LookPath              func(string) (string, error)
	LookPathInEnvironment func(string, []string) (string, error)
	RunChild              ChildRunner
	HTTPClient            *http.Client
}

// ChildRunner starts the already-resolved executable with the exact args and
// environment selected by the launcher.
type ChildRunner func(ctx context.Context, path string, args, environment []string, stdin io.Reader, stdout, stderr io.Writer) error

// AgentLaunchError identifies launcher input or executable failures. Attach
// failures are deliberately not returned: attribution is best effort and
// must never prevent an otherwise valid agent launch.
type AgentLaunchError struct {
	Agent string
	Err   error
}

func (e *AgentLaunchError) Error() string {
	if e == nil {
		return "agent launch failed"
	}
	return fmt.Sprintf("launch %s: %v", e.Agent, e.Err)
}

func (e *AgentLaunchError) Unwrap() error { return e.Err }

// LaunchResult reports which attribution tier was reached. Tier 1 is an
// Agent Manager attachment; tier 4 is an ungoverned local launch.
type LaunchResult struct {
	Tier          string
	AttachFailure string
	// Scope is the platform-go ScopeRef of the session's ceiling ("cgroup:/…",
	// "pgid:N", "job:N", "none") and ContainmentMethod how it was applied
	// (systemd-run, transient-unit, cgroup-write, rlimit-shim, job, none).
	// ContainmentSource says whether the ceilings came from the converged
	// vrooli-agents.slice or the launcher's defaults; ContainmentFailure is
	// why a session runs uncontained. Phase 10's editor lease stores them.
	//
	// ContainmentUnsupported distinguishes the two meanings of an empty
	// ceiling: true is a platform with no primitive (permanent and expected),
	// false with a non-empty ContainmentFailure is a host that should have
	// contained the session and did not. Only the second is a regression.
	Scope                  string
	ContainmentMethod      string
	ContainmentSource      string
	ContainmentFailure     string
	ContainmentUnsupported bool
	// LeaseRecorded says whether the session's editor lease reached the
	// runtime registry; ClaimOverlaps names live holders of overlapping
	// claims (advisory).
	LeaseRecorded bool
	ClaimOverlaps []ClaimHolder
	// CeilingWarning is why the containment ceiling was about to refuse work,
	// when it was; empty when the ceiling had room or could not be read.
	CeilingWarning string
}

// errorWriter is where a launcher diagnostic goes: the request's stderr, else
// the process's. It is never the agent's stdout, which the operator may be
// piping somewhere that expects only the agent's output.
func errorWriter(request AgentLaunchRequest) io.Writer {
	if request.Stderr != nil {
		return request.Stderr
	}
	return os.Stderr
}

var ungovernedLaunches atomic.Uint64

func UngovernedLaunchCount() uint64 { return ungovernedLaunches.Load() }

// LaunchCodingAgent resolves one supported coding-agent binary, makes a
// bounded best-effort attach, starts the child, and quietly best-effort
// detaches it after the child exits. Attach/detach failures never alter child
// execution or its exit status.
func LaunchCodingAgent(ctx context.Context, request AgentLaunchRequest) error {
	_, err := LaunchCodingAgentResult(ctx, request)
	return err
}

// LaunchCodingAgentResult is LaunchCodingAgent with an observability result.
// Attribution remains fail-open so a missing Agent Manager never prevents the
// underlying coding agent from starting.
// LaunchCommandSessionResult runs typed command argv under a protected, zero-scope
// external-session identity. It starts no coding agent and makes no assertion about
// privileged identity of its surrounding executor. The wrapper waits and detaches.
// Args includes the executable as its first element; Agent is ignored.
func LaunchCommandSessionResult(ctx context.Context, request AgentLaunchRequest) (LaunchResult, error) {
	if len(request.Args) == 0 || strings.TrimSpace(request.Args[0]) == "" {
		return LaunchResult{}, errors.New("command argv required")
	}
	if task := request.TaskID; task != "" {
		raw, err := hex.DecodeString(strings.ReplaceAll(task, "-", ""))
		if len(task) != 36 || task[8] != '-' || task[13] != '-' || task[18] != '-' || task[23] != '-' || err != nil || len(raw) != 16 || strings.Trim(strings.ReplaceAll(task, "-", ""), "0") == "" {
			return LaunchResult{}, errors.New("existing task UUID required")
		}
	}
	binary := request.Args[0]
	request.Args = append([]string(nil), request.Args[1:]...)
	request.Agent = "command-session"
	request.RequireNativeIdentity = true
	return launchNativeSession(ctx, request, binary)
}

func LaunchCodingAgentResult(ctx context.Context, request AgentLaunchRequest) (LaunchResult, error) {
	return launchNativeSession(ctx, request, "")
}

func launchNativeSession(ctx context.Context, request AgentLaunchRequest, command string) (LaunchResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	launchContext, err := ResolveLaunchContext(LaunchContextRequest{
		WorkingDir:  request.WorkingDir,
		Environment: request.Environment,
	})
	if err != nil {
		return LaunchResult{}, &AgentLaunchError{Agent: request.Agent, Err: err}
	}
	request.WorkingDir = launchContext.WorkingDir
	request.Environment = PrepareLaunchEnvironment(request.Environment, launchContext)
	harnessKind, binary := "command-session", command
	if command == "" {
		harnessKind, binary, err = codingAgentSpec(request.Agent)
		if err != nil {
			return LaunchResult{}, &AgentLaunchError{Agent: request.Agent, Err: err}
		}
	}

	var path string
	if command != "" && strings.ContainsRune(binary, '/') {
		path = binary
		if !filepath.IsAbs(path) {
			path = filepath.Join(request.WorkingDir, path)
		}
		if !isExecutableFile(path) {
			err = fmt.Errorf("command is not executable")
		}
	} else if request.LookPathInEnvironment != nil {
		path, err = request.LookPathInEnvironment(binary, request.Environment)
	} else if request.LookPath != nil {
		path, err = request.LookPath(binary)
	} else {
		path, err = lookPathInEnvironment(binary, request.Environment)
	}
	if err != nil {
		return LaunchResult{}, &AgentLaunchError{Agent: request.Agent, Err: fmt.Errorf("resolve %q: %w", binary, err)}
	}

	environment := append([]string(nil), request.Environment...)
	if environment == nil {
		environment = os.Environ()
	}
	// Coding-agent children are an explicit delegation boundary. Preserve only
	// the identity credential that is intentionally delegated and drop scenario
	// ownership/session observations before any launcher-owned overlay is added.
	environment = []string(envkit.WithOverlay(envkit.Env(environment), envkit.DelegatedAgent, nil))
	// The agent's shell inherits the build-width floor, so a hand-typed
	// `go build` inside a session is polite by inheritance on both the exec
	// and the spawn branch.
	environment = []string(envkit.Toolchain(envkit.Env(environment), envkit.ToolchainOptions{}))
	environment = PrepareWebConsoleAgentHome(request.Agent, environment)
	// The overlay drops scenario observations, so the marker is read from the
	// inherited environment before it and re-applied after.
	if inherited := environmentValue(request.Environment, AgentSessionEnv); inherited != "" {
		environment = withEnvironmentValue(environment, AgentSessionEnv, inherited)
	} else if inherited := os.Getenv(AgentSessionEnv); inherited != "" {
		environment = withEnvironmentValue(environment, AgentSessionEnv, inherited)
	}

	// Protected startup consults exactly the ordinary authority/socket first.
	// Never turn a responding refusal into inherited or attachment fallback.
	protectedToken := ""
	if request.RequireNativeIdentity {
		protectedToken, err = ResolveAgentCredential(ctx)
		if err != nil {
			return LaunchResult{}, &AgentLaunchError{Agent: request.Agent, Err: err}
		}
		if protectedToken != "" {
			environment = withEnvironmentValue(environment, AgentManagerIdentityTokenEnv, protectedToken)
		}
		if environmentValue(environment, AgentSessionEnv) != "" && protectedToken == "" {
			return LaunchResult{}, &AgentLaunchError{Agent: request.Agent, Err: errors.New("existing session has no native identity")}
		}
	}

	// A question asked of the binary is not a session. Answering it needs no
	// run row, no lease and no scope, and creating them costs a D-Bus round
	// trip and a cgroup each time. On 2026-09-05 a probe loop asking all five
	// agents `--version` every 15 seconds was minting roughly 29,000 scopes
	// and 29,000 lease rows a day, which filled `vrooli agent list` with
	// sessions that never existed and made the agent slice's occupancy
	// unreadable.
	if command == "" && informationalInvocation(request.Args) {
		if request.RequireNativeIdentity && protectedToken == "" {
			return LaunchResult{}, &AgentLaunchError{Agent: request.Agent, Err: errors.New("existing native credential required for protected informational invocation")}
		}
		return runUngovernedProbe(ctx, request, path, binary, environment)
	}

	// A later launcher stage inherits the session the first one opened. It
	// must not open a second: one session is one lease, one scope and one run
	// row, whichever stage of the chain the operator's command entered at.
	if inherited := environmentValue(environment, AgentSessionEnv); inherited != "" {
		if request.RequireNativeIdentity {
			current, custodyErr := resolveAgentCredential(ctx, AgentCredentialSocket(), protectedToken)
			if custodyErr != nil {
				return LaunchResult{}, &AgentLaunchError{Agent: request.Agent, Err: custodyErr}
			}
			if current != protectedToken {
				return LaunchResult{}, &AgentLaunchError{Agent: request.Agent, Err: errors.New("native credential changed during launch")}
			}
		}
		if command != "" && request.RunChild == nil {
			// An in-task command wrapper waits even when it adopts an existing session.
			request.RunChild = runNativeChild(request, binary)
		}
		return runInheritedSession(ctx, request, path, binary, environment, inherited)
	}

	// Deciding this before attaching matters: a launch that replaces its own
	// process image has no "after" in which to detach, so it reports its pid at
	// attach time instead. Because exec keeps the pid, the pid reported here is
	// the agent's pid, and agent-manager's reconciler closes the run by
	// process liveness (see attachedRunProcessAlive).
	willExec := command == "" && request.RunChild == nil && execReplaceSupported && stdioIsInherited(request)

	// The scope is named from the harness session id before attachment, so the
	// lease and cgroup use a stable name whether or not a run id is minted.
	scope := agentScopeName("", harnessSessionID(harnessKind))
	containment, containmentSource := agentContainmentFn()
	var report containmentReport
	// Process-bound grants are Linux-only. Other platforms retain their child
	// container path (for example Darwin needs a fresh child process group).
	commandCallerBoundary := command != "" && runtime.GOOS == "linux"
	if commandCallerBoundary && protectedToken == "" {
		// The waiting command wrapper is the approval caller. Establish its final
		// boundary before attachment and lease registration; its children must
		// inherit that same boundary for process-bound credential delivery.
		containSelf(request, scope, containment, containmentSource, &report)
	}
	var attach codingAgentAttachment
	if request.RequireNativeIdentity && protectedToken != "" {
		attach = codingAgentAttachment{token: protectedToken}
	} else {
		attach = attachCodingAgent(ctx, request, harnessKind, willExec || command != "")
	}
	if request.RequireNativeIdentity {
		if attach.token == "" {
			if attach.err != nil {
				return LaunchResult{}, &AgentLaunchError{Agent: request.Agent, Err: attach.err}
			}
			return LaunchResult{}, &AgentLaunchError{Agent: request.Agent, Err: errors.New("native identity attachment unavailable: " + attach.failure)}
		}
		// Reconsult the same live authority with the privately held candidate. This
		// cannot use another namespace or inherited leaf to evade a broker denial.
		current, custodyErr := resolveAgentCredential(ctx, AgentCredentialSocket(), attach.token)
		if custodyErr != nil || current != attach.token {
			if custodyErr == nil {
				custodyErr = errors.New("native credential changed during launch")
			}
			if attach.runID != "" {
				if cleanupErr := detachCodingAgent(context.Background(), request, attach.runID, attach.token); cleanupErr != nil {
					custodyErr = fmt.Errorf("%w; owned attachment %s cleanup unconfirmed: %v", custodyErr, attach.runID, cleanupErr)
				}
			}
			return LaunchResult{}, &AgentLaunchError{Agent: request.Agent, Err: custodyErr}
		}
	}
	launchResult := LaunchResult{Tier: "tier-4"}
	// The session is visible for as long as it lives: the lease carries the
	// tree, the scope and the claims; the registry expires it only on proof
	// of death. The exec branch cannot heartbeat (the launcher becomes the
	// agent), so it relies on that expiry alone.
	launchResult.ClaimOverlaps = reportClaimOverlaps(ctx, request)
	stopLease, recorded := recordEditorLease(ctx, EditorLeaseRecord{
		SessionID: sessionIDForLease(attach.runID, harnessKind), Harness: harnessKind, Agent: request.Agent, PID: os.Getpid(),
		WorkingDir: launchWorkingDir(request), Scope: scope, Claims: request.Claims,
	}, !willExec)
	launchResult.LeaseRecorded = recorded
	defer stopLease("session ended")
	environment = withEnvironmentValue(environment, AgentSessionEnv, scope)
	// Single-use: the human-owner delegation was consumed by the attach call
	// above (forwarded on its header). It is a short-lived operator credential and
	// must never reach the spawned agent's environment or persist for reuse, so
	// strip it from what the child inherits regardless of whether the attach
	// succeeded. The agent is attributed by the run the attach created, not by
	// holding the delegation.
	environment = withoutEnvironmentValue(environment, HumanOwnerDelegationEnv)
	// The ceiling refuses silently; the agent that trips over it reports the
	// refusal in its own words. Say the real reason before it does.
	if warning := ceilingWarning(); warning != "" {
		launchResult.CeilingWarning = warning
		fmt.Fprintln(errorWriter(request), "vrooli: "+warning)
	}
	if attach.token != "" {
		environment = withEnvironmentValue(environment, AgentManagerIdentityTokenEnv, attach.token)
		launchResult.Tier = "tier-1"
	} else {
		ungovernedLaunches.Add(1)
		launchResult.AttachFailure = attach.failure
		if launchResult.AttachFailure == "" {
			launchResult.AttachFailure = "agent-manager attachment unavailable"
		}
		log.Printf("agent launch attribution degraded agent=%s base=%s", request.Agent, agentManagerLauncherBase(request))
	}

	if willExec {
		// The launcher moves itself into the session scope first; exec keeps
		// the pid and the ceiling, so the agent is born contained.
		containSelf(request, scope, containment, containmentSource, &report)
		// Returns only when the exec itself failed. That is a launcher problem,
		// never the operator's, so fall through to spawn-and-wait rather than
		// refusing to start the agent.
		_ = execReplace(path, append([]string{binary}, request.Args...), environment, request.WorkingDir)
	}

	runChild := request.RunChild
	if runChild == nil {
		if commandCallerBoundary {
			// Command children inherit the wrapper's final boundary, including
			// when containment is unavailable or existing custody was adopted.
			runChild = runNativeChild(request, binary)
		} else {
			runChild = runContainedChild(request, binary, scope, containment, containmentSource, &report)
		}
	}
	err = runChild(ctx, path, append([]string(nil), request.Args...), environment, request.Stdin, request.Stdout, request.Stderr)
	if attach.runID != "" {
		if cleanupErr := detachCodingAgent(context.Background(), request, attach.runID, attach.token); cleanupErr != nil && request.RequireNativeIdentity {
			err = errors.Join(err, fmt.Errorf("owned attachment %s cleanup unconfirmed: %w", attach.runID, cleanupErr))
		}
	}
	launchResult.Scope, launchResult.ContainmentMethod = report.Scope, report.Method
	launchResult.ContainmentSource, launchResult.ContainmentFailure = report.Source, report.Failure
	launchResult.ContainmentUnsupported = report.Unsupported
	return launchResult, err
}

func lookPathInEnvironment(binary string, environment []string) (string, error) {
	for _, dir := range filepath.SplitList(environmentValue(environment, "PATH")) {
		if strings.TrimSpace(dir) == "" {
			dir = "."
		}
		for _, candidate := range executableCandidates(filepath.Join(dir, binary)) {
			if isExecutableFile(candidate) {
				return candidate, nil
			}
		}
	}
	return "", fmt.Errorf("resolve %q: not found on PATH", binary)
}

// informationalFlags are the arguments that make a coding agent answer and
// exit rather than start a session. The set is deliberately closed: anything
// unrecognised is a session, so a new agent flag is governed until someone
// decides otherwise.
var informationalFlags = map[string]bool{
	"--version": true, "-v": true, "-V": true,
	"--help": true, "-h": true,
	"doctor": true,
}

// informationalInvocation reports whether argv only asks the binary a
// question. Every argument must be informational: `codex --version` is a
// probe, `codex --help-me-write-this` is not, and neither is an empty argv,
// which starts an interactive session.
func informationalInvocation(args []string) bool {
	if len(args) == 0 {
		return false
	}
	for _, arg := range args {
		if !informationalFlags[strings.TrimSpace(arg)] {
			return false
		}
	}
	return true
}

// runUngovernedProbe answers the question and gets out of the way: no
// attachment, no lease, no scope, no ceiling. The result says so, so a caller
// that expected a governed session can tell the difference.
func runUngovernedProbe(ctx context.Context, request AgentLaunchRequest, path, binary string, environment []string) (LaunchResult, error) {
	result := LaunchResult{Tier: "tier-4", AttachFailure: "informational invocation: not a session"}
	runChild := request.RunChild
	if runChild == nil {
		runChild = runNativeChild(request, binary)
	}
	return result, runChild(ctx, path, append([]string(nil), request.Args...), environment, request.Stdin, request.Stdout, request.Stderr)
}

// environmentValue reads one key from an environment slice; an empty result
// means absent, which is what an unmarked environment looks like.
func environmentValue(environment []string, key string) string {
	for _, entry := range environment {
		name, value, ok := strings.Cut(entry, "=")
		if ok && environmentKeyEqual(name, key) {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// runInheritedSession runs the agent inside the session an earlier launcher
// stage already opened: no second lease, no second scope, no second run row.
// The ceiling is already on this process, and exec keeps it.
func runInheritedSession(ctx context.Context, request AgentLaunchRequest, path, binary string, environment []string, scope string) (LaunchResult, error) {
	result := LaunchResult{Tier: "tier-4", Scope: scope, AttachFailure: "inherited an open session from an earlier launcher stage"}
	if request.RequireNativeIdentity {
		result.Tier, result.AttachFailure = "tier-1", ""
	}
	if request.RunChild == nil && execReplaceSupported && stdioIsInherited(request) {
		_ = execReplace(path, append([]string{binary}, request.Args...), environment, request.WorkingDir)
	}
	runChild := request.RunChild
	if runChild == nil {
		runChild = runNativeChild(request, binary)
	}
	return result, runChild(ctx, path, append([]string(nil), request.Args...), environment, request.Stdin, request.Stdout, request.Stderr)
}

// sessionIDForLease is the run id when agent-manager minted one, else the
// harness session id the launcher already reports; the scope name is derived
// from the same choice so the lease and the cgroup agree.
func sessionIDForLease(runID, harnessKind string) string {
	if id := strings.TrimSpace(runID); id != "" {
		return id
	}
	return harnessSessionID(harnessKind)
}

// launchWorkingDir is the already-normalized tree the session edits. Launches
// pass through ResolveLaunchContext before attribution, so an empty value here
// would be a launcher bug rather than an expected fallback.
func launchWorkingDir(request AgentLaunchRequest) string {
	return strings.TrimSpace(request.WorkingDir)
}

// stdioIsInherited reports whether the request's streams are exactly this
// process's own standard streams. Only then is replacing the process image
// equivalent to spawning: exec inherits file descriptors and cannot copy
// bytes through an arbitrary io.Reader or io.Writer, so a caller that supplied
// a buffer or a pipe must keep the spawn-and-wait path.
func stdioIsInherited(request AgentLaunchRequest) bool {
	stdinOK := request.Stdin == nil
	if file, ok := request.Stdin.(*os.File); ok && file == os.Stdin {
		stdinOK = true
	}
	stdoutOK := request.Stdout == nil
	if file, ok := request.Stdout.(*os.File); ok && file == os.Stdout {
		stdoutOK = true
	}
	stderrOK := request.Stderr == nil
	if file, ok := request.Stderr.(*os.File); ok && file == os.Stderr {
		stderrOK = true
	}
	return stdinOK && stdoutOK && stderrOK
}

type codingAgentAttachment struct {
	runID   string
	token   string
	failure string
	err     error
}

func codingAgentSpec(agent string) (harnessKind, binary string, err error) {
	switch strings.ToLower(strings.TrimSpace(agent)) {
	case "claude", "claude-code":
		return "claude-code", "claude", nil
	case "codex":
		return "codex", "codex", nil
	case "grok":
		return "grok", "grok", nil
	case "opencode":
		return "opencode", "opencode", nil
	case "agy", "antigravity":
		return "antigravity", "agy", nil
	default:
		return "", "", fmt.Errorf("unsupported coding-agent runner %q (supported: claude-code, codex, grok, opencode, antigravity)", agent)
	}
}

// attachCodingAgent makes the bounded, best-effort attribution call. Every
// failure path returns a zero attachment: attribution is observability and must
// never decide whether an agent starts.
//
// reportPID says the caller is about to replace its process image, so this
// process's pid will still be the agent's pid afterwards. Sending it turns
// agent-manager's liveness check into a single stat of /proc/<pid> instead of a
// scan of every process environment.
func attachCodingAgent(parent context.Context, request AgentLaunchRequest, harnessKind string, reportPID bool) codingAgentAttachment {
	// An identity already in the environment means something else already owns a
	// run for this work. agent-manager mints one per run and scrubs inherited
	// ones precisely so ownership is unambiguous, so the right move here is to
	// adopt what is already there rather than open a second run for the same
	// process. Without this, launching an agent from inside an agent-manager run
	// produces two run records for one agent.
	if inherited := strings.TrimSpace(os.Getenv(AgentManagerIdentityTokenEnv)); inherited != "" {
		return codingAgentAttachment{token: inherited}
	}

	base := agentManagerLauncherBase(request)
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return codingAgentAttachment{failure: "bad base URL"}
	}

	sessionID := harnessSessionID(harnessKind)
	body := map[string]any{
		"harness_kind":       harnessKind,
		"harness_session_id": sessionID,
	}
	if taskID := strings.TrimSpace(request.TaskID); taskID != "" {
		body["task_id"] = taskID
	}
	if reportPID {
		body["process_id"] = os.Getpid()
	}
	// Tree and containment metadata belong to the editor lease. The attach API
	// accepts only harness identity, optional task, PID and title; adding lease
	// fields here causes strict request decoding to reject the entire attach.
	payload, err := json.Marshal(body)
	if err != nil {
		return codingAgentAttachment{failure: "malformed request"}
	}

	timeout := launcherAttachTimeout(request.AttachTimeout)
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/v1/runs/attach", strings.NewReader(string(payload)))
	if err != nil {
		return codingAgentAttachment{failure: "bad base URL"}
	}
	req.Header.Set("Content-Type", "application/json")
	// Forward the inherited human-owner delegation (if any) so agent-manager can
	// verify it and attribute the attached run to the operator. A label only: a
	// missing or unverifiable value leaves the run unattributed, never refused.
	// Generic for every coding agent — this is the one shared launcher path.
	if delegation := strings.TrimSpace(os.Getenv(HumanOwnerDelegationEnv)); delegation != "" {
		req.Header.Set(HumanOwnerDelegationHeader, delegation)
		// The delegation's wc_session_id claim is the web-console session id; forward
		// it so agent-manager binds the delegation against the right identifier (the
		// harness session id differs). Not a secret — purely a correlation id.
		if wcSession := strings.TrimSpace(os.Getenv(webConsoleSessionIDEnv)); wcSession != "" {
			req.Header.Set(WebConsoleSessionIDHeader, wcSession)
		}
	}
	client := request.HTTPClient
	if client == nil {
		client = &http.Client{}
	}
	response, err := client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return codingAgentAttachment{failure: "timeout", err: err}
		}
		return codingAgentAttachment{failure: "agent-manager unreachable", err: err}
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return codingAgentAttachment{failure: "non-2xx response"}
	}
	var result struct {
		Run struct {
			ID string `json:"id"`
		} `json:"run"`
		IdentityToken string `json:"identity_token"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return codingAgentAttachment{failure: "malformed response"}
	}
	attachment := codingAgentAttachment{runID: strings.TrimSpace(result.Run.ID), token: strings.TrimSpace(result.IdentityToken)}
	if attachment.token == "" {
		attachment.failure = "malformed response"
	}
	return attachment
}

func detachCodingAgent(parent context.Context, request AgentLaunchRequest, runID, token string) error {
	base := agentManagerLauncherBase(request)
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return err
	}
	body, err := json.Marshal(map[string]string{"run_id": runID, "reason": "coding agent exited"})
	if err != nil {
		return err
	}
	timeout := launcherAttachTimeout(request.AttachTimeout)
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/v1/runs/"+url.PathEscape(runID)+"/detach", strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if request.RequireNativeIdentity {
		if token == "" {
			return errors.New("held native credential required for owned cleanup")
		}
		pin := &agentCredentialPin{token: token, resolve: func(ctx context.Context) (string, error) {
			return resolveAgentCredential(ctx, AgentCredentialSocket(), token)
		}}
		req = req.WithContext(context.WithValue(req.Context(), agentCredentialPinKey{}, pin))
		req.Header.Set(HeaderAgentIdentityToken, token)
		if err := ApplyAgentIdentity(req); err != nil {
			return err
		}
	}
	client := request.HTTPClient
	if client == nil {
		client = &http.Client{}
	}
	if request.RequireNativeIdentity {
		copyClient := *client
		copyClient.CheckRedirect = func(*http.Request, []*http.Request) error { return errors.New("protected cleanup redirect refused") }
		next := client.Transport
		if next == nil {
			next = http.DefaultTransport
		}
		// Every wire hop retains the pin, including redirect attempts. No alternate
		// transport is selected after denial.
		copyClient.Transport = identityForwardingTransport{next: next}
		client = &copyClient
	}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("detach returned HTTP %d", response.StatusCode)
	}
	return nil
}

func harnessSessionID(harnessKind string) string {
	var environments []string
	if harnessKind != "command-session" {
		environments = harnessSessionEnvironments(harnessKind)
	}
	for _, env := range environments {
		if value := strings.TrimSpace(os.Getenv(env)); value != "" {
			return value
		}
	}
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err == nil {
		return harnessKind + "-" + hex.EncodeToString(raw[:])
	}
	return fmt.Sprintf("%s-%d", harnessKind, time.Now().UTC().UnixNano())
}

func agentManagerLauncherBase(request AgentLaunchRequest) string {
	defaultBase := defaultAgentManagerLauncherBase
	if port := strings.TrimSpace(detectAgentManagerPort()); port != "" {
		defaultBase = "http://127.0.0.1:" + port
	}
	return strings.TrimRight(DetermineAPIBase(APIBaseOptions{
		Override:    request.APIBase,
		EnvVars:     []string{AgentManagerLauncherBaseEnv, "AGENT_MANAGER_API_URL"},
		DefaultBase: defaultBase,
	}), "/")
}

func launcherAttachTimeout(requested time.Duration) time.Duration {
	if requested > 0 {
		return requested
	}
	if configured, err := time.ParseDuration(strings.TrimSpace(os.Getenv(AgentManagerLauncherTimeoutEnv))); err == nil && configured > 0 {
		return configured
	}
	return defaultAgentManagerAttachTimeout
}

func harnessSessionEnvironments(harnessKind string) []string {
	switch harnessKind {
	case "claude-code":
		return []string{EnvClaudeCodeSessionID, "VROOLI_HARNESS_SESSION_ID"}
	case "codex":
		return []string{EnvCodexThreadID, "VROOLI_HARNESS_SESSION_ID"}
	default:
		return []string{"VROOLI_HARNESS_SESSION_ID"}
	}
}

func withEnvironmentValue(environment []string, key, value string) []string {
	result := make([]string, 0, len(environment)+1)
	replaced := false
	for _, entry := range environment {
		name, _, hasValue := strings.Cut(entry, "=")
		if hasValue && environmentKeyEqual(name, key) {
			if !replaced {
				result = append(result, key+"="+value)
				replaced = true
			}
			continue
		}
		result = append(result, entry)
	}
	if !replaced {
		result = append(result, key+"="+value)
	}
	return result
}

// withoutEnvironmentValue returns the environment with every entry for key
// removed. Used to make a single-use credential (the human-owner delegation)
// non-inheritable by the spawned child after it has been consumed.
func withoutEnvironmentValue(environment []string, key string) []string {
	result := make([]string, 0, len(environment))
	for _, entry := range environment {
		name, _, hasValue := strings.Cut(entry, "=")
		if hasValue && environmentKeyEqual(name, key) {
			continue
		}
		result = append(result, entry)
	}
	return result
}

func runNativeChild(request AgentLaunchRequest, argv0 string) ChildRunner {
	return func(ctx context.Context, path string, args, environment []string, stdin io.Reader, stdout, stderr io.Writer) error {
		command := exec.CommandContext(ctx, path, args...)
		// LookPath resolves the executable location, but the requested binary
		// name remains argv[0]. This keeps the native process view unchanged
		// while still avoiding a shell lookup at execution time.
		if len(command.Args) > 0 {
			command.Args[0] = argv0
		}
		command.Env = environment
		command.Stdin = stdin
		command.Stdout = stdout
		command.Stderr = stderr
		command.Dir = request.WorkingDir
		return command.Run()
	}
}

// ChildExitCode returns the native process exit code when err came from an
// exec.Cmd. A negative result means the error was not an exited child.
func ChildExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}
