package backlog

import (
	"context"
	"path/filepath"
	"testing"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/promptmanager"
	"swarm-manager/internal/testutil"
	"swarm-manager/internal/transitions"
)

func setupTestHandler(t *testing.T) (*Handler, string) {
	t.Helper()
	rootDir := t.TempDir()
	for _, dir := range backlogKindDirs {
		testutil.MakeDir(t, filepath.Join(rootDir, dir))
	}
	h := NewHandler(rootDir, rootDir)
	installTransitionRegistry(t, h)
	scopeExecutionQueuerForTest(t, h, rootDir, nil)
	return h, rootDir
}

func disableAutoWorkshopSettings(t *testing.T, rootDir string) {
	t.Helper()
	t.Setenv("SCENARIO_ROOT", rootDir)
}

func installTransitionRegistry(t *testing.T, h *Handler) {
	t.Helper()
	registry, err := transitions.LoadDir(filepath.Join("..", "..", "..", ".vrooli", "swarm-transitions"))
	if err != nil {
		t.Fatalf("load transition registry: %v", err)
	}
	h.SetTransitionRegistry(registry)
}

func setupTestHandlerWithAgent(t *testing.T, agent agentmanager.Service) (*Handler, string) {
	t.Helper()
	rootDir := t.TempDir()
	for _, dir := range backlogKindDirs {
		testutil.MakeDir(t, filepath.Join(rootDir, dir))
	}
	h := NewHandlerWithClients(rootDir, rootDir, agent, &promptmanager.MockClient{Result: "test prompt"})
	installTransitionRegistry(t, h)
	scopeExecutionQueuerForTest(t, h, rootDir, agent)
	return h, rootDir
}

func scopeExecutionQueuerForTest(t *testing.T, h *Handler, rootDir string, agent agentmanager.Service) {
	t.Helper()
	h.SetExecutionQueuer(execution.NewService(execution.ServiceConfig{
		DataRoot: rootDir, RepoRoot: rootDir,
		StorePath:    filepath.Join(rootDir, ".vrooli", "execution-runs.json"),
		AgentService: agent, PromptClient: &promptmanager.MockClient{Result: "test prompt"},
	}))
}

func createTestItem(t *testing.T, rootDir string, kind BacklogKind, item BacklogItem) {
	t.Helper()
	item.Kind = kind
	testutil.WriteJSONFile(t, filepath.Join(rootDir, backlogKindDirs[kind], item.Name, "spec.json"), item)
}

func createReadyTestItem(t *testing.T, rootDir string, kind BacklogKind, item BacklogItem) {
	t.Helper()
	if item.PlanRef == nil {
		item.PlanRef = &PlanRef{Provider: PlanRefProviderPlanManager, PlanID: "test-plan-" + item.Name, Slug: "test-plan-" + item.Name, Role: PlanRefRoleExecutionSpec}
	}
	createTestItem(t, rootDir, kind, item)
}

type mockAgentService struct {
	result agentmanager.RunResult
}

type backlogListResponse struct {
	Items []BacklogItem `json:"items"`
}

type backlogItemResponse struct {
	Item BacklogItem `json:"item"`
}

type backlogFilesResponse struct {
	Files []BacklogFile `json:"files"`
}

func strPtr(value string) *string { return &value }

func (m *mockAgentService) IsEnabled() bool                    { return true }
func (m *mockAgentService) IsAvailable(_ context.Context) bool { return true }
func (m *mockAgentService) ResolveURL(_ context.Context) (string, error) {
	return "http://agent", nil
}
func (m *mockAgentService) GetProfileID() string                               { return "" }
func (m *mockAgentService) ApproveRun(_ context.Context, _, _, _ string) error { return nil }
func (m *mockAgentService) ContinueRun(_ context.Context, _, _ string) error   { return nil }
func (m *mockAgentService) GetRunState(_ context.Context, _ string) (agentmanager.RunState, error) {
	return agentmanager.RunState{}, nil
}

func (m *mockAgentService) GetRunDiff(_ context.Context, runID string) (agentmanager.RunDiff, error) {
	return agentmanager.RunDiff{RunID: runID}, nil
}
func (m *mockAgentService) StopRun(_ context.Context, _ string) error { return nil }
