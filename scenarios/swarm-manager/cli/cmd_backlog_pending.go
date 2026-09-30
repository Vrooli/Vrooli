package main

import (
	"flag"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdBacklogPendingQuestions(args []string) error {
	fs := flag.NewFlagSet("backlog pending-questions", flag.ContinueOnError)
	sourceFlag := fs.String("source", "review", "Question source: review")
	limitFlag := fs.Int("limit", 0, "Maximum number of backlog items to return (0 = unlimited)")
	milestoneFlag := fs.String("milestone", "", "Restrict to backlog items in the given milestone")
	briefFlag := fs.Bool("brief", false, "Return a small agent-oriented pending-question brief")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	source := strings.ToLower(strings.TrimSpace(*sourceFlag))
	switch source {
	case "review":
	default:
		return fmt.Errorf("invalid source %q: must be review", *sourceFlag)
	}
	if *limitFlag < 0 {
		return fmt.Errorf("invalid limit %d: must be a non-negative integer", *limitFlag)
	}

	query := url.Values{}
	query.Set("source", source)
	if *limitFlag > 0 {
		query.Set("limit", strconv.Itoa(*limitFlag))
	}
	if milestone := strings.TrimSpace(*milestoneFlag); milestone != "" {
		query.Set("milestone", milestone)
	}
	if *briefFlag && *limitFlag == 0 {
		query.Set("limit", "8")
	}

	body, err := a.core.Get("/backlog/pending-questions", query)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[PendingQuestionsResponse](body)
	if err != nil {
		return err
	}

	totalQuestions := 0
	for _, item := range response.Items {
		totalQuestions += len(item.Questions)
	}

	if len(response.Items) == 0 {
		printSection("Summary")
		fmt.Println("  No pending questions found.")
		printCommandListSection("Next Steps", []string{
			cliCommand("backlog", "list"),
		})
		return nil
	}

	printSection("Summary")
	fmt.Printf("  Found %d backlog item(s) with %d pending question(s)\n", len(response.Items), totalQuestions)
	fmt.Printf("  Source: %s\n", source)
	if *limitFlag > 0 {
		fmt.Printf("  Limit: %d\n", *limitFlag)
	}
	if milestone := strings.TrimSpace(*milestoneFlag); milestone != "" {
		fmt.Printf("  Milestone: %s\n", milestone)
	}

	printSection("Results")
	for _, item := range response.Items {
		fmt.Printf("  [%s] %s (%d question(s))\n", item.Kind, item.Name, len(item.Questions))
		for _, question := range item.Questions {
			summary := summarizePendingQuestion(question)
			fmt.Printf("    - %s | %s\n", question.ID, summary)
		}
		fmt.Println()
	}

	first := response.Items[0]
	printCommandListSection("Retrieval Hints", []string{
		cliCommand("backlog", "get", "--kind", first.Kind, "--name", first.Name),
		cliCommand("backlog", "files", "--kind", first.Kind, "--name", first.Name),
	})
	return nil
}

func summarizePendingQuestion(question PendingQuestion) string {
	label := firstNonEmpty(question.Title, question.Description, "review item")
	if question.ReviewType != "" {
		return fmt.Sprintf("%s (%s)", label, question.ReviewType)
	}
	return label
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
