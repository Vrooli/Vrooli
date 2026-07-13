package planlog

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	planmodel "plan-manager/internal/planmodel"
)

type staticURLResolver map[string]string

func (r staticURLResolver) ResolveScenarioURLDefault(_ context.Context, scenario string) (string, error) {
	if u := r[scenario]; u != "" {
		return u, nil
	}
	return "", errors.New("missing scenario URL")
}

func TestScenarioQABugReporterForwardsFullPayloadAndDraftDisposition(t *testing.T) {
	var got bugCaptureRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/teams/scenario-qa/bugs/capture" || r.Method != http.MethodPost {
			t.Fatalf("request %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(bugCaptureResponse{Disposition: "draft", DraftID: "bug-1", Needs: []string{"actual"}, NextAction: []string{"prompt-manager", "team", "bug-repair", "scenario-qa", "bug-1"}})
	}))
	defer server.Close()

	ref, err := NewScenarioQABugReporter(server.Client(), staticURLResolver{promptManagerSystem: server.URL}).FileBug(context.Background(), Entry{ID: "entry-1", Title: "cache drift", Bug: planmodel.BugReportPayload{SignalType: "regression", Severity: "major", Repro: []string{"start service"}, Expected: "fresh cache", Actual: "stale cache", Description: "details", Context: map[string]string{"scenario": "plan-manager"}, HonestyFlags: []string{"minimal-context"}}})
	if err != nil {
		t.Fatal(err)
	}
	if got.SignalType != "regression" || got.Severity != "major" || got.Actual != "stale cache" || got.Context["scenario"] != "plan-manager" {
		t.Fatalf("payload was reshaped: %#v", got)
	}
	if ref.Reference != "bug-1" || ref.Capture.State != "draft" || ref.Capture.DraftID != "bug-1" || len(ref.Capture.NextAction) == 0 {
		t.Fatalf("capture = %#v", ref)
	}
}

func TestSwarmRecordWriterForwardsClassificationAndPublishedDisposition(t *testing.T) {
	var got recordCaptureRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/records/capture" || r.Method != http.MethodPost {
			t.Fatalf("request %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(recordCaptureResponse{Disposition: "published", Record: recordResponse{ID: "rec-1"}})
	}))
	defer server.Close()

	ref, err := NewSwarmRecordWriter(server.Client(), staticURLResolver{swarmManagerSystem: server.URL}).WriteRecord(context.Background(), Entry{ID: "entry-1", Record: planmodel.RecordPayload{Kind: "refactor", Scenario: "swarm-manager", Trigger: "test failure", Approach: "repair", Evidence: "go test", Outcome: "partial", CreatedBy: "agent-7"}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != "refactor" || got.Scenario != "swarm-manager" || got.Outcome != "partial" || got.CreatedBy != "agent-7" {
		t.Fatalf("classification was reshaped: %#v", got)
	}
	if ref.Reference != "rec-1" || ref.Capture.State != "published" {
		t.Fatalf("capture = %#v", ref)
	}
}

func TestDownstreamAdaptersMapResolutionFailureToUnavailable(t *testing.T) {
	_, err := NewScenarioQABugReporter(http.DefaultClient, staticURLResolver{}).FileBug(context.Background(), Entry{ID: "entry-1"})
	var unavailable ErrDownstreamUnavailable
	if !errors.As(err, &unavailable) {
		t.Fatalf("FileBug error = %T %[1]v", err)
	}
	_, err = NewSwarmRecordWriter(http.DefaultClient, staticURLResolver{}).WriteRecord(context.Background(), Entry{ID: "entry-1"})
	if !errors.As(err, &unavailable) {
		t.Fatalf("WriteRecord error = %T %[1]v", err)
	}
}
