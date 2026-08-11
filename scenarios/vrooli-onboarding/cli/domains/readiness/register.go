package readiness

import (
	"encoding/json"
	"fmt"
	"os"

	"vrooli-onboarding/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// ExitError preserves the machine-readable readiness contract at the process
// boundary. A required item that is not ready is deliberately distinct from a
// transport or command error.
type ExitError struct {
	Code int
	Text string
}

func (e *ExitError) Error() string { return e.Text }
func (e *ExitError) ExitCode() int { return e.Code }

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{Title: "Readiness", Commands: []cliapp.Command{
		{Name: "readiness", Description: "Validate required onboarding items", NeedsAPI: true, Run: func(args []string) error {
			return run(core, args)
		}},
	}}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("readiness")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	body, err := core.Get("/v2/readiness", nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		if _, err := os.Stdout.Write(append(body, '\n')); err != nil {
			return err
		}
	} else {
		var value any
		if err := json.Unmarshal(body, &value); err != nil {
			return fmt.Errorf("decode readiness response: %w", err)
		}
		pretty, _ := json.MarshalIndent(value, "", "  ")
		if _, err := fmt.Fprintln(os.Stdout, string(pretty)); err != nil {
			return err
		}
	}
	var response struct {
		Status      string `json:"status"`
		Credentials []struct {
			Required bool   `json:"required"`
			Status   string `json:"status"`
		} `json:"credentials"`
		Hosts []struct {
			Required bool   `json:"required"`
			Status   string `json:"status"`
		} `json:"hosts"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("decode readiness status: %w", err)
	}
	for _, item := range response.Credentials {
		if item.Required && item.Status != "configured" {
			return &ExitError{Code: 2, Text: "required credential is not ready"}
		}
	}
	for _, item := range response.Hosts {
		if item.Required && item.Status != "ready" {
			return &ExitError{Code: 2, Text: "required host item is not ready"}
		}
	}
	if response.Status == "missing" || response.Status == "unsupported" {
		return &ExitError{Code: 2, Text: "required onboarding item is not ready"}
	}
	return nil
}
