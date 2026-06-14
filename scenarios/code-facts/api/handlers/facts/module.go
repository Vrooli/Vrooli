package facts

import (
	"database/sql"
	"log"
	"net/http"

	"code-facts/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/discovery"

	internalfacts "code-facts/internal/facts"

	factsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts/facts_v1connect"
)

func Module(db *sql.DB, logger *log.Logger) module.Module {
	resolver := discovery.NewResolver(discovery.ResolverConfig{})
	opts := []internalfacts.ServiceOption{
		internalfacts.WithBroker(internalfacts.NewBroker(
			internalfacts.NewGoGraphProvider(resolver, http.DefaultClient),
			internalfacts.NewTypeScriptGraphProvider(resolver, http.DefaultClient),
		)),
	}
	if db != nil {
		opts = append(opts, internalfacts.WithCacheRepository(internalfacts.NewSQLiteCacheRepository(db)))
	}
	svc := internalfacts.NewService(opts...)
	connectPath, connectHandler := factsconnect.NewCodeFactsServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "facts",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

var Endpoints = []module.EndpointDescriptor{
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
		CLIMapping: &module.CLIMapping{Command: "code-facts facts describe", Args: []string{"<target>", "--include", "<families>"}},
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
		Errors:     []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Target kind or target identifier is missing"}},
		CLIMapping: &module.CLIMapping{Command: "code-facts facts surfaces", Args: []string{"<target>"}},
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
		Errors:     []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Invalid limit or scenario enumeration failure"}},
		CLIMapping: &module.CLIMapping{Command: "code-facts facts fleet-imports", Args: []string{"--scenario", "<name>", "--json"}},
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
		CLIMapping:  &module.CLIMapping{Command: "code-facts facts proto-adoption", Args: []string{"<target>"}},
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
		CLIMapping:  &module.CLIMapping{Command: "code-facts facts endpoint-proof", Args: []string{"<target>", "--endpoint", "<id>"}},
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
			"cache_key": "string",
			"entries":   "int64",
		}},
		Errors:     []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Target kind or target identifier is missing"}},
		CLIMapping: &module.CLIMapping{Command: "code-facts cache status", Args: []string{"<target>"}},
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
		Errors:     []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Target kind or target identifier is missing"}},
		CLIMapping: &module.CLIMapping{Command: "code-facts cache inspect", Args: []string{"<target>", "--cache-key", "<key>"}},
	},
	{
		ID:          "facts_cache_clear",
		Path:        factsconnect.CodeFactsServiceClearCacheProcedure,
		Method:      "POST",
		Summary:     "Clear cache entries",
		Description: "Plans or clears cache entries for a target/options tuple.",
		Category:    "cache",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"target":  "CodeTarget (required)",
			"dry_run": "bool",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"cache_key":       "string",
			"matched_entries": "int64",
			"cleared_entries": "int64",
			"dry_run":         "bool",
		}},
		Errors:     []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Target kind or target identifier is missing"}},
		CLIMapping: &module.CLIMapping{Command: "code-facts cache clear", Args: []string{"<target>", "--dry-run"}},
	},
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
