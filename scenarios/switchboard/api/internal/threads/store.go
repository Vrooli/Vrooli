package threads

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"switchboard/internal/channels"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
type Thread struct {
	ID, ChannelID, ThreadKey  string
	IsGroup                   bool
	Position                  int
	TurnBudget, SpendCapCents int64
}
type Store struct{ db SQLExecutor }

func NewStore(db SQLExecutor) *Store { return &Store{db: db} }
func (s *Store) Upsert(ctx context.Context, e channels.Envelope, group bool) (Thread, error) {
	var t Thread
	err := s.db.QueryRowContext(ctx, `SELECT id,channel_id,thread_key,is_group,position,turn_budget,spend_cap_cents FROM switchboard_threads WHERE channel_id=? AND thread_key=?`, e.ChannelID, e.ThreadKey).Scan(&t.ID, &t.ChannelID, &t.ThreadKey, &t.IsGroup, &t.Position, &t.TurnBudget, &t.SpendCapCents)
	if errors.Is(err, sql.ErrNoRows) {
		t = Thread{ID: uuid.NewString(), ChannelID: e.ChannelID, ThreadKey: e.ThreadKey, IsGroup: group, TurnBudget: 20}
		_, err = s.db.ExecContext(ctx, `INSERT INTO switchboard_threads(id,channel_id,thread_key,is_group,position,turn_budget,spend_cap_cents,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?)`, t.ID, t.ChannelID, t.ThreadKey, group, 0, t.TurnBudget, 0, time.Now().UTC().Format(time.RFC3339Nano), time.Now().UTC().Format(time.RFC3339Nano))
		return t, err
	}
	if err != nil {
		return Thread{}, fmt.Errorf("find thread: %w", err)
	}
	return t, nil
}

func (s *Store) Append(ctx context.Context, thread Thread, e channels.Envelope) (bool, error) {
	media, _ := json.Marshal(nonNilMedia(e.Media))
	received := e.ReceivedAt
	if received.IsZero() {
		received = time.Now()
	}
	result, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO switchboard_messages(thread_id,channel_id,remote_id,author_kind,sender_address,text,reply_to_remote_id,received_at,media_json) VALUES(?,?,?,?,?,?,?,?,?)`, thread.ID, e.ChannelID, e.RemoteMessageID, e.AuthorKind, e.SenderAddress, e.Text, e.ReplyToRemoteID, received.UTC().Format(time.RFC3339Nano), string(media))
	if err != nil {
		return false, err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return false, nil
	}
	_, err = s.db.ExecContext(ctx, `UPDATE switchboard_threads SET position=position+1,updated_at=? WHERE id=?`, time.Now().UTC().Format(time.RFC3339Nano), thread.ID)
	return true, err
}

// AllowTurn atomically admits one turn against the thread's hourly and spend
// ceilings. A small reservation is charged before execution; this is
// deliberately conservative because provider cost is not known until the run
// completes.
func (s *Store) AllowTurn(ctx context.Context, thread Thread, now time.Time, reservationCents int64) (allowed, notifyOwner bool, err error) {
	nowText := now.UTC().Format(time.RFC3339Nano)
	_, err = s.db.ExecContext(ctx, `INSERT OR IGNORE INTO switchboard_thread_budget(thread_id,window_started_at,used,spent_cents,owner_notified) VALUES(?,?,0,0,0)`, thread.ID, nowText)
	if err != nil {
		return false, false, err
	}
	_, err = s.db.ExecContext(ctx, `UPDATE switchboard_thread_budget SET window_started_at=?,used=0,spent_cents=0,owner_notified=0 WHERE thread_id=? AND window_started_at <= ?`, nowText, thread.ID, now.Add(-time.Hour).UTC().Format(time.RFC3339Nano))
	if err != nil {
		return false, false, err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE switchboard_thread_budget SET used=used+1,spent_cents=spent_cents+? WHERE thread_id=? AND (? <= 0 OR used < ?) AND (? <= 0 OR spent_cents+? <= ?)`, reservationCents, thread.ID, thread.TurnBudget, thread.TurnBudget, thread.SpendCapCents, reservationCents, thread.SpendCapCents)
	if err != nil {
		return false, false, err
	}
	rows, _ := result.RowsAffected()
	if rows == 1 {
		return true, false, nil
	}
	result, err = s.db.ExecContext(ctx, `UPDATE switchboard_thread_budget SET owner_notified=1 WHERE thread_id=? AND owner_notified=0`, thread.ID)
	if err != nil {
		return false, false, err
	}
	notified, _ := result.RowsAffected()
	return false, notified == 1, nil
}

func (s *Store) RunID(ctx context.Context, e channels.Envelope) (string, error) {
	var runID string
	err := s.db.QueryRowContext(ctx, `SELECT r.run_id FROM switchboard_thread_runs r JOIN switchboard_threads t ON t.id=r.thread_id WHERE t.channel_id=? AND t.thread_key=?`, e.ChannelID, e.ThreadKey).Scan(&runID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("find thread run: %w", err)
	}
	return runID, nil
}

func (s *Store) SetRunID(ctx context.Context, e channels.Envelope, runID string) error {
	thread, err := s.Upsert(ctx, e, false)
	if err != nil {
		return err
	}
	if strings.TrimSpace(runID) == "" {
		return fmt.Errorf("run id is required")
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO switchboard_thread_runs(thread_id,run_id,updated_at) VALUES(?,?,?) ON CONFLICT(thread_id) DO UPDATE SET run_id=excluded.run_id,updated_at=excluded.updated_at`, thread.ID, runID, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func nonNilMedia(in []channels.Media) []channels.Media {
	if in == nil {
		return []channels.Media{}
	}
	return in
}

// AppendOutbound records an agent-authored message on the thread that is
// about to receive it, so the transcript survives a reload and the loop
// breaker can see who wrote what. The remote id is minted here because the
// adapter has not assigned one yet.
func (s *Store) AppendOutbound(ctx context.Context, out channels.Outbound, agentID string) error {
	thread, err := s.Upsert(ctx, channels.Envelope{ChannelID: out.ChannelID, ThreadKey: out.ThreadKey}, false)
	if err != nil {
		return err
	}
	_, err = s.Append(ctx, thread, channels.Envelope{
		ChannelID: out.ChannelID, ThreadKey: out.ThreadKey, RemoteMessageID: "agent-" + uuid.NewString(),
		SenderAddress: agentID, AuthorKind: channels.AuthorAgent, Text: out.Text, Media: out.Media,
		ReplyToRemoteID: out.ReplyToRemoteID, ReceivedAt: time.Now(),
	})
	return err
}

// TurnEvent is one admission decision for one inbound message.
type TurnEvent struct {
	ID            string `json:"id"`
	ThreadID      string `json:"thread_id"`
	AgentID       string `json:"agent_id"`
	ChannelID     string `json:"channel_id"`
	SenderAddress string `json:"sender_address"`
	Outcome       string `json:"outcome"`
	Reason        string `json:"reason"`
	CreatedAt     string `json:"created_at"`
}

// RecordEvent persists the outcome of one turn admission.
func (s *Store) RecordEvent(ctx context.Context, e channels.Envelope, agentID, outcome, reason string) error {
	thread, err := s.Upsert(ctx, e, e.Group)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO switchboard_turn_events(id,thread_id,agent_id,channel_id,sender_address,outcome,reason,created_at) VALUES(?,?,?,?,?,?,?,?)`,
		uuid.NewString(), thread.ID, agentID, e.ChannelID, e.SenderAddress, outcome, reason, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}
