package lifecycle

import (
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	h := newHandlers(core)
	return []cliapp.CommandGroup{{
		Title: "Template Lifecycle",
		Commands: []cliapp.Command{
			(cliapp.Command{
				Name:        "generate",
				Description: "Generate a scenario from a governed template",
				NeedsAPI:    true,
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{{Name: "template", Required: true}},
					Flags: []cliapp.Flag{
						{Name: "id", Required: true},
						{Name: "display-name", Required: true},
						{Name: "description", Required: true},
						{Name: "dest"},
						{Name: "design"},
						{Name: "var"},
						{Name: "force", Bool: true},
						{Name: "dry-run", Bool: true},
						{Name: "run-hooks", Bool: true},
					},
				},
			}).WithPrimitive(cliapp.ProtoMutation(h.generateCall, h.generateReport)),
			(cliapp.Command{
				Name:        "orient",
				Description: "Show or finalize generated scenario orientation progress",
				NeedsAPI:    true,
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{{Name: "scenario", Required: true}},
					Flags:       []cliapp.Flag{{Name: "finalize", Bool: true}},
				},
			}).WithPrimitive(cliapp.ProtoList(h.orientCall, h.orientReport)),
			(cliapp.Command{
				Name:        "detemplate",
				Description: "Remove the template example domain from a generated scenario",
				NeedsAPI:    true,
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{{Name: "scenario", Required: true}},
					Flags:       []cliapp.Flag{{Name: "dry-run", Bool: true}},
				},
			}).WithPrimitive(cliapp.ProtoMutation(h.detemplateCall, h.detemplateReport)),
		},
	}}
}

func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	h := newHandlers(core)
	return []cliapp.SubcommandGroup{
		{
			Name:        "lifecycle",
			Description: "Generate, orient, and detemplate scenarios",
			NeedsAPI:    true,
			Subcommands: []cliapp.Command{
				(cliapp.Command{
					Name:        "generate",
					Description: "Generate a scenario from a governed template",
					Args: cliapp.ArgSchema{
						Positionals: []cliapp.Positional{{Name: "template", Required: true}},
						Flags: []cliapp.Flag{
							{Name: "id", Required: true},
							{Name: "display-name", Required: true},
							{Name: "description", Required: true},
							{Name: "dest"},
							{Name: "design"},
							{Name: "var"},
							{Name: "force", Bool: true},
							{Name: "dry-run", Bool: true},
							{Name: "run-hooks", Bool: true},
						},
					},
				}).WithPrimitive(cliapp.ProtoMutation(h.generateCall, h.generateReport)),
				(cliapp.Command{
					Name:        "orient",
					Description: "Show or finalize generated scenario orientation progress",
					Args: cliapp.ArgSchema{
						Positionals: []cliapp.Positional{{Name: "scenario", Required: true}},
						Flags:       []cliapp.Flag{{Name: "finalize", Bool: true}},
					},
				}).WithPrimitive(cliapp.ProtoList(h.orientCall, h.orientReport)),
				(cliapp.Command{
					Name:        "detemplate",
					Description: "Remove the template example domain from a generated scenario",
					Args: cliapp.ArgSchema{
						Positionals: []cliapp.Positional{{Name: "scenario", Required: true}},
						Flags:       []cliapp.Flag{{Name: "dry-run", Bool: true}},
					},
				}).WithPrimitive(cliapp.ProtoMutation(h.detemplateCall, h.detemplateReport)),
			},
		},
		{
			Name:        "template",
			Description: "Validate templates, inspect drift, and clean validation workspaces",
			NeedsAPI:    true,
			Subcommands: []cliapp.Command{
				(cliapp.Command{Name: "validate", Description: "Validate scenario templates", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "template"}, {Name: "mode", Default: "shallow"}, {Name: "test-preset"}, {Name: "warning-policy"}, {Name: "retain-temp", Bool: true}}}}).WithPrimitive(cliapp.ProtoList(h.validateCall, h.validateReport)),
				(cliapp.Command{Name: "drift", Description: "Report template drift", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario"}}, Flags: []cliapp.Flag{{Name: "all", Bool: true}, {Name: "verbose", Bool: true}}}}).WithPrimitive(cliapp.ProtoList(h.driftCall, h.driftReport)),
				(cliapp.Command{Name: "cleanup", Description: "Clean retained or stale deep-validation workspaces", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "dry-run", Bool: true}, {Name: "older-than"}, {Name: "include-retained", Bool: true}, {Name: "run"}}}}).WithPrimitive(cliapp.ProtoMutation(h.cleanupCall, h.cleanupReport)),
			},
		},
		{
			Name:        "design",
			Description: "Inspect and validate scenario design kits",
			NeedsAPI:    true,
			Subcommands: []cliapp.Command{
				(cliapp.Command{Name: "list", Description: "List design kits"}).WithPrimitive(cliapp.ProtoList(h.designListCall, h.designListReport)),
				(cliapp.Command{Name: "show", Description: "Show one design kit", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true}}}}).WithPrimitive(cliapp.ProtoList(h.designShowCall, h.designShowReport)),
				(cliapp.Command{Name: "validate", Description: "Validate design kits", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "id"}, {Name: "all", Bool: true}}}}).WithPrimitive(cliapp.ProtoListOutcome(h.designValidateCall, h.designValidateReport, h.designValidateOutcome)),
			},
		},
	}
}

func parseVars(values []string) (map[string]string, error) {
	out := make(map[string]string, len(values))
	for _, value := range values {
		key, val, ok := strings.Cut(value, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("--var must be KEY=VALUE")
		}
		out[key] = val
	}
	return out, nil
}
