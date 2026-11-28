// Package promptmanager provides a client for reading skill prompts from prompt-manager.
//
// DOC: docs/internal/SEAMS.md
package promptmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	skillsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills"
	skillsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills/skills_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type SkillSourceSnapshot struct {
	SkillID           string
	Revision          int
	SelectedVariantID string
	Content           string
	ContentHash       string
	TemplateVariables []string
}

// Client reads skill prompts from prompt-manager.
type Client interface {
	ReadSkill(ctx context.Context, skillID string, variables map[string]string, withScope bool) (string, error)
}

// SourceClient exposes immutable source metadata for consumers that must pin
// prompt interpretation across retries and service restarts.
type SourceClient interface {
	ReadSkillSource(ctx context.Context, skillID string, expectedVariables []string) (SkillSourceSnapshot, error)
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

// ReadSkill fetches a single skill from prompt-manager with variable substitution.
func (c *HTTPClient) ReadSkill(ctx context.Context, skillID string, variables map[string]string, withScope bool) (string, error) {
	resp, err := c.readSkills(ctx, &skillsv1.ReadSkillsRequest{Identifiers: []string{skillID}, Variables: variables, Output: "combined", WithScope: withScope})
	if err != nil {
		return "", err
	}
	return resp.GetCombined(), nil
}

func (c *HTTPClient) ReadSkillSource(ctx context.Context, skillID string, _ []string) (SkillSourceSnapshot, error) {
	readResp, err := c.readSkills(ctx, &skillsv1.ReadSkillsRequest{Identifiers: []string{skillID}, Output: "both"})
	if err != nil {
		return SkillSourceSnapshot{}, err
	}
	if len(readResp.GetSkills()) != 1 || strings.TrimSpace(readResp.GetCombined()) == "" || strings.TrimSpace(readResp.GetCombinedHash()) == "" {
		return SkillSourceSnapshot{}, fmt.Errorf("promptmanager: source response for %q is incomplete", skillID)
	}
	skill := readResp.GetSkills()[0]
	variables := make([]string, 0, len(skill.GetVariables()))
	for _, variable := range skill.GetVariables() {
		if name := strings.TrimSpace(variable.GetName()); name != "" {
			variables = append(variables, name)
		}
	}
	sort.Strings(variables)
	variant := strings.TrimSpace(readResp.GetSelectedVariantId())
	if variant == "" {
		variant = "control"
	}
	return SkillSourceSnapshot{
		SkillID: skill.GetId(), Revision: int(skill.GetRevision()), SelectedVariantID: variant,
		Content: readResp.GetCombined(), ContentHash: readResp.GetCombinedHash(), TemplateVariables: variables,
	}, nil
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

// MockClient implements Client and AdminClient for testing.
type MockClient struct {
	Result       string
	Err          error
	Skills       []PromptSkill
	Skill        PromptSkill
	SkillByID    map[string]PromptSkill
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

// resolvePromptManagerBaseURL resolves prompt-manager using api-core discovery.
func resolvePromptManagerBaseURL(ctx context.Context) (string, error) {
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, "prompt-manager")
	if err != nil {
		return "", fmt.Errorf("resolve prompt-manager: %w", err)
	}
	return baseURL, nil
}
