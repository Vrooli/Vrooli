package backlog

import (
	"swarm-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "backlog",
		Description: "Backlog item management",
		Subcommands: []cliapp.Command{
			support.APICommand("list", "List backlog items [--kind KIND]", deps.BacklogList),
			support.APICommand("pending-questions", "List pending independent-review questions [--source review] [--limit N] [--milestone NAME] [--brief]", deps.BacklogPendingQuestions),
			support.APICommand("get", "Get full backlog item details (--kind KIND --name NAME)", deps.BacklogGet),
			support.APICommand("create", "Create a backlog item (--data JSON)", deps.BacklogCreate),
			support.APICommand("update", "Update a backlog item (--kind KIND --name NAME --data JSON)", deps.BacklogUpdate),
			support.APICommand("delete", "Delete a backlog item (--kind KIND --name NAME)", deps.BacklogDelete),
			support.APICommand("dismiss", "Dismiss a suggested auto-filer item (--kind KIND --name NAME [--reason MSG])", deps.BacklogDismiss),
			support.APICommand("plan-workshop", "Open the Plan Workshop for a backlog item (--kind KIND --name NAME [--start-review])", deps.BacklogPlanWorkshop),
			support.APICommand("recreate", "Archive and clone a backlog item with lineage (--kind KIND --name NAME) [--json]", deps.BacklogRecreate),
			support.APICommand("reset-artifacts", "Remove selected derived item artifacts (--kind KIND --name NAME --scope SCOPE,... ) [--json]", deps.BacklogResetArtifacts),
			support.APICommand("files", "List backlog item files (--kind KIND --name NAME)", deps.BacklogFiles),
			support.APICommand("file-get", "Get a file from a backlog item (--kind KIND --name NAME --path PATH)", deps.BacklogFileGet),
			support.APICommand("file-upload", "Upload a file to a backlog item (--kind KIND --name NAME --path PATH --file FILE|--content CONTENT)", deps.BacklogFileUpload),
			support.APICommand("process-preflight", "Check processing readiness (--kind KIND --name NAME)", deps.BacklogProcess),
			support.APICommand("queue", "Preview/queue a backlog item (--kind KIND --name NAME [--execute] [--force])", deps.BacklogQueue),
			support.APICommand("batch-create", "Batch create backlog items from a plan file (--file items.json [--preview])", deps.BacklogBatchCreate),
			support.APICommand("batch-queue", "Batch queue backlog items (--items kind/name,kind/name [--execute] [--force] [--mode MODE])", deps.BacklogBatchQueue),
			support.APICommand("export", "Export backlog items to markdown for offline editing", deps.BacklogExport),
			support.APICommand("import", "Import edited markdown back into the backlog (--file FILE)", deps.BacklogImport),
			support.APICommand("review-decide", "Decide the terminal status of a review_pending item (--kind KIND --name NAME --accept|--fail|--followup [--rationale MSG])", deps.BacklogReviewDecide),
			support.APICommand("recover-review", "Recover an item stranded in_review with no live review round → review_pending (default) or backlog (--kind KIND --name NAME [--to review_pending|backlog] [--rationale MSG])", deps.BacklogRecoverReview),
			support.APICommand("retry", "Re-dispatch the latest terminal execution as a NEW attempt (--kind KIND --name NAME [--note MSG])", deps.BacklogRetry),
			support.APICommand("search-ai", "Semantic search over backlog items (<query> [--limit N] [--kind K,...] [--status S,...] [--milestone N] [--include-archived] [--json])", deps.BacklogSearchAI),
		},
	}
}
