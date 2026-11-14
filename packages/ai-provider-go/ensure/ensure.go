// Package ensure implements the role-only dependency validation shared by
// hosted AI provider resources. Local providers may use the same config shape
// but keep their own download/pull behavior.
package ensure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/vrooli/ai-provider-go/policy"
)

// Config is the scenario-provided payload for `ensure`. The greenfield contract
// is role-only: concrete model fields are captured solely so ensure can emit a
// precise failure rather than silently ignoring them.
type Config struct {
	ModelRoles []policy.RoleRequest `json:"model_roles,omitempty"`

	DeprecatedModel  json.RawMessage `json:"model,omitempty"`
	DeprecatedModels json.RawMessage `json:"models,omitempty"`
}

func ParseConfig(data []byte) (Config, error) {
	var cfg Config
	if len(data) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse ensure config: %w", err)
	}
	return cfg, nil
}

// DeprecatedFields reports concrete-model fields that violate the role-only
// contract.
func (c Config) DeprecatedFields() []string {
	var found []string
	if len(c.DeprecatedModel) > 0 && string(c.DeprecatedModel) != "null" {
		found = append(found, "model")
	}
	if len(c.DeprecatedModels) > 0 && string(c.DeprecatedModels) != "null" {
		found = append(found, "models")
	}
	return found
}

// ResolveRequest returns the role-only resolution request.
func (c Config) ResolveRequest() policy.ResolveRequest {
	return policy.ResolveRequest{ModelRoles: c.ModelRoles}
}

// CatalogChecker is the best-effort live-availability seam. Any error degrades
// to a non-blocking warning: ensure refuses only on a definitive policy
// violation, never on catalog uncertainty.
type CatalogChecker interface {
	Present(ctx context.Context, endpoints, models []string) (map[string]bool, error)
}

// Run validates every declared role and, when a checker is supplied, verifies
// the resolved models remain visible in the live provider catalog.
func Run(ctx context.Context, cfg Config, catalog policy.Catalog, checker CatalogChecker, stdout io.Writer, logPrefix string) error {
	if logPrefix == "" {
		logPrefix = "ensure:"
	}
	if deprecated := cfg.DeprecatedFields(); len(deprecated) > 0 {
		return fmt.Errorf("%s dependency config uses forbidden concrete-model field(s) %s; declare model_roles instead (greenfield contract)",
			logPrefix, strings.Join(deprecated, ", "))
	}
	if len(cfg.ModelRoles) == 0 {
		fmt.Fprintf(stdout, "%s no model_roles declared; nothing to validate\n", logPrefix)
		return nil
	}

	resolution, err := policy.Resolve(catalog, cfg.ResolveRequest())
	if err != nil {
		return fmt.Errorf("%s %w", logPrefix, err)
	}
	for _, warning := range resolution.Warnings {
		fmt.Fprintf(stdout, "%s warning: %s\n", logPrefix, warning)
	}

	endpointSet := map[string]struct{}{}
	wantModels := map[string][]string{}
	for _, role := range resolution.Roles {
		fmt.Fprintf(stdout, "%s resolved role %s -> %s [%s]\n", logPrefix, role.Role, role.Model, role.Endpoint)
		endpointSet[role.Endpoint] = struct{}{}
		definition := catalog.Roles[role.Role]
		models := append([]string{definition.Model}, definition.Fallbacks...)
		wantModels[role.Endpoint] = append(wantModels[role.Endpoint], models...)
	}

	if checker == nil {
		fmt.Fprintf(stdout, "%s catalog availability check skipped (no live checker / key)\n", logPrefix)
		return nil
	}

	endpoints := keysOf(endpointSet)
	allModels := dedupe(flatten(wantModels))
	present, err := checker.Present(ctx, endpoints, allModels)
	if err != nil {
		fmt.Fprintf(stdout, "%s catalog availability check degraded (best-effort): %v\n", logPrefix, err)
		return nil
	}
	var missing []string
	for _, model := range allModels {
		if !present[model] {
			missing = append(missing, model)
		}
	}
	sort.Strings(missing)
	for _, model := range missing {
		fmt.Fprintf(stdout, "%s warning: model %q is not visible in the live provider catalog; update model-policy.json if it was retired\n", logPrefix, model)
	}
	if len(missing) == 0 {
		fmt.Fprintf(stdout, "%s all %d resolved model(s) present in live catalog\n", logPrefix, len(allModels))
	}
	return nil
}

func keysOf(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for key := range m {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func flatten(m map[string][]string) []string {
	var out []string
	for _, values := range m {
		out = append(out, values...)
	}
	return out
}

func dedupe(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, value := range in {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
