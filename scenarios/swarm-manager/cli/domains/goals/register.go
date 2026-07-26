package goals

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
	"swarm-manager/cli/internal/support"
)

// Register exposes goal operations through renderer-separated GoalService
// primitives. Workflow-start commands remain in the legacy group until their
// REST-only endpoints receive an equivalent Connect contract.
func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "goals", Description: "Goal and milestone management", Subcommands: []cliapp.Command{
		listCommand(), getCommand(), createCommand(), updateCommand(), deleteCommand(),
		archiveCommand(), unarchiveCommand(), contextCommand(), targetsCommand("targets-add", true),
		targetsCommand("targets-remove", false),
		workflowPendingCommand(), workflowApplyCommand(), closeOutCommand(),
		support.APICommand("plan-run", "Start goal planning (--name NAME) [--json]", deps.GoalsPlanRun),
		support.APICommand("discover-run", "Start goal discovery (--name NAME) [--json]", deps.GoalsDiscoverRun),
	}}
}

func client(op cliapp.OperationContext) apiconnect.GoalServiceClient {
	h, base := cliapp.NewConnectHTTPClient(op.Core())
	return apiconnect.NewGoalServiceClient(h, base)
}

func flags(flags ...cliapp.Flag) cliapp.ArgSchema { return cliapp.ArgSchema{Flags: flags} }

func goalReport(_ cliapp.OperationContext, response *apipb.GoalResponse) cliapp.MutationReport {
	goal := response.GetGoal()
	if goal == nil {
		return cliapp.MutationReport{Result: []string{"Goal updated."}}
	}
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Goal %s — %s", goal.GetName(), goal.GetTitle())}}
}

func goalListReport(_ cliapp.OperationContext, response *apipb.ListGoalsResponse) cliapp.ListReport {
	rows := make([]string, 0, len(response.GetGoals()))
	for _, item := range response.GetGoals() {
		goal := item.GetGoal()
		if goal != nil {
			rows = append(rows, fmt.Sprintf("%s — %s", goal.GetName(), goal.GetTitle()))
		}
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Goals: %d", len(rows))}, ResultsHeading: "Goals", Results: rows}
}

func nameFlag() cliapp.Flag {
	return cliapp.Flag{Name: "name", Description: "Goal name", Required: true}
}

func listCommand() cliapp.Command {
	cmd := cliapp.Command{Name: "list", NeedsAPI: true, Description: "List goals [--json]"}
	return cmd.WithPrimitive(cliapp.ProtoList(func(op cliapp.OperationContext) (*apipb.ListGoalsResponse, error) {
		response, err := client(op).ListGoals(context.Background(), connect.NewRequest(&apipb.ListGoalsRequest{}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	}, goalListReport))
}

func getCommand() cliapp.Command {
	cmd := cliapp.Command{Name: "get", NeedsAPI: true, Description: "Get a goal with scope (--name NAME) [--json]", Args: flags(nameFlag())}
	return cmd.WithPrimitive(cliapp.ProtoList(func(op cliapp.OperationContext) (*apipb.GoalResponse, error) {
		response, err := client(op).GetGoal(context.Background(), connect.NewRequest(&apipb.GetGoalRequest{Name: strings.TrimSpace(op.Flag("name"))}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *apipb.GoalResponse) cliapp.ListReport {
		return cliapp.ListReport{Summary: goalReport(nil, response).Result}
	}))
}

func createCommand() cliapp.Command {
	cmd := cliapp.Command{Name: "create", NeedsAPI: true, Description: "Create goal (--name NAME --title TITLE [--targets kind/name,...]) [--json]", Args: flags(nameFlag(), cliapp.Flag{Name: "title", Description: "Goal title", Required: true}, cliapp.Flag{Name: "targets", Description: "Comma-separated item references"})}
	return cmd.WithPrimitive(cliapp.ProtoMutation(func(op cliapp.OperationContext) (*apipb.GoalResponse, error) {
		response, err := client(op).CreateGoal(context.Background(), connect.NewRequest(&apipb.CreateGoalRequest{Name: strings.TrimSpace(op.Flag("name")), Title: strings.TrimSpace(op.Flag("title")), Targets: splitCSV(op.Flag("targets"))}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	}, goalReport))
}

func updateCommand() cliapp.Command {
	cmd := cliapp.Command{Name: "update", NeedsAPI: true, Description: "Update goal (--name NAME [--title TITLE] [--description TEXT] [--priority N]) [--json]", Args: flags(nameFlag(), cliapp.Flag{Name: "title", Description: "Goal title"}, cliapp.Flag{Name: "description", Description: "Goal description"}, cliapp.Flag{Name: "priority", Description: "Goal priority"})}
	return cmd.WithPrimitive(cliapp.ProtoMutation(func(op cliapp.OperationContext) (*apipb.GoalResponse, error) {
		req := &apipb.UpdateGoalRequest{Name: strings.TrimSpace(op.Flag("name"))}
		if title := op.Flag("title"); title != "" {
			req.Title = &title
		}
		if description := op.Flag("description"); description != "" {
			req.Description = &description
		}
		if raw := op.Flag("priority"); raw != "" {
			value, err := strconv.ParseInt(raw, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("--priority must be an integer: %w", err)
			}
			priority := int32(value)
			req.Priority = &priority
		}
		if req.Title == nil && req.Description == nil && req.Priority == nil {
			return nil, fmt.Errorf("one of --title, --description, or --priority is required")
		}
		response, err := client(op).UpdateGoal(context.Background(), connect.NewRequest(req))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	}, goalReport))
}

func deleteCommand() cliapp.Command {
	cmd := cliapp.Command{Name: "delete", NeedsAPI: true, Description: "Delete goal (--name NAME) [--json]", Args: flags(nameFlag())}
	return cmd.WithPrimitive(cliapp.ProtoMutation(func(op cliapp.OperationContext) (*apipb.EmptyGoalResponse, error) {
		response, err := client(op).DeleteGoal(context.Background(), connect.NewRequest(&apipb.DeleteGoalRequest{Name: strings.TrimSpace(op.Flag("name"))}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, _ *apipb.EmptyGoalResponse) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"Goal deleted."}}
	}))
}
func archiveCommand() cliapp.Command {
	return goalNameMutation("archive", "Archive goal (--name NAME) [--json]", func(c apiconnect.GoalServiceClient, name string) (*connect.Response[apipb.GoalResponse], error) {
		return c.ArchiveGoal(context.Background(), connect.NewRequest(&apipb.ArchiveGoalRequest{Name: name}))
	})
}
func unarchiveCommand() cliapp.Command {
	return goalNameMutation("unarchive", "Unarchive goal (--name NAME) [--json]", func(c apiconnect.GoalServiceClient, name string) (*connect.Response[apipb.GoalResponse], error) {
		return c.UnarchiveGoal(context.Background(), connect.NewRequest(&apipb.UnarchiveGoalRequest{Name: name}))
	})
}

func goalNameMutation(name, description string, call func(apiconnect.GoalServiceClient, string) (*connect.Response[apipb.GoalResponse], error)) cliapp.Command {
	cmd := cliapp.Command{Name: name, NeedsAPI: true, Description: description, Args: flags(nameFlag())}
	return cmd.WithPrimitive(cliapp.ProtoMutation(func(op cliapp.OperationContext) (*apipb.GoalResponse, error) {
		response, err := call(client(op), strings.TrimSpace(op.Flag("name")))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	}, goalReport))
}

func contextCommand() cliapp.Command {
	cmd := cliapp.Command{Name: "context", NeedsAPI: true, Description: "Get a goal graph snapshot (--name NAME) [--json]", Args: flags(nameFlag())}
	return cmd.WithPrimitive(cliapp.ProtoList(func(op cliapp.OperationContext) (*apipb.GoalScopeResponse, error) {
		response, err := client(op).GetScope(context.Background(), connect.NewRequest(&apipb.GetGoalRequest{Name: strings.TrimSpace(op.Flag("name"))}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *apipb.GoalScopeResponse) cliapp.ListReport {
		scope := response.GetScope()
		return cliapp.ListReport{Summary: []string{fmt.Sprintf("Goal scope: %d total, %d ready, %d blocked", len(scope.GetClosure()), len(scope.GetReady()), len(scope.GetBlocked()))}}
	}))
}

func targetsCommand(name string, add bool) cliapp.Command {
	cmd := cliapp.Command{Name: name, NeedsAPI: true, Description: "Update goal targets (--name NAME --targets kind/name,...) [--json]", Args: flags(nameFlag(), cliapp.Flag{Name: "targets", Description: "Comma-separated item references", Required: true})}
	return cmd.WithPrimitive(cliapp.ProtoMutation(func(op cliapp.OperationContext) (*apipb.GoalResponse, error) {
		req := connect.NewRequest(&apipb.UpdateGoalTargetsRequest{Name: strings.TrimSpace(op.Flag("name")), Targets: splitCSV(op.Flag("targets"))})
		var response *connect.Response[apipb.GoalResponse]
		var err error
		if add {
			response, err = client(op).AddTargets(context.Background(), req)
		} else {
			response, err = client(op).RemoveTargets(context.Background(), req)
		}
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	}, goalReport))
}

func workflowPendingCommand() cliapp.Command {
	cmd := cliapp.Command{Name: "workflow-pending", NeedsAPI: true, Description: "List goal workflow results awaiting application [--goal NAME] [--json]", Args: flags(cliapp.Flag{Name: "goal", Description: "Restrict to one goal"})}
	return cmd.WithPrimitive(cliapp.ProtoList(func(op cliapp.OperationContext) (*apipb.ListPendingGoalWorkflowsResponse, error) {
		response, err := client(op).ListPendingGoalWorkflows(context.Background(), connect.NewRequest(&apipb.ListPendingGoalWorkflowsRequest{GoalName: strings.TrimSpace(op.Flag("goal"))}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *apipb.ListPendingGoalWorkflowsResponse) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{fmt.Sprintf("Pending goal workflow results: %d", len(response.GetPending()))}}
	}))
}

func workflowApplyCommand() cliapp.Command {
	cmd := cliapp.Command{Name: "workflow-apply", NeedsAPI: true, Description: "Apply a terminal goal workflow result (--goal NAME --execution-id ID) [--json]", Args: flags(cliapp.Flag{Name: "goal", Description: "Goal name", Required: true}, cliapp.Flag{Name: "execution-id", Description: "Workflow execution ID", Required: true})}
	return cmd.WithPrimitive(cliapp.ProtoMutation(func(op cliapp.OperationContext) (*apipb.ApplyGoalWorkflowResponse, error) {
		response, err := client(op).ApplyGoalWorkflow(context.Background(), connect.NewRequest(&apipb.ApplyGoalWorkflowRequest{GoalName: strings.TrimSpace(op.Flag("goal")), ExecutionId: strings.TrimSpace(op.Flag("execution-id"))}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *apipb.ApplyGoalWorkflowResponse) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{fmt.Sprintf("Applied %s", response.GetExecutionId())}}
	}))
}

func closeOutCommand() cliapp.Command {
	cmd := cliapp.Command{Name: "close-out", NeedsAPI: true, Description: "Mark a goal achieved after every milestone is verified delivered (--name NAME) [--json]", Args: flags(nameFlag())}
	return cmd.WithPrimitive(cliapp.ProtoMutation(func(op cliapp.OperationContext) (*apipb.GoalResponse, error) {
		response, err := client(op).CloseOutGoal(context.Background(), connect.NewRequest(&apipb.CloseOutGoalRequest{Name: strings.TrimSpace(op.Flag("name"))}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	}, goalReport))
}

func splitCSV(value string) []string {
	var values []string
	for _, part := range strings.Split(value, ",") {
		if part = strings.TrimSpace(part); part != "" {
			values = append(values, part)
		}
	}
	return values
}
