package pipeline

import (
	"context"
	"testing"
)

// mockOrchestrator implements the domain seam consumed by ConnectService.
type mockOrchestrator struct {
	runResult     *Status
	runError      error
	getResult     *Status
	getFound      bool
	cancelSuccess bool
	pipelines     []*Status
	resumeResult  *Status
	resumeError   error
	createResult  *Status
	createError   error
	startResult   *Status
	startError    error
	updatedConfig *Config
	updateError   error
}

func (m *mockOrchestrator) RunPipeline(_ context.Context, _ *Config) (*Status, error) {
	return m.runResult, m.runError
}

func (m *mockOrchestrator) RunPipelineBlocking(_ context.Context, _ *Config, _ int) (*Status, error) {
	return m.runResult, m.runError
}

func (m *mockOrchestrator) CreateIdlePipeline(_ *Config) (*Status, error) {
	return m.createResult, m.createError
}

func (m *mockOrchestrator) StartPipeline(_ context.Context, _ string) (*Status, error) {
	return m.startResult, m.startError
}

func (m *mockOrchestrator) StartPipelineBlocking(_ context.Context, _ string, _ int) (*Status, error) {
	return m.startResult, m.startError
}

func (m *mockOrchestrator) UpdatePipelineConfig(_ string, config *Config) error {
	m.updatedConfig = config
	return m.updateError
}
func (m *mockOrchestrator) GetStatus(_ string) (*Status, bool) { return m.getResult, m.getFound }
func (m *mockOrchestrator) CancelPipeline(_ string) bool       { return m.cancelSuccess }
func (m *mockOrchestrator) ListPipelines() []*Status           { return m.pipelines }
func (m *mockOrchestrator) ResumePipeline(_ context.Context, _ string, _ *Config) (*Status, error) {
	return m.resumeResult, m.resumeError
}

func TestHandlerIsDomainDependencyHolder(t *testing.T) {
	orchestrator := &mockOrchestrator{}
	handler := NewHandler(WithOrchestrator(orchestrator))
	if handler.orchestrator != orchestrator {
		t.Fatal("handler did not retain the pipeline orchestration seam")
	}
}

func TestCleanBundleRejectsStagingWithoutPipelineID(t *testing.T) {
	if _, err := NewHandler().cleanBundle("example", FrameworkElectron, "staging", ""); err == nil {
		t.Fatal("cleanBundle accepted staging cleanup without a pipeline id")
	}
}
