package capture

import (
	"context"
	"errors"

	"github.com/google/uuid"
	basebase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"

	"github.com/vrooli/browser-automation-studio/services/workflow"
)

// fakeExecutor records the last request it received and returns a
// programmable canned response. Calls counts every invocation so dry-run
// assertions can verify the executor was never reached.
type fakeExecutor struct {
	LastReq  *basexecution.ExecuteAdhocRequest
	LastOpts *workflow.ExecuteOptions
	Calls    int
	Resp     *basexecution.ExecuteAdhocResponse
	Err      error
}

func (f *fakeExecutor) ExecuteAdhocWorkflowAPIWithOptions(
	_ context.Context,
	req *basexecution.ExecuteAdhocRequest,
	opts *workflow.ExecuteOptions,
) (*basexecution.ExecuteAdhocResponse, error) {
	f.Calls++
	f.LastReq = req
	f.LastOpts = opts
	if f.Err != nil {
		return nil, f.Err
	}
	if f.Resp != nil {
		return f.Resp, nil
	}
	return &basexecution.ExecuteAdhocResponse{
		ExecutionId: uuid.NewString(),
		Status:      basebase.ExecutionStatus_EXECUTION_STATUS_COMPLETED,
	}, nil
}

// fakeResolver makes scenario= shorthand deterministic in tests.
type fakeResolver struct {
	URL string
	Err error
}

func (f *fakeResolver) ResolveScenarioURLDefault(_ context.Context, slug string) (string, error) {
	if f.Err != nil {
		return "", f.Err
	}
	if f.URL == "" {
		return "", errors.New("no fake URL configured for slug " + slug)
	}
	return f.URL, nil
}
