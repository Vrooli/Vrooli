package report

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"
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
	if p.scoreRunner == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), completenessBudget)
	defer cancel()

	raw, err := p.scoreRunner(ctx, p.scenario)
	if err != nil {
		return
	}
	var payload completenessPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return
	}
	if payload.Composite.Score == 0 && payload.Composite.Classification == "" {
		// Empty or unrecognizable payload — nothing trustworthy to render.
		return
	}

	fmt.Fprintln(p.w, p.color.Bold("COMPLETENESS (scenario-completeness-scoring):"))

	rung := payload.Maturity.WorkingRung
	if payload.Maturity.LadderClean {
		rung = "ladder clean through R4"
	} else if rung == "" {
		rung = "R0"
	}
	fmt.Fprintf(p.w, "  • Score: %s (%s) · working rung: %s\n",
		p.color.Bold(fmt.Sprintf("%d/100", payload.Composite.Score)),
		payload.Composite.Classification, rung)

	for i, rec := range payload.Recommendations {
		if i >= maxCompletenessRecommendations {
			break
		}
		impact := ""
		if rec.ImpactPoints > 0 {
			impact = fmt.Sprintf(" (+%s pts)", strings.TrimSuffix(fmt.Sprintf("%.1f", rec.ImpactPoints), ".0"))
		}
		fmt.Fprintf(p.w, "      %d. [%s] %s%s\n", i+1, rec.Priority, rec.Description, impact)
	}

	var stale []string
	for _, phase := range payload.Freshness.Phases {
		if phase.Verdict == "stale" || phase.Verdict == "unknown" {
			stale = append(stale, phase.Phase)
		}
	}
	if len(stale) > 0 {
		sort.Strings(stale)
		line := fmt.Sprintf("  • Stale evidence: %s", strings.Join(stale, ", "))
		if cmd := payload.Freshness.SuggestedCommand; cmd != "" {
			line += " — refresh: " + cmd
		}
		fmt.Fprintln(p.w, p.color.Yellow(line))
	}
	fmt.Fprintln(p.w)
}
