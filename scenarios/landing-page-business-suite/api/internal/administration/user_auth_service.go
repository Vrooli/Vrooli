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
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/vrooli/api-core/consumeridentity"
)

// MagicLinkSender sends a completed magic-link URL without coupling identity
// policy to a particular mail provider.
type MagicLinkSender interface {
	SendMagicLink(email, magicLink, appName string) error
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
	magicLinkTTL   time.Duration
	baseURL        string // For magic link URLs
	appName        string // For email subject lines
	// Test hook for capturing generated tokens
	onMagicLinkGenerated MagicLinkTokenCallback
	log                  func(string, map[string]interface{})
	logError             func(string, map[string]interface{})
}

// UseTokenCallback sets a callback that will be invoked when a magic link is generated.
// This follows the Use*() injection pattern for test seams.
func (s *UserAuthService) UseTokenCallback(callback MagicLinkTokenCallback) {
	s.onMagicLinkGenerated = callback
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
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"` // "Bearer"
}

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
	MagicLinkTTL          time.Duration
	Log                   func(string, map[string]interface{})
	LogError              func(string, map[string]interface{})
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
		magicLinkTTL:   magicLinkTTL,
		baseURL:        baseURL,
		appName:        appName,
		log:            log,
		logError:       logError,
	}
}

// RequestMagicLink generates and sends a magic link to the user's email.
func (s *UserAuthService) RequestMagicLink(ctx context.Context, email, ipAddress, userAgent string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || !strings.Contains(email, "@") {
		return errors.New("valid email address is required")
	}
	if strings.TrimSpace(s.baseURL) == "" {
		return errors.New("AUTH_MAGIC_LINK_BASE_URL must be configured before requesting a magic link")
	}

	// Get or create user
	user, err := s.GetOrCreateUser(ctx, email)
	if err != nil {
		return fmt.Errorf("get or create user: %w", err)
	}

	// Generate secure random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("generate token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	// Hash the token for storage
	tokenHash := HashToken(token)

	// Store token in database (use UTC for consistent timezone handling with PostgreSQL)
	expiresAt := time.Now().UTC().Add(s.magicLinkTTL)
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO auth_tokens (user_id, token_hash, token_type, expires_at, ip_address, user_agent)
		VALUES ($1, $2, 'magic_link', $3, $4::inet, $5)
	`, user.ID, tokenHash, expiresAt, toNullableParam(ipAddress), toNullableParam(userAgent))
	if err != nil {
		return fmt.Errorf("store auth token: %w", err)
	}

	// Build magic link URL
	magicLink := fmt.Sprintf("%s?token=%s", s.baseURL, url.QueryEscape(token))

	// Call test hook if configured (for capturing tokens in tests)
	if s.onMagicLinkGenerated != nil {
		s.onMagicLinkGenerated(email, token, magicLink)
	}

	// Send email
	if s.emailService != nil {
		if err := s.emailService.SendMagicLink(email, magicLink, s.appName); err != nil {
			s.logError("send_magic_link_failed", map[string]interface{}{
				"error": err.Error(),
				"email": email,
			})
		}
		// Don't return error to user (don't reveal email existence)
	}

	s.log("magic_link_requested", map[string]interface{}{
		"level":   "info",
		"user_id": user.ID,
		"email":   email,
	})

	return nil
}

// VerifyMagicLink validates a magic link token and returns tokens for the user.
func (s *UserAuthService) VerifyMagicLink(ctx context.Context, token, ipAddress, userAgent string) (*TokenPair, *User, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, nil, ErrTokenInvalid
	}

	tokenHash := HashToken(token)

	// Find and validate token
	var tokenID, userID string
	var expiresAt time.Time
	var usedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, expires_at, used_at
		FROM auth_tokens
		WHERE token_hash = $1 AND token_type = 'magic_link'
	`, tokenHash).Scan(&tokenID, &userID, &expiresAt, &usedAt)

	if err == sql.ErrNoRows {
		return nil, nil, ErrTokenInvalid
	}
	if err != nil {
		return nil, nil, fmt.Errorf("query auth token: %w", err)
	}

	// Check if token was already used
	if usedAt.Valid {
		return nil, nil, ErrTokenUsed
	}

	// Check if token has expired (use UTC for consistent timezone handling)
	if time.Now().UTC().After(expiresAt) {
		return nil, nil, ErrTokenExpired
	}

	// Mark token as used (atomic)
	result, err := s.db.ExecContext(ctx, `
		UPDATE auth_tokens
		SET used_at = NOW()
		WHERE id = $1 AND used_at IS NULL
	`, tokenID)
	if err != nil {
		return nil, nil, fmt.Errorf("mark token used: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// Token was used between our check and update (race condition)
		return nil, nil, ErrTokenUsed
	}

	// Get user
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("get user: %w", err)
	}

	// Mark email as verified and update last login
	now := time.Now().UTC()
	_, err = s.db.ExecContext(ctx, `
		UPDATE users
		SET email_verified = TRUE, last_login_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, userID)
	if err != nil {
		s.logError("update_user_after_magic_link_failed", map[string]interface{}{
			"error":   err.Error(),
			"user_id": userID,
		})
	} else {
		// Update the user object to reflect the database changes
		user.EmailVerified = true
		user.LastLoginAt = &now
	}

	// Create session and generate tokens
	tokenPair, err := s.CreateSession(ctx, user, ipAddress, userAgent)
	if err != nil {
		return nil, nil, fmt.Errorf("create session: %w", err)
	}

	s.log("magic_link_verified", map[string]interface{}{
		"level":   "info",
		"user_id": userID,
		"email":   user.Email,
	})

	return tokenPair, user, nil
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
	// Generate refresh token
	refreshBytes := make([]byte, 32)
	if _, err := rand.Read(refreshBytes); err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}
	refreshToken := hex.EncodeToString(refreshBytes)
	refreshHash := HashToken(refreshToken)

	// Create session (use UTC for consistent timezone handling)
	var sessionID string
	expiresAt := time.Now().UTC().Add(s.refreshTTL)

	err := s.db.QueryRowContext(ctx, `
		INSERT INTO user_sessions (user_id, refresh_token_hash, expires_at, ip_address, user_agent)
		VALUES ($1, $2, $3, $4::inet, $5)
		RETURNING id
	`, user.ID, refreshHash, expiresAt, toNullableParam(ipAddress), toNullableParam(userAgent)).Scan(&sessionID)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	// Generate access token
	accessToken, accessExpiresAt, err := s.GenerateAccessToken(user.ID, user.Email, sessionID)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    accessExpiresAt,
		TokenType:    "Bearer",
	}, nil
}

// generateAccessToken creates a signed JWT access token.
func (s *UserAuthService) GenerateAccessToken(userID, email, sessionID string) (string, time.Time, error) {
	if s == nil || s.consumerSigner == nil {
		return "", time.Time{}, errors.New("consumer signing key is unavailable")
	}
	return s.consumerSigner.Sign(consumeridentity.Claims{
		Subject:   userID,
		UserID:    userID,
		Email:     email,
		SessionID: sessionID,
	})
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
