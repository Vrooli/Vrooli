package readiness

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

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
			if len(args) > 0 && args[0] == "acknowledge-degraded" {
				return acknowledgeDegraded(core, args[1:])
			}
			return run(core, args)
		}},
	}}
}

// blocker mirrors the API's metadata-only completion blocker. It never carries
// a credential value.
type blocker struct {
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	Reason      string `json:"reason"`
	Remediation string `json:"remediation"`
}

// acknowledgeDegraded records the operator's acceptance of the exact set of
// degraded optional items readiness reports now. The digest identifies that
// set, so an acknowledgement cannot carry over to a different gap.
func acknowledgeDegraded(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("readiness acknowledge-degraded")
	digest := fs.String("digest", "", "Degraded-set digest from `readiness` output; omit to read the current one")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	value := strings.TrimSpace(*digest)
	if value == "" {
		body, err := core.Get("/v2/readiness", nil)
		if err != nil {
			return err
		}
		var current struct {
			DegradedDigest string `json:"degraded_digest"`
		}
		if err := json.Unmarshal(body, &current); err != nil {
			return fmt.Errorf("decode readiness response: %w", err)
		}
		value = strings.TrimSpace(current.DegradedDigest)
		if value == "" {
			return &ExitError{Code: 2, Text: "there is no degraded set to acknowledge"}
		}
	}
	payload, err := json.Marshal(map[string]string{"readiness_digest": value})
	if err != nil {
		return err
	}
	response, err := core.Request("POST", "/v2/readiness/degraded-acknowledgement", nil, payload)
	if err != nil {
		return err
	}
	if *jsonOutput {
		_, err := os.Stdout.Write(append(response, '\n'))
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, "Recorded the degraded acknowledgement for digest", value)
	return err
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
	// The verdict comes from the API's typed blockers rather than from a second
	// derivation here. Two implementations of the same rule are how the wizard
	// and the completion marker came to disagree in the first place.
	var response struct {
		Blockers             []blocker `json:"blockers"`
		Degraded             []blocker `json:"degraded"`
		DegradedDigest       string    `json:"degraded_digest"`
		DegradedAcknowledged bool      `json:"degraded_acknowledged"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("decode readiness status: %w", err)
	}
	for _, item := range response.Blockers {
		if _, err := fmt.Fprintf(os.Stdout, "Blocked: %s %s — %s. Next: %s\n", item.Kind, item.Name, item.Reason, item.Remediation); err != nil {
			return err
		}
	}
	if len(response.Blockers) > 0 {
		return &ExitError{Code: 2, Text: fmt.Sprintf("configuration is not complete: %d blocking item(s) remain; blockers: %s", len(response.Blockers), blockerDetails(response.Blockers))}
	}
	if len(response.Degraded) > 0 && !response.DegradedAcknowledged {
		for _, item := range response.Degraded {
			if _, err := fmt.Fprintf(os.Stdout, "Degraded: %s %s — %s. Next: %s\n", item.Kind, item.Name, item.Reason, item.Remediation); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(os.Stdout, "Accept them with: vrooli-onboarding readiness acknowledge-degraded --digest %s\n", response.DegradedDigest); err != nil {
			return err
		}
		return &ExitError{Code: 2, Text: fmt.Sprintf("configuration is not complete: %d optional item(s) need an explicit acknowledgement; degraded: %s", len(response.Degraded), blockerDetails(response.Degraded))}
	}
	return nil
}

// blockerDetails keeps the process error useful to non-interactive callers
// that do not retain the CLI's stdout. It contains only the metadata already
// exposed by the readiness API; credentials and other secret values never
// enter this string.
func blockerDetails(items []blocker) string {
	details := make([]string, 0, len(items))
	for _, item := range items {
		details = append(details, fmt.Sprintf("%s %s — %s; next: %s", item.Kind, item.Name, item.Reason, item.Remediation))
	}
	return strings.Join(details, " | ")
}
