package usage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
)

const runtimeFrictionAPIPath = "/api/v1/runtime/friction"

const maxRuntimeFrictionResponseBytes = 2 << 20

type runtimeFrictionResponse struct {
	Window struct {
		Since time.Time `json:"since"`
		Until time.Time `json:"until"`
	} `json:"window"`
	Coverage struct {
		State                   string `json:"state"`
		Reason                  string `json:"reason"`
		RunsObserved            int    `json:"runsObserved"`
		ImportedRuns            int    `json:"importedRuns"`
		ImportedRunsIncluded    bool   `json:"importedRunsIncluded"`
		UnknownRuns             int    `json:"unknownRuns"`
		UnclassifiedBlockedRuns int    `json:"unclassifiedBlockedRuns"`
	} `json:"coverage"`
	Totals struct {
		TotalDurationMS   int64 `json:"totalDurationMs"`
		KnownDurationMS   int64 `json:"knownDurationMs"`
		UnknownDurationMS int64 `json:"unknownDurationMs"`
	} `json:"totals"`
	Categories []struct {
		Category      string `json:"category"`
		DurationMS    int64  `json:"durationMs"`
		Runs          int    `json:"runs"`
		EvidenceCount int    `json:"evidenceCount"`
	} `json:"categories"`
	BlockedChecks []struct {
		CheckID     string `json:"checkId"`
		Check       string `json:"check"`
		Runs        int    `json:"runs"`
		Occurrences int    `json:"occurrences"`
		Outcomes    struct {
			Blocked int `json:"blocked"`
			Allowed int `json:"allowed"`
			Unknown int `json:"unknown"`
		} `json:"outcomes"`
		EnforcementDurationMS      int64 `json:"enforcementDurationMs"`
		EnforcementDurationKnown   int   `json:"enforcementDurationKnown"`
		EnforcementDurationUnknown int   `json:"enforcementDurationUnknown"`
		Responses                  struct {
			Stopped    int `json:"stopped"`
			Workaround int `json:"workaround"`
			Asked      int `json:"asked"`
			Succeeded  int `json:"succeeded"`
			Unknown    int `json:"unknown"`
		} `json:"responses"`
		RunIDs []string `json:"runIds"`
	} `json:"blockedChecks"`
}

func runtimeFrictionCommand(newHTTP paceHTTPFor) cliapp.Command {
	return cliapp.Command{
		Name:        "runtime-friction",
		NeedsAPI:    true,
		Description: "Classify observed run time and blocked check friction [--since RFC3339] [--until RFC3339] [--json]",
		Usage:       "agent-manager usage runtime-friction [--since RFC3339] [--until RFC3339] [--json]",
		HelpText:    "Reads measured duration categories, report coverage, structured guard outcomes, enforcement request durations, and later agent responses. Imported runs are included when available. `checkId` is blank when the source records only a command path or refusal without authoritative identity. Unknowns remain explicit.",
		Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
			{Name: "since", Description: "Start of the report window: RFC 3339, YYYY-MM-DD or an age such as 7d (default: 7 days before --until)"},
			{Name: "until", Description: "End of the report window: RFC 3339, YYYY-MM-DD or an age such as 1d (default: now)"},
		}},
		RunCtx: func(ctx cliapp.RunContext) error {
			query := url.Values{}
			for _, name := range []string{"since", "until"} {
				if value := strings.TrimSpace(ctx.Flag(name)); value != "" {
					query.Set(name, value)
				}
			}
			client, base := newHTTP(ctx.Core())
			endpoint := strings.TrimRight(base, "/") + runtimeFrictionAPIPath
			if encoded := query.Encode(); encoded != "" {
				endpoint += "?" + encoded
			}
			request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, endpoint, nil)
			if err != nil {
				return err
			}
			response, err := client.Do(request)
			if err != nil {
				return err
			}
			defer response.Body.Close()
			raw, err := io.ReadAll(io.LimitReader(response.Body, maxRuntimeFrictionResponseBytes+1))
			if err != nil {
				return err
			}
			if response.StatusCode != http.StatusOK {
				if len(raw) > maxRuntimeFrictionResponseBytes {
					raw = raw[:maxRuntimeFrictionResponseBytes]
				}
				return fmt.Errorf("runtime friction request failed (%d): %s", response.StatusCode, strings.TrimSpace(string(raw)))
			}
			if len(raw) > maxRuntimeFrictionResponseBytes {
				return fmt.Errorf("runtime friction response exceeds 2 MiB limit")
			}
			if ctx.JSON() {
				var pretty bytes.Buffer
				if err := json.Indent(&pretty, raw, "", "  "); err != nil {
					return err
				}
				_, err := fmt.Fprintln(ctx.Stdout(), pretty.String())
				return err
			}
			var report runtimeFrictionResponse
			if err := json.Unmarshal(raw, &report); err != nil {
				return fmt.Errorf("decode runtime friction response: %w", err)
			}
			return renderRuntimeFriction(ctx.Stdout(), report)
		},
	}
}

func renderRuntimeFriction(out io.Writer, report runtimeFrictionResponse) error {
	if _, err := fmt.Fprintf(out, "Runtime friction: %s to %s\nCoverage: %s; %d runs", report.Window.Since.UTC().Format(time.RFC3339), report.Window.Until.UTC().Format(time.RFC3339), report.Coverage.State, report.Coverage.RunsObserved); err != nil {
		return err
	}
	if report.Coverage.ImportedRunsIncluded {
		if _, err := fmt.Fprintf(out, "; %d imported included", report.Coverage.ImportedRuns); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(out, "; %s known, %s unknown\n", durationText(report.Totals.KnownDurationMS), durationText(report.Totals.UnknownDurationMS)); err != nil {
		return err
	}
	if report.Coverage.Reason != "" {
		if _, err := fmt.Fprintf(out, "Coverage note: %s\n", report.Coverage.Reason); err != nil {
			return err
		}
	}
	for _, category := range report.Categories {
		if _, err := fmt.Fprintf(out, "- %s: %s across %d runs (%d evidence rows)\n", category.Category, durationText(category.DurationMS), category.Runs, category.EvidenceCount); err != nil {
			return err
		}
	}
	for _, check := range report.BlockedChecks {
		identity := "no evaluator ID recorded"
		if check.CheckID != "" {
			identity = "checkId=" + check.CheckID
		}
		if _, err := fmt.Fprintf(out, "- blocked check %q (%s; %d runs, %d occurrences): stopped %d, workaround %d, asked %d, succeeded %d, unknown %d", check.Check, identity, check.Runs, check.Occurrences, check.Responses.Stopped, check.Responses.Workaround, check.Responses.Asked, check.Responses.Succeeded, check.Responses.Unknown); err != nil {
			return err
		}
		if check.Outcomes.Blocked+check.Outcomes.Allowed+check.Outcomes.Unknown > 0 {
			if _, err := fmt.Fprintf(out, "; enforcement blocked %d, allowed %d, unknown %d", check.Outcomes.Blocked, check.Outcomes.Allowed, check.Outcomes.Unknown); err != nil {
				return err
			}
		}
		if check.EnforcementDurationKnown+check.EnforcementDurationUnknown > 0 {
			if _, err := fmt.Fprintf(out, "; enforcement request %s measured across %d occurrences, %d unknown", enforcementDurationText(check.EnforcementDurationMS), check.EnforcementDurationKnown, check.EnforcementDurationUnknown); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(out); err != nil {
			return err
		}
	}
	return nil
}

func durationText(milliseconds int64) string {
	return (time.Duration(milliseconds) * time.Millisecond).Round(time.Second).String()
}

func enforcementDurationText(milliseconds int64) string {
	if milliseconds < 1000 {
		return fmt.Sprintf("%dms", milliseconds)
	}
	return durationText(milliseconds)
}
