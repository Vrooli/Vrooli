package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/vrooli/cli-core/teamattribution"
	"net/http"
	"net/http/httptest"
	"testing"

	"prompt-manager/internal/paths"
	"prompt-manager/internal/store"

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

func TestBugCaptureOwnerKeepsTeamAttributionAndTaxonomyChecks(t *testing.T) {
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
	sourceSkill, targetTeam := "report-bug", scenarioQATeamID
	validAttribution := encodeAttribution(t, store.AttributionInfo{
		Kind: store.KnowledgeKindWriterSkill, SourceSkillID: &sourceSkill, TeamID: &targetTeam,
		SpawnOrigin: store.SpawnOriginHeartbeat,
	})
	call := func(teamID, attribution string, payload BugCaptureRequest) *httptest.ResponseRecorder {
		t.Helper()
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, "/capture", bytes.NewReader(body))
		if attribution != "" {
			req.Header.Set(attributionHeaderName, attribution)
		}
		req = mux.SetURLVars(req, map[string]string{"id": teamID})
		w := httptest.NewRecorder()
		h.CaptureBug(w, req)
		return w
	}

	if got := call("other-team", validAttribution, BugCaptureRequest{}); got.Code != http.StatusBadRequest {
		t.Fatalf("non-QA target status=%d body=%s", got.Code, got.Body.String())
	}
	if got := call(scenarioQATeamID, "", BugCaptureRequest{}); got.Code != http.StatusBadRequest {
		t.Fatalf("missing attribution status=%d body=%s", got.Code, got.Body.String())
	}
	partial := call(scenarioQATeamID, validAttribution, BugCaptureRequest{Title: "partial observation", SignalType: "made-up", Severity: "major"})
	if partial.Code != http.StatusOK {
		t.Fatalf("invalid/incomplete taxonomy status=%d body=%s", partial.Code, partial.Body.String())
	}
	var result BugCaptureResponse
	if err := json.Unmarshal(partial.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Disposition != "draft" || result.DraftID == "" || len(result.Invalid) == 0 || len(result.Needs) == 0 {
		t.Fatalf("invalid/incomplete intake was not retained as a diagnostic draft: %+v", result)
	}
	if entries, err := teamStore.ListTeamCorpus(context.Background(), scenarioQATeamID, "", "bug-inbox/", 0); err != nil || len(entries) != 0 {
		t.Fatalf("invalid/incomplete report reached inbox: entries=%+v err=%v", entries, err)
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

func TestBugCaptureUsesRetryKeyWithoutCollapsingLongTitles(t *testing.T) {
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	h := NewHandlers(HandlersDeps{TeamStore: teamStore, AgentStore: agentStore, RelationStore: fileStore.Relations(), Executor: newTestExecutor(t, teamStore, agentStore, nil, "", nil, nil)})
	if err := teamStore.Create(context.Background(), newIndependentTestTeam(scenarioQATeamID, "Scenario QA")); err != nil {
		t.Fatal(err)
	}
	header := teamattribution.Encode(teamattribution.WriterSkill(teamattribution.Info{SpawnOrigin: teamattribution.SpawnOriginHeartbeat}, "report-bug", scenarioQATeamID))
	capture := func(key, title string) BugCaptureResponse {
		t.Helper()
		body, err := json.Marshal(BugCaptureRequest{Title: title, IdempotencyKey: key, SignalType: "code-defect", Severity: "minor", Repro: []string{"Run the owner check"}, Expected: "current observation", Actual: "stale observation", Description: "Measured maintenance finding", HonestyFlags: []string{"ai-generated-summary"}})
		if err != nil {
			t.Fatal(err)
		}
		req := mux.SetURLVars(httptest.NewRequest(http.MethodPost, "/capture", bytes.NewReader(body)), map[string]string{"id": scenarioQATeamID})
		req.Header.Set(attributionHeaderName, header)
		w := httptest.NewRecorder()
		h.CaptureBug(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("capture: %d %s", w.Code, w.Body.String())
		}
		var result BugCaptureResponse
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result.Disposition != "published" || result.Knowledge == nil {
			t.Fatalf("writer contract rejected: %+v", result)
		}
		return result
	}
	prefix := "model policy drift provider runner catalog revision observation "
	first := capture("finding-stale", prefix+"stale")
	second := capture("finding-version", prefix+"version")
	if first.Knowledge.ID == second.Knowledge.ID {
		t.Fatal("distinct retry keys collapsed into one old truncated-title topic")
	}
	replay := capture("finding-stale", "a corrected display title")
	if replay.Knowledge.ID != first.Knowledge.ID {
		t.Fatal("same retry key published a duplicate after display-title change")
	}
}
