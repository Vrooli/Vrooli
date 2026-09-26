package accountsecurity

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	admin "landing-page-business-suite-api/internal/administration"

	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Store is the narrow persistence seam for customer-owned session controls.
type Store interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type Service struct {
	store        Store
	auth         *admin.UserAuthService
	reauthMaxAge time.Duration
	now          func() time.Time
}

type Options struct {
	Store        Store
	Auth         *admin.UserAuthService
	ReauthMaxAge time.Duration
	Now          func() time.Time
}

func NewService(opts Options) *Service {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	maxAge := opts.ReauthMaxAge
	if maxAge <= 0 {
		maxAge = 15 * time.Minute
	}
	return &Service{store: opts.Store, auth: opts.Auth, reauthMaxAge: maxAge, now: now}
}

func (s *Service) ListSessions(ctx context.Context, userID, currentSessionID string) ([]*lpbsv1.AccountSession, error) {
	rows, err := s.store.QueryContext(ctx, `
		SELECT id, created_at, last_used_at, expires_at, absolute_expires_at, auth_method, ip_address::text, user_agent
		FROM user_sessions
		WHERE user_id = $1 AND revoked = FALSE AND expires_at > NOW() AND absolute_expires_at > NOW()
		ORDER BY last_used_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list customer sessions: %w", err)
	}
	defer rows.Close()
	out := make([]*lpbsv1.AccountSession, 0)
	for rows.Next() {
		var id, authMethod string
		var createdAt, lastUsedAt, expiresAt, absoluteExpiresAt time.Time
		var ipAddress, userAgent sql.NullString
		if err := rows.Scan(&id, &createdAt, &lastUsedAt, &expiresAt, &absoluteExpiresAt, &authMethod, &ipAddress, &userAgent); err != nil {
			return nil, fmt.Errorf("scan customer session: %w", err)
		}
		out = append(out, &lpbsv1.AccountSession{
			Id: id, CreatedAt: timestamppb.New(createdAt), LastUsedAt: timestamppb.New(lastUsedAt),
			ExpiresAt: timestamppb.New(expiresAt), AbsoluteExpiresAt: timestamppb.New(absoluteExpiresAt),
			DeviceLabel: deviceLabel(userAgent.String), IpHint: maskIP(ipAddress.String),
			AuthMethod: authMethod, Current: id == currentSessionID,
		})
	}
	return out, rows.Err()
}

func (s *Service) RevokeSession(ctx context.Context, userID, sessionID string) (bool, error) {
	result, err := s.store.ExecContext(ctx, `
		UPDATE user_sessions SET revoked = TRUE, revoked_at = NOW(), revoked_reason = 'user'
		WHERE id = $1 AND user_id = $2 AND revoked = FALSE`, sessionID, userID)
	if err != nil {
		return false, fmt.Errorf("revoke customer session: %w", err)
	}
	count, err := result.RowsAffected()
	return count == 1, err
}

func (s *Service) RevokeOtherSessions(ctx context.Context, userID, currentSessionID string) (int64, error) {
	result, err := s.store.ExecContext(ctx, `
		UPDATE user_sessions SET revoked = TRUE, revoked_at = NOW(), revoked_reason = 'user_others'
		WHERE user_id = $1 AND id <> $2 AND revoked = FALSE`, userID, currentSessionID)
	if err != nil {
		return 0, fmt.Errorf("revoke other customer sessions: %w", err)
	}
	return result.RowsAffected()
}

func (s *Service) StartReauthentication(ctx context.Context, userID, sessionID, browserBinding, ipAddress, userAgent string) (time.Time, error) {
	var email string
	if err := s.store.QueryRowContext(ctx, `SELECT u.email FROM users u JOIN user_sessions s ON s.user_id = u.id WHERE s.id = $1 AND s.user_id = $2 AND s.revoked = FALSE`, sessionID, userID).Scan(&email); err != nil {
		return time.Time{}, err
	}
	started, err := s.auth.RequestSignIn(ctx, admin.SignInRequest{Email: email, IPAddress: ipAddress, UserAgent: userAgent, BrowserBinding: browserBinding, Context: []byte(fmt.Sprintf(`{"purpose":"reauth","session_id":%q}`, sessionID))})
	if err != nil {
		return time.Time{}, err
	}
	return started.ExpiresAt, nil
}

func (s *Service) Reauthenticate(ctx context.Context, userID, sessionID, email, code, browserBinding, ipAddress, userAgent string) (time.Time, error) {
	var sessionEmail string
	if err := s.store.QueryRowContext(ctx, `SELECT u.email FROM users u JOIN user_sessions s ON s.user_id = u.id WHERE s.id = $1 AND s.user_id = $2 AND s.revoked = FALSE`, sessionID, userID).Scan(&sessionEmail); err != nil {
		return time.Time{}, err
	}
	if strings.TrimSpace(email) == "" {
		email = sessionEmail
	}
	user, err := s.auth.VerifySignInCodeWithoutSession(ctx, admin.SignInVerification{Email: email, Code: code, BrowserBinding: browserBinding, IPAddress: ipAddress, UserAgent: userAgent})
	if err != nil {
		return time.Time{}, err
	}
	if !strings.EqualFold(user.Email, sessionEmail) {
		return time.Time{}, errors.New("reauthentication address does not match session")
	}
	now := s.now().UTC()
	if _, err := s.store.ExecContext(ctx, `UPDATE user_sessions SET authenticated_at = $1, last_used_at = $1 WHERE id = $2 AND user_id = $3 AND revoked = FALSE`, now, sessionID, userID); err != nil {
		return time.Time{}, fmt.Errorf("update reauthentication time: %w", err)
	}
	return now, nil
}

func (s *Service) RecordReauthentication(ctx context.Context, userID, sessionID string, at time.Time) error {
	result, err := s.store.ExecContext(ctx, `UPDATE user_sessions SET authenticated_at = $1, last_used_at = $1 WHERE id = $2 AND user_id = $3 AND revoked = FALSE`, at.UTC(), sessionID, userID)
	if err != nil {
		return fmt.Errorf("update passkey reauthentication time: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil || count != 1 {
		return errors.New("session is not valid")
	}
	return nil
}

// IssueAccessToken returns a replacement access token after reauthentication
// without creating another session or rotating the refresh credential.
func (s *Service) IssueAccessToken(ctx context.Context, userID, sessionID string) (string, time.Time, error) {
	var email string
	if err := s.store.QueryRowContext(ctx, `SELECT email FROM users WHERE id = $1`, userID).Scan(&email); err != nil {
		return "", time.Time{}, err
	}
	return s.auth.GenerateAccessToken(userID, email, sessionID)
}

func (s *Service) IsRecent(ctx context.Context, sessionID string) (bool, error) {
	var authenticatedAt time.Time
	if err := s.store.QueryRowContext(ctx, `SELECT authenticated_at FROM user_sessions WHERE id = $1 AND revoked = FALSE`, sessionID).Scan(&authenticatedAt); err != nil {
		return false, err
	}
	return s.now().UTC().Sub(authenticatedAt) <= s.reauthMaxAge, nil
}

func (s *Service) SignInDeliveryStatus(ctx context.Context, email, browserBinding string) (string, string, time.Time, error) {
	var status, providerStatus, reasonClass string
	var expiresAt time.Time
	hash := bindingHash(browserBinding)
	err := s.store.QueryRowContext(ctx, `SELECT COALESCE(delivery_status, 'pending'), COALESCE(provider_status, ''), COALESCE(provider_reason_class, ''), expires_at FROM auth_tokens WHERE token_type = 'magic_link' AND lower(email) = lower($1) AND browser_binding_hash = $2 ORDER BY created_at DESC LIMIT 1`, email, hash).Scan(&status, &providerStatus, &reasonClass, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return "pending", "", time.Time{}, nil
	}
	if err != nil {
		return "pending", "", time.Time{}, err
	}
	switch strings.ToLower(providerStatus) {
	case "delivered":
		status = "delivered"
	case "deferred":
		status = "deferred"
	case "bounce", "dropped", "spamreport":
		status = "rejected"
	case "processed":
		status = "pending"
	default:
		if status == "failed" {
			status = "rejected"
		} else if status != "delivered" {
			status = "pending"
		}
	}
	return status, reasonClass, expiresAt, nil
}

func bindingHash(value string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(digest[:])
}

var chromeRE = regexp.MustCompile(`Chrome|CriOS`)
var firefoxRE = regexp.MustCompile(`Firefox|FxiOS`)
var edgeRE = regexp.MustCompile(`Edg|EdgiOS|EdgA`)

func deviceLabel(userAgent string) string {
	ua := strings.TrimSpace(userAgent)
	browser := "Browser"
	switch {
	case edgeRE.MatchString(ua):
		browser = "Edge"
	case chromeRE.MatchString(ua):
		browser = "Chrome"
	case firefoxRE.MatchString(ua):
		browser = "Firefox"
	case strings.Contains(ua, "Safari"):
		browser = "Safari"
	}
	os := "Unknown device"
	switch {
	case strings.Contains(ua, "iPhone") || strings.Contains(ua, "iPad"):
		os = "iOS"
	case strings.Contains(ua, "Android"):
		os = "Android"
	case strings.Contains(ua, "Mac OS X"):
		os = "macOS"
	case strings.Contains(ua, "Windows"):
		os = "Windows"
	case strings.Contains(ua, "Linux"):
		os = "Linux"
	}
	return browser + " on " + os
}

func maskIP(value string) string {
	ip := net.ParseIP(strings.TrimSpace(value))
	if ip == nil {
		return "unknown"
	}
	if ip4 := ip.To4(); ip4 != nil {
		return fmt.Sprintf("%d.%d.%d.x", ip4[0], ip4[1], ip4[2])
	}
	parts := strings.Split(ip.String(), ":")
	if len(parts) >= 3 {
		return strings.Join(parts[:3], ":") + ":*"
	}
	return "*:"
}
