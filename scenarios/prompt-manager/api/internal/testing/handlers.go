// Package testing provides LLM-based skill testing via Ollama.
//
// DOC: docs/reference/api-endpoints.md#testing-ollama
// DOC: docs/internal/SEAMS.md#6-testingollamaclient
package testing

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"prompt-manager/internal/skills"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

var (
	ErrTestingUnavailable = errors.New("Skill testing is not available (Ollama not configured)")
	ErrSkillNotFound      = errors.New("Skill not found")
	ErrContentLoad        = errors.New("Failed to load skill content")
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

func (h *Handlers) repoFor(ctx context.Context) TestRepository {
	if scoped, ok := h.repo.(interface {
		WithRequestContext(context.Context) TestRepository
	}); ok {
		return scoped.WithRequestContext(ctx)
	}
	return h.repo
}

// RunTest executes a skill test and best-effort persists its result. It is
// transport-neutral so Connect and the legacy HTTP endpoint share behavior.
func (h *Handlers) RunTest(ctx context.Context, id string, req TestRequest) (TestResponse, error) {
	if !h.llmClient.IsEnabled() {
		return TestResponse{}, ErrTestingUnavailable
	}

	skill, folder, err := h.store.FindByID(id)
	if err != nil {
		return TestResponse{}, ErrSkillNotFound
	}
	content, err := h.store.GetContent(folder, skill.File)
	if err != nil {
		return TestResponse{}, ErrContentLoad
	}

	if req.Role == "" {
		req.Role = "chat.small"
	}
	maxTokens := 1000
	if req.MaxTokens != nil {
		maxTokens = *req.MaxTokens
	}
	temperature := 0.7
	if req.Temperature != nil {
		temperature = *req.Temperature
	}
	finalContent := content
	for key, value := range req.Variables {
		finalContent = strings.ReplaceAll(finalContent, "{{"+key+"}}", value)
	}

	llmResp, responseTime, err := h.llmClient.Generate(req.Role, finalContent, maxTokens, temperature)
	if err != nil {
		return TestResponse{}, err
	}

	testedAt := time.Now()
	testID := uuid.New().String()
	varsJSON, _ := json.Marshal(req.Variables)
	varsStr := string(varsJSON)
	result := &TestResult{
		ID: testID, SkillID: id, Role: req.Role, InputVars: &varsStr,
		Response: &llmResp.Response, ResponseTime: &responseTime,
		TokenCount: &llmResp.EvalCount, TestedAt: testedAt,
	}
	_ = h.repoFor(ctx).Save(result)

	return TestResponse{
		TestID: testID, Role: req.Role, Response: llmResp.Response,
		ResponseTime: responseTime, TokenCount: llmResp.EvalCount, TestedAt: testedAt,
	}, nil
}

// History returns recent test results for a skill.
func (h *Handlers) History(ctx context.Context, id string, limit int) ([]TestResult, error) {
	if limit <= 0 {
		limit = 20
	}
	return h.repoFor(ctx).GetHistory(id, limit)
}

// Test handles POST /skills/{id}/test - tests a skill with Ollama.
func (h *Handlers) Test(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req TestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := h.RunTest(r.Context(), id, req)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrTestingUnavailable):
			status = http.StatusServiceUnavailable
		case errors.Is(err, ErrSkillNotFound):
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// GetHistory handles GET /skills/{id}/test-history - returns test history.
func (h *Handlers) GetHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	limit := 20
	if value := r.URL.Query().Get("limit"); value != "" {
		if parsed, parseErr := strconv.Atoi(value); parseErr == nil && parsed > 0 {
			limit = parsed
		}
	}
	results, err := h.History(r.Context(), id, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(results)
}
