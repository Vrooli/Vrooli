package validations

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"deployment-manager/cli/cmdutil"

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
	format := fs.String("format", "", "Output format (json)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("profile ID is required")
	}

	payload := map[string]interface{}{
		"profile_id":   remaining[0],
		"record_video": *record,
		"platform":     *platform,
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

	fmt.Fprintf(os.Stderr, "Validation details:\n%s\n", string(body))
	fmt.Fprintf(os.Stderr, "To download video, use: curl -o %s <api-base>/api/v1/validations/%s/video\n", *output, remaining[0])
	return nil
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

	payload := map[string]interface{}{
		"decision": *decision,
		"notes":    *notes,
	}

	body, err := c.api.Request("POST", "/api/v1/validations/"+remaining[0]+"/review", nil, payload)
	if err != nil {
		return err
	}
	cmdutil.PrintByFormat(*format, body)
	return nil
}

func (c *Commands) list(args []string) error {
	fs := flag.NewFlagSet("validations list", flag.ContinueOnError)
	profile := fs.String("profile", "", "Filter by profile ID")
	format := fs.String("format", "", "Output format (json)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if *profile == "" {
		return errors.New("--profile is required")
	}

	body, err := c.api.Request("GET", "/api/v1/profiles/"+*profile+"/validations", nil, nil)
	if err != nil {
		return err
	}
	cmdutil.PrintByFormat(*format, body)
	return nil
}
