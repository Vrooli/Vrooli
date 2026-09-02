package readiness

import (
	readinesscmd "deployment-manager/cli/readiness"
	"github.com/vrooli/cli-core/cliapp"
)

func Register(app *cliapp.ScenarioApp) cliapp.CommandGroup {
	commands := readinesscmd.New(app.APIClient)
	return cliapp.CommandGroup{Title: "Readiness", Commands: []cliapp.Command{{Name: "readiness", NeedsAPI: true, Description: "Aggregate deployment readiness verdicts", Run: commands.Run}}}
}
