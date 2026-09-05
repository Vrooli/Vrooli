package milestones

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/shared"
	"swarm-manager/cli/internal/support"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "milestones", Description: "Goal milestone management", Subcommands: []cliapp.Command{
		definitionCommand("create", false), definitionCommand("update", true), itemsCommand("assign", true), itemsCommand("unassign", false), archiveCommand(),
		support.APICommand("review-run", "Start milestone review (--goal NAME --milestone NAME)", deps.GoalsMilestoneReviewRun),
	}}
}

func client(op cliapp.OperationContext) apiconnect.GoalServiceClient {
	h, base := cliapp.NewConnectHTTPClient(op.Core())
	return apiconnect.NewGoalServiceClient(h, base)
}
func schema(flags ...cliapp.Flag) cliapp.ArgSchema { return cliapp.ArgSchema{Flags: flags} }
func required(name, description string) cliapp.Flag {
	return cliapp.Flag{Name: name, Description: description, Required: true}
}

func report(_ cliapp.OperationContext, response *apipb.GoalResponse) cliapp.MutationReport {
	goal := response.GetGoal()
	if goal == nil {
		return cliapp.MutationReport{Result: []string{"Milestone updated."}}
	}
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Goal %s updated", goal.GetName())}}
}

func definitionCommand(name string, update bool) cliapp.Command {
	description := "Create milestone (--goal NAME --name NAME --title TITLE --acceptance CRITERION ...)"
	if update {
		description = "Replace milestone definition (--goal NAME --name NAME --title TITLE --acceptance CRITERION ...)"
	}
	cmd := cliapp.Command{Name: name, NeedsAPI: true, Description: description, Args: schema(required("goal", "Goal name"), required("name", "Milestone name"), required("title", "Milestone title"), required("acceptance", "Acceptance criterion in Given/When/Then form; repeat for each criterion"), cliapp.Flag{Name: "description", Description: "Milestone description"})}
	return cmd.WithPrimitive(cliapp.ProtoMutation(func(op cliapp.OperationContext) (*apipb.GoalResponse, error) {
		milestone := &sharedpb.Milestone{Name: strings.TrimSpace(op.Flag("name")), Title: strings.TrimSpace(op.Flag("title")), Description: op.Flag("description"), AcceptanceCriteria: op.FlagValues("acceptance")}
		var response *connect.Response[apipb.GoalResponse]
		var err error
		if update {
			response, err = client(op).UpdateMilestone(context.Background(), connect.NewRequest(&apipb.UpdateMilestoneRequest{GoalName: strings.TrimSpace(op.Flag("goal")), Milestone: milestone}))
		} else {
			response, err = client(op).CreateMilestone(context.Background(), connect.NewRequest(&apipb.CreateMilestoneRequest{GoalName: strings.TrimSpace(op.Flag("goal")), Milestone: milestone}))
		}
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	}, report))
}

func itemsCommand(name string, assign bool) cliapp.Command {
	description := "Assign scoped items (--goal NAME --milestone NAME --items kind/name,...)"
	if !assign {
		description = "Unassign scoped items (--goal NAME --milestone NAME --items kind/name,...)"
	}
	cmd := cliapp.Command{Name: name, NeedsAPI: true, Description: description, Args: schema(required("goal", "Goal name"), required("milestone", "Milestone name"), required("items", "Comma-separated item references"))}
	return cmd.WithPrimitive(cliapp.ProtoMutation(func(op cliapp.OperationContext) (*apipb.GoalResponse, error) {
		req := connect.NewRequest(&apipb.UpdateMilestoneItemsRequest{GoalName: strings.TrimSpace(op.Flag("goal")), MilestoneName: strings.TrimSpace(op.Flag("milestone")), Items: splitCSV(op.Flag("items"))})
		var response *connect.Response[apipb.GoalResponse]
		var err error
		if assign {
			response, err = client(op).AssignMilestoneItems(context.Background(), req)
		} else {
			response, err = client(op).UnassignMilestoneItems(context.Background(), req)
		}
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	}, report))
}

func archiveCommand() cliapp.Command {
	cmd := cliapp.Command{Name: "archive", NeedsAPI: true, Description: "Archive milestone (--goal NAME --milestone NAME)", Args: schema(required("goal", "Goal name"), required("milestone", "Milestone name"))}
	return cmd.WithPrimitive(cliapp.ProtoMutation(func(op cliapp.OperationContext) (*apipb.GoalResponse, error) {
		response, err := client(op).ArchiveMilestone(context.Background(), connect.NewRequest(&apipb.ArchiveMilestoneRequest{GoalName: strings.TrimSpace(op.Flag("goal")), MilestoneName: strings.TrimSpace(op.Flag("milestone"))}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	}, report))
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
