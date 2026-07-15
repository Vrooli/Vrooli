package agentoperations

import (
	"swarm-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

// Register exposes the read-only agent-operations diagnostics surface: resolve a
// binding, dry-run-validate an invocation, and inspect the durable workflow and
// immutable execution provenance. Every subcommand is a thin Connect client over
// AgentOperationsService.
func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "agent-operations",
		Description: "Declarative agent-operations diagnostics (bindings, workflows, provenance)",
		Subcommands: []cliapp.Command{
			support.APICommand("resolve-binding", "Resolve the winning binding for an operation on a target (--target-kind KIND --target REF --operation OP [--version V]) [--json]", deps.AgentOpsResolveBinding),
			support.APICommand("validate", "Dry-run-validate an invocation without starting it (--target-kind KIND --target REF --operation OP [--version V]) [--json]", deps.AgentOpsValidateInvocation),
			support.APICommand("inspect-workflow", "Inspect the durable workflow instance for a target (--target-kind KIND --target REF) [--json]", deps.AgentOpsInspectWorkflow),
			support.APICommand("inspect-execution", "Inspect an execution's pinned provenance + reproducibility (--target-kind KIND --target REF --execution ID) [--json]", deps.AgentOpsInspectExecution),
		},
	}
}
