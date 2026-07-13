package records

import (
	"swarm-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

// recordsCreateHelpText is the full --help body for `records create`. Every
// enum is spelled out: transcript analysis showed the old one-line usage
// (which elided the outcome values as `[--outcome ...]`) drove the single
// largest class of failed creates — agents passing prose as the outcome.
const recordsCreateHelpText = `Flags:
  --kind K            Record kind: idea|research|fix|execute|chore (required;
                      common aliases like "feature"/"bugfix" are accepted)
  --scenario X        Target slug: a scenario/package/resource directory name,
                      or "vrooli" for repo-level work (required)
  --trigger '...'     One-line symptom/goal that started the work (required)
  --approach '...'    What was understood/built — the story future agents recall
  --evidence '...'    Validation results (test suites, baseline diffs, live checks)
  --ruled-out '...'   Hypothesis considered and rejected (repeatable)
  --commit SHA        Commit that shipped the work
  --files PATH        Repo-relative file touched (repeatable)
  --backlog-ref k/n   Backlog item this work closes (kind/name)
  --initiative-id ID  Initiative this work belongs to
  --supersedes ID     Record this one amends/replaces
  --outcome O         shipped (default) | partial | abandoned | duplicate
  --created-by ID     Author identifier (agent id or human)
  --json              Emit raw JSON

Example:
  swarm-manager records create --kind fix --scenario web-console \
    --trigger 'voice mic stayed live after UI went idle' \
    --approach 'single capture controller owns provider lifecycle; atomic replace disposes old first' \
    --ruled-out 'lease-registry leak (leases were released correctly)' \
    --evidence 'ui vitest 1261 green; scenario restart healthy' \
    --files ui/src/hooks/voiceCaptureController.ts --outcome shipped

Notes:
  - If you closed a backlog item via 'backlog review-decide --accept', a stub
    record already exists — fill it with 'records edit --id <stub-id>' instead.
  - Records are immutable once filled; amend via a new record with --supersedes.`

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "records",
		Description: "Narrative artifacts of completed work (recursive-learning write side)",
		Subcommands: []cliapp.Command{
			support.APICommand("list", "List records [--scenario X] [--kind K] [--backlog-ref kind/name] [--include-stubs] [--limit N] [--offset N] [--json]", deps.RecordsList),
			support.APICommand("get", "Get a record (--id ID) [--json]", deps.RecordsGet),
			support.APICommandHelp("create",
				"Create a record (--kind K --scenario X --trigger '...' [--approach '...'] [--evidence '...'] [--ruled-out '...']... [--commit SHA] [--files PATH]... [--backlog-ref kind/name] [--supersedes ID] [--outcome shipped|partial|abandoned|duplicate]) [--json]",
				recordsCreateHelpText, deps.RecordsCreate),
			support.APICommand("capture", "Capture a record; complete input publishes, incomplete input saves a private repairable draft [--json]", deps.RecordsCapture),
			support.APICommand("edit", "Fill a stub record's narrative (--id ID --trigger '...' --approach '...' [--evidence '...'] [--ruled-out '...']... [--commit SHA] [--files PATH]... [--outcome shipped|partial|abandoned|duplicate]) [--json]", deps.RecordsEdit),
			support.APICommand("search", "Semantic search over records ('<query>' [--kind K] [--scenario X] [--limit N]) [--json]", deps.RecordsSearch),
			support.APICommand("supersede", "Mark a record superseded by a successor (--id ID --by SUCCESSOR-ID [--reason '...']) [--json]", deps.RecordsSupersede),
		},
	}
}
