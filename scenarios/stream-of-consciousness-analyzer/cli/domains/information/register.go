package information

import (
	"flag"
	"fmt"
	"os"

	"stream-of-consciousness-analyzer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "stream-of-consciousness-analyzer"

type item struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Content string `json:"content"`
}

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "info",
		Description: "Manage supporting information",
		Subcommands: []cliapp.Command{
			{Name: "list", NeedsAPI: true, Description: "List information items", Run: func(args []string) error { return runList(core, args) }},
			{Name: "create", NeedsAPI: true, Description: "Create an information item", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "update", NeedsAPI: true, Description: "Update an information item", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "delete", NeedsAPI: true, Description: "Delete an information item", Run: func(args []string) error { return runDelete(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs, jsonOut, err := support.ParseFlags("info list", args)
	if err != nil {
		return err
	}
	if err := support.RequireArg(fs, "info list <scheme-id> [--json]"); err != nil {
		return err
	}

	body, err := core.Get("/schemes/"+fs.Arg(0)+"/information", nil)
	if err != nil {
		return err
	}

	var items []item
	if err := support.Unmarshal(body, &items); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{
			"Scheme: " + fs.Arg(0),
			fmt.Sprintf("Information items: %d", len(items)),
		},
		Results: renderList(items),
		RetrievalHints: []string{
			cliName + " info create --scheme " + fs.Arg(0) + " --content \"...\"",
			cliName + " scheme export " + fs.Arg(0),
		},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("info create", flag.ContinueOnError)
	schemeID := fs.String("scheme", "", "Scheme ID (required)")
	infoType := fs.String("type", "text", "Information type")
	content := fs.String("content", "", "Content (required)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *schemeID == "" || *content == "" {
		return fmt.Errorf("usage: info create --scheme ID --content TEXT [--type TYPE]")
	}

	body, err := core.Request("POST", "/schemes/"+*schemeID+"/information", nil, map[string]any{
		"type":    *infoType,
		"content": *content,
	})
	if err != nil {
		return err
	}

	var created item
	if err := support.Unmarshal(body, &created); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{"Information item created", "Item ID: " + created.ID},
		Changes: []string{
			"Scheme: " + *schemeID,
			"Type: " + *infoType,
			"Content: " + support.Truncate(*content, 80),
		},
		NextCommand: []string{
			cliName + " info list " + *schemeID,
			cliName + " scheme export " + *schemeID,
		},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("info update", flag.ContinueOnError)
	schemeID := fs.String("scheme", "", "Scheme ID (required)")
	content := fs.String("content", "", "New content")
	infoType := fs.String("type", "", "New type")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 || *schemeID == "" {
		return fmt.Errorf("usage: info update <info-id> --scheme SCHEME_ID [--content TEXT] [--type TYPE]")
	}
	input := map[string]any{}
	if *content != "" {
		input["content"] = *content
	}
	if *infoType != "" {
		input["type"] = *infoType
	}
	if len(input) == 0 {
		return fmt.Errorf("at least one of --content or --type is required")
	}

	if _, err := core.Request("PUT", "/schemes/"+*schemeID+"/information/"+fs.Arg(0), nil, input); err != nil {
		return err
	}

	changes := []string{"Scheme: " + *schemeID}
	if *content != "" {
		changes = append(changes, "Content: "+support.Truncate(*content, 80))
	}
	if *infoType != "" {
		changes = append(changes, "Type: "+*infoType)
	}
	report := cliapp.MutationReport{
		Result:      []string{"Information item updated", "Item ID: " + fs.Arg(0)},
		Changes:     changes,
		NextCommand: []string{cliName + " info list " + *schemeID},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("info delete", flag.ContinueOnError)
	schemeID := fs.String("scheme", "", "Scheme ID (required)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 || *schemeID == "" {
		return fmt.Errorf("usage: info delete <info-id> --scheme SCHEME_ID")
	}

	if _, err := core.Request("DELETE", "/schemes/"+*schemeID+"/information/"+fs.Arg(0), nil, nil); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Information item deleted", "Item ID: " + fs.Arg(0)},
		Changes:     []string{"Scheme: " + *schemeID},
		NextCommand: []string{cliName + " info list " + *schemeID},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func renderList(items []item) []string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, fmt.Sprintf("%s  [%s] %s", item.ID, item.Type, support.Truncate(item.Content, 60)))
	}
	return lines
}
