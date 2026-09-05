package accounts

import (
	"context"
	"errors"
	"strings"
	"time"

	"scenario-authenticator/internal/audit"
	"scenario-authenticator/internal/authcrypto"
	"scenario-authenticator/internal/authorization"
	"scenario-authenticator/internal/realm"
	"scenario-authenticator/internal/sessions"

	"github.com/vrooli/api-core/schedule"
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
	Roles    []string
	Scopes   []string
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
	Scopes    []string
	Realm     string
	ExpiresAt time.Time
}

// Service orchestrates the account auth core over the persistence, crypto,
// hot-state, and audit seams.
type Service struct {
	repo             Repository
	signer           *authcrypto.Signer
	sessions         *sessions.Manager
	audit            audit.Logger
	authorization    *authorization.Service
	machineBindings  MachineBindingStore
	breakGlass       BreakGlassProvisioner
	breakGlassIssuer BreakGlassIssuer
	clock            schedule.Clock
	lockThreshold    int
	lockDuration     time.Duration
}

// ServiceConfig configures a Service. Zero lockout fields fall back to defaults.
type ServiceConfig struct {
	Repo             Repository
	Signer           *authcrypto.Signer
	Sessions         *sessions.Manager
	Audit            audit.Logger
	Authorization    *authorization.Service
	MachineBindings  MachineBindingStore
	BreakGlass       BreakGlassProvisioner
	BreakGlassIssuer BreakGlassIssuer
	Clock            schedule.Clock
	LockThreshold    int
	LockDuration     time.Duration
}

// NewService constructs the orchestrator.
func NewService(cfg ServiceConfig) *Service {
	if cfg.Clock == nil {
		cfg.Clock = schedule.System()
	}
	if cfg.LockThreshold <= 0 {
		cfg.LockThreshold = defaultLockThreshold
	}
	if cfg.LockDuration <= 0 {
		cfg.LockDuration = defaultLockDuration
	}
	return &Service{
		repo: cfg.Repo, signer: cfg.Signer, sessions: cfg.Sessions, audit: cfg.Audit,
		authorization:    cfg.Authorization,
		machineBindings:  cfg.MachineBindings,
		breakGlass:       cfg.BreakGlass,
		breakGlassIssuer: cfg.BreakGlassIssuer,
		clock:            cfg.Clock, lockThreshold: cfg.LockThreshold, lockDuration: cfg.LockDuration,
	}
}

// IssueBreakGlass creates a time-boxed offline capability for an already
// authenticated account. Scope validation remains in the trust-posture
// issuer, which applies the account's current ceiling before signing.
func (s *Service) IssueBreakGlass(ctx context.Context, accessToken string, requested []string, meta RequestMeta) (string, time.Time, error) {
	vt, ok, err := s.Validate(ctx, accessToken)
	if err != nil || !ok || s.breakGlassIssuer == nil {
		return "", time.Time{}, ErrMachineExchangeRefused
	}
	token, expiresAt, err := s.breakGlassIssuer.Issue(ctx, vt.UserID, vt.Realm, requested, s.clock.Now().UTC())
	if err != nil {
		s.logEvent(ctx, vt.UserID, vt.Realm, "break_glass.issue.refused", meta, false, map[string]any{"reason": err.Error()})
		return "", time.Time{}, err
	}
	s.logEvent(ctx, vt.UserID, vt.Realm, "break_glass.issued", meta, true, map[string]any{"expires_at": expiresAt.UTC().Format(time.RFC3339)})
	return token, expiresAt, nil
}

// LinkMachineAccount binds an operator-authenticated account to a local
// principal. The signed-in principal may link only itself; administration of
// another account remains deliberately out of scope.
func (s *Service) LinkMachineAccount(ctx context.Context, accessToken, machineID, localPrincipal, realmID string, isDefault bool, meta RequestMeta) (MachineBinding, error) {
	vt, ok, err := s.Validate(ctx, accessToken)
	if err != nil {
		return MachineBinding{}, err
	}
	if !ok || s.machineBindings == nil {
		return MachineBinding{}, ErrMachineExchangeRefused
	}
	realmID = resolveRealm(realmID)
	if realmID != vt.Realm {
		return MachineBinding{}, ErrMachineExchangeRefused
	}
	binding := MachineBinding{
		MachineID: strings.TrimSpace(machineID), LocalPrincipal: strings.TrimSpace(localPrincipal),
		AccountID: vt.UserID, RealmID: realmID, IsDefault: isDefault, LinkedAt: s.clock.Now().UTC(),
	}
	if err := validateMachineBinding(binding); err != nil {
		return MachineBinding{}, err
	}
	if s.breakGlass != nil {
		scopes := []string{}
		var scopeErr error
		if s.authorization != nil {
			scopes, scopeErr = s.authorization.List(ctx, vt.UserID)
		}
		if scopeErr != nil {
			return MachineBinding{}, scopeErr
		}
		if err := s.breakGlass.Provision(ctx, vt.UserID, realmID, scopes, binding.LinkedAt); err != nil {
			return MachineBinding{}, err
		}
	}
	linked, err := s.machineBindings.LinkMachineBinding(ctx, binding)
	if err != nil {
		return MachineBinding{}, err
	}
	s.logEvent(ctx, vt.UserID, realmID, "machine.binding.linked", meta, true, map[string]any{"machine_id": linked.MachineID, "local_principal": linked.LocalPrincipal})
	return linked, nil
}

// ExchangeMachinePrincipal issues a normal access/refresh pair after the
// socket listener has authenticated the OS principal and binding resolution
// returns exactly one default account.
func (s *Service) ExchangeMachinePrincipal(ctx context.Context, machineID, localPrincipal string, meta RequestMeta) (AuthResult, error) {
	if s.machineBindings == nil {
		return AuthResult{}, ErrMachineExchangeRefused
	}
	binding, err := s.machineBindings.ResolveDefaultMachineBinding(ctx, strings.TrimSpace(machineID), strings.TrimSpace(localPrincipal))
	if err != nil {
		s.logEvent(ctx, "", "", "machine.exchange.refused", meta, false, map[string]any{"reason": err.Error()})
		return AuthResult{}, err
	}
	acc, err := s.repo.FindByID(ctx, binding.AccountID)
	if err != nil || acc.RealmID != binding.RealmID {
		s.logEvent(ctx, binding.AccountID, binding.RealmID, "machine.exchange.refused", meta, false, map[string]any{"reason": "account_binding_mismatch"})
		return AuthResult{}, ErrMachineExchangeRefused
	}
	aud, err := s.repo.RealmAudience(ctx, acc.RealmID)
	if err != nil {
		return AuthResult{}, err
	}
	res, err := s.issueTokens(ctx, acc, aud, meta)
	if err != nil {
		return AuthResult{}, err
	}
	s.logEvent(ctx, acc.ID, acc.RealmID, "machine.exchange.accepted", meta, true, map[string]any{"machine_id": binding.MachineID, "local_principal": binding.LocalPrincipal})
	return res, nil
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
		PasswordHash: hash, Roles: append([]string(nil), p.Roles...),
	})
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			return AuthResult{}, ErrEmailTaken
		}
		return AuthResult{}, err
	}
	if len(p.Scopes) > 0 {
		if s.authorization == nil {
			return AuthResult{}, errors.New("authorization service unavailable")
		}
		for _, scope := range p.Scopes {
			if _, err := s.authorization.Grant(ctx, acc.ID, scope, authorization.Meta{RealmID: realmID, IPAddress: meta.IP, UserAgent: meta.UserAgent}); err != nil {
				return AuthResult{}, err
			}
		}
		acc.Scopes = append([]string(nil), p.Scopes...)
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
		Scopes: nonNilStrings(claims.Scopes),
		Realm:  realm.DefaultID, ExpiresAt: exp,
	}, true, nil
}

func (s *Service) issueTokens(ctx context.Context, acc Account, aud string, meta RequestMeta) (AuthResult, error) {
	scopes, err := s.scopes(ctx, acc)
	if err != nil {
		return AuthResult{}, err
	}
	access, err := s.signer.Sign(authcrypto.TokenInput{
		UserID: acc.ID, Email: acc.Email, Roles: acc.Roles, Scopes: scopes, Audience: aud,
	})
	if err != nil {
		return AuthResult{}, err
	}
	acc.Scopes = scopes
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

func (s *Service) scopes(ctx context.Context, acc Account) ([]string, error) {
	if s.authorization == nil {
		return nonNilStrings(acc.Scopes), nil
	}
	scopes, err := s.authorization.List(ctx, acc.ID)
	if err != nil {
		return nil, err
	}
	return nonNilStrings(scopes), nil
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return append([]string(nil), values...)
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
