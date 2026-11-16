package stt

import (
	"context"

	"audio-tools/internal/modulekit"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	sttconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt/stt_v1connect"
)

// DefaultCredentialResolver returns a configured, server-side BYOK
// credential for a capability. It returns only the provider id and plaintext
// key to the in-process request path; neither value is included in health,
// capability, or error surfaces. Explicit request credentials take priority.
type DefaultCredentialResolver func(ctx context.Context, capability string) (provider, key string, ok bool)

func Module(d Deps) modulekit.Module {
	h := NewConnectHandler(d)
	runtimePath, runtimeHandler := sttconnect.NewSTTServiceHandler(h)
	adminPath, adminHandler := sttconnect.NewSTTAdminServiceHandler(h)
	return modulekit.Module{
		Name: "stt",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r,
				connectx.ServiceMount{Path: runtimePath, Handler: runtimeHandler},
				connectx.ServiceMount{Path: adminPath, Handler: adminHandler},
			)
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }
