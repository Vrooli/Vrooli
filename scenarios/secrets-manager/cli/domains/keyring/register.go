package keyring

import (
	"context"
	"fmt"

	"secrets-manager/cli/internal/credentials"
	"secrets-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(_ *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "keyring",
		Description: "Inspect and repair the platform credential keyring",
		Subcommands: []cliapp.Command{
			{Name: "inspect", Description: "Inspect a keyring without modifying it", Run: func(args []string) error { return run(args, false) }},
			{Name: "repair", Description: "Repair Vrooli-owned keyring entries", Run: func(args []string) error { return run(args, true) }},
		},
	}
}

func run(args []string, repair bool) error {
	fs := support.NewFlagSet("keyring")
	path := fs.String("path", "", "keyring path")
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
	var report any
	if repair {
		report, err = client.KeyringRepair(context.Background(), *path)
	} else {
		report, err = client.KeyringInspect(context.Background(), *path)
	}
	if err != nil {
		return err
	}
	return support.PrintList(jsonOutput, report, cliapp.ListReport{Summary: []string{fmt.Sprintf("Keyring repair: %t", repair)}, ResultsHeading: "Keyring report"})
}
