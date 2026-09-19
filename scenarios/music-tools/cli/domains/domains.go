package domains

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/vrooli/cli-core/cliapp"
	"net/url"
	"os"
	"strings"
)

// CommandGroups aggregates flat command groups from domain packages.
//
// Keep app.go focused on CLI metadata and cli-core wiring. As the scenario
// grows, add domains like domains/tasks or domains/projects and append their
// registrations here. For greenfield scenarios, domain packages are the
// default architecture; do not treat flat command files as the long-term plan.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return nil
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
//
// Each domain package owns a Register(core, manifest) function returning a
// SubcommandGroup built from the scenario's cli/manifest.json. The aggregator
// passes the embedded manifest bytes through unchanged; per-domain Register
// implementations call cliapp.LoadFromManifest with the relevant group name.
//
// This is the CLI side of the domain-module pattern; the API side uses
// the same one-liner-per-domain shape via server.New(deps, modules...).
// See docs/concepts/ARCHITECTURE.md "Domain modules" for the canonical
// pattern when swapping the example domain for your scenario's first
// domain.
//
// For API-backed commands the manifest carries the declarative surface
// (governance, flags, positionals, RPC binding). Handlers stay in
// handlers.go and are wired via the bindings map; refer to
// templates/scenarios/react-vite/docs/internal/SEAMS.md (manifest ↔
// handlers bindings seam) for the contract.
func SubcommandGroups(core *cliapp.ScenarioApp, manifest []byte) ([]cliapp.SubcommandGroup, error) {
	_ = manifest
	return []cliapp.SubcommandGroup{
		{Name: "styles", Description: "Manage composition styles", NeedsAPI: true, Subcommands: []cliapp.Command{
			{Name: "create", Description: "Create a style from a JSON file", Run: func(args []string) error {
				body, err := styleBody(args)
				if err != nil {
					return err
				}
				return request(core, "POST", "/v1/styles", body)
			}},
			{Name: "list", Description: "List styles", Run: func(args []string) error { return get(core, "/v1/styles") }},
			{Name: "export", Description: "Export a style as portable JSON", Run: func(args []string) error { return oneArgGet(core, args, "/v1/styles/", "/export") }},
			{Name: "compile", Description: "Compile a style for composition", Run: func(args []string) error { return oneArgGet(core, args, "/v1/styles/", "/compile") }},
			{Name: "delete", Description: "Delete a custom style", Run: func(args []string) error { return oneArgRequest(core, args, "DELETE", "/v1/styles/") }},
			{Name: "import", Description: "Import a style from a JSON file", Run: func(args []string) error {
				body, err := styleBody(args)
				if err != nil {
					return err
				}
				return request(core, "POST", "/v1/styles/import", body)
			}},
		}},
		{Name: "compose", Description: "Queue peer music takes", NeedsAPI: true, DefaultSubcommand: "run", Subcommands: []cliapp.Command{{Name: "run", Description: "Queue a composition batch", Run: func(args []string) error { return compose(core, args) }}}},
		{Name: "jobs", Description: "Inspect composition jobs", NeedsAPI: true, Subcommands: []cliapp.Command{
			{Name: "get", Description: "Get a job", Run: func(args []string) error { return oneArgGet(core, args, "/v1/jobs/") }},
			{Name: "list", Description: "List jobs", Run: func(args []string) error { return get(core, "/v1/jobs") }},
			{Name: "wait", Description: "Wait for a job", Run: func(args []string) error { return oneArgGet(core, args, "/v1/jobs/", "/wait") }},
			{Name: "cancel", Description: "Cancel a queued or running job", Run: func(args []string) error { return oneArgRequest(core, args, "POST", "/v1/jobs/", "/cancel") }},
		}},
		{Name: "takes", Description: "Inspect generated takes", NeedsAPI: true, Subcommands: []cliapp.Command{
			{Name: "list", Description: "List generated takes", Run: func(args []string) error { return listTakes(core, args) }},
			{Name: "get", Description: "Get one take", Run: func(args []string) error { return oneArgGet(core, args, "/v1/takes/") }},
			{Name: "reserve", Description: "Reserve one take", Run: func(args []string) error { return transition(core, args, "/reserve") }},
			{Name: "consume", Description: "Consume one reserved take", Run: func(args []string) error { return transition(core, args, "/consume") }},
			{Name: "release", Description: "Release one reserved take", Run: func(args []string) error { return transition(core, args, "/release") }},
			{Name: "discard", Description: "Discard one take", Run: func(args []string) error { return transition(core, args, "/discard") }},
		}},
		{Name: "pool", Description: "Reserve composition takes", NeedsAPI: true, Subcommands: []cliapp.Command{
			{Name: "draw", Description: "Draw an available take", Run: func(args []string) error { return poolDraw(core, args) }},
			{Name: "status", Description: "Show pool depth", Run: func(args []string) error { return get(core, "/v1/pool/status") }},
			{Name: "configure", Description: "Configure pool depth", Run: func(args []string) error { return poolConfigure(core, args) }},
		}},
		{Name: "models", Description: "Inspect music model registry", NeedsAPI: true, Subcommands: []cliapp.Command{
			{Name: "list", Description: "List available models", Run: func(args []string) error { return get(core, "/v1/models") }},
			{Name: "get", Description: "Get one model", Run: func(args []string) error { return oneArgGet(core, args, "/v1/models/") }},
		}},
	}, nil
}

func styleBody(args []string) (any, error) {
	fs := flag.NewFlagSet("styles create", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	file := fs.String("file", "", "style JSON file")
	id := fs.String("id", "", "style ID")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if *file == "" {
		return nil, errors.New("--file is required")
	}
	b, err := os.ReadFile(*file)
	if err != nil {
		return nil, err
	}
	var body map[string]any
	if err := json.Unmarshal(b, &body); err != nil {
		return nil, fmt.Errorf("decode style: %w", err)
	}
	if *id != "" {
		body["id"] = *id
	}
	return body, nil
}

func compose(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("compose", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	style := fs.String("style", "", "style ID")
	takes := fs.Int("takes", 10, "number of peer takes")
	duration := fs.Int("duration", 0, "duration in seconds")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *style == "" {
		return errors.New("--style is required")
	}
	body := map[string]any{"style_id": *style, "takes": *takes}
	if *duration > 0 {
		body["duration"] = *duration
	}
	return request(core, "POST", "/v1/compose", body)
}

func poolDraw(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("pool draw", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	style := fs.String("style", "", "style ID")
	holder := fs.String("holder", "cli", "reservation holder")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *style == "" {
		return errors.New("--style is required")
	}
	return request(core, "POST", "/v1/pool/draw", map[string]any{"style_id": *style, "holder": *holder})
}

func get(core *cliapp.ScenarioApp, path string) error {
	path = strings.TrimPrefix(path, "/v1")
	b, err := core.Get(path, nil)
	if err != nil {
		return err
	}
	return printJSON(b)
}

func oneArgGet(core *cliapp.ScenarioApp, args []string, prefix string, suffix ...string) error {
	if len(args) != 1 {
		return errors.New("an id is required")
	}
	path := prefix + args[0]
	for _, part := range suffix {
		path += part
	}
	return get(core, path)
}

func oneArgRequest(core *cliapp.ScenarioApp, args []string, method, prefix string, suffix ...string) error {
	if len(args) != 1 {
		return errors.New("an id is required")
	}
	path := prefix + args[0]
	for _, part := range suffix {
		path += part
	}
	return request(core, method, path, nil)
}

func transition(core *cliapp.ScenarioApp, args []string, action string) error {
	if len(args) != 1 {
		return errors.New("a take id is required")
	}
	return request(core, "POST", "/v1/takes/"+args[0]+action, map[string]any{"holder": "cli"})
}

func poolConfigure(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("pool configure", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	style := fs.String("style", "", "style ID")
	target := fs.Int("target", 10, "target depth")
	threshold := fs.Int("below", 3, "replenish below")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *style == "" {
		return errors.New("--style is required")
	}
	return request(core, "POST", "/v1/pool/configure", map[string]any{"style_id": *style, "target_depth": *target, "replenish_below": *threshold})
}

func listTakes(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("takes list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	job := fs.String("job", "", "job ID")
	style := fs.String("style", "", "style ID")
	if err := fs.Parse(args); err != nil {
		return err
	}
	path := "/v1/takes?style_id=" + url.QueryEscape(*style) + "&job_id=" + url.QueryEscape(*job)
	return get(core, path)
}
func request(core *cliapp.ScenarioApp, method, path string, body any) error {
	path = strings.TrimPrefix(path, "/v1")
	b, err := core.Request(method, path, nil, body)
	if err != nil {
		return err
	}
	return printJSON(b)
}
func printJSON(b []byte) error {
	var value any
	if err := json.Unmarshal(b, &value); err != nil {
		fmt.Print(string(b))
		return nil
	}
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(encoded))
	return nil
}
