package secrets

import (
	"errors"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(app *cliapp.ScenarioApp) cliapp.CommandGroup {
	_ = app
	return cliapp.CommandGroup{
		Title: "Secrets",
		Commands: []cliapp.Command{
			{Name: "secrets", NeedsAPI: true, Description: "Secret discovery and templates (retired)", Run: func([]string) error {
				return errors.New("secrets commands are retired from deployment-manager; use the owning secrets-manager or typed release operations")
			}},
		},
	}
}
