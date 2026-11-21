// Package annotations exposes the data-driven run annotation surface as CLI
// commands. The API owns evaluation and storage; these commands only present
// coverage, gate state and observations. Empty data is printed as unknown, not
// as "no friction".
package annotations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
)

// Register declares the `annotations` group.
func Register() cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "annotations",
		Description: "Inspect and run data-driven run annotations (label packs, coverage, gate state)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			packsCommand(),
			labelsCommand(),
			coverageCommand(),
			showCommand(),
			runCommand(),
			evaluateCommand(),
			schedulerCommand(),
			benchmarkCommand(),
			goldCommand(),
		},
	}
}

type requestFunc func(method, path string, query url.Values, body any) ([]byte, error)

type packSummary struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	PackHash    string            `json:"pack_hash"`
	Labels      int               `json:"labels"`
	Mode        string            `json:"mode"`
	LabelHashes map[string]string `json:"label_hashes"`
}

type packsResponse struct {
	Packs []packSummary `json:"packs"`
}

type stateCounts struct {
	Observed    int `json:"observed"`
	Compatible  int `json:"compatible"`
	NotObserved int `json:"not_observed"`
	Unavailable int `json:"unavailable"`
	// Stale rows belong to an earlier definition revision and do not count as
	// observations of the loaded pack.
	Stale int `json:"stale,omitempty"`
}

func (c stateCounts) total() int { return c.Observed + c.Compatible + c.NotObserved + c.Unavailable }

type gate struct {
	Precision float64 `json:"precision"`
	Recall    float64 `json:"recall"`
	MinGold   int     `json:"min_gold"`
}

type labelView struct {
	ID        string      `json:"id"`
	Group     string      `json:"group"`
	Method    string      `json:"method"`
	Optional  bool        `json:"optional"`
	Gate      gate        `json:"gate"`
	GateState string      `json:"gate_state"`
	Proven    bool        `json:"proven"`
	Coverage  stateCounts `json:"coverage"`
	HasData   bool        `json:"has_data"`
}

type labelsResponse struct {
	Pack    string      `json:"pack"`
	Version string      `json:"version"`
	Window  string      `json:"window"`
	Labels  []labelView `json:"labels"`
}

type gateStatus struct {
	State     string  `json:"state"`
	Precision float64 `json:"precision"`
	Recall    float64 `json:"recall"`
	Gold      int     `json:"gold"`
	MinGold   int     `json:"min_gold"`
	Reason    string  `json:"reason"`
}

type coverageLabel struct {
	ID             string     `json:"id"`
	Group          string     `json:"group"`
	State          string     `json:"state"`
	Gate           gateStatus `json:"gate"`
	Observed       int        `json:"observed"`
	Eligible       int        `json:"eligible"`
	Evaluated      int        `json:"evaluated"`
	Rate           *float64   `json:"rate,omitempty"`
	Baseline       *float64   `json:"baseline,omitempty"`
	Ratio          *float64   `json:"ratio,omitempty"`
	BaselineState  string     `json:"baseline_state,omitempty"`
	SearchFacet    string     `json:"search_facet"`
	MinProbability *float64   `json:"min_probability,omitempty"`
	Friction       friction   `json:"friction"`
}

type friction struct {
	Enabled        bool     `json:"enabled"`
	MinProbability *float64 `json:"min_probability,omitempty"`
	// MaxScore makes a signed score a friction positive only at or below the
	// ceiling; confidence alone must not turn praise into friction.
	MaxScore *float64 `json:"max_score,omitempty"`
	// MinScore is the high-is-bad direction: friction only at or above it.
	MinScore   *float64 `json:"min_score,omitempty"`
	CostUnit   string   `json:"cost_unit,omitempty"`
	Importance string   `json:"importance,omitempty"`
}

type windowRange struct {
	Since string `json:"since,omitempty"`
	Until string `json:"until,omitempty"`
}

type packRef struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type weighting struct {
	Mode       string  `json:"mode"`
	SampleRate float64 `json:"sample_rate"`
}

type coverageResponse struct {
	Window        windowRange     `json:"window"`
	Pack          packRef         `json:"pack"`
	EligibleRuns  int             `json:"eligible_runs"`
	ObservedRuns  int             `json:"observed_runs"`
	Coverage      float64         `json:"coverage"`
	CoverageBasis string          `json:"coverage_basis"`
	Weighting     weighting       `json:"weighting"`
	Labels        []coverageLabel `json:"labels"`
	Summary       stateCounts     `json:"summary"`
	Notes         []string        `json:"notes,omitempty"`
}

type value struct {
	Kind          string             `json:"kind"`
	Choice        string             `json:"choice,omitempty"`
	Score         *float64           `json:"score,omitempty"`
	Noul          *float64           `json:"noul,omitempty"`
	Number        *float64           `json:"number,omitempty"`
	Bool          *bool              `json:"bool,omitempty"`
	Text          string             `json:"text,omitempty"`
	Probabilities map[string]float64 `json:"probabilities,omitempty"`
}

type annotationView struct {
	ID              string   `json:"id"`
	State           string   `json:"state"`
	Method          string   `json:"method"`
	EvaluatorMethod string   `json:"evaluator_method,omitempty"`
	Value           value    `json:"value"`
	Probability     *float64 `json:"probability,omitempty"`
	Confidence      *float64 `json:"confidence,omitempty"`
	// FrictionProbability is the calibrated mass in the declared friction
	// region, which is the right confidence for a multi-level score.
	FrictionProbability *float64   `json:"friction_probability,omitempty"`
	DefinitionHash      string     `json:"definition_hash"`
	Model               string     `json:"model,omitempty"`
	Provider            string     `json:"provider,omitempty"`
	EvidenceEventIDs    []string   `json:"evidence_event_ids,omitempty"`
	ObservedAt          string     `json:"observed_at"`
	Reason              string     `json:"reason,omitempty"`
	Gate                gateStatus `json:"gate"`
	GateState           string     `json:"gate_state"`
	Proven              bool       `json:"proven"`
	Friction            friction   `json:"friction"`
}

type skippedLabel struct {
	LabelID string `json:"label_id"`
	Reason  string `json:"reason"`
}

type runCoverage struct {
	Observed int `json:"observed"`
	Labels   int `json:"labels"`
}

type runView struct {
	RunID    string           `json:"run_id"`
	Pack     string           `json:"pack"`
	Version  string           `json:"version"`
	State    string           `json:"state"`
	Coverage runCoverage      `json:"coverage"`
	Labels   []annotationView `json:"labels"`
	Counts   stateCounts      `json:"counts"`
	Missing  []string         `json:"missing,omitempty"`
	Skipped  []skippedLabel   `json:"skipped,omitempty"`
	Notes    []string         `json:"notes,omitempty"`
}

type runError struct {
	RunID  string `json:"run_id"`
	Reason string `json:"reason"`
}

type evaluateResult struct {
	Window      windowRange     `json:"window"`
	Submitted   int             `json:"submitted"`
	Remaining   int             `json:"remaining"`
	Truncated   bool            `json:"truncated"`
	Observed    int             `json:"observed"`
	Unavailable int             `json:"unavailable"`
	Coverage    float64         `json:"coverage"`
	Labels      []coverageLabel `json:"labels"`
	Runs        []runView       `json:"runs,omitempty"`
	Errors      []runError      `json:"errors,omitempty"`
	Counts      stateCounts     `json:"counts"`
	Passes      int             `json:"passes,omitempty"`
}

func packsCommand() cliapp.Command {
	return cliapp.Command{
		Name: "packs", NeedsAPI: true,
		Description: "List loaded label packs with version and content hash",
		Usage:       "agent-manager annotations packs [--json]",
		Args:        cliapp.ArgSchema{Flags: []cliapp.Flag{}},
		RunCtx: func(ctx cliapp.RunContext) error {
			body, err := ctx.Core().Request("GET", "annotations/packs", nil, nil)
			if err != nil {
				return err
			}
			if ctx.JSON() {
				_, err = ctx.Stdout().Write(append(body, '\n'))
				return err
			}
			var response packsResponse
			if err := json.Unmarshal(body, &response); err != nil {
				return err
			}
			renderPacks(ctx.Stdout(), response)
			return nil
		},
	}
}

func labelsCommand() cliapp.Command {
	return cliapp.Command{
		Name: "labels", NeedsAPI: true,
		Description: "List labels, gates and window coverage",
		Usage:       "agent-manager annotations labels [--pack NAME] [--window 7d] [--json]",
		Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
			{Name: "pack", Description: "Pack name (default: the loaded pack)"},
			{Name: "window", Default: "7d", Description: "Coverage window: 24h, 7d, 30d, all or RFC3339"},
		}},
		RunCtx: func(ctx cliapp.RunContext) error {
			query := url.Values{}
			if pack := strings.TrimSpace(ctx.Flag("pack")); pack != "" {
				query.Set("pack", pack)
			}
			query.Set("window", normalizedWindow(ctx.Flag("window")))
			body, err := ctx.Core().Request("GET", "annotations/labels", query, nil)
			if err != nil {
				return err
			}
			if ctx.JSON() {
				_, err = ctx.Stdout().Write(append(body, '\n'))
				return err
			}
			var response labelsResponse
			if err := json.Unmarshal(body, &response); err != nil {
				return err
			}
			renderLabels(ctx.Stdout(), response)
			return nil
		},
	}
}

func coverageCommand() cliapp.Command {
	return cliapp.Command{
		Name: "coverage", NeedsAPI: true,
		Description: "Show observed/compatible/not_observed/unavailable counts per label",
		Usage:       "agent-manager annotations coverage [--window 7d] [--json]",
		Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
			{Name: "window", Default: "7d", Description: "Coverage window: 24h, 7d, 30d, all or RFC3339"},
		}},
		RunCtx: func(ctx cliapp.RunContext) error {
			query := url.Values{"window": []string{normalizedWindow(ctx.Flag("window"))}}
			body, err := ctx.Core().Request("GET", "annotations/coverage", query, nil)
			if err != nil {
				return err
			}
			if ctx.JSON() {
				_, err = ctx.Stdout().Write(append(body, '\n'))
				return err
			}
			var response coverageResponse
			if err := json.Unmarshal(body, &response); err != nil {
				return err
			}
			renderCoverage(ctx.Stdout(), response)
			return nil
		},
	}
}

func showCommand() cliapp.Command {
	return cliapp.Command{
		Name: "show", NeedsAPI: true,
		Description: "Show the latest annotations and evidence refs for a run",
		Usage:       "agent-manager annotations show <run-id> [--json]",
		Args: cliapp.ArgSchema{
			Positionals: []cliapp.Positional{{Name: "run-id", Required: true, Description: "Run id"}},
			Flags:       []cliapp.Flag{},
		},
		RunCtx: func(ctx cliapp.RunContext) error {
			body, err := ctx.Core().Request("GET", "annotations/run/"+url.PathEscape(ctx.Positional("run-id")), nil, nil)
			if err != nil {
				return err
			}
			if ctx.JSON() {
				_, err = ctx.Stdout().Write(append(body, '\n'))
				return err
			}
			var view runView
			if err := json.Unmarshal(body, &view); err != nil {
				return err
			}
			renderRun(ctx.Stdout(), view)
			return nil
		},
	}
}

func runCommand() cliapp.Command {
	return cliapp.Command{
		Name: "run", NeedsAPI: true,
		Description: "Evaluate the pack for one run and show what was observed",
		Usage:       "agent-manager annotations run <run-id> [--force] [--include-optional] [--labels a,b] [--json]",
		Args: cliapp.ArgSchema{
			Positionals: []cliapp.Positional{{Name: "run-id", Required: true, Description: "Run id"}},
			Flags: []cliapp.Flag{
				{Name: "force", Bool: true, LocalOnly: true, Description: "Re-evaluate even when a cached observation exists"},
				{Name: "include-optional", Bool: true, LocalOnly: true, Description: "Include optional labels such as operator.emotion"},
				{Name: "labels", Description: "Comma-separated label ids (default: every active label)"},
				{Name: "max-labels", Description: "Maximum labels to evaluate"},
			},
		},
		RunCtx: func(ctx cliapp.RunContext) error {
			maxLabels, err := parsePositiveInt(ctx.Flag("max-labels"), "max-labels")
			if err != nil {
				return err
			}
			body, err := json.Marshal(map[string]any{
				"force":            ctx.BoolFlag("force"),
				"include_optional": ctx.BoolFlag("include-optional"),
				"labels":           parseLabels(ctx.Flag("labels")),
				"max_labels":       maxLabels,
			})
			if err != nil {
				return err
			}
			response, err := ctx.Core().Request("POST", "annotations/run/"+url.PathEscape(ctx.Positional("run-id")), nil, json.RawMessage(body))
			if err != nil {
				return err
			}
			if ctx.JSON() {
				_, err = ctx.Stdout().Write(append(response, '\n'))
				return err
			}
			var view runView
			if err := json.Unmarshal(response, &view); err != nil {
				return err
			}
			renderRun(ctx.Stdout(), view)
			return nil
		},
	}
}

func evaluateCommand() cliapp.Command {
	return cliapp.Command{
		Name: "evaluate", NeedsAPI: true,
		Description: "Run the annotation batch over a bounded run set",
		Usage:       "agent-manager annotations evaluate [--since RFC3339] [--limit N] [--run-id ID]... [--json]",
		Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
			{Name: "since", Description: "Only runs started at or after this RFC3339 time"},
			{Name: "limit", Default: "20", Description: "Maximum runs (capped by the pack budget)"},
			{Name: "run-id", Repeated: true, Description: "Explicit run id (repeatable); skips selection"},
			{Name: "include-optional", Bool: true, LocalOnly: true, Description: "Include optional labels"},
			{Name: "force", Bool: true, LocalOnly: true, Description: "Ignore cached observations"},
		}},
		RunCtx: func(ctx cliapp.RunContext) error {
			limit, err := parsePositiveInt(ctx.Flag("limit"), "limit")
			if err != nil {
				return err
			}
			explicit := ctx.FlagValues("run-id")
			// A pass stops at its wall-clock budget and persists what it
			// finished. Continue with the remainder here so one command still
			// backfills the requested number of runs without holding a single
			// HTTP request open past a gateway timeout. Explicit run ids cannot
			// be re-selected, so they stay one pass.
			result := evaluateResult{}
			remaining := limit
			for pass := 1; pass <= maxEvaluatePasses && remaining > 0; pass++ {
				body, err := evaluateBody(ctx.Flag("since"), remaining, explicit, ctx.BoolFlag("include-optional"), ctx.BoolFlag("force"))
				if err != nil {
					return err
				}
				response, err := ctx.Core().Request("POST", "annotations/evaluate", nil, body)
				if err != nil {
					return err
				}
				var current evaluateResult
				if err := json.Unmarshal(response, &current); err != nil {
					return err
				}
				accumulateEvaluate(&result, current)
				result.Passes = pass
				if len(explicit) > 0 || !current.Truncated || current.Remaining == 0 || current.Submitted == 0 {
					break
				}
				remaining = current.Remaining
				if !ctx.JSON() {
					fmt.Fprintf(ctx.Stderr(), "pass %d saved %d run(s); %d still queued\n", pass, current.Submitted, current.Remaining)
				}
			}
			if ctx.JSON() {
				encoded, err := json.Marshal(result)
				if err != nil {
					return err
				}
				_, err = ctx.Stdout().Write(append(encoded, '\n'))
				return err
			}
			renderEvaluate(ctx.Stdout(), result)
			return nil
		},
	}
}

// maxEvaluatePasses bounds the client-side continuation of a truncated pass.
// It is a safety valve, not a target: each pass is independently budgeted.
const maxEvaluatePasses = 20

// accumulateEvaluate folds one pass into the running result. Coverage counts
// add; labels come from the last pass because they describe the whole window
// snapshot rather than the slice this pass happened to read.
func accumulateEvaluate(total *evaluateResult, pass evaluateResult) {
	total.Window = pass.Window
	total.Submitted += pass.Submitted
	total.Observed += pass.Observed
	total.Unavailable += pass.Unavailable
	total.Remaining = pass.Remaining
	total.Truncated = pass.Truncated
	total.Labels = pass.Labels
	total.Runs = append(total.Runs, pass.Runs...)
	total.Errors = append(total.Errors, pass.Errors...)
	total.Counts.Observed += pass.Counts.Observed
	total.Counts.Compatible += pass.Counts.Compatible
	total.Counts.NotObserved += pass.Counts.NotObserved
	total.Counts.Unavailable += pass.Counts.Unavailable
	if total.Submitted > 0 {
		total.Coverage = float64(total.Observed) / float64(total.Submitted)
	}
}

// evaluateBody is pure so the CLI parser can be tested without an API.
func evaluateBody(since string, limit int, runIDs []string, includeOptional, force bool) (map[string]any, error) {
	since = strings.TrimSpace(since)
	if since != "" {
		if _, err := time.Parse(time.RFC3339, since); err != nil {
			return nil, fmt.Errorf("--since must be RFC3339: %w", err)
		}
	}
	body := map[string]any{
		"limit": limit, "include_optional": includeOptional, "force": force,
	}
	if since != "" {
		body["since"] = since
	}
	if len(runIDs) > 0 {
		body["run_ids"] = append([]string(nil), runIDs...)
	}
	return body, nil
}

func renderPacks(out interface{ Write([]byte) (int, error) }, response packsResponse) {
	for _, pack := range response.Packs {
		fmt.Fprintf(out, "%-20s %-24s %3d labels  mode=%s  hash=%s\n", pack.Name, pack.Version, pack.Labels, pack.Mode, shortHash(pack.PackHash))
	}
}

func renderLabels(out interface{ Write([]byte) (int, error) }, response labelsResponse) {
	fmt.Fprintf(out, "Pack %s %s  window=%s\n", response.Pack, response.Version, response.Window)
	for _, label := range response.Labels {
		optional := ""
		if label.Optional {
			optional = "  [optional]"
		}
		fmt.Fprintf(out, "%-30s %-14s gate=%-16s observed=%d compatible=%d not_observed=%d unavailable=%d%s\n",
			label.ID, label.Method, gateState(label.GateState), label.Coverage.Observed,
			label.Coverage.Compatible, label.Coverage.NotObserved, label.Coverage.Unavailable, optional)
		if label.Coverage.total() == 0 {
			fmt.Fprintf(out, "  %-28s coverage: no rows in the window (unknown, not zero)\n", "")
		}
	}
}

func renderCoverage(out interface{ Write([]byte) (int, error) }, response coverageResponse) {
	fmt.Fprintf(out, "Coverage: %s (%d/%d eligible runs)  pack=%s %s  window=%s..%s  basis=%s  weighting=%s(sample_rate=%.2f)\n",
		coveragePercent(response.Coverage), response.ObservedRuns, response.EligibleRuns,
		response.Pack.Name, response.Pack.Version, emptyAs(response.Window.Since, "start"),
		emptyAs(response.Window.Until, "now"), emptyAs(response.CoverageBasis, "annotated_population"),
		response.Weighting.Mode, response.Weighting.SampleRate)
	for _, label := range response.Labels {
		renderCoverageLabel(out, label)
	}
	fmt.Fprintf(out, "Summary: observed=%d compatible=%d not_observed=%d unavailable=%d",
		response.Summary.Observed, response.Summary.Compatible, response.Summary.NotObserved, response.Summary.Unavailable)
	if response.Summary.Stale > 0 {
		fmt.Fprintf(out, " stale=%d", response.Summary.Stale)
	}
	fmt.Fprintln(out)
	for _, note := range response.Notes {
		fmt.Fprintln(out, note)
	}
}

func renderCoverageLabel(out interface{ Write([]byte) (int, error) }, label coverageLabel) {
	rate := "-"
	if label.Rate != nil {
		rate = fmt.Sprintf("%.1f%%", *label.Rate*100)
	}
	friction := "off"
	if label.Friction.Enabled {
		friction = "on"
	}
	fmt.Fprintf(out, "%-30s %-11s gate=%-15s observed=%d/%d rate=%s friction=%s baseline=%s\n",
		label.ID, label.State, gateState(label.Gate.State), label.Observed, label.Eligible,
		rate, friction, emptyAs(label.BaselineState, "not_measured"))
	if label.Eligible == 0 {
		fmt.Fprintf(out, "  %-28s no rows in the window (unknown, not zero)\n", "")
	}
}

func renderRun(out interface{ Write([]byte) (int, error) }, view runView) {
	fmt.Fprintf(out, "Coverage: %d/%d labels observed  run=%s  pack=%s %s\n",
		view.Coverage.Observed, view.Coverage.Labels, view.RunID, view.Pack, view.Version)
	if len(view.Labels) == 0 {
		fmt.Fprintln(out, "No annotations recorded: coverage is unknown, not zero friction.")
	}
	sort.SliceStable(view.Labels, func(i, j int) bool { return view.Labels[i].ID < view.Labels[j].ID })
	for _, annotation := range view.Labels {
		fmt.Fprintf(out, "  %-30s %-11s method=%-14s %s gate=%s\n",
			annotation.ID, annotation.State, annotation.Method, annotationValue(annotation), gateState(annotation.GateState))
		if annotation.Reason != "" {
			fmt.Fprintf(out, "    reason: %s\n", annotation.Reason)
		}
		if len(annotation.EvidenceEventIDs) > 0 {
			fmt.Fprintf(out, "    evidence: %s\n", strings.Join(annotation.EvidenceEventIDs, ", "))
		}
	}
	fmt.Fprintf(out, "Counts: observed=%d compatible=%d not_observed=%d unavailable=%d\n",
		view.Counts.Observed, view.Counts.Compatible, view.Counts.NotObserved, view.Counts.Unavailable)
	if len(view.Missing) > 0 {
		fmt.Fprintf(out, "No observation for: %s (unknown, not absent)\n", strings.Join(view.Missing, ", "))
	}
	for _, skipped := range view.Skipped {
		if skipped.LabelID == "*" {
			continue
		}
		fmt.Fprintf(out, "  skipped %-30s %s\n", skipped.LabelID, skipped.Reason)
	}
	for _, note := range view.Notes {
		fmt.Fprintln(out, note)
	}
}

func renderEvaluate(out interface{ Write([]byte) (int, error) }, result evaluateResult) {
	fmt.Fprintf(out, "Coverage: %s (%d/%d submitted runs observed)  window=%s..%s\n",
		coveragePercent(result.Coverage), result.Observed, result.Submitted,
		emptyAs(result.Window.Since, "start"), emptyAs(result.Window.Until, "now"))
	for _, label := range result.Labels {
		renderCoverageLabel(out, label)
	}
	for _, run := range result.Runs {
		fmt.Fprintf(out, "  %-40s observed=%d compatible=%d not_observed=%d unavailable=%d\n",
			run.RunID, run.Counts.Observed, run.Counts.Compatible, run.Counts.NotObserved, run.Counts.Unavailable)
	}
	for _, failure := range result.Errors {
		fmt.Fprintf(out, "  %-40s error: %s\n", failure.RunID, failure.Reason)
	}
	if result.Submitted == 0 {
		fmt.Fprintln(out, "No runs were selected: coverage is unknown, not zero friction.")
	}
	if result.Truncated && result.Remaining > 0 {
		fmt.Fprintf(out, "Stopped at the pass budget with %d run(s) still queued; run the command again to continue.\n", result.Remaining)
	}
}

func coveragePercent(value float64) string { return fmt.Sprintf("%.1f%%", value*100) }

func emptyAs(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func annotationValue(annotation annotationView) string {
	switch {
	case annotation.Probability != nil:
		return fmt.Sprintf("p=%.2f", *annotation.Probability)
	case annotation.Value.Score != nil:
		return fmt.Sprintf("score=%.2f (no distribution)", *annotation.Value.Score)
	case annotation.Value.Noul != nil:
		return fmt.Sprintf("noul=%.2f", *annotation.Value.Noul)
	case annotation.Value.Choice != "":
		return "choice=" + annotation.Value.Choice + " (no distribution)"
	case annotation.Value.Text != "":
		return "value=" + annotation.Value.Text
	case annotation.Value.Number != nil:
		return fmt.Sprintf("value=%.2f", *annotation.Value.Number)
	default:
		return "value=unknown"
	}
}

func gateState(state string) string {
	if state == "" {
		return "unmeasured"
	}
	return state
}

func normalizedWindow(window string) string {
	window = strings.TrimSpace(window)
	if window == "" {
		return "7d"
	}
	return window
}

func parseLabels(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parsePositiveInt(raw, name string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return 0, fmt.Errorf("--%s must be a positive integer", name)
	}
	return value, nil
}

func shortHash(hash string) string {
	if len(hash) <= 12 {
		return hash
	}
	return hash[:12]
}

func schedulerCommand() cliapp.Command {
	return cliapp.Command{
		Name: "scheduler", NeedsAPI: true,
		Description: "Show the annotation scheduler queue, budget and deferral reasons",
		Usage:       "agent-manager annotations scheduler [--json]",
		HelpText:    "Reads the background annotation scheduler's status: mode, queue depth, running jobs, oldest due time, deferral reasons and the rolling-hour budget. A passive scheduler has a queue only when an on-demand pass is running; it never starts background inference until the operator selects idle, balanced or eager.",
		RunCtx: func(ctx cliapp.RunContext) error {
			body, err := ctx.Core().Request("GET", "annotations/scheduler", nil, nil)
			if err != nil {
				return err
			}
			if ctx.JSON() {
				_, err = ctx.Stdout().Write(append(body, '\n'))
				return err
			}
			var pretty bytes.Buffer
			if err := json.Indent(&pretty, body, "", "  "); err != nil {
				return err
			}
			_, err = fmt.Fprintln(ctx.Stdout(), pretty.String())
			return err
		},
	}
}
