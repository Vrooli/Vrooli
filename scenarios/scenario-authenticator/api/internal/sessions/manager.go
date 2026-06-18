// Package sessions owns the Redis-backed hot state of authentication: live
// sessions, the refresh-token-family state (with single-use rotation + reuse
// detection), and the access-token blacklist. Key shapes are ported from the
// old auth/session.go; the refresh-family reuse-detection is NEW (the old code
// rotated but never detected replay of a rotated token).
//
// Refresh-family model (reuse detection):
//   - refresh:<H(token)>      → "<familyID>|<userID>"  (a currently-valid token)
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
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
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
	id, err := authcrypto.GenerateRefreshToken() // 32-byte random id, same generator
	if err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	now := m.now().UTC()
	sess := Session{
		ID:        id,
		UserID:    userID,
		IPAddress: ip,
		UserAgent: userAgent,
		CreatedAt: now,
		ExpiresAt: now.Add(sessionTTL),
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
			_ = m.store.SRem(ctx, userSessionsKey(s.UserID), sessionID)
		}
	}
	return m.store.Del(ctx, sessionKey(sessionID))
}

// RevokeAllSessions drops every session for a user and returns how many were
// removed.
func (m *Manager) RevokeAllSessions(ctx context.Context, userID string) (int, error) {
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
	return len(ids), nil
}

// ---- refresh-token family -------------------------------------------------

// IssueRefresh mints a fresh refresh token in a NEW family for userID and
// returns the token (the family id is internal).
func (m *Manager) IssueRefresh(ctx context.Context, userID string) (string, error) {
	familyID, err := authcrypto.GenerateSecureToken(16)
	if err != nil {
		return "", err
	}
	return m.mintInFamily(ctx, userID, familyID)
}

// RotateRefresh validates a presented refresh token, rotates it single-use, and
// returns the new token + the owning userID. Replaying an already-rotated token
// revokes the whole family and returns ErrRefreshReuse.
func (m *Manager) RotateRefresh(ctx context.Context, presented string) (newToken, userID string, err error) {
	hash := authcrypto.HashToken(presented)

	raw, ok, err := m.store.Get(ctx, refreshKey(hash))
	if err != nil {
		return "", "", err
	}
	if ok {
		familyID, uid, perr := parseRefreshValue(raw)
		if perr != nil {
			return "", "", ErrInvalidRefresh
		}
		// Single-use rotation: retire the presented token, mint its successor.
		if err := m.store.Del(ctx, refreshKey(hash)); err != nil {
			return "", "", err
		}
		if err := m.store.Set(ctx, refreshUsedKey(hash), familyID, refreshTTL); err != nil {
			return "", "", err
		}
		nt, err := m.mintInFamily(ctx, uid, familyID)
		if err != nil {
			return "", "", err
		}
		return nt, uid, nil
	}

	// Not a live token. If it was rotated out, this is a replay → revoke family.
	if famID, used, err := m.store.Get(ctx, refreshUsedKey(hash)); err == nil && used {
		_ = m.revokeFamily(ctx, famID)
		return "", "", ErrRefreshReuse
	}
	return "", "", ErrInvalidRefresh
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
	famID, _, perr := parseRefreshValue(raw)
	if perr != nil {
		return nil
	}
	return m.revokeFamily(ctx, famID)
}

func (m *Manager) mintInFamily(ctx context.Context, userID, familyID string) (string, error) {
	token, err := authcrypto.GenerateRefreshToken()
	if err != nil {
		return "", err
	}
	hash := authcrypto.HashToken(token)
	if err := m.store.Set(ctx, refreshKey(hash), familyID+"|"+userID, refreshTTL); err != nil {
		return "", err
	}
	if err := m.store.SAdd(ctx, refreshFamilyKey(familyID), hash); err != nil {
		return "", err
	}
	if err := m.store.Expire(ctx, refreshFamilyKey(familyID), refreshTTL); err != nil {
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

func sessionKey(id string) string            { return "session:" + id }
func userSessionsKey(uid string) string      { return "usersessions:" + uid }
func refreshKey(hash string) string          { return "refresh:" + hash }
func refreshUsedKey(hash string) string      { return "refreshused:" + hash }
func refreshFamilyKey(fid string) string     { return "refreshfam:" + fid }
func refreshFamilyDeadKey(fid string) string { return "refreshfamdead:" + fid }
func blacklistKey(hash string) string        { return "blacklist:" + hash }

func parseRefreshValue(raw string) (familyID, userID string, err error) {
	parts := strings.SplitN(raw, "|", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("malformed refresh value")
	}
	return parts[0], parts[1], nil
}
