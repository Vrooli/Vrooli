package httpserver

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	"test-genie/internal/containment"
	"test-genie/internal/security"
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
	Preamble        string   `json:"preamble"` // Immutable safety preamble (server validates)
	Model           string   `json:"model"`
	Concurrency     int      `json:"concurrency"`
	MaxTurns        int      `json:"maxTurns"`
	TimeoutSeconds  int      `json:"timeoutSeconds"`
	AllowedTools    []string `json:"allowedTools"`
	SkipPermissions bool     `json:"skipPermissions"`
	Scenario        string   `json:"scenario"`
	Scope           []string `json:"scope"`
	Phases          []string `json:"phases"`
	IdempotencyKey  string   `json:"idempotencyKey"`  // Client-provided key for deduplication
	MaxFilesChanged int      `json:"maxFilesChanged"` // Max files an agent can modify (0 = default 50)
	MaxBytesWritten int64    `json:"maxBytesWritten"` // Max total bytes written (0 = default 1MB)
	NetworkEnabled  bool     `json:"networkEnabled"`  // Enable network access for agents (default: false)
}

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

// handleSpawnAgents spawns one or more agents to execute prompts.
//
// PROCESSING FLOW (in order):
// 1. Parse & Field Validation - Decode JSON, validate required fields (prompts, model, scenario)
// 2. Security Validation - Check tools allowlist, prompt content, scope paths (via SpawnSecurityValidator)
// 3. Preamble Injection - Generate and inject server-authoritative security preamble into prompts
// 4. Idempotency Check - Handle duplicate requests via idempotency key (optional)
// 5. Conflict Detection - Check for scope conflicts with running agents
// 6. Spawn Session Tracking - Prevent duplicate spawns across browser tabs
// 7. Agent Execution - Spawn agents concurrently (up to config.MaxConcurrentAgents)
// 8. Result Aggregation - Collect results, update spawn intent, return response
//
// CHANGE_AXIS: Spawn Request Handling
// This handler orchestrates spawn request processing but delegates security
// validation to SpawnSecurityValidator. To modify security policy, see:
//   - internal/agents/spawn_security_validator.go (consolidated security checks)
//   - internal/agents/spawn_validator.go (field validation)
func (s *Server) handleSpawnAgents(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var payload agentSpawnRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	// --- Field Validation (non-security) ---
	// Sanitize and validate prompts
	prompts := agents.SanitizePrompts(payload.Prompts)
	if len(prompts) == 0 {
		s.writeError(w, http.StatusBadRequest, "at least one prompt is required")
		return
	}

	// Use config for max prompts limit
	agentCfg := s.agentService.GetConfig()
	maxPrompts := agentCfg.MaxPromptsPerSpawn
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

	// --- Security Validation (consolidated in SpawnSecurityValidator) ---
	// CHANGE_AXIS: All security policy decisions are in spawn_security_validator.go
	repoRoot := strings.TrimSpace(os.Getenv("VROOLI_ROOT"))

	securityInput := agents.SpawnSecurityInput{
		Prompts:         prompts,
		AllowedTools:    payload.AllowedTools,
		SkipPermissions: payload.SkipPermissions,
		Scenario:        scenario,
		ScopePaths:      payload.Scope,
		RepoRoot:        repoRoot,
	}

	securityResult := agents.ValidateSpawnSecurity(securityInput)
	if !securityResult.Valid {
		// Log security validation failures for audit trail
		s.log("spawn security validation failed", map[string]interface{}{
			"scenario":     scenario,
			"error":        securityResult.FirstError,
			"failedChecks": len(securityResult.GetFailedChecks()),
		})
		s.writeError(w, http.StatusBadRequest, securityResult.FirstError)
		return
	}

	// Use sanitized tools from security validation, or defaults
	sanitizedTools := securityResult.SanitizedTools
	if len(sanitizedTools) == 0 {
		sanitizedTools = agents.GetDefaultSafeTools()
	}

	// --- Preamble Generation (server-authoritative) ---
	// CHANGE_COUPLE: Preamble uses BashCommandValidator.GetAllowedCommands()
	// See internal/security/preamble.go for preamble generation logic
	// This ensures clients cannot tamper with or omit security constraints

	// Apply execution defaults from config when not specified in request
	maxFilesChanged := payload.MaxFilesChanged
	if maxFilesChanged <= 0 {
		maxFilesChanged = agentCfg.DefaultMaxFilesChanged
	}
	maxBytesWritten := payload.MaxBytesWritten
	if maxBytesWritten <= 0 {
		maxBytesWritten = agentCfg.DefaultMaxBytesWritten
	}
	// NetworkEnabled is a toggle, so we use the request value or fall back to config default
	networkEnabled := payload.NetworkEnabled
	if !payload.NetworkEnabled && agentCfg.DefaultNetworkEnabled {
		networkEnabled = true
	}

	serverPreamble := security.GenerateSafetyPreamble(security.PreambleConfig{
		Scenario:       scenario,
		Scope:          payload.Scope,
		RepoRoot:       repoRoot,
		MaxFiles:       maxFilesChanged,
		MaxBytes:       maxBytesWritten,
		NetworkEnabled: networkEnabled,
	})

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
		// Use config for idempotency TTL
		spawnIntentTTL := agentCfg.IdempotencyTTL()

		intent, isNew, err := s.agentService.AcquireSpawnIntent(r.Context(), idempotencyKey, scenario, payload.Scope, spawnIntentTTL)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to acquire spawn intent: %s", err.Error()))
			return
		}

		// Use the centralized idempotency decision function
		decision := agents.ClassifyIdempotencyAction(intent, isNew)

		s.log("idempotency decision", map[string]interface{}{
			"idempotencyKey": idempotencyKey,
			"scenario":       scenario,
			"action":         string(decision.Action),
			"reason":         decision.Reason,
		})

		switch decision.Action {
		case agents.IdempotencyActionReturnCached:
			// Return the cached result
			if decision.CachedResult != "" {
				var cachedResponse agentSpawnResponse
				if err := json.Unmarshal([]byte(decision.CachedResult), &cachedResponse); err == nil {
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

		case agents.IdempotencyActionConflict:
			// Another request is still processing - return 409 to indicate in-progress
			s.writeJSON(w, http.StatusConflict, agentSpawnResponse{
				Items:           []agentSpawnResult{},
				Count:           0,
				ValidationError: fmt.Sprintf("spawn request with key %s is already in progress", idempotencyKey),
				Idempotent:      true,
			})
			return

		case agents.IdempotencyActionRetry, agents.IdempotencyActionProceed:
			// Continue with spawn - either new request or retry after failure
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

	// SECURITY: Check for server-side spawn session conflicts
	// This prevents duplicate spawns across browser tabs/windows
	userIdentifier := getUserIdentifier(r)
	sessionConflicts, err := s.agentService.CheckSpawnSessionConflicts(r.Context(), userIdentifier, scenario, payload.Scope)
	if err != nil {
		s.log("failed to check spawn session conflicts", map[string]interface{}{
			"error":          err.Error(),
			"userIdentifier": userIdentifier,
			"scenario":       scenario,
		})
		// Non-fatal: continue with spawn even if session tracking fails
	} else if len(sessionConflicts) > 0 {
		// Return conflict info for UI to display
		s.log("spawn session conflict detected", map[string]interface{}{
			"userIdentifier": userIdentifier,
			"scenario":       scenario,
			"conflictCount":  len(sessionConflicts),
		})
		response := agentSpawnResponse{
			Items:           []agentSpawnResult{},
			Count:           0,
			Capped:          capped,
			ValidationError: fmt.Sprintf("You already have %d active spawn session(s) for this scenario. Clear them to spawn new agents.", len(sessionConflicts)),
		}
		s.writeJSON(w, http.StatusConflict, response)
		return
	}

	// Normalize concurrency using config limits
	concurrency := payload.Concurrency
	if concurrency <= 0 {
		concurrency = agentCfg.DefaultConcurrency
	}
	if concurrency > agentCfg.MaxConcurrentAgents {
		concurrency = agentCfg.MaxConcurrentAgents
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
				// Use structured failure classification for better error messages
				failure := agents.ClassifyError(err)
				decision := agents.DecideOnRegistrationFailure(failure)
				results[i] = agentSpawnResult{
					PromptIndex: i,
					Status:      "failed",
					Error:       decision.WarningForUser,
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

			// Apply execution defaults for timeout and maxTurns
			timeoutSeconds := payload.TimeoutSeconds
			if timeoutSeconds <= 0 {
				timeoutSeconds = agentCfg.DefaultTimeoutSeconds
			}
			maxTurns := payload.MaxTurns
			if maxTurns <= 0 {
				maxTurns = agentCfg.DefaultMaxTurns
			}

			// Run the agent
			res := s.runAgentPromptWithService(ctx, agentSpawnRequest{
				Model:          model,
				MaxTurns:       maxTurns,
				TimeoutSeconds: timeoutSeconds,
				AllowedTools:   sanitizedTools,
				Scenario:       scenario,
				NetworkEnabled: networkEnabled,
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

	// Collect agent IDs for session tracking
	agentIDs := make([]string, 0, len(results))
	for _, res := range results {
		if res.AgentID != "" {
			agentIDs = append(agentIDs, res.AgentID)
		}
	}

	// Create server-side spawn session for tracking
	// This enables cross-tab conflict detection
	if len(agentIDs) > 0 {
		_, err := s.agentService.CreateSpawnSession(r.Context(), agents.CreateSpawnSessionInput{
			UserIdentifier: userIdentifier,
			Scenario:       scenario,
			Scope:          payload.Scope,
			Phases:         payload.Phases,
			AgentIDs:       agentIDs,
			TTL:            agentCfg.SpawnSessionTTL(), // Use config for session TTL
		})
		if err != nil {
			s.log("failed to create spawn session", map[string]interface{}{
				"error":          err.Error(),
				"userIdentifier": userIdentifier,
				"scenario":       scenario,
				"agentCount":     len(agentIDs),
			})
			// Non-fatal: spawn succeeded, just tracking failed
		} else {
			s.log("spawn session created", map[string]interface{}{
				"userIdentifier": userIdentifier,
				"scenario":       scenario,
				"agentCount":     len(agentIDs),
			})
		}
	}

	response := agentSpawnResponse{
		Items:  results,
		Count:  len(results),
		Capped: capped,
	}

	// Update the spawn intent with the result (for idempotency)
	if idempotencyKey != "" {
		// Serialize successful results for caching
		resultJSON, _ := json.Marshal(response)
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
	repoRoot := strings.TrimSpace(os.Getenv("VROOLI_ROOT"))
	if repoRoot != "" {
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

	// SECURITY: Use containment system when available
	// This provides OS-level isolation (Docker, bubblewrap) in addition to tool-level restrictions
	// The containment selector is injected via the server for testability
	provider := s.containmentSelector.SelectProvider(runCtx)

	// Build environment variables for agent context
	envVars := map[string]string{}
	if repoRoot != "" {
		envVars["VROOLI_ROOT"] = repoRoot
	}
	if scenarioDir != "" {
		envVars["SCENARIO_ROOT"] = scenarioDir
	}
	if payload.Scenario != "" {
		envVars["SCENARIO_NAME"] = payload.Scenario
	}
	// Copy existing env vars that might be needed
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			// Only copy specific vars to avoid leaking sensitive info
			switch key {
			case "PATH", "HOME", "USER", "SHELL", "TERM", "LANG", "LC_ALL",
				"OPENROUTER_API_KEY", "ANTHROPIC_API_KEY", "OPENAI_API_KEY":
				if _, exists := envVars[key]; !exists {
					envVars[key] = parts[1]
				}
			}
		}
	}

	// Build the full command with resource-opencode
	fullCommand := append([]string{"resource-opencode"}, args...)

	// Get containment config for resource limits
	// Use defaults from containment.DefaultConfig() which can be overridden via env vars
	containmentCfg := containment.LoadConfigFromEnv()

	// Configure containment execution
	execConfig := containment.ExecutionConfig{
		WorkingDir: scenarioDir,
		// SECURITY: Only allow access to the scenario directory and common read-only paths
		AllowedPaths: []string{}, // Working dir is already allowed
		ReadOnlyPaths: []string{
			"/usr/bin", "/usr/local/bin", // For test tools
			"/etc/ssl", // For HTTPS
		},
		Environment: envVars,
		// SECURITY: Resource limits from config to prevent runaway agents
		MaxMemoryMB:   containmentCfg.MaxMemoryMB,
		MaxCPUPercent: containmentCfg.MaxCPUPercent,
		// Network access controlled by user toggle (default: false)
		// When disabled, agents cannot make outbound network requests
		NetworkAccess:  payload.NetworkEnabled,
		Command:        fullCommand,
		TimeoutSeconds: payload.TimeoutSeconds + 5,
	}

	var cmd *exec.Cmd
	var containmentUsed string

	// Use containment if available (Docker preferred)
	if provider.Type() != containment.ContainmentTypeNone {
		preparedCmd, err := provider.PrepareCommand(runCtx, execConfig)
		if err != nil {
			s.log("containment preparation failed, falling back", map[string]interface{}{
				"error":    err.Error(),
				"provider": string(provider.Type()),
				"agentId":  agentID,
			})
			// Fall through to direct execution
		} else {
			cmd = preparedCmd
			containmentUsed = string(provider.Type())
			s.log("using containment for agent", map[string]interface{}{
				"provider":      containmentUsed,
				"agentId":       agentID,
				"networkAccess": execConfig.NetworkAccess,
				"memoryLimit":   execConfig.MaxMemoryMB,
			})
		}
	}

	// Fallback to direct execution if containment not available or failed
	if cmd == nil {
		cmd = exec.CommandContext(runCtx, "resource-opencode", args...)
		containmentUsed = "none"
		// Set working directory to match the scoped directory
		if scenarioDir != "" {
			cmd.Dir = scenarioDir
		}
		// Set environment variables directly
		cmd.Env = os.Environ() // Inherit existing environment
		for k, v := range envVars {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
		s.log("running agent without containment", map[string]interface{}{
			"agentId": agentID,
			"warning": "No OS-level isolation - consider installing Docker",
		})
	}

	// Log containment status for audit trail
	s.log("agent execution config", map[string]interface{}{
		"agentId":       agentID,
		"containment":   containmentUsed,
		"workingDir":    scenarioDir,
		"networkAccess": execConfig.NetworkAccess,
	})

	// Store cancel and cmd in agent service for stop functionality
	s.agentService.SetAgentProcess(agentID, cancel, cmd)

	// Start heartbeat goroutine to renew locks periodically
	heartbeatCtx, stopHeartbeat := context.WithCancel(runCtx)
	go s.runHeartbeatLoop(heartbeatCtx, agentID)

	// Set up output streaming instead of CombinedOutput
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		stopHeartbeat()
		failure := agents.NewProcessPipeFailure("stdout", err)
		decision := agents.DecideOnProcessFailure(failure)
		return agentSpawnResult{
			PromptIndex: index,
			Status:      string(agents.AgentStatusFailed),
			Error:       decision.WarningForUser,
		}
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		stopHeartbeat()
		failure := agents.NewProcessPipeFailure("stderr", err)
		decision := agents.DecideOnProcessFailure(failure)
		return agentSpawnResult{
			PromptIndex: index,
			Status:      string(agents.AgentStatusFailed),
			Error:       decision.WarningForUser,
		}
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		stopHeartbeat()
		failure := agents.NewProcessStartFailure(err)
		decision := agents.DecideOnProcessFailure(failure)
		return agentSpawnResult{
			PromptIndex: index,
			Status:      string(agents.AgentStatusFailed),
			Error:       decision.WarningForUser,
		}
	}

	// Record PID in database for orphan detection on restart
	if cmd.Process != nil {
		if pidErr := s.agentService.SetAgentPID(ctx, agentID, cmd.Process.Pid); pidErr != nil {
			// Log but don't fail - PID tracking is nice-to-have, not critical
			s.logger.Printf("warning: failed to record PID %d for agent %s: %v", cmd.Process.Pid, agentID, pidErr)
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
			failure := agents.NewProcessTimeoutFailure(payload.TimeoutSeconds)
			decision := agents.DecideOnProcessFailure(failure)
			res.Status = string(agents.AgentStatusTimeout)
			res.Error = decision.WarningForUser
			return res
		}
		if runCtx.Err() == context.Canceled {
			failure := agents.NewProcessCanceledFailure()
			decision := agents.DecideOnProcessFailure(failure)
			res.Status = string(agents.AgentStatusStopped)
			res.Error = decision.WarningForUser
			return res
		}
		// Process exited with error - use the output if available for context
		failure := agents.NewProcessExitFailure(cmdErr, text)
		decision := agents.DecideOnProcessFailure(failure)
		res.Status = string(agents.AgentStatusFailed)
		// For exit errors, prefer showing the actual output if available (contains error details)
		// Otherwise use the structured decision warning
		if text != "" {
			res.Error = failure.Message // Use the truncated message from the failure
		} else {
			res.Error = decision.WarningForUser
		}
		return res
	}

	res.Status = string(agents.AgentStatusCompleted)
	res.SessionID = extractSessionID(text)
	return res
}

var sessionIDPattern = regexp.MustCompile(`Created OpenCode session:\s*([A-Za-z0-9\-\_]+)`)

// runHeartbeatLoop periodically renews locks for an agent while it's running.
// It runs every HeartbeatInterval to prevent lock expiration.
// If heartbeat fails consecutively (per config), the agent is stopped to prevent orphaned locks.
func (s *Server) runHeartbeatLoop(ctx context.Context, agentID string) {
	agentCfg := s.agentService.GetConfig()
	ticker := time.NewTicker(agentCfg.HeartbeatInterval())
	defer ticker.Stop()

	maxConsecutiveFailures := agentCfg.MaxHeartbeatFailures
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
				failure := agents.ClassifyError(err)

				// Use decision function for consistent handling
				decision := agents.DecideOnHeartbeatFailure(consecutiveFailures, maxConsecutiveFailures, failure)

				s.log("heartbeat renewal failed", map[string]interface{}{
					"agentId":             agentID,
					"error":               err.Error(),
					"consecutiveFailures": consecutiveFailures,
					"maxFailures":         maxConsecutiveFailures,
					"shouldContinue":      decision.ShouldContinue,
					"reason":              decision.Reason,
				})

				// Stop the agent if decision says we should not continue
				if !decision.ShouldContinue {
					s.log("stopping agent due to heartbeat failures", map[string]interface{}{
						"agentId":             agentID,
						"consecutiveFailures": consecutiveFailures,
						"reason":              decision.Reason,
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
		PromptText:  a.PromptText,
		Output:      a.Output,
		Error:       a.Error,
		PID:         a.PID,
		Hostname:    a.Hostname,
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
	// CHANGE_AXIS: Use status classification helpers instead of hard-coded status checks.
	// This ensures handler stays in sync with model.go's definition of active vs terminal.
	// See agents.ClassifyAgentStatus() for the single source of truth.
	if !agent.Status.IsActive() {
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

// handleGetAgentConfig returns the current agent and containment control surface.
// This provides visibility into all tunable levers for operators and developers.
func (s *Server) handleGetAgentConfig(w http.ResponseWriter, r *http.Request) {
	agentCfg := s.agentService.GetConfig()
	containmentCfg := containment.LoadConfigFromEnv()

	response := map[string]interface{}{
		// Agent config grouped by concern
		"agents": map[string]interface{}{
			"lockingAndCoordination": map[string]interface{}{
				"lockTimeoutMinutes":       agentCfg.LockTimeoutMinutes,
				"heartbeatIntervalMinutes": agentCfg.HeartbeatIntervalMinutes,
				"maxHeartbeatFailures":     agentCfg.MaxHeartbeatFailures,
			},
			"executionDefaults": map[string]interface{}{
				"defaultTimeoutSeconds":  agentCfg.DefaultTimeoutSeconds,
				"defaultMaxTurns":        agentCfg.DefaultMaxTurns,
				"defaultMaxFilesChanged": agentCfg.DefaultMaxFilesChanged,
				"defaultMaxBytesWritten": agentCfg.DefaultMaxBytesWritten,
				"defaultNetworkEnabled":  agentCfg.DefaultNetworkEnabled,
			},
			"spawnLimits": map[string]interface{}{
				"maxPromptsPerSpawn":  agentCfg.MaxPromptsPerSpawn,
				"maxConcurrentAgents": agentCfg.MaxConcurrentAgents,
				"defaultConcurrency":  agentCfg.DefaultConcurrency,
			},
			"retentionAndCleanup": map[string]interface{}{
				"retentionDays":          agentCfg.RetentionDays,
				"cleanupIntervalMinutes": agentCfg.CleanupIntervalMinutes,
			},
			"sessionManagement": map[string]interface{}{
				"spawnSessionTTLMinutes": agentCfg.SpawnSessionTTLMinutes,
				"idempotencyTTLMinutes":  agentCfg.IdempotencyTTLMinutes,
			},
		},
		// Containment config
		"containment": map[string]interface{}{
			"containerImage": containmentCfg.DockerImage,
			"resourceLimits": map[string]interface{}{
				"maxMemoryMB":   containmentCfg.MaxMemoryMB,
				"maxCPUPercent": containmentCfg.MaxCPUPercent,
			},
			"availability": map[string]interface{}{
				"timeoutSeconds": containmentCfg.AvailabilityTimeoutSeconds,
				"preferDocker":   containmentCfg.PreferDocker,
				"allowFallback":  containmentCfg.AllowFallback,
			},
			"securityHardening": map[string]interface{}{
				"dropAllCapabilities": containmentCfg.DropAllCapabilities,
				"noNewPrivileges":     containmentCfg.NoNewPrivileges,
				"readOnlyRootFS":      containmentCfg.ReadOnlyRootFS,
			},
		},
		// Environment variable reference
		"environmentVariables": map[string]interface{}{
			"agents": []string{
				"AGENT_LOCK_TIMEOUT_MINUTES",
				"AGENT_HEARTBEAT_INTERVAL_MINUTES",
				"AGENT_MAX_HEARTBEAT_FAILURES",
				"AGENT_DEFAULT_TIMEOUT_SECONDS",
				"AGENT_DEFAULT_MAX_TURNS",
				"AGENT_DEFAULT_MAX_FILES",
				"AGENT_DEFAULT_MAX_BYTES",
				"AGENT_DEFAULT_NETWORK_ENABLED",
				"AGENT_RETENTION_DAYS",
				"AGENT_CLEANUP_INTERVAL_MINUTES",
				"AGENT_MAX_PROMPTS",
				"AGENT_MAX_CONCURRENT",
				"AGENT_DEFAULT_CONCURRENCY",
				"AGENT_SPAWN_SESSION_TTL_MINUTES",
				"AGENT_IDEMPOTENCY_TTL_MINUTES",
			},
			"containment": []string{
				"CONTAINMENT_DOCKER_IMAGE",
				"CONTAINMENT_MAX_MEMORY_MB",
				"CONTAINMENT_MAX_CPU_PERCENT",
				"CONTAINMENT_AVAILABILITY_TIMEOUT_SECONDS",
				"CONTAINMENT_PREFER_DOCKER",
				"CONTAINMENT_ALLOW_FALLBACK",
				"CONTAINMENT_DROP_ALL_CAPABILITIES",
				"CONTAINMENT_NO_NEW_PRIVILEGES",
				"CONTAINMENT_READ_ONLY_ROOT_FS",
			},
		},
		"note": "All levers can be tuned via environment variables. " +
			"Values are validated and clamped to safe ranges on load. " +
			"See the config package documentation for detailed tradeoff descriptions.",
	}
	s.writeJSON(w, http.StatusOK, response)
}

// handleGetBlockedCommands returns command validation info (allowlist and blocklist).
func (s *Server) handleGetBlockedCommands(w http.ResponseWriter, r *http.Request) {
	validator := security.NewBashCommandValidator()

	// Get allowed commands (primary - allowlist)
	allowedCmds := validator.GetAllowedCommands()
	allowedCommands := make([]map[string]string, 0, len(allowedCmds))
	for _, ac := range allowedCmds {
		allowedCommands = append(allowedCommands, map[string]string{
			"prefix":      ac.Prefix,
			"description": ac.Description,
		})
	}

	// Get blocked patterns (secondary - defense in depth for prompt scanning)
	blockedPats := validator.GetBlockedPatterns()
	blockedPatterns := make([]map[string]string, 0, len(blockedPats))
	for _, bp := range blockedPats {
		blockedPatterns = append(blockedPatterns, map[string]string{
			"pattern":     bp.Pattern.String(),
			"description": bp.Description,
		})
	}

	response := map[string]interface{}{
		"allowedBashCommands": allowedCommands,
		"blockedPatterns":     blockedPatterns,
		"safeDefaults":        security.DefaultSafeTools(),
		"safeBashPatterns":    security.SafeBashPatterns(),
		"securityModel":       "allowlist",
		"note":                "Bash commands must start with an allowed prefix. The blocklist is used for defense-in-depth prompt scanning only.",
	}
	s.writeJSON(w, http.StatusOK, response)
}

// handleContainmentStatus returns the current OS-level containment status.
// This helps users understand what security measures are in place for agent execution.
func (s *Server) handleContainmentStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Use the injected containment selector for testability
	status := s.containmentSelector.GetStatus(ctx)

	// Get provider info for all registered providers
	providerInfos := s.containmentSelector.ListProviders(ctx)
	providers := make([]map[string]interface{}, 0, len(providerInfos))
	for _, info := range providerInfos {
		providers = append(providers, map[string]interface{}{
			"type":          string(info.Type),
			"name":          info.Name,
			"description":   info.Description,
			"securityLevel": info.SecurityLevel,
			"requirements":  info.Requirements,
		})
	}

	// Convert available providers to strings
	available := make([]string, 0, len(status.AvailableProviders))
	for _, p := range status.AvailableProviders {
		available = append(available, string(p))
	}

	// Get current containment configuration for display
	containmentCfg := containment.LoadConfigFromEnv()

	response := map[string]interface{}{
		"activeProvider":     string(status.ActiveProvider),
		"availableProviders": available,
		"securityLevel":      status.SecurityLevel,
		"maxSecurityLevel":   10,
		"warnings":           status.Warnings,
		"providers":          providers,
		// Expose current containment config as control surface
		"config": map[string]interface{}{
			"dockerImage":                containmentCfg.DockerImage,
			"maxMemoryMB":                containmentCfg.MaxMemoryMB,
			"maxCPUPercent":              containmentCfg.MaxCPUPercent,
			"availabilityTimeoutSeconds": containmentCfg.AvailabilityTimeoutSeconds,
			"preferDocker":               containmentCfg.PreferDocker,
			"allowFallback":              containmentCfg.AllowFallback,
			"dropAllCapabilities":        containmentCfg.DropAllCapabilities,
			"noNewPrivileges":            containmentCfg.NoNewPrivileges,
			"readOnlyRootFS":             containmentCfg.ReadOnlyRootFS,
		},
		"note": "Containment provides OS-level isolation for agent execution. " +
			"Higher security levels provide stronger isolation. " +
			"Docker (level 7) is recommended for production use. " +
			"Config values can be overridden via CONTAINMENT_* environment variables.",
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

	// Check required commands exist (pre-flight command validation)
	commandResults, commandWarnings, commandErrors := validateRequiredCommands(scenario)
	results = append(results, commandResults...)
	warnings = append(warnings, commandWarnings...)
	errors = append(errors, commandErrors...)

	response := map[string]interface{}{
		"valid":    len(errors) == 0,
		"paths":    results,
		"warnings": warnings,
		"errors":   errors,
	}
	s.writeJSON(w, http.StatusOK, response)
}

// validateRequiredCommands checks that commands the agent might need are available.
// Returns results, warnings, and errors.
func validateRequiredCommands(scenario string) ([]map[string]interface{}, []string, []string) {
	results := make([]map[string]interface{}, 0)
	warnings := make([]string, 0)
	errors := make([]string, 0)

	// Required commands that must exist for agent execution
	requiredCommands := []struct {
		Command     string
		Description string
		Required    bool // If true, blocks spawn; if false, just warns
	}{
		{"resource-opencode", "OpenCode CLI for agent execution", true},
	}

	// Optional commands that enhance agent capabilities (scenario-agnostic)
	optionalCommands := []struct {
		Command     string
		Description string
	}{
		{"go", "Go toolchain (for Go scenarios)"},
		{"pnpm", "pnpm package manager (for Node.js scenarios)"},
		{"npm", "npm package manager (for Node.js scenarios)"},
		{"vitest", "Vitest test runner"},
		{"jest", "Jest test runner"},
		{"bats", "BATS test runner (for bash tests)"},
		{"pytest", "Python pytest runner"},
		{"docker", "Docker (for containment)"},
	}

	// Check required commands
	for _, cmd := range requiredCommands {
		path, err := exec.LookPath(cmd.Command)
		exists := err == nil

		result := map[string]interface{}{
			"path":        path,
			"description": cmd.Description,
			"exists":      exists,
			"isDirectory": false,
			"required":    cmd.Required,
			"isCommand":   true,
		}
		results = append(results, result)

		if !exists {
			if cmd.Required {
				errors = append(errors, fmt.Sprintf("%s not found: %s is required for agent execution", cmd.Description, cmd.Command))
			} else {
				warnings = append(warnings, fmt.Sprintf("%s not found: %s (some functionality may be limited)", cmd.Description, cmd.Command))
			}
		}
	}

	// Check optional commands (just informational, no errors)
	availableOptional := make([]string, 0)
	for _, cmd := range optionalCommands {
		path, err := exec.LookPath(cmd.Command)
		exists := err == nil

		result := map[string]interface{}{
			"path":        path,
			"description": cmd.Description,
			"exists":      exists,
			"isDirectory": false,
			"required":    false,
			"isCommand":   true,
		}
		results = append(results, result)

		if exists {
			availableOptional = append(availableOptional, cmd.Command)
		}
	}

	// Add a summary note about available tools
	if len(availableOptional) > 0 {
		warnings = append(warnings, fmt.Sprintf("Available test tools: %s", strings.Join(availableOptional, ", ")))
	}

	return results, warnings, errors
}

// getUserIdentifier extracts a user identifier from the request for session tracking.
// Priority: X-User-ID header > X-API-Key header > X-Forwarded-For > RemoteAddr
func getUserIdentifier(r *http.Request) string {
	// Check for explicit user ID (from authenticated requests)
	if userID := r.Header.Get("X-User-ID"); userID != "" {
		return "user:" + userID
	}

	// Check for API key (from programmatic access)
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		// Hash the API key to avoid storing it directly
		h := sha256.New()
		h.Write([]byte(apiKey))
		return "apikey:" + hex.EncodeToString(h.Sum(nil))[:16]
	}

	// Fall back to IP address
	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		// Take the first IP in the chain (client IP)
		parts := strings.Split(forwarded, ",")
		ip = strings.TrimSpace(parts[0])
	}

	// Strip port if present
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		// Handle IPv6 addresses in brackets
		if !strings.Contains(ip[idx:], "]") {
			ip = ip[:idx]
		}
	}

	return "ip:" + ip
}

// handleCheckSpawnSessionConflicts checks for server-side spawn session conflicts.
func (s *Server) handleCheckSpawnSessionConflicts(w http.ResponseWriter, r *http.Request) {
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

	userIdentifier := getUserIdentifier(r)

	conflicts, err := s.agentService.CheckSpawnSessionConflicts(r.Context(), userIdentifier, scenario, payload.Scope)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to check spawn session conflicts: %s", err.Error()))
		return
	}

	response := map[string]interface{}{
		"hasConflicts":   len(conflicts) > 0,
		"conflicts":      conflicts,
		"userIdentifier": userIdentifier, // Return so UI knows which identifier is being used
	}
	s.writeJSON(w, http.StatusOK, response)
}

// handleGetSpawnSessions returns active spawn sessions for the current user.
func (s *Server) handleGetSpawnSessions(w http.ResponseWriter, r *http.Request) {
	scenario := r.URL.Query().Get("scenario")
	userIdentifier := getUserIdentifier(r)

	sessions, err := s.agentService.GetActiveSpawnSessions(r.Context(), userIdentifier, scenario)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get spawn sessions: %s", err.Error()))
		return
	}

	response := map[string]interface{}{
		"sessions":       sessions,
		"count":          len(sessions),
		"userIdentifier": userIdentifier,
	}
	s.writeJSON(w, http.StatusOK, response)
}

// handleClearSpawnSessions clears all active spawn sessions for the current user.
func (s *Server) handleClearSpawnSessions(w http.ResponseWriter, r *http.Request) {
	userIdentifier := getUserIdentifier(r)

	cleared, err := s.agentService.ClearSpawnSessions(r.Context(), userIdentifier)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to clear spawn sessions: %s", err.Error()))
		return
	}

	s.log("spawn sessions cleared", map[string]interface{}{
		"userIdentifier": userIdentifier,
		"clearedCount":   cleared,
	})

	response := map[string]interface{}{
		"message":        "spawn sessions cleared",
		"clearedCount":   cleared,
		"userIdentifier": userIdentifier,
	}
	s.writeJSON(w, http.StatusOK, response)
}
