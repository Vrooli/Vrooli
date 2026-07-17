package onboard

import (
	"vrooli-bridge/internal/module"

	onboardconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/onboard/onboard_v1connect"
)

// Endpoints is the machine-readable description of the onboard module's public
// surface. Connect-RPC method paths reference the generated *Procedure
// constants, so renaming an RPC in onboard.proto breaks this file at compile
// time. The global parity test (TestProtoConnectParity) asserts every rpc has
// exactly one entry here once onboard is listed in AllProtoFiles().
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "onboard_start_onboarding",
		Path:        onboardconnect.OnboardServiceStartOnboardingProcedure,
		Method:      "POST",
		Summary:     "Onboard a raw SSH host into the fleet (one-shot, durable)",
		Description: "Drives a raw SSH-reachable host from bare OS to a paired, ONLINE, auto-starting fleet agent in one durable, server-owned op (SSH first-touch → push bootstrap script → remote bootstrap → verify online). The owner's SSH password is used once and never persisted; the pairing code is issued server-side and injected over stdin. target_revision defaults to the control plane's current commit (pass \"@cp\" to say so explicitly); the resolved commit is preflighted against the clone remote, so an unpushed commit is rejected with push-first guidance. Honours X-Dry-Run. Owner-gated.",
		Category:    "onboard",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"host":                   "string (required)",
			"port":                   "int32",
			"user":                   "string",
			"ssh_password":           "string",
			"node_name":              "string",
			"target_revision":        "string (defaults to the control plane's commit; \"@cp\" = same)",
			"repo_url":               "string",
			"checkout_dir":           "string",
			"control_plane_url":      "string (required unless BRIDGE_CONTROL_PLANE_URL is set)",
			"capabilities":           "array<string>",
			"verify_timeout_seconds": "int32",
			"skip_setup":             "bool",
			"skip_prereqs":           "bool",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"op_id":   "string",
			"dry_run": "bool",
		}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing host/control_plane_url, or an invalid revision (e.g. a relative ref like HEAD~1)"},
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 412, Code: "failed_precondition", Description: "The resolved commit is not on the clone remote (push it first) or the control plane's commit could not be determined"},
			{Status: 500, Code: "internal", Description: "Op persistence failure"},
		},
		Examples: []module.Example{
			{Name: "Onboard a host", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.onboard.OnboardService/StartOnboarding -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{\"host\":\"web-01.example.com\",\"ssh_password\":\"<pw>\",\"node_name\":\"web-01\",\"target_revision\":\"a1b2c3d\",\"control_plane_url\":\"https://cp.example.com\"}'"},
		},
	},
	{
		ID:          "onboard_get_onboarding",
		Path:        onboardconnect.OnboardServiceGetOnboardingProcedure,
		Method:      "POST",
		Summary:     "Get an onboarding op by id with its step-event history",
		Description: "Returns one durable onboarding op and its full persisted step-event history (orchestrator phases + every bootstrap step). Re-attaching after a client disconnect is just calling this again. Owner-gated.",
		Category:    "onboard",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"op": "OnboardingOp", "events": "array<OnboardingStepEvent>"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 404, Code: "not_found", Description: "No op with that id"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "Get onboarding op", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.onboard.OnboardService/GetOnboarding -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{\"id\":\"op-123\"}'"},
		},
	},
	{
		ID:          "onboard_list_onboardings",
		Path:        onboardconnect.OnboardServiceListOnboardingsProcedure,
		Method:      "POST",
		Summary:     "List onboarding ops",
		Description: "Returns onboarding ops newest-first, optionally filtered by host. Owner-gated.",
		Category:    "onboard",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"host": "string", "limit": "int32"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"ops": "array<OnboardingOp>"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "List onboarding ops", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.onboard.OnboardService/ListOnboardings -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "onboard_wait_onboarding",
		Path:        onboardconnect.OnboardServiceWaitOnboardingProcedure,
		Method:      "POST",
		Summary:     "Block-once wait for an onboarding op to finish",
		Description: "Blocks server-side and returns EXACTLY ONCE when the op reaches a terminal state (no polling). Returns timed_out=true if the wait deadline elapses first. Owner-gated.",
		Category:    "onboard",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string (required)", "timeout_seconds": "int64"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"op": "OnboardingOp", "timed_out": "bool"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 404, Code: "not_found", Description: "No op with that id"},
			{Status: 499, Code: "canceled", Description: "Client cancelled the wait"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "Wait for onboarding op", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.onboard.OnboardService/WaitOnboarding -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{\"id\":\"op-123\",\"timeout_seconds\":1800}'"},
		},
	},
	{
		ID:          "onboard_cancel_onboarding",
		Path:        onboardconnect.OnboardServiceCancelOnboardingProcedure,
		Method:      "POST",
		Summary:     "Cancel a running onboarding op",
		Description: "Requests cancellation of a non-terminal op; the orchestrator aborts the in-flight remote work at the next boundary and drives the op to CANCELLED. Cancelling a terminal op is a no-op. Owner-gated.",
		Category:    "onboard",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"op": "OnboardingOp"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 404, Code: "not_found", Description: "No op with that id"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "Cancel onboarding op", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.onboard.OnboardService/CancelOnboarding -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{\"id\":\"op-123\"}'"},
		},
	},
	{
		ID:          "onboard_remove_failed_onboarding",
		Path:        onboardconnect.OnboardServiceRemoveFailedOnboardingProcedure,
		Method:      "POST",
		Summary:     "Remove a failed onboarding attempt",
		Description: "Permanently removes a FAILED onboarding operation and its local diagnostic history. It never contacts the target machine and cannot remove a live, cancelled, or successfully paired node. Owner-gated.",
		Category:    "onboard",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string (required)"}},
		Response:    &module.Schema{Type: "object"},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing id or operation is not failed"},
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 404, Code: "not_found", Description: "No op with that id"},
			{Status: 500, Code: "internal", Description: "Repository delete failure"},
		},
		Examples: []module.Example{
			{Name: "Remove failed onboarding", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.onboard.OnboardService/RemoveFailedOnboarding -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{\"id\":\"op-123\"}'"},
		},
	},
}
