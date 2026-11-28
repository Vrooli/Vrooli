package usage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
)

const usagePaceAPIPath = "/api/v1/usage/pace"

type paceHTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type paceHTTPFor func(*cliapp.ScenarioApp) (paceHTTPDoer, string)

func paceHTTPClient(core *cliapp.ScenarioApp) (paceHTTPDoer, string) {
	return cliapp.NewConnectHTTPClient(core)
}

type paceResponse struct {
	GeneratedAt            time.Time  `json:"generated_at"`
	SelectedTier           string     `json:"selected_tier"`
	SelectedTargetRunOutAt *time.Time `json:"selected_target_run_out_at"`
	BaselineMaxConcurrent  int        `json:"baseline_max_concurrent_runs"`
	EffectiveMaxConcurrent int        `json:"effective_max_concurrent_runs"`
	Tiers                  []paceTier `json:"tiers"`
}

type paceTier struct {
	ID                     string     `json:"id"`
	Label                  string     `json:"label"`
	TargetRunOutAt         *time.Time `json:"target_run_out_at"`
	AvailableSlots         *int       `json:"available_slots"`
	EffectiveMaxConcurrent *int       `json:"effective_max_concurrent_runs"`
	Status                 string     `json:"status"`
	Basis                  string     `json:"basis"`
	Reason                 string     `json:"reason"`
	SourceScopes           []string   `json:"source_scopes"`
	UnavailableScopes      []struct {
		Provider string `json:"provider"`
		Pool     string `json:"pool"`
		Reason   string `json:"reason"`
	} `json:"unavailable_scopes"`
}

func paceCommand(newHTTP paceHTTPFor) cliapp.Command {
	return cliapp.Command{
		Name:        "pace",
		NeedsAPI:    true,
		Description: "Read the quota-based work pace forecast, or change the admission ceiling [set --tier TIER] [--json]",
		Usage:       "agent-manager usage pace [status|set] [--tier baseline|at_reset|use_by_thursday|use_by_wednesday|extra_allowance] [--json]",
		HelpText:    "Shows the saved concurrency baseline and provider-evidence-based slot estimates for each run-out target. Estimates assume quota burn scales linearly with admitted concurrency; unavailable readings remain unknown. `set --tier` applies an available tier through the normal settings path; `baseline` restores the saved ceiling.",
		Args: cliapp.ArgSchema{
			Positionals: []cliapp.Positional{{Name: "action", Description: "status (default) or set"}},
			Flags:       []cliapp.Flag{{Name: "tier", Description: "baseline, at_reset, use_by_thursday, use_by_wednesday or extra_allowance"}},
		},
		RunCtx: func(ctx cliapp.RunContext) error {
			action := strings.TrimSpace(ctx.Positional("action"))
			if action == "" {
				action = "status"
			}
			method := http.MethodGet
			var body io.Reader
			switch action {
			case "status":
				if strings.TrimSpace(ctx.Flag("tier")) != "" {
					return fmt.Errorf("--tier is only valid with `pace set`")
				}
			case "set":
				tier := strings.TrimSpace(ctx.Flag("tier"))
				if tier == "" {
					return fmt.Errorf("usage: agent-manager usage pace set --tier baseline|at_reset|use_by_thursday|use_by_wednesday|extra_allowance [--json]")
				}
				encoded, err := json.Marshal(map[string]string{"selected_tier": tier})
				if err != nil {
					return err
				}
				method = http.MethodPut
				body = bytes.NewReader(encoded)
			default:
				return fmt.Errorf("unknown pace action %q; expected status or set", action)
			}
			client, base := newHTTP(ctx.Core())
			request, err := http.NewRequestWithContext(context.Background(), method, strings.TrimRight(base, "/")+usagePaceAPIPath, body)
			if err != nil {
				return err
			}
			if method == http.MethodPut {
				request.Header.Set("Content-Type", "application/json")
			}
			response, err := client.Do(request)
			if err != nil {
				return err
			}
			defer response.Body.Close()
			raw, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
			if err != nil {
				return err
			}
			if response.StatusCode != http.StatusOK {
				return fmt.Errorf("usage pace request failed (%d): %s", response.StatusCode, strings.TrimSpace(string(raw)))
			}
			if ctx.JSON() {
				var pretty bytes.Buffer
				if err := json.Indent(&pretty, raw, "", "  "); err != nil {
					return err
				}
				_, err = fmt.Fprintln(ctx.Stdout(), pretty.String())
				return err
			}
			var result paceResponse
			if err := json.Unmarshal(raw, &result); err != nil {
				return fmt.Errorf("decode usage pace response: %w", err)
			}
			return renderPace(ctx.Stdout(), result)
		},
	}
}

func renderPace(out io.Writer, response paceResponse) error {
	selected := response.SelectedTier
	if selected == "" {
		selected = "baseline"
	}
	if _, err := fmt.Fprintf(out, "Selected pace: %s\nAdmission ceiling: %d concurrent runs (baseline %d)\n", selected, response.EffectiveMaxConcurrent, response.BaselineMaxConcurrent); err != nil {
		return err
	}
	for _, tier := range response.Tiers {
		line := "- " + tier.Label + ": "
		if tier.Status != "available" || tier.AvailableSlots == nil {
			line += "unknown (" + paceTextOr(tier.Reason, "forecast unavailable") + ")"
		} else {
			line += fmt.Sprintf("%d slots", *tier.AvailableSlots)
			if tier.TargetRunOutAt != nil {
				line += "; target " + tier.TargetRunOutAt.UTC().Format(time.RFC3339)
			}
		}
		if _, err := fmt.Fprintln(out, line); err != nil {
			return err
		}
		if tier.Status == "available" {
			// A measured ceiling does not constrain windows it could not measure.
			for _, scope := range tier.UnavailableScopes {
				if _, err := fmt.Fprintf(out, "    not measured: %s %s (%s)\n", scope.Provider, scope.Pool, paceTextOr(scope.Reason, "no reason reported")); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func paceTextOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
