package fix

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

// scoreListBudget bounds the priority-list subprocess; the score list is a cached
// read by contract.
const scoreListBudget = 15 * time.Second

// priorityScenarios returns the fleet's scenarios in priority order (highest
// first) by shelling `scenario-completeness-scoring score list --json`. This is
// the same keystone fleet query the background fleet scheduler ranks by, so
// `fix --fleet` walks the fleet in the same order the scheduler would re-test it
// (most-important-and-stale first). Mirrors how the execute report and scheduler
// reach the scoring CLI; no generated client dependency.
var priorityScenarios = func(ctx context.Context) ([]string, error) {
	path, err := exec.LookPath("scenario-completeness-scoring")
	if err != nil {
		return nil, fmt.Errorf("scenario-completeness-scoring CLI not found: %w", err)
	}
	cctx, cancel := context.WithTimeout(ctx, scoreListBudget)
	defer cancel()
	out, err := exec.CommandContext(cctx, path, "score", "list",
		"--json", "--sort", "priority", "--order", "desc", "--limit", "500").Output()
	if err != nil {
		return nil, fmt.Errorf("score list failed: %w", err)
	}
	var payload struct {
		Scores []struct {
			Scenario string  `json:"scenario"`
			Priority float64 `json:"priority"`
		} `json:"scores"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return nil, fmt.Errorf("parse score list: %w", err)
	}
	// Server already sorts by priority desc, but re-sort defensively for a
	// deterministic order even if the contract changes.
	sort.SliceStable(payload.Scores, func(i, j int) bool {
		if payload.Scores[i].Priority != payload.Scores[j].Priority {
			return payload.Scores[i].Priority > payload.Scores[j].Priority
		}
		return payload.Scores[i].Scenario < payload.Scores[j].Scenario
	})
	names := make([]string, 0, len(payload.Scores))
	for _, s := range payload.Scores {
		if name := strings.TrimSpace(s.Scenario); name != "" {
			names = append(names, name)
		}
	}
	return names, nil
}

// fleetFixScenario is one scenario's outcome in a fleet remediation pass.
type fleetFixScenario struct {
	Scenario        string `json:"scenario"`
	TotalCandidates int    `json:"totalCandidates"`
	AppliedCount    int    `json:"appliedCount"`
	Status          string `json:"status"` // fixed | clean | unreachable | error
	Error           string `json:"error,omitempty"`
}

// fleetFixReport is the fleet-wide remediation report (--json consumable so the
// meta-optimization loop can raise "X% of fleet failures are autofixable").
type fleetFixReport struct {
	Applied          bool               `json:"applied"`
	ScenariosWalked  int                `json:"scenariosWalked"`
	ScenariosDropped int                `json:"scenariosDropped"` // beyond --max-scenarios
	TotalCandidates  int                `json:"totalCandidates"`
	TotalApplied     int                `json:"totalApplied"`
	Scenarios        []fleetFixScenario `json:"scenarios"`
}

// runFleet walks the priority-ordered fleet, calling the Stage-2 deterministic
// aggregate per scenario (dry-run unless apply), bounded by maxScenarios +
// concurrency. It is the composition of Stage 2 (per-scenario deterministic
// remediation) and Stage 3 (priority-ordered fleet selection). Unreachable /
// no-provider scenarios are reported as such, never fatal.
func runFleet(apiClient *cliutil.APIClient, apply bool, rules, providers stringList, asJSON bool, maxScenarios, concurrency int, out io.Writer) error {
	names, err := priorityScenarios(context.Background())
	if err != nil {
		return err
	}
	dropped := 0
	if maxScenarios > 0 && len(names) > maxScenarios {
		dropped = len(names) - maxScenarios
		names = names[:maxScenarios]
	}
	if concurrency <= 0 {
		concurrency = 1
	}
	// Applying writes to disk; force sequential to keep the run auditable and
	// avoid concurrent writers, even if a higher concurrency was requested.
	if apply {
		concurrency = 1
	}

	report := fleetFixReport{Applied: apply, ScenariosWalked: len(names), ScenariosDropped: dropped}
	results := make([]fleetFixScenario, len(names))

	jobs := make(chan int)
	var wg sync.WaitGroup
	if concurrency > len(names) {
		concurrency = len(names)
	}
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				results[idx] = fixOneScenario(apiClient, names[idx], apply, rules, providers)
			}
		}()
	}
	for i := range names {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	for _, r := range results {
		report.Scenarios = append(report.Scenarios, r)
		report.TotalCandidates += r.TotalCandidates
		report.TotalApplied += r.AppliedCount
	}

	if asJSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}
	report.render(out)
	return nil
}

// fixOneScenario calls the per-scenario deterministic fix endpoint and folds the
// response into a fleet-row outcome. A request error becomes status=error
// (never aborts the fleet walk); zero candidates is status=clean.
func fixOneScenario(apiClient *cliutil.APIClient, scenario string, apply bool, rules, providers stringList) fleetFixScenario {
	row := fleetFixScenario{Scenario: scenario}
	body := map[string]any{"apply": apply}
	if len(rules) > 0 {
		body["ruleIds"] = []string(rules)
	}
	if len(providers) > 0 {
		body["providers"] = []string(providers)
	}
	path := fmt.Sprintf("/api/v1/scenarios/%s/fix/deterministic", url.PathEscape(scenario))
	raw, err := apiClient.Request(http.MethodPost, path, nil, body)
	if err != nil {
		row.Status = "error"
		row.Error = err.Error()
		return row
	}
	var rep deterministicReport
	if err := json.Unmarshal(raw, &rep); err != nil {
		row.Status = "error"
		row.Error = fmt.Sprintf("parse report: %v", err)
		return row
	}
	row.TotalCandidates = rep.TotalCandidates
	for _, p := range rep.Providers {
		for _, c := range p.Candidates {
			if c.Applied {
				row.AppliedCount++
			}
		}
	}
	switch {
	case rep.TotalCandidates == 0:
		row.Status = "clean"
	default:
		row.Status = "fixed"
	}
	return row
}

func (r fleetFixReport) render(out io.Writer) {
	mode := "DRY-RUN (preview)"
	if r.Applied {
		mode = "APPLIED"
	}
	fmt.Fprintf(out, "Fleet deterministic fix — %s\n", mode)
	fmt.Fprintf(out, "Scenarios walked: %d", r.ScenariosWalked)
	if r.ScenariosDropped > 0 {
		fmt.Fprintf(out, " (%d dropped beyond --max-scenarios)", r.ScenariosDropped)
	}
	fmt.Fprintf(out, "\nTotal candidates: %d", r.TotalCandidates)
	if r.Applied {
		fmt.Fprintf(out, " (%d applied)", r.TotalApplied)
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out)

	// Surface scenarios with candidates or errors first; clean ones are summarized.
	clean, errored := 0, 0
	for _, s := range r.Scenarios {
		switch {
		case s.Status == "clean":
			clean++
			continue
		case s.Status == "error":
			errored++
		}
		marker := "would fix"
		if r.Applied {
			marker = fmt.Sprintf("applied %d/", s.AppliedCount)
		}
		if s.Status == "error" {
			fmt.Fprintf(out, "• %-32s ! %s\n", s.Scenario, s.Error)
			continue
		}
		fmt.Fprintf(out, "• %-32s %s %d candidate(s)\n", s.Scenario, marker, s.TotalCandidates)
	}
	fmt.Fprintf(out, "\n%d clean, %d with candidates, %d errored.\n",
		clean, len(r.Scenarios)-clean-errored, errored)
	if r.TotalCandidates > 0 && !r.Applied {
		fmt.Fprintln(out, "Re-run with --apply to write these changes.")
	}
}
