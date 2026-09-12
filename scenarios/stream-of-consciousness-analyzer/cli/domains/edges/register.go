package edges

import (
	"flag"
	"fmt"
	"os"

	"stream-of-consciousness-analyzer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "stream-of-consciousness-analyzer"

type edge struct {
	ID       string `json:"id"`
	SourceID string `json:"source_id"`
	TargetID string `json:"target_id"`
	Label    string `json:"label"`
}

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "edge",
		Description: "Manage graph edges",
		Subcommands: []cliapp.Command{
			{Name: "list", NeedsAPI: true, Description: "List edges for a thought", Run: func(args []string) error { return runList(core, args) }},
			{Name: "create", NeedsAPI: true, Description: "Create an edge", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "delete", NeedsAPI: true, Description: "Delete an edge", Run: func(args []string) error { return runDelete(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs, jsonOut, err := support.ParseFlags("edge list", args)
	if err != nil {
		return err
	}
	if err := support.RequireArg(fs, "edge list <thought-id> [--json]"); err != nil {
		return err
	}

	body, err := core.Get("/thoughts/"+fs.Arg(0)+"/edges", nil)
	if err != nil {
		return err
	}

	var edges []edge
	if err := support.Unmarshal(body, &edges); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{
			"Parent thought: " + fs.Arg(0),
			fmt.Sprintf("Total edges: %d", len(edges)),
		},
		Results: renderList(edges),
		RetrievalHints: []string{
			cliName + " edge create " + fs.Arg(0) + " --target <thought-id>",
			cliName + " thought get " + fs.Arg(0),
		},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("edge create", flag.ContinueOnError)
	target := fs.String("target", "", "Target thought ID (required)")
	label := fs.String("label", "", "Edge label")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := support.RequireArg(fs, "edge create <source-thought-id> --target TARGET_ID [--label LABEL]"); err != nil {
		return err
	}
	if *target == "" {
		return fmt.Errorf("--target is required")
	}

	body, err := core.Request("POST", "/thoughts/"+fs.Arg(0)+"/edges", nil, map[string]string{
		"target_id": *target,
		"label":     *label,
	})
	if err != nil {
		return err
	}

	var item edge
	if err := support.Unmarshal(body, &item); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{"Edge created", "Edge ID: " + item.ID},
		Changes: []string{
			"Source: " + item.SourceID,
			"Target: " + item.TargetID,
			"Label: " + item.Label,
		},
		NextCommand: []string{
			cliName + " edge list " + fs.Arg(0),
			cliName + " thought get " + *target,
		},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("edge delete", flag.ContinueOnError)
	thoughtID := fs.String("thought", "", "Parent thought ID (required)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 || *thoughtID == "" {
		return fmt.Errorf("usage: edge delete <edge-id> --thought THOUGHT_ID")
	}

	if _, err := core.Request("DELETE", "/thoughts/"+*thoughtID+"/edges/"+fs.Arg(0), nil, nil); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Edge deleted", "Edge ID: " + fs.Arg(0)},
		Changes:     []string{"Parent thought: " + *thoughtID},
		NextCommand: []string{cliName + " edge list " + *thoughtID},
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func renderList(edges []edge) []string {
	lines := make([]string, 0, len(edges))
	for _, item := range edges {
		lines = append(lines, fmt.Sprintf("%s  %s -> %s  [%s]", item.ID, support.ShortID(item.SourceID), support.ShortID(item.TargetID), item.Label))
	}
	return lines
}
