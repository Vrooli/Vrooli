package control

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	devicedomain "device-control/internal/devices"
	strategyregistry "device-control/strategy/registry"
	"github.com/google/uuid"
)

type Service struct {
	registry      *strategyregistry.Registry
	db            *sql.DB
	attached      AttachedReader
	mu            sync.Mutex
	sessions      map[string]Session
	audits        []Audit
	agents        map[string]AgentRun
	devices       *devicedomain.Store
	artifacts     map[string]string
	artifactKinds map[string]string
	evidenceDir   string
	activeCancels map[string]context.CancelFunc
}

func New(registry *strategyregistry.Registry) *Service {
	dir := filepath.Join(os.TempDir(), "device-control-evidence")
	if configDir, err := os.UserConfigDir(); err == nil {
		dir = filepath.Join(configDir, "vrooli", "device-control", "evidence")
	}
	if configured := strings.TrimSpace(os.Getenv("DEVICE_CONTROL_EVIDENCE_DIR")); configured != "" {
		dir = configured
	}
	return &Service{registry: registry, sessions: map[string]Session{}, audits: []Audit{}, agents: map[string]AgentRun{}, devices: devicedomain.NewStore(), artifacts: map[string]string{}, artifactKinds: map[string]string{}, evidenceDir: dir, activeCancels: map[string]context.CancelFunc{}}
}

func NewWithAttached(registry *strategyregistry.Registry, reader AttachedReader) *Service {
	s := New(registry)
	s.attached = reader
	return s
}

// NewWithDB keeps the in-memory registry fast while making operator state
// durable. The API already owns the SQLite connection; passing it here avoids
// a second connection and keeps test-mode routing under api-core's control.
func NewWithDB(registry *strategyregistry.Registry, db *sql.DB) (*Service, error) {
	s := New(registry)
	s.db = db
	if db == nil {
		return s, nil
	}
	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS device_control_sessions (
 id TEXT PRIMARY KEY, device_id TEXT NOT NULL, actor TEXT NOT NULL,
 state TEXT NOT NULL, lease_token TEXT NOT NULL DEFAULT '', kill_reason TEXT NOT NULL DEFAULT '',
 expires_at TEXT NOT NULL, created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS device_control_sessions_device ON device_control_sessions(device_id, state);
CREATE TABLE IF NOT EXISTS device_control_audits (
 id TEXT PRIMARY KEY, actor TEXT NOT NULL, device_id TEXT NOT NULL, lease_id TEXT NOT NULL,
 verb TEXT NOT NULL, outcome TEXT NOT NULL, created_at TEXT NOT NULL, redaction_verified INTEGER NOT NULL,
 redaction_opted_out INTEGER NOT NULL DEFAULT 0
);`); err != nil {
		return nil, fmt.Errorf("initialize device-control state: %w", err)
	}
	// Existing local databases predate the opt-out audit field. This additive
	// migration is intentionally best-effort because SQLite reports a duplicate
	// column when the database has already been upgraded.
	_, _ = db.Exec(`ALTER TABLE device_control_audits ADD COLUMN redaction_opted_out INTEGER NOT NULL DEFAULT 0`)
	return s, nil
}

func NewWithDBAndAttached(registry *strategyregistry.Registry, db *sql.DB, reader AttachedReader) (*Service, error) {
	s, err := NewWithDB(registry, db)
	if err != nil {
		return nil, err
	}
	s.attached = reader
	return s, nil
}

func (s *Service) Acquire(deviceID, actor string, ttl time.Duration) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for id, old := range s.sessions {
		if old.State == "held" && now.After(old.ExpiresAt) {
			old.State = "expired"
			s.sessions[id] = old
		}
	}
	for _, old := range s.sessions {
		if old.DeviceID == deviceID && old.State == "held" {
			return Session{}, fmt.Errorf("device %s already has a held lease owned by %q until %s", deviceID, old.Actor, old.ExpiresAt.Format(time.RFC3339))
		}
	}
	if _, ok := s.strategyForDevice(deviceID); !ok {
		return Session{}, fmt.Errorf("unknown device %q", deviceID)
	}
	if ttl <= 0 || ttl > 30*time.Minute {
		ttl = 10 * time.Minute
	}
	sess := Session{ID: uuid.NewString(), DeviceID: deviceID, Actor: actor, State: "held", LeaseToken: uuid.NewString(), ExpiresAt: now.Add(ttl), CreatedAt: now}
	if s.db != nil {
		if _, err := s.db.Exec(`INSERT INTO device_control_sessions (id, device_id, actor, state, lease_token, kill_reason, expires_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, sess.ID, sess.DeviceID, sess.Actor, sess.State, sess.LeaseToken, sess.KillReason, sess.ExpiresAt.Format(time.RFC3339Nano), sess.CreatedAt.Format(time.RFC3339Nano)); err != nil {
			return Session{}, fmt.Errorf("persist lease: %w", err)
		}
	}
	s.sessions[sess.ID] = sess
	return sess, nil
}

func (s *Service) ListSessions() []Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		rows, err := s.db.Query(`SELECT id, device_id, actor, state, lease_token, kill_reason, expires_at, created_at FROM device_control_sessions ORDER BY created_at DESC`)
		if err == nil {
			defer rows.Close()
			out := make([]Session, 0)
			for rows.Next() {
				var v Session
				var expires, created string
				if err := rows.Scan(&v.ID, &v.DeviceID, &v.Actor, &v.State, &v.LeaseToken, &v.KillReason, &expires, &created); err != nil {
					continue
				}
				v.ExpiresAt, _ = time.Parse(time.RFC3339Nano, expires)
				v.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
				out = append(out, v)
			}
			return out
		}
	}
	out := make([]Session, 0, len(s.sessions))
	for _, v := range s.sessions {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

// ListLiveSessions returns only leases that can currently authorize device
// work. Historical released, killed, and expired sessions remain available to
// ListSessions for audit/reconstruction, but must not appear in operator live
// lease surfaces.
func (s *Service) ListLiveSessions() []Session {
	now := time.Now()
	live := make([]Session, 0)
	for _, session := range s.ListSessions() {
		if session.State == "held" && now.Before(session.ExpiresAt) {
			live = append(live, session)
		}
	}
	return live
}
func (s *Service) Kill(id, reason string) (Session, error) { return s.finish(id, "killed", reason) }
func (s *Service) Release(id string) (Session, error)      { return s.finish(id, "released", "") }

func (s *Service) sessionForLease(deviceID, token string) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for _, session := range s.sessions {
		if session.LeaseToken == token {
			if session.DeviceID != deviceID {
				return Session{}, fmt.Errorf("lease token is bound to device %q", session.DeviceID)
			}
			if session.State != "held" {
				return Session{}, fmt.Errorf("lease is %s", session.State)
			}
			if now.After(session.ExpiresAt) {
				return Session{}, fmt.Errorf("lease expired at %s", session.ExpiresAt.Format(time.RFC3339))
			}
			return session, nil
		}
	}
	return Session{}, fmt.Errorf("lease token is invalid or no longer held")
}

func (s *Service) finish(id, state, reason string) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.sessions[id]
	if !ok {
		return Session{}, fmt.Errorf("session %q not found", id)
	}
	v.State = state
	v.KillReason = reason
	v.LeaseToken = ""
	if s.db != nil {
		if _, err := s.db.Exec(`UPDATE device_control_sessions SET state = ?, kill_reason = ?, lease_token = ? WHERE id = ?`, v.State, v.KillReason, v.LeaseToken, v.ID); err != nil {
			return Session{}, fmt.Errorf("persist session state: %w", err)
		}
	}
	s.sessions[id] = v
	if cancel := s.activeCancels[id]; cancel != nil {
		cancel()
		delete(s.activeCancels, id)
	}
	return v, nil
}

func (s *Service) Audit() []Audit {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		rows, err := s.db.Query(`SELECT id, actor, device_id, lease_id, verb, outcome, created_at, redaction_verified, redaction_opted_out FROM device_control_audits ORDER BY created_at DESC`)
		if err == nil {
			defer rows.Close()
			out := make([]Audit, 0)
			for rows.Next() {
				var v Audit
				var created string
				var verified, optedOut int
				if err := rows.Scan(&v.ID, &v.Actor, &v.DeviceID, &v.LeaseID, &v.Verb, &v.Outcome, &created, &verified, &optedOut); err != nil {
					continue
				}
				v.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
				v.RedactionVerified = verified != 0
				v.RedactionOptedOut = optedOut != 0
				out = append(out, v)
			}
			return out
		}
	}
	return append([]Audit{}, s.audits...)
}
