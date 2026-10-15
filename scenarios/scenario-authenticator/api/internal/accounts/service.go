package accounts

import (
	"context"
	"errors"
	"log"
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
	ErrMFARequired   = errors.New("multi-factor authentication is required")
	ErrMFACode       = errors.New("multi-factor authentication code is invalid")
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

// MFAProvider is the narrow boundary between password authentication and the
// authenticator-owned second-factor store. Login receives a challenge first;
// only VerifyLogin success allows the service to issue a full token pair.
type MFAProvider interface {
	Required(context.Context, string) (bool, error)
	Enrolled(context.Context, string) (bool, error)
	StartChallenge(context.Context, string) (string, time.Time, error)
	VerifyLogin(context.Context, string, string, string, string) (bool, error)
}

// AuthResult is the outcome of register/login: the account plus its issued
// token pair.
type AuthResult struct {
	Account         Account
	AccessToken     string
	RefreshToken    string
	AccessExpiresAt time.Time
	Audience        string
	MFARequired     bool
	MFAChallenge    string
}

// RegisterParams / LoginParams are the service inputs.
type RegisterParams struct {
	Email    string
	Password string
	Username string
	Realm    string
	Roles    []string
	Scopes   []string
	Resource string
}

type LoginParams struct {
	Email        string
	Password     string
	Realm        string
	TOTPCode     string
	RecoveryCode string
	MFAChallenge string
	Resource     string
}

// ValidatedToken is the result of a successful Validate.
type ValidatedToken struct {
	UserID      string
	Email       string
	Roles       []string
	Scopes      []string
	Realm       string
	ExpiresAt   time.Time
	Audience    string
	SessionID   string
	AuthVersion int64
	// PrincipalID is the target selected by an authorized management call. It
	// is empty for ordinary token validation and prevents transport handlers
	// from accidentally authorizing a different target.
	PrincipalID string
}

// Service orchestrates the account auth core over the persistence, crypto,
// hot-state, and audit seams.
type Service struct {
	repo              Repository
	signer            *authcrypto.Signer
	sessions          *sessions.Manager
	audit             audit.Logger
	authorization     *authorization.Service
	machineBindings   MachineBindingStore
	breakGlass        BreakGlassProvisioner
	breakGlassIssuer  BreakGlassIssuer
	mfa               MFAProvider
	clock             schedule.Clock
	lockThreshold     int
	lockDuration      time.Duration
	acceptedAudiences []string
	resourceAudiences map[string]string
	servicePrincipals map[string]ServicePrincipalProfile
	serviceCustody    ServiceCredentialCustodian
	humanAssertion    HumanAssertionVerifier
}

// HumanAssertionVerifier verifies a non-account human proof (a Cloudflare Access
// assertion) and returns the verified operator subject. It lets MintSessionDelegation
// attribute a tunnel operator who authenticated via Cloudflare Access rather than an
// account bearer — the same assertion web-console and agent-manager already trust,
// verified here against the web-console Access binding. Verifying it introduces no
// new trust: it mints only a no-capability, session-scoped owner label. An
// implementation returns ok=false for any missing/invalid/non-human proof.
type HumanAssertionVerifier interface {
	VerifyHuman(ctx context.Context, rawAssertion string) (subject, email string, ok bool)
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
	MFA              MFAProvider
	Clock            schedule.Clock
	LockThreshold    int
	LockDuration     time.Duration
	// AcceptedAudiences is a temporary verification-only audience migration set.
	AcceptedAudiences []string
	// ResourceAudiences maps a registered resource id to its audience. Empty
	// resource requests retain the default realm audience.
	ResourceAudiences map[string]string
	// ServicePrincipals is the operator-owned service-principal profile set.
	ServicePrincipals []ServicePrincipalProfile
	// ServiceCustody stores issued service-principal credentials.
	ServiceCustody ServiceCredentialCustodian
	// HumanAssertion optionally verifies a Cloudflare Access assertion as human
	// proof for MintSessionDelegation. Nil disables the Cf-Access mint path (the
	// account-bearer path still works).
	HumanAssertion HumanAssertionVerifier
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
		mfa:              cfg.MFA,
		clock:            cfg.Clock, lockThreshold: cfg.LockThreshold, lockDuration: cfg.LockDuration,
		acceptedAudiences: append([]string(nil), cfg.AcceptedAudiences...),
		resourceAudiences: copyStringMap(cfg.ResourceAudiences),
		servicePrincipals: indexServicePrincipals(cfg.ServicePrincipals),
		serviceCustody:    cfg.ServiceCustody,
		humanAssertion:    cfg.HumanAssertion,
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

// HumanActorKind is the actor_kind claim stamped on a human session-delegation
// token, mirroring ServiceActorKind for service-principal tokens. A relying
// party uses it to confirm the delegation represents a person.
const HumanActorKind = "human"

// DefaultSessionDelegationTTL bounds a Web Console session-delegation token.
// It is a pure attribution credential, so it is deliberately short: long enough
// for the launched agent to self-attach, not long enough to be a standing grant.
const DefaultSessionDelegationTTL = 10 * time.Minute

// MintSessionDelegation mints a short-lived, session-scoped human-owner
// delegation for the verified human presenting accessToken. The human proof is
// accepted from EITHER an account bearer (Validate, which rejects any non-human
// actor_kind) OR — for a tunnel operator — a Cloudflare Access assertion verified
// against the web-console Access binding (the same assertion web-console and
// agent-manager already trust). The subject is taken from the verified proof,
// never from the request, so a caller cannot name a different operator; a
// service/agent/anonymous caller yields no verified human and is refused. The
// delegation carries actor_kind "human", an audience, a wc_session_id claim, a
// tight expiry, and NO capability scopes: it attributes, it does not empower. An
// empty resource resolves to the realm's default audience (what a relying party
// accepts out of the box); a non-empty resource must be a registered resource
// audience, else the mint is refused.
func (s *Service) MintSessionDelegation(ctx context.Context, accessToken, wcSessionID, resource string, meta RequestMeta) (string, time.Time, error) {
	wcSessionID = strings.TrimSpace(wcSessionID)
	resource = strings.TrimSpace(resource)
	if wcSessionID == "" {
		return "", time.Time{}, InvalidInputError{Msg: "session delegation requires a web console session id"}
	}

	// Resolve the verified human from either proof. Account bearer first; then a
	// Cloudflare Access assertion (the tunnel operator). Neither ⇒ refuse.
	subject, email, realmID, err := s.resolveDelegationHuman(ctx, accessToken)
	if err != nil {
		return "", time.Time{}, err
	}

	aud, err := s.resolveAudience(ctx, realmID, resource)
	if err != nil {
		return "", time.Time{}, err
	}
	ttl := DefaultSessionDelegationTTL
	token, err := s.signer.Sign(authcrypto.TokenInput{
		UserID:      subject,
		Email:       email,
		Audience:    aud,
		ActorKind:   HumanActorKind,
		WCSessionID: wcSessionID,
		ExpiresIn:   ttl,
	})
	if err != nil {
		s.logEvent(ctx, subject, realmID, "session_delegation.mint.refused", meta, false, map[string]any{"reason": err.Error()})
		return "", time.Time{}, err
	}
	expiresAt := s.clock.Now().Add(ttl)
	s.logEvent(ctx, subject, realmID, "session_delegation.minted", meta, true, map[string]any{"wc_session_id": wcSessionID, "audience": aud, "expires_at": expiresAt.UTC().Format(time.RFC3339)})
	return token, expiresAt, nil
}

// resolveDelegationHuman returns the verified operator subject/email/realm for a
// delegation mint. It tries the account bearer first (Validate), then falls back
// to a Cloudflare Access assertion when a human-assertion verifier is configured.
// A Cf-Access human has no scenario-authenticator realm, so it is attributed in
// the default realm (the delegation is a cross-scenario owner label, not an
// account token). Any proof that is missing, invalid, or not a verified human
// yields ErrInvalidCredentials.
func (s *Service) resolveDelegationHuman(ctx context.Context, accessToken string) (subject, email, realmID string, err error) {
	if vt, ok, vErr := s.Validate(ctx, accessToken); vErr != nil {
		return "", "", "", vErr
	} else if ok {
		log.Printf("[phase5-diag] mint: resolved via account bearer human=%s realm=%s", vt.UserID, vt.Realm)
		return vt.UserID, vt.Email, vt.Realm, nil
	}
	if s.humanAssertion != nil {
		sub, mail, ok := s.humanAssertion.VerifyHuman(ctx, accessToken)
		log.Printf("[phase5-diag] mint: account-bearer not a verified human; Cf-Access verify ok=%t subjectPresent=%t", ok, strings.TrimSpace(sub) != "")
		if ok && strings.TrimSpace(sub) != "" {
			return strings.TrimSpace(sub), strings.TrimSpace(mail), realm.DefaultID, nil
		}
	} else {
		log.Printf("[phase5-diag] mint: account-bearer not a verified human and NO Cf-Access verifier configured")
	}
	return "", "", "", ErrInvalidCredentials
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

// RevokeMachineAccount removes the caller's binding, or an administrator's
// selected binding for another account. It is intentionally idempotent.
func (s *Service) RevokeMachineAccount(ctx context.Context, accessToken, machineID, localPrincipal, principalID string, meta RequestMeta) (int, error) {
	vt, err := s.authorizedPrincipal(ctx, accessToken, principalID)
	if err != nil {
		return 0, err
	}
	if s.machineBindings == nil {
		return 0, ErrMachineExchangeRefused
	}
	if principalID != "" && principalID != vt.UserID && !hasRole(vt.Roles, "admin") {
		return 0, ErrInvalidCredentials
	}
	target, err := s.repo.FindByID(ctx, vt.PrincipalID)
	if err != nil || target.RealmID != vt.Realm {
		return 0, ErrInvalidCredentials
	}
	count, err := s.machineBindings.RevokeMachineBinding(ctx, strings.TrimSpace(machineID), strings.TrimSpace(localPrincipal), vt.PrincipalID)
	if err != nil {
		return 0, err
	}
	s.logEvent(ctx, vt.UserID, vt.Realm, "machine.binding.revoked", meta, true, map[string]any{
		"machine_id": machineID, "local_principal": localPrincipal, "target_user_id": vt.PrincipalID, "revoked_count": count,
	})
	return count, nil
}

// ExchangeMachinePrincipal issues a normal access/refresh pair after the
// socket listener has authenticated the OS principal and binding resolution
// returns exactly one default account.
func (s *Service) ExchangeMachinePrincipal(ctx context.Context, machineID, localPrincipal, resource string, meta RequestMeta) (AuthResult, error) {
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
	aud, err := s.resolveAudience(ctx, acc.RealmID, resource)
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
	aud, err := s.resolveAudience(ctx, realmID, p.Resource)
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
	roles := append([]string(nil), p.Roles...)
	firstAccount := false
	if counter, ok := s.repo.(AccountCounter); ok {
		count, countErr := counter.CountAccounts(ctx, realmID)
		if countErr != nil {
			return AuthResult{}, countErr
		}
		firstAccount = count == 0
		if firstAccount && !hasRole(roles, "admin") {
			roles = append(roles, "admin")
		}
	}
	acc, err := s.repo.Create(ctx, CreateInput{
		RealmID: realmID, Email: email, Username: strings.TrimSpace(p.Username),
		PasswordHash: hash, Roles: roles,
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
	if s.mfa != nil {
		required, err := s.mfa.Required(ctx, realmID)
		if err != nil {
			return AuthResult{}, err
		}
		if required {
			s.logEvent(ctx, acc.ID, realmID, "user.registered.mfa_required", meta, false, map[string]any{"reason": "enrollment_required_before_session"})
			return AuthResult{}, ErrMFARequired
		}
	}
	res, err := s.issueTokens(ctx, acc, aud, meta)
	if err != nil {
		return AuthResult{}, err
	}
	s.logEvent(ctx, acc.ID, realmID, "user.registered", meta, true, nil)
	if firstAccount {
		s.logEvent(ctx, acc.ID, realmID, "admin.bootstrap", meta, true, map[string]any{"reason": "first_account_in_realm"})
	}
	return res, nil
}

// Login verifies credentials and issues a fresh token pair, enforcing lockout.
func (s *Service) Login(ctx context.Context, p LoginParams, meta RequestMeta) (AuthResult, error) {
	email := strings.TrimSpace(p.Email)
	realmID := resolveRealm(p.Realm)
	aud, err := s.resolveAudience(ctx, realmID, p.Resource)
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

	if s.mfa != nil {
		required, err := s.mfa.Required(ctx, realmID)
		if err != nil {
			return AuthResult{}, err
		}
		enrolled, err := s.mfa.Enrolled(ctx, acc.ID)
		if err != nil {
			return AuthResult{}, err
		}
		if required && !enrolled {
			s.logEvent(ctx, acc.ID, realmID, "user.login.mfa_required", meta, false, map[string]any{"reason": "mfa_not_enrolled"})
			return AuthResult{}, ErrMFARequired
		}
		if enrolled {
			challenge := strings.TrimSpace(p.MFAChallenge)
			if challenge == "" {
				challenge, _, err = s.mfa.StartChallenge(ctx, acc.ID)
				if err != nil {
					return AuthResult{}, err
				}
			}
			if strings.TrimSpace(p.TOTPCode) == "" && strings.TrimSpace(p.RecoveryCode) == "" {
				return AuthResult{Account: acc, MFARequired: true, MFAChallenge: challenge}, nil
			}
			valid, verifyErr := s.mfa.VerifyLogin(ctx, acc.ID, challenge, p.TOTPCode, p.RecoveryCode)
			if verifyErr != nil || !valid {
				s.logEvent(ctx, acc.ID, realmID, "user.login.mfa_failed", meta, false, map[string]any{"reason": "invalid_code"})
				return AuthResult{}, ErrMFACode
			}
		}
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
	claims, err := s.signer.ValidateAny(accessToken, aud, s.acceptedAudienceValues()...)
	if err != nil {
		return ValidatedToken{}, false, nil
	}
	// Account operations accept human account tokens only. A service-principal
	// token is valid at its relying party, never as account authority here.
	if strings.TrimSpace(claims.ActorKind) != "" {
		return ValidatedToken{}, false, nil
	}
	authVersion, err := s.sessions.CurrentAuthVersion(ctx, claims.UserID)
	if err != nil {
		return ValidatedToken{}, false, err
	}
	if claims.AuthVersion != authVersion {
		return ValidatedToken{}, false, nil
	}
	if claims.SessionID != "" {
		session, found, err := s.sessions.FindSession(ctx, claims.SessionID)
		if err != nil {
			return ValidatedToken{}, false, err
		}
		if !found || session.UserID != claims.UserID || (!session.ExpiresAt.IsZero() && !session.ExpiresAt.After(s.clock.Now())) {
			return ValidatedToken{}, false, nil
		}
	}
	var exp time.Time
	if claims.ExpiresAt != nil {
		exp = claims.ExpiresAt.Time
	}
	audience := ""
	if len(claims.Audience) > 0 {
		audience = claims.Audience[0]
	}
	return ValidatedToken{
		UserID: claims.UserID, Email: claims.Email, Roles: claims.Roles,
		Scopes: nonNilStrings(claims.Scopes),
		Realm:  realm.DefaultID, ExpiresAt: exp, Audience: audience,
		SessionID: claims.SessionID, AuthVersion: claims.AuthVersion,
	}, true, nil
}

func (s *Service) issueTokens(ctx context.Context, acc Account, aud string, meta RequestMeta) (AuthResult, error) {
	scopes, err := s.scopes(ctx, acc)
	if err != nil {
		return AuthResult{}, err
	}
	acc.Scopes = scopes
	refresh, err := s.sessions.IssueRefresh(ctx, acc.ID, aud)
	if err != nil {
		return AuthResult{}, err
	}
	familyID, _, err := s.sessions.RefreshFamilyForToken(ctx, refresh)
	if err != nil {
		return AuthResult{}, err
	}
	sessionID, err := s.sessions.StoreSessionForFamily(ctx, acc.ID, meta.IP, meta.UserAgent, familyID)
	if err != nil {
		return AuthResult{}, err
	}
	authVersion, err := s.sessions.CurrentAuthVersion(ctx, acc.ID)
	if err != nil {
		return AuthResult{}, err
	}
	access, err := s.signer.Sign(authcrypto.TokenInput{
		UserID: acc.ID, Email: acc.Email, Roles: acc.Roles, Scopes: scopes, Audience: aud,
		SessionID: sessionID, AuthVersion: authVersion,
	})
	if err != nil {
		return AuthResult{}, err
	}
	return AuthResult{
		Account: acc, AccessToken: access, RefreshToken: refresh,
		AccessExpiresAt: s.clock.Now().Add(s.signer.Expiry()), Audience: aud,
	}, nil
}

func (s *Service) resolveAudience(ctx context.Context, realmID, resource string) (string, error) {
	canonical, err := s.repo.RealmAudience(ctx, realmID)
	if err != nil {
		return "", err
	}
	resource = strings.TrimSpace(resource)
	if resource == "" {
		return canonical, nil
	}
	aud, ok := s.resourceAudiences[resource]
	if !ok || strings.TrimSpace(aud) == "" {
		return "", InvalidInputError{Msg: "unknown authentication resource"}
	}
	return strings.TrimSpace(aud), nil
}

func (s *Service) acceptedAudienceValues() []string {
	values := append([]string(nil), s.acceptedAudiences...)
	for _, aud := range s.resourceAudiences {
		if strings.TrimSpace(aud) != "" {
			values = append(values, strings.TrimSpace(aud))
		}
	}
	return values
}

func copyStringMap(values map[string]string) map[string]string {
	out := make(map[string]string, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key != "" && value != "" {
			out[key] = value
		}
	}
	return out
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
