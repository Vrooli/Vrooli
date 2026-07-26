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
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/shared"
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
	fs := flag.NewFlagSet("milestones create", flag.ContinueOnError)
	goal := fs.String("goal", "", "Goal name")
	name := fs.String("name", "", "Milestone name")
	title := fs.String("title", "", "Milestone title")
	description := fs.String("description", "", "Milestone description")
	var acceptance stringSlice
	fs.Var(&acceptance, "acceptance", "Acceptance criterion in Given/When/Then form; repeat for each criterion")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *goal == "" || *name == "" || *title == "" || len(acceptance) == 0 {
		return fmt.Errorf("usage: milestones create --goal NAME --name NAME --title TITLE --acceptance CRITERION [--acceptance CRITERION ...] [--description TEXT]")
	}
	milestone := &sharedpb.Milestone{Name: *name, Title: *title, Description: *description, AcceptanceCriteria: acceptance}
	result, err := a.goalClient().CreateMilestone(context.Background(), connect.NewRequest(&apipb.CreateMilestoneRequest{GoalName: *goal, Milestone: milestone}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return printGoalJSON(result.Msg, true)
	}
	printGoal(result.Msg)
	return nil
}

// cmdGoalsMilestoneUpdate replaces a milestone's definition. The update is a
// whole-definition replacement, so it restates title and acceptance criteria
// every time rather than patching individual fields. Member items and archive
// state are owned by assignment and archive operations and are preserved.
func (a *App) cmdGoalsMilestoneUpdate(args []string) error {
	fs := flag.NewFlagSet("milestones update", flag.ContinueOnError)
	goal := fs.String("goal", "", "Goal name")
	name := fs.String("name", "", "Milestone name")
	title := fs.String("title", "", "Milestone title")
	description := fs.String("description", "", "Milestone description")
	var acceptance stringSlice
	fs.Var(&acceptance, "acceptance", "Acceptance criterion in Given/When/Then form; repeat for each criterion")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *goal == "" || *name == "" || *title == "" || len(acceptance) == 0 {
		return fmt.Errorf("usage: milestones update --goal NAME --name NAME --title TITLE --acceptance CRITERION [--acceptance CRITERION ...] [--description TEXT]")
	}
	milestone := &sharedpb.Milestone{Name: *name, Title: *title, Description: *description, AcceptanceCriteria: acceptance}
	result, err := a.goalClient().UpdateMilestone(context.Background(), connect.NewRequest(&apipb.UpdateMilestoneRequest{GoalName: *goal, Milestone: milestone}))
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
	fs := flag.NewFlagSet("milestones assign", flag.ContinueOnError)
	goal := fs.String("goal", "", "Goal name")
	milestone := fs.String("milestone", "", "Milestone name")
	items := fs.String("items", "", "Comma-separated item refs")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *goal == "" || *milestone == "" || *items == "" {
		return fmt.Errorf("usage: milestones assign --goal NAME --milestone NAME --items kind/name,...")
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
	return a.cmdGoalsMilestoneItems(args, "milestones unassign", false)
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
	fs := flag.NewFlagSet("milestones archive", flag.ContinueOnError)
	goal, milestone := fs.String("goal", "", "Goal name"), fs.String("milestone", "", "Milestone name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *goal == "" || *milestone == "" {
		return fmt.Errorf("usage: milestones archive --goal NAME --milestone NAME")
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

func (a *App) cmdGoalsPlanRun(args []string) error {
	return a.cmdGoalsWorkflowRun(args, "goals plan-run", "plan-run")
}

func (a *App) cmdGoalsDiscoverRun(args []string) error {
	return a.cmdGoalsWorkflowRun(args, "goals discover-run", "discover-run")
}

func (a *App) cmdGoalsWorkflowRun(args []string, command, action string) error {
	name, jsonOut, err := goalName(args, command)
	if err != nil {
		return err
	}
	body, err := a.requestMultipart("POST", "/goals/"+name+"/"+action, []byte(`{}`), "application/json")
	if err != nil {
		return err
	}
	if printJSONIfRequested(jsonOut, body) {
		return nil
	}
	var result struct {
		ExecutionID string `json:"execution_id"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}
	fmt.Printf("Started %s workflow: %s\n", action, result.ExecutionID)
	return nil
}

func (a *App) cmdGoalsMilestoneReviewRun(args []string) error {
	fs := flag.NewFlagSet("milestones review-run", flag.ContinueOnError)
	goal, milestone := fs.String("goal", "", "Goal name"), fs.String("milestone", "", "Milestone name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("goal", *goal, "milestone", *milestone); err != nil {
		return err
	}
	body, err := a.requestMultipart("POST", "/goals/"+*goal+"/milestones/"+*milestone+"/review-run", []byte(`{}`), "application/json")
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	var result struct {
		ExecutionID string `json:"execution_id"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}
	fmt.Printf("Started milestone review workflow: %s\n", result.ExecutionID)
	return nil
}

func printGoal(response *apipb.GoalResponse) {
	fmt.Printf("Goal %s — %s\n", response.Goal.Name, response.Goal.Title)
	fmt.Printf("  Scope: %d total, %d complete, %d ready, %d blocked\n", len(response.Scope.Closure), len(response.Scope.Completed), len(response.Scope.Ready), len(response.Scope.Blocked))
	fmt.Printf("  Milestones: %d; unassigned: %d\n", len(response.Scope.Milestones), len(response.Scope.Unassigned))
}

// cmdGoalsCloseOut is the operator-only assertion that a goal's outcome is
// delivered. It is evidence-gated server-side: every non-archived milestone
// must already carry an independent-review verdict.
func (a *App) cmdGoalsCloseOut(args []string) error {
	name, jsonOut, err := goalName(args, "goals close-out")
	if err != nil {
		return err
	}
	result, err := a.goalClient().CloseOutGoal(context.Background(), connect.NewRequest(&apipb.CloseOutGoalRequest{Name: name}))
	if err != nil {
		return err
	}
	if jsonOut {
		return printGoalJSON(result.Msg, true)
	}
	fmt.Printf("Closed out %s — status %s\n", result.Msg.Goal.Name, result.Msg.Goal.Status)
	return nil
}

// cmdGoalsWorkflowPending reports terminal goal workflow results that have not
// been applied yet. A non-empty listing that never drains means the apply hop
// is stalled — the failure this command exists to make visible.
func (a *App) cmdGoalsWorkflowPending(args []string) error {
	fs := flag.NewFlagSet("goals workflow-pending", flag.ContinueOnError)
	goal := fs.String("goal", "", "Restrict to one goal")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	result, err := a.goalClient().ListPendingGoalWorkflows(context.Background(), connect.NewRequest(&apipb.ListPendingGoalWorkflowsRequest{GoalName: strings.TrimSpace(*goal)}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return printGoalJSON(result.Msg, true)
	}
	if len(result.Msg.Pending) == 0 {
		fmt.Println("No goal workflow results are awaiting application.")
		return nil
	}
	fmt.Printf("Pending goal workflow results: %d\n", len(result.Msg.Pending))
	for _, record := range result.Msg.Pending {
		target := record.GoalName
		if record.Milestone != "" {
			target += "/" + record.Milestone
		}
		fmt.Printf("  %s  %s  %s\n", target, record.Transition, record.ExecutionId)
		if record.Stale {
			fmt.Printf("      STALE — the goal changed after the run started; re-run it\n")
		}
		if record.LastError != "" {
			fmt.Printf("      last error (attempt %d, %s): %s\n", record.Attempts, record.LastAttemptAt, record.LastError)
		}
	}
	return nil
}

// cmdGoalsWorkflowApply applies one terminal result on demand. The sweeper does
// this automatically; this command exists for the operator who does not want to
// wait for the next tick, and to surface the reason when one will not apply.
func (a *App) cmdGoalsWorkflowApply(args []string) error {
	fs := flag.NewFlagSet("goals workflow-apply", flag.ContinueOnError)
	goal, executionID := fs.String("goal", "", "Goal name"), fs.String("execution-id", "", "Workflow execution id")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("goal", *goal, "execution-id", *executionID); err != nil {
		return err
	}
	result, err := a.goalClient().ApplyGoalWorkflow(context.Background(), connect.NewRequest(&apipb.ApplyGoalWorkflowRequest{GoalName: strings.TrimSpace(*goal), ExecutionId: strings.TrimSpace(*executionID)}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return printGoalJSON(result.Msg, true)
	}
	if result.Msg.AlreadyApplied {
		fmt.Printf("Already applied: %s (session %s)\n", result.Msg.ExecutionId, result.Msg.SessionId)
		return nil
	}
	fmt.Printf("Applied %s — outcome %s, session %s, %d proposal(s)\n", result.Msg.ExecutionId, result.Msg.Outcome, result.Msg.SessionId, len(result.Msg.ProposalIds))
	return nil
}
