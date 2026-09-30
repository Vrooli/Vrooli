package validations

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"deployment-manager/cli/cmdutil"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Commands provides CLI commands for visual validation management.
type Commands struct {
	api *cliutil.APIClient
}

// New creates a new validations command set.
func New(api *cliutil.APIClient) *Commands {
	return &Commands{api: api}
}

// Run dispatches validation subcommands.
func (c *Commands) Run(args []string) error {
	if len(args) == 0 {
		return errors.New("validation subcommand is required: run, status, video, review, list")
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "run":
		return c.runValidation(rest)
	case "status":
		return c.status(rest)
	case "video":
		return c.video(rest)
	case "review":
		return c.review(rest)
	case "list":
		return c.list(rest)
	default:
		return fmt.Errorf("unknown validation subcommand: %s", sub)
	}
}

func (c *Commands) runValidation(args []string) error {
	fs := flag.NewFlagSet("validations run", flag.ContinueOnError)
	record := fs.Bool("record", false, "Enable video recording")
	platform := fs.String("platform", "linux", "Target platform")
	commit := fs.String("commit", "", "Git commit hash (required)")
	format := fs.String("format", "", "Output format (json)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("profile ID is required")
	}
	if *commit == "" {
		return errors.New("--commit is required for new validations")
	}

	payload := map[string]interface{}{
		"profile_id":      remaining[0],
		"record_video":    *record,
		"platform":        *platform,
		"git_commit_hash": *commit,
	}

	body, err := c.api.Request("POST", "/api/v1/validations", nil, payload)
	if err != nil {
		return err
	}
	cmdutil.PrintByFormat(*format, body)
	return nil
}

func (c *Commands) status(args []string) error {
	fs := flag.NewFlagSet("validations status", flag.ContinueOnError)
	format := fs.String("format", "", "Output format (json)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("validation ID is required")
	}

	body, err := c.api.Request("GET", "/api/v1/validations/"+remaining[0], nil, nil)
	if err != nil {
		return err
	}
	cmdutil.PrintByFormat(*format, body)
	return nil
}

func (c *Commands) video(args []string) error {
	fs := flag.NewFlagSet("validations video", flag.ContinueOnError)
	output := fs.String("output", "./validation.mp4", "Output file path")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("validation ID is required")
	}

	// Use the API client to get the video URL, then download
	body, err := c.api.Request("GET", "/api/v1/validations/"+remaining[0], nil, nil)
	if err != nil {
		return err
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Validation: %s", remaining[0]),
		},
		ResultsHeading: "Retrieval",
		Results: []string{
			"Video download is an external fetch from the API video endpoint.",
		},
		RetrievalHints: []string{
			fmt.Sprintf("curl -o %s <api-base>/api/v1/validations/%s/video", *output, remaining[0]),
			fmt.Sprintf("deployment-manager validations status %s --format json", remaining[0]),
		},
	}
	if strings.TrimSpace(string(body)) != "" {
		report.RetrievalHints = append(report.RetrievalHints, fmt.Sprintf("Validation details: %s", strings.TrimSpace(string(body))))
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func (c *Commands) review(args []string) error {
	fs := flag.NewFlagSet("validations review", flag.ContinueOnError)
	decision := fs.String("decision", "", "approve or reject")
	notes := fs.String("notes", "", "Review notes")
	format := fs.String("format", "", "Output format (json)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("validation ID is required")
	}
	if *decision != "approve" && *decision != "reject" {
		return errors.New("--decision must be 'approve' or 'reject'")
	}

	// Map CLI decision names to API values.
	apiDecision := *decision + "d" // "approve" -> "approved", "reject" -> "rejected"
	if *decision == "reject" {
		apiDecision = "rejected"
	}

	payload := map[string]interface{}{
		"decision": apiDecision,
		"notes":    *notes,
	}

	body, err := c.api.Request("POST", "/api/v1/validations/"+remaining[0]+"/review", nil, payload)
	if err != nil {
		return err
	}

	if *format == "json" {
		cmdutil.PrintByFormat(*format, body)
		return nil
	}

	var resp map[string]interface{}
	if jsonErr := json.Unmarshal(body, &resp); jsonErr == nil {
		report := cliapp.MutationReport{
			Result: []string{
				fmt.Sprintf("Review submitted: %s", apiDecision),
			},
			Changes: []string{
				fmt.Sprintf("Validation: %s", remaining[0]),
				fmt.Sprintf("Notes: %s", fallbackValidation(*notes)),
			},
			NextCommand: []string{
				fmt.Sprintf("deployment-manager validations status %s", remaining[0]),
			},
		}
		if aid, ok := resp["approval_id"].(string); ok && aid != "" {
			report.Changes = append(report.Changes, fmt.Sprintf("Deployment approval: %s", aid))
			report.Changes = append(report.Changes, fmt.Sprintf("Approval status: %v", resp["approval_status"]))
			report.NextCommand = append(report.NextCommand, fmt.Sprintf("deployment-manager approvals get %s", aid))
		}
		return cliapp.RenderMutationReport(os.Stdout, report)
	} else {
		cmdutil.PrintByFormat(*format, body)
	}
	return nil
}

func fallbackValidation(value string) string {
	if strings.TrimSpace(value) == "" {
		return "(none)"
	}
	return value
}

func (c *Commands) list(args []string) error {
	fs := flag.NewFlagSet("validations list", flag.ContinueOnError)
	profile := fs.String("profile", "", "Filter by profile ID")
	commit := fs.String("commit", "", "Filter by git commit hash")
	format := fs.String("format", "", "Output format (json)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if *profile == "" {
		return errors.New("--profile is required")
	}

	path := "/api/v1/profiles/" + *profile + "/validations"
	if *commit != "" {
		path += "?commit=" + *commit
	}

	body, err := c.api.Request("GET", path, nil, nil)
	if err != nil {
		return err
	}
	cmdutil.PrintByFormat(*format, body)
	return nil
}
