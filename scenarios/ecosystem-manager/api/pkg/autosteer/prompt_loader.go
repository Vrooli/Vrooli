package autosteer

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// PromptResponse is the response from prompt-manager for a single prompt.
type PromptResponse struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	Content             string   `json:"content"`
	Modes               []string `json:"modes"`
	Tags                []string `json:"tags"`
	Icon                string   `json:"icon,omitempty"`
	TargetToolID        *string  `json:"targetToolId,omitempty"`
	Draft               bool     `json:"draft"`
	Folder              string   `json:"folder"`
	CreatedAt           string   `json:"createdAt"`
	UpdatedAt           string   `json:"updatedAt"`
	UsageCount          int      `json:"usageCount"`
	LastUsed            *string  `json:"lastUsed,omitempty"`
	EffectivenessRating *int     `json:"effectivenessRating,omitempty"`
}

// SyncResponse is the response from prompt-manager's sync endpoint.
type SyncResponse struct {
	Skills      []PromptResponse `json:"skills"`
	LastUpdated string           `json:"lastUpdated"`
	Hash        string           `json:"hash"`
}

// phasePromptData contains parsed prompt data for a steering mode.
type phasePromptData struct {
	Instructions        string
	SuccessCriteria     []string
	ToolRecommendations []string
	Raw                 string
}

// PromptLoaderConfig contains configuration for the prompt loader.
type PromptLoaderConfig struct {
	PromptManagerURL string
	CacheTTL         time.Duration
	Timeout          time.Duration
}

// DefaultPromptLoaderConfig returns a default configuration.
func DefaultPromptLoaderConfig() *PromptLoaderConfig {
	// Check for explicit override first (useful for testing)
	url := os.Getenv("PROMPT_MANAGER_URL")
	if url == "" {
		// Use api-core discovery to resolve prompt-manager URL
		resolver := discovery.NewResolver(discovery.ResolverConfig{})
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		resolvedURL, err := resolver.ResolveScenarioURLDefault(ctx, "prompt-manager")
		if err != nil {
			// Discovery failed - will try again on first sync
			log.Printf("Auto Steer: prompt-manager discovery failed: %v", err)
			url = "" // Empty URL signals unavailable
		} else {
			url = resolvedURL
		}
	}

	return &PromptLoaderConfig{
		PromptManagerURL: url,
		CacheTTL:         5 * time.Minute,
		Timeout:          30 * time.Second,
	}
}

// fallbackInstructions provides default steering content when prompt-manager is unavailable.
var fallbackInstructions = map[SteerMode]string{
	ModeProgress:    "Focus on advancing the task toward completion. Make meaningful progress on the primary objectives.",
	ModeExplore:     "Explore the problem space thoroughly. Gather information before committing to solutions.",
	ModeRefactor:    "Improve code quality while preserving behavior. Focus on readability and maintainability.",
	ModeTest:        "Strengthen test coverage. Add tests for edge cases and failure scenarios.",
	ModePolish:      "Apply final touches. Fix typos, improve formatting, ensure consistency.",
	ModeUX:          "Improve user experience. Focus on usability, clarity, and accessibility.",
	ModePerformance: "Optimize performance. Profile first, then optimize hot paths.",
	ModeSecurity:    "Review for security vulnerabilities. Check input validation and access controls.",
}

// PromptLoader fetches prompts from prompt-manager API with caching.
// Supports graceful degradation when prompt-manager is unavailable.
type PromptLoader struct {
	cfg         *PromptLoaderConfig
	client      *http.Client
	mu          sync.RWMutex
	cache       map[string]*cachedPrompt
	rawSkills   []PromptResponse // Store raw skills for UI access
	lastSync    time.Time
	available   bool      // is prompt-manager reachable?
	lastAttempt time.Time // when did we last try to connect?
}

type cachedPrompt struct {
	data      phasePromptData
	fetchedAt time.Time
}

// NewPromptLoader creates a new prompt loader.
// Does NOT fail if prompt-manager is unavailable - starts in degraded mode.
func NewPromptLoader(cfg *PromptLoaderConfig) *PromptLoader {
	if cfg == nil {
		cfg = DefaultPromptLoaderConfig()
	}

	loader := &PromptLoader{
		cfg: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		cache:     make(map[string]*cachedPrompt),
		available: false,
	}

	// Attempt initial sync, but don't fail if unavailable
	if cfg.PromptManagerURL != "" {
		if err := loader.syncAll(); err != nil {
			log.Printf("Auto Steer: prompt-manager unavailable at startup: %v (operating in degraded mode)", err)
			loader.lastAttempt = time.Now()
		} else {
			loader.available = true
			log.Printf("Auto Steer: prompt-manager connected, %d prompts cached", len(loader.cache))
		}
	} else {
		log.Printf("Auto Steer: prompt-manager URL not resolved (operating in degraded mode)")
		loader.lastAttempt = time.Now()
	}

	return loader
}

// IsAvailable returns whether prompt-manager is reachable.
func (l *PromptLoader) IsAvailable() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.available
}

// syncAll fetches all skill prompts from prompt-manager.
func (l *PromptLoader) syncAll() error {
	url := fmt.Sprintf("%s/api/v1/skills/sync", l.cfg.PromptManagerURL)

	resp, err := l.client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch prompts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("prompt-manager returned status %d: %s", resp.StatusCode, string(body))
	}

	var syncResp SyncResponse
	if err := json.NewDecoder(resp.Body).Decode(&syncResp); err != nil {
		return fmt.Errorf("failed to decode sync response: %w", err)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	customModes := make([]SteerMode, 0)
	for _, p := range syncResp.Skills {
		if len(p.Modes) == 0 || strings.TrimSpace(strings.ToLower(p.Modes[0])) != "steer" {
			continue
		}

		normalizedID := normalizeSteerMode(SteerMode(p.ID))
		if normalizedID == "" {
			continue
		}
		normalizedKey := string(normalizedID)

		data, err := parsePhasePrompt(p.Content)
		if err != nil {
			log.Printf("Warning: could not parse skill %s: %v", p.ID, err)
			continue
		}

		l.cache[normalizedKey] = &cachedPrompt{
			data:      data,
			fetchedAt: now,
		}

		customModes = append(customModes, normalizedID)
	}

	// Store raw skills for UI access
	l.rawSkills = syncResp.Skills

	if len(customModes) > 0 {
		RegisterSteerModes(customModes...)
	}

	l.lastSync = now
	log.Printf("Synced %d skills from prompt-manager", len(syncResp.Skills))

	return nil
}

// loadPrompt loads a prompt by mode ID.
// Returns empty data and false if unavailable (instead of error).
func (l *PromptLoader) loadPrompt(mode SteerMode) (phasePromptData, bool) {
	modeStr := string(normalizeSteerMode(mode))

	// Check cache first
	l.mu.RLock()
	if cached, ok := l.cache[modeStr]; ok {
		if time.Since(cached.fetchedAt) < l.cfg.CacheTTL {
			l.mu.RUnlock()
			return cached.data, true
		}
	}
	available := l.available
	lastAttempt := l.lastAttempt
	l.mu.RUnlock()

	// If unavailable and 30s+ since last attempt, try reconnecting
	if !available && time.Since(lastAttempt) > 30*time.Second {
		l.mu.Lock()
		l.lastAttempt = time.Now()
		l.mu.Unlock()

		// Try to re-resolve URL if empty
		if l.cfg.PromptManagerURL == "" {
			l.cfg = DefaultPromptLoaderConfig()
		}

		if l.cfg.PromptManagerURL != "" {
			if err := l.syncAll(); err == nil {
				l.mu.Lock()
				l.available = true
				l.mu.Unlock()
				log.Printf("Auto Steer: prompt-manager recovered")
			}
		}
	} else if available {
		// Normal refresh attempt
		if err := l.syncAll(); err != nil {
			l.mu.Lock()
			l.available = false
			l.mu.Unlock()
			log.Printf("Auto Steer: prompt-manager became unavailable: %v", err)
		}
	}

	// Return from cache if available
	l.mu.RLock()
	defer l.mu.RUnlock()
	if cached, ok := l.cache[modeStr]; ok {
		return cached.data, true
	}

	// Return fallback if available
	if fallback, ok := fallbackInstructions[mode]; ok {
		return phasePromptData{
			Instructions: fallback,
			Raw:          fallback,
		}, true
	}

	return phasePromptData{}, false
}

// GetInstructions returns the detailed instructions for a given mode.
// Returns empty string if prompt-manager unavailable.
func (l *PromptLoader) GetInstructions(mode SteerMode) string {
	data, ok := l.loadPrompt(mode)
	if !ok {
		return "" // Empty - caller should handle gracefully
	}
	return data.Instructions
}

// GetToolRecommendations returns recommended tools for a mode.
func (l *PromptLoader) GetToolRecommendations(mode SteerMode) []string {
	data, ok := l.loadPrompt(mode)
	if !ok {
		return nil
	}
	return data.ToolRecommendations
}

// GetSuccessCriteria returns success criteria for a mode.
func (l *PromptLoader) GetSuccessCriteria(mode SteerMode) []string {
	data, ok := l.loadPrompt(mode)
	if !ok {
		return nil
	}
	return data.SuccessCriteria
}

// FormatConditionProgress renders stop conditions with their evaluated status.
func (l *PromptLoader) FormatConditionProgress(conditions []StopCondition, metrics MetricsSnapshot, evaluator ConditionEvaluatorAPI) string {
	if len(conditions) == 0 {
		return ""
	}

	eval := evaluator
	if eval == nil {
		eval = NewConditionEvaluator()
	}

	var builder strings.Builder
	for _, condition := range conditions {
		builder.WriteString("- ")
		builder.WriteString(eval.FormatCondition(condition, metrics))
		builder.WriteString("\n")
	}

	return strings.TrimSpace(builder.String())
}

// FormatModeContent formats the mode content with the steer focus header.
func FormatModeContent(mode SteerMode, content string) string {
	// Normalize mode name: "refactor" -> "Refactor"
	modeName := strings.Title(strings.ReplaceAll(string(mode), "_", " "))
	return fmt.Sprintf("## Steer focus: %s\n\n%s", modeName, content)
}

// RefreshCache forces a refresh of the cache from prompt-manager.
func (l *PromptLoader) RefreshCache() error {
	return l.syncAll()
}

// GetCachedSkills returns the raw skills that were synced from prompt-manager.
// This is used by the UI to display available skills.
func (l *PromptLoader) GetCachedSkills() []PromptResponse {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.rawSkills
}

// parsePhasePrompt parses a markdown prompt into structured data.
func parsePhasePrompt(content string) (phasePromptData, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	section := "instructions"
	var instructions []string
	var success []string
	var tools []string

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)

		switch {
		case strings.HasPrefix(lower, "## success criteria"):
			section = "success"
			continue
		case strings.HasPrefix(lower, "## recommended tools"), strings.HasPrefix(lower, "## tools"):
			section = "tools"
			continue
		}

		switch section {
		case "instructions":
			instructions = append(instructions, line)
		case "success":
			if strings.HasPrefix(trimmed, "-") {
				success = append(success, strings.TrimSpace(strings.TrimPrefix(trimmed, "-")))
			}
		case "tools":
			if strings.HasPrefix(trimmed, "-") {
				tools = append(tools, strings.TrimSpace(strings.TrimPrefix(trimmed, "-")))
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return phasePromptData{}, fmt.Errorf("failed to parse phase prompt: %w", err)
	}

	instructionText := strings.TrimSpace(strings.Join(instructions, "\n"))
	if instructionText == "" {
		return phasePromptData{}, fmt.Errorf("phase prompt missing instructions section")
	}

	return phasePromptData{
		Instructions:        instructionText,
		SuccessCriteria:     success,
		ToolRecommendations: tools,
		Raw:                 content,
	}, nil
}
