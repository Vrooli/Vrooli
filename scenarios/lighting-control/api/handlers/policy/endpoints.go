package policy

import (
	policyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/policy/policyconnect"
	"lighting-control/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "policy_get", Path: policyconnect.PolicyServiceGetPolicyProcedure, Method: "POST", Summary: "Read lighting policy", Description: "Returns the persisted policy and current evidence-backed effective preset.", Category: "policy"},
	{ID: "policy_set", Path: policyconnect.PolicyServiceSetPolicyProcedure, Method: "POST", Summary: "Configure lighting policy", Description: "Persists policy thresholds, preset definitions, brightness limits, and stale-evidence fallback.", Category: "policy"},
	{ID: "policy_set_manual", Path: policyconnect.PolicyServiceSetManualPresetProcedure, Method: "POST", Summary: "Set a temporary manual lighting preset", Description: "Applies a selected preset at higher priority than automatic evidence until it is cleared or expires.", Category: "policy"},
	{ID: "policy_clear_manual", Path: policyconnect.PolicyServiceClearManualPresetProcedure, Method: "POST", Summary: "Clear the manual lighting preset", Description: "Clears the persisted manual policy request and resumes configured evidence-driven behavior.", Category: "policy"},
	{ID: "policy_evaluate", Path: policyconnect.PolicyServiceEvaluatePolicyProcedure, Method: "POST", Summary: "Evaluate and reconcile lighting policy", Description: "Reads current source evidence, persists evaluator state, and applies desired states to enrolled zones.", Category: "policy"},
}
