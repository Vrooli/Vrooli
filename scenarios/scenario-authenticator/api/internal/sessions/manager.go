// Package sessions owns the Redis-backed hot state of authentication: live
// sessions, the refresh-token-family state (with single-use rotation + reuse
// detection), and the access-token blacklist. Key shapes are ported from the
// old auth/session.go; the refresh-family reuse-detection is NEW (the old code
// rotated but never detected replay of a rotated token).
//
// Refresh-family model (reuse detection):
//   - refresh:<H(token)>      → "<familyID>|<userID>|<audience>|<authVersion>"
//     (a currently-valid token)
//   - refreshused:<H(token)>  → "<familyID>"           (a token already rotated out)
//   - refreshfam:<familyID>   → set of every H(token) ever issued in the family
//   - refreshfamdead:<familyID> → "1"                  (family revoked)
//
// Presenting a token whose refresh:* key is gone but whose refreshused:* key
// exists is a replay of a rotated token: the whole family is revoked and the
// caller audits it. This is the standard OAuth refresh-token rotation defense.
package sessions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"scenario-authenticator/internal/authcrypto"
	"scenario-authenticator/internal/redisstate"
)

// TTLs. Refresh + session lifetime is 7d (ported); the blacklist entry lives
// only as long as the access token it shadows.
const (
	refreshTTL = 7 * 24 * time.Hour
	sessionTTL = 7 * 24 * time.Hour
)

// Sentinel errors.
var (
	// ErrInvalidRefresh — the refresh token is unknown/expired.
	ErrInvalidRefresh = errors.New("invalid refresh token")
	// ErrRefreshReuse — a rotated-out refresh token was replayed; the family
	// has been revoked. Distinct so the caller can audit it as a security event.
	ErrRefreshReuse = errors.New("refresh token reuse detected")
)

// Session is the stored hot-state shape of a live session.
type Session struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	IPAddress       string    `json:"ip_address"`
	UserAgent       string    `json:"user_agent"`
	CreatedAt       time.Time `json:"created_at"`
	ExpiresAt       time.Time `json:"expires_at"`
	RefreshFamilyID string    `json:"refresh_family_id,omitempty"`
}

// Manager owns sessions + refresh-family + blacklist over a redisstate.Store.
type Manager struct {
	store redisstate.Store
	now   func() time.Time
}

// NewManager constructs a Manager. now defaults to time.Now.
func NewManager(store redisstate.Store, now func() time.Time) *Manager {
	if now == nil {
		now = time.Now
	}
	return &Manager{store: store, now: now}
}

// ---- sessions -------------------------------------------------------------

// StoreSession persists a new session and indexes it under the user, returning
// the new session id.
func (m *Manager) StoreSession(ctx context.Context, userID, ip, userAgent string) (string, error) {
	return m.StoreSessionForFamily(ctx, userID, ip, userAgent, "")
}

// StoreSessionForFamily persists a session bound to one refresh-token family.
// The binding lets a per-session revoke invalidate the refresh material that
// could otherwise recreate the revoked session.
func (m *Manager) StoreSessionForFamily(ctx context.Context, userID, ip, userAgent, familyID string) (string, error) {
	id, err := authcrypto.GenerateRefreshToken() // 32-byte random id, same generator
	if err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	now := m.now().UTC()
	sess := Session{
		ID:              id,
		UserID:          userID,
		IPAddress:       ip,
		UserAgent:       userAgent,
		CreatedAt:       now,
		ExpiresAt:       now.Add(sessionTTL),
		RefreshFamilyID: strings.TrimSpace(familyID),
	}
	data, err := json.Marshal(sess)
	if err != nil {
		return "", fmt.Errorf("marshal session: %w", err)
	}
	if err := m.store.Set(ctx, sessionKey(id), string(data), sessionTTL); err != nil {
		return "", fmt.Errorf("store session: %w", err)
	}
	if err := m.store.SAdd(ctx, userSessionsKey(userID), id); err != nil {
		return "", fmt.Errorf("index session: %w", err)
	}
	return id, nil
}

// ListSessions returns the live sessions for a user (skipping any that have
// expired out of the store).
func (m *Manager) ListSessions(ctx context.Context, userID string) ([]Session, error) {
	ids, err := m.store.SMembers(ctx, userSessionsKey(userID))
	if err != nil {
		return nil, fmt.Errorf("list session ids: %w", err)
	}
	var out []Session
	for _, id := range ids {
		raw, ok, err := m.store.Get(ctx, sessionKey(id))
		if err != nil {
			return nil, err
		}
		if !ok {
			_ = m.store.SRem(ctx, userSessionsKey(userID), id) // prune dangling index entry
			continue
		}
		var s Session
		if err := json.Unmarshal([]byte(raw), &s); err != nil {
			continue
		}
		out = append(out, s)
	}
	return out, nil
}

// RevokeSession drops a single session. Idempotent: revoking a missing/blank
// session is a no-op success (preserves the REST 200/204/404 contract
// device-sync-hub relies on).
func (m *Manager) RevokeSession(ctx context.Context, sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil
	}
	if raw, ok, err := m.store.Get(ctx, sessionKey(sessionID)); err == nil && ok {
		var s Session
		if json.Unmarshal([]byte(raw), &s) == nil && s.UserID != "" {
			if err := m.store.SRem(ctx, userSessionsKey(s.UserID), sessionID); err != nil {
				return err
			}
			if s.RefreshFamilyID != "" {
				if err := m.revokeFamily(ctx, s.RefreshFamilyID); err != nil {
					return err
				}
			}
		}
	}
	return m.store.Del(ctx, sessionKey(sessionID))
}

// FindSession returns a live session by id without exposing token material.
// The accounts service uses this narrow lookup to authorize user-facing
// single-session revocation before calling the idempotent delete operation.
func (m *Manager) FindSession(ctx context.Context, sessionID string) (Session, bool, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return Session{}, false, nil
	}
	raw, ok, err := m.store.Get(ctx, sessionKey(sessionID))
	if err != nil || !ok {
		return Session{}, ok, err
	}
	var session Session
	if err := json.Unmarshal([]byte(raw), &session); err != nil {
		return Session{}, false, fmt.Errorf("decode session: %w", err)
	}
	return session, true, nil
}

// RevokeAllSessions drops every session for a user and returns how many were
// removed.
func (m *Manager) RevokeAllSessions(ctx context.Context, userID string) (int, error) {
	if _, err := m.BumpAuthVersion(ctx, userID); err != nil {
		return 0, err
	}
	ids, err := m.store.SMembers(ctx, userSessionsKey(userID))
	if err != nil {
		return 0, fmt.Errorf("list session ids: %w", err)
	}
	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, sessionKey(id))
	}
	if len(keys) > 0 {
		if err := m.store.Del(ctx, keys...); err != nil {
			return 0, err
		}
	}
	if err := m.store.Del(ctx, userSessionsKey(userID)); err != nil {
		return 0, err
	}
	families, err := m.store.SMembers(ctx, userRefreshFamiliesKey(userID))
	if err != nil {
		return 0, err
	}
	for _, familyID := range families {
		if err := m.revokeFamily(ctx, familyID); err != nil {
			return 0, err
		}
	}
	if err := m.store.Del(ctx, userRefreshFamiliesKey(userID)); err != nil {
		return 0, err
	}
	return len(ids), nil
}

// CurrentAuthVersion returns the account-wide authorization version. Missing
// state is version zero so legacy accounts remain valid until their first
// explicit revocation/password change.
func (m *Manager) CurrentAuthVersion(ctx context.Context, userID string) (int64, error) {
	raw, found, err := m.store.Get(ctx, authVersionKey(userID))
	if err != nil || !found {
		return 0, err
	}
	var version int64
	if _, err := fmt.Sscan(raw, &version); err != nil || version < 0 {
		return 0, fmt.Errorf("invalid auth version for user %q", userID)
	}
	return version, nil
}

// BumpAuthVersion invalidates all access tokens issued before the bump.
func (m *Manager) BumpAuthVersion(ctx context.Context, userID string) (int64, error) {
	version, err := m.store.Incr(ctx, authVersionKey(userID))
	if err != nil {
		return 0, err
	}
	if err := m.store.Expire(ctx, authVersionKey(userID), refreshTTL); err != nil {
		return 0, err
	}
	return version, nil
}

// RefreshFamilyForToken returns the family and account bound to a live
// refresh token. It is used to bind access tokens and sessions to the same
// revocable family.
func (m *Manager) RefreshFamilyForToken(ctx context.Context, token string) (familyID, userID string, err error) {
	raw, found, err := m.store.Get(ctx, refreshKey(authcrypto.HashToken(token)))
	if err != nil {
		return "", "", err
	}
	if !found {
		return "", "", ErrInvalidRefresh
	}
	familyID, userID, _, _, err = parseRefreshValue(raw)
	if err != nil {
		return "", "", ErrInvalidRefresh
	}
	return familyID, userID, nil
}

// SessionForRefreshFamily finds the session associated with a refresh family.
// A family normally has exactly one session; the fallback supports legacy
// refresh tokens that predate family-bound session records.
func (m *Manager) SessionForRefreshFamily(ctx context.Context, userID, familyID string) (Session, bool, error) {
	sessions, err := m.ListSessions(ctx, userID)
	if err != nil {
		return Session{}, false, err
	}
	for _, session := range sessions {
		if session.RefreshFamilyID == familyID {
			return session, true, nil
		}
	}
	return Session{}, false, nil
}

// ---- refresh-token family -------------------------------------------------

// IssueRefresh mints a fresh refresh token in a NEW family for userID and
// returns the token (the family id is internal).
func (m *Manager) IssueRefresh(ctx context.Context, userID, audience string) (string, error) {
	familyID, err := authcrypto.GenerateSecureToken(16)
	if err != nil {
		return "", err
	}
	version, err := m.CurrentAuthVersion(ctx, userID)
	if err != nil {
		return "", err
	}
	return m.mintInFamily(ctx, userID, audience, familyID, version)
}

// RotateRefresh validates a presented refresh token, rotates it single-use, and
// returns the new token + the owning userID. Replaying an already-rotated token
// revokes the whole family and returns ErrRefreshReuse.
func (m *Manager) RotateRefresh(ctx context.Context, presented string) (newToken, userID, audience string, err error) {
	hash := authcrypto.HashToken(presented)

	raw, ok, err := m.store.Get(ctx, refreshKey(hash))
	if err != nil {
		return "", "", "", err
	}
	if ok {
		if strings.HasPrefix(raw, "used|") {
			_ = m.revokeFamily(ctx, strings.TrimPrefix(raw, "used|"))
			return "", "", "", ErrRefreshReuse
		}
		familyID, uid, audience, version, perr := parseRefreshValue(raw)
		if perr != nil {
			return "", "", "", ErrInvalidRefresh
		}
		currentVersion, versionErr := m.CurrentAuthVersion(ctx, uid)
		if versionErr != nil {
			return "", "", "", versionErr
		}
		if version != currentVersion {
			_ = m.revokeFamily(ctx, familyID)
			return "", "", "", ErrInvalidRefresh
		}
		// Single-use rotation: atomically mark the presented token consumed before
		// minting its successor. A concurrent caller therefore observes replay,
		// even when requests land on different replicas sharing Redis.
		consumed, err := m.store.CompareAndSwap(ctx, refreshKey(hash), raw, "used|"+familyID, refreshTTL)
		if err != nil {
			return "", "", "", err
		}
		if !consumed {
			_ = m.revokeFamily(ctx, familyID)
			return "", "", "", ErrRefreshReuse
		}
		if err := m.store.Set(ctx, refreshUsedKey(hash), familyID, refreshTTL); err != nil {
			return "", "", "", err
		}
		nt, err := m.mintInFamily(ctx, uid, audience, familyID, currentVersion)
		if err != nil {
			return "", "", "", err
		}
		return nt, uid, audience, nil
	}

	// Not a live token. If it was rotated out, this is a replay → revoke family.
	if famID, used, err := m.store.Get(ctx, refreshUsedKey(hash)); err == nil && used {
		_ = m.revokeFamily(ctx, famID)
		return "", "", "", ErrRefreshReuse
	}
	return "", "", "", ErrInvalidRefresh
}

// RevokeRefreshFamilyForToken revokes the family a (currently-valid) refresh
// token belongs to. Used on logout/all-session revoke. A no-op for an unknown
// token.
func (m *Manager) RevokeRefreshFamilyForToken(ctx context.Context, token string) error {
	hash := authcrypto.HashToken(token)
	raw, ok, err := m.store.Get(ctx, refreshKey(hash))
	if err != nil || !ok {
		return err
	}
	famID, _, _, _, perr := parseRefreshValue(raw)
	if perr != nil {
		return nil
	}
	return m.revokeFamily(ctx, famID)
}

func (m *Manager) mintInFamily(ctx context.Context, userID, audience, familyID string, version int64) (string, error) {
	token, err := authcrypto.GenerateRefreshToken()
	if err != nil {
		return "", err
	}
	hash := authcrypto.HashToken(token)
	if err := m.store.Set(ctx, refreshKey(hash), fmt.Sprintf("%s|%s|%s|%d", familyID, userID, audience, version), refreshTTL); err != nil {
		return "", err
	}
	if err := m.store.SAdd(ctx, refreshFamilyKey(familyID), hash); err != nil {
		return "", err
	}
	if err := m.store.Expire(ctx, refreshFamilyKey(familyID), refreshTTL); err != nil {
		return "", err
	}
	if err := m.store.SAdd(ctx, userRefreshFamiliesKey(userID), familyID); err != nil {
		return "", err
	}
	if err := m.store.Expire(ctx, userRefreshFamiliesKey(userID), refreshTTL); err != nil {
		return "", err
	}
	return token, nil
}

func (m *Manager) revokeFamily(ctx context.Context, familyID string) error {
	hashes, err := m.store.SMembers(ctx, refreshFamilyKey(familyID))
	if err != nil {
		return err
	}
	keys := make([]string, 0, len(hashes)*2+1)
	for _, h := range hashes {
		keys = append(keys, refreshKey(h), refreshUsedKey(h))
	}
	keys = append(keys, refreshFamilyKey(familyID))
	if len(keys) > 0 {
		if err := m.store.Del(ctx, keys...); err != nil {
			return err
		}
	}
	return m.store.Set(ctx, refreshFamilyDeadKey(familyID), "1", refreshTTL)
}

// ---- access-token blacklist ----------------------------------------------

// BlacklistAccess marks an access token revoked until its own expiry. A token
// already at/after expiry is a no-op (it is invalid anyway).
func (m *Manager) BlacklistAccess(ctx context.Context, token string, expiresAt time.Time) error {
	ttl := expiresAt.Sub(m.now())
	if ttl <= 0 {
		return nil
	}
	return m.store.Set(ctx, blacklistKey(authcrypto.HashToken(token)), "1", ttl)
}

// IsBlacklisted reports whether an access token has been revoked.
func (m *Manager) IsBlacklisted(ctx context.Context, token string) (bool, error) {
	return m.store.Exists(ctx, blacklistKey(authcrypto.HashToken(token)))
}

// ---- key shapes -----------------------------------------------------------

func sessionKey(id string) string              { return "session:" + id }
func userSessionsKey(uid string) string        { return "usersessions:" + uid }
func refreshKey(hash string) string            { return "refresh:" + hash }
func refreshUsedKey(hash string) string        { return "refreshused:" + hash }
func refreshFamilyKey(fid string) string       { return "refreshfam:" + fid }
func refreshFamilyDeadKey(fid string) string   { return "refreshfamdead:" + fid }
func userRefreshFamiliesKey(uid string) string { return "userrefreshfams:" + uid }
func authVersionKey(uid string) string         { return "authversion:" + uid }
func blacklistKey(hash string) string          { return "blacklist:" + hash }

func parseRefreshValue(raw string) (familyID, userID, audience string, version int64, err error) {
	parts := strings.Split(raw, "|")
	if (len(parts) != 2 && len(parts) != 3 && len(parts) != 4) || parts[0] == "" || parts[1] == "" {
		return "", "", "", 0, fmt.Errorf("malformed refresh value")
	}
	if len(parts) == 2 {
		return parts[0], parts[1], "", 0, nil
	}
	if len(parts) == 3 {
		return parts[0], parts[1], parts[2], 0, nil
	}
	version, err = strconv.ParseInt(parts[3], 10, 64)
	if err != nil || version < 0 {
		return "", "", "", 0, fmt.Errorf("malformed refresh auth version")
	}
	return parts[0], parts[1], parts[2], version, nil
}
