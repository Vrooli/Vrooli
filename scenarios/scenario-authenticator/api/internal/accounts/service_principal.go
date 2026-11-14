package accounts

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"scenario-authenticator/internal/authcrypto"
	"scenario-authenticator/internal/realm"
)

// ServiceActorKind is the actor_kind claim stamped on service-principal tokens.
const ServiceActorKind = "service"

// serviceCredentialPrefix versions the credential format. The principal name is
// carried in clear so the exchange can locate its custody entry; the secret
// part is 256 random bits.
const serviceCredentialPrefix = "vsp1."

var (
	// ErrServicePrincipalUnknown — no operator profile names this principal.
	ErrServicePrincipalUnknown = errors.New("service principal is not configured")
	// ErrServicePrincipalForbidden — the issuer is not a verified administrator.
	ErrServicePrincipalForbidden = errors.New("service principal issuance requires a verified authorized issuer")
	// ErrServiceCredentialRejected — the presented credential is not the
	// currently issued one (or the principal is unconfigured/revoked).
	ErrServiceCredentialRejected = errors.New("service credential rejected")
	// ErrServiceCustodyUnavailable — the credential authority cannot be used.
	ErrServiceCustodyUnavailable = errors.New("service credential custody unavailable")
)

var servicePrincipalName = regexp.MustCompile(`^[a-z][a-z0-9-]{1,62}$`)

// ServicePrincipalProfile is one operator-owned service principal. The
// audience is resolved from the registered resource; scopes are the complete,
// fixed capability list carried by every token the principal receives.
// Issuers names the verified human accounts the operator authorizes to mint
// it; a realm administrator may also issue.
type ServicePrincipalProfile struct {
	Name     string   `json:"name"`
	Resource string   `json:"resource"`
	Scopes   []string `json:"scopes"`
	Issuers  []string `json:"issuers,omitempty"`
}

// ServiceCredentialCustodian stores issued credentials in the credential
// authority. Deleting the entry revokes the principal: the exchange compares
// against the stored value and refuses when it is absent.
type ServiceCredentialCustodian interface {
	Put(name, resource, credential string) error
	Resolve(name, resource string) (string, error)
	Address(name, resource string) (identity, field string)
}

// ParseServicePrincipalProfiles decodes the operator-owned JSON profile list.
// An empty value configures no service principals. Malformed or duplicate
// profiles fail configuration rather than being silently skipped.
func ParseServicePrincipalProfiles(raw string) ([]ServicePrincipalProfile, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var profiles []ServicePrincipalProfile
	if err := json.Unmarshal([]byte(raw), &profiles); err != nil {
		return nil, fmt.Errorf("invalid service principal profiles: %w", err)
	}
	seen := map[string]bool{}
	for i, p := range profiles {
		p.Name = strings.TrimSpace(p.Name)
		p.Resource = strings.TrimSpace(p.Resource)
		if !servicePrincipalName.MatchString(p.Name) || p.Resource == "" || seen[p.Name] {
			return nil, fmt.Errorf("invalid service principal profile %d", i)
		}
		scopes := make([]string, 0, len(p.Scopes))
		for _, scope := range p.Scopes {
			scope = strings.TrimSpace(scope)
			if scope == "" || strings.Contains(scope, "*") {
				return nil, fmt.Errorf("invalid scope in service principal profile %q", p.Name)
			}
			scopes = append(scopes, scope)
		}
		p.Scopes = scopes
		issuers := make([]string, 0, len(p.Issuers))
		for _, issuer := range p.Issuers {
			if issuer = strings.TrimSpace(issuer); issuer == "" {
				return nil, fmt.Errorf("invalid issuer in service principal profile %q", p.Name)
			}
			issuers = append(issuers, issuer)
		}
		p.Issuers = issuers
		seen[p.Name] = true
		profiles[i] = p
	}
	return profiles, nil
}

func indexServicePrincipals(profiles []ServicePrincipalProfile) map[string]ServicePrincipalProfile {
	out := make(map[string]ServicePrincipalProfile, len(profiles))
	for _, p := range profiles {
		out[p.Name] = ServicePrincipalProfile{Name: p.Name, Resource: p.Resource, Scopes: append([]string{}, p.Scopes...), Issuers: append([]string{}, p.Issuers...)}
	}
	return out
}

// ServicePrincipalIssue describes an issued principal. It never carries the
// credential.
type ServicePrincipalIssue struct {
	Name               string
	Audience           string
	Scopes             []string
	CredentialIdentity string
	CredentialField    string
	IssuedAt           time.Time
}

// ServiceAccess is a short-lived service access token.
type ServiceAccess struct {
	AccessToken string
	ExpiresAt   time.Time
	Audience    string
	Subject     string
	Scopes      []string
}

// IssueServicePrincipal mints (or rotates) the credential for a configured
// principal. Only a verified human named as a profile issuer, or a realm
// administrator, may issue; the profile, not
// the request, fixes the audience and scopes. Rotation overwrites the stored
// credential, so the previous credential stops exchanging immediately.
func (s *Service) IssueServicePrincipal(ctx context.Context, accessToken, name string, meta RequestMeta) (ServicePrincipalIssue, error) {
	vt, ok, err := s.Validate(ctx, accessToken)
	if err != nil {
		return ServicePrincipalIssue{}, err
	}
	if !ok {
		s.logEvent(ctx, "", realm.DefaultID, "service_principal.issue.refused", meta, false, map[string]any{"name": name, "reason": "unverified"})
		return ServicePrincipalIssue{}, ErrServicePrincipalForbidden
	}
	admin := hasRole(vt.Roles, "admin")
	profile, configured := s.servicePrincipals[strings.TrimSpace(name)]
	if !configured {
		s.logEvent(ctx, vt.UserID, vt.Realm, "service_principal.issue.refused", meta, false, map[string]any{"name": name, "reason": "unconfigured"})
		if !admin {
			return ServicePrincipalIssue{}, ErrServicePrincipalForbidden
		}
		return ServicePrincipalIssue{}, ErrServicePrincipalUnknown
	}
	if !admin && !slices.Contains(profile.Issuers, vt.UserID) {
		s.logEvent(ctx, vt.UserID, vt.Realm, "service_principal.issue.refused", meta, false, map[string]any{"name": name, "reason": "not_authorized_issuer"})
		return ServicePrincipalIssue{}, ErrServicePrincipalForbidden
	}
	aud, err := s.resolveAudience(ctx, realm.DefaultID, profile.Resource)
	if err != nil {
		return ServicePrincipalIssue{}, err
	}
	if s.serviceCustody == nil {
		return ServicePrincipalIssue{}, ErrServiceCustodyUnavailable
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return ServicePrincipalIssue{}, err
	}
	credential := serviceCredentialPrefix + profile.Name + "." + base64.RawURLEncoding.EncodeToString(secret)
	if err := s.serviceCustody.Put(profile.Name, profile.Resource, credential); err != nil {
		s.logEvent(ctx, vt.UserID, vt.Realm, "service_principal.issue.refused", meta, false, map[string]any{"name": profile.Name, "reason": "custody_unavailable"})
		return ServicePrincipalIssue{}, fmt.Errorf("%w: %v", ErrServiceCustodyUnavailable, err)
	}
	identity, field := s.serviceCustody.Address(profile.Name, profile.Resource)
	issuedAt := s.clock.Now().UTC()
	s.logEvent(ctx, vt.UserID, vt.Realm, "service_principal.issued", meta, true, map[string]any{
		"name": profile.Name, "audience": aud, "scopes": profile.Scopes, "credential_identity": identity, "credential_field": field,
	})
	return ServicePrincipalIssue{
		Name: profile.Name, Audience: aud, Scopes: append([]string{}, profile.Scopes...),
		CredentialIdentity: identity, CredentialField: field, IssuedAt: issuedAt,
	}, nil
}

// ExchangeServiceCredential returns a short-lived service token when the
// presented credential equals the currently stored one for a still-configured
// principal. It issues no refresh token and no session.
func (s *Service) ExchangeServiceCredential(ctx context.Context, credential string, meta RequestMeta) (ServiceAccess, error) {
	credential = strings.TrimSpace(credential)
	rest, ok := strings.CutPrefix(credential, serviceCredentialPrefix)
	name, _, found := strings.Cut(rest, ".")
	if !ok || !found {
		s.logEvent(ctx, "", realm.DefaultID, "service_principal.exchange.refused", meta, false, map[string]any{"reason": "malformed"})
		return ServiceAccess{}, ErrServiceCredentialRejected
	}
	profile, configured := s.servicePrincipals[name]
	if !configured || s.serviceCustody == nil {
		s.logEvent(ctx, "", realm.DefaultID, "service_principal.exchange.refused", meta, false, map[string]any{"name": name, "reason": "unconfigured"})
		return ServiceAccess{}, ErrServiceCredentialRejected
	}
	stored, err := s.serviceCustody.Resolve(profile.Name, profile.Resource)
	if err != nil || stored == "" || subtle.ConstantTimeCompare([]byte(stored), []byte(credential)) != 1 {
		s.logEvent(ctx, "", realm.DefaultID, "service_principal.exchange.refused", meta, false, map[string]any{"name": name, "reason": "credential_mismatch_or_revoked"})
		return ServiceAccess{}, ErrServiceCredentialRejected
	}
	aud, err := s.resolveAudience(ctx, realm.DefaultID, profile.Resource)
	if err != nil {
		return ServiceAccess{}, err
	}
	token, err := s.signer.Sign(authcrypto.TokenInput{
		UserID: profile.Name, Scopes: profile.Scopes, Audience: aud, ActorKind: ServiceActorKind,
	})
	if err != nil {
		return ServiceAccess{}, err
	}
	s.logEvent(ctx, "", realm.DefaultID, "service_principal.exchange.accepted", meta, true, map[string]any{"name": profile.Name, "audience": aud})
	return ServiceAccess{
		AccessToken: token, ExpiresAt: s.clock.Now().Add(s.signer.Expiry()), Audience: aud,
		Subject: profile.Name, Scopes: append([]string{}, profile.Scopes...),
	}, nil
}
