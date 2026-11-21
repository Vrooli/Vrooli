package routing

import (
	"context"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing/routing_v1connect"

	"ai-gateway/internal/module"
	internalrouting "ai-gateway/internal/routing"
)

var ProtoFile = routingv1.File_ai_gateway_v1_routing_routing_proto

// seam: MediaRecoveryReporter controls diagnostics from startup receipt recovery.
type MediaRecoveryReporter func(error)

func reportMediaRecoveryError(err error) {
	log.Printf("media receipt startup recovery incomplete: %v", err)
}

var _ MediaRecoveryReporter = reportMediaRecoveryError

func Module(deps Deps) module.Module {
	h := NewConnectHandler(deps)
	if err := h.RecoverMedia(context.Background()); err != nil {
		report := deps.ReportMediaRecoveryError
		if report == nil {
			report = reportMediaRecoveryError
		}
		report(err)
	}
	connectPath, connectHandler := routingconnect.NewRoutingServiceHandler(h)
	return module.Module{
		Name: "routing",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
			mountPrivatePolicyRoutes(r, h)
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return internalrouting.Schema() }
