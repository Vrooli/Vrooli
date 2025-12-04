package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"test-genie/internal/shared"
)

type agentModel struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Provider    string `json:"provider"`
	DisplayName string `json:"display_name"`
	Source      string `json:"source"`
	Description string `json:"description"`
}

type listAgentModelsResponse struct {
	Models []agentModel `json:"models"`
	Items  []agentModel `json:"items"`
	Data   []agentModel `json:"data"`
}

func (s *Server) handleListAgentModels(w http.ResponseWriter, r *http.Request) {
	provider := strings.TrimSpace(r.URL.Query().Get("provider"))
	if provider == "" {
		provider = "openrouter"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()

	models, err := listAgentModels(ctx, provider)
	if err != nil {
		s.log("list agent models failed", map[string]interface{}{"error": err.Error(), "provider": provider})
		var vErr shared.ValidationError
		if errors.As(err, &vErr) {
			s.writeError(w, http.StatusBadRequest, vErr.Error())
			return
		}
		s.writeError(w, http.StatusBadGateway, fmt.Sprintf("failed to load agent models: %s", err.Error()))
		return
	}

	payload := map[string]interface{}{
		"items": models,
		"count": len(models),
	}
	s.writeJSON(w, http.StatusOK, payload)
}

func listAgentModels(ctx context.Context, provider string) ([]agentModel, error) {
	path, err := exec.LookPath("resource-opencode")
	if err != nil {
		return nil, shared.NewValidationError("resource-opencode is not installed or not on PATH")
	}

	args := []string{"models", "--provider", provider, "--json"}
	cmd := exec.CommandContext(ctx, path, args...)
	output, err := cmd.CombinedOutput()
	cleaned := trimToJSON(output)

	// Attempt to parse as a plain array first.
	var plain []agentModel
	if err := json.Unmarshal(cleaned, &plain); err == nil && len(plain) > 0 {
		return normalizeAgentModels(plain), nil
	}

	var envelope listAgentModelsResponse
	if err := json.Unmarshal(cleaned, &envelope); err == nil {
		switch {
		case len(envelope.Models) > 0:
			return normalizeAgentModels(envelope.Models), nil
		case len(envelope.Items) > 0:
			return normalizeAgentModels(envelope.Items), nil
		case len(envelope.Data) > 0:
			return normalizeAgentModels(envelope.Data), nil
		}
	}

	// Last resort: attempt to parse slice of generic objects.
	var generic []map[string]interface{}
	if err := json.Unmarshal(cleaned, &generic); err == nil && len(generic) > 0 {
		mapped := make([]agentModel, 0, len(generic))
		for _, item := range generic {
			id, _ := item["id"].(string)
			name, _ := item["name"].(string)
			display, _ := item["display_name"].(string)
			prov, _ := item["provider"].(string)
			desc, _ := item["description"].(string)
			src, _ := item["source"].(string)
			mapped = append(mapped, agentModel{
				ID:          id,
				Name:        name,
				DisplayName: display,
				Provider:    prov,
				Description: desc,
				Source:      src,
			})
		}
		return normalizeAgentModels(mapped), nil
	}

	if err != nil {
		return nil, fmt.Errorf("resource-opencode models failed: %w: %s", err, strings.TrimSpace(string(output)))
	}

	return nil, fmt.Errorf("could not parse models response")
}

// trimToJSON removes leading non-JSON lines (warnings/logs) to allow parsing.
func trimToJSON(raw []byte) []byte {
	data := strings.TrimSpace(string(raw))
	if data == "" {
		return raw
	}

	// Find the first '{' or '[' which should start the JSON payload.
	idxObj := strings.IndexRune(data, '{')
	idxArr := strings.IndexRune(data, '[')

	start := -1
	if idxObj >= 0 && idxArr >= 0 {
		start = idxObj
		if idxArr < idxObj {
			start = idxArr
		}
	} else if idxObj >= 0 {
		start = idxObj
	} else if idxArr >= 0 {
		start = idxArr
	}

	if start > 0 {
		return []byte(data[start:])
	}
	return []byte(data)
}

func normalizeAgentModels(models []agentModel) []agentModel {
	seen := make(map[string]agentModel)
	for _, m := range models {
		id := strings.TrimSpace(m.ID)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = agentModel{
			ID:          id,
			Name:        strings.TrimSpace(m.Name),
			DisplayName: strings.TrimSpace(m.DisplayName),
			Provider:    strings.TrimSpace(m.Provider),
			Source:      strings.TrimSpace(m.Source),
			Description: strings.TrimSpace(m.Description),
		}
	}

	list := make([]agentModel, 0, len(seen))
	for _, m := range seen {
		list = append(list, m)
	}

	sort.Slice(list, func(i, j int) bool {
		if list[i].Provider == list[j].Provider {
			return list[i].ID < list[j].ID
		}
		return list[i].Provider < list[j].Provider
	})
	return list
}

type agentSpawnRequest struct {
	Prompts         []string `json:"prompts"`
	Model           string   `json:"model"`
	Concurrency     int      `json:"concurrency"`
	MaxTurns        int      `json:"maxTurns"`
	TimeoutSeconds  int      `json:"timeoutSeconds"`
	AllowedTools    []string `json:"allowedTools"`
	SkipPermissions bool     `json:"skipPermissions"`
}

type agentSpawnResult struct {
	PromptIndex int    `json:"promptIndex"`
	Status      string `json:"status"`
	SessionID   string `json:"sessionId,omitempty"`
	Output      string `json:"output,omitempty"`
	Error       string `json:"error,omitempty"`
}

type agentSpawnResponse struct {
	Items  []agentSpawnResult `json:"items"`
	Count  int                `json:"count"`
	Capped bool               `json:"capped"`
}

func (s *Server) handleSpawnAgents(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var payload agentSpawnRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	prompts := make([]string, 0, len(payload.Prompts))
	for _, p := range payload.Prompts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			prompts = append(prompts, trimmed)
		}
	}
	if len(prompts) == 0 {
		s.writeError(w, http.StatusBadRequest, "at least one prompt is required")
		return
	}
	const maxPrompts = 20
	capped := false
	if len(prompts) > maxPrompts {
		prompts = prompts[:maxPrompts]
		capped = true
	}

	model := strings.TrimSpace(payload.Model)
	if model == "" {
		s.writeError(w, http.StatusBadRequest, "model is required")
		return
	}

	concurrency := payload.Concurrency
	if concurrency <= 0 {
		concurrency = 3
	}
	if concurrency > 10 {
		concurrency = 10
	}
	if concurrency > len(prompts) {
		concurrency = len(prompts)
	}

	results := make([]agentSpawnResult, len(prompts))
	wg := sync.WaitGroup{}
	sem := make(chan struct{}, concurrency)

	for idx, prompt := range prompts {
		wg.Add(1)
		sem <- struct{}{}

		go func(i int, text string) {
			defer wg.Done()
			defer func() { <-sem }()

			res := runAgentPrompt(r.Context(), agentSpawnRequest{
				Model:           model,
				MaxTurns:        payload.MaxTurns,
				TimeoutSeconds:  payload.TimeoutSeconds,
				AllowedTools:    payload.AllowedTools,
				SkipPermissions: payload.SkipPermissions,
			}, text, i)
			results[i] = res
		}(idx, prompt)
	}

	wg.Wait()

	response := agentSpawnResponse{
		Items:  results,
		Count:  len(results),
		Capped: capped,
	}
	s.writeJSON(w, http.StatusOK, response)
}

func runAgentPrompt(ctx context.Context, payload agentSpawnRequest, prompt string, index int) agentSpawnResult {
	args := []string{"agents", "run", "--prompt", prompt, "--model", payload.Model, "--provider", "openrouter"}

	if len(payload.AllowedTools) > 0 {
		allowed := strings.Join(payload.AllowedTools, ",")
		args = append(args, "--allowed-tools", allowed)
	}
	if payload.SkipPermissions {
		args = append(args, "--skip-permissions")
	}
	if payload.MaxTurns > 0 {
		args = append(args, "--max-turns", fmt.Sprintf("%d", payload.MaxTurns))
	}
	if payload.TimeoutSeconds > 0 {
		args = append(args, "--timeout", fmt.Sprintf("%d", payload.TimeoutSeconds))
	}

	runCtx := ctx
	var cancel context.CancelFunc
	if payload.TimeoutSeconds > 0 {
		runCtx, cancel = context.WithTimeout(ctx, time.Duration(payload.TimeoutSeconds+5)*time.Second)
		defer cancel()
	}

	cmd := exec.CommandContext(runCtx, "resource-opencode", args...)
	output, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(output))

	res := agentSpawnResult{
		PromptIndex: index,
		Output:      text,
	}

	if err != nil {
		if runCtx.Err() == context.DeadlineExceeded {
			res.Status = "timeout"
			res.Error = "agent run exceeded timeout"
			return res
		}
		res.Status = "failed"
		if text != "" {
			res.Error = text
		} else {
			res.Error = err.Error()
		}
		return res
	}

	res.Status = "completed"
	res.SessionID = extractSessionID(text)
	return res
}

var sessionIDPattern = regexp.MustCompile(`Created OpenCode session:\s*([A-Za-z0-9\-\_]+)`)

func extractSessionID(output string) string {
	matches := sessionIDPattern.FindStringSubmatch(output)
	if len(matches) == 2 {
		return matches[1]
	}
	return ""
}
