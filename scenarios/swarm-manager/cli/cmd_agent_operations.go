package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
)

// Top-level `swarm-manager agent-operations {resolve-binding,validate,
// inspect-workflow,inspect-execution}` diagnostics over the declarative
// agent-operations runtime. These are thin Connect clients: the server
// (AgentOperationsService) owns every decision; the CLI only formats the typed
// result.

func (a *App) agentOperationsClient() apiconnect.AgentOperationsServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(a.core)
	return apiconnect.NewAgentOperationsServiceClient(httpClient, baseURL)
}

func agentOpsTargetKind(raw string) (domainpb.OperatingModeTargetKind, error) {
	switch raw {
	case "backlog-item":
		return domainpb.OperatingModeTargetKind_OPERATING_MODE_TARGET_KIND_BACKLOG_ITEM, nil
	case "initiative":
		return domainpb.OperatingModeTargetKind_OPERATING_MODE_TARGET_KIND_INITIATIVE, nil
	case "plan-execution":
		return domainpb.OperatingModeTargetKind_OPERATING_MODE_TARGET_KIND_PLAN_EXECUTION, nil
	default:
		return 0, fmt.Errorf("unknown target kind %q (want backlog-item|initiative|plan-execution)", raw)
	}
}

func agentOpsTargetFlags(fs *flag.FlagSet) (*string, *string) {
	return fs.String("target-kind", "", "target kind: backlog-item|initiative|plan-execution"),
		fs.String("target", "", "target id (backlog item kind/name, or initiative name)")
}

func (a *App) agentOpsSelector(kindRaw, id string) (*apipb.AgentOpsTargetSelector, error) {
	kind, err := agentOpsTargetKind(kindRaw)
	if err != nil {
		return nil, err
	}
	if id == "" {
		return nil, fmt.Errorf("--target is required")
	}
	return &apipb.AgentOpsTargetSelector{Kind: kind, Id: id}, nil
}

func (a *App) cmdAgentOpsResolveBinding(args []string) error {
	fs := flag.NewFlagSet("agent-operations resolve-binding", flag.ContinueOnError)
	kind, target := agentOpsTargetFlags(fs)
	operation := fs.String("operation", "", "operation id")
	version := fs.String("version", "", "exact operation version (optional)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	sel, err := a.agentOpsSelector(*kind, *target)
	if err != nil {
		return err
	}
	resp, err := a.agentOperationsClient().ResolveBinding(context.Background(), connect.NewRequest(&apipb.AgentOpsResolveBindingRequest{
		Target: sel, Operation: *operation, OperationVersion: *version,
	}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, resp.Msg)
	}
	b := resp.Msg.GetResolved()
	printSection("Resolved binding")
	fmt.Printf("  operation : %s\n", *operation)
	fmt.Printf("  mode      : %s@%s\n", b.GetMode(), b.GetModeRevision())
	fmt.Printf("  layer     : %s\n", b.GetLayer())
	if resp.Msg.GetPolicyId() != "" {
		fmt.Printf("  policy    : %s (%s)\n", resp.Msg.GetPolicyId(), resp.Msg.GetPolicyRevision())
	}
	return nil
}

func (a *App) cmdAgentOpsValidate(args []string) error {
	fs := flag.NewFlagSet("agent-operations validate", flag.ContinueOnError)
	kind, target := agentOpsTargetFlags(fs)
	operation := fs.String("operation", "", "operation id")
	version := fs.String("version", "", "exact operation version (optional)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	sel, err := a.agentOpsSelector(*kind, *target)
	if err != nil {
		return err
	}
	resp, err := a.agentOperationsClient().ValidateInvocation(context.Background(), connect.NewRequest(&apipb.AgentOpsValidateInvocationRequest{
		Target: sel, Operation: *operation, OperationVersion: *version,
	}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, resp.Msg)
	}
	printSection("Invocation validation")
	fmt.Printf("  operation declared : %v\n", resp.Msg.GetOperationDeclared())
	fmt.Printf("  target compatible  : %v\n", resp.Msg.GetTargetCompatible())
	fmt.Printf("  binding resolved   : %v\n", resp.Msg.GetBindingResolved())
	if caps := resp.Msg.GetMissingCapabilities(); len(caps) > 0 {
		fmt.Printf("  missing capabilities:\n")
		for _, c := range caps {
			fmt.Printf("    - %s\n", c)
		}
	}
	return nil
}

func (a *App) cmdAgentOpsInspectWorkflow(args []string) error {
	fs := flag.NewFlagSet("agent-operations inspect-workflow", flag.ContinueOnError)
	kind, target := agentOpsTargetFlags(fs)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	sel, err := a.agentOpsSelector(*kind, *target)
	if err != nil {
		return err
	}
	resp, err := a.agentOperationsClient().InspectWorkflow(context.Background(), connect.NewRequest(&apipb.AgentOpsInspectWorkflowRequest{Target: sel}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, resp.Msg)
	}
	if !resp.Msg.GetFound() {
		fmt.Println("  (no workflow instance for this target)")
		return nil
	}
	w := resp.Msg.GetWorkflow()
	printSection("Workflow instance")
	fmt.Printf("  id      : %s\n", w.GetInstanceId())
	fmt.Printf("  state   : %s (v%d)\n", w.GetState(), w.GetVersion())
	fmt.Printf("  operations:\n")
	for _, op := range w.GetOperations() {
		fmt.Printf("    - %s exec=%s state=%s outcome=%s\n", op.GetOperation(), op.GetExecutionId(), op.GetState(), op.GetOutcome())
	}
	return nil
}

func (a *App) cmdAgentOpsInspectExecution(args []string) error {
	fs := flag.NewFlagSet("agent-operations inspect-execution", flag.ContinueOnError)
	kind, target := agentOpsTargetFlags(fs)
	executionID := fs.String("execution", "", "execution id")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	sel, err := a.agentOpsSelector(*kind, *target)
	if err != nil {
		return err
	}
	if *executionID == "" {
		return fmt.Errorf("--execution is required")
	}
	resp, err := a.agentOperationsClient().InspectExecution(context.Background(), connect.NewRequest(&apipb.AgentOpsInspectExecutionRequest{Target: sel, ExecutionId: *executionID}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, resp.Msg)
	}
	if !resp.Msg.GetFound() {
		fmt.Println("  (no such execution for this target)")
		return nil
	}
	p := resp.Msg.GetProvenance()
	printSection("Execution provenance")
	fmt.Printf("  operation    : %s@%s\n", p.GetOperation(), p.GetOperationVersion())
	fmt.Printf("  mode         : %s@%s\n", p.GetMode(), p.GetModeRevision())
	fmt.Printf("  outcome      : %s\n", resp.Msg.GetOutcome())
	fmt.Printf("  reproducible : %v\n", resp.Msg.GetReproducible())
	fmt.Printf("  mode digest  : %s\n", p.GetCompiledModeDigest())
	fmt.Printf("  input digest : %s\n", p.GetCallerInputDigest())
	return nil
}
