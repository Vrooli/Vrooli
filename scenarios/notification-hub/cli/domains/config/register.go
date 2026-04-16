package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"notification-hub/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(d support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			{
				Name:        "configure",
				Description: "View or update CLI settings (api_base, token/api_key, profile_id)",
				Run: func(args []string) error {
					return runConfigure(d, args)
				},
			},
		},
	}
}

func runConfigure(d support.Dependencies, args []string) error {
	core := d.ScenarioApp()
	if core == nil {
		return fmt.Errorf("scenario app is not initialized")
	}

	defaults := d.DefaultConfig()
	if len(args) == 0 {
		current := struct {
			APIBase   string `json:"api_base"`
			Token     string `json:"token,omitempty"`
			ProfileID string `json:"profile_id,omitempty"`
		}{
			APIBase:   strings.TrimSpace(core.Config.APIBase),
			Token:     strings.TrimSpace(core.Config.Token),
			ProfileID: strings.TrimSpace(defaults.ProfileID),
		}
		payload, err := json.MarshalIndent(current, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(os.Stdout, string(payload))
		return err
	}

	if len(args) != 2 {
		return fmt.Errorf("usage: configure <api_base|token|api_key|profile_id> <value>")
	}

	key := strings.TrimSpace(args[0])
	value := strings.TrimSpace(args[1])
	switch key {
	case "api_base":
		core.Config.APIBase = value
		if err := core.SaveConfig(); err != nil {
			return err
		}
	case "token", "api_key":
		core.Config.Token = value
		if err := core.SaveConfig(); err != nil {
			return err
		}
	case "profile_id":
		defaults.ProfileID = value
		if err := d.DefaultsStore().Save(defaults); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	_, err := fmt.Fprintf(os.Stdout, "Updated %s\n", key)
	return err
}
