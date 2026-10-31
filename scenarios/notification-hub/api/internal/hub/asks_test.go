package hub_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

func TestAskWithSpec_PersistsContextAndRefusesChangedImmutableTerms(t *testing.T) {
	service, _ := testService(t)
	now := time.Date(2026, time.October, 16, 12, 0, 0, 0, time.UTC)
	service.SetClockForTest(fixedClock{now: now})
	spec := decisionSpec(now, "evt-context")
	spec.BigPicture = "  Why it matters\r\nfor the goal.  "
	spec.TechnicalDetails = " Evidence and limits. "

	first, _, err := service.AskWithSpec(context.Background(), spec)
	require.NoError(t, err)
	require.Equal(t, "Why it matters\nfor the goal.", first.BigPicture)
	require.Equal(t, "Evidence and limits.", first.TechnicalDetails)
	require.Eventually(t, func() bool {
		notification, _, getErr := service.Get(context.Background(), first.NotificationID)
		return getErr == nil && notification.State == "unroutable"
	}, time.Second, 10*time.Millisecond, "drain the created notification before closing the test store")

	retry, _, err := service.AskWithSpec(context.Background(), spec)
	require.NoError(t, err, "same immutable terms remain idempotent")
	require.Equal(t, first.ID, retry.ID)

	changed := spec
	changed.BigPicture = "Different purpose"
	_, _, err = service.AskWithSpec(context.Background(), changed)
	require.ErrorIs(t, err, hub.ErrAskIdentityConflict)

	changed = spec
	changed.Options = append([]hub.AskOption(nil), spec.Options...)
	changed.Options[0].Label = "Changed approval terms"
	_, _, err = service.AskWithSpec(context.Background(), changed)
	require.ErrorIs(t, err, hub.ErrAskIdentityConflict)

	changed = spec
	changed.Body = "A changed notification body"
	_, _, err = service.AskWithSpec(context.Background(), changed)
	require.ErrorIs(t, err, hub.ErrAskIdentityConflict)

	changed = spec
	changed.Source = "another-producer"
	_, _, err = service.AskWithSpec(context.Background(), changed)
	require.ErrorIs(t, err, hub.ErrAskIdentityConflict, "same recipient and notification key cannot alias a different ask identity")
}

func TestAskWithSpec_ConcurrentRequesterIdentityHasOneImmutableOwner(t *testing.T) {
	now := time.Date(2026, time.October, 16, 12, 0, 0, 0, time.UTC)

	t.Run("same terms return one ask and one delivery", func(t *testing.T) {
		service, db := testService(t)
		sender := &recordingSender{}
		service.SetPushSender(sender)
		registerSubscription(t, service, "alice")
		base := decisionSpec(now, "evt-concurrent-same")
		const callers = 12
		start := make(chan struct{})
		type result struct {
			ask hub.AskRecord
			err error
		}
		results := make(chan result, callers)
		var wg sync.WaitGroup
		for i := 0; i < callers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				ask, _, err := service.AskWithSpec(context.Background(), base)
				results <- result{ask: ask, err: err}
			}()
		}
		close(start)
		wg.Wait()
		close(results)
		var askID string
		for got := range results {
			require.NoError(t, got.err)
			if askID == "" {
				askID = got.ask.ID
			}
			require.Equal(t, askID, got.ask.ID)
		}
		var asks int
		require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM asks WHERE source=? AND requester_ref=?`, base.Source, base.IdempotencyKey).Scan(&asks))
		require.Equal(t, 1, asks)
		persisted, err := service.GetAsk(context.Background(), askID)
		require.NoError(t, err)
		require.Eventually(t, func() bool {
			notification, _, getErr := service.Get(context.Background(), persisted.NotificationID)
			return getErr == nil && notification.State == "delivered"
		}, time.Second, 10*time.Millisecond)
		require.Eventually(t, func() bool {
			sender.mu.Lock()
			defer sender.mu.Unlock()
			return len(sender.messages) == 1
		}, time.Second, 10*time.Millisecond)
	})

	t.Run("different recipients cannot split one identity", func(t *testing.T) {
		service, db := testService(t)
		base := decisionSpec(now, "evt-concurrent-recipients")
		specs := []hub.AskSpec{base, base}
		specs[1].Recipient = "bob"
		const callers = 2
		start := make(chan struct{})
		type result struct {
			ask hub.AskRecord
			err error
		}
		results := make(chan result, callers)
		var wg sync.WaitGroup
		for _, spec := range specs {
			wg.Add(1)
			go func(spec hub.AskSpec) {
				defer wg.Done()
				<-start
				ask, _, err := service.AskWithSpec(context.Background(), spec)
				results <- result{ask: ask, err: err}
			}(spec)
		}
		close(start)
		wg.Wait()
		close(results)
		var winner hub.AskRecord
		var success, conflicts int
		for got := range results {
			switch {
			case got.err == nil:
				success++
				winner = got.ask
			case errors.Is(got.err, hub.ErrAskIdentityConflict):
				conflicts++
			default:
				require.NoError(t, got.err)
			}
		}
		require.Equal(t, 1, success)
		require.Equal(t, 1, conflicts)
		var asks int
		require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM asks WHERE source=? AND requester_ref=?`, base.Source, base.IdempotencyKey).Scan(&asks))
		require.Equal(t, 1, asks)
		var notifications int
		require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM notifications WHERE idempotency_key=?`, base.IdempotencyKey).Scan(&notifications))
		require.Equal(t, 1, notifications, "the rejected recipient's notification must roll back with its Ask insert")
		persisted, err := service.GetAsk(context.Background(), winner.ID)
		require.NoError(t, err)
		require.Contains(t, []string{"alice", "bob"}, persisted.Recipient)
		require.Eventually(t, func() bool {
			notification, _, getErr := service.Get(context.Background(), persisted.NotificationID)
			return getErr == nil && notification.State == "unroutable"
		}, time.Second, 10*time.Millisecond, "finish the committed winner's worker delivery before closing the test store")
	})
}

func TestAskWithSpec_RejectsInvalidOrOversizeContext(t *testing.T) {
	service, _ := testService(t)
	now := time.Now().UTC()
	for name, mutate := range map[string]func(*hub.AskSpec){
		"big picture":       func(spec *hub.AskSpec) { spec.BigPicture = strings.Repeat("😀", 4_001) },
		"technical details": func(spec *hub.AskSpec) { spec.TechnicalDetails = strings.Repeat("🧭", 12_001) },
		"invalid utf8":      func(spec *hub.AskSpec) { spec.TechnicalDetails = string([]byte{0xff}) },
	} {
		t.Run(name, func(t *testing.T) {
			spec := decisionSpec(now, "evt-bad-context-"+strings.ReplaceAll(name, " ", "-"))
			mutate(&spec)
			_, _, err := service.AskWithSpec(context.Background(), spec)
			require.ErrorIs(t, err, hub.ErrInvalidArgument)
		})
	}
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

// [REQ:NOTIFICA-P1-011]
func TestProcessEscalations_CriticalAskAdvancesAndRecordsStep(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	start := time.Date(2026, time.October, 7, 12, 0, 0, 0, time.UTC)
	service.SetClockForTest(fixedClock{now: start})
	spec := decisionSpec(start, "evt-critical-escalation")
	spec.DefaultAnswer = ""
	spec.Reversible = false
	spec.Urgency = "critical"
	spec.SensitivityLabel = "public"
	ask, _, err := service.AskWithSpec(ctx, spec)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		n, _, getErr := service.Get(ctx, ask.NotificationID)
		return getErr == nil && n.State == "unroutable"
	}, time.Second, 10*time.Millisecond)

	// Configure the fake adapter after initial routing so this call can only
	// observe a delivery caused by the escalation step.
	email := &recordingEmail{}
	service.SetEmailSender(email)
	device, err := service.UpsertDevice(ctx, "alice", hub.Device{Name: "owner laptop", MachineID: "local"})
	require.NoError(t, err)
	_, err = service.UpsertChannelAddress(ctx, "alice", hub.ChannelAddress{DeviceID: device.ID, Channel: "email", Address: "owner@example.test", ApprovedLabels: []string{"public"}})
	require.NoError(t, err)
	_, err = service.SetEscalationChain(ctx, "alice", []string{"email"})
	require.NoError(t, err)
	service.SetClockForTest(fixedClock{now: start.Add(2 * time.Hour)})
	require.NoError(t, service.ProcessEscalations(ctx))
	got, err := service.GetAsk(ctx, ask.ID)
	require.NoError(t, err)
	require.Equal(t, hub.AskEscalated, got.State)
	require.Equal(t, "critical", got.Urgency)
	require.Equal(t, []string{"owner@example.test"}, email.addresses)
	var steps int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM escalation_steps WHERE ask_id=? AND channel='email' AND outcome='delivered'`, ask.ID).Scan(&steps))
	require.Equal(t, 1, steps)
}

// [REQ:NOTIFICA-P1-011]
func TestProcessEscalations_NonCriticalAskExpiresWithoutEscalationEffects(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	start := time.Date(2026, time.October, 7, 12, 0, 0, 0, time.UTC)
	service.SetClockForTest(fixedClock{now: start})
	spec := decisionSpec(start, "evt-high-expiry")
	spec.DefaultAnswer = ""
	spec.Reversible = false
	spec.Urgency = "high"
	spec.SensitivityLabel = "public"
	ask, _, err := service.AskWithSpec(ctx, spec)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		n, _, getErr := service.Get(ctx, ask.NotificationID)
		return getErr == nil && n.State == "unroutable"
	}, time.Second, 10*time.Millisecond)

	// A configured chain and recording adapter make any accidental escalation
	// visible while keeping this package test fully isolated.
	email := &recordingEmail{}
	service.SetEmailSender(email)
	device, err := service.UpsertDevice(ctx, "alice", hub.Device{Name: "owner laptop", MachineID: "local"})
	require.NoError(t, err)
	_, err = service.UpsertChannelAddress(ctx, "alice", hub.ChannelAddress{DeviceID: device.ID, Channel: "email", Address: "owner@example.test", ApprovedLabels: []string{"public"}})
	require.NoError(t, err)
	_, err = service.SetEscalationChain(ctx, "alice", []string{"email"})
	require.NoError(t, err)
	var attemptsBefore, receiptsBefore int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM delivery_attempts WHERE notification_id=?`, ask.NotificationID).Scan(&attemptsBefore))
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM receipts WHERE notification_id=?`, ask.NotificationID).Scan(&receiptsBefore))

	service.SetClockForTest(fixedClock{now: start.Add(2 * time.Hour)})
	require.NoError(t, service.ProcessEscalations(ctx))
	got, err := service.GetAsk(ctx, ask.ID)
	require.NoError(t, err)
	require.Equal(t, hub.AskExpired, got.State)
	require.Contains(t, got.Reason, "non-critical")
	require.Empty(t, email.addresses, "non-critical expiry must not call the escalation adapter")
	var steps, attempts, receipts int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM escalation_steps WHERE ask_id=?`, ask.ID).Scan(&steps))
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM delivery_attempts WHERE notification_id=?`, ask.NotificationID).Scan(&attempts))
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM receipts WHERE notification_id=?`, ask.NotificationID).Scan(&receipts))
	require.Zero(t, steps, "direct expiry must not create an escalation step or report delivery success")
	require.Equal(t, attemptsBefore, attempts, "direct expiry must not create an escalation delivery attempt")
	require.Equal(t, receiptsBefore, receipts, "direct expiry must not create a delivery receipt")
	n, _, err := service.Get(ctx, ask.NotificationID)
	require.NoError(t, err)
	require.NotEqual(t, "delivered", n.State)
}

// [REQ:NOTIFICA-P1-010]
func TestProcess_SuppressesResolvedAskNotificationsButPreservesOpenAndDefaulted(t *testing.T) {
	ctx := context.Background()
	for _, notificationState := range []string{"pending", "held"} {
		for _, askState := range []string{hub.AskAnswered, hub.AskWithdrawn, hub.AskExpired} {
			t.Run(notificationState+"/"+askState, func(t *testing.T) {
				service, db := testService(t)
				sender := &recordingSender{}
				service.SetPushSender(sender)
				registerSubscription(t, service, "alice")
				notificationID := insertPending(t, db, "alice", "stale approval question", "private")
				now := time.Now().UTC().Format(time.RFC3339Nano)
				_, err := db.ExecContext(ctx, `INSERT INTO asks (id, notification_id, question, allowed_answers, deadline, state, created_at, updated_at, source, requester_ref) VALUES (?, ?, 'Approve?', '["yes","no"]', ?, ?, ?, ?, 'agent-manager', ?)`, "ask-"+notificationID, notificationID, time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano), askState, now, now, "request-"+notificationID)
				require.NoError(t, err)
				if notificationState == "held" {
					_, err = db.ExecContext(ctx, `INSERT INTO notification_events (notification_id, state, reason, created_at) VALUES (?, 'held', 'quiet-hours hold', ?)`, notificationID, now)
					require.NoError(t, err)
				}

				require.NoError(t, service.Process(ctx, notificationID))
				got, _, err := service.Get(ctx, notificationID)
				require.NoError(t, err)
				require.Equal(t, "suppressed", got.State)
				require.Empty(t, sender.messages, "a resolved Ask must not reach the sender after recovery or hold release")
			})
		}
	}

	for _, askState := range []string{hub.AskPending, hub.AskEscalated, hub.AskDefaulted} {
		t.Run("linked-"+askState+"-still-routes", func(t *testing.T) {
			service, db := testService(t)
			sender := &recordingSender{}
			service.SetPushSender(sender)
			registerSubscription(t, service, "alice")
			notificationID := insertPending(t, db, "alice", "current approval question", "private")
			now := time.Now().UTC().Format(time.RFC3339Nano)
			defaultAnswer, reversible, defaultAppliedAt, resolvedAt := "", 0, "", ""
			if askState == hub.AskDefaulted {
				defaultAnswer, reversible, defaultAppliedAt, resolvedAt = "yes", 1, now, now
			}
			_, err := db.ExecContext(ctx, `INSERT INTO asks (id, notification_id, question, allowed_answers, deadline, state, created_at, updated_at, source, requester_ref, default_answer, reversible, default_applied_at, resolved_at) VALUES (?, ?, 'Approve?', '["yes","no"]', ?, ?, ?, ?, 'agent-manager', ?, ?, ?, ?, ?)`, "ask-"+notificationID, notificationID, time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano), askState, now, now, "request-"+notificationID, defaultAnswer, reversible, defaultAppliedAt, resolvedAt)
			require.NoError(t, err)
			require.NoError(t, service.Process(ctx, notificationID))
			got, _, err := service.Get(ctx, notificationID)
			require.NoError(t, err)
			require.Equal(t, "delivered", got.State)
			require.Len(t, sender.messages, 1, "open and defaulted Asks retain their established routing/late-answer behavior")
		})
	}

	t.Run("ordinary-fyi-without-ask-still-routes", func(t *testing.T) {
		service, db := testService(t)
		sender := &recordingSender{}
		service.SetPushSender(sender)
		registerSubscription(t, service, "alice")
		notificationID := insertPending(t, db, "alice", "ordinary update", "public")
		require.NoError(t, service.Process(ctx, notificationID))
		got, _, err := service.Get(ctx, notificationID)
		require.NoError(t, err)
		require.Equal(t, "delivered", got.State)
		require.Len(t, sender.messages, 1)
	})
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
func TestAsk_QuietWindowHoldsCriticalAskAndItsEscalation(t *testing.T) {
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
	_, err = service.SetQuietWindow(context.Background(), "alice", hub.QuietWindow{Weekday: 1, Start: "22:00", End: "23:59", Timezone: "UTC", CriticalOverride: false})
	require.NoError(t, err)

	ask, n, err := service.AskWithSpec(context.Background(), hub.AskSpec{
		Recipient: "alice", Question: "approve?", Options: []hub.AskOption{{Key: "yes", Label: "yes"}, {Key: "no", Label: "no"}},
		Urgency: "critical", SensitivityLabel: "public", Deadline: quietNow.Add(time.Minute), IdempotencyKey: "quiet-ask",
	})
	require.NoError(t, err)
	askID := ask.ID
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

// [REQ:NOTIFICA-P1-009] An answer submitted by the recipient records the ACTUAL
// verified human (AnsweredBy) as answered_by — distinct from the recipient
// subject the authorization check uses — and that real subject rides
// ask.resolved.v1 downstream. The recipient-only check still refuses a
// non-recipient regardless of AnsweredBy.
func TestAnswerAsk_RecordsTheVerifiedHumanAsAnsweredBy(t *testing.T) {
	service, _ := testService(t)
	publisher := &recordingPublisher{}
	service.SetResolutionPublisher(publisher)
	now := time.Date(2026, time.October, 16, 12, 0, 0, 0, time.UTC)
	service.SetClockForTest(fixedClock{now: now})
	ask, _, err := service.AskWithSpec(context.Background(), decisionSpec(now, "evt-attribution"))
	require.NoError(t, err)

	// Recipient "alice" answers; the verified human behind this request is a
	// distinct per-human subject (a second household member on the shared inbox).
	answered, err := service.AnswerAsk(context.Background(), hub.AnswerInput{
		AskID: ask.ID, Answer: "close", Actor: "alice", AnsweredBy: "access:bob@example.test",
	})
	require.NoError(t, err)
	require.Equal(t, hub.AskAnswered, answered.State)
	require.Equal(t, "access:bob@example.test", answered.AnsweredBy, "answered_by must be the verified human, not the recipient subject")

	require.NoError(t, service.PublishPendingResolutions(context.Background()))
	require.Eventually(t, func() bool { return len(publisher.snapshot()) == 1 }, time.Second, 10*time.Millisecond)
	require.Equal(t, "access:bob@example.test", publisher.snapshot()[0].Payload["answered_by"], "ask.resolved.v1 must carry the real verified human")

	// A non-recipient is refused by the recipient-only check even when it presents
	// a plausible AnsweredBy — authorization is keyed on the recipient, not on
	// who claims to have answered.
	second, _, err := service.AskWithSpec(context.Background(), decisionSpec(now, "evt-attribution-2"))
	require.NoError(t, err)
	_, err = service.AnswerAsk(context.Background(), hub.AnswerInput{
		AskID: second.ID, Answer: "close", Actor: "mallory", AnsweredBy: "access:mallory@example.test",
	})
	require.ErrorIs(t, err, hub.ErrNotFound)

	// With no AnsweredBy supplied (bearer/cookie path), answered_by falls back to
	// the recipient-check actor — preserving prior behavior.
	third, _, err := service.AskWithSpec(context.Background(), decisionSpec(now, "evt-attribution-3"))
	require.NoError(t, err)
	fell, err := service.AnswerAsk(context.Background(), hub.AnswerInput{AskID: third.ID, Answer: "close", Actor: "alice"})
	require.NoError(t, err)
	require.Equal(t, "alice", fell.AnsweredBy, "answered_by falls back to the actor when no verified human is supplied")
}

// Real-volume regression: filtering must precede page boundaries, including ties.
func TestAskSearchPagesAllRetainedRecordsAndKeepsRecipientScope(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	service.SetClockForTest(fixedClock{now: now})
	ids := make([]string, 0, 235)
	for i := 0; i < 235; i++ {
		spec := decisionSpec(now, fmt.Sprintf("page-%03d", i))
		spec.Question = fmt.Sprintf("Review request %03d", i)
		if i == 0 {
			spec.Question = "Find literal 100%_done older request"
		}
		if i == 234 {
			spec.Recipient = "bob"
			spec.Question = "Foreign private request"
		}
		ask, _, err := service.AskWithSpec(ctx, spec)
		require.NoError(t, err)
		ids = append(ids, ask.ID)
		// Identical timestamps intentionally exercise the ID tiebreaker. Creating
		// rows via the real domain preserves schema/recipient and answer invariants.
		if i < 210 {
			_, err = db.ExecContext(ctx, `UPDATE asks SET state='expired', resolved_at=? WHERE id=?`, now.Format(time.RFC3339Nano), ask.ID)
			require.NoError(t, err)
		} else if i%2 == 0 && i != 234 {
			_, err = db.ExecContext(ctx, `UPDATE asks SET state='escalated' WHERE id=?`, ask.ID)
			require.NoError(t, err)
		}
	}
	seen := map[string]bool{}
	cursor := ""
	var previous string
	for {
		page, err := service.ListAsksPage(ctx, "alice", hub.AskListOptions{Limit: 50, Cursor: cursor})
		require.NoError(t, err)
		require.LessOrEqual(t, len(page.Asks), 50)
		for _, ask := range page.Asks {
			require.False(t, seen[ask.ID], "no duplicates at equal timestamps")
			require.NotEqual(t, ids[234], ask.ID)
			if previous != "" {
				require.Greater(t, previous, ask.ID)
			}
			previous = ask.ID
			seen[ask.ID] = true
		}
		cursor = page.NextCursor
		if cursor == "" {
			break
		}
	}
	require.Len(t, seen, 234, "all accessible rows remain reachable past 200")

	// Put the only match behind >200 newer rows. Search still finds it directly.
	_, err := db.ExecContext(ctx, `UPDATE asks SET created_at=? WHERE id=?`, now.Add(-time.Hour).Format(time.RFC3339Nano), ids[0])
	require.NoError(t, err)
	match, err := service.ListAsksPage(ctx, "alice", hub.AskListOptions{Limit: 10, Query: "100%_done", StateFilter: "expired"})
	require.NoError(t, err)
	require.Len(t, match.Asks, 1)
	require.Equal(t, ids[0], match.Asks[0].ID)
	require.Empty(t, match.NextCursor)
	absent, err := service.ListAsksPage(ctx, "alice", hub.AskListOptions{Query: "100XXdone"})
	require.NoError(t, err)
	require.Empty(t, absent.Asks)
	foreign, err := service.ListAsksPage(ctx, "alice", hub.AskListOptions{Query: "Foreign private"})
	require.NoError(t, err)
	require.Empty(t, foreign.Asks)
	awaiting, err := service.ListAsksPage(ctx, "alice", hub.AskListOptions{Limit: 200, StateFilter: "awaiting"})
	require.NoError(t, err)
	require.Len(t, awaiting.Asks, 24)
	pending, escalated := false, false
	for _, ask := range awaiting.Asks {
		require.True(t, ask.Open())
		pending = pending || ask.State == hub.AskPending
		escalated = escalated || ask.State == hub.AskEscalated
	}
	require.True(t, pending)
	require.True(t, escalated)

	// Insertion ahead of the cursor cannot duplicate/skip an unchanged older row.
	first, err := service.ListAsksPage(ctx, "alice", hub.AskListOptions{Limit: 20})
	require.NoError(t, err)
	require.NotEmpty(t, first.NextCursor)
	service.SetClockForTest(fixedClock{now: now.Add(time.Minute)})
	_, _, err = service.AskWithSpec(ctx, decisionSpec(now.Add(time.Minute), "new-head"))
	require.NoError(t, err)
	next, err := service.ListAsksPage(ctx, "alice", hub.AskListOptions{Limit: 20, Cursor: first.NextCursor})
	require.NoError(t, err)
	for _, a := range first.Asks {
		for _, b := range next.Asks {
			require.NotEqual(t, a.ID, b.ID)
		}
	}
	for _, opts := range []hub.AskListOptions{
		{Cursor: first.NextCursor, Query: "different"},
		{Cursor: first.NextCursor, StateFilter: "awaiting"},
		{Cursor: "broken"},
		{StateFilter: "invented"},
		{OpenOnly: true, StateFilter: "answered"},
		{Query: strings.Repeat("x", 501)},
	} {
		_, err := service.ListAsksPage(ctx, "alice", opts)
		require.ErrorIs(t, err, hub.ErrInvalidArgument)
	}
	_, err = service.ListAsksPage(ctx, "bob", hub.AskListOptions{Cursor: first.NextCursor})
	require.ErrorIs(t, err, hub.ErrInvalidArgument)
	_, err = service.ListAsksPage(ctx, "", hub.AskListOptions{})
	require.ErrorIs(t, err, hub.ErrNotFound)
}
