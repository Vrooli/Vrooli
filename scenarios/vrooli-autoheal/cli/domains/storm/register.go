// Package storm reports the storm authority's state: which agent session
// scopes exist, which are frozen, and the decision rows the authority wrote.
package storm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"vrooli-autoheal/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	platformgo "github.com/vrooli/platform-go"
)

const probeTimeout = 5 * time.Second

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "storm",
		Description: "Inspect frozen agent session scopes and the storm authority's decisions",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "status", Description: "List agent session scopes, whether each is frozen, and recent contain-storm decisions", Run: func(args []string) error { return status(core, args) }},
		},
	}
}

// Scope is one agent session scope as the host reports it.
type Scope struct {
	Unit   string `json:"unit"`
	Path   string `json:"path"`
	Frozen bool   `json:"frozen"`
	Reason string `json:"reason,omitempty"`
}

// Decision is one storm decision row.
type Decision struct {
	At      time.Time       `json:"at"`
	Epoch   string          `json:"epoch"`
	State   string          `json:"state"`
	Reason  string          `json:"reason"`
	Details json.RawMessage `json:"details,omitempty"`
}

// Report is the status payload.
type Report struct {
	Scopes    []Scope    `json:"scopes"`
	Decisions []Decision `json:"decisions"`
	Warnings  []string   `json:"warnings,omitempty"`
}

func status(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("storm status")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()
	report := Report{Scopes: []Scope{}, Decisions: []Decision{}}
	scopes, err := listAgentScopes(ctx)
	if err != nil {
		report.Warnings = append(report.Warnings, "scopes undetermined: "+err.Error())
	}
	report.Scopes = scopes
	decisions, err := listDecisions(core)
	if err != nil {
		report.Warnings = append(report.Warnings, "decisions undetermined: "+err.Error())
	}
	report.Decisions = decisions
	if *jsonOutput {
		data, _ := json.MarshalIndent(report, "", "  ")
		fmt.Fprintln(os.Stdout, string(data))
		return nil
	}
	lines := []string{fmt.Sprintf("Agent session scopes: %d", len(report.Scopes))}
	for _, scope := range report.Scopes {
		state := "running"
		if scope.Frozen {
			state = "FROZEN (thaw with `vrooli agent thaw " + strings.TrimSuffix(scope.Unit, ".scope") + "`)"
		}
		if scope.Reason != "" {
			state = "undetermined: " + scope.Reason
		}
		lines = append(lines, fmt.Sprintf("  %s  %s", scope.Unit, state))
	}
	decisionLines := make([]string, 0, len(report.Decisions))
	for _, decision := range report.Decisions {
		decisionLines = append(decisionLines, fmt.Sprintf("%s  %s  epoch=%s  %s", decision.At.Local().Format(time.RFC3339), decision.State, decision.Epoch, decision.Reason))
	}
	if len(decisionLines) == 0 {
		decisionLines = append(decisionLines, "none recorded")
	}
	return cliapp.RenderOperationalReport(os.Stdout, cliapp.OperationalReport{
		Status: lines,
		Triage: []cliapp.TriageGroup{{Heading: "Storm decisions (newest first)", Items: decisionLines}, {Heading: "Warnings", Items: report.Warnings}},
	})
}

// listAgentScopes asks the user manager for every vrooli-agent-* scope and
// reads each scope's freeze state from its cgroup. Membership is the unit
// list, never a command-line substring.
func listAgentScopes(ctx context.Context) ([]Scope, error) {
	out, err := exec.CommandContext(ctx, "systemctl", "--user", "list-units", "vrooli-agent-*", "--plain", "--no-legend", "--all").Output()
	if err != nil {
		return nil, fmt.Errorf("systemctl --user list-units: %w", err)
	}
	scopes := []Scope{}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || !strings.HasSuffix(fields[0], ".scope") {
			continue
		}
		scope := Scope{Unit: fields[0]}
		control, err := exec.CommandContext(ctx, "systemctl", "--user", "show", "-p", "ControlGroup", "--value", fields[0]).Output()
		if err != nil || strings.TrimSpace(string(control)) == "" {
			scope.Reason = "control group unreadable"
			scopes = append(scopes, scope)
			continue
		}
		scope.Path = strings.TrimSpace(string(control))
		frozen, err := platformgo.ScopeFrozen(platformgo.ScopeRef{Name: strings.TrimSuffix(fields[0], ".scope"), Kind: platformgo.ScopeKindCgroup, Path: scope.Path})
		if err != nil {
			scope.Reason = err.Error()
		}
		scope.Frozen = frozen
		scopes = append(scopes, scope)
	}
	return scopes, nil
}

// listDecisions reads the authority's rows through the autoheal API, which
// owns the registry read; the CLI never opens the registry itself.
func listDecisions(core *cliapp.ScenarioApp) ([]Decision, error) {
	body, err := core.Get("/storm/status", nil)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Decisions []struct {
			EpochID     string    `json:"EpochID"`
			State       string    `json:"State"`
			Reason      string    `json:"Reason"`
			CreatedAt   time.Time `json:"CreatedAt"`
			DetailsJSON string    `json:"DetailsJSON"`
		} `json:"decisions"`
		DecisionsError string `json:"decisions_error"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if payload.DecisionsError != "" {
		return nil, fmt.Errorf("%s", payload.DecisionsError)
	}
	decisions := make([]Decision, 0, len(payload.Decisions))
	for _, row := range payload.Decisions {
		decisions = append(decisions, Decision{At: row.CreatedAt, Epoch: row.EpochID, State: row.State, Reason: row.Reason, Details: json.RawMessage(row.DetailsJSON)})
	}
	return decisions, nil
}
