package queue

import (
	"fmt"
	"os"
	"strings"

	"document-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `queue` subcommand group wrapping /api/queue and /api/queue/batch.
// Individual approve/reject flows go through `batch` with a single ID; the API exposes
// no per-item approve/reject endpoint (only batch).
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "queue",
		Description: "Inspect and operate on the improvement queue",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List queue items (sorted by severity)", Run: func(args []string) error { return runList(core, args) }},
			{Name: "create", Description: "Create a queue item from --body-file JSON", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "delete", Aliases: []string{"rm"}, Description: "Delete a queue item by ID", Run: func(args []string) error { return runDelete(core, args) }},
			{Name: "batch", Description: "Approve / reject / delete multiple queue items", Run: func(args []string) error { return runBatch(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("queue list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/queue", nil)
	if err != nil {
		return err
	}
	var items []support.QueueItem
	if err := support.Decode(body, &items); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Queue items: %d", len(items))},
		ResultsHeading: "Queue",
		Results:        queueRows(items),
		RetrievalHints: []string{
			fmt.Sprintf("%s queue batch --action approve --ids <id1>,<id2>", support.CLIName),
			fmt.Sprintf("%s agents list", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("queue create")
	bodyFile := fs.String("body-file", "", "Path to JSON request body, or '-' for stdin (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	raw, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/queue", nil, raw)
	if err != nil {
		return err
	}
	var created support.QueueItem
	if err := support.Decode(body, &created); err != nil {
		return err
	}

	message := support.EnvelopeMessage(body)
	if message == "" {
		message = "Queue item created"
	}
	changes := []string{}
	if created.ID != "" {
		changes = append(changes, fmt.Sprintf("ID: %s", created.ID))
	}
	if created.Title != "" {
		changes = append(changes, fmt.Sprintf("Title: %s", created.Title))
	}
	if created.Severity != "" {
		changes = append(changes, fmt.Sprintf("Severity: %s", created.Severity))
	}
	if created.Status != "" {
		changes = append(changes, fmt.Sprintf("Status: %s", created.Status))
	}

	report := cliapp.MutationReport{
		Result:      []string{message},
		Changes:     changes,
		NextCommand: []string{fmt.Sprintf("%s queue list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("queue delete")
	id := fs.String("id", "", "Queue item ID (or pass as positional argument)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	itemID := *id
	if itemID == "" && fs.NArg() >= 1 {
		itemID = fs.Arg(0)
	}
	if itemID == "" {
		return fmt.Errorf("usage: queue delete <id> | --id <id>")
	}

	query := support.BuildQuery(map[string]string{"id": itemID})
	body, err := core.Request("DELETE", "/queue", query, nil)
	if err != nil {
		return err
	}
	message := support.EnvelopeMessage(body)
	if message == "" {
		message = fmt.Sprintf("Queue item %s deleted", itemID)
	}

	report := cliapp.MutationReport{
		Result:      []string{message},
		Changes:     []string{fmt.Sprintf("Deleted queue item %s", itemID)},
		NextCommand: []string{fmt.Sprintf("%s queue list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runBatch(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("queue batch")
	action := fs.String("action", "", "Batch action: approve | reject | delete")
	idsFlag := fs.String("ids", "", "Comma-separated queue item IDs")
	bodyFile := fs.String("body-file", "", "Optional JSON request body (overrides --action/--ids); use '-' for stdin")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if *bodyFile != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if *action == "" || *idsFlag == "" {
			return fmt.Errorf("usage: queue batch --action approve|reject|delete --ids <id1>,<id2>,... | --body-file <path>")
		}
		ids := splitCSV(*idsFlag)
		if len(ids) == 0 {
			return fmt.Errorf("--ids must contain at least one id")
		}
		payload = map[string]interface{}{
			"action": *action,
			"ids":    ids,
		}
	}

	body, err := core.Request("POST", "/queue/batch", nil, payload)
	if err != nil {
		return err
	}
	var resp support.BatchQueueResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	verb := *action
	if verb == "" {
		verb = "batch"
	}
	report := cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Batch %s: %d succeeded, %d failed (of %d)",
			verb, len(resp.Succeeded), len(resp.Failed), resp.Total)},
		Changes: batchChanges(resp),
		NextCommand: []string{
			fmt.Sprintf("%s queue list", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func batchChanges(resp support.BatchQueueResponse) []string {
	changes := []string{}
	if len(resp.Succeeded) > 0 {
		changes = append(changes, fmt.Sprintf("Succeeded: %s", strings.Join(resp.Succeeded, ", ")))
	}
	if len(resp.Failed) > 0 {
		changes = append(changes, fmt.Sprintf("Failed: %s", strings.Join(resp.Failed, ", ")))
	}
	if len(changes) == 0 {
		changes = []string{"(no changes reported)"}
	}
	return changes
}

func queueRows(items []support.QueueItem) []string {
	if len(items) == 0 {
		return []string{"No items in the improvement queue"}
	}
	rows := make([]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, fmt.Sprintf("%s (%s) | %s | %s | agent=%s | app=%s",
			item.Title, support.ShortID(item.ID), item.Severity, item.Status, item.AgentName, item.ApplicationName))
	}
	return rows
}
