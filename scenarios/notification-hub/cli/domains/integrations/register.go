package integrations

import (
	"encoding/json"
	"fmt"

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

// GroupName is the manifest group these commands belong to.
const GroupName = "integrations"

// Register builds the integrations group from cli/manifest.json. Both commands
// call the API's event-integration settings route directly (it is an HTTP
// settings document, not an RPC), so the manifest declares them local and this
// package supplies their handlers by command name.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	return cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"event-config":     func(ctx cliapp.RunContext) error { return get(core, ctx) },
		"event-config-set": func(ctx cliapp.RunContext) error { return set(core, ctx) },
	})
}

func get(core *cliapp.ScenarioApp, ctx cliapp.RunContext) error {
	body, err := core.Get("/config/event-integration", nil)
	if err != nil {
		return err
	}
	_, err = ctx.Stdout().Write(append(body, '\n'))
	return err
}

func set(core *cliapp.ScenarioApp, ctx cliapp.RunContext) error {
	base, target, pattern := ctx.Flag("events-api-base"), ctx.Flag("webhook-url"), ctx.Flag("pattern")
	currentBody, err := core.Get("/config/event-integration", nil)
	if err != nil {
		return err
	}
	var current config
	if err := json.Unmarshal(currentBody, &current); err != nil {
		return fmt.Errorf("decode current event integration config: %w", err)
	}
	if base != "" {
		current.EventsAPIBase = base
	}
	if target != "" {
		current.WebhookURL = target
	}
	if pattern != "" {
		current.Pattern = pattern
	}
	updated, err := core.Request("PUT", "/config/event-integration", nil, current)
	if err != nil {
		return err
	}
	_, err = ctx.Stdout().Write(append(updated, '\n'))
	return err
}
