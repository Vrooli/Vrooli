// Package testing provides LLM-based skill testing via Ollama.
//
// DOC: docs/reference/api-endpoints.md#testing-ollama
// DOC: docs/internal/SEAMS.md#6-testingollamaclient
package testing

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"prompt-manager/skills"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Handlers provides HTTP handlers for skill testing operations.
type Handlers struct {
	repo      TestRepository
	llmClient LLMClient
	store     skills.SkillStore
}

// NewHandlers creates a new testing handler.
func NewHandlers(repo TestRepository, llmClient LLMClient, store skills.SkillStore) *Handlers {
	return &Handlers{
		repo:      repo,
		llmClient: llmClient,
		store:     store,
	}
}

// Test handles POST /skills/{id}/test - tests a skill with Ollama.
func (h *Handlers) Test(w http.ResponseWriter, r *http.Request) {
	if !h.llmClient.IsEnabled() {
		http.Error(w, "Skill testing is not available (Ollama not configured)", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	skill, folder, err := h.store.FindByID(id)
	if err != nil {
		http.Error(w, "Skill not found", http.StatusNotFound)
		return
	}

	content, err := h.store.GetContent(folder, skill.File)
	if err != nil {
		http.Error(w, "Failed to load skill content", http.StatusInternalServerError)
		return
	}

	var req TestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Default values
	if req.Model == "" {
		req.Model = "llama3.2"
	}
	maxTokens := 1000
	if req.MaxTokens != nil {
		maxTokens = *req.MaxTokens
	}
	temperature := 0.7
	if req.Temperature != nil {
		temperature = *req.Temperature
	}

	// Replace variables in content
	finalContent := content
	for key, value := range req.Variables {
		finalContent = strings.ReplaceAll(finalContent, "{{"+key+"}}", value)
	}

	// Call LLM
	llmResp, responseTime, err := h.llmClient.Generate(req.Model, finalContent, maxTokens, temperature)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Store test result
	testID := uuid.New().String()
	varsJSON, _ := json.Marshal(req.Variables)
	varsStr := string(varsJSON)

	result := &TestResult{
		ID:           testID,
		SkillID:      id,
		Model:        req.Model,
		InputVars:    &varsStr,
		Response:     &llmResp.Response,
		ResponseTime: &responseTime,
		TokenCount:   &llmResp.EvalCount,
		TestedAt:     time.Now(),
	}

	// Best effort save - don't fail the request if storage fails
	h.repo.Save(result)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(TestResponse{
		TestID:       testID,
		Model:        req.Model,
		Response:     llmResp.Response,
		ResponseTime: responseTime,
		TokenCount:   llmResp.EvalCount,
		TestedAt:     time.Now(),
	})
}

// GetHistory handles GET /skills/{id}/test-history - returns test history.
func (h *Handlers) GetHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	results, err := h.repo.GetHistory(id, 20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
