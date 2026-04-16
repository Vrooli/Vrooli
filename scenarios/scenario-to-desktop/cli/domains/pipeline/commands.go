// Package pipeline provides CLI commands for pipeline management.
package pipeline

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"strings"

	"scenario-to-desktop/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
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
	deps support.Dependencies
}

// New creates a new pipeline Commands instance.
func New(deps support.Dependencies) *Commands {
	return &Commands{deps: deps}
}

func (c *Commands) apiGet(path string, query map[string]string) ([]byte, error) {
	return c.deps.Get(path, query)
}

func (c *Commands) apiPost(path string, body interface{}) ([]byte, error) {
	return c.deps.Request("POST", path, nil, body)
}

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	cmds := New(deps)
	return cliapp.SubcommandGroup{
		Name:        "pipeline",
		Description: "Build pipeline operations (run 'pipeline help' for details)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "run", Description: "Start a new pipeline: run <scenario> [--stages ...] [--platforms ...] [--wait]", Run: cmds.Run},
			{Name: "status", Description: "Get pipeline status: status <id> [--verbose]", Run: cmds.Status},
			{Name: "resume", Description: "Resume a stopped pipeline: resume <id>", Run: cmds.Resume},
			{Name: "cancel", Description: "Cancel a running pipeline: cancel <id>", Run: cmds.Cancel},
			{Name: "list", Description: "List all pipelines", Run: cmds.List},
			{Name: "active", Description: "Get active pipeline for scenario: active <scenario>", Run: cmds.Active},
			{Name: "create", Description: "Create new pipeline for scenario: create <scenario>", Run: cmds.Create},
			{Name: "reset", Description: "Reset active pipeline for scenario: reset <scenario>", Run: cmds.Reset},
			{Name: "history", Description: "Get pipeline history: history <scenario> [--limit N]", Run: cmds.History},
			{Name: "start", Description: "Start active pipeline: start <scenario> [--stages ...] [--platforms ...] [--wait]", Run: cmds.Start},
			{Name: "gate", Description: "Show approval gate status: gate <id>", Run: cmds.Gate},
		},
	}
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

// Gate shows approval gate status for a pipeline.
func (c *Commands) Gate(args []string) error {
	fs := flag.NewFlagSet("pipeline-gate", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: pipeline-gate <id>")
	}

	pipelineID := fs.Args()[0]
	body, err := c.apiGet("/pipeline/"+pipelineID, map[string]string{"verbose": "true"})
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp struct {
		CurrentState string `json:"current_state"`
		CurrentStage string `json:"current_stage"`
		Status       string `json:"status"`
		ProgressMsg  string `json:"progress_message"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if resp.CurrentState == "gate_blocked" {
		fmt.Printf("Gate: BLOCKED (stage=%s)\n", resp.CurrentStage)
		fmt.Printf("Status: %s\n", resp.ProgressMsg)
		fmt.Println("Approve the release in deployment-manager to proceed.")
	} else if resp.Status == "running" {
		fmt.Printf("Gate: not blocked (state=%s, stage=%s)\n", resp.CurrentState, resp.CurrentStage)
	} else {
		fmt.Printf("Pipeline %s: %s\n", resp.Status, resp.ProgressMsg)
	}
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
