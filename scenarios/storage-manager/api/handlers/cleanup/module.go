package cleanup

import (
	"io"
	"log"

	"storage-manager/hostfs"
	"storage-manager/hostpaths"
	"storage-manager/internal/clock"
	"storage-manager/internal/module"
	"storage-manager/internal/orchestrator"
	"storage-manager/internal/providers"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	cleanupconnect "github.com/vrooli/vrooli/packages/proto/gen/go/storage-manager/v1/cleanup/cleanup_v1connect"
)

// Module builds the cleanup domain.
//
// The database handle backs the durable half of the store: the active policy
// and the audit trail. Passing nil falls back to fully in-memory state, which
// is what the endpoint-codegen binary and unit tests want — neither has a live
// database, and neither needs the operator's policy to survive anything.
func Module(logger *log.Logger, db *database.RoutedDB) module.Module {
	registry, err := defaultRegistry()
	if err != nil {
		if logger == nil {
			logger = log.New(io.Discard, "", 0)
		}
		logger.Fatalf("cleanup registry: %v", err)
	}

	var store orchestrator.Store = orchestrator.NewMemoryStore()
	if db != nil {
		store = orchestrator.NewSQLiteStore(db)
	}
	return ModuleWithService(orchestrator.NewService(registry, store, nil))
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

// defaultRegistry builds the production provider registry.
//
// Every file provider needs two things to do anything at all: a filesystem seam
// to walk, and roots to walk within. Until this wiring existed it had neither —
// BuiltInDeps.FileSystem was left nil and all four root lists were empty
// literals — so each provider reported "filesystem seam unavailable" and
// estimated zero bytes on a host with 70 GB of reclaimable temp files. The
// planning and policy layers were correct throughout; nothing was ever
// connected to the disk.
func defaultRegistry() (*providers.Registry, error) {
	roots := hostpaths.Resolve()
	files := hostfs.New(hostfs.Options{})

	builtIns, err := providers.ConservativeBuiltIns(providers.BuiltInDeps{
		FileSystem: files,
		Clock:      clock.System{},

		TrashRoots:           roots.Trash,
		TmpRoots:             roots.Tmp,
		GoBuildCacheRoots:    roots.GoBuildCache,
		PlaywrightCacheRoots: roots.PlaywrightCache,
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
		ID:          "cleanup_pressure_report",
		Path:        cleanupconnect.CleanupServiceReportPressureProcedure,
		Method:      "POST",
		Summary:     "Report disk pressure",
		Description: "Inbound disk-pressure signal from a safeguard. Warning records the observation; high runs estimate and preview without deleting; critical applies safe-tier providers with no operator present. Duplicate concurrent reports of the same partition and band collapse into one execution.",
		Category:    "cleanup",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"source_scenario": "string",
			"partition":       "string (required)",
			"used_percent":    "double",
			"band":            "enum (PRESSURE_BAND_WARNING|PRESSURE_BAND_HIGH|PRESSURE_BAND_CRITICAL, required)",
			"available_bytes": "int64",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"action":                   "enum (observed|previewed|applied|deduplicated|suppressed)",
			"plan_id":                  "string",
			"estimated_bytes":          "int64",
			"reclaimed_bytes":          "int64",
			"providers_applied":        "array<string>",
			"providers_withheld":       "array<string>",
			"autonomous_apply_enabled": "bool",
		}},
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
