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
	model := role.Model
	if p.Declaration.Family == FamilyOllama {
		data, err := p.request(ctx, http.MethodGet, "/api/tags", nil, "")
		if err != nil {
			return ResolvedRole{}, err
		}
		var tags struct {
			Models []struct {
				Name string `json:"name"`
			} `json:"models"`
		}
		if err = json.Unmarshal(data, &tags); err != nil {
			return ResolvedRole{}, err
		}
		installed := map[string]bool{}
		for _, m := range tags.Models {
			installed[m.Name] = true
		}
		model = ""
		for _, candidate := range append([]string{role.Model}, role.Fallbacks...) {
			if installed[candidate] {
				model = candidate
				break
			}
		}
		if model == "" {
			return ResolvedRole{}, errors.New("no declared local model is installed")
		}
	}
	cap := role.MaxTokens
	if cap == 0 {
		cap = role.Defaults.MaxTokens
	}
	return ResolvedRole{Provider: p.Declaration.ID, Role: name, Model: model, CanonicalModel: model, Capabilities: role.Capabilities, SamplingSupport: parseSamplingSupport(role.Sampling), MaxOutputTokens: cap, ContextWindow: role.ContextWindow, Locality: p.Declaration.Locality}, nil
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
	var body map[string]any
	path := "/chat/completions"
	key := ""
	switch p.Declaration.Family {
	case FamilyOllama:
		path = "/api/generate"
		options := map[string]any{}
		if cap > 0 {
			options["num_predict"] = cap
		}
		if req.Temperature != nil {
			options["temperature"] = *req.Temperature
		}
		body = map[string]any{"model": resolved.Model, "prompt": req.InputText, "stream": false, "think": false, "options": options}
		if req.SchemaJSON != "" {
			var schema any
			if err = json.Unmarshal([]byte(req.SchemaJSON), &schema); err != nil {
				return ExecutionResult{}, err
			}
			body["format"] = schema
		}
	case FamilyOpenAICompatible:
		if p.Declaration.CredentialIdentity == "" {
			return ExecutionResult{}, errors.New("provider has no declared own-key credential")
		}
		resolve := p.ResolveKey
		if resolve == nil {
			resolve = resolveProviderKey
		}
		key, err = resolve(ctx, p.Declaration.CredentialIdentity, p.Declaration.CredentialField)
		if err != nil {
			return ExecutionResult{}, err
		}
		if key == "" {
			return ExecutionResult{}, errors.New("own provider key is not configured")
		}
		body = map[string]any{"model": resolved.Model, "messages": []map[string]string{{"role": "user", "content": req.InputText}}, "stream": false}
		if cap > 0 {
			body["max_tokens"] = cap
		}
		if req.Temperature != nil {
			body["temperature"] = *req.Temperature
		}
		if req.SchemaJSON != "" {
			var schema any
			if err = json.Unmarshal([]byte(req.SchemaJSON), &schema); err != nil {
				return ExecutionResult{}, err
			}
			body["response_format"] = map[string]any{"type": "json_schema", "json_schema": map[string]any{"name": "gateway_result", "strict": true, "schema": schema}}
		}
	default:
		return ExecutionResult{}, errors.New("unsupported provider wire format")
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
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(data)
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
