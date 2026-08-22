package hub_test

import (
	"context"
	"errors"
	"fmt"
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
	errors   []error
}

func (s *recordingSender) Send(_ context.Context, _ hub.PushSubscription, _, body string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attempts++
	s.contents = append(s.contents, body)
	index := s.attempts - 1
	if index < len(s.errors) && s.errors[index] != nil {
		return "", s.errors[index]
	}
	return fmt.Sprintf("provider-%d", s.attempts), nil
}

type goneSender struct{}

func (goneSender) Send(context.Context, hub.PushSubscription, string, string) (string, error) {
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
	askID, _, err := service.Ask(context.Background(), "alice", "approve", []string{"yes", "no"}, clock.now.Add(time.Second), "critical", "ask-1")
	require.NoError(t, err)
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
	require.NoError(t, service.Answer(context.Background(), askID, "yes", "owner"))
	state, answer, _, err := service.Wait(context.Background(), askID, clock.now.Add(time.Minute))
	require.NoError(t, err)
	require.Equal(t, "answered", state)
	require.Equal(t, "yes", answer)
}
