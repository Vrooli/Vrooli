package agentoperations

import (
	"swarm-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

// Register exposes the agent-operations operator surface: read-only projections
// (catalog, compatibility, resolved bindings, workflow, execution history,
// migration status), the mutating binding-override administration
// (overrides set/clear), and the idempotent reconciliation sweep. Every
// subcommand is a thin Connect client over AgentOperationsService; the server
// owns every decision (precedence, compatibility, digests).
func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "agent-operations",
		Description: "Declarative agent-operations diagnostics + binding controls (catalog, bindings, workflows, provenance)",
		Subcommands: []cliapp.Command{
			support.APICommand("resolve-binding", "Resolve the winning binding for an operation on a target (--target-kind KIND --target REF --operation OP [--version V]) [--json]", deps.AgentOpsResolveBinding),
			support.APICommand("validate", "Dry-run-validate an invocation without starting it (--target-kind KIND --target REF --operation OP [--version V]) [--json]", deps.AgentOpsValidateInvocation),
			support.APICommand("inspect-workflow", "Inspect the durable workflow instance for a target (--target-kind KIND --target REF) [--json]", deps.AgentOpsInspectWorkflow),
			support.APICommand("inspect-execution", "Inspect an execution's pinned provenance + reproducibility (--target-kind KIND --target REF --execution ID) [--json]", deps.AgentOpsInspectExecution),
			support.APICommand("catalog", "List every authored operation contract with revision + compatible target kinds [--json]", deps.AgentOpsCatalog),
			support.APICommand("compatible-modes", "List authored modes with per-operation compatibility verdicts for a target (--target-kind KIND --target REF [--operation OP]) [--json]", deps.AgentOpsCompatibleModes),
			support.APICommand("bindings", "Resolve the winning binding per catalog operation for a target, with source layer (--target-kind KIND --target REF [--verbose]) [--json]", deps.AgentOpsBindings),
			support.APICommand("overrides", "Administer binding overrides: list|set|clear (--owner-kind KIND --owner REF; set/clear MUTATE the owner's layer) [--json]", deps.AgentOpsOverrides),
			support.APICommand("workflow", "Canonical workflow projection: state, ops with mode@rev/layer/attempt, decisions, legal actions (--target-kind KIND --target REF) [--json]", deps.AgentOpsWorkflow),
			support.APICommand("history", "Execution provenance summaries for a target, newest first (--target-kind KIND --target REF [--limit N]) [--json]", deps.AgentOpsHistory),
			support.APICommand("migration-status", "Read the persisted-state migration status document (Phase-8 migrator writes it) [--json]", deps.AgentOpsMigrationStatus),
			support.APICommand("reconcile", "Run the idempotent orphan-snapshot reconciliation sweep (same as the startup sweep; MUTATES by reaping aged orphans) [--json]", deps.AgentOpsReconcile),
		},
	}
}
