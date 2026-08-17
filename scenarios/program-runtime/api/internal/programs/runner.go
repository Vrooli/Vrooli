package programs

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/pyenv-go"
	"program-runtime/internal/pydeps"
)

type kernelProcess struct {
	command           *exec.Cmd
	stdin             *bufio.Writer
	stdout            *bufio.Reader
	bindingPath       string
	libraryPath       string
	workingDir        string
	cleanupWorkingDir bool
	mu                sync.Mutex
}

type Invocation struct {
	BindingID string `json:"binding_id"`
	Effect    string `json:"effect"`
}

type BindingSpec struct {
	ID                 string   `json:"id"`
	Namespace          string   `json:"namespace,omitempty"`
	Scenario           string   `json:"scenario"`
	Group              string   `json:"group"`
	Command            string   `json:"command"`
	Effect             string   `json:"effect"`
	Reachable          bool     `json:"reachable"`
	ReachabilityReason string   `json:"reachability_reason"`
	RowsField          string   `json:"rows_field,omitempty"`
	MetaFields         []string `json:"meta_fields,omitempty"`
	RowFieldCandidates []string `json:"row_field_candidates,omitempty"`
}

// LibrarySpec is the immutable library projection captured when a kernel is
// created. A running program must not observe a promotion or current-version
// change halfway through execution.
type LibrarySpec struct {
	Name        string `json:"name"`
	Version     int64  `json:"version"`
	Source      string `json:"source,omitempty"`
	Description string `json:"description,omitempty"`
	Current     bool   `json:"current"`
}

type Delegator interface {
	Delegate(context.Context, DelegationRequest) (map[string]any, error)
}

type AsyncDelegator interface {
	Start(context.Context, DelegationRequest) (map[string]any, error)
	Collect(context.Context, string, string, int) (map[string]any, error)
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
	path            string
	bindings        []BindingSpec
	bridgeURL       string
	agentBridgeURL  string
	discoveryURL    string
	scratchRoot     string
	workspaces      map[string]string
	mu              sync.Mutex
	processes       map[string]*kernelProcess
	locks           map[string]*sync.Mutex
	sessionLimits   map[string]ExecutionLimits
	bindingProvider func() []BindingSpec
	libraries       []LibrarySpec
	libraryProvider func() []LibrarySpec
}

// DeadlineExceededError is retained for callers that still classify legacy
// runner errors. Current execution deadlines come from the session budget and
// are reported by the service as named budget exhaustion.
type DeadlineExceededError struct{ Limit time.Duration }

func (e *DeadlineExceededError) Error() string {
	return fmt.Sprintf("deadline_exceeded: submission exceeded %s; session variables were lost because the kernel restarted", e.Limit)
}

type MemorySampler interface{ MemoryBytes(string) (int64, bool) }

type ProgressRunner interface {
	ExecuteWithMetadataAndLimitsAndProgress(context.Context, string, string, string, string, bool, ExecutionLimits, func(Result)) (Result, error)
}

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
	return &SubprocessRunner{path: path, bindings: append([]BindingSpec(nil), bindings...), bridgeURL: strings.TrimSpace(bridgeURL), agentBridgeURL: agentURL, scratchRoot: scratchRoot, processes: make(map[string]*kernelProcess), locks: make(map[string]*sync.Mutex), workspaces: make(map[string]string), sessionLimits: make(map[string]ExecutionLimits)}
}

// SetBindings replaces the binding projection used by kernels created after
// the call. A running kernel keeps the binding set it received at startup so
// an in-flight program sees one consistent contract generation.
func (r *SubprocessRunner) SetBindings(bindings []BindingSpec) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.bindings = append([]BindingSpec(nil), bindings...)
}

// SetBindingProvider lets the runtime refresh the kernel projection whenever a
// new process is created, without polling or restarting existing programs.
func (r *SubprocessRunner) SetBindingProvider(provider func() []BindingSpec) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.bindingProvider = provider
}

// SetLibraryProvider refreshes the versioned library projection for kernels
// created after the call. Existing kernels retain their captured projection.
func (r *SubprocessRunner) SetLibraryProvider(provider func() []LibrarySpec) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.libraryProvider = provider
}

// SetDiscoveryURL configures the private runtime endpoint used by the Python
// kernel for semantic intent discovery.
func (r *SubprocessRunner) SetDiscoveryURL(url string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.discoveryURL = strings.TrimSpace(url)
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
	if r.bindingProvider != nil {
		r.bindings = append([]BindingSpec(nil), r.bindingProvider()...)
	}
	if r.libraryProvider != nil {
		r.libraries = append([]LibrarySpec(nil), r.libraryProvider()...)
	}
	python, err := pythonInterpreter()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(python, "-I", "-u", r.path)
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
	libraryPath := ""
	if len(r.bindings) > 0 || len(r.libraries) > 0 {
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
		}
		if len(r.libraries) > 0 {
			encoded, err := json.Marshal(r.libraries)
			if err != nil {
				return nil, fmt.Errorf("encode kernel libraries: %w", err)
			}
			file, err := os.CreateTemp("", "vrooli-program-runtime-libraries-*.json")
			if err != nil {
				return nil, fmt.Errorf("create kernel libraries file: %w", err)
			}
			libraryPath = file.Name()
			if err := file.Chmod(0o600); err != nil {
				_ = file.Close()
				_ = os.Remove(libraryPath)
				return nil, fmt.Errorf("protect kernel libraries file: %w", err)
			}
			if _, err := file.Write(encoded); err != nil {
				_ = file.Close()
				_ = os.Remove(libraryPath)
				return nil, fmt.Errorf("write kernel libraries file: %w", err)
			}
			if err := file.Close(); err != nil {
				_ = os.Remove(libraryPath)
				return nil, fmt.Errorf("close kernel libraries file: %w", err)
			}
		}
		if bindingPath != "" {
			cmd.Env = append(cmd.Env, "PROGRAM_RUNTIME_BINDINGS_FILE="+bindingPath)
		}
		if libraryPath != "" {
			cmd.Env = append(cmd.Env, "PROGRAM_RUNTIME_LIBRARIES_FILE="+libraryPath)
		}
		cmd.Env = append(cmd.Env, "PROGRAM_RUNTIME_SESSION_ID="+sessionID)
		if r.bridgeURL != "" {
			cmd.Env = append(cmd.Env, "PROGRAM_RUNTIME_BRIDGE_URL="+r.bridgeURL)
		}
		if r.agentBridgeURL != "" {
			cmd.Env = append(cmd.Env, "PROGRAM_RUNTIME_AGENT_BRIDGE_URL="+r.agentBridgeURL)
		}
		if r.discoveryURL != "" {
			cmd.Env = append(cmd.Env, "PROGRAM_RUNTIME_DISCOVERY_URL="+r.discoveryURL)
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
		if libraryPath != "" {
			_ = os.Remove(libraryPath)
		}
		return nil, fmt.Errorf("start kernel: %w", err)
	}
	if err := applyProcessLimits(cmd.Process.Pid, r.sessionLimits[sessionID]); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, err
	}
	p := &kernelProcess{command: cmd, stdin: bufio.NewWriter(stdin), stdout: bufio.NewReader(stdout), bindingPath: bindingPath, libraryPath: libraryPath, workingDir: workingDir, cleanupWorkingDir: cleanupWorkingDir}
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

func (r *SubprocessRunner) ExecuteWithMetadataAndLimits(ctx context.Context, sessionID, programID, provenance, source string, includeMaterialized bool, limits ExecutionLimits) (Result, error) {
	return r.ExecuteWithMetadataAndLimitsAndProgress(ctx, sessionID, programID, provenance, source, includeMaterialized, limits, nil)
}

func (r *SubprocessRunner) ExecuteWithMetadataAndLimitsAndProgress(ctx context.Context, sessionID, programID, provenance, source string, includeMaterialized bool, limits ExecutionLimits, progress func(Result)) (Result, error) {
	r.mu.Lock()
	r.sessionLimits[sessionID] = limits
	r.mu.Unlock()
	return r.executeWithProgress(ctx, sessionID, programID, provenance, source, includeMaterialized, progress)
}

func (r *SubprocessRunner) execute(ctx context.Context, sessionID, programID, provenance, source string, includeMaterialized bool) (Result, error) {
	return r.executeWithProgress(ctx, sessionID, programID, provenance, source, includeMaterialized, nil)
}

func (r *SubprocessRunner) executeWithProgress(ctx context.Context, sessionID, programID, provenance, source string, includeMaterialized bool, progress func(Result)) (Result, error) {
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
	type kernelResponse struct {
		Type             string       `json:"type"`
		OK               bool         `json:"ok"`
		Stdout           string       `json:"stdout"`
		Error            string       `json:"error"`
		ContextBytes     int64        `json:"context_bytes"`
		AgentBytes       int64        `json:"agent_bytes"`
		OutputLimitBytes int64        `json:"output_limit_bytes"`
		Invocations      []Invocation `json:"invocations"`
	}
	type decodedResponse struct {
		response kernelResponse
		err      error
	}
	responses := make(chan decodedResponse, 4)
	go func() {
		decoder := json.NewDecoder(p.stdout)
		for {
			var response kernelResponse
			if err := decoder.Decode(&response); err != nil {
				responses <- decodedResponse{err: err}
				return
			}
			responses <- decodedResponse{response: response}
			if response.Type == "result" || response.Type == "" {
				return
			}
		}
	}()
	for {
		select {
		case decoded := <-responses:
			if decoded.err != nil {
				r.killProcess(sessionID, p)
				return Result{}, fmt.Errorf("read kernel response: %w", decoded.err)
			}
			result := Result{Stdout: decoded.response.Stdout, ContextBytes: decoded.response.ContextBytes, AgentBytes: decoded.response.AgentBytes, OutputLimitBytes: decoded.response.OutputLimitBytes, Invocations: decoded.response.Invocations}
			if decoded.response.Type == "progress" {
				if progress != nil {
					progress(result)
				}
				continue
			}
			if !decoded.response.OK {
				return result, fmt.Errorf("%s", decoded.response.Error)
			}
			return result, nil
		case <-ctx.Done():
			r.killProcess(sessionID, p)
			return Result{}, ctx.Err()
		}
	}
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
	delete(r.sessionLimits, sessionID)
	r.mu.Unlock()
	if p.bindingPath != "" {
		_ = os.Remove(p.bindingPath)
	}
	if p.libraryPath != "" {
		_ = os.Remove(p.libraryPath)
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
	keys := []string{"PATH", "HOME", "LANG"}
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

func (r *SubprocessRunner) CPUTime(sessionID string) (time.Duration, bool) {
	r.mu.Lock()
	p := r.processes[sessionID]
	r.mu.Unlock()
	if p == nil || p.command.Process == nil {
		return 0, false
	}
	return processCPUTime(p.command.Process.Pid)
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

func processCPUTime(pid int) (time.Duration, bool) {
	if runtime.GOOS != "linux" {
		return 0, false
	}
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return 0, false
	}
	fields := strings.Fields(string(data))
	if len(fields) < 15 {
		return 0, false
	}
	user, err := strconv.ParseInt(fields[13], 10, 64)
	if err != nil {
		return 0, false
	}
	system, err := strconv.ParseInt(fields[14], 10, 64)
	if err != nil {
		return 0, false
	}
	return time.Duration((user + system) * int64(time.Second) / 100), true
}

func pythonInterpreter() (string, error) {
	root := strings.TrimSpace(os.Getenv("SCENARIO_DATA_DIR"))
	if root == "" {
		root = os.TempDir()
	}
	lockPath, err := pydeps.Materialize(filepath.Join(root, "program-runtime", "python"))
	if err != nil {
		return "", fmt.Errorf("kernel unavailable: materialize Python lock: %w", err)
	}
	interp, err := pyenv.Ensure(context.Background(), pyenv.Spec{
		VenvDir:    filepath.Join(root, "program-runtime", "python", "venv"),
		LockFile:   lockPath,
		BasePython: "3.12",
		UV:         "uv",
	}, nil)
	if err != nil {
		return "", fmt.Errorf("kernel unavailable: uv-managed Python 3.12 environment is not ready; run `vrooli host install uv`: %w", err)
	}
	return interp.Python, nil
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
		if p.libraryPath != "" {
			_ = os.Remove(p.libraryPath)
		}
		if p.cleanupWorkingDir && p.workingDir != "" {
			_ = os.RemoveAll(p.workingDir)
		}
		delete(r.processes, id)
	}
	return first
}
