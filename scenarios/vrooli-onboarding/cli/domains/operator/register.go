package operator

import (
	"os"

	"vrooli-onboarding/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes the V2 onboarding control plane. It owns no state: the API
// persists operator decisions and derives scenarios/readiness from manifests.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "operator", Description: "Inspect and commit V2 operator state", NeedsAPI: true, Subcommands: []cliapp.Command{
		{Name: "show", Description: "Show persisted operator state", Run: func(args []string) error { return support.GetJSON(core, "operator", args, "/v1/operator-state") }},
		{Name: "patch", Description: "Atomically apply an RFC 7386 operator-state merge patch from --body-file", Run: func(args []string) error { return runPatch(core, args) }},
		{Name: "scenarios", Description: "Show manifest-derived scenario choices", Run: func(args []string) error { return support.GetJSON(core, "operator", args, "/v2/scenarios") }},
		{Name: "readiness", Description: "Show metadata-safe composed readiness", Run: func(args []string) error { return support.GetJSON(core, "operator", args, "/v2/readiness") }},
	}}
}

func runShow(core *cliapp.ScenarioApp, args []string) error {
	return support.GetJSON(core, "operator", args, "/operator-state")
}

func runPatch(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("operator patch")
	bodyFile := fs.String("body-file", "", "Path to an RFC 7386 operator-state merge patch")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	body, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	response, err := core.Request("PATCH", "/v2/operator-state", nil, body)
	if err != nil {
		return err
	}
	if *jsonOutput {
		_, err = os.Stdout.Write(append(response, '\n'))
		return err
	}
	return cliapp.RenderMutationReport(os.Stdout, cliapp.MutationReport{Result: []string{"Operator state committed atomically"}, NextCommand: []string{support.CLIName + " operator readiness", support.CLIName + " operator scenarios"}})
}
