package main

import (
	"context"
	"sync"
	"testing"
)

// FakeWorkspaceSandboxAPI implements WorkspaceSandboxAPI for testing.
// Records all calls and returns configurable responses.
type FakeWorkspaceSandboxAPI struct {
	mu sync.Mutex

	// Configurable responses
	CommitPreviewResult *workspaceSandboxCommitPreview
	CommitPreviewError  error
	MarkCommittedError  error
	ProvenanceResult    *workspaceSandboxProvenanceResponse
	ProvenanceError     error

	// Recorded calls
	markCommittedCalls []markCommittedCall
	provenanceCalls    []string // projectRoot values
}

type markCommittedCall struct {
	ProjectRoot   string
	FilePaths     []string
	CommitHash    string
	CommitMessage string
}

func NewFakeWorkspaceSandboxAPI() *FakeWorkspaceSandboxAPI {
	return &FakeWorkspaceSandboxAPI{}
}

func (f *FakeWorkspaceSandboxAPI) GetCommitPreview(ctx context.Context, projectRoot string) (*workspaceSandboxCommitPreview, error) {
	if f.CommitPreviewError != nil {
		return nil, f.CommitPreviewError
	}
	if f.CommitPreviewResult != nil {
		return f.CommitPreviewResult, nil
	}
	return &workspaceSandboxCommitPreview{}, nil
}

func (f *FakeWorkspaceSandboxAPI) GetCommitPreviewForPaths(ctx context.Context, projectRoot string, paths []string) (*workspaceSandboxCommitPreview, error) {
	return f.GetCommitPreview(ctx, projectRoot)
}

func (f *FakeWorkspaceSandboxAPI) MarkCommitted(ctx context.Context, projectRoot string, filePaths []string, commitHash, commitMessage string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.markCommittedCalls = append(f.markCommittedCalls, markCommittedCall{
		ProjectRoot:   projectRoot,
		FilePaths:     filePaths,
		CommitHash:    commitHash,
		CommitMessage: commitMessage,
	})

	return f.MarkCommittedError
}

func (f *FakeWorkspaceSandboxAPI) GetProvenanceByRun(ctx context.Context, projectRoot string) (*workspaceSandboxProvenanceResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.provenanceCalls = append(f.provenanceCalls, projectRoot)

	if f.ProvenanceError != nil {
		return nil, f.ProvenanceError
	}
	if f.ProvenanceResult != nil {
		return f.ProvenanceResult, nil
	}
	return &workspaceSandboxProvenanceResponse{}, nil
}

// MarkCommittedCalls returns a copy of recorded MarkCommitted calls.
func (f *FakeWorkspaceSandboxAPI) MarkCommittedCalls() []markCommittedCall {
	f.mu.Lock()
	defer f.mu.Unlock()

	result := make([]markCommittedCall, len(f.markCommittedCalls))
	copy(result, f.markCommittedCalls)
	return result
}

// AssertMarkCommittedCalled verifies MarkCommitted was called at least once.
func (f *FakeWorkspaceSandboxAPI) AssertMarkCommittedCalled(t *testing.T) {
	t.Helper()
	calls := f.MarkCommittedCalls()
	if len(calls) == 0 {
		t.Errorf("expected MarkCommitted to be called, but it was not")
	}
}

// AssertMarkCommittedNotCalled verifies MarkCommitted was never called.
func (f *FakeWorkspaceSandboxAPI) AssertMarkCommittedNotCalled(t *testing.T) {
	t.Helper()
	calls := f.MarkCommittedCalls()
	if len(calls) > 0 {
		t.Errorf("expected MarkCommitted not to be called, but got %d calls", len(calls))
	}
}
