// Package toolregistry provides the tool discovery service for workspace-sandbox.
//
// This package implements the Tool Discovery Protocol, enabling external consumers
// (like agent-inbox) to discover what tools this scenario provides.
//
// ARCHITECTURE:
// - Registry: Central coordinator that manages tool providers
// - ToolProvider: Interface for components that contribute tools
// - ToolDefinitions: Static tool definitions (sandbox lifecycle, execution, files, diff)
//
// TESTING SEAMS:
// - ToolProvider interface enables mocking tool sources
// - Registry accepts providers via dependency injection
// - All dependencies are interfaces, not concrete types
package toolregistry

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ToolProvider is the interface for components that contribute tools.
// Implement this interface to add tools from different sources.
type ToolProvider interface {
	// Name returns a unique identifier for this provider.
	Name() string

	// Tools returns the tool definitions this provider contributes.
	// Context allows for cancellation of expensive operations.
	Tools(ctx context.Context) []*toolspb.ToolDefinition

	// Categories returns the category definitions for this provider's tools.
	Categories(ctx context.Context) []*toolspb.ToolCategory
}

// Registry manages tool discovery and manifest generation.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]ToolProvider

	// Scenario metadata
	scenarioName    string
	scenarioVersion string
	scenarioDesc    string
}

// RegistryConfig holds configuration for creating a Registry.
type RegistryConfig struct {
	ScenarioName        string
	ScenarioVersion     string
	ScenarioDescription string
}

// NewRegistry creates a new tool registry with the given configuration.
func NewRegistry(cfg RegistryConfig) *Registry {
	return &Registry{
		providers:       make(map[string]ToolProvider),
		scenarioName:    cfg.ScenarioName,
		scenarioVersion: cfg.ScenarioVersion,
		scenarioDesc:    cfg.ScenarioDescription,
	}
}

// RegisterProvider adds a tool provider to the registry.
// If a provider with the same name exists, it is replaced.
func (r *Registry) RegisterProvider(provider ToolProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[provider.Name()] = provider
}

// UnregisterProvider removes a tool provider from the registry.
func (r *Registry) UnregisterProvider(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.providers, name)
}

// GetManifest generates the complete tool manifest.
// This is the main entry point for the GET /api/v1/tools endpoint.
func (r *Registry) GetManifest(ctx context.Context) *toolspb.ToolManifest {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Collect tools and categories from all providers
	var allTools []*toolspb.ToolDefinition
	categoryMap := make(map[string]*toolspb.ToolCategory)

	for _, provider := range r.providers {
		tools := provider.Tools(ctx)
		allTools = append(allTools, tools...)

		for _, cat := range provider.Categories(ctx) {
			// Later providers can override categories
			categoryMap[cat.Id] = cat
		}
	}

	// Convert category map to slice
	categories := make([]*toolspb.ToolCategory, 0, len(categoryMap))
	for _, cat := range categoryMap {
		categories = append(categories, cat)
	}

	return &toolspb.ToolManifest{
		ProtocolVersion: ToolProtocolVersion,
		Scenario: &toolspb.ScenarioInfo{
			Name:        r.scenarioName,
			Version:     r.scenarioVersion,
			Description: r.scenarioDesc,
		},
		Tools:       allTools,
		Categories:  categories,
		GeneratedAt: timestamppb.Now(),
	}
}

// GetTool returns a specific tool by name.
// Returns nil if the tool is not found.
func (r *Registry) GetTool(ctx context.Context, name string) *toolspb.ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, provider := range r.providers {
		for _, tool := range provider.Tools(ctx) {
			if tool.Name == name {
				return tool
			}
		}
	}
	return nil
}

// ListToolNames returns the names of all registered tools.
func (r *Registry) ListToolNames(ctx context.Context) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var names []string
	for _, provider := range r.providers {
		for _, tool := range provider.Tools(ctx) {
			names = append(names, tool.Name)
		}
	}
	return names
}

// ProviderCount returns the number of registered providers.
func (r *Registry) ProviderCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.providers)
}

// -----------------------------------------------------------------------------
// HTTP Handlers
// -----------------------------------------------------------------------------

// protojson marshaler configured for compatibility with OpenAI format
var protoMarshaler = protojson.MarshalOptions{
	UseProtoNames:   false, // Use camelCase field names for JSON
	EmitUnpopulated: false, // Don't emit zero values
}

// HandleGetManifest handles GET /api/v1/tools
// Returns the complete tool manifest for this scenario.
func (r *Registry) HandleGetManifest(w http.ResponseWriter, req *http.Request) {
	// Handle CORS preflight
	if req.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	manifest := r.GetManifest(req.Context())

	// Use protojson for proper proto JSON serialization
	data, err := protoMarshaler.Marshal(manifest)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=60") // Cache for 1 minute
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// HandleGetTool handles GET /api/v1/tools/{name}
// Returns a specific tool definition by name.
func (r *Registry) HandleGetTool(w http.ResponseWriter, req *http.Request) {
	// Handle CORS preflight
	if req.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Extract tool name from path
	name := extractPathParam(req.URL.Path, "/api/v1/tools/")
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "tool name is required")
		return
	}

	tool := r.GetTool(req.Context(), name)
	if tool == nil {
		writeJSONError(w, http.StatusNotFound, "tool not found: "+name)
		return
	}

	// Use protojson for proper proto JSON serialization
	data, err := protoMarshaler.Marshal(tool)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// extractPathParam extracts the parameter after a prefix from the URL path.
func extractPathParam(path, prefix string) string {
	if len(path) <= len(prefix) {
		return ""
	}
	param := path[len(prefix):]
	// Remove any trailing slashes or query parameters
	if idx := strings.IndexAny(param, "/?"); idx != -1 {
		param = param[:idx]
	}
	return param
}

// writeJSONError writes a JSON error response.
func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := map[string]string{
		"error":   http.StatusText(status),
		"message": message,
	}
	_ = json.NewEncoder(w).Encode(resp)
}
