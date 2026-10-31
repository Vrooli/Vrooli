package gateway

import (
	"ai-gateway/internal/module"

	gatewayconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/gateway/gateway_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "gateway_validate_gateway_request",
		Path:        gatewayconnect.GatewayServiceValidateGatewayRequestProcedure,
		Method:      "POST",
		Summary:     "Validate a provider-neutral gateway request",
		Description: "Validates role/profile/privacy constraints and rejects provider-specific fields before provider adapters can run.",
		Category:    "gateway",
		Request:     &module.Schema{Type: "ValidateGatewayRequestRequest", Properties: map[string]string{"request": "GatewayRequest"}},
		Response:    &module.Schema{Type: "ValidateGatewayRequestResponse", Properties: map[string]string{"valid": "bool", "issues": "array<ValidationIssue>", "accepted_profiles": "array<string>"}},
		Errors:      []module.ErrorDesc{{Status: 500, Code: "internal", Description: "Unexpected validation failure"}},
		Examples: []module.Example{
			{Name: "Validate gateway request", Curl: "curl http://localhost:${API_PORT}/vrooli.ai_gateway.v1.gateway.GatewayService/ValidateGatewayRequest -H 'Content-Type: application/json' -d '{\"request\":{\"kind\":\"REQUEST_KIND_TEXT_GENERATION\",\"role\":\"chat.default\",\"profile\":\"PROFILE_LOCAL_FIRST\",\"privacyClass\":\"PRIVACY_CLASS_INTERNAL\"}}'"},
		},
	},
}
