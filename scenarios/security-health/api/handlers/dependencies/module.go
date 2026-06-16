package dependencies

import (
	"log"

	"security-health/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	depdomain "security-health/internal/dependencies"

	dependenciesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/security-health/v1/dependencies"
	dependenciesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/security-health/v1/dependencies/dependencies_v1connect"
)

// ProtoFile exposes the dependencies domain's proto FileDescriptor for the
// global parity test.
var ProtoFile = dependenciesv1.File_security_health_v1_dependencies_dependencies_proto

// Module mounts the DependencyService handler. The service (discovery + store +
// annotator) is constructed in main.go and injected so the reconcile loop and
// the reindex handler share one instance.
func Module(logger *log.Logger, svc Searcher) module.Module {
	connectPath, connectHandler := dependenciesconnect.NewDependencyServiceHandler(NewConnectHandler(Deps{
		Logger:  logger,
		Service: svc,
	}))
	return module.Module{
		Name: "dependencies",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema returns the dependencies corpus schema (the SQLite SBOM table).
func Schema() string { return depdomain.Schema() }

// Endpoints describes the dependencies module's public surface.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "dependencies_search",
		Path:        dependenciesconnect.DependencyServiceSearchProcedure,
		Method:      "POST",
		Summary:     "Search the fleet dependency & vulnerability index",
		Description: "Queries the fleet SBOM corpus with a free-text query plus structured filters (ecosystem, vulnerable-only, name glob). Answers 'which scenarios are exposed to CVE-X?' in one call. TEXT/structured mode is always available; MODE_AI ranks by semantic similarity (Ollama embeddings + Qdrant) and degrades to TEXT when those backends are down. The response's mode_used reports which served the request.",
		Category:    "dependencies",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"query":           "string",
				"limit":           "int32",
				"mode":            "Mode (UNSPECIFIED|AI|TEXT)",
				"ecosystem":       "Ecosystem (GO|NPM)",
				"vulnerable_only": "boolean",
				"name_glob":       "string",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"results": "array<SearchResult>", "mode_used": "Mode"},
		},
		Examples: []module.Example{
			{Name: "Vulnerable deps", Curl: "curl http://localhost:${API_PORT}/vrooli.security_health.v1.dependencies.DependencyService/Search -H 'Content-Type: application/json' -d '{\"vulnerable_only\":true}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "security-health deps search", Args: []string{"<query>"}},
	},
	{
		ID:          "dependencies_status",
		Path:        dependenciesconnect.DependencyServiceStatusProcedure,
		Method:      "POST",
		Summary:     "Report dependency-index backend availability and reconcile state",
		Description: "Returns indexed/vulnerable counts, AI backend availability (ollama/qdrant), and the last reconcile timestamp + outcome.",
		Category:    "dependencies",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"available": "boolean", "ollama": "boolean", "qdrant": "boolean",
				"indexed_count": "int32", "vulnerable_count": "int32",
				"last_reconcile_at": "string", "last_reconcile_outcome": "string",
			},
		},
		Examples: []module.Example{
			{Name: "Status", Curl: "curl http://localhost:${API_PORT}/vrooli.security_health.v1.dependencies.DependencyService/Status -H 'Content-Type: application/json' -d '{}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "security-health deps status"},
	},
	{
		ID:          "dependencies_vulnerabilities",
		Path:        dependenciesconnect.DependencyServiceListVulnerabilitiesProcedure,
		Method:      "POST",
		Summary:     "List structured dependency vulnerability evidence",
		Description: "Returns vulnerability evidence grouped by package/version with affected ranges, fixed-version guidance, source scanner, reachability, confidence, and affected scenarios/source files.",
		Category:    "dependencies",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"ecosystem":          "Ecosystem (GO|NPM)",
				"package_name":       "string",
				"scenario":           "string",
				"vulnerability_id":   "string",
				"minimum_confidence": "EvidenceConfidence (DEGRADED|ADVISORY|GATING)",
				"limit":              "int32",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"vulnerabilities": "array<VulnerabilityRecord>", "total": "int32"},
		},
		Examples: []module.Example{
			{Name: "Vulnerable dependencies", Curl: "curl http://localhost:${API_PORT}/vrooli.security_health.v1.dependencies.DependencyService/ListVulnerabilities -H 'Content-Type: application/json' -d '{}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "security-health deps vulnerabilities"},
	},
	{
		ID:          "dependencies_explain_vulnerability",
		Path:        dependenciesconnect.DependencyServiceExplainVulnerabilityProcedure,
		Method:      "POST",
		Summary:     "Explain one dependency vulnerability",
		Description: "Returns one structured vulnerability record for an advisory id, optionally scoped by ecosystem/package, including fixed-version and affected-scenario evidence.",
		Category:    "dependencies",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"vulnerability_id": "string",
				"ecosystem":        "Ecosystem (GO|NPM)",
				"package_name":     "string",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"vulnerability": "VulnerabilityRecord", "found": "boolean"},
		},
		Examples: []module.Example{
			{Name: "Explain advisory", Curl: "curl http://localhost:${API_PORT}/vrooli.security_health.v1.dependencies.DependencyService/ExplainVulnerability -H 'Content-Type: application/json' -d '{\"vulnerability_id\":\"GHSA-example\"}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "security-health deps explain", Args: []string{"<vulnerability>"}},
	},
}
