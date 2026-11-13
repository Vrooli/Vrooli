package rehearsals

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
)

const maxN8NRuntimeCommandOutput = 32 << 10

type N8NTaskRuntimeLease struct {
	SandboxID string
	RuntimeID string
}

type N8NWorkflowExecutionEvidence struct {
	OperationID string
	Status      string
	OutputSHA   string
	OutputBytes int
	NodeCount   int
	RunCount    int
	ItemCount   int
	FailedRuns  int
}

// N8NWorkflowRuntime owns the fixed control-plane calls used by one task run.
// It never takes Docker arguments, paths, node IDs, or credentials from a
// caller. The n8n resource remains the authority for runtime state and import.
type N8NWorkflowRuntime interface {
	Start(context.Context, string) (N8NTaskRuntimeLease, error)
	ImportAndExecuteManual(context.Context, N8NTaskRuntimeLease, string) (N8NWorkflowExecutionEvidence, error)
	Stop(context.Context, N8NTaskRuntimeLease) error
}

type N8NWorkflowRuntimeCLI struct {
	Binary string
	Runner N8NWorkflowCommandRunner
	// seam: rehearsals.N8NWorkflowRuntimeCLI.Environment substitutes the
	// allowlisted child-process environment for deterministic tests.
	Environment func() []string
}

// seam: rehearsals.N8NWorkflowCommandRunner substitutes the owner command
// runner while retaining fixed, adapter-owned arguments.
// N8NWorkflowCommandRunner is the deterministic seam around the fixed CLI
// commands used by this owner adapter. Implementations receive only owner-built
// argv and bounded stdin; they cannot be selected by workflow data.
type N8NWorkflowCommandRunner interface {
	Run(context.Context, string, []string, []byte, ...string) ([]byte, error)
}

type execN8NWorkflowCommandRunner struct{}

func (execN8NWorkflowCommandRunner) Run(ctx context.Context, binary string, environment []string, input []byte, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Env = environment
	if input != nil {
		cmd.Stdin = bytes.NewReader(input)
	}
	stdout, stderr := &boundedCommandOutput{}, &boundedCommandOutput{}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	if err := cmd.Run(); err != nil {
		// Owner stderr can contain third-party node names or workflow fields.
		return nil, fmt.Errorf("n8n resource owner command failed: %w", err)
	}
	return stdout.Bytes(), nil
}

func (c N8NWorkflowRuntimeCLI) Start(ctx context.Context, sandboxID string) (N8NTaskRuntimeLease, error) {
	if c.Binary == "" || !sandboxIDPattern.MatchString(sandboxID) {
		return N8NTaskRuntimeLease{}, errors.New("n8n task runtime requires a control-plane executable and Sandbox UUID")
	}
	output, err := c.run(ctx, nil, "resource", "n8n", "runtime", "start", "--sandbox-id", sandboxID, "--json")
	if err != nil {
		return N8NTaskRuntimeLease{}, err
	}
	var result struct {
		SandboxID string `json:"sandboxId"`
		RuntimeID string `json:"runtimeId"`
		Status    string `json:"status"`
	}
	if err := json.Unmarshal(output, &result); err != nil || result.SandboxID != sandboxID || result.RuntimeID == "" || result.Status != "running" {
		return N8NTaskRuntimeLease{}, errors.New("n8n owner returned an invalid runtime start receipt")
	}
	return N8NTaskRuntimeLease{SandboxID: sandboxID, RuntimeID: result.RuntimeID}, nil
}

func (c N8NWorkflowRuntimeCLI) ImportAndExecuteManual(ctx context.Context, lease N8NTaskRuntimeLease, workflowJSON string) (N8NWorkflowExecutionEvidence, error) {
	if !validTaskRuntimeLease(lease) || strings.TrimSpace(workflowJSON) == "" {
		return N8NWorkflowExecutionEvidence{}, errors.New("n8n workflow import requires a valid task runtime and workflow JSON")
	}
	operationID := uuid.NewString()
	args := []string{"resource", "n8n", "runtime", "import-workflow", "--sandbox-id", lease.SandboxID, "--runtime-id", lease.RuntimeID, "--operation-id", operationID, "--json"}
	output, err := c.run(ctx, []byte(workflowJSON), args...)
	if err != nil {
		return N8NWorkflowExecutionEvidence{}, err
	}
	var imported struct {
		SandboxID   string `json:"sandboxId"`
		RuntimeID   string `json:"runtimeId"`
		OperationID string `json:"operationId"`
		Status      string `json:"status"`
	}
	if err := json.Unmarshal(output, &imported); err != nil || imported.SandboxID != lease.SandboxID || imported.RuntimeID != lease.RuntimeID || imported.OperationID != operationID || imported.Status != "imported-inactive" {
		return N8NWorkflowExecutionEvidence{}, errors.New("n8n owner returned an invalid inactive workflow import receipt")
	}
	output, err = c.run(ctx, nil, "resource", "n8n", "runtime", "execute-manual-workflow", "--sandbox-id", lease.SandboxID, "--runtime-id", lease.RuntimeID, "--operation-id", operationID, "--json")
	if err != nil {
		return N8NWorkflowExecutionEvidence{}, err
	}
	var executed struct {
		SandboxID   string `json:"sandboxId"`
		RuntimeID   string `json:"runtimeId"`
		OperationID string `json:"operationId"`
		Status      string `json:"status"`
		Evidence    struct {
			OutputSHA string `json:"outputSha256"`
			Bytes     int    `json:"outputBytes"`
			Nodes     int    `json:"nodeCount"`
			Runs      int    `json:"executionCount"`
			Items     int    `json:"itemCount"`
			Failed    int    `json:"failedNodeExecutionCount"`
		} `json:"evidence"`
	}
	if err := json.Unmarshal(output, &executed); err != nil || executed.SandboxID != lease.SandboxID || executed.RuntimeID != lease.RuntimeID || executed.OperationID != operationID || executed.Status != "completed" {
		return N8NWorkflowExecutionEvidence{OperationID: operationID, Status: executed.Status}, errors.New("n8n owner returned an incomplete manual execution receipt")
	}
	return N8NWorkflowExecutionEvidence{OperationID: operationID, Status: executed.Status, OutputSHA: executed.Evidence.OutputSHA, OutputBytes: executed.Evidence.Bytes, NodeCount: executed.Evidence.Nodes, RunCount: executed.Evidence.Runs, ItemCount: executed.Evidence.Items, FailedRuns: executed.Evidence.Failed}, nil
}

func (c N8NWorkflowRuntimeCLI) Stop(ctx context.Context, lease N8NTaskRuntimeLease) error {
	if !validTaskRuntimeLease(lease) {
		return errors.New("n8n task runtime stop requires a valid owner-issued Sandbox/runtime identity")
	}
	_, err := c.run(ctx, nil, "resource", "n8n", "runtime", "stop", "--sandbox-id", lease.SandboxID, "--runtime-id", lease.RuntimeID, "--json")
	return err
}

func validTaskRuntimeLease(lease N8NTaskRuntimeLease) bool {
	return sandboxIDPattern.MatchString(lease.SandboxID) && strings.HasPrefix(lease.RuntimeID, "n8n:") && len(lease.RuntimeID) <= 128
}

func (c N8NWorkflowRuntimeCLI) run(ctx context.Context, input []byte, args ...string) ([]byte, error) {
	if c.Binary == "" {
		return nil, errors.New("vrooli control-plane executable is not configured")
	}
	callCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	runner := c.Runner
	if runner == nil {
		runner = execN8NWorkflowCommandRunner{}
	}
	environment := c.Environment
	if environment == nil {
		return nil, errors.New("n8n workflow command environment is not configured")
	}
	output, err := runner.Run(callCtx, c.Binary, environment(), input, args...)
	if err != nil {
		return nil, err
	}
	if len(output) > maxN8NRuntimeCommandOutput {
		return nil, fmt.Errorf("owner command output exceeds %d bytes", maxN8NRuntimeCommandOutput)
	}
	return output, nil
}

type boundedCommandOutput struct{ bytes.Buffer }

func (w *boundedCommandOutput) Write(data []byte) (int, error) {
	if w.Len()+len(data) > maxN8NRuntimeCommandOutput {
		return 0, fmt.Errorf("owner command output exceeds %d bytes", maxN8NRuntimeCommandOutput)
	}
	return w.Buffer.Write(data)
}

var _ io.Writer = (*boundedCommandOutput)(nil)
