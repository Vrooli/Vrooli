package resourcetemplate

import "github.com/vrooli/cli-core/cliapp"

func Register(core *cliapp.ScenarioApp, _ []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "resource-template",
		Description: "Inspect, validate, and generate resource templates",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			(cliapp.Command{Name: "list", Description: "List resource templates"}).WithPrimitive(cliapp.ProtoList(h.listCall, h.listReport)),
			(cliapp.Command{
				Name:        "show",
				Description: "Show one resource template",
				Args:        cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "name", Required: true}}},
			}).WithPrimitive(cliapp.ProtoList(h.showCall, h.showReport)),
			(cliapp.Command{Name: "validate", Description: "Validate resource templates"}).WithPrimitive(cliapp.ProtoList(h.validateCall, h.validateReport)),
			(cliapp.Command{
				Name:        "generate",
				Description: "Generate files from a resource template",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{{Name: "template"}},
					Flags: []cliapp.Flag{
						{Name: "from-blueprint"},
						{Name: "dest"},
						{Name: "var"},
						{Name: "force", Bool: true},
						{Name: "dry-run", Bool: true},
					},
				},
			}).WithPrimitive(cliapp.ProtoMutation(h.generateCall, h.generateReport)),
		},
	}, nil
}
