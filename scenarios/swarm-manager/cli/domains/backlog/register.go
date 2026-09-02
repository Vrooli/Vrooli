package backlog

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/shared"
	"swarm-manager/cli/internal/support"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "backlog",
		Description: "Backlog item management",
		Subcommands: []cliapp.Command{
			listCommand(),
			support.APICommand("pending-questions", "List pending independent-review questions [--source review] [--limit N] [--milestone NAME] [--brief]", deps.BacklogPendingQuestions),
			getCommand(),
			criteriaSetCommand(),
			support.APICommand("create", "Create a backlog item (--data JSON)", deps.BacklogCreate),
			support.APICommand("update", "Update a backlog item (--kind KIND --name NAME --data JSON)", deps.BacklogUpdate),
			deleteCommand(),
			dismissCommand(),
			support.APICommand("plan-workshop", "Open the Plan Workshop for a backlog item (--kind KIND --name NAME [--start-review])", deps.BacklogPlanWorkshop),
			support.APICommand("recreate", "Archive and clone a backlog item with lineage (--kind KIND --name NAME) [--json]", deps.BacklogRecreate),
			support.APICommand("reset-artifacts", "Remove selected derived item artifacts (--kind KIND --name NAME --scope SCOPE,... ) [--json]", deps.BacklogResetArtifacts),
			support.APICommand("files", "List backlog item files (--kind KIND --name NAME)", deps.BacklogFiles),
			support.APICommand("file-get", "Get a file from a backlog item (--kind KIND --name NAME --path PATH)", deps.BacklogFileGet),
			support.APICommand("file-upload", "Upload a file to a backlog item (--kind KIND --name NAME --path PATH --file FILE|--content CONTENT)", deps.BacklogFileUpload),
			support.APICommand("process-preflight", "Check processing readiness (--kind KIND --name NAME)", deps.BacklogProcess),
			support.APICommand("queue", "Preview/queue a backlog item (--kind KIND --name NAME [--execute] [--force])", deps.BacklogQueue),
			support.APICommand("plan-accept", "Accept the current canonical execution plan (--kind KIND --name NAME --actor ACTOR) [--json]", deps.BacklogPlanAccept),
			support.APICommand("batch-create", "Batch create backlog items from a plan file (--file items.json [--preview])", deps.BacklogBatchCreate),
			support.APICommand("batch-queue", "Batch queue backlog items (--items kind/name,kind/name [--execute] [--force] [--mode MODE])", deps.BacklogBatchQueue),
			support.APICommand("export", "Export backlog items to markdown for offline editing", deps.BacklogExport),
			support.APICommand("reconcile-counts", "Compare record-derived overview/export counts and label event-derived statistics", deps.BacklogReconcileCounts),
			support.APICommand("import", "Import edited markdown back into the backlog (--file FILE)", deps.BacklogImport),
			reviewDecideCommand(),
			support.APICommand("recover-review", "Recover an item stranded in_review with no live review round → review_pending (default) or backlog (--kind KIND --name NAME [--to review_pending|backlog] [--rationale MSG])", deps.BacklogRecoverReview),
			support.APICommand("retry", "Re-dispatch the latest terminal execution as a NEW attempt (--kind KIND --name NAME [--note MSG])", deps.BacklogRetry),
			support.APICommand("search-ai", "Semantic search over backlog items (<query> [--limit N] [--kind K,...] [--status S,...] [--milestone N] [--include-archived] [--json])", deps.BacklogSearchAI),
		},
	}
}

func reviewDecideCommand() cliapp.Command {
	cmd := cliapp.Command{Name: "review-decide", NeedsAPI: true, Description: "Decide review (--kind KIND --name NAME --round N --decided-by ACTOR --accept|--fail|--drop|--followup [--rationale MSG])", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "kind", Required: true}, {Name: "name", Required: true}, {Name: "round", Required: true}, {Name: "decided-by", Required: true}, {Name: "accept", Bool: true}, {Name: "fail", Bool: true}, {Name: "drop", Bool: true}, {Name: "followup", Bool: true}, {Name: "steering"}, {Name: "disposition", Values: []string{"follow_up_run", "replan", "new_items"}}, {Name: "rationale"}}}}
	return cmd.WithPrimitive(cliapp.ProtoMutation(
		func(op cliapp.OperationContext) (*apipb.DecideAttemptResponse, error) {
			decision := ""
			for _, candidate := range []string{"accept", "fail", "drop", "followup"} {
				if op.BoolFlag(candidate) {
					if decision != "" {
						return nil, fmt.Errorf("exactly one decision flag is required")
					}
					decision = candidate
				}
			}
			if decision == "" {
				return nil, fmt.Errorf("exactly one decision flag is required")
			}
			kind, name := strings.TrimSpace(op.Flag("kind")), strings.TrimSpace(op.Flag("name"))
			round, err := strconv.ParseUint(strings.TrimSpace(op.Flag("round")), 10, 32)
			if err != nil || round == 0 {
				return nil, fmt.Errorf("--round must be a positive integer")
			}
			req := &apipb.DecideAttemptRequest{SubjectKind: "backlog-item", SubjectRef: kind + "/" + name, RoundNum: uint32(round), Actor: strings.TrimSpace(op.Flag("decided-by")), Decision: decision, Rationale: strings.TrimSpace(op.Flag("rationale"))}
			if decision == "followup" {
				req.FollowUp = &apipb.ReviewFollowUp{Steering: strings.TrimSpace(op.Flag("steering")), Disposition: strings.TrimSpace(op.Flag("disposition"))}
			}
			response, err := backlogClient(op).DecideAttempt(context.Background(), connect.NewRequest(req))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, response *apipb.DecideAttemptResponse) cliapp.MutationReport {
			return cliapp.MutationReport{Result: []string{fmt.Sprintf("Review %s: %s", response.GetDecision(), response.GetStatus())}}
		},
	))
}

func criteriaSetCommand() cliapp.Command {
	cmd := cliapp.Command{Name: "criteria-set", NeedsAPI: true, Description: "Replace typed acceptance criteria (--kind KIND --name NAME --criteria JSON)", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "kind", Description: "Backlog item kind", Required: true}, {Name: "name", Description: "Backlog item name", Required: true}, {Name: "criteria", Description: "JSON array of {gherkin,check?}", Required: true}}}}
	return cmd.WithPrimitive(cliapp.ProtoMutation(
		func(op cliapp.OperationContext) (*apipb.BacklogItemResponse, error) {
			var criteria []*sharedpb.BacklogCriterion
			if err := json.Unmarshal([]byte(op.Flag("criteria")), &criteria); err != nil {
				return nil, fmt.Errorf("--criteria must be a JSON array of criteria: %w", err)
			}
			response, err := backlogClient(op).UpdateItem(context.Background(), connect.NewRequest(&apipb.UpdateItemRequest{Kind: strings.TrimSpace(op.Flag("kind")), Name: strings.TrimSpace(op.Flag("name")), Fields: []string{"acceptance_criteria"}, Patch: &apipb.UpdateBacklogItemRequest{AcceptanceCriteria: criteria}}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, response *apipb.BacklogItemResponse) cliapp.MutationReport {
			return cliapp.MutationReport{Result: []string{fmt.Sprintf("Stored %d acceptance criteria.", len(response.GetItem().GetAcceptanceCriteria()))}}
		},
	))
}

func listCommand() cliapp.Command {
	cmd := cliapp.Command{
		Name:        "list",
		NeedsAPI:    true,
		Description: "List backlog items [--kind KIND] [--actor-id PROFILE]",
		Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
			{Name: "kind", Description: "Comma-separated backlog kinds"},
			{Name: "status", Description: "Comma-separated backlog statuses"},
			{Name: "archived", Description: "Show archived items: true, false, or all", Values: []string{"true", "false", "all"}},
			{Name: "scenario", Description: "Comma-separated scenario names"},
			{Name: "actor-id", Description: "Verified team-member/profile identity"},
		}},
	}
	return cmd.WithPrimitive(cliapp.ProtoList(
		func(op cliapp.OperationContext) (*apipb.ListBacklogItemsResponse, error) {
			req := &apipb.ListBacklogItemsRequest{
				Kinds:     splitCSV(op.Flag("kind")),
				Statuses:  splitCSV(op.Flag("status")),
				Scenarios: splitCSV(op.Flag("scenario")),
				ActorId:   optionalString(op.Flag("actor-id")),
			}
			switch strings.TrimSpace(op.Flag("archived")) {
			case "", "false":
				req.Archived = apipb.ArchivedFilter_ARCHIVED_FILTER_EXCLUDE
			case "true":
				req.Archived = apipb.ArchivedFilter_ARCHIVED_FILTER_ONLY
			case "all":
				req.Archived = apipb.ArchivedFilter_ARCHIVED_FILTER_ALL
			default:
				return nil, fmt.Errorf("--archived must be true, false, or all")
			}
			response, err := backlogClient(op).ListItems(context.Background(), connect.NewRequest(req))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, response *apipb.ListBacklogItemsResponse) cliapp.ListReport {
			rows := make([]string, 0, len(response.GetItems()))
			for _, item := range response.GetItems() {
				rows = append(rows, fmt.Sprintf("[%s] %s — %s (priority: %d, status: %s)", item.GetKind(), item.GetName(), item.GetTitle(), item.GetPriority(), item.GetStatus()))
			}
			return cliapp.ListReport{Summary: []string{fmt.Sprintf("Backlog items: %d", len(rows))}, ResultsHeading: "Results", Results: rows}
		},
	))
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func backlogClient(op cliapp.OperationContext) apiconnect.BacklogServiceClient {
	h, base := cliapp.NewConnectHTTPClient(op.Core())
	return apiconnect.NewBacklogServiceClient(h, base)
}

func getCommand() cliapp.Command {
	cmd := cliapp.Command{
		Name:        "get",
		NeedsAPI:    true,
		Description: "Get full backlog item details (--kind KIND --name NAME) [--json]",
		Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
			{Name: "kind", Description: "Backlog item kind", Required: true},
			{Name: "name", Description: "Backlog item name", Required: true},
		}},
	}
	return cmd.WithPrimitive(cliapp.ProtoList(
		func(op cliapp.OperationContext) (*apipb.BacklogItemResponse, error) {
			response, err := backlogClient(op).GetItem(context.Background(), connect.NewRequest(&apipb.GetBacklogItemRequest{
				Kind: strings.TrimSpace(op.Flag("kind")),
				Name: strings.TrimSpace(op.Flag("name")),
			}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, response *apipb.BacklogItemResponse) cliapp.ListReport {
			item := response.GetItem()
			if item == nil {
				return cliapp.ListReport{Summary: []string{"Backlog item not found."}}
			}
			rows := []string{
				"Status: " + item.GetStatus(),
				fmt.Sprintf("Priority: %d", item.GetPriority()),
			}
			if item.GetDescription() != "" {
				rows = append(rows, "Description: "+item.GetDescription())
			}
			if len(item.GetTags()) > 0 {
				rows = append(rows, "Tags: "+strings.Join(item.GetTags(), ", "))
			}
			for _, criterion := range item.GetAcceptanceCriteria() {
				rows = append(rows, fmt.Sprintf("Criterion [%s]: %s", criterion.GetId(), criterion.GetGherkin()))
			}
			return cliapp.ListReport{Summary: []string{fmt.Sprintf("%s/%s — %s", item.GetKind(), item.GetName(), item.GetTitle())}, ResultsHeading: "Details", Results: rows}
		},
	))
}

func deleteCommand() cliapp.Command {
	cmd := cliapp.Command{Name: "delete", NeedsAPI: true, Description: "Delete a backlog item (--kind KIND --name NAME) [--json]", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "kind", Description: "Backlog item kind", Required: true},
		{Name: "name", Description: "Backlog item name", Required: true},
	}}}
	return cmd.WithPrimitive(cliapp.ProtoMutation(
		func(op cliapp.OperationContext) (*apipb.DeleteBacklogItemResponse, error) {
			response, err := backlogClient(op).DeleteItem(context.Background(), connect.NewRequest(&apipb.DeleteBacklogItemRequest{
				Kind: strings.TrimSpace(op.Flag("kind")),
				Name: strings.TrimSpace(op.Flag("name")),
			}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, response *apipb.DeleteBacklogItemResponse) cliapp.MutationReport {
			if !response.GetDeleted() {
				return cliapp.MutationReport{Result: []string{"Backlog item was already absent."}}
			}
			return cliapp.MutationReport{Result: []string{"Backlog item deleted."}}
		},
	))
}

func dismissCommand() cliapp.Command {
	cmd := cliapp.Command{
		Name:        "dismiss",
		NeedsAPI:    true,
		Description: "Dismiss a suggested auto-filer item (--kind KIND --name NAME [--reason MSG])",
		Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
			{Name: "kind", Description: "Backlog item kind", Required: true},
			{Name: "name", Description: "Backlog item name", Required: true},
			{Name: "reason", Description: "Dismissal reason"},
		}},
	}
	return cmd.WithPrimitive(cliapp.ProtoMutation(
		func(op cliapp.OperationContext) (*apipb.DismissAutoFilerSuggestionResponse, error) {
			h, base := cliapp.NewConnectHTTPClient(op.Core())
			response, err := apiconnect.NewAutoFilerServiceClient(h, base).DismissSuggestion(context.Background(), connect.NewRequest(&apipb.DismissAutoFilerSuggestionRequest{
				Kind:   strings.TrimSpace(op.Flag("kind")),
				Name:   strings.TrimSpace(op.Flag("name")),
				Reason: optionalString(op.Flag("reason")),
			}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, response *apipb.DismissAutoFilerSuggestionResponse) cliapp.MutationReport {
			item := response.GetItem()
			changes := []string{fmt.Sprintf("Dismissed %s/%s", item.GetKind(), item.GetName())}
			if item.GetArchivedAt() != "" {
				changes = append(changes, "Archived at: "+item.GetArchivedAt())
			}
			if item.GetFindingRef() != "" {
				changes = append(changes, "Finding: "+item.GetFindingRef())
			}
			return cliapp.MutationReport{Result: []string{"Suggested auto-filer item dismissed."}, Changes: changes, NextCommand: []string{"swarm-manager autofiler status", "swarm-manager backlog list --status suggested"}}
		},
	))
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
