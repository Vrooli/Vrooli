package agentharness

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/cli-core/agentcatalog"
	"github.com/vrooli/cli-core/cliapp"
)

const ModelDiscoverySchemaVersion = agentcatalog.ModelDiscoverySchemaVersion

type ModelDiscoveryConfig struct {
	Runner      string
	CatalogPath string
}

type ModelResolution = agentcatalog.ModelResolution

func ModelDiscoveryCommands(cfg ModelDiscoveryConfig) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "models",
		Description: "Discover models offered by this runner",
		Subcommands: []cliapp.Command{
			{
				Name:        "list",
				Description: "List the runner's live model catalog",
				Run: func(args []string) error {
					fs := policyFlagSet("models list", os.Stderr)
					jsonOut := fs.Bool("json", false, "Emit JSON")
					if err := fs.Parse(args); err != nil {
						return err
					}
					catalog, err := agentcatalog.DiscoverModels(context.Background(), cfg.Runner)
					if err != nil {
						return err
					}
					if *jsonOut {
						return writeJSON(os.Stdout, catalog)
					}
					for _, model := range catalog.Models {
						fmt.Fprintln(os.Stdout, model)
					}
					return nil
				},
			},
			{
				Name:        "resolve",
				Description: "Resolve one runner model through the resource-owned policy",
				Run: func(args []string) error {
					return resolveModel(cfg, args)
				},
			},
		},
	}
}

func resolveModel(cfg ModelDiscoveryConfig, args []string) error {
	fs := policyFlagSet("models resolve", os.Stderr)
	model := fs.String("model", "", "Runner model identifier")
	jsonOut := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	modelValue := strings.TrimSpace(*model)
	if modelValue == "" {
		return errors.New("--model is required")
	}
	resolution, err := ResolveCatalogModel(cfg.Runner, cfg.CatalogPath, modelValue)
	if err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(os.Stdout, resolution)
	}
	fmt.Fprintf(os.Stdout, "%s -> %s\n", resolution.Model, resolution.CanonicalModel)
	return nil
}

// ResolveCatalogModel resolves one model without invoking a CLI. Resources
// use it to implement their command surface; control-plane tests can exercise
// the same deterministic contract.
func ResolveCatalogModel(runner, catalogPath, modelValue string) (ModelResolution, error) {
	runner = strings.TrimSpace(runner)
	modelValue = strings.TrimSpace(modelValue)
	if runner == "" {
		return ModelResolution{}, errors.New("runner is required")
	}
	if modelValue == "" {
		return ModelResolution{}, errors.New("model is required")
	}
	catalog, data, err := agentcatalog.LoadCodingRoleCatalog(runner, catalogPath)
	if err != nil {
		return ModelResolution{}, err
	}
	alias, ok := catalog.ModelAliases[modelValue]
	if !ok {
		for _, role := range catalog.Roles {
			if role.Model == modelValue && strings.TrimSpace(role.CanonicalModel) != "" {
				alias = agentcatalog.ModelAlias{CanonicalModel: role.CanonicalModel}
				ok = true
				break
			}
		}
	}
	if !ok || strings.TrimSpace(alias.CanonicalModel) == "" {
		return ModelResolution{}, fmt.Errorf("model %q has no resource-owned canonical mapping", modelValue)
	}
	return ModelResolution{
		SchemaVersion:  ModelDiscoverySchemaVersion,
		Runner:         runner,
		Model:          modelValue,
		CanonicalModel: alias.CanonicalModel,
		Provider:       alias.Provider,
		Source:         catalog.Provenance.Source,
		PolicyPath:     catalogPath,
		PolicyDigest:   digest(data),
	}, nil
}
