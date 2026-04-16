package posts

import (
	"fmt"
	"os"
	"strings"

	"social-media-scheduler/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `post` subcommand group wrapping /api/v1/posts/*.
// Operations with complex nested payloads (update, platform variants)
// accept --body-file; the simple schedule case is also exposed via flags.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "post",
		Description: "Manage scheduled social media posts",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "schedule", Description: "Schedule a new post (flags or --body-file)", Run: func(args []string) error { return runSchedule(core, args) }},
			{Name: "list", Aliases: []string{"calendar", "ls"}, Description: "List posts on the scheduling calendar", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show a single post", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "update", Description: "Update a post (--body-file PATH)", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "delete", Aliases: []string{"rm"}, Description: "Delete a post", Run: func(args []string) error { return runDelete(core, args) }},
			{Name: "optimize", Description: "Re-run AI optimization for a post", Run: func(args []string) error { return runAction(core, args, "optimize", "Optimize") }},
			{Name: "duplicate", Description: "Duplicate a post", Run: func(args []string) error { return runAction(core, args, "duplicate", "Duplicate") }},
			{Name: "preview", Description: "Preview how a post will render per platform", Run: func(args []string) error { return runPreview(core, args) }},
		},
	}
}

func runSchedule(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("post schedule")
	title := fs.String("title", "", "Post title")
	content := fs.String("content", "", "Post body content")
	platforms := fs.String("platforms", "", "Comma-separated list (twitter,instagram,linkedin,facebook)")
	scheduledAt := fs.String("scheduled-at", "", "RFC3339 time, e.g. 2026-07-15T14:00:00Z")
	timezone := fs.String("timezone", "UTC", "IANA timezone for the scheduled_at value")
	campaignID := fs.String("campaign", "", "Campaign ID to associate the post with")
	mediaCSV := fs.String("media", "", "Comma-separated media file URLs")
	autoOptimize := fs.Bool("auto-optimize", true, "Let the API generate platform-specific variants")
	bodyFile := fs.String("body-file", "", "Optional JSON file with the full schedule payload (overrides flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if raw, err := support.ReadJSONFile(*bodyFile, false); err != nil {
		return err
	} else if raw != nil {
		payload = raw
	} else {
		if strings.TrimSpace(*title) == "" || strings.TrimSpace(*content) == "" ||
			strings.TrimSpace(*platforms) == "" || strings.TrimSpace(*scheduledAt) == "" {
			return fmt.Errorf("usage: post schedule --title <t> --content <c> --platforms <csv> --scheduled-at <rfc3339> [--timezone UTC] [--campaign <id>] [--media <csv>] [--auto-optimize=true|false]")
		}
		body := map[string]interface{}{
			"title":         *title,
			"content":       *content,
			"platforms":     support.SplitCSV(*platforms),
			"scheduled_at":  *scheduledAt,
			"timezone":      *timezone,
			"auto_optimize": *autoOptimize,
		}
		if strings.TrimSpace(*campaignID) != "" {
			body["campaign_id"] = *campaignID
		}
		if media := support.SplitCSV(*mediaCSV); len(media) > 0 {
			body["media_files"] = media
		}
		payload = body
	}

	respBody, err := core.Request("POST", "/posts/schedule", nil, payload)
	if err != nil {
		return err
	}
	var post support.ScheduledPost
	if err := support.Decode(respBody, &post); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Scheduled post: %s", post.Title),
			fmt.Sprintf("ID: %s", post.ID),
			fmt.Sprintf("Scheduled for: %s", support.FormatTimeValue(post.ScheduledAt)),
		},
		Changes: []string{fmt.Sprintf("Platforms: %s", strings.Join(post.Platforms, ", "))},
		NextCommand: []string{
			fmt.Sprintf("%s post get %s", support.CLIName, post.ID),
			fmt.Sprintf("%s post preview %s", support.CLIName, post.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("post list")
	startDate := fs.String("start-date", "", "Earliest scheduled_at filter (RFC3339 or date)")
	endDate := fs.String("end-date", "", "Latest scheduled_at filter (RFC3339 or date)")
	platforms := fs.String("platforms", "", "Comma-separated platform filter")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{
		"start_date": *startDate,
		"end_date":   *endDate,
	})
	for _, p := range support.SplitCSV(*platforms) {
		query.Add("platforms", p)
	}

	body, err := core.Get("/posts/calendar", query)
	if err != nil {
		return err
	}
	var posts []support.ScheduledPost
	if err := support.Decode(body, &posts); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Scheduled posts: %d", len(posts))},
		ResultsHeading: "Posts",
		Results:        postRows(posts),
		RetrievalHints: []string{
			fmt.Sprintf("%s post get <post-id>", support.CLIName),
			fmt.Sprintf("%s post list --start-date 2026-04-01 --end-date 2026-04-30", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("post get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: post get <post-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/posts/"+id, nil)
	if err != nil {
		return err
	}
	var post support.ScheduledPost
	if err := support.Decode(body, &post); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("ID: %s", post.ID),
		fmt.Sprintf("Title: %s", post.Title),
		fmt.Sprintf("Status: %s", post.Status),
		fmt.Sprintf("Scheduled: %s", support.FormatTimeValue(post.ScheduledAt)),
	}
	if len(post.Platforms) > 0 {
		results = append(results, fmt.Sprintf("Platforms: %s", strings.Join(post.Platforms, ", ")))
	}
	if post.PostedAt != nil {
		results = append(results, fmt.Sprintf("Posted: %s", support.FormatTimeValue(*post.PostedAt)))
	}
	if post.CampaignID != nil && *post.CampaignID != "" {
		results = append(results, fmt.Sprintf("Campaign: %s", *post.CampaignID))
	}
	if len(post.MediaURLs) > 0 {
		results = append(results, fmt.Sprintf("Media: %s", strings.Join(post.MediaURLs, ", ")))
	}
	if len(post.PlatformVariants) > 0 {
		results = append(results, "Platform variants:")
		for k, v := range post.PlatformVariants {
			results = append(results, fmt.Sprintf("  %s: %s", k, truncate(v, 120)))
		}
	}
	results = append(results,
		fmt.Sprintf("Content: %s", truncate(post.Content, 200)),
		fmt.Sprintf("Created: %s", support.FormatTimeValue(post.CreatedAt)),
	)

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Post: %s (%s)", post.Title, post.Status)},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s post optimize %s", support.CLIName, post.ID),
			fmt.Sprintf("%s post preview %s", support.CLIName, post.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("post update")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the update payload")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: post update <post-id> --body-file PATH")
	}
	id := fs.Arg(0)
	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("PUT", "/posts/"+id, nil, payload)
	if err != nil {
		return err
	}
	return renderMutation(body, fmt.Sprintf("Updated post %s", id), id, *jsonOutput)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("post delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: post delete <post-id>")
	}
	id := fs.Arg(0)

	body, err := core.Request("DELETE", "/posts/"+id, nil, nil)
	if err != nil {
		return err
	}
	return renderMutation(body, fmt.Sprintf("Deleted post %s", id), id, *jsonOutput)
}

func runAction(core *cliapp.ScenarioApp, args []string, verb, display string) error {
	fs := support.NewFlagSet("post " + verb)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: post %s <post-id>", verb)
	}
	id := fs.Arg(0)

	body, err := core.Request("POST", "/posts/"+id+"/"+verb, nil, nil)
	if err != nil {
		return err
	}
	return renderMutation(body, fmt.Sprintf("%s issued for post %s", display, id), id, *jsonOutput)
}

func runPreview(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("post preview")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: post preview <post-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/posts/"+id+"/preview", nil)
	if err != nil {
		return err
	}

	var generic map[string]interface{}
	_ = support.Decode(body, &generic)
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Preview for post %s", id)},
		ResultsHeading: "Preview",
		Results:        support.MapRows(generic),
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func renderMutation(body []byte, fallback, id string, asJSON bool) error {
	msg := support.EnvelopeMessage(body)
	if msg == "" {
		msg = fallback
	}
	report := cliapp.MutationReport{
		Result:      []string{msg},
		Changes:     []string{fmt.Sprintf("Post %s mutated", id)},
		NextCommand: []string{fmt.Sprintf("%s post get %s", support.CLIName, id)},
	}
	if asJSON {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func postRows(posts []support.ScheduledPost) []string {
	if len(posts) == 0 {
		return []string{"No posts in range"}
	}
	rows := make([]string, 0, len(posts))
	for _, p := range posts {
		rows = append(rows, fmt.Sprintf("%s | %s | status=%s | when=%s | platforms=%s",
			support.ShortID(p.ID), truncate(p.Title, 40), p.Status,
			support.FormatTimeValue(p.ScheduledAt), strings.Join(p.Platforms, ",")))
	}
	return rows
}

func truncate(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
