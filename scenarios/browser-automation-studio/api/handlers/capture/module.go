// Package capture hosts the BAS CaptureService Connect-RPC handler.
//
// CaptureService.Capture is the first proto-first Connect handler in BAS.
// It lives side-by-side with the chi REST router established in main.go;
// no existing REST endpoint is moved or wrapped to support it.
package capture

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/connectx"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	captureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture/captureconnect"

	"github.com/vrooli/browser-automation-studio/services/workflow"
)

// Executor is the narrow seam the capture handler depends on. It is
// satisfied by *workflow.WorkflowService today and mocked in tests.
//
// Keeping this interface to a single method (instead of embedding the
// full workflow.ExecutionService) means the capture handler can be
// tested without standing up the entire execution pipeline.
type Executor interface {
	ExecuteAdhocWorkflowAPIWithOptions(
		ctx context.Context,
		req *basexecution.ExecuteAdhocRequest,
		opts *workflow.ExecuteOptions,
	) (*basexecution.ExecuteAdhocResponse, error)
}

// URLResolver resolves the `scenario=<slug>` shorthand to a base URL.
// Implemented by *discovery.Resolver from packages/api-core. Optional —
// when nil, shorthand URLs return InvalidArgument.
type URLResolver interface {
	ResolveScenarioURLDefault(ctx context.Context, slug string) (string, error)
}

// Deps wires the capture handler. Executor and Logger are required;
// Resolver and Now are optional (Now defaults to time.Now).
type Deps struct {
	Executor Executor
	Resolver URLResolver
	Logger   *logrus.Logger
	Now      func() time.Time
}

// Module builds the CaptureService Connect handler and returns it
// wrapped in a connectx.ServiceMount ready for connectx.RegisterChi.
//
// Required deps: Executor, Logger. Missing required deps panic so a
// forgotten wire-up surfaces at boot, not at first request.
func Module(d Deps) connectx.ServiceMount {
	if d.Executor == nil {
		panic("capture.Module requires Deps.Executor")
	}
	if d.Logger == nil {
		panic("capture.Module requires Deps.Logger")
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	path, handler := captureconnect.NewCaptureServiceHandler(&service{deps: d})
	return connectx.ServiceMount{Path: path, Handler: handler}
}
