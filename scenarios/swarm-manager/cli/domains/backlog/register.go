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
			support.APICommand("pending-questions", "List pending workshop/review questions [--source workshop|review|all] [--limit N] [--initiative NAME]", deps.BacklogPendingQuestions),
			support.APICommand("get", "Get full backlog item details (--kind KIND --name NAME)", deps.BacklogGet),
			support.APICommand("create", "Create a backlog item (--data JSON)", deps.BacklogCreate),
			support.APICommand("update", "Update a backlog item (--kind KIND --name NAME --data JSON)", deps.BacklogUpdate),
			support.APICommand("delete", "Delete a backlog item (--kind KIND --name NAME)", deps.BacklogDelete),
			support.APICommand("workshop-reset", "Reset all workshop data for a backlog item (--kind KIND --name NAME)", deps.BacklogWorkshopReset),
			support.APICommand("re-workshop", "Reset and re-queue the workshop for a stale plan (--kind KIND --name NAME)", deps.BacklogReWorkshop),
			support.APICommand("files", "List backlog item files (--kind KIND --name NAME)", deps.BacklogFiles),
			support.APICommand("file-get", "Get a file from a backlog item (--kind KIND --name NAME --path PATH)", deps.BacklogFileGet),
			support.APICommand("file-upload", "Upload a file to a backlog item (--kind KIND --name NAME --path PATH --file FILE|--content CONTENT)", deps.BacklogFileUpload),
			support.APICommand("process-preflight", "Check processing readiness (--kind KIND --name NAME)", deps.BacklogProcess),
			support.APICommand("queue", "Preview/queue a backlog item (--kind KIND --name NAME [--execute] [--force])", deps.BacklogQueue),
			support.APICommand("research", "Spawn research agent (--kind KIND --name NAME [--data JSON])", deps.BacklogResearch),
			support.APICommand("prompt-trace", "Get latest backlog research prompt trace (--kind KIND --name NAME)", deps.BacklogPromptTrace),
			support.APICommand("batch-create", "Batch create backlog items from a plan file (--file items.json [--preview])", deps.BacklogBatchCreate),
			support.APICommand("batch-queue", "Batch queue backlog items (--items kind/name,kind/name [--execute] [--force] [--mode MODE])", deps.BacklogBatchQueue),
			support.APICommand("export", "Export backlog items to markdown for offline editing", deps.BacklogExport),
			support.APICommand("import", "Import edited markdown back into the backlog (--file FILE)", deps.BacklogImport),
			support.APICommand("clarify", "Start a decision clarification (--kind KIND --name NAME --round N --item ID [--message MSG])", deps.BacklogClarify),
			support.APICommand("clarify-get", "Get a clarification thread (--kind KIND --name NAME --thread ID)", deps.BacklogClarifyGet),
			support.APICommand("clarify-continue", "Continue a clarification thread (--kind KIND --name NAME --thread ID --message MSG)", deps.BacklogClarifyNext),
			support.APICommand("clarify-action", "Apply post-clarification action (--kind KIND --name NAME --thread ID --action ACTION)", deps.BacklogClarifyAction),
			support.APICommand("review-decide", "Decide the terminal status of a review_pending item (--kind KIND --name NAME --accept|--fail|--followup [--rationale MSG])", deps.BacklogReviewDecide),
			support.APICommand("retry", "Re-dispatch the latest terminal execution as a NEW attempt (--kind KIND --name NAME [--note MSG])", deps.BacklogRetry),
			support.APICommand("search-ai", "Semantic search over backlog items (<query> [--limit N] [--kind K,...] [--status S,...] [--initiative N] [--include-archived] [--json])", deps.BacklogSearchAI),
		},
	}
}
