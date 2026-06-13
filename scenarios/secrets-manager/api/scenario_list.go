package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	vroolicli "github.com/vrooli/vrooli-cli-go"
)

// cliClient is the shared typed Vrooli CLI client. It decodes the
// vrooli.cli.v1 contracts instead of hand-parsing CLI JSON, so a CLI output
// change is a compile error here rather than a silently empty or wrong result.
var cliClient = vroolicli.New()

// -----------------------------------------------------------------------------
// ScenarioCLI Interface
// -----------------------------------------------------------------------------

// ScenarioCLI abstracts scenario CLI operations for testability.
// This interface enables mocking scenario list responses in tests without
// requiring the actual vrooli CLI to be installed.
type ScenarioCLI interface {
	// ListScenarios retrieves the list of available scenarios.
	ListScenarios(ctx context.Context) ([]scenarioSummary, error)
}

// DefaultScenarioCLI implements ScenarioCLI using the vrooli CLI.
type DefaultScenarioCLI struct{}

// NewDefaultScenarioCLI creates the production ScenarioCLI implementation.
func NewDefaultScenarioCLI() *DefaultScenarioCLI {
	return &DefaultScenarioCLI{}
}

// ListScenarios implements ScenarioCLI by calling vrooli scenario list.
func (c *DefaultScenarioCLI) ListScenarios(ctx context.Context) ([]scenarioSummary, error) {
	return fetchScenarioListImpl(ctx)
}

// defaultScenarioCLI is the package-level scenario CLI instance.
// It can be replaced in tests via SetScenarioCLI.
var defaultScenarioCLI ScenarioCLI = NewDefaultScenarioCLI()

// SetScenarioCLI replaces the default scenario CLI implementation.
// This is primarily used for testing with mock implementations.
func SetScenarioCLI(cli ScenarioCLI) {
	defaultScenarioCLI = cli
}

// -----------------------------------------------------------------------------
// Types
// -----------------------------------------------------------------------------

type scenarioSummary struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Version     string   `json:"version,omitempty"`
	Status      string   `json:"status,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Path        string   `json:"path,omitempty"`
}

// -----------------------------------------------------------------------------
// Handlers
// -----------------------------------------------------------------------------

type ScenarioHandlers struct {
	scenarioCLI ScenarioCLI
}

// NewScenarioHandlers creates a ScenarioHandlers with the default ScenarioCLI.
func NewScenarioHandlers() *ScenarioHandlers {
	return &ScenarioHandlers{
		scenarioCLI: defaultScenarioCLI,
	}
}

// NewScenarioHandlersWithCLI creates a ScenarioHandlers with a custom ScenarioCLI.
// This is primarily used for testing with mock implementations.
func NewScenarioHandlersWithCLI(cli ScenarioCLI) *ScenarioHandlers {
	return &ScenarioHandlers{
		scenarioCLI: cli,
	}
}

// RegisterRoutes exposes scenario list endpoints for UI selection lists.
func (h *ScenarioHandlers) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("", h.ScenarioList).Methods("GET")
}

func (h *ScenarioHandlers) ScenarioList(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	scenarios, err := h.scenarioCLI.ListScenarios(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to load scenarios: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"scenarios": scenarios,
		"count":     len(scenarios),
	})
}

// fetchScenarioList retrieves scenarios using the package-level ScenarioCLI.
// This function is retained for backward compatibility.
func fetchScenarioList(ctx context.Context) ([]scenarioSummary, error) {
	return defaultScenarioCLI.ListScenarios(ctx)
}

// fetchScenarioListImpl is the underlying implementation using the typed CLI
// client, mapping the vrooli.cli.v1 scenario-list contract onto scenarioSummary.
func fetchScenarioListImpl(ctx context.Context) ([]scenarioSummary, error) {
	resp, err := cliClient.ListScenarios(ctx)
	if err != nil {
		return nil, fmt.Errorf("vrooli scenario list failed: %w", err)
	}
	if !resp.GetSuccess() && len(resp.GetScenarios()) == 0 {
		return nil, fmt.Errorf("scenario list did not return results")
	}

	// Normalize names to avoid trailing slashes or whitespace from CLI output.
	scenarios := make([]scenarioSummary, 0, len(resp.GetScenarios()))
	for _, s := range resp.GetScenarios() {
		scenarios = append(scenarios, scenarioSummary{
			Name:        strings.TrimSpace(s.GetName()),
			Description: strings.TrimSpace(s.GetDescription()),
			Version:     strings.TrimSpace(s.GetVersion()),
			Status:      strings.TrimSpace(s.GetStatus()),
			Tags:        s.GetTags(),
			Path:        strings.TrimSpace(s.GetPath()),
		})
	}

	return scenarios, nil
}
