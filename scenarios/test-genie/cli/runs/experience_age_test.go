package runs

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestRunsExperienceAgeShowsStoredVerdictAndFailsClosed(t *testing.T) {
	tests := []struct {
		name, state, resultStatus, wantText string
		wantCode                            int
	}{
		{"current", "evaluated_after_last_ui_change", "passed", "Evaluated after the last UI change", 0},
		{"current failed", "evaluated_after_last_ui_change", "failed", "Evaluated after the last UI change", exitRegression},
		{"current not evaluated", "evaluated_after_last_ui_change", "not_evaluated", "Evaluated after the last UI change", exitNotComparable},
		{"stale", "older_than_last_ui_change", "passed", "Older than the last UI change", exitRegression},
		{"never", "never_run", "", "Never run", exitRegression},
		{"no ui", "ui_surface_unavailable", "passed", "No UI surface to compare", exitNotComparable},
		{"unknown", "future_state", "passed", "Unknown freshness", exitNotComparable},
		{"missing", "", "passed", "Unknown freshness", exitNotComparable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/scenarios/demo" {
					t.Errorf("request path = %q", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				if tt.state == "never_run" {
					fmt.Fprint(w, `{"scenarioName":"demo","experienceFreshness":"never_run"}`)
					return
				}
				fmt.Fprintf(w, `{"scenarioName":"demo","experienceFreshness":%q,"lastExperienceStatus":%q,"lastExperienceAt":"2026-11-15T10:00:00Z","lastExperienceRunId":"run-1","lastUIChangeAt":"2026-11-15T09:00:00Z"}`, tt.state, tt.resultStatus)
			}))
			defer server.Close()
			apiClient := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{Client: server.Client()}), func() cliutil.APIBaseOptions {
				return cliutil.APIBaseOptions{Override: server.URL}
			}, nil)

			var out strings.Builder
			err := runExperienceAge(apiClient, []string{"--scenario", "demo"}, &out)
			if tt.wantCode == 0 {
				if err != nil {
					t.Fatalf("runExperienceAge error: %v", err)
				}
			} else {
				var coded interface{ ExitCode() int }
				if !errors.As(err, &coded) || coded.ExitCode() != tt.wantCode {
					t.Fatalf("runExperienceAge error = %v, want exit %d", err, tt.wantCode)
				}
			}
			if !strings.Contains(out.String(), tt.wantText) {
				t.Fatalf("output %q does not show freshness %q", out.String(), tt.wantText)
			}
			if tt.resultStatus != "" && !strings.Contains(out.String(), "Stored verdict: "+displayStatus(tt.resultStatus)+" · evaluated 2026-11-15T10:00:00Z · run run-1") {
				t.Fatalf("output did not identify result as stored with age and run: %q", out.String())
			}
		})
	}
}

func TestValidateExperienceAgeReportRejectsCurrentWithoutTimestamps(t *testing.T) {
	err := validateExperienceAgeReport(experienceAgeReport{
		ScenarioName:         "demo",
		ExperienceFreshness:  "evaluated_after_last_ui_change",
		LastExperienceStatus: "passed",
	})
	if err == nil {
		t.Fatal("current status without stored/UI timestamps must be rejected")
	}
}

func TestRunsExperienceAgeJSONIncludesBackendFreshness(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"scenarioName":"demo","experienceFreshness":"older_than_last_ui_change","lastExperienceAt":"2026-11-15T10:00:00Z","lastUIChangeAt":"2026-11-15T11:00:00Z"}`)
	}))
	defer server.Close()
	apiClient := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{Client: server.Client()}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{Override: server.URL}
	}, nil)
	var out strings.Builder
	err := runExperienceAge(apiClient, []string{"--scenario", "demo", "--json"}, &out)
	var coded interface{ ExitCode() int }
	if !errors.As(err, &coded) || coded.ExitCode() != exitRegression {
		t.Fatalf("stale JSON run error = %v, want regression exit", err)
	}
	if !strings.Contains(out.String(), `"experienceFreshness": "older_than_last_ui_change"`) {
		t.Fatalf("JSON missing backend freshness: %s", out.String())
	}
}
