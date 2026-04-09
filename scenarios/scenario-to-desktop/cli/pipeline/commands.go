// Package pipeline provides CLI commands for pipeline management.
package pipeline

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"strings"

	"scenario-to-desktop/cli/cmdutil"

	"github.com/vrooli/cli-core/cliutil"
)

const appName = "scenario-to-desktop"

// ErrAlreadyPrinted is a sentinel error indicating the error was already printed to the user.
// This prevents duplicate error output when the error is returned up the call stack.
type ErrAlreadyPrinted struct {
	Err error
}

func (e *ErrAlreadyPrinted) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return "error already printed"
}

func (e *ErrAlreadyPrinted) Unwrap() error {
	return e.Err
}

// IsAlreadyPrinted checks if an error is an ErrAlreadyPrinted.
func IsAlreadyPrinted(err error) bool {
	var ap *ErrAlreadyPrinted
	return errors.As(err, &ap)
}

// Commands provides pipeline CLI commands.
type Commands struct {
	api *cliutil.APIClient
}

// New creates a new pipeline Commands instance.
func New(api *cliutil.APIClient) *Commands {
	return &Commands{api: api}
}

func (c *Commands) apiPath(path string) string {
	return cmdutil.APIPath(path)
}

func (c *Commands) apiGet(path string, query map[string]string) ([]byte, error) {
	return c.api.Get(c.apiPath(path), cmdutil.MapToValues(query))
}

func (c *Commands) apiPost(path string, body interface{}) ([]byte, error) {
	return c.api.Request("POST", c.apiPath(path), nil, body)
}

// Resume resumes a stopped pipeline.
func (c *Commands) Resume(args []string) error {
	fs := flag.NewFlagSet("pipeline-resume", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: pipeline-resume <id>")
	}

	pipelineID := fs.Args()[0]
	body, err := c.apiPost("/pipeline/"+pipelineID+"/resume", nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Println("Pipeline resumed")
	cliutil.PrintJSON(body)
	return nil
}

// Cancel cancels a running pipeline.
func (c *Commands) Cancel(args []string) error {
	fs := flag.NewFlagSet("pipeline-cancel", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: pipeline-cancel <id>")
	}

	pipelineID := fs.Args()[0]
	body, err := c.apiPost("/pipeline/"+pipelineID+"/cancel", nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Println("Pipeline cancellation requested")
	return nil
}

// List lists all pipelines.
func (c *Commands) List(args []string) error {
	fs := flag.NewFlagSet("pipeline-list", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := c.apiGet("/pipelines", nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp struct {
		Pipelines []struct {
			PipelineID   string `json:"pipeline_id"`
			ScenarioName string `json:"scenario_name"`
			Status       string `json:"status"`
		} `json:"pipelines"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if len(resp.Pipelines) == 0 {
		fmt.Println("No pipelines found")
		return nil
	}

	fmt.Println("Pipelines:")
	for _, p := range resp.Pipelines {
		fmt.Printf("  %-36s %-20s %s\n", p.PipelineID, p.ScenarioName, p.Status)
	}
	return nil
}

// Active gets active pipeline for scenario.
func (c *Commands) Active(args []string) error {
	fs := flag.NewFlagSet("pipeline-active", flag.ContinueOnError)
	noCreate := fs.Bool("no-create", false, "Don't create if none exists")
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: pipeline-active <scenario> [--no-create]")
	}

	scenario := fs.Args()[0]
	query := make(map[string]string)
	if *noCreate {
		query["auto_create"] = "false"
	}

	body, err := c.apiGet("/scenarios/"+scenario+"/pipeline/active", query)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}

// Create creates a new pipeline for scenario.
func (c *Commands) Create(args []string) error {
	fs := flag.NewFlagSet("pipeline-create", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: pipeline-create <scenario>")
	}

	scenario := fs.Args()[0]
	body, err := c.apiPost("/scenarios/"+scenario+"/pipeline", nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Println("Pipeline created")
	cliutil.PrintJSON(body)
	return nil
}

// Reset resets active pipeline for scenario.
func (c *Commands) Reset(args []string) error {
	fs := flag.NewFlagSet("pipeline-reset", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: pipeline-reset <scenario>")
	}

	scenario := fs.Args()[0]
	body, err := c.apiPost("/scenarios/"+scenario+"/pipeline/reset", nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Println("Pipeline reset")
	cliutil.PrintJSON(body)
	return nil
}

// History gets pipeline history.
func (c *Commands) History(args []string) error {
	fs := flag.NewFlagSet("pipeline-history", flag.ContinueOnError)
	limit := fs.Int("limit", 10, "Number of pipelines to return")
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: pipeline-history <scenario> [--limit N]")
	}

	scenario := fs.Args()[0]
	query := map[string]string{"limit": fmt.Sprintf("%d", *limit)}

	body, err := c.apiGet("/scenarios/"+scenario+"/pipeline/history", query)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}

func normalizeBumpValue(input string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(input))
	switch value {
	case "patch", "minor", "medium", "major":
		return value, nil
	case "auto":
		return "patch", nil
	default:
		return "", fmt.Errorf("invalid --bump-version %q (expected patch, minor, medium, major, auto)", input)
	}
}
