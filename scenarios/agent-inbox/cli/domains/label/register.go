package label

import (
	"agent-inbox/cli/internal/support"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

type labelRecord struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	CreatedAt string `json:"created_at"`
}

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "label",
		Description: "Label management for chats",
		Subcommands: []cliapp.Command{
			{Name: "list", NeedsAPI: true, Description: "List labels", Run: func(args []string) error { return RunList(core, args) }},
			{Name: "create", NeedsAPI: true, Description: "Create a label", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "assign", NeedsAPI: true, Description: "Assign a label to a chat", Run: func(args []string) error { return runAssign(core, args) }},
			{Name: "remove", NeedsAPI: true, Description: "Remove a label from a chat", Run: func(args []string) error { return runRemove(core, args) }},
		},
	}
}

func RunList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("label list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var labels []labelRecord
	if err := support.GetJSON(core, "/labels", &labels); err != nil {
		return err
	}

	results := make([]string, 0, len(labels))
	for _, item := range labels {
		results = append(results, fmt.Sprintf("%s | %s | %s", item.ID, item.Name, item.Color))
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Labels: %d", len(labels))},
		ResultsHeading: "Labels",
		Results:        results,
		RetrievalHints: []string{support.CLIName + " label create --name urgent --color '#ef4444'", support.CLIName + " label assign <chat-id> <label-id>"},
	}
	return support.PrintList(*jsonOutput, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("label create")
	name := fs.String("name", "", "Label name")
	color := fs.String("color", "", "Label color hex")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" {
		return fmt.Errorf("--name is required")
	}

	input := map[string]interface{}{"name": strings.TrimSpace(*name)}
	if strings.TrimSpace(*color) != "" {
		input["color"] = strings.TrimSpace(*color)
	}

	body, err := core.Request("POST", "/labels", nil, input)
	if err != nil {
		return err
	}
	var label labelRecord
	if err := support.Decode(body, &label); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Label created", "Label ID: " + label.ID},
		Changes:     []string{"Name: " + label.Name, "Color: " + label.Color},
		NextCommand: []string{support.CLIName + " label list"},
	}
	return support.PrintMutation(*jsonOutput, report)
}

func runAssign(core *cliapp.ScenarioApp, args []string) error {
	return runAssignment(core, "assign", "PUT", true, args)
}

func runRemove(core *cliapp.ScenarioApp, args []string) error {
	return runAssignment(core, "remove", "DELETE", false, args)
}

func runAssignment(core *cliapp.ScenarioApp, name, method string, assigned bool, args []string) error {
	fs := support.NewFlagSet("label " + name)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: label %s <chat-id> <label-id> [--json]", name)
	}
	chatID := fs.Arg(0)
	labelID := fs.Arg(1)

	if _, err := core.Request(method, "/chats/"+chatID+"/labels/"+labelID, nil, nil); err != nil {
		return err
	}

	action := "removed"
	if assigned {
		action = "assigned"
	}
	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Label %s", action)},
		Changes:     []string{"Chat ID: " + chatID, "Label ID: " + labelID},
		NextCommand: []string{support.CLIName + " chat get " + chatID},
	}
	return support.PrintMutation(*jsonOutput, report)
}
