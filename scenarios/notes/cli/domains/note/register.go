package note

import (
	"fmt"
	"os"
	"strings"

	"notes/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `note` subcommand group covering CRUD on /api/notes.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "note",
		Description: "Create, list, read, update, and delete notes",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List notes", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show", "view"}, Description: "Show one note", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "create", Aliases: []string{"new", "add"}, Description: "Create a new note", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "update", Aliases: []string{"edit"}, Description: "Update an existing note", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "delete", Aliases: []string{"rm", "remove"}, Description: "Delete a note", Run: func(args []string) error { return runDelete(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("note list")
	userID := fs.String("user-id", "", "Filter by user ID (server-side)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{
		"user_id": *userID,
	})
	body, err := core.Get("/notes", query)
	if err != nil {
		return err
	}

	var notes []support.Note
	if err := support.Decode(body, &notes); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Notes: %d", len(notes))},
		ResultsHeading: "Notes",
		Results:        noteRows(notes),
		RetrievalHints: []string{
			fmt.Sprintf("%s note get <note-id>", support.CLIName),
			fmt.Sprintf("%s search text --query \"...\"", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("note get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: note get <note-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/notes/"+id, nil)
	if err != nil {
		return err
	}
	var note support.Note
	if err := support.Decode(body, &note); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("ID: %s", note.ID),
		fmt.Sprintf("Title: %s", note.Title),
	}
	if note.ContentType != "" {
		results = append(results, fmt.Sprintf("Type: %s", note.ContentType))
	}
	if note.WordCount > 0 {
		results = append(results, fmt.Sprintf("Words: %d", note.WordCount))
	}
	if note.ReadingTime > 0 {
		results = append(results, fmt.Sprintf("Reading time: %d min", note.ReadingTime))
	}
	if !note.CreatedAt.IsZero() {
		results = append(results, fmt.Sprintf("Created: %s", support.FormatTimeValue(note.CreatedAt)))
	}
	if !note.UpdatedAt.IsZero() {
		results = append(results, fmt.Sprintf("Updated: %s", support.FormatTimeValue(note.UpdatedAt)))
	}
	if note.IsPinned {
		results = append(results, "Pinned: true")
	}
	if note.IsFavorite {
		results = append(results, "Favorite: true")
	}
	if note.IsArchived {
		results = append(results, "Archived: true")
	}
	if len(note.Tags) > 0 {
		results = append(results, fmt.Sprintf("Tags: %s", strings.Join(note.Tags, ", ")))
	}
	if note.Content != "" {
		results = append(results, "", "Content:", note.Content)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Note: %s", note.Title)},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s note update %s --title \"...\"", support.CLIName, note.ID),
			fmt.Sprintf("%s note delete %s", support.CLIName, note.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("note create")
	title := fs.String("title", "", "Note title (required)")
	content := fs.String("content", "", "Note content")
	contentType := fs.String("content-type", "", "Content type (markdown|text|html)")
	folderID := fs.String("folder-id", "", "Folder ID to place the note into")
	userID := fs.String("user-id", "", "Owner user ID")
	bodyFile := fs.String("body-file", "", "Path to JSON file with the full create payload (overrides flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if strings.TrimSpace(*title) == "" {
			return fmt.Errorf("--title is required (or use --body-file)")
		}
		body := map[string]interface{}{
			"title": *title,
		}
		if strings.TrimSpace(*content) != "" {
			body["content"] = *content
		}
		if strings.TrimSpace(*contentType) != "" {
			body["content_type"] = *contentType
		}
		if strings.TrimSpace(*folderID) != "" {
			body["folder_id"] = *folderID
		}
		if strings.TrimSpace(*userID) != "" {
			body["user_id"] = *userID
		}
		payload = body
	}

	respBody, err := core.Request("POST", "/notes", nil, payload)
	if err != nil {
		return err
	}

	var created support.Note
	if err := support.Decode(respBody, &created); err != nil {
		return err
	}

	changes := []string{}
	if created.Title != "" {
		changes = append(changes, fmt.Sprintf("Title: %s", created.Title))
	}
	if created.ID != "" {
		changes = append(changes, fmt.Sprintf("ID: %s", created.ID))
	}

	next := []string{}
	if created.ID != "" {
		next = append(next, fmt.Sprintf("%s note get %s", support.CLIName, created.ID))
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Created note %q", created.Title)},
		Changes:     changes,
		NextCommand: next,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("note update")
	title := fs.String("title", "", "New title")
	content := fs.String("content", "", "New content")
	folderID := fs.String("folder-id", "", "New folder ID (use empty string to clear via --body-file)")
	pinned := fs.String("pinned", "", "Set pinned (true|false)")
	favorite := fs.String("favorite", "", "Set favorite (true|false)")
	bodyFile := fs.String("body-file", "", "Path to JSON file with the full update payload (overrides flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: note update <note-id> [--title ...] [--content ...] [--folder-id ...] [--pinned true|false] [--favorite true|false] [--body-file PATH]")
	}
	id := fs.Arg(0)

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		body := map[string]interface{}{}
		if strings.TrimSpace(*title) != "" {
			body["title"] = *title
		}
		if strings.TrimSpace(*content) != "" {
			body["content"] = *content
		}
		if strings.TrimSpace(*folderID) != "" {
			body["folder_id"] = *folderID
		}
		if b, ok := parseBool(*pinned); ok {
			body["is_pinned"] = b
		}
		if b, ok := parseBool(*favorite); ok {
			body["is_favorite"] = b
		}
		if len(body) == 0 {
			return fmt.Errorf("no fields to update — pass at least one of --title, --content, --folder-id, --pinned, --favorite, or --body-file")
		}
		payload = body
	}

	if _, err := core.Request("PUT", "/notes/"+id, nil, payload); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Updated note %s", id)},
		NextCommand: []string{fmt.Sprintf("%s note get %s", support.CLIName, id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("note delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: note delete <note-id>")
	}
	id := fs.Arg(0)

	if _, err := core.Request("DELETE", "/notes/"+id, nil, nil); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Deleted note %s", id)},
		NextCommand: []string{fmt.Sprintf("%s note list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func parseBool(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "t", "yes", "y", "1":
		return true, true
	case "false", "f", "no", "n", "0":
		return false, true
	default:
		return false, false
	}
}

func noteRows(notes []support.Note) []string {
	if len(notes) == 0 {
		return []string{"No notes found"}
	}
	rows := make([]string, 0, len(notes))
	for _, n := range notes {
		updated := support.FormatTimeValue(n.UpdatedAt)
		row := fmt.Sprintf("%s | %s | %d words | updated=%s",
			support.ShortID(n.ID), n.Title, n.WordCount, updated)
		if n.IsPinned {
			row += " (pinned)"
		}
		rows = append(rows, row)
	}
	return rows
}
