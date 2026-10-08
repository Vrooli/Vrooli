package hub_test

import (
	"context"
	"sync"
	"testing"
	"time"

	hub "notification-hub/internal/hub"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/eventbus"
)

type recordingPublisher struct {
	mu     sync.Mutex
	events []eventbus.DomainEvent
}

func (p *recordingPublisher) PublishDomainEvent(_ context.Context, event eventbus.DomainEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, event)
	return nil
}

func (p *recordingPublisher) snapshot() []eventbus.DomainEvent {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]eventbus.DomainEvent(nil), p.events...)
}

func decisionSpec(now time.Time, key string) hub.AskSpec {
	return hub.AskSpec{
		Recipient:            "alice",
		Question:             "Close the BAS goal or re-aim at quality?",
		Options:              []hub.AskOption{{Key: "close", Label: "Close the goal"}, {Key: "re-aim", Label: "Re-aim at quality"}},
		Recommended:          "re-aim",
		RecommendationReason: "journeys are green and quality gaps remain",
		DefaultAnswer:        "re-aim",
		Reversible:           true,
		Deadline:             now.Add(time.Hour),
		SensitivityLabel:     "private",
		IdempotencyKey:       key,
		Source:               "agent-manager",
		SourceEventType:      "agent_manager.effort.decision_requested.v1",
		Correlation:          map[string]string{"decision_id": "BAS-D-065"},
	}
}

// [REQ:NOTIFICA-P1-009][REQ:NOTIFICA-P1-010]
func TestAskWithSpec_PersistsStructuredDecisionAtHighUrgency(t *testing.T) {
	service, _ := testService(t)
	now := time.Date(2026, time.October, 7, 12, 0, 0, 0, time.UTC)
	service.SetClockForTest(fixedClock{now: now})

	ask, n, err := service.AskWithSpec(context.Background(), decisionSpec(now, "evt-1"))
	require.NoError(t, err)
	require.Equal(t, "high", n.Urgency, "asks default to high, not critical, so quiet hours hold them")
	require.Contains(t, n.Body, "Recommended: Re-aim at quality")

	got, err := service.GetAsk(context.Background(), ask.ID)
	require.NoError(t, err)
	require.Equal(t, hub.AskPending, got.State)
	require.Equal(t, []hub.AskOption{{Key: "close", Label: "Close the goal"}, {Key: "re-aim", Label: "Re-aim at quality"}}, got.Options)
	require.Equal(t, "re-aim", got.Recommended)
	require.Equal(t, "re-aim", got.DefaultAnswer)
	require.True(t, got.Reversible)
	require.Equal(t, "evt-1", got.RequesterRef)
	require.Equal(t, map[string]string{"decision_id": "BAS-D-065"}, got.Correlation)
}

// [REQ:NOTIFICA-P1-010]
func TestAskWithSpec_RejectsUnsafeOrInconsistentDecisions(t *testing.T) {
	service, _ := testService(t)
	now := time.Now().UTC()
	notReversible := decisionSpec(now, "evt-a")
	notReversible.Reversible = false
	unknownRecommendation := decisionSpec(now, "evt-b")
	unknownRecommendation.Recommended = "delete"
	badUrgency := decisionSpec(now, "evt-c")
	badUrgency.Urgency = "urgent"
	for name, spec := range map[string]hub.AskSpec{"default without reversible": notReversible, "unknown recommendation": unknownRecommendation, "bad urgency": badUrgency} {
		_, _, err := service.AskWithSpec(context.Background(), spec)
		require.ErrorIs(t, err, hub.ErrInvalidArgument, name)
	}
}

// [REQ:NOTIFICA-P1-010]
func TestAskWithSpec_RepeatedRequestReturnsOneAskAndOneDelivery(t *testing.T) {
	service, db := testService(t)
	sender := &recordingSender{}
	service.SetPushSender(sender)
	registerSubscription(t, service, "alice")
	now := time.Now().UTC()

	first, _, err := service.AskWithSpec(context.Background(), decisionSpec(now, "evt-repeat"))
	require.NoError(t, err)
	second, _, err := service.AskWithSpec(context.Background(), decisionSpec(now, "evt-repeat"))
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)

	require.Eventually(t, func() bool {
		n, _, getErr := service.Get(context.Background(), first.NotificationID)
		return getErr == nil && n.State == "delivered"
	}, time.Second, 10*time.Millisecond)
	var asks int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM asks`).Scan(&asks))
	require.Equal(t, 1, asks)
	sender.mu.Lock()
	defer sender.mu.Unlock()
	require.Len(t, sender.messages, 1)
	require.Equal(t, "asks/"+first.ID, sender.messages[0].URL, "the push deep-links to the ask page")
	require.Equal(t, first.NotificationID, sender.messages[0].Tag)
}

// [REQ:NOTIFICA-P1-010]
// Regression: a read past the deadline used to expire a pending ask, so the
// autoheal approval check (Wait with deadline=now) killed the approval.
func TestGetAskAndWait_NeverExpireAnOpenAsk(t *testing.T) {
	service, _ := testService(t)
	now := time.Date(2026, time.October, 7, 12, 0, 0, 0, time.UTC)
	service.SetClockForTest(fixedClock{now: now})
	askID, _, err := service.Ask(context.Background(), "alice", "approve remediation?", []string{"approve", "reject"}, now.Add(time.Minute), "public", "approval-1")
	require.NoError(t, err)

	service.SetClockForTest(fixedClock{now: now.Add(2 * time.Minute)})
	state, answer, reason, err := service.Wait(context.Background(), askID, now.Add(2*time.Minute))
	require.NoError(t, err)
	require.Equal(t, hub.AskPending, state)
	require.Empty(t, answer)
	require.Contains(t, reason, "still open")
	got, err := service.GetAsk(context.Background(), askID)
	require.NoError(t, err)
	require.Equal(t, hub.AskPending, got.State)

	require.NoError(t, service.Answer(context.Background(), askID, "approve", "alice"))
	state, answer, _, err = service.Wait(context.Background(), askID, now.Add(2*time.Minute))
	require.NoError(t, err)
	require.Equal(t, hub.AskAnswered, state)
	require.Equal(t, "approve", answer)
}

// [REQ:NOTIFICA-P1-011]
func TestProcessEscalations_WithholdsDefaultUntilDeliveredThenFloorPasses(t *testing.T) {
	service, _ := testService(t)
	publisher := &recordingPublisher{}
	service.SetResolutionPublisher(publisher)
	start := time.Date(2026, time.October, 7, 12, 0, 0, 0, time.UTC)
	service.SetClockForTest(fixedClock{now: start})
	ask, _, err := service.AskWithSpec(context.Background(), decisionSpec(start, "evt-default"))
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		n, _, getErr := service.Get(context.Background(), ask.NotificationID)
		return getErr == nil && n.State == "unroutable"
	}, time.Second, 10*time.Millisecond)

	// Deadline passed, but nothing was delivered: the default is withheld and
	// the ask neither escalates nor expires.
	service.SetClockForTest(fixedClock{now: start.Add(48 * time.Hour)})
	require.NoError(t, service.ProcessEscalations(context.Background()))
	got, err := service.GetAsk(context.Background(), ask.ID)
	require.NoError(t, err)
	require.Equal(t, hub.AskPending, got.State)
	require.Contains(t, got.Reason, "operator not reached")
	require.Empty(t, publisher.snapshot())
}

// [REQ:NOTIFICA-P1-011]
func TestProcessEscalations_AppliesDefaultAtLeastTheFloorAfterDelivery(t *testing.T) {
	service, _ := testService(t)
	publisher := &recordingPublisher{}
	service.SetResolutionPublisher(publisher)
	service.SetPushSender(&recordingSender{})
	registerSubscription(t, service, "alice")
	start := time.Date(2026, time.October, 7, 12, 0, 0, 0, time.UTC)
	service.SetClockForTest(fixedClock{now: start})
	ask, _, err := service.AskWithSpec(context.Background(), decisionSpec(start, "evt-floor"))
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		got, getErr := service.GetAsk(context.Background(), ask.ID)
		return getErr == nil && got.FirstDeliveredAt != ""
	}, time.Second, 10*time.Millisecond)

	// Past the 1 h deadline but inside the 12 h floor after delivery.
	service.SetClockForTest(fixedClock{now: start.Add(2 * time.Hour)})
	require.NoError(t, service.ProcessEscalations(context.Background()))
	got, err := service.GetAsk(context.Background(), ask.ID)
	require.NoError(t, err)
	require.Equal(t, hub.AskPending, got.State)
	require.Contains(t, got.Reason, "applies at")

	service.SetClockForTest(fixedClock{now: start.Add(12*time.Hour + time.Minute)})
	require.NoError(t, service.ProcessEscalations(context.Background()))
	got, err = service.GetAsk(context.Background(), ask.ID)
	require.NoError(t, err)
	require.Equal(t, hub.AskDefaulted, got.State)
	require.Equal(t, "re-aim", got.Answer)
	require.Equal(t, hub.DefaultAnsweredBy, got.AnsweredBy)

	require.NoError(t, service.PublishPendingResolutions(context.Background()))
	require.Eventually(t, func() bool { return len(publisher.snapshot()) == 1 }, time.Second, 10*time.Millisecond)
	event := publisher.snapshot()[0]
	require.Equal(t, hub.AskResolvedEventType, event.EventType)
	require.Equal(t, true, event.Payload["by_default"])
	require.Equal(t, "re-aim", event.Payload["answer"])
	require.Equal(t, "evt-floor", event.Payload["requester_ref"])
}

// [REQ:NOTIFICA-P1-009][REQ:NOTIFICA-P1-010]
func TestAnswerAsk_PublishesResolutionOnceAndLateAnswerSupersedesDefault(t *testing.T) {
	service, db := testService(t)
	publisher := &recordingPublisher{}
	service.SetResolutionPublisher(publisher)
	now := time.Date(2026, time.October, 7, 12, 0, 0, 0, time.UTC)
	service.SetClockForTest(fixedClock{now: now})
	ask, _, err := service.AskWithSpec(context.Background(), decisionSpec(now, "evt-answer"))
	require.NoError(t, err)

	answered, err := service.AnswerAsk(context.Background(), hub.AnswerInput{AskID: ask.ID, Answer: "close", Actor: "alice", Note: "keep macOS last"})
	require.NoError(t, err)
	require.Equal(t, hub.AskAnswered, answered.State)
	require.Equal(t, "keep macOS last", answered.Note)
	require.NoError(t, service.PublishPendingResolutions(context.Background()))
	require.NoError(t, service.PublishPendingResolutions(context.Background()))
	require.Eventually(t, func() bool { return len(publisher.snapshot()) == 1 }, time.Second, 10*time.Millisecond)
	payload := publisher.snapshot()[0].Payload
	require.Equal(t, "close", payload["answer"])
	require.Equal(t, "Close the goal", payload["answer_label"])
	require.Equal(t, "keep macOS last", payload["note"])
	require.Equal(t, map[string]any{"decision_id": "BAS-D-065"}, payload["correlation"])
	_, err = service.AnswerAsk(context.Background(), hub.AnswerInput{AskID: ask.ID, Answer: "re-aim", Actor: "alice"})
	require.ErrorIs(t, err, hub.ErrAskResolved)

	// A defaulted ask accepts exactly one late answer, which supersedes it.
	second, _, err := service.AskWithSpec(context.Background(), decisionSpec(now, "evt-late"))
	require.NoError(t, err)
	_, err = db.ExecContext(context.Background(), `UPDATE asks SET state='defaulted', default_applied_at=?, resolved_at=? WHERE id=?`, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), second.ID)
	require.NoError(t, err)
	late, err := service.AnswerAsk(context.Background(), hub.AnswerInput{AskID: second.ID, Answer: "close", Actor: "alice"})
	require.NoError(t, err)
	require.Equal(t, hub.AskAnswered, late.State)
	require.True(t, late.Late)
	_, err = service.AnswerAsk(context.Background(), hub.AnswerInput{AskID: second.ID, Answer: "re-aim", Actor: "alice"})
	require.ErrorIs(t, err, hub.ErrAskResolved)
}

// [REQ:NOTIFICA-P0-005][REQ:NOTIFICA-P1-011]
func TestAsk_QuietWindowHoldsHighUrgencyAskAndItsEscalation(t *testing.T) {
	service, db := testService(t)
	email := &recordingEmail{}
	service.SetEmailSender(email)
	quietNow := time.Date(2026, time.March, 2, 22, 30, 0, 0, time.UTC) // Monday
	service.SetClockForTest(fixedClock{now: quietNow})
	device, err := service.UpsertDevice(context.Background(), "alice", hub.Device{Name: "owner laptop", MachineID: "local"})
	require.NoError(t, err)
	_, err = service.UpsertChannelAddress(context.Background(), "alice", hub.ChannelAddress{DeviceID: device.ID, Channel: "email", Address: "owner@example.test", ApprovedLabels: []string{"public"}})
	require.NoError(t, err)
	_, err = service.SetEscalationChain(context.Background(), "alice", []string{"email"})
	require.NoError(t, err)
	_, err = service.SetQuietWindow(context.Background(), "alice", hub.QuietWindow{Weekday: 1, Start: "22:00", End: "23:59", Timezone: "UTC", CriticalOverride: true})
	require.NoError(t, err)

	askID, n, err := service.Ask(context.Background(), "alice", "approve?", []string{"yes", "no"}, quietNow.Add(time.Minute), "public", "quiet-ask")
	require.NoError(t, err)
	require.NoError(t, service.Process(context.Background(), n.ID))
	held, _, err := service.Get(context.Background(), n.ID)
	require.NoError(t, err)
	require.Equal(t, "held", held.State)

	service.SetClockForTest(fixedClock{now: quietNow.Add(2 * time.Minute)})
	require.NoError(t, service.ProcessEscalations(context.Background()))
	var state string
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT state FROM asks WHERE id=?`, askID).Scan(&state))
	require.Equal(t, hub.AskPending, state, "escalation waits out the quiet window")
	require.Empty(t, email.bodies)
}

// [REQ:NOTIFICA-P1-009]
func TestAsks_OnlyTheRecipientSeesAndAnswersThem(t *testing.T) {
	service, _ := testService(t)
	now := time.Now().UTC()
	ask, _, err := service.AskWithSpec(context.Background(), decisionSpec(now, "evt-owner"))
	require.NoError(t, err)

	_, err = service.AnswerAsk(context.Background(), hub.AnswerInput{AskID: ask.ID, Answer: "close", Actor: "mallory"})
	require.ErrorIs(t, err, hub.ErrNotFound)
	open, err := service.ListAsks(context.Background(), "mallory", true, 10)
	require.NoError(t, err)
	require.Empty(t, open)
	open, err = service.ListAsks(context.Background(), "alice", true, 10)
	require.NoError(t, err)
	require.Len(t, open, 1)
	require.Equal(t, ask.ID, open[0].ID)
}
