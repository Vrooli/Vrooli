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
)

// Workflow projection, execution history, migration status, and the
// operator-invoked reconciliation sweep over AgentOperationsService.

func (a *App) cmdAgentOpsWorkflow(args []string) error {
	fs := flag.NewFlagSet("agent-operations workflow", flag.ContinueOnError)
	kind, target := agentOpsTargetFlags(fs)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	sel, err := a.agentOpsSelector(*kind, *target)
	if err != nil {
		return err
	}
	resp, err := a.agentOperationsClient().GetWorkflowProjection(context.Background(), connect.NewRequest(&apipb.AgentOpsGetWorkflowProjectionRequest{Target: sel}))
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
	printSection("Workflow projection")
	fmt.Printf("  id        : %s\n", w.GetInstanceId())
	fmt.Printf("  domain    : %s/%s\n", w.GetDomainKind(), w.GetDomainId())
	fmt.Printf("  state     : %s (v%d)\n", agentOpsEnumLabel(w.GetState().String(), "AGENT_OPS_WORKFLOW_STATE_"), w.GetVersion())
	if resp.Msg.GetPolicyId() != "" {
		fmt.Printf("  policy    : %s (%s)\n", resp.Msg.GetPolicyId(), resp.Msg.GetPolicyRevision())
	}
	fmt.Printf("  decisions : %d\n", len(w.GetDecisions()))
	if actions := w.GetLegalActions(); len(actions) > 0 {
		fmt.Printf("  legal actions:\n")
		for _, act := range actions {
			fmt.Printf("    - %s\n", agentOpsEnumLabel(act.String(), "AGENT_OPS_DOMAIN_ACTION_"))
		}
	}
	ops := resp.Msg.GetOperations()
	fmt.Printf("  operations (%d):\n", len(ops))
	for _, op := range ops {
		modeInfo := "(snapshot missing)"
		if op.GetLegacyImport() {
			modeInfo = "[legacy import]"
		} else if op.GetSnapshotFound() {
			modeInfo = fmt.Sprintf("%s@%s [%s]", op.GetMode(), op.GetModeRevision(), agentOpsLayerLabel(op.GetBindingLayer()))
		}
		fmt.Printf("    - %s exec=%s attempt=%d state=%s outcome=%s %s\n",
			op.GetOperation(), op.GetExecutionId(), op.GetAttempt(), op.GetState(), op.GetOutcome(), modeInfo)
		if op.GetPriorExecutionId() != "" {
			fmt.Printf("      retry of %s\n", op.GetPriorExecutionId())
		}
	}
	return nil
}

func (a *App) cmdAgentOpsHistory(args []string) error {
	fs := flag.NewFlagSet("agent-operations history", flag.ContinueOnError)
	kind, target := agentOpsTargetFlags(fs)
	limit := fs.Int("limit", 0, "cap the number of summaries returned (0 = no cap)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	sel, err := a.agentOpsSelector(*kind, *target)
	if err != nil {
		return err
	}
	resp, err := a.agentOperationsClient().ListExecutionHistory(context.Background(), connect.NewRequest(&apipb.AgentOpsListExecutionHistoryRequest{
		Target: sel, Limit: int32(*limit),
	}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, resp.Msg)
	}
	execs := resp.Msg.GetExecutions()
	printSection(fmt.Sprintf("Execution history (%d, newest first)", len(execs)))
	if len(execs) == 0 {
		fmt.Println("  (no recorded executions for this target)")
		return nil
	}
	for _, e := range execs {
		if e.GetLegacyImport() {
			// A Phase-8 legacy execution import: pre-cutover run with no
			// mode/binding provenance — labeled, never fabricated.
			fmt.Printf("  %s %s [legacy import]\n", e.GetRecordedAt(), e.GetOperation())
			fmt.Printf("    exec=%s outcome=%s (imported pre-cutover record; no provenance)\n", e.GetExecutionId(), e.GetOutcome())
			continue
		}
		repro := "reproducible"
		if !e.GetReproducible() {
			repro = "NOT reproducible"
		}
		fmt.Printf("  %s %s@%s -> %s@%s [%s]\n", e.GetRecordedAt(), e.GetOperation(), e.GetOperationVersion(), e.GetMode(), e.GetModeRevision(), agentOpsLayerLabel(e.GetBindingLayer()))
		fmt.Printf("    exec=%s outcome=%s (%s)\n", e.GetExecutionId(), e.GetOutcome(), repro)
	}
	return nil
}

func (a *App) cmdAgentOpsMigrationStatus(args []string) error {
	fs := flag.NewFlagSet("agent-operations migration-status", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	resp, err := a.agentOperationsClient().GetMigrationStatus(context.Background(), connect.NewRequest(&apipb.AgentOpsGetMigrationStatusRequest{}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, resp.Msg)
	}
	printSection("Persisted-state migration status")
	fmt.Printf("  state       : %s\n", resp.Msg.GetState())
	if !resp.Msg.GetDocumentFound() {
		fmt.Println("  (no migration-status document; the Phase-8 migrator has not run)")
		return nil
	}
	fmt.Printf("  epoch       : %d\n", resp.Msg.GetEpoch())
	fmt.Printf("  staged      : %d\n", resp.Msg.GetStagedCount())
	fmt.Printf("  promoted    : %d\n", resp.Msg.GetPromotedCount())
	fmt.Printf("  quarantined : %d\n", resp.Msg.GetQuarantinedCount())
	fmt.Printf("  started     : %s\n", resp.Msg.GetStartedAt())
	fmt.Printf("  updated     : %s\n", resp.Msg.GetUpdatedAt())
	if resp.Msg.GetReportPath() != "" {
		fmt.Printf("  report      : %s\n", resp.Msg.GetReportPath())
	}
	return nil
}

func (a *App) cmdAgentOpsReconcile(args []string) error {
	fs := flag.NewFlagSet("agent-operations reconcile", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if a.globalDry {
		fmt.Println("[dry-run] would run the orphan-snapshot reconciliation sweep (no mutation performed)")
		return nil
	}
	resp, err := a.agentOperationsClient().RunReconciliation(context.Background(), connect.NewRequest(&apipb.AgentOpsRunReconciliationRequest{}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cliapp.PrintProtoJSON(os.Stdout, resp.Msg)
	}
	printSection("Reconciliation sweep")
	fmt.Printf("  dirs scanned       : %d\n", resp.Msg.GetDirsScanned())
	fmt.Printf("  snapshots seen     : %d\n", resp.Msg.GetSnapshotsSeen())
	fmt.Printf("  skipped (too new)  : %d\n", resp.Msg.GetSkippedTooRecent())
	fmt.Printf("  reaped             : %d\n", len(resp.Msg.GetReaped()))
	for _, p := range resp.Msg.GetReaped() {
		fmt.Printf("    - %s\n", p)
	}
	return nil
}
