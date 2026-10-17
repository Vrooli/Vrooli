package integrations

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsNotificationRequestSuffixMatching(t *testing.T) {
	cases := map[string]bool{
		"personal_planner.reminder.notification.v1": true,
		"chat.message.notification.v1":              true,
		"  spaced.notification.v1  ":                true,
		"incident.opened.v1":                        false,
		"agent.effort.decision_requested.v1":        false,
		"something.notification.v2":                 false,
		"notification.v1.other":                     false,
	}
	for eventType, want := range cases {
		require.Equalf(t, want, IsNotificationRequest(eventType), "IsNotificationRequest(%q)", eventType)
	}
}

func TestParseNotificationFactsRequiresBody(t *testing.T) {
	_, ok := parseNotificationFacts(json.RawMessage(`{"title":"hi"}`))
	require.False(t, ok, "no body => not usable neutral facts")

	_, ok = parseNotificationFacts(nil)
	require.False(t, ok)

	// An incident-shaped payload (message, not body) must NOT parse as neutral
	// facts, so the legacy incident renderer stays the fallback.
	_, ok = parseNotificationFacts(json.RawMessage(`{"incident_id":"inc-1","message":"driver drift","severity":"critical"}`))
	require.False(t, ok)

	facts, ok := parseNotificationFacts(json.RawMessage(`{"body":"something happened"}`))
	require.True(t, ok)
	require.Equal(t, "something happened", facts.Body)
}

func TestNotificationSendInputDefaultsAndThreading(t *testing.T) {
	facts, ok := parseNotificationFacts(json.RawMessage(`{"body":"remember to stretch","category":"reminder","context_url":"/planner/today","correlation":{"task":"t-1"}}`))
	require.True(t, ok)

	in := notificationSendInput("alice", "planner", "planner.reminder.notification.v1", "evt-1", facts)
	require.Equal(t, "alice", in.RequestedBy)
	require.Equal(t, "remember to stretch", in.Body)
	require.Equal(t, "normal", in.Urgency, "urgency defaults to normal")
	require.Equal(t, "private", in.SensitivityLabel, "sensitivity defaults to private")
	require.Equal(t, "evt-1", in.IdempotencyKey)
	require.Equal(t, "planner.reminder.notification.v1", in.DedupeKey, "dedupe_key defaults to the event type, never an incident id")
	require.Equal(t, "planner", in.SourceScenario)
	require.Equal(t, "reminder", in.Category)
	require.Equal(t, "/planner/today", in.ContextURL)
}

func TestNotificationSendInputHonorsDeclaredFields(t *testing.T) {
	facts, ok := parseNotificationFacts(json.RawMessage(`{"body":"disk is 95% full","urgency":"high","sensitivity":"sensitive","dedupe_key":"disk-alert"}`))
	require.True(t, ok)
	in := notificationSendInput("alice", "storage", "storage.alert.notification.v1", "evt-2", facts)
	require.Equal(t, "high", in.Urgency)
	require.Equal(t, "sensitive", in.SensitivityLabel)
	require.Equal(t, "disk-alert", in.DedupeKey, "producer-supplied dedupe key wins")
}

func TestNotificationSendInputClampsInvalidSensitivity(t *testing.T) {
	facts, ok := parseNotificationFacts(json.RawMessage(`{"body":"hi","sensitivity":"ultra-top-secret"}`))
	require.True(t, ok)
	in := notificationSendInput("alice", "planner", "planner.reminder.notification.v1", "evt-3", facts)
	require.Equal(t, "private", in.SensitivityLabel, "an invalid producer sensitivity clamps to private")
}

// TestIncidentFallbackParity proves the legacy incident renderer is unchanged:
// the neutral path does not steal incident events, which carry message (not
// body) facts, and the rendered copy is identical to before.
func TestIncidentFallbackParity(t *testing.T) {
	rendered := RenderEvent("incident.opened.v1", json.RawMessage(`{"check_id":"host-kernel-module-drift","severity":"critical","message":"driver drift"}`))
	require.Equal(t, "Incident opened: host-kernel-module-drift", rendered.Title)
	require.Equal(t, "driver drift", rendered.Body)
	require.Equal(t, "critical", rendered.SensitivityLabel)

	// The incident payload must not parse as neutral facts.
	_, ok := parseNotificationFacts(json.RawMessage(`{"check_id":"host-kernel-module-drift","severity":"critical","message":"driver drift"}`))
	require.False(t, ok)
}
