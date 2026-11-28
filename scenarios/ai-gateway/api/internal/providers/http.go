package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"ai-gateway/internal/catalog"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// HTTPProvider executes the provider's declared wire format using its packaged
// resource policy. Credential values stay in the credential authority.
type HTTPProvider struct {
	Declaration catalog.Provider
	Client      *http.Client
	ResolveKey  func(context.Context, string, string) (string, error)
}

type httpRole struct {
	Model         string            `json:"model"`
	Fallbacks     []string          `json:"fallbacks"`
	Capabilities  []string          `json:"required_capabilities"`
	Sampling      map[string]string `json:"sampling_support"`
	MaxTokens     int32             `json:"max_tokens"`
	ContextWindow int32             `json:"context_window"`
	Defaults      struct {
		MaxTokens int32 `json:"max_tokens"`
	} `json:"request_defaults"`
}

func (p *HTTPProvider) roles() (map[string]httpRole, error) {
	var policy struct {
		Roles map[string]httpRole `json:"roles"`
	}
	if err := json.Unmarshal(p.Declaration.Policy, &policy); err != nil {
		return nil, err
	}
	return policy.Roles, nil
}

func (p *HTTPProvider) ListRoles() Inventory {
	roles, err := p.roles()
	if err != nil {
		return Inventory{Warnings: []string{err.Error()}}
	}
	result := Inventory{}
	for name, role := range roles {
		result.Roles = append(result.Roles, Role{Provider: p.Declaration.ID, Role: name, Capabilities: role.Capabilities, Locality: p.Declaration.Locality, Status: "available"})
	}
	sort.Slice(result.Roles, func(i, j int) bool { return result.Roles[i].Role < result.Roles[j].Role })
	return result
}

func (p *HTTPProvider) ResolveRole(ctx context.Context, name string) (ResolvedRole, error) {
	roles, err := p.roles()
	if err != nil {
		return ResolvedRole{}, err
	}
	role, ok := roles[name]
	if !ok || role.Model == "" {
		return ResolvedRole{}, fmt.Errorf("provider role %q unavailable", name)
	}
	model, err := p.resolveModel(ctx, role)
	if err != nil {
		return ResolvedRole{}, err
	}
	cap := role.MaxTokens
	if cap == 0 {
		cap = role.Defaults.MaxTokens
	}
	return ResolvedRole{Provider: p.Declaration.ID, Role: name, Model: model, CanonicalModel: model, Capabilities: role.Capabilities, SamplingSupport: parseSamplingSupport(role.Sampling), MaxOutputTokens: cap, ContextWindow: role.ContextWindow, Locality: p.Declaration.Locality}, nil
}

func (p *HTTPProvider) resolveModel(ctx context.Context, role httpRole) (string, error) {
	if p.Declaration.Family != FamilyOllama {
		return role.Model, nil
	}
	data, err := p.request(ctx, http.MethodGet, "/api/tags", nil, "")
	if err != nil {
		return "", err
	}
	var tags struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(data, &tags); err != nil {
		return "", err
	}
	installed := make(map[string]bool, len(tags.Models))
	for _, model := range tags.Models {
		installed[model.Name] = true
	}
	for _, candidate := range append([]string{role.Model}, role.Fallbacks...) {
		if installed[candidate] {
			return candidate, nil
		}
	}
	return "", errors.New("no declared local model is installed")
}

func (p *HTTPProvider) Execute(ctx context.Context, req ExecutionRequest) (ExecutionResult, error) {
	if req.SpendLimit != nil {
		return ExecutionResult{}, errors.New("direct provider does not own spend-cap authorization")
	}
	if len(req.Attachments) > 0 {
		return ExecutionResult{}, errors.New("direct provider attachment format is not configured")
	}
	resolved, err := p.ResolveRole(ctx, req.Role)
	if err != nil {
		return ExecutionResult{}, err
	}
	cap := req.MaxOutputTokens
	if cap == 0 {
		cap = resolved.MaxOutputTokens
	}
	path, body, key, err := p.executionBody(ctx, req, resolved, cap)
	if err != nil {
		return ExecutionResult{}, err
	}
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}
	data, err := p.request(ctx, http.MethodPost, path, body, key)
	if err != nil {
		return ExecutionResult{}, err
	}
	return ExecutionResult{OutputText: string(data)}, nil
}

func (p *HTTPProvider) executionBody(ctx context.Context, req ExecutionRequest, resolved ResolvedRole, cap int32) (string, map[string]any, string, error) {
	switch p.Declaration.Family {
	case FamilyOllama:
		body, err := ollamaRequestBody(req, resolved.Model, cap)
		return "/api/generate", body, "", err
	case FamilyOpenAICompatible:
		body, key, err := p.openAIRequestBody(ctx, req, resolved.Model, cap)
		return "/chat/completions", body, key, err
	default:
		return "", nil, "", errors.New("unsupported provider wire format")
	}
}

func ollamaRequestBody(req ExecutionRequest, model string, cap int32) (map[string]any, error) {
	options := map[string]any{}
	if cap > 0 {
		options["num_predict"] = cap
	}
	if req.Temperature != nil {
		options["temperature"] = *req.Temperature
	}
	body := map[string]any{"model": model, "prompt": req.InputText, "stream": false, "think": false, "options": options}
	if req.SchemaJSON != "" {
		var schema any
		if err := json.Unmarshal([]byte(req.SchemaJSON), &schema); err != nil {
			return nil, err
		}
		body["format"] = schema
	}
	return body, nil
}

func (p *HTTPProvider) openAIRequestBody(ctx context.Context, req ExecutionRequest, model string, cap int32) (map[string]any, string, error) {
	if p.Declaration.CredentialIdentity == "" {
		return nil, "", errors.New("provider has no declared own-key credential")
	}
	resolve := p.ResolveKey
	if resolve == nil {
		resolve = resolveProviderKey
	}
	key, err := resolve(ctx, p.Declaration.CredentialIdentity, p.Declaration.CredentialField)
	if err != nil {
		return nil, "", err
	}
	if key == "" {
		return nil, "", errors.New("own provider key is not configured")
	}
	body := map[string]any{"model": model, "messages": []map[string]string{{"role": "user", "content": req.InputText}}, "stream": false}
	if cap > 0 {
		body["max_tokens"] = cap
	}
	if req.Temperature != nil {
		body["temperature"] = *req.Temperature
	}
	if req.SchemaJSON != "" {
		var schema any
		if err := json.Unmarshal([]byte(req.SchemaJSON), &schema); err != nil {
			return nil, "", err
		}
		body["response_format"] = map[string]any{"type": "json_schema", "json_schema": map[string]any{"name": "gateway_result", "strict": true, "schema": schema}}
	}
	return body, key, nil
}

func resolveProviderKey(_ context.Context, id, field string) (string, error) {
	a, err := credentialauthority.Default()
	if err != nil {
		return "", err
	}
	identity, err := credentialauthority.ParseIdentity(id)
	if err != nil {
		return "", err
	}
	return a.Require(identity, field)
}

func (p *HTTPProvider) request(ctx context.Context, method, path string, body any, key string) ([]byte, error) {
	reader, err := httpProviderBody(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(p.Declaration.Endpoint, "/")+path, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Minute, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return readHTTPProviderResponse(resp)
}

func httpProviderBody(body any) (io.Reader, error) {
	if body == nil {
		return nil, nil
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func readHTTPProviderResponse(resp *http.Response) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("provider returned HTTP %d", resp.StatusCode)
	}
	return data, nil
}

func NewPackagedAdapters(bundle catalog.Bundle) []Adapter {
	result := []Adapter{}
	for _, p := range bundle.Providers {
		result = append(result, Adapter{Provider: p.ID, Family: p.Family, Locality: p.Locality, HTTP: &HTTPProvider{Declaration: p}})
	}
	return result
}
