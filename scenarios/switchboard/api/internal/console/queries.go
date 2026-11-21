// Package console is the read model behind the operator console. It joins the
// per-domain tables into the shapes the pages render and owns no writes.
package console

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"switchboard/internal/channels"
	"switchboard/internal/contacts"
	"switchboard/internal/gates"
	"switchboard/internal/trust"
)

type DB interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// ChannelRef is how a channel appears next to anything else: id, display name
// and accent, so consumers never map either by id.
type ChannelRef struct {
	ChannelID          string `json:"channel_id"`
	ChannelDisplayName string `json:"channel_display_name"`
	ChannelAccent      string `json:"channel_accent"`
}

type ThreadBudget struct {
	ThreadID        string `json:"thread_id"`
	ChannelID       string `json:"channel_id"`
	ThreadKey       string `json:"thread_key"`
	AgentID         string `json:"agent_id"`
	TurnBudget      int64  `json:"turn_budget"`
	Used            int64  `json:"used"`
	SpendCapCents   int64  `json:"spend_cap_cents"`
	SpentCents      int64  `json:"spent_cents"`
	WindowStartedAt string `json:"window_started_at"`
	Exhausted       bool   `json:"exhausted"`
}

// Pressure reports whether a budget is close enough to its ceiling to warrant
// attention (70% or exhausted).
func (b ThreadBudget) Pressure() bool {
	if b.Exhausted {
		return true
	}
	if b.TurnBudget > 0 && float64(b.Used)/float64(b.TurnBudget) >= 0.7 {
		return true
	}
	if b.SpendCapCents > 0 && float64(b.SpentCents)/float64(b.SpendCapCents) >= 0.7 {
		return true
	}
	return false
}

type LastMessage struct {
	Text          string `json:"text"`
	AuthorKind    string `json:"author_kind"`
	SenderAddress string `json:"sender_address"`
	DisplayName   string `json:"display_name"`
	ReceivedAt    string `json:"received_at"`
}

type Thread struct {
	ID string `json:"id"`
	ChannelRef
	ThreadKey        string       `json:"thread_key"`
	IsGroup          bool         `json:"is_group"`
	AgentID          string       `json:"agent_id"`
	AgentDisplayName string       `json:"agent_display_name"`
	CeilingTier      string       `json:"ceiling_tier"`
	ParticipantCount int64        `json:"participant_count"`
	MessageCount     int64        `json:"message_count"`
	PendingGates     int64        `json:"pending_gates"`
	LastMessage      *LastMessage `json:"last_message"`
	Budget           ThreadBudget `json:"budget"`
	CreatedAt        string       `json:"created_at"`
	UpdatedAt        string       `json:"updated_at"`
}

type Message struct {
	ID              int64            `json:"id"`
	RemoteID        string           `json:"remote_id"`
	AuthorKind      string           `json:"author_kind"`
	SenderAddress   string           `json:"sender_address"`
	DisplayName     string           `json:"display_name"`
	Text            string           `json:"text"`
	ReplyToRemoteID string           `json:"reply_to_remote_id"`
	ReceivedAt      string           `json:"received_at"`
	Media           []channels.Media `json:"media"`
}

type Refusal struct {
	ThreadID string `json:"thread_id"`
	ChannelRef
	ThreadKey     string `json:"thread_key"`
	SenderAddress string `json:"sender_address"`
	AgentID       string `json:"agent_id"`
	Reason        string `json:"reason"`
	At            string `json:"at"`
}

type ActivityEntry struct {
	Kind      string `json:"kind"`
	ThreadID  string `json:"thread_id"`
	ChannelID string `json:"channel_id"`
	Text      string `json:"text"`
	Reason    string `json:"reason"`
	At        string `json:"at"`
}

type Activity struct {
	Turns24h    int64 `json:"turns_24h"`
	Refusals24h int64 `json:"refusals_24h"`
	Threads     int64 `json:"threads"`
}

// Queries is the console read model. Names resolves agent display names by
// reference and may be nil.
type Queries struct {
	DB       DB
	Registry *channels.Registry
	Contacts *contacts.Store
	Now      func() time.Time
}

func (q Queries) channel(id string) ChannelRef {
	ref := ChannelRef{ChannelID: id, ChannelDisplayName: id}
	if q.Registry != nil {
		if d, ok := q.Registry.Get(id); ok {
			ref.ChannelDisplayName, ref.ChannelAccent = d.DisplayName, d.Accent
		}
	}
	return ref
}

func (q Queries) now() time.Time {
	if q.Now != nil {
		return q.Now()
	}
	return time.Now()
}

const threadSelect = `SELECT t.id,t.channel_id,t.thread_key,t.is_group,t.turn_budget,t.spend_cap_cents,t.created_at,t.updated_at,
 COALESCE(b.used,0),COALESCE(b.spent_cents,0),COALESCE(b.window_started_at,''),
 (SELECT COUNT(*) FROM switchboard_messages m WHERE m.thread_id=t.id),
 (SELECT COUNT(*) FROM switchboard_capability_gates g WHERE g.thread_id=t.id AND g.status='pending' AND g.expires_at > ?),
 COALESCE((SELECT bd.agent_id FROM switchboard_bindings bd WHERE bd.channel_id=t.channel_id AND bd.thread_key=t.thread_key ORDER BY bd.created_at LIMIT 1),'')
 FROM switchboard_threads t LEFT JOIN switchboard_thread_budget b ON b.thread_id=t.id `

// scanThread reads one row. It issues no further queries because the process
// runs SQLite on a single connection: a nested query while the cursor is open
// would deadlock. Enrichment happens in enrich, after the cursor is closed.
func (q Queries) scanThread(rows interface{ Scan(...any) error }) (Thread, error) {
	var t Thread
	var window string
	if err := rows.Scan(&t.ID, &t.ChannelID, &t.ThreadKey, &t.IsGroup, &t.Budget.TurnBudget, &t.Budget.SpendCapCents, &t.CreatedAt, &t.UpdatedAt,
		&t.Budget.Used, &t.Budget.SpentCents, &window, &t.MessageCount, &t.PendingGates, &t.AgentID); err != nil {
		return Thread{}, err
	}
	t.ChannelRef = q.channel(t.ChannelID)
	t.Budget.ThreadID, t.Budget.ChannelID, t.Budget.ThreadKey, t.Budget.AgentID, t.Budget.WindowStartedAt = t.ID, t.ChannelID, t.ThreadKey, t.AgentID, window
	// The hourly window resets lazily on the next turn; report it as fresh once it has aged out.
	if started, err := time.Parse(time.RFC3339Nano, window); err == nil && q.now().Sub(started) >= time.Hour {
		t.Budget.Used, t.Budget.SpentCents = 0, 0
	}
	t.Budget.Exhausted = (t.Budget.TurnBudget > 0 && t.Budget.Used >= t.Budget.TurnBudget) || (t.Budget.SpendCapCents > 0 && t.Budget.SpentCents >= t.Budget.SpendCapCents)
	return t, nil
}

func (q Queries) enrich(ctx context.Context, t Thread) (Thread, error) {
	if q.Contacts != nil {
		roster, err := q.Contacts.Roster(ctx, t.ID)
		if err != nil {
			return Thread{}, err
		}
		t.ParticipantCount = int64(len(roster))
		t.CeilingTier = contacts.Ceiling(roster).String()
	} else {
		t.CeilingTier = trust.Stranger.String()
	}
	var last LastMessage
	err := q.DB.QueryRowContext(ctx, `SELECT m.text,m.author_kind,m.sender_address,m.received_at,
 COALESCE((SELECT c.display_name FROM switchboard_contacts c WHERE c.channel_id=m.channel_id AND c.address=m.sender_address),'')
 FROM switchboard_messages m WHERE m.thread_id=? ORDER BY m.received_at DESC, m.id DESC LIMIT 1`, t.ID).Scan(&last.Text, &last.AuthorKind, &last.SenderAddress, &last.ReceivedAt, &last.DisplayName)
	if err == nil {
		t.LastMessage = &last
	} else if !errors.Is(err, sql.ErrNoRows) {
		return Thread{}, err
	}
	return t, nil
}

func (q Queries) ListThreads(ctx context.Context) ([]Thread, error) {
	rows, err := q.DB.QueryContext(ctx, threadSelect+`ORDER BY t.updated_at DESC`, q.now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("list threads: %w", err)
	}
	out := make([]Thread, 0)
	for rows.Next() {
		t, err := q.scanThread(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, t)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for i := range out {
		if out[i], err = q.enrich(ctx, out[i]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

var ErrNotFound = errors.New("thread not found")

func (q Queries) GetThread(ctx context.Context, id string) (Thread, error) {
	row := q.DB.QueryRowContext(ctx, threadSelect+`WHERE t.id=?`, q.now().UTC().Format(time.RFC3339Nano), id)
	t, err := q.scanThread(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Thread{}, ErrNotFound
	}
	if err != nil {
		return Thread{}, err
	}
	return q.enrich(ctx, t)
}

func (q Queries) Messages(ctx context.Context, threadID string) ([]Message, error) {
	rows, err := q.DB.QueryContext(ctx, `SELECT m.id,m.remote_id,m.author_kind,m.sender_address,m.text,m.reply_to_remote_id,m.received_at,m.media_json,
 COALESCE((SELECT c.display_name FROM switchboard_contacts c WHERE c.channel_id=m.channel_id AND c.address=m.sender_address),'')
 FROM switchboard_messages m WHERE m.thread_id=? ORDER BY m.received_at, m.id`, threadID)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()
	out := make([]Message, 0)
	for rows.Next() {
		var m Message
		var media string
		if err := rows.Scan(&m.ID, &m.RemoteID, &m.AuthorKind, &m.SenderAddress, &m.Text, &m.ReplyToRemoteID, &m.ReceivedAt, &media, &m.DisplayName); err != nil {
			return nil, err
		}
		m.Media = []channels.Media{}
		_ = json.Unmarshal([]byte(media), &m.Media)
		if m.Media == nil {
			m.Media = []channels.Media{}
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (q Queries) RunID(ctx context.Context, threadID string) (string, error) {
	var runID string
	err := q.DB.QueryRowContext(ctx, `SELECT run_id FROM switchboard_thread_runs WHERE thread_id=?`, threadID).Scan(&runID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return runID, err
}

// Gates lists gates by status, newest first. An empty status lists all.
func (q Queries) Gates(ctx context.Context, status, threadID string) ([]gates.Gate, error) {
	query := `SELECT id,thread_id,owner_id,scope,withheld,unblock,created_at,expires_at,status,grant_once FROM switchboard_capability_gates WHERE 1=1`
	args := []any{}
	if status != "" {
		query += ` AND status=?`
		args = append(args, status)
	}
	if threadID != "" {
		query += ` AND thread_id=?`
		args = append(args, threadID)
	}
	rows, err := q.DB.QueryContext(ctx, query+` ORDER BY created_at DESC`, args...)
	if err != nil {
		return nil, fmt.Errorf("list gates: %w", err)
	}
	defer rows.Close()
	out := make([]gates.Gate, 0)
	now := q.now()
	for rows.Next() {
		var g gates.Gate
		var created, expires string
		if err := rows.Scan(&g.ID, &g.ThreadID, &g.OwnerID, &g.Scope, &g.Withheld, &g.Unblock, &created, &expires, &g.Status, &g.GrantOnce); err != nil {
			return nil, err
		}
		g.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		g.ExpiresAt, _ = time.Parse(time.RFC3339Nano, expires)
		if g.Status == gates.Pending && !now.Before(g.ExpiresAt) {
			g.Status = gates.Expired
			if status == string(gates.Pending) {
				continue
			}
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (q Queries) Refusals(ctx context.Context, limit int) ([]Refusal, error) {
	rows, err := q.DB.QueryContext(ctx, `SELECT e.thread_id,e.channel_id,COALESCE(t.thread_key,''),e.sender_address,e.agent_id,e.reason,e.created_at
 FROM switchboard_turn_events e LEFT JOIN switchboard_threads t ON t.id=e.thread_id WHERE e.outcome='refused' ORDER BY e.created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list refusals: %w", err)
	}
	defer rows.Close()
	out := make([]Refusal, 0)
	for rows.Next() {
		var r Refusal
		if err := rows.Scan(&r.ThreadID, &r.ChannelID, &r.ThreadKey, &r.SenderAddress, &r.AgentID, &r.Reason, &r.At); err != nil {
			return nil, err
		}
		r.ChannelRef = q.channel(r.ChannelID)
		out = append(out, r)
	}
	return out, rows.Err()
}

// ChannelCounts returns bindings and threads per channel id.
func (q Queries) ChannelCounts(ctx context.Context) (bindings, threads map[string]int64, err error) {
	bindings, threads = map[string]int64{}, map[string]int64{}
	rows, err := q.DB.QueryContext(ctx, `SELECT channel_id, COUNT(*) FROM switchboard_bindings GROUP BY channel_id`)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var id string
		var n int64
		if err := rows.Scan(&id, &n); err != nil {
			rows.Close()
			return nil, nil, err
		}
		bindings[id] = n
	}
	rows.Close()
	rows, err = q.DB.QueryContext(ctx, `SELECT channel_id, COUNT(*) FROM switchboard_threads GROUP BY channel_id`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var n int64
		if err := rows.Scan(&id, &n); err != nil {
			return nil, nil, err
		}
		threads[id] = n
	}
	return bindings, threads, rows.Err()
}

func (q Queries) AgentActivity(ctx context.Context, agentID string) (Activity, error) {
	var a Activity
	since := q.now().Add(-24 * time.Hour).UTC().Format(time.RFC3339Nano)
	if err := q.DB.QueryRowContext(ctx, `SELECT
 (SELECT COUNT(*) FROM switchboard_turn_events WHERE agent_id=? AND outcome='accepted' AND created_at>=?),
 (SELECT COUNT(*) FROM switchboard_turn_events WHERE agent_id=? AND outcome='refused' AND created_at>=?),
 (SELECT COUNT(DISTINCT t.id) FROM switchboard_threads t JOIN switchboard_bindings b ON b.channel_id=t.channel_id AND b.thread_key=t.thread_key WHERE b.agent_id=?)`,
		agentID, since, agentID, since, agentID).Scan(&a.Turns24h, &a.Refusals24h, &a.Threads); err != nil {
		return Activity{}, fmt.Errorf("agent activity: %w", err)
	}
	return a, nil
}

func (q Queries) AgentActivityLog(ctx context.Context, agentID string, limit int) ([]ActivityEntry, error) {
	rows, err := q.DB.QueryContext(ctx, `SELECT e.outcome,e.thread_id,e.channel_id,e.reason,e.created_at,
 COALESCE((SELECT m.text FROM switchboard_messages m WHERE m.thread_id=e.thread_id AND m.sender_address=e.sender_address AND m.received_at<=e.created_at ORDER BY m.received_at DESC LIMIT 1),'')
 FROM switchboard_turn_events e WHERE e.agent_id=? ORDER BY e.created_at DESC LIMIT ?`, agentID, limit)
	if err != nil {
		return nil, fmt.Errorf("agent activity log: %w", err)
	}
	out := make([]ActivityEntry, 0)
	for rows.Next() {
		var e ActivityEntry
		var outcome string
		if err := rows.Scan(&outcome, &e.ThreadID, &e.ChannelID, &e.Reason, &e.At, &e.Text); err != nil {
			return nil, err
		}
		switch outcome {
		case "accepted":
			e.Kind = "turn"
		case "refused":
			e.Kind = "refusal"
		default:
			e.Kind = "suppressed"
		}
		out = append(out, e)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	gateRows, err := q.DB.QueryContext(ctx, `SELECT g.thread_id,t.channel_id,g.scope,g.withheld,g.status,g.created_at FROM switchboard_capability_gates g
 JOIN switchboard_threads t ON t.id=g.thread_id JOIN switchboard_bindings b ON b.channel_id=t.channel_id AND b.thread_key=t.thread_key WHERE b.agent_id=? ORDER BY g.created_at DESC LIMIT ?`, agentID, limit)
	if err != nil {
		return nil, fmt.Errorf("agent gates: %w", err)
	}
	defer gateRows.Close()
	for gateRows.Next() {
		var e ActivityEntry
		var scope, withheld, status string
		if err := gateRows.Scan(&e.ThreadID, &e.ChannelID, &scope, &withheld, &status, &e.At); err != nil {
			return nil, err
		}
		e.Kind, e.Text, e.Reason = "gate", withheld, scope+" "+status
		out = append(out, e)
	}
	sortByAtDesc(out)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, gateRows.Err()
}

func sortByAtDesc(entries []ActivityEntry) {
	for i := 1; i < len(entries); i++ {
		for j := i; j > 0 && entries[j].At > entries[j-1].At; j-- {
			entries[j], entries[j-1] = entries[j-1], entries[j]
		}
	}
}
