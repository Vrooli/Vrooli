package evidence

import (
	"swarm-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

// Register exposes the canonical evidence ledger's operator workflows. These
// commands intentionally use the same run, entity, and repair vocabulary as
// Session and operating-mode surfaces rather than creating per-owner tools.
func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "evidence",
		Description: "Query and reconcile canonical run evidence",
		Subcommands: []cliapp.Command{
			support.APICommand("run", "List evidence for a verified run (--run-id ID) [--json]", deps.EvidenceRun),
			support.APICommand("entity", "List evidence for an entity (--kind KIND --id ID) [--json]", deps.EvidenceEntity),
			support.APICommand("reconcile", "Retry evidence producers for a run (--run-id ID) [--json]", deps.EvidenceReconcile),
			support.APICommand("verify", "Append an operator-verified observation (--owner-kind K --owner-id ID --event-id ID --run-id ID --subject-kind K --subject-id ID --action A --actor ID --reason TEXT) [--metadata JSON] [--json]", deps.EvidenceVerify),
		},
	}
}
