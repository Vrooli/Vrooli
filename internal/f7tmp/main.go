package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	n8n "github.com/vrooli/vrooli/internal/resources/n8n"
)

type result struct {
	Action    string      `json:"action"`
	Runtime   n8n.Runtime `json:"runtime,omitempty"`
	Import    any         `json:"import,omitempty"`
	Execution any         `json:"execution,omitempty"`
	Stop      any         `json:"stop,omitempty"`
	Error     string      `json:"error,omitempty"`
}

func main() {
	if len(os.Args) < 3 {
		fail("usage")
	}
	action, sandboxID := os.Args[1], os.Args[2]
	owner := n8n.NewOwner("/home/matthalloran8/.vrooli")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	out := result{Action: action}
	switch action {
	case "start":
		runtime, err := owner.Start(ctx, sandboxID)
		if err != nil {
			fail("start_failed")
		}
		out.Runtime = runtime
	case "run":
		if len(os.Args) != 5 {
			fail("usage")
		}
		workflow, err := os.ReadFile(os.Args[3])
		if err != nil {
			fail("workflow_read_failed")
		}
		op, err := uuid.Parse(os.Args[4])
		if err != nil || op == uuid.Nil {
			fail("operation_invalid")
		}
		runtimeID := "n8n:" + sandboxID
		imp, err := owner.ImportSyntheticWorkflow(ctx, sandboxID, runtimeID, n8n.SyntheticWorkflowImport{OperationID: op, JSON: workflow})
		if err != nil {
			fail("import_failed")
		}
		out.Import = imp
		exec, err := owner.ExecuteSyntheticWorkflow(ctx, sandboxID, runtimeID, op)
		if err != nil {
			out.Execution = exec
			out.Error = "execution_failed"
			print(out)
			os.Exit(2)
		}
		out.Execution = exec
	case "stop":
		stopped, err := owner.Stop(ctx, sandboxID, "n8n:"+sandboxID)
		if err != nil {
			fail("stop_failed")
		}
		out.Stop = stopped
	default:
		fail("unknown_action")
	}
	print(out)
}

func fail(reason string) { print(result{Action: "failed", Error: reason}); os.Exit(2) }
func print(v any) {
	if err := json.NewEncoder(os.Stdout).Encode(v); err != nil {
		fmt.Fprintln(os.Stderr, "receipt_encode_failed")
		os.Exit(3)
	}
}
