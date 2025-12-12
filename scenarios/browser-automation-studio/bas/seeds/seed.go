package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type projectResponse struct {
	ProjectID string `json:"project_id"`
	Project   struct {
		ID string `json:"id"`
	} `json:"project"`
	ID string `json:"id"`
}

type workflowResponse struct {
	WorkflowID string `json:"workflow_id"`
	Workflow   struct {
		ID string `json:"id"`
	} `json:"workflow"`
	ID string `json:"id"`
}

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
	apiURL := fmt.Sprintf("http://localhost:%s/api/v1", apiPort)

	client := &http.Client{Timeout: 10 * time.Second}

	if err := healthCheck(client, apiURL); err != nil {
		return fmt.Errorf("api health check failed: %w", err)
	}

	statePath := filepath.Join(scenarioDir, "coverage", "runtime", "seed-state.json")
	if err := os.MkdirAll(filepath.Dir(statePath), 0o755); err != nil {
		return fmt.Errorf("create seed-state dir: %w", err)
	}

	seed, err := loadExistingState(statePath)
	if err != nil {
		return err
	}

	if seed.ProjectName == "" {
		seed.ProjectName = fmt.Sprintf("Demo Browser Automations %d", time.Now().Unix())
	}
	if seed.WorkflowName == "" {
		seed.WorkflowName = fmt.Sprintf("Demo Smoke Workflow %d", time.Now().Unix())
	}
	if seed.ProjectFolder == "" {
		seed.ProjectFolder = "/demo"
	}

	projectID, err := createProject(client, apiURL, seed.ProjectName, scenarioDir)
	if err != nil {
		return err
	}
	seed.ProjectID = projectID

	workflowID, err := createWorkflow(client, apiURL, seed.WorkflowName, seed.ProjectFolder, projectID)
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
	if port := strings.TrimSpace(os.Getenv("API_PORT")); port != "" {
		return trimPort(port)
	}
	out, err := exec.Command("vrooli", "scenario", "port", scenario, "API_PORT").Output()
	if err != nil {
		return "", fmt.Errorf("resolve API_PORT: %w", err)
	}
	return trimPort(string(out))
}

func trimPort(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("API_PORT empty")
	}
	if strings.Contains(raw, "=") {
		parts := strings.Split(raw, "=")
		raw = parts[len(parts)-1]
		raw = strings.TrimSpace(raw)
	}
	return raw, nil
}

func healthCheck(client *http.Client, apiURL string) error {
	req, _ := http.NewRequest("GET", apiURL+"/health", nil)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
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

func createProject(client *http.Client, apiURL, name, scenarioDir string) (string, error) {
	payload := map[string]string{
		"name":        name,
		"description": "Seeded project for BAS integration testing",
		"folder_path": filepath.Join(scenarioDir, "data", "projects", "demo"),
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", apiURL+"/projects", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("create project request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("create project unexpected status: %d", resp.StatusCode)
	}

	var parsed projectResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("decode project response: %w", err)
	}

	id := firstNonEmpty(parsed.ProjectID, parsed.Project.ID, parsed.ID)
	if id == "" {
		return "", errors.New("project id missing in response")
	}
	return id, nil
}

func createWorkflow(client *http.Client, apiURL, name, folder, projectID string) (string, error) {
	workflowDefinition := map[string]interface{}{
		"nodes": []map[string]interface{}{
			{
				"id":   "seed-evaluate",
				"type": "evaluate",
				"position": map[string]interface{}{
					"x": 0, "y": 0,
				},
				"data": map[string]interface{}{
					"label":       "Get page title",
					"expression":  "document.title",
					"storeResult": "pageTitle",
				},
			},
			{
				"id":   "seed-wait",
				"type": "wait",
				"position": map[string]interface{}{
					"x": 240, "y": 0,
				},
				"data": map[string]interface{}{
					"label":      "Wait for test assertions",
					"durationMs": 5000,
					"waitType":   "duration",
				},
			},
		},
		"edges": []map[string]interface{}{
			{"id": "seed-edge", "source": "seed-evaluate", "target": "seed-wait", "type": "smoothstep"},
		},
	}

	payload := map[string]interface{}{
		"project_id":      projectID,
		"name":            name,
		"folder_path":     folder,
		"flow_definition": workflowDefinition,
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", apiURL+"/workflows/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("create workflow request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("create workflow unexpected status: %d", resp.StatusCode)
	}

	var parsed workflowResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("decode workflow response: %w", err)
	}

	id := firstNonEmpty(parsed.WorkflowID, parsed.Workflow.ID, parsed.ID)
	if id == "" {
		return "", errors.New("workflow id missing in response")
	}
	return id, nil
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

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
