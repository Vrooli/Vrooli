package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type EcosystemTaskStatusResponse struct {
	Configured bool                  `json:"configured"`
	Supported  bool                  `json:"supported"`
	Error      string                `json:"error,omitempty"`
	ManageURL  string                `json:"manage_url,omitempty"`
	Task       *EcosystemTaskSummary `json:"task,omitempty"`
}

type EcosystemTaskSummary struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	Priority  string `json:"priority"`
	Category  string `json:"category"`
	Operation string `json:"operation"`
	Target    string `json:"target"`
	ViewURL   string `json:"view_url,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type EcosystemTaskCreationRequest struct {
	Title          string `json:"title"`
	Priority       string `json:"priority"`
	Category       string `json:"category"`
	EffortEstimate string `json:"effort_estimate"`
	Notes          string `json:"notes"`
}

type ecosystemTaskListResponse struct {
	Tasks []ecosystemTaskItem `json:"tasks"`
	Count int                 `json:"count"`
}

type ecosystemTaskItem struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Type      string   `json:"type"`
	Operation string   `json:"operation"`
	Target    string   `json:"target"`
	Targets   []string `json:"targets"`
	Status    string   `json:"status"`
	Priority  string   `json:"priority"`
	Category  string   `json:"category"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

type ecosystemTaskCreatePayload struct {
	ID                 string   `json:"id,omitempty"`
	Title              string   `json:"title"`
	Type               string   `json:"type"`
	Operation          string   `json:"operation"`
	Target             string   `json:"target"`
	Targets            []string `json:"targets"`
	Category           string   `json:"category"`
	Priority           string   `json:"priority"`
	EffortEstimate     string   `json:"effort_estimate"`
	Urgency            string   `json:"urgency"`
	Status             string   `json:"status"`
	CreatedBy          string   `json:"created_by"`
	Notes              string   `json:"notes"`
	Tags               []string `json:"tags"`
	RelatedScenarios   []string `json:"related_scenarios"`
	ValidationCriteria []string `json:"validation_criteria"`
}

var ecosystemHTTPClient = &http.Client{Timeout: 15 * time.Second}

func resolveEcosystemManagerBaseURL(ctx context.Context) (string, error) {
	value := strings.TrimSpace(os.Getenv("ECOSYSTEM_MANAGER_URL"))
	if value != "" {
		return strings.TrimRight(value, "/"), nil
	}

	port, err := resolveScenarioPortViaCLI(ctx, "ecosystem-manager", "API_PORT")
	if err != nil {
		return "", fmt.Errorf("ecosystem-manager is not running (start the scenario or set ECOSYSTEM_MANAGER_URL)")
	}
	if port <= 0 {
		return "", fmt.Errorf("ecosystem-manager port could not be resolved")
	}

	return fmt.Sprintf("http://127.0.0.1:%d", port), nil
}

func handleGetEcosystemTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entityType := strings.ToLower(vars["type"])
	entityName := strings.TrimSpace(vars["name"])
	if entityName == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "entity name is required"})
		return
	}

	baseURL, resolveErr := resolveEcosystemManagerBaseURL(r.Context())
	status := EcosystemTaskStatusResponse{
		Configured: resolveErr == nil,
		Supported:  strings.ToLower(entityType) == EntityTypeScenario,
	}

	if resolveErr != nil {
		status.Error = resolveErr.Error()
		respondJSON(w, http.StatusOK, status)
		return
	}
	status.ManageURL = strings.TrimRight(baseURL, "/")
	if !status.Supported {
		status.Error = "ecosystem tasks are only supported for scenarios at this time"
		respondJSON(w, http.StatusOK, status)
		return
	}

	task, err := lookupEcosystemTask(baseURL, entityType, entityName)
	if err != nil {
		status.Error = err.Error()
	} else {
		status.Task = task
	}

	respondJSON(w, http.StatusOK, status)
}

func handleCreateEcosystemTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entityType := strings.ToLower(vars["type"])
	entityName := strings.TrimSpace(vars["name"])
	if entityName == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "entity name is required"})
		return
	}

	baseURL, resolveErr := resolveEcosystemManagerBaseURL(r.Context())
	if resolveErr != nil {
		respondJSON(w, http.StatusPreconditionFailed, map[string]string{"error": resolveErr.Error()})
		return
	}
	if entityType != EntityTypeScenario {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "ecosystem tasks currently require a scenario target"})
		return
	}

	var req EcosystemTaskCreationRequest
	if r.Body != nil {
		body, _ := io.ReadAll(r.Body)
		if len(body) > 0 {
			if err := json.Unmarshal(body, &req); err != nil {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request payload"})
				return
			}
		}
	}

	payload := ecosystemTaskCreatePayload{
		Title:              deriveTaskTitle(req.Title, entityName),
		Type:               entityType,
		Operation:          "improver",
		Target:             entityName,
		Targets:            []string{entityName},
		Category:           defaultString(req.Category, "ai_tools"),
		Priority:           defaultString(req.Priority, "high"),
		EffortEstimate:     defaultString(req.EffortEstimate, "medium"),
		Urgency:            "standard",
		Status:             "pending",
		CreatedBy:          "prd-control-tower",
		Notes:              buildTaskNotes(req.Notes, entityType, entityName),
		Tags:               []string{"prd-control-tower", entityName},
		RelatedScenarios:   []string{entityName},
		ValidationCriteria: []string{"All operational targets linked to requirements", "All requirements passing"},
	}

	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(payload); err != nil {
		respondInternalError(w, "failed to encode ecosystem task", err)
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/tasks", strings.TrimRight(baseURL, "/"))
	resp, err := ecosystemHTTPClient.Post(endpoint, "application/json", buf)
	if err != nil {
		respondJSON(w, http.StatusBadGateway, map[string]string{"error": fmt.Sprintf("ecosystem-manager unavailable: %v", err)})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		respondJSON(w, resp.StatusCode, map[string]string{"error": strings.TrimSpace(string(body))})
		return
	}

	task, err := lookupEcosystemTask(baseURL, entityType, entityName)
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]string{"message": "task created but status lookup failed", "error": err.Error()})
		return
	}

	status := EcosystemTaskStatusResponse{
		Configured: true,
		Supported:  true,
		ManageURL:  strings.TrimRight(baseURL, "/"),
		Task:       task,
	}
	respondJSON(w, http.StatusCreated, status)
}

func lookupEcosystemTask(baseURL, entityType, entityName string) (*EcosystemTaskSummary, error) {
	statuses := []string{"in-progress", "pending", "review"}
	trimmedName := strings.ToLower(strings.TrimSpace(entityName))
	if trimmedName == "" {
		return nil, fmt.Errorf("entity name is required")
	}

	for _, status := range statuses {
		endpoint := fmt.Sprintf("%s/api/v1/tasks?status=%s&type=%s&operation=improver",
			strings.TrimRight(baseURL, "/"), url.QueryEscape(status), url.QueryEscape(entityType))
		resp, err := ecosystemHTTPClient.Get(endpoint)
		if err != nil {
			return nil, fmt.Errorf("failed to query ecosystem-manager: %w", err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read ecosystem-manager response: %w", err)
		}
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("ecosystem-manager returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}

		var list ecosystemTaskListResponse
		if err := json.Unmarshal(body, &list); err != nil {
			return nil, fmt.Errorf("failed to parse ecosystem response: %w", err)
		}

		for _, item := range list.Tasks {
			if !strings.EqualFold(item.Operation, "improver") {
				continue
			}
			if matchesTaskTarget(item, trimmedName) {
				return &EcosystemTaskSummary{
					ID:        item.ID,
					Title:     item.Title,
					Status:    item.Status,
					Priority:  item.Priority,
					Category:  item.Category,
					Operation: item.Operation,
					Target:    item.Target,
					ViewURL:   fmt.Sprintf("%s/tasks/%s", strings.TrimRight(baseURL, "/"), item.ID),
					CreatedAt: item.CreatedAt,
					UpdatedAt: item.UpdatedAt,
				}, nil
			}
		}
	}

	return nil, nil
}

func matchesTaskTarget(item ecosystemTaskItem, targetLower string) bool {
	if strings.EqualFold(item.Target, targetLower) {
		return true
	}
	for _, candidate := range item.Targets {
		if strings.EqualFold(candidate, targetLower) {
			return true
		}
	}
	return false
}

func deriveTaskTitle(requested, entityName string) string {
	trimmed := strings.TrimSpace(requested)
	if trimmed != "" {
		return trimmed
	}
	return fmt.Sprintf("Implement %s requirements", entityName)
}

func defaultString(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func buildTaskNotes(custom, entityType, entityName string) string {
	base := fmt.Sprintf("Autogenerated from PRD Control Tower for %s/%s. Source PRD: %s/%s/PRD.md",
		entityType, entityName, entityType+"s", entityName)
	trimmed := strings.TrimSpace(custom)
	if trimmed == "" {
		return base
	}
	return fmt.Sprintf("%s\n\n%s", base, trimmed)
}
