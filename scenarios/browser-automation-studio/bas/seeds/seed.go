package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

type projectListResponse struct {
	Projects []struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		FolderPath string `json:"folder_path"`
		FolderPB   string `json:"folderPath"`
	} `json:"projects"`
}

type workflowResponse struct {
	WorkflowID string `json:"workflow_id"`
	Workflow   struct {
		ID string `json:"id"`
	} `json:"workflow"`
	ID string `json:"id"`
}

type workflowListResponse struct {
	Workflows []struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		ProjectID string `json:"project_id"`
		ProjectPB string `json:"projectId"`
		Folder    string `json:"folder_path"`
		FolderPB  string `json:"folderPath"`
	} `json:"workflows"`
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
		seed.ProjectName = "Demo Browser Automations"
	}
	if seed.WorkflowName == "" {
		seed.WorkflowName = "Demo Smoke Workflow"
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
	// Prefer the scenario lifecycle port resolver so we always hit the running BAS API.
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

	// Prefer a clean numeric line (some lifecycle commands print extra context).
	cleaned := strings.ReplaceAll(raw, "\r", "\n")
	lines := strings.Split(cleaned, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		allDigits := true
		for _, r := range line {
			if r < '0' || r > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			return line, nil
		}
	}

	if strings.Contains(raw, "=") {
		parts := strings.Split(raw, "=")
		raw = parts[len(parts)-1]
		raw = strings.TrimSpace(raw)
	}
	raw = strings.TrimSpace(raw)
	orig := raw
	allDigits := true
	for _, r := range raw {
		if r < '0' || r > '9' {
			allDigits = false
			break
		}
	}
	if !allDigits {
		// Fall back to extracting the first number from the string.
		var b strings.Builder
		for _, r := range strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(orig, "\r", " "), "\n", " ")) {
			if r >= '0' && r <= '9' {
				b.WriteRune(r)
			} else if b.Len() > 0 {
				break
			}
		}
		raw = b.String()
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("API_PORT empty")
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
	folder := filepath.Join(resolveVrooliRoot(scenarioDir), "projects", "demo")
	payload := map[string]string{
		"name":        name,
		"description": "Seeded project for BAS integration testing",
		"folder_path": folder,
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", apiURL+"/projects", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("create project request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		id, err := findExistingProjectID(client, apiURL, folder, name)
		if err != nil {
			return "", err
		}
		_ = ensureProjectName(client, apiURL, id, name)
		return id, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create project unexpected status: %d body=%s", resp.StatusCode, strings.TrimSpace(string(raw)))
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
	// Seed a minimal-but-real workflow so the builder canvas has nodes to render and
	// the Execute flow produces observable execution state.
	workflowDefinition := map[string]any{
		"nodes": []map[string]any{
			{
				"id":   "navigate-dashboard",
				"type": "navigate",
				"data": map[string]any{
					"label":           "Navigate to BAS dashboard",
					"destinationType": "scenario",
					"scenario":        "browser-automation-studio",
					"scenarioPath":    "/",
					"waitUntil":       "networkidle",
					"timeoutMs":       45000,
					"waitForMs":       1000,
				},
			},
			{
				"id":   "assert-app-ready",
				"type": "assert",
				"data": map[string]any{
					"label":          "App ready",
					"selector":       "[data-testid=\"app-ready\"]",
					"assertMode":     "exists",
					"timeoutMs":      15000,
					"failureMessage": "BAS UI should load for the seeded demo workflow",
				},
			},
		},
		"edges": []map[string]any{
			{
				"id":     "e1",
				"source": "navigate-dashboard",
				"target": "assert-app-ready",
				"type":   "smoothstep",
			},
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
	if resp.StatusCode == http.StatusConflict {
		return findExistingWorkflowID(client, apiURL, name, folder, projectID)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create workflow unexpected status: %d body=%s", resp.StatusCode, strings.TrimSpace(string(raw)))
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

func findExistingProjectID(client *http.Client, apiURL, folderPath, projectName string) (string, error) {
	req, _ := http.NewRequest("GET", apiURL+"/projects", nil)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("list projects request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("list projects unexpected status: %d", resp.StatusCode)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read project list response: %w", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", fmt.Errorf("decode project list response: %w", err)
	}

	projectsAny, ok := payload["projects"].([]any)
	if !ok {
		return "", errors.New("unexpected project list response shape")
	}

	folderPath = filepath.Clean(folderPath)
	projectName = strings.TrimSpace(projectName)

	for _, item := range projectsAny {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}

		projectMap := itemMap
		if nested, ok := itemMap["project"].(map[string]any); ok {
			projectMap = nested
		}

		pID := firstNonEmpty(stringField(projectMap, "id"))
		pFolder := firstNonEmpty(stringField(projectMap, "folder_path"), stringField(projectMap, "folderPath"))
		pName := firstNonEmpty(stringField(projectMap, "name"))
		if strings.TrimSpace(pID) == "" || strings.TrimSpace(pFolder) == "" {
			continue
		}
		if filepath.Clean(pFolder) == folderPath {
			return strings.TrimSpace(pID), nil
		}

		if projectName != "" && strings.TrimSpace(pName) == projectName {
			return strings.TrimSpace(pID), nil
		}
	}
	return "", errors.New("project already exists but could not be resolved by folder path or name")
}

func findExistingWorkflowID(client *http.Client, apiURL, name, folder, projectID string) (string, error) {
	req, _ := http.NewRequest("GET", apiURL+"/workflows?folder_path="+url.QueryEscape(folder), nil)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("list workflows request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("list workflows unexpected status: %d", resp.StatusCode)
	}
	var parsed workflowListResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("decode workflow list response: %w", err)
	}
	for _, wf := range parsed.Workflows {
		wfProject := firstNonEmpty(wf.ProjectID, wf.ProjectPB)
		wfFolder := firstNonEmpty(wf.Folder, wf.FolderPB)
		if strings.TrimSpace(wf.ID) == "" {
			continue
		}
		if wfProject == projectID && wfFolder == folder && wf.Name == name {
			return strings.TrimSpace(wf.ID), nil
		}
	}
	return "", errors.New("workflow already exists but could not be resolved by name/folder/project")
}

func ensureProjectName(client *http.Client, apiURL, projectID, name string) error {
	payload := map[string]string{"name": name}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("PUT", apiURL+"/projects/"+projectID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func stringField(obj map[string]any, key string) string {
	raw, ok := obj[key]
	if !ok || raw == nil {
		return ""
	}
	s, _ := raw.(string)
	return s
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
