package cleanup

import (
	"io"
	"log"

	"cleanup-manager/internal/module"
	"cleanup-manager/internal/orchestrator"
	"cleanup-manager/internal/providers"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	cleanupconnect "github.com/vrooli/vrooli/packages/proto/gen/go/cleanup-manager/v1/cleanup/cleanup_v1connect"
)

func Module(logger *log.Logger) module.Module {
	registry, err := defaultRegistry()
	if err != nil {
		if logger == nil {
			logger = log.New(io.Discard, "", 0)
		}
		logger.Fatalf("cleanup registry: %v", err)
	}
	return ModuleWithService(orchestrator.NewService(registry, orchestrator.NewMemoryStore(), nil))
}

func ModuleWithService(service Service) module.Module {
	connectPath, connectHandler := cleanupconnect.NewCleanupServiceHandler(NewConnectHandler(requireService(service)))
	return module.Module{
		Name: "cleanup",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

func defaultRegistry() (*providers.Registry, error) {
	builtIns, err := providers.ConservativeBuiltIns(providers.BuiltInDeps{
		TrashRoots:           []string{},
		TmpRoots:             []string{},
		GoBuildCacheRoots:    []string{},
		PlaywrightCacheRoots: []string{},
	})
	if err != nil {
		return nil, err
	}
	return providers.NewRegistry(builtIns...)
}

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "cleanup_provider_list",
		Path:        cleanupconnect.CleanupServiceListProvidersProcedure,
		Method:      "POST",
		Summary:     "List cleanup providers",
		Description: "Returns the registered cleanup providers and their safety metadata.",
		Category:    "cleanup",
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"providers": "array<Provider>"}},
	},
	{
		ID:          "cleanup_policy_get",
		Path:        cleanupconnect.CleanupServiceGetPolicyProcedure,
		Method:      "POST",
		Summary:     "Get cleanup policy",
		Description: "Returns the active cleanup policy profile and per-provider settings.",
		Category:    "cleanup",
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"policy": "Policy"}},
	},
	{
		ID:          "cleanup_policy_set_profile",
		Path:        cleanupconnect.CleanupServiceSetPolicyProfileProcedure,
		Method:      "POST",
		Summary:     "Set cleanup policy profile",
		Description: "Switches the active policy to conservative, balanced, or aggressive defaults.",
		Category:    "cleanup",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"profile": "string (conservative|balanced|aggressive)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"policy": "Policy"}},
	},
	{
		ID:          "cleanup_plan_create",
		Path:        cleanupconnect.CleanupServiceCreatePlanProcedure,
		Method:      "POST",
		Summary:     "Create cleanup plan",
		Description: "Runs provider estimates and previews to create a stable cleanup plan id without mutating host state.",
		Category:    "cleanup",
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"plan": "Plan"}},
	},
	{
		ID:          "cleanup_plan_apply",
		Path:        cleanupconnect.CleanupServiceApplyPlanProcedure,
		Method:      "POST",
		Summary:     "Apply cleanup plan",
		Description: "Applies an approved plan using policy/provider version checks and an idempotency key.",
		Category:    "cleanup",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"plan_id":         "string (required)",
			"policy_version":  "string (required)",
			"approval_mode":   "string",
			"approval_token":  "string",
			"idempotency_key": "string (required)",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"reclaimed_bytes": "int64", "already_applied": "bool"}},
	},
	{
		ID:          "cleanup_audit_list",
		Path:        cleanupconnect.CleanupServiceListAuditProcedure,
		Method:      "POST",
		Summary:     "List cleanup audit events",
		Description: "Returns immutable cleanup policy, plan, apply, and replay audit events with redacted messages.",
		Category:    "cleanup",
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"events": "array<AuditEvent>"}},
	},
}
