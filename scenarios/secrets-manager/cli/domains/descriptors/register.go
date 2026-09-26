package descriptors

import (
	"context"
	"fmt"

	"secrets-manager/cli/internal/credentials"
	"secrets-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(_ *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "descriptors",
		Description: "Inspect and validate declared credential descriptors",
		Subcommands: []cliapp.Command{
			{Name: "list", Description: "List repository-declared credential descriptors", Run: runList},
			{Name: "validate", Description: "Validate descriptor identity and field metadata", Run: runValidate},
		},
	}
}

func runList(args []string) error {
	return run(args, false)
}

func runValidate(args []string) error {
	return run(args, true)
}

func run(args []string, validate bool) error {
	fs := support.NewFlagSet("descriptors")
	jsonFlag, format := support.JSONFlags(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	jsonOutput, err := support.ResolveJSONOutput(*jsonFlag, *format)
	if err != nil {
		return err
	}
	client, err := credentials.New()
	if err != nil {
		return err
	}
	refs, err := client.List(rctx())
	if err != nil {
		return err
	}
	invalid := 0
	for _, ref := range refs {
		if ref.LogicalID == "" || ref.Field == "" {
			invalid++
		}
	}
	if validate && invalid > 0 {
		return fmt.Errorf("descriptor validation failed: %d descriptor(s) lack a logical identity or field", invalid)
	}
	return support.PrintList(jsonOutput, refs, cliapp.ListReport{Summary: []string{fmt.Sprintf("Descriptors: %d", len(refs)), fmt.Sprintf("Invalid: %d", invalid)}, ResultsHeading: "Declared descriptors"})
}

// Kept local so descriptor commands always use the typed client context and do
// not acquire a second discovery or authority implementation.
func rctx() context.Context { return context.Background() }
