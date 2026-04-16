package model

import (
	"fmt"

	"agent-inbox/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

type modelRecord struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Context     int    `json:"context_length"`
	Pricing     struct {
		Prompt     string `json:"prompt"`
		Completion string `json:"completion"`
	} `json:"pricing"`
}

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "model",
		Description: "Available LLM models",
		Subcommands: []cliapp.Command{
			{Name: "list", NeedsAPI: true, Description: "List available models", Run: func(args []string) error { return RunList(core, args) }},
		},
	}
}

func RunList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("model list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var models []modelRecord
	if err := support.GetJSON(core, "/models", &models); err != nil {
		return err
	}

	results := make([]string, 0, len(models))
	for _, model := range models {
		name := model.ID
		if model.Name != "" {
			name = model.Name + " (" + model.ID + ")"
		}
		line := name
		if model.Context > 0 {
			line += fmt.Sprintf(" | context=%d", model.Context)
		}
		if model.Description != "" {
			line += "\n  " + support.Truncate(model.Description, 100)
		}
		results = append(results, line)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Models: %d", len(models))},
		ResultsHeading: "Models",
		Results:        results,
		RetrievalHints: []string{support.CLIName + " chat create --model <model-id>", support.CLIName + " chat update <chat-id> --model <model-id>"},
	}
	return support.PrintList(*jsonOutput, report)
}
