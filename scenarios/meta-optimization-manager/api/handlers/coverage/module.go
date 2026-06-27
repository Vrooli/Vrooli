// Package coverage is the API handler for the CoverageService — the readiness
// scoreboard. It is the proto translation edge over internal/coverage; all
// business logic lives in internal/coverage behind seams.
package coverage

import (
	"log"

	"meta-optimization-manager/internal/clock"
	internalcoverage "meta-optimization-manager/internal/coverage"
	"meta-optimization-manager/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	coverageconnect "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/coverage/coverage_v1connect"
)

// Module returns the coverage domain's contribution to the API: the generated
// CoverageService Connect-RPC handler, backed by the production seams (live
// space-reader + numerator-joiner) and the short-TTL SQLite snapshot cache.
func Module(db *database.RoutedDB, clk clock.Clock, logger *log.Logger) module.Module {
	svc := internalcoverage.NewService(internalcoverage.Deps{
		Snapshots: internalcoverage.NewSQLiteSnapshotRepository(db, clk),
		Clock:     clk,
	})
	connectPath, connectHandler := coverageconnect.NewCoverageServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "coverage",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports internalcoverage.Schema so the modules registry collects
// endpoints and schema from one symbol per handler package.
func Schema() string { return internalcoverage.Schema() }

// Endpoints is the machine-readable description of the coverage module's public
// surface. The Connect-RPC method paths reference the generated *Procedure
// constants, so renaming an RPC in coverage.proto breaks this at compile time;
// the global parity test (registry_test.go) asserts every RPC has exactly one
// entry here.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "coverage_status",
		Path:        coverageconnect.CoverageServiceGetStatusProcedure,
		Method:      "POST",
		Summary:     "Readiness scoreboard",
		Description: "Returns per-projection coverage + denominator-confidence + the latest empirical trend (OT-P0-001). Degrades gracefully per-projection when an owner is unreachable.",
		Category:    "coverage",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"projections": "array<ProjectionCoverage>", "latest_trial_trend": "EmpiricalTrendPoint"},
		},
	},
	{
		ID:          "coverage_list_cells",
		Path:        coverageconnect.CoverageServiceListCellsProcedure,
		Method:      "POST",
		Summary:     "List denominator cells",
		Description: "Returns the denominator grid rows with live status applied, optionally filtered by projection and status.",
		Category:    "coverage",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"cells": "array<Cell>"},
		},
	},
	{
		ID:          "coverage_explain_cell",
		Path:        coverageconnect.CoverageServiceExplainCellProcedure,
		Method:      "POST",
		Summary:     "Explain a cell",
		Description: "Returns one cell by namespaced id (<projection>/<id>) with its provenance citations.",
		Category:    "coverage",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"cell": "Cell"},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No cell with that id"},
		},
	},
	{
		ID:          "coverage_validate_docs",
		Path:        coverageconnect.CoverageServiceValidateBaseDocsProcedure,
		Method:      "POST",
		Summary:     "Validate base docs (self-honesty gate)",
		Description: "Checks the space docs themselves (OT-P0-004): guide rows map to a skill, authored-NOW cells have a live provider. ok=false when any ERROR-severity issue exists.",
		Category:    "coverage",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"issues": "array<BaseDocIssue>", "ok": "bool"},
		},
	},
}
