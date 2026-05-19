// Package executions hosts the BAS ExecutionsService Connect-RPC handler.
//
// ExecutionsService owns execution-state queries (list/get/timeline),
// lifecycle controls (stop/resume), JSON listings of execution artifact
// metadata (screenshots, recorded videos, traces, HAR), and deferred
// seed-cleanup scheduling. Binary artifact downloads and replay-export
// streaming are intentionally left on chi REST with RESTException markers
// because they stream bytes or write to caller-supplied output dirs.
package executions

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/connectx"

	workflowservice "github.com/vrooli/browser-automation-studio/services/workflow"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api/apiconnect"
)

// Executor is the narrow seam onto workflow.ExecutionService for the
// transport-layer operations exposed by ExecutionsService.
type Executor interface {
	workflowservice.ExecutionService
}

// SeedScheduler abstracts deferred seed-cleanup scheduling so it can be
// substituted in tests.
type SeedScheduler interface {
	Schedule(executionID string, seedScenario string, cleanupToken string) error
}

// Deps wires the executions handler.
type Deps struct {
	// Executor delivers all execution-state read operations and lifecycle
	// controls. Must be non-nil.
	Executor Executor
	// SeedScheduler receives ScheduleExecutionSeedCleanup requests. When
	// nil the RPC returns connect.CodeFailedPrecondition.
	SeedScheduler SeedScheduler
	// RecordingsRoot is consulted by include_exportability=true list
	// requests to probe for a recorded video artifact. Empty means
	// "skip the recorded-video probe".
	RecordingsRoot string
	// Logger is required.
	Logger *logrus.Logger
}

// Module builds the ExecutionsService Connect handler.
func Module(d Deps) connectx.ServiceMount {
	if d.Logger == nil {
		panic("executions.Module requires Deps.Logger")
	}
	if d.Executor == nil {
		panic("executions.Module requires Deps.Executor")
	}
	path, handler := apiconnect.NewExecutionsServiceHandler(&service{deps: d})
	return connectx.ServiceMount{Path: path, Handler: handler}
}

// ensure service satisfies the connect handler interface at compile-time.
var _ apiconnect.ExecutionsServiceHandler = (*service)(nil)

// Compile-time guard: this package is consumed by main.go and tests.
var _ = context.Background
