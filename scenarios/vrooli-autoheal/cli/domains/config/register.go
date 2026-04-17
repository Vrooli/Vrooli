package config

import (
	"fmt"
	"os"
	"strings"
	"vrooli-autoheal/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "config",
		Description: "Inspect and update autoheal runtime configuration",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "show", Description: "Show the current runtime config", Run: func(args []string) error { return get(core, "/config", args) }},
			{Name: "defaults", Description: "Show the default runtime config", Run: func(args []string) error { return get(core, "/config/defaults", args) }},
			{Name: "global", Description: "Show the global runtime config section", Run: func(args []string) error { return get(core, "/config/global", args) }},
			{Name: "ui", Description: "Show the UI runtime config section", Run: func(args []string) error { return get(core, "/config/ui", args) }},
			{Name: "validate", Description: "Validate a config JSON payload from a file", Run: func(args []string) error { return validate(core, args) }},
			{Name: "import", Description: "Import config JSON from a file", Run: func(args []string) error { return importConfig(core, args) }},
			{Name: "export", Description: "Export the current config JSON", Run: func(args []string) error { return exportConfig(core, args) }},
			{Name: "check-enabled", Description: "Set whether a check is enabled", Run: func(args []string) error { return setCheckEnabled(core, args) }},
			{Name: "check-autoheal", Description: "Set whether a check can auto-heal", Run: func(args []string) error { return setCheckAutoHeal(core, args) }},
			{Name: "bulk", Description: "Apply a bulk check config action", Run: func(args []string) error { return bulk(core, args) }},
		},
	}
}

func get(core *cliapp.ScenarioApp, path string, args []string) error {
	fs := support.NewFlagSet("config")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	body, err := core.Get(path, nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}
	fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
	return nil
}

func validate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("config validate")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: vrooli-autoheal config validate <config.json>")
	}
	payload, err := cliutil.ReadFileString(fs.Arg(0))
	if err != nil {
		return err
	}
	var body interface{}
	if err := support.Decode([]byte(payload), &body); err != nil {
		return err
	}
	respBody, err := core.Request("POST", "/config/validate", nil, body)
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(respBody))
		return nil
	}
	fmt.Fprintln(os.Stdout, support.PrettyJSON(respBody))
	return nil
}

func importConfig(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("config import")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: vrooli-autoheal config import <config.json>")
	}
	payload, err := cliutil.ReadFileString(fs.Arg(0))
	if err != nil {
		return err
	}
	var body interface{}
	if err := support.Decode([]byte(payload), &body); err != nil {
		return err
	}
	respBody, err := core.Request("POST", "/config/import", nil, body)
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(respBody))
		return nil
	}
	fmt.Fprintln(os.Stdout, support.PrettyJSON(respBody))
	return nil
}

func exportConfig(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("config export")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	body, err := core.Get("/config/export", nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}
	fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
	return nil
}

func setCheckEnabled(core *cliapp.ScenarioApp, args []string) error {
	return mutateCheckBool(core, args, "enabled", "/config/checks/%s/enabled")
}

func setCheckAutoHeal(core *cliapp.ScenarioApp, args []string) error {
	return mutateCheckBool(core, args, "autoHeal", "/config/checks/%s/autoheal")
}

func mutateCheckBool(core *cliapp.ScenarioApp, args []string, key string, pathFmt string) error {
	fs := support.NewFlagSet("config mutate")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 2 {
		return fmt.Errorf("usage: <command> <check-id> <true|false>")
	}
	checkID := fs.Arg(0)
	value := strings.EqualFold(fs.Arg(1), "true")
	body, err := core.Request("PUT", fmt.Sprintf(pathFmt, checkID), nil, map[string]bool{key: value})
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}
	fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
	return nil
}

func bulk(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("config bulk")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: vrooli-autoheal config bulk <enableAll|disableAll|autoHealAll|disableAutoHealAll>")
	}
	body, err := core.Request("PUT", "/config/checks/bulk", nil, map[string]string{"action": fs.Arg(0)})
	if err != nil {
		return err
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
		return nil
	}
	fmt.Fprintln(os.Stdout, support.PrettyJSON(body))
	return nil
}
