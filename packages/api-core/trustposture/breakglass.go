package trustposture

import (
	"crypto/aes"
	"crypto/cipher"
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

	"golang.org/x/crypto/argon2"
)

const (
	breakGlassAlgorithm = "EdDSA"
	breakGlassType      = "VrooliBreakGlass"
	breakGlassKDF       = "argon2id"
	breakGlassVersion   = 1
	breakGlassMemoryKiB = 64 * 1024
	breakGlassTime      = 3
	breakGlassThreads   = 2
	breakGlassKeySize   = 32

	// BreakGlassUninstallAudience and BreakGlassUninstallScope are the shared
	// purpose contract used by the control-plane destructive gate and its
	// issuer. Keeping them here prevents those strings from drifting again.
	BreakGlassUninstallAudience = "vrooli:uninstall"
	BreakGlassUninstallScope    = "vrooli:uninstall"
)

var (
	ErrBreakGlassAudienceMismatch = errors.New("break-glass: audience mismatch")
	ErrBreakGlassPassphrase       = errors.New("break-glass: passphrase authorization failed")
	ErrBreakGlassTargetMismatch   = errors.New("break-glass: target mismatch")
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
	Dir            string
	Private        string
	WrappedPrivate string
	Public         string
	Metadata       string
	Credential     string
}

type provisionMetadata struct {
	AccountID   string   `json:"account_id"`
	Audience    string   `json:"audience"`
	Target      string   `json:"target"`
	Scopes      []string `json:"scopes"`
	Provisioned int64    `json:"provisioned_at"`
}

type wrappedPrivateEnvelope struct {
	Version    int    `json:"version"`
	KDF        string `json:"kdf"`
	Memory     uint32 `json:"memory_kib"`
	Time       uint32 `json:"time_cost"`
	Threads    uint8  `json:"threads"`
	Salt       string `json:"salt"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
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
		Dir: dir, Private: filepath.Join(dir, "private.key"), WrappedPrivate: filepath.Join(dir, "private.key"),
		Public: filepath.Join(dir, "public.key"), Metadata: filepath.Join(dir, "provisioning.json"), Credential: filepath.Join(dir, "credential"),
	}, nil
}

// ResolveAuthenticatorKeyPaths returns the separate server-side key location.
// The authenticated scenario issuer is a running service, not the operator's
// local break-glass command; keeping its compatibility key separate means a
// plaintext server-side migration cannot accidentally weaken the operator
// credential stored under ResolveKeyPaths.
func ResolveAuthenticatorKeyPaths() (KeyPaths, error) {
	dir, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(dir) == "" {
		return KeyPaths{}, errors.New("break-glass: resolve user home")
	}
	dir = filepath.Join(dir, ".vrooli", "identity", "authenticator-break-glass")
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

// ProvisionWrapped creates operator-controlled material without ever writing
// the private key in plaintext. The passphrase is not persisted or logged.
func ProvisionWrapped(paths KeyPaths, passphrase, accountID, audience, target string, scopes []string, now time.Time) error {
	if strings.TrimSpace(passphrase) == "" || strings.TrimSpace(accountID) == "" || strings.TrimSpace(audience) == "" || strings.TrimSpace(target) == "" || now.IsZero() {
		return errors.New("break-glass: passphrase, account, audience, target and timestamp are required")
	}
	if strings.TrimSpace(paths.WrappedPrivate) == "" {
		return errors.New("break-glass: wrapped private-key path is required")
	}
	metadata := provisionMetadata{AccountID: strings.TrimSpace(accountID), Audience: strings.TrimSpace(audience), Target: strings.TrimSpace(target), Scopes: uniqueScopes(scopes), Provisioned: now.Unix()}
	wrappedRaw, wrappedErr := os.ReadFile(paths.WrappedPrivate)
	publicRaw, publicErr := os.ReadFile(paths.Public)
	metadataRaw, metadataErr := os.ReadFile(paths.Metadata)
	if wrappedErr == nil || publicErr == nil || metadataErr == nil {
		if wrappedErr != nil || publicErr != nil || metadataErr != nil {
			return errors.New("break-glass: incomplete existing wrapped key material")
		}
		private, err := UnwrapPrivate(wrappedRaw, passphrase)
		if err != nil {
			return errors.New("break-glass: existing key material could not be opened")
		}
		zeroBytes(private)
		var existing provisionMetadata
		if json.Unmarshal(metadataRaw, &existing) != nil || existing.AccountID != metadata.AccountID || existing.Audience != metadata.Audience || existing.Target != metadata.Target {
			return errors.New("break-glass: existing key material belongs to another target or purpose")
		}
		if len(publicRaw) != ed25519.PublicKeySize {
			return errors.New("break-glass: invalid existing public key")
		}
		return nil
	}
	if !errors.Is(wrappedErr, os.ErrNotExist) || !errors.Is(publicErr, os.ErrNotExist) || !errors.Is(metadataErr, os.ErrNotExist) {
		return errors.New("break-glass: inspect existing wrapped key material")
	}
	keys, err := GenerateKeyMaterial()
	if err != nil {
		return err
	}
	wrapped, err := WrapPrivate(keys.PrivateKey, passphrase)
	zeroBytes(keys.PrivateKey)
	if err != nil {
		return err
	}
	if err := writeKey(paths.WrappedPrivate, wrapped, 0o600, false); err != nil {
		return err
	}
	if err := WritePublic(paths.Public, keys.PublicKey); err != nil {
		return err
	}
	return writeProvisionMetadata(paths.Metadata, metadata)
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
	defer zeroBytes(private)
	metadataRaw, err := os.ReadFile(paths.Metadata)
	if err != nil {
		return "", fmt.Errorf("break-glass: read provisioning metadata: %w", err)
	}
	var metadata provisionMetadata
	if err := json.Unmarshal(metadataRaw, &metadata); err != nil {
		return "", fmt.Errorf("break-glass: parse provisioning metadata: %w", err)
	}
	return IssueForAccount(ed25519.PrivateKey(private), metadata.Scopes, requested, BreakGlassClaims{
		Subject: metadata.AccountID, Audience: metadata.Audience, Target: metadata.Target,
		IssuedAt: now.Unix(), ExpiresAt: now.Add(ttl).Unix(),
	})
}

// IssueFromProvisionForTarget is the compatibility issuer used by the
// authenticated scenario service, whose private key is kept in its separate
// server-owned key directory. The service supplies its current local target;
// operator-controlled issuance must use IssueFromWrappedProvision instead.
func IssueFromProvisionForTarget(paths KeyPaths, target string, requested []string, now time.Time, ttl time.Duration) (string, error) {
	if strings.TrimSpace(target) == "" {
		return "", errors.New("break-glass: target is required")
	}
	private, err := os.ReadFile(paths.Private)
	if err != nil {
		return "", fmt.Errorf("break-glass: read private key: %w", err)
	}
	defer zeroBytes(private)
	metadata, err := readProvisionMetadata(paths.Metadata)
	if err != nil {
		return "", err
	}
	return IssueForAccount(ed25519.PrivateKey(private), metadata.Scopes, requested, BreakGlassClaims{
		Subject: metadata.AccountID, Audience: metadata.Audience, Target: strings.TrimSpace(target),
		IssuedAt: now.Unix(), ExpiresAt: now.Add(ttl).Unix(),
	})
}

// IssueFromWrappedProvision opens the operator-wrapped key only for the
// duration of signing. Audience and target are explicit issuance inputs and
// must match the provisioned purpose and machine binding.
func IssueFromWrappedProvision(paths KeyPaths, passphrase, audience, target string, requested []string, now time.Time, ttl time.Duration) (string, error) {
	if strings.TrimSpace(passphrase) == "" || strings.TrimSpace(audience) == "" || strings.TrimSpace(target) == "" {
		return "", errors.New("break-glass: passphrase, audience and target are required")
	}
	if ttl <= 0 || now.IsZero() {
		return "", errors.New("break-glass: lifetime is required")
	}
	wrapper, err := os.ReadFile(paths.WrappedPrivate)
	if err != nil {
		return "", fmt.Errorf("break-glass: read wrapped private key: %w", err)
	}
	private, err := UnwrapPrivate(wrapper, passphrase)
	if err != nil {
		return "", err
	}
	defer zeroBytes(private)
	metadata, err := readProvisionMetadata(paths.Metadata)
	if err != nil {
		return "", err
	}
	if metadata.Audience != strings.TrimSpace(audience) || metadata.Target != strings.TrimSpace(target) {
		return "", errors.New("break-glass: issuance purpose or target does not match provisioning")
	}
	return IssueForAccount(ed25519.PrivateKey(private), metadata.Scopes, requested, BreakGlassClaims{
		Subject: metadata.AccountID, Audience: metadata.Audience, Target: metadata.Target,
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
	Target    string   `json:"target"`
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
	if len(private) != ed25519.PrivateKeySize || strings.TrimSpace(claims.Subject) == "" || strings.TrimSpace(claims.Audience) == "" || strings.TrimSpace(claims.Target) == "" {
		return "", errors.New("break-glass: key, subject, audience and target are required")
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
func Verify(public ed25519.PublicKey, token, audience, target string, now time.Time) (BreakGlassClaims, error) {
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
	if claims.Subject == "" || claims.IssuedAt <= 0 || claims.ExpiresAt <= claims.IssuedAt {
		return BreakGlassClaims{}, errors.New("break-glass: invalid claims")
	}
	if claims.Audience != audience {
		return BreakGlassClaims{}, fmt.Errorf("%w: expected %q", ErrBreakGlassAudienceMismatch, audience)
	}
	if claims.Target != target {
		return BreakGlassClaims{}, fmt.Errorf("%w: expected %q", ErrBreakGlassTargetMismatch, target)
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

// Status reports operator-material presence without opening or exposing the
// wrapped key. It is safe for diagnostics and does not require a passphrase.
type KeyStatus struct {
	WrappedPrivate bool     `json:"wrapped_private"`
	Public         bool     `json:"public"`
	Metadata       bool     `json:"metadata"`
	Complete       bool     `json:"complete"`
	AccountID      string   `json:"account_id,omitempty"`
	Audience       string   `json:"audience,omitempty"`
	Target         string   `json:"target,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
	ProvisionedAt  int64    `json:"provisioned_at,omitempty"`
}

func Status(paths KeyPaths) (KeyStatus, error) {
	status := KeyStatus{}
	for _, item := range []struct {
		path  string
		field *bool
	}{
		{paths.WrappedPrivate, &status.WrappedPrivate},
		{paths.Public, &status.Public},
		{paths.Metadata, &status.Metadata},
	} {
		if strings.TrimSpace(item.path) == "" {
			continue
		}
		_, err := os.Stat(item.path)
		if err == nil {
			*item.field = true
			continue
		}
		if !errors.Is(err, os.ErrNotExist) {
			return KeyStatus{}, fmt.Errorf("break-glass: inspect %s: %w", item.path, err)
		}
	}
	status.Complete = status.WrappedPrivate && status.Public && status.Metadata
	if status.Metadata {
		metadata, err := readProvisionMetadata(paths.Metadata)
		if err != nil {
			return KeyStatus{}, err
		}
		status.AccountID = metadata.AccountID
		status.Audience = metadata.Audience
		status.Target = metadata.Target
		status.Scopes = append([]string(nil), metadata.Scopes...)
		status.ProvisionedAt = metadata.Provisioned
	}
	return status, nil
}

// WriteCredential stores a just-issued credential in the owner-only material
// directory. The credential itself is intentionally never printed by the CLI.
func WriteCredential(paths KeyPaths, token string) error {
	if strings.TrimSpace(paths.Credential) == "" || strings.TrimSpace(token) == "" {
		return errors.New("break-glass: credential path and token are required")
	}
	return writeKey(paths.Credential, []byte(strings.TrimSpace(token)+"\n"), 0o600, true)
}

// WrapPrivate encrypts an Ed25519 private key with an operator passphrase.
// Argon2id parameters are persisted in the envelope so verification remains
// deterministic if the work factor is raised in a future format version.
func WrapPrivate(private ed25519.PrivateKey, passphrase string) ([]byte, error) {
	if len(private) != ed25519.PrivateKeySize || strings.TrimSpace(passphrase) == "" {
		return nil, errors.New("break-glass: private key and passphrase are required")
	}
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("break-glass: generate wrap salt: %w", err)
	}
	key := argon2.IDKey([]byte(passphrase), salt, breakGlassTime, breakGlassMemoryKiB, breakGlassThreads, breakGlassKeySize)
	defer zeroBytes(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("break-glass: create wrap cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("break-glass: create wrap mode: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("break-glass: generate wrap nonce: %w", err)
	}
	envelope := wrappedPrivateEnvelope{
		Version: breakGlassVersion, KDF: breakGlassKDF, Memory: breakGlassMemoryKiB, Time: breakGlassTime, Threads: breakGlassThreads,
		Salt: base64.RawStdEncoding.EncodeToString(salt), Nonce: base64.RawStdEncoding.EncodeToString(nonce),
		Ciphertext: base64.RawStdEncoding.EncodeToString(gcm.Seal(nil, nonce, private, nil)),
	}
	return json.Marshal(envelope)
}

// UnwrapPrivate decrypts an operator-wrapped private key and returns a fresh
// byte slice owned by the caller. Wrong passphrases and corrupted envelopes
// share an authorization-style error so diagnostics do not reveal which part
// of the secret material was wrong.
func UnwrapPrivate(raw []byte, passphrase string) ([]byte, error) {
	if strings.TrimSpace(passphrase) == "" {
		return nil, errors.New("break-glass: passphrase is required")
	}
	var envelope wrappedPrivateEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil || envelope.Version != breakGlassVersion || envelope.KDF != breakGlassKDF || !validArgon2Parameters(envelope) {
		return nil, errors.New("break-glass: unsupported wrapped private-key envelope")
	}
	salt, err := base64.RawStdEncoding.DecodeString(envelope.Salt)
	if err != nil || len(salt) < 16 {
		return nil, errors.New("break-glass: invalid wrapped private-key salt")
	}
	nonce, err := base64.RawStdEncoding.DecodeString(envelope.Nonce)
	if err != nil {
		return nil, errors.New("break-glass: invalid wrapped private-key nonce")
	}
	ciphertext, err := base64.RawStdEncoding.DecodeString(envelope.Ciphertext)
	if err != nil {
		return nil, errors.New("break-glass: invalid wrapped private-key ciphertext")
	}
	key := argon2.IDKey([]byte(passphrase), salt, envelope.Time, envelope.Memory, envelope.Threads, breakGlassKeySize)
	defer zeroBytes(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.New("break-glass: invalid wrapped private-key cipher")
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil || len(nonce) != gcm.NonceSize() {
		return nil, errors.New("break-glass: invalid wrapped private-key nonce")
	}
	private, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil || len(private) != ed25519.PrivateKeySize {
		return nil, ErrBreakGlassPassphrase
	}
	return private, nil
}

func validArgon2Parameters(envelope wrappedPrivateEnvelope) bool {
	return envelope.Memory >= 16*1024 && envelope.Memory <= 256*1024 && envelope.Time >= 1 && envelope.Time <= 10 && envelope.Threads >= 1 && envelope.Threads <= 8
}

func writeProvisionMetadata(path string, metadata provisionMetadata) error {
	raw, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("break-glass: encode provisioning metadata: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("break-glass: create metadata directory: %w", err)
	}
	return writeKey(path, append(raw, '\n'), 0o600, false)
}

func readProvisionMetadata(path string) (provisionMetadata, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return provisionMetadata{}, fmt.Errorf("break-glass: read provisioning metadata: %w", err)
	}
	var metadata provisionMetadata
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return provisionMetadata{}, fmt.Errorf("break-glass: parse provisioning metadata: %w", err)
	}
	return metadata, nil
}

func zeroBytes(value []byte) {
	for i := range value {
		value[i] = 0
	}
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
