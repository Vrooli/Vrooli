// Package grouptemplates is the HTTP-handler home for the group-template
// domain. It exposes the generated Connect-RPC GroupTemplatesService (proto
// schema: packages/proto/schemas/web-console/v1/grouptemplates).
//
// RPCs (mounted at /vrooli.web_console.v1.grouptemplates.GroupTemplatesService/...):
//
//	ListTemplates  — every stored template
//	UpsertTemplate — create or update a template by id
//	DeleteTemplate — idempotent delete by id
//
// Domain types and the Store interface live in
// web-console/internal/grouptemplates; this package owns the transport
// surface only.
package grouptemplates

import (
	"context"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	grouptemplatesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/grouptemplates/grouptemplates_v1connect"

	gtdomain "web-console/internal/grouptemplates"
	"web-console/internal/module"
)

// Service is the seam the Connect handler depends on. The production
// implementation is the domain Store itself — there is no per-request policy
// to add, so an adapter would be a pass-through with no reason to exist.
type Service interface {
	List(ctx context.Context) ([]Template, error)
	Upsert(ctx context.Context, req UpsertRequest) (Template, error)
	Delete(ctx context.Context, id string) (bool, error)
}

// Transport types are aliases to the domain types so handlers and internal
// share one shape. No conversion code lives in this package.
type (
	Template      = gtdomain.Template
	TemplateRole  = gtdomain.TemplateRole
	UpsertRequest = gtdomain.UpsertRequest
)

// Module wires the group-template domain into the API server.
func Module(svc Service, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	connectPath, connectHandler := grouptemplatesconnect.NewGroupTemplatesServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "grouptemplates",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}
