package programs

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type kernelProcess struct {
	command           *exec.Cmd
	stdin             *bufio.Writer
	stdout            *bufio.Reader
	bindingPath       string
	workingDir        string
	cleanupWorkingDir bool
	mu                sync.Mutex
}

type Invocation struct {
	BindingID string `json:"binding_id"`
	Effect    string `json:"effect"`
}

type BindingSpec struct {
	ID       string `json:"id"`
	Scenario string `json:"scenario"`
	Group    string `json:"group"`
	Command  string `json:"command"`
	Effect   string `json:"effect"`
}

type Delegator interface {
	Delegate(context.Context, DelegationRequest) (map[string]any, error)
}

type DelegationRequest struct {
	SessionID        string         `json:"session_id"`
	Owner            string         `json:"owner"`
	WorkflowKey      string         `json:"workflow_key"`
	DefinitionDigest string         `json:"definition_digest,omitempty"`
	Input            map[string]any `json:"input,omitempty"`
	IdempotencyKey   string         `json:"idempotency_key,omitempty"`
}

type SubprocessRunner struct {
	path               string
	bindings           []BindingSpec
	bridgeURL          string
	agentBridgeURL     string
	scratchRoot        string
	workspaces         map[string]string
	mu                 sync.Mutex
	processes          map[string]*kernelProcess
	locks              map[string]*sync.Mutex
	SubmissionDeadline time.Duration
}

// DeadlineExceededError is returned when the supervisor kills a kernel after
// the configured submission limit. The session namespace is intentionally
// considered lost because the next submission gets a fresh interpreter.
type DeadlineExceededError struct{ Limit time.Duration }

func (e *DeadlineExceededError) Error() string {
	return fmt.Sprintf("deadline_exceeded: submission exceeded %s; session variables were lost because the kernel restarted", e.Limit)
}

type MemorySampler interface{ MemoryBytes(string) (int64, bool) }

func NewSubprocessRunner(path string) *SubprocessRunner {
	return NewSubprocessRunnerWithBindings(path, nil, "")
}

func NewSubprocessRunnerWithBindings(path string, bindings []BindingSpec, bridgeURL string, agentBridgeURL ...string) *SubprocessRunner {
	agentURL := ""
	if len(agentBridgeURL) > 0 {
		agentURL = strings.TrimSpace(agentBridgeURL[0])
	}
	scratchRoot := strings.TrimSpace(os.Getenv("SCENARIO_DATA_DIR"))
	if scratchRoot == "" {
		scratchRoot = os.TempDir()
	}
	return &SubprocessRunner{path: path, bindings: append([]BindingSpec(nil), bindings...), bridgeURL: strings.TrimSpace(bridgeURL), agentBridgeURL: agentURL, scratchRoot: scratchRoot, processes: make(map[string]*kernelProcess), locks: make(map[string]*sync.Mutex), workspaces: make(map[string]string), SubmissionDeadline: 120 * time.Second}
}

// SetSessionWorkspace pins future kernels for a session to a validated,
// scenario-owned workspace root. It is called before the first submission.
func (r *SubprocessRunner) SetSessionWorkspace(sessionID, workspace string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.workspaces[sessionID] = workspace
}

func (r *SubprocessRunner) ClearSessionWorkspace(sessionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.workspaces, sessionID)
}

func (r *SubprocessRunner) lockFor(sessionID string) *sync.Mutex {
	r.mu.Lock()
	defer r.mu.Unlock()
	if lock := r.locks[sessionID]; lock != nil {
		return lock
	}
	lock := &sync.Mutex{}
	r.locks[sessionID] = lock
	return lock
}

func (r *SubprocessRunner) process(sessionID string) (*kernelProcess, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p := r.processes[sessionID]; p != nil {
		return p, nil
	}
	python, err := pythonInterpreter()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(python, "-u", r.path)
	workingDir := r.workspaces[sessionID]
	cleanupWorkingDir := false
	if workingDir == "" {
		workingDir = filepath.Join(r.scratchRoot, "program-runtime", "sessions", safeSessionID(sessionID))
		cleanupWorkingDir = true
		if err := os.MkdirAll(workingDir, 0o700); err != nil {
			return nil, fmt.Errorf("create kernel workspace: %w", err)
		}
	}
	cmd.Dir = workingDir
	cmd.Env = allowlistedEnvironment()
	if err := configureProcessGroup(cmd); err != nil {
		return nil, err
	}
	bindingPath := ""
	if len(r.bindings) > 0 {
		encoded, err := json.Marshal(r.bindings)
		if err != nil {
			return nil, fmt.Errorf("encode kernel bindings: %w", err)
		}
		// The fleet registry is large enough to exceed the host's exec argument
		// and environment limit. Pass the boot-time projection through a private
		// temporary file and keep the per-request protocol on stdin.
		file, err := os.CreateTemp("", "vrooli-program-runtime-bindings-*.json")
		if err != nil {
			return nil, fmt.Errorf("create kernel bindings file: %w", err)
		}
		bindingPath = file.Name()
		if err := file.Chmod(0o600); err != nil {
			_ = file.Close()
			_ = os.Remove(bindingPath)
			return nil, fmt.Errorf("protect kernel bindings file: %w", err)
		}
		if _, err := file.Write(encoded); err != nil {
			_ = file.Close()
			_ = os.Remove(bindingPath)
			return nil, fmt.Errorf("write kernel bindings file: %w", err)
		}
		if err := file.Close(); err != nil {
			_ = os.Remove(bindingPath)
			return nil, fmt.Errorf("close kernel bindings file: %w", err)
		}
		cmd.Env = append(cmd.Env, "PROGRAM_RUNTIME_BINDINGS_FILE="+bindingPath)
		cmd.Env = append(cmd.Env, "PROGRAM_RUNTIME_SESSION_ID="+sessionID)
		if r.bridgeURL != "" {
			cmd.Env = append(cmd.Env, "PROGRAM_RUNTIME_BRIDGE_URL="+r.bridgeURL)
		}
		if r.agentBridgeURL != "" {
			cmd.Env = append(cmd.Env, "PROGRAM_RUNTIME_AGENT_BRIDGE_URL="+r.agentBridgeURL)
		}
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("open kernel stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("open kernel stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		if bindingPath != "" {
			_ = os.Remove(bindingPath)
		}
		return nil, fmt.Errorf("start kernel: %w", err)
	}
	if err := applyProcessLimits(cmd.Process.Pid); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, err
	}
	p := &kernelProcess{command: cmd, stdin: bufio.NewWriter(stdin), stdout: bufio.NewReader(stdout), bindingPath: bindingPath, workingDir: workingDir, cleanupWorkingDir: cleanupWorkingDir}
	r.processes[sessionID] = p
	return p, nil
}

func (r *SubprocessRunner) Execute(ctx context.Context, sessionID, source string, includeMaterialized bool) (Result, error) {
	return r.execute(ctx, sessionID, "", "", source, includeMaterialized)
}

// ExecuteWithMetadata carries the durable program identity across the kernel
// protocol so bridge-side invocation rows can be attributed to a submission.
func (r *SubprocessRunner) ExecuteWithMetadata(ctx context.Context, sessionID, programID, provenance, source string, includeMaterialized bool) (Result, error) {
	return r.execute(ctx, sessionID, programID, provenance, source, includeMaterialized)
}

func (r *SubprocessRunner) execute(ctx context.Context, sessionID, programID, provenance, source string, includeMaterialized bool) (Result, error) {
	lock := r.lockFor(sessionID)
	lock.Lock()
	defer lock.Unlock()
	p, err := r.process(sessionID)
	if err != nil {
		return Result{}, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	request := map[string]any{"source": source, "include_materialized": includeMaterialized, "program_id": programID, "provenance": provenance}
	if err := json.NewEncoder(p.stdin).Encode(request); err != nil {
		r.killProcess(sessionID, p)
		return Result{}, fmt.Errorf("write program: %w", err)
	}
	if err := p.stdin.Flush(); err != nil {
		r.killProcess(sessionID, p)
		return Result{}, fmt.Errorf("flush program: %w", err)
	}
	var response struct {
		OK               bool         `json:"ok"`
		Stdout           string       `json:"stdout"`
		Error            string       `json:"error"`
		ContextBytes     int64        `json:"context_bytes"`
		OutputLimitBytes int64        `json:"output_limit_bytes"`
		Invocations      []Invocation `json:"invocations"`
	}
	decoded := make(chan error, 1)
	go func() { decoded <- json.NewDecoder(p.stdout).Decode(&response) }()
	deadline := r.SubmissionDeadline
	if deadline <= 0 {
		deadline = 120 * time.Second
	}
	timer := time.NewTimer(deadline)
	defer timer.Stop()
	select {
	case decodeErr := <-decoded:
		if decodeErr != nil {
			r.killProcess(sessionID, p)
			return Result{}, fmt.Errorf("read kernel response: %w", decodeErr)
		}
	case <-ctx.Done():
		r.killProcess(sessionID, p)
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return Result{}, &DeadlineExceededError{Limit: deadline}
		}
		return Result{}, ctx.Err()
	case <-timer.C:
		r.killProcess(sessionID, p)
		return Result{}, &DeadlineExceededError{Limit: deadline}
	}
	result := Result{Stdout: response.Stdout, ContextBytes: response.ContextBytes, OutputLimitBytes: response.OutputLimitBytes, Invocations: response.Invocations}
	if !response.OK {
		return result, fmt.Errorf("%s", response.Error)
	}
	return result, nil
}

func (r *SubprocessRunner) KillSession(sessionID string) {
	r.mu.Lock()
	p := r.processes[sessionID]
	r.mu.Unlock()
	if p != nil {
		r.killProcess(sessionID, p)
	}
}

func (r *SubprocessRunner) killProcess(sessionID string, p *kernelProcess) {
	r.mu.Lock()
	if current := r.processes[sessionID]; current == p {
		delete(r.processes, sessionID)
	}
	r.mu.Unlock()
	if p.bindingPath != "" {
		_ = os.Remove(p.bindingPath)
	}
	if p.cleanupWorkingDir && p.workingDir != "" {
		_ = os.RemoveAll(p.workingDir)
	}
	if p.command.Process != nil {
		killProcessGroup(p.command.Process.Pid)
		_ = p.command.Wait()
	}
}

func allowlistedEnvironment() []string {
	keys := []string{"PATH", "HOME", "LANG", "PYTHONPATH", "PROGRAM_RUNTIME_BINDINGS_FILE", "PROGRAM_RUNTIME_SESSION_ID", "PROGRAM_RUNTIME_BRIDGE_URL", "PROGRAM_RUNTIME_AGENT_BRIDGE_URL"}
	env := make([]string, 0, len(keys))
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			env = append(env, key+"="+value)
		}
	}
	return env
}

func safeSessionID(id string) string {
	id = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, id)
	if id == "" {
		return "session"
	}
	return id
}

func (r *SubprocessRunner) MemoryBytes(sessionID string) (int64, bool) {
	r.mu.Lock()
	p := r.processes[sessionID]
	r.mu.Unlock()
	if p == nil || p.command.Process == nil {
		return 0, false
	}
	return processMemoryBytes(p.command.Process.Pid)
}

func processMemoryBytes(pid int) (int64, bool) {
	if runtime.GOOS != "linux" {
		return 0, false
	}
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/statm")
	if err != nil {
		return 0, false
	}
	parts := strings.Fields(string(data))
	if len(parts) < 2 {
		return 0, false
	}
	pages, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, false
	}
	return pages * int64(os.Getpagesize()), true
}

func pythonInterpreter() (string, error) {
	for _, candidate := range []string{"python3", "python"} {
		if path, err := exec.LookPath(candidate); err == nil {
			return path, nil
		}
	}
	return "", errors.New("python interpreter not found (tried python3 and python)")
}

func (r *SubprocessRunner) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	var first error
	for id, p := range r.processes {
		if p.command.Process != nil {
			killProcessGroup(p.command.Process.Pid)
			_ = p.command.Wait()
		}
		if p.bindingPath != "" {
			_ = os.Remove(p.bindingPath)
		}
		if p.cleanupWorkingDir && p.workingDir != "" {
			_ = os.RemoveAll(p.workingDir)
		}
		delete(r.processes, id)
	}
	return first
}
