package phases

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestRunListJSONUsesAPIPayload(t *testing.T) {
	api := testAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/phases" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"items":[{"name":"unit","provider":"unit-health","source":"validation-provider"}],"count":1}`)
	})

	var out bytes.Buffer
	if err := run(api, []string{"list", "--json"}, &out); err != nil {
		t.Fatalf("run list: %v", err)
	}
	if !strings.Contains(out.String(), `"provider":"unit-health"`) {
		t.Fatalf("expected raw JSON payload, got %s", out.String())
	}
}

func TestRunListHumanRendersDisplayNameWithPhaseKey(t *testing.T) {
	api := testAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/phases" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"items":[{"name":"ui-health","displayName":"UI Health","provider":"ui-health","source":"validation-provider","description":"Validates UI contracts."}],"count":1}`)
	})

	var out bytes.Buffer
	if err := run(api, []string{"list"}, &out); err != nil {
		t.Fatalf("run list: %v", err)
	}
	if !strings.Contains(out.String(), "UI Health (ui-health)") {
		t.Fatalf("expected display name with phase key, got %s", out.String())
	}
}

func TestRunPlanHumanRendersDisplayNameWithPhaseKey(t *testing.T) {
	api := testAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/executions/plan" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"scenarioName":"demo","phases":[{"name":"ui-health","displayName":"UI Health","selectionStatus":"selected"}]}`)
	})

	var out bytes.Buffer
	if err := run(api, []string{"plan", "demo"}, &out); err != nil {
		t.Fatalf("run plan: %v", err)
	}
	if !strings.Contains(out.String(), "UI Health (ui-health)") {
		t.Fatalf("expected planned phase display name with key, got %s", out.String())
	}
}

func TestRunPlanHumanRendersAdaptiveProfileAndOmissions(t *testing.T) {
	api := testAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/executions/plan" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		fmt.Fprint(w, `{
			"scenarioName":"demo",
			"presetUsed":"quick",
			"profile":{"name":"quick","strategy":"budget_fast_feedback","budgetSeconds":180},
			"summary":{"estimatedDurationSeconds":70},
			"phases":[{"name":"structure","displayName":"Structure","selectionStatus":"selected","estimatedDurationSeconds":20,"estimateSource":"unknown","estimateUnknown":true,"selectionReasons":["selected_required"]}],
			"omittedPhases":[{"name":"performance","displayName":"Performance","selectionStatus":"omitted","estimatedDurationSeconds":200,"estimateSource":"scenario_history","omissionReasons":["omitted_budget_exceeded"]}]
		}`)
	})

	var out bytes.Buffer
	if err := run(api, []string{"plan", "demo", "--preset", "quick"}, &out); err != nil {
		t.Fatalf("run plan: %v", err)
	}
	text := out.String()
	for _, want := range []string{
		"profile: quick strategy=budget_fast_feedback budget=180s estimated=70s",
		"Structure (structure)",
		"selected: selected_required",
		"Performance (performance)",
		"omitted: omitted_budget_exceeded",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected output to contain %q, got %s", want, text)
		}
	}
}

func TestRunApplicabilityJSONPassesTargetAndPhase(t *testing.T) {
	api := testAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/phases/applicability" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("target"); got != "demo" {
			t.Fatalf("target query = %q, want demo", got)
		}
		if got := r.URL.Query().Get("phase"); got != "search" {
			t.Fatalf("phase query = %q, want search", got)
		}
		fmt.Fprint(w, `{"scenarioName":"demo","phase":{"name":"search","applicabilityStatus":"not_applicable"}}`)
	})

	var out bytes.Buffer
	if err := run(api, []string{"applicability", "demo", "--phase", "search", "--json"}, &out); err != nil {
		t.Fatalf("run applicability: %v", err)
	}
	if !strings.Contains(out.String(), `"applicabilityStatus":"not_applicable"`) {
		t.Fatalf("expected applicability JSON, got %s", out.String())
	}
}

func testAPIClient(t *testing.T, handler http.HandlerFunc) *cliutil.APIClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: server.URL}
	}, nil)
}
