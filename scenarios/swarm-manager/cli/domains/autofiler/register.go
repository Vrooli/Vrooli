package autofiler

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
)

// Register exposes auto-filer status and its governed on-demand cycle through
// renderer-separated Connect primitives.
func Register() cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "autofiler", Description: "Backlog auto-filer operations", Subcommands: []cliapp.Command{
		statusCommand(),
		runNowCommand(),
	}}
}

func client(op cliapp.OperationContext) apiconnect.AutoFilerServiceClient {
	h, base := cliapp.NewConnectHTTPClient(op.Core())
	return apiconnect.NewAutoFilerServiceClient(h, base)
}

func statusCommand() cliapp.Command {
	cmd := cliapp.Command{Name: "status", NeedsAPI: true, Description: "Show backlog auto-filer policy and latest cycle status [--json]"}
	return cmd.WithPrimitive(cliapp.ProtoOperational(
		func(op cliapp.OperationContext) (*apipb.AutoFilerStatusResponse, error) {
			response, err := client(op).GetStatus(context.Background(), connect.NewRequest(&apipb.AutoFilerStatusRequest{}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		autoFilerReport,
	))
}

func runNowCommand() cliapp.Command {
	cmd := cliapp.Command{Name: "run-now", NeedsAPI: true, Description: "Run one governed auto-filer cycle immediately [--json]"}
	return cmd.WithPrimitive(cliapp.ProtoMutation(
		func(op cliapp.OperationContext) (*apipb.AutoFilerStatusResponse, error) {
			response, err := client(op).RunNow(context.Background(), connect.NewRequest(&apipb.AutoFilerRunNowRequest{}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, status *apipb.AutoFilerStatusResponse) cliapp.MutationReport {
			return cliapp.MutationReport{Result: []string{"Auto-filer cycle completed."}, Changes: autoFilerLines(status), NextCommand: []string{"swarm-manager autofiler status", "swarm-manager backlog list --status suggested"}}
		},
	))
}

func autoFilerReport(_ cliapp.OperationContext, status *apipb.AutoFilerStatusResponse) cliapp.OperationalReport {
	state := "disabled"
	if status.GetEnabled() {
		state = "enabled"
	}
	return cliapp.OperationalReport{Status: []string{fmt.Sprintf("Auto-filer: %s", state), fmt.Sprintf("Mode: %s", status.GetMode()), fmt.Sprintf("Strategy: %s", status.GetStrategy())}, Triage: []cliapp.TriageGroup{{Heading: "Policy", Items: autoFilerLines(status)}}, NextSteps: []string{"swarm-manager autofiler run-now", "swarm-manager backlog list --status suggested"}}
}

func autoFilerLines(status *apipb.AutoFilerStatusResponse) []string {
	lines := []string{fmt.Sprintf("Open auto-filed: %d / %d", status.GetOpenAutoFiled(), status.GetMaxOpenAutoFiled()), fmt.Sprintf("Remaining budget: %d", status.GetRemainingBudget()), fmt.Sprintf("Latest cycle: %d candidates, %d findings, %d created", status.GetCandidates(), status.GetFindings(), status.GetCreated())}
	if status.GetLastCycleTime() != "" {
		lines = append(lines, "Last cycle: "+status.GetLastCycleTime())
	}
	if status.GetLastError() != "" {
		lines = append(lines, "Last error: "+status.GetLastError())
	}
	return lines
}
