// Package flows is the CLI's flow-inventory command surface. Subcommands
// run in-process against api/internal/flows (no HTTP round-trip) so they
// keep working when the scenario isn't started.
package flows

import (
	"fmt"
	"strings"

	"flow-verifier/internal/flows"
	"flow-verifier/internal/flows/layout"

	"github.com/vrooli/cli-core/cliapp"
)

// Register returns the `flows` subcommand group.
func Register(_ *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	rootFlag := cliapp.Flag{Name: "root", Description: "Repository root to scan (default: cwd)", Default: "."}
	return cliapp.SubcommandGroup{
		Name:        "flows",
		Description: "Discover, validate, scaffold, and explain flow.json contracts",
		Subcommands: []cliapp.Command{
			{
				Name:        "list",
				Description: "List every discovered flow",
				Args:        cliapp.ArgSchema{Flags: []cliapp.Flag{rootFlag}},
				RunCtx:      runList,
			},
			{
				Name:        "validate",
				Description: "Validate every flow against the embedded schema",
				Args:        cliapp.ArgSchema{Flags: []cliapp.Flag{rootFlag}},
				RunCtx:      runValidate,
			},
			{
				Name:        "new",
				Description: "Scaffold a new flow under <feature-dir>/flow/",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{{Name: "feature-dir", Required: true, Description: "Parent directory for the new flow/"}},
					Flags: []cliapp.Flag{
						{Name: "flow-id", Required: true, Description: "Flow identifier"},
						{Name: "lang", Description: "Target language (ts or go); defaults to ts"},
						rootFlag,
					},
				},
				RunCtx: runNew,
			},
			{
				Name:        "explain",
				Description: "Print the typed flow plus its latest verification status",
				Args: cliapp.ArgSchema{
					Flags: []cliapp.Flag{
						{Name: "flow", Required: true, Description: "Flow id to explain"},
						rootFlag,
					},
				},
				RunCtx: runExplain,
			},
		},
	}
}

func runList(ctx cliapp.RunContext) error {
	summaries, err := flows.List(ctx.Flag("root"), "")
	if err != nil {
		return err
	}
	out := ctx.Stdout()
	if len(summaries) == 0 {
		fmt.Fprintln(out, "no flows discovered")
		return nil
	}
	for _, s := range summaries {
		fmt.Fprintf(out, "%s\t%s\t%s\n", s.FlowID, s.Language, s.ContractPath)
	}
	return nil
}

func runValidate(ctx cliapp.RunContext) error {
	summaries, err := flows.Validate(ctx.Flag("root"), "")
	if err != nil {
		return err
	}
	out := ctx.Stdout()
	for _, s := range summaries {
		fmt.Fprintf(out, "valid %s\n", s.FlowID)
	}
	return nil
}

func runNew(ctx cliapp.RunContext) error {
	parent := ctx.Positional("feature-dir")
	flowID := ctx.Flag("flow-id")
	langStr := strings.ToLower(ctx.Flag("lang"))
	var lang layout.Language
	switch langStr {
	case "", "ts", "typescript":
		lang = layout.LanguageTypeScript
	case "go":
		lang = layout.LanguageGo
	default:
		return fmt.Errorf("--lang must be ts or go, got %q", langStr)
	}
	flowDir, err := flows.New(flows.NewOptions{
		Root:      ctx.Flag("root"),
		ParentDir: parent,
		FlowID:    flowID,
		Language:  lang,
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout(), "scaffolded %s\n", flowDir)
	return nil
}

func runExplain(ctx cliapp.RunContext) error {
	report, err := flows.Explain(ctx.Flag("root"), ctx.Flag("flow"))
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(ctx.Stdout(), report)
	return err
}
