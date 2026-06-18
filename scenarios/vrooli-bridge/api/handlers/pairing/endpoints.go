package pairing

import (
	"vrooli-bridge/internal/module"

	pairingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/pairing/pairing_v1connect"
)

// Endpoints describes the PairingService surface (OT-P0-002). IssuePairingCode/
// ApprovePairing/ListPairingRequests are owner-gated; RedeemPairingCode and
// RequestPairing are open node-facing calls authed by the code / pending owner
// approval. The global parity test asserts every rpc has exactly one entry once
// pairing is listed in AllProtoFiles().
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "pairing_issue_code",
		Path:        pairingconnect.PairingServiceIssuePairingCodeProcedure,
		Method:      "POST",
		Summary:     "Issue a pairing code",
		Description: "Mints a single-use, short-TTL pairing code and returns the control-plane public key to pin. Owner-gated. The plaintext code is returned once.",
		Category:    "pairing",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"name":        "string",
			"scopes":      "array<string>",
			"ttl_seconds": "int64",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"code":                     "string (shown once)",
			"control_plane_public_key": "string (base64 Ed25519)",
			"expires_at":               "Timestamp",
		}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 500, Code: "internal", Description: "Code generation/persistence failure"},
		},
		Examples: []module.Example{
			{Name: "Issue a code", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.pairing.PairingService/IssuePairingCode -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{\"name\":\"mac-mini\",\"scopes\":[\"scenario test*\"]}'"},
		},
	},
	{
		ID:          "pairing_redeem_code",
		Path:        pairingconnect.PairingServiceRedeemPairingCodeProcedure,
		Method:      "POST",
		Summary:     "Redeem a pairing code",
		Description: "Node-facing bootstrap: redeems a still-valid code, registers the node, stores its Ed25519 public key, and returns the node id + control-plane public key to pin. Open (authed by possessing the code).",
		Category:    "pairing",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"code":            "string (required)",
			"node_public_key": "string (base64 Ed25519, required)",
			"name":            "string",
			"os":              "string",
			"arch":            "string",
			"endpoint":        "string",
			"capabilities":    "array<string>",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"node_id":                  "string",
			"control_plane_public_key": "string (base64 Ed25519)",
		}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing/invalid code or public key"},
			{Status: 401, Code: "unauthenticated", Description: "Unknown, expired, or already-redeemed code"},
			{Status: 500, Code: "internal", Description: "Registration/persistence failure"},
		},
		Examples: []module.Example{
			{Name: "Redeem a code", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.pairing.PairingService/RedeemPairingCode -H 'Content-Type: application/json' -d '{\"code\":\"...\",\"node_public_key\":\"<base64>\",\"os\":\"linux\",\"arch\":\"amd64\"}'"},
		},
	},
	{
		ID:          "pairing_request",
		Path:        pairingconnect.PairingServiceRequestPairingProcedure,
		Method:      "POST",
		Summary:     "Request pairing (no code)",
		Description: "Node-facing fallback when there is no pre-shared code: records a pending join request the owner approves. Open.",
		Category:    "pairing",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"node_public_key": "string (base64 Ed25519, required)",
			"name":            "string",
			"os":              "string",
			"arch":            "string",
			"endpoint":        "string",
			"capabilities":    "array<string>",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"request_id": "string",
			"status":     "PairingRequestStatus",
		}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Invalid public key"},
			{Status: 500, Code: "internal", Description: "Persistence failure"},
		},
		Examples: []module.Example{
			{Name: "Request pairing", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.pairing.PairingService/RequestPairing -H 'Content-Type: application/json' -d '{\"node_public_key\":\"<base64>\",\"name\":\"ci-box\"}'"},
		},
	},
	{
		ID:          "pairing_approve",
		Path:        pairingconnect.PairingServiceApprovePairingProcedure,
		Method:      "POST",
		Summary:     "Approve/reject a pairing request",
		Description: "Decides a pending join request — approving mints the node and stores its credential. Owner-gated.",
		Category:    "pairing",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"request_id": "string (required)",
			"approve":    "boolean",
			"scopes":     "array<string>",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"node_id": "string (set when approved)",
			"status":  "PairingRequestStatus",
		}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 404, Code: "not_found", Description: "Unknown request"},
			{Status: 409, Code: "failed_precondition", Description: "Request already decided"},
			{Status: 500, Code: "internal", Description: "Registration/persistence failure"},
		},
		Examples: []module.Example{
			{Name: "Approve a request", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.pairing.PairingService/ApprovePairing -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{\"request_id\":\"...\",\"approve\":true,\"scopes\":[\"scenario test*\"]}'"},
		},
	},
	{
		ID:          "pairing_list_requests",
		Path:        pairingconnect.PairingServiceListPairingRequestsProcedure,
		Method:      "POST",
		Summary:     "List pairing requests",
		Description: "Returns pending join requests awaiting owner approval (include_decided adds decided ones). Owner-gated.",
		Category:    "pairing",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"include_decided": "boolean"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"requests": "array<PairingRequest>"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "List pending requests", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.pairing.PairingService/ListPairingRequests -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{}'"},
		},
	},
}
