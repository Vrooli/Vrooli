package signing

import (
	"errors"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(app *cliapp.ScenarioApp) cliapp.CommandGroup {
	_ = app
	return cliapp.CommandGroup{
		Title: "Code Signing",
		Commands: []cliapp.Command{
			{Name: "signing", NeedsAPI: true, Description: "Configure code signing for deployments (retired)", Run: func([]string) error {
				return errors.New("signing commands are retired from deployment-manager; configure signing through scenario-to-desktop")
			}},
		},
	}
}
