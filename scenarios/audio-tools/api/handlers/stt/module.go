package stt

import (
	"context"

	"audio-tools/internal/modulekit"

	"connectrpc.com/connect"
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
	return moduleWithReadLimit(d, DefaultMaxRequestBytes)
}

// DefaultMaxRequestBytes bounds one STT message before Connect decodes it.
// The unary Transcribe payload is capped at sttpipeline.MaxAudioSize (10 MiB);
// 16 MiB leaves room for base64 expansion and the request envelope. Streaming
// messages are PCM batches of a few kilobytes, so the same ceiling is ample
// while preventing a single oversized frame from being buffered in memory.
const DefaultMaxRequestBytes = 16 << 20

func moduleWithReadLimit(d Deps, maxRequestBytes int) modulekit.Module {
	h := NewConnectHandler(d)
	opts := []connect.HandlerOption{connect.WithReadMaxBytes(maxRequestBytes)}
	runtimePath, runtimeHandler := sttconnect.NewSTTServiceHandler(h, opts...)
	adminPath, adminHandler := sttconnect.NewSTTAdminServiceHandler(h, opts...)
	return modulekit.Module{
		Name: "stt",
		Mount: func(r *mux.Router) {
			r.HandleFunc(BrowserTranscribeStreamWebSocketPath, h.serveTranscribeStreamWebSocket).Methods("GET")
			connectx.RegisterServices(r,
				connectx.ServiceMount{Path: runtimePath, Handler: runtimeHandler},
				connectx.ServiceMount{Path: adminPath, Handler: adminHandler},
			)
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }
