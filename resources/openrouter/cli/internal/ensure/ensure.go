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

	sharedensure "github.com/vrooli/ai-provider-go/ensure"

	"github.com/vrooli/vrooli/resources/openrouter/cli/internal/policy"

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
// live catalog availability. The mechanics live in the shared ensure package;
// this resource supplies its OpenRouter catalog, log prefix, and checker.
func Run(ctx context.Context, cfg Config, p policy.Policy, checker CatalogChecker, stdout io.Writer) error {
	return sharedensure.Run(ctx, sharedensure.Config{
		ModelRoles:       cfg.ModelRoles,
		DeprecatedModel:  cfg.DeprecatedModel,
		DeprecatedModels: cfg.DeprecatedModels,
	}, p.Catalog, checker, stdout, logPrefix)
}
