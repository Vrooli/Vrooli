// Package snippets exposes sender-owned reusable text over Connect-RPC.
package snippets

import (
	"context"
	"log"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	snippetsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/snippets/snippets_v1connect"

	"web-console/internal/module"
	snippetdomain "web-console/internal/snippets"
)

type Service interface {
	List(context.Context) ([]Snippet, error)
	Upsert(context.Context, UpsertRequest) (Snippet, error)
	Delete(context.Context, string) (bool, error)
	Touch(context.Context, string, time.Time) (Snippet, error)
}

type (
	Snippet       = snippetdomain.Snippet
	UpsertRequest = snippetdomain.UpsertRequest
)

// Schema re-exports the domain schema through the handler-owned registry seam.
func Schema() string { return snippetdomain.Schema() }

func Module(service Service, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	path, handler := snippetsconnect.NewSnippetsServiceHandler(NewConnectHandler(Deps{Service: service, Logger: logger}))
	return module.Module{
		Name: "snippets",
		Mount: func(router *mux.Router) {
			connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}
