package trustposture

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	breakGlassAlgorithm = "EdDSA"
	breakGlassType      = "VrooliBreakGlass"
)

// KeyMaterial is the local private/public credential pair. PrivateKey must
// remain owner-only; PublicKey is the value pinned by a relying party.
type KeyMaterial struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
}

// KeyPaths is the per-machine break-glass material location. The private key
// and its provisioning metadata are owner-only; the public key is the pinned
// verifier input read by the local control plane.
type KeyPaths struct {
	Dir      string
	Private  string
	Public   string
	Metadata string
}

type provisionMetadata struct {
	AccountID   string   `json:"account_id"`
	Audience    string   `json:"audience"`
	Scopes      []string `json:"scopes"`
	Provisioned int64    `json:"provisioned_at"`
}

// ResolveKeyPaths returns the stable per-install location shared by the
// authenticator provisioner and the bridge verifier. VROOLI_BREAK_GLASS_DIR
// is an explicit operator override for isolated installations and tests.
func ResolveKeyPaths() (KeyPaths, error) {
	dir := strings.TrimSpace(os.Getenv("VROOLI_BREAK_GLASS_DIR"))
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil || strings.TrimSpace(home) == "" {
			return KeyPaths{}, errors.New("break-glass: resolve user home")
		}
		dir = filepath.Join(home, ".vrooli", "identity", "break-glass")
	}
	return KeyPaths{
		Dir: dir, Private: filepath.Join(dir, "private.key"),
		Public: filepath.Join(dir, "public.key"), Metadata: filepath.Join(dir, "provisioning.json"),
	}, nil
}

// Provision creates the local key pair once and records the account scope
// ceiling needed for later, offline credential issuance. Existing material is
// never replaced; relinking the same machine is therefore idempotent only for
// the same account and audience.
func Provision(paths KeyPaths, accountID, audience string, scopes []string, now time.Time) error {
	if strings.TrimSpace(accountID) == "" || strings.TrimSpace(audience) == "" || now.IsZero() {
		return errors.New("break-glass: account, audience and timestamp are required")
	}
	metadata := provisionMetadata{AccountID: strings.TrimSpace(accountID), Audience: strings.TrimSpace(audience), Scopes: uniqueScopes(scopes), Provisioned: now.Unix()}
	privateRaw, privateErr := os.ReadFile(paths.Private)
	publicRaw, publicErr := os.ReadFile(paths.Public)
	metadataRaw, metadataErr := os.ReadFile(paths.Metadata)
	if privateErr == nil || publicErr == nil || metadataErr == nil {
		if privateErr != nil || publicErr != nil || metadataErr != nil || len(privateRaw) != ed25519.PrivateKeySize || len(publicRaw) != ed25519.PublicKeySize {
			return errors.New("break-glass: incomplete existing key material")
		}
		var existing provisionMetadata
		if err := json.Unmarshal(metadataRaw, &existing); err != nil || existing.AccountID != metadata.AccountID || existing.Audience != metadata.Audience {
			return errors.New("break-glass: existing key material belongs to another account")
		}
		return nil
	}
	if !errors.Is(privateErr, os.ErrNotExist) || !errors.Is(publicErr, os.ErrNotExist) || !errors.Is(metadataErr, os.ErrNotExist) {
		return errors.New("break-glass: inspect existing key material")
	}
	keys, err := GenerateKeyMaterial()
	if err != nil {
		return err
	}
	if err := WritePrivate(paths.Private, keys.PrivateKey); err != nil {
		return err
	}
	if err := WritePublic(paths.Public, keys.PublicKey); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("break-glass: encode provisioning metadata: %w", err)
	}
	if err := os.WriteFile(paths.Metadata, raw, 0o600); err != nil {
		return fmt.Errorf("break-glass: write provisioning metadata: %w", err)
	}
	return nil
}

// IssueFromProvision signs a short-lived offline credential from the
// owner-only private half, applying the scope ceiling recorded at linking.
func IssueFromProvision(paths KeyPaths, requested []string, now time.Time, ttl time.Duration) (string, error) {
	if ttl <= 0 || now.IsZero() {
		return "", errors.New("break-glass: lifetime is required")
	}
	private, err := os.ReadFile(paths.Private)
	if err != nil {
		return "", fmt.Errorf("break-glass: read private key: %w", err)
	}
	metadataRaw, err := os.ReadFile(paths.Metadata)
	if err != nil {
		return "", fmt.Errorf("break-glass: read provisioning metadata: %w", err)
	}
	var metadata provisionMetadata
	if err := json.Unmarshal(metadataRaw, &metadata); err != nil {
		return "", fmt.Errorf("break-glass: parse provisioning metadata: %w", err)
	}
	return IssueForAccount(ed25519.PrivateKey(private), metadata.Scopes, requested, BreakGlassClaims{
		Subject: metadata.AccountID, Audience: metadata.Audience,
		IssuedAt: now.Unix(), ExpiresAt: now.Add(ttl).Unix(),
	})
}

func uniqueScopes(scopes []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if scope != "" {
			if _, ok := seen[scope]; !ok {
				seen[scope] = struct{}{}
				result = append(result, scope)
			}
		}
	}
	return result
}

// GenerateKeyMaterial creates a new offline break-glass signing key.
func GenerateKeyMaterial() (KeyMaterial, error) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return KeyMaterial{}, fmt.Errorf("generate break-glass key: %w", err)
	}
	return KeyMaterial{PrivateKey: private, PublicKey: public}, nil
}

// BreakGlassClaims are the signed, time-boxed capability claims.
type BreakGlassClaims struct {
	Subject   string   `json:"sub"`
	Audience  string   `json:"aud"`
	Scopes    []string `json:"scope"`
	IssuedAt  int64    `json:"iat"`
	ExpiresAt int64    `json:"exp"`
}

type breakGlassHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

// Issue signs an offline credential. Callers should use IssueForAccount when
// the requested scopes came from an account grant.
func Issue(private ed25519.PrivateKey, claims BreakGlassClaims) (string, error) {
	if len(private) != ed25519.PrivateKeySize || strings.TrimSpace(claims.Subject) == "" || strings.TrimSpace(claims.Audience) == "" {
		return "", errors.New("break-glass: key, subject and audience are required")
	}
	if claims.ExpiresAt <= claims.IssuedAt || claims.IssuedAt <= 0 {
		return "", errors.New("break-glass: invalid lifetime")
	}
	claims.Scopes = append([]string{}, claims.Scopes...)
	header, _ := json.Marshal(breakGlassHeader{Algorithm: breakGlassAlgorithm, Type: breakGlassType})
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("break-glass: encode claims: %w", err)
	}
	encodedHeader := base64.RawURLEncoding.EncodeToString(header)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signingInput := encodedHeader + "." + encodedPayload
	signature := ed25519.Sign(private, []byte(signingInput))
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

// IssueForAccount applies the account-scope ceiling before signing.
func IssueForAccount(private ed25519.PrivateKey, accountScopes, requested []string, claims BreakGlassClaims) (string, error) {
	ceiling, err := ScopeCeiling(accountScopes, requested)
	if err != nil {
		return "", err
	}
	claims.Scopes = ceiling
	return Issue(private, claims)
}

// Verify verifies the signature, header, audience, lifetime, and explicit
// non-empty subject. It performs no network call.
func Verify(public ed25519.PublicKey, token, audience string, now time.Time) (BreakGlassClaims, error) {
	if len(public) != ed25519.PublicKeySize {
		return BreakGlassClaims{}, errors.New("break-glass: invalid pinned public key")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return BreakGlassClaims{}, errors.New("break-glass: malformed credential")
	}
	headerRaw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return BreakGlassClaims{}, errors.New("break-glass: malformed header")
	}
	var header breakGlassHeader
	if err := json.Unmarshal(headerRaw, &header); err != nil || header.Algorithm != breakGlassAlgorithm || header.Type != breakGlassType {
		return BreakGlassClaims{}, errors.New("break-glass: unsupported credential header")
	}
	payloadRaw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return BreakGlassClaims{}, errors.New("break-glass: malformed claims")
	}
	var claims BreakGlassClaims
	if err := json.Unmarshal(payloadRaw, &claims); err != nil {
		return BreakGlassClaims{}, errors.New("break-glass: malformed claims")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !ed25519.Verify(public, []byte(parts[0]+"."+parts[1]), signature) {
		return BreakGlassClaims{}, errors.New("break-glass: signature verification failed")
	}
	if claims.Subject == "" || claims.Audience != audience || claims.IssuedAt <= 0 || claims.ExpiresAt <= claims.IssuedAt {
		return BreakGlassClaims{}, errors.New("break-glass: invalid claims")
	}
	if now.Unix() < claims.IssuedAt || now.Unix() >= claims.ExpiresAt {
		return BreakGlassClaims{}, errors.New("break-glass: credential expired or not yet valid")
	}
	if claims.Scopes == nil {
		return BreakGlassClaims{}, errors.New("break-glass: scope claim is required")
	}
	return claims, nil
}

// ScopeCeiling ensures every break-glass scope is granted by the account.
func ScopeCeiling(accountScopes, requested []string) ([]string, error) {
	result := make([]string, 0, len(requested))
	seen := map[string]struct{}{}
	for _, scope := range requested {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			return nil, errors.New("break-glass: empty requested scope")
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		allowed := false
		for _, grant := range accountScopes {
			if scopeMatches(grant, scope) {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, fmt.Errorf("break-glass: scope %q exceeds account ceiling", scope)
		}
		result = append(result, scope)
	}
	return result, nil
}

func scopeMatches(grant, requested string) bool {
	grant, requested = strings.TrimSpace(grant), strings.TrimSpace(requested)
	if grant == "*" || grant == requested {
		return true
	}
	parts := strings.Split(requested, ":")
	if len(parts) != 2 {
		return false
	}
	return grant == parts[0]+":*" || grant == "*:"+parts[1]
}

// WritePrivate writes owner-only private material, refusing to replace an
// existing credential. This is the provisioning seam used by Phase 6.
func WritePrivate(path string, private ed25519.PrivateKey) error {
	if len(private) != ed25519.PrivateKeySize {
		return errors.New("break-glass: invalid private key")
	}
	return writeKey(path, private, 0o600, false)
}

// WritePublic writes the pinned public half. It is not a secret, but remains
// in a controlled directory selected by the scenario.
func WritePublic(path string, public ed25519.PublicKey) error {
	if len(public) != ed25519.PublicKeySize {
		return errors.New("break-glass: invalid public key")
	}
	return writeKey(path, public, 0o644, false)
}

func writeKey(path string, key []byte, mode os.FileMode, replace bool) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("break-glass: create key directory: %w", err)
	}
	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if replace {
		flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}
	f, err := os.OpenFile(path, flags, mode)
	if err != nil {
		return fmt.Errorf("break-glass: create key file: %w", err)
	}
	if _, err := f.Write(key); err != nil {
		_ = f.Close()
		return fmt.Errorf("break-glass: write key file: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("break-glass: close key file: %w", err)
	}
	return nil
}
