// Runtime switch for AUTH-01 CreateRun caller enforcement.
package owneridentity

import (
	"os"
	"strings"
)

// CreateRunCallerEnforceEnv is the single runtime switch for AUTH-01 caller
// enforcement. It is off unless set to 1, true, yes or on (operator decision
// P-18, docs/agent-system/DELIVERY_RELIABILITY.md). It is the ONE switch read;
// owners that honor it pass CreateRunCallerEnforced() as the `enforced` argument
// to the single AuthorizeCreateRunCaller / RequireCreateRunCaller /
// CarryCreateRunCaller family in create_run_caller.go. Owners that always
// enforce pass true. There is no separate "Admitted" wrapper family.
const CreateRunCallerEnforceEnv = "VROOLI_AUTH01_ENFORCE"

// CreateRunCallerEnforced reports the switch. It is read on each call so an
// owner never caches a stale value across its own reconfiguration or tests.
func CreateRunCallerEnforced() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(CreateRunCallerEnforceEnv))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}
