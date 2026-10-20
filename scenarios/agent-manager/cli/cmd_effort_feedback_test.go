package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/goalhome"
	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
)

const feedbackFixtureRun = "60b571e2-876c-4574-8744-5423f75e285e"

func feedbackFixtureServer(t *testing.T, runJSON, homeRef string, failSignals *bool) (*Services, *[]map[string]any) {
	t.Helper()
	var requests []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/runs/"+feedbackFixtureRun {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(runJSON))
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == apiconnect.AgentManagerServiceGetEffortBoardProcedure {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"rows":[{"enrollment":{"effortRef":"effort:fixture","destinationRef":"path:` + homeRef + `/GOAL.md"},"goalHome":{"path":"` + homeRef + `"}}]}`))
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == apiconnect.AgentManagerServiceRecordEffortSignalProcedure {
			var req map[string]any
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("decode signal request: %v", err)
			}
			requests = append(requests, req)
			if failSignals != nil && *failSignals {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"message":"fixture unavailable"}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"effortRef":"effort:fixture","seq":"7","kind":"feedback","runId":"60b571e2-876c-4574-8744-5423f75e285e","detail":"FB-201 — Reconcile the run"}`))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(server.Close)
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: server.URL}
	}, nil)
	return NewServices(api), &requests
}

func feedbackFixtureRunJSON(runID, effortRef string, verified bool) string {
	verifiedJSON := "false"
	if verified {
		verifiedJSON = "true"
	}
	return `{"run":{"id":"` + runID + `","workReferences":[{"kind":"effort","id":"` + effortRef + `","verified":` + verifiedJSON + `,"visibility":"WORK_REFERENCE_VISIBILITY_PUBLIC","state":"WORK_REFERENCE_STATE_ACTIVE","relationship":"orchestrator"}]}}`
}

func feedbackArgs(home, detail string) []string {
	return []string{"add", home, "--effort-ref", "effort:fixture", "--run", feedbackFixtureRun, "--id", "FB-201", "--title", "Reconcile the run", "--detail", detail, "--json"}
}

func feedbackHome(t *testing.T) (string, string) {
	t.Helper()
	root := cliutil.ResolveRepoRoot()
	home, err := os.MkdirTemp(root, ".agent-manager-feedback-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(home) })
	if err := os.WriteFile(filepath.Join(home, goalhome.FeedbackFile), []byte("# Feedback\n\n## Open\n\n## Resolved\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(root, home)
	if err != nil {
		t.Fatal(err)
	}
	return home, filepath.ToSlash(rel)
}

func TestEffortFeedbackWritesThenSignalsAndIdenticalRetryDoesNotDuplicate(t *testing.T) {
	home, homeRef := feedbackHome(t)
	services, requests := feedbackFixtureServer(t, feedbackFixtureRunJSON(feedbackFixtureRun, "effort:fixture", true), homeRef, nil)
	app := &App{services: services}
	for i := 0; i < 2; i++ {
		if err := app.effortFeedback(feedbackArgs(home, "Use the admitted run reference.")); err != nil {
			t.Fatal(err)
		}
	}
	content, err := os.ReadFile(filepath.Join(home, goalhome.FeedbackFile))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(content), "### FB-201") != 1 || len(*requests) != 2 {
		t.Fatalf("retry duplicated file entry or skipped signal retry: entries=%d signals=%d", strings.Count(string(content), "### FB-201"), len(*requests))
	}
	for _, req := range *requests {
		if req["dedupe_key"] != "feedback:FB-201" || req["run_id"] != feedbackFixtureRun || req["effort_ref"] != "effort:fixture" || req["kind"] != "feedback" {
			t.Fatalf("signal is not exact/idempotent: %#v", req)
		}
	}
}

func TestEffortFeedbackPreflightRefusesMismatchedRunWithoutWritingOrSignaling(t *testing.T) {
	home, homeRef := feedbackHome(t)
	services, requests := feedbackFixtureServer(t, feedbackFixtureRunJSON(feedbackFixtureRun, "effort:other", true), homeRef, nil)
	app := &App{services: services}
	before, _ := os.ReadFile(filepath.Join(home, goalhome.FeedbackFile))
	if err := app.effortFeedback(feedbackArgs(home, "Must not be written.")); err == nil {
		t.Fatal("mismatched active effort reference was accepted")
	}
	after, _ := os.ReadFile(filepath.Join(home, goalhome.FeedbackFile))
	if string(before) != string(after) || len(*requests) != 0 {
		t.Fatal("preflight refusal had a file or signal effect")
	}
}

func TestEffortFeedbackRefusesUnenrolledGoalHomeBeforeWritingOrSignaling(t *testing.T) {
	_, actualHomeRef := feedbackHome(t)
	otherHome, _ := feedbackHome(t)
	services, requests := feedbackFixtureServer(t, feedbackFixtureRunJSON(feedbackFixtureRun, "effort:fixture", true), actualHomeRef, nil)
	app := &App{services: services}
	before, _ := os.ReadFile(filepath.Join(otherHome, goalhome.FeedbackFile))
	if err := app.effortFeedback(feedbackArgs(otherHome, "Must target the enrolled home.")); err == nil {
		t.Fatal("non-enrolled goal home was accepted")
	}
	after, _ := os.ReadFile(filepath.Join(otherHome, goalhome.FeedbackFile))
	if string(before) != string(after) || len(*requests) != 0 {
		t.Fatal("goal-home mismatch had a file or signal effect")
	}
}

func TestEffortFeedbackRequiresVerifiedPublicActiveSupervisorReference(t *testing.T) {
	cases := []struct {
		name string
		json string
	}{
		{"unverified", feedbackFixtureRunJSON(feedbackFixtureRun, "effort:fixture", false)},
		{"private", strings.Replace(feedbackFixtureRunJSON(feedbackFixtureRun, "effort:fixture", true), "WORK_REFERENCE_VISIBILITY_PUBLIC", "WORK_REFERENCE_VISIBILITY_PRIVATE", 1)},
		{"expired", strings.Replace(feedbackFixtureRunJSON(feedbackFixtureRun, "effort:fixture", true), "WORK_REFERENCE_STATE_ACTIVE", "WORK_REFERENCE_STATE_EXPIRED", 1)},
		{"worker", strings.Replace(feedbackFixtureRunJSON(feedbackFixtureRun, "effort:fixture", true), "orchestrator", "worker", 1)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home, homeRef := feedbackHome(t)
			services, requests := feedbackFixtureServer(t, tc.json, homeRef, nil)
			app := &App{services: services}
			before, _ := os.ReadFile(filepath.Join(home, goalhome.FeedbackFile))
			if err := app.effortFeedback(feedbackArgs(home, "Must not be written.")); err == nil {
				t.Fatal("unqualified work reference was accepted")
			}
			after, _ := os.ReadFile(filepath.Join(home, goalhome.FeedbackFile))
			if string(before) != string(after) || len(*requests) != 0 {
				t.Fatal("preflight refusal had a file or signal effect")
			}
		})
	}
}

func TestEffortFeedbackSignalFailureReportsFileOnlyAndRetryConverges(t *testing.T) {
	fail := true
	home, homeRef := feedbackHome(t)
	services, requests := feedbackFixtureServer(t, feedbackFixtureRunJSON(feedbackFixtureRun, "effort:fixture", true), homeRef, &fail)
	app := &App{services: services}
	call := captureRunIdentityStdoutAllowError(t, func() error {
		return app.effortFeedback(feedbackArgs(home, "Retry after the signal transport recovers."))
	})
	var result struct {
		FileStatus   string `json:"file_status"`
		SignalStatus string `json:"signal_status"`
	}
	if err := json.Unmarshal([]byte(call.stdout), &result); err != nil {
		t.Fatalf("partial result was not valid JSON: %v; output=%s", err, call.stdout)
	}
	if call.err == nil || result.FileStatus != "written" || result.SignalStatus != "unknown" || len(*requests) != 1 {
		t.Fatalf("partial result was hidden or mislabeled: err=%v output=%s requests=%d", call.err, call.stdout, len(*requests))
	}
	fail = false
	if err := app.effortFeedback(feedbackArgs(home, "Retry after the signal transport recovers.")); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(home, goalhome.FeedbackFile))
	if err != nil || strings.Count(string(content), "### FB-201") != 1 || len(*requests) != 2 {
		t.Fatalf("retry failed to converge: err=%v entries=%d requests=%d", err, strings.Count(string(content), "### FB-201"), len(*requests))
	}
}

func TestEffortFeedbackContentConflictDoesNotSignalAgain(t *testing.T) {
	home, homeRef := feedbackHome(t)
	services, requests := feedbackFixtureServer(t, feedbackFixtureRunJSON(feedbackFixtureRun, "effort:fixture", true), homeRef, nil)
	app := &App{services: services}
	if err := app.effortFeedback(feedbackArgs(home, "Original body.")); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(filepath.Join(home, goalhome.FeedbackFile))
	if err := app.effortFeedback(feedbackArgs(home, "Changed body.")); err == nil {
		t.Fatal("changed content reused existing feedback ID")
	}
	after, _ := os.ReadFile(filepath.Join(home, goalhome.FeedbackFile))
	if string(before) != string(after) || len(*requests) != 1 {
		t.Fatal("content conflict changed the file or signaled the orchestrator")
	}
}
