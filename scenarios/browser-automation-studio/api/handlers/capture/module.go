// Package capture hosts the BAS CaptureService Connect-RPC handler.
//
// CaptureService.Capture is the first proto-first Connect handler in BAS.
// It lives side-by-side with the chi REST router established in main.go;
// no existing REST endpoint is moved or wrapped to support it.
package capture

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/connectx"
	captureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture/captureconnect"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"

	"github.com/vrooli/browser-automation-studio/internal/paths"
	"github.com/vrooli/browser-automation-studio/services/readiness"
	"github.com/vrooli/browser-automation-studio/services/workflow"
	"github.com/vrooli/browser-automation-studio/storage"
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

	// ExportToFolder writes every artifact produced by an execution
	// (screenshots/, console-logs.md, network-activity.md, timeline.json,
	// result.json, …) into outputDir. The capture handler delegates
	// artifact production to this seam so the executor remains the single
	// owner of artifact-write semantics — capture only assembles the
	// CaptureArtifact response list by walking outputDir afterwards.
	ExportToFolder(
		ctx context.Context,
		executionID uuid.UUID,
		outputDir string,
		storageClient storage.StorageInterface,
	) error
}

// URLResolver resolves the `scenario=<slug>` shorthand to a base URL.
// Implemented by *discovery.Resolver from packages/api-core. Optional —
// when nil, shorthand URLs return InvalidArgument.
type URLResolver interface {
	ResolveScenarioURLDefault(ctx context.Context, slug string) (string, error)
}

// uiURLResolver is implemented by api-core's discovery resolver. Keep it
// optional so focused handler tests and legacy callers that only expose a
// default API URL retain a safe fallback.
type uiURLResolver interface {
	ResolveScenarioURL(ctx context.Context, slug, portKey string) (string, error)
}

// ReadinessProfileResolver obtains the Experience Manager-owned compiled
// profile for a known local scenario. It is optional: unavailable profiles
// preserve generic capture behavior.
//
// Aliased to the shared services/readiness definitions because the workflow
// executor resolves the same contract; keeping one type keeps the two callers
// from drifting.
type ReadinessProfileResolver = readiness.Resolver

// ReadinessResolution preserves the contract provenance alongside the graph
// waits so Capture reports why a declared strategy was selected.
type ReadinessResolution = readiness.Resolution

// Deps wires the capture handler. Executor and Logger are required;
// Storage is required to produce real screenshot files; Resolver and Now
// are optional (Now defaults to time.Now). Producers, InlineDom and
// InlineAccessibility are optional and default to DefaultProducerRegistry /
// package inline defaults.
type Deps struct {
	Executor            Executor
	Storage             storage.StorageInterface
	Resolver            URLResolver
	ReadinessResolver   ReadinessProfileResolver
	Logger              *logrus.Logger
	Now                 func() time.Time
	Producers           *ProducerRegistry
	InlineDom           InlineDomConfig
	InlineAccessibility InlineAccessibilityConfig

	// CapturesRoot anchors relative (and omitted) out_dir values so capture
	// bundles land in scenario-owned storage rather than the API process's
	// working directory. Defaults to paths.ResolveCapturesRoot.
	CapturesRoot string
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
	if d.Producers == nil {
		d.Producers = DefaultProducerRegistry()
	}
	if strings.TrimSpace(d.CapturesRoot) == "" {
		d.CapturesRoot = paths.ResolveCapturesRoot(d.Logger)
	}
	d.InlineDom = d.InlineDom.withDefaults()
	d.InlineAccessibility = d.InlineAccessibility.withDefaults()
	path, handler := captureconnect.NewCaptureServiceHandler(&service{deps: d})
	return connectx.ServiceMount{Path: path, Handler: handler}
}
