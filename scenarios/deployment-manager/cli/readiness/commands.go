package readiness

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"deployment-manager/cli/cmdutil"
	"github.com/vrooli/cli-core/cliutil"
)

type Commands struct{ api *cliutil.APIClient }

func New(api *cliutil.APIClient) *Commands { return &Commands{api: api} }

func (c *Commands) Run(args []string) error {
	if len(args) == 0 || (args[0] != "verdict" && args[0] != "goal-open" && args[0] != "waiver") {
		return errors.New("subcommand required: verdict, goal-open, or waiver")
	}
	if args[0] == "waiver" {
		return c.runWaiver(args[1:])
	}
	if args[0] == "goal-open" {
		return c.runGoalOpen(args[1:])
	}
	fs := flag.NewFlagSet("readiness verdict", flag.ContinueOnError)
	scenario := fs.String("scenario", "", "Scenario name")
	commit := fs.String("commit", "", "Commit being evaluated")
	signalsFile := fs.String("signals-file", "", "JSON file containing readiness signals")
	format := fs.String("format", "", "Output format (json)")
	if err := cliutil.ParseInterspersed(fs, args[1:]); err != nil {
		return err
	}
	if strings.TrimSpace(*scenario) == "" || strings.TrimSpace(*commit) == "" || *signalsFile == "" {
		return errors.New("--scenario, --commit, and --signals-file are required")
	}
	data, err := os.ReadFile(*signalsFile)
	if err != nil {
		return fmt.Errorf("read signals file: %w", err)
	}
	var signals json.RawMessage
	if err := json.Unmarshal(data, &signals); err != nil {
		return fmt.Errorf("decode signals file: %w", err)
	}
	payload := map[string]any{"scenario": *scenario, "commit": *commit, "signals": json.RawMessage(signals)}
	body, err := c.api.Request("POST", "/api/v1/readiness/verdict", nil, payload)
	if err != nil {
		return err
	}
	if strings.EqualFold(cmdutil.ResolveFormat(*format), "json") {
		cliutil.PrintJSON(body)
		return nil
	}
	var verdict struct {
		Approved bool                                         `json:"approved"`
		Findings []struct{ ItemID, Severity, Message string } `json:"findings"`
	}
	if err := json.Unmarshal(body, &verdict); err != nil {
		return err
	}
	fmt.Printf("readiness approved=%t findings=%d\n", verdict.Approved, len(verdict.Findings))
	return nil
}

func (c *Commands) runWaiver(args []string) error {
	fs := flag.NewFlagSet("readiness waiver", flag.ContinueOnError)
	profile := fs.String("profile-id", "", "Profile identifier")
	commit := fs.String("commit", "", "Commit being waived")
	reason := fs.String("reason", "", "Reason for the waiver")
	actor := fs.String("actor", "", "Operator recording the waiver")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*profile) == "" || strings.TrimSpace(*commit) == "" || strings.TrimSpace(*reason) == "" || strings.TrimSpace(*actor) == "" {
		return errors.New("--profile-id, --commit, --reason, and --actor are required")
	}
	body, err := c.api.Request("POST", "/api/v1/readiness/waiver", nil, map[string]string{"profile_id": *profile, "commit": *commit, "reason": *reason, "actor": *actor})
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func (c *Commands) runGoalOpen(args []string) error {
	fs := flag.NewFlagSet("readiness goal-open", flag.ContinueOnError)
	scenario := fs.String("scenario", "", "Scenario name")
	commit := fs.String("commit", "", "Commit being reviewed")
	signalsFile := fs.String("signals-file", "", "JSON file containing readiness signals")
	deliverable := fs.String("deliverable", "", "Deliverable served by the goal")
	trigger := fs.String("trigger", "", "Trigger that required the review")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*scenario) == "" || strings.TrimSpace(*commit) == "" || *signalsFile == "" {
		return errors.New("--scenario, --commit, and --signals-file are required")
	}
	data, err := os.ReadFile(*signalsFile)
	if err != nil {
		return fmt.Errorf("read signals file: %w", err)
	}
	var signals json.RawMessage
	if err := json.Unmarshal(data, &signals); err != nil {
		return fmt.Errorf("decode signals file: %w", err)
	}
	body, err := c.api.Request("POST", "/api/v1/readiness/goal", nil, map[string]any{"scenario": *scenario, "commit": *commit, "deliverable": *deliverable, "trigger": *trigger, "signals": signals})
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}
