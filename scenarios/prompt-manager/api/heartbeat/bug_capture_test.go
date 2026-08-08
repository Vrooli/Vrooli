package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"prompt-manager/internal/paths"
	"prompt-manager/store"

	"github.com/gorilla/mux"
	_ "modernc.org/sqlite"
)

func TestAssessBugCapturePublishesOnlyCompleteTaxonomy(t *testing.T) {
	accepted, needs, invalid := assessBugCapture(BugCaptureRequest{Title: "cache stale", SignalType: "code defect", Severity: "high", Repro: []string{"start"}, Expected: "fresh", Actual: "stale", Description: "details", HonestyFlags: []string{"minimal-context"}})
	if len(needs) != 0 || len(invalid) != 0 {
		t.Fatalf("needs=%v invalid=%v", needs, invalid)
	}
	if accepted["signal_type"] != "code-defect" || accepted["severity"] != "blocker" {
		t.Fatalf("accepted=%v", accepted)
	}
}

func TestAssessBugCaptureRetainsHonestPartialObservation(t *testing.T) {
	_, needs, invalid := assessBugCapture(BugCaptureRequest{Title: "partial", SignalType: "made-up", Severity: "major", HonestyFlags: []string{"repro-not-attempted", "minimal-context"}})
	if len(invalid) != 1 || invalid[0].Field != "signal_type" {
		t.Fatalf("invalid=%v", invalid)
	}
	if !containsCaptureValue(needs, "expected") || !containsCaptureValue(needs, "actual") {
		t.Fatalf("needs=%v", needs)
	}
	if containsCaptureValue(needs, "repro (or honesty_flags=repro-not-attempted)") {
		t.Fatalf("honest repro omission should be accepted: %v", needs)
	}
}

func TestBugCaptureTopicIsStableAndPrivateDraftCommandIsExact(t *testing.T) {
	if got := bugSlug("Cache drift: plan sync!"); got != "cache-drift-plan-sync" {
		t.Fatalf("slug=%q", got)
	}
	argv := bugRepairCommand("bug-1")
	if len(argv) < 5 || argv[0] != "prompt-manager" || argv[3] != "scenario-qa" || argv[4] != "bug-1" {
		t.Fatalf("argv=%v", argv)
	}
}

func TestBugCaptureDraftRemainsPrivateUntilRepairPublishes(t *testing.T) {
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	h := NewHandlers(HandlersDeps{
		TeamStore:     teamStore,
		AgentStore:    agentStore,
		RelationStore: fileStore.Relations(),
		Executor:      newTestExecutor(t, teamStore, agentStore, nil, "", nil, nil),
	})
	if err := teamStore.Create(context.Background(), newIndependentTestTeam(scenarioQATeamID, "Scenario QA")); err != nil {
		t.Fatal(err)
	}
	header := encodeAttribution(t, store.AttributionInfo{Kind: store.KnowledgeKindOperatorDirect, SpawnOrigin: store.SpawnOriginOperatorCLI})
	invoke := func(method, path string, payload BugCaptureRequest) BugCaptureResponse {
		t.Helper()
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set(attributionHeaderName, header)
		vars := map[string]string{"id": scenarioQATeamID}
		if method != http.MethodPost {
			vars["draftId"] = path[len("/draft/"):]
		}
		req = mux.SetURLVars(req, vars)
		w := httptest.NewRecorder()
		if method == http.MethodPost {
			h.CaptureBug(w, req)
		} else {
			h.RepairBugCapture(w, req)
		}
		if w.Code != http.StatusOK {
			t.Fatalf("%s %s = %d: %s", method, path, w.Code, w.Body.String())
		}
		var result BugCaptureResponse
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		return result
	}

	draft := invoke(http.MethodPost, "/capture", BugCaptureRequest{Title: "cache stale", SignalType: "regression"})
	if draft.Disposition != "draft" || draft.DraftID == "" {
		t.Fatalf("draft = %+v", draft)
	}
	if entries, err := teamStore.ListTeamCorpus(context.Background(), scenarioQATeamID, "", "bug-inbox/", 0); err != nil || len(entries) != 0 {
		t.Fatalf("private draft leaked into inbox: entries=%+v err=%v", entries, err)
	}
	published := invoke(http.MethodPatch, "/draft/"+draft.DraftID, BugCaptureRequest{Severity: "major", Repro: []string{"start"}, Expected: "fresh", Actual: "stale", Description: "details"})
	if published.Disposition != "published" || published.Knowledge == nil {
		t.Fatalf("published = %+v", published)
	}
	if _, err := teamStore.GetBugDraft(context.Background(), scenarioQATeamID, draft.DraftID); err == nil {
		t.Fatal("published draft was retained privately")
	}
}

func containsCaptureValue(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
