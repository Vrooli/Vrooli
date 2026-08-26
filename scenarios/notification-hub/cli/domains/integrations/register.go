package integrations

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/vrooli/cli-core/cliapp"
)

type config struct {
	EventsAPIBase string `json:"events_api_base"`
	WebhookURL    string `json:"webhook_url"`
	Pattern       string `json:"pattern"`
	Templates     map[string]struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	} `json:"templates,omitempty"`
}

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "integrations",
		Description: "Inspect and update live event integration settings",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "event-config", Description: "Show live event integration settings", Run: func(args []string) error { return get(core) }},
			{Name: "event-config-set", Description: "Update live event integration settings without restarting", Run: func(args []string) error { return set(core, args) }},
		},
	}
}

func get(core *cliapp.ScenarioApp) error {
	body, err := core.Get("/config/event-integration", nil)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(append(body, '\n'))
	return err
}

func set(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("notification-hub integrations event-config-set", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	base := fs.String("events-api-base", "", "vrooli-events API base URL")
	target := fs.String("webhook-url", "", "notification-hub event webhook URL")
	pattern := fs.String("pattern", "", "event subscription pattern")
	if err := fs.Parse(args); err != nil {
		return err
	}
	currentBody, err := core.Get("/config/event-integration", nil)
	if err != nil {
		return err
	}
	var current config
	if err := json.Unmarshal(currentBody, &current); err != nil {
		return fmt.Errorf("decode current event integration config: %w", err)
	}
	if *base != "" {
		current.EventsAPIBase = *base
	}
	if *target != "" {
		current.WebhookURL = *target
	}
	if *pattern != "" {
		current.Pattern = *pattern
	}
	updated, err := core.Request("PUT", "/config/event-integration", nil, current)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(append(updated, '\n'))
	return err
}
