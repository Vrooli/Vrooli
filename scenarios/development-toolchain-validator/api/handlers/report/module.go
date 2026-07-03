// Package report is the HTTP/Connect handler edge for the report
// domain. The service has no SQL of its own — it composes reads over
// the peer domains.
package report

import (
	"log"

	"github.com/vrooli/api-core/database"

	"development-toolchain-validator/internal/clock"
	manifest "development-toolchain-validator/internal/manifest"
	"development-toolchain-validator/internal/module"
	reportdom "development-toolchain-validator/internal/report"
	skillcatalog "development-toolchain-validator/internal/skill_catalog"
	staleness "development-toolchain-validator/internal/staleness"
	vr "development-toolchain-validator/internal/validation_record"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	reportconnect "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/report/report_v1connect"

	golden "development-toolchain-validator/internal/golden"
)

// Module returns the report domain's contribution to the API.
//
// Like staleness, the service composes reads from peer domains;
// repositories are constructed inline against the shared *database.RoutedDB so we
// don't invert through services.
func Module(db *database.RoutedDB, clk clock.Clock, source skillcatalog.SkillCatalogSource, logger *log.Logger) module.Module {
	skillRepo := skillcatalog.NewSQLiteRepository(db, clk)
	skillSvc := skillcatalog.NewService(skillRepo, source, clk)
	manifestRepo := manifest.NewSQLiteRepository(db, clk)
	manifestSvc := manifest.NewService(manifestRepo, clk)
	recordsRepo := vr.NewSQLiteRepository(db)
	recordsSvc := vr.NewService(recordsRepo, clk)
	goldenRepo := golden.NewSQLiteRepository(db, clk)
	stalenessSvc := staleness.NewService(
		staleness.ManifestSourceFromService{Svc: manifestSvc, Repo: manifestRepo},
		staleness.GoldenSourceFromRepo{Repo: goldenRepo},
		staleness.SkillSourceFromRepo{Repo: skillRepo},
	)

	svc := reportdom.NewService(skillSvc, manifestSvc, recordsSvc, stalenessSvc)

	connectPath, connectHandler := reportconnect.NewReportServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "report",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema is empty — report owns no SQL.
func Schema() string { return "" }

// Endpoints describes the report module's public surface.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "report_golden_summary",
		Path:        reportconnect.ReportServiceGetGoldenSummaryProcedure,
		Method:      "POST",
		Summary:     "Per-golden verdict roll-up",
		Description: "Latest verdict per skill and per tool for a given golden, plus a stale-tuple count.",
		Category:    "report",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"golden_slug": "string (required)"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"summary": "GoldenSummary"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing golden_slug"},
			{Status: 500, Code: "internal", Description: "Composition failure"},
		},
		Examples: []module.Example{
			{Name: "Golden summary", Curl: "curl http://localhost:${API_PORT}/vrooli.development_toolchain_validator.v1.report.ReportService/GetGoldenSummary -H 'Content-Type: application/json' -d '{\"golden_slug\":\"reference-react-vite\"}'"},
		},
	},
	{
		ID:          "report_tuple_history",
		Path:        reportconnect.ReportServiceGetTupleHistoryProcedure,
		Method:      "POST",
		Summary:     "Paginated history for one (tuple_kind, subject, golden) tuple",
		Description: "Cursor-paginated list of validation records for one tuple, ordered by ended_at descending.",
		Category:    "report",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"tuple_kind":  "TupleKind (required)",
				"subject_id":  "string (required)",
				"golden_slug": "string (required)",
				"page_size":   "int32",
				"page_token":  "string",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"history": "TupleHistory"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing subject_id/golden_slug"},
			{Status: 500, Code: "internal", Description: "Composition failure"},
		},
		Examples: []module.Example{
			{Name: "Tuple history", Curl: "curl http://localhost:${API_PORT}/vrooli.development_toolchain_validator.v1.report.ReportService/GetTupleHistory -H 'Content-Type: application/json' -d '{\"tuple_kind\":1,\"subject_id\":\"implementation-plan-authoring\",\"golden_slug\":\"reference-react-vite\"}'"},
		},
	},
	{
		ID:          "report_coverage",
		Path:        reportconnect.ReportServiceGetCoverageProcedure,
		Method:      "POST",
		Summary:     "Per-golden coverage grid",
		Description: "One row per skill in the catalog with the latest verdict, staleness flag, and manifest presence.",
		Category:    "report",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"golden_slug": "string (required)"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"coverage": "Coverage"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing golden_slug"},
			{Status: 500, Code: "internal", Description: "Composition failure"},
		},
		Examples: []module.Example{
			{Name: "Coverage", Curl: "curl http://localhost:${API_PORT}/vrooli.development_toolchain_validator.v1.report.ReportService/GetCoverage -H 'Content-Type: application/json' -d '{\"golden_slug\":\"reference-react-vite\"}'"},
		},
	},
	{
		ID:          "report_skill_fitness",
		Path:        reportconnect.ReportServiceGetSkillFitnessProcedure,
		Method:      "POST",
		Summary:     "Cross-golden trust/cost/convergence aggregate for one skill",
		Description: "Folds every validation record for a skill across all goldens into run counts by verdict, pass rate, token/cost/duration totals, convergence ratio, latest verdict, staleness, and a derived fitness verdict (UNKNOWN/GREEN/YELLOW/RED). Consumed by ecosystem-manager's selection controller.",
		Category:    "report",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"skill_id": "string (required)"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"fitness": "SkillFitness"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing skill_id"},
			{Status: 500, Code: "internal", Description: "Composition failure"},
		},
		Examples: []module.Example{
			{Name: "Skill fitness", Curl: "curl http://localhost:${API_PORT}/vrooli.development_toolchain_validator.v1.report.ReportService/GetSkillFitness -H 'Content-Type: application/json' -d '{\"skill_id\":\"implementation-plan-authoring\"}'"},
		},
	},
}
