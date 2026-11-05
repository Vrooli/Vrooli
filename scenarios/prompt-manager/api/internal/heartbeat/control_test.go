package heartbeat

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/identity"
	heartbeatconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/heartbeat/heartbeat_v1connect"
	"prompt-manager/internal/requestpolicy"
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
	data, err := os.ReadFile(s.path)
	if err != nil {
		t.Fatalf("read persisted heartbeat control: %v", err)
	}
	if len(data) == 0 || data[len(data)-1] != '\n' {
		t.Fatal("persisted heartbeat control must retain its trailing newline")
	}
	info, err := os.Stat(s.path)
	if err != nil {
		t.Fatalf("stat persisted heartbeat control: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Fatalf("persisted heartbeat control mode = %04o, want 0644", got)
	}
	restarted := &HeartbeatControlStore{path: s.path, now: s.now}
	reloaded, err := restarted.TeamStatus(context.Background(), "team-a")
	if err != nil || reloaded.Status != HeartbeatControlStatusPausedManual {
		t.Fatalf("reloaded paused control state = %+v, err=%v", reloaded, err)
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

func TestTeamResumeReleasesOnlyItsCurrentGlobalManualHold(t *testing.T) {
	now := time.Date(2026, 10, 24, 13, 0, 0, 0, time.UTC)
	s := newControlStoreForTest(t, now)
	if _, err := s.Pause(context.Background(), "", "hold all", operatorAttribution()); err != nil {
		t.Fatal(err)
	}
	before, _ := s.Status(context.Background(), nil)
	s.SetNowForTests(func() time.Time { return now.Add(time.Nanosecond) })
	if _, err := s.Resume(context.Background(), "team-a", agentAttribution("team-a", "agent")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AllowStart(context.Background(), "team-a"); err != nil {
		t.Fatalf("approved selected-team resume still globally held: %v", err)
	}
	if _, err := s.AllowStart(context.Background(), "team-b"); !errors.Is(err, ErrHeartbeatPaused) {
		t.Fatal("another team resumed", err)
	}
	after, _ := s.Status(context.Background(), nil)
	if after.Status != HeartbeatControlStatusPausedManual || *after.LastHumanEngagementAt != *before.LastHumanEngagementAt {
		t.Fatal("delegated resume cleared global pause or manufactured human engagement")
	}
	reloaded := &HeartbeatControlStore{path: s.path, now: s.now}
	if _, err := reloaded.AllowStart(context.Background(), "team-a"); err != nil {
		t.Fatal("selected resume did not survive restart", err)
	}
	s.SetNowForTests(func() time.Time { return now.Add(2 * time.Nanosecond) })
	if _, err := s.Pause(context.Background(), "", "new global hold", operatorAttribution()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AllowStart(context.Background(), "team-a"); !errors.Is(err, ErrHeartbeatPaused) {
		t.Fatal("older team resume defeated a later global pause", err)
	}
}

func TestApprovedTeamControlUsesCanonicalGrantWithoutHumanAttribution(t *testing.T) {
	for _, tc := range []struct {
		name, operation, target string
		expired, allowed        bool
	}{
		{"approved", "team.manage", "prompt-manager:team/team-1", false, true},
		{"another team", "team.manage", "prompt-manager:team/other", false, false},
		{"preparation only", "team.prepare", "prompt-manager:team/team-1", false, false},
		{"expired", "team.manage", "prompt-manager:team/team-1", true, false},
		{"unrelated operation", "run.create", "prompt-manager:team/team-1", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			teams, _, _ := setupAdmissionDeferredFailureEnvironment(t)
			h := NewHandlers(HandlersDeps{TeamStore: teams})
			s := newControlStoreForTest(t, time.Date(2026, 10, 24, 13, 0, 0, 0, time.UTC))
			h.SetControlStore(s)
			expiry := time.Now().Add(time.Hour)
			if tc.expired {
				expiry = time.Now().Add(-time.Minute)
			}
			p := identity.Principal{Kind: identity.ActorAgent, Subject: "real-agent", Owner: "owner", Verified: true, Scopes: []string{"prompt-manager:write"}, Grant: &identity.OperationGrant{ID: "approved", Approver: "owner", Operations: []string{tc.operation}, Targets: []string{tc.target}, ExpiresAt: expiry}}
			req := httptest.NewRequest(http.MethodPost, heartbeatconnect.HeartbeatServiceResumeTeamHeartbeatControlProcedure, nil)
			req.Header.Set(attributionHeaderName, encodeAttribution(t, operatorAttribution()))
			req = req.WithContext(identity.WithPrincipal(req.Context(), p))
			req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
			rec := httptest.NewRecorder()
			requestpolicy.PreparationBoundary(http.HandlerFunc(h.ResumeTeamHeartbeatControl)).ServeHTTP(rec, req)
			if tc.allowed && rec.Code != http.StatusOK {
				t.Fatalf("approved delegated resume needs a human command: %d %s", rec.Code, rec.Body.String())
			}
			if !tc.allowed && rec.Code != http.StatusForbidden {
				t.Fatalf("uncovered team control admitted: %d %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestTeamResumeCannotClearGlobalAutomaticHoldOrForgeGlobalAuthority(t *testing.T) {
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	s := newControlStoreForTest(t, now)
	if _, err := s.Status(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	s.SetNowForTests(func() time.Time { return now.AddDate(0, 0, HeartbeatControlDefaultPauseDays+1) })
	if _, err := s.Resume(context.Background(), "team-a", agentAttribution("team-a", "agent")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AllowStart(context.Background(), "team-a"); !errors.Is(err, ErrHeartbeatPaused) {
		t.Fatal("agent resume cleared automatic inactivity hold", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/global/resume", nil)
	req.Header.Set(attributionHeaderName, encodeAttribution(t, operatorAttribution()))
	p := identity.Principal{Kind: identity.ActorAgent, Subject: "agent", Verified: true, Scopes: []string{"prompt-manager:write"}, Grant: &identity.OperationGrant{Approver: "owner", Operations: []string{"team.manage"}, Targets: []string{"prompt-manager:*"}, ExpiresAt: time.Now().Add(time.Hour)}}
	req = req.WithContext(identity.WithPrincipal(req.Context(), p))
	if _, err := (&Handlers{}).heartbeatControlActor(req, ""); err == nil {
		t.Fatal("team grant and forged attribution acquired global control")
	}
}

func TestTeamControlRejectsForgedHumanAttributionWithoutVerifiedAuthority(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/team/resume", nil)
	req.Header.Set(attributionHeaderName, encodeAttribution(t, operatorAttribution()))
	if _, err := (&Handlers{}).heartbeatControlActor(req, "team-a"); err == nil {
		t.Fatal("an attribution header supplied missing team-control authority")
	}
}

func TestVerifiedHumanTeamControlRetainsOwnerPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/team/resume", nil)
	req.Header.Set(attributionHeaderName, encodeAttribution(t, operatorAttribution()))
	p := identity.Principal{Kind: identity.ActorHuman, Subject: "owner", Verified: true, Scopes: []string{"prompt-manager:write"}}
	req = req.WithContext(identity.WithPrincipal(req.Context(), p))
	attr, err := (&Handlers{}).heartbeatControlActor(req, "team-a")
	if err != nil || attr.Kind != store.KnowledgeKindOperatorDirect {
		t.Fatalf("verified owner team control was lost: %+v %v", attr, err)
	}
}

func TestHeartbeatControlStore_ManualPauseRequiresReason(t *testing.T) {
	s := newControlStoreForTest(t, time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC))
	if _, err := s.Pause(context.Background(), "team-a", "  ", operatorAttribution()); err == nil || !strings.Contains(err.Error(), "reason is required") {
		t.Fatalf("Pause error = %v, want required reason", err)
	}
	status, err := s.TeamStatus(context.Background(), "team-a")
	if err != nil {
		t.Fatal(err)
	}
	if status.Status == HeartbeatControlStatusPausedManual {
		t.Fatalf("blank-reason pause mutated status to %q", status.Status)
	}
}
