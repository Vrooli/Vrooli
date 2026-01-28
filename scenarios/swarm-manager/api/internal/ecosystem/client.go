// Package ecosystem provides a client interface for ecosystem-manager integration.
//
// This package implements the seam pattern - all ecosystem-manager communication
// goes through the EcosystemClient interface, which can be substituted for testing.
//
// Design Goals:
//   - Testability: Mock the interface to test handlers without HTTP
//   - Isolation: Changes to ecosystem-manager API are localized here
//   - Observability: Centralized error handling and logging
//
// Related PRD targets: OT-P0-005 (queue ideas for processing)
// DOC: docs/internal/SEAMS.md#api-to-integration-seam
package ecosystem

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Task represents a task in the ecosystem-manager queue.
type Task struct {
	ID        string `json:"id,omitempty"`
	Title     string `json:"title"`
	Type      string `json:"type"`      // "scenario"
	Operation string `json:"operation"` // "generator" or "improver"
	Category  string `json:"category,omitempty"`
	Priority  string `json:"priority"`
	Status    string `json:"status,omitempty"`
}

// CreateTaskRequest contains the parameters for creating a task.
type CreateTaskRequest struct {
	Title     string
	Operation string // "generator" or "improver"
	Priority  int    // 1-10, maps to high/medium/low
	Category  string // Optional category (typically first tag)
}

// Sentinel errors for ecosystem-manager integration.
var (
	// ErrNotAvailable is returned when ecosystem-manager cannot be reached.
	ErrNotAvailable = errors.New("ecosystem-manager not available")

	// ErrTaskCreationFailed is returned when task creation fails.
	ErrTaskCreationFailed = errors.New("failed to create task in ecosystem-manager")
)

// Client is the interface for ecosystem-manager operations.
// This is the seam - implementations can be swapped for testing.
type Client interface {
	// CreateTask creates a task in the ecosystem-manager queue.
	// Returns the created task ID on success.
	CreateTask(req CreateTaskRequest) (string, error)
}

// HTTPClient implements Client using HTTP calls to ecosystem-manager.
type HTTPClient struct {
	// portResolver allows injection of port resolution logic for testing.
	portResolver func() (string, error)
	// httpClient allows injection of HTTP client for testing.
	httpClient HTTPDoer
}

// HTTPDoer is the interface for HTTP operations (allows mocking http.Client).
type HTTPDoer interface {
	Post(url, contentType string, body *strings.Reader) (*http.Response, error)
}

// defaultHTTPDoer wraps http.DefaultClient to implement HTTPDoer.
type defaultHTTPDoer struct{}

func (d *defaultHTTPDoer) Post(url, contentType string, body *strings.Reader) (*http.Response, error) {
	return http.Post(url, contentType, body)
}

// NewHTTPClient creates a new ecosystem-manager HTTP client.
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		portResolver: resolveEcosystemPort,
		httpClient:   &defaultHTTPDoer{},
	}
}

// NewHTTPClientWithResolver creates an HTTP client with a custom port resolver.
// This is useful for testing.
func NewHTTPClientWithResolver(resolver func() (string, error), httpClient HTTPDoer) *HTTPClient {
	return &HTTPClient{
		portResolver: resolver,
		httpClient:   httpClient,
	}
}

// CreateTask creates a task in the ecosystem-manager queue.
func (c *HTTPClient) CreateTask(req CreateTaskRequest) (string, error) {
	port, err := c.portResolver()
	if err != nil {
		return "", err
	}

	// Map priority to string
	priority := mapPriorityToString(req.Priority)

	// Build task payload
	task := Task{
		Title:     req.Title,
		Type:      "scenario",
		Operation: req.Operation,
		Category:  req.Category,
		Priority:  priority,
	}

	taskJSON, err := json.Marshal(task)
	if err != nil {
		return "", fmt.Errorf("failed to marshal task: %w", err)
	}

	// POST to ecosystem-manager
	url := "http://localhost:" + port + "/api/tasks"
	resp, err := c.httpClient.Post(url, "application/json", strings.NewReader(string(taskJSON)))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrNotAvailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: status %d", ErrTaskCreationFailed, resp.StatusCode)
	}

	// Parse response to get task ID
	var createdTask Task
	if err := json.NewDecoder(resp.Body).Decode(&createdTask); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return createdTask.ID, nil
}

// resolveEcosystemPort finds the ecosystem-manager port from environment or port file.
func resolveEcosystemPort() (string, error) {
	// Check environment first
	if port := os.Getenv("ECOSYSTEM_MANAGER_PORT"); port != "" {
		return port, nil
	}

	// Try to read from port file
	portFile := filepath.Join("scenarios", "ecosystem-manager", ".vrooli", "ports", "API_PORT")
	data, err := os.ReadFile(portFile)
	if err != nil {
		return "", ErrNotAvailable
	}

	port := strings.TrimSpace(string(data))
	if port == "" {
		return "", ErrNotAvailable
	}

	return port, nil
}

// mapPriorityToString converts a numeric priority (1-10) to ecosystem-manager's string format.
func mapPriorityToString(priority int) string {
	switch {
	case priority <= 2:
		return "high"
	case priority >= 8:
		return "low"
	default:
		return "medium"
	}
}
