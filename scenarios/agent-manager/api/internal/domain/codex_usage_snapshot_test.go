package domain

import (
	"encoding/json"
	"testing"
)

func TestUsagePayloadRoundTripRetainsInternalCodexBaseline(t *testing.T) {
	complete := false
	want := &UsageEventData{
		PayloadKind: PayloadKindUsage, InputTokens: 90, CacheReadTokens: 20, OutputTokens: 30,
		RunnerType: string(RunnerTypeCodex), ReconciliationAuthority: true,
		ReconciliationComplete: &complete,
		CodexUsageSnapshot:     &CodexUsageSnapshot{SessionID: "same-session", InputTokens: 110, CachedInputTokens: 20, OutputTokens: 30},
	}
	raw, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeEventPayload(EventTypeMetric, raw)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := decoded.(*UsageEventData)
	if !ok || got.CodexUsageSnapshot == nil || *got.ReconciliationComplete || got.CodexUsageSnapshot.SessionID != "same-session" || got.CodexUsageSnapshot.InputTokens != 110 || got.CodexUsageSnapshot.CachedInputTokens != 20 || got.CodexUsageSnapshot.OutputTokens != 30 {
		t.Fatalf("internal baseline did not survive event persistence codec: %#v", decoded)
	}
}
