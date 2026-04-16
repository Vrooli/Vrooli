package notes

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "reference-react-vite"

type noteResponse struct {
	ID        string `json:"id"`
	TaskID    string `json:"task_id"`
	Content   string `json:"content"`
	Author    string `json:"author,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type listResponse struct {
	Items  json.RawMessage `json:"items"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "note",
		Description: "Note operations",
		Subcommands: []cliapp.Command{
			{Name: "list", NeedsAPI: true, Description: "List notes for a task", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", NeedsAPI: true, Description: "Get note by ID", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "create", NeedsAPI: true, Description: "Create a new note on a task", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "update", NeedsAPI: true, Description: "Update an existing note", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "delete", NeedsAPI: true, Description: "Delete a note", Run: func(args []string) error { return runDelete(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("note list", flag.ContinueOnError)
	taskID := fs.String("task", "", "Task ID to list notes for (required)")
	limit := fs.Int("limit", 20, "Maximum number of notes to return")
	offset := fs.Int("offset", 0, "Number of notes to skip")
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *taskID == "" {
		return fmt.Errorf("--task is required")
	}

	query := url.Values{}
	query.Set("limit", strconv.Itoa(*limit))
	query.Set("offset", strconv.Itoa(*offset))

	body, err := core.Get("/tasks/"+*taskID+"/notes", query)
	if err != nil {
		return err
	}

	var resp listResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	var notes []noteResponse
	if err := json.Unmarshal(resp.Items, &notes); err != nil {
		return fmt.Errorf("parse notes: %w", err)
	}

	report := cliapp.ListReport{
		Summary: []string{
			"Task: " + *taskID,
			fmt.Sprintf("Total notes: %d", resp.Total),
			fmt.Sprintf("Window: %d-%d", resp.Offset+1, resp.Offset+len(notes)),
		},
		Results:        renderNoteRows(notes),
		RetrievalHints: []string{cliName + " note get <note-id>", cliName + " note create --task " + *taskID + " --content \"...\""},
	}
	if len(notes) == 0 {
		report.Summary[2] = "Window: 0-0"
	}

	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("note get", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: note get <id> [--json]")
	}
	id := fs.Arg(0)

	body, err := core.Get("/notes/"+id, nil)
	if err != nil {
		return err
	}

	var note noteResponse
	if err := json.Unmarshal(body, &note); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Note: %s", note.ID), "Task: " + note.TaskID},
		ResultsHeading: "Details",
		Results:        noteDetails(note),
		RetrievalHints: []string{cliName + " note update " + note.ID + " --content \"...\"", cliName + " note list --task " + note.TaskID},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("note create", flag.ContinueOnError)
	taskID := fs.String("task", "", "Task ID to attach note to (required)")
	content := fs.String("content", "", "Note content (required)")
	author := fs.String("author", "", "Note author")
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *taskID == "" {
		return fmt.Errorf("--task is required")
	}
	if *content == "" {
		return fmt.Errorf("--content is required")
	}

	input := map[string]interface{}{"content": *content}
	if *author != "" {
		input["author"] = *author
	}

	body, err := core.Request("POST", "/tasks/"+*taskID+"/notes", nil, input)
	if err != nil {
		return err
	}

	var note noteResponse
	if err := json.Unmarshal(body, &note); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := cliapp.MutationReport{
		Result: []string{"Note created", "Note ID: " + note.ID},
		Changes: []string{
			"Task: " + note.TaskID,
			"Content preview: " + preview(note.Content),
		},
		NextCommand: []string{cliName + " note get " + note.ID, cliName + " note list --task " + note.TaskID},
	}
	if note.Author != "" {
		report.Changes = append(report.Changes, "Author: "+note.Author)
	}

	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("note update", flag.ContinueOnError)
	content := fs.String("content", "", "New note content")
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: note update <id> --content CONTENT [--json]")
	}
	id := fs.Arg(0)
	if *content == "" {
		return fmt.Errorf("--content is required")
	}

	input := map[string]interface{}{"content": *content}

	body, err := core.Request("PATCH", "/notes/"+id, nil, input)
	if err != nil {
		return err
	}

	var note noteResponse
	if err := json.Unmarshal(body, &note); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	report := cliapp.MutationReport{
		Result: []string{"Note updated", "Note ID: " + note.ID},
		Changes: []string{
			"Task: " + note.TaskID,
			"Content preview: " + preview(note.Content),
		},
		NextCommand: []string{cliName + " note get " + note.ID, cliName + " note list --task " + note.TaskID},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("note delete", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: note delete <id>")
	}
	id := fs.Arg(0)

	if _, err := core.Request("DELETE", "/notes/"+id, nil, nil); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{"Note deleted", "Note ID: " + id},
		Changes:     []string{"Removed note from its task timeline"},
		NextCommand: []string{cliName + " task list", cliName + " note list --task <task-id>"},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func renderNoteRows(notes []noteResponse) []string {
	if len(notes) == 0 {
		return nil
	}
	rows := make([]string, 0, len(notes))
	for _, note := range notes {
		author := note.Author
		if author == "" {
			author = "anonymous"
		}
		rows = append(rows, fmt.Sprintf("%s [%s] %s", shortID(note.ID), author, preview(note.Content)))
	}
	return rows
}

func noteDetails(note noteResponse) []string {
	lines := []string{
		"Task: " + note.TaskID,
		"Content: " + note.Content,
		"Created: " + note.CreatedAt,
		"Updated: " + note.UpdatedAt,
	}
	if note.Author != "" {
		lines = append(lines, "Author: "+note.Author)
	}
	return lines
}

func preview(content string) string {
	if len(content) <= 50 {
		return content
	}
	return content[:47] + "..."
}

func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}
