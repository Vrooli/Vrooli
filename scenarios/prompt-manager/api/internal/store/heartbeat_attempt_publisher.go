package store

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/database"
	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const heartbeatAttemptPublicationSchema = `
CREATE TABLE IF NOT EXISTS heartbeat_attempt_publications (
  event_id TEXT PRIMARY KEY,
  attempt_id TEXT NOT NULL,
  team_id TEXT NOT NULL,
  agent_id TEXT NOT NULL,
  phase TEXT NOT NULL,
  source_snapshot BLOB NOT NULL,
  envelope BLOB NOT NULL,
  envelope_sha256 TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  attempts INTEGER NOT NULL DEFAULT 0,
  next_attempt_at TEXT NOT NULL,
  lease_until TEXT NOT NULL DEFAULT '',
  last_error TEXT NOT NULL DEFAULT '',
  accepted_at TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS heartbeat_attempt_publications_due_idx
  ON heartbeat_attempt_publications(status, next_attempt_at, lease_until, created_at);
CREATE INDEX IF NOT EXISTS heartbeat_attempt_publications_receipt_idx
  ON heartbeat_attempt_publications(status, accepted_at, event_id);
`

const heartbeatPublicationTimestampLayout = "2006-01-02T15:04:05.000000000Z"

func heartbeatPublicationTimestamp(value time.Time) string {
	return value.UTC().Format(heartbeatPublicationTimestampLayout)
}

// HeartbeatAttemptPublicationSchema is registered with Prompt Manager's shared
// RoutedDB schema set so production and leased Test Genie pools own the same
// durable publication boundary.
func HeartbeatAttemptPublicationSchema() string { return heartbeatAttemptPublicationSchema }

// HeartbeatAttemptPublisher freezes each lifecycle fact in Prompt Manager's
// routed database before network publication. It intentionally owns only PM
// heartbeat attempts; it is not a general outbox framework.
type HeartbeatAttemptPublisher struct {
	db        *database.RoutedDB
	resolve   func(context.Context) (string, error)
	client    *http.Client
	wake      chan struct{}
	mu        sync.Mutex
	cancel    context.CancelFunc
	done      chan struct{}
	started   bool
	lastPrune time.Time
}

func NewHeartbeatAttemptPublisher(db *database.RoutedDB, resolve func(context.Context) (string, error)) *HeartbeatAttemptPublisher {
	return &HeartbeatAttemptPublisher{
		db: db, resolve: resolve, client: &http.Client{Timeout: 10 * time.Second},
		wake: make(chan struct{}, 1), done: make(chan struct{}),
	}
}

// Capture persists the exact event envelope with insert-if-absent semantics.
// Reusing an immutable fact key with changed terms fails closed.
func (p *HeartbeatAttemptPublisher) Capture(ctx context.Context, entry *HeartbeatAttempt) error {
	if p == nil || p.db == nil || entry == nil {
		return fmt.Errorf("heartbeat attempt publisher unavailable")
	}
	if strings.TrimSpace(entry.ID) == "" {
		if entry.RetainedOwnerState != "" {
			entry.ID = strings.TrimSpace(entry.OwnerObligationID)
		} else {
			entry.ID = fmt.Sprintf("heartbeat-attempt-%d", time.Now().UTC().UnixNano())
		}
	}
	snapshot, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("snapshot heartbeat attempt: %w", err)
	}
	logicalID, phase, err := heartbeatAttemptFactIdentity(entry)
	if err != nil {
		return err
	}
	eventID := heartbeatAttemptFactEventID(logicalID, phase)
	var existingSnapshot []byte
	var existingEnvelope []byte
	var existingDigest string
	err = p.db.QueryRowContext(ctx, `SELECT source_snapshot, envelope, envelope_sha256 FROM heartbeat_attempt_publications WHERE event_id=?`, eventID).Scan(&existingSnapshot, &existingEnvelope, &existingDigest)
	if err == nil {
		if !bytes.Equal(existingSnapshot, snapshot) {
			return fmt.Errorf("heartbeat attempt fact identity conflict for %s", eventID)
		}
		storedDigest := sha256.Sum256(existingEnvelope)
		if existingDigest != hex.EncodeToString(storedDigest[:]) {
			return fmt.Errorf("captured heartbeat attempt envelope is corrupt for %s", eventID)
		}
		if !database.IsTestMode(ctx) {
			p.signal()
		}
		return nil
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("read captured heartbeat attempt publication: %w", err)
	}
	body, eventID, err := heartbeatAttemptEnvelope(entry)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(body)
	digestText := hex.EncodeToString(digest[:])
	now := heartbeatPublicationTimestamp(time.Now())
	_, err = p.db.ExecContext(ctx, `INSERT INTO heartbeat_attempt_publications
(event_id, attempt_id, team_id, agent_id, phase, source_snapshot, envelope, envelope_sha256, status, attempts, next_attempt_at, created_at)
VALUES(?,?,?,?,?,?,?,?, 'pending',0,?,?) ON CONFLICT(event_id) DO NOTHING`,
		eventID, logicalID, entry.TeamID, entry.AgentID, phase, snapshot, body, digestText, now, now)
	if err != nil {
		return fmt.Errorf("capture heartbeat attempt publication: %w", err)
	}
	var persistedSnapshot, existing []byte
	if err := p.db.QueryRowContext(ctx, `SELECT source_snapshot, envelope, envelope_sha256 FROM heartbeat_attempt_publications WHERE event_id=?`, eventID).Scan(&persistedSnapshot, &existing, &existingDigest); err != nil {
		return fmt.Errorf("read captured heartbeat attempt publication: %w", err)
	}
	if !bytes.Equal(persistedSnapshot, snapshot) {
		return fmt.Errorf("heartbeat attempt fact identity conflict for %s", eventID)
	}
	storedDigest := sha256.Sum256(existing)
	if existingDigest != hex.EncodeToString(storedDigest[:]) {
		return fmt.Errorf("captured heartbeat attempt envelope is corrupt for %s", eventID)
	}
	if !database.IsTestMode(ctx) {
		p.signal()
	}
	return nil
}

func (p *HeartbeatAttemptPublisher) signal() {
	select {
	case p.wake <- struct{}{}:
	default:
	}
}

// Start launches one bounded retry worker. Startup reconciliation is the first
// pass, so pending rows survive and resume after process restart.
func (p *HeartbeatAttemptPublisher) Start(parent context.Context) error {
	if p == nil || p.db == nil {
		return fmt.Errorf("heartbeat attempt publisher unavailable")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.started {
		return nil
	}
	ctx, cancel := context.WithCancel(parent)
	p.cancel = cancel
	p.started = true
	go p.run(ctx)
	return nil
}

// Shutdown cancels the worker and waits within the caller's cleanup budget.
func (p *HeartbeatAttemptPublisher) Shutdown(ctx context.Context) error {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	if !p.started {
		p.mu.Unlock()
		return nil
	}
	cancel := p.cancel
	done := p.done
	p.mu.Unlock()
	cancel()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *HeartbeatAttemptPublisher) run(ctx context.Context) {
	defer close(p.done)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		p.drain(ctx, 8)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-p.wake:
		}
	}
}

func (p *HeartbeatAttemptPublisher) drain(ctx context.Context, limit int) {
	p.pruneAcceptedReceipts(ctx)
	for i := 0; i < limit && ctx.Err() == nil; i++ {
		row, err := p.claim(ctx)
		if err != nil || row == nil {
			if err != nil {
				logHeartbeatPublicationError("claim", err)
			}
			return
		}
		p.publish(ctx, row)
	}
}

func (p *HeartbeatAttemptPublisher) pruneAcceptedReceipts(ctx context.Context) {
	now := time.Now().UTC()
	p.mu.Lock()
	if !p.lastPrune.IsZero() && now.Sub(p.lastPrune) < time.Hour {
		p.mu.Unlock()
		return
	}
	p.mu.Unlock()
	_, err := p.db.ExecContext(ctx, `DELETE FROM heartbeat_attempt_publications WHERE event_id IN (
  SELECT event_id FROM heartbeat_attempt_publications
  WHERE status='accepted' AND accepted_at<? ORDER BY accepted_at, event_id LIMIT 100
	)`, heartbeatPublicationTimestamp(now.Add(-30*24*time.Hour)))
	if err != nil {
		logHeartbeatPublicationError("prune receipts", err)
		return
	}
	p.mu.Lock()
	p.lastPrune = now
	p.mu.Unlock()
}

type heartbeatPublication struct {
	eventID    string
	body       []byte
	digest     string
	attempt    int
	leaseToken string
}

func (p *HeartbeatAttemptPublisher) claim(ctx context.Context) (*heartbeatPublication, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	nowText := heartbeatPublicationTimestamp(now)
	var row heartbeatPublication
	err = tx.QueryRowContext(ctx, `SELECT event_id, envelope, envelope_sha256, attempts FROM heartbeat_attempt_publications
WHERE status='pending' AND next_attempt_at<=? AND (lease_until='' OR lease_until<=?)
ORDER BY created_at, event_id LIMIT 1`, nowText, nowText).Scan(&row.eventID, &row.body, &row.digest, &row.attempt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	lease := heartbeatPublicationTimestamp(now.Add(30 * time.Second))
	result, err := tx.ExecContext(ctx, `UPDATE heartbeat_attempt_publications SET lease_until=?
WHERE event_id=? AND status='pending' AND (lease_until='' OR lease_until<=?)`, lease, row.eventID, nowText)
	if err != nil {
		return nil, err
	}
	n, err := result.RowsAffected()
	if err != nil || n != 1 {
		if err != nil {
			return nil, err
		}
		return nil, nil
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	row.leaseToken = lease
	return &row, nil
}

func (p *HeartbeatAttemptPublisher) publish(ctx context.Context, row *heartbeatPublication) {
	digest := sha256.Sum256(row.body)
	if row.digest != hex.EncodeToString(digest[:]) {
		p.retry(ctx, row, fmt.Errorf("frozen envelope digest mismatch for %s", row.eventID))
		return
	}
	endpoint := ""
	var err error
	if p.resolve != nil {
		endpoint, err = p.resolve(ctx)
	}
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if err == nil && endpoint == "" {
		err = fmt.Errorf("vrooli-events endpoint unavailable")
	}
	if err == nil {
		var req *http.Request
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"/api/v1/events", bytes.NewReader(row.body))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
			var resp *http.Response
			resp, err = p.client.Do(req)
			if err == nil {
				_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
				_ = resp.Body.Close()
				if resp.StatusCode == http.StatusAccepted {
					_, err = p.db.ExecContext(ctx, `UPDATE heartbeat_attempt_publications SET status='accepted', accepted_at=?, lease_until='', last_error=''
WHERE event_id=? AND status='pending' AND lease_until=? AND lease_until>?`, heartbeatPublicationTimestamp(time.Now()), row.eventID, row.leaseToken, heartbeatPublicationTimestamp(time.Now()))
					if err != nil {
						logHeartbeatPublicationError("record accepted", err)
					}
					return
				}
				if resp.StatusCode == http.StatusConflict {
					verified, verifyErr := p.verifyExisting(ctx, endpoint, row.eventID, row.body)
					if verifyErr == nil && verified {
						_, err = p.db.ExecContext(ctx, `UPDATE heartbeat_attempt_publications SET status='accepted', accepted_at=?, lease_until='', last_error=''
WHERE event_id=? AND status='pending' AND lease_until=? AND lease_until>?`, heartbeatPublicationTimestamp(time.Now()), row.eventID, row.leaseToken, heartbeatPublicationTimestamp(time.Now()))
						if err != nil {
							logHeartbeatPublicationError("record verified duplicate", err)
						}
						return
					}
					if verifyErr != nil {
						err = fmt.Errorf("verify duplicate event %s: %w", row.eventID, verifyErr)
					} else {
						err = fmt.Errorf("event id %s exists with different terms", row.eventID)
					}
					p.retry(ctx, row, err)
					return
				}
				err = fmt.Errorf("vrooli-events returned %s", resp.Status)
			}
		}
	}
	p.retry(ctx, row, err)
}

func (p *HeartbeatAttemptPublisher) verifyExisting(ctx context.Context, endpoint, eventID string, expected []byte) (bool, error) {
	requestURL := endpoint + "/api/v1/events?event_id=" + url.QueryEscape(eventID) + "&limit=2"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return false, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("read existing event returned %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return false, err
	}
	var rows []json.RawMessage
	if err := json.Unmarshal(body, &rows); err != nil {
		return false, err
	}
	if len(rows) != 1 {
		return false, nil
	}
	var actual, wanted domain.EventEnvelope
	if err := protojson.Unmarshal(rows[0], &actual); err != nil {
		return false, err
	}
	if err := protojson.Unmarshal(expected, &wanted); err != nil {
		return false, err
	}
	return proto.Equal(&actual, &wanted), nil
}

func (p *HeartbeatAttemptPublisher) retry(ctx context.Context, row *heartbeatPublication, publishErr error) {
	attempt := row.attempt + 1
	shift := attempt - 1
	if shift > 8 {
		shift = 8
	}
	delay := time.Second << shift
	if delay > 5*time.Minute {
		delay = 5 * time.Minute
	}
	message := publishErr.Error()
	if len(message) > 512 {
		message = message[:512]
	}
	_, updateErr := p.db.ExecContext(ctx, `UPDATE heartbeat_attempt_publications SET attempts=?, next_attempt_at=?, lease_until='', last_error=?
WHERE event_id=? AND status='pending' AND lease_until=? AND lease_until>?`, attempt, heartbeatPublicationTimestamp(time.Now().Add(delay)), message, row.eventID, row.leaseToken, heartbeatPublicationTimestamp(time.Now()))
	if updateErr != nil {
		logHeartbeatPublicationError("record retry", updateErr)
	}
}

// heartbeatAttemptEnvelope freezes one lifecycle fact. The logical attempt ID
// remains in the payload while the Events ID is unique per immutable phase.
func heartbeatAttemptEnvelope(entry *HeartbeatAttempt) ([]byte, string, error) {
	if strings.TrimSpace(entry.ID) == "" {
		if entry.RetainedOwnerState != "" {
			entry.ID = strings.TrimSpace(entry.OwnerObligationID)
		} else {
			entry.ID = fmt.Sprintf("heartbeat-attempt-%d", time.Now().UTC().UnixNano())
		}
	}
	logicalID, phase, err := heartbeatAttemptFactIdentity(entry)
	if err != nil {
		return nil, "", err
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return nil, "", fmt.Errorf("encode heartbeat attempt: %w", err)
	}
	var values map[string]any
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, "", fmt.Errorf("prepare heartbeat attempt event: %w", err)
	}
	structData, err := structpb.NewStruct(values)
	if err != nil {
		return nil, "", fmt.Errorf("pack heartbeat attempt payload: %w", err)
	}
	packed, err := anypb.New(structData)
	if err != nil {
		return nil, "", fmt.Errorf("pack heartbeat attempt payload: %w", err)
	}
	packed.Value, err = proto.MarshalOptions{Deterministic: true}.Marshal(structData)
	if err != nil {
		return nil, "", fmt.Errorf("deterministically pack heartbeat attempt payload: %w", err)
	}
	occurredAtText := strings.TrimSpace(entry.EndedAt)
	if entry.RetainedOwnerState != "" {
		occurredAtText = strings.TrimSpace(entry.ObservedAt)
	} else if occurredAtText == "" {
		occurredAtText = strings.TrimSpace(entry.StartedAt)
	}
	occurredAt, err := time.Parse(time.RFC3339Nano, occurredAtText)
	if err != nil {
		return nil, "", fmt.Errorf("heartbeat fact %s has no stable event timestamp: %w", logicalID, err)
	}
	eventType := heartbeatAttemptEventType
	operation := "heartbeat-attempt"
	if entry.RetainedOwnerState != "" {
		eventType = heartbeatOwnerStateEventType
		operation = "heartbeat-owner-state"
	}
	env := &domain.EventEnvelope{
		EventId: heartbeatAttemptFactEventID(logicalID, phase), EventType: eventType, OccurredAt: timestamppb.New(occurredAt.UTC()),
		Source: &domain.EventSource{Scenario: "prompt-manager", ActorKind: "system"},
		Target: &domain.EventTarget{Scenario: "prompt-manager", Operation: operation, Protocol: "http"},
		Data:   packed,
	}
	if entry.RunID != "" || entry.TaskID != "" {
		env.Correlation = &domain.EventCorrelation{AgentRunId: entry.RunID, TaskId: entry.TaskID}
	}
	body, err := protojson.Marshal(env)
	if err != nil {
		return nil, "", fmt.Errorf("encode heartbeat attempt event envelope: %w", err)
	}
	return body, env.GetEventId(), nil
}

func heartbeatAttemptFactIdentity(entry *HeartbeatAttempt) (logicalID, phase string, err error) {
	if entry == nil {
		return "", "", fmt.Errorf("heartbeat fact is nil")
	}
	phase = strings.TrimSpace(entry.Phase)
	if entry.RetainedOwnerState != "" {
		logicalID = strings.TrimSpace(entry.OwnerObligationID)
		if logicalID == "" {
			return "", "", fmt.Errorf("retained owner-state fact requires an owner obligation identity")
		}
		switch entry.RetainedOwnerState {
		case HeartbeatOwnerStateDispatchUncertain, HeartbeatOwnerStateOwnerUnreachable,
			HeartbeatOwnerStateNeedsReview, HeartbeatOwnerStateWatcherUnavailable,
			HeartbeatOwnerStatePreparationInterrupted:
		default:
			return "", "", fmt.Errorf("unsupported retained owner state %q", entry.RetainedOwnerState)
		}
		if entry.RetainedOwnerState == HeartbeatOwnerStatePreparationInterrupted &&
			(strings.TrimSpace(entry.TaskID) != "" || strings.TrimSpace(entry.RunID) != "" || strings.TrimSpace(entry.StartedAt) != "") {
			return "", "", fmt.Errorf("preparation-interrupted owner state cannot claim task, run, or owner start terms")
		}
		if phase == "" || !strings.HasPrefix(phase, "retained_owner_state:") {
			return "", "", fmt.Errorf("retained owner-state fact requires a distinct retained_owner_state phase")
		}
		if strings.TrimSpace(entry.ObservedAt) == "" {
			return "", "", fmt.Errorf("retained owner-state fact requires its persisted observation timestamp")
		}
		return logicalID, phase, nil
	}
	logicalID = strings.TrimSpace(entry.ID)
	if logicalID == "" {
		return "", "", fmt.Errorf("heartbeat attempt requires a logical attempt identity")
	}
	if phase == "" {
		return "", "", fmt.Errorf("heartbeat attempt requires a lifecycle phase")
	}
	return logicalID, phase, nil
}

func logHeartbeatPublicationError(stage string, err error) {
	if err != nil {
		// Keep logging bounded and avoid copying event payloads or secrets.
		message := err.Error()
		if len(message) > 512 {
			message = message[:512]
		}
		log.Printf("heartbeat attempt publication %s: %s", stage, message)
	}
}
