package bundle

import (
	"fmt"

	"scenario-dependency-analyzer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "bundle",
		Description: "Inspect generated bundle manifests",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "manifest", Description: "Show the deployment bundle manifest", Run: func(args []string) error { return runManifest(core, args) }},
		},
	}
}

func runManifest(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("bundle manifest")
	var jsonOutput bool
	var refresh bool
	fs.BoolVar(&jsonOutput, "json", false, "Output JSON")
	fs.BoolVar(&refresh, "refresh", false, "Refresh deployment report first")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	positionals := fs.Args()
	if len(positionals) != 1 {
		return fmt.Errorf("usage: %s bundle manifest <scenario> [--refresh] [--json]", support.AppName)
	}
	scenario := positionals[0]
	query := support.BuildQuery(map[string]string{"refresh": support.BoolWord(refresh, "true", "")})
	body, err := core.Get("/scenarios/"+scenario+"/bundle/manifest", query)
	if err != nil {
		return err
	}
	if jsonOutput {
		return support.PrintAPIJSON(body)
	}
	var resp map[string]interface{}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}
	manifest := support.Map(resp["manifest"])
	results := []string{}
	for _, dep := range support.Maps(manifest["dependencies"]) {
		results = append(results, fmt.Sprintf("dependency %s :: %s", support.String(dep["type"]), support.String(dep["name"])))
	}
	for _, file := range support.Maps(manifest["files"]) {
		results = append(results, fmt.Sprintf("file %s :: %s", support.String(file["type"]), support.String(file["path"])))
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Scenario: %s", scenario),
			fmt.Sprintf("Generated: %s", support.String(resp["generated"])),
			fmt.Sprintf("Dependency entries: %d", len(support.Maps(manifest["dependencies"]))),
			fmt.Sprintf("File entries: %d", len(support.Maps(manifest["files"]))),
		},
		ResultsHeading: "Manifest Entries",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s deployment %s", support.AppName, scenario),
			fmt.Sprintf("%s dag export %s", support.AppName, scenario),
		},
	}
	return support.PrintList(false, report, nil)
}
