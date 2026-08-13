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

	authdomain "device-control/internal/auth"
	devicedomain "device-control/internal/devices"
	internalflows "device-control/internal/flows"
	"device-control/strategy"
	strategyregistry "device-control/strategy/registry"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/filerouting"
)

type routedDB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type Service struct {
	registry            *strategyregistry.Registry
	db                  routedDB
	fileRoots           *filerouting.RoutedRoots
	attached            AttachedReader
	mu                  sync.Mutex
	sessions            map[string]Session
	audits              []Audit
	agents              map[string]AgentRun
	devices             *devicedomain.Store
	artifacts           map[string]string
	artifactKinds       map[string]string
	evidenceDir         string
	activeCancels       map[string]context.CancelFunc
	transportStrategies map[string]strategy.Strategy
	transportStates     map[string]transportState
	anchors             *internalflows.AnchorStore
	runs                map[string]RunResult
	flowRuns            map[string]Flow
	auth                *authdomain.Store
}

type transportState struct {
	DeviceID   string
	Serial     string
	StrategyID string
	Transport  string
	Endpoint   string
	UpdatedAt  time.Time
}

func New(registry *strategyregistry.Registry) *Service {
	dir := filepath.Join(os.TempDir(), "device-control-evidence")
	if configDir, err := os.UserConfigDir(); err == nil {
		dir = filepath.Join(configDir, "vrooli", "device-control", "evidence")
	}
	if configured := strings.TrimSpace(os.Getenv("DEVICE_CONTROL_EVIDENCE_DIR")); configured != "" {
		dir = configured
	}
	authStore, _ := authdomain.NewStore(nil, nil)
	return &Service{registry: registry, sessions: map[string]Session{}, audits: []Audit{}, agents: map[string]AgentRun{}, devices: devicedomain.NewStore(), artifacts: map[string]string{}, artifactKinds: map[string]string{}, evidenceDir: dir, activeCancels: map[string]context.CancelFunc{}, transportStrategies: map[string]strategy.Strategy{}, transportStates: map[string]transportState{}, anchors: internalflows.NewAnchorStore(), runs: map[string]RunResult{}, flowRuns: map[string]Flow{}, auth: authStore}
}

func NewWithAttached(registry *strategyregistry.Registry, reader AttachedReader) *Service {
	s := New(registry)
	s.attached = reader
	return s
}

// NewWithDB keeps the in-memory registry fast while making operator state
// durable. The API already owns the SQLite connection; passing it here avoids
// a second connection and keeps test-mode routing under api-core's control.
func NewWithDB(registry *strategyregistry.Registry, db routedDB, roots ...*filerouting.RoutedRoots) (*Service, error) {
	s := New(registry)
	s.db = db
	if len(roots) > 0 {
		s.fileRoots = roots[0]
	}
	if db == nil {
		return s, nil
	}
	anchors, err := internalflows.NewAnchorStoreWithDB(db)
	if err != nil {
		return nil, err
	}
	s.anchors = anchors
	if _, err := db.ExecContext(context.Background(), `
CREATE TABLE IF NOT EXISTS device_control_sessions (
 id TEXT PRIMARY KEY, device_id TEXT NOT NULL, actor TEXT NOT NULL,
 state TEXT NOT NULL, lease_token TEXT NOT NULL DEFAULT '', kill_reason TEXT NOT NULL DEFAULT '',
 expires_at TEXT NOT NULL, created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS device_control_sessions_device ON device_control_sessions(device_id, state);
CREATE TABLE IF NOT EXISTS device_control_audits (
 id TEXT PRIMARY KEY, actor TEXT NOT NULL, device_id TEXT NOT NULL, lease_id TEXT NOT NULL,
 verb TEXT NOT NULL, outcome TEXT NOT NULL, created_at TEXT NOT NULL, redaction_verified INTEGER NOT NULL,
 redaction_opted_out INTEGER NOT NULL DEFAULT 0, profile_id TEXT NOT NULL DEFAULT '',
 method TEXT NOT NULL DEFAULT '', attempts INTEGER NOT NULL DEFAULT 0,
 provider_state TEXT NOT NULL DEFAULT '', before_lock_state TEXT NOT NULL DEFAULT '',
 after_lock_state TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS device_control_transports (
 device_id TEXT PRIMARY KEY, serial TEXT NOT NULL, strategy_id TEXT NOT NULL,
 transport TEXT NOT NULL, endpoint TEXT NOT NULL, updated_at TEXT NOT NULL

);`); err != nil {
		return nil, fmt.Errorf("initialize device-control state: %w", err)
	}
	// Existing local databases predate the opt-out audit field. This additive
	// migration is intentionally best-effort because SQLite reports a duplicate
	// column when the database has already been upgraded.
	_, _ = db.ExecContext(context.Background(), `ALTER TABLE device_control_audits ADD COLUMN redaction_opted_out INTEGER NOT NULL DEFAULT 0`)
	for _, column := range []string{
		`profile_id TEXT NOT NULL DEFAULT ''`,
		`method TEXT NOT NULL DEFAULT ''`,
		`attempts INTEGER NOT NULL DEFAULT 0`,
		`provider_state TEXT NOT NULL DEFAULT ''`,
		`before_lock_state TEXT NOT NULL DEFAULT ''`,
		`after_lock_state TEXT NOT NULL DEFAULT ''`,
	} {
		_, _ = db.ExecContext(context.Background(), `ALTER TABLE device_control_audits ADD COLUMN `+column)
	}
	authStore, err := authdomain.NewStore(db, nil)
	if err != nil {
		return nil, err
	}
	s.auth = authStore
	if err := s.loadSessions(); err != nil {
		return nil, err
	}
	if err := s.loadTransportStates(); err != nil {
		return nil, err
	}
	s.restoreTransportStrategies()
	return s, nil
}

func (s *Service) loadSessions() error {
	if s.db == nil {
		return nil
	}
	rows, err := s.db.QueryContext(context.Background(), `SELECT id, device_id, actor, state, lease_token, kill_reason, expires_at, created_at FROM device_control_sessions`)
	if err != nil {
		return fmt.Errorf("load device sessions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var session Session
		var expires, created string
		if err := rows.Scan(&session.ID, &session.DeviceID, &session.Actor, &session.State, &session.LeaseToken, &session.KillReason, &expires, &created); err != nil {
			return fmt.Errorf("read device session: %w", err)
		}
		session.ExpiresAt, err = time.Parse(time.RFC3339Nano, expires)
		if err != nil {
			return fmt.Errorf("parse device session expiry: %w", err)
		}
		session.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return fmt.Errorf("parse device session creation: %w", err)
		}
		s.sessions[session.ID] = session
	}
	return rows.Err()
}

func (s *Service) Anchors() *internalflows.AnchorStore { return s.anchors }

func (s *Service) ReadDeviceState(ctx context.Context, deviceID string) (strategy.DeviceState, error) {
	transport := "usb"
	if record, found := s.devices.Get(deviceID); found && record.Transport != "" {
		transport = record.Transport
	}
	adapter, ok := s.strategyForFlow(deviceID, transport)
	if !ok {
		return strategy.DeviceState{}, fmt.Errorf("unknown or unavailable device %q", deviceID)
	}
	reader, ok := adapter.(strategy.StateReader)
	if !ok {
		return strategy.DeviceState{}, &strategy.AvailabilityError{Reason: "strategy does not expose device state", NextAction: "Use a strategy that declares device-state reads."}
	}
	return reader.ReadState(ctx)
}

func (s *Service) loadTransportStates() error {
	if s.db == nil {
		return nil
	}
	rows, err := s.db.QueryContext(context.Background(), `SELECT device_id, serial, strategy_id, transport, endpoint, updated_at FROM device_control_transports`)
	if err != nil {
		return fmt.Errorf("load device transports: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var state transportState
		var updated string
		if err := rows.Scan(&state.DeviceID, &state.Serial, &state.StrategyID, &state.Transport, &state.Endpoint, &updated); err != nil {
			return fmt.Errorf("read device transport: %w", err)
		}
		state.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		s.transportStates[state.DeviceID] = state
		if state.Transport == "wireless" {
			s.devices.Upsert(devicedomain.Record{ID: state.DeviceID, Kind: "physical", Serial: state.Serial, StrategyID: state.StrategyID, Transport: state.Transport, Status: strategy.HealthUnreachable, Health: strategy.HealthUnreachable, HealthReason: "restored wireless transport has not been probed yet"})
		}
	}
	return rows.Err()
}

func (s *Service) restoreTransportStrategies() {
	for id, state := range s.transportStates {
		if state.Transport != "wireless" || state.Endpoint == "" {
			continue
		}
		base, ok := s.registry.Get(state.StrategyID)
		if !ok {
			continue
		}
		if scoped, ok := base.(strategy.DeviceScoped); ok && state.Serial != "" {
			base = scoped.ForDevice(state.Serial)
		}
		restorer, ok := base.(interface {
			RestoreWireless(string) strategy.Strategy
		})
		if !ok {
			continue
		}
		s.transportStrategies[id] = restorer.RestoreWireless(state.Endpoint)
	}
}

func (s *Service) persistTransportState(ctx context.Context, state transportState) error {
	if s.db == nil {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO device_control_transports (device_id, serial, strategy_id, transport, endpoint, updated_at) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(device_id) DO UPDATE SET serial=excluded.serial, strategy_id=excluded.strategy_id, transport=excluded.transport, endpoint=excluded.endpoint, updated_at=excluded.updated_at`, state.DeviceID, state.Serial, state.StrategyID, state.Transport, state.Endpoint, state.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("persist device transport: %w", err)
	}
	return nil
}

func NewWithDBAndAttached(registry *strategyregistry.Registry, db routedDB, reader AttachedReader, roots ...*filerouting.RoutedRoots) (*Service, error) {
	s, err := NewWithDB(registry, db, roots...)
	if err != nil {
		return nil, err
	}
	s.attached = reader
	return s, nil
}

func (s *Service) Acquire(deviceID, actor string, ttl time.Duration) (Session, error) {
	return s.AcquireContext(context.Background(), deviceID, actor, ttl)
}

func (s *Service) AcquireContext(ctx context.Context, deviceID, actor string, ttl time.Duration) (Session, error) {
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
		if _, err := s.db.ExecContext(ctx, `INSERT INTO device_control_sessions (id, device_id, actor, state, lease_token, kill_reason, expires_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, sess.ID, sess.DeviceID, sess.Actor, sess.State, sess.LeaseToken, sess.KillReason, sess.ExpiresAt.Format(time.RFC3339Nano), sess.CreatedAt.Format(time.RFC3339Nano)); err != nil {
			return Session{}, fmt.Errorf("persist lease: %w", err)
		}
	}
	s.sessions[sess.ID] = sess
	return sess, nil
}

func (s *Service) ListSessions() []Session {
	return s.ListSessionsContext(context.Background())
}

func (s *Service) ListSessionsContext(ctx context.Context) []Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		rows, err := s.db.QueryContext(ctx, `SELECT id, device_id, actor, state, lease_token, kill_reason, expires_at, created_at FROM device_control_sessions ORDER BY created_at DESC`)
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
	return s.ListLiveSessionsContext(context.Background())
}

func (s *Service) ListLiveSessionsContext(ctx context.Context) []Session {
	now := time.Now()
	live := make([]Session, 0)
	for _, session := range s.ListSessionsContext(ctx) {
		if session.State == "held" && now.Before(session.ExpiresAt) {
			live = append(live, session)
		}
	}
	return live
}
func (s *Service) Kill(id, reason string) (Session, error) {
	return s.KillContext(context.Background(), id, reason)
}

func (s *Service) KillContext(ctx context.Context, id, reason string) (Session, error) {
	return s.finishContext(ctx, id, "killed", reason)
}

func (s *Service) Release(id string) (Session, error) {
	return s.ReleaseContext(context.Background(), id)
}

func (s *Service) ReleaseContext(ctx context.Context, id string) (Session, error) {
	return s.finishContext(ctx, id, "released", "")
}

func (s *Service) sessionForLease(_ context.Context, deviceID, token string) (Session, error) {
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

func (s *Service) finishContext(ctx context.Context, id, state, reason string) (Session, error) {
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
		if _, err := s.db.ExecContext(ctx, `UPDATE device_control_sessions SET state = ?, kill_reason = ?, lease_token = ? WHERE id = ?`, v.State, v.KillReason, v.LeaseToken, v.ID); err != nil {
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
	return s.AuditContext(context.Background())
}

func (s *Service) AuditContext(ctx context.Context) []Audit {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		rows, err := s.db.QueryContext(ctx, `SELECT id, actor, device_id, lease_id, verb, outcome, profile_id, method, attempts, provider_state, before_lock_state, after_lock_state, created_at, redaction_verified, redaction_opted_out FROM device_control_audits ORDER BY created_at DESC`)
		if err == nil {
			defer rows.Close()
			out := make([]Audit, 0)
			for rows.Next() {
				var v Audit
				var created string
				var verified, optedOut int
				if err := rows.Scan(&v.ID, &v.Actor, &v.DeviceID, &v.LeaseID, &v.Verb, &v.Outcome, &v.ProfileID, &v.Method, &v.Attempts, &v.ProviderState, &v.BeforeLockState, &v.AfterLockState, &created, &verified, &optedOut); err != nil {
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
