// Package testing provides LLM-based prompt testing via Ollama.
package testing

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"prompt-manager/prompts"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Handlers provides HTTP handlers for prompt testing operations.
type Handlers struct {
	repo         *Repository
	ollamaClient *OllamaClient
	store        *prompts.Store
}

// NewHandlers creates a new testing handler.
func NewHandlers(repo *Repository, ollamaClient *OllamaClient, store *prompts.Store) *Handlers {
	return &Handlers{
		repo:         repo,
		ollamaClient: ollamaClient,
		store:        store,
	}
}

// Test handles POST /prompts/{id}/test - tests a prompt with Ollama.
func (h *Handlers) Test(w http.ResponseWriter, r *http.Request) {
	if !h.ollamaClient.IsEnabled() {
		http.Error(w, "Prompt testing is not available (Ollama not configured)", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	prompt, folder, err := h.store.FindByID(id)
	if err != nil {
		http.Error(w, "Prompt not found", http.StatusNotFound)
		return
	}

	content, err := h.store.GetContent(folder, prompt.File)
	if err != nil {
		http.Error(w, "Failed to load prompt content", http.StatusInternalServerError)
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

	// Call Ollama
	ollamaResp, responseTime, err := h.ollamaClient.Generate(req.Model, finalContent, maxTokens, temperature)
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
		PromptID:     id,
		Model:        req.Model,
		InputVars:    &varsStr,
		Response:     &ollamaResp.Response,
		ResponseTime: &responseTime,
		TokenCount:   &ollamaResp.EvalCount,
		TestedAt:     time.Now(),
	}

	// Best effort save - don't fail the request if storage fails
	h.repo.Save(result)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(TestResponse{
		TestID:       testID,
		Model:        req.Model,
		Response:     ollamaResp.Response,
		ResponseTime: responseTime,
		TokenCount:   ollamaResp.EvalCount,
		TestedAt:     time.Now(),
	})
}

// GetHistory handles GET /prompts/{id}/test-history - returns test history.
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
