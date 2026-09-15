// Package observability hosts the BAS ObservabilityService Connect-RPC
// handler.
//
// ObservabilityService is BAS's facade onto playwright-driver's
// /observability surface (status snapshots, diagnostics, session
// inventory, metrics, runtime config, pipeline self-test) plus the
// in-process debug-mode toggle consumed by the diagnostics UI.
//
// This package is a thin Connect adapter onto the transport-agnostic
// Fetch* methods exposed by the parent handlers package
// (FetchObservability, FetchObservabilityDiagnostics, ...) and the
// GetDebugModeSnapshot / SetDebugModeState helpers for the debug-mode
// toggle. Wire-format payloads from playwright-driver are owned by the
// downstream process; they are round-tripped via google.protobuf.Struct
// rather than re-modeled here.
package observability

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/connectx"

	"github.com/vrooli/browser-automation-studio/handlers"
	observabilityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/observability/observabilityconnect"
)

// Proxy is the narrow seam over the parent handlers.Handler's
// transport-agnostic Fetch*/Update*/Reset* methods. Tests can substitute
// a stub without standing up the full Handler graph.
type Proxy interface {
	FetchObservability(ctx context.Context, depth string, noCache bool) (map[string]any, error)
	FetchObservabilityRefresh(ctx context.Context) (map[string]any, error)
	FetchObservabilityDiagnostics(ctx context.Context, options map[string]any) (map[string]any, error)
	FetchObservabilitySessions(ctx context.Context) (map[string]any, error)
	FetchObservabilityCleanup(ctx context.Context) (map[string]any, error)
	FetchObservabilityMetrics(ctx context.Context) (map[string]any, error)
	FetchObservabilityPipelineTest(ctx context.Context, options map[string]any) (map[string]any, error)
	FetchObservabilityConfigRuntime(ctx context.Context) (map[string]any, error)
	UpdateObservabilityConfig(ctx context.Context, envVar, value string) (map[string]any, error)
	ResetObservabilityConfig(ctx context.Context, envVar string) (map[string]any, error)
}

// Verify the parent Handler still satisfies the seam.
var _ Proxy = (*handlers.Handler)(nil)

// Deps wires the ObservabilityService handler.
//
// Proxy and Logger are required; the debug-mode toggle lives in
// package-level state shared with the parent handlers package and does
// not need explicit wiring here.
type Deps struct {
	Proxy  Proxy
	Logger *logrus.Logger
}

// Module builds the ObservabilityService Connect handler.
func Module(d Deps) connectx.ServiceMount {
	if d.Logger == nil {
		panic("observability.Module requires Deps.Logger")
	}
	if d.Proxy == nil {
		panic("observability.Module requires Deps.Proxy")
	}
	path, handler := observabilityconnect.NewObservabilityServiceHandler(&service{deps: d})
	return connectx.ServiceMount{Path: path, Handler: handler}
}

var _ observabilityconnect.ObservabilityServiceHandler = (*service)(nil)
