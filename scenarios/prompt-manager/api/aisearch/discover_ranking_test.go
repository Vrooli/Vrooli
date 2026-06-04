package aisearch

import (
	"context"
	"strings"
	"testing"

	"prompt-manager/search"
	"prompt-manager/skills"
	"prompt-manager/store"
)

// --- Test fixtures for the block-aware ranking invariants (I1–I8) ---

func rankSkillFixture(id string, score float64) searchResultFixture {
	return searchResultFixture{
		ID:    "uuid-" + id,
		Score: score,
		Payload: map[string]interface{}{
			"skill_id":    id,
			"name":        id,
			"description": "",
			"folder":      "core",
			"tags":        []interface{}{},
			"modes":       []interface{}{},
		},
	}
}

func rankTopicFixture(topicID string, score float64) searchResultFixture {
	return searchResultFixture{
		ID:      "uuid-" + topicID,
		Score:   score,
		Payload: map[string]interface{}{"topic_id": topicID, "name": topicID},
	}
}

// rankSkill registers a skill in the mock store with the given content length.
func rankSkill(m *MockSkillStore, id string, chars int) {
	m.AddSkill("core", skills.Metadata{
		ID: id, Name: id, Description: "", File: id + ".md",
		Tags: []string{}, Modes: []string{"steer"},
	}, strings.Repeat("x", chars))
}

// rankTopic registers a topic that accumulates the given skills (own + ancestors
// already flattened — the mock returns AccumulateSkills verbatim).
func rankTopic(m *MockTopicStoreReader, topicID string, skillIDs ...string) {
	m.topics[topicID] = &store.Topic{ID: topicID, Name: topicID, Skills: skillIDs}
	m.ancestors[topicID] = nil
	m.skills[topicID] = skillIDs
}

func newRankService(t *testing.T, mockSkills *MockSkillStore, mockTopics *MockTopicStoreReader, topicFx, skillFx []searchResultFixture, ranking DiscoverRankingConfig) (*Service, func()) {
	t.Helper()
	embedder, closeEmbed := newTestEmbedder(t)
	topicVS, closeTopic := newTestVectorStore(t, "prompt-manager-topics", topicFx)
	skillVS, closeSkill := newTestVectorStore(t, "prompt-manager-skills", skillFx)
	svc := &Service{
		embedder:         embedder,
		vectorStore:      skillVS,
		skillStore:       mockSkills,
		searchService:    search.NewService(mockSkills),
		threshold:        0.5,
		topicVectorStore: topicVS,
		topicStore:       mockTopics,
		rankingConfig:    &MockRankingConfigProvider{cfg: ranking},
	}
	return svc, func() { closeEmbed(); closeTopic(); closeSkill() }
}

// indexOf returns the position of a result ID, or -1.
func indexOf(results []DiscoverResult, id string) int {
	for i, r := range results {
		if r.ID == id {
			return i
		}
	}
	return -1
}

func findResult(results []DiscoverResult, id string) *DiscoverResult {
	for i := range results {
		if results[i].ID == id {
			return &results[i]
		}
	}
	return nil
}

// I1/I2: a topic below the gate is not force-included; a topic at/above the gate
// includes its whole pack, each skill carrying the topic score (not its own).
func TestRanking_I1_I2_GateAndPackCarriesTopicScore(t *testing.T) {
	mockSkills := NewMockSkillStore()
	for _, id := range []string{"gated-a", "gated-b", "below-gate"} {
		rankSkill(mockSkills, id, 100)
	}
	mockTopics := NewMockTopicStoreReader()
	rankTopic(mockTopics, "tg", "gated-a", "gated-b")
	rankTopic(mockTopics, "tl", "below-gate")

	// tg clears the 0.55 gate; tl (0.52) does not. No direct skill hits, so pack
	// members can only carry their topic's score.
	topicFx := []searchResultFixture{rankTopicFixture("tg", 0.60), rankTopicFixture("tl", 0.52)}
	svc, cleanup := newRankService(t, mockSkills, mockTopics, topicFx, nil, DefaultDiscoverRankingConfig())
	defer cleanup()

	resp, err := svc.Discover(context.Background(), []string{"q"}, "", 20)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	for _, id := range []string{"gated-a", "gated-b"} {
		r := findResult(resp.Results, id)
		if r == nil {
			t.Fatalf("gated pack skill %q missing from results", id)
		}
		if r.Source != "topic" {
			t.Errorf("%q source = %q, want topic", id, r.Source)
		}
		if r.Score != 0.60 {
			t.Errorf("%q score = %.2f, want 0.60 (topic score, I1)", id, r.Score)
		}
	}
	if findResult(resp.Results, "below-gate") != nil {
		t.Error("below-gate skill should not be force-included (topic below gate, I2)")
	}
}

// I3/I5: a strong direct match ranks above the pack; the above-pack set is capped;
// remaining (lower) high-confidence individuals fall into the tail below the pack.
func TestRanking_I3_I5_StrongIndividualAbovePack(t *testing.T) {
	mockSkills := NewMockSkillStore()
	for _, id := range []string{"p1", "p2", "strong", "weak"} {
		rankSkill(mockSkills, id, 100)
	}
	mockTopics := NewMockTopicStoreReader()
	rankTopic(mockTopics, "tg", "p1", "p2")

	topicFx := []searchResultFixture{rankTopicFixture("tg", 0.60)}
	skillFx := []searchResultFixture{
		rankSkillFixture("strong", 0.81), // >= 0.65 bar → above pack
		rankSkillFixture("weak", 0.55),   // < 0.65 bar → tail (below pack)
	}
	svc, cleanup := newRankService(t, mockSkills, mockTopics, topicFx, skillFx, DefaultDiscoverRankingConfig())
	defer cleanup()

	resp, err := svc.Discover(context.Background(), []string{"q"}, "", 20)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	strongIdx := indexOf(resp.Results, "strong")
	p1Idx := indexOf(resp.Results, "p1")
	weakIdx := indexOf(resp.Results, "weak")
	if strongIdx == -1 || p1Idx == -1 || weakIdx == -1 {
		t.Fatalf("missing results: %+v", resp.Results)
	}
	if strongIdx > p1Idx {
		t.Errorf("strong (#%d) should rank above pack p1 (#%d) (I3)", strongIdx, p1Idx)
	}
	if weakIdx < p1Idx {
		t.Errorf("weak individual (#%d) should fall below the pack (#%d) (tail)", weakIdx, p1Idx)
	}
}

// I5: the above-pack count is bounded by MaxIndividualsAbovePack.
func TestRanking_I5_AbovePackCapBounds(t *testing.T) {
	mockSkills := NewMockSkillStore()
	for _, id := range []string{"p1", "hi1", "hi2", "hi3"} {
		rankSkill(mockSkills, id, 100)
	}
	mockTopics := NewMockTopicStoreReader()
	rankTopic(mockTopics, "tg", "p1")

	ranking := DefaultDiscoverRankingConfig()
	ranking.MaxIndividualsAbovePack = 1 // only the single strongest sits above

	topicFx := []searchResultFixture{rankTopicFixture("tg", 0.60)}
	skillFx := []searchResultFixture{
		rankSkillFixture("hi1", 0.90),
		rankSkillFixture("hi2", 0.85),
		rankSkillFixture("hi3", 0.80),
	}
	svc, cleanup := newRankService(t, mockSkills, mockTopics, topicFx, skillFx, ranking)
	defer cleanup()

	resp, err := svc.Discover(context.Background(), []string{"q"}, "", 20)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	p1Idx := indexOf(resp.Results, "p1")
	above := 0
	for i := 0; i < p1Idx; i++ {
		above++
	}
	if above != 1 {
		t.Errorf("expected exactly 1 individual above the pack, got %d", above)
	}
	// hi2/hi3 must appear below the pack despite clearing the bar.
	if indexOf(resp.Results, "hi2") < p1Idx || indexOf(resp.Results, "hi3") < p1Idx {
		t.Error("high-confidence individuals beyond the cap must fall below the pack")
	}
}

// I5 fallback: with no pack selected, results rank purely by score up to the
// limit — the above-pack cap does not apply.
func TestRanking_I5_NoPackPureScoreFallback(t *testing.T) {
	mockSkills := NewMockSkillStore()
	for _, id := range []string{"a", "b", "c", "d"} {
		rankSkill(mockSkills, id, 100)
	}
	mockTopics := NewMockTopicStoreReader()
	rankTopic(mockTopics, "tl", "x") // exists but stays below the gate

	topicFx := []searchResultFixture{rankTopicFixture("tl", 0.52)} // below 0.55 gate
	skillFx := []searchResultFixture{
		rankSkillFixture("a", 0.90),
		rankSkillFixture("b", 0.85),
		rankSkillFixture("c", 0.80),
		rankSkillFixture("d", 0.75),
	}
	svc, cleanup := newRankService(t, mockSkills, mockTopics, topicFx, skillFx, DefaultDiscoverRankingConfig())
	defer cleanup()

	resp, err := svc.Discover(context.Background(), []string{"q"}, "", 20)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	want := []string{"a", "b", "c", "d"}
	for i, id := range want {
		if i >= len(resp.Results) || resp.Results[i].ID != id {
			t.Fatalf("pure-score order mismatch at #%d: got %+v, want %v", i, resp.Results, want)
		}
	}
}

// I4: a selected pack is never fully crowded out, even by many higher-scoring
// individuals.
func TestRanking_I4_PackNeverCrowdedOut(t *testing.T) {
	mockSkills := NewMockSkillStore()
	rankSkill(mockSkills, "p1", 100)
	skillFx := []searchResultFixture{}
	for i := 0; i < 10; i++ {
		id := "ind" + string(rune('0'+i))
		rankSkill(mockSkills, id, 100)
		skillFx = append(skillFx, rankSkillFixture(id, 0.90))
	}
	mockTopics := NewMockTopicStoreReader()
	rankTopic(mockTopics, "tg", "p1")

	topicFx := []searchResultFixture{rankTopicFixture("tg", 0.60)}
	svc, cleanup := newRankService(t, mockSkills, mockTopics, topicFx, skillFx, DefaultDiscoverRankingConfig())
	defer cleanup()

	resp, err := svc.Discover(context.Background(), []string{"q"}, "", 20)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if findResult(resp.Results, "p1") == nil {
		t.Error("selected pack skill p1 was crowded out by high-scoring individuals (I4 violation)")
	}
}

// I6: a skill in both a selected pack and direct search is included once, via the
// pack, ranked by max(individual, topic).
func TestRanking_I6_DedupMaxScore(t *testing.T) {
	mockSkills := NewMockSkillStore()
	rankSkill(mockSkills, "dup", 100)
	mockTopics := NewMockTopicStoreReader()
	rankTopic(mockTopics, "tg", "dup")

	topicFx := []searchResultFixture{rankTopicFixture("tg", 0.60)}
	skillFx := []searchResultFixture{rankSkillFixture("dup", 0.95)} // stronger than topic
	svc, cleanup := newRankService(t, mockSkills, mockTopics, topicFx, skillFx, DefaultDiscoverRankingConfig())
	defer cleanup()

	resp, err := svc.Discover(context.Background(), []string{"q"}, "", 20)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	count := 0
	for _, r := range resp.Results {
		if r.ID == "dup" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("dup appeared %d times, want 1 (I6 dedup)", count)
	}
	r := findResult(resp.Results, "dup")
	if r.Source != "topic" {
		t.Errorf("dup source = %q, want topic (pack inclusion is authoritative)", r.Source)
	}
	if r.Score != 0.95 {
		t.Errorf("dup score = %.2f, want 0.95 (max of individual/topic, I6)", r.Score)
	}
}

// I7: over budget, the trim keeps the top-N individual + the pack and trims the
// embedding tail first — never amputating a protected item.
func TestRanking_I7_BlockAwareTrim(t *testing.T) {
	mockSkills := NewMockSkillStore()
	rankSkill(mockSkills, "hi", 1000)
	rankSkill(mockSkills, "p1", 1000)
	rankSkill(mockSkills, "t1", 3000)
	rankSkill(mockSkills, "t2", 3000)
	mockTopics := NewMockTopicStoreReader()
	rankTopic(mockTopics, "tg", "p1")

	topicFx := []searchResultFixture{rankTopicFixture("tg", 0.60)}
	skillFx := []searchResultFixture{
		rankSkillFixture("hi", 0.85), // protected: above pack
		rankSkillFixture("t1", 0.55), // tail
		rankSkillFixture("t2", 0.55), // tail
	}
	svc, cleanup := newRankService(t, mockSkills, mockTopics, topicFx, skillFx, DefaultDiscoverRankingConfig())
	defer cleanup()

	// minor budget = 4000; protected core (hi 1000 + p1 1000) = 2000 fits; each
	// 3000-char tail item overflows.
	resp, err := svc.Discover(context.Background(), []string{"q"}, "minor", 20)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if resp.BudgetStatus != "over" {
		t.Fatalf("budget status = %q, want over", resp.BudgetStatus)
	}
	rec := resp.RecommendedReadCommand
	if !strings.Contains(rec, "hi") || !strings.Contains(rec, "p1") {
		t.Errorf("recommended should keep protected hi + p1, got %q", rec)
	}
	if strings.Contains(rec, "t1") || strings.Contains(rec, "t2") {
		t.Errorf("recommended should trim the tail first, got %q", rec)
	}
}

// Topic-skill cap: packs are added whole in relevance order; an overflowing pack
// is skipped while a smaller, still-relevant one is kept (skip-and-continue).
func TestRanking_TopicSkillCap(t *testing.T) {
	mockSkills := NewMockSkillStore()
	for _, id := range []string{"s1", "s2", "x1", "b1", "b2", "b3", "b4", "b5", "b6", "b7", "b8"} {
		rankSkill(mockSkills, id, 100)
	}
	mockTopics := NewMockTopicStoreReader()
	rankTopic(mockTopics, "tsmall", "s1", "s2")                                   // 2 skills, most relevant
	rankTopic(mockTopics, "tbig", "b1", "b2", "b3", "b4", "b5", "b6", "b7", "b8") // 8 skills, overflows
	rankTopic(mockTopics, "tsmall2", "x1")                                        // 1 skill, fits after tbig is skipped

	ranking := DefaultDiscoverRankingConfig()
	ranking.TopicSkillCap = 3

	topicFx := []searchResultFixture{
		rankTopicFixture("tsmall", 0.70),
		rankTopicFixture("tbig", 0.65),
		rankTopicFixture("tsmall2", 0.60),
	}
	svc, cleanup := newRankService(t, mockSkills, mockTopics, topicFx, nil, ranking)
	defer cleanup()

	resp, err := svc.Discover(context.Background(), []string{"q"}, "", 50)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	// tsmall (2) included; tbig (8) overflows the cap and is skipped; tsmall2 (1)
	// fits in the remaining slot → proves whole-pack skip-and-continue.
	for _, id := range []string{"s1", "s2", "x1"} {
		if findResult(resp.Results, id) == nil {
			t.Errorf("expected %q from a kept pack", id)
		}
	}
	if findResult(resp.Results, "b1") != nil {
		t.Error("tbig overflowed the cap and must be skipped whole")
	}
}

// I8: operational modes (all/action) do NOT force-include topic packs — a skill
// reachable only via a gated topic must be absent.
func TestRanking_I8_OperationalModeNoPackInjection(t *testing.T) {
	mockSkills := NewMockSkillStore()
	rankSkill(mockSkills, "pack-only", 100)
	rankSkill(mockSkills, "direct", 100)
	mockTopics := NewMockTopicStoreReader()
	rankTopic(mockTopics, "tg", "pack-only") // strong topic, but operational mode ignores packs

	topicFx := []searchResultFixture{rankTopicFixture("tg", 0.90)}
	skillFx := []searchResultFixture{rankSkillFixture("direct", 0.70)}
	svc, cleanup := newRankService(t, mockSkills, mockTopics, topicFx, skillFx, DefaultDiscoverRankingConfig())
	svc.SetActionSearch(nil, &MockActionStore{actions: []store.Action{{
		ID: "scenario.status.show", Name: "Show Status", Description: "status",
		Status: store.StatusActive, Owner: store.ActionOwner{Type: "project", ID: "vrooli"},
		Command: store.ActionCommand{Argv: []string{"vrooli", "scenario", "status"}},
	}}})
	defer cleanup()

	// --type all: no pack injection.
	resp, err := svc.DiscoverTyped(context.Background(), []string{"q"}, "", 20, "all")
	if err != nil {
		t.Fatalf("DiscoverTyped(all): %v", err)
	}
	if findResult(resp.Results, "pack-only") != nil {
		t.Error("operational --type all must NOT inject a gated topic's pack (I8)")
	}
	if findResult(resp.Results, "direct") == nil {
		t.Error("direct search match should still appear in operational mode")
	}

	// Sanity: skill mode DOES inject the pack (proves the difference is mode, not data).
	respSkill, err := svc.DiscoverTyped(context.Background(), []string{"q"}, "", 20, "skill")
	if err != nil {
		t.Fatalf("DiscoverTyped(skill): %v", err)
	}
	if findResult(respSkill.Results, "pack-only") == nil {
		t.Error("skill mode should force-include the gated topic's pack")
	}
}
