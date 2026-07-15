package backlog

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/testutil"
	"swarm-manager/internal/workshop"

	"github.com/gorilla/mux"
	"google.golang.org/protobuf/encoding/protojson"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

// Integration coverage for the data seam used by the workshop-decision-prep
// heartbeat and the workshop-decision-sync skill.
//
// Heartbeat reference:
//   scenarios/prompt-manager/store/teams/director-swarm/members/workshop-decision-prep/HEARTBEAT.md
// Skill reference:
//   scenarios/prompt-manager/store/skills/packs/core/workshop-decision-sync/SKILL.md
//
// The narrative being pinned: prep heartbeat reads pending decisions ->
// produces a freshness-checkable cache -> sync skill answers one decision via
// workshop/save -> next pending-questions response drops the answered
// decision. Async clarify is exercised at the HTTP seam so the skill's
// clarify-and-move-on contract has a CI safety net.

// canonicalDecisionHash reproduces step 4 of HEARTBEAT.md:
//
//	"Recompute the canonical SHA-256 hash for each live decision using:
//	   topic, text, context, options.
//	 Ignore prose summaries and anticipated Q&A when hashing."
//
// If that recipe drifts (field rename, reorder, drop, or coverage change),
// the freshness sub-tests below fail loudly.
func canonicalDecisionHash(q PendingQuestion) string {
	optionsJSON, _ := json.Marshal(q.Options)
	payload := strings.Join([]string{q.Topic, q.Text, q.Context, string(optionsJSON)}, "\x1f")
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

// briefKey identifies a cached brief by the four machine-checkable fields the
// heartbeat preserves on each block (kind, name, round, item_id).
func briefKey(q PendingQuestion) string {
	return fmt.Sprintf("%s/%s#%d:%s", q.ItemKind, q.ItemName, q.RoundNumber, q.ID)
}

// twoDecisionRound returns a workshop round shaped like the production
// briefing payload: two unresolved decisions with options carrying labels and
// recommendation flags.
func twoDecisionRound(topicA, topicB string) workshop.Round {
	return workshop.Round{
		RoundNum:    1,
		GeneratedAt: "2026-01-01T00:00:00Z",
		Readiness:   map[string]int{"problem_clarity": 1, "scope_defined": 1, "approach_solid": 0, "testable": 0, "risk_awareness": 0},
		Items: []workshop.Item{
			{
				ID:      "d1",
				Type:    "decision",
				Topic:   topicA,
				Text:    "Choose " + topicA,
				Context: "Background for " + topicA,
				Options: []workshop.Option{
					{Key: "A", Label: topicA + "-A", Rationale: "rationale-A", Recommended: true},
					{Key: "B", Label: topicA + "-B", Rationale: "rationale-B"},
				},
			},
			{
				ID:      "d2",
				Type:    "decision",
				Topic:   topicB,
				Text:    "Choose " + topicB,
				Context: "Background for " + topicB,
				Options: []workshop.Option{
					{Key: "A", Label: topicB + "-A", Rationale: "rationale-A"},
					{Key: "B", Label: topicB + "-B", Rationale: "rationale-B", Recommended: true},
				},
			},
		},
	}
}

// seedTwoInitiativesThreeItems lays down the canonical fixture: 2 initiatives,
// 3 backlog items spread across them, each with two unresolved decisions.
// Returns the per-item round so individual sub-tests can mutate and re-save.
func seedTwoInitiativesThreeItems(t *testing.T, rootDir string) map[string]workshop.Round {
	t.Helper()
	specs := []struct {
		Name       string
		Initiative string
		Priority   int
		TopicA     string
		TopicB     string
	}{
		{"alpha", "north-star", 1, "Architecture", "Stack"},
		{"beta", "north-star", 2, "Persistence", "Auth"},
		{"gamma", "side-quest", 3, "Indexing", "Caching"},
	}
	rounds := make(map[string]workshop.Round, len(specs))
	for _, s := range specs {
		title := strings.ToUpper(s.Name[:1]) + s.Name[1:]
		createTestItem(t, rootDir, KindIdea, BacklogItem{
			Name: s.Name, Title: title, Status: StatusBacklog,
			Priority: s.Priority, Initiative: s.Initiative,
			Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
		})
		round := twoDecisionRound(s.TopicA, s.TopicB)
		testutil.WriteJSONFile(t,
			filepath.Join(rootDir, "ideas", s.Name, "workshop", "round-001.json"),
			round)
		rounds[s.Name] = round
	}
	return rounds
}

func decodePendingQuestions(t *testing.T, w *httptest.ResponseRecorder) PendingQuestionsResponse {
	t.Helper()
	testutil.AssertStatusOK(t, w)
	var resp PendingQuestionsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode pending-questions: %v", err)
	}
	return resp
}

func flattenWorkshopQuestions(resp PendingQuestionsResponse) []PendingQuestion {
	var out []PendingQuestion
	for _, item := range resp.Items {
		for _, q := range item.Questions {
			if q.Source == "workshop" {
				out = append(out, q)
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return briefKey(out[i]) < briefKey(out[j])
	})
	return out
}

func TestWorkshopDecisionPrep_Seed_PendingQuestionsExposesAllOpenDecisions(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	seedTwoInitiativesThreeItems(t, rootDir)

	resp := decodePendingQuestions(t, doPendingQuestions(t, h, "source=workshop&limit=20"))

	if len(resp.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(resp.Items))
	}
	questions := flattenWorkshopQuestions(resp)
	if len(questions) != 6 {
		t.Fatalf("expected 6 open workshop decisions across the seed, got %d", len(questions))
	}
	for _, q := range questions {
		if q.Topic == "" || q.Text == "" || q.Context == "" || len(q.Options) == 0 {
			t.Errorf("decision %s missing one of the four hash-input fields: %+v", briefKey(q), q)
		}
		if q.Selected != nil {
			t.Errorf("decision %s should be unresolved, got selected=%v", briefKey(q), *q.Selected)
		}
	}
}

func TestWorkshopDecisionPrep_FreshnessAllValid_NoEnrichmentNeeded(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	seedTwoInitiativesThreeItems(t, rootDir)

	first := flattenWorkshopQuestions(decodePendingQuestions(t, doPendingQuestions(t, h, "source=workshop&limit=20")))
	cache := make(map[string]string, len(first))
	for _, q := range first {
		cache[briefKey(q)] = canonicalDecisionHash(q)
	}

	second := flattenWorkshopQuestions(decodePendingQuestions(t, doPendingQuestions(t, h, "source=workshop&limit=20")))
	if len(second) != len(first) {
		t.Fatalf("re-query returned %d decisions, want %d", len(second), len(first))
	}

	var stale []string
	for _, q := range second {
		if cache[briefKey(q)] != canonicalDecisionHash(q) {
			stale = append(stale, briefKey(q))
		}
	}
	if len(stale) != 0 {
		t.Fatalf("freshness check should have reused all cached briefs, but these are stale: %v", stale)
	}
}

func TestWorkshopDecisionPrep_MutateOptions_OnlyMutatedBriefIsStale(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	rounds := seedTwoInitiativesThreeItems(t, rootDir)

	first := flattenWorkshopQuestions(decodePendingQuestions(t, doPendingQuestions(t, h, "source=workshop&limit=20")))
	cache := make(map[string]string, len(first))
	for _, q := range first {
		cache[briefKey(q)] = canonicalDecisionHash(q)
	}

	mutatedKey := "idea/alpha#1:d1"
	round := rounds["alpha"]
	for i := range round.Items {
		if round.Items[i].ID == "d1" {
			round.Items[i].Options = append(round.Items[i].Options, workshop.Option{
				Key: "C", Label: "Newly added option", Rationale: "ops added late",
			})
		}
	}
	testutil.WriteJSONFile(t,
		filepath.Join(rootDir, "ideas", "alpha", "workshop", "round-001.json"),
		round)

	second := flattenWorkshopQuestions(decodePendingQuestions(t, doPendingQuestions(t, h, "source=workshop&limit=20")))
	if len(second) != len(first) {
		t.Fatalf("re-query returned %d decisions, want %d", len(second), len(first))
	}

	var stale []string
	for _, q := range second {
		if cache[briefKey(q)] != canonicalDecisionHash(q) {
			stale = append(stale, briefKey(q))
		}
	}
	if len(stale) != 1 || stale[0] != mutatedKey {
		t.Fatalf("expected exactly %q to be stale, got %v", mutatedKey, stale)
	}

	// Field-coverage guard: if a future refactor drops one of the four hash
	// inputs, this guard catches it. Each field, mutated in isolation, must
	// shift the canonical hash.
	base := PendingQuestion{
		Topic:   "t",
		Text:    "x",
		Context: "c",
		Options: []WorkshopOption{{Key: "A", Label: "L"}},
	}
	baseline := canonicalDecisionHash(base)
	mutations := []func(*PendingQuestion){
		func(q *PendingQuestion) { q.Topic += "!" },
		func(q *PendingQuestion) { q.Text += "!" },
		func(q *PendingQuestion) { q.Context += "!" },
		func(q *PendingQuestion) { q.Options = append(q.Options, WorkshopOption{Key: "B", Label: "L2"}) },
	}
	for i, m := range mutations {
		mutated := base
		mutated.Options = append([]WorkshopOption{}, base.Options...)
		m(&mutated)
		if canonicalDecisionHash(mutated) == baseline {
			t.Errorf("hash recipe regression: mutation %d did not change the canonical hash; "+
				"the recipe in HEARTBEAT.md step 4 must cover topic, text, context, and options", i)
		}
	}
}

func TestWorkshopDecisionPrep_AnswerOneDecision_PendingQuestionsDropsIt(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{RunID: "run-answer", TaskID: "task-answer"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)
	enableAutoAdvanceSettings(t, rootDir)
	rounds := seedTwoInitiativesThreeItems(t, rootDir)

	first := flattenWorkshopQuestions(decodePendingQuestions(t, doPendingQuestions(t, h, "source=workshop&limit=20")))
	if len(first) != 6 {
		t.Fatalf("seed sanity: expected 6 open decisions, got %d", len(first))
	}

	// Answer beta/d1 by patching `selected` and round-tripping through the
	// existing workshop/save endpoint — same fetch-patch-save shape the skill
	// uses (see SKILL.md section 7).
	target := rounds["beta"]
	for i := range target.Items {
		if target.Items[i].ID == "d1" {
			target.Items[i].Selected = strPtr("A")
		}
	}
	body := makeWorkshopSaveBody(1, target)
	w := httptest.NewRecorder()
	h.WorkshopSave(w, workshopSaveRequest("idea", "beta", body))
	if w.Code != 200 {
		t.Fatalf("workshop/save returned %d: %s", w.Code, w.Body.String())
	}

	second := flattenWorkshopQuestions(decodePendingQuestions(t, doPendingQuestions(t, h, "source=workshop&limit=20")))
	if len(second) != 5 {
		t.Fatalf("after answering beta/d1, expected 5 open decisions, got %d", len(second))
	}
	for _, q := range second {
		if q.ItemName == "beta" && q.ID == "d1" {
			t.Fatalf("beta/d1 should have been dropped after workshop/save, but pending-questions still returned it")
		}
	}
}

func TestWorkshopDecisionPrep_SpawnClarification_DoesNotMutateDecision(t *testing.T) {
	h, rootDir, phase, _ := setupTestHandlerWithRunner(t, "run-clarify")
	seedTwoInitiativesThreeItems(t, rootDir)

	reqBody, err := protojson.Marshal(&apipb.CreateClarificationRequest{
		RoundNumber: 1,
		ItemId:      "d2",
		Message:     "Why does option B carry the recommendation here?",
	})
	if err != nil {
		t.Fatalf("marshal clarification request: %v", err)
	}

	httpReq := httptest.NewRequest("POST",
		"/api/v1/backlog/idea/gamma/workshop/clarification",
		strings.NewReader(string(reqBody)))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq = mux.SetURLVars(httpReq, map[string]string{"kind": "idea", "name": "gamma"})

	rec := httptest.NewRecorder()
	h.CreateClarification(rec, httpReq)
	if rec.Code != 201 {
		t.Fatalf("clarification create returned %d: %s", rec.Code, rec.Body.String())
	}

	var resp apipb.CreateClarificationResponse
	if err := protojson.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode clarification response: %v", err)
	}
	if resp.GetThread().GetId() == "" {
		t.Fatalf("expected clarification response to carry thread.id, got %+v", resp.GetThread())
	}
	if resp.GetThread().GetRunId() != "run-clarify" {
		t.Errorf("expected thread.run_id=run-clarify, got %q", resp.GetThread().GetRunId())
	}
	if !phase.started {
		t.Fatal("clarification handler should have started the clarification operation through the runner")
	}

	// The decision must NOT have been answered as a side effect — the skill's
	// clarify path is "spawn and move on", never "spawn and silently resolve".
	roundPath := filepath.Join(rootDir, "ideas", "gamma", "workshop", "round-001.json")
	saved := testutil.ReadJSONFile[workshop.Round](t, roundPath)
	var d2 *workshop.Item
	for i := range saved.Items {
		if saved.Items[i].ID == "d2" {
			d2 = &saved.Items[i]
		}
	}
	if d2 == nil {
		t.Fatal("expected gamma/round-001 to still contain decision d2")
	}
	if d2.Selected != nil {
		t.Errorf("clarification must not mutate selected; got selected=%v", *d2.Selected)
	}
	if d2.ClarificationID == nil || *d2.ClarificationID == "" {
		t.Errorf("expected ClarificationID to be linked to decision d2 after spawn, got %v", d2.ClarificationID)
	}

	// And the decision should still surface in the next pending-questions
	// response — it remains operator-actionable until they pick an option.
	post := flattenWorkshopQuestions(decodePendingQuestions(t, doPendingQuestions(t, h, "source=workshop&limit=20")))
	var stillPending bool
	for _, q := range post {
		if q.ItemName == "gamma" && q.ID == "d2" {
			stillPending = true
			break
		}
	}
	if !stillPending {
		t.Fatal("gamma/d2 should still appear in pending-questions after clarify spawn")
	}
}
