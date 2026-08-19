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
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	experimentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/experiments"
	experimentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/experiments/experiments_v1connect"
	skillsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills"
	skillsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills/skills_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
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
	HonestyFlags       []string
	RecurrenceEvidence string
}

// FrictionIntakeClient is an optional prompt-manager capability. Keeping it
// separate from Client preserves prompt reading as the required dependency for
// workflow execution.
type FrictionIntakeClient interface {
	PublishFriction(context.Context, FrictionReport) (string, error)
}

// ScenarioQABug is the typed, complete payload used by scheduled safeguards.
// The deterministic idempotency key is also reflected in the title so the
// prompt-manager bug inbox remains deduplicated if the local state is lost.
type ScenarioQABug struct {
	Title          string            `json:"title"`
	SignalType     string            `json:"signal_type"`
	Severity       string            `json:"severity"`
	Repro          []string          `json:"repro"`
	Expected       string            `json:"expected"`
	Actual         string            `json:"actual"`
	Description    string            `json:"description"`
	Context        map[string]string `json:"context"`
	HonestyFlags   []string          `json:"honesty_flags"`
	IdempotencyKey string            `json:"idempotency_key"`
}

type ScenarioQAReporter interface {
	PublishScenarioQABug(context.Context, ScenarioQABug) error
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

// ReadSkill fetches a single skill from prompt-manager with variable substitution.
func (c *HTTPClient) ReadSkill(ctx context.Context, skillID string, variables map[string]string, withScope bool) (string, error) {
	resp, err := c.readSkills(ctx, &skillsv1.ReadSkillsRequest{Identifiers: []string{skillID}, Variables: variables, Output: "combined", WithScope: withScope})
	if err != nil {
		return "", err
	}
	return resp.GetCombined(), nil
}

// ReadSkillSource resolves a skill and returns its content alongside the
// immutable revision metadata used to pin a promptRef into a workflow revision.
func (c *HTTPClient) ReadSkillSource(ctx context.Context, skillID, experimentID string, variables map[string]string, withScope bool) (SkillSourceSnapshot, error) {
	reqBody := &skillsv1.ReadSkillsRequest{Identifiers: []string{skillID}, Variables: variables, Output: "both", WithScope: withScope}
	// An empty experimentID pins the read so a running experiment never silently
	// arms this resolution; a non-empty one deliberately selects that experiment.
	if experimentID == "" {
		reqBody.VariantPolicy = "pinned"
	} else {
		reqBody.ExperimentId = experimentID
	}
	readResp, err := c.readSkills(ctx, reqBody)
	if err != nil {
		return SkillSourceSnapshot{}, err
	}
	if len(readResp.GetSkills()) != 1 || strings.TrimSpace(readResp.GetCombined()) == "" || strings.TrimSpace(readResp.GetCombinedHash()) == "" {
		return SkillSourceSnapshot{}, fmt.Errorf("promptmanager: source response for %q is incomplete: %w", skillID, ErrSkillSourceMissing)
	}
	skill := readResp.GetSkills()[0]
	variant := strings.TrimSpace(readResp.GetSelectedVariantId())
	if variant == "" {
		variant = "control"
	}
	return SkillSourceSnapshot{
		SkillID:      skill.GetId(),
		Revision:     int(skill.GetRevision()),
		VariantID:    variant,
		ExperimentID: strings.TrimSpace(readResp.GetExperimentId()),
		Content:      readResp.GetCombined(),
		ContentHash:  readResp.GetCombinedHash(),
	}, nil
}

func (c *HTTPClient) AssignExperimentPrompt(ctx context.Context, assignment AssignmentRequest) (SkillSourceSnapshot, error) {
	if assignment.ExperimentID == "" || assignment.SkillID == "" || assignment.ExecutionID == "" || assignment.NodeID == "" || assignment.AttemptKey == "" || assignment.IdempotencyKey == "" {
		return SkillSourceSnapshot{}, fmt.Errorf("promptmanager: complete workflow assignment identity is required")
	}
	client, err := c.experimentsClient(ctx)
	if err != nil {
		return SkillSourceSnapshot{}, err
	}
	resp, err := client.AssignExperiment(ctx, connect.NewRequest(&experimentsv1.AssignExperimentRequest{
		ExperimentId: assignment.ExperimentID, ExecutionId: assignment.ExecutionID,
		NodeId: assignment.NodeID, AttemptKey: assignment.AttemptKey,
		IdempotencyKey: assignment.IdempotencyKey, Variables: assignment.Variables, WithScope: assignment.WithScope,
	}))
	if err != nil {
		return SkillSourceSnapshot{}, fmt.Errorf("promptmanager: assign experiment: %w", err)
	}
	assigned := resp.Msg
	if assigned.GetExperimentId() != assignment.ExperimentID || assigned.GetSkillId() != assignment.SkillID || assigned.GetVariantId() == "" || strings.TrimSpace(assigned.GetContent()) == "" || assigned.GetContentHash() == "" {
		return SkillSourceSnapshot{}, fmt.Errorf("promptmanager: assignment response is incomplete or mismatched")
	}
	return SkillSourceSnapshot{SkillID: assigned.GetSkillId(), ExperimentID: assigned.GetExperimentId(), VariantID: assigned.GetVariantId(), Content: assigned.GetContent(), ContentHash: assigned.GetContentHash()}, nil
}

// RecordExperimentOutcome posts an outcome to a running experiment, attributing
// a served variant to its run result.
func (c *HTTPClient) RecordExperimentOutcome(ctx context.Context, experimentID string, outcome ExperimentOutcome) error {
	if strings.TrimSpace(experimentID) == "" {
		return fmt.Errorf("promptmanager: experimentID is required")
	}
	client, err := c.experimentsClient(ctx)
	if err != nil {
		return err
	}
	data, err := rawJSONValue(outcome.Data)
	if err != nil {
		return fmt.Errorf("promptmanager: encode outcome data: %w", err)
	}
	var controlled *structpb.Struct
	if outcome.Controlled != nil {
		raw, marshalErr := json.Marshal(outcome.Controlled)
		if marshalErr != nil {
			return fmt.Errorf("promptmanager: encode controlled outcome: %w", marshalErr)
		}
		var object map[string]any
		if unmarshalErr := json.Unmarshal(raw, &object); unmarshalErr != nil {
			return fmt.Errorf("promptmanager: encode controlled outcome: %w", unmarshalErr)
		}
		controlled, err = structpb.NewStruct(object)
		if err != nil {
			return fmt.Errorf("promptmanager: encode controlled outcome: %w", err)
		}
	}
	_, err = client.RecordOutcome(ctx, connect.NewRequest(&experimentsv1.RecordOutcomeRequest{
		ExperimentId: experimentID, IdempotencyKey: outcome.IdempotencyKey,
		VariantId: outcome.VariantID, Source: outcome.Source, Data: data, Controlled: controlled,
	}))
	if err != nil {
		return fmt.Errorf("promptmanager: record experiment outcome: %w", err)
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

// PublishScenarioQABug files a complete typed report in scenario-qa's
// canonical bug inbox. It intentionally does not attempt to repair partial
// reports: scheduled safeguard findings have enough evidence to publish.
func (c *HTTPClient) PublishScenarioQABug(ctx context.Context, report ScenarioQABug) error {
	baseURL, err := c.baseURLResolver(ctx)
	if err != nil {
		return fmt.Errorf("promptmanager: resolve URL: %w", err)
	}
	body, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("promptmanager: marshal scenario-qa bug: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/teams/scenario-qa/bugs/capture", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("promptmanager: create scenario-qa request: %w", err)
	}
	attribution, err := json.Marshal(map[string]any{"kind": "investigation", "run_id": report.IdempotencyKey, "spawn_origin": "investigation"})
	if err != nil {
		return fmt.Errorf("promptmanager: marshal scenario-qa attribution: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Vrooli-Attribution", base64.StdEncoding.EncodeToString(attribution))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("promptmanager: scenario-qa request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		payload, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("promptmanager: scenario-qa status %d: %s", resp.StatusCode, string(payload))
	}
	return nil
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
	flags := append([]string(nil), report.HonestyFlags...)
	if len(flags) == 0 {
		flags = []string{"auto-generated"}
	}
	for i := range flags {
		flags[i] = strings.TrimSpace(flags[i])
	}
	flagText := "[" + strings.Join(flags, ", ") + "]"
	recurrence := strings.TrimSpace(report.RecurrenceEvidence)
	if recurrence == "" {
		recurrence = "not captured"
	}
	return fmt.Sprintf("---\nseverity: %s\nscope: %s\nreporter: agent-manager\nreporter_team: meta-optimization\nobserved_at: %s\ncontext:\n  scenario: agent-manager\n  skill: null\n  member: null\n  command: null\n  doc: null\n  task: %s\nexpected: %s\nactual: %s\ndescription: |\n  An Agent Manager investigation produced this durable finding.\n  Fingerprint: %s.\n  Evidence: %s\nrecurrence_evidence: %s\nhonesty_flags: %s\n---\n\nWhat happened\n\n%s\n", severity, scope, time.Now().UTC().Format("2006-01-02"), report.InvestigationRunID, yamlLine(recommendation), yamlLine(evidence), report.Fingerprint, indentLine(evidence), yamlLine(recurrence), flagText, recommendation)
}

func yamlLine(value string) string { return strconv.Quote(strings.Join(strings.Fields(value), " ")) }

func indentLine(value string) string {
	return strings.ReplaceAll(strings.TrimSpace(value), "\n", "\n  ")
}

// ListSkills fetches prompt-manager skill metadata with optional tag filtering.
func (c *HTTPClient) ListSkills(ctx context.Context, tag string) ([]PromptSkill, error) {
	client, err := c.skillsClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.ListSkills(ctx, connect.NewRequest(&skillsv1.ListSkillsRequest{Tag: strings.TrimSpace(tag)}))
	if err != nil {
		return nil, fmt.Errorf("promptmanager: list skills: %w", err)
	}
	var envelope struct {
		Skills []PromptSkill `json:"skills"`
	}
	if err := protoConvert(resp.Msg, &envelope); err != nil {
		return nil, err
	}
	return envelope.Skills, nil
}

// GetSkill fetches full details for a single skill.
func (c *HTTPClient) GetSkill(ctx context.Context, skillID string) (PromptSkill, error) {
	client, err := c.skillsClient(ctx)
	if err != nil {
		return PromptSkill{}, err
	}
	resp, err := client.GetSkill(ctx, connect.NewRequest(&skillsv1.GetSkillRequest{Id: strings.TrimSpace(skillID)}))
	if err != nil {
		return PromptSkill{}, fmt.Errorf("promptmanager: get skill: %w", err)
	}
	var result PromptSkill
	if err := protoConvert(resp.Msg.GetSkill(), &result); err != nil {
		return PromptSkill{}, err
	}
	return result, nil
}

// UpdateSkill applies a partial update to a skill and returns the updated record.
func (c *HTTPClient) UpdateSkill(ctx context.Context, skillID string, patch PromptSkillUpdate) (PromptSkill, error) {
	client, err := c.skillsClient(ctx)
	if err != nil {
		return PromptSkill{}, err
	}
	req := &skillsv1.UpdateSkillRequest{Id: strings.TrimSpace(skillID), Name: patch.Name, Description: patch.Description, Content: patch.Content, DefaultScope: patch.DefaultScope, Draft: patch.Draft, Folder: patch.Folder}
	resp, err := client.UpdateSkill(ctx, connect.NewRequest(req))
	if err != nil {
		return PromptSkill{}, fmt.Errorf("promptmanager: update skill: %w", err)
	}
	var result PromptSkill
	if err := protoConvert(resp.Msg.GetSkill(), &result); err != nil {
		return PromptSkill{}, err
	}
	return result, nil
}

// GetSkillVersions returns stored version history for a skill.
func (c *HTTPClient) GetSkillVersions(ctx context.Context, skillID string) (PromptSkillVersions, error) {
	client, err := c.skillsClient(ctx)
	if err != nil {
		return PromptSkillVersions{}, err
	}
	resp, err := client.ListSkillVersions(ctx, connect.NewRequest(&skillsv1.ListSkillVersionsRequest{Id: strings.TrimSpace(skillID)}))
	if err != nil {
		return PromptSkillVersions{}, fmt.Errorf("promptmanager: list versions: %w", err)
	}
	var result PromptSkillVersions
	if err := protoConvert(resp.Msg, &result); err != nil {
		return PromptSkillVersions{}, err
	}
	return result, nil
}

// RevertSkillVersion reverts a skill to a previous version in prompt-manager.
func (c *HTTPClient) RevertSkillVersion(ctx context.Context, skillID string, version int) error {
	client, err := c.skillsClient(ctx)
	if err != nil {
		return err
	}
	_, err = client.RevertSkill(ctx, connect.NewRequest(&skillsv1.RevertSkillRequest{Id: strings.TrimSpace(skillID), Version: int32(version)}))
	if err != nil {
		return fmt.Errorf("promptmanager: revert skill: %w", err)
	}
	return nil
}

func (c *HTTPClient) skillsClient(ctx context.Context) (skillsconnect.SkillsServiceClient, error) {
	baseURL, err := c.baseURLResolver(ctx)
	if err != nil {
		return nil, fmt.Errorf("promptmanager: resolve URL: %w", err)
	}
	return skillsconnect.NewSkillsServiceClient(c.httpClient, strings.TrimRight(baseURL, "/")), nil
}

func (c *HTTPClient) experimentsClient(ctx context.Context) (experimentsconnect.ExperimentsServiceClient, error) {
	baseURL, err := c.baseURLResolver(ctx)
	if err != nil {
		return nil, fmt.Errorf("promptmanager: resolve URL: %w", err)
	}
	return experimentsconnect.NewExperimentsServiceClient(c.httpClient, strings.TrimRight(baseURL, "/")), nil
}

func rawJSONValue(raw json.RawMessage) (*structpb.Value, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return structpb.NewValue(value)
}

func (c *HTTPClient) readSkills(ctx context.Context, req *skillsv1.ReadSkillsRequest) (*skillsv1.ReadSkillsResponse, error) {
	client, err := c.skillsClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.ReadSkills(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("promptmanager: read skills: %w", err)
	}
	return resp.Msg, nil
}

func protoConvert(source proto.Message, target any) error {
	raw, err := protojson.Marshal(source)
	if err != nil {
		return fmt.Errorf("promptmanager: encode protobuf response: %w", err)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("promptmanager: decode protobuf response: %w", err)
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
