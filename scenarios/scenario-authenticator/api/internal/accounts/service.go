package accounts

import (
	"context"
	"errors"
	"strings"
	"time"

	"scenario-authenticator/internal/audit"
	"scenario-authenticator/internal/authcrypto"
	"scenario-authenticator/internal/clock"
	"scenario-authenticator/internal/realm"
	"scenario-authenticator/internal/sessions"
)

// Lockout defaults: after N consecutive failed logins, lock the account for a
// cooldown. The columns existed in the old schema but were never enforced.
const (
	defaultLockThreshold = 5
	defaultLockDuration  = 15 * time.Minute
)

// Service-level errors the Connect handler maps to codes. Login failures are
// deliberately indistinguishable (anti-enumeration): unknown account and wrong
// password both yield ErrInvalidCredentials with an identical message.
var (
	// ErrInvalidCredentials — unknown account or wrong password (UNAUTHENTICATED).
	ErrInvalidCredentials = errors.New("invalid email or password")
	// ErrAccountLocked — too many failed attempts (PERMISSION_DENIED).
	ErrAccountLocked = errors.New("account temporarily locked due to failed login attempts")
)

// InvalidInputError carries a validation message surfaced to the caller
// (INVALID_ARGUMENT). Used for malformed email / weak password / unknown realm
// on register.
type InvalidInputError struct{ Msg string }

func (e InvalidInputError) Error() string { return e.Msg }

// RequestMeta is the request-scoped context recorded on sessions + audit rows.
type RequestMeta struct {
	IP        string
	UserAgent string
}

// AuthResult is the outcome of register/login: the account plus its issued
// token pair.
type AuthResult struct {
	Account         Account
	AccessToken     string
	RefreshToken    string
	AccessExpiresAt time.Time
}

// RegisterParams / LoginParams are the service inputs.
type RegisterParams struct {
	Email    string
	Password string
	Username string
	Realm    string
}

type LoginParams struct {
	Email    string
	Password string
	Realm    string
}

// ValidatedToken is the result of a successful Validate.
type ValidatedToken struct {
	UserID    string
	Email     string
	Roles     []string
	Realm     string
	ExpiresAt time.Time
}

// Service orchestrates the account auth core over the persistence, crypto,
// hot-state, and audit seams.
type Service struct {
	repo          Repository
	signer        *authcrypto.Signer
	sessions      *sessions.Manager
	audit         audit.Logger
	clock         clock.Clock
	lockThreshold int
	lockDuration  time.Duration
}

// ServiceConfig configures a Service. Zero lockout fields fall back to defaults.
type ServiceConfig struct {
	Repo          Repository
	Signer        *authcrypto.Signer
	Sessions      *sessions.Manager
	Audit         audit.Logger
	Clock         clock.Clock
	LockThreshold int
	LockDuration  time.Duration
}

// NewService constructs the orchestrator.
func NewService(cfg ServiceConfig) *Service {
	if cfg.Clock == nil {
		cfg.Clock = clock.System{}
	}
	if cfg.LockThreshold <= 0 {
		cfg.LockThreshold = defaultLockThreshold
	}
	if cfg.LockDuration <= 0 {
		cfg.LockDuration = defaultLockDuration
	}
	return &Service{
		repo: cfg.Repo, signer: cfg.Signer, sessions: cfg.Sessions, audit: cfg.Audit,
		clock: cfg.Clock, lockThreshold: cfg.LockThreshold, lockDuration: cfg.LockDuration,
	}
}

// Register creates an account and returns it auto-signed-in.
func (s *Service) Register(ctx context.Context, p RegisterParams, meta RequestMeta) (AuthResult, error) {
	email := strings.TrimSpace(p.Email)
	if !ValidateEmail(email) {
		return AuthResult{}, InvalidInputError{Msg: "Invalid email format"}
	}
	if ok, msg := ValidatePassword(p.Password); !ok {
		return AuthResult{}, InvalidInputError{Msg: msg}
	}
	realmID := resolveRealm(p.Realm)
	aud, err := s.repo.RealmAudience(ctx, realmID)
	if err != nil {
		if errors.Is(err, ErrRealmNotFound) {
			return AuthResult{}, InvalidInputError{Msg: "unknown realm"}
		}
		return AuthResult{}, err
	}
	hash, err := HashPassword(p.Password)
	if err != nil {
		return AuthResult{}, err
	}
	acc, err := s.repo.Create(ctx, CreateInput{
		RealmID: realmID, Email: email, Username: strings.TrimSpace(p.Username),
		PasswordHash: hash, Roles: []string{"user"},
	})
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			return AuthResult{}, ErrEmailTaken
		}
		return AuthResult{}, err
	}
	res, err := s.issueTokens(ctx, acc, aud, meta)
	if err != nil {
		return AuthResult{}, err
	}
	s.logEvent(ctx, acc.ID, realmID, "user.registered", meta, true, nil)
	return res, nil
}

// Login verifies credentials and issues a fresh token pair, enforcing lockout.
func (s *Service) Login(ctx context.Context, p LoginParams, meta RequestMeta) (AuthResult, error) {
	email := strings.TrimSpace(p.Email)
	realmID := resolveRealm(p.Realm)
	aud, err := s.repo.RealmAudience(ctx, realmID)
	if err != nil {
		// Unknown realm must not leak; treat as invalid credentials.
		return AuthResult{}, ErrInvalidCredentials
	}

	acc, hash, err := s.repo.FindByEmail(ctx, realmID, email)
	if err != nil {
		if errors.Is(err, ErrAccountNotFound) {
			s.logEvent(ctx, "", realmID, "user.login.failed", meta, false,
				map[string]any{"email": email, "reason": "user_not_found"})
			return AuthResult{}, ErrInvalidCredentials
		}
		return AuthResult{}, err
	}

	now := s.clock.Now()
	if acc.Locked(now) {
		s.logEvent(ctx, acc.ID, realmID, "user.login.locked", meta, false,
			map[string]any{"reason": "account_locked"})
		return AuthResult{}, ErrAccountLocked
	}

	ok, _ := VerifyPassword(p.Password, hash)
	if !ok {
		s.recordFailedLogin(ctx, acc, realmID, meta)
		return AuthResult{}, ErrInvalidCredentials
	}

	if err := s.repo.SetLoginSuccess(ctx, acc.ID, now); err != nil {
		return AuthResult{}, err
	}
	res, err := s.issueTokens(ctx, acc, aud, meta)
	if err != nil {
		return AuthResult{}, err
	}
	s.logEvent(ctx, acc.ID, realmID, "user.logged_in", meta, true, nil)
	return res, nil
}

// Validate verifies an access token server-side against the default realm's
// audience (a token for any other aud is rejected — OT-P0-008) and that it is
// not blacklisted.
func (s *Service) Validate(ctx context.Context, accessToken string) (ValidatedToken, bool, error) {
	if strings.TrimSpace(accessToken) == "" {
		return ValidatedToken{}, false, nil
	}
	blacklisted, err := s.sessions.IsBlacklisted(ctx, accessToken)
	if err != nil {
		return ValidatedToken{}, false, err
	}
	if blacklisted {
		return ValidatedToken{}, false, nil
	}
	aud, err := s.repo.RealmAudience(ctx, realm.DefaultID)
	if err != nil {
		return ValidatedToken{}, false, err
	}
	claims, err := s.signer.Validate(accessToken, aud)
	if err != nil {
		return ValidatedToken{}, false, nil
	}
	var exp time.Time
	if claims.ExpiresAt != nil {
		exp = claims.ExpiresAt.Time
	}
	return ValidatedToken{
		UserID: claims.UserID, Email: claims.Email, Roles: claims.Roles,
		Realm: realm.DefaultID, ExpiresAt: exp,
	}, true, nil
}

func (s *Service) issueTokens(ctx context.Context, acc Account, aud string, meta RequestMeta) (AuthResult, error) {
	access, err := s.signer.Sign(authcrypto.TokenInput{
		UserID: acc.ID, Email: acc.Email, Roles: acc.Roles, Audience: aud,
	})
	if err != nil {
		return AuthResult{}, err
	}
	refresh, err := s.sessions.IssueRefresh(ctx, acc.ID)
	if err != nil {
		return AuthResult{}, err
	}
	if _, err := s.sessions.StoreSession(ctx, acc.ID, meta.IP, meta.UserAgent); err != nil {
		return AuthResult{}, err
	}
	return AuthResult{
		Account: acc, AccessToken: access, RefreshToken: refresh,
		AccessExpiresAt: s.clock.Now().Add(s.signer.Expiry()),
	}, nil
}

func (s *Service) recordFailedLogin(ctx context.Context, acc Account, realmID string, meta RequestMeta) {
	attempts := acc.FailedLoginAttempts + 1
	var lockedUntil time.Time
	locked := false
	if attempts >= s.lockThreshold {
		lockedUntil = s.clock.Now().Add(s.lockDuration)
		locked = true
	}
	_ = s.repo.SetLoginFailure(ctx, acc.ID, attempts, lockedUntil)
	md := map[string]any{"reason": "invalid_password", "attempts": attempts}
	action := "user.login.failed"
	if locked {
		action = "user.account.locked"
		md["locked_until"] = lockedUntil.UTC().Format(time.RFC3339)
	}
	s.logEvent(ctx, acc.ID, realmID, action, meta, false, md)
}

func (s *Service) logEvent(ctx context.Context, userID, realmID, action string, meta RequestMeta, success bool, md map[string]any) {
	if s.audit == nil {
		return
	}
	// Best-effort: an audit write failure never fails the auth operation.
	_ = s.audit.Log(ctx, audit.Event{
		UserID: userID, RealmID: realmID, Action: action,
		IPAddress: meta.IP, UserAgent: meta.UserAgent, Success: success, Metadata: md,
	})
}

func resolveRealm(r string) string {
	r = strings.TrimSpace(r)
	if r == "" {
		return realm.DefaultID
	}
	return r
}
