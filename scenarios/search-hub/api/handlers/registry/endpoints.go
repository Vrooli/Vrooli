package registry

import (
	"search-hub/internal/module"

	registryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry/registry_v1connect"
)

// Endpoints is the machine-readable description of the registry module's public
// surface. Connect-RPC method paths reference the generated *Procedure
// constants, so adding or renaming an RPC in registry.proto breaks this file at
// compile time. The global parity test (TestProtoConnectParity in
// api/internal/modules/registry_test.go) walks the proto FileDescriptor and
// asserts every rpc has exactly one entry here.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "providers_register",
		Path:        registryconnect.RegistryServiceRegisterProviderProcedure,
		Method:      "POST",
		Summary:     "Register (upsert) a provider",
		Description: "Validates and persists a ProviderDescriptor (one (corpus, type) leaf). Upsert keyed by provider_id; created=false when an existing leaf was updated.",
		Category:    "registry",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"descriptor": "ProviderDescriptor (required)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"descriptor": "ProviderDescriptor",
				"created":    "bool",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Descriptor failed validation"},
			{Status: 500, Code: "internal", Description: "Registry write failure"},
		},
		Examples: []module.Example{
			{Name: "Register provider", Curl: "curl http://localhost:${API_PORT}/vrooli.search_hub.v1.registry.RegistryService/RegisterProvider -H 'Content-Type: application/json' -d '{\"descriptor\":{\"provider_id\":\"example-provider.commands\",\"provider_group\":\"example-provider\",\"bucket\":\"BUCKET_DO\",\"type\":\"command\",\"description\":\"CLI command corpus\"}}'"},
		},
	},
	{
		ID:          "providers_list",
		Path:        registryconnect.RegistryServiceListProvidersProcedure,
		Method:      "POST",
		Summary:     "List registered providers",
		Description: "Returns registered provider descriptors, optionally filtered by bucket, type, and/or state. Includes capability_gap stubs.",
		Category:    "registry",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"bucket": "Bucket (optional filter)",
				"type":   "string (optional filter)",
				"state":  "ProviderState (optional filter)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"providers": "array<ProviderDescriptor>",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Registry read failure"},
		},
		Examples: []module.Example{
			{Name: "List providers", Curl: "curl http://localhost:${API_PORT}/vrooli.search_hub.v1.registry.RegistryService/ListProviders -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "embedding_migration_execute",
		Path:        registryconnect.RegistryServiceExecuteEmbeddingMigrationProcedure,
		Method:      "POST",
		Summary:     "Execute an authenticated provider embedding migration action",
		Description: "Proxies a shadow, status, cutover, or rollback action through the provider's declared reindex endpoint while retaining the provider control token inside Search Hub.",
		Category:    "registry",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"provider_id": "string (required)", "action": "string (shadow|status|cutover|rollback)", "job_id": "string (status only)",
			"shadow_collection": "string", "rollback_collection": "string", "embedding_model": "string", "embedding_role": "string", "embedding_dimensions": "int32", "embedding_policy_schema_version": "string", "scope": "string", "dry_run": "boolean",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"job_id": "string", "state": "string", "planned_upserts": "int32", "planned_deletes": "int32", "processed": "int32", "total": "int32", "error": "string"}},
		Errors:   []module.ErrorDesc{{Status: 500, Code: "internal", Description: "Provider control endpoint failed"}},
	},
	{
		ID:          "providers_remove",
		Path:        registryconnect.RegistryServiceDeregisterProviderProcedure,
		Method:      "POST",
		Summary:     "Deregister a provider",
		Description: "Removes the provider leaf with the given provider_id. Idempotent: removed=false when no such leaf existed.",
		Category:    "registry",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"provider_id": "string (required)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"removed": "bool",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Registry delete failure"},
		},
		Examples: []module.Example{
			{Name: "Deregister provider", Curl: "curl http://localhost:${API_PORT}/vrooli.search_hub.v1.registry.RegistryService/DeregisterProvider -H 'Content-Type: application/json' -d '{\"provider_id\":\"example-provider.commands\"}'"},
		},
	},
	{
		ID:          "maturity_targets_list",
		Path:        registryconnect.RegistryServiceListMaturityTargetsProcedure,
		Method:      "POST",
		Summary:     "List Search Hub maturity targets",
		Description: "Returns the descriptor/capability scenario set used by Search Hub maturity scans, including scenarios without a registered provider leaf.",
		Category:    "registry",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"targets": "array<MaturityTarget>"}},
		Errors:      []module.ErrorDesc{{Status: 500, Code: "internal", Description: "Maturity target discovery failure"}},
	},
}
