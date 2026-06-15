package browsercapture

import (
	"context"

	"test-genie/internal/playbooks/execution"
)

// FakeWorkflowClient is the test double for WorkflowClient. It returns a canned
// ParsedTimeline so capture logic can be exercised without a live BAS engine.
//
// seam: FakeWorkflowClient is the test wiring for the WorkflowClient seam
// (browsercapture.go). It records the executed workflow definition and returns
// Timeline / the configured errors.
type FakeWorkflowClient struct {
	// Timeline is returned by GetTimeline.
	Timeline *execution.ParsedTimeline
	// ExecuteErr, when set, fails ExecuteWorkflowWithParams.
	ExecuteErr error
	// WaitErr, when set, is returned by WaitForCompletionWithProgress (a
	// non-fatal outcome — capture proceeds to read the timeline).
	WaitErr error
	// TimelineErr, when set, fails GetTimeline.
	TimelineErr error
	// Asset, when set, is returned by DownloadAsset (the screenshot bytes).
	Asset []byte

	// LastDefinition records the workflow map passed to the last execute call.
	LastDefinition map[string]any
}

// ExecuteWorkflowWithParams records the definition and returns a fixed id.
func (f *FakeWorkflowClient) ExecuteWorkflowWithParams(ctx context.Context, definition map[string]any, name, description string, params *execution.ExecutionParams) (string, error) {
	f.LastDefinition = definition
	if f.ExecuteErr != nil {
		return "", f.ExecuteErr
	}
	return "fake-exec-id", nil
}

// WaitForCompletionWithProgress returns the configured (non-fatal) wait error.
func (f *FakeWorkflowClient) WaitForCompletionWithProgress(ctx context.Context, executionID string, callback execution.ProgressCallback) error {
	return f.WaitErr
}

// GetTimeline returns the canned timeline or the configured error.
func (f *FakeWorkflowClient) GetTimeline(ctx context.Context, executionID string) (parsedTimeline, error) {
	if f.TimelineErr != nil {
		return nil, f.TimelineErr
	}
	return f.Timeline, nil
}

// DownloadAsset returns the canned asset bytes.
func (f *FakeWorkflowClient) DownloadAsset(ctx context.Context, assetURL string) ([]byte, error) {
	return f.Asset, nil
}
