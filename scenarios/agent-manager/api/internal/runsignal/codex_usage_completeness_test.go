package runsignal

import (
	"testing"

	"agent-manager/internal/domain"
)

func TestSelectCanonicalUsageKeepsMixedLegacyAndSeededCodexHistoryUnknown(t *testing.T) {
	legacyIncomplete := false
	seededComplete := true
	events := []*domain.RunEvent{
		{Data: &domain.StatusEventData{NewStatus: string(domain.RunStatusRunning)}},
		{Data: &domain.UsageEventData{InputTokens: 100, ReconciliationAuthority: true, ReconciliationComplete: &legacyIncomplete}},
		{Data: &domain.StatusEventData{NewStatus: string(domain.RunStatusRunning)}},
		{Data: &domain.UsageEventData{InputTokens: 25, ReconciliationAuthority: true, ReconciliationComplete: &seededComplete}},
	}
	selection := SelectCanonicalUsage(events)
	if selection.AllInvocationsAuthoritative {
		t.Fatal("a later seeded continuation must not certify an earlier unqualified interval")
	}
	if !selection.LatestInvocationAuthoritative {
		t.Fatal("the latest seeded invocation should retain its own complete status")
	}
	if !selection.Selected[1] || !selection.Selected[3] {
		t.Fatalf("completeness changed quantity selection: selected=%v", selection.Selected)
	}

	// A later complete turn cannot hide an earlier unqualified turn within
	// the same invocation either.
	sameInvocation := []*domain.RunEvent{
		{Data: &domain.StatusEventData{NewStatus: string(domain.RunStatusRunning)}},
		{Data: &domain.UsageEventData{InputTokens: 100, RunnerType: string(domain.RunnerTypeCodex), ReconciliationAuthority: true, ReconciliationComplete: &legacyIncomplete}},
		{Data: &domain.UsageEventData{InputTokens: 25, RunnerType: string(domain.RunnerTypeCodex), ReconciliationAuthority: true, ReconciliationComplete: &seededComplete}},
	}
	sameSelection := SelectCanonicalUsage(sameInvocation)
	if sameSelection.AllInvocationsAuthoritative || sameSelection.LatestInvocationAuthoritative {
		t.Fatalf("later complete receipt hid an earlier unqualified interval: %+v", sameSelection)
	}
}

func TestSelectCanonicalUsagePreservesLegacyAndExplicitCompleteReceipts(t *testing.T) {
	events := []*domain.RunEvent{{Data: &domain.StatusEventData{NewStatus: string(domain.RunStatusRunning)}}, {Data: &domain.UsageEventData{InputTokens: 10, ReconciliationAuthority: true}}}
	if got := SelectCanonicalUsage(events); !got.AllInvocationsAuthoritative || !got.LatestInvocationAuthoritative {
		t.Fatalf("legacy non-Codex authority behavior changed: %+v", got)
	}
	complete := true
	events[1].Data = &domain.UsageEventData{InputTokens: 10, ReconciliationAuthority: true, ReconciliationComplete: &complete}
	if got := SelectCanonicalUsage(events); !got.AllInvocationsAuthoritative || !got.LatestInvocationAuthoritative {
		t.Fatalf("explicit complete receipt not recognized: %+v", got)
	}
	legacyCodex := []*domain.RunEvent{
		{Data: &domain.StatusEventData{NewStatus: string(domain.RunStatusRunning)}},
		{Data: &domain.UsageEventData{InputTokens: 10, RunnerType: string(domain.RunnerTypeCodex), ReconciliationAuthority: true}},
	}
	if got := SelectCanonicalUsage(legacyCodex); got.AllInvocationsAuthoritative || got.LatestInvocationAuthoritative {
		t.Fatalf("legacy Codex receipt without baseline provenance was treated as complete: %+v", got)
	}
	interimThenComplete := []*domain.RunEvent{
		{Data: &domain.StatusEventData{NewStatus: string(domain.RunStatusRunning)}},
		{Data: &domain.UsageEventData{InputTokens: 10, RunnerType: string(domain.RunnerTypeCodex)}},
		{Data: &domain.UsageEventData{InputTokens: 12, RunnerType: string(domain.RunnerTypeCodex), ReconciliationAuthority: true, ReconciliationComplete: &complete}},
	}
	if got := SelectCanonicalUsage(interimThenComplete); !got.AllInvocationsAuthoritative || !got.LatestInvocationAuthoritative {
		t.Fatalf("interim Codex usage without receipt metadata blocked complete terminal receipt: %+v", got)
	}
}
