package initiatives

import (
	"swarm-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "initiatives",
		Description: "Initiative management",
		Subcommands: []cliapp.Command{
			support.APICommand("list", "List initiatives [--json]", deps.InitiativesList),
			support.APICommand("get", "Get initiative details (--name NAME) [--json]", deps.InitiativesGet),
			support.APICommand("context", "Get an initiative with its member items + related initiatives (--name NAME) [--json]", deps.InitiativesContext),
			support.APICommand("candidates", "Get bounded initiative next-action candidates [--purpose next-action] [--json]", deps.InitiativesCandidates),
			support.APICommand("create", "Create initiative (--data JSON) [--json]", deps.InitiativesCreate),
			support.APICommand("update", "Update initiative (--name NAME --data JSON) [--json]", deps.InitiativesUpdate),
			support.APICommand("delete", "Delete initiative (--name NAME)", deps.InitiativesDelete),
			support.APICommand("add-items", "Add items to initiative (--name NAME --items kind/name,...) [--json]", deps.InitiativesAddItems),
			support.APICommand("remove-items", "Remove items from initiative (--name NAME --items kind/name,...) [--json]", deps.InitiativesRemove),
			support.APICommand("files", "List files in an initiative (--name NAME) [--json]", deps.InitiativesFiles),
			support.APICommand("file-get", "Get a file from an initiative (--name NAME --path PATH) [--out FILE] [--json]", deps.InitiativesFileGet),
			support.APICommand("file-upload", "Upload a file to an initiative (--name NAME --path PATH) (--stdin|--file|--content)", deps.InitiativesFileUp),
			support.APICommand("file-op", "File operation on initiative (--name NAME --op OP --source PATH) [--dest PATH] [--json]", deps.InitiativesFileOp),
			support.APICommand("search-ai", "Semantic search over initiatives (<query> [--limit N] [--status S,...] [--include-archived] [--json])", deps.InitiativesSearchAI),

			// Feedback rounds (user signal → proposed mutations → decision).
			support.APICommand("feedback-list", "List feedback rounds on an initiative (--name NAME) [--json]", deps.InitiativesFeedbackList),
			support.APICommand("feedback-get", "Get a feedback round (--name NAME --round N) [--json]", deps.InitiativesFeedbackGet),
			support.APICommand("feedback-submit", "Start a feedback round (--name NAME --type feedback|note --text MSG [--file PATH ...] [--slug SLUG] [--override] [--decided-by WHO] [--json])", deps.InitiativesFeedbackSubmit),
			support.APICommand("feedback-continue", "Continue a feedback round (--name NAME --round N --text MSG [--file PATH ...] [--decided-by WHO] [--json])", deps.InitiativesFeedbackContinue),
			support.APICommand("feedback-decide", "Decide a feedback round (--name NAME --round N (--accept|--reject|--dismiss) [--mutations m1,m3] [--rationale MSG] [--decided-by WHO] [--json])", deps.InitiativesFeedbackDecide),
			support.APICommand("feedback-cancel", "Cancel an in-flight feedback round, stopping the agent run (--name NAME --round N [--rationale MSG] [--decided-by WHO] [--json])", deps.InitiativesFeedbackCancel),
			support.APICommand("feedback-delete", "Delete a terminal feedback round permanently (--name NAME --round N)", deps.InitiativesFeedbackDelete),
			support.APICommand("feedback-lock", "Show feedback lock status (--name NAME) [--json]", deps.InitiativesFeedbackLock),

			// Initiative review.
			support.APICommand("review-list", "List initiative review rounds (--name NAME) [--json]", deps.InitiativesReviewList),
			support.APICommand("review-get", "Get a review round (--name NAME --round N) [--json]", deps.InitiativesReviewGet),
			support.APICommand("review-trigger", "Trigger a review round (--name NAME) [--json]", deps.InitiativesReviewTrigger),
			support.APICommand("review-decide", "Decide an initiative review (--name NAME (--accept|--fail|--followup) [--rationale MSG] [--decided-by WHO] [--json])", deps.InitiativesReviewDecide),
			support.APICommand("review-decisions", "List past review decisions (--name NAME) [--json]", deps.InitiativesReviewDecisions),

			// Materialized item-graph projection.
			support.APICommand("graph-show", "Show materialized graph.json for an initiative (--name NAME) [--json]", deps.InitiativesGraphShow),
		},
	}
}
