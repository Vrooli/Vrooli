package programs

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type kernelProcess struct {
	command *exec.Cmd
	stdin   *bufio.Writer
	stdout  *bufio.Reader
	mu      sync.Mutex
}

type Invocation struct {
	BindingID string `json:"binding_id"`
	Effect    string `json:"effect"`
}

type BindingSpec struct {
	ID      string `json:"id"`
	Group   string `json:"group"`
	Command string `json:"command"`
	Effect  string `json:"effect"`
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
	path           string
	bindings       []BindingSpec
	bridgeURL      string
	agentBridgeURL string
	mu             sync.Mutex
	processes      map[string]*kernelProcess
}

func NewSubprocessRunner(path string) *SubprocessRunner {
	return NewSubprocessRunnerWithBindings(path, nil, "")
}

func NewSubprocessRunnerWithBindings(path string, bindings []BindingSpec, bridgeURL string, agentBridgeURL ...string) *SubprocessRunner {
	agentURL := ""
	if len(agentBridgeURL) > 0 {
		agentURL = strings.TrimSpace(agentBridgeURL[0])
	}
	return &SubprocessRunner{path: path, bindings: append([]BindingSpec(nil), bindings...), bridgeURL: strings.TrimSpace(bridgeURL), agentBridgeURL: agentURL, processes: make(map[string]*kernelProcess)}
}

func (r *SubprocessRunner) process(sessionID string) (*kernelProcess, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p := r.processes[sessionID]; p != nil {
		return p, nil
	}
	cmd := exec.Command("python3", "-u", r.path)
	if len(r.bindings) > 0 {
		encoded, err := json.Marshal(r.bindings)
		if err != nil {
			return nil, fmt.Errorf("encode kernel bindings: %w", err)
		}
		cmd.Env = append(os.Environ(), "PROGRAM_RUNTIME_BINDINGS="+string(encoded))
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
		return nil, fmt.Errorf("start kernel: %w", err)
	}
	p := &kernelProcess{command: cmd, stdin: bufio.NewWriter(stdin), stdout: bufio.NewReader(stdout)}
	r.processes[sessionID] = p
	return p, nil
}

func (r *SubprocessRunner) Execute(ctx context.Context, sessionID, source string) (Result, error) {
	p, err := r.process(sessionID)
	if err != nil {
		return Result{}, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if err := json.NewEncoder(p.stdin).Encode(map[string]string{"source": source}); err != nil {
		return Result{}, fmt.Errorf("write program: %w", err)
	}
	if err := p.stdin.Flush(); err != nil {
		return Result{}, fmt.Errorf("flush program: %w", err)
	}
	var response struct {
		OK           bool         `json:"ok"`
		Stdout       string       `json:"stdout"`
		Error        string       `json:"error"`
		ContextBytes int64        `json:"context_bytes"`
		Invocations  []Invocation `json:"invocations"`
	}
	if err := json.NewDecoder(p.stdout).Decode(&response); err != nil {
		return Result{}, fmt.Errorf("read kernel response: %w", err)
	}
	result := Result{Stdout: response.Stdout, ContextBytes: response.ContextBytes, Invocations: response.Invocations}
	if !response.OK {
		return result, fmt.Errorf("%s", response.Error)
	}
	return result, nil
}

func (r *SubprocessRunner) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	var first error
	for id, p := range r.processes {
		if err := p.command.Process.Kill(); err != nil && first == nil {
			first = err
		}
		delete(r.processes, id)
	}
	return first
}
