package heartbeat

import (
	"context"
	"errors"
	"testing"
	"time"

	"prompt-manager/internal/store"
)

func newControlStoreForTest(t *testing.T, now time.Time) *HeartbeatControlStore {
	t.Helper()
	s := NewHeartbeatControlStore(t.TempDir())
	s.SetNowForTests(func() time.Time { return now })
	return s
}

func operatorAttribution() store.AttributionInfo {
	return store.AttributionInfo{
		Kind:        store.KnowledgeKindOperatorDirect,
		SpawnOrigin: store.SpawnOriginOperatorCLI,
	}
}

func agentAttribution(teamID, memberID string) store.AttributionInfo {
	return store.AttributionInfo{
		Kind:        store.KnowledgeKindAgentMember,
		TeamID:      &teamID,
		MemberID:    &memberID,
		SpawnOrigin: store.SpawnOriginHeartbeat,
	}
}

func TestHeartbeatControlStore_DefaultInitializesActiveWithCurrentEngagement(t *testing.T) {
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	s := newControlStoreForTest(t, now)

	status, err := s.Status(context.Background(), nil)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.Status != HeartbeatControlStatusActive {
		t.Fatalf("status = %q, want active", status.Status)
	}
	if status.LastHumanEngagementAt == nil || *status.LastHumanEngagementAt != now.Format(time.RFC3339) {
		t.Fatalf("last engagement = %v, want %s", status.LastHumanEngagementAt, now.Format(time.RFC3339))
	}
	if status.EffectivePolicy.PauseAfterDaysWithoutHumanEngagement != HeartbeatControlDefaultPauseDays {
		t.Fatalf("pause days = %d", status.EffectivePolicy.PauseAfterDaysWithoutHumanEngagement)
	}
}

func TestHeartbeatControlStore_EvaluatesWarningAndAutoPause(t *testing.T) {
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	s := newControlStoreForTest(t, start)
	if _, err := s.Status(context.Background(), nil); err != nil {
		t.Fatalf("prime status: %v", err)
	}

	s.SetNowForTests(func() time.Time { return start.AddDate(0, 0, HeartbeatControlDefaultWarningDays) })
	status, err := s.Status(context.Background(), nil)
	if err != nil {
		t.Fatalf("warning Status: %v", err)
	}
	if status.Status != HeartbeatControlStatusWarningIdle {
		t.Fatalf("warning status = %q", status.Status)
	}

	s.SetNowForTests(func() time.Time { return start.AddDate(0, 0, HeartbeatControlDefaultPauseDays) })
	status, err = s.Status(context.Background(), nil)
	if err != nil {
		t.Fatalf("pause Status: %v", err)
	}
	if status.Status != HeartbeatControlStatusPausedAuto {
		t.Fatalf("pause status = %q", status.Status)
	}
	if _, err := s.AllowStart(context.Background(), "team-a"); !errors.Is(err, ErrHeartbeatPaused) {
		t.Fatalf("AllowStart err = %v, want ErrHeartbeatPaused", err)
	}
}

func TestHeartbeatControlStore_OperatorEngagementResetsAutoPauseAgentDoesNot(t *testing.T) {
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	s := newControlStoreForTest(t, start)
	if _, err := s.Status(context.Background(), nil); err != nil {
		t.Fatalf("prime status: %v", err)
	}

	pausedAt := start.AddDate(0, 0, HeartbeatControlDefaultPauseDays)
	s.SetNowForTests(func() time.Time { return pausedAt })
	if _, err := s.Status(context.Background(), nil); err != nil {
		t.Fatalf("pause status: %v", err)
	}
	if err := s.RecordHumanEngagement(context.Background(), HumanEngagementEvent{
		TeamID:      "team-a",
		Reason:      "work-dispositioned",
		Attribution: agentAttribution("team-a", "agent-a"),
	}); err != nil {
		t.Fatalf("agent engagement: %v", err)
	}
	status, err := s.Status(context.Background(), nil)
	if err != nil {
		t.Fatalf("agent status: %v", err)
	}
	if status.Status != HeartbeatControlStatusPausedAuto {
		t.Fatalf("agent status = %q, want auto-paused", status.Status)
	}

	operatorAt := pausedAt.Add(time.Hour)
	s.SetNowForTests(func() time.Time { return operatorAt })
	if err := s.RecordHumanEngagement(context.Background(), HumanEngagementEvent{
		TeamID:      "team-a",
		Reason:      "work-dispositioned",
		Attribution: operatorAttribution(),
	}); err != nil {
		t.Fatalf("operator engagement: %v", err)
	}
	status, err = s.Status(context.Background(), nil)
	if err != nil {
		t.Fatalf("operator status: %v", err)
	}
	if status.Status != HeartbeatControlStatusActive {
		t.Fatalf("operator status = %q, want active", status.Status)
	}
	if status.LastHumanEngagementReason != "work-dispositioned" {
		t.Fatalf("reason = %q", status.LastHumanEngagementReason)
	}
}

func TestHeartbeatControlStore_ManualPauseAndResume(t *testing.T) {
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	s := newControlStoreForTest(t, now)

	status, err := s.Pause(context.Background(), "", "operator requested quiet period", operatorAttribution())
	if err != nil {
		t.Fatalf("Pause: %v", err)
	}
	if status.Status != HeartbeatControlStatusPausedManual {
		t.Fatalf("pause status = %q", status.Status)
	}
	if _, err := s.AllowStart(context.Background(), "team-a"); !errors.Is(err, ErrHeartbeatPaused) {
		t.Fatalf("AllowStart err = %v, want ErrHeartbeatPaused", err)
	}

	status, err = s.Resume(context.Background(), "", operatorAttribution())
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if status.Status != HeartbeatControlStatusActive {
		t.Fatalf("resume status = %q", status.Status)
	}
	if _, err := s.AllowStart(context.Background(), "team-a"); err != nil {
		t.Fatalf("AllowStart after resume: %v", err)
	}
}
