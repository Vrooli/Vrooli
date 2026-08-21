package facts

import (
	"database/sql"
	"log"
	"net/http"

	"code-facts/internal/indexcontrol"
	"code-facts/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/discovery"

	internalfacts "code-facts/internal/facts"

	factsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts/facts_v1connect"
	controlconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/control/control_v1connect"
)

func Module(db *sql.DB, logger *log.Logger, cacheMaxBytes int64, admission *Admission, indexControlToken string, dynamicTokens ...func(string) bool) module.Module {
	resolver := discovery.NewResolver(discovery.ResolverConfig{})
	opts := []internalfacts.ServiceOption{
		internalfacts.WithBroker(internalfacts.NewBroker(
			internalfacts.NewGoGraphProvider(resolver, http.DefaultClient),
			internalfacts.NewTypeScriptGraphProvider(resolver, http.DefaultClient),
		)),
		internalfacts.WithFileDomainProvider(internalfacts.NewCartographerFileDomainProvider(resolver, http.DefaultClient)),
		internalfacts.WithAdmission(admission),
	}
	if db != nil {
		opts = append(opts, internalfacts.WithCacheRepository(internalfacts.NewSQLiteCacheRepository(db, cacheMaxBytes)))
	}
	svc := internalfacts.NewService(opts...)
	var indexController IndexController
	var jobs indexcontrol.JobStore
	var tokenMatcher func(string) bool
	if len(dynamicTokens) > 0 {
		tokenMatcher = dynamicTokens[0]
	}
	if db != nil {
		jobs = indexcontrol.NewSQLiteJobStore(db)
		indexController = NewRuntimeIndexController(indexcontrol.NewSQLiteStatusReader(db, jobs), nil)
	}
	authorizer := NewCompositeIndexAuthorizer(indexControlToken, tokenMatcher)
	connectPath, connectHandler := factsconnect.NewCodeFactsServiceHandler(NewConnectHandler(Deps{
		Service:         svc,
		Index:           indexController,
		IndexAuthorizer: authorizer,
		Logger:          logger,
	}))
	controlPath, controlHandler := controlconnect.NewSearchControlServiceHandler(&SearchControlHandler{
		Controller: indexController, Authorizer: authorizer, Jobs: jobs,
	})
	return module.Module{
		Name: "facts",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
			connectx.RegisterServices(r, connectx.ServiceMount{Path: controlPath, Handler: controlHandler})
		},
		Endpoints: Endpoints,
	}
}

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "facts_search",
		Path:        factsconnect.CodeFactsServiceSearchProcedure,
		Method:      "POST",
		Summary:     "Search code evidence",
		Description: "Runs a bounded lexical search over symbols, file domains, and contract facts while preserving analyzer and source-range provenance.",
		Category:    "facts",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"query": "string (required)", "limit": "int32 (default 10)", "target": "CodeTarget (optional; defaults to project)",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"results": "array<SearchHit>"}},
		Errors:   []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Query is empty or target is invalid"}},
	},
	{
		ID:          "facts_describe",
		Path:        factsconnect.CodeFactsServiceDescribeCodeFactsProcedure,
		Method:      "POST",
		Summary:     "Describe code facts",
		Description: "Returns a selective Code Facts report for a path, scenario, module, project, or repo target.",
		Category:    "facts",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"target":  "CodeTarget (required)",
				"include": "array<FactFamily>",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"target":      "TargetContext",
				"surfaces":    "array<Surface>",
				"parse_units": "array<ParseUnit>",
				"facts":       "array<GenericFact>",
				"warnings":    "array<Warning>",
				"cache":       "CacheMetadata",
			},
		},
		Errors: []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Target kind or target identifier is missing"}},
		Examples: []module.Example{{
			Name: "Describe this scenario",
			Curl: "curl http://localhost:${API_PORT}/vrooli.code_facts.v1.facts.CodeFactsService/DescribeCodeFacts -H 'Content-Type: application/json' -d '{\"target\":{\"kind\":\"TARGET_KIND_SCENARIO\",\"scenario\":\"code-facts\"},\"include\":[\"FACT_FAMILY_SURFACES\"]}'",
		}},
	},
	{
		ID:          "facts_surfaces",
		Path:        factsconnect.CodeFactsServiceListSurfacesProcedure,
		Method:      "POST",
		Summary:     "List target surfaces",
		Description: "Returns the scenario or generic target surface inventory portion of a Code Facts report.",
		Category:    "facts",
		Request:     targetRequestSchema(),
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"target":   "TargetContext",
			"surfaces": "array<Surface>",
			"warnings": "array<Warning>",
			"cache":    "CacheMetadata",
		}},
		Errors: []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Target kind or target identifier is missing"}},
	},
	{
		ID:          "facts_fleet_imports",
		Path:        factsconnect.CodeFactsServiceDescribeFleetImportsProcedure,
		Method:      "POST",
		Summary:     "Describe fleet import facts",
		Description: "Returns FACT_FAMILY_IMPORTS reports for many or all scenarios without resolving imports to scenario dependencies.",
		Category:    "facts",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"scenarios":       "array<string> (optional, empty means all scenarios)",
			"limit":           "integer (optional, 0 means no limit, max 500)",
			"use_cache":       "bool",
			"repo_root":       "string",
			"language_filter": "array<string>",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"results": "array<CodeFactsResult>",
		}},
		Errors: []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Invalid limit or scenario enumeration failure"}},
	},
	{
		ID:          "facts_proto_adoption",
		Path:        factsconnect.CodeFactsServiceCheckProtoAdoptionProcedure,
		Method:      "POST",
		Summary:     "Check proto adoption evidence",
		Description: "Returns generated proto adoption proof evidence for selected API, CLI, and UI surfaces.",
		Category:    "facts",
		Request:     targetRequestSchema(),
		Response:    proofResponseSchema(),
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Target kind or target identifier is missing"}},
	},
	{
		ID:          "facts_endpoint_proof",
		Path:        factsconnect.CodeFactsServiceCheckEndpointProofProcedure,
		Method:      "POST",
		Summary:     "Check endpoint implementation evidence",
		Description: "Returns static REST exception endpoint proof evidence for selected endpoint ids.",
		Category:    "facts",
		Request:     targetRequestSchema(),
		Response:    proofResponseSchema(),
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Target kind or target identifier is missing"}},
	},
	{
		ID:          "facts_cache_status",
		Path:        factsconnect.CodeFactsServiceGetCacheStatusProcedure,
		Method:      "POST",
		Summary:     "Get cache status",
		Description: "Returns inspectable cache metadata for a target/options tuple.",
		Category:    "cache",
		Request:     targetRequestSchema(),
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"cache_key":           "string",
			"entries":             "int64",
			"total_rows":          "int64",
			"total_payload_bytes": "int64",
			"budget_bytes":        "int64",
			"utilization":         "double",
			"scopes":              "array<CacheScopeStatus>",
			"last_sweep_at_unix":  "int64",
		}},
		Errors: []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Target kind or target identifier is missing"}},
	},
	{
		ID:          "facts_cache_inspect",
		Path:        factsconnect.CodeFactsServiceInspectCacheProcedure,
		Method:      "POST",
		Summary:     "Inspect cache entries",
		Description: "Returns matching cache entries with key, freshness hash, analyzer, and scope metadata.",
		Category:    "cache",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"target":    "CodeTarget (required)",
			"cache_key": "string",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"cache_key":        "string",
			"entries":          "int64",
			"entries_metadata": "array<CacheMetadata>",
		}},
		Errors: []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Target kind or target identifier is missing"}},
	},
	{
		ID:          "facts_cache_clear",
		Path:        factsconnect.CodeFactsServiceClearCacheProcedure,
		Method:      "POST",
		Summary:     "Clear cache entries",
		Description: "Plans or clears cache entries for a target/options tuple.",
		Category:    "cache",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"target":  "CodeTarget (required unless all is true)",
			"dry_run": "bool",
			"all":     "bool",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"cache_key":       "string",
			"matched_entries": "int64",
			"cleared_entries": "int64",
			"dry_run":         "bool",
		}},
		Errors: []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Target kind or target identifier is missing"}},
	},
	indexEndpoint("index_status", factsconnect.CodeFactsServiceGetIndexStatusProcedure, "Get index status", "Returns active generation, durable jobs, drift, counts, storage, and degraded stages."),
	indexEndpoint("index_reconcile", factsconnect.CodeFactsServiceReconcileIndexProcedure, "Reconcile index", "Applies bounded changed and deleted source work to a generation."),
	indexEndpoint("index_reindex", factsconnect.CodeFactsServiceReindexProcedure, "Reindex into a shadow generation", "Starts a confirmed full shadow-generation rebuild."),
	indexEndpoint("index_cancel", factsconnect.CodeFactsServiceCancelIndexJobProcedure, "Cancel index job", "Durably requests cancellation of an active index job."),
	indexEndpoint("index_promote", factsconnect.CodeFactsServicePromoteIndexGenerationProcedure, "Promote index generation", "Validates and atomically promotes a confirmed complete generation."),
	indexEndpoint("index_rollback", factsconnect.CodeFactsServiceRollbackIndexGenerationProcedure, "Rollback index generation", "Restores a confirmed retained complete generation."),
	indexEndpoint("index_cleanup", factsconnect.CodeFactsServiceCleanupIndexProcedure, "Clean retired index generations", "Plans or deletes derived generations while retaining rollback state."),
	sharedControlEndpoint("search_control_reindex", controlconnect.SearchControlServiceReindexProcedure, "Start provider-neutral reindex", "Adapts the shared token-gated Search Hub reindex request onto Code Facts shadow generation controls."),
	sharedControlEndpoint("search_control_status", controlconnect.SearchControlServiceReindexStatusProcedure, "Inspect provider-neutral reindex", "Returns durable Code Facts job progress through the shared Search Hub control contract."),
	sharedControlEndpoint("search_control_cancel", controlconnect.SearchControlServiceReindexCancelProcedure, "Cancel provider-neutral reindex", "Cooperatively cancels a durable Code Facts index job through the shared Search Hub control contract."),
}

func indexEndpoint(id, path, summary, description string) module.EndpointDescriptor {
	return module.EndpointDescriptor{
		ID: "facts_" + id, Path: path, Method: "POST", Summary: summary, Description: description, Category: "index",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"generation": "string", "confirmed": "bool", "dry_run": "bool", "job_id": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"job": "IndexJob", "status": "IndexStatus", "message": "string"}},
		Errors:   []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Required identifier is missing"}, {Status: 412, Code: "failed_precondition", Description: "Confirmation is required"}, {Status: 503, Code: "unavailable", Description: "Index controller is unavailable"}},
	}
}

func sharedControlEndpoint(id, path, summary, description string) module.EndpointDescriptor {
	return module.EndpointDescriptor{
		ID: id, Path: path, Method: "POST", Summary: summary, Description: description, Category: "index",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"job_id": "string", "scope": "string", "dry_run": "bool", "control_token": "string (required)"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"job_id": "string", "state": "string", "processed": "int32", "total": "int32", "cancelled": "bool"}},
		Errors:   []module.ErrorDesc{{Status: 403, Code: "permission_denied", Description: "Control token is missing or invalid"}, {Status: 503, Code: "unavailable", Description: "Index controller is unavailable"}},
	}
}

func targetRequestSchema() *module.Schema {
	return &module.Schema{Type: "object", Properties: map[string]string{"target": "CodeTarget (required)"}}
}

func proofResponseSchema() *module.Schema {
	return &module.Schema{Type: "object", Properties: map[string]string{
		"target":   "TargetContext",
		"family":   "FactFamily",
		"facts":    "array<GenericFact>",
		"evidence": "array<Evidence>",
		"warnings": "array<Warning>",
		"cache":    "CacheMetadata",
	}}
}
