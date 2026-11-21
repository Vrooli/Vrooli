// Package mfa contains the authenticator-owned second-factor primitives.
// Transport, enrollment policy, persistence, and session issuance remain
// above this package so callers cannot accidentally treat a code check as a
// completed login.
package mfa

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" // RFC 6238 defines HOTP/TOTP with HMAC-SHA-1.
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/argon2"
)

const (
	secretBytes       = 20
	defaultStep       = 30 * time.Second
	defaultDigits     = 6
	recoveryCodeBytes = 10
	argonMemory       = 64 * 1024
	argonTime         = 3
	argonThreads      = 2
	argonKeyLen       = 32
	argonSaltLen      = 16
)

// GenerateSecret returns a base32 RFC 6238 secret without padding.
func GenerateSecret() (string, error) {
	secret := make([]byte, secretBytes)
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("generate MFA secret: %w", err)
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret), nil
}

// ProvisioningURI returns an otpauth URI suitable for a QR code.
func ProvisioningURI(issuer, account, secret string) (string, error) {
	issuer = strings.TrimSpace(issuer)
	account = strings.TrimSpace(account)
	secret = strings.ToUpper(strings.TrimSpace(secret))
	if issuer == "" || account == "" || secret == "" {
		return "", errors.New("issuer, account, and secret are required")
	}
	if _, err := decodeSecret(secret); err != nil {
		return "", err
	}
	label := url.PathEscape(issuer + ":" + account)
	values := url.Values{
		"secret":    {secret},
		"issuer":    {issuer},
		"algorithm": {"SHA1"},
		"digits":    {strconv.Itoa(defaultDigits)},
		"period":    {strconv.Itoa(int(defaultStep / time.Second))},
	}
	return "otpauth://totp/" + label + "?" + values.Encode(), nil
}

// Code generates the six-digit TOTP for at. It exists separately from Verify
// so integration tests can use a fixed clock without weakening verification.
func Code(secret string, at time.Time) (string, error) {
	key, err := decodeSecret(secret)
	if err != nil {
		return "", err
	}
	if at.IsZero() {
		return "", errors.New("time is required")
	}
	counter := uint64(at.Unix() / int64(defaultStep/time.Second))
	var message [8]byte
	binary.BigEndian.PutUint64(message[:], counter)
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(message[:])
	digest := mac.Sum(nil)
	offset := digest[len(digest)-1] & 0x0f
	value := (uint32(digest[offset])&0x7f)<<24 |
		uint32(digest[offset+1])<<16 |
		uint32(digest[offset+2])<<8 |
		uint32(digest[offset+3])
	return fmt.Sprintf("%0*d", defaultDigits, value%1000000), nil
}

// Verify accepts the current step and one adjacent step on either side to
// tolerate bounded clock skew. It never accepts malformed or non-six-digit
// values.
func Verify(secret, presented string, at time.Time) bool {
	presented = strings.TrimSpace(presented)
	if len(presented) != defaultDigits {
		return false
	}
	for _, delta := range []time.Duration{-defaultStep, 0, defaultStep} {
		expected, err := Code(secret, at.Add(delta))
		if err != nil {
			return false
		}
		if subtle.ConstantTimeCompare([]byte(expected), []byte(presented)) == 1 {
			return true
		}
	}
	return false
}

func decodeSecret(secret string) ([]byte, error) {
	secret = strings.ToUpper(strings.TrimSpace(secret))
	if secret == "" {
		return nil, errors.New("MFA secret is required")
	}
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil || len(decoded) < 10 {
		return nil, errors.New("invalid MFA secret")
	}
	return decoded, nil
}

// GenerateRecoveryCodes returns plaintext codes exactly once. Callers must
// hash them before persistence and present the plaintext only to the owner.
func GenerateRecoveryCodes(count int) ([]string, error) {
	if count <= 0 || count > 32 {
		return nil, errors.New("recovery code count must be between 1 and 32")
	}
	codes := make([]string, count)
	for i := range codes {
		bytes := make([]byte, recoveryCodeBytes)
		if _, err := rand.Read(bytes); err != nil {
			return nil, fmt.Errorf("generate recovery code: %w", err)
		}
		codes[i] = fmt.Sprintf("%x-%x", bytes[:5], bytes[5:])
	}
	return codes, nil
}

// HashRecoveryCode produces an Argon2id PHC-like value. Only this value may
// be stored; the plaintext belongs in the enrollment response transiently.
func HashRecoveryCode(code string) (string, error) {
	code = normalizeRecoveryCode(code)
	if code == "" {
		return "", errors.New("recovery code is required")
	}
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate recovery salt: %w", err)
	}
	hash := argon2.IDKey([]byte(code), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$mfa-argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", argonMemory, argonTime, argonThreads,
		base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(salt),
		base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(hash)), nil
}

// VerifyRecoveryCode verifies a stored hash without exposing the stored
// digest or accepting a differently formatted code.
func VerifyRecoveryCode(code, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "mfa-argon2id" || parts[2] != "v=19" {
		return false
	}
	var memory, iterations, threads uint32
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &threads); err != nil {
		return false
	}
	salt, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(parts[4])
	if err != nil || len(salt) == 0 {
		return false
	}
	expected, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(parts[5])
	if err != nil || len(expected) == 0 || memory == 0 || iterations == 0 || threads == 0 {
		return false
	}
	actual := argon2.IDKey([]byte(normalizeRecoveryCode(code)), salt, iterations, memory, uint8(threads), uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

// RecoverySet is an in-memory reference implementation of the atomic
// consume rule. A durable owner should persist the same hash and used-at
// transition in one transaction; callers must never mark a code used after a
// separate successful read.
type RecoverySet struct {
	mu     sync.Mutex
	hashes []string
	used   []bool
}

// NewRecoverySet hashes the supplied plaintext codes and discards them before
// returning. It is intended for tests and for wiring the same consume logic to
// a durable repository.
func NewRecoverySet(codes []string) (*RecoverySet, error) {
	if len(codes) == 0 {
		return nil, errors.New("at least one recovery code is required")
	}
	set := &RecoverySet{hashes: make([]string, len(codes)), used: make([]bool, len(codes))}
	for i, code := range codes {
		hash, err := HashRecoveryCode(code)
		if err != nil {
			return nil, err
		}
		set.hashes[i] = hash
	}
	return set, nil
}

// Consume atomically verifies and marks one recovery code used. Replaying a
// code returns false even when its hash remains present.
func (s *RecoverySet) Consume(code string) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, hash := range s.hashes {
		if s.used[i] || !VerifyRecoveryCode(code, hash) {
			continue
		}
		s.used[i] = true
		return true
	}
	return false
}

func normalizeRecoveryCode(code string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(code), " ", ""))
}
