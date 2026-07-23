package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

func (a *App) cmdWorkflow(args []string) error {
	if len(args) == 0 {
		return a.workflowHelp()
	}
	switch args[0] {
	case "help", "-h", "--help":
		return a.workflowHelp()
	case "validate":
		return a.workflowValidate(args[1:])
	case "plan":
		return a.workflowReconcile(args[1:], "/api/v1/workflows/plan", true)
	case "reconcile-scenario":
		return a.workflowReconcile(args[1:], "/api/v1/workflows/reconcile-scenario", false)
	case "reload":
		return a.workflowReconcile(args[1:], "/api/v1/workflows/reload", false)
	case "list":
		return a.workflowList(args[1:])
	case "get":
		return a.workflowGet(args[1:], "/api/v1/workflows/revision")
	case "explain":
		return a.workflowGet(args[1:], "/api/v1/workflows/explain")
	case "start":
		return a.workflowStart(args[1:])
	case "execution-get":
		return a.workflowExecution(args[1:], false)
	case "execution-list":
		return a.workflowExecutionList(args[1:])
	case "execution-runs":
		return a.workflowExecutionRuns(args[1:])
	case "execution-result":
		return a.workflowExecutionResult(args[1:])
	case "execution-advance":
		return a.workflowExecution(args[1:], true)
	case "execution-wait":
		return a.workflowExecutionWait(args[1:])
	case "trace":
		return a.workflowTrace(args[1:])
	case "signal":
		return a.workflowSignal(args[1:])
	case "cancel", "retry", "resume":
		return a.workflowControl(args[0], args[1:])
	case "simulate":
		return a.workflowSimulate(args[1:])
	default:
		return fmt.Errorf("unknown workflow command %q", args[0])
	}
}

func (a *App) workflowHelp() error {
	fmt.Println(`Usage: agent-manager workflow <subcommand> [options]

Subcommands:
  validate           Validate and canonicalize a workflow definition file
  plan               Validate a scenario's workflow sources without writes
  reconcile-scenario Reconcile a scenario's workflow sources into the catalog
  reload             Reload a scenario's workflow sources (alias of reconcile)
  list               List workflow revisions for an owner
  get                Get a workflow revision by owner/key or digest
  explain            Explain the active workflow revision
  simulate           Simulate a workflow execution plan from input
  start              Start a workflow execution
  execution-list     List workflow executions
  execution-runs     List node attempts and dispatched runs for one execution
  execution-get      Get a workflow execution
  execution-result   Get a workflow execution with input/output payloads
  execution-advance  Advance a workflow execution (ops recovery)
  execution-wait     Block until a workflow execution is terminal or times out
  trace              Show a workflow execution journal
  signal             Deliver a signal to a waiting execution
  cancel             Cancel a workflow execution
  retry              Retry a workflow execution
  resume             Resume a workflow execution

Options:
  --json             Output raw JSON

Examples:
  agent-manager workflow validate --file definition.json
  agent-manager workflow reconcile-scenario --scenario swarm-manager
  agent-manager workflow list --owner swarm-manager`)
	return nil
}

func (a *App) workflowExecutionList(args []string) error {
	fs := flag.NewFlagSet("workflow execution-list", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	owner := fs.String("owner", "", "Filter by owning scenario")
	key := fs.String("key", "", "Filter by workflow key")
	status := fs.String("status", "", "Filter by execution status")
	limit := fs.Int("limit", 50, "Maximum executions (1-200)")
	offset := fs.Int("offset", 0, "Executions to skip")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, resp, err := a.services.Workflows.ListExecutions(*owner, *key, *status, *limit, *offset)
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut || resp == nil {
		cliutil.PrintJSON(body)
		return nil
	}
	for _, execution := range resp.Executions {
		fmt.Printf("%s %s %s node=%s version=%d depth=%d\n", execution.Id, execution.WorkflowKey, execution.Status.String(), execution.CurrentNodeId, execution.Version, execution.Depth)
	}
	return nil
}

func (a *App) workflowExecutionRuns(args []string) error {
	fs := flag.NewFlagSet("workflow execution-runs", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("execution-runs requires an execution id")
	}
	body, resp, err := a.services.Workflows.ExecutionRuns(fs.Args()[0])
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut || resp == nil {
		cliutil.PrintJSON(body)
		return nil
	}
	for _, attempt := range resp.Attempts {
		fmt.Printf("%s node=%s status=%s run=%s attempt=%d\n", attempt.Id, attempt.NodeId, attempt.Status, attempt.RunId, attempt.Ordinal)
	}
	return nil
}

func (a *App) workflowExecutionResult(args []string) error {
	fs := flag.NewFlagSet("workflow execution-result", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	authorized := fs.Bool("explicitly-authorized", false, "Confirm authorization to reveal input and output payloads")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 || !*authorized {
		return fmt.Errorf("execution-result requires an execution id and --explicitly-authorized")
	}
	body, resp, err := a.services.Workflows.ExecutionResult(fs.Args()[0])
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut || resp == nil {
		cliutil.PrintJSON(body)
		return nil
	}
	printWorkflowExecution(resp.Execution)
	fmt.Printf("Input: %v\nOutput: %v\n", resp.Execution.GetInput(), resp.Execution.GetOutput())
	return nil
}

func (a *App) workflowSignal(args []string) error {
	fs := flag.NewFlagSet("workflow signal", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	signal := fs.String("signal", "", "Expected wait signal name")
	payloadFile := fs.String("payload-file", "", "Typed signal payload JSON")
	idem := fs.String("idempotency-key", "", "Replay-safe caller key")
	version := fs.Int64("expected-version", 0, "Expected execution version")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 || *signal == "" || *payloadFile == "" || *idem == "" {
		return fmt.Errorf("signal requires an execution id, --signal, --payload-file, and --idempotency-key")
	}
	payload, err := readWorkflowInput(*payloadFile)
	if err != nil {
		return err
	}
	req := &apipb.SignalWorkflowExecutionRequest{ExecutionId: fs.Args()[0], Signal: *signal, Payload: payload, IdempotencyKey: *idem, ExpectedVersion: *version}
	body, resp, err := a.services.Workflows.Signal(req)
	if err != nil {
		return apiError(body, err)
	}
	return printWorkflowOperation(body, resp, *jsonOut)
}

func (a *App) workflowControl(operation string, args []string) error {
	fs := flag.NewFlagSet("workflow "+operation, flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	idem := fs.String("idempotency-key", "", "Replay-safe caller key")
	version := fs.Int64("expected-version", 0, "Expected execution version")
	reason := fs.String("reason", "", "Operator reason")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 || *idem == "" {
		return fmt.Errorf("%s requires an execution id and --idempotency-key", operation)
	}
	req := &apipb.WorkflowExecutionOperationRequest{ExecutionId: fs.Args()[0], IdempotencyKey: *idem, ExpectedVersion: *version, Reason: *reason}
	body, resp, err := a.services.Workflows.Control(operation, req)
	if err != nil {
		return apiError(body, err)
	}
	return printWorkflowOperation(body, resp, *jsonOut)
}

func printWorkflowOperation(body []byte, resp *apipb.WorkflowExecutionOperationResponse, jsonOut bool) error {
	if jsonOut || resp == nil {
		cliutil.PrintJSON(body)
		return nil
	}
	printWorkflowExecution(resp.Execution)
	fmt.Printf("Idempotent replay: %t\n", resp.Idempotent)
	return nil
}

func (a *App) workflowTrace(args []string) error {
	fs := flag.NewFlagSet("workflow trace", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	after := fs.Int64("after-sequence", 0, "Only journal events after this sequence")
	limit := fs.Int("limit", 200, "Maximum journal events")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("workflow trace requires an execution id")
	}
	body, resp, err := a.services.Workflows.Trace(fs.Args()[0], *after, *limit)
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut || resp == nil {
		cliutil.PrintJSON(body)
		return nil
	}
	printWorkflowExecution(resp.Execution)
	fmt.Printf("Attempts: %d  Journal events: %d\n", len(resp.Attempts), len(resp.Journal))
	for _, entry := range resp.Journal {
		fmt.Printf("- %d %s node=%s attempt=%s payload=%s bytes=%d\n", entry.Sequence, entry.Kind, entry.NodeId, entry.AttemptId, entry.PayloadDigest, entry.PayloadSizeBytes)
	}
	return nil
}

func readWorkflowInput(path string) (*structpb.Value, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	return structpb.NewValue(value)
}

func (a *App) workflowStart(args []string) error {
	fs := flag.NewFlagSet("workflow start", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	owner := fs.String("owner", "", "Owning scenario")
	key := fs.String("key", "", "Workflow key")
	digest := fs.String("digest", "", "Pinned revision digest")
	inputFile := fs.String("input-file", "", "Workflow input JSON")
	idem := fs.String("idempotency-key", "", "Replay-safe caller key")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *owner == "" || (*key == "" && *digest == "") || *inputFile == "" || *idem == "" {
		return fmt.Errorf("--owner, --input-file, --idempotency-key, and --key or --digest are required")
	}
	input, err := readWorkflowInput(*inputFile)
	if err != nil {
		return err
	}
	body, resp, err := a.services.Workflows.StartExecution(&apipb.StartWorkflowExecutionRequest{Owner: *owner, WorkflowKey: *key, DefinitionDigest: *digest, Input: input, IdempotencyKey: *idem})
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut || resp == nil {
		cliutil.PrintJSON(body)
		return nil
	}
	printWorkflowExecution(resp.Execution)
	return nil
}

func (a *App) workflowExecution(args []string, advance bool) error {
	fs := flag.NewFlagSet("workflow execution", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("workflow execution command requires an execution id")
	}
	body, resp, err := a.services.Workflows.Execution(fs.Args()[0], advance)
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut || resp == nil {
		cliutil.PrintJSON(body)
		return nil
	}
	printWorkflowExecution(resp.Execution)
	return nil
}

func (a *App) workflowExecutionWait(args []string) error {
	fs := flag.NewFlagSet("workflow execution-wait", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	timeout := fs.Int("timeout-seconds", 0, "Server-side wait bound in seconds (0 blocks until terminal)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("execution-wait requires an execution id")
	}
	body, resp, err := a.services.Workflows.Wait(fs.Args()[0], *timeout)
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut || resp == nil {
		cliutil.PrintJSON(body)
		return nil
	}
	printWorkflowExecution(resp.Execution)
	fmt.Printf("Timed out: %t\n", resp.TimedOut)
	return nil
}

func (a *App) workflowSimulate(args []string) error {
	fs := flag.NewFlagSet("workflow simulate", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	owner := fs.String("owner", "", "Owning scenario")
	key := fs.String("key", "", "Workflow key")
	digest := fs.String("digest", "", "Pinned revision digest")
	inputFile := fs.String("input-file", "", "Workflow input JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *owner == "" || (*key == "" && *digest == "") || *inputFile == "" {
		return fmt.Errorf("--owner, --input-file, and --key or --digest are required")
	}
	input, err := readWorkflowInput(*inputFile)
	if err != nil {
		return err
	}
	body, resp, err := a.services.Workflows.Simulate(&apipb.SimulateWorkflowRequest{Owner: *owner, WorkflowKey: *key, DefinitionDigest: *digest, Input: input})
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut || resp == nil {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Printf("Valid: %t\nDigest: %s\n", resp.Valid, resp.DefinitionDigest)
	for _, node := range resp.Nodes {
		fmt.Printf("- %s %s %s profile=%s role=%s continue=%s\n", node.NodeId, node.Kind, node.ExecutionStrategy, node.ProfileKey, node.RoleRef, node.ContinuationSource)
	}
	return nil
}

func printWorkflowExecution(x *domainpb.WorkflowExecution) {
	if x == nil {
		fmt.Println("No workflow execution")
		return
	}
	fmt.Printf("ID: %s\nWorkflow: %s\nDigest: %s\nStatus: %s\nNode: %s\nVersion: %d\n", x.Id, x.WorkflowKey, x.DefinitionDigest, x.Status.String(), x.CurrentNodeId, x.Version)
	if x.TerminalReason != nil {
		fmt.Printf("Terminal: %s %s\n", x.TerminalReason.Code, x.TerminalReason.Message)
	}
}

func (a *App) workflowValidate(args []string) error {
	fs := flag.NewFlagSet("workflow validate", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	file := fs.String("file", "", "Workflow definition JSON file")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *file == "" {
		return fmt.Errorf("--file is required")
	}
	data, err := os.ReadFile(*file)
	if err != nil {
		return err
	}
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil {
		return err
	}
	definition, err := structpb.NewStruct(object)
	if err != nil {
		return err
	}
	body, resp, err := a.services.Workflows.Validate(&apipb.ValidateWorkflowRequest{Definition: definition})
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut || resp == nil {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Printf("Valid: %t\nDigest: %s\n", resp.Valid, resp.Digest)
	for _, d := range resp.Diagnostics {
		fmt.Printf("- %s %s: %s\n", d.Code, d.Path, d.Message)
	}
	if !resp.Valid {
		return fmt.Errorf("workflow validation failed")
	}
	return nil
}

func (a *App) workflowReconcile(args []string, path string, forceDry bool) error {
	fs := flag.NewFlagSet("workflow reconcile", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	scenario := fs.String("scenario", "", "Owning scenario slug")
	dry := fs.Bool("dry-run", false, "Validate without writes")
	validateOnly := fs.Bool("validate-only", false, "Only validate sources")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*scenario) == "" {
		return fmt.Errorf("--scenario is required")
	}
	body, resp, err := a.services.Workflows.Reconcile(path, &apipb.ReconcileScenarioWorkflowsRequest{Scenario: strings.TrimSpace(*scenario), DryRun: *dry || forceDry, ValidateOnly: *validateOnly})
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut || resp == nil {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Printf("Scenario: %s\nCreated: %d  Activated: %d  Unchanged: %d  Skipped: %d  Failed: %d\n", resp.Scenario, resp.Created, resp.Activated, resp.Unchanged, resp.Skipped, resp.Failed)
	for _, item := range resp.Results {
		fmt.Printf("- %s@%s %s %s\n", item.WorkflowKey, item.Version, item.Status.String(), item.Message)
	}
	if resp.Failed > 0 {
		return fmt.Errorf("workflow reconciliation failed validation")
	}
	return nil
}

func (a *App) workflowList(args []string) error {
	fs := flag.NewFlagSet("workflow list", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	owner := fs.String("owner", "", "Owning scenario")
	key := fs.String("key", "", "Workflow key")
	limit := fs.Int("limit", 0, "Maximum revisions")
	offset := fs.Int("offset", 0, "Revisions to skip")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *owner == "" {
		return fmt.Errorf("--owner is required")
	}
	body, resp, err := a.services.Workflows.List(*owner, *key, *limit, *offset)
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut || resp == nil {
		cliutil.PrintJSON(body)
		return nil
	}
	for _, r := range resp.Revisions {
		fmt.Printf("%s %s %s active=%t\n", r.Key, r.SemanticVersion, r.Digest, r.Active)
	}
	return nil
}

func (a *App) workflowGet(args []string, path string) error {
	fs := flag.NewFlagSet("workflow get", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	owner := fs.String("owner", "", "Owning scenario")
	key := fs.String("key", "", "Workflow key")
	digest := fs.String("digest", "", "Immutable revision digest")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *owner == "" || (*key == "" && *digest == "") {
		return fmt.Errorf("--owner and either --key or --digest are required")
	}
	body, resp, err := a.services.Workflows.Get(path, *owner, *key, *digest)
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut || resp == nil {
		cliutil.PrintJSON(body)
		return nil
	}
	r := resp.Revision
	if r == nil {
		fmt.Println("No workflow revision found")
		return nil
	}
	fmt.Printf("Key: %s\nVersion: %s\nDigest: %s\nSource: %s\nActive: %t\n", r.Key, r.SemanticVersion, r.Digest, r.SourcePath, r.Active)
	return nil
}
