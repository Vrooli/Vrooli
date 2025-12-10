package httpserver

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"

	"test-genie/internal/agents"
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

	// Drop any leading log/warning lines so JSON parsing is resilient to noisy CLIs.
	lines := strings.Split(data, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "[WARNING]") ||
			strings.HasPrefix(trimmed, "[INFO]") ||
			strings.HasPrefix(trimmed, "[ERROR]") ||
			strings.HasPrefix(trimmed, "[SUCCESS]") ||
			strings.HasPrefix(trimmed, "[HEADER]") ||
			strings.HasPrefix(trimmed, "[SECTION]") ||
			strings.HasPrefix(trimmed, "[PROMPT]") ||
			strings.HasPrefix(trimmed, "[DEBUG]") {
			continue
		}

		if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
			return []byte(strings.Join(lines[i:], "\n"))
		}

		if idx := strings.IndexAny(trimmed, "{["); idx >= 0 {
			payload := trimmed[idx:]
			if i+1 < len(lines) {
				payload = payload + "\n" + strings.Join(lines[i+1:], "\n")
			}
			return []byte(payload)
		}
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
	Preamble        string   `json:"preamble"`         // Immutable safety preamble (server validates)
	Model           string   `json:"model"`
	Concurrency     int      `json:"concurrency"`
	MaxTurns        int      `json:"maxTurns"`
	TimeoutSeconds  int      `json:"timeoutSeconds"`
	AllowedTools    []string `json:"allowedTools"`
	SkipPermissions bool     `json:"skipPermissions"`
	Scenario        string   `json:"scenario"`
	Scope           []string `json:"scope"`
	Phases          []string `json:"phases"`
	IdempotencyKey  string   `json:"idempotencyKey"`   // Client-provided key for deduplication
	MaxFilesChanged int      `json:"maxFilesChanged"`  // Max files an agent can modify (0 = default 50)
	MaxBytesWritten int64    `json:"maxBytesWritten"`  // Max total bytes written (0 = default 1MB)
}

const (
	// DefaultMaxFilesChanged is the default limit on files an agent can modify per run.
	DefaultMaxFilesChanged = 50
	// DefaultMaxBytesWritten is the default limit on total bytes an agent can write per run (1MB).
	DefaultMaxBytesWritten = 1024 * 1024
)

type agentSpawnResult struct {
	PromptIndex int    `json:"promptIndex"`
	AgentID     string `json:"agentId,omitempty"`
	Status      string `json:"status"`
	SessionID   string `json:"sessionId,omitempty"`
	Output      string `json:"output,omitempty"`
	Error       string `json:"error,omitempty"`
}

type agentSpawnResponse struct {
	Items           []agentSpawnResult `json:"items"`
	Count           int                `json:"count"`
	Capped          bool               `json:"capped"`
	ScopeConflicts  []string           `json:"scopeConflicts,omitempty"`
	ValidationError string             `json:"validationError,omitempty"`
	Idempotent      bool               `json:"idempotent,omitempty"` // True if this is a replayed response
}

func (s *Server) handleSpawnAgents(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var payload agentSpawnRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	// Validate prompts
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

	// Validate model
	model := strings.TrimSpace(payload.Model)
	if model == "" {
		s.writeError(w, http.StatusBadRequest, "model is required")
		return
	}
	if _, err := exec.LookPath("resource-opencode"); err != nil {
		s.writeError(w, http.StatusBadRequest, "resource-opencode is not installed or not on PATH")
		return
	}

	// Validate scenario
	scenario := strings.TrimSpace(payload.Scenario)
	if scenario == "" {
		s.writeError(w, http.StatusBadRequest, "scenario is required for agent spawning")
		return
	}

	// SECURITY: Block skipPermissions for spawned agents
	if payload.SkipPermissions {
		s.log("rejected skipPermissions for spawned agent", map[string]interface{}{"scenario": scenario})
		s.writeError(w, http.StatusBadRequest, "skipPermissions is not allowed for spawned agents; this would bypass all safety controls")
		return
	}

	// SECURITY: Validate and sanitize allowed tools
	allowedTools := payload.AllowedTools
	if len(allowedTools) == 0 {
		allowedTools = DefaultSafeTools()
	}
	sanitizedTools, err := SanitizeAllowedTools(allowedTools)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid allowed tools: %s", err.Error()))
		return
	}

	// SECURITY: Validate prompts don't contain obviously dangerous commands
	validator := NewDestructiveCommandValidator()
	for i, prompt := range prompts {
		if err := validator.ValidatePrompt(prompt); err != nil {
			s.writeError(w, http.StatusBadRequest, fmt.Sprintf("prompt %d contains blocked content: %s", i+1, err.Error()))
			return
		}
	}

	// SECURITY: Validate scope paths are within the scenario directory
	// This prevents path traversal attacks where an agent could access files outside its scope
	if len(payload.Scope) > 0 {
		if err := ValidateScopePaths(scenario, payload.Scope, ""); err != nil {
			s.log("rejected scope paths for spawned agent", map[string]interface{}{
				"scenario": scenario,
				"scope":    payload.Scope,
				"error":    err.Error(),
			})
			s.writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid scope paths: %s", err.Error()))
			return
		}
	}

	// SECURITY: Generate and validate the safety preamble server-side
	// This ensures clients cannot tamper with or omit security constraints
	repoRoot := strings.TrimSpace(os.Getenv("VROOLI_ROOT"))
	maxFiles := payload.MaxFilesChanged
	maxBytes := payload.MaxBytesWritten
	serverPreamble := generateSafetyPreamble(scenario, payload.Scope, repoRoot, maxFiles, maxBytes)

	// If client provided a preamble, validate it matches (modulo whitespace)
	if payload.Preamble != "" {
		clientPreamble := strings.TrimSpace(payload.Preamble)
		expectedPreamble := strings.TrimSpace(serverPreamble)
		if clientPreamble != expectedPreamble {
			s.log("preamble mismatch detected - using server-generated preamble", map[string]interface{}{
				"scenario": scenario,
			})
			// We don't reject, but we log the mismatch and always use server-generated version
		}
	}

	// Prepend the server-generated preamble to each prompt to ensure security constraints
	// This happens regardless of what the client sent - the server is authoritative
	for i, prompt := range prompts {
		// Check if prompt already contains the preamble (from honest client)
		if !strings.Contains(prompt, "## SECURITY CONSTRAINTS") {
			prompts[i] = serverPreamble + prompt
		} else {
			// Replace any client-provided preamble with server-generated one
			// Find where the task body starts (after the preamble separator)
			if idx := strings.Index(prompt, "---\n\n"); idx != -1 {
				body := prompt[idx+5:] // Skip "---\n\n"
				prompts[i] = serverPreamble + body
			}
		}
	}

	// IDEMPOTENCY: Check if this is a duplicate request
	// If idempotencyKey is provided, acquire a spawn intent lock to prevent race conditions
	idempotencyKey := strings.TrimSpace(payload.IdempotencyKey)
	if idempotencyKey != "" {
		const spawnIntentTTL = 30 * time.Minute // Intent locks expire after 30 minutes

		intent, isNew, err := s.agentService.AcquireSpawnIntent(r.Context(), idempotencyKey, scenario, payload.Scope, spawnIntentTTL)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to acquire spawn intent: %s", err.Error()))
			return
		}

		if !isNew {
			// This is a duplicate request - return cached result or current status
			s.log("duplicate spawn request detected", map[string]interface{}{
				"idempotencyKey": idempotencyKey,
				"scenario":       scenario,
				"intentStatus":   intent.Status,
			})

			switch intent.Status {
			case "completed":
				// Return the cached result
				if intent.ResultJSON != "" {
					var cachedResponse agentSpawnResponse
					if err := json.Unmarshal([]byte(intent.ResultJSON), &cachedResponse); err == nil {
						cachedResponse.Idempotent = true
						s.writeJSON(w, http.StatusOK, cachedResponse)
						return
					}
				}
				// Fallthrough if no cached result
				s.writeJSON(w, http.StatusOK, agentSpawnResponse{
					Items:      []agentSpawnResult{},
					Count:      0,
					Idempotent: true,
				})
				return

			case "pending":
				// Another request is still processing - return 409 to indicate in-progress
				s.writeJSON(w, http.StatusConflict, agentSpawnResponse{
					Items:           []agentSpawnResult{},
					Count:           0,
					ValidationError: fmt.Sprintf("spawn request with key %s is already in progress", idempotencyKey),
					Idempotent:      true,
				})
				return

			case "failed":
				// Previous attempt failed - allow retry by continuing with new spawn
				// (We'll update the intent status as we go)
			}
		}
	}

	// Check for scope conflicts with currently running agents
	conflictIDs, err := s.agentService.CheckConflictsSimple(r.Context(), scenario, payload.Scope)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to check conflicts: %s", err.Error()))
		return
	}
	if len(conflictIDs) > 0 {
		response := agentSpawnResponse{
			Items:          []agentSpawnResult{},
			Count:          0,
			Capped:         capped,
			ScopeConflicts: conflictIDs,
		}
		s.writeJSON(w, http.StatusConflict, response)
		return
	}

	// Normalize concurrency
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

	ctx := r.Context()

	for idx, prompt := range prompts {
		wg.Add(1)
		sem <- struct{}{}

		go func(i int, text string) {
			defer wg.Done()
			defer func() { <-sem }()

			// Create agent input
			promptHash := agents.HashPrompt(text)
			input := agents.CreateAgentInput{
				Scenario:    scenario,
				Scope:       payload.Scope,
				Phases:      payload.Phases,
				Model:       model,
				PromptHash:  promptHash,
				PromptIndex: i,
				PromptText:  text,
			}

			// Register agent (also acquires scope lock)
			agent, err := s.agentService.Register(ctx, input)
			if err != nil {
				results[i] = agentSpawnResult{
					PromptIndex: i,
					Status:      "failed",
					Error:       fmt.Sprintf("registration failed: %s", err.Error()),
				}
				return
			}

			// Broadcast agent registered (pending)
			s.wsManager.BroadcastAgentUpdate(agentToActiveAgent(agent))

			// Update status to running
			_ = s.agentService.UpdateStatus(ctx, agent.ID, agents.AgentStatusRunning, "", "", "")

			// Broadcast agent started running
			if updatedAgent, _ := s.agentService.Get(ctx, agent.ID); updatedAgent != nil {
				s.wsManager.BroadcastAgentUpdate(agentToActiveAgent(updatedAgent))
			}

			// Run the agent
			res := s.runAgentPromptWithService(ctx, agentSpawnRequest{
				Model:          model,
				MaxTurns:       payload.MaxTurns,
				TimeoutSeconds: payload.TimeoutSeconds,
				AllowedTools:   sanitizedTools,
				Scenario:       scenario,
			}, text, i, agent.ID)

			// Update service with result
			status := agents.AgentStatus(res.Status)
			_ = s.agentService.UpdateStatus(ctx, agent.ID, status, res.SessionID, res.Output, res.Error)

			// Broadcast agent completed/failed
			if updatedAgent, _ := s.agentService.Get(ctx, agent.ID); updatedAgent != nil {
				s.wsManager.BroadcastAgentUpdate(agentToActiveAgent(updatedAgent))
			}

			res.AgentID = agent.ID
			results[i] = res
		}(idx, prompt)
	}

	wg.Wait()

	response := agentSpawnResponse{
		Items:  results,
		Count:  len(results),
		Capped: capped,
	}

	// Update the spawn intent with the result (for idempotency)
	if idempotencyKey != "" {
		// Serialize successful results for caching
		resultJSON, _ := json.Marshal(response)
		agentIDs := make([]string, 0, len(results))
		for _, r := range results {
			if r.AgentID != "" {
				agentIDs = append(agentIDs, r.AgentID)
			}
		}
		firstAgentID := ""
		if len(agentIDs) > 0 {
			firstAgentID = agentIDs[0]
		}
		_ = s.agentService.UpdateSpawnIntent(r.Context(), idempotencyKey, firstAgentID, "completed", string(resultJSON))
	}

	s.writeJSON(w, http.StatusOK, response)
}

func (s *Server) runAgentPromptWithService(ctx context.Context, payload agentSpawnRequest, prompt string, index int, agentID string) agentSpawnResult {
	args := []string{"agents", "run", "--prompt", prompt, "--model", payload.Model, "--provider", "openrouter"}

	if len(payload.AllowedTools) > 0 {
		allowed := strings.Join(payload.AllowedTools, ",")
		args = append(args, "--allowed-tools", allowed)
	}
	// Note: skipPermissions is explicitly not passed - it's blocked at the handler level
	if payload.MaxTurns > 0 {
		args = append(args, "--max-turns", fmt.Sprintf("%d", payload.MaxTurns))
	}
	if payload.TimeoutSeconds > 0 {
		args = append(args, "--timeout", fmt.Sprintf("%d", payload.TimeoutSeconds))
	}

	// Determine the scenario directory for file access scoping
	var scenarioDir string
	if repoRoot := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); repoRoot != "" {
		if scenario := strings.TrimSpace(payload.Scenario); scenario != "" {
			candidateDir := filepath.Join(repoRoot, "scenarios", scenario)
			if info, err := os.Stat(candidateDir); err == nil && info.IsDir() {
				scenarioDir = candidateDir
			}
		}
		// Fallback to repo root if scenario dir doesn't exist
		if scenarioDir == "" {
			scenarioDir = repoRoot
		}
	}

	// SECURITY: Pass --directory to scope file access to the scenario root.
	// This restricts the agent's Read/Write/Edit tools to only access files within this directory.
	if scenarioDir != "" {
		args = append(args, "--directory", scenarioDir)
	}

	runCtx := ctx
	var cancel context.CancelFunc
	if payload.TimeoutSeconds > 0 {
		runCtx, cancel = context.WithTimeout(ctx, time.Duration(payload.TimeoutSeconds+5)*time.Second)
	} else {
		runCtx, cancel = context.WithCancel(ctx)
	}

	cmd := exec.CommandContext(runCtx, "resource-opencode", args...)
	// Set working directory to match the scoped directory
	if scenarioDir != "" {
		cmd.Dir = scenarioDir
	}

	// Inject environment variables for agent context
	// This allows agents to use these in their commands without hardcoding paths
	repoRoot := strings.TrimSpace(os.Getenv("VROOLI_ROOT"))
	cmd.Env = os.Environ() // Inherit existing environment
	if repoRoot != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("VROOLI_ROOT=%s", repoRoot))
	}
	if scenarioDir != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("SCENARIO_ROOT=%s", scenarioDir))
	}
	if payload.Scenario != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("SCENARIO_NAME=%s", payload.Scenario))
	}

	// Store cancel and cmd in agent service for stop functionality
	s.agentService.SetAgentProcess(agentID, cancel, cmd)

	// Start heartbeat goroutine to renew locks periodically
	heartbeatCtx, stopHeartbeat := context.WithCancel(runCtx)
	go s.runHeartbeatLoop(heartbeatCtx, agentID)

	// Set up output streaming instead of CombinedOutput
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		stopHeartbeat()
		return agentSpawnResult{
			PromptIndex: index,
			Status:      string(agents.AgentStatusFailed),
			Error:       fmt.Sprintf("failed to create stdout pipe: %v", err),
		}
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		stopHeartbeat()
		return agentSpawnResult{
			PromptIndex: index,
			Status:      string(agents.AgentStatusFailed),
			Error:       fmt.Sprintf("failed to create stderr pipe: %v", err),
		}
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		stopHeartbeat()
		return agentSpawnResult{
			PromptIndex: index,
			Status:      string(agents.AgentStatusFailed),
			Error:       fmt.Sprintf("failed to start command: %v", err),
		}
	}

	// Stream output in a goroutine
	var outputBuilder strings.Builder
	var sequence int64
	var outputMu sync.Mutex
	outputDone := make(chan struct{})

	// Helper to stream lines and broadcast via WebSocket
	streamOutput := func(reader io.Reader) {
		scanner := bufio.NewScanner(reader)
		// Increase buffer size for potentially long lines
		const maxScanTokenSize = 1024 * 1024 // 1MB
		buf := make([]byte, 64*1024)
		scanner.Buffer(buf, maxScanTokenSize)

		for scanner.Scan() {
			line := scanner.Text() + "\n"
			outputMu.Lock()
			outputBuilder.WriteString(line)
			outputMu.Unlock()

			// Broadcast output via WebSocket
			seq := atomic.AddInt64(&sequence, 1)
			if s.wsManager != nil {
				s.wsManager.BroadcastAgentOutput(agentID, line, seq)
			}
		}
	}

	// Stream both stdout and stderr
	var streamWg sync.WaitGroup
	streamWg.Add(2)
	go func() {
		defer streamWg.Done()
		streamOutput(stdoutPipe)
	}()
	go func() {
		defer streamWg.Done()
		streamOutput(stderrPipe)
	}()

	// Wait for streams to complete in a goroutine
	go func() {
		streamWg.Wait()
		close(outputDone)
	}()

	// Wait for command to complete
	cmdErr := cmd.Wait()

	// Wait for output streaming to finish
	<-outputDone

	// Stop the heartbeat goroutine
	stopHeartbeat()

	outputMu.Lock()
	text := strings.TrimSpace(outputBuilder.String())
	outputMu.Unlock()

	res := agentSpawnResult{
		PromptIndex: index,
		Output:      text,
	}

	if cmdErr != nil {
		if runCtx.Err() == context.DeadlineExceeded {
			res.Status = string(agents.AgentStatusTimeout)
			res.Error = "agent run exceeded timeout"
			return res
		}
		if runCtx.Err() == context.Canceled {
			res.Status = string(agents.AgentStatusStopped)
			res.Error = "agent was stopped"
			return res
		}
		res.Status = string(agents.AgentStatusFailed)
		if text != "" {
			res.Error = text
		} else {
			res.Error = fmt.Sprintf("resource-opencode failed: %v", cmdErr)
		}
		return res
	}

	res.Status = string(agents.AgentStatusCompleted)
	res.SessionID = extractSessionID(text)
	return res
}

var sessionIDPattern = regexp.MustCompile(`Created OpenCode session:\s*([A-Za-z0-9\-\_]+)`)

// runHeartbeatLoop periodically renews locks for an agent while it's running.
// It runs every HeartbeatInterval (5 minutes) to prevent lock expiration.
// If heartbeat fails 3 consecutive times, the agent is stopped to prevent orphaned locks.
func (s *Server) runHeartbeatLoop(ctx context.Context, agentID string) {
	ticker := time.NewTicker(agents.HeartbeatInterval)
	defer ticker.Stop()

	const maxConsecutiveFailures = 3
	consecutiveFailures := 0

	for {
		select {
		case <-ctx.Done():
			// Agent finished or was stopped
			return
		case <-ticker.C:
			// Renew locks
			if err := s.agentService.RenewLocks(ctx, agentID); err != nil {
				consecutiveFailures++
				s.log("heartbeat renewal failed", map[string]interface{}{
					"agentId":             agentID,
					"error":               err.Error(),
					"consecutiveFailures": consecutiveFailures,
					"maxFailures":         maxConsecutiveFailures,
				})

				// Stop the agent after too many consecutive failures
				// This prevents orphaned agents from holding locks forever
				if consecutiveFailures >= maxConsecutiveFailures {
					s.log("stopping agent due to heartbeat failures", map[string]interface{}{
						"agentId":             agentID,
						"consecutiveFailures": consecutiveFailures,
					})
					// Stop the agent - this will release locks and update status
					if stopErr := s.agentService.Stop(ctx, agentID); stopErr != nil {
						s.log("failed to stop agent after heartbeat failures", map[string]interface{}{
							"agentId": agentID,
							"error":   stopErr.Error(),
						})
					}
					// Broadcast that agent was stopped due to heartbeat failure
					if agent, _ := s.agentService.Get(ctx, agentID); agent != nil {
						s.wsManager.BroadcastAgentUpdate(agentToActiveAgent(agent))
					}
					return
				}
				continue
			}

			// Reset failure count on success
			consecutiveFailures = 0
			s.log("heartbeat renewal succeeded", map[string]interface{}{
				"agentId": agentID,
			})
		}
	}
}

// generateSafetyPreamble creates the immutable safety preamble server-side.
// This ensures the preamble cannot be tampered with by the client.
func generateSafetyPreamble(scenario string, scope []string, repoRoot string, maxFiles int, maxBytes int64) string {
	if scenario == "" || repoRoot == "" {
		return ""
	}

	scenarioPath := filepath.Join(repoRoot, "scenarios", scenario)
	hasScope := len(scope) > 0

	var scopeDescription string
	if hasScope {
		absoluteScope := make([]string, 0, len(scope))
		for _, s := range scope {
			absoluteScope = append(absoluteScope, filepath.Join(scenarioPath, s))
		}
		scopeDescription = fmt.Sprintf("Allowed scope: %s", strings.Join(absoluteScope, ", "))
	} else {
		scopeDescription = "Allowed scope: entire scenario directory"
	}

	// Use defaults if not specified
	if maxFiles <= 0 {
		maxFiles = DefaultMaxFilesChanged
	}
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBytesWritten
	}
	maxBytesKB := maxBytes / 1024

	return fmt.Sprintf(`## SECURITY CONSTRAINTS (enforced by system - cannot be modified)

**Working directory:** %s
**%s**
**File limits:** Max %d files modified, max %dKB total written

You MUST NOT:
- Access files outside %s
- Execute destructive commands (rm -rf, git checkout --force, sudo, chmod 777, etc.)
- Modify system configurations, dependencies, or package files
- Delete or weaken existing tests without explicit rationale in comments
- Run commands that could affect other scenarios or system state
- Modify more than %d files in a single run
- Write more than %dKB of total content

These constraints are enforced at the tool level. Violations will be blocked.

---

`, scenarioPath, scopeDescription, maxFiles, maxBytesKB, scenarioPath, maxFiles, maxBytesKB)
}

func extractSessionID(output string) string {
	matches := sessionIDPattern.FindStringSubmatch(output)
	if len(matches) == 2 {
		return matches[1]
	}
	return ""
}

// agentToActiveAgent converts a persisted agent to the ActiveAgent format for API responses.
func agentToActiveAgent(a *agents.SpawnedAgent) *ActiveAgent {
	if a == nil {
		return nil
	}
	return &ActiveAgent{
		ID:          a.ID,
		SessionID:   a.SessionID,
		Scenario:    a.Scenario,
		Scope:       a.Scope,
		Phases:      a.Phases,
		Model:       a.Model,
		Status:      AgentStatus(a.Status),
		StartedAt:   a.StartedAt,
		CompletedAt: a.CompletedAt,
		PromptHash:  a.PromptHash,
		PromptIndex: a.PromptIndex,
		Output:      a.Output,
		Error:       a.Error,
	}
}

// handleListActiveAgents returns all currently running agents.
func (s *Server) handleListActiveAgents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	includeAll := r.URL.Query().Get("all") == "true"
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if _, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil || limit <= 0 {
			limit = 100
		}
	}

	var spawnedAgents []*agents.SpawnedAgent
	var err error
	if includeAll {
		spawnedAgents, err = s.agentService.ListAll(ctx, limit)
	} else {
		spawnedAgents, err = s.agentService.ListActive(ctx)
	}
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list agents: %s", err.Error()))
		return
	}

	// Convert to ActiveAgent format for API response
	activeAgents := make([]*ActiveAgent, 0, len(spawnedAgents))
	for _, a := range spawnedAgents {
		activeAgents = append(activeAgents, agentToActiveAgent(a))
	}

	locks, err := s.agentService.GetActiveLocks(ctx)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get active locks: %s", err.Error()))
		return
	}

	// Convert locks to API format
	lockItems := make([]map[string]interface{}, 0, len(locks))
	for _, lock := range locks {
		lockItems = append(lockItems, map[string]interface{}{
			"agentId":    lock.AgentID,
			"scenario":   lock.Scenario,
			"path":       lock.Path,
			"acquiredAt": lock.AcquiredAt,
			"expiresAt":  lock.ExpiresAt,
		})
	}

	payload := map[string]interface{}{
		"items":       activeAgents,
		"count":       len(activeAgents),
		"activeLocks": lockItems,
	}
	s.writeJSON(w, http.StatusOK, payload)
}

// handleGetAgent returns details for a specific agent.
func (s *Server) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["id"]

	agent, err := s.agentService.Get(r.Context(), agentID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get agent: %s", err.Error()))
		return
	}
	if agent == nil {
		s.writeError(w, http.StatusNotFound, fmt.Sprintf("agent not found: %s", agentID))
		return
	}

	s.writeJSON(w, http.StatusOK, agentToActiveAgent(agent))
}

// handleStopAgent stops a running agent.
func (s *Server) handleStopAgent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	agentID := vars["id"]

	if err := s.agentService.Stop(ctx, agentID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.writeError(w, http.StatusNotFound, err.Error())
			return
		}
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.log("agent stopped", map[string]interface{}{"agentId": agentID})

	// Broadcast the stop event
	s.wsManager.BroadcastAgentStopped(agentID)

	agent, _ := s.agentService.Get(ctx, agentID)
	if agent != nil {
		s.wsManager.BroadcastAgentUpdate(agentToActiveAgent(agent))
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "agent stopped",
		"agent":   agentToActiveAgent(agent),
	})
}

// handleStopAllAgents stops all running agents.
func (s *Server) handleStopAllAgents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	stoppedIDs, err := s.agentService.StopAll(ctx)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to stop agents: %s", err.Error()))
		return
	}

	s.log("all agents stopped", map[string]interface{}{"stoppedCount": len(stoppedIDs), "stoppedIds": stoppedIDs})

	// Broadcast the stop all event
	s.wsManager.BroadcastAgentsStoppedAll(len(stoppedIDs))

	// Also broadcast individual agent updates
	for _, agentID := range stoppedIDs {
		if agent, _ := s.agentService.Get(ctx, agentID); agent != nil {
			s.wsManager.BroadcastAgentUpdate(agentToActiveAgent(agent))
		}
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "all agents stopped",
		"stoppedCount": len(stoppedIDs),
		"stoppedIds":   stoppedIDs,
	})
}

// handleCheckScopeConflicts checks if spawning agents with the given scope would conflict.
func (s *Server) handleCheckScopeConflicts(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var payload struct {
		Scenario string   `json:"scenario"`
		Scope    []string `json:"scope"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	scenario := strings.TrimSpace(payload.Scenario)
	if scenario == "" {
		s.writeError(w, http.StatusBadRequest, "scenario is required")
		return
	}

	conflicts, err := s.agentService.CheckConflicts(r.Context(), scenario, payload.Scope)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to check conflicts: %s", err.Error()))
		return
	}

	// Convert to API format with detailed conflict info
	// Also fetch agent details to include model and phases
	conflictDetails := make([]map[string]interface{}, 0, len(conflicts))
	for _, c := range conflicts {
		lockedByInfo := map[string]interface{}{
			"path":      c.LockedBy.Path,
			"agentId":   c.LockedBy.AgentID,
			"scenario":  c.LockedBy.Scenario,
			"startedAt": c.LockedBy.StartedAt,
			"expiresAt": c.LockedBy.ExpiresAt,
		}

		// Fetch additional agent details for enhanced UI
		if agent, err := s.agentService.Get(r.Context(), c.LockedBy.AgentID); err == nil && agent != nil {
			lockedByInfo["model"] = agent.Model
			lockedByInfo["phases"] = agent.Phases
			lockedByInfo["status"] = string(agent.Status)
			// Calculate running duration for estimated completion hints
			if !agent.StartedAt.IsZero() {
				runningSeconds := time.Since(agent.StartedAt).Seconds()
				lockedByInfo["runningSeconds"] = int(runningSeconds)
			}
		}

		conflictDetails = append(conflictDetails, map[string]interface{}{
			"path":     c.Path,
			"lockedBy": lockedByInfo,
		})
	}

	response := map[string]interface{}{
		"hasConflicts": len(conflicts) > 0,
		"conflicts":    conflictDetails,
	}
	s.writeJSON(w, http.StatusOK, response)
}

// handleCleanupAgents removes old completed agents from the registry.
func (s *Server) handleCleanupAgents(w http.ResponseWriter, r *http.Request) {
	olderThanStr := r.URL.Query().Get("olderThanMinutes")
	olderThanMinutes := 60 // default: 1 hour
	if olderThanStr != "" {
		if _, err := fmt.Sscanf(olderThanStr, "%d", &olderThanMinutes); err != nil || olderThanMinutes <= 0 {
			olderThanMinutes = 60
		}
	}

	removed, err := s.agentService.CleanupCompleted(r.Context(), time.Duration(olderThanMinutes)*time.Minute)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to cleanup agents: %s", err.Error()))
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"removed":          removed,
		"olderThanMinutes": olderThanMinutes,
	})
}

// handleAgentHeartbeat extends the lock timeout for a running agent.
// This should be called periodically by agents to prevent lock expiration.
func (s *Server) handleAgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	agentID := vars["id"]

	if agentID == "" {
		s.writeError(w, http.StatusBadRequest, "agent ID is required")
		return
	}

	// Verify agent exists and is still running
	agent, err := s.agentService.Get(ctx, agentID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get agent: %s", err.Error()))
		return
	}
	if agent == nil {
		s.writeError(w, http.StatusNotFound, fmt.Sprintf("agent not found: %s", agentID))
		return
	}
	if agent.Status != agents.AgentStatusRunning && agent.Status != agents.AgentStatusPending {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("agent %s is not running (status: %s)", agentID, agent.Status))
		return
	}

	// Renew the locks
	if err := s.agentService.RenewLocks(ctx, agentID); err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to renew locks: %s", err.Error()))
		return
	}

	s.log("agent heartbeat", map[string]interface{}{"agentId": agentID})

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":     "heartbeat received",
		"agentId":     agentID,
		"lockTimeout": s.agentService.GetLockTimeout().String(),
	})
}

// handleGetBlockedCommands returns command validation info (allowlist and blocklist).
func (s *Server) handleGetBlockedCommands(w http.ResponseWriter, r *http.Request) {
	validator := NewBashCommandValidator()

	// Get allowed commands (primary - allowlist)
	allowedCommands := make([]map[string]string, 0, len(validator.allowedCommands))
	for _, ac := range validator.GetAllowedCommands() {
		allowedCommands = append(allowedCommands, map[string]string{
			"prefix":      ac.Prefix,
			"description": ac.Description,
		})
	}

	// Get blocked patterns (secondary - defense in depth for prompt scanning)
	blockedPatterns := make([]map[string]string, 0, len(validator.blockedPatterns))
	for _, bp := range validator.blockedPatterns {
		blockedPatterns = append(blockedPatterns, map[string]string{
			"pattern":     bp.Pattern.String(),
			"description": bp.Description,
		})
	}

	response := map[string]interface{}{
		"allowedBashCommands": allowedCommands,
		"blockedPatterns":     blockedPatterns,
		"safeDefaults":        DefaultSafeTools(),
		"safeBashPatterns":    SafeBashPatterns(),
		"securityModel":       "allowlist",
		"note":                "Bash commands must start with an allowed prefix. The blocklist is used for defense-in-depth prompt scanning only.",
	}
	s.writeJSON(w, http.StatusOK, response)
}

// handleValidatePromptPaths validates that paths referenced in the prompt exist.
// This is a pre-flight check before spawning agents to catch configuration errors early.
func (s *Server) handleValidatePromptPaths(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var payload struct {
		Scenario string   `json:"scenario"`
		Phases   []string `json:"phases"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	scenario := strings.TrimSpace(payload.Scenario)
	if scenario == "" {
		s.writeError(w, http.StatusBadRequest, "scenario is required")
		return
	}

	repoRoot := strings.TrimSpace(os.Getenv("VROOLI_ROOT"))
	if repoRoot == "" {
		s.writeError(w, http.StatusInternalServerError, "VROOLI_ROOT not configured")
		return
	}

	scenarioRoot := filepath.Join(repoRoot, "scenarios", scenario)
	testGenieRoot := filepath.Join(repoRoot, "scenarios", "test-genie")

	// Build list of paths that the prompt will reference
	pathsToCheck := []struct {
		Path        string `json:"path"`
		Description string `json:"description"`
		Required    bool   `json:"required"`
	}{
		{scenarioRoot, "Scenario root directory", true},
		{filepath.Join(scenarioRoot, "PRD.md"), "Product Requirements Document", false},
		{filepath.Join(scenarioRoot, "requirements"), "Requirements directory", false},
		{filepath.Join(scenarioRoot, "docs"), "Scenario docs directory", false},
		{filepath.Join(testGenieRoot, "docs", "guides", "test-generation.md"), "General test generation guide", false},
	}

	// Add phase-specific docs
	phaseDocPaths := map[string]string{
		"unit":        "/docs/phases/unit.md",
		"integration": "/docs/phases/integration.md",
		"e2e":         "/docs/phases/e2e.md",
		"playbook":    "/docs/phases/playbook.md",
		"business":    "/docs/phases/business.md",
	}
	for _, phase := range payload.Phases {
		if docPath, ok := phaseDocPaths[phase]; ok {
			pathsToCheck = append(pathsToCheck, struct {
				Path        string `json:"path"`
				Description string `json:"description"`
				Required    bool   `json:"required"`
			}{
				Path:        filepath.Join(testGenieRoot, docPath),
				Description: fmt.Sprintf("Phase docs for %s", phase),
				Required:    false,
			})
		}
	}

	// Check which paths exist
	results := make([]map[string]interface{}, 0, len(pathsToCheck))
	warnings := make([]string, 0)
	errors := make([]string, 0)

	for _, pathInfo := range pathsToCheck {
		exists := false
		isDir := false

		if info, err := os.Stat(pathInfo.Path); err == nil {
			exists = true
			isDir = info.IsDir()
		}

		result := map[string]interface{}{
			"path":        pathInfo.Path,
			"description": pathInfo.Description,
			"exists":      exists,
			"isDirectory": isDir,
			"required":    pathInfo.Required,
		}
		results = append(results, result)

		if !exists {
			if pathInfo.Required {
				errors = append(errors, fmt.Sprintf("%s not found: %s", pathInfo.Description, pathInfo.Path))
			} else {
				warnings = append(warnings, fmt.Sprintf("%s not found: %s (agents may have reduced context)", pathInfo.Description, pathInfo.Path))
			}
		}
	}

	response := map[string]interface{}{
		"valid":    len(errors) == 0,
		"paths":    results,
		"warnings": warnings,
		"errors":   errors,
	}
	s.writeJSON(w, http.StatusOK, response)
}
