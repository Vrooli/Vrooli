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

	"github.com/vrooli/cli-core/cliapp"

	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
)

const annotationsCoveragePath = "/api/v1/annotations/coverage"

// annotationsCommand is the fourth friction lens: gated inferred labels over a
// bounded run window. It is separate from `usage friction` because the
// annotation vocabulary is data and its gate state must be read before any rate
// is quoted.
func annotationsCommand(newHTTP paceHTTPFor) cliapp.Command {
	return cliapp.Command{
		Name:        "annotations",
		NeedsAPI:    true,
		Description: "What inferred labels does the run window carry, and which are proven? [--window 7d] [--json]",
		Usage:       "agent-manager usage annotations [--window 7d] [--json]",
		HelpText:    "Reads the annotation pack's per-label coverage for the window: observed and eligible runs, the gate state and measured precision/recall, and a rate only for a proven label. An unproven label is evidence, never a rate. An empty window is unknown coverage, not zero friction. Run `agent-manager annotations run <run-id>` for one run's labels and evidence.",
		Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
			{Name: "window", Description: "Window such as 7d, 24h or 30d (default 7d)"},
		}},
		RunCtx: func(ctx cliapp.RunContext) error {
			window := strings.TrimSpace(ctx.Flag("window"))
			if window == "" {
				window = "7d"
			}
			client, base := newHTTP(ctx.Core())
			endpoint := strings.TrimRight(base, "/") + annotationsCoveragePath + "?window=" + url.QueryEscape(window)
			request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, endpoint, nil)
			if err != nil {
				return err
			}
			response, err := client.Do(request)
			if err != nil {
				return fmt.Errorf("annotation coverage: %w", err)
			}
			defer response.Body.Close()
			raw, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
			if err != nil {
				return err
			}
			if response.StatusCode != http.StatusOK {
				return fmt.Errorf("annotation coverage failed (%d): %s", response.StatusCode, strings.TrimSpace(string(raw)))
			}
			if ctx.JSON() {
				var pretty bytes.Buffer
				if err := json.Indent(&pretty, raw, "", "  "); err != nil {
					return err
				}
				_, err := fmt.Fprintln(ctx.Stdout(), pretty.String())
				return err
			}
			var report annotationCoverage
			if err := json.Unmarshal(raw, &report); err != nil {
				return fmt.Errorf("decode annotation coverage: %w", err)
			}
			renderAnnotationCoverage(ctx.Stdout(), report)
			return nil
		},
	}
}

type annotationCoverage struct {
	Window struct {
		Since string `json:"since"`
		Until string `json:"until"`
	} `json:"window"`
	Pack struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"pack"`
	EligibleRuns int     `json:"eligible_runs"`
	ObservedRuns int     `json:"observed_runs"`
	Coverage     float64 `json:"coverage"`
	Weighting    struct {
		Mode       string  `json:"mode"`
		SampleRate float64 `json:"sample_rate"`
	} `json:"weighting"`
	Labels []annotationLabel `json:"labels"`
}

type annotationLabel struct {
	ID             string   `json:"id"`
	Group          string   `json:"group"`
	State          string   `json:"state"`
	Observed       int      `json:"observed"`
	Eligible       int      `json:"eligible"`
	Rate           *float64 `json:"rate"`
	Baseline       *float64 `json:"baseline"`
	Ratio          *float64 `json:"ratio"`
	BaselineState  string   `json:"baseline_state"`
	SearchFacet    string   `json:"search_facet"`
	MinProbability float64  `json:"min_probability"`
	Gate           struct {
		State           string  `json:"state"`
		Precision       float64 `json:"precision"`
		Recall          float64 `json:"recall"`
		Gold            int     `json:"gold"`
		MinGold         int     `json:"min_gold"`
		TargetPrecision float64 `json:"target_precision"`
		TargetRecall    float64 `json:"target_recall"`
		Reason          string  `json:"reason"`
	} `json:"gate"`
	Friction struct {
		Enabled bool `json:"enabled"`
	} `json:"friction"`
}

func renderAnnotationCoverage(out io.Writer, report annotationCoverage) {
	pack := strings.TrimSpace(report.Pack.Name)
	if pack == "" {
		pack = "unknown"
	}
	window := "unknown window"
	if report.Window.Since != "" || report.Window.Until != "" {
		window = report.Window.Since + " to " + report.Window.Until
	}
	fmt.Fprintf(out, "Inferred annotations: pack %s %s\n", pack, report.Pack.Version)
	fmt.Fprintf(out, "Window: %s\n", window)
	fmt.Fprintf(out, "Coverage: %.0f%% (%d of %d runs observed", 100*report.Coverage, report.ObservedRuns, report.EligibleRuns)
	if report.Weighting.Mode != "" {
		fmt.Fprintf(out, "; weighting %s", report.Weighting.Mode)
		if report.Weighting.SampleRate > 0 {
			fmt.Fprintf(out, " sample %.2f", report.Weighting.SampleRate)
		}
	}
	fmt.Fprintln(out, ")")
	if len(report.Labels) == 0 {
		fmt.Fprintln(out, "No annotations in this window. Coverage is zero; this is unknown, not zero friction.")
		return
	}
	fmt.Fprintln(out, "Labels:")
	shown, silent := 0, 0
	for _, label := range report.Labels {
		if label.Observed == 0 && label.State != "proven" {
			silent++
			continue
		}
		gate := label.Gate.State
		if gate == "" {
			gate = label.State
		}
		line := fmt.Sprintf("  [%s] %s", gate, label.ID)
		if label.Observed > 0 {
			line += fmt.Sprintf(" · observed %d/%d", label.Observed, label.Eligible)
		}
		if label.Rate != nil {
			line += fmt.Sprintf(" · rate %.1f%%", 100*(*label.Rate))
			if label.Baseline != nil {
				line += fmt.Sprintf(" · baseline %.1f%%", 100*(*label.Baseline))
			}
			if label.Ratio != nil {
				line += fmt.Sprintf(" · %.2fx", *label.Ratio)
			}
		} else {
			switch label.Gate.State {
			case "unproven":
				line += fmt.Sprintf(" · unproven (precision %.2f / recall %.2f, gold %d/%d)",
					label.Gate.Precision, label.Gate.Recall, label.Gate.Gold, label.Gate.MinGold)
			case "insufficient_gold":
				line += fmt.Sprintf(" · gold %d/%d, needs more labelled rows", label.Gate.Gold, label.Gate.MinGold)
			default:
				line += fmt.Sprintf(" · unmeasured (target %.2f / %.2f, no hold-out)", label.Gate.TargetPrecision, label.Gate.TargetRecall)
			}
		}
		if label.SearchFacet != "" {
			line += " · facet " + label.SearchFacet
		}
		fmt.Fprintln(out, line)
		shown++
	}
	if shown == 0 {
		fmt.Fprintln(out, "  no label has an observation in this window; unknown, not zero friction.")
	}
	if silent > 0 {
		fmt.Fprintf(out, "  %d label(s) have no observation in this window (unknown, not zero).\n", silent)
	}
	fmt.Fprintln(out, "Hints: agent-manager annotations run <run-id>  # one run's labels and evidence")
	fmt.Fprintln(out, "       agent-manager annotations evaluate --since <RFC3339> --limit 10  # run a bounded pass")
}

// inferredAnnotationLines renders the gated labels the friction read already
// carries. Reading them from the response keeps one authoritative source and
// avoids a second network call that could disagree with the finding list.
func inferredAnnotationLines(section *measurepb.UsageInferredAnnotations) []string {
	header := fmt.Sprintf("Inferred annotations: coverage %.0f%% (%d of %d runs observed)",
		100*section.GetCoverage(), section.GetObservedRuns(), section.GetEligibleRuns())
	if pack := strings.TrimSpace(strings.TrimSpace(section.GetPackName()) + " " + strings.TrimSpace(section.GetPackVersion())); pack != "" {
		header += " · pack " + pack
	}
	lines := []string{header}
	if note := strings.TrimSpace(section.GetNote()); note != "" {
		lines = append(lines, "   "+note)
	}
	if len(section.GetLabels()) == 0 {
		return append(lines, "   no observations in this window; unknown, not zero friction")
	}
	shown, silent := 0, 0
	for _, label := range section.GetLabels() {
		if !label.GetFriction() {
			// The friction lens reports friction labels only; run.type and
			// run.outcome are useful facets but not friction signals.
			continue
		}
		if label.GetObserved() == 0 && label.GetGateState() != "proven" {
			silent++
			continue
		}
		line := fmt.Sprintf("   [%s] %s", label.GetGateState(), label.GetId())
		if rate := label.GetRate(); label.Rate != nil {
			line += fmt.Sprintf(" · %.1f%% of observed runs", 100*rate)
		} else {
			line += fmt.Sprintf(" · no rate (target %.2f / %.2f, gold %d)",
				label.GetTargetPrecision(), label.GetTargetRecall(), label.GetGold())
		}
		if label.GetObserved() > 0 {
			line += fmt.Sprintf(" · observed %d/%d", label.GetObserved(), label.GetEligible())
		}
		lines = append(lines, line)
		shown++
	}
	if shown == 0 && silent == 0 {
		lines = append(lines, "   no friction label has an observation in this window; unknown, not zero friction")
	}
	if silent > 0 {
		lines = append(lines, fmt.Sprintf("   %d label(s) have no observation in this window (unknown, not zero).", silent))
	}
	return append(lines, "   Hints: agent-manager annotations gold --label <id> --out gold.jsonl  ·  agent-manager annotations benchmark --gold gold.jsonl --label <id>")
}
