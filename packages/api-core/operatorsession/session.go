// Package operatorsession is the shared contract for owner-facing clients.
// It describes enrollment and diagnosis without owning a transport or storing
// bearer material. Concrete clients (Bridge CLI/UI and future scenarios) use
// this vocabulary instead of inventing provider-specific error strings.
package operatorsession

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Mode string

const (
	ModePersonal Mode = "personal"
	ModeShared   Mode = "shared"
	ModeHosted   Mode = "hosted"
)

type Enrollment struct {
	OperatorID       string    `json:"operator_id"`
	IdentityProvider string    `json:"identity_provider"`
	Mode             Mode      `json:"session_mode"`
	Reference        string    `json:"enrollment_reference"`
	EnrolledAt       time.Time `json:"enrolled_at"`
	ScopeCeiling     []string  `json:"scope_ceiling,omitempty"`
}

// LocalSession is the compact, signed credential minted from an enrollment.
// The private key never leaves the enrolled client. The reference is opaque
// and lets the Bridge revoke an enrollment without putting bearer material in
// operator-state.json.
type LocalSession struct {
	EnrollmentReference string   `json:"enrollment_reference"`
	OperatorID          string   `json:"operator_id"`
	Scopes              []string `json:"scopes,omitempty"`
	IssuedAt            int64    `json:"iat"`
	ExpiresAt           int64    `json:"exp"`
}

var (
	ErrInvalidLocalSession = errors.New("operator session: invalid local session")
	ErrLocalSessionExpired = errors.New("operator session: local session expired")
)

const (
	LocalSessionScheme            = "LocalSession"
	LocalSessionTTL               = 15 * time.Minute
	IdentityProviderAuthenticator = "scenario-authenticator"
)

// GenerateKey creates the client key used by one enrollment.
func GenerateKey() (ed25519.PrivateKey, error) {
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate operator session key: %w", err)
	}
	return private, nil
}

// PublicKey returns a defensive copy of the key's public half.
func PublicKey(private ed25519.PrivateKey) (ed25519.PublicKey, error) {
	if len(private) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("operator session: private key must be %d bytes", ed25519.PrivateKeySize)
	}
	return append(ed25519.PublicKey(nil), private[ed25519.SeedSize:]...), nil
}

// Mint creates a short-lived local credential. It is intentionally not a JWT:
// it is an internal Bridge contract whose verifier is pinned to the
// enrollment record, not a general-purpose bearer token for other services.
func Mint(private ed25519.PrivateKey, enrollmentReference, operatorID string, scopes []string, now time.Time, ttl time.Duration) (string, error) {
	if len(private) != ed25519.PrivateKeySize {
		return "", fmt.Errorf("operator session: private key must be %d bytes", ed25519.PrivateKeySize)
	}
	if strings.TrimSpace(enrollmentReference) == "" || strings.TrimSpace(operatorID) == "" {
		return "", errors.New("operator session: enrollment reference and operator id are required")
	}
	if now.IsZero() {
		now = time.Now()
	}
	if ttl <= 0 {
		ttl = LocalSessionTTL
	}
	claims := LocalSession{EnrollmentReference: strings.TrimSpace(enrollmentReference), OperatorID: strings.TrimSpace(operatorID), Scopes: append([]string(nil), scopes...), IssuedAt: now.Unix(), ExpiresAt: now.Add(ttl).Unix()}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("encode operator session: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	signature := ed25519.Sign(private, []byte(encoded))
	return "OS1." + encoded + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

// Verify checks a locally minted credential against the enrolled public key.
// Caller policy checks the returned scope ceiling; this function only proves
// possession, identity binding, and freshness.
func Verify(public ed25519.PublicKey, token string, now time.Time) (LocalSession, error) {
	if len(public) != ed25519.PublicKeySize {
		return LocalSession{}, ErrInvalidLocalSession
	}
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 || parts[0] != "OS1" {
		return LocalSession{}, ErrInvalidLocalSession
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !ed25519.Verify(public, []byte(parts[1]), signature) {
		return LocalSession{}, ErrInvalidLocalSession
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return LocalSession{}, ErrInvalidLocalSession
	}
	var claims LocalSession
	if err := json.Unmarshal(payload, &claims); err != nil || strings.TrimSpace(claims.EnrollmentReference) == "" || strings.TrimSpace(claims.OperatorID) == "" || claims.IssuedAt <= 0 || claims.ExpiresAt <= claims.IssuedAt {
		return LocalSession{}, ErrInvalidLocalSession
	}
	if now.IsZero() {
		now = time.Now()
	}
	if now.Unix() >= claims.ExpiresAt {
		return LocalSession{}, ErrLocalSessionExpired
	}
	return claims, nil
}

// ContainsAll reports whether requested is contained in ceiling. Empty
// requested scopes are allowed and mean the caller asked for no extra power.
func ContainsAll(ceiling, requested []string) bool {
	allowed := make(map[string]struct{}, len(ceiling))
	for _, scope := range ceiling {
		if value := strings.TrimSpace(scope); value != "" {
			allowed[value] = struct{}{}
		}
	}
	for _, scope := range requested {
		if value := strings.TrimSpace(scope); value != "" {
			if _, ok := allowed[value]; !ok {
				return false
			}
		}
	}
	return true
}

type DiagnosisKind string

const (
	DiagnosisUnauthenticated     DiagnosisKind = "unauthenticated"
	DiagnosisProviderUnavailable DiagnosisKind = "provider_unavailable"
	DiagnosisEnrollmentRequired  DiagnosisKind = "enrollment_required"
)

type Diagnosis struct {
	Kind     DiagnosisKind `json:"kind"`
	Provider string        `json:"provider,omitempty"`
	Mode     Mode          `json:"mode,omitempty"`
	Reason   string        `json:"reason,omitempty"`
	Recovery string        `json:"recovery,omitempty"`
}

func (d Diagnosis) Error() string {
	parts := []string{string(d.Kind)}
	if d.Provider != "" {
		parts = append(parts, "provider="+d.Provider)
	}
	if d.Mode != "" {
		parts = append(parts, "mode="+string(d.Mode))
	}
	if d.Reason != "" {
		parts = append(parts, "reason="+d.Reason)
	}
	if d.Recovery != "" {
		parts = append(parts, "recovery="+d.Recovery)
	}
	return strings.Join(parts, ": ")
}

func ProviderUnavailable(cause error, mode Mode) Diagnosis {
	reason := "operator identity provider is unavailable"
	if cause != nil && strings.TrimSpace(cause.Error()) != "" {
		reason = cause.Error()
	}
	return Diagnosis{Kind: DiagnosisProviderUnavailable, Provider: IdentityProviderAuthenticator, Mode: mode, Reason: reason, Recovery: "start scenario-authenticator and enroll this machine once; already-enrolled machines mint locally"}
}

func EnrollmentRequired(cause error, mode Mode) Diagnosis {
	reason := "this machine has no operator enrollment"
	if cause != nil && strings.TrimSpace(cause.Error()) != "" {
		reason = cause.Error()
	}
	return Diagnosis{Kind: DiagnosisEnrollmentRequired, Provider: IdentityProviderAuthenticator, Mode: mode, Reason: reason, Recovery: "authenticate once to enroll this machine"}
}

func (e Enrollment) Validate() error {
	if strings.TrimSpace(e.OperatorID) == "" {
		return fmt.Errorf("operator enrollment: operator_id is required")
	}
	if strings.TrimSpace(e.IdentityProvider) == "" {
		return fmt.Errorf("operator enrollment: identity_provider is required")
	}
	if e.Mode != ModePersonal && e.Mode != ModeShared && e.Mode != ModeHosted {
		return fmt.Errorf("operator enrollment: invalid session_mode %q", e.Mode)
	}
	if strings.TrimSpace(e.Reference) == "" {
		return fmt.Errorf("operator enrollment: enrollment_reference is required")
	}
	if e.EnrolledAt.IsZero() {
		return fmt.Errorf("operator enrollment: enrolled_at is required")
	}
	return nil
}
