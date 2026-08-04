package administration

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"landing-page-business-suite-api/internal/securevalue"
)

const (
	remoteProfileCookieName    = "admin_session"
	remoteProfileStatusUnknown = "unknown"
	remoteProfileStatusActive  = "active"
	remoteProfileStatusExpired = "expired"
	remoteProfileStatusError   = "error"
)

const (
	RemoteProfileCookieName    = remoteProfileCookieName
	RemoteProfileStatusUnknown = remoteProfileStatusUnknown
	RemoteProfileStatusActive  = remoteProfileStatusActive
	RemoteProfileStatusExpired = remoteProfileStatusExpired
	RemoteProfileStatusError   = remoteProfileStatusError
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
	// RemoteProfileProxyBodyLimit bounds an admin proxy request body.
	RemoteProfileProxyBodyLimit   = 1 << 20 // 1MB
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

func (e *RemoteProfileError) Error() string {
	return e.Message
}

// RemoteProfileService manages remote profile storage and remote admin sessions.
type RemoteProfileService struct {
	DB            RemoteProfileStore
	EncryptionKey []byte
	HTTPClient    HTTPDoer
	Now           func() time.Time
	ResolveSecret func(string) string
	IsProduction  func() bool
	LogEvent      func(string, map[string]interface{})
	LogError      func(string, map[string]interface{})
}

// NewRemoteProfileService creates a RemoteProfileService with defaults.
func NewRemoteProfileService(db RemoteProfileStore) (*RemoteProfileService, error) {
	client := &http.Client{
		Timeout: 20 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return NewRemoteProfileServiceWithOptions(db, client)
}

// NewRemoteProfileServiceWithOptions creates a RemoteProfileService with a custom HTTP client.
func NewRemoteProfileServiceWithOptions(db RemoteProfileStore, client HTTPDoer) (*RemoteProfileService, error) {
	return NewRemoteProfileServiceWithRuntime(db, client, nil, nil, nil, nil)
}

// NewRemoteProfileServiceWithRuntime wires application-owned secret, environment,
// and logging behavior at the composition boundary.
func NewRemoteProfileServiceWithRuntime(db RemoteProfileStore, client HTTPDoer, resolveSecret func(string) string, isProduction func() bool, logEvent func(string, map[string]interface{}), logError func(string, map[string]interface{})) (*RemoteProfileService, error) {
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	if resolveSecret == nil {
		resolveSecret = resolveRemoteProfileSecret
	}
	if isProduction == nil {
		isProduction = isRemoteProfileProductionEnvironment
	}
	if logEvent == nil {
		logEvent = logRemoteProfileEvent
	}
	if logError == nil {
		logError = logRemoteProfileError
	}
	if db != nil {
		if _, err := db.ExecContext(context.Background(), `ALTER TABLE remote_profiles ADD COLUMN IF NOT EXISTS encryption_state VARCHAR(16) NOT NULL DEFAULT 'unknown'`); err != nil {
			return nil, fmt.Errorf("ensure remote profile encryption state: %w", err)
		}
	}
	key, err := loadRemoteProfileEncryptionKey(resolveSecret, isProduction, logEvent)
	if err != nil {
		return nil, err
	}
	now := time.Now
	return &RemoteProfileService{
		DB:            db,
		EncryptionKey: key,
		HTTPClient:    client,
		Now:           now,
		ResolveSecret: resolveSecret,
		IsProduction:  isProduction,
		LogEvent:      logEvent,
		LogError:      logError,
	}, nil
}

func loadRemoteProfileEncryptionKey(resolveSecret func(string) string, isProduction func() bool, logEvent func(string, map[string]interface{})) ([]byte, error) {
	keyStr := resolveSecret("LPBS_REMOTE_PROFILE_ENCRYPTION_KEY")
	keySource := "LPBS_REMOTE_PROFILE_ENCRYPTION_KEY"
	if keyStr == "" {
		fallback := resolveSecret("LPBS_API_KEY_ENCRYPTION_KEY")
		if fallback != "" {
			keyStr = fallback
			keySource = "LPBS_API_KEY_ENCRYPTION_KEY"
			logEvent("remote_profiles_encryption_key_fallback", map[string]interface{}{
				"level":    "warn",
				"message":  "LPBS_REMOTE_PROFILE_ENCRYPTION_KEY not set; falling back to LPBS_API_KEY_ENCRYPTION_KEY",
				"security": true,
				"action":   "Set LPBS_REMOTE_PROFILE_ENCRYPTION_KEY before production use",
			})
		}
	}
	if keyStr == "" {
		if isProduction() {
			return nil, fmt.Errorf(
				"LPBS_REMOTE_PROFILE_ENCRYPTION_KEY is required in production. " +
					"Set LPBS_REMOTE_PROFILE_ENCRYPTION_KEY (preferred) or LPBS_API_KEY_ENCRYPTION_KEY as a fallback",
			)
		}
		logEvent("remote_profiles_no_encryption_key_dev", map[string]interface{}{
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
	return securevalue.Encrypt(s.EncryptionKey, plaintext)
}

func (s *RemoteProfileService) decrypt(ciphertext string) (string, error) {
	return securevalue.Decrypt(s.EncryptionKey, ciphertext)
}

func (s *RemoteProfileService) Encrypt(plaintext string) (string, error) { return s.encrypt(plaintext) }
func (s *RemoteProfileService) Decrypt(ciphertext string) (string, error) {
	return s.decrypt(ciphertext)
}
func NormalizeRemoteProfileTag(tag string) (string, error) { return normalizeRemoteProfileTag(tag) }
func NormalizeRemoteProfileLabel(label string) *string     { return normalizeRemoteProfileLabel(label) }
func NormalizeRemoteProfileAPIBase(raw string) (string, error) {
	return normalizeRemoteProfileAPIBase(raw)
}
func NormalizeRemoteProxyPath(raw string) (string, error) { return normalizeRemoteProxyPath(raw) }
func ReadLimitedBody(reader io.Reader, limit int64) ([]byte, error) {
	return readLimitedBody(reader, limit)
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
	return normalizeRemoteProfileAPIBaseForEnvironment(raw, isRemoteProfileProductionEnvironment)
}

func normalizeRemoteProfileAPIBaseForEnvironment(raw string, isProduction func() bool) (string, error) {
	clean, err := validateRemoteProfileURL(raw)
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(clean)
	if err != nil || parsed.Host == "" {
		return "", errRemoteProfileURLInvalid
	}
	if parsed.User != nil {
		return "", fmt.Errorf("api_base must not include credentials")
	}
	if isProduction != nil && isProduction() && parsed.Scheme != "https" {
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

func (s *RemoteProfileService) normalizeAPIBase(raw string) (string, error) {
	isProduction := s.IsProduction
	if isProduction == nil {
		isProduction = isRemoteProfileProductionEnvironment
	}
	return normalizeRemoteProfileAPIBaseForEnvironment(raw, isProduction)
}

func (s *RemoteProfileService) logEvent(name string, fields map[string]interface{}) {
	if s.LogEvent != nil {
		s.LogEvent(name, fields)
	}
}

func (s *RemoteProfileService) logError(name string, fields map[string]interface{}) {
	if s.LogError != nil {
		s.LogError(name, fields)
	}
}

func generateRemoteConnectorID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func (s *RemoteProfileService) remoteProfileOriginLabel() string {
	resolveSecret := s.ResolveSecret
	if resolveSecret == nil {
		resolveSecret = resolveRemoteProfileSecret
	}
	if value := SanitizeRemoteProfileSessionMetaValue(resolveSecret("LPBS_REMOTE_PROFILE_ORIGIN")); value != "" {
		return value
	}
	if host, err := os.Hostname(); err == nil {
		if value := SanitizeRemoteProfileSessionMetaValue(host); value != "" {
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

func IsAllowedRemoteProxyPath(path string) bool { return isAllowedRemoteProxyPath(path) }

func (s *RemoteProfileService) List(ctx context.Context) ([]RemoteProfile, error) {
	rows, err := s.DB.QueryContext(ctx, `
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
	apiBase, err := s.normalizeAPIBase(req.APIBase)
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
	err = s.DB.QueryRowContext(ctx, `
		INSERT INTO remote_profiles (tag, label, api_base, connector_id, status, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING id
	`, tag, stringToNullString(label), apiBase, connectorID, remoteProfileStatusUnknown, int64ToNullInt64(createdByID)).Scan(&id)
	if err != nil {
		return nil, err
	}

	profile, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	s.logEvent("remote_profile_created", map[string]interface{}{
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
		updatedLabel = stringToNullString(normalizeRemoteProfileLabel(*req.Label))
	}

	updatedAPIBase := rec.APIBase
	if req.APIBase != nil {
		updatedAPIBase, err = s.normalizeAPIBase(*req.APIBase)
		if err != nil {
			return nil, err
		}
	}

	apiBaseChanged := updatedAPIBase != rec.APIBase
	if apiBaseChanged {
		_, err = s.DB.ExecContext(ctx, `
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
		_, err = s.DB.ExecContext(ctx, `
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

	s.logEvent("remote_profile_updated", map[string]interface{}{
		"level": "info",
		"id":    profile.ID,
		"tag":   profile.Tag,
	})

	return profile, nil
}

func (s *RemoteProfileService) Delete(ctx context.Context, id int64) error {
	res, err := s.DB.ExecContext(ctx, `DELETE FROM remote_profiles WHERE id = $1`, id)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrRemoteProfileNotFound
	}

	s.logEvent("remote_profile_deleted", map[string]interface{}{
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
		Origin:      s.remoteProfileOriginLabel(),
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

	s.logEvent("remote_profile_login", map[string]interface{}{
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
				s.logError("remote_profile_remote_logout_failed", map[string]interface{}{
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

	s.logEvent("remote_profile_logout", map[string]interface{}{
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
			ErrorType: apiErrorTypeUnauthorized,
			Message:   "Remote session expired. Please log in again.",
		}
	}

	profile, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return profile, nil
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
		Label:            nullStringValue(r.Label),
		APIBase:          r.APIBase,
		ConnectorID:      strings.TrimSpace(r.ConnectorID.String),
		RemoteSessionID:  nullStringValue(r.RemoteSessionID),
		Status:           status,
		HasSession:       hasSession,
		SessionExpiresAt: nullTimeValue(r.SessionExpiresAt),
		LastLoginAt:      nullTimeValue(r.LastLoginAt),
		LastUsedAt:       nullTimeValue(r.LastUsedAt),
		CreatedBy:        nullInt64Value(r.CreatedBy),
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
