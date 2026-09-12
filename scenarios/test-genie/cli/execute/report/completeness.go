package report

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	execTypes "test-genie/cli/internal/execute"
)

// completenessBudget bounds the supplement's subprocess. The score read is
// <1s warm by contract; anything slower means the host is struggling and the
// report should not wait on it.
const completenessBudget = 2 * time.Second

// maxCompletenessRecommendations caps the rendered recommendation list.
const maxCompletenessRecommendations = 3

// ScoreRunner is the seam between the report printer and the
// scenario-completeness-scoring CLI. It returns the raw stdout of
// `scenario-completeness-scoring score get <scenario> --json` (protojson of
// GetScoreResponse) or an error. Tests substitute it; production wires
// RunScoreCLI via Printer.SetScoreRunner.
type ScoreRunner func(ctx context.Context, scenario string) ([]byte, error)

// RunScoreCLI shells the scoring CLI directly (NOT via `bash -lc` — login
// profile sourcing costs ~3s, which alone would blow the budget). A missing
// binary surfaces as an exec.ErrNotFound-wrapped error and the supplement
// silently skips.
func RunScoreCLI(ctx context.Context, scenario string) ([]byte, error) {
	path, err := exec.LookPath("scenario-completeness-scoring")
	if err != nil {
		return nil, err
	}
	return exec.CommandContext(ctx, path, "score", "get", scenario, "--json").Output()
}

// completenessPayload is the minimal slice of the GetScoreResponse protojson
// the supplement renders. Unknown fields are ignored by encoding/json, so
// the wire contract can grow without breaking this reader.
type completenessPayload struct {
	Maturity struct {
		WorkingRung      string `json:"workingRung"`
		LadderClean      bool   `json:"ladderClean"`
		SatisfiedThrough string `json:"satisfiedThrough"`
	} `json:"maturity"`
	Composite struct {
		Score          int32  `json:"score"`
		Classification string `json:"classification"`
	} `json:"composite"`
	Trend *struct {
		PreviousScore        int32  `json:"previousScore"`
		PreviousCalculatedAt string `json:"previousCalculatedAt"`
		Delta                int32  `json:"delta"`
	} `json:"trend"`
	Freshness struct {
		Phases []struct {
			Phase   string `json:"phase"`
			Verdict string `json:"verdict"`
		} `json:"phases"`
		SuggestedCommand string `json:"suggestedCommand"`
	} `json:"freshness"`
	Recommendations []struct {
		Priority     string  `json:"priority"`
		Description  string  `json:"description"`
		ImpactPoints float64 `json:"impactPoints"`
	} `json:"recommendations"`
}

// printCompletenessSummary renders the COMPLETENESS supplement after the
// requirements summary: maturity rung + composite score from the fast cached
// status layer (scenario-completeness-scoring), top recommendations, and a
// stale-phase hint. Best-effort by contract: hard 2s budget, and on ANY
// failure (binary absent, timeout, non-zero exit, malformed JSON) it renders
// nothing and never affects the report's exit code.
func (p *Printer) printCompletenessSummary() {
	summary := LoadCompletenessSummary(context.Background(), p.scoreRunner, p.scenario)
	p.PrintCompletenessSummary(summary)
}

// LoadCompletenessSummary reads the cached scenario-completeness-scoring payload
// into the shared terminal view projection. It is best-effort by contract:
// missing binaries, timeouts, non-zero exits, malformed JSON, and empty payloads
// all return nil so rendering never affects the suite result.
func LoadCompletenessSummary(parent context.Context, runner ScoreRunner, scenario string) *execTypes.CompletenessSummary {
	if runner == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(parent, completenessBudget)
	defer cancel()

	raw, err := runner(ctx, scenario)
	if err != nil {
		return nil
	}
	var payload completenessPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}
	if payload.Composite.Score == 0 && payload.Composite.Classification == "" {
		// Empty or unrecognizable payload — nothing trustworthy to render.
		return nil
	}

	summary := &execTypes.CompletenessSummary{
		Score:          payload.Composite.Score,
		Classification: payload.Composite.Classification,
		WorkingRung:    payload.Maturity.WorkingRung,
		LadderClean:    payload.Maturity.LadderClean,
		RefreshCommand: payload.Freshness.SuggestedCommand,
	}
	if summary.LadderClean {
		summary.WorkingRung = "ladder clean through R4"
	} else if summary.WorkingRung == "" {
		summary.WorkingRung = "R0"
	}
	if payload.Trend != nil {
		trend := &execTypes.CompletenessTrend{
			PreviousScore: payload.Trend.PreviousScore,
			Delta:         payload.Trend.Delta,
		}
		if parsed, err := time.Parse(time.RFC3339Nano, payload.Trend.PreviousCalculatedAt); err == nil {
			trend.PreviousDate = parsed.Format("2006-01-02")
		}
		summary.Trend = trend
	}
	for i, rec := range payload.Recommendations {
		if i >= maxCompletenessRecommendations {
			break
		}
		summary.Recommendations = append(summary.Recommendations, execTypes.CompletenessRecommendation{
			Priority:     rec.Priority,
			Description:  rec.Description,
			ImpactPoints: rec.ImpactPoints,
		})
	}
	for _, phase := range payload.Freshness.Phases {
		if phase.Verdict == "stale" || phase.Verdict == "unknown" {
			summary.StaleEvidence = append(summary.StaleEvidence, phase.Phase)
		}
	}
	sort.Strings(summary.StaleEvidence)
	return summary
}

// PrintCompletenessSummary renders the completeness portion of a shared
// RunStandingView. Callers pass the already-built summary so human and --json
// outputs cannot drift.
func (p *Printer) PrintCompletenessSummary(summary *execTypes.CompletenessSummary) {
	if summary == nil {
		return
	}
	fmt.Fprintln(p.w, p.color.Bold("COMPLETENESS (scenario-completeness-scoring):"))

	fmt.Fprintf(p.w, "  • Score: %s (%s) · working rung: %s\n",
		p.color.Bold(fmt.Sprintf("%d/100", summary.Score)),
		summary.Classification, summary.WorkingRung)
	if summary.Trend != nil {
		when := summary.Trend.PreviousDate
		if when == "" {
			when = "unknown"
		}
		fmt.Fprintf(p.w, "  • Trend: %s since %s (previous %d/100)\n",
			formatCompletenessDelta(summary.Trend.Delta),
			when,
			summary.Trend.PreviousScore,
		)
	}

	for i, rec := range summary.Recommendations {
		impact := ""
		if rec.ImpactPoints > 0 {
			impact = fmt.Sprintf(" (+%s pts)", strings.TrimSuffix(fmt.Sprintf("%.1f", rec.ImpactPoints), ".0"))
		}
		fmt.Fprintf(p.w, "      %d. [%s] %s%s\n", i+1, rec.Priority, rec.Description, impact)
	}

	if len(summary.StaleEvidence) > 0 {
		line := fmt.Sprintf("  • Stale evidence: %s", strings.Join(summary.StaleEvidence, ", "))
		if cmd := summary.RefreshCommand; cmd != "" {
			line += " — refresh: " + cmd
		}
		fmt.Fprintln(p.w, p.color.Yellow(line))
	}
	fmt.Fprintln(p.w)
}

func formatCompletenessDelta(delta int32) string {
	switch {
	case delta > 0:
		return fmt.Sprintf("↑%d", delta)
	case delta < 0:
		return fmt.Sprintf("↓%d", -delta)
	default:
		return "0"
	}
}
