package control

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/database"
)

func (s *Service) loadSessions() error {
	if s.db == nil {
		return nil
	}
	rows, err := s.db.QueryContext(context.Background(), `SELECT id, device_id, actor, state, lease_token, kill_reason, expires_at, created_at, observation_only FROM device_control_sessions`)
	if err != nil {
		return fmt.Errorf("load device sessions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var session Session
		var expires, created string
		if err := rows.Scan(&session.ID, &session.DeviceID, &session.Actor, &session.State, &session.LeaseToken, &session.KillReason, &expires, &created, &session.ObservationOnly); err != nil {
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

// registerSessionOperation atomically fences one bounded operation against
// lease release and other operations using the same lease.
func (s *Service) registerSessionOperation(ctx context.Context, session Session, deviceID string, duration time.Duration) (context.Context, func(), error) {
	operationCtx, cancel := context.WithTimeout(ctx, duration)
	s.sessionMu.Lock()
	current, held := s.sessions[session.ID]
	if !held || current.State != "held" || current.DeviceID != deviceID || current.LeaseToken != session.LeaseToken || !time.Now().Before(current.ExpiresAt) {
		s.sessionMu.Unlock()
		cancel()
		return nil, nil, fmt.Errorf("lease is no longer held")
	}
	if _, active := s.activeCancels[session.ID]; active {
		s.sessionMu.Unlock()
		cancel()
		return nil, nil, fmt.Errorf("lease already has an active run")
	}
	s.activeCancels[session.ID] = cancel
	s.sessionMu.Unlock()
	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			cancel()
			s.sessionMu.Lock()
			delete(s.activeCancels, session.ID)
			s.sessionMu.Unlock()
		})
	}
	return operationCtx, cleanup, nil
}

func (s *Service) Acquire(deviceID, actor string, ttl time.Duration) (Session, error) {
	return s.AcquireContext(context.Background(), deviceID, actor, ttl)
}

func (s *Service) AcquireContext(ctx context.Context, deviceID, actor string, ttl time.Duration) (Session, error) {
	return s.acquireSession(ctx, deviceID, actor, ttl, false)
}

// AcquireObservationContext creates bounded read authority without occupying the
// exclusive input lease. No existing actuation token validator accepts it.
func (s *Service) AcquireObservationContext(ctx context.Context, deviceID, actor string, ttl time.Duration) (Session, error) {
	if ttl <= 0 || ttl > 10*time.Minute {
		return Session{}, fmt.Errorf("invalid observation lifetime")
	}
	return s.acquireSession(ctx, deviceID, actor, ttl, true)
}

func (s *Service) acquireSession(ctx context.Context, deviceID, actor string, ttl time.Duration, observationOnly bool) (Session, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	// Routed test pools (and other per-request database selectors) can be
	// installed after Service construction. Reconcile the additive session
	// column on the active pool before the first write so an older or freshly
	// created routed table cannot fail at lease admission.
	if s.db != nil {
		if err := database.EnsureSchemas(ctx, s.db, database.SchemaProviderFunc(func() string {
			return `CREATE TABLE IF NOT EXISTS device_control_sessions (
 id TEXT PRIMARY KEY, device_id TEXT NOT NULL, actor TEXT NOT NULL,
 state TEXT NOT NULL, lease_token TEXT NOT NULL DEFAULT '', kill_reason TEXT NOT NULL DEFAULT '',
 expires_at TEXT NOT NULL, created_at TEXT NOT NULL, observation_only INTEGER NOT NULL DEFAULT 0
);`
		})); err != nil {
			return Session{}, fmt.Errorf("ensure session schema: %w", err)
		}
	}
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	now := time.Now().UTC()
	for id, old := range s.sessions {
		if old.State == "held" && now.After(old.ExpiresAt) {
			old.State = "expired"
			s.sessions[id] = old
		}
	}
	for _, old := range s.sessions {
		if old.DeviceID == deviceID && old.State == "held" && !old.ObservationOnly && !observationOnly {
			return Session{}, fmt.Errorf("device %s already has a held lease owned by %q until %s", deviceID, old.Actor, old.ExpiresAt.Format(time.RFC3339))
		}
	}
	if observationOnly {
		count := 0
		for _, old := range s.sessions {
			if old.DeviceID == deviceID && old.State == "held" && old.ObservationOnly {
				count++
			}
		}
		if count >= 64 {
			return Session{}, fmt.Errorf("observation session limit reached")
		}
	}
	if _, ok := s.strategyForDevice(deviceID); !ok {
		return Session{}, fmt.Errorf("unknown device %q", deviceID)
	}
	if ttl <= 0 || ttl > 30*time.Minute {
		ttl = 10 * time.Minute
	}
	sess := Session{ObservationOnly: observationOnly, ID: uuid.NewString(), DeviceID: deviceID, Actor: actor, State: "held", LeaseToken: uuid.NewString(), ExpiresAt: now.Add(ttl), CreatedAt: now}
	if s.db != nil {
		if _, err := s.db.ExecContext(ctx, `INSERT INTO device_control_sessions (id, device_id, actor, state, lease_token, kill_reason, expires_at, created_at, observation_only) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, sess.ID, sess.DeviceID, sess.Actor, sess.State, sess.LeaseToken, sess.KillReason, sess.ExpiresAt.Format(time.RFC3339Nano), sess.CreatedAt.Format(time.RFC3339Nano), sess.ObservationOnly); err != nil {
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
	if ctx == nil {
		ctx = context.Background()
	}
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	if s.db != nil {
		timeout := s.sessionQueryTimeout
		if timeout <= 0 {
			timeout = 750 * time.Millisecond
		}
		queryCtx, cancel := context.WithTimeout(ctx, timeout)
		rows, err := s.db.QueryContext(queryCtx, `SELECT id, device_id, actor, state, lease_token, kill_reason, expires_at, created_at, observation_only FROM device_control_sessions ORDER BY created_at DESC`)
		if err == nil {
			defer cancel()
			defer rows.Close()
			out := make([]Session, 0)
			for rows.Next() {
				var v Session
				var expires, created string
				if err := rows.Scan(&v.ID, &v.DeviceID, &v.Actor, &v.State, &v.LeaseToken, &v.KillReason, &expires, &created, &v.ObservationOnly); err != nil {
					continue
				}
				v.ExpiresAt, _ = time.Parse(time.RFC3339Nano, expires)
				v.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
				out = append(out, v)
			}
			if err := rows.Err(); err == nil {
				return out
			}
		} else {
			cancel()
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

func (s *Service) sessionForLease(ctx context.Context, deviceID, token string) (Session, error) {
	return s.sessionForAccess(ctx, deviceID, token, false)
}

func (s *Service) sessionForAccess(ctx context.Context, deviceID, token string, observation bool) (Session, error) {
	if err := ctx.Err(); err != nil {
		return Session{}, err
	}
	// Attached-device trust is owned by Bridge. Re-read it at the input
	// boundary so revoking a peripheral grant immediately fences an already
	// held lease while leaving unrelated local sessions untouched. Inventory is
	// allowed to cache descriptors; actuation is not allowed to cache trust.
	if !observation {
		if device, found := s.devices.Get(deviceID); found && !(device.Kind == "desktop" && (strings.EqualFold(strings.TrimSpace(device.StrategyID), "host-desktop") || strings.EqualFold(strings.TrimSpace(device.Transport), "local"))) && strings.TrimSpace(device.HostNodeID) != "" && s.attached != nil {
			var attached AttachedDevice
			var foundAttached bool
			var err error
			if reader, ok := s.attached.(AttachedRevocationReader); ok {
				attached, err = reader.Get(ctx, deviceID)
				foundAttached = err == nil
			} else {
				var devices []AttachedDevice
				devices, err = s.attached.List(ctx)
				if err == nil {
					for _, candidate := range devices {
						if candidate.ID == deviceID {
							attached, foundAttached = candidate, true
							break
						}
					}
				}
			}
			if err != nil || !foundAttached {
				if err == nil {
					err = fmt.Errorf("device %q was not returned by attached inventory", deviceID)
				}
				return Session{}, fmt.Errorf("attached device trust state unavailable: %w", err)
			}
			if strings.EqualFold(strings.TrimSpace(attached.TrustState), "revoked") || !strings.EqualFold(strings.TrimSpace(attached.Reachability), "reachable") {
				return Session{}, fmt.Errorf("attached device %q is no longer authorized", deviceID)
			}
		}
	}
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	now := time.Now().UTC()
	for _, session := range s.sessions {
		if session.LeaseToken == token {
			if session.DeviceID != deviceID {
				return Session{}, fmt.Errorf("lease token is bound to device %q", session.DeviceID)
			}
			if session.State != "held" {
				return Session{}, fmt.Errorf("lease is %s", session.State)
			}
			if !now.Before(session.ExpiresAt) {
				return Session{}, fmt.Errorf("lease expired at %s", session.ExpiresAt.Format(time.RFC3339))
			}
			if session.ObservationOnly && !observation {
				return Session{}, fmt.Errorf("observation session cannot authorize input")
			}
			return session, nil
		}
	}
	return Session{}, fmt.Errorf("lease token is invalid or no longer held")
}

// ValidateLease is the narrow public validation seam for long-running
// delivery ramps. It deliberately returns no session or token material.
func (s *Service) ValidateLease(ctx context.Context, deviceID, token string) error {
	_, err := s.sessionForLease(ctx, deviceID, token)
	return err
}

func (s *Service) finishContext(ctx context.Context, id, state, reason string) (Session, error) {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
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
	}
	return v, nil
}

// ValidateObservationLease accepts current observation or control authority;
// it cannot be used as the input admission validator.
func (s *Service) ValidateObservationLease(ctx context.Context, deviceID, token string) error {
	_, err := s.sessionForAccess(ctx, deviceID, token, true)
	return err
}
