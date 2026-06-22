package advisor

import (
	"log"

	"storage-health/internal/advisor"
	"storage-health/internal/module"
	"storage-health/internal/validation"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	advisorv1 "github.com/vrooli/vrooli/packages/proto/gen/go/storage-health/v1/advisor"
	advisorconnect "github.com/vrooli/vrooli/packages/proto/gen/go/storage-health/v1/advisor/advisor_v1connect"
)

// ProtoFile is the FileDescriptor backing the Connect-mounted AdvisorService.
var ProtoFile = advisorv1.File_storage_health_v1_advisor_advisor_proto

// Module mounts the AdvisorService backed by the real reader (the storage
// validation engine projected onto migration facts) and the real enumerator.
func Module(logger *log.Logger, repoRoot string) module.Module {
	// Fleet-wide migration analysis classifies every discovered scenario, so it
	// uses the fast filesystem detector rather than the per-call code-facts parse.
	validator := validation.New(validation.Deps{RepoRoot: repoRoot, Detector: validation.FilesystemDetector{}, Logger: logger})
	svc := advisor.NewService(newReader(validator), newCLIEnumerator())
	handler := NewHandler(svc, logger)
	path, connectHandler := advisorconnect.NewAdvisorServiceHandler(handler)
	return module.Module{
		Name: "advisor",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — advisor is stateless (it reads the fleet live).
func Schema() string { return "" }

// Endpoints is the static endpoint metadata for codegen and the parity test.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "advisor_analyze_migrations",
		Path:        advisorconnect.AdvisorServiceAnalyzeMigrationsProcedure,
		Method:      "POST",
		Summary:     "Grade migration hygiene against deploy stage",
		Description: "Analyzes each requested scenario's (or every discovered scenario's) migration posture against its deploy stage — greenfield expects schema-as-desired-state; brownfield expects idempotent, ALTER-free, per-domain schema with a forward-migration path — and returns per-scenario hygiene plus stage-aware debt notes.",
		Category:    "advisor",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenarios": "array<string>"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"entries": "array<MigrationHygiene>", "scenario_count": "int32", "with_migrations_count": "int32", "debt_count": "int32", "errors": "array<AdvisorScanError>",
		}},
		Errors: []module.ErrorDesc{{Status: 500, Code: "internal", Description: "Migration analysis failure"}},
		Examples: []module.Example{
			{Name: "Analyze fleet migrations", Curl: "curl http://localhost:${API_PORT}/vrooli.storage_health.v1.advisor.AdvisorService/AnalyzeMigrations -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "advisor_advise_engines",
		Path:        advisorconnect.AdvisorServiceAdviseEnginesProcedure,
		Method:      "POST",
		Summary:     "Rank Postgres→SQLite migration candidates",
		Description: "Scores each requested scenario's (or every discovered scenario's) engine fitness and returns a ranked list of Postgres→SQLite migration candidates with rationale, fitness score, and any blockers. Vrooli is local-first; a single-node scenario carrying Postgres usually pays an external-service cost it does not need.",
		Category:    "advisor",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenarios": "array<string>"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"candidates": "array<EngineCandidate>", "scenario_count": "int32", "errors": "array<AdvisorScanError>",
		}},
		Errors: []module.ErrorDesc{{Status: 500, Code: "internal", Description: "Engine advisor failure"}},
		Examples: []module.Example{
			{Name: "Advise engine migrations", Curl: "curl http://localhost:${API_PORT}/vrooli.storage_health.v1.advisor.AdvisorService/AdviseEngines -H 'Content-Type: application/json' -d '{}'"},
		},
	},
}
