// Package hub owns the notification lifecycle. Transport handlers, CLI
// clients, and background workers all call this package; none of them make
// routing or delivery decisions themselves.
package hub

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/database"
)

const (
	MaxAttempts       = 3
	DefaultDedupeWind = 5 * time.Minute
	// EventSaturationLimit bounds repeated notifications for one fingerprint
	// even when they arrive outside the shorter duplicate-collapse window.
	EventSaturationLimit  = 3
	EventSaturationWindow = time.Hour
	PendingSweepAfter     = 2 * time.Minute
)

var (
	ErrInvalidArgument = errors.New("invalid notification argument")
	ErrNotFound        = errors.New("notification not found")
	ErrExpired         = errors.New("ask deadline expired")
)

type Clock interface{ Now() time.Time }

type PushSubscription struct {
	ID, Endpoint, P256DH, Auth, Origin string
}

// PushSender is intentionally narrow. Production wires the RFC 8291 sender;
// tests inject deterministic senders that can fail or succeed per attempt.
type PushSender interface {
	Send(context.Context, PushSubscription, string, string) (string, error)
}

// SubscriptionGone is implemented by transports when a provider has
// permanently retired an endpoint (HTTP 404/410 for Web Push).
type SubscriptionGone interface {
	error
	Gone() bool
}

// RemoteDelivery is the only seam the hub needs from the fleet. Bridge owns
// machine lineage, authorization, durable dispatch, and remote run waiting;
// the hub owns the notification body and receipt shape.
type RemoteDelivery interface {
	Deliver(context.Context, string, Notification, string) (string, error)
	ChannelsStatus(context.Context, string, string) (ChannelStatus, error)
}

// ChannelDelivery delegates one-shot delivery to Switchboard's shared
// descriptor and adapter registry. Notification-hub does not own a second
// implementation for conversational channels.
type ChannelDelivery interface {
	Send(context.Context, string, string, string, string) (string, error)
}

type unavailablePush struct{}

func (unavailablePush) Send(context.Context, PushSubscription, string, string) (string, error) {
	return "", errors.New("web push sender is not configured")
}

type Service struct {
	db                *database.RoutedDB
	clock             Clock
	log               *log.Logger
	mu                sync.RWMutex
	push              PushSender
	email             EmailSender
	desktop           DesktopSender
	remote            RemoteDelivery
	channel           ChannelDelivery
	defaultRecipient  string
	recipientResolver func(context.Context) string
	webPushPublicKey  string
	worker            chan string
}

func New(db *database.RoutedDB, clock Clock, logger *log.Logger) *Service {
	if clock == nil {
		clock = systemClock{}
	}
	if logger == nil {
		logger = log.Default()
	}
	s := &Service{db: db, clock: clock, log: logger, push: unavailablePush{}, email: unavailableEmail{}, desktop: unavailableDesktop{}, worker: make(chan string, 128)}
	go s.runWorker()
	return s
}

func (s *Service) SetPushSender(sender PushSender) {
	if sender == nil {
		sender = unavailablePush{}
	}
	s.mu.Lock()
	s.push = sender
	s.mu.Unlock()
}

// SetWebPushPublicKey publishes only the VAPID public key to authenticated
// clients that need to create a browser subscription. The private key remains
// inside the transport and is never part of the service response.
func (s *Service) SetWebPushPublicKey(publicKey string) {
	s.mu.Lock()
	s.webPushPublicKey = strings.TrimSpace(publicKey)
	s.mu.Unlock()
}

func (s *Service) WebPushPublicKey() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.webPushPublicKey
}

func (s *Service) SetEmailSender(sender EmailSender) {
	if sender == nil {
		sender = unavailableEmail{}
	}
	s.mu.Lock()
	s.email = sender
	s.mu.Unlock()
}

func (s *Service) SetDesktopSender(sender DesktopSender) {
	if sender == nil {
		sender = unavailableDesktop{}
	}
	s.mu.Lock()
	s.desktop = sender
	s.mu.Unlock()
}

func (s *Service) SetRemoteDelivery(sender RemoteDelivery) {
	s.mu.Lock()
	s.remote = sender
	s.mu.Unlock()
}

func (s *Service) SetChannelDelivery(sender ChannelDelivery) {
	s.mu.Lock()
	s.channel = sender
	s.mu.Unlock()
}

// SetDefaultRecipient supplies the owner subject for inbound integrations
// that do not carry an end-user identity (for example durable event fanout).
// It is the explicit environment override; SetRecipientResolver supplies the
// durable operator setting consulted when no override is set.
func (s *Service) SetDefaultRecipient(subject string) {
	s.mu.Lock()
	s.defaultRecipient = strings.TrimSpace(subject)
	s.mu.Unlock()
}

// SetRecipientResolver supplies the operator-state lookup for the recipient
// of inbound integrations; it is consulted after the explicit override.
func (s *Service) SetRecipientResolver(resolve func(context.Context) string) {
	s.mu.Lock()
	s.recipientResolver = resolve
	s.mu.Unlock()
}

// ResolveRecipient answers who an inbound integration's notification is for:
// the explicit override first, then the operator state, else "". A caller
// that gets "" must record the notification as unroutable with
// RecipientSettingHint so the fix is named, never silently drop it.
func (s *Service) ResolveRecipient(ctx context.Context) string {
	s.mu.RLock()
	override := s.defaultRecipient
	resolve := s.recipientResolver
	s.mu.RUnlock()
	if override != "" {
		return override
	}
	if resolve != nil {
		return strings.TrimSpace(resolve(ctx))
	}
	return ""
}

// RecipientSettingHint names the settings that route an integration's
// notifications to a person.
const RecipientSettingHint = "set notifications.recipient in operator-state.json (vrooli-onboarding host set-recipient --subject <subject>) or VROOLI_NOTIFICATION_RECIPIENT, then register a channel address for that recipient"

func (s *Service) DefaultRecipient() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.defaultRecipient
}

// SetClockForTest replaces the clock seam for deterministic routing tests.
func (s *Service) SetClockForTest(clock Clock) {
	if clock == nil {
		return
	}
	s.clock = clock
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

type SendInput struct {
	RequestedBy, Title, Body, Urgency, SensitivityLabel, IdempotencyKey, DedupeKey string
	DedupeWindow                                                                   time.Duration
	ScheduledAt                                                                    time.Time
	DigestWindow                                                                   time.Duration
}

type Notification struct {
	ID, RequestedBy, Title, Body, Urgency, SensitivityLabel, IdempotencyKey, DedupeKey string
	ScheduledAt, DigestKey                                                             string
	DigestWindowSeconds                                                                int
	State, Reason, CreatedAt, UpdatedAt                                                string
}

type Receipt struct {
	ID, NotificationID, Channel, MachineID, ProviderID, DeliveredAt string
}

type ChannelStatus struct {
	Channel, MachineID, Disposition, Reason, ObservedAt string
}

type Device struct {
	ID, Name, MachineID string
	Channels            []string
}

type QuietWindow struct {
	ID                   string
	Weekday              int
	Start, End, Timezone string
	CriticalOverride     bool
}

func (s *Service) Send(ctx context.Context, in SendInput) (Notification, error) {
	if strings.TrimSpace(in.RequestedBy) == "" || strings.TrimSpace(in.Body) == "" || strings.TrimSpace(in.SensitivityLabel) == "" || strings.TrimSpace(in.IdempotencyKey) == "" {
		return Notification{}, fmt.Errorf("%w: requested_by, body, sensitivity_label, and idempotency_key are required", ErrInvalidArgument)
	}
	if in.Urgency == "" {
		in.Urgency = "normal"
	}
	if in.DedupeWindow <= 0 {
		in.DedupeWindow = DefaultDedupeWind
	}
	if err := s.EnsureRecipient(ctx, in.RequestedBy); err != nil {
		return Notification{}, err
	}
	now := s.clock.Now().UTC().Format(time.RFC3339Nano)
	scheduledAt := ""
	if !in.ScheduledAt.IsZero() {
		scheduledAt = in.ScheduledAt.UTC().Format(time.RFC3339Nano)
		if in.ScheduledAt.Before(s.clock.Now()) {
			return Notification{}, fmt.Errorf("%w: scheduled_at must be in the future", ErrInvalidArgument)
		}
	}
	digestKey := ""
	if in.DigestWindow > 0 && in.Urgency == "low" {
		digestKey = in.DedupeKey
	}
	n := Notification{ID: uuid.NewString(), RequestedBy: in.RequestedBy, Title: in.Title, Body: in.Body, Urgency: in.Urgency, SensitivityLabel: in.SensitivityLabel, IdempotencyKey: in.IdempotencyKey, DedupeKey: in.DedupeKey, ScheduledAt: scheduledAt, DigestKey: digestKey, DigestWindowSeconds: int(in.DigestWindow.Seconds()), State: "pending", CreatedAt: now, UpdatedAt: now}
	_, err := s.db.ExecContext(ctx, `INSERT INTO notifications (id, requested_by, title, body, urgency, sensitivity_label, idempotency_key, dedupe_key, dedupe_window_seconds, scheduled_at, digest_key, digest_window_seconds, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, n.ID, n.RequestedBy, n.Title, n.Body, n.Urgency, n.SensitivityLabel, n.IdempotencyKey, n.DedupeKey, int(in.DedupeWindow.Seconds()), n.ScheduledAt, n.DigestKey, n.DigestWindowSeconds, now, now)
	if err != nil {
		if existing, lookupErr := s.findByIdempotency(ctx, in.RequestedBy, in.IdempotencyKey); lookupErr == nil {
			return existing, nil
		}
		return Notification{}, fmt.Errorf("persist notification: %w", err)
	}
	if err := s.appendEvent(ctx, n.ID, "pending", "accepted before routing", now); err != nil {
		return Notification{}, err
	}
	// The id is durable before this channel is scheduled. A caller never waits
	// on a provider to learn that its request exists.
	select {
	case s.worker <- n.ID:
	default:
		go func() { s.worker <- n.ID }()
	}
	return n, nil
}

func (s *Service) findByIdempotency(ctx context.Context, requestedBy, key string) (Notification, error) {
	return s.scanNotification(s.db.QueryRowContext(ctx, `SELECT id, requested_by, title, body, urgency, sensitivity_label, idempotency_key, dedupe_key, scheduled_at, digest_key, digest_window_seconds, created_at, updated_at FROM notifications WHERE requested_by = ? AND idempotency_key = ?`, requestedBy, key))
}

func (s *Service) Get(ctx context.Context, id string) (Notification, []Receipt, error) {
	n, err := s.scanNotification(s.db.QueryRowContext(ctx, `SELECT id, requested_by, title, body, urgency, sensitivity_label, idempotency_key, dedupe_key, scheduled_at, digest_key, digest_window_seconds, created_at, updated_at FROM notifications WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Notification{}, nil, ErrNotFound
	}
	if err != nil {
		return Notification{}, nil, err
	}
	state, reason, err := s.currentState(ctx, id)
	if err != nil {
		return Notification{}, nil, err
	}
	n.State, n.Reason = state, reason
	receipts, err := s.listReceipts(ctx, id)
	return n, receipts, err
}

func (s *Service) List(ctx context.Context, limit int) ([]Notification, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, requested_by, title, body, urgency, sensitivity_label, idempotency_key, dedupe_key, scheduled_at, digest_key, digest_window_seconds, created_at, updated_at FROM notifications ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Notification
	for rows.Next() {
		n, scanErr := s.scanNotification(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		n.State, n.Reason, scanErr = s.currentState(ctx, n.ID)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (s *Service) runWorker() {
	s.recoverPending(context.Background())
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case id := <-s.worker:
			if err := s.Process(context.Background(), id); err != nil {
				s.log.Printf("notification %s: %v", id, err)
			}
		case <-ticker.C:
			s.recoverPending(context.Background())
			if err := s.ProcessEscalations(context.Background()); err != nil {
				s.log.Printf("process ask escalations: %v", err)
			}
		}
	}
}

func (s *Service) recoverPending(ctx context.Context) {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM notifications ORDER BY created_at ASC`)
	if err != nil {
		return
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) != nil {
			continue
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return
	}
	_ = rows.Close()
	for _, id := range ids {
		n, _, getErr := s.Get(ctx, id)
		if getErr == nil && (n.State == "pending" || n.State == "held") {
			if processErr := s.Process(ctx, id); processErr != nil {
				s.log.Printf("recover notification %s: %v", id, processErr)
			}
		}
	}
}

func (s *Service) Process(ctx context.Context, id string) error {
	n, _, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if isTerminal(n.State) {
		return nil
	}
	if released, wait, err := s.releaseScheduledHold(ctx, n); err != nil {
		return err
	} else if wait {
		return nil
	} else if released {
		// Continue through routing after the durable hold is released.
		n, _, err = s.Get(ctx, id)
		if err != nil {
			return err
		}
	}
	if n.ScheduledAt != "" {
		scheduled, parseErr := time.Parse(time.RFC3339Nano, n.ScheduledAt)
		if parseErr != nil {
			return s.appendEvent(ctx, n.ID, "failed", "scheduled_at is invalid", s.clock.Now().UTC().Format(time.RFC3339Nano))
		}
		if s.clock.Now().Before(scheduled) {
			return s.holdUntil(ctx, n.ID, scheduled, "scheduled delivery")
		}
	}
	if n.DigestKey != "" && n.DigestWindowSeconds > 0 && n.Urgency == "low" {
		firstID, firstCreated, err := s.firstDigestNotification(ctx, n)
		if err != nil {
			return err
		}
		if firstID != "" && firstID != n.ID {
			return s.appendEvent(ctx, n.ID, "suppressed", "collapsed into digest "+firstID, s.clock.Now().UTC().Format(time.RFC3339Nano))
		}
		if firstCreated != "" {
			created, parseErr := time.Parse(time.RFC3339Nano, firstCreated)
			if parseErr == nil {
				releaseAt := created.Add(time.Duration(n.DigestWindowSeconds) * time.Second)
				if s.clock.Now().Before(releaseAt) {
					return s.holdUntil(ctx, n.ID, releaseAt, "digest window")
				}
				var count int
				if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM notifications WHERE requested_by=? AND digest_key=? AND created_at < ?`, n.RequestedBy, n.DigestKey, releaseAt.UTC().Format(time.RFC3339Nano)).Scan(&count); err == nil && count > 1 {
					n.Body = fmt.Sprintf("%s (%d notifications in digest)", n.Body, count)
				}
			}
		}
	}
	if n.DedupeKey != "" && n.DigestKey == "" {
		var collapsed string
		err = s.db.QueryRowContext(ctx, `SELECT id FROM notifications WHERE requested_by = ? AND dedupe_key = ? AND id != ? AND julianday(created_at) >= julianday(?) - (dedupe_window_seconds / 86400.0) ORDER BY created_at ASC LIMIT 1`, n.RequestedBy, n.DedupeKey, n.ID, n.CreatedAt).Scan(&collapsed)
		if err == nil && collapsed != "" {
			return s.recordSuppression(ctx, n, collapsed, "duplicate event within dedupe window")
		}
	}
	if n.DedupeKey != "" && n.DigestKey == "" {
		var prior int
		if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM notifications WHERE requested_by = ? AND dedupe_key = ? AND id != ? AND julianday(created_at) >= julianday(?) - (? / 86400.0)`, n.RequestedBy, n.DedupeKey, n.ID, n.CreatedAt, EventSaturationWindow.Seconds()).Scan(&prior); err != nil {
			return err
		}
		if prior >= EventSaturationLimit {
			var collapsed string
			if err := s.db.QueryRowContext(ctx, `SELECT id FROM notifications WHERE requested_by = ? AND dedupe_key = ? AND id != ? ORDER BY created_at ASC LIMIT 1`, n.RequestedBy, n.DedupeKey, n.ID).Scan(&collapsed); err != nil {
				return err
			}
			return s.recordSuppression(ctx, n, collapsed, fmt.Sprintf("saturation cap reached: %d notifications per %s", EventSaturationLimit, EventSaturationWindow))
		}
	}
	quiet, err := s.inQuietWindow(ctx, n.RequestedBy, n.Urgency, s.clock.Now())
	if err != nil {
		return err
	}
	if quiet {
		return s.appendEvent(ctx, n.ID, "held", "quiet window active; critical override did not apply", s.clock.Now().UTC().Format(time.RFC3339Nano))
	}
	targets, err := s.channelTargets(ctx, n.RequestedBy)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		subs, subErr := s.liveSubscriptions(ctx, n.RequestedBy)
		if subErr != nil {
			return subErr
		}
		for _, sub := range subs {
			targets = append(targets, channelTarget{Channel: "web_push", Address: sub.Endpoint, MachineID: "local", ApprovedLabels: []string{"public", "private", "sensitive", "critical"}})
		}
	}
	if len(targets) == 0 {
		now := s.clock.Now().UTC().Format(time.RFC3339Nano)
		reason := "no recipient configured: no channel address or live web push subscription is registered for recipient " + strconv.Quote(n.RequestedBy) + "; " + RecipientSettingHint
		if _, err := s.db.ExecContext(ctx, `INSERT INTO delivery_attempts (notification_id, channel, machine_id, attempt_number, outcome, reason, next_attempt_at, created_at) VALUES (?, 'none', '', 1, 'unroutable', ?, '', ?)`, n.ID, reason, now); err != nil {
			return err
		}
		return s.appendEvent(ctx, n.ID, "unroutable", reason, now)
	}
	_ = s.appendEvent(ctx, n.ID, "routed", "channel targets selected from recipient registry", s.clock.Now().UTC().Format(time.RFC3339Nano))
	s.mu.RLock()
	sender := s.push
	emailSender := s.email
	desktopSender := s.desktop
	remoteSender := s.remote
	s.mu.RUnlock()
	var lastErr error
	delivered := 0
	for _, target := range targets {
		deliveredForTarget := false
		approved := approvedLabel(target.ApprovedLabels, n.SensitivityLabel)
		contentMode := "body"
		body := n.Body
		if !approved {
			contentMode = "pointer"
			body = "Notification available in notification-hub"
		}
		_, _ = s.db.ExecContext(ctx, `INSERT INTO routing_decisions (id, notification_id, channel, machine_id, approved, content_mode, reason, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, uuid.NewString(), n.ID, target.Channel, target.MachineID, boolInt(approved), contentMode, func() string {
			if approved {
				return "channel approved for sensitivity label"
			}
			return "channel not approved; content-free pointer used"
		}(), s.clock.Now().UTC().Format(time.RFC3339Nano))
		for attempt := 1; attempt <= MaxAttempts; attempt++ {
			started := s.clock.Now().UTC().Format(time.RFC3339Nano)
			provider, sendErr := s.deliverTarget(ctx, target, n, body, sender, emailSender, desktopSender, remoteSender)
			outcome, reason := "delivered", ""
			if sendErr != nil {
				outcome, reason, lastErr = "failed", safeReason(sendErr), sendErr
			}
			_, _ = s.db.ExecContext(ctx, `INSERT INTO delivery_attempts (notification_id, channel, machine_id, attempt_number, outcome, reason, next_attempt_at, created_at) VALUES (?, ?, ?, ?, ?, ?, '', ?)`, n.ID, target.Channel, target.MachineID, attempt, outcome, reason, started)
			if sendErr == nil {
				now := s.clock.Now().UTC().Format(time.RFC3339Nano)
				r := Receipt{ID: uuid.NewString(), NotificationID: n.ID, Channel: "web_push", ProviderID: provider, DeliveredAt: now}
				r.Channel = target.Channel
				r.MachineID = target.MachineID
				_, _ = s.db.ExecContext(ctx, `INSERT INTO receipts (id, notification_id, channel, machine_id, provider_id, delivered_at) VALUES (?, ?, ?, ?, ?, ?)`, r.ID, r.NotificationID, r.Channel, r.MachineID, r.ProviderID, r.DeliveredAt)
				delivered++
				deliveredForTarget = true
				break
			}
			var gone SubscriptionGone
			if target.Channel == "web_push" && errors.As(sendErr, &gone) && gone.Gone() {
				if err := s.RemovePushSubscription(ctx, n.RequestedBy, target.Address); err != nil {
					return err
				}
				break
			}
			if attempt < MaxAttempts {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(time.Duration(1<<(attempt-1)) * 10 * time.Millisecond):
				}
			}
		}
		if deliveredForTarget {
			continue
		}
	}
	if delivered > 0 {
		return s.appendEvent(ctx, n.ID, "delivered", "at least one approved channel accepted the payload", s.clock.Now().UTC().Format(time.RFC3339Nano))
	}
	return s.appendEvent(ctx, n.ID, "failed", fmt.Sprintf("retry budget exhausted: %s", safeReason(lastErr)), s.clock.Now().UTC().Format(time.RFC3339Nano))
}

func (s *Service) recordSuppression(ctx context.Context, n Notification, collapsedInto, reason string) error {
	now := s.clock.Now().UTC().Format(time.RFC3339Nano)
	if _, err := s.db.ExecContext(ctx, `INSERT INTO suppressions (id, notification_id, collapsed_into, dedupe_key, created_at) VALUES (?, ?, ?, ?, ?)`, uuid.NewString(), n.ID, collapsedInto, n.DedupeKey, now); err != nil {
		return err
	}
	return s.appendEvent(ctx, n.ID, "suppressed", reason+"; collapsed into "+collapsedInto, now)
}

func (s *Service) deliverTarget(ctx context.Context, target channelTarget, n Notification, body string, push PushSender, email EmailSender, desktop DesktopSender, remote RemoteDelivery) (string, error) {
	if target.MachineID != "" && target.MachineID != "local" {
		if remote == nil {
			return "", fmt.Errorf("remote machine %q is not configured", target.MachineID)
		}
		return remote.Deliver(ctx, target.MachineID, n, body)
	}
	switch target.Channel {
	case "web_push":
		sub, err := s.pushSubscription(ctx, n.RequestedBy, target.Address)
		if err != nil {
			return "", err
		}
		return push.Send(ctx, sub, n.Title, body)
	case "email":
		return email.Send(ctx, target.Address, n.Title, body)
	case "macos_notification", "imessage", "linux_notification":
		return desktop.Send(ctx, target.Channel, target.Address, n.Title, body)
	case "in-app", "telegram", "slack":
		s.mu.RLock()
		channel := s.channel
		s.mu.RUnlock()
		if channel == nil {
			return "", fmt.Errorf("shared switchboard channel registry is not configured")
		}
		return channel.Send(ctx, target.Channel, target.Address, n.Title, body)
	default:
		return "", fmt.Errorf("channel %q has no adapter", target.Channel)
	}
}

func (s *Service) pushSubscription(ctx context.Context, recipient, endpoint string) (PushSubscription, error) {
	var sub PushSubscription
	err := s.db.QueryRowContext(ctx, `SELECT id, endpoint, p256dh, auth, origin FROM push_subscriptions WHERE recipient_id=(SELECT id FROM recipients WHERE subject=?) AND endpoint=?`, recipient, endpoint).Scan(&sub.ID, &sub.Endpoint, &sub.P256DH, &sub.Auth, &sub.Origin)
	return sub, err
}

func (s *Service) holdUntil(ctx context.Context, notificationID string, releaseAt time.Time, reason string) error {
	now := s.clock.Now().UTC().Format(time.RFC3339Nano)
	if _, err := s.db.ExecContext(ctx, `INSERT INTO holds (id, notification_id, release_at, released_at, reason) VALUES (?, ?, ?, '', ?) ON CONFLICT(notification_id) DO UPDATE SET release_at=excluded.release_at, reason=excluded.reason`, uuid.NewString(), notificationID, releaseAt.UTC().Format(time.RFC3339Nano), reason); err != nil {
		return err
	}
	var state string
	_ = s.db.QueryRowContext(ctx, `SELECT state FROM notification_events WHERE notification_id=? ORDER BY id DESC LIMIT 1`, notificationID).Scan(&state)
	if state != "held" {
		return s.appendEvent(ctx, notificationID, "held", reason+"; release at "+releaseAt.UTC().Format(time.RFC3339Nano), now)
	}
	return nil
}

// releaseScheduledHold materializes the hold decision before any nested query
// so the single-connection SQLite pool cannot deadlock on an open cursor.
func (s *Service) releaseScheduledHold(ctx context.Context, n Notification) (released, wait bool, err error) {
	var releaseAt, releasedAt string
	err = s.db.QueryRowContext(ctx, `SELECT release_at, released_at FROM holds WHERE notification_id=?`, n.ID).Scan(&releaseAt, &releasedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	if releasedAt != "" {
		return false, false, nil
	}
	due, parseErr := time.Parse(time.RFC3339Nano, releaseAt)
	if parseErr != nil {
		return false, false, fmt.Errorf("invalid hold release time: %w", parseErr)
	}
	if s.clock.Now().Before(due) {
		return false, true, nil
	}
	now := s.clock.Now().UTC().Format(time.RFC3339Nano)
	if _, err := s.db.ExecContext(ctx, `UPDATE holds SET released_at=? WHERE notification_id=? AND released_at=''`, now, n.ID); err != nil {
		return false, false, err
	}
	return true, false, nil
}

func (s *Service) firstDigestNotification(ctx context.Context, n Notification) (id, created string, err error) {
	err = s.db.QueryRowContext(ctx, `SELECT id, created_at FROM notifications WHERE requested_by=? AND digest_key=? AND created_at <= ? ORDER BY created_at ASC LIMIT 1`, n.RequestedBy, n.DigestKey, n.CreatedAt).Scan(&id, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", nil
	}
	return id, created, err
}

func (s *Service) ListDevices(ctx context.Context, recipient string) ([]Device, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT d.id, d.name, d.machine_id FROM devices d JOIN recipients r ON r.id=d.recipient_id WHERE r.subject=? ORDER BY d.created_at DESC`, recipient)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var devices []Device
	for rows.Next() {
		var id, name, machineID string
		if err := rows.Scan(&id, &name, &machineID); err != nil {
			return nil, err
		}
		devices = append(devices, Device{ID: id, Name: name, MachineID: machineID})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	_ = rows.Close()
	var out []Device
	for _, device := range devices {
		channelRows, channelErr := s.db.QueryContext(ctx, `SELECT channel FROM channel_addresses WHERE device_id=? ORDER BY channel`, device.ID)
		if channelErr != nil {
			return nil, channelErr
		}
		defer channelRows.Close()
		for channelRows.Next() {
			var channel string
			if scanErr := channelRows.Scan(&channel); scanErr != nil {
				return nil, scanErr
			}
			device.Channels = append(device.Channels, channel)
		}
		if err := channelRows.Err(); err != nil {
			return nil, err
		}
		out = append(out, device)
	}
	return out, nil
}

func (s *Service) SetQuietWindow(ctx context.Context, recipient string, window QuietWindow) (QuietWindow, error) {
	if recipient == "" || window.Weekday < 0 || window.Weekday > 6 || window.Start == "" || window.End == "" || window.Timezone == "" {
		return QuietWindow{}, fmt.Errorf("%w: recipient, weekday, start, end, and timezone are required", ErrInvalidArgument)
	}
	if _, err := time.LoadLocation(window.Timezone); err != nil {
		return QuietWindow{}, fmt.Errorf("%w: invalid timezone", ErrInvalidArgument)
	}
	if err := s.EnsureRecipient(ctx, recipient); err != nil {
		return QuietWindow{}, err
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM quiet_windows WHERE recipient_id = (SELECT id FROM recipients WHERE subject=?) AND weekday = ?`, recipient, window.Weekday)
	if err != nil {
		return QuietWindow{}, err
	}
	window.ID = uuid.NewString()
	_, err = s.db.ExecContext(ctx, `INSERT INTO quiet_windows (id, recipient_id, weekday, start_time, end_time, timezone, critical_override) VALUES (?, (SELECT id FROM recipients WHERE subject=?), ?, ?, ?, ?, ?)`, window.ID, recipient, window.Weekday, window.Start, window.End, window.Timezone, boolInt(window.CriticalOverride))
	return window, err
}

func (s *Service) inQuietWindow(ctx context.Context, recipient, urgency string, now time.Time) (bool, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT weekday, start_time, end_time, timezone, critical_override FROM quiet_windows WHERE recipient_id = (SELECT id FROM recipients WHERE subject=?)`, recipient)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var weekday int
		var start, end, timezone string
		var criticalOverride int
		if err := rows.Scan(&weekday, &start, &end, &timezone, &criticalOverride); err != nil {
			return false, err
		}
		location, err := time.LoadLocation(timezone)
		if err != nil {
			continue
		}
		local := now.In(location)
		if weekday != int(local.Weekday()) {
			continue
		}
		current := local.Hour()*60 + local.Minute()
		parseMinutes := func(value string) int {
			parsed, parseErr := time.Parse("15:04", value)
			if parseErr != nil {
				return -1
			}
			return parsed.Hour()*60 + parsed.Minute()
		}
		from, until := parseMinutes(start), parseMinutes(end)
		if from < 0 || until < 0 {
			continue
		}
		active := current >= from && current < until
		if from > until {
			active = current >= from || current < until
		}
		if active {
			if urgency == "critical" && criticalOverride != 0 {
				return false, nil
			}
			return true, nil
		}
	}
	return false, rows.Err()
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func (s *Service) liveSubscriptions(ctx context.Context, recipient string) ([]PushSubscription, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, endpoint, p256dh, auth, origin FROM push_subscriptions WHERE recipient_id = (SELECT id FROM recipients WHERE subject=?) ORDER BY updated_at DESC`, recipient)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PushSubscription
	for rows.Next() {
		var p PushSubscription
		if err := rows.Scan(&p.ID, &p.Endpoint, &p.P256DH, &p.Auth, &p.Origin); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Service) RegisterPushSubscription(ctx context.Context, recipient string, p PushSubscription) error {
	if recipient == "" || p.Endpoint == "" || p.P256DH == "" || p.Auth == "" || p.Origin == "" {
		return fmt.Errorf("%w: recipient and complete push subscription are required", ErrInvalidArgument)
	}
	now := s.clock.Now().UTC().Format(time.RFC3339Nano)
	if err := s.EnsureRecipient(ctx, recipient); err != nil {
		return err
	}
	device, err := s.UpsertDevice(ctx, recipient, Device{Name: p.Origin, MachineID: "local"})
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO channel_addresses (id, device_id, channel, address, approved_labels, created_at) VALUES (?, ?, 'web_push', ?, '["public","private","sensitive","critical"]', ?) ON CONFLICT(device_id, channel) DO UPDATE SET address=excluded.address, approved_labels=excluded.approved_labels`, uuid.NewString(), device.ID, p.Endpoint, now)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO push_subscriptions (id, recipient_id, device_id, endpoint, p256dh, auth, origin, created_at, updated_at) VALUES (?, (SELECT id FROM recipients WHERE subject=?), ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(recipient_id, endpoint) DO UPDATE SET device_id=excluded.device_id, p256dh=excluded.p256dh, auth=excluded.auth, origin=excluded.origin, updated_at=excluded.updated_at`, uuid.NewString(), recipient, device.ID, p.Endpoint, p.P256DH, p.Auth, p.Origin, now, now)
	return err
}

func (s *Service) RemovePushSubscription(ctx context.Context, recipient, endpoint string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM push_subscriptions WHERE recipient_id = (SELECT id FROM recipients WHERE subject=?) AND endpoint = ?`, recipient, endpoint)
	return err
}

func (s *Service) ChannelsStatus(ctx context.Context, recipient, machine string) ([]ChannelStatus, error) {
	targets, err := s.channelTargets(ctx, recipient)
	if err != nil {
		return nil, err
	}
	now := s.clock.Now().UTC().Format(time.RFC3339Nano)
	if machine == "" {
		machine = "local"
	}
	s.mu.RLock()
	email, desktop, remote := s.email, s.desktop, s.remote
	s.mu.RUnlock()
	status := make([]ChannelStatus, 0, len(targets)+1)
	seen := make(map[string]bool)
	add := func(channel, machineID, disposition, reason string) {
		key := machineID + "\x00" + channel
		if seen[key] {
			return
		}
		seen[key] = true
		item := ChannelStatus{Channel: channel, MachineID: machineID, Disposition: disposition, Reason: reason, ObservedAt: now}
		status = append(status, item)
		_, _ = s.db.ExecContext(ctx, `INSERT INTO machine_channel_status (machine_id, channel, disposition, reason, observed_at) VALUES (?, ?, ?, ?, ?) ON CONFLICT(machine_id, channel) DO UPDATE SET disposition=excluded.disposition, reason=excluded.reason, observed_at=excluded.observed_at`, machineID, channel, disposition, reason, now)
	}
	for _, target := range targets {
		machineID := target.MachineID
		if machineID == "" {
			machineID = machine
		}
		if machineID != "local" {
			if remote == nil {
				add(target.Channel, machineID, "not_configured", "remote bridge delivery is not configured")
				continue
			}
			remoteStatus, remoteErr := remote.ChannelsStatus(ctx, machineID, recipient)
			if remoteErr != nil {
				add(target.Channel, machineID, "degraded", remoteErr.Error())
			} else {
				add(target.Channel, machineID, remoteStatus.Disposition, remoteStatus.Reason)
			}
			continue
		}
		ready, reason := false, "channel is not configured"
		switch target.Channel {
		case "web_push":
			ready, reason = true, "live subscription available"
		case "email":
			ready, reason = email.Available()
		case "macos_notification", "imessage":
			ready, reason = desktop.Available(target.Channel)
		default:
			reason = "channel adapter is not recognized"
		}
		if ready {
			add(target.Channel, machineID, "ready", reason)
		} else {
			add(target.Channel, machineID, "not_configured", reason)
		}
	}
	if len(status) == 0 {
		add("web_push", machine, "not_configured", "recipient has no registered channel address")
	}
	return status, nil
}

func (s *Service) Ask(ctx context.Context, recipient string, question string, allowed []string, deadline time.Time, sensitivity, key string) (string, Notification, error) {
	if deadline.IsZero() || len(allowed) == 0 {
		return "", Notification{}, fmt.Errorf("%w: deadline and allowed answers are required", ErrInvalidArgument)
	}
	n, err := s.Send(ctx, SendInput{RequestedBy: recipient, Title: "Decision required", Body: question, Urgency: "critical", SensitivityLabel: sensitivity, IdempotencyKey: key})
	if err != nil {
		return "", Notification{}, err
	}
	var existingAskID string
	if err := s.db.QueryRowContext(ctx, `SELECT asks.id FROM asks JOIN notifications ON notifications.id = asks.notification_id WHERE notifications.requested_by = ? AND notifications.idempotency_key = ?`, recipient, key).Scan(&existingAskID); err == nil && existingAskID != "" {
		return existingAskID, n, nil
	}
	encoded, _ := json.Marshal(allowed)
	id := uuid.NewString()
	now := s.clock.Now().UTC().Format(time.RFC3339Nano)
	_, err = s.db.ExecContext(ctx, `INSERT INTO asks (id, notification_id, question, allowed_answers, deadline, state, reason, created_at, updated_at) VALUES (?, ?, ?, ?, ?, 'pending', '', ?, ?)`, id, n.ID, question, string(encoded), deadline.UTC().Format(time.RFC3339Nano), now, now)
	return id, n, err
}

func (s *Service) Answer(ctx context.Context, askID, answer, actor string) error {
	var allowedJSON, state, deadline string
	if err := s.db.QueryRowContext(ctx, `SELECT allowed_answers, state, deadline FROM asks WHERE id = ?`, askID).Scan(&allowedJSON, &state, &deadline); err != nil {
		return err
	}
	if state != "pending" && state != "escalated" {
		return fmt.Errorf("ask is already %s", state)
	}
	var allowed []string
	_ = json.Unmarshal([]byte(allowedJSON), &allowed)
	found := false
	for _, value := range allowed {
		if value == answer {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("answer is not one of the allowed answers")
	}
	now := s.clock.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(ctx, `INSERT INTO answers (id, ask_id, answer, answered_by, answered_at) VALUES (?, ?, ?, ?, ?)`, uuid.NewString(), askID, answer, actor, now)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `UPDATE asks SET state='answered', updated_at=? WHERE id=?`, now, askID)
	return err
}

func (s *Service) Wait(ctx context.Context, askID string, deadline time.Time) (string, string, string, error) {
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		_ = s.ProcessEscalations(ctx)
		var state, reason string
		var answer sql.NullString
		err := s.db.QueryRowContext(ctx, `SELECT state, reason FROM asks WHERE id = ?`, askID).Scan(&state, &reason)
		if err != nil {
			return "", "", "", err
		}
		if state == "answered" {
			_ = s.db.QueryRowContext(ctx, `SELECT answer FROM answers WHERE ask_id = ?`, askID).Scan(&answer)
			return state, answer.String, reason, nil
		}
		if state == "expired" {
			return state, "", reason, nil
		}
		if !deadline.IsZero() && !s.clock.Now().Before(deadline) {
			now := s.clock.Now().UTC().Format(time.RFC3339Nano)
			_, _ = s.db.ExecContext(ctx, `UPDATE asks SET state='expired', reason='caller deadline expired', updated_at=? WHERE id=? AND state IN ('pending','escalated')`, now, askID)
			return "expired", "", "caller deadline expired", nil
		}
		select {
		case <-ctx.Done():
			return "", "", "", ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *Service) Retain(ctx context.Context, before time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM notifications WHERE created_at < ?`, before.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *Service) appendEvent(ctx context.Context, notificationID, state, reason, at string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO notification_events (notification_id, state, reason, created_at) VALUES (?, ?, ?, ?)`, notificationID, state, safeReasonString(reason), at)
	return err
}

func (s *Service) currentState(ctx context.Context, id string) (string, string, error) {
	var state, reason string
	err := s.db.QueryRowContext(ctx, `SELECT state, reason FROM notification_events WHERE notification_id = ? ORDER BY id DESC LIMIT 1`, id).Scan(&state, &reason)
	return state, reason, err
}

func (s *Service) listReceipts(ctx context.Context, id string) ([]Receipt, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, notification_id, channel, machine_id, provider_id, delivered_at FROM receipts WHERE notification_id = ? ORDER BY delivered_at ASC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Receipt
	for rows.Next() {
		var r Receipt
		if err := rows.Scan(&r.ID, &r.NotificationID, &r.Channel, &r.MachineID, &r.ProviderID, &r.DeliveredAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type rowScanner interface{ Scan(...any) error }

func (s *Service) scanNotification(row rowScanner) (Notification, error) {
	var n Notification
	err := row.Scan(&n.ID, &n.RequestedBy, &n.Title, &n.Body, &n.Urgency, &n.SensitivityLabel, &n.IdempotencyKey, &n.DedupeKey, &n.ScheduledAt, &n.DigestKey, &n.DigestWindowSeconds, &n.CreatedAt, &n.UpdatedAt)
	return n, err
}

func isTerminal(state string) bool {
	return state == "delivered" || state == "failed" || state == "unroutable" || state == "suppressed"
}

func safeReason(err error) string {
	if err == nil {
		return ""
	}
	return safeReasonString(err.Error())
}

func safeReasonString(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 500 {
		return value[:500]
	}
	return value
}

// DeliveryProjection is one notification with its delivery attempts, as the
// coverage checks of other scenarios read it. DedupeKey carries the producer's
// join key ("<event type>:<incident id>" for autoheal incidents).
type DeliveryProjection struct {
	NotificationID string                  `json:"notification_id"`
	RequestedBy    string                  `json:"requested_by"`
	DedupeKey      string                  `json:"dedupe_key"`
	IdempotencyKey string                  `json:"idempotency_key"`
	CreatedAt      string                  `json:"created_at"`
	Attempts       []DeliveryAttemptRecord `json:"attempts"`
}

// DeliveryAttemptRecord is one durable delivery attempt.
type DeliveryAttemptRecord struct {
	Channel   string `json:"channel"`
	MachineID string `json:"machine_id"`
	Attempt   int    `json:"attempt"`
	Outcome   string `json:"outcome"`
	Reason    string `json:"reason,omitempty"`
	CreatedAt string `json:"created_at"`
}

// ListDeliveryProjection returns the newest notifications whose dedupe key
// starts with prefix, each with every delivery attempt recorded for it. An
// unroutable notification appears with its single 'unroutable' attempt, so a
// reader can tell "never reached anyone" from "never entered delivery".
func (s *Service) ListDeliveryProjection(ctx context.Context, prefix string, limit int) ([]DeliveryProjection, error) {
	if limit <= 0 || limit > maxDeliveryProjection {
		limit = maxDeliveryProjection
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, requested_by, dedupe_key, idempotency_key, created_at FROM notifications WHERE dedupe_key LIKE ? ORDER BY created_at DESC LIMIT ?`, prefix+"%", limit)
	if err != nil {
		return nil, err
	}
	var out []DeliveryProjection
	for rows.Next() {
		var item DeliveryProjection
		if err := rows.Scan(&item.NotificationID, &item.RequestedBy, &item.DedupeKey, &item.IdempotencyKey, &item.CreatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for i := range out {
		attempts, err := s.db.QueryContext(ctx, `SELECT channel, machine_id, attempt_number, outcome, reason, created_at FROM delivery_attempts WHERE notification_id=? ORDER BY created_at, attempt_number`, out[i].NotificationID)
		if err != nil {
			return nil, err
		}
		for attempts.Next() {
			var record DeliveryAttemptRecord
			if err := attempts.Scan(&record.Channel, &record.MachineID, &record.Attempt, &record.Outcome, &record.Reason, &record.CreatedAt); err != nil {
				attempts.Close()
				return nil, err
			}
			out[i].Attempts = append(out[i].Attempts, record)
		}
		if err := attempts.Close(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

const maxDeliveryProjection = 200
