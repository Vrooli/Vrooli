package hub_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	hub "notification-hub/internal/hub"
	"notification-hub/internal/preferences"
)

// stubPreferences is a fixed-decision seam for the delivery-worker suppression
// tests. It defers the urgency comparison to the real preferences ladder.
type stubPreferences struct {
	observed  []string
	decision  hub.PreferenceDecision
	returnErr error
}

func (s *stubPreferences) ObserveCatalog(_ context.Context, source, category, _ string) error {
	s.observed = append(s.observed, source+"/"+category)
	return s.returnErr
}

func (s *stubPreferences) GetPreference(_ context.Context, _, _, _, _ string) (hub.PreferenceDecision, error) {
	if s.returnErr != nil {
		return hub.PreferenceDecision{}, s.returnErr
	}
	return s.decision, nil
}

func (s *stubPreferences) UrgencyAtLeast(have, min string) bool {
	return preferences.UrgencyAtLeast(have, min)
}

func terminalState(t *testing.T, service *hub.Service, id string) (string, string) {
	t.Helper()
	var state, reason string
	require.Eventually(t, func() bool {
		n, _, err := service.Get(context.Background(), id)
		if err != nil {
			return false
		}
		state, reason = n.State, n.Reason
		switch state {
		case "delivered", "failed", "unroutable", "suppressed":
			return true
		}
		return false
	}, 2*time.Second, 10*time.Millisecond)
	return state, reason
}

func TestProcessSuppressesMutedSourceCategory(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	registerSubscription(t, service, "alice") // so a non-suppressed note would deliver

	service.SetPreferences(&stubPreferences{decision: hub.PreferenceDecision{Enabled: false}})

	n, err := service.Send(ctx, hub.SendInput{
		RequestedBy: "alice", Body: "stretch", SensitivityLabel: "private",
		IdempotencyKey: "pref-muted-1", SourceScenario: "planner", Category: "reminder", Urgency: "normal",
	})
	require.NoError(t, err)

	state, reason := terminalState(t, service, n.ID)
	require.Equal(t, "suppressed", state)
	require.Contains(t, reason, "muted by recipient preference for planner/reminder")
}

func TestProcessSuppressesBelowMinUrgency(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	registerSubscription(t, service, "alice")

	service.SetPreferences(&stubPreferences{decision: hub.PreferenceDecision{Enabled: true, MinUrgency: "high"}})

	n, err := service.Send(ctx, hub.SendInput{
		RequestedBy: "alice", Body: "fyi", SensitivityLabel: "private",
		IdempotencyKey: "pref-lowurg-1", SourceScenario: "planner", Category: "reminder", Urgency: "normal",
	})
	require.NoError(t, err)

	state, reason := terminalState(t, service, n.ID)
	require.Equal(t, "suppressed", state)
	require.Contains(t, reason, "below recipient minimum")
}

func TestProcessDeliversWhenPreferenceAllows(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	registerSubscription(t, service, "alice")
	sender := &recordingSender{}
	service.SetPushSender(sender)

	service.SetPreferences(&stubPreferences{decision: hub.PreferenceDecision{Enabled: true, MinUrgency: "low"}})

	n, err := service.Send(ctx, hub.SendInput{
		RequestedBy: "alice", Body: "ok", SensitivityLabel: "private",
		IdempotencyKey: "pref-allow-1", SourceScenario: "planner", Category: "reminder", Urgency: "normal",
	})
	require.NoError(t, err)

	state, _ := terminalState(t, service, n.ID)
	require.Equal(t, "delivered", state)
}

func TestSendObservesCatalogAndNilPreferencesIsUnchanged(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	registerSubscription(t, service, "alice")
	service.SetPushSender(&recordingSender{})
	stub := &stubPreferences{decision: hub.PreferenceDecision{Enabled: true}}
	service.SetPreferences(stub)

	_, err := service.Send(ctx, hub.SendInput{
		RequestedBy: "alice", Body: "seen", SensitivityLabel: "private",
		IdempotencyKey: "catalog-1", SourceScenario: "planner", Category: "reminder",
	})
	require.NoError(t, err)
	require.Contains(t, stub.observed, "planner/reminder")

	// With no preference seam and no source/category, behavior is unchanged:
	// the notification routes and reaches a terminal delivery state.
	service.SetPreferences(nil)
	n, err := service.Send(ctx, hub.SendInput{
		RequestedBy: "alice", Body: "plain", SensitivityLabel: "private", IdempotencyKey: "plain-1",
	})
	require.NoError(t, err)
	state, _ := terminalState(t, service, n.ID)
	require.Equal(t, "delivered", state)
}
