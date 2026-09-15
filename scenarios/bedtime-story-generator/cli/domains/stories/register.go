package stories

import (
	"fmt"
	"os"
	"strings"

	"bedtime-story-generator/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `stories` subcommand group covering the full story
// lifecycle exposed by /api/v1/stories. The API owns all generation, storage,
// favorite toggling, and deletion; this package is a thin wrapper that formats
// responses through the standard output contracts.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "stories",
		Description: "Generate, read, and manage bedtime stories",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List saved stories", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"read", "show"}, Description: "Read a specific story", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "generate", Aliases: []string{"create"}, Description: "Generate a new bedtime story", Run: func(args []string) error { return runGenerate(core, args) }},
			{Name: "favorite", Aliases: []string{"fav"}, Description: "Toggle a story's favorite flag", Run: func(args []string) error { return runFavorite(core, args) }},
			{Name: "delete", Aliases: []string{"rm"}, Description: "Delete a story", Run: func(args []string) error { return runDelete(core, args) }},
			{Name: "export", Description: "Export a story as PDF", Run: func(args []string) error { return runExport(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("stories list")
	ageGroup := fs.String("age-group", "", "Filter by age group (3-5, 6-8, 9-12)")
	favoritesOnly := fs.Bool("favorites", false, "Show only stories marked as favorite")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/stories", nil)
	if err != nil {
		return err
	}
	var stories []support.Story
	if err := support.Decode(body, &stories); err != nil {
		return err
	}

	filtered := filterStories(stories, *ageGroup, *favoritesOnly)

	summary := []string{fmt.Sprintf("Stories: %d (of %d total)", len(filtered), len(stories))}
	if *ageGroup != "" {
		summary = append(summary, fmt.Sprintf("Age group: %s", *ageGroup))
	}
	if *favoritesOnly {
		summary = append(summary, "Showing favorites only")
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Library",
		Results:        storyRows(filtered),
		RetrievalHints: []string{
			fmt.Sprintf("%s stories get <story-id>", support.CLIName),
			fmt.Sprintf("%s stories list --favorites", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("stories get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: stories get <story-id>")
	}
	id := fs.Arg(0)

	// Record the read in the API; ignore body. Any error here is surfaced to
	// avoid masking connectivity problems.
	if _, err := core.Request("POST", "/stories/"+id+"/read", nil, nil); err != nil {
		return err
	}

	body, err := core.Get("/stories/"+id, nil)
	if err != nil {
		return err
	}
	var story support.Story
	if err := support.Decode(body, &story); err != nil {
		return err
	}

	results := storyDetails(story)
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%s (%s)", story.Title, story.AgeGroup)},
		ResultsHeading: "Story",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s stories favorite %s", support.CLIName, story.ID),
			fmt.Sprintf("%s stories export %s --output story.pdf", support.CLIName, story.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGenerate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("stories generate")
	ageGroup := fs.String("age-group", "6-8", "Age group (3-5, 6-8, 9-12)")
	theme := fs.String("theme", "Adventure", "Story theme")
	length := fs.String("length", "medium", "Length (short, medium, long)")
	characters := fs.String("characters", "", "Comma-separated character names")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	req := support.GenerateStoryRequest{
		AgeGroup:       strings.TrimSpace(*ageGroup),
		Theme:          strings.TrimSpace(*theme),
		Length:         strings.TrimSpace(*length),
		CharacterNames: support.SplitCSV(*characters),
	}

	body, err := core.Request("POST", "/stories/generate", nil, req)
	if err != nil {
		return err
	}
	var story support.Story
	if err := support.Decode(body, &story); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Title: %s", story.Title),
			fmt.Sprintf("Story ID: %s", story.ID),
			fmt.Sprintf("Reading time: %d minutes", story.ReadingTime),
			fmt.Sprintf("Pages: %d", story.PageCount),
		},
		Changes: []string{
			fmt.Sprintf("Created story %s (age=%s, theme=%s, length=%s)",
				support.ShortID(story.ID), req.AgeGroup, req.Theme, req.Length),
		},
		NextCommand: []string{
			fmt.Sprintf("%s stories get %s", support.CLIName, story.ID),
			fmt.Sprintf("%s stories favorite %s", support.CLIName, story.ID),
			fmt.Sprintf("%s stories export %s --output story.pdf", support.CLIName, story.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runFavorite(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("stories favorite")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: stories favorite <story-id>")
	}
	id := fs.Arg(0)

	body, err := core.Request("POST", "/stories/"+id+"/favorite", nil, nil)
	if err != nil {
		return err
	}
	var resp support.FavoriteResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	state := "unfavorited"
	if resp.IsFavorite {
		state = "favorited"
	}
	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Story %s is now %s", id, state)},
		Changes:     []string{fmt.Sprintf("is_favorite=%t", resp.IsFavorite)},
		NextCommand: []string{fmt.Sprintf("%s stories list --favorites", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("stories delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: stories delete <story-id>")
	}
	id := fs.Arg(0)

	// DELETE returns 204 No Content; body is empty and support.Decode is not invoked.
	if _, err := core.Request("DELETE", "/stories/"+id, nil, nil); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Story %s deleted", id)},
		Changes:     []string{fmt.Sprintf("Removed story %s", id)},
		NextCommand: []string{fmt.Sprintf("%s stories list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runExport(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("stories export")
	output := fs.String("output", "", "File path to write the PDF (defaults to <story-id>.pdf)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: stories export <story-id> [--output path.pdf]")
	}
	id := fs.Arg(0)

	body, err := core.Get("/stories/"+id+"/export", nil)
	if err != nil {
		return err
	}

	path := strings.TrimSpace(*output)
	if path == "" {
		path = id + ".pdf"
	}
	if err := support.WriteOutput(path, body); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Exported story %s", id), fmt.Sprintf("Wrote %d bytes to %s", len(body), path)},
		Changes:     []string{fmt.Sprintf("Created PDF at %s", path)},
		NextCommand: []string{fmt.Sprintf("%s stories get %s", support.CLIName, id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func filterStories(stories []support.Story, ageGroup string, favoritesOnly bool) []support.Story {
	if ageGroup == "" && !favoritesOnly {
		return stories
	}
	ageGroup = strings.TrimSpace(ageGroup)
	out := make([]support.Story, 0, len(stories))
	for _, s := range stories {
		if favoritesOnly && !s.IsFavorite {
			continue
		}
		if ageGroup != "" && s.AgeGroup != ageGroup {
			continue
		}
		out = append(out, s)
	}
	return out
}

func storyRows(stories []support.Story) []string {
	if len(stories) == 0 {
		return []string{"No stories found"}
	}
	rows := make([]string, 0, len(stories))
	for _, s := range stories {
		fav := " "
		if s.IsFavorite {
			fav = "*"
		}
		title := s.Title
		if len(title) > 48 {
			title = title[:45] + "..."
		}
		rows = append(rows, fmt.Sprintf("%s %s %-48s | age=%s | theme=%s | %dmin",
			fav, support.ShortID(s.ID), title, s.AgeGroup, s.Theme, s.ReadingTime))
	}
	return rows
}

func storyDetails(s support.Story) []string {
	rows := []string{
		fmt.Sprintf("ID: %s", s.ID),
		fmt.Sprintf("Title: %s", s.Title),
		fmt.Sprintf("Age group: %s", s.AgeGroup),
		fmt.Sprintf("Theme: %s", s.Theme),
		fmt.Sprintf("Length: %s", s.StoryLength),
		fmt.Sprintf("Reading time: %d minutes", s.ReadingTime),
		fmt.Sprintf("Pages: %d", s.PageCount),
		fmt.Sprintf("Times read: %d", s.TimesRead),
		fmt.Sprintf("Favorite: %t", s.IsFavorite),
		fmt.Sprintf("Created: %s", support.FormatTimeValue(s.CreatedAt)),
	}
	if s.LastRead != nil {
		rows = append(rows, fmt.Sprintf("Last read: %s", support.FormatTimeValue(*s.LastRead)))
	}
	if len(s.CharacterNames) > 0 {
		rows = append(rows, fmt.Sprintf("Characters: %s", strings.Join(s.CharacterNames, ", ")))
	}
	if strings.TrimSpace(s.Content) != "" {
		rows = append(rows, "", "--- Content ---", s.Content)
	}
	return rows
}
