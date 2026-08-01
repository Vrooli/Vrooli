package monitoring

import (
	"errors"
	"fmt"
	"os"

	"vrooli-autoheal/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "monitoring",
		Description: "Manage monitored scenarios and resources",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "show", Description: "Show monitoring configuration", Run: func(args []string) error { return get(core, args) }},
			{Name: "add-scenario", Description: "Add a monitored scenario", Run: func(args []string) error { return addScenario(core, args) }},
			{Name: "remove-scenario", Description: "Remove a monitored scenario", Run: func(args []string) error { return removeScenario(core, args) }},
			{Name: "set-scenario-critical", Description: "Set whether a monitored scenario is critical", Run: func(args []string) error { return setScenarioCritical(core, args) }},
			{Name: "add-resource", Description: "Add a monitored resource", Run: func(args []string) error { return addResource(core, args) }},
			{Name: "remove-resource", Description: "Remove a monitored resource", Run: func(args []string) error { return removeResource(core, args) }},
		},
	}
}

func get(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("monitoring show")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	body, err := core.Get("/config/monitoring", nil)
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

func addScenario(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("monitoring add-scenario")
	critical := fs.Bool("critical", false, "Mark the scenario as critical")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: vrooli-autoheal monitoring add-scenario <name> [--critical]")
	}
	body, err := core.Request("POST", "/config/monitoring/scenarios", nil, map[string]interface{}{
		"name":     fs.Arg(0),
		"critical": *critical,
	})
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

func removeScenario(core *cliapp.ScenarioApp, args []string) error {
	return deleteByName(core, args, "/config/monitoring/scenarios/%s", "usage: vrooli-autoheal monitoring remove-scenario <name>")
}

func setScenarioCritical(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("monitoring set-scenario-critical")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 2 {
		return fmt.Errorf("usage: vrooli-autoheal monitoring set-scenario-critical <name> <true|false>")
	}
	body, err := core.Request("PUT", fmt.Sprintf("/config/monitoring/scenarios/%s/critical", fs.Arg(0)), nil, map[string]bool{
		"critical": fs.Arg(1) == "true",
	})
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

func addResource(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("monitoring add-resource")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: vrooli-autoheal monitoring add-resource <name>")
	}
	body, err := core.Request("POST", "/config/monitoring/resources", nil, map[string]string{"name": fs.Arg(0)})
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

func removeResource(core *cliapp.ScenarioApp, args []string) error {
	return deleteByName(core, args, "/config/monitoring/resources/%s", "usage: vrooli-autoheal monitoring remove-resource <name>")
}

func deleteByName(core *cliapp.ScenarioApp, args []string, pathFmt string, usage string) error {
	fs := support.NewFlagSet("monitoring delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New(usage)
	}
	body, err := core.Request("DELETE", fmt.Sprintf(pathFmt, fs.Arg(0)), nil, nil)
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
