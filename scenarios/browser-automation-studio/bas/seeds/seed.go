// Command seed bootstraps the demo project + smoke workflow used by
// integration tests and the local dev environment. It speaks to the BAS
// API via Connect-RPC (the only supported transport now that the
// REST/JSON surface has been retired) so the seed shares schema and
// validation with every other client.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"

	apiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api/apiconnect"
	projectsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/projects"
	"github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/projects/projectsconnect"
	workflowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
)

type seedState struct {
	ProjectID     string `json:"projectId"`
	ProjectName   string `json:"projectName"`
	ProjectFolder string `json:"projectFolder"`
	WorkflowID    string `json:"workflowId"`
	WorkflowName  string `json:"workflowName"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "seed failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	scenarioDir, err := resolveScenarioDir()
	if err != nil {
		return err
	}
	scenarioName := filepath.Base(scenarioDir)

	apiPort, err := resolveAPIPort(scenarioName)
	if err != nil {
		return err
	}
	baseURL := fmt.Sprintf("http://localhost:%s", apiPort)

	httpClient := &http.Client{Timeout: 15 * time.Second}
	projects := projectsconnect.NewProjectsServiceClient(httpClient, baseURL)
	workflows := apiconnect.NewWorkflowsServiceClient(httpClient, baseURL)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	statePath := filepath.Join(scenarioDir, "coverage", "runtime", "seed-state.json")
	if err := os.MkdirAll(filepath.Dir(statePath), 0o755); err != nil {
		return fmt.Errorf("create seed-state dir: %w", err)
	}

	seed, err := loadExistingState(statePath)
	if err != nil {
		return err
	}
	if seed.ProjectName == "" {
		seed.ProjectName = "Demo Browser Automations"
	}
	if seed.WorkflowName == "" {
		seed.WorkflowName = "Demo Smoke Workflow"
	}
	if seed.ProjectFolder == "" {
		seed.ProjectFolder = "/demo"
	}

	folder := filepath.Join(resolveVrooliRoot(scenarioDir), "projects", "demo")
	projectID, err := ensureProject(ctx, projects, seed.ProjectName, folder)
	if err != nil {
		return err
	}
	seed.ProjectID = projectID

	workflowID, err := ensureWorkflow(ctx, workflows, projectID, seed.WorkflowName, seed.ProjectFolder)
	if err != nil {
		return err
	}
	seed.WorkflowID = workflowID

	if err := writeSeedState(statePath, seed); err != nil {
		return err
	}

	fmt.Printf("✅ BAS seed data applied (project %s, workflow %s)\n", seed.ProjectID, seed.WorkflowID)
	fmt.Printf("📝 Wrote seed state to %s\n", statePath)
	return nil
}

func resolveScenarioDir() (string, error) {
	if _, filename, _, ok := runtime.Caller(0); ok {
		// This file lives at <scenario>/bas/seeds/seed.go.
		scenarioDir := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
		if scenarioDir != "" {
			return scenarioDir, nil
		}
	}
	if env := strings.TrimSpace(os.Getenv("TEST_GENIE_SCENARIO_DIR")); env != "" {
		return filepath.Abs(env)
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("determine working directory: %w", err)
	}
	return wd, nil
}

func resolveAPIPort(scenario string) (string, error) {
	if out, err := exec.Command("vrooli", "scenario", "port", scenario, "API_PORT").Output(); err == nil {
		return trimPort(string(out))
	}
	if port := strings.TrimSpace(os.Getenv("API_PORT")); port != "" {
		return trimPort(port)
	}
	return "", errors.New("resolve API_PORT: unable to determine port")
}

func resolveVrooliRoot(scenarioDir string) string {
	if env := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); env != "" {
		return env
	}
	// scenarioDir is expected to be <vrooli-root>/scenarios/<scenario-name>
	return filepath.Clean(filepath.Join(scenarioDir, "..", ".."))
}

func trimPort(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("API_PORT empty")
	}
	cleaned := strings.ReplaceAll(raw, "\r", "\n")
	for _, line := range reverseLines(strings.Split(cleaned, "\n")) {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if isAllDigits(line) {
			return line, nil
		}
	}
	// Best-effort: pull the first run of digits from the input.
	var b strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
			continue
		}
		if b.Len() > 0 {
			break
		}
	}
	if b.Len() == 0 {
		return "", errors.New("API_PORT contains no digits")
	}
	return b.String(), nil
}

func reverseLines(in []string) []string {
	out := make([]string, len(in))
	for i, line := range in {
		out[len(in)-1-i] = line
	}
	return out
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func loadExistingState(path string) (seedState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return seedState{}, nil
		}
		return seedState{}, fmt.Errorf("read seed-state: %w", err)
	}
	var state seedState
	if err := json.Unmarshal(data, &state); err != nil {
		return seedState{}, fmt.Errorf("parse seed-state: %w", err)
	}
	return state, nil
}

// ensureProject creates the demo project if missing, or returns the
// existing project's ID. Idempotent across reruns.
func ensureProject(ctx context.Context, client projectsconnect.ProjectsServiceClient, name, folder string) (string, error) {
	req := connect.NewRequest(&projectsv1.CreateProjectRequest{
		Name:        name,
		Description: "Seeded project for BAS integration testing",
		FolderPath:  folder,
	})
	resp, err := client.CreateProject(ctx, req)
	if err == nil {
		project := resp.Msg.GetProject()
		if project == nil || project.GetId() == "" {
			return "", errors.New("CreateProject returned empty project")
		}
		return project.GetId(), nil
	}

	// AlreadyExists → look up by folder/name. Any other code is fatal.
	var connErr *connect.Error
	if !errors.As(err, &connErr) || connErr.Code() != connect.CodeAlreadyExists {
		return "", fmt.Errorf("create project: %w", err)
	}
	return findProjectByFolderOrName(ctx, client, folder, name)
}

func findProjectByFolderOrName(ctx context.Context, client projectsconnect.ProjectsServiceClient, folder, name string) (string, error) {
	resp, err := client.ListProjects(ctx, connect.NewRequest(&projectsv1.ListProjectsRequest{Limit: 200}))
	if err != nil {
		return "", fmt.Errorf("list projects: %w", err)
	}
	folder = filepath.Clean(folder)
	name = strings.TrimSpace(name)
	for _, entry := range resp.Msg.GetProjects() {
		project := entry.GetProject()
		if project == nil {
			continue
		}
		if filepath.Clean(project.GetFolderPath()) == folder {
			return project.GetId(), nil
		}
		if name != "" && strings.TrimSpace(project.GetName()) == name {
			return project.GetId(), nil
		}
	}
	return "", fmt.Errorf("project exists but no entry matches folder=%q name=%q", folder, name)
}

// ensureWorkflow creates the smoke workflow if missing, or returns the
// existing one. Workflow contents are defined by demoSmokeWorkflowJSON
// so the enum values round-trip via protojson without ever going
// through map[string]any (which previously lost ChangeSource/wait enums
// during snake-case JSON encoding).
func ensureWorkflow(ctx context.Context, client apiconnect.WorkflowsServiceClient, projectID, name, folder string) (string, error) {
	flow := &workflowsv1.WorkflowDefinitionV2{}
	opts := protojson.UnmarshalOptions{DiscardUnknown: false}
	if err := opts.Unmarshal([]byte(demoSmokeWorkflowJSON), flow); err != nil {
		return "", fmt.Errorf("decode smoke workflow definition: %w", err)
	}

	req := connect.NewRequest(&apiv1.CreateWorkflowRequest{
		ProjectId:      projectID,
		Name:           name,
		FolderPath:     folder,
		FlowDefinition: flow,
	})
	resp, err := client.CreateWorkflow(ctx, req)
	if err == nil {
		wf := resp.Msg.GetWorkflow()
		if wf == nil || wf.GetId() == "" {
			return "", errors.New("CreateWorkflow returned empty workflow")
		}
		return wf.GetId(), nil
	}

	var connErr *connect.Error
	if !errors.As(err, &connErr) || connErr.Code() != connect.CodeAlreadyExists {
		return "", fmt.Errorf("create workflow: %w", err)
	}
	return findWorkflowByName(ctx, client, projectID, folder, name)
}

func findWorkflowByName(ctx context.Context, client apiconnect.WorkflowsServiceClient, projectID, folder, name string) (string, error) {
	limit := int32(500)
	resp, err := client.ListWorkflows(ctx, connect.NewRequest(&apiv1.ListWorkflowsRequest{
		ProjectId: &projectID,
		Limit:     &limit,
	}))
	if err != nil {
		return "", fmt.Errorf("list workflows: %w", err)
	}
	for _, wf := range resp.Msg.GetWorkflows() {
		if wf == nil {
			continue
		}
		if wf.GetFolderPath() != folder {
			continue
		}
		if strings.TrimSpace(wf.GetName()) != name {
			continue
		}
		return wf.GetId(), nil
	}
	return "", fmt.Errorf("workflow exists but no entry matches project=%s folder=%q name=%q", projectID, folder, name)
}

func writeSeedState(path string, state seedState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal seed-state: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write seed-state: %w", err)
	}
	return nil
}

// demoSmokeWorkflowJSON encodes the seed workflow in protojson form (the
// canonical wire shape). Enum values use the full proto names so the
// decoder never has to guess — guards against the snake-case-shortcut
// regression that previously silently dropped enum fields.
const demoSmokeWorkflowJSON = `{
  "metadata": {
    "name": "Demo Smoke Workflow",
    "description": "Seeded workflow for BAS integration testing. Navigates to example.com and asserts the body exists.",
    "labels": { "seeded": "true" }
  },
  "settings": {
    "viewportWidth": 1280,
    "viewportHeight": 720,
    "headless": true,
    "timeoutMs": 180000
  },
  "nodes": [
    {
      "id": "11111111-1111-1111-1111-111111111111",
      "action": {
        "type": "ACTION_TYPE_NAVIGATE",
        "navigate": {
          "url": "https://example.com/",
          "destinationType": "NAVIGATE_DESTINATION_TYPE_URL",
          "waitUntil": "NAVIGATE_WAIT_EVENT_LOAD",
          "timeoutMs": 45000
        }
      },
      "position": { "x": 0, "y": 0 }
    },
    {
      "id": "22222222-2222-2222-2222-222222222222",
      "action": {
        "type": "ACTION_TYPE_ASSERT",
        "assert": {
          "selector": "body",
          "mode": "ASSERTION_MODE_EXISTS"
        }
      },
      "executionSettings": {
        "timeoutMs": 15000,
        "waitAfterMs": 500
      },
      "position": { "x": 320, "y": 0 }
    }
  ],
  "edges": [
    {
      "id": "33333333-3333-3333-3333-333333333333",
      "source": "11111111-1111-1111-1111-111111111111",
      "target": "22222222-2222-2222-2222-222222222222",
      "type": "WORKFLOW_EDGE_TYPE_DEFAULT"
    }
  ]
}`
