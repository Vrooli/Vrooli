// Package promptmanager provides a client for reading skill prompts from prompt-manager.
//
// DOC: docs/internal/SEAMS.md
package promptmanager

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// ErrSkillSourceMissing marks a definitive answer from prompt-manager: the
// request succeeded but the skill no longer resolves to usable source content
// (deleted or gutted skill). Callers can distinguish this from transport
// failures, where no comparison was possible at all.
var ErrSkillSourceMissing = errors.New("skill source missing")

// FrictionReport is the bounded, already-classified observation that Agent
// Manager publishes to meta-optimization's friction intake.
type FrictionReport struct {
	InvestigationRunID string
	Fingerprint        string
	Category           string
	Severity           string
	Occurrences        int
	Recommendation     string
	Evidence           string
	TargetPath         string
}

// FrictionIntakeClient is an optional prompt-manager capability. Keeping it
// separate from Client preserves prompt reading as the required dependency for
// workflow execution.
type FrictionIntakeClient interface {
	PublishFriction(context.Context, FrictionReport) (string, error)
}

// Client reads skill prompts from prompt-manager.
type Client interface {
	ReadSkill(ctx context.Context, skillID string, variables map[string]string, withScope bool) (string, error)
}

// SkillSourceSnapshot is the immutable provenance of one resolved skill read: the
// content plus the metadata that pins which skill revision produced it, so a
// promptRef resolution can be recorded in a digest-deterministic way.
type SkillSourceSnapshot struct {
	SkillID      string
	Revision     int
	VariantID    string
	ExperimentID string
	Content      string
	ContentHash  string
}

// SourceClient exposes immutable source metadata for consumers that must pin
// prompt interpretation into a durable revision. A non-empty experimentID arms
// the read against that experiment; an empty experimentID pins the read so a
// running experiment never silently varies the resolved prompt.
type SourceClient interface {
	ReadSkillSource(ctx context.Context, skillID, experimentID string, variables map[string]string, withScope bool) (SkillSourceSnapshot, error)
}

// AssignmentRequest identifies the exact workflow dispatch that must receive a
// stable experimental treatment.
type AssignmentRequest struct {
	ExperimentID   string
	SkillID        string
	ExecutionID    string
	NodeID         string
	AttemptKey     string
	IdempotencyKey string
	Variables      map[string]string
	WithScope      bool
}

type AssignmentClient interface {
	AssignExperimentPrompt(ctx context.Context, request AssignmentRequest) (SkillSourceSnapshot, error)
}

// ExperimentOutcome is the attribution payload posted back to prompt-manager
// when a served variant reaches a run outcome.
type ExperimentOutcome struct {
	IdempotencyKey string                       `json:"idempotencyKey,omitempty"`
	VariantID      string                       `json:"variantId"`
	Source         string                       `json:"source"`
	Data           json.RawMessage              `json:"data,omitempty"`
	Controlled     *ControlledExperimentOutcome `json:"controlled,omitempty"`
}

type ControlledExperimentOutcome struct {
	AssignmentID         string `json:"assignmentId"`
	ExecutionID          string `json:"executionId"`
	EvaluatorAttemptID   string `json:"evaluatorAttemptId"`
	EvaluatorRunID       string `json:"evaluatorRunId"`
	Verdict              string `json:"verdict,omitempty"`
	Success              *bool  `json:"success,omitempty"`
	OutcomeStatus        string `json:"outcomeStatus"`
	RubricHash           string `json:"rubricHash"`
	EvaluatorPromptHash  string `json:"evaluatorPromptHash"`
	StructuredSchemaHash string `json:"structuredSchemaHash"`
	EvidenceHash         string `json:"evidenceHash,omitempty"`
}

// OutcomeClient reports experiment outcomes to prompt-manager, attributing a
// served variant to its run result.
type OutcomeClient interface {
	RecordExperimentOutcome(ctx context.Context, experimentID string, outcome ExperimentOutcome) error
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

// AdminClient exposes management operations used by the investigation settings proxy.
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
	Identifiers   []string          `json:"identifiers"`
	Variables     map[string]string `json:"variables,omitempty"`
	Output        string            `json:"output"`
	WithScope     bool              `json:"withScope,omitempty"`
	ExperimentID  string            `json:"experimentId,omitempty"`
	VariantPolicy string            `json:"variantPolicy,omitempty"`
	Source        string            `json:"source,omitempty"`
}

// readResponse is the response from the skill read endpoint.
type readResponse struct {
	Combined          string `json:"combined"`
	CombinedHash      string `json:"combinedHash,omitempty"`
	SelectedVariantID string `json:"selectedVariantId,omitempty"`
	ExperimentID      string `json:"experimentId,omitempty"`
	Skills            []struct {
		ID          string `json:"id"`
		Revision    int    `json:"revision,omitempty"`
		ContentHash string `json:"contentHash,omitempty"`
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

	reqURL := baseURL + "/api/v1/skills/read"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("promptmanager: create request: %w", err)
	}
	// Deliberately no X-Agent-Identity-Token: this is agent-manager's own
	// service-side read, not an agent run's. Controlled-lane attribution rides
	// on dispatch assignments; per-run exposure receipts belong to the
	// observational lane, whose reads go through the CLI and carry the token.
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

// ReadSkillSource resolves a skill and returns its content alongside the
// immutable revision metadata used to pin a promptRef into a workflow revision.
func (c *HTTPClient) ReadSkillSource(ctx context.Context, skillID, experimentID string, variables map[string]string, withScope bool) (SkillSourceSnapshot, error) {
	baseURL, err := c.baseURLResolver(ctx)
	if err != nil {
		return SkillSourceSnapshot{}, fmt.Errorf("promptmanager: resolve URL: %w", err)
	}
	reqBody := readRequest{
		Identifiers: []string{skillID},
		Variables:   variables,
		Output:      "both",
		WithScope:   withScope,
	}
	// An empty experimentID pins the read so a running experiment never silently
	// arms this resolution; a non-empty one deliberately selects that experiment.
	if experimentID == "" {
		reqBody.VariantPolicy = "pinned"
	} else {
		reqBody.ExperimentID = experimentID
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
		return SkillSourceSnapshot{}, fmt.Errorf("promptmanager: source response for %q is incomplete: %w", skillID, ErrSkillSourceMissing)
	}
	skill := readResp.Skills[0]
	variant := strings.TrimSpace(readResp.SelectedVariantID)
	if variant == "" {
		variant = "control"
	}
	return SkillSourceSnapshot{
		SkillID:      skill.ID,
		Revision:     skill.Revision,
		VariantID:    variant,
		ExperimentID: strings.TrimSpace(readResp.ExperimentID),
		Content:      readResp.Combined,
		ContentHash:  readResp.CombinedHash,
	}, nil
}

func (c *HTTPClient) AssignExperimentPrompt(ctx context.Context, assignment AssignmentRequest) (SkillSourceSnapshot, error) {
	if assignment.ExperimentID == "" || assignment.SkillID == "" || assignment.ExecutionID == "" || assignment.NodeID == "" || assignment.AttemptKey == "" || assignment.IdempotencyKey == "" {
		return SkillSourceSnapshot{}, fmt.Errorf("promptmanager: complete workflow assignment identity is required")
	}
	baseURL, err := c.baseURLResolver(ctx)
	if err != nil {
		return SkillSourceSnapshot{}, fmt.Errorf("promptmanager: resolve URL: %w", err)
	}
	body, err := json.Marshal(map[string]any{"executionId": assignment.ExecutionID, "nodeId": assignment.NodeID, "attemptKey": assignment.AttemptKey, "idempotencyKey": assignment.IdempotencyKey, "variables": assignment.Variables, "withScope": assignment.WithScope})
	if err != nil {
		return SkillSourceSnapshot{}, fmt.Errorf("promptmanager: marshal assignment: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/experiments/"+url.PathEscape(assignment.ExperimentID)+"/assignments", bytes.NewReader(body))
	if err != nil {
		return SkillSourceSnapshot{}, fmt.Errorf("promptmanager: create assignment request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return SkillSourceSnapshot{}, fmt.Errorf("promptmanager: assignment request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		payload, _ := io.ReadAll(resp.Body)
		return SkillSourceSnapshot{}, fmt.Errorf("promptmanager: assignment status %d: %s", resp.StatusCode, string(payload))
	}
	var decoded struct {
		ExperimentID string `json:"experimentId"`
		SkillID      string `json:"skillId"`
		VariantID    string `json:"variantId"`
		Content      string `json:"content"`
		ContentHash  string `json:"contentHash"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return SkillSourceSnapshot{}, fmt.Errorf("promptmanager: decode assignment: %w", err)
	}
	if decoded.ExperimentID != assignment.ExperimentID || decoded.SkillID != assignment.SkillID || decoded.VariantID == "" || strings.TrimSpace(decoded.Content) == "" || decoded.ContentHash == "" {
		return SkillSourceSnapshot{}, fmt.Errorf("promptmanager: assignment response is incomplete or mismatched")
	}
	return SkillSourceSnapshot{SkillID: decoded.SkillID, ExperimentID: decoded.ExperimentID, VariantID: decoded.VariantID, Content: decoded.Content, ContentHash: decoded.ContentHash}, nil
}

// RecordExperimentOutcome posts an outcome to a running experiment, attributing
// a served variant to its run result.
func (c *HTTPClient) RecordExperimentOutcome(ctx context.Context, experimentID string, outcome ExperimentOutcome) error {
	if strings.TrimSpace(experimentID) == "" {
		return fmt.Errorf("promptmanager: experimentID is required")
	}
	baseURL, err := c.baseURLResolver(ctx)
	if err != nil {
		return fmt.Errorf("promptmanager: resolve URL: %w", err)
	}
	body, err := json.Marshal(outcome)
	if err != nil {
		return fmt.Errorf("promptmanager: marshal outcome: %w", err)
	}
	endpoint := baseURL + "/api/v1/experiments/" + url.PathEscape(experimentID) + "/outcomes"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("promptmanager: create outcome request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("promptmanager: outcome request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("promptmanager: outcome status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// PublishFriction writes one investigation observation to the existing
// meta-optimization friction inbox endpoint. The stored topic is used by the
// caller as its durable idempotency marker.
func (c *HTTPClient) PublishFriction(ctx context.Context, report FrictionReport) (string, error) {
	if strings.TrimSpace(report.InvestigationRunID) == "" || strings.TrimSpace(report.Fingerprint) == "" {
		return "", fmt.Errorf("promptmanager: investigation run ID and finding fingerprint are required")
	}
	baseURL, err := c.baseURLResolver(ctx)
	if err != nil {
		return "", fmt.Errorf("promptmanager: resolve URL: %w", err)
	}
	scope := frictionScope(report.Category)
	topic := "friction-inbox/" + scope + "/agent-manager-finding-" + shortFingerprint(report.Fingerprint)
	severity := "one-off"
	if report.Occurrences > 1 || strings.EqualFold(strings.TrimSpace(report.Severity), "recurring") {
		severity = "recurring"
	}
	body, err := json.Marshal(struct {
		Topic      string `json:"topic"`
		Content    string `json:"content"`
		CallerNote string `json:"caller_note"`
		Source     string `json:"source"`
	}{Topic: topic, Content: frictionContent(report, scope, severity), CallerNote: "Agent Manager investigation finding", Source: "agent-manager"})
	if err != nil {
		return "", fmt.Errorf("promptmanager: marshal friction report: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/teams/meta-optimization/knowledge", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("promptmanager: create friction request: %w", err)
	}
	attribution, err := json.Marshal(map[string]any{"kind": "investigation", "run_id": report.InvestigationRunID, "spawn_origin": "investigation"})
	if err != nil {
		return "", fmt.Errorf("promptmanager: marshal friction attribution: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Vrooli-Attribution", base64.StdEncoding.EncodeToString(attribution))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("promptmanager: friction request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		payload, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("promptmanager: friction status %d: %s", resp.StatusCode, string(payload))
	}
	return topic, nil
}

func shortFingerprint(fingerprint string) string {
	fingerprint = strings.ToLower(strings.TrimSpace(fingerprint))
	if len(fingerprint) > 12 {
		return fingerprint[:12]
	}
	return fingerprint
}

func frictionScope(category string) string {
	category = strings.ToLower(category)
	switch {
	case strings.Contains(category, "tool"):
		return "toolchain"
	case strings.Contains(category, "run"), strings.Contains(category, "process"), strings.Contains(category, "coordination"):
		return "run-execution"
	case strings.Contains(category, "prompt"), strings.Contains(category, "storage"), strings.Contains(category, "team"):
		return "prompt-team-agent-storage"
	default:
		return "unknown"
	}
}

func frictionContent(report FrictionReport, scope, severity string) string {
	recommendation := strings.TrimSpace(report.Recommendation)
	evidence := strings.TrimSpace(report.Evidence)
	if evidence == "" {
		evidence = "The structured investigation produced this recommendation."
	}
	return fmt.Sprintf("---\nseverity: %s\nscope: %s\nreporter: agent-manager\nreporter_team: meta-optimization\nobserved_at: %s\ncontext:\n  scenario: agent-manager\n  skill: null\n  member: null\n  command: null\n  doc: null\n  task: %s\nexpected: %s\nactual: %s\ndescription: |\n  An Agent Manager investigation produced this durable finding.\n  Fingerprint: %s.\n  Evidence: %s\nhonesty_flags: [auto-generated]\n---\n\nWhat happened\n\n%s\n", severity, scope, time.Now().UTC().Format("2006-01-02"), report.InvestigationRunID, yamlLine(recommendation), yamlLine(evidence), report.Fingerprint, indentLine(evidence), recommendation)
}

func yamlLine(value string) string { return strconv.Quote(strings.Join(strings.Fields(value), " ")) }

func indentLine(value string) string {
	return strings.ReplaceAll(strings.TrimSpace(value), "\n", "\n  ")
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

// MockClient implements Client for testing by consumers of this package.
type MockClient struct {
	Result       string
	Err          error
	Skills       []PromptSkill
	Skill        PromptSkill
	Versions     PromptSkillVersions
	UpdatedSkill PromptSkill
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
func (m *MockClient) GetSkill(_ context.Context, _ string) (PromptSkill, error) {
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

// resolvePromptManagerBaseURL resolves prompt-manager using api-core discovery.
func resolvePromptManagerBaseURL(ctx context.Context) (string, error) {
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, "prompt-manager")
	if err != nil {
		return "", fmt.Errorf("resolve prompt-manager: %w", err)
	}
	return baseURL, nil
}
