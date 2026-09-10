package validations

import (
	"errors"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(app *cliapp.ScenarioApp) cliapp.CommandGroup {
	_ = app
	return cliapp.CommandGroup{
		Title: "Validations",
		Commands: []cliapp.Command{
			{Name: "validations", NeedsAPI: true, Description: "Visual validation quality gate (retired)", Run: func([]string) error {
				return errors.New("visual validation commands are retired from deployment-manager; use the release readiness and evidence operations")
			}},
		},
	}
}
