package main

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"time"
)

const (
	remoteProfileCookieName    = "admin_session"
	remoteProfileStatusUnknown = "unknown"
	remoteProfileStatusActive  = "active"
	remoteProfileStatusExpired = "expired"
	remoteProfileStatusError   = "error"
)

// remoteProfileSessionCookie describes the admin session credential used for
// an outbound remote-profile request. Request.AddCookie transmits only its
// name and value, but keeping the policy on the cookie value prevents this
// call path from becoming an insecure-cookie exception as it evolves.
func remoteProfileSessionCookie(value string) *http.Cookie {
	return &http.Cookie{
		Name:     remoteProfileCookieName,
		Value:    value,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
}

const (
	remoteProfileProxyBodyLimit   = 1 << 20 // 1MB
	remoteProfileProxyResponseMax = 2 << 20 // 2MB
)

var (
	ErrRemoteProfileNotFound       = errors.New("remote profile not found")
	ErrRemoteProfileTagExists      = errors.New("remote profile tag already exists")
	ErrRemoteProfileInvalid        = errors.New("invalid remote profile")
	ErrRemoteProfileSessionMissing = errors.New("remote profile session missing")
	ErrRemoteProfileDisallowedPath = errors.New("remote proxy path not allowed")
)

var remoteProfileTagPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-_]{0,63}$`)

var remoteProfileProxyAllowlist = []string{
	"/admin/download-storage",
	"/admin/download-artifacts",
	"/admin/download-assets",
	"/admin/download-apps",
}

var remoteProfileProxyAllowedHeaders = map[string]bool{
	"accept":        true,
	"content-type":  true,
	"if-match":      true,
	"if-none-match": true,
}

// RemoteProfile represents a stored connection to a remote LPBS deployment.
type RemoteProfile struct {
	ID               int64      `json:"id"`
	Tag              string     `json:"tag"`
	Label            *string    `json:"label,omitempty"`
	APIBase          string     `json:"api_base"`
	ConnectorID      string     `json:"connector_id,omitempty"`
	RemoteSessionID  *string    `json:"remote_session_id,omitempty"`
	Status           string     `json:"status"`
	HasSession       bool       `json:"has_session"`
	SessionExpiresAt *time.Time `json:"session_expires_at,omitempty"`
	LastLoginAt      *time.Time `json:"last_login_at,omitempty"`
	LastUsedAt       *time.Time `json:"last_used_at,omitempty"`
	CreatedBy        *int64     `json:"created_by,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// RemoteProfileCreateRequest defines the payload to create a remote profile.
type RemoteProfileCreateRequest struct {
	Tag     string `json:"tag"`
	Label   string `json:"label"`
	APIBase string `json:"api_base"`
}

// RemoteProfileUpdateRequest defines the payload to update a remote profile.
type RemoteProfileUpdateRequest struct {
	Tag     *string `json:"tag,omitempty"`
	Label   *string `json:"label,omitempty"`
	APIBase *string `json:"api_base,omitempty"`
}

// RemoteProfileLoginRequest defines the payload for remote login.
type RemoteProfileLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RemoteProfileProxyRequest defines the payload for proxying admin requests.
type RemoteProfileProxyRequest struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Query   map[string]string `json:"query,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    json.RawMessage   `json:"body,omitempty"`
}

// RemoteProxyResponse is the result of a proxied request.
type RemoteProxyResponse struct {
	StatusCode  int
	Body        []byte
	ContentType string
}

type IncomingRemoteProfileSession struct {
	SessionID    string    `json:"session_id"`
	AdminEmail   string    `json:"admin_email"`
	ConnectorID  string    `json:"connector_id"`
	ProfileTag   string    `json:"profile_tag,omitempty"`
	Origin       string    `json:"origin,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	LastActivity time.Time `json:"last_activity"`
	ExpiresAt    time.Time `json:"expires_at"`
	IPAddress    *string   `json:"ip_address,omitempty"`
	UserAgent    *string   `json:"user_agent,omitempty"`
}

type RemoteProfileSessionLinks struct {
	ProfileID             int64                          `json:"profile_id"`
	ProfileTag            string                         `json:"profile_tag"`
	ConnectorID           string                         `json:"connector_id"`
	LocalHasSession       bool                           `json:"local_has_session"`
	LocalStatus           string                         `json:"local_status"`
	LocalSessionExpiresAt *time.Time                     `json:"local_session_expires_at,omitempty"`
	RemoteSessionID       *string                        `json:"remote_session_id,omitempty"`
	RemoteSessions        []IncomingRemoteProfileSession `json:"remote_sessions"`
}

// RemoteProfileError wraps errors with HTTP status + type for handlers.
type RemoteProfileError struct {
	Status    int
	ErrorType string
	Message   string
}

// RemoteProfileStore is the context-aware persistence contract for remote
// profile configuration and encrypted remote sessions.
//
// seam: RemoteProfileStore keeps remote-profile persistence independent of a
// concrete pool and preserves request-scoped test isolation.
type RemoteProfileStore interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func (e *RemoteProfileError) Error() string {
	return e.Message
}

// RemoteProfileService manages remote profile storage and remote admin sessions.
type RemoteProfileService struct {
	db            RemoteProfileStore
	encryptionKey []byte
	httpClient    HTTPDoer
	now           func() time.Time
	dialects      *DialectHelper
}

// NewRemoteProfileService creates a RemoteProfileService with defaults.
func NewRemoteProfileService(db RemoteProfileStore) (*RemoteProfileService, error) {
	client := &http.Client{
		Timeout: 20 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return NewRemoteProfileServiceWithOptions(db, client, "postgres")
}

// NewRemoteProfileServiceWithOptions creates a RemoteProfileService with a custom client/dialect.
func NewRemoteProfileServiceWithOptions(db RemoteProfileStore, client HTTPDoer, dialect string) (*RemoteProfileService, error) {
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	key, err := loadRemoteProfileEncryptionKey()
	if err != nil {
		return nil, err
	}
	now := time.Now
	return &RemoteProfileService{
		db:            db,
		encryptionKey: key,
		httpClient:    client,
		now:           now,
		dialects:      NewDialectHelper(dialect),
	}, nil
}

func loadRemoteProfileEncryptionKey() ([]byte, error) {
	keyStr := resolveSecret("LPBS_REMOTE_PROFILE_ENCRYPTION_KEY")
	keySource := "LPBS_REMOTE_PROFILE_ENCRYPTION_KEY"
	if keyStr == "" {
		fallback := resolveSecret("LPBS_API_KEY_ENCRYPTION_KEY")
		if fallback != "" {
			keyStr = fallback
			keySource = "LPBS_API_KEY_ENCRYPTION_KEY"
			logStructured("remote_profiles_encryption_key_fallback", map[string]interface{}{
				"level":    "warn",
				"message":  "LPBS_REMOTE_PROFILE_ENCRYPTION_KEY not set; falling back to LPBS_API_KEY_ENCRYPTION_KEY",
				"security": true,
				"action":   "Set LPBS_REMOTE_PROFILE_ENCRYPTION_KEY before production use",
			})
		}
	}
	if keyStr == "" {
		if isProductionEnvironment() {
			return nil, fmt.Errorf(
				"LPBS_REMOTE_PROFILE_ENCRYPTION_KEY is required in production. " +
					"Set LPBS_REMOTE_PROFILE_ENCRYPTION_KEY (preferred) or LPBS_API_KEY_ENCRYPTION_KEY as a fallback",
			)
		}
		logStructured("remote_profiles_no_encryption_key_dev", map[string]interface{}{
			"level":    "warn",
			"message":  "Remote profile sessions will be stored unencrypted (development mode)",
			"security": true,
			"action":   "Set LPBS_REMOTE_PROFILE_ENCRYPTION_KEY before deploying to production",
		})
		return nil, nil
	}
	key, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", keySource, err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("%s must be 32 bytes (got %d)", keySource, len(key))
	}
	return key, nil
}

func (s *RemoteProfileService) encrypt(plaintext string) (string, error) {
	if s.encryptionKey == nil {
		return plaintext, nil
	}
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (s *RemoteProfileService) decrypt(ciphertext string) (string, error) {
	if s.encryptionKey == nil {
		return ciphertext, nil
	}
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func normalizeRemoteProfileTag(tag string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(tag))
	if normalized == "" {
		return "", fmt.Errorf("tag is required")
	}
	if !remoteProfileTagPattern.MatchString(normalized) {
		return "", fmt.Errorf("tag must be 1-64 chars of lowercase letters, numbers, '-' or '_' and start with a letter/number")
	}
	return normalized, nil
}

func normalizeRemoteProfileLabel(label string) *string {
	trimmed := strings.TrimSpace(label)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func normalizeRemoteProfileSessionID(sessionID string) string {
	return strings.TrimSpace(sessionID)
}

func normalizeRemoteProfileAPIBase(raw string) (string, error) {
	clean, err := ValidateURL(raw)
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(clean)
	if err != nil || parsed.Host == "" {
		return "", ErrURLInvalid
	}
	if parsed.User != nil {
		return "", fmt.Errorf("api_base must not include credentials")
	}
	if isProductionEnvironment() && parsed.Scheme != "https" {
		return "", fmt.Errorf("api_base must use https in production")
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	pathValue := strings.TrimSuffix(parsed.Path, "/")
	if !strings.HasSuffix(pathValue, "/api/v1") {
		return "", fmt.Errorf("api_base must end with /api/v1")
	}
	parsed.Path = pathValue
	return strings.TrimSuffix(parsed.String(), "/"), nil
}

func generateRemoteConnectorID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func remoteProfileOriginLabel() string {
	if value := sanitizeRemoteProfileSessionMetaValue(resolveSecret("LPBS_REMOTE_PROFILE_ORIGIN")); value != "" {
		return value
	}
	if host, err := os.Hostname(); err == nil {
		if value := sanitizeRemoteProfileSessionMetaValue(host); value != "" {
			return value
		}
	}
	return "unknown"
}

func normalizeRemoteProxyPath(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("path is required")
	}
	if strings.Contains(trimmed, "://") {
		return "", fmt.Errorf("path must be relative")
	}
	if strings.Contains(trimmed, "\\") {
		return "", fmt.Errorf("path must not include backslashes")
	}
	if strings.Contains(trimmed, "?") || strings.Contains(trimmed, "#") {
		return "", fmt.Errorf("path must not include query or fragment")
	}
	if !strings.HasPrefix(trimmed, "/") {
		return "", fmt.Errorf("path must start with '/'")
	}
	if strings.Contains(trimmed, "..") {
		return "", fmt.Errorf("path must not include '..'")
	}
	cleaned := path.Clean(trimmed)
	if !strings.HasPrefix(cleaned, "/admin/") && cleaned != "/admin" {
		return "", fmt.Errorf("path must start with /admin")
	}
	return cleaned, nil
}

func isAllowedRemoteProxyPath(path string) bool {
	for _, allowed := range remoteProfileProxyAllowlist {
		if path == allowed || strings.HasPrefix(path, allowed+"/") {
			return true
		}
	}
	return false
}

func (s *RemoteProfileService) List(ctx context.Context) ([]RemoteProfile, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, tag, label, api_base, connector_id, remote_session_id, status, encrypted_session,
		       session_expires_at, remote_session_last_synced_at, last_login_at, last_used_at,
		       created_by, created_at, updated_at
		FROM remote_profiles
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	profiles := []RemoteProfile{}
	for rows.Next() {
		rec, err := scanRemoteProfile(rows)
		if err != nil {
			return nil, err
		}
		connectorID := remoteProfileConnectorID(rec)
		if connectorID == "" {
			connectorID, ensureErr := s.ensureConnectorID(ctx, rec.ID, connectorID)
			if ensureErr != nil {
				return nil, ensureErr
			}
			rec.ConnectorID = sql.NullString{String: connectorID, Valid: true}
		}
		profiles = append(profiles, rec.toProfile(s.nowTime()))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return profiles, nil
}

func (s *RemoteProfileService) Create(ctx context.Context, req RemoteProfileCreateRequest, createdByEmail string) (*RemoteProfile, error) {
	tag, err := normalizeRemoteProfileTag(req.Tag)
	if err != nil {
		return nil, err
	}
	apiBase, err := normalizeRemoteProfileAPIBase(req.APIBase)
	if err != nil {
		return nil, err
	}
	label := normalizeRemoteProfileLabel(req.Label)
	connectorID, err := generateRemoteConnectorID()
	if err != nil {
		return nil, err
	}

	exists, err := s.tagExists(ctx, tag, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrRemoteProfileTagExists
	}

	createdByID, err := s.lookupAdminID(ctx, createdByEmail)
	if err != nil {
		return nil, err
	}

	var id int64
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO remote_profiles (tag, label, api_base, connector_id, status, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING id
	`, tag, StringToNullString(label), apiBase, connectorID, remoteProfileStatusUnknown, Int64ToNullInt64(createdByID)).Scan(&id)
	if err != nil {
		return nil, err
	}

	profile, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	logStructured("remote_profile_created", map[string]interface{}{
		"level": "info",
		"id":    profile.ID,
		"tag":   profile.Tag,
	})

	return profile, nil
}

func (s *RemoteProfileService) Update(ctx context.Context, id int64, req RemoteProfileUpdateRequest) (*RemoteProfile, error) {
	rec, err := s.getRecordByID(ctx, id)
	if err != nil {
		return nil, err
	}

	updatedTag := rec.Tag
	if req.Tag != nil {
		updatedTag, err = normalizeRemoteProfileTag(*req.Tag)
		if err != nil {
			return nil, err
		}
		if updatedTag != rec.Tag {
			exists, err := s.tagExists(ctx, updatedTag, id)
			if err != nil {
				return nil, err
			}
			if exists {
				return nil, ErrRemoteProfileTagExists
			}
		}
	}

	updatedLabel := rec.Label
	if req.Label != nil {
		updatedLabel = StringToNullString(normalizeRemoteProfileLabel(*req.Label))
	}

	updatedAPIBase := rec.APIBase
	if req.APIBase != nil {
		updatedAPIBase, err = normalizeRemoteProfileAPIBase(*req.APIBase)
		if err != nil {
			return nil, err
		}
	}

	apiBaseChanged := updatedAPIBase != rec.APIBase
	if apiBaseChanged {
		_, err = s.db.ExecContext(ctx, `
			UPDATE remote_profiles
			SET tag = $1,
			    label = $2,
			    api_base = $3,
			    encrypted_session = NULL,
			    remote_session_id = NULL,
			    remote_session_last_synced_at = NOW(),
			    session_expires_at = NULL,
			    status = $4,
			    last_used_at = NOW(),
			    updated_at = NOW()
			WHERE id = $5
		`, updatedTag, updatedLabel, updatedAPIBase, remoteProfileStatusUnknown, id)
	} else {
		_, err = s.db.ExecContext(ctx, `
			UPDATE remote_profiles
			SET tag = $1, label = $2, api_base = $3, updated_at = NOW()
			WHERE id = $4
		`, updatedTag, updatedLabel, updatedAPIBase, id)
	}
	if err != nil {
		return nil, err
	}

	profile, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	logStructured("remote_profile_updated", map[string]interface{}{
		"level": "info",
		"id":    profile.ID,
		"tag":   profile.Tag,
	})

	return profile, nil
}

func (s *RemoteProfileService) Delete(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM remote_profiles WHERE id = $1`, id)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrRemoteProfileNotFound
	}

	logStructured("remote_profile_deleted", map[string]interface{}{
		"level": "info",
		"id":    id,
	})

	return nil
}

func (s *RemoteProfileService) Login(ctx context.Context, id int64, email string, password string) (*RemoteProfile, error) {
	email = strings.TrimSpace(email)
	password = strings.TrimSpace(password)
	if email == "" || password == "" {
		return nil, ErrRemoteProfileInvalid
	}

	rec, err := s.getRecordByID(ctx, id)
	if err != nil {
		return nil, err
	}
	connectorID, err := s.ensureConnectorID(ctx, id, remoteProfileConnectorID(rec))
	if err != nil {
		return nil, err
	}
	rec.ConnectorID = sql.NullString{String: connectorID, Valid: true}

	meta := RemoteProfileSessionMetadata{
		ConnectorID: connectorID,
		ProfileTag:  rec.Tag,
		Origin:      remoteProfileOriginLabel(),
	}
	sessionValue, remoteSessionID, expiresAt, err := s.remoteLogin(ctx, rec.APIBase, email, password, meta)
	if err != nil {
		return nil, err
	}

	if err := s.setSession(ctx, id, sessionValue, remoteSessionID, expiresAt); err != nil {
		return nil, err
	}

	profile, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	logStructured("remote_profile_login", map[string]interface{}{
		"level": "info",
		"id":    profile.ID,
		"tag":   profile.Tag,
	})

	return profile, nil
}

func (s *RemoteProfileService) Logout(ctx context.Context, id int64) (*RemoteProfile, error) {
	rec, err := s.getRecordByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if rec.EncryptedSession.Valid && rec.EncryptedSession.String != "" {
		sessionValue, err := s.decrypt(rec.EncryptedSession.String)
		if err != nil {
			return nil, err
		}
		if sessionValue != "" {
			if err := s.remoteLogout(ctx, rec.APIBase, sessionValue); err != nil {
				logStructuredError("remote_profile_remote_logout_failed", map[string]interface{}{
					"error": err.Error(),
					"id":    id,
					"tag":   rec.Tag,
				})
			}
		}
	}

	if err := s.clearSession(ctx, id, remoteProfileStatusExpired); err != nil {
		return nil, err
	}

	profile, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	logStructured("remote_profile_logout", map[string]interface{}{
		"level": "info",
		"id":    profile.ID,
		"tag":   profile.Tag,
	})

	return profile, nil
}

func (s *RemoteProfileService) Test(ctx context.Context, id int64) (*RemoteProfile, error) {
	rec, err := s.getRecordByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if !rec.EncryptedSession.Valid || rec.EncryptedSession.String == "" {
		return nil, ErrRemoteProfileSessionMissing
	}

	sessionValue, err := s.decrypt(rec.EncryptedSession.String)
	if err != nil {
		return nil, err
	}
	if sessionValue == "" {
		return nil, ErrRemoteProfileSessionMissing
	}

	authenticated, err := s.remoteSessionCheck(ctx, rec.APIBase, sessionValue)
	if err != nil {
		_ = s.updateStatus(ctx, id, remoteProfileStatusError)
		return nil, err
	}

	if authenticated {
		if err := s.updateStatus(ctx, id, remoteProfileStatusActive); err != nil {
			return nil, err
		}
	} else {
		if err := s.clearSession(ctx, id, remoteProfileStatusExpired); err != nil {
			return nil, err
		}
		return nil, &RemoteProfileError{
			Status:    http.StatusUnauthorized,
			ErrorType: ApiErrorTypeUnauthorized,
			Message:   "Remote session expired. Please log in again.",
		}
	}

	profile, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return profile, nil
}

func (s *RemoteProfileService) SessionLinks(ctx context.Context, id int64) (*RemoteProfileSessionLinks, error) {
	rec, err := s.getRecordByID(ctx, id)
	if err != nil {
		return nil, err
	}

	links := &RemoteProfileSessionLinks{
		ProfileID:             rec.ID,
		ProfileTag:            rec.Tag,
		ConnectorID:           remoteProfileConnectorID(rec),
		LocalHasSession:       rec.EncryptedSession.Valid && strings.TrimSpace(rec.EncryptedSession.String) != "",
		LocalStatus:           rec.Status,
		LocalSessionExpiresAt: NullTimeValue(rec.SessionExpiresAt),
		RemoteSessionID:       NullStringValue(rec.RemoteSessionID),
		RemoteSessions:        []IncomingRemoteProfileSession{},
	}
	if !links.LocalHasSession {
		return links, nil
	}

	sessionValue, err := s.decrypt(rec.EncryptedSession.String)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(sessionValue) == "" {
		return links, nil
	}

	sessions, err := s.listIncomingRemoteSessions(ctx, rec.APIBase, sessionValue, remoteProfileConnectorID(rec))
	if err != nil {
		var remoteErr *RemoteProfileError
		if errors.As(err, &remoteErr) && remoteErr.Status == http.StatusUnauthorized {
			_ = s.clearSession(ctx, id, remoteProfileStatusExpired)
		}
		return nil, err
	}
	links.RemoteSessions = sessions
	return links, nil
}

func (s *RemoteProfileService) RevokeRemoteSessions(ctx context.Context, id int64) (*RemoteProfileSessionLinks, error) {
	rec, err := s.getRecordByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !rec.EncryptedSession.Valid || strings.TrimSpace(rec.EncryptedSession.String) == "" {
		return nil, ErrRemoteProfileSessionMissing
	}

	sessionValue, err := s.decrypt(rec.EncryptedSession.String)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(sessionValue) == "" {
		return nil, ErrRemoteProfileSessionMissing
	}

	sessions, err := s.listIncomingRemoteSessions(ctx, rec.APIBase, sessionValue, remoteProfileConnectorID(rec))
	if err != nil {
		return nil, err
	}
	for _, session := range sessions {
		if revokeErr := s.revokeIncomingRemoteSession(ctx, rec.APIBase, sessionValue, session.SessionID); revokeErr != nil {
			return nil, revokeErr
		}
	}

	if err := s.clearSession(ctx, id, remoteProfileStatusExpired); err != nil {
		return nil, err
	}
	return s.SessionLinks(ctx, id)
}

func (s *RemoteProfileService) Proxy(ctx context.Context, id int64, req RemoteProfileProxyRequest) (*RemoteProxyResponse, error) {
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		return nil, &RemoteProfileError{Status: http.StatusBadRequest, ErrorType: ApiErrorTypeValidation, Message: "method is required"}
	}
	allowedMethods := map[string]bool{"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true}
	if !allowedMethods[method] {
		return nil, &RemoteProfileError{Status: http.StatusBadRequest, ErrorType: ApiErrorTypeValidation, Message: "unsupported method"}
	}

	pathValue, err := normalizeRemoteProxyPath(req.Path)
	if err != nil {
		return nil, &RemoteProfileError{Status: http.StatusBadRequest, ErrorType: ApiErrorTypeValidation, Message: err.Error()}
	}
	if !isAllowedRemoteProxyPath(pathValue) {
		return nil, ErrRemoteProfileDisallowedPath
	}

	rec, err := s.getRecordByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !rec.EncryptedSession.Valid || rec.EncryptedSession.String == "" {
		return nil, ErrRemoteProfileSessionMissing
	}
	sessionValue, err := s.decrypt(rec.EncryptedSession.String)
	if err != nil {
		return nil, err
	}
	if sessionValue == "" {
		return nil, ErrRemoteProfileSessionMissing
	}

	remoteURL, err := s.buildRemoteURL(rec.APIBase, pathValue, req.Query)
	if err != nil {
		return nil, &RemoteProfileError{Status: http.StatusBadRequest, ErrorType: ApiErrorTypeValidation, Message: err.Error()}
	}

	var body io.Reader
	if len(req.Body) > 0 {
		body = bytes.NewReader(req.Body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, remoteURL, body)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "application/json")
	for key, value := range req.Headers {
		keyLower := strings.ToLower(strings.TrimSpace(key))
		if keyLower == "" || !remoteProfileProxyAllowedHeaders[keyLower] {
			continue
		}
		if strings.TrimSpace(value) == "" {
			continue
		}
		httpReq.Header.Set(key, value)
	}
	if len(req.Body) > 0 && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	httpReq.AddCookie(remoteProfileSessionCookie(sessionValue))

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		_ = s.updateStatus(ctx, id, remoteProfileStatusError)
		return nil, classifyRemoteError(err)
	}
	defer resp.Body.Close()

	bodyBytes, readErr := readLimitedBody(resp.Body, remoteProfileProxyResponseMax)
	if readErr != nil {
		_ = s.updateStatus(ctx, id, remoteProfileStatusError)
		return nil, readErr
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		_ = s.clearSession(ctx, id, remoteProfileStatusExpired)
	} else if resp.StatusCode >= 500 {
		_ = s.updateStatus(ctx, id, remoteProfileStatusError)
	} else {
		_ = s.updateStatus(ctx, id, remoteProfileStatusActive)
	}

	contentType := resp.Header.Get("Content-Type")
	return &RemoteProxyResponse{
		StatusCode:  resp.StatusCode,
		Body:        bodyBytes,
		ContentType: contentType,
	}, nil
}

func (s *RemoteProfileService) GetByID(ctx context.Context, id int64) (*RemoteProfile, error) {
	rec, err := s.getRecordByID(ctx, id)
	if err != nil {
		return nil, err
	}
	profile := rec.toProfile(s.nowTime())
	return &profile, nil
}

func (s *RemoteProfileService) getRecordByID(ctx context.Context, id int64) (*remoteProfileRecord, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, tag, label, api_base, connector_id, remote_session_id, status, encrypted_session,
		       session_expires_at, remote_session_last_synced_at, last_login_at, last_used_at,
		       created_by, created_at, updated_at
		FROM remote_profiles
		WHERE id = $1
	`, id)
	rec, err := scanRemoteProfileRow(row)
	if err != nil {
		return nil, err
	}
	connectorID, err := s.ensureConnectorID(ctx, id, remoteProfileConnectorID(rec))
	if err != nil {
		return nil, err
	}
	rec.ConnectorID = sql.NullString{String: connectorID, Valid: true}
	return rec, nil
}

func (s *RemoteProfileService) tagExists(ctx context.Context, tag string, excludeID int64) (bool, error) {
	var count int
	if excludeID > 0 {
		if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM remote_profiles WHERE tag = $1 AND id <> $2`, tag, excludeID).Scan(&count); err != nil {
			return false, err
		}
	} else {
		if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM remote_profiles WHERE tag = $1`, tag).Scan(&count); err != nil {
			return false, err
		}
	}
	return count > 0, nil
}

func (s *RemoteProfileService) ensureConnectorID(ctx context.Context, id int64, current string) (string, error) {
	trimmed := strings.TrimSpace(current)
	if trimmed != "" {
		return trimmed, nil
	}

	connectorID, err := generateRemoteConnectorID()
	if err != nil {
		return "", err
	}

	if _, err := s.db.ExecContext(ctx, `
		UPDATE remote_profiles
		SET connector_id = $1,
		    updated_at = NOW()
		WHERE id = $2 AND (connector_id IS NULL OR connector_id = '')
	`, connectorID, id); err != nil {
		return "", err
	}

	var resolved sql.NullString
	if err := s.db.QueryRowContext(ctx, `SELECT connector_id FROM remote_profiles WHERE id = $1`, id).Scan(&resolved); err != nil {
		return "", err
	}
	if !resolved.Valid || strings.TrimSpace(resolved.String) == "" {
		return "", fmt.Errorf("connector id missing for remote profile %d", id)
	}
	return strings.TrimSpace(resolved.String), nil
}

func (s *RemoteProfileService) lookupAdminID(ctx context.Context, email string) (*int64, error) {
	trimmed := strings.TrimSpace(email)
	if trimmed == "" {
		return nil, nil
	}
	var id int64
	err := s.db.QueryRowContext(ctx, `SELECT id FROM admin_users WHERE LOWER(email) = LOWER($1)`, trimmed).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func (s *RemoteProfileService) setSession(ctx context.Context, id int64, sessionValue string, remoteSessionID string, expiresAt *time.Time) error {
	encrypted, err := s.encrypt(sessionValue)
	if err != nil {
		return err
	}
	normalizedRemoteSessionID := normalizeRemoteProfileSessionID(remoteSessionID)
	var remoteSessionIDPtr *string
	if normalizedRemoteSessionID != "" {
		remoteSessionIDPtr = &normalizedRemoteSessionID
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE remote_profiles
		SET encrypted_session = $1,
		    remote_session_id = $2,
		    remote_session_last_synced_at = NOW(),
		    session_expires_at = $3,
		    status = $4,
		    last_login_at = NOW(),
		    last_used_at = NOW(),
		    updated_at = NOW()
		WHERE id = $5
	`, encrypted, StringToNullString(remoteSessionIDPtr), TimeToNullTime(expiresAt), remoteProfileStatusActive, id)
	return err
}

func (s *RemoteProfileService) clearSession(ctx context.Context, id int64, status string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE remote_profiles
		SET encrypted_session = NULL,
		    remote_session_id = NULL,
		    remote_session_last_synced_at = NOW(),
		    session_expires_at = NULL,
		    status = $1,
		    last_used_at = NOW(),
		    updated_at = NOW()
		WHERE id = $2
	`, status, id)
	return err
}

func (s *RemoteProfileService) updateStatus(ctx context.Context, id int64, status string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE remote_profiles
		SET status = $1,
		    last_used_at = NOW(),
		    updated_at = NOW()
		WHERE id = $2
	`, status, id)
	return err
}

func (s *RemoteProfileService) buildRemoteURL(apiBase string, pathValue string, query map[string]string) (string, error) {
	parsed, err := url.Parse(apiBase)
	if err != nil {
		return "", err
	}
	parsed.Path = strings.TrimSuffix(parsed.Path, "/") + pathValue
	values := url.Values{}
	for key, value := range query {
		if strings.TrimSpace(key) == "" {
			continue
		}
		values.Set(key, value)
	}
	parsed.RawQuery = values.Encode()
	return parsed.String(), nil
}

func (s *RemoteProfileService) remoteLogin(ctx context.Context, apiBase string, email string, password string, metadata RemoteProfileSessionMetadata) (string, string, *time.Time, error) {
	// #nosec G117 -- this password is intentionally sent only to a configured remote
	// admin login endpoint; production profile validation requires an HTTPS API base.
	payload, err := json.Marshal(LoginRequest{Email: email, Password: password})
	if err != nil {
		return "", "", nil, err
	}
	urlValue := strings.TrimSuffix(apiBase, "/") + "/admin/login"
	headers := map[string]string{
		"User-Agent": buildRemoteProfileSessionUserAgent(metadata),
	}
	resp, body, err := s.doJSONRequestWithHeaders(ctx, http.MethodPost, urlValue, payload, nil, headers)
	if err != nil {
		return "", "", nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := extractRemoteErrorMessage(body)
		return "", "", nil, &RemoteProfileError{
			Status:    mapRemoteStatus(resp.StatusCode),
			ErrorType: inferErrorType(resp.StatusCode),
			Message:   message,
		}
	}

	cookie := findCookie(resp.Cookies(), remoteProfileCookieName)
	if cookie == nil || cookie.Value == "" {
		return "", "", nil, &RemoteProfileError{
			Status:    http.StatusBadGateway,
			ErrorType: ApiErrorTypeServerError,
			Message:   "Remote login did not return a session cookie",
		}
	}
	var sessionResp LoginResponse
	if err := json.Unmarshal(body, &sessionResp); err != nil {
		return "", "", nil, err
	}
	if !sessionResp.Authenticated {
		return "", "", nil, &RemoteProfileError{
			Status:    http.StatusUnauthorized,
			ErrorType: ApiErrorTypeUnauthorized,
			Message:   "Remote login failed",
		}
	}

	authenticated, err := s.remoteSessionCheck(ctx, apiBase, cookie.Value)
	if err != nil {
		return "", "", nil, err
	}
	if !authenticated {
		return "", "", nil, &RemoteProfileError{
			Status:    http.StatusUnauthorized,
			ErrorType: ApiErrorTypeUnauthorized,
			Message:   "Remote session verification failed",
		}
	}

	expiresAt := deriveCookieExpiry(cookie, s.nowTime())
	return cookie.Value, normalizeRemoteProfileSessionID(sessionResp.SessionID), expiresAt, nil
}

func (s *RemoteProfileService) nowTime() time.Time {
	if s == nil || s.now == nil {
		return time.Now()
	}
	return s.now()
}

func (s *RemoteProfileService) remoteSessionCheck(ctx context.Context, apiBase string, sessionValue string) (bool, error) {
	urlValue := strings.TrimSuffix(apiBase, "/") + "/admin/session"
	cookies := []*http.Cookie{remoteProfileSessionCookie(sessionValue)}
	resp, body, err := s.doJSONRequest(ctx, http.MethodGet, urlValue, nil, cookies)
	if err != nil {
		return false, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return false, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := extractRemoteErrorMessage(body)
		return false, &RemoteProfileError{
			Status:    mapRemoteStatus(resp.StatusCode),
			ErrorType: inferErrorType(resp.StatusCode),
			Message:   message,
		}
	}
	var sessionResp LoginResponse
	if err := json.Unmarshal(body, &sessionResp); err != nil {
		return false, err
	}
	return sessionResp.Authenticated, nil
}

func (s *RemoteProfileService) remoteLogout(ctx context.Context, apiBase string, sessionValue string) error {
	urlValue := strings.TrimSuffix(apiBase, "/") + "/admin/logout"
	cookies := []*http.Cookie{remoteProfileSessionCookie(sessionValue)}
	resp, _, err := s.doJSONRequest(ctx, http.MethodPost, urlValue, nil, cookies)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &RemoteProfileError{
			Status:    mapRemoteStatus(resp.StatusCode),
			ErrorType: inferErrorType(resp.StatusCode),
			Message:   "Remote logout failed",
		}
	}
	return nil
}

func (s *RemoteProfileService) listIncomingRemoteSessions(ctx context.Context, apiBase string, sessionValue string, connectorID string) ([]IncomingRemoteProfileSession, error) {
	query := url.Values{}
	if trimmed := strings.TrimSpace(connectorID); trimmed != "" {
		query.Set("connector_id", trimmed)
	}
	urlValue := strings.TrimSuffix(apiBase, "/") + "/admin/remote-profile-sessions"
	if encoded := query.Encode(); encoded != "" {
		urlValue += "?" + encoded
	}

	cookies := []*http.Cookie{remoteProfileSessionCookie(sessionValue)}
	resp, body, err := s.doJSONRequest(ctx, http.MethodGet, urlValue, nil, cookies)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, &RemoteProfileError{
			Status:    http.StatusUnauthorized,
			ErrorType: ApiErrorTypeUnauthorized,
			Message:   "Remote session expired. Please log in again.",
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &RemoteProfileError{
			Status:    mapRemoteStatus(resp.StatusCode),
			ErrorType: inferErrorType(resp.StatusCode),
			Message:   extractRemoteErrorMessage(body),
		}
	}

	var payload struct {
		Sessions []IncomingRemoteProfileSession `json:"sessions"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if payload.Sessions == nil {
		payload.Sessions = []IncomingRemoteProfileSession{}
	}
	return payload.Sessions, nil
}

func (s *RemoteProfileService) revokeIncomingRemoteSession(ctx context.Context, apiBase string, sessionValue string, sessionID string) error {
	urlValue := strings.TrimSuffix(apiBase, "/") + "/admin/remote-profile-sessions/" + url.PathEscape(strings.TrimSpace(sessionID))
	cookies := []*http.Cookie{remoteProfileSessionCookie(sessionValue)}
	resp, body, err := s.doJSONRequest(ctx, http.MethodDelete, urlValue, nil, cookies)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return &RemoteProfileError{
			Status:    http.StatusUnauthorized,
			ErrorType: ApiErrorTypeUnauthorized,
			Message:   "Remote session expired. Please log in again.",
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &RemoteProfileError{
			Status:    mapRemoteStatus(resp.StatusCode),
			ErrorType: inferErrorType(resp.StatusCode),
			Message:   extractRemoteErrorMessage(body),
		}
	}
	return nil
}

func (s *RemoteProfileService) doJSONRequest(ctx context.Context, method, urlValue string, body []byte, cookies []*http.Cookie) (*http.Response, []byte, error) {
	return s.doJSONRequestWithHeaders(ctx, method, urlValue, body, cookies, nil)
}

func (s *RemoteProfileService) doJSONRequestWithHeaders(ctx context.Context, method, urlValue string, body []byte, cookies []*http.Cookie, headers map[string]string) (*http.Response, []byte, error) {
	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, urlValue, reader)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Accept", "application/json")
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		req.Header.Set(key, value)
	}
	for _, cookie := range cookies {
		if cookie != nil {
			req.AddCookie(cookie)
		}
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, nil, classifyRemoteError(err)
	}
	defer resp.Body.Close()
	bodyBytes, err := readLimitedBody(resp.Body, remoteProfileProxyResponseMax)
	if err != nil {
		return resp, nil, err
	}
	return resp, bodyBytes, nil
}

func classifyRemoteError(err error) *RemoteProfileError {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &RemoteProfileError{
			Status:    http.StatusGatewayTimeout,
			ErrorType: ApiErrorTypeTimeout,
			Message:   "Remote request timed out",
		}
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return &RemoteProfileError{
				Status:    http.StatusGatewayTimeout,
				ErrorType: ApiErrorTypeTimeout,
				Message:   "Remote request timed out",
			}
		}
	}
	return &RemoteProfileError{
		Status:    http.StatusBadGateway,
		ErrorType: ApiErrorTypeNetwork,
		Message:   "Remote connection failed",
	}
}

func mapRemoteStatus(status int) int {
	if status >= 500 {
		return http.StatusBadGateway
	}
	return status
}

func extractRemoteErrorMessage(body []byte) string {
	if len(body) == 0 {
		return "Remote request failed"
	}
	var apiErr ApiErrorResponse
	if err := json.Unmarshal(body, &apiErr); err == nil {
		if strings.TrimSpace(apiErr.Error) != "" {
			return apiErr.Error
		}
	}
	msg := strings.TrimSpace(string(body))
	if msg == "" {
		return "Remote request failed"
	}
	return msg
}

func deriveCookieExpiry(cookie *http.Cookie, now time.Time) *time.Time {
	if cookie == nil {
		return nil
	}
	if !cookie.Expires.IsZero() {
		expiry := cookie.Expires.UTC()
		return &expiry
	}
	if cookie.MaxAge > 0 {
		expiry := now.Add(time.Duration(cookie.MaxAge) * time.Second).UTC()
		return &expiry
	}
	return nil
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie != nil && cookie.Name == name {
			return cookie
		}
	}
	return nil
}

type remoteProfileRecord struct {
	ID                        int64
	Tag                       string
	Label                     sql.NullString
	APIBase                   string
	ConnectorID               sql.NullString
	RemoteSessionID           sql.NullString
	Status                    string
	EncryptedSession          sql.NullString
	SessionExpiresAt          sql.NullTime
	RemoteSessionLastSyncedAt sql.NullTime
	LastLoginAt               sql.NullTime
	LastUsedAt                sql.NullTime
	CreatedBy                 sql.NullInt64
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

func (r *remoteProfileRecord) toProfile(now time.Time) RemoteProfile {
	status := r.Status
	if status == "" {
		status = remoteProfileStatusUnknown
	}
	hasSession := r.EncryptedSession.Valid && r.EncryptedSession.String != ""
	if r.SessionExpiresAt.Valid && now.After(r.SessionExpiresAt.Time) {
		status = remoteProfileStatusExpired
	}
	return RemoteProfile{
		ID:               r.ID,
		Tag:              r.Tag,
		Label:            NullStringValue(r.Label),
		APIBase:          r.APIBase,
		ConnectorID:      strings.TrimSpace(r.ConnectorID.String),
		RemoteSessionID:  NullStringValue(r.RemoteSessionID),
		Status:           status,
		HasSession:       hasSession,
		SessionExpiresAt: NullTimeValue(r.SessionExpiresAt),
		LastLoginAt:      NullTimeValue(r.LastLoginAt),
		LastUsedAt:       NullTimeValue(r.LastUsedAt),
		CreatedBy:        NullInt64Value(r.CreatedBy),
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}
}

func remoteProfileConnectorID(rec *remoteProfileRecord) string {
	if !rec.ConnectorID.Valid {
		return ""
	}
	return strings.TrimSpace(rec.ConnectorID.String)
}

func scanRemoteProfile(rows *sql.Rows) (*remoteProfileRecord, error) {
	var rec remoteProfileRecord
	if err := rows.Scan(
		&rec.ID,
		&rec.Tag,
		&rec.Label,
		&rec.APIBase,
		&rec.ConnectorID,
		&rec.RemoteSessionID,
		&rec.Status,
		&rec.EncryptedSession,
		&rec.SessionExpiresAt,
		&rec.RemoteSessionLastSyncedAt,
		&rec.LastLoginAt,
		&rec.LastUsedAt,
		&rec.CreatedBy,
		&rec.CreatedAt,
		&rec.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &rec, nil
}

func scanRemoteProfileRow(row *sql.Row) (*remoteProfileRecord, error) {
	var rec remoteProfileRecord
	result := row.Scan(
		&rec.ID,
		&rec.Tag,
		&rec.Label,
		&rec.APIBase,
		&rec.ConnectorID,
		&rec.RemoteSessionID,
		&rec.Status,
		&rec.EncryptedSession,
		&rec.SessionExpiresAt,
		&rec.RemoteSessionLastSyncedAt,
		&rec.LastLoginAt,
		&rec.LastUsedAt,
		&rec.CreatedBy,
		&rec.CreatedAt,
		&rec.UpdatedAt,
	)
	if result == sql.ErrNoRows {
		return nil, ErrRemoteProfileNotFound
	}
	if result != nil {
		return nil, result
	}
	return &rec, nil
}

func readLimitedBody(reader io.Reader, limit int64) ([]byte, error) {
	if limit <= 0 {
		return io.ReadAll(reader)
	}
	limited := &io.LimitedReader{R: reader, N: limit + 1}
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("response exceeds %d bytes", limit)
	}
	return data, nil
}
