package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ladderFixture is Offer Desk's live GetReleaseLadder response captured
// 2026-09-15, re-stamped so the producer time is fresh (or removed).
func ladderFixture(t *testing.T, generatedAt *time.Time) json.RawMessage {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "offer-desk-release-ladder.json"))
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	delete(payload, "generatedAt")
	if generatedAt != nil {
		payload["generatedAt"] = generatedAt.UTC().Format(time.RFC3339Nano)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func ladderMetric() MetricEntry {
	return MetricEntry{ID: "release_ladder", Label: "Release ladder", Kind: "ladder", Unit: "count", Coverage: CoverageInReach, TTLSeconds: 300,
		Source: SourceBinding{Binding: "scenario:offer-desk", Read: "/vrooli.offer_desk.v1.offers.ReleaseLadderService/GetReleaseLadder", IntegrationID: "offer-desk", FeatureID: "release_ladder", Selector: "release_ladder", ContractVersion: "legacy.v1", ExpectedUnit: "count", SourceTimePolicy: "producer_required", TTLSeconds: 300}}
}

func readLadder(t *testing.T, body json.RawMessage) MetricEntry {
	t.Helper()
	s := NewServer(testRegistry())
	s.offer = staticUpstreamClient{name: "offer-desk", body: body}
	out, _ := s.readings(context.Background(), []MetricEntry{ladderMetric()})
	return out[0]
}

func TestLadderReadingCarriesTheWholeScheduleWhenProducerTimeIsFresh(t *testing.T) { // [REQ:CC-P1-015]
	now := time.Now().Add(-5 * time.Second)
	m := readLadder(t, ladderFixture(t, &now))
	if m.Trust != TrustValid || m.ObservedAt == nil {
		t.Fatalf("trust=%s reason=%q, want VALID with producer time", m.Trust, m.TrustReason)
	}
	if m.Ladder == nil || len(m.Ladder.Rungs) != 14 || len(m.Rows) != 14 || m.Value != float64(14) {
		t.Fatalf("ladder rungs/rows/value = %v/%d/%v, want all 14", m.Ladder, len(m.Rows), m.Value)
	}
	ladder := m.Ladder
	if ladder.NextRank != 1 || ladder.Rungs[0].Name != "web-console" || ladder.Rungs[0].Status != "TRIGGER_MET" {
		t.Fatalf("next rung = %d %+v", ladder.NextRank, ladder.Rungs[0])
	}
	blockers := map[string]bool{}
	for _, work := range ladder.Rungs[0].Blockers {
		blockers[work.Name] = work.Direct
	}
	want := map[string]bool{"ai-gateway": true, "landing-page-business-suite": true, "scenario-to-desktop": false, "deployment-manager": false}
	if len(blockers) != len(want) {
		t.Fatalf("rung 1 blockers = %v, want %v", blockers, want)
	}
	for name, direct := range want {
		if got, ok := blockers[name]; !ok || got != direct {
			t.Fatalf("rung 1 blocker %s direct=%v present=%v, want direct=%v", name, got, ok, direct)
		}
	}
	if len(ladder.Rungs[0].Goals) != 1 || ladder.Rungs[0].Goals[0].Name != "release-ladder-offer-desk" {
		t.Fatalf("rung 1 goals = %+v", ladder.Rungs[0].Goals)
	}
	if ladder.Rungs[0].Readiness.Reported {
		t.Fatal("readiness must read as not reported when the producer states no readiness goal")
	}
	opens := map[string]int{}
	for _, unlock := range ladder.Reach {
		opens[unlock.Kind+":"+unlock.Name] = unlock.OpensAt
	}
	for key, rank := range map[string]int{"ramp:desktop": 1, "ramp:agent": 2, "ramp:local": 7, "ramp:mobile": 9, "stream:voice_minutes": 1, "stream:ai_credits": 4, "stream:compute_minutes": 10, "stream:workflow_executions": 10, "audience:developer": 1, "audience:business": 4, "audience:personal": 5} {
		if opens[key] != rank {
			t.Errorf("%s opens at %d, want %d", key, opens[key], rank)
		}
	}
	if len(ladder.Unscheduled) != 1 || ladder.Unscheduled[0].Name != "treasury" {
		t.Fatalf("unscheduled = %+v, want treasury", ladder.Unscheduled)
	}
	if m.Rows[0].Share != 0.6 || m.Rows[0].Detail != "TRIGGER_MET" || m.Rows[1].Share != 0.2 {
		t.Fatalf("rows carry lifecycle stage: %+v %+v", m.Rows[0], m.Rows[1])
	}
}

func TestLadderWithoutProducerTimeIsUntrusted(t *testing.T) { // [REQ:CC-P1-015]
	m := readLadder(t, ladderFixture(t, nil))
	if m.Trust != TrustUntrusted || m.TrustReason != "producer did not supply observation time" {
		t.Fatalf("trust=%s reason=%q", m.Trust, m.TrustReason)
	}
}

func TestLadderWithDuplicateRanksIsUntrusted(t *testing.T) { // [REQ:CC-P1-015]
	now := time.Now().Add(-5 * time.Second)
	body := `{"generatedAt":"` + now.UTC().Format(time.RFC3339) + `","entries":[{"deliverable":{"id":"a","name":"a","releaseRank":1,"status":"IDEA"}},{"deliverable":{"id":"b","name":"b","releaseRank":1,"status":"IDEA"}}]}`
	m := readLadder(t, json.RawMessage(body))
	if m.Trust != TrustUntrusted || !strings.Contains(m.TrustReason, "no plausible schedule") {
		t.Fatalf("trust=%s reason=%q", m.Trust, m.TrustReason)
	}
}

func TestLadderNextRungSkipsShippedWorkAndBlockersExcludeShippedEnablers(t *testing.T) { // [REQ:CC-P1-015]
	var payload any
	raw := `{"entries":[
		{"deliverable":{"id":"b","name":"second","releaseRank":2,"status":"IDEA"}},
		{"deliverable":{"id":"a","name":"first","releaseRank":1,"status":"SHIPPED"}}],
	 "enabling":[{"node":{"name":"done-work","status":"SHIPPED"},"derivedUrgency":1},{"node":{"name":"open-work","status":"ACTIVE"},"derivedUrgency":2},{"node":{"name":"later-work","status":"IDEA"},"derivedUrgency":3}]}`
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatal(err)
	}
	ladder, ok := releaseLadder(payload)
	if !ok {
		t.Fatal("valid ladder refused")
	}
	if ladder.Rungs[0].Name != "first" || ladder.NextRank != 2 {
		t.Fatalf("rungs sorted by rank and next skips shipped: %+v next=%d", ladder.Rungs, ladder.NextRank)
	}
	blockers := ladder.Rungs[1].Blockers
	if len(blockers) != 1 || blockers[0].Name != "open-work" {
		t.Fatalf("rung 2 blockers = %+v, want only open work due by rank 2", blockers)
	}
}

func TestRegistryRefusesUnknownKindAndLayout(t *testing.T) {
	metric := `{"id":"x","label":"x","kind":%KIND%,"coverage":"NOW","source":{"binding":"scenario:offer-desk","integrationId":"offer-desk","featureId":"x","selector":"x","sourceTimePolicy":"producer_required"}}`
	write := func(body string) string {
		path := filepath.Join(t.TempDir(), "registry.json")
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	ok := `{"schemaVersion":"1","rooms":[{"id":"r","title":"r","metricIds":["x"],"beats":[{"hero":"x","layout":"wide"}]}],"metrics":[` + strings.ReplaceAll(metric, "%KIND%", `"ladder"`) + `]}`
	if _, err := LoadRegistry(write(ok)); err != nil {
		t.Fatalf("ladder kind with wide layout refused: %v", err)
	}
	badKind := strings.ReplaceAll(ok, `"kind":"ladder"`, `"kind":"chart"`)
	if _, err := LoadRegistry(write(badKind)); err == nil || !strings.Contains(err.Error(), "invalid kind") {
		t.Fatalf("unknown kind accepted: %v", err)
	}
	badLayout := strings.ReplaceAll(ok, `"layout":"wide"`, `"layout":"sideways"`)
	if _, err := LoadRegistry(write(badLayout)); err == nil || !strings.Contains(err.Error(), "invalid layout") {
		t.Fatalf("unknown layout accepted: %v", err)
	}
}
