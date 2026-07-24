package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/goals"
)

// The operator inbox regressed from milliseconds to seconds once before,
// because the projection quietly became O(items x whole-store scans). These
// two tests guard that from both sides: one asserts the shape of the algorithm
// without needing a server, the other asserts the latency an operator actually
// experiences against a live instance.

const (
	// Matches scripts/perf-baseline.sh defaults. Deliberately loose relative to
	// measured post-fix medians (feed 2ms warm / ~90ms cold, plan 77ms) so the
	// check catches regressions rather than noise.
	perfSmokeFeedBudgetMs = 300
	perfSmokePlanBudgetMs = 600

	// Production scale at the time of writing was 431 non-archived items and
	// 172 goals. The hermetic test runs above that so the budget is not met by
	// shrinking the corpus.
	perfScaleItems = 500
	perfScaleGoals = 60

	// One projection over the corpus above. Generous relative to measured cost
	// because CI hardware varies; a reintroduced per-item store scan costs
	// seconds, not milliseconds, so it fails this by orders of magnitude.
	perfScaleBudget = 3 * time.Second
)

// TestPerfSmokeEndpointBudgets runs the same measurement script an operator
// runs, against a live instance. It skips loudly when no instance is reachable:
// a skip is not a pass, and the hermetic scale test below still runs.
func TestPerfSmokeEndpointBudgets(t *testing.T) {
	script, err := filepath.Abs(filepath.Join("..", "scripts", "perf-baseline.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(script); statErr != nil {
		t.Fatalf("perf harness missing at %s: %v", script, statErr)
	}

	cmd := exec.Command(script, "--json", "--check")
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("PERF_FEED_MAX_MS=%d", perfSmokeFeedBudgetMs),
		fmt.Sprintf("PERF_PLAN_MAX_MS=%d", perfSmokePlanBudgetMs),
	)
	output, runErr := cmd.Output()
	if len(output) == 0 {
		t.Skipf("no running swarm-manager instance to measure (start it with 'vrooli scenario start swarm-manager'); the endpoint budgets were NOT checked")
	}

	var report struct {
		Endpoints []struct {
			Path     string `json:"path"`
			MedianMs int    `json:"median_ms"`
			BudgetMs int    `json:"budget_ms"`
			Status   string `json:"status"`
		} `json:"endpoints"`
		WithinBudget bool `json:"within_budget"`
	}
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("perf harness produced unreadable output: %v\n%s", err, output)
	}
	for _, endpoint := range report.Endpoints {
		t.Logf("%s median=%dms budget=%dms", endpoint.Path, endpoint.MedianMs, endpoint.BudgetMs)
	}
	if !report.WithinBudget || runErr != nil {
		for _, endpoint := range report.Endpoints {
			if endpoint.Status != "ok" {
				t.Errorf("%s median %dms exceeds its %dms budget", endpoint.Path, endpoint.MedianMs, endpoint.BudgetMs)
			}
		}
		if !t.Failed() {
			t.Fatalf("perf harness reported a budget breach: %s", output)
		}
	}
}

// TestNextActionProjectionHoldsAtProductionScale is the server-free half of the
// guard. It builds a corpus larger than production and asserts that one whole
// projection stays within budget while scanning each contributing store once.
// A reintroduced per-item scan fails this without needing a running instance.
func TestNextActionProjectionHoldsAtProductionScale(t *testing.T) {
	root := t.TempDir()
	handler := backlog.NewHandler(root, root)
	refs := make([]string, 0, perfScaleItems)
	for index := range perfScaleItems {
		item := backlog.BacklogItem{
			Name:    fmt.Sprintf("scale-item-%04d", index),
			Title:   fmt.Sprintf("Scale item %04d", index),
			Kind:    backlog.KindIdea,
			Status:  backlog.StatusSuggested,
			Created: "2026-07-24T00:00:00Z",
			Updated: "2026-07-24T00:00:00Z",
		}
		if err := os.MkdirAll(handler.Store().ItemDir(item.Kind, item.Name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := handler.Store().SaveItem(item); err != nil {
			t.Fatal(err)
		}
		refs = append(refs, backlog.ItemRef(item))
	}

	goalService := goals.NewService(goals.NewStore(root), handler.Store())
	for index := range perfScaleGoals {
		targets := refs[index*5 : index*5+5]
		if _, err := goalService.Create(goals.CreateRequest{
			Name:     fmt.Sprintf("scale-goal-%03d", index),
			Title:    fmt.Sprintf("Scale goal %03d", index),
			Priority: index % 10,
			Targets:  targets,
		}); err != nil {
			t.Fatal(err)
		}
	}

	counter := &countingDecisionCounter{counts: readyDecisionCounts{items: map[string]int{}, goals: map[string]int{}}}
	cache := newNextActionProjectionCache(nextActionFeed{backlog: handler, goals: goalService, decisions: counter})

	start := time.Now()
	entries, err := cache.Entries(t.Context())
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) < perfScaleItems {
		t.Fatalf("projection produced %d entries for %d items; the corpus is not being projected", len(entries), perfScaleItems)
	}
	if counter.scans != 1 {
		t.Fatalf("proposal store scanned %d times for a %d-item projection; want exactly 1", counter.scans, perfScaleItems)
	}
	if elapsed > perfScaleBudget {
		t.Fatalf("projection over %d items and %d goals took %s, over the %s budget", perfScaleItems, perfScaleGoals, elapsed, perfScaleBudget)
	}
	t.Logf("projection over %d items / %d goals: %s (budget %s), 1 proposal-store scan", perfScaleItems, perfScaleGoals, elapsed, perfScaleBudget)
}
