package administration

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"math/big"
	"net/url"
	"strings"
	"time"

	"landing-page-business-suite-api/internal/emaildelivery"
)

// Sign-in failure modes beyond the shared token errors.
var (
	ErrCodeInvalid         = errors.New("sign-in code is invalid or expired")
	ErrDeliveryUnavailable = errors.New("sign-in email could not be delivered")
	ErrContextInvalid      = errors.New("sign-in context is invalid")
)

// Sign-in flows derived from the stored request context.
const (
	SignInFlowBrowser     = "browser"
	SignInFlowNativeApp   = "native_app"
	SignInFlowDesktopLink = "desktop_link"
)

const (
	signInTokenType      = "magic_link"
	maxSignInContextSize = 4096
	signInCodeDigits     = 6
)

// SignInMessage is everything a sender needs to render the sign-in email.
type SignInMessage struct {
	To        string
	Link      string
	Code      string
	AppName   string
	ExpiresIn time.Duration
	RequestID string
	Subject   string
	TextBody  string
	HTMLBody  string
}

// SignInRequest starts passwordless sign-in. No user row is created until the
// address is proven by the link or the code.
type SignInRequest struct {
	Email          string
	IPAddress      string
	UserAgent      string
	BrowserBinding string
	// Context is the requesting page's native-app or desktop-link parameters.
	// It is stored server-side so a link opened in a new tab still completes
	// the flow the user started.
	Context json.RawMessage
}

// SignInStarted describes an accepted request without revealing secrets.
type SignInStarted struct {
	RequestID string
	ExpiresAt time.Time
}

// SignInVerification proves an address with either the link token or the
// code. A code is only accepted from the browser that requested it.
type SignInVerification struct {
	Token          string
	Email          string
	Code           string
	BrowserBinding string
	IPAddress      string
	UserAgent      string
}

// SignInResult is a completed sign-in. Context is returned only to the browser
// that started the request.
type SignInResult struct {
	Tokens          *TokenPair
	User            *User
	Context         json.RawMessage
	SameBrowser     bool
	AuthenticatedAt time.Time
}

// SignInPreview lets the verify page confirm intent before consuming a link,
// so mail scanners that open links cannot burn them.
type SignInPreview struct {
	EmailHint   string    `json:"email_hint"`
	ExpiresAt   time.Time `json:"expires_at"`
	Flow        string    `json:"flow"`
	SameBrowser bool      `json:"same_browser"`
	// Context is the requesting page's app parameters, returned only to the
	// browser that started the request so it can resume that app's flow.
	Context json.RawMessage `json:"context,omitempty"`
}

// RequestMagicLink is retained for callers that only need the link.
func (s *UserAuthService) RequestMagicLink(ctx context.Context, email, ipAddress, userAgent string) error {
	_, err := s.RequestSignIn(ctx, SignInRequest{Email: email, IPAddress: ipAddress, UserAgent: userAgent})
	return err
}

// RequestSignIn stores a pending sign-in and emails its link and code.
func (s *UserAuthService) RequestSignIn(ctx context.Context, request SignInRequest) (*SignInStarted, error) {
	email := NormalizeEmail(request.Email)
	if !LooksLikeEmail(email) {
		return nil, errors.New("valid email address is required")
	}
	if strings.TrimSpace(s.baseURL) == "" {
		return nil, errors.New("AUTH_MAGIC_LINK_BASE_URL must be configured before requesting a magic link")
	}
	contextValue, err := normalizeSignInContext(request.Context)
	if err != nil {
		return nil, err
	}

	token, err := randomHex(32)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}
	code, err := randomDigits(signInCodeDigits)
	if err != nil {
		return nil, fmt.Errorf("generate code: %w", err)
	}

	expiresAt := time.Now().UTC().Add(s.magicLinkTTL)
	link := fmt.Sprintf("%s?token=%s", s.baseURL, url.QueryEscape(token))
	subject, textBody, htmlBody := renderSignInMessage(code, link, s.appName, s.magicLinkTTL)
	var requestID string
	var dedupeKey string
	if s.outboxEnabled {
		db, ok := s.db.(*sql.DB)
		if !ok {
			return nil, errors.New("durable email outbox requires a SQL database")
		}
		tx, txErr := db.BeginTx(ctx, nil)
		if txErr != nil {
			return nil, fmt.Errorf("begin sign-in transaction: %w", txErr)
		}
		defer tx.Rollback()
		err = tx.QueryRowContext(ctx, `
		INSERT INTO auth_tokens (token_hash, token_type, expires_at, ip_address, user_agent, email, code_hash, request_context, browser_binding_hash, delivery_status)
		VALUES ($1, $2, $3, $4::inet, $5, $6, $7, $8, $9, 'pending')
		RETURNING id
	`, HashToken(token), signInTokenType, expiresAt, toNullableParam(request.IPAddress), toNullableParam(request.UserAgent),
			email, signInCodeHash(email, code), contextValue, bindingHash(request.BrowserBinding)).Scan(&requestID)
		if err != nil {
			return nil, fmt.Errorf("store auth token: %w", err)
		}
		dedupeKey = "signin:" + HashToken(token)
		templateData, marshalErr := json.Marshal(map[string]string{"auth_token_id": requestID})
		if marshalErr != nil {
			return nil, fmt.Errorf("encode sign-in email metadata: %w", marshalErr)
		}
		if err := emaildelivery.EnqueueTx(ctx, tx, emaildelivery.Message{
			DedupeKey: dedupeKey, IdempotencyKey: dedupeKey, Purpose: emaildelivery.PurposeSignIn,
			Recipient: email, Sender: s.senderIdentity, Subject: subject, TextBody: textBody, HTMLBody: htmlBody,
			TemplateRef: "auth.sign-in", TemplateData: templateData, Priority: 100, ExpiresAt: &expiresAt,
		}); err != nil {
			return nil, fmt.Errorf("enqueue sign-in email: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit sign-in transaction: %w", err)
		}
	} else {
		err = s.db.QueryRowContext(ctx, `
			INSERT INTO auth_tokens (token_hash, token_type, expires_at, ip_address, user_agent, email, code_hash, request_context, browser_binding_hash, delivery_status)
			VALUES ($1, $2, $3, $4::inet, $5, $6, $7, $8, $9, 'pending')
			RETURNING id
		`, HashToken(token), signInTokenType, expiresAt, toNullableParam(request.IPAddress), toNullableParam(request.UserAgent),
			email, signInCodeHash(email, code), contextValue, bindingHash(request.BrowserBinding)).Scan(&requestID)
	}
	if err != nil {
		return nil, fmt.Errorf("store auth token: %w", err)
	}

	if s.onMagicLinkGenerated != nil {
		s.onMagicLinkGenerated(email, token, link)
	}
	if s.onSignInCodeGenerated != nil {
		s.onSignInCodeGenerated(email, code)
	}
	if dedupeKey != "" {
		if _, err := s.db.ExecContext(ctx, `UPDATE auth_tokens SET delivery_status = 'queued' WHERE id = $1`, requestID); err != nil {
			return nil, fmt.Errorf("mark sign-in queued: %w", err)
		}
		s.log("magic_link_queued", map[string]interface{}{"level": "info", "email": email})
		return &SignInStarted{RequestID: requestID, ExpiresAt: expiresAt}, nil
	}

	delivery, sendErr := s.deliverSignIn(SignInMessage{To: email, Link: link, Code: code, AppName: s.appName, ExpiresIn: s.magicLinkTTL, RequestID: requestID, Subject: subject, TextBody: textBody, HTMLBody: htmlBody})
	if sendErr != nil {
		// Delivery failure is infrastructure failure, not customer use. Keep the
		// token usable so a retry can succeed without consuming the attempt.
		if _, markErr := s.db.ExecContext(ctx, `
			UPDATE auth_tokens SET delivery_status = 'failed', delivery_error = $2 WHERE id = $1
		`, requestID, truncate(sendErr.Error(), 500)); markErr != nil {
			s.logError("sign_in_delivery_mark_failed", map[string]interface{}{"error": markErr.Error()})
		}
		s.logError("send_magic_link_failed", map[string]interface{}{"error": sendErr.Error(), "email": email})
		return nil, fmt.Errorf("%w: %v", ErrDeliveryUnavailable, sendErr)
	}
	if _, err := s.db.ExecContext(ctx, `
		UPDATE auth_tokens
		SET delivery_status = 'sent', provider = $2, provider_message_id = $3
		WHERE id = $1
		`, requestID, nullableDeliveryProvider(delivery), nullableDeliveryMessageID(delivery)); err != nil {
		s.logError("sign_in_delivery_mark_failed", map[string]interface{}{"error": err.Error()})
	}
	if dedupeKey != "" {
		if _, err := s.db.ExecContext(ctx, `UPDATE email_outbox SET status = 'accepted', provider_message_id = $2, accepted_at = NOW(), settled_at = NOW(), updated_at = NOW() WHERE dedupe_key = $1`, dedupeKey, nullableDeliveryMessageID(delivery)); err != nil {
			s.logError("sign_in_outbox_mark_accepted_failed", map[string]interface{}{"error": err.Error()})
		}
	}

	s.log("magic_link_requested", map[string]interface{}{"level": "info", "email": email})
	return &SignInStarted{RequestID: requestID, ExpiresAt: expiresAt}, nil
}

func renderSignInMessage(code, link, appName string, ttl time.Duration) (string, string, string) {
	appName = strings.TrimSpace(appName)
	if appName == "" {
		appName = "App"
	}
	subject := fmt.Sprintf("%s is your %s sign-in code", code, appName)
	textBody := fmt.Sprintf("Your %s sign-in code is %s.\n\nSign in: %s\n\nThis code expires in %s.", appName, code, link, ttl)
	htmlBody := fmt.Sprintf("<p>Your %s sign-in code is <strong>%s</strong>.</p><p><a href=\"%s\">Sign in</a></p><p>This code expires in %s.</p>", html.EscapeString(appName), html.EscapeString(code), html.EscapeString(link), html.EscapeString(ttl.String()))
	return subject, textBody, htmlBody
}

func nullableDeliveryProvider(delivery *SignInDelivery) any {
	if delivery == nil || strings.TrimSpace(delivery.Provider) == "" {
		return nil
	}
	return delivery.Provider
}

func nullableDeliveryMessageID(delivery *SignInDelivery) any {
	if delivery == nil || strings.TrimSpace(delivery.ProviderMessageID) == "" {
		return nil
	}
	return delivery.ProviderMessageID
}

func (s *UserAuthService) deliverSignIn(message SignInMessage) (*SignInDelivery, error) {
	switch sender := s.emailService.(type) {
	case nil:
		return nil, errors.New("magic-link email provider is unavailable")
	case SignInMessageSender:
		return sender.SendSignIn(message)
	default:
		if err := sender.SendMagicLink(message.To, message.Link, message.AppName); err != nil {
			return nil, err
		}
		return &SignInDelivery{Provider: "unknown"}, nil
	}
}

// PreviewSignIn describes a link without consuming it.
func (s *UserAuthService) PreviewSignIn(ctx context.Context, token, browserBinding string) (*SignInPreview, error) {
	row, err := s.findPendingByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	preview := &SignInPreview{
		EmailHint:   MaskEmail(row.email),
		ExpiresAt:   row.expiresAt,
		Flow:        signInFlow(row.context),
		SameBrowser: row.bindingMatches(browserBinding),
	}
	if preview.SameBrowser && len(row.context) > 0 {
		preview.Context = row.context
	}
	return preview, nil
}

// VerifyMagicLink is retained for callers that verify a link without a
// browser binding.
func (s *UserAuthService) VerifyMagicLink(ctx context.Context, token, ipAddress, userAgent string) (*TokenPair, *User, error) {
	result, err := s.VerifySignIn(ctx, SignInVerification{Token: token, IPAddress: ipAddress, UserAgent: userAgent})
	if err != nil {
		return nil, nil, err
	}
	return result.Tokens, result.User, nil
}

// VerifySignIn completes a pending sign-in by link token or by code.
func (s *UserAuthService) VerifySignIn(ctx context.Context, verification SignInVerification) (*SignInResult, error) {
	var (
		row *pendingSignIn
		err error
	)
	if strings.TrimSpace(verification.Code) == "" {
		row, err = s.findPendingByToken(ctx, verification.Token)
	} else {
		row, err = s.findPendingByCode(ctx, verification.Email, verification.Code, verification.BrowserBinding)
	}
	if err != nil {
		return nil, err
	}
	if row.usedAt.Valid {
		return nil, ErrTokenUsed
	}
	if time.Now().UTC().After(row.expiresAt) {
		return nil, ErrTokenExpired
	}

	result, err := s.db.ExecContext(ctx, `UPDATE auth_tokens SET used_at = NOW() WHERE id = $1 AND used_at IS NULL`, row.id)
	if err != nil {
		return nil, fmt.Errorf("mark token used: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return nil, ErrTokenUsed
	}

	user, err := s.userForSignIn(ctx, row)
	if err != nil {
		return nil, err
	}
	// Every other outstanding link or code for this address is retired: one
	// completed sign-in ends the request, not just the credential that won.
	if _, err := s.db.ExecContext(ctx, `
		UPDATE auth_tokens SET used_at = NOW()
		WHERE token_type = $1 AND used_at IS NULL AND id <> $2 AND (email = $3 OR user_id = $4)
	`, signInTokenType, row.id, user.Email, user.ID); err != nil {
		s.logError("retire_sibling_sign_ins_failed", map[string]interface{}{"error": err.Error()})
	}

	now := time.Now().UTC()
	if _, err := s.db.ExecContext(ctx, `
		UPDATE users SET email_verified = TRUE, last_login_at = NOW(), updated_at = NOW() WHERE id = $1
	`, user.ID); err != nil {
		s.logError("update_user_after_magic_link_failed", map[string]interface{}{"error": err.Error(), "user_id": user.ID})
	} else {
		user.EmailVerified = true
		user.LastLoginAt = &now
	}

	tokens, err := s.CreateSession(ctx, user, verification.IPAddress, verification.UserAgent)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	method := "link"
	if verification.Code != "" {
		method = "code"
	}
	s.log("magic_link_verified", map[string]interface{}{"level": "info", "user_id": user.ID, "email": user.Email, "method": method})

	out := &SignInResult{Tokens: tokens, User: user, SameBrowser: row.bindingMatches(verification.BrowserBinding), AuthenticatedAt: now}
	if out.SameBrowser && len(row.context) > 0 {
		out.Context = row.context
	}
	return out, nil
}

// VerifySignInCodeWithoutSession consumes a reauthentication code and proves
// the address without creating a second browser session. The caller owns the
// session update after comparing the returned user with its current session.
func (s *UserAuthService) VerifySignInCodeWithoutSession(ctx context.Context, verification SignInVerification) (*User, error) {
	row, err := s.findPendingByCode(ctx, verification.Email, verification.Code, verification.BrowserBinding)
	if err != nil {
		return nil, err
	}
	if row.usedAt.Valid {
		return nil, ErrTokenUsed
	}
	if time.Now().UTC().After(row.expiresAt) {
		return nil, ErrTokenExpired
	}
	result, err := s.db.ExecContext(ctx, `UPDATE auth_tokens SET used_at = NOW() WHERE id = $1 AND used_at IS NULL`, row.id)
	if err != nil {
		return nil, fmt.Errorf("mark reauthentication code used: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return nil, ErrTokenUsed
	}
	return s.userForSignIn(ctx, row)
}

// VerifySignInWithoutSession consumes a browser proof for a native grant but
// deliberately does not create user_sessions. The session is created only
// after the native client proves its PKCE verifier.
func (s *UserAuthService) VerifySignInWithoutSession(ctx context.Context, verification SignInVerification) (*SignInResult, error) {
	var row *pendingSignIn
	var err error
	if strings.TrimSpace(verification.Code) == "" {
		row, err = s.findPendingByToken(ctx, verification.Token)
	} else {
		row, err = s.findPendingByCode(ctx, verification.Email, verification.Code, verification.BrowserBinding)
	}
	if err != nil {
		return nil, err
	}
	if row.usedAt.Valid {
		return nil, ErrTokenUsed
	}
	if time.Now().UTC().After(row.expiresAt) {
		return nil, ErrTokenExpired
	}
	result, err := s.db.ExecContext(ctx, `UPDATE auth_tokens SET used_at = NOW() WHERE id = $1 AND used_at IS NULL`, row.id)
	if err != nil {
		return nil, fmt.Errorf("mark native sign-in used: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return nil, ErrTokenUsed
	}
	user, err := s.userForSignIn(ctx, row)
	if err != nil {
		return nil, err
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE auth_tokens SET used_at = NOW() WHERE token_type = $1 AND used_at IS NULL AND id <> $2 AND (email = $3 OR user_id = $4)`, signInTokenType, row.id, user.Email, user.ID); err != nil {
		s.logError("retire_native_sibling_sign_ins_failed", map[string]interface{}{"error": err.Error()})
	}
	now := time.Now().UTC()
	if _, err := s.db.ExecContext(ctx, `UPDATE users SET email_verified = TRUE, last_login_at = NOW(), updated_at = NOW() WHERE id = $1`, user.ID); err != nil {
		s.logError("update_user_after_native_sign_in_failed", map[string]interface{}{"error": err.Error(), "user_id": user.ID})
	} else {
		user.EmailVerified = true
		user.LastLoginAt = &now
	}
	out := &SignInResult{User: user, SameBrowser: row.bindingMatches(verification.BrowserBinding), AuthenticatedAt: now}
	if out.SameBrowser && len(row.context) > 0 {
		out.Context = row.context
	}
	return out, nil
}

// PurgeExpiredSignIns deletes sign-in requests that can no longer be used.
func (s *UserAuthService) PurgeExpiredSignIns(ctx context.Context) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM auth_tokens WHERE token_type = $1 AND expires_at < NOW() - INTERVAL '1 day'
	`, signInTokenType)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// PurgeEndedUserSessions removes sessions that ended at least 30 days ago.
// The retention period preserves recent revocation evidence while ensuring
// ended sessions do not grow without bound.
func (s *UserAuthService) PurgeEndedUserSessions(ctx context.Context) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM user_sessions
		WHERE (revoked = TRUE OR expires_at < NOW())
		  AND last_used_at < NOW() - INTERVAL '30 days'
	`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// PurgeRefreshHistory removes retired refresh tokens after the maximum
// customer session lifetime plus a safety margin.
func (s *UserAuthService) PurgeRefreshHistory(ctx context.Context) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM refresh_token_history
		WHERE retired_at < NOW() - INTERVAL '100 days'
	`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// SignInDeliveryHealth summarizes recent sign-in email outcomes.
type SignInDeliveryHealth struct {
	Sent        int        `json:"sent"`
	Failed      int        `json:"failed"`
	LastFailure *time.Time `json:"last_failure,omitempty"`
	LastError   string     `json:"last_error,omitempty"`
	LastSuccess *time.Time `json:"last_success,omitempty"`
}

// DeliveryHealth reports sign-in email outcomes over the window.
func (s *UserAuthService) DeliveryHealth(ctx context.Context, window time.Duration) (*SignInDeliveryHealth, error) {
	health := &SignInDeliveryHealth{}
	var lastFailure, lastSuccess sql.NullTime
	var lastError sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE delivery_status = 'sent'),
			COUNT(*) FILTER (WHERE delivery_status = 'failed'),
			MAX(created_at) FILTER (WHERE delivery_status = 'failed'),
			MAX(created_at) FILTER (WHERE delivery_status = 'sent'),
			(SELECT delivery_error FROM auth_tokens WHERE token_type = $1 AND delivery_status = 'failed' ORDER BY created_at DESC LIMIT 1)
		FROM auth_tokens
		WHERE token_type = $1 AND created_at > NOW() - ($2 * INTERVAL '1 second')
	`, signInTokenType, int64(window/time.Second)).Scan(&health.Sent, &health.Failed, &lastFailure, &lastSuccess, &lastError)
	if err != nil {
		return nil, err
	}
	if lastFailure.Valid {
		health.LastFailure = &lastFailure.Time
		health.LastError = lastError.String
	}
	if lastSuccess.Valid {
		health.LastSuccess = &lastSuccess.Time
	}
	return health, nil
}

type pendingSignIn struct {
	id          string
	userID      sql.NullString
	email       string
	expiresAt   time.Time
	usedAt      sql.NullTime
	context     json.RawMessage
	bindingHash sql.NullString
}

func (p *pendingSignIn) bindingMatches(binding string) bool {
	if p == nil || !p.bindingHash.Valid || strings.TrimSpace(binding) == "" {
		return false
	}
	want, _ := bindingHash(binding).(string)
	return subtle.ConstantTimeCompare([]byte(want), []byte(p.bindingHash.String)) == 1
}

const pendingSignInColumns = `t.id, t.user_id, COALESCE(t.email, u.email, ''), t.expires_at, t.used_at, t.request_context, t.browser_binding_hash`

func scanPendingSignIn(row *sql.Row) (*pendingSignIn, error) {
	var pending pendingSignIn
	var contextValue []byte
	if err := row.Scan(&pending.id, &pending.userID, &pending.email, &pending.expiresAt, &pending.usedAt, &contextValue, &pending.bindingHash); err != nil {
		return nil, err
	}
	if len(contextValue) > 0 {
		pending.context = json.RawMessage(contextValue)
	}
	return &pending, nil
}

func (s *UserAuthService) findPendingByToken(ctx context.Context, token string) (*pendingSignIn, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrTokenInvalid
	}
	row, err := scanPendingSignIn(s.db.QueryRowContext(ctx, `
		SELECT `+pendingSignInColumns+`
		FROM auth_tokens t LEFT JOIN users u ON u.id = t.user_id
		WHERE t.token_hash = $1 AND t.token_type = $2
	`, HashToken(token), signInTokenType))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTokenInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("query auth token: %w", err)
	}
	if !row.usedAt.Valid && time.Now().UTC().After(row.expiresAt) {
		return row, ErrTokenExpired
	}
	if row.usedAt.Valid {
		return row, ErrTokenUsed
	}
	return row, nil
}

// findPendingByCode matches a code against the newest outstanding requests for
// the address from the same browser, so a resend never invalidates a code the
// user is already typing. Callers bound guesses with AuthThrottle.
func (s *UserAuthService) findPendingByCode(ctx context.Context, email, code, browserBinding string) (*pendingSignIn, error) {
	email = NormalizeEmail(email)
	code = strings.TrimSpace(code)
	binding, _ := bindingHash(browserBinding).(string)
	if email == "" || len(code) != signInCodeDigits || binding == "" {
		return nil, ErrCodeInvalid
	}
	row, err := scanPendingSignIn(s.db.QueryRowContext(ctx, `
		SELECT `+pendingSignInColumns+`
		FROM auth_tokens t LEFT JOIN users u ON u.id = t.user_id
		WHERE t.token_type = $1 AND t.email = $2 AND t.browser_binding_hash = $3 AND t.code_hash = $4
		  AND t.used_at IS NULL AND t.expires_at > NOW()
		ORDER BY t.created_at DESC LIMIT 1
	`, signInTokenType, email, binding, signInCodeHash(email, code)))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCodeInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("query sign-in code: %w", err)
	}
	return row, nil
}

func (s *UserAuthService) userForSignIn(ctx context.Context, row *pendingSignIn) (*User, error) {
	if row.email != "" {
		user, err := s.GetOrCreateUser(ctx, row.email)
		if err != nil {
			return nil, fmt.Errorf("get or create user: %w", err)
		}
		return user, nil
	}
	if row.userID.Valid {
		user, err := s.GetUserByID(ctx, row.userID.String)
		if err != nil {
			return nil, fmt.Errorf("get user: %w", err)
		}
		return user, nil
	}
	return nil, ErrTokenInvalid
}

// NormalizeEmail trims and lowercases an address.
func NormalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

// LooksLikeEmail applies the minimal structural check shared by transports.
func LooksLikeEmail(email string) bool {
	at := strings.LastIndex(email, "@")
	if at < 1 || at == len(email)-1 || len(email) > 254 || strings.ContainsAny(email, " \t\r\n<>\"") {
		return false
	}
	domain := email[at+1:]
	return strings.Contains(domain, ".") && !strings.HasPrefix(domain, ".") && !strings.HasSuffix(domain, ".")
}

// MaskEmail keeps enough of an address to recognize it.
func MaskEmail(email string) string {
	at := strings.LastIndex(email, "@")
	if at < 1 {
		return ""
	}
	local, domain := []rune(email[:at]), email[at+1:]
	visible := 1
	if len(local) > 4 {
		visible = 2
	}
	return string(local[:visible]) + "•••@" + domain
}

func signInFlow(contextValue json.RawMessage) string {
	if len(contextValue) == 0 {
		return SignInFlowBrowser
	}
	var fields map[string]any
	if json.Unmarshal(contextValue, &fields) != nil {
		return SignInFlowBrowser
	}
	if desktop, _ := fields["desktop_link"].(bool); desktop {
		return SignInFlowDesktopLink
	}
	if redirect, _ := fields["redirect_uri"].(string); redirect != "" {
		return SignInFlowNativeApp
	}
	return SignInFlowBrowser
}

// normalizeSignInContext accepts only a small JSON object whose callback, if
// present, targets the local machine or the vrooli scheme.
func normalizeSignInContext(raw json.RawMessage) (any, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}
	if len(trimmed) > maxSignInContextSize {
		return nil, ErrContextInvalid
	}
	var fields map[string]any
	if err := json.Unmarshal([]byte(trimmed), &fields); err != nil {
		return nil, ErrContextInvalid
	}
	if redirect, ok := fields["redirect_uri"]; ok {
		value, isString := redirect.(string)
		if !isString || !allowedSignInCallback(value) {
			return nil, ErrContextInvalid
		}
	}
	compact, err := json.Marshal(fields)
	if err != nil {
		return nil, ErrContextInvalid
	}
	return string(compact), nil
}

func allowedSignInCallback(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.User != nil {
		return false
	}
	switch parsed.Scheme {
	case "vrooli":
		return true
	case "http":
		host := strings.ToLower(parsed.Hostname())
		return host == "127.0.0.1" || host == "::1" || host == "localhost"
	}
	return false
}

func signInCodeHash(email, code string) string {
	return HashToken("sign-in-code:" + NormalizeEmail(email) + ":" + strings.TrimSpace(code))
}

func bindingHash(binding string) any {
	binding = strings.TrimSpace(binding)
	if len(binding) < 16 {
		return nil
	}
	return HashToken("browser-binding:" + binding)
}

func randomHex(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

func randomDigits(digits int) (string, error) {
	limit := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(digits)), nil)
	value, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", digits, value.Int64()), nil
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}
