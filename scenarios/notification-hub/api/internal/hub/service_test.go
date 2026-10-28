package hub_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	hub "notification-hub/internal/hub"
	"notification-hub/internal/modules"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
)

type recordingSender struct {
	mu       sync.Mutex
	attempts int
	contents []string
	messages []hub.PushMessage
	errors   []error
}

func (s *recordingSender) Send(_ context.Context, _ hub.PushSubscription, message hub.PushMessage) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attempts++
	s.contents = append(s.contents, message.Body)
	s.messages = append(s.messages, message)
	index := s.attempts - 1
	if index < len(s.errors) && s.errors[index] != nil {
		return "", s.errors[index]
	}
	return fmt.Sprintf("provider-%d", s.attempts), nil
}

type goneSender struct{}

func (goneSender) Send(context.Context, hub.PushSubscription, hub.PushMessage) (string, error) {
	return "", goneTestError{}
}

type goneTestError struct{}

func (goneTestError) Error() string { return "subscription retired" }
func (goneTestError) Gone() bool    { return true }

type recordingEmail struct {
	addresses []string
	bodies    []string
}

func (s *recordingEmail) Send(_ context.Context, address, _ string, body string) (string, error) {
	s.addresses = append(s.addresses, address)
	s.bodies = append(s.bodies, body)
	return "smtp-provider", nil
}

func (s *recordingEmail) Available() (bool, string) { return true, "test SMTP sender" }

type recordingDesktop struct {
	channels []string
	bodies   []string
}

func (s *recordingDesktop) Send(_ context.Context, channel, _, _ string, body string) (string, error) {
	s.channels = append(s.channels, channel)
	s.bodies = append(s.bodies, body)
	return "desktop-provider", nil
}

func (s *recordingDesktop) Available(string) (bool, string) { return true, "test desktop sender" }

type recordingRemote struct {
	machineID string
	bodies    []string
}

func (r *recordingRemote) Deliver(_ context.Context, machineID string, _ hub.Notification, body string) (string, error) {
	r.machineID = machineID
	r.bodies = append(r.bodies, body)
	return "bridge-run-1", nil
}

func (r *recordingRemote) ChannelsStatus(context.Context, string, string) (hub.ChannelStatus, error) {
	return hub.ChannelStatus{Channel: "remote", Disposition: "ready"}, nil
}

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

func testService(t *testing.T) (*hub.Service, *database.RoutedDB) {
	t.Helper()
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), primary, modules.AllSchemas()...))
	db := database.NewFromPrimary(primary)
	return hub.New(db, nil, nil), db
}

func insertPending(t *testing.T, db *database.RoutedDB, recipient, body, sensitivity string) string {
	t.Helper()
	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := db.ExecContext(context.Background(), `INSERT INTO notifications (id, requested_by, title, body, urgency, sensitivity_label, idempotency_key, dedupe_key, dedupe_window_seconds, created_at, updated_at) VALUES (?, ?, 'Test', ?, 'normal', ?, ?, '', 300, ?, ?)`, id, recipient, body, sensitivity, uuid.NewString(), now, now)
	require.NoError(t, err)
	_, err = db.ExecContext(context.Background(), `INSERT INTO notification_events (notification_id, state, reason, created_at) VALUES (?, 'pending', 'test accepted', ?)`, id, now)
	require.NoError(t, err)
	return id
}

func registerSubscription(t *testing.T, service *hub.Service, recipient string) string {
	t.Helper()
	endpoint := "https://push.example.test/subscription/" + uuid.NewString()
	require.NoError(t, service.RegisterPushSubscription(context.Background(), recipient, hub.PushSubscription{
		ID: "device-1", Endpoint: endpoint, P256DH: "client-key", Auth: "auth-secret", Origin: "https://app.example.test",
	}))
	return endpoint
}

// [REQ:NOTIFICA-P0-002][REQ:NOTIFICA-P0-004][REQ:NOTIFICA-P0-012]
func TestSend_IsIdempotentAndLeavesDurableTerminalState(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	input := hub.SendInput{RequestedBy: "alice", Body: "hello", SensitivityLabel: "public", IdempotencyKey: "request-1"}
	first, err := service.Send(ctx, input)
	require.NoError(t, err)
	second, err := service.Send(ctx, input)
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)

	// No registered device is a deliberate terminal outcome, not an
	// indefinitely pending queue item.
	require.Eventually(t, func() bool {
		n, _, getErr := service.Get(ctx, first.ID)
		return getErr == nil && n.State == "unroutable"
	}, time.Second, 10*time.Millisecond)
	var pending int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM (SELECT notification_id, state, ROW_NUMBER() OVER (PARTITION BY notification_id ORDER BY id DESC) AS rank FROM notification_events) WHERE rank = 1 AND state = 'pending'`).Scan(&pending))
	require.Zero(t, pending)
	var outcome, reason string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT outcome, reason FROM delivery_attempts WHERE notification_id = ?`, first.ID).Scan(&outcome, &reason))
	require.Equal(t, "unroutable", outcome)
	require.Contains(t, reason, "no recipient configured")
}

func TestListDeliveryAttemptsForRecipientIsBoundedAndScoped(t *testing.T) {
	service, db := testService(t)
	id := insertPending(t, db, "alice", "private body", "private")
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for i := 1; i <= 3; i++ {
		_, err := db.ExecContext(context.Background(), `INSERT INTO delivery_attempts (notification_id, channel, machine_id, attempt_number, outcome, reason, next_attempt_at, created_at) VALUES (?, 'push', 'secret-machine', ?, 'retrying', 'timeout', 'later', ?)`, id, i, now)
		require.NoError(t, err)
	}
	rows, truncated, err := service.ListDeliveryAttemptsForRecipient(context.Background(), id, "alice", 2)
	require.NoError(t, err)
	require.True(t, truncated)
	require.Len(t, rows, 2)
	require.Equal(t, 1, rows[0].Attempt)
	require.NotEmpty(t, rows[0].CreatedAt)
	_, _, err = service.ListDeliveryAttemptsForRecipient(context.Background(), id, "", 2)
	require.ErrorIs(t, err, hub.ErrNotFound)
	_, _, err = service.ListDeliveryAttemptsForRecipient(context.Background(), id, "bob", 2)
	require.ErrorIs(t, err, hub.ErrNotFound)
}

// [REQ:NOTIFICA-P0-005][REQ:NOTIFICA-P0-008][REQ:NOTIFICA-P0-010]
func TestProcess_RedactsSensitiveContentAndRecordsRetryReceipt(t *testing.T) {
	service, db := testService(t)
	recipient := "alice"
	registerSubscription(t, service, recipient)
	sender := &recordingSender{errors: []error{errors.New("temporary provider failure"), errors.New("temporary provider failure"), nil}}
	service.SetPushSender(sender)
	id := insertPending(t, db, recipient, "private incident details", "secret")

	require.NoError(t, service.Process(context.Background(), id))
	n, receipts, err := service.Get(context.Background(), id)
	require.NoError(t, err)
	require.Equal(t, "delivered", n.State)
	require.Len(t, receipts, 1)
	require.Equal(t, 3, sender.attempts)
	require.Equal(t, []string{
		"Notification available in notification-hub",
		"Notification available in notification-hub",
		"Notification available in notification-hub",
	}, sender.contents)
	var pending int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM (SELECT state, ROW_NUMBER() OVER (PARTITION BY notification_id ORDER BY id DESC) AS rank FROM notification_events WHERE notification_id = ?) WHERE rank = 1 AND state = 'pending'`, id).Scan(&pending))
	require.Zero(t, pending)
}

// A delivered web_push notification carries the source scenario's resolved app
// icon URL so the operator's device shows the originating app's icon instead of
// notification-hub's own.
func TestProcess_WebPushCarriesSourceIconURL(t *testing.T) {
	service, db := testService(t)
	recipient := "alice"
	registerSubscription(t, service, recipient)
	sender := &recordingSender{}
	service.SetPushSender(sender)
	service.SetIconResolver(func(_ context.Context, source string) string {
		if source == "web-console" {
			return "https://web-console.example/public/icon-192.png"
		}
		return ""
	})

	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := db.ExecContext(context.Background(), `INSERT INTO notifications (id, requested_by, title, body, urgency, sensitivity_label, idempotency_key, dedupe_key, dedupe_window_seconds, source_scenario, created_at, updated_at) VALUES (?, ?, 'Test', 'hello', 'normal', 'public', ?, '', 300, 'web-console', ?, ?)`, id, recipient, uuid.NewString(), now, now)
	require.NoError(t, err)
	_, err = db.ExecContext(context.Background(), `INSERT INTO notification_events (notification_id, state, reason, created_at) VALUES (?, 'pending', 'test accepted', ?)`, id, now)
	require.NoError(t, err)

	require.NoError(t, service.Process(context.Background(), id))
	n, _, err := service.Get(context.Background(), id)
	require.NoError(t, err)
	require.Equal(t, "delivered", n.State)
	require.Len(t, sender.messages, 1)
	require.Equal(t, "https://web-console.example/public/icon-192.png", sender.messages[0].Icon)
}

// [REQ:NOTIFICA-P0-001][REQ:NOTIFICA-P0-014][REQ:NOTIFICA-P0-015]
func TestProcess_RemovesGoneSubscription(t *testing.T) {
	service, db := testService(t)
	endpoint := registerSubscription(t, service, "alice")
	service.SetPushSender(goneSender{})
	id := insertPending(t, db, "alice", "hello", "public")

	require.NoError(t, service.Process(context.Background(), id))
	var subscriptions int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM push_subscriptions WHERE endpoint = ?`, endpoint).Scan(&subscriptions))
	require.Zero(t, subscriptions)
	n, _, err := service.Get(context.Background(), id)
	require.NoError(t, err)
	require.Equal(t, "failed", n.State)
}

// [REQ:NOTIFICA-P0-003] [REQ:NOTIFICA-P0-005]
func TestRecipients_DevicesAndQuietWindowsAreDurable(t *testing.T) {
	service, db := testService(t)
	service.SetClockForTest(fixedClock{now: time.Date(2026, time.March, 2, 22, 30, 0, 0, time.UTC)})
	require.NoError(t, service.RegisterPushSubscription(context.Background(), "alice", hub.PushSubscription{Endpoint: "https://push.example/device", P256DH: "key", Auth: "auth", Origin: "https://app.example"}))
	window, err := service.SetQuietWindow(context.Background(), "alice", hub.QuietWindow{Weekday: 1, Start: "22:00", End: "23:00", Timezone: "UTC", CriticalOverride: true})
	require.NoError(t, err)
	require.NotEmpty(t, window.ID)
	devices, err := service.ListDevices(context.Background(), "alice")
	require.NoError(t, err)
	require.Len(t, devices, 1)
	id := insertPending(t, db, "alice", "quiet", "public")
	require.NoError(t, service.Process(context.Background(), id))
	n, _, err := service.Get(context.Background(), id)
	require.NoError(t, err)
	require.Equal(t, "held", n.State)
}

func TestSetQuietWindowRejectsAmbiguousOrHeldTimesWithoutReplacingCurrentWindow(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	current, err := service.SetQuietWindow(ctx, "alice", hub.QuietWindow{Weekday: 1, Start: "22:00", End: "23:00", Timezone: "UTC"})
	require.NoError(t, err)
	for _, tc := range []struct{ start, end string }{{"22:00", "22:00"}, {"bad", "23:00"}, {"22:00", "08:00"}} {
		_, err := service.SetQuietWindow(ctx, "alice", hub.QuietWindow{Weekday: 1, Start: tc.start, End: tc.end, Timezone: "UTC"})
		require.ErrorIs(t, err, hub.ErrInvalidArgument)
		windows, readErr := service.ListQuietWindows(ctx, "alice")
		require.NoError(t, readErr)
		require.Len(t, windows, 1)
		require.Equal(t, current.ID, windows[0].ID)
		require.Equal(t, "22:00", windows[0].Start)
	}
	_, err = service.SetQuietWindow(ctx, "alice", hub.QuietWindow{Weekday: 1, Start: "21:00", End: "23:00", Timezone: "Not/A_Real_Zone"})
	require.ErrorIs(t, err, hub.ErrInvalidArgument)
	windows, err := service.ListQuietWindows(ctx, "alice")
	require.NoError(t, err)
	require.Len(t, windows, 1)
	require.Equal(t, current.ID, windows[0].ID)
}

func TestSetQuietWindowReplacementIsAtomicWhenNewRowInsertFails(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	current, err := service.SetQuietWindow(ctx, "alice", hub.QuietWindow{Weekday: 1, Start: "22:00", End: "23:00", Timezone: "UTC"})
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `CREATE TRIGGER fail_quiet_insert BEFORE INSERT ON quiet_windows BEGIN SELECT RAISE(ABORT, 'fixture insert failure'); END`)
	require.NoError(t, err)
	_, err = service.SetQuietWindow(ctx, "alice", hub.QuietWindow{Weekday: 1, Start: "21:00", End: "23:00", Timezone: "Asia/Tokyo"})
	require.Error(t, err)
	windows, err := service.ListQuietWindows(ctx, "alice")
	require.NoError(t, err)
	require.Len(t, windows, 1)
	require.Equal(t, current.ID, windows[0].ID)
	require.Equal(t, "22:00", windows[0].Start)
	require.Equal(t, "UTC", windows[0].Timezone)
}

func TestSetEscalationChainIsAtomicAndPreservesUnknownChannelKeys(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	_, err := service.SetEscalationChain(ctx, "alice", []string{"email", "future-channel"})
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `CREATE TRIGGER fail_escalation_insert BEFORE INSERT ON escalation_chains WHEN NEW.channel='blocked-channel' BEGIN SELECT RAISE(ABORT, 'fixture insert failure'); END`)
	require.NoError(t, err)
	_, err = service.SetEscalationChain(ctx, "alice", []string{"webhook", "blocked-channel"})
	require.Error(t, err)
	chain, err := service.GetEscalationChain(ctx, "alice")
	require.NoError(t, err)
	require.Len(t, chain.Steps, 2)
	require.Equal(t, "email", chain.Steps[0].Channel)
	require.Equal(t, "future-channel", chain.Steps[1].Channel)
}

// [REQ:NOTIFICA-P0-006]
func TestProcess_DedupeCollapsesRepeatedKey(t *testing.T) {
	service, db := testService(t)
	first := insertPending(t, db, "alice", "first", "public")
	second := insertPending(t, db, "alice", "second", "public")
	_, err := db.ExecContext(context.Background(), `UPDATE notifications SET dedupe_key='same-key', dedupe_window_seconds=300 WHERE id IN (?, ?)`, first, second)
	require.NoError(t, err)
	require.NoError(t, service.Process(context.Background(), second))
	n, _, err := service.Get(context.Background(), second)
	require.NoError(t, err)
	require.Equal(t, "suppressed", n.State)
	var reason string
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT dedupe_key FROM suppressions WHERE notification_id = ?`, second).Scan(&reason))
	require.Equal(t, "same-key", reason)
}

func TestProcess_SaturationGuardRecordsDroppedNotification(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	now := time.Now().UTC()
	ids := make([]string, 0, hub.EventSaturationLimit+1)
	for i := 0; i <= hub.EventSaturationLimit; i++ {
		id := uuid.NewString()
		created := now.Add(-time.Duration(hub.EventSaturationLimit-i) * 10 * time.Minute).Format(time.RFC3339Nano)
		_, err := db.ExecContext(ctx, `INSERT INTO notifications (id, requested_by, title, body, urgency, sensitivity_label, idempotency_key, dedupe_key, dedupe_window_seconds, created_at, updated_at) VALUES (?, 'alice', 'Test', 'body', 'normal', 'critical', ?, 'incident.opened.v1:incident-1', 300, ?, ?)`, id, uuid.NewString(), created, created)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, `INSERT INTO notification_events (notification_id, state, reason, created_at) VALUES (?, 'pending', 'test accepted', ?)`, id, created)
		require.NoError(t, err)
		ids = append(ids, id)
	}
	require.NoError(t, service.Process(ctx, ids[len(ids)-1]))
	n, _, err := service.Get(ctx, ids[len(ids)-1])
	require.NoError(t, err)
	require.Equal(t, "suppressed", n.State)
	var reason string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT reason FROM notification_events WHERE notification_id = ? AND state = 'suppressed' ORDER BY id DESC LIMIT 1`, ids[len(ids)-1]).Scan(&reason))
	require.Contains(t, reason, "saturation cap reached")
	var recorded string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT dedupe_key FROM suppressions WHERE notification_id = ?`, ids[len(ids)-1]).Scan(&recorded))
	require.Equal(t, "incident.opened.v1:incident-1", recorded)
}

func TestListDeliveryProjectionIncludesOnlyRequestedAutohealAskIdentity(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	legacyID := uuid.NewString()
	autohealAskID := uuid.NewString()
	otherSourceAskID := uuid.NewString()
	otherIncidentAskID := uuid.NewString()

	insertProjectionNotification(t, db, legacyID, "incident.opened.v1:inc-1", now)
	insertProjectionAttempt(t, db, legacyID, now)
	insertProjectionNotification(t, db, autohealAskID, "", now)
	insertProjectionAsk(t, db, autohealAskID, "vrooli-autoheal", `{"incident_id":"inc-1","incident_fingerprint":"sensitive-fingerprint","candidate_id":"sensitive-candidate","candidate_risk_level":"hostile-risk"}`, now)
	insertProjectionAttempt(t, db, autohealAskID, now)
	insertProjectionNotification(t, db, otherSourceAskID, "", now)
	insertProjectionAsk(t, db, otherSourceAskID, "another-producer", `{"incident_id":"inc-1"}`, now)
	insertProjectionAttempt(t, db, otherSourceAskID, now)
	insertProjectionNotification(t, db, otherIncidentAskID, "", now)
	insertProjectionAsk(t, db, otherIncidentAskID, "vrooli-autoheal", `{"incident_id":"inc-2"}`, now)
	insertProjectionAttempt(t, db, otherIncidentAskID, now)

	items, truncated, err := service.ListDeliveryProjection(ctx, "incident.", []string{"inc-1"}, 20)
	require.NoError(t, err)
	require.False(t, truncated)
	require.Len(t, items, 2)
	byNotification := make(map[string]hub.DeliveryProjection, len(items))
	for _, item := range items {
		byNotification[item.NotificationID] = item
		require.Len(t, item.Attempts, 1)
	}
	require.Equal(t, "incident.opened.v1:inc-1", byNotification[legacyID].DedupeKey)
	require.Empty(t, byNotification[legacyID].AutohealIncidentID)
	require.Equal(t, "inc-1", byNotification[autohealAskID].AutohealIncidentID)
	require.NotContains(t, byNotification, otherSourceAskID)
	require.NotContains(t, byNotification, otherIncidentAskID)
	encoded, err := json.Marshal(items)
	require.NoError(t, err)
	for _, secret := range []string{"private ask content", "private ask question", "sensitive-fingerprint", "sensitive-candidate", "hostile-risk"} {
		require.NotContains(t, string(encoded), secret)
	}
}

func TestListDeliveryProjectionReportsTruncation(t *testing.T) {
	service, db := testService(t)
	now := time.Now().UTC()
	for i := 0; i < 3; i++ {
		created := now.Add(time.Duration(i) * time.Second).Format(time.RFC3339Nano)
		insertProjectionNotification(t, db, uuid.NewString(), fmt.Sprintf("incident.opened.v1:inc-%d", i), created)
	}

	items, truncated, err := service.ListDeliveryProjection(context.Background(), "incident.", nil, 2)
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.True(t, truncated)
}

func TestListDeliveryProjectionBoundsAttemptsPerNotification(t *testing.T) {
	service, db := testService(t)
	notificationID := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	insertProjectionNotification(t, db, notificationID, "incident.opened.v1:inc-many", now)
	for attempt := 1; attempt <= 101; attempt++ {
		insertProjectionAttemptNumber(t, db, notificationID, attempt, now)
	}

	items, truncated, err := service.ListDeliveryProjection(context.Background(), "incident.", nil, 10)
	require.NoError(t, err)
	require.True(t, truncated, "omitted attempt history must make the projection incomplete")
	require.Len(t, items, 1)
	require.Len(t, items[0].Attempts, 100, "per-notification attempt history must be capped")
}

func TestListDeliveryProjectionBoundsTotalAttempts(t *testing.T) {
	service, db := testService(t)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for notification := 0; notification < 11; notification++ {
		id := uuid.NewString()
		insertProjectionNotification(t, db, id, fmt.Sprintf("incident.opened.v1:inc-%d", notification), now)
		for attempt := 1; attempt <= 100; attempt++ {
			insertProjectionAttemptNumber(t, db, id, attempt, now)
		}
	}

	items, truncated, err := service.ListDeliveryProjection(context.Background(), "incident.", nil, 20)
	require.NoError(t, err)
	require.True(t, truncated, "omitted attempt history after the total cap must be reported")
	totalAttempts := 0
	for _, item := range items {
		require.LessOrEqual(t, len(item.Attempts), 100, "one notification must never exceed its attempt cap")
		totalAttempts += len(item.Attempts)
	}
	require.Equal(t, 1000, totalAttempts, "attempt rows returned across the response must respect the total cap")
}

func TestListDeliveryProjectionUsesOrderedAttemptIndex(t *testing.T) {
	_, db := testService(t)
	rows, err := db.QueryContext(context.Background(), `EXPLAIN QUERY PLAN SELECT channel, machine_id, attempt_number, outcome, reason, created_at FROM delivery_attempts WHERE notification_id=? ORDER BY created_at, attempt_number, id LIMIT ?`, "notification-id", 101)
	require.NoError(t, err)
	defer rows.Close()

	var details []string
	for rows.Next() {
		var id, parent, notused int
		var detail string
		require.NoError(t, rows.Scan(&id, &parent, &notused, &detail))
		details = append(details, detail)
	}
	require.NoError(t, rows.Err())
	require.Contains(t, strings.Join(details, "\n"), "idx_delivery_attempts_projection")
	require.NotContains(t, strings.Join(details, "\n"), "USE TEMP B-TREE")
}

func insertProjectionNotification(t *testing.T, db *database.RoutedDB, id, dedupeKey, createdAt string) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `INSERT INTO notifications (id, requested_by, title, body, urgency, sensitivity_label, idempotency_key, dedupe_key, created_at, updated_at) VALUES (?, 'alice', 'Test', 'private ask content', 'high', 'private', ?, ?, ?, ?)`, id, uuid.NewString(), dedupeKey, createdAt, createdAt)
	require.NoError(t, err)
}

func insertProjectionAsk(t *testing.T, db *database.RoutedDB, notificationID, source, correlation, createdAt string) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `INSERT INTO asks (id, notification_id, question, allowed_answers, deadline, state, created_at, updated_at, source, requester_ref, correlation_json) VALUES (?, ?, 'private ask question', '["approve","reject"]', ?, 'pending', ?, ?, ?, ?, ?)`, uuid.NewString(), notificationID, time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano), createdAt, createdAt, source, uuid.NewString(), correlation)
	require.NoError(t, err)
}

func insertProjectionAttempt(t *testing.T, db *database.RoutedDB, notificationID, createdAt string) {
	insertProjectionAttemptNumber(t, db, notificationID, 1, createdAt)
}

func insertProjectionAttemptNumber(t *testing.T, db *database.RoutedDB, notificationID string, number int, createdAt string) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `INSERT INTO delivery_attempts (notification_id, channel, attempt_number, outcome, reason, created_at) VALUES (?, 'push', ?, 'delivered', '', ?)`, notificationID, number, createdAt)
	require.NoError(t, err)
}

// [REQ:NOTIFICA-P0-013]
func TestRetain_DeletesExpiredNotificationHistory(t *testing.T) {
	service, db := testService(t)
	id := insertPending(t, db, "alice", "old", "public")
	_, err := db.ExecContext(context.Background(), `UPDATE notifications SET created_at='2020-01-01T00:00:00Z' WHERE id=?`, id)
	require.NoError(t, err)
	removed, err := service.Retain(context.Background(), time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.EqualValues(t, 1, removed)
	_, _, err = service.Get(context.Background(), id)
	require.ErrorIs(t, err, hub.ErrNotFound)
}

// [REQ:NOTIFICA-P1-004][REQ:NOTIFICA-P1-006][REQ:NOTIFICA-P1-007]
func TestProcess_RoutesRegisteredChannelsAndRedactsByApproval(t *testing.T) {
	service, db := testService(t)
	email := &recordingEmail{}
	desktop := &recordingDesktop{}
	service.SetEmailSender(email)
	service.SetDesktopSender(desktop)
	device, err := service.UpsertDevice(context.Background(), "alice", hub.Device{Name: "owner laptop", MachineID: "local"})
	require.NoError(t, err)
	_, err = service.UpsertChannelAddress(context.Background(), "alice", hub.ChannelAddress{DeviceID: device.ID, Channel: "email", Address: "owner@example.test", ApprovedLabels: []string{"public"}})
	require.NoError(t, err)
	_, err = service.UpsertChannelAddress(context.Background(), "alice", hub.ChannelAddress{DeviceID: device.ID, Channel: "macos_notification", Address: "owner", ApprovedLabels: []string{"private"}})
	require.NoError(t, err)
	id := insertPending(t, db, "alice", "private incident", "private")
	require.NoError(t, service.Process(context.Background(), id))
	require.Equal(t, []string{"Notification available in notification-hub"}, email.bodies)
	require.Equal(t, []string{"private incident"}, desktop.bodies)
	n, receipts, err := service.Get(context.Background(), id)
	require.NoError(t, err)
	require.Equal(t, "delivered", n.State)
	require.Len(t, receipts, 2)
	analytics, err := service.Analytics(context.Background(), "", "")
	require.NoError(t, err)
	require.EqualValues(t, 1, analytics.TotalNotifications)
	require.Len(t, analytics.Channels, 2)
}

// [REQ:NOTIFICA-P1-001]
func TestProcess_RelaysRemoteMachineDelivery(t *testing.T) {
	service, db := testService(t)
	remote := &recordingRemote{}
	service.SetRemoteDelivery(remote)
	device, err := service.UpsertDevice(context.Background(), "alice", hub.Device{Name: "paired Mac", MachineID: "mac-node"})
	require.NoError(t, err)
	_, err = service.UpsertChannelAddress(context.Background(), "alice", hub.ChannelAddress{DeviceID: device.ID, Channel: "imessage", Address: "owner@example.test", ApprovedLabels: []string{"public"}})
	require.NoError(t, err)
	id := insertPending(t, db, "alice", "remote delivery", "public")

	require.NoError(t, service.Process(context.Background(), id))
	require.Equal(t, "mac-node", remote.machineID)
	require.Equal(t, []string{"remote delivery"}, remote.bodies)
	n, receipts, err := service.Get(context.Background(), id)
	require.NoError(t, err)
	require.Equal(t, "delivered", n.State)
	require.Len(t, receipts, 1)
	require.Equal(t, "mac-node", receipts[0].MachineID)
}

// [REQ:NOTIFICA-P1-002]
func TestProcess_DeliversIMessageThroughDesktopAdapter(t *testing.T) {
	service, db := testService(t)
	desktop := &recordingDesktop{}
	service.SetDesktopSender(desktop)
	device, err := service.UpsertDevice(context.Background(), "alice", hub.Device{Name: "paired Mac", MachineID: "local"})
	require.NoError(t, err)
	_, err = service.UpsertChannelAddress(context.Background(), "alice", hub.ChannelAddress{DeviceID: device.ID, Channel: "imessage", Address: "+15555550123", ApprovedLabels: []string{"public"}})
	require.NoError(t, err)
	id := insertPending(t, db, "alice", "iMessage delivery", "public")

	require.NoError(t, service.Process(context.Background(), id))
	require.Equal(t, []string{"imessage"}, desktop.channels)
	require.Equal(t, []string{"iMessage delivery"}, desktop.bodies)
	n, receipts, err := service.Get(context.Background(), id)
	require.NoError(t, err)
	require.Equal(t, "delivered", n.State)
	require.Len(t, receipts, 1)
	require.Equal(t, "imessage", receipts[0].Channel)
}

// [REQ:NOTIFICA-P1-006]
func TestSend_ScheduledNotificationIsHeldUntilRelease(t *testing.T) {
	clock := fixedClock{now: time.Date(2026, time.March, 2, 12, 0, 0, 0, time.UTC)}
	service, _ := testService(t)
	service.SetClockForTest(clock)
	n, err := service.Send(context.Background(), hub.SendInput{RequestedBy: "alice", Body: "later", SensitivityLabel: "public", IdempotencyKey: "scheduled-1", ScheduledAt: clock.now.Add(time.Hour)})
	require.NoError(t, err)
	require.NoError(t, service.Process(context.Background(), n.ID))
	got, _, err := service.Get(context.Background(), n.ID)
	require.NoError(t, err)
	require.Equal(t, "held", got.State)
	clock.now = clock.now.Add(2 * time.Hour)
	service.SetClockForTest(clock)
	require.NoError(t, service.Process(context.Background(), n.ID))
	got, _, err = service.Get(context.Background(), n.ID)
	require.NoError(t, err)
	require.Equal(t, "unroutable", got.State)
}

// [REQ:NOTIFICA-P1-011]
func TestAsk_EscalatesThroughConfiguredChannelsThenExpires(t *testing.T) {
	service, db := testService(t)
	clock := fixedClock{now: time.Date(2026, time.March, 2, 12, 0, 0, 0, time.UTC)}
	service.SetClockForTest(clock)
	email := &recordingEmail{}
	service.SetEmailSender(email)
	device, err := service.UpsertDevice(context.Background(), "alice", hub.Device{Name: "owner laptop", MachineID: "local"})
	require.NoError(t, err)
	_, err = service.UpsertChannelAddress(context.Background(), "alice", hub.ChannelAddress{DeviceID: device.ID, Channel: "email", Address: "owner@example.test", ApprovedLabels: []string{"critical"}})
	require.NoError(t, err)
	_, err = service.SetEscalationChain(context.Background(), "alice", []string{"email"})
	require.NoError(t, err)
	ask, _, err := service.AskWithSpec(context.Background(), hub.AskSpec{
		Recipient: "alice", Question: "approve", Options: []hub.AskOption{{Key: "yes", Label: "yes"}, {Key: "no", Label: "no"}},
		Urgency: "critical", SensitivityLabel: "critical", Deadline: clock.now.Add(time.Second), IdempotencyKey: "ask-1",
	})
	require.NoError(t, err)
	askID := ask.ID
	clock.now = clock.now.Add(2 * time.Second)
	service.SetClockForTest(clock)
	require.NoError(t, service.ProcessEscalations(context.Background()))
	var state string
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT state FROM asks WHERE id=?`, askID).Scan(&state))
	require.Equal(t, "escalated", state)
	clock.now = clock.now.Add(6 * time.Minute)
	service.SetClockForTest(clock)
	require.NoError(t, service.ProcessEscalations(context.Background()))
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT state FROM asks WHERE id=?`, askID).Scan(&state))
	require.Equal(t, "expired", state)
}

// Re-enabling browser notifications (re-adding the PWA mints a fresh service
// worker + endpoint under the same origin) must be idempotent: the device is
// keyed by (recipient, origin-name), so the second enable reuses the device
// instead of tripping UNIQUE(recipient_id, name), and the NEW endpoint is
// stored so delivery follows the live subscription.
func TestRegisterPushSubscription_ReEnableRefreshesEndpoint(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	const origin = "https://notification-hub.itsagitime.com"

	require.NoError(t, service.RegisterPushSubscription(ctx, "alice", hub.PushSubscription{
		Endpoint: "https://web.push.apple.com/old", P256DH: "k1", Auth: "a1", Origin: origin,
	}))
	// Second enable: same origin (same device name), brand-new endpoint.
	require.NoError(t, service.RegisterPushSubscription(ctx, "alice", hub.PushSubscription{
		Endpoint: "https://web.push.apple.com/new", P256DH: "k2", Auth: "a2", Origin: origin,
	}))

	// Exactly one device for this origin — the second enable reused it.
	var deviceCount int
	require.NoError(t, db.QueryRowContext(ctx,
		`SELECT count(*) FROM devices WHERE recipient_id=(SELECT id FROM recipients WHERE subject=?) AND name=?`,
		"alice", origin).Scan(&deviceCount))
	require.Equal(t, 1, deviceCount)

	// The live endpoint is stored and bound to that device.
	var boundDevice string
	require.NoError(t, db.QueryRowContext(ctx,
		`SELECT device_id FROM push_subscriptions WHERE recipient_id=(SELECT id FROM recipients WHERE subject=?) AND endpoint=?`,
		"alice", "https://web.push.apple.com/new").Scan(&boundDevice))
	require.NotEmpty(t, boundDevice)
}

// [REQ:NOTIFICA-P1-005]
func TestSend_LowUrgencyDigestCollapsesIntoOneSummary(t *testing.T) {
	service, _ := testService(t)
	clock := fixedClock{now: time.Date(2026, time.March, 2, 12, 0, 0, 0, time.UTC)}
	service.SetClockForTest(clock)
	email := &recordingEmail{}
	service.SetEmailSender(email)
	device, err := service.UpsertDevice(context.Background(), "alice", hub.Device{Name: "owner laptop", MachineID: "local"})
	require.NoError(t, err)
	_, err = service.UpsertChannelAddress(context.Background(), "alice", hub.ChannelAddress{DeviceID: device.ID, Channel: "email", Address: "owner@example.test", ApprovedLabels: []string{"public"}})
	require.NoError(t, err)
	first, err := service.Send(context.Background(), hub.SendInput{RequestedBy: "alice", Body: "first", SensitivityLabel: "public", IdempotencyKey: "digest-1", DedupeKey: "digest", Urgency: "low", DigestWindow: time.Minute})
	require.NoError(t, err)
	clock.now = clock.now.Add(10 * time.Second)
	service.SetClockForTest(clock)
	second, err := service.Send(context.Background(), hub.SendInput{RequestedBy: "alice", Body: "second", SensitivityLabel: "public", IdempotencyKey: "digest-2", DedupeKey: "digest", Urgency: "low", DigestWindow: time.Minute})
	require.NoError(t, err)
	require.NoError(t, service.Process(context.Background(), second.ID))
	collapsed, _, err := service.Get(context.Background(), second.ID)
	require.NoError(t, err)
	require.Equal(t, "suppressed", collapsed.State)
	clock.now = clock.now.Add(2 * time.Minute)
	service.SetClockForTest(clock)
	require.NoError(t, service.Process(context.Background(), first.ID))
	require.Len(t, email.bodies, 1)
	require.Contains(t, email.bodies[0], "2 notifications in digest")
}

// [REQ:NOTIFICA-P1-009][REQ:NOTIFICA-P1-010]
func TestAsk_AnswerIsValidatedAndWaitReturnsChosenAction(t *testing.T) {
	service, _ := testService(t)
	clock := fixedClock{now: time.Date(2026, time.March, 2, 12, 0, 0, 0, time.UTC)}
	service.SetClockForTest(clock)
	askID, _, err := service.Ask(context.Background(), "alice", "approve", []string{"yes", "no"}, clock.now.Add(time.Minute), "public", "ask-answer-1")
	require.NoError(t, err)
	require.NoError(t, service.Answer(context.Background(), askID, "yes", "alice"))
	state, answer, _, err := service.Wait(context.Background(), askID, clock.now.Add(time.Minute))
	require.NoError(t, err)
	require.Equal(t, "answered", state)
	require.Equal(t, "yes", answer)
}

func TestRecipientReadViewsDoNotCrossRecipients(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	aliceID := insertPending(t, db, "alice", "alice body", "public")
	bobID := insertPending(t, db, "bob", "bob body", "public")
	createdAt := time.Now().UTC().Format(time.RFC3339Nano)
	insertProjectionAttempt(t, db, aliceID, createdAt)
	insertProjectionAttempt(t, db, bobID, createdAt)

	items, err := service.ListForRecipient(ctx, "alice", 50)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, aliceID, items[0].ID)

	items, err = service.ListForRecipient(ctx, "", 50)
	require.NoError(t, err)
	require.Empty(t, items, "an unset recipient must not become an unscoped list")

	_, _, err = service.GetForRecipient(ctx, bobID, "alice")
	require.ErrorIs(t, err, hub.ErrNotFound)
	_, _, err = service.GetForRecipient(ctx, aliceID, "")
	require.ErrorIs(t, err, hub.ErrNotFound)

	analytics, err := service.AnalyticsForRecipient(ctx, "", "", "alice")
	require.NoError(t, err)
	require.EqualValues(t, 1, analytics.TotalNotifications)
	require.Len(t, analytics.Channels, 1)
	require.EqualValues(t, 1, analytics.Channels[0].Attempts)

	analytics, err = service.AnalyticsForRecipient(ctx, "", "", "")
	require.NoError(t, err)
	require.Zero(t, analytics.TotalNotifications)
	require.Empty(t, analytics.Channels)
}
