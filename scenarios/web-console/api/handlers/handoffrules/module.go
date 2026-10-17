// Package handoffrules is the HTTP-handler home for the capture-rule domain.
// It exposes the generated Connect-RPC HandoffRulesService (proto schema:
// packages/proto/schemas/web-console/v1/handoffrules).
//
// RPCs (mounted at /vrooli.web_console.v1.handoffrules.HandoffRulesService/...):
//
//	ListRules  — every stored rule
//	UpsertRule — create or update a rule by id
//	DeleteRule — idempotent delete by id
//
// A rule decides when a handoff is SUGGESTED. Nothing in this package, or
// anywhere downstream of it, can send a handoff.
package handoffrules

import (
	"context"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	handoffrulesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/handoffrules/handoffrules_v1connect"

	hrdomain "web-console/internal/handoffrules"
	"web-console/internal/module"
)

// Service is the seam the Connect handler depends on. The production
// implementation is the domain Store itself: there is no per-request policy
// to add, so an adapter would be a pass-through with no reason to exist.
type Service interface {
	List(ctx context.Context) ([]Rule, error)
	Upsert(ctx context.Context, req UpsertRequest) (Rule, error)
	Delete(ctx context.Context, id string) (bool, error)
}

// Transport types are aliases to the domain types so handlers and internal
// share one shape. No conversion code lives in this package.
type (
	Rule          = hrdomain.Rule
	UpsertRequest = hrdomain.UpsertRequest
)

// Module wires the capture-rule domain into the API server.
func Module(svc Service, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	connectPath, connectHandler := handoffrulesconnect.NewHandoffRulesServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "handoffrules",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}
