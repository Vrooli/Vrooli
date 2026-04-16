package execution

import (
	"swarm-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "execution",
		Description: "Execution run controls",
		Subcommands: []cliapp.Command{
			support.APICommand("list", "List execution runs", deps.ExecutionList),
			support.APICommand("get", "Get execution details (--id ID)", deps.ExecutionGet),
			support.APICommand("create", "Create execution from backlog item (--kind KIND --name NAME)", deps.ExecutionCreate),
			support.APICommand("policy-get", "Get execution policy defaults (via settings)", deps.ExecutionPolicyGet),
			support.APICommand("policy-update", "Update execution policy defaults (--mode MODE --delay-seconds N [--auto-fixup] [--max-fixup-attempts N])", deps.ExecutionPolicyPut),
			support.APICommand("prompt-trace", "Get execution prompt trace (--id ID)", deps.ExecutionPromptTrace),
			support.APICommand("start", "Start an execution (--id ID)", deps.ExecutionStart),
			support.APICommand("cancel", "Cancel an execution (--id ID)", deps.ExecutionCancel),
			support.APICommand("retry", "Retry a failed execution (--id ID)", deps.ExecutionRetry),
			support.APICommand("circuit-breaker-reset", "Reset circuit breaker for an item (--item KIND/NAME)", deps.CircuitBreakerReset),
			support.APICommand("review-list", "List review evidence rounds (--kind KIND --name NAME)", deps.ReviewList),
			support.APICommand("review-verify", "Mark evidence item as verified (--kind KIND --name NAME --round N --evidence-id ID)", deps.ReviewVerify),
			support.APICommand("review-request", "Request additional evidence (--kind KIND --name NAME --round N --message MSG)", deps.ReviewRequest),
			support.APICommand("review-trigger", "Manually trigger review agent (--id EXECUTION_ID --kind KIND --name NAME)", deps.ReviewTrigger),
		},
	}
}
