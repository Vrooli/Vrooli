package facts

import (
	"database/sql"
	"log"
	"net/http"

	"code-facts/internal/indexcontrol"
	"github.com/vrooli/api-core/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/discovery"

	internalfacts "code-facts/internal/facts"

	factsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts/facts_v1connect"
	controlconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/control/control_v1connect"
)

func Module(db *sql.DB, logger *log.Logger, cacheMaxBytes int64, admission *Admission, indexControlToken string, dynamicTokens ...func(string) bool) module.Module {
	resolver := discovery.NewResolver(discovery.ResolverConfig{})
	var indexController IndexController
	var jobs indexcontrol.JobStore
	opts := []internalfacts.ServiceOption{
		internalfacts.WithBroker(internalfacts.NewBroker(
			internalfacts.NewGoGraphProvider(),
			internalfacts.NewTypeScriptGraphProvider(resolver, http.DefaultClient),
			internalfacts.NewJavaScriptGraphProvider(resolver, http.DefaultClient),
		)),
		internalfacts.WithFileDomainProvider(internalfacts.NewCartographerFileDomainProvider(resolver, http.DefaultClient)),
		internalfacts.WithAdmission(admission),
	}
	if db != nil {
		opts = append(opts, internalfacts.WithCacheRepository(internalfacts.NewSQLiteCacheRepository(db, cacheMaxBytes)))
		productionIndex, err := NewProductionIndex(db, admission)
		if err != nil {
			if logger != nil {
				logger.Printf("code-facts indexed runtime unavailable: %v", err)
			}
		} else {
			opts = append(opts, internalfacts.WithFileMetricsProvider(productionIndex))
			opts = append(opts, internalfacts.WithSearchBackend(productionIndex))
			jobs = productionIndex.jobs
			indexController = NewRuntimeIndexController(indexcontrol.NewSQLiteStatusReader(db, jobs), productionIndex)
			productionIndex.Bootstrap()
		}
	}
	svc := internalfacts.NewService(opts...)
	var tokenMatcher func(string) bool
	if len(dynamicTokens) > 0 {
		tokenMatcher = dynamicTokens[0]
	}
	if db != nil && indexController == nil {
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
			r.Handle("/api/v1/invariants/candidate-receipt-match", CandidateReceiptMatchHTTPHandler(resolver, http.DefaultClient)).Methods(http.MethodPost)
			r.Handle("/api/v1/flows/search", SearchFlowHTTPHandler(svc)).Methods(http.MethodPost)
			r.Handle("/api/v1/ambient-seams/search", SearchAmbientSeamsHTTPHandler(svc)).Methods(http.MethodPost)
			r.Handle("/api/v1/persistence-patterns/search", SearchPersistencePatternsHTTPHandler(svc)).Methods(http.MethodPost)
			r.Handle("/api/v1/endpoint-anatomy/search", SearchEndpointAnatomyHTTPHandler(svc)).Methods(http.MethodPost)
			r.Handle("/api/v1/proto-handler-conformance", ProtoHandlerConformanceHTTPHandler(svc)).Methods(http.MethodPost)
		},
		Endpoints: Endpoints,
	}
}

var Endpoints = []module.EndpointDescriptor{
	{
		ID: "facts_candidate_receipt_match", Path: "/api/v1/invariants/candidate-receipt-match", Method: "POST",
		Summary:     "Check one invariant candidate against one durable Test Genie receipt",
		Description: "Calls Test Genie's candidate-to-receipt owner check for one caller-selected receipt and exact Code Facts candidate identity. EXACT_MATCH confirms the source observation is bound to a successful receipt; UNKNOWN preserves missing, legacy, mismatched, changed, or unavailable evidence. This does not establish invariant enforcement or requirement/test-tag identity, and service errors remain unavailable rather than zero-result.",
		Category:    "facts",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"receiptId": "one durable Test Genie receipt ID", "candidate": "exact Code Facts candidate ID, scenario, relativePath, sourceLine, and sourceFileSha256"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"status": "EXACT_MATCH | UNKNOWN", "receiptId": "owner receipt ID", "candidate": "candidate source identity", "reason": "owner explanation"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Receipt or candidate reference is incomplete or rejected by Test Genie"}, {Status: 503, Code: "unavailable", Description: "Test Genie could not be resolved or returned no usable response"}},
	},
	{
		ID:          "facts_proto_handler_conformance",
		Path:        "/api/v1/proto-handler-conformance",
		Method:      "POST",
		Summary:     "Compare proto RPC method sets, generated Connect interfaces, and concrete handler signatures",
		Description: "For one scenario, lexically extracts RPC method names from the matching packages/proto/schemas package and compares them with the imported generated ServiceHandler interface, then compares concrete Go Connect handler method parameter/result types with that interface. Source locators include file, line, and digest. This does not validate full proto grammar or message field shapes, prove generated-code freshness, inspect endpoint registration/mounting, runtime behavior, or client compatibility.",
		Category:    "facts",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "one scenario slug"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"services": "array of .proto RPC method-set parity plus generated Connect service and concrete handler signature comparisons", "degradations": "array<string>"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "One scenario slug is required"}, {Status: 503, Code: "unavailable", Description: "Bounded generated or handler source is unavailable"}},
	},
	{
		ID:          "facts_flow_search",
		Path:        "/api/v1/flows/search",
		Method:      "POST",
		Summary:     "Search bounded feature source flows",
		Description: "Returns query-ranked candidate paths from the selected scenario's Go call graph, capped by package source size, 20 matching start functions, eight call sites per function, eight paths per start, five results, and three call-site edges. Distinct branches from one start are preserved with stable call-site IDs; branch_truncated and flow_branch_result_limit mark bounded omissions. Each step carries call-site ID, caller path/line/source-graph hash, plus resolved target path/hash or explicit unresolved-symbol status. Source edges do not prove complete flow or runtime reachability.",
		Category:    "facts",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"query": "string (required)", "scenario": "one scenario slug", "path": "relative source subdirectory inside the scenario", "limit": "integer (optional, max 5)",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"results": "array<FlowSearchHit {depth_limited: bool, branch_truncated: bool}>", "degradations": "array<FlowDegradation>"}},
		Errors:   []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Query, scenario, relative source path, or result limit is invalid"}, {Status: 503, Code: "unavailable", Description: "Source graph is unavailable or exceeds the bounded analysis scope"}},
	},
	{
		ID: "facts_ambient_seam_search", Path: "/api/v1/ambient-seams/search", Method: "POST",
		Summary:     "Search static ambient seam source evidence",
		Description: "Classifies bounded Go clock, environment, HTTP-client and logger source call sites as test-file or non-test-file candidates with source path/line/hash, static argument context, and any declared go:build/+build condition. Build conditions remain unevaluated for target GOOS/GOARCH/tags; this does not prove binary membership, runtime wiring, or test substitution.", Category: "facts",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"query": "string (required)", "scenario": "one scenario slug", "limit": "integer (optional, max 12)"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"results": "array<AmbientSeam>", "degradations": "array<AmbientSeamDegradation>", "scope": "string"}},
		Errors:   []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Query, scenario slug or limit is invalid"}, {Status: 503, Code: "unavailable", Description: "Scenario source is unavailable or exceeds bounded scope"}},
	},
	{
		ID: "facts_persistence_pattern_search", Path: "/api/v1/persistence-patterns/search", Method: "POST",
		Summary:     "Search joined static persistence evidence",
		Description: "Joins only exact unquoted schema identifiers whose schema.go embed/Schema binding, version-1 migration registration AST, and repository SQL string-literal references are proven. Unsupported or ambiguous lineage is surfaced as UNKNOWN; this does not report live database or migration state.", Category: "facts",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"query": "string (required)", "scenario": "one scenario slug; currently code-facts only", "limit": "integer (optional, max 5)"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"results": "array<PersistencePattern>", "unknowns": "array<PersistenceUnknown>", "scope": "string", "degradations": "array<string>"}},
		Errors:   []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Query, supported scenario, or result limit is invalid"}, {Status: 503, Code: "unavailable", Description: "Bounded persistence source files are unavailable"}},
	},
	{
		ID: "facts_endpoint_anatomy_search", Path: "/api/v1/endpoint-anatomy/search", Method: "POST",
		Summary:     "Search joined endpoint surface evidence",
		Description: "Joins literal HTTP JSON endpoint descriptors with mux routes and Connect Procedure descriptors with generated path constants plus the direct service-constructor → explicit ServiceMount → RegisterServices chain. Every Connect edge has path/line/hash provenance. Duplicate literal HTTP registrations for one method/path are returned as ROUTE_AMBIGUOUS with each registration and handler provenance; no source-order handler is selected and runtime dispatch precedence remains unknown. This is owner-local source evidence only; it does not prove outer module mounting, runtime reachability, request success, or client compatibility.", Category: "facts",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"query": "string (required)", "scenario": "one scenario slug; currently code-facts only", "limit": "integer (optional, max 10)"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"results": "array<EndpointAnatomy (HTTP JSON or Connect; optional transport/connect_service/connect_method/connect_chain; ambiguous HTTP includes route_candidates, route_candidate_total, route_candidates_omitted)", "scope": "string", "degradations": "array<string>"}},
		Errors:   []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Query, supported scenario, or result limit is invalid"}, {Status: 503, Code: "unavailable", Description: "Bounded endpoint source evidence is unavailable"}},
	},
	{
		ID: "facts_invariant_candidates", Path: factsconnect.CodeFactsServiceSearchInvariantCandidatesProcedure, Method: "POST",
		Summary:     "Search candidate invariant source evidence",
		Description: "Returns bounded lexical source locations by candidate mechanism. Candidate locations do not prove enforcement; type, database, test and runtime verification layers remain UNKNOWN without authoritative evidence.", Category: "facts",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"query": "string (required)", "scenario": "one scenario slug", "path": "relative source directory inside scenario", "limit": "integer (optional, max 10)"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"results": "array<InvariantCandidate>", "degradations": "array<InvariantCandidateDegradation>", "scope": "string"}},
		Errors:   []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Query, scenario, relative source path, or result limit is invalid"}, {Status: 503, Code: "unavailable", Description: "Source inventory is unavailable or outside bounded scope"}},
	},
	{
		ID:          "facts_search",
		Path:        factsconnect.CodeFactsServiceSearchProcedure,
		Method:      "POST",
		Summary:     "Search code evidence",
		Description: "Queries the active persistent catalog generation with exact and hybrid lexical retrieval while preserving source hash, generation, ranking explanation, and proof provenance. IMPORTS is scenario-scoped; Go imports and bounded literal JavaScript/TypeScript import, re-export, dynamic-import and require edges preserve source path/hash/generation/line. These are source dependency declarations, not runtime reachability or resolved package dependencies; unsupported forms are UNKNOWN with degradation.",
		Category:    "facts",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"query": "string (required)", "limit": "int32 (default 10)", "target": "CodeTarget (optional; defaults to project)",
			"families": "array<FactFamily> (ranked search supports SYMBOLS, REFERENCES, CALLS, IMPORTS, PROTO_ADOPTION, ENDPOINT_PROOFS, or ALL)",
			"scope":    "string (IMPORTS requires empty or exact scenario:<target.scenario>)", "languages": "array<string> (IMPORTS supports go, typescript, javascript)",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"results": "array<SearchHit>"}},
		Errors:   []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Query/target is invalid, a requested family is not indexed, or IMPORTS scope/language is unsupported"}, {Status: 503, Code: "unavailable", Description: "The active generation cannot satisfy the requested import-language scope; reconcile Code Facts"}},
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
		ID:          "facts_composition",
		Path:        factsconnect.CodeFactsServiceDescribeCompositionProcedure,
		Method:      "POST",
		Summary:     "Describe composition root bindings",
		Description: "Returns Go syntax evidence for observed production and test constructor-to-interface bindings with source hashes. It supports direct literals, one unique top-level same-function local initialized directly by T{} or &T{}, and one zero-argument same-package call to a unique function whose sole statement directly returns T{} or &T{}; supported static main-to-constructor calls include a derived source chain. The analyzer does not type-check packages; parameterized or conditional call returns, alias chains and dynamic dispatch remain UNKNOWN/PARTIAL.",
		Category:    "facts",
		Request:     targetRequestSchema(),
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"target": "TargetContext", "bindings": "array<CompositionBinding>",
		}},
		Errors: []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Target kind or target identifier is missing"}},
	},
	{
		ID:          "facts_composition_search",
		Path:        factsconnect.CodeFactsServiceSearchCompositionProcedure,
		Method:      "POST",
		Summary:     "Search composition evidence",
		Description: "Returns query-ranked Go composition evidence for exactly one scenario, preserving direct-literal, unique same-function local-initializer, and narrow direct same-package constructor-return provenance; parameterized/conditional call-return, alias-chain and dynamic forms remain UNKNOWN/PARTIAL.",
		Category:    "facts",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"query": "string (required)", "scenario": "string (required; one scenario slug)", "limit": "integer (optional, max 50)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"results": "array<SearchHit>", "retrieval_regime": "string"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Query and exactly one scenario slug are required"}},
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
