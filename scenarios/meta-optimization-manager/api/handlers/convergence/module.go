// Package convergence is the API handler for the ConvergenceService — the
// template & generated-golden fitness domain. It is the proto translation edge
// over internal/convergence; all business logic lives in internal/convergence
// behind seams.
package convergence

import (
	"log"

	"meta-optimization-manager/internal/clock"
	internalconv "meta-optimization-manager/internal/convergence"
	"meta-optimization-manager/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	convergenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/convergence/convergence_v1connect"
)

// Module returns the convergence domain's contribution to the API: the generated
// ConvergenceService Connect-RPC handler, backed by the live filesystem fitness
// scanner + reference scanner and the SQLite fitness-audit index.
func Module(db *database.RoutedDB, clk clock.Clock, logger *log.Logger) module.Module {
	svc := internalconv.NewService(internalconv.Deps{
		Repo:  internalconv.NewSQLiteRepository(db),
		Clock: clk,
	})
	connectPath, connectHandler := convergenceconnect.NewConvergenceServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "convergence",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports internalconv.Schema so the modules registry collects
// endpoints and schema from one symbol per handler package.
func Schema() string { return internalconv.Schema() }

// Endpoints is the machine-readable description of the convergence module's
// public surface. The Connect-RPC method paths reference the generated
// *Procedure constants, so renaming an RPC in convergence.proto breaks this at
// compile time; the global parity test asserts every RPC has exactly one entry.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "convergence_status",
		Path:        convergenceconnect.ConvergenceServiceGetConvergenceStatusProcedure,
		Method:      "POST",
		Summary:     "Convergence status",
		Description: "Returns per-template four-lens fitness + gold-star generated-golden health across all templates (OT-P1-002). Computed live; persists a dated audit record for the trend.",
		Category:    "convergence",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"templates": "array<TemplateFitness>", "references": "array<ReferenceHealth>"},
		},
	},
	{
		ID:          "convergence_template_fitness",
		Path:        convergenceconnect.ConvergenceServiceGetTemplateFitnessProcedure,
		Method:      "POST",
		Summary:     "Template fitness",
		Description: "Returns the four-lens fitness counts + advisory tier for one template (or all).",
		Category:    "convergence",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"templates": "array<TemplateFitness>"},
		},
	},
	{
		ID:          "convergence_references",
		Path:        convergenceconnect.ConvergenceServiceListReferencesProcedure,
		Method:      "POST",
		Summary:     "List reference health",
		Description: "Returns gold-star generated-golden health + eligibility (stale-from-template, clean-on-all-tools, ≥60d stability, breadth), optionally filtered by eligibility.",
		Category:    "convergence",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"references": "array<ReferenceHealth>"},
		},
	},
	{
		ID:          "convergence_trend",
		Path:        convergenceconnect.ConvergenceServiceGetConvergenceTrendProcedure,
		Method:      "POST",
		Summary:     "Convergence trend",
		Description: "Returns per-replica-cost / coordinated-edit points over the dated fitness-audit records (the compounding proof), optionally for one template.",
		Category:    "convergence",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"points": "array<FitnessTrendPoint>"},
		},
	},
}
