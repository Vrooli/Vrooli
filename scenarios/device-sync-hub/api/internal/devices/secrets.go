package devices

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

// Secrets is the seam that mints the two opaque credentials this domain issues:
// the long, unguessable hub device token and the short, human-typable pairing
// code. Declared as a seam so pairing/redeem tests run against a deterministic
// generator instead of real entropy.
type Secrets interface {
	// DeviceToken returns a high-entropy opaque token. Only its hash is stored.
	DeviceToken() (string, error)
	// PairingCode returns a short, human-typable, case-insensitive code.
	PairingCode() (string, error)
}

// CryptoSecrets is the production Secrets backed by crypto/rand.
type CryptoSecrets struct{}

// Compile-time guarantee.
var _ Secrets = CryptoSecrets{}

// DeviceToken returns 32 bytes of crypto-random entropy, base64url-encoded.
func (CryptoSecrets) DeviceToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("read entropy for device token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// pairingCodeAlphabet is Crockford-ish base32 over the standard RFC4648 set —
// enough entropy (10 chars × 5 bits = 50 bits) to resist online guessing
// within the short TTL while staying typable. Hyphenated for readability.
var pairingCodeEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// PairingCode returns a 10-character base32 code, grouped as XXXXX-XXXXX.
func (CryptoSecrets) PairingCode() (string, error) {
	b := make([]byte, 7) // 7 bytes -> ceil(56/5)=12 base32 chars; trim to 10.
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("read entropy for pairing code: %w", err)
	}
	raw := pairingCodeEncoding.EncodeToString(b)[:10]
	return raw[:5] + "-" + raw[5:], nil
}

// normalizePairingCode canonicalizes a user-entered code for hashing: upper
// case, hyphens/spaces stripped. So "abcde-fghij", "ABCDE FGHIJ", and
// "ABCDEFGHIJ" all hash identically.
func normalizePairingCode(code string) string {
	r := strings.NewReplacer("-", "", " ", "")
	return strings.ToUpper(r.Replace(strings.TrimSpace(code)))
}

// hashSecret is the one-way hash applied to both device tokens and pairing
// codes before persistence — the raw value never touches the database.
func hashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// hashPairingCode normalizes then hashes a pairing code.
func hashPairingCode(code string) string { return hashSecret(normalizePairingCode(code)) }
