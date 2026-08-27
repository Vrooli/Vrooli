package hygiene

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/vrooli/vrooli/internal/tuning"
)

// Test-freshness check: asks test-genie whether the scenarios touched by the
// current change-set have had their required phases (the quick preset) run
// against their current trees. The check is strictly advisory — stale
// scenarios surface as warning findings (non-blocking under the default
// --fail-on error), and every infrastructure failure (test-genie CLI not
// installed, API down, timeout, not a git repo) degrades to a passing
// info-severity "skipped" check rather than a finding.
const freshnessCheckBudget = tuning.ServiceHealthTimeout

var errNoTestGenieCLI = errors.New("test-genie CLI not installed")

// freshnessAdvisory mirrors the JSON emitted by
// `test-genie runs freshness --changed --json`.
type freshnessAdvisory struct {
	Checked   bool               `json:"checked"`
	Scenarios []string           `json:"scenarios,omitempty"`
	Warnings  []freshnessWarning `json:"warnings,omitempty"`
	Truncated bool               `json:"truncated,omitempty"`
}

type freshnessWarning struct {
	Scenario    string   `json:"scenario"`
	StalePhases []string `json:"stale_phases"`
	Command     string   `json:"command"`
}

// testFreshnessOutput is a package var so tests can fake the test-genie CLI.
// `runs freshness --changed --json` writes the advisory JSON to stdout even
// when it exits 1 (stale scenarios found), so any stdout wins over the error.
var testFreshnessOutput = func(ctx context.Context, root string) ([]byte, error) {
	path, err := exec.LookPath("test-genie")
	if err != nil {
		return nil, errNoTestGenieCLI
	}
	cmd := exec.CommandContext(ctx, path, "runs", "freshness", "--changed", "--json")
	cmd.Dir = root
	out, err := cmd.Output()
	if len(out) > 0 {
		return out, nil
	}
	return nil, err
}

func (s Service) checkTestFreshness(report *Report) {
	ctx, cancel := context.WithTimeout(context.Background(), freshnessCheckBudget)
	defer cancel()

	out, err := testFreshnessOutput(ctx, s.Root)
	if err != nil {
		reason := "test-genie CLI did not answer within the check budget"
		if errors.Is(err, errNoTestGenieCLI) {
			reason = "test-genie CLI not installed"
		}
		report.addCheck("test_freshness", true, SeverityInfo, "skipped: "+reason)
		return
	}
	var advisory freshnessAdvisory
	if err := json.Unmarshal(out, &advisory); err != nil {
		report.addCheck("test_freshness", true, SeverityInfo, "skipped: unexpected test-genie output")
		return
	}
	if !advisory.Checked {
		report.addCheck("test_freshness", true, SeverityInfo, "skipped: test-genie API unreachable or not a git repo")
		return
	}
	if len(advisory.Warnings) == 0 {
		message := "all changed scenarios fresh"
		if len(advisory.Scenarios) == 0 {
			message = "no scenario changes in the current change-set"
		}
		report.addCheck("test_freshness", true, SeverityInfo, message)
		return
	}

	message := fmt.Sprintf("%d changed scenario(s) have required test phases that have not run against their current trees", len(advisory.Warnings))
	if advisory.Truncated {
		message += " (change-set touches more scenarios; only the first few were checked)"
	}
	report.addCheck("test_freshness", false, SeverityWarning, message)
	for _, warning := range advisory.Warnings {
		report.addFinding(Finding{
			Severity: SeverityWarning,
			Code:     "test_freshness",
			Path:     "scenarios/" + warning.Scenario,
			Message: fmt.Sprintf("test-genie hasn't run [%s] since the latest changes in %s",
				strings.Join(warning.StalePhases, ", "), warning.Scenario),
			Why:        "Required test phases (the quick preset) should run against a scenario's current tree before its changes are committed.",
			Fixability: FixabilityGuided,
			NextActions: []Action{{
				Code:       "run_required_test_phases",
				Message:    fmt.Sprintf("Run the required test phases for %s.", warning.Scenario),
				Command:    warning.Command,
				Fixability: FixabilityGuided,
			}},
		})
	}
}
