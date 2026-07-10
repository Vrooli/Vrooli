// Package promptmanager provides a client for reading skill prompts from prompt-manager.
//
// DOC: docs/internal/SEAMS.md
package promptmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// ReadSkillResult holds the content and optional variant selection from a prompt-manager read.
type ReadSkillResult struct {
	Content   string `json:"content"`
	VariantID string `json:"selectedVariantId,omitempty"`
}

type SkillSourceSnapshot struct {
	SkillID           string
	Revision          int
	SelectedVariantID string
	Content           string
	ContentHash       string
	TemplateVariables []string
}

// ExperimentOutcomeRequest is the request body for recording an experiment outcome.
type ExperimentOutcomeRequest struct {
	VariantID     string          `json:"variantId"`
	Source        string          `json:"source"`
	SchemaVersion int             `json:"schemaVersion"`
	Data          json.RawMessage `json:"data"`
}

// Client reads skill prompts from prompt-manager.
type Client interface {
	ReadSkill(ctx context.Context, skillID string, variables map[string]string, withScope bool) (string, error)
	ReadSkillWithExperiment(ctx context.Context, skillID string, variables map[string]string, withScope bool, experimentID string) (ReadSkillResult, error)
}

// SourceClient exposes immutable source metadata for consumers that must pin
// prompt interpretation across retries and service restarts.
type SourceClient interface {
	ReadSkillSource(ctx context.Context, skillID string, expectedVariables []string) (SkillSourceSnapshot, error)
}

// ExperimentClient provides experiment outcome operations against prompt-manager.
type ExperimentClient interface {
	RecordExperimentOutcome(ctx context.Context, experimentID string, outcome ExperimentOutcomeRequest) error
	ListExperimentOutcomes(ctx context.Context, experimentID string) ([]json.RawMessage, error)
}

// PromptSkill represents prompt-manager skill metadata and optional content.
type PromptSkill struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Content      string `json:"content,omitempty"`
	DefaultScope string `json:"defaultScope,omitempty"`
	Draft        bool   `json:"draft"`
	Folder       string `json:"folder,omitempty"`
	CreatedAt    string `json:"createdAt,omitempty"`
	UpdatedAt    string `json:"updatedAt,omitempty"`
}

// PromptSkillUpdate captures mutable skill fields exposed by prompt-manager.
type PromptSkillUpdate struct {
	Name         *string `json:"name,omitempty"`
	Description  *string `json:"description,omitempty"`
	Content      *string `json:"content,omitempty"`
	DefaultScope *string `json:"defaultScope,omitempty"`
	Draft        *bool   `json:"draft,omitempty"`
	Folder       *string `json:"folder,omitempty"`
}

// PromptSkillVersion captures one version history entry from prompt-manager.
type PromptSkillVersion struct {
	Version   int    `json:"version"`
	Content   string `json:"content"`
	Name      string `json:"name"`
	UpdatedAt string `json:"updatedAt"`
	CreatedBy string `json:"createdBy,omitempty"`
}

// PromptSkillVersions contains version history for a single skill.
type PromptSkillVersions struct {
	SkillID  string               `json:"skillId"`
	Current  int                  `json:"current"`
	Versions []PromptSkillVersion `json:"versions"`
}

// AdminClient exposes management operations used by the Swarm Manager Prompt Center.
type AdminClient interface {
	Client
	ListSkills(ctx context.Context, tag string) ([]PromptSkill, error)
	GetSkill(ctx context.Context, skillID string) (PromptSkill, error)
	UpdateSkill(ctx context.Context, skillID string, patch PromptSkillUpdate) (PromptSkill, error)
	GetSkillVersions(ctx context.Context, skillID string) (PromptSkillVersions, error)
	RevertSkillVersion(ctx context.Context, skillID string, version int) error
}

// BaseURLResolver resolves the base URL for prompt-manager.
type BaseURLResolver func(ctx context.Context) (string, error)

// HTTPDoer allows injecting HTTP client for tests.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// HTTPClient implements Client via prompt-manager's HTTP API.
type HTTPClient struct {
	baseURLResolver BaseURLResolver
	httpClient      HTTPDoer
}

// NewHTTPClient creates a new prompt-manager HTTP client with default settings.
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		baseURLResolver: resolvePromptManagerBaseURL,
		httpClient:      &http.Client{Timeout: 10 * time.Second},
	}
}

// NewHTTPClientWithResolver creates a client with custom resolver and HTTP client (for tests).
func NewHTTPClientWithResolver(resolver BaseURLResolver, httpClient HTTPDoer) *HTTPClient {
	if resolver == nil {
		resolver = resolvePromptManagerBaseURL
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &HTTPClient{
		baseURLResolver: resolver,
		httpClient:      httpClient,
	}
}

// readRequest is the request body for the skill read endpoint.
type readRequest struct {
	Identifiers  []string          `json:"identifiers"`
	Variables    map[string]string `json:"variables,omitempty"`
	Output       string            `json:"output"`
	WithScope    bool              `json:"withScope,omitempty"`
	ExperimentID string            `json:"experimentId,omitempty"`
}

// readResponse is the response from the skill read endpoint.
type readResponse struct {
	Combined          string `json:"combined"`
	CombinedHash      string `json:"combinedHash,omitempty"`
	SelectedVariantID string `json:"selectedVariantId,omitempty"`
	Skills            []struct {
		ID          string `json:"id"`
		Revision    int    `json:"revision,omitempty"`
		ContentHash string `json:"contentHash,omitempty"`
		Variables   []struct {
			Name string `json:"name"`
		} `json:"variables,omitempty"`
	} `json:"skills,omitempty"`
}

// ReadSkill fetches a single skill from prompt-manager with variable substitution.
func (c *HTTPClient) ReadSkill(ctx context.Context, skillID string, variables map[string]string, withScope bool) (string, error) {
	baseURL, err := c.baseURLResolver(ctx)
	if err != nil {
		return "", fmt.Errorf("promptmanager: resolve URL: %w", err)
	}

	reqBody := readRequest{
		Identifiers: []string{skillID},
		Variables:   variables,
		Output:      "combined",
		WithScope:   withScope,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("promptmanager: marshal request: %w", err)
	}

	url := baseURL + "/api/v1/skills/read"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("promptmanager: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("promptmanager: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("promptmanager: status %d: %s", resp.StatusCode, string(respBody))
	}

	var readResp readResponse
	if err := json.NewDecoder(resp.Body).Decode(&readResp); err != nil {
		return "", fmt.Errorf("promptmanager: decode response: %w", err)
	}

	return readResp.Combined, nil
}

func (c *HTTPClient) ReadSkillSource(ctx context.Context, skillID string, _ []string) (SkillSourceSnapshot, error) {
	baseURL, err := c.baseURLResolver(ctx)
	if err != nil {
		return SkillSourceSnapshot{}, fmt.Errorf("promptmanager: resolve URL: %w", err)
	}
	reqBody := readRequest{
		Identifiers: []string{skillID},
		Output:      "both",
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return SkillSourceSnapshot{}, fmt.Errorf("promptmanager: marshal source request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/skills/read", bytes.NewReader(body))
	if err != nil {
		return SkillSourceSnapshot{}, fmt.Errorf("promptmanager: create source request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return SkillSourceSnapshot{}, fmt.Errorf("promptmanager: source request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return SkillSourceSnapshot{}, fmt.Errorf("promptmanager: source status %d: %s", resp.StatusCode, string(respBody))
	}
	var readResp readResponse
	if err := json.NewDecoder(resp.Body).Decode(&readResp); err != nil {
		return SkillSourceSnapshot{}, fmt.Errorf("promptmanager: decode source response: %w", err)
	}
	if len(readResp.Skills) != 1 || strings.TrimSpace(readResp.Combined) == "" || strings.TrimSpace(readResp.CombinedHash) == "" {
		return SkillSourceSnapshot{}, fmt.Errorf("promptmanager: source response for %q is incomplete", skillID)
	}
	skill := readResp.Skills[0]
	variables := make([]string, 0, len(skill.Variables))
	for _, variable := range skill.Variables {
		if name := strings.TrimSpace(variable.Name); name != "" {
			variables = append(variables, name)
		}
	}
	sort.Strings(variables)
	variant := strings.TrimSpace(readResp.SelectedVariantID)
	if variant == "" {
		variant = "control"
	}
	return SkillSourceSnapshot{
		SkillID: skill.ID, Revision: skill.Revision, SelectedVariantID: variant,
		Content: readResp.Combined, ContentHash: readResp.CombinedHash, TemplateVariables: variables,
	}, nil
}

// ReadSkillWithExperiment fetches a skill from prompt-manager with experiment-based variant selection.
func (c *HTTPClient) ReadSkillWithExperiment(ctx context.Context, skillID string, variables map[string]string, withScope bool, experimentID string) (ReadSkillResult, error) {
	baseURL, err := c.baseURLResolver(ctx)
	if err != nil {
		return ReadSkillResult{}, fmt.Errorf("promptmanager: resolve URL: %w", err)
	}

	reqBody := readRequest{
		Identifiers:  []string{skillID},
		Variables:    variables,
		Output:       "combined",
		WithScope:    withScope,
		ExperimentID: experimentID,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return ReadSkillResult{}, fmt.Errorf("promptmanager: marshal request: %w", err)
	}

	reqURL := baseURL + "/api/v1/skills/read"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return ReadSkillResult{}, fmt.Errorf("promptmanager: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ReadSkillResult{}, fmt.Errorf("promptmanager: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return ReadSkillResult{}, fmt.Errorf("promptmanager: status %d: %s", resp.StatusCode, string(respBody))
	}

	var readResp readResponse
	if err := json.NewDecoder(resp.Body).Decode(&readResp); err != nil {
		return ReadSkillResult{}, fmt.Errorf("promptmanager: decode response: %w", err)
	}

	return ReadSkillResult{
		Content:   readResp.Combined,
		VariantID: readResp.SelectedVariantID,
	}, nil
}

// RecordExperimentOutcome posts an outcome to prompt-manager for an experiment.
func (c *HTTPClient) RecordExperimentOutcome(ctx context.Context, experimentID string, outcome ExperimentOutcomeRequest) error {
	baseURL, err := c.baseURLResolver(ctx)
	if err != nil {
		return fmt.Errorf("promptmanager: resolve URL: %w", err)
	}

	body, err := json.Marshal(outcome)
	if err != nil {
		return fmt.Errorf("promptmanager: marshal outcome: %w", err)
	}

	reqURL := baseURL + "/api/v1/experiments/" + url.PathEscape(strings.TrimSpace(experimentID)) + "/outcomes"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("promptmanager: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("promptmanager: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("promptmanager: status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// ListExperimentOutcomes fetches raw outcomes for an experiment from prompt-manager.
func (c *HTTPClient) ListExperimentOutcomes(ctx context.Context, experimentID string) ([]json.RawMessage, error) {
	baseURL, err := c.baseURLResolver(ctx)
	if err != nil {
		return nil, fmt.Errorf("promptmanager: resolve URL: %w", err)
	}

	reqURL := baseURL + "/api/v1/experiments/" + url.PathEscape(strings.TrimSpace(experimentID)) + "/outcomes"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("promptmanager: create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("promptmanager: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("promptmanager: status %d: %s", resp.StatusCode, string(respBody))
	}

	var result []json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("promptmanager: decode response: %w", err)
	}
	return result, nil
}

// ListSkills fetches prompt-manager skill metadata with optional tag filtering.
func (c *HTTPClient) ListSkills(ctx context.Context, tag string) ([]PromptSkill, error) {
	baseURL, err := c.baseURLResolver(ctx)
	if err != nil {
		return nil, fmt.Errorf("promptmanager: resolve URL: %w", err)
	}

	query := ""
	trimmedTag := strings.TrimSpace(tag)
	if trimmedTag != "" {
		query = "?tag=" + url.QueryEscape(trimmedTag)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/v1/skills"+query, nil)
	if err != nil {
		return nil, fmt.Errorf("promptmanager: create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("promptmanager: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("promptmanager: status %d: %s", resp.StatusCode, string(respBody))
	}

	var result []PromptSkill
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("promptmanager: decode response: %w", err)
	}
	return result, nil
}

// GetSkill fetches full details for a single skill.
func (c *HTTPClient) GetSkill(ctx context.Context, skillID string) (PromptSkill, error) {
	baseURL, err := c.baseURLResolver(ctx)
	if err != nil {
		return PromptSkill{}, fmt.Errorf("promptmanager: resolve URL: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		baseURL+"/api/v1/skills/"+url.PathEscape(strings.TrimSpace(skillID)),
		nil,
	)
	if err != nil {
		return PromptSkill{}, fmt.Errorf("promptmanager: create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return PromptSkill{}, fmt.Errorf("promptmanager: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return PromptSkill{}, fmt.Errorf("promptmanager: status %d: %s", resp.StatusCode, string(respBody))
	}

	var result PromptSkill
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return PromptSkill{}, fmt.Errorf("promptmanager: decode response: %w", err)
	}
	return result, nil
}

// UpdateSkill applies a partial update to a skill and returns the updated record.
func (c *HTTPClient) UpdateSkill(ctx context.Context, skillID string, patch PromptSkillUpdate) (PromptSkill, error) {
	baseURL, err := c.baseURLResolver(ctx)
	if err != nil {
		return PromptSkill{}, fmt.Errorf("promptmanager: resolve URL: %w", err)
	}

	body, err := json.Marshal(patch)
	if err != nil {
		return PromptSkill{}, fmt.Errorf("promptmanager: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPut,
		baseURL+"/api/v1/skills/"+url.PathEscape(strings.TrimSpace(skillID)),
		bytes.NewReader(body),
	)
	if err != nil {
		return PromptSkill{}, fmt.Errorf("promptmanager: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return PromptSkill{}, fmt.Errorf("promptmanager: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return PromptSkill{}, fmt.Errorf("promptmanager: status %d: %s", resp.StatusCode, string(respBody))
	}

	var result PromptSkill
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return PromptSkill{}, fmt.Errorf("promptmanager: decode response: %w", err)
	}
	return result, nil
}

// GetSkillVersions returns stored version history for a skill.
func (c *HTTPClient) GetSkillVersions(ctx context.Context, skillID string) (PromptSkillVersions, error) {
	baseURL, err := c.baseURLResolver(ctx)
	if err != nil {
		return PromptSkillVersions{}, fmt.Errorf("promptmanager: resolve URL: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		baseURL+"/api/v1/skills/"+url.PathEscape(strings.TrimSpace(skillID))+"/versions",
		nil,
	)
	if err != nil {
		return PromptSkillVersions{}, fmt.Errorf("promptmanager: create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return PromptSkillVersions{}, fmt.Errorf("promptmanager: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return PromptSkillVersions{}, fmt.Errorf("promptmanager: status %d: %s", resp.StatusCode, string(respBody))
	}

	var result PromptSkillVersions
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return PromptSkillVersions{}, fmt.Errorf("promptmanager: decode response: %w", err)
	}
	return result, nil
}

// RevertSkillVersion reverts a skill to a previous version in prompt-manager.
func (c *HTTPClient) RevertSkillVersion(ctx context.Context, skillID string, version int) error {
	baseURL, err := c.baseURLResolver(ctx)
	if err != nil {
		return fmt.Errorf("promptmanager: resolve URL: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		baseURL+"/api/v1/skills/"+url.PathEscape(strings.TrimSpace(skillID))+"/revert/"+strconv.Itoa(version),
		nil,
	)
	if err != nil {
		return fmt.Errorf("promptmanager: create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("promptmanager: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("promptmanager: status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// MockClient implements Client, AdminClient, and ExperimentClient for testing.
type MockClient struct {
	Result             string
	Err                error
	Skills             []PromptSkill
	Skill              PromptSkill
	SkillByID          map[string]PromptSkill
	Versions           PromptSkillVersions
	UpdatedSkill       PromptSkill
	ReadSkillResult    ReadSkillResult
	ReadSkillResultErr error
	RecordedOutcomes   []ExperimentOutcomeRequest
	RecordOutcomeErr   error
	ExperimentOutcomes []json.RawMessage
	ListOutcomesErr    error
}

// ReadSkill returns the mock result.
func (m *MockClient) ReadSkill(_ context.Context, _ string, _ map[string]string, _ bool) (string, error) {
	return m.Result, m.Err
}

// ListSkills returns mock skills.
func (m *MockClient) ListSkills(_ context.Context, _ string) ([]PromptSkill, error) {
	return m.Skills, m.Err
}

// GetSkill returns mock skill details.
func (m *MockClient) GetSkill(_ context.Context, skillID string) (PromptSkill, error) {
	if m.SkillByID != nil {
		if skill, ok := m.SkillByID[skillID]; ok {
			return skill, m.Err
		}
	}
	return m.Skill, m.Err
}

// UpdateSkill returns a mock updated skill.
func (m *MockClient) UpdateSkill(_ context.Context, _ string, _ PromptSkillUpdate) (PromptSkill, error) {
	if m.UpdatedSkill.ID != "" {
		return m.UpdatedSkill, m.Err
	}
	return m.Skill, m.Err
}

// GetSkillVersions returns mock version history.
func (m *MockClient) GetSkillVersions(_ context.Context, _ string) (PromptSkillVersions, error) {
	return m.Versions, m.Err
}

// RevertSkillVersion applies a no-op mock revert.
func (m *MockClient) RevertSkillVersion(_ context.Context, _ string, _ int) error {
	return m.Err
}

// ReadSkillWithExperiment returns the mock ReadSkillResult.
func (m *MockClient) ReadSkillWithExperiment(_ context.Context, _ string, _ map[string]string, _ bool, _ string) (ReadSkillResult, error) {
	if m.ReadSkillResultErr != nil {
		return ReadSkillResult{}, m.ReadSkillResultErr
	}
	if m.ReadSkillResult.Content != "" || m.ReadSkillResult.VariantID != "" {
		return m.ReadSkillResult, nil
	}
	return ReadSkillResult{Content: m.Result}, m.Err
}

// RecordExperimentOutcome records a mock outcome.
func (m *MockClient) RecordExperimentOutcome(_ context.Context, _ string, outcome ExperimentOutcomeRequest) error {
	m.RecordedOutcomes = append(m.RecordedOutcomes, outcome)
	return m.RecordOutcomeErr
}

// ListExperimentOutcomes returns mock outcomes.
func (m *MockClient) ListExperimentOutcomes(_ context.Context, _ string) ([]json.RawMessage, error) {
	return m.ExperimentOutcomes, m.ListOutcomesErr
}

// resolvePromptManagerBaseURL resolves prompt-manager using api-core discovery.
func resolvePromptManagerBaseURL(ctx context.Context) (string, error) {
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, "prompt-manager")
	if err != nil {
		return "", fmt.Errorf("resolve prompt-manager: %w", err)
	}
	return baseURL, nil
}
