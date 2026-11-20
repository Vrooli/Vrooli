package conformance

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"connectrpc.com/connect"
)

// NewDiscoveredSearchHubModelEvalSource creates a lazy client using Vrooli's
// service discovery. It never starts Search Hub or calls inference.
func NewDiscoveredSearchHubModelEvalSource(resolve func(context.Context, string) (string, error), httpClient connect.HTTPClient) ModelEvalReadinessSource {
	return discoveredModelEvalSource{resolve: resolve, http: httpClient}
}

// ReadModelEvalReadiness inspects accepted suite snapshots without execution.
// It is separate from RunSuite and therefore safe without inference admission.
func ReadModelEvalReadiness(ctx context.Context, source ModelEvalReadinessSource, roles ModelEvalRoleDenominators) (ModelEvalReadiness, error) {
	if source == nil {
		return ModelEvalReadiness{}, fmt.Errorf("model-eval accepted-suite source is not configured")
	}
	return source.Readiness(ctx, roles)
}

func LoadGatewayTaskRoles(catalogPath string) ([]string, error) {
	if strings.TrimSpace(catalogPath) == "" {
		return nil, fmt.Errorf("AI Gateway task-role catalog path is not configured")
	}
	raw, err := os.ReadFile(catalogPath)
	if err != nil {
		return nil, fmt.Errorf("read AI Gateway task-role catalog: %w", err)
	}
	var catalog struct {
		Roles map[string]json.RawMessage `json:"roles"`
	}
	if err := json.Unmarshal(raw, &catalog); err != nil {
		return nil, fmt.Errorf("decode AI Gateway task-role catalog: %w", err)
	}
	roles := make([]string, 0, len(catalog.Roles))
	for role := range catalog.Roles {
		role = strings.TrimSpace(role)
		if role != "" {
			roles = append(roles, role)
		}
	}
	if len(roles) == 0 {
		return nil, fmt.Errorf("AI Gateway task-role catalog declares no roles")
	}
	sort.Strings(roles)
	return roles, nil
}

func LoadModelEvalRoles(policyPath string) ([]string, error) {
	if strings.TrimSpace(policyPath) == "" {
		return nil, fmt.Errorf("authoritative Ollama model policy path is not configured")
	}
	raw, err := os.ReadFile(policyPath)
	if err != nil {
		return nil, fmt.Errorf("read authoritative Ollama model policy: %w", err)
	}
	var policy struct {
		Roles map[string]json.RawMessage `json:"roles"`
	}
	if err := json.Unmarshal(raw, &policy); err != nil {
		return nil, fmt.Errorf("decode authoritative Ollama model policy: %w", err)
	}
	roles := make([]string, 0, len(policy.Roles))
	for role := range policy.Roles {
		role = strings.TrimSpace(role)
		if role != "" {
			roles = append(roles, role)
		}
	}
	if len(roles) == 0 {
		return nil, fmt.Errorf("authoritative Ollama model policy declares no roles")
	}
	sort.Strings(roles)
	return roles, nil
}
