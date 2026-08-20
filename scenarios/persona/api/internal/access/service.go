package access

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/cli-core/cliutil"

	"persona/internal/journal"
	"persona/internal/personas"
)

var (
	ErrMissingPersona       = errors.New("persona_id is required")
	ErrAuthorityUnreachable = errors.New("identity authority is unreachable")
	ErrInvalidIdentity      = errors.New("identity token is invalid or expired")
	ErrPersonaBinding       = errors.New("identity token is not bound to this persona")
	ErrScopeMissing         = errors.New("identity token lacks the persona act-as scope")
	ErrGrantMissing         = errors.New("verified subject has no grant for this persona")
	ErrProposeOnly          = errors.New("verified subject may propose but may not act")
	ErrAgentACLMutation     = errors.New("persona ACL mutations are operator-only")
	ErrGrantNotFound        = errors.New("persona grant not found")
	ErrAttestationExpiry    = errors.New("attestation expiry must be in the future and no later than the run token")
	ErrAttestationSigner    = errors.New("attestation signer is not configured")
)

type GrantLevel string

const (
	GrantAct     GrantLevel = "act"
	GrantPropose GrantLevel = "propose"
)

type Grant struct {
	ID           string
	PersonaID    string
	HumanSubject string
	Level        GrantLevel
	Source       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Claims struct {
	RunID     string
	Subject   string
	Scopes    []string
	Meta      map[string]string
	ExpiresAt time.Time
}

type PersonaResolver interface {
	Get(context.Context, string) (personas.Persona, error)
}

type Verifier interface {
	Verify(context.Context, string) (*Claims, error)
}

type LiveVerifier struct{}

func (LiveVerifier) Verify(_ context.Context, token string) (*Claims, error) {
	result, err := (cliutil.IdentityEnv{Token: strings.TrimSpace(token)}).VerifyIdentity()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAuthorityUnreachable, err)
	}
	if result == nil || !result.Valid || result.Claims == nil {
		return nil, ErrInvalidIdentity
	}
	claims := &Claims{RunID: result.Claims.RunID, Subject: result.Claims.Subject, Scopes: append([]string(nil), result.Claims.Scopes...), Meta: result.Claims.Meta}
	if result.Claims.ExpiresAt > 0 {
		claims.ExpiresAt = time.Unix(result.Claims.ExpiresAt, 0).UTC()
	}
	return claims, nil
}

type Service interface {
	ActAs(context.Context, string, string, string) (ActSession, error)
	AuthorizeProposal(context.Context, string, string) (string, string, error)
	ResolvePersona(context.Context, string, string, []string) (Resolution, error)
	CreateGrant(context.Context, GrantInput) (Grant, error)
	ChangeGrant(context.Context, GrantChangeInput) (Grant, error)
	ListGrants(context.Context, string) ([]Grant, error)
	RemoveGrant(context.Context, string) error
	IssueAttestation(context.Context, string, string, string, time.Time) (Attestation, error)
}

type GrantInput struct {
	PersonaID    string
	HumanSubject string
	Level        GrantLevel
	Source       string
}

type GrantChangeInput struct {
	GrantID string
	Level   GrantLevel
	Source  string
}

type ActSession struct {
	PersonaID        string
	RunID            string
	AccountSubject   string
	AuthorisingHuman string
	GrantedAt        time.Time
}

type Resolution struct {
	PersonaID       string
	Kind            personas.Kind
	DisplayName     string
	LegalSubjectID  string
	ControlledEmail string
	AddressIDs      []string
	ReturnedFields  []string
}

type Attestation struct {
	Issuer         string
	Audience       string
	LegalPerson    string
	PersonaID      string
	AccountSubject string
	RunID          string
	IssuedAt       time.Time
	ExpiresAt      time.Time
	ClaimFormat    string
	Signature      []byte
	KeyID          string
}

type service struct {
	repo     Repository
	personas PersonaResolver
	journal  journal.Service
	verifier Verifier
	clock    schedule.Clock
	secret   []byte
	keyID    string
}

type ServiceOptions struct {
	Clock  schedule.Clock
	Secret []byte
	KeyID  string
}

func NewService(repo Repository, personaResolver PersonaResolver, actionJournal journal.Service, verifier Verifier, opts ServiceOptions) Service {
	if opts.Clock == nil {
		opts.Clock = schedule.System()
	}
	return &service{repo: repo, personas: personaResolver, journal: actionJournal, verifier: verifier, clock: opts.Clock, secret: append([]byte(nil), opts.Secret...), keyID: opts.KeyID}
}

var _ Service = (*service)(nil)

func (s *service) ActAs(ctx context.Context, personaID, token, action string) (ActSession, error) {
	if strings.TrimSpace(personaID) == "" {
		return ActSession{}, ErrMissingPersona
	}
	claims, err := s.verify(ctx, token)
	if err != nil {
		constraint := "authority_unreachable"
		if errors.Is(err, ErrInvalidIdentity) {
			constraint = "invalid_identity"
		}
		s.record(ctx, personaID, "act_as_refused", claimsOrEmpty(claims), "refused", constraint, map[string]string{"action": action})
		return ActSession{}, err
	}
	if bound := strings.TrimSpace(claims.Meta["persona_id"]); bound != "" && bound != personaID {
		s.record(ctx, personaID, "act_as_refused", claims, "refused", "persona_binding_mismatch", nil)
		return ActSession{}, ErrPersonaBinding
	}
	if !hasScope(claims.Scopes, "persona.act-as:"+personaID) {
		s.record(ctx, personaID, "act_as_refused", claims, "refused", "scope_missing", nil)
		return ActSession{}, ErrScopeMissing
	}
	grants, err := s.repo.ListGrants(ctx, personaID)
	if err != nil {
		return ActSession{}, fmt.Errorf("list persona grants: %w", err)
	}
	grant := findGrant(grants, claims.Subject)
	if grant == nil {
		s.record(ctx, personaID, "act_as_refused", claims, "refused", "grant_missing", nil)
		return ActSession{}, ErrGrantMissing
	}
	if grant.Level != GrantAct {
		s.record(ctx, personaID, "act_as_refused", claims, "refused", "propose_only", nil)
		return ActSession{}, ErrProposeOnly
	}
	if _, err := s.personas.Get(ctx, personaID); err != nil {
		return ActSession{}, fmt.Errorf("resolve persona: %w", err)
	}
	at := s.clock.Now().UTC()
	s.record(ctx, personaID, "act_as_granted", claims, "granted", "", map[string]string{"action": action, "grant_id": grant.ID})
	return ActSession{PersonaID: personaID, RunID: claims.RunID, AccountSubject: claims.Subject, AuthorisingHuman: grant.HumanSubject, GrantedAt: at}, nil
}

// AuthorizeProposal verifies the caller and permits either ACL level to open a
// human handoff. It deliberately does not grant any act-as capability.
func (s *service) AuthorizeProposal(ctx context.Context, personaID, token string) (string, string, error) {
	if strings.TrimSpace(personaID) == "" {
		return "", "", ErrMissingPersona
	}
	claims, err := s.verify(ctx, token)
	if err != nil {
		s.record(ctx, personaID, "handoff_proposal_refused", claimsOrEmpty(claims), "refused", "identity_unverified", nil)
		return "", "", err
	}
	if bound := strings.TrimSpace(claims.Meta["persona_id"]); bound != "" && bound != personaID {
		s.record(ctx, personaID, "handoff_proposal_refused", claims, "refused", "persona_binding_mismatch", nil)
		return "", "", ErrPersonaBinding
	}
	if !hasScope(claims.Scopes, "persona.propose:"+personaID) {
		s.record(ctx, personaID, "handoff_proposal_refused", claims, "refused", "proposal_scope_missing", nil)
		return "", "", ErrScopeMissing
	}
	grants, err := s.repo.ListGrants(ctx, personaID)
	if err != nil {
		return "", "", fmt.Errorf("list persona grants: %w", err)
	}
	grant := findGrant(grants, claims.Subject)
	if grant == nil {
		s.record(ctx, personaID, "handoff_proposal_refused", claims, "refused", "grant_missing", nil)
		return "", "", ErrGrantMissing
	}
	if grant.Level != GrantAct && grant.Level != GrantPropose {
		s.record(ctx, personaID, "handoff_proposal_refused", claims, "refused", "grant_level_invalid", nil)
		return "", "", ErrProposeOnly
	}
	if _, err := s.personas.Get(ctx, personaID); err != nil {
		return "", "", fmt.Errorf("resolve persona: %w", err)
	}
	s.record(ctx, personaID, "handoff_proposal_granted", claims, "granted", "", map[string]string{"grant_id": grant.ID})
	return claims.RunID, grant.HumanSubject, nil
}

func (s *service) ResolvePersona(ctx context.Context, personaID, token string, fields []string) (Resolution, error) {
	if strings.TrimSpace(personaID) == "" {
		return Resolution{}, ErrMissingPersona
	}
	claims, err := s.verify(ctx, token)
	if err != nil {
		return Resolution{}, err
	}
	p, err := s.personas.Get(ctx, personaID)
	if err != nil {
		return Resolution{}, err
	}
	requested := make(map[string]bool, len(fields))
	for _, field := range fields {
		requested[strings.TrimSpace(field)] = true
	}
	resolution := Resolution{PersonaID: p.ID, Kind: p.Kind, ReturnedFields: []string{"persona_id", "kind"}}
	if requested["display_name"] && hasScope(claims.Scopes, "persona.resolve.display") {
		resolution.DisplayName = p.DisplayName
		resolution.ReturnedFields = append(resolution.ReturnedFields, "display_name")
	}
	if requested["legal_subject_id"] && hasScope(claims.Scopes, "persona.resolve.legal") {
		resolution.LegalSubjectID = p.LegalBasis.SubjectID
		resolution.ReturnedFields = append(resolution.ReturnedFields, "legal_subject_id")
	}
	sort.Strings(resolution.ReturnedFields)
	s.record(ctx, personaID, "persona_resolved", claims, "granted", "", map[string]string{"fields": strings.Join(resolution.ReturnedFields, ",")})
	return resolution, nil
}

func (s *service) CreateGrant(ctx context.Context, in GrantInput) (Grant, error) {
	if strings.TrimSpace(in.PersonaID) == "" || strings.TrimSpace(in.HumanSubject) == "" {
		return Grant{}, ErrMissingPersona
	}
	if in.Level != GrantAct && in.Level != GrantPropose {
		return Grant{}, errors.New("grant level must be act or propose")
	}
	if _, err := s.personas.Get(ctx, in.PersonaID); err != nil {
		return Grant{}, err
	}
	grant, err := s.repo.CreateGrant(ctx, Grant{PersonaID: in.PersonaID, HumanSubject: in.HumanSubject, Level: in.Level, Source: sourceOrLocal(in.Source)})
	if err == nil {
		s.recordGrantChange(ctx, grant, "grant_created", map[string]string{"grant_id": grant.ID, "level": string(grant.Level), "source": grant.Source})
	}
	return grant, err
}

func (s *service) ChangeGrant(ctx context.Context, in GrantChangeInput) (Grant, error) {
	if strings.TrimSpace(in.GrantID) == "" {
		return Grant{}, ErrGrantNotFound
	}
	if in.Level != GrantAct && in.Level != GrantPropose {
		return Grant{}, errors.New("grant level must be act or propose")
	}
	grant, err := s.repo.GetGrant(ctx, in.GrantID)
	if err != nil {
		return Grant{}, err
	}
	previousLevel := grant.Level
	grant.Level = in.Level
	if strings.TrimSpace(in.Source) != "" {
		grant.Source = in.Source
	}
	grant, err = s.repo.UpdateGrant(ctx, grant)
	if err == nil {
		s.recordGrantChange(ctx, grant, "grant_changed", map[string]string{"grant_id": grant.ID, "previous_level": string(previousLevel), "level": string(grant.Level), "source": grant.Source})
	}
	return grant, err
}

func (s *service) ListGrants(ctx context.Context, personaID string) ([]Grant, error) {
	if strings.TrimSpace(personaID) == "" {
		return nil, ErrMissingPersona
	}
	return s.repo.ListGrants(ctx, personaID)
}

func (s *service) RemoveGrant(ctx context.Context, grantID string) error {
	grant, err := s.repo.GetGrant(ctx, grantID)
	if err != nil {
		return err
	}
	if err := s.repo.RemoveGrant(ctx, grantID); err != nil {
		return err
	}
	s.recordGrantChange(ctx, grant, "grant_removed", map[string]string{"grant_id": grantID})
	return nil
}

func (s *service) IssueAttestation(ctx context.Context, personaID, token, audience string, requestedExpiry time.Time) (Attestation, error) {
	if len(s.secret) == 0 {
		return Attestation{}, ErrAttestationSigner
	}
	claims, err := s.verify(ctx, token)
	if err != nil {
		return Attestation{}, err
	}
	if _, err := s.ActAs(ctx, personaID, token, "issue_identity_attestation"); err != nil {
		return Attestation{}, err
	}
	p, err := s.personas.Get(ctx, personaID)
	if err != nil {
		return Attestation{}, err
	}
	now := s.clock.Now().UTC()
	expires := requestedExpiry.UTC()
	if expires.IsZero() || !expires.After(now) || claims.ExpiresAt.IsZero() || expires.After(claims.ExpiresAt) {
		return Attestation{}, ErrAttestationExpiry
	}
	a := Attestation{Issuer: "vrooli.persona", Audience: audience, LegalPerson: p.LegalBasis.SubjectID, PersonaID: personaID, AccountSubject: claims.Subject, RunID: claims.RunID, IssuedAt: now, ExpiresAt: expires, ClaimFormat: "kya-os/v1", KeyID: s.keyID}
	payload := fmt.Sprintf("%s|%s|%s|%s|%s|%d|%d|%s", a.Issuer, a.Audience, a.LegalPerson, a.PersonaID, a.AccountSubject, a.IssuedAt.Unix(), a.ExpiresAt.Unix(), a.ClaimFormat)
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(payload))
	a.Signature = mac.Sum(nil)
	s.record(ctx, personaID, "attestation_issued", claims, "granted", "", map[string]string{"audience": audience, "claim_format": a.ClaimFormat})
	return a, nil
}

func (s *service) verify(ctx context.Context, token string) (*Claims, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrInvalidIdentity
	}
	claims, err := s.verifier.Verify(ctx, token)
	if err != nil {
		if errors.Is(err, ErrInvalidIdentity) {
			return nil, ErrInvalidIdentity
		}
		return nil, fmt.Errorf("%w: %v", ErrAuthorityUnreachable, err)
	}
	if claims == nil || strings.TrimSpace(claims.RunID) == "" || strings.TrimSpace(claims.Subject) == "" {
		return nil, ErrInvalidIdentity
	}
	if !claims.ExpiresAt.IsZero() && !claims.ExpiresAt.After(s.clock.Now()) {
		return nil, ErrInvalidIdentity
	}
	return claims, nil
}

func (s *service) record(ctx context.Context, personaID string, verb string, claims *Claims, outcome, constraint string, details map[string]string) {
	if s.journal == nil {
		return
	}
	runID, human := "", ""
	if claims != nil {
		runID, human = claims.RunID, claims.Subject
	}
	_, _ = s.journal.Append(ctx, journal.Entry{PersonaID: personaID, Actor: "agent", Verb: verb, RunID: runID, AuthorisingHuman: human, Outcome: outcome, Constraint: constraint, Details: details})
}

func (s *service) recordGrantChange(ctx context.Context, grant Grant, verb string, details map[string]string) {
	if s.journal == nil {
		return
	}
	_, _ = s.journal.Append(ctx, journal.Entry{PersonaID: grant.PersonaID, Actor: "operator", Verb: verb, Outcome: "granted", AuthorisingHuman: grant.HumanSubject, Details: details})
}

func claimsOrEmpty(claims *Claims) *Claims {
	if claims != nil {
		return claims
	}
	return &Claims{}
}

func hasScope(scopes []string, required string) bool {
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if scope == "*" || scope == required || (strings.HasSuffix(scope, "*") && strings.HasPrefix(required, strings.TrimSuffix(scope, "*"))) {
			return true
		}
	}
	return false
}

func findGrant(grants []Grant, subject string) *Grant {
	for i := range grants {
		if grants[i].HumanSubject == subject {
			return &grants[i]
		}
	}
	return nil
}

func sourceOrLocal(source string) string {
	if strings.TrimSpace(source) == "" {
		return "local_acl"
	}
	return source
}
