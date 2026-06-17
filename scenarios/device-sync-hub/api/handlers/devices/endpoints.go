package devices

import (
	"device-sync-hub/internal/module"

	devicesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/devices/devices_v1connect"
)

// Endpoints is the machine-readable description of the devices module's public
// surface. Paths reference the generated *Procedure constants, so renaming an
// RPC in devices.proto breaks this file at compile time. The global parity
// test (TestProtoConnectParity) asserts every rpc has exactly one entry here.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "devices_list",
		Path:        devicesconnect.DevicesServiceListDevicesProcedure,
		Method:      "POST",
		Summary:     "List trusted devices",
		Description: "Returns every device in the authenticated owner's trust group, newest first.",
		Category:    "devices",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"devices": "array<Device>"},
		},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Missing or invalid owner token"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "List devices", Curl: "curl http://localhost:${API_PORT}/vrooli.device_sync_hub.v1.devices.DevicesService/ListDevices -H 'Authorization: Bearer <owner-jwt>' -H 'Content-Type: application/json' -d '{}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "device-sync-hub devices list"},
	},
	{
		ID:          "devices_get",
		Path:        devicesconnect.DevicesServiceGetDeviceProcedure,
		Method:      "POST",
		Summary:     "Get a device by id",
		Description: "Returns one owner-scoped device.",
		Category:    "devices",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"device": "Device"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Missing or invalid owner token"},
			{Status: 404, Code: "not_found", Description: "No such device for this owner"},
		},
		Examples: []module.Example{
			{Name: "Get device", Curl: "curl http://localhost:${API_PORT}/vrooli.device_sync_hub.v1.devices.DevicesService/GetDevice -H 'Authorization: Bearer <owner-jwt>' -d '{\"id\":\"<device-id>\"}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "device-sync-hub devices get", Args: []string{"<id>"}},
	},
	{
		ID:          "devices_issue_pairing_code",
		Path:        devicesconnect.DevicesServiceIssuePairingCodeProcedure,
		Method:      "POST",
		Summary:     "Issue a pairing code",
		Description: "Mints a short-TTL, single-use pairing code (and QR payload) for a new device to redeem. Owner-authed.",
		Category:    "devices",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"device_name": "string (optional slot label)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"pairing_code": "PairingCode (raw code returned once)"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Missing or invalid owner token"},
			{Status: 500, Code: "internal", Description: "Code generation or persistence failure"},
		},
		Examples: []module.Example{
			{Name: "Issue code", Curl: "curl http://localhost:${API_PORT}/vrooli.device_sync_hub.v1.devices.DevicesService/IssuePairingCode -H 'Authorization: Bearer <owner-jwt>' -d '{\"device_name\":\"My Phone\"}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "device-sync-hub devices pair", Args: []string{"--name", "<name>"}},
	},
	{
		ID:          "devices_redeem_pairing_code",
		Path:        devicesconnect.DevicesServiceRedeemPairingCodeProcedure,
		Method:      "POST",
		Summary:     "Redeem a pairing code",
		Description: "Called by a new device presenting a valid code. Registers a TRUSTED device and returns its one-time hub device token. No owner token required.",
		Category:    "devices",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"code": "string", "profile": "DeviceProfile"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"device": "Device", "device_token": "string (returned once)"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Unknown, expired, or already-used code"},
			{Status: 500, Code: "internal", Description: "Persistence failure"},
		},
		Examples: []module.Example{
			{Name: "Redeem code", Curl: "curl http://localhost:${API_PORT}/vrooli.device_sync_hub.v1.devices.DevicesService/RedeemPairingCode -d '{\"code\":\"ABCDE-FGHIJ\",\"profile\":{\"device_name\":\"Tablet\",\"kind\":\"tablet\"}}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "device-sync-hub devices redeem", Args: []string{"--code", "<code>"}},
	},
	{
		ID:          "devices_request_pairing",
		Path:        devicesconnect.DevicesServiceRequestPairingProcedure,
		Method:      "POST",
		Summary:     "Request pairing (fallback)",
		Description: "Fallback join path for a device that cannot enter a code: registers a PENDING device and returns an inert token that activates when the owner approves.",
		Category:    "devices",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"profile": "DeviceProfile"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"device": "Device (pending)", "device_token": "string (inert until approved)"}}, //nolint:gosec // "device_token" is a doc field label in an API schema descriptor, not a hardcoded credential
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "failed_precondition", Description: "No owner yet — first device must use a code"},
			{Status: 500, Code: "internal", Description: "Persistence failure"},
		},
		Examples: []module.Example{
			{Name: "Request pairing", Curl: "curl http://localhost:${API_PORT}/vrooli.device_sync_hub.v1.devices.DevicesService/RequestPairing -d '{\"profile\":{\"device_name\":\"Tablet\"}}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "device-sync-hub devices request"},
	},
	{
		ID:          "devices_approve_pairing",
		Path:        devicesconnect.DevicesServiceApprovePairingProcedure,
		Method:      "POST",
		Summary:     "Approve a pending device",
		Description: "Promotes a PENDING device (from the request path) to TRUSTED. Owner-authed.",
		Category:    "devices",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"device_id": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"device": "Device"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Missing or invalid owner token"},
			{Status: 404, Code: "not_found", Description: "No such device"},
			{Status: 400, Code: "failed_precondition", Description: "Device is revoked"},
		},
		Examples: []module.Example{
			{Name: "Approve device", Curl: "curl http://localhost:${API_PORT}/vrooli.device_sync_hub.v1.devices.DevicesService/ApprovePairing -H 'Authorization: Bearer <owner-jwt>' -d '{\"device_id\":\"<id>\"}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "device-sync-hub devices approve", Args: []string{"<device-id>"}},
	},
	{
		ID:          "devices_rename",
		Path:        devicesconnect.DevicesServiceRenameDeviceProcedure,
		Method:      "POST",
		Summary:     "Rename a device",
		Description: "Updates a device's human-facing name. Owner-authed.",
		Category:    "devices",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"device_id": "string", "name": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"device": "Device"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Missing or invalid owner token"},
			{Status: 400, Code: "invalid_argument", Description: "Blank name"},
			{Status: 404, Code: "not_found", Description: "No such device"},
		},
		Examples: []module.Example{
			{Name: "Rename device", Curl: "curl http://localhost:${API_PORT}/vrooli.device_sync_hub.v1.devices.DevicesService/RenameDevice -H 'Authorization: Bearer <owner-jwt>' -d '{\"device_id\":\"<id>\",\"name\":\"Laptop\"}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "device-sync-hub devices rename", Args: []string{"<device-id>", "--name", "<name>"}},
	},
	{
		ID:          "devices_revoke",
		Path:        devicesconnect.DevicesServiceRevokeDeviceProcedure,
		Method:      "POST",
		Summary:     "Revoke a device",
		Description: "Immediately severs a device's access: flips it to REVOKED (hub token rejected at once) and revokes its authenticator session. Owner-authed.",
		Category:    "devices",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"device_id": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"device": "Device"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Missing or invalid owner token"},
			{Status: 404, Code: "not_found", Description: "No such device"},
		},
		Examples: []module.Example{
			{Name: "Revoke device", Curl: "curl http://localhost:${API_PORT}/vrooli.device_sync_hub.v1.devices.DevicesService/RevokeDevice -H 'Authorization: Bearer <owner-jwt>' -d '{\"device_id\":\"<id>\"}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "device-sync-hub devices revoke", Args: []string{"<device-id>"}},
	},
}
