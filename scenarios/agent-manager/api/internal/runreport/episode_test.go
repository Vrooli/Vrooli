package runreport

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/runsignal"

	"github.com/google/uuid"
)

func TestDeriveEpisodesEmitsBoundedPatterns(t *testing.T) {
	corpus, err := os.ReadFile(filepath.Join("testdata", "episode_corpus.json"))
	if err != nil {
		t.Fatalf("read frozen episode corpus: %v", err)
	}
	var expected struct {
		ClassifierVersion string   `json:"classifierVersion"`
		Patterns          []string `json:"patterns"`
	}
	if err := json.Unmarshal(corpus, &expected); err != nil {
		t.Fatalf("decode frozen episode corpus: %v", err)
	}
	run := uuid.New()
	now := time.Now().UTC()
	call := func(n int, id string) *domain.RunEvent {
		return &domain.RunEvent{ID: uuid.New(), RunID: run, Timestamp: now.Add(time.Duration(n) * time.Minute), Data: &domain.ToolCallEventData{ToolName: "shell", ToolCallID: id, Input: map[string]interface{}{"command": "agent-manager run report"}}}
	}
	a, b, c := call(0, "a"), call(1, "b"), call(8, "c")
	facts := []runsignal.InvocationFact{{CallEventID: a.ID.String(), ToolCallID: "a", Executable: "agent-manager", CommandPath: "agent-manager run report", Ownership: "resolved", Outcome: "failure", Fingerprint: "same"}, {CallEventID: b.ID.String(), ToolCallID: "b", Executable: "agent-manager", CommandPath: "agent-manager run report", Ownership: "resolved", Outcome: "failure", HelpRecovery: true, Fingerprint: "same"}}
	usage := &domain.RunEvent{ID: uuid.New(), RunID: run, Timestamp: now.Add(30 * time.Second), Data: &domain.UsageEventData{InputTokens: 3, OutputTokens: 2}}
	episodes := runsignal.DeriveEpisodes(facts, []*domain.RunEvent{a, usage, b, c})
	seen := map[string]bool{}
	for _, e := range episodes {
		seen[e.Pattern] = true
		if e.ClassifierVersion != runsignal.EpisodeClassifierVersion || e.Fingerprint == "" || e.CauseScope == "" || len(e.HonestyFlags) != 1 {
			t.Fatalf("invalid episode: %+v", e)
		}
	}
	if runsignal.EpisodeClassifierVersion != expected.ClassifierVersion {
		t.Fatalf("classifier version = %q, corpus pins %q", runsignal.EpisodeClassifierVersion, expected.ClassifierVersion)
	}
	for _, want := range expected.Patterns {
		if !seen[want] {
			t.Errorf("missing %s: %+v", want, episodes)
		}
	}
	for _, episode := range episodes {
		if episode.Pattern != "stall" && episode.Tokens != 5 {
			t.Fatalf("%s tokens = %d, want window token cost 5", episode.Pattern, episode.Tokens)
		}
	}
}

func TestDeriveEpisodesFrozenCorpusDoesNotPromoteUnknownCatalogCommand(t *testing.T) {
	run := uuid.New()
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	first := &domain.RunEvent{ID: uuid.New(), RunID: run, Timestamp: now}
	second := &domain.RunEvent{ID: uuid.New(), RunID: run, Timestamp: now.Add(time.Second)}
	facts := []runsignal.InvocationFact{
		{CallEventID: first.ID.String(), Executable: "agent-manager", CommandPath: "agent-manager imaginary", Ownership: "unknown", Outcome: "failure", Fingerprint: "same"},
		{CallEventID: second.ID.String(), Executable: "agent-manager", CommandPath: "agent-manager imaginary", Ownership: "unknown", Outcome: "failure", Fingerprint: "same"},
	}
	for _, episode := range runsignal.DeriveEpisodes(facts, []*domain.RunEvent{first, second}) {
		if episode.SuspectedOwnerScenario != "" || episode.SuspectedOwnerCommand != "" || episode.OwnerConfidence != "unknown" {
			t.Fatalf("unknown catalog command became owned episode: %+v", episode)
		}
	}
}

func TestUpgradeEpisodeOwnershipUsesReceiptTimeWindow(t *testing.T) {
	run := uuid.New()
	start := &domain.RunEvent{ID: uuid.New(), RunID: run, Timestamp: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)}
	end := &domain.RunEvent{ID: uuid.New(), RunID: run, Timestamp: start.Timestamp.Add(time.Minute)}
	episodes := []runsignal.FrictionEpisode{{StartEventID: start.ID.String(), EndEventID: end.ID.String(), SuspectedOwnerScenario: "catalog-owner", OwnerConfidence: "manifest-derived"}}
	calls := []CrossScenarioCall{
		{TargetScenario: "catalog-owner", ReceiptEventID: "in-window", OccurredAt: start.Timestamp.Add(30 * time.Second), Outcome: "failure"},
		{TargetScenario: "different-owner", ReceiptEventID: "outside-window", OccurredAt: end.Timestamp.Add(time.Minute), Outcome: "failure"},
	}
	got := UpgradeEpisodeOwnership(episodes, []*domain.RunEvent{start, end}, calls, Availability{State: AvailabilityAvailable})
	if got[0].OwnerConfidence != "receipt-verified" || got[0].FailedJoinedCalls != 1 || len(got[0].EvidenceEventIDs) != 1 || got[0].EvidenceEventIDs[0] != "in-window" {
		t.Fatalf("ownership must use only receipts inside the episode window: %+v", got[0])
	}
}

func TestUpgradeEpisodeOwnershipMarksConflictingReceiptTarget(t *testing.T) {
	run := uuid.New()
	start := &domain.RunEvent{ID: uuid.New(), RunID: run, Timestamp: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)}
	end := &domain.RunEvent{ID: uuid.New(), RunID: run, Timestamp: start.Timestamp.Add(time.Minute)}
	episodes := []runsignal.FrictionEpisode{{StartEventID: start.ID.String(), EndEventID: end.ID.String(), SuspectedOwnerScenario: "manifest-owner", OwnerConfidence: "manifest-derived"}}
	got := UpgradeEpisodeOwnership(episodes, []*domain.RunEvent{start, end}, []CrossScenarioCall{{TargetScenario: "receipt-owner", ReceiptEventID: "receipt-conflict", OccurredAt: start.Timestamp.Add(time.Second)}}, Availability{State: AvailabilityAvailable})
	if got[0].OwnerConfidence != "conflicting" || got[0].SuspectedOwnerScenario != "manifest-owner" || len(got[0].EvidenceEventIDs) != 1 || got[0].EvidenceEventIDs[0] != "receipt-conflict" {
		t.Fatalf("conflicting receipt must retain manifest owner and record receipt evidence: %+v", got[0])
	}
}
