package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
)

func (a *App) goalClient() apiconnect.GoalServiceClient {
	h, base := cliapp.NewConnectHTTPClient(a.core)
	return apiconnect.NewGoalServiceClient(h, base)
}

func printGoalJSON(value any, requested bool) error {
	if !requested {
		return nil
	}
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(encoded))
	return nil
}

func goalName(args []string, command string) (string, bool, error) {
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	name := fs.String("name", "", "Goal name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return "", false, err
	}
	if strings.TrimSpace(*name) == "" {
		return "", false, fmt.Errorf("usage: %s --name NAME [--json]", command)
	}
	return strings.TrimSpace(*name), *jsonOut, nil
}

func (a *App) cmdGoalsList(args []string) error {
	fs := flag.NewFlagSet("goals list", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	result, err := a.goalClient().ListGoals(context.Background(), connect.NewRequest(&apipb.ListGoalsRequest{}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return printGoalJSON(result.Msg, true)
	}
	fmt.Printf("Goals: %d\n", len(result.Msg.Goals))
	for _, goal := range result.Msg.Goals {
		fmt.Printf("  %s — %s (%d/%d complete)\n", goal.Goal.Name, goal.Goal.Title, len(goal.Scope.Completed), len(goal.Scope.Closure))
	}
	return nil
}

func (a *App) cmdGoalsGet(args []string) error {
	name, jsonOut, err := goalName(args, "goals get")
	if err != nil {
		return err
	}
	result, err := a.goalClient().GetGoal(context.Background(), connect.NewRequest(&apipb.GetGoalRequest{Name: name}))
	if err != nil {
		return err
	}
	if jsonOut {
		return printGoalJSON(result.Msg, true)
	}
	printGoal(result.Msg)
	return nil
}

func (a *App) cmdGoalsContext(args []string) error {
	name, jsonOut, err := goalName(args, "goals context")
	if err != nil {
		return err
	}
	result, err := a.goalClient().GetScope(context.Background(), connect.NewRequest(&apipb.GetGoalRequest{Name: name}))
	if err != nil {
		return err
	}
	if jsonOut {
		return printGoalJSON(result.Msg, true)
	}
	fmt.Printf("Goal scope %s: %d total, %d ready, %d blocked\n", name, len(result.Msg.Scope.Closure), len(result.Msg.Scope.Ready), len(result.Msg.Scope.Blocked))
	return nil
}

func (a *App) cmdGoalsCreate(args []string) error {
	fs := flag.NewFlagSet("goals create", flag.ContinueOnError)
	name := fs.String("name", "", "Goal name")
	title := fs.String("title", "", "Goal title")
	targets := fs.String("targets", "", "Comma-separated item refs")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" || strings.TrimSpace(*title) == "" {
		return fmt.Errorf("usage: goals create --name NAME --title TITLE [--targets kind/name,...]")
	}
	result, err := a.goalClient().CreateGoal(context.Background(), connect.NewRequest(&apipb.CreateGoalRequest{Name: *name, Title: *title, Targets: splitCSV(*targets)}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return printGoalJSON(result.Msg, true)
	}
	printGoal(result.Msg)
	return nil
}

func (a *App) cmdGoalsArchive(args []string) error {
	name, jsonOut, err := goalName(args, "goals archive")
	if err != nil {
		return err
	}
	result, err := a.goalClient().ArchiveGoal(context.Background(), connect.NewRequest(&apipb.ArchiveGoalRequest{Name: name}))
	if err != nil {
		return err
	}
	if jsonOut {
		return printGoalJSON(result.Msg, true)
	}
	printGoal(result.Msg)
	return nil
}

func (a *App) cmdGoalsUnarchive(args []string) error {
	name, jsonOut, err := goalName(args, "goals unarchive")
	if err != nil {
		return err
	}
	result, err := a.goalClient().UnarchiveGoal(context.Background(), connect.NewRequest(&apipb.UnarchiveGoalRequest{Name: name}))
	if err != nil {
		return err
	}
	if jsonOut {
		return printGoalJSON(result.Msg, true)
	}
	printGoal(result.Msg)
	return nil
}

func (a *App) cmdGoalsDelete(args []string) error {
	name, jsonOut, err := goalName(args, "goals delete")
	if err != nil {
		return err
	}
	result, err := a.goalClient().DeleteGoal(context.Background(), connect.NewRequest(&apipb.DeleteGoalRequest{Name: name}))
	if err != nil {
		return err
	}
	if jsonOut {
		return printGoalJSON(result.Msg, true)
	}
	fmt.Printf("Deleted goal %s\n", name)
	return nil
}

func (a *App) cmdGoalsUpdate(args []string) error {
	fs := flag.NewFlagSet("goals update", flag.ContinueOnError)
	name, title, description := fs.String("name", "", "Goal name"), fs.String("title", "", "Goal title"), fs.String("description", "", "Goal description")
	priority := fs.Int("priority", -1, "Goal priority")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" || (*title == "" && *description == "" && *priority < 0) {
		return fmt.Errorf("usage: goals update --name NAME [--title TITLE] [--description TEXT] [--priority N]")
	}
	req := &apipb.UpdateGoalRequest{Name: *name}
	if *title != "" {
		req.Title = title
	}
	if *description != "" {
		req.Description = description
	}
	if *priority >= 0 {
		value := int32(*priority)
		req.Priority = &value
	}
	result, err := a.goalClient().UpdateGoal(context.Background(), connect.NewRequest(req))
	if err != nil {
		return err
	}
	if *jsonOut {
		return printGoalJSON(result.Msg, true)
	}
	printGoal(result.Msg)
	return nil
}

func (a *App) cmdGoalsTargetsAdd(args []string) error {
	return a.cmdGoalsTargets(args, "goals targets-add", true)
}
func (a *App) cmdGoalsTargetsRemove(args []string) error {
	return a.cmdGoalsTargets(args, "goals targets-remove", false)
}
func (a *App) cmdGoalsTargets(args []string, command string, add bool) error {
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	name, targets := fs.String("name", "", "Goal name"), fs.String("targets", "", "Comma-separated item refs")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *name == "" || *targets == "" {
		return fmt.Errorf("usage: %s --name NAME --targets kind/name,...", command)
	}
	req := connect.NewRequest(&apipb.UpdateGoalTargetsRequest{Name: *name, Targets: splitCSV(*targets)})
	var result *connect.Response[apipb.GoalResponse]
	var err error
	if add {
		result, err = a.goalClient().AddTargets(context.Background(), req)
	} else {
		result, err = a.goalClient().RemoveTargets(context.Background(), req)
	}
	if err != nil {
		return err
	}
	if *jsonOut {
		return printGoalJSON(result.Msg, true)
	}
	printGoal(result.Msg)
	return nil
}

func (a *App) cmdGoalsMilestoneCreate(args []string) error {
	fs := flag.NewFlagSet("goals milestone-create", flag.ContinueOnError)
	goal := fs.String("goal", "", "Goal name")
	name := fs.String("name", "", "Milestone name")
	title := fs.String("title", "", "Milestone title")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *goal == "" || *name == "" || *title == "" {
		return fmt.Errorf("usage: goals milestone-create --goal NAME --name NAME --title TITLE")
	}
	result, err := a.goalClient().CreateMilestone(context.Background(), connect.NewRequest(&apipb.CreateMilestoneRequest{GoalName: *goal, Milestone: &domainpb.Milestone{Name: *name, Title: *title}}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return printGoalJSON(result.Msg, true)
	}
	printGoal(result.Msg)
	return nil
}

func (a *App) cmdGoalsMilestoneAssign(args []string) error {
	fs := flag.NewFlagSet("goals milestone-assign", flag.ContinueOnError)
	goal := fs.String("goal", "", "Goal name")
	milestone := fs.String("milestone", "", "Milestone name")
	items := fs.String("items", "", "Comma-separated item refs")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *goal == "" || *milestone == "" || *items == "" {
		return fmt.Errorf("usage: goals milestone-assign --goal NAME --milestone NAME --items kind/name,...")
	}
	result, err := a.goalClient().AssignMilestoneItems(context.Background(), connect.NewRequest(&apipb.UpdateMilestoneItemsRequest{GoalName: *goal, MilestoneName: *milestone, Items: splitCSV(*items)}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return printGoalJSON(result.Msg, true)
	}
	printGoal(result.Msg)
	return nil
}

func (a *App) cmdGoalsMilestoneUnassign(args []string) error {
	return a.cmdGoalsMilestoneItems(args, "goals milestone-unassign", false)
}
func (a *App) cmdGoalsMilestoneItems(args []string, command string, assign bool) error {
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	goal, milestone, items := fs.String("goal", "", "Goal name"), fs.String("milestone", "", "Milestone name"), fs.String("items", "", "Comma-separated item refs")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *goal == "" || *milestone == "" || *items == "" {
		return fmt.Errorf("usage: %s --goal NAME --milestone NAME --items kind/name,...", command)
	}
	req := connect.NewRequest(&apipb.UpdateMilestoneItemsRequest{GoalName: *goal, MilestoneName: *milestone, Items: splitCSV(*items)})
	var result *connect.Response[apipb.GoalResponse]
	var err error
	if assign {
		result, err = a.goalClient().AssignMilestoneItems(context.Background(), req)
	} else {
		result, err = a.goalClient().UnassignMilestoneItems(context.Background(), req)
	}
	if err != nil {
		return err
	}
	if *jsonOut {
		return printGoalJSON(result.Msg, true)
	}
	printGoal(result.Msg)
	return nil
}

func (a *App) cmdGoalsMilestoneArchive(args []string) error {
	fs := flag.NewFlagSet("goals milestone-archive", flag.ContinueOnError)
	goal, milestone := fs.String("goal", "", "Goal name"), fs.String("milestone", "", "Milestone name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *goal == "" || *milestone == "" {
		return fmt.Errorf("usage: goals milestone-archive --goal NAME --milestone NAME")
	}
	result, err := a.goalClient().ArchiveMilestone(context.Background(), connect.NewRequest(&apipb.ArchiveMilestoneRequest{GoalName: *goal, MilestoneName: *milestone}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return printGoalJSON(result.Msg, true)
	}
	printGoal(result.Msg)
	return nil
}

func printGoal(response *apipb.GoalResponse) {
	fmt.Printf("Goal %s — %s\n", response.Goal.Name, response.Goal.Title)
	fmt.Printf("  Scope: %d total, %d complete, %d ready, %d blocked\n", len(response.Scope.Closure), len(response.Scope.Completed), len(response.Scope.Ready), len(response.Scope.Blocked))
	fmt.Printf("  Milestones: %d; unassigned: %d\n", len(response.Scope.Milestones), len(response.Scope.Unassigned))
}
