package registry

import (
	"vrooli-bridge/internal/module"

	registryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry/registry_v1connect"
)

// Endpoints is the machine-readable description of the registry module's public
// surface. Connect-RPC method paths reference the generated *Procedure
// constants, so renaming an RPC in registry.proto breaks this file at compile
// time. The global parity test (TestProtoConnectParity in
// internal/modules/registry_test.go) asserts every rpc has exactly one entry
// here once registry is listed in AllProtoFiles().
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "registry_register_node",
		Path:        registryconnect.NodeRegistryServiceRegisterNodeProcedure,
		Method:      "POST",
		Summary:     "Register a trusted node",
		Description: "Creates a durable node record (OS/arch/endpoint/capabilities/scopes). Owner-gated.",
		Category:    "registry",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"name":         "string (required)",
			"os":           "string (required)",
			"arch":         "string (required)",
			"endpoint":     "string",
			"capabilities": "array<string>",
			"scopes":       "array<string>",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"node": "Node"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing name/os/arch"},
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 500, Code: "internal", Description: "Repository write failure"},
		},
		Examples: []module.Example{
			{Name: "Register a node", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.registry.NodeRegistryService/RegisterNode -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{\"name\":\"mac-mini\",\"os\":\"darwin\",\"arch\":\"arm64\"}'"},
		},
	},
	{
		ID:          "registry_list_nodes",
		Path:        registryconnect.NodeRegistryServiceListNodesProcedure,
		Method:      "POST",
		Summary:     "List fleet nodes",
		Description: "Returns every node in the owner's fleet, newest-first, with the live presence overlay applied. Owner-gated.",
		Category:    "registry",
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"nodes": "array<Node>"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "List nodes", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.registry.NodeRegistryService/ListNodes -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "registry_get_node",
		Path:        registryconnect.NodeRegistryServiceGetNodeProcedure,
		Method:      "POST",
		Summary:     "Get a node by id",
		Description: "Returns one node by id with the presence overlay applied. Owner-gated.",
		Category:    "registry",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"node": "Node"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 404, Code: "not_found", Description: "No node with that id"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "Get node", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.registry.NodeRegistryService/GetNode -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{\"id\":\"abc123\"}'"},
		},
	},
	{
		ID:          "registry_update_node",
		Path:        registryconnect.NodeRegistryServiceUpdateNodeProcedure,
		Method:      "POST",
		Summary:     "Update a node",
		Description: "Mutates a node's owner-editable fields (name/endpoint/capabilities/scopes/revision). Owner-gated.",
		Category:    "registry",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"id":           "string (required)",
			"name":         "string (required)",
			"endpoint":     "string",
			"capabilities": "array<string>",
			"scopes":       "array<string>",
			"revision":     "string",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"node": "Node"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing id/name"},
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 404, Code: "not_found", Description: "No node with that id"},
			{Status: 500, Code: "internal", Description: "Repository write failure"},
		},
		Examples: []module.Example{
			{Name: "Update node", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.registry.NodeRegistryService/UpdateNode -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{\"id\":\"abc123\",\"name\":\"renamed\"}'"},
		},
	},
	{
		ID:          "registry_revoke_node",
		Path:        registryconnect.NodeRegistryServiceRevokeNodeProcedure,
		Method:      "POST",
		Summary:     "Revoke a node",
		Description: "Severs a node atomically — marks the record revoked and (Phase 2) destroys credentials + kills channel/job/provisioning rights. Idempotent. Owner-gated.",
		Category:    "registry",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"node": "Node"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 404, Code: "not_found", Description: "No node with that id"},
			{Status: 500, Code: "internal", Description: "Repository write failure"},
		},
		Examples: []module.Example{
			{Name: "Revoke node", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.registry.NodeRegistryService/RevokeNode -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{\"id\":\"abc123\"}'"},
		},
	},
	{
		ID: "registry_remove_node", Path: registryconnect.NodeRegistryServiceRemoveNodeProcedure, Method: "POST", Summary: "Remove a revoked node", Description: "Permanently removes a revoked registry record. Active nodes must be revoked first. Owner-gated.", Category: "registry",
		Request: &module.Schema{Type: "object", Properties: map[string]string{"id": "string"}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"removed_node_id": "string"}},
		Errors: []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Missing id"}, {Status: 401, Code: "unauthenticated", Description: "Owner token required"}, {Status: 404, Code: "not_found", Description: "No node with that id"}, {Status: 412, Code: "failed_precondition", Description: "Node must be revoked first"}},
	},
	{
		ID: "registry_get_node_readiness", Path: registryconnect.NodeRegistryServiceGetNodeReadinessProcedure, Method: "POST", Summary: "Diagnose node readiness", Description: "Returns independent registry, heartbeat, channel, protocol, and dispatchability facts. Owner-gated.", Category: "registry",
		Request: &module.Schema{Type: "object", Properties: map[string]string{"id": "string"}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"node": "Node"}},
		Errors: []module.ErrorDesc{{Status: 401, Code: "unauthenticated", Description: "Owner token required"}, {Status: 404, Code: "not_found", Description: "No node with that id"}},
	},
}
