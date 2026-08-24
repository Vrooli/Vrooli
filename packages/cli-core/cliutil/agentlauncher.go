package cliutil

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
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
	Agent       string
	TaskID      string
	Args        []string
	Environment []string
	APIBase     string

	// AttachTimeout bounds only the optional attribution request. It never
	// bounds or cancels the coding agent itself.
	AttachTimeout time.Duration

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	// Test seams keep fallback behavior deterministic without starting a real
	// coding agent or requiring a running Agent Manager.
	LookPath   func(string) (string, error)
	RunChild   ChildRunner
	HTTPClient *http.Client
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

// LaunchCodingAgent resolves one supported coding-agent binary, makes a
// bounded best-effort attach, starts the child, and quietly best-effort
// detaches it after the child exits. Attach/detach failures never alter child
// execution or its exit status.
func LaunchCodingAgent(ctx context.Context, request AgentLaunchRequest) error {
	if ctx == nil {
		ctx = context.Background()
	}
	harnessKind, binary, err := codingAgentSpec(request.Agent)
	if err != nil {
		return &AgentLaunchError{Agent: request.Agent, Err: err}
	}

	lookPath := request.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	path, err := lookPath(binary)
	if err != nil {
		return &AgentLaunchError{Agent: request.Agent, Err: fmt.Errorf("resolve %q: %w", binary, err)}
	}

	environment := append([]string(nil), request.Environment...)
	if environment == nil {
		environment = os.Environ()
	}

	// Deciding this before attaching matters: a launch that replaces its own
	// process image has no "after" in which to detach, so it reports its pid at
	// attach time instead. Because exec keeps the pid, the pid reported here is
	// the agent's pid, and agent-manager's reconciler closes the run by
	// process liveness (see attachedRunProcessAlive).
	willExec := request.RunChild == nil && execReplaceSupported && stdioIsInherited(request)

	attach := attachCodingAgent(ctx, request, harnessKind, willExec)
	if attach.token != "" {
		environment = withEnvironmentValue(environment, AgentManagerIdentityTokenEnv, attach.token)
	}

	if willExec {
		// Returns only when the exec itself failed. That is a launcher problem,
		// never the operator's, so fall through to spawn-and-wait rather than
		// refusing to start the agent.
		_ = execReplace(path, append([]string{binary}, request.Args...), environment)
	}

	if attach.runID != "" {
		defer func() {
			// Detach is cleanup only. It never signals or kills the child and
			// its failure must not mask the child result. Only the
			// spawn-and-wait path ever reaches this.
			_ = detachCodingAgent(context.Background(), request, attach.runID)
		}()
	}

	runChild := request.RunChild
	if runChild == nil {
		runChild = runNativeChild(request, binary)
	}
	return runChild(ctx, path, append([]string(nil), request.Args...), environment, request.Stdin, request.Stdout, request.Stderr)
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
	runID string
	token string
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
		return codingAgentAttachment{}
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
	payload, err := json.Marshal(body)
	if err != nil {
		return codingAgentAttachment{}
	}

	timeout := launcherAttachTimeout(request.AttachTimeout)
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/v1/runs/attach", strings.NewReader(string(payload)))
	if err != nil {
		return codingAgentAttachment{}
	}
	req.Header.Set("Content-Type", "application/json")
	client := request.HTTPClient
	if client == nil {
		client = &http.Client{}
	}
	response, err := client.Do(req)
	if err != nil {
		return codingAgentAttachment{}
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return codingAgentAttachment{}
	}
	var result struct {
		Run struct {
			ID string `json:"id"`
		} `json:"run"`
		IdentityToken string `json:"identity_token"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return codingAgentAttachment{}
	}
	return codingAgentAttachment{runID: strings.TrimSpace(result.Run.ID), token: strings.TrimSpace(result.IdentityToken)}
}

func detachCodingAgent(parent context.Context, request AgentLaunchRequest, runID string) error {
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
	client := request.HTTPClient
	if client == nil {
		client = &http.Client{}
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
	for _, env := range harnessSessionEnvironments(harnessKind) {
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
	return strings.TrimRight(DetermineAPIBase(APIBaseOptions{
		Override:    request.APIBase,
		EnvVars:     []string{AgentManagerLauncherBaseEnv, "AGENT_MANAGER_API_URL"},
		DefaultBase: defaultAgentManagerLauncherBase,
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
	prefix := key + "="
	result := make([]string, 0, len(environment)+1)
	replaced := false
	for _, entry := range environment {
		if strings.HasPrefix(entry, prefix) {
			if !replaced {
				result = append(result, prefix+value)
				replaced = true
			}
			continue
		}
		result = append(result, entry)
	}
	if !replaced {
		result = append(result, prefix+value)
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
