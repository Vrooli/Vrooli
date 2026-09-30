// Package contacts owns sender identity: every address that has ever reached
// an agent, the trust tier it holds, and the thread rosters it appears in.
// The tier is the only editable field and it is the edit with the largest
// blast radius, because a group's ceiling is the minimum across its roster.
package contacts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"switchboard/internal/trust"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Contact struct {
	ID           string `json:"id"`
	ChannelID    string `json:"channel_id"`
	Address      string `json:"address"`
	DisplayName  string `json:"display_name"`
	Tier         string `json:"tier"`
	FirstSeen    string `json:"first_seen"`
	LastSeen     string `json:"last_seen"`
	MessageCount int64  `json:"message_count"`
	RoomCount    int64  `json:"room_count"`
}

type Participant struct {
	ContactID   string `json:"contact_id"`
	Address     string `json:"address"`
	DisplayName string `json:"display_name"`
	Tier        string `json:"tier"`
}

// Room is a thread seen from a contact's point of view.
type Room struct {
	ThreadID         string `json:"thread_id"`
	ChannelID        string `json:"channel_id"`
	ThreadKey        string `json:"thread_key"`
	IsGroup          bool   `json:"is_group"`
	CeilingTier      string `json:"ceiling_tier"`
	ParticipantCount int64  `json:"participant_count"`
}

// CeilingChange records how one tier edit moved one room's ceiling.
type CeilingChange struct {
	ThreadID        string `json:"thread_id"`
	ChannelID       string `json:"channel_id"`
	ThreadKey       string `json:"thread_key"`
	PreviousCeiling string `json:"previous_ceiling"`
	NewCeiling      string `json:"new_ceiling"`
}

var ErrNotFound = errors.New("contact not found")

type Store struct {
	db  SQLExecutor
	now func() time.Time
}

func NewStore(db SQLExecutor) *Store { return &Store{db: db, now: time.Now} }

func (s *Store) stamp() string { return s.now().UTC().Format(time.RFC3339Nano) }

// Seen upserts the sender of one inbound message. A new address is created at
// defaultTier, which the channel descriptor supplies; an existing contact
// never has its tier touched here, so a message can never widen a sender.
func (s *Store) Seen(ctx context.Context, channelID, address, defaultTier string) (Contact, error) {
	channelID, address = strings.TrimSpace(channelID), strings.TrimSpace(address)
	if channelID == "" || address == "" {
		return Contact{}, fmt.Errorf("channel_id and address are required")
	}
	if _, err := trust.ParseTier(defaultTier); err != nil {
		return Contact{}, err
	}
	now := s.stamp()
	_, err := s.db.ExecContext(ctx, `INSERT INTO switchboard_contacts(id,channel_id,address,display_name,tier,first_seen,last_seen,message_count)
 VALUES(?,?,?,?,?,?,?,1)
 ON CONFLICT(channel_id,address) DO UPDATE SET last_seen=excluded.last_seen, message_count=switchboard_contacts.message_count+1`,
		uuid.NewString(), channelID, address, "", defaultTier, now, now)
	if err != nil {
		return Contact{}, fmt.Errorf("record contact: %w", err)
	}
	return s.byAddress(ctx, channelID, address)
}

// Join adds a contact to a thread roster. Idempotent.
func (s *Store) Join(ctx context.Context, threadID, contactID string) error {
	if threadID == "" || contactID == "" {
		return fmt.Errorf("thread_id and contact_id are required")
	}
	_, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO switchboard_participants(thread_id,contact_id,joined_at) VALUES(?,?,?)`, threadID, contactID, s.stamp())
	if err != nil {
		return fmt.Errorf("join roster: %w", err)
	}
	return nil
}

// Roster lists everyone in a thread, lowest tier first so the ceiling is the
// first row.
func (s *Store) Roster(ctx context.Context, threadID string) ([]Participant, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT c.id,c.address,c.display_name,c.tier FROM switchboard_participants p JOIN switchboard_contacts c ON c.id=p.contact_id WHERE p.thread_id=? ORDER BY p.joined_at,c.address`, threadID)
	if err != nil {
		return nil, fmt.Errorf("read roster: %w", err)
	}
	defer rows.Close()
	out := make([]Participant, 0)
	for rows.Next() {
		var p Participant
		if err := rows.Scan(&p.ContactID, &p.Address, &p.DisplayName, &p.Tier); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Ceiling computes the room ceiling for a thread: the lowest tier present in
// its roster. An empty roster is a stranger room.
func (s *Store) Ceiling(ctx context.Context, threadID string) (trust.Tier, error) {
	roster, err := s.Roster(ctx, threadID)
	if err != nil {
		return trust.Stranger, err
	}
	return Ceiling(roster), nil
}

// Ceiling is the pure form of Store.Ceiling.
func Ceiling(roster []Participant) trust.Tier {
	if len(roster) == 0 {
		return trust.Stranger
	}
	result := trust.Owner
	for _, p := range roster {
		tier, err := trust.ParseTier(p.Tier)
		if err != nil {
			return trust.Stranger
		}
		if tier < result {
			result = tier
		}
	}
	return result
}

func (s *Store) List(ctx context.Context) ([]Contact, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT c.id,c.channel_id,c.address,c.display_name,c.tier,c.first_seen,c.last_seen,c.message_count,
 (SELECT COUNT(*) FROM switchboard_participants p WHERE p.contact_id=c.id) FROM switchboard_contacts c ORDER BY c.last_seen DESC, c.address`)
	if err != nil {
		return nil, fmt.Errorf("list contacts: %w", err)
	}
	defer rows.Close()
	out := make([]Contact, 0)
	for rows.Next() {
		var c Contact
		if err := rows.Scan(&c.ID, &c.ChannelID, &c.Address, &c.DisplayName, &c.Tier, &c.FirstSeen, &c.LastSeen, &c.MessageCount, &c.RoomCount); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) Get(ctx context.Context, id string) (Contact, error) {
	return s.one(ctx, `WHERE c.id=?`, id)
}

func (s *Store) byAddress(ctx context.Context, channelID, address string) (Contact, error) {
	return s.one(ctx, `WHERE c.channel_id=? AND c.address=?`, channelID, address)
}

func (s *Store) one(ctx context.Context, where string, args ...any) (Contact, error) {
	var c Contact
	err := s.db.QueryRowContext(ctx, `SELECT c.id,c.channel_id,c.address,c.display_name,c.tier,c.first_seen,c.last_seen,c.message_count,
 (SELECT COUNT(*) FROM switchboard_participants p WHERE p.contact_id=c.id) FROM switchboard_contacts c `+where, args...).
		Scan(&c.ID, &c.ChannelID, &c.Address, &c.DisplayName, &c.Tier, &c.FirstSeen, &c.LastSeen, &c.MessageCount, &c.RoomCount)
	if errors.Is(err, sql.ErrNoRows) {
		return Contact{}, ErrNotFound
	}
	if err != nil {
		return Contact{}, fmt.Errorf("read contact: %w", err)
	}
	return c, nil
}

// Rooms lists the threads a contact appears in with each room's current
// ceiling, so the operator can see what a tier change will touch.
func (s *Store) Rooms(ctx context.Context, contactID string) ([]Room, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT t.id,t.channel_id,t.thread_key,t.is_group FROM switchboard_participants p JOIN switchboard_threads t ON t.id=p.thread_id WHERE p.contact_id=? ORDER BY t.updated_at DESC`, contactID)
	if err != nil {
		return nil, fmt.Errorf("list rooms: %w", err)
	}
	var rooms []Room
	for rows.Next() {
		var r Room
		if err := rows.Scan(&r.ThreadID, &r.ChannelID, &r.ThreadKey, &r.IsGroup); err != nil {
			rows.Close()
			return nil, err
		}
		rooms = append(rooms, r)
	}
	rows.Close()
	if rooms == nil {
		rooms = []Room{}
	}
	for i := range rooms {
		roster, err := s.Roster(ctx, rooms[i].ThreadID)
		if err != nil {
			return nil, err
		}
		rooms[i].ParticipantCount = int64(len(roster))
		rooms[i].CeilingTier = Ceiling(roster).String()
	}
	return rooms, nil
}

// Update changes the tier and/or display name of one contact and reports every
// room whose ceiling moved as a result.
func (s *Store) Update(ctx context.Context, id string, tier, displayName *string) (Contact, []CeilingChange, error) {
	before, err := s.Rooms(ctx, id)
	if err != nil {
		return Contact{}, nil, err
	}
	if _, err := s.Get(ctx, id); err != nil {
		return Contact{}, nil, err
	}
	if tier != nil {
		parsed, err := trust.ParseTier(*tier)
		if err != nil {
			return Contact{}, nil, err
		}
		if _, err := s.db.ExecContext(ctx, `UPDATE switchboard_contacts SET tier=? WHERE id=?`, parsed.String(), id); err != nil {
			return Contact{}, nil, fmt.Errorf("update tier: %w", err)
		}
	}
	if displayName != nil {
		if _, err := s.db.ExecContext(ctx, `UPDATE switchboard_contacts SET display_name=? WHERE id=?`, strings.TrimSpace(*displayName), id); err != nil {
			return Contact{}, nil, fmt.Errorf("update display name: %w", err)
		}
	}
	after, err := s.Rooms(ctx, id)
	if err != nil {
		return Contact{}, nil, err
	}
	changes := make([]CeilingChange, 0)
	for i := range before {
		if i < len(after) && before[i].CeilingTier != after[i].CeilingTier {
			changes = append(changes, CeilingChange{ThreadID: before[i].ThreadID, ChannelID: before[i].ChannelID, ThreadKey: before[i].ThreadKey, PreviousCeiling: before[i].CeilingTier, NewCeiling: after[i].CeilingTier})
		}
	}
	contact, err := s.Get(ctx, id)
	return contact, changes, err
}
