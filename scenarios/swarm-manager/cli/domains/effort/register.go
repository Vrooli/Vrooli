package effort

import (
	"swarm-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

// Register exposes the versioned owner effort-control aggregate. Admission and
// amendment are owner-version-checked, and evidence completion stays separate
// from authenticated human product acceptance.
func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "effort",
		Description: "Versioned effort-control admission, amendment and completion",
		Subcommands: []cliapp.Command{
			support.APICommandHelp("get", "Get the current admitted effort revision (--effort-id ID) [--json]", "Get the current revision:\n  swarm-manager effort get --effort-id aquila-launch-2026-09-17 --json", deps.EffortGet),
			support.APICommandHelp("admit", "Admit the first effort revision from reviewed JSON (--effort-id ID --data JSON) [--json]", "Admit the first revision.\n  --data is the reviewed effort-control JSON object (effort_id is filled from --effort-id when absent).\n  Admitting over an existing effort is refused; use effort amend.", deps.EffortAdmit),
			support.APICommandHelp("amend", "Amend effort authority at the exact next revision (--effort-id ID --data JSON) [--json]", "Amend authority at the exact next revision.\n  --data must carry revision = current + 1. A changed target or grant must ride an amended revision.", deps.EffortAmend),
			support.APICommandHelp("evidence-complete", "Record owner evidence completion; never human acceptance (--effort-id ID [--refs a,b]) [--json]", "Mark owner evidence complete.\n  This never sets human_accepted; product acceptance requires effort accept.", deps.EffortEvidenceComplete),
			support.APICommandHelp("accept", "Record authenticated human product acceptance (--effort-id ID --actor ACTOR) [--json]", "Record authenticated human product acceptance.\n  A blank actor is refused; owner evidence completion is not acceptance.", deps.EffortAccept),
		},
	}
}
