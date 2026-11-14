package rehearsals

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

type scriptedN8NRunner struct {
	calls [][]string
	input [][]byte
}

func (r *scriptedN8NRunner) Run(_ context.Context, binary string, environment []string, input []byte, args ...string) ([]byte, error) {
	if binary != "/opt/vrooli/bin/vrooli" || len(environment) == 0 || environment[0] != "PATH=/usr/bin:/bin" {
		return nil, errUnexpectedN8NCommand
	}
	r.calls = append(r.calls, append([]string(nil), args...))
	r.input = append(r.input, append([]byte(nil), input...))
	value := func(flag string) string {
		for i := 0; i+1 < len(args); i++ {
			if args[i] == flag {
				return args[i+1]
			}
		}
		return ""
	}
	sandboxID, runtimeID, operationID := value("--sandbox-id"), value("--runtime-id"), value("--operation-id")
	switch {
	case strings.Contains(strings.Join(args, " "), " runtime start "):
		return []byte(`{"sandboxId":"` + sandboxID + `","runtimeId":"n8n:` + sandboxID + `","status":"running"}`), nil
	case strings.Contains(strings.Join(args, " "), " import-workflow "):
		return []byte(`{"sandboxId":"` + sandboxID + `","runtimeId":"` + runtimeID + `","operationId":"` + operationID + `","status":"imported-inactive"}`), nil
	case strings.Contains(strings.Join(args, " "), " execute-manual-workflow "):
		return []byte(`{"sandboxId":"` + sandboxID + `","runtimeId":"` + runtimeID + `","operationId":"` + operationID + `","status":"completed","evidence":{"outputSha256":"synthetic-hash","outputBytes":32,"nodeCount":2,"executionCount":1,"itemCount":1,"failedNodeExecutionCount":0}}`), nil
	default:
		return []byte(`{}`), nil
	}
}

var errUnexpectedN8NCommand = &unexpectedN8NCommandError{}

type unexpectedN8NCommandError struct{}

func (*unexpectedN8NCommandError) Error() string { return "unexpected n8n command invocation" }

func TestN8NWorkflowRuntimeCLIUsesInjectedOwnerCommandSeam(t *testing.T) {
	const sandboxID = "9c54f876-6b12-4a55-a11d-65e18af3d223"
	runner := &scriptedN8NRunner{}
	environmentCalls := 0
	owner := N8NWorkflowRuntimeCLI{Binary: "/opt/vrooli/bin/vrooli", Runner: runner, Environment: func() []string {
		environmentCalls++
		return []string{"PATH=/usr/bin:/bin", "VROOLI_ROOT=/safe/root"}
	}}
	lease, err := owner.Start(context.Background(), sandboxID)
	if err != nil {
		t.Fatal(err)
	}
	if lease.SandboxID != sandboxID || lease.RuntimeID != "n8n:"+sandboxID {
		t.Fatalf("unexpected runtime lease: %+v", lease)
	}
	workflow := `{"name":"synthetic workflow","active":false}`
	evidence, err := owner.ImportAndExecuteManual(context.Background(), lease, workflow)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Status != "completed" || evidence.NodeCount != 2 || evidence.RunCount != 1 || evidence.ItemCount != 1 || evidence.OutputSHA != "synthetic-hash" {
		t.Fatalf("unexpected content-free execution evidence: %+v", evidence)
	}
	if err := owner.Stop(context.Background(), lease); err != nil {
		t.Fatal(err)
	}
	if len(runner.calls) != 4 || environmentCalls != 4 {
		t.Fatalf("runner calls=%d environment calls=%d, want 4 each", len(runner.calls), environmentCalls)
	}
	if got := strings.Join(runner.calls[1], " "); !strings.Contains(got, "import-workflow") || !strings.Contains(got, "--operation-id "+evidence.OperationID) {
		t.Fatalf("workflow import did not use owner-issued operation ID: %s", got)
	}
	if string(runner.input[1]) != workflow || len(runner.input[0]) != 0 || len(runner.input[2]) != 0 || len(runner.input[3]) != 0 {
		t.Fatal("workflow body must be staged only on the fixed import command")
	}
	var output any
	if json.Unmarshal(runner.input[1], &output) != nil {
		t.Fatal("runner did not receive the complete bounded workflow JSON")
	}
}

func TestN8NWorkflowRuntimeRequiresInjectedCommandEnvironment(t *testing.T) {
	owner := N8NWorkflowRuntimeCLI{Binary: "/opt/vrooli/bin/vrooli", Runner: &scriptedN8NRunner{}}
	_, err := owner.Start(context.Background(), "9c54f876-6b12-4a55-a11d-65e18af3d223")
	if err == nil || !strings.Contains(err.Error(), "environment is not configured") {
		t.Fatalf("Start error = %v, want missing injected environment", err)
	}
}
