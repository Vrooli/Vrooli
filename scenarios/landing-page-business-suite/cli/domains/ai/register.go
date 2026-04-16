package ai

import (
	"flag"
	"fmt"

	"landing-page-business-suite/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	commands := deps.EndpointCommands([]support.EndpointDef{
		{Name: "ai-models", Method: "GET", Path: "/ai/models", Description: "List AI models"},
		{Name: "ai-health", Method: "GET", Path: "/ai/health", Description: "AI health"},
		{Name: "ai-chat", Method: "POST", Path: "/ai/chat", Description: "Run AI chat completion"},
		{Name: "ai-usage", Method: "GET", Path: "/ai/usage", Description: "Get AI usage"},
	})
	commands = append(commands, cliapp.Command{
		Name:        "ai-stream",
		NeedsAPI:    true,
		Description: "Stream AI chat completion",
		Run:         func(args []string) error { return runStream(deps, args) },
	})
	return cliapp.CommandGroup{Title: "AI Gateway", Commands: commands}
}

func runStream(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("ai-stream", flag.ContinueOnError)
	body := fs.String("body", "", "JSON body payload or @file.json")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	payload, err := support.ParseBody(*body)
	if err != nil {
		return err
	}
	if payload == nil {
		return fmt.Errorf("usage: ai-stream --body '{...}'")
	}
	return deps.StreamEndpoint(support.EndpointDef{Method: "POST", Path: "/ai/stream"}, "/ai/stream", nil, payload)
}
