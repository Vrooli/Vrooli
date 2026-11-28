package domain

import (
	"os"
	"strings"
)

// DeliveryEffortEnforceEnv switches the exact delivery-effort gate. The gate
// refuses a code.economy.delivery run whose requested effort differs from the
// resource-owned declaration for the resolved model. It is part of the
// rollout-held AUTH-01/Marketing source work (AUTH01_CALLER_MIGRATION.md), so
// it is OFF by default, like AUTH-01 caller enforcement (P-18): the shared
// high-effort delivery orchestrator profile and its parked runs keep working.
// Resource effort evidence is still resolved and retained in the snapshot.
const DeliveryEffortEnforceEnv = "VROOLI_DELIVERY_EFFORT_ENFORCE"

// DeliveryEffortEnforced reports the switch. It is read on each call so an
// owner never caches a stale value across reconfiguration or tests.
func DeliveryEffortEnforced() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(DeliveryEffortEnforceEnv))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}
