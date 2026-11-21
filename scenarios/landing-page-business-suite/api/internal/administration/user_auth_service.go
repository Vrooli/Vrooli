package administration

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/vrooli/api-core/consumeridentity"
	entitlementclient "github.com/vrooli/vrooli/packages/entitlementclient-go"
)

// MagicLinkSender sends a completed magic-link URL without coupling identity
// policy to a particular mail provider.
type MagicLinkSender interface {
	SendMagicLink(email, magicLink, appName string) error
}

// SignInMessageSender delivers a sign-in email that carries both the one-use
// link and the short code. Senders that implement it are preferred over
// MagicLinkSender so the code reaches the user.
type SignInMessageSender interface {
	SendSignIn(message SignInMessage) (*SignInDelivery, error)
}

// SignInDelivery records the provider that accepted a sign-in message and
// the provider's correlation identifier when one is available.
type SignInDelivery struct {
	Provider          string
	ProviderMessageID string
}

// MagicLinkTokenCallback receives generated token details. It is a narrow
// deterministic test seam; production composition leaves it unset.
type MagicLinkTokenCallback func(email, token, magicLink string)

// UserAuthStore is the context-aware persistence contract for user identity,
// tokens, and sessions.
//
// seam: UserAuthStore keeps user authentication persistence independent of a
// concrete pool and preserves request-scoped test isolation.
type UserAuthStore interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// UserAuthService handles user authentication (magic links + JWT).
type UserAuthService struct {
	db             UserAuthStore
	emailService   MagicLinkSender
	consumerSigner *consumeridentity.Signer
	consumerKeys   *consumeridentity.KeySet
	jwtIssuer      string
	accessTTL      time.Duration
	consumerLeeway time.Duration
	refreshTTL     time.Duration
	idleTTL        time.Duration
	absoluteTTL    time.Duration
	refreshGrace   time.Duration
	magicLinkTTL   time.Duration
	baseURL        string // For magic link URLs
	appName        string // For email subject lines
	// Test hook for capturing generated tokens
	onMagicLinkGenerated  MagicLinkTokenCallback
	onSignInCodeGenerated func(email, code string)
	log                   func(string, map[string]interface{})
	logError              func(string, map[string]interface{})
	outboxEnabled         bool
	senderIdentity        string
}

// UseTokenCallback sets a callback that will be invoked when a magic link is generated.
// This follows the Use*() injection pattern for test seams.
func (s *UserAuthService) UseTokenCallback(callback MagicLinkTokenCallback) {
	s.onMagicLinkGenerated = callback
}

// UseCodeCallback captures generated sign-in codes in tests.
func (s *UserAuthService) UseCodeCallback(callback func(email, code string)) {
	s.onSignInCodeGenerated = callback
}

// User represents an authenticated user.
type User struct {
	ID               string     `json:"id"`
	Email            string     `json:"email"`
	EmailVerified    bool       `json:"email_verified"`
	StripeCustomerID *string    `json:"stripe_customer_id,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	LastLoginAt      *time.Time `json:"last_login_at,omitempty"`
}

// TokenPair contains access and refresh tokens.
type TokenPair struct {
	AccessToken      string    `json:"access_token"`
	RefreshToken     string    `json:"refresh_token"`
	ExpiresAt        time.Time `json:"expires_at"`
	TokenType        string    `json:"token_type"` // "Bearer"
	SessionExpiresAt time.Time `json:"-"`
	GraceResponse    bool      `json:"-"`
	SessionID        string    `json:"-"`
}

type RefreshSource string

const (
	RefreshSourceBody   RefreshSource = "body"
	RefreshSourceCookie RefreshSource = "cookie"
)

// UserClaims are the JWT claims for user authentication.
type UserClaims struct {
	jwt.RegisteredClaims
	UserID    string `json:"uid"`
	Email     string `json:"email"`
	SessionID string `json:"sid"`
}

// ErrTokenExpired is returned when a token has expired.
var ErrTokenExpired = errors.New("token has expired")

// ErrTokenUsed is returned when a magic link token has already been used.
var ErrTokenUsed = errors.New("token has already been used")

// ErrTokenInvalid is returned when a token is invalid.
var ErrTokenInvalid = errors.New("invalid token")

// ErrSessionRevoked is returned when a session has been revoked.
var ErrSessionRevoked = errors.New("session has been revoked")

// ErrSessionExpired is returned when a session exceeded idle or absolute TTL.
var ErrSessionExpired = errors.New("session has expired")

// UserAuthServiceOptions supplies application-owned configuration at the
// composition boundary. Identity policy does not read environment variables or
// depend on a root-package mail implementation.
type UserAuthServiceOptions struct {
	Store                 UserAuthStore
	EmailService          MagicLinkSender
	ConsumerSigningKeyPEM string
	ConsumerSigningKeyID  string
	ConsumerPreviousKeys  []consumeridentity.PublicKey
	JWTIssuer             string
	BaseURL               string
	AppName               string
	AccessTTL             time.Duration
	ConsumerClockSkew     time.Duration
	RefreshTTL            time.Duration
	IdleTTL               time.Duration
	AbsoluteTTL           time.Duration
	ReauthMaxAge          time.Duration
	RefreshGrace          time.Duration
	MagicLinkTTL          time.Duration
	Log                   func(string, map[string]interface{})
	LogError              func(string, map[string]interface{})
	OutboxEnabled         bool
	SenderIdentity        string
}

// NewUserAuthService creates a user-authentication service from explicit
// composition inputs. Authentication origins are deliberately not defaulted:
// an unset origin must not result in a magic link for an unrelated localhost
// application.
func NewUserAuthService(opts UserAuthServiceOptions) *UserAuthService {
	log := opts.Log
	if log == nil {
		log = func(string, map[string]interface{}) {}
	}
	logError := opts.LogError
	if logError == nil {
		logError = func(string, map[string]interface{}) {}
	}
	jwtIssuer := opts.JWTIssuer
	if jwtIssuer == "" {
		jwtIssuer = "landing-page-business-suite"
	}

	baseURL := opts.BaseURL

	appName := opts.AppName
	if appName == "" {
		appName = "App"
	}

	accessTTL := opts.AccessTTL
	if accessTTL == 0 {
		accessTTL = 15 * time.Minute
	}
	refreshTTL := opts.RefreshTTL
	if refreshTTL == 0 {
		refreshTTL = 7 * 24 * time.Hour
	}
	idleTTL := opts.IdleTTL
	if idleTTL == 0 {
		idleTTL = 30 * 24 * time.Hour
	}
	absoluteTTL := opts.AbsoluteTTL
	if absoluteTTL == 0 {
		absoluteTTL = 90 * 24 * time.Hour
	}
	refreshGrace := opts.RefreshGrace
	if refreshGrace == 0 {
		refreshGrace = 30 * time.Second
	}
	magicLinkTTL := opts.MagicLinkTTL
	if magicLinkTTL == 0 {
		magicLinkTTL = 15 * time.Minute
	}

	keyID := strings.TrimSpace(opts.ConsumerSigningKeyID)
	if keyID == "" {
		keyID = "local-development"
	}
	var signingKey *rsa.PrivateKey
	var keyErr error
	if strings.TrimSpace(opts.ConsumerSigningKeyPEM) != "" {
		signingKey, keyErr = consumeridentity.ParsePrivateKeyPEM(opts.ConsumerSigningKeyPEM)
	} else {
		var generated *consumeridentity.Signer
		generated, keyErr = consumeridentity.GenerateSigner(keyID, jwtIssuer, accessTTL)
		if generated != nil {
			signingKey = generated.Private
		}
	}
	if keyErr != nil {
		logError("consumer_signing_key_invalid", map[string]interface{}{"error": keyErr.Error()})
		return nil
	}
	signer, keyErr := consumeridentity.NewSigner(keyID, signingKey, jwtIssuer, accessTTL)
	if keyErr != nil {
		logError("consumer_signing_key_invalid", map[string]interface{}{"error": keyErr.Error()})
		return nil
	}
	keys := consumeridentity.NewKeySet(consumeridentity.PublicKey{ID: keyID, Key: &signingKey.PublicKey})
	for _, previous := range opts.ConsumerPreviousKeys {
		keys.Add(previous)
	}
	return &UserAuthService{
		db:             opts.Store,
		emailService:   opts.EmailService,
		consumerSigner: signer,
		consumerKeys:   keys,
		jwtIssuer:      jwtIssuer,
		accessTTL:      accessTTL,
		consumerLeeway: opts.ConsumerClockSkew,
		refreshTTL:     refreshTTL,
		idleTTL:        idleTTL,
		absoluteTTL:    absoluteTTL,
		refreshGrace:   refreshGrace,
		magicLinkTTL:   magicLinkTTL,
		baseURL:        baseURL,
		appName:        appName,
		log:            log,
		logError:       logError,
		outboxEnabled:  opts.OutboxEnabled,
		senderIdentity: strings.TrimSpace(opts.SenderIdentity),
	}
}

// GetOrCreateUser returns an existing user by email or creates a new one.
func (s *UserAuthService) GetOrCreateUser(ctx context.Context, email string) (*User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return nil, errors.New("email is required")
	}

	// Try to get existing user first
	user, err := s.GetUserByEmail(ctx, email)
	if err == nil && user != nil {
		return user, nil
	}

	// Create new user
	var userID string
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO users (email)
		VALUES ($1)
		ON CONFLICT (email) DO UPDATE SET updated_at = NOW()
		RETURNING id
	`, email).Scan(&userID)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return s.GetUserByID(ctx, userID)
}

// GetUserByID returns a user by their ID.
func (s *UserAuthService) GetUserByID(ctx context.Context, userID string) (*User, error) {
	var user User
	var stripeCustomerID sql.NullString
	var lastLoginAt sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, email_verified, stripe_customer_id, created_at, updated_at, last_login_at
		FROM users
		WHERE id = $1
	`, userID).Scan(&user.ID, &user.Email, &user.EmailVerified, &stripeCustomerID, &user.CreatedAt, &user.UpdatedAt, &lastLoginAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found: %s", userID)
	}
	if err != nil {
		return nil, err
	}

	if stripeCustomerID.Valid {
		user.StripeCustomerID = &stripeCustomerID.String
	}
	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}

	return &user, nil
}

// GetUserByEmail returns a user by their email.
func (s *UserAuthService) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	var user User
	var stripeCustomerID sql.NullString
	var lastLoginAt sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, email_verified, stripe_customer_id, created_at, updated_at, last_login_at
		FROM users
		WHERE email = $1
	`, email).Scan(&user.ID, &user.Email, &user.EmailVerified, &stripeCustomerID, &user.CreatedAt, &user.UpdatedAt, &lastLoginAt)

	if err == sql.ErrNoRows {
		return nil, nil // User doesn't exist, not an error
	}
	if err != nil {
		return nil, err
	}

	if stripeCustomerID.Valid {
		user.StripeCustomerID = &stripeCustomerID.String
	}
	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}

	return &user, nil
}

// LinkStripeCustomer associates a Stripe customer ID with a user.
func (s *UserAuthService) LinkStripeCustomer(ctx context.Context, email, customerID string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	customerID = strings.TrimSpace(customerID)

	if email == "" || customerID == "" {
		return errors.New("email and customer ID are required")
	}

	// Get or create user
	user, err := s.GetOrCreateUser(ctx, email)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE users
		SET stripe_customer_id = $1, updated_at = NOW()
		WHERE id = $2
	`, customerID, user.ID)
	if err != nil {
		return fmt.Errorf("link stripe customer: %w", err)
	}

	s.log("stripe_customer_linked", map[string]interface{}{
		"level":       "info",
		"user_id":     user.ID,
		"email":       email,
		"customer_id": customerID,
	})

	return nil
}

// createSession creates a new session and returns a token pair.
func (s *UserAuthService) CreateSession(ctx context.Context, user *User, ipAddress, userAgent string) (*TokenPair, error) {
	return s.CreateSessionWithMetadata(ctx, user, ipAddress, userAgent, "email_code", time.Now().UTC())
}

// CreateSessionWithMetadata creates a session with an explicit authentication
// method and proof time so absolute and recent-auth policy share one source.
func (s *UserAuthService) CreateSessionWithMetadata(ctx context.Context, user *User, ipAddress, userAgent, authMethod string, authenticatedAt time.Time) (*TokenPair, error) {
	// Generate refresh token
	refreshBytes := make([]byte, 32)
	if _, err := rand.Read(refreshBytes); err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}
	refreshToken := hex.EncodeToString(refreshBytes)
	refreshHash := HashToken(refreshToken)

	// Create session (use UTC for consistent timezone handling)
	var sessionID string
	if authenticatedAt.IsZero() {
		authenticatedAt = time.Now().UTC()
	}
	absoluteExpiresAt := authenticatedAt.Add(s.absoluteTTL)
	expiresAt := time.Now().UTC().Add(s.idleTTL)
	if expiresAt.After(absoluteExpiresAt) {
		expiresAt = absoluteExpiresAt
	}

	err := s.db.QueryRowContext(ctx, `
		INSERT INTO user_sessions (user_id, refresh_token_hash, expires_at, absolute_expires_at, authenticated_at, auth_method, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7::inet, $8)
		RETURNING id
	`, user.ID, refreshHash, expiresAt, absoluteExpiresAt, authenticatedAt, authMethod, toNullableParam(ipAddress), toNullableParam(userAgent)).Scan(&sessionID)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	// Generate access token
	accessToken, accessExpiresAt, err := s.GenerateAccessToken(user.ID, user.Email, sessionID)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		ExpiresAt:        accessExpiresAt,
		TokenType:        "Bearer",
		SessionExpiresAt: expiresAt,
		SessionID:        sessionID,
	}, nil
}

// generateAccessToken creates a signed JWT access token.
func (s *UserAuthService) GenerateAccessToken(userID, email, sessionID string) (string, time.Time, error) {
	if s == nil || s.consumerSigner == nil {
		return "", time.Time{}, errors.New("consumer signing key is unavailable")
	}
	claims := consumeridentity.Claims{
		Subject:   userID,
		UserID:    userID,
		Email:     email,
		SessionID: sessionID,
	}
	if s.db != nil && strings.TrimSpace(sessionID) != "" {
		var authenticatedAt time.Time
		if err := s.db.QueryRowContext(context.Background(), `SELECT authenticated_at FROM user_sessions WHERE id = $1`, sessionID).Scan(&authenticatedAt); err == nil {
			claims.AuthTime = authenticatedAt.Unix()
		}
	}
	return s.consumerSigner.Sign(claims)
}

// ValidateSession checks server-side revocation and both customer expiry
// limits. JWT validity alone is not sufficient for an active session.
func (s *UserAuthService) ValidateSession(ctx context.Context, sessionID string) error {
	var revoked bool
	var expiresAt, absoluteExpiresAt time.Time
	err := s.db.QueryRowContext(ctx, `SELECT revoked, expires_at, absolute_expires_at FROM user_sessions WHERE id = $1`, sessionID).Scan(&revoked, &expiresAt, &absoluteExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrSessionRevoked
	}
	if err != nil {
		return fmt.Errorf("validate session: %w", err)
	}
	if revoked {
		return ErrSessionRevoked
	}
	now := time.Now().UTC()
	if now.After(expiresAt) || now.After(absoluteExpiresAt) {
		return ErrSessionExpired
	}
	return nil
}

// SignEntitlementLease signs the authority-owned entitlement snapshot. The
// private consumer key never leaves LPBS; bundled apps receive only the
// short-lived lease and the public JWKS needed to verify it.
func (s *UserAuthService) SignEntitlementLease(payload entitlementclient.Payload) (string, error) {
	if s == nil || s.consumerSigner == nil {
		return "", errors.New("consumer signing key is unavailable")
	}
	return entitlementclient.Sign(payload, s.consumerSigner.KeyID, s.consumerSigner.Private)
}

// HashToken returns the SHA-256 hash of a token for persistence comparisons.
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// toNullableParam returns nil if the string is empty, otherwise the string value.
// Used for nullable database column parameters.
func toNullableParam(s string) interface{} {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return s
}
