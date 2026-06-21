package fix

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

// fleetTestClient builds an APIClient pointed at a deterministic-fix stub server.
// The handler returns a candidate report keyed by scenario so tests can script
// per-scenario outcomes (candidates / clean / error via 500).
func fleetTestClient(t *testing.T, perScenario map[string]deterministicReport, errScenarios map[string]bool) (*cliutil.APIClient, func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// path: /api/v1/scenarios/{name}/fix/deterministic
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		name := ""
		for i, p := range parts {
			if p == "scenarios" && i+1 < len(parts) {
				name = parts[i+1]
			}
		}
		if errScenarios[name] {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"no providers"}`))
			return
		}
		rep := perScenario[name]
		rep.Scenario = name
		_ = json.NewEncoder(w).Encode(rep)
	}))
	client := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}),
		func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: srv.URL} },
		func() string { return "" })
	return client, srv.Close
}

func withPriorityScenarios(t *testing.T, names []string) {
	t.Helper()
	prev := priorityScenarios
	priorityScenarios = func(context.Context) ([]string, error) { return names, nil }
	t.Cleanup(func() { priorityScenarios = prev })
}

func candReport(n int) deterministicReport {
	rep := deterministicReport{TotalCandidates: n}
	if n > 0 {
		var p struct {
			Provider   string `json:"provider"`
			Status     string `json:"status"`
			Candidates []struct {
				RuleID      string `json:"ruleId"`
				FilePath    string `json:"filePath"`
				Description string `json:"description"`
				Applied     bool   `json:"applied"`
			} `json:"candidates"`
			Messages []string `json:"messages"`
			Error    string   `json:"error"`
		}
		p.Provider = "quality-health"
		p.Status = "fixed"
		for i := 0; i < n; i++ {
			p.Candidates = append(p.Candidates, struct {
				RuleID      string `json:"ruleId"`
				FilePath    string `json:"filePath"`
				Description string `json:"description"`
				Applied     bool   `json:"applied"`
			}{RuleID: "R", FilePath: "f", Description: "d"})
		}
		rep.Providers = append(rep.Providers, p)
	}
	return rep
}

func TestRunFleetWalksPriorityOrderAndAggregates(t *testing.T) {
	withPriorityScenarios(t, []string{"flaky", "healthy", "broken"})
	client, closeFn := fleetTestClient(t,
		map[string]deterministicReport{"flaky": candReport(2), "healthy": candReport(0)},
		map[string]bool{"broken": true},
	)
	defer closeFn()

	var buf bytes.Buffer
	if err := runFleet(client, false, nil, nil, true, 0, 1, &buf); err != nil {
		t.Fatalf("runFleet: %v", err)
	}
	var rep fleetFixReport
	if err := json.Unmarshal(buf.Bytes(), &rep); err != nil {
		t.Fatalf("parse report: %v\n%s", err, buf.String())
	}
	if rep.ScenariosWalked != 3 || rep.TotalCandidates != 2 {
		t.Fatalf("report = %+v, want 3 walked / 2 candidates", rep)
	}
	if rep.Scenarios[0].Scenario != "flaky" || rep.Scenarios[0].Status != "fixed" {
		t.Fatalf("first scenario = %+v, want flaky/fixed (priority order)", rep.Scenarios[0])
	}
	statusByName := map[string]string{}
	for _, s := range rep.Scenarios {
		statusByName[s.Scenario] = s.Status
	}
	if statusByName["healthy"] != "clean" || statusByName["broken"] != "error" {
		t.Fatalf("statuses = %v, want healthy=clean broken=error", statusByName)
	}
}

func TestRunFleetMaxScenariosCap(t *testing.T) {
	withPriorityScenarios(t, []string{"a", "b", "c", "d"})
	client, closeFn := fleetTestClient(t, map[string]deterministicReport{}, nil)
	defer closeFn()

	var buf bytes.Buffer
	if err := runFleet(client, false, nil, nil, true, 2, 1, &buf); err != nil {
		t.Fatalf("runFleet: %v", err)
	}
	var rep fleetFixReport
	_ = json.Unmarshal(buf.Bytes(), &rep)
	if rep.ScenariosWalked != 2 || rep.ScenariosDropped != 2 {
		t.Fatalf("report = walked %d dropped %d, want 2/2", rep.ScenariosWalked, rep.ScenariosDropped)
	}
}

func TestRunFleetApplyForcesSequential(t *testing.T) {
	withPriorityScenarios(t, []string{"a", "b"})
	client, closeFn := fleetTestClient(t,
		map[string]deterministicReport{"a": appliedReport(1), "b": appliedReport(2)}, nil)
	defer closeFn()

	var buf bytes.Buffer
	// concurrency 8 requested, but --apply forces 1; assert it still aggregates.
	if err := runFleet(client, true, nil, nil, true, 0, 8, &buf); err != nil {
		t.Fatalf("runFleet apply: %v", err)
	}
	var rep fleetFixReport
	_ = json.Unmarshal(buf.Bytes(), &rep)
	if !rep.Applied || rep.TotalApplied != 3 {
		t.Fatalf("report = %+v, want applied / 3 applied", rep)
	}
}

func appliedReport(n int) deterministicReport {
	rep := candReport(n)
	for i := range rep.Providers {
		for j := range rep.Providers[i].Candidates {
			rep.Providers[i].Candidates[j].Applied = true
		}
	}
	return rep
}
