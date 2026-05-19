package capture

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	basebase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"

	"github.com/vrooli/browser-automation-studio/services/workflow"
	"github.com/vrooli/browser-automation-studio/storage"
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

	// ExportCalls counts ExportToFolder invocations so tests can assert
	// the harvest seam was (or was not) reached.
	ExportCalls   int
	LastExportID  uuid.UUID
	LastExportDir string
	// ExportLayout, when set, dictates files to create under outputDir
	// during ExportToFolder. Keys are relative paths; values are file
	// contents. Lets tests exercise harvestArtifacts deterministically
	// without standing up the real export pipeline.
	ExportLayout map[string]string
	ExportErr    error
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

func (f *fakeExecutor) ExportToFolder(
	_ context.Context,
	executionID uuid.UUID,
	outputDir string,
	_ storage.StorageInterface,
) error {
	f.ExportCalls++
	f.LastExportID = executionID
	f.LastExportDir = outputDir
	if f.ExportErr != nil {
		return f.ExportErr
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	for relPath, content := range f.ExportLayout {
		full := filepath.Join(outputDir, relPath)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
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
