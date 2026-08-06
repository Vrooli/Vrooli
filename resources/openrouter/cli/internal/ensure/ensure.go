// Package ensure implements `resource-openrouter ensure`: it reads a scenario's
// OpenRouter dependency config (model roles only) and validates it against the
// resource-owned policy. OpenRouter is cloud-hosted, so ensure NEVER downloads
// anything — it validates policy integrity and, best-effort, that resolved
// models still appear in the live catalog. It is the role-only admission point
// for scenario start.
package ensure

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"resource-openrouter/cli/internal/policy"

	"github.com/vrooli/cli-core/cliapp"
)

const logPrefix = "openrouter-ensure:"

// CatalogChecker is the best-effort live-availability seam. Present reports which
// of the requested model slugs are visible in the live OpenRouter catalog for
// the given endpoint families. Any error degrades to a non-blocking warning —
// ensure refuses only on a definitive policy violation, never on catalog
// uncertainty (the key may be absent at scenario start).
type CatalogChecker interface {
	Present(ctx context.Context, endpoints, models []string) (map[string]bool, error)
}

// CommandGroup registers the `ensure` verb. The optional checker wires the live
// catalog availability probe; nil skips it.
func CommandGroup(checker CatalogChecker) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Dependency Ensurance",
		Commands: []cliapp.Command{
			{
				Name:        "ensure",
				Description: "Validate the OpenRouter model roles declared in a scenario's dependency config",
				Usage:       "resource-openrouter ensure --config-base64 <base64-json> [--skip-catalog-check]",
				Run:         func(args []string) error { return runCLI(args, checker) },
			},
		},
	}
}

func runCLI(args []string, checker CatalogChecker) error {
	var (
		configB64        string
		skipCatalogCheck bool
	)
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--config-base64":
			if i+1 >= len(args) {
				return errors.New("--config-base64 requires a value")
			}
			configB64 = args[i+1]
			i++
		case "--skip-catalog-check":
			skipCatalogCheck = true
		default:
			return fmt.Errorf("unknown flag %q", args[i])
		}
	}
	if configB64 == "" {
		return errors.New("--config-base64 is required")
	}
	raw, err := base64.StdEncoding.DecodeString(configB64)
	if err != nil {
		return fmt.Errorf("decode --config-base64: %w", err)
	}
	cfg, err := ParseConfig(raw)
	if err != nil {
		return err
	}
	p, _, err := policy.LoadDefaultFile(os.Getenv)
	if err != nil {
		return err
	}
	if skipCatalogCheck {
		checker = nil
	}
	return Run(context.Background(), cfg, p, checker, os.Stdout)
}

// Run is the testable entry point: validate deprecated fields, resolve every
// declared role through policy (unknown roles are fatal), then best-effort verify
// live catalog availability.
func Run(ctx context.Context, cfg Config, p policy.Policy, checker CatalogChecker, stdout io.Writer) error {
	if deprecated := cfg.DeprecatedFields(); len(deprecated) > 0 {
		return fmt.Errorf("%s OpenRouter dependency config uses forbidden concrete-model field(s) %s; declare model_roles instead (greenfield contract)",
			logPrefix, strings.Join(deprecated, ", "))
	}
	if len(cfg.ModelRoles) == 0 {
		fmt.Fprintf(stdout, "%s no model_roles declared; nothing to validate\n", logPrefix)
		return nil
	}

	resolution, err := p.Resolve(cfg.ResolveRequest())
	if err != nil {
		return fmt.Errorf("%s %w", logPrefix, err)
	}
	for _, w := range resolution.Warnings {
		fmt.Fprintf(stdout, "%s warning: %s\n", logPrefix, w)
	}

	endpointSet := map[string]struct{}{}
	wantModels := map[string][]string{} // endpoint -> model slugs (default + fallbacks)
	for _, r := range resolution.Roles {
		fmt.Fprintf(stdout, "%s resolved role %s -> %s [%s]\n", logPrefix, r.Role, r.Model, r.Endpoint)
		endpointSet[r.Endpoint] = struct{}{}
		role := p.Roles[r.Role]
		models := append([]string{role.Model}, role.Fallbacks...)
		wantModels[r.Endpoint] = append(wantModels[r.Endpoint], models...)
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
	for _, m := range allModels {
		if !present[m] {
			missing = append(missing, m)
		}
	}
	sort.Strings(missing)
	for _, m := range missing {
		fmt.Fprintf(stdout, "%s warning: model %q is not visible in the live OpenRouter catalog; update model-policy.json if it was retired\n", logPrefix, m)
	}
	if len(missing) == 0 {
		fmt.Fprintf(stdout, "%s all %d resolved model(s) present in live catalog\n", logPrefix, len(allModels))
	}
	return nil
}

func keysOf(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func flatten(m map[string][]string) []string {
	var out []string
	for _, v := range m {
		out = append(out, v...)
	}
	return out
}

func dedupe(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
