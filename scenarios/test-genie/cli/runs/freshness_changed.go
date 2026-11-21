package runs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliutil"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

// `runs freshness --changed` is the advisory fan-out mode: it maps the repo's
// current git change-set to the scenarios it touches and asks CheckFreshness
// for each one. It is the surface `vrooli hygiene` (and through it the
// pre-commit hook) consumes, so its degradation contract matters more than its
// output: every failure path collapses to {checked:false} rather than an
// error, and exit 1 means "stale scenarios found", never "check broke".

// maxChangedScenarios bounds the per-invocation work; a change-set touching
// more scenarios gets verdicts for the first N (sorted) plus a truncation note.
const maxChangedScenarios = 5

const changedFreshnessTimeout = 10 * time.Second

// FreshnessWarning is one stale-scenario advisory.
type FreshnessWarning struct {
	Scenario    string   `json:"scenario"`
	StalePhases []string `json:"stale_phases"`
	Command     string   `json:"command"`
}

// FreshnessAdvisory is the advisory result for the current change-set.
// Plain JSON (not protojson): it aggregates many CheckFreshness responses.
type FreshnessAdvisory struct {
	// Checked is true when git enumeration worked and test-genie returned at
	// least one verdict; false means the check was skipped (advisory
	// degradation), never that everything is fresh.
	Checked   bool               `json:"checked"`
	Scenarios []string           `json:"scenarios,omitempty"`
	Warnings  []FreshnessWarning `json:"warnings,omitempty"`
	Truncated bool               `json:"truncated,omitempty"`
}

// gitLines is a package var so tests can fake git enumeration. An empty dir
// runs in the current working directory.
var gitLines = func(ctx context.Context, dir string, args ...string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, line := range strings.Split(string(out), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

func runFreshnessChanged(apiClient *cliutil.APIClient, jsonOut bool, w io.Writer) error {
	ctx, cancel := context.WithTimeout(context.Background(), changedFreshnessTimeout)
	defer cancel()

	advisory := adviseChanged(ctx, apiClient)
	if jsonOut {
		encoded, err := json.MarshalIndent(advisory, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(encoded))
	} else {
		renderChangedAdvisory(w, advisory)
	}
	if len(advisory.Warnings) > 0 {
		return &exitErr{code: exitRegression, err: fmt.Errorf("%d changed scenario(s) have stale phases", len(advisory.Warnings))}
	}
	return nil
}

func adviseChanged(ctx context.Context, apiClient *cliutil.APIClient) FreshnessAdvisory {
	advisory := FreshnessAdvisory{}
	roots, err := gitLines(ctx, "", "rev-parse", "--show-toplevel")
	if err != nil || len(roots) == 0 {
		return advisory // not a git repo: skip, do not fail
	}
	root := roots[0]

	scenarios := changedScenarios(ctx, root)
	if len(scenarios) == 0 {
		advisory.Checked = true
		return advisory
	}
	if len(scenarios) > maxChangedScenarios {
		scenarios = scenarios[:maxChangedScenarios]
		advisory.Truncated = true
	}
	advisory.Scenarios = scenarios

	cl, err := client(apiClient)
	if err != nil {
		return FreshnessAdvisory{}
	}

	// Per-scenario checks run concurrently: each involves a server-side tree
	// digest, so sequential calls would make multi-scenario change-sets slow.
	// A failed check skips that scenario only; if every check fails
	// (test-genie down) the advisory degrades whole.
	type verdict struct {
		warning *FreshnessWarning
		ok      bool
	}
	verdicts := make([]verdict, len(scenarios))
	var wg sync.WaitGroup
	for i, scenario := range scenarios {
		wg.Add(1)
		go func(i int, scenario string) {
			defer wg.Done()
			resp, err := cl.CheckFreshness(ctx, connect.NewRequest(&runspb.CheckFreshnessRequest{Target: scenario}))
			if err != nil {
				return // advisory means advisory — skip silently
			}
			verdicts[i].ok = true
			var stale []string
			for _, p := range resp.Msg.GetPhases() {
				if p.GetStatus() != "fresh" {
					stale = append(stale, p.GetPhase())
				}
			}
			if len(stale) > 0 {
				verdicts[i].warning = &FreshnessWarning{
					Scenario:    scenario,
					StalePhases: stale,
					Command:     resp.Msg.GetSuggestedCommand(),
				}
			}
		}(i, scenario)
	}
	wg.Wait()

	anyOK := false
	for _, v := range verdicts {
		if v.ok {
			anyOK = true
		}
		if v.warning != nil {
			advisory.Warnings = append(advisory.Warnings, *v.warning)
		}
	}
	if !anyOK {
		return FreshnessAdvisory{}
	}
	advisory.Checked = true
	return advisory
}

// changedScenarios maps the repo's current change-set (staged, unstaged, and
// untracked paths) to the scenario names they touch, sorted and de-duplicated.
func changedScenarios(ctx context.Context, root string) []string {
	var paths []string
	for _, args := range [][]string{
		{"diff", "--cached", "--name-only"},
		{"diff", "--name-only"},
		{"ls-files", "--others", "--exclude-standard"},
	} {
		lines, err := gitLines(ctx, root, args...)
		if err != nil {
			continue
		}
		paths = append(paths, lines...)
	}
	return ScenariosFromPaths(paths)
}

// ScenariosFromPaths extracts scenario names from repo-relative paths by the
// `scenarios/<name>/...` prefix. Paths outside scenarios/ (shared packages,
// docs, root files) map to nothing — the v1 freshness digest only scopes to
// scenario directories, so warning on them would be noise we cannot back up.
func ScenariosFromPaths(paths []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, p := range paths {
		p = strings.TrimPrefix(strings.ReplaceAll(strings.TrimSpace(p), "\\", "/"), "./")
		rest, ok := strings.CutPrefix(p, "scenarios/")
		if !ok {
			continue
		}
		name, _, ok := strings.Cut(rest, "/")
		if !ok || strings.TrimSpace(name) == "" {
			continue // a file directly under scenarios/, not a scenario dir
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func renderChangedAdvisory(w io.Writer, advisory FreshnessAdvisory) {
	if !advisory.Checked {
		fmt.Fprintln(w, "Freshness check skipped (not a git repo, or the test-genie API is unreachable).")
		return
	}
	if len(advisory.Scenarios) == 0 {
		fmt.Fprintln(w, "No scenario changes in the current change-set.")
		return
	}
	if len(advisory.Warnings) == 0 {
		fmt.Fprintf(w, "All %d changed scenario(s) fresh against their current trees.\n", len(advisory.Scenarios))
		return
	}
	for _, warning := range advisory.Warnings {
		fmt.Fprintf(w, "⚠ test-genie hasn't run [%s] since the latest changes in %s.\n  Run: %s\n",
			strings.Join(warning.StalePhases, ", "), warning.Scenario, warning.Command)
	}
	if advisory.Truncated {
		fmt.Fprintln(w, "  (change-set touches more scenarios; only the first few were checked)")
	}
}
