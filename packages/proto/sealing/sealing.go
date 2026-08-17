// Package sealing provides the small, dependency-free envelope used for
// operator secrets crossing the Bridge control plane. It deliberately lives
// in the shared wire module so the CLI and node helper cannot implement
// subtly different formats.
package sealing

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

const (
	version       = "VCS1"
	publicKeySize = 32
	maxPlaintext  = 64 * 1024
)

var p25519 = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 255), big.NewInt(19))

// Context creates the canonical additional-authentication-data binding for a
// cleanup authorization envelope. Empty fields remain positional so target,
// operation, and plan substitutions cannot produce the same bytes.
func Context(machineID, nodeID, target, scope, planHash, operationID, operatorID string) []byte {
	return []byte(strings.Join([]string{"vrooli-cleanup-context-v1", machineID, nodeID, target, scope, planHash, operationID, operatorID}, "\x00"))
}

// PrivateKeyFromEd25519Seed derives the X25519 sealing key from the existing
// node credential seed. The seed never leaves the node. Deriving rather than
// storing another private key keeps pairing and service installation atomic.
func PrivateKeyFromEd25519Seed(seed []byte) (*ecdh.PrivateKey, error) {
	if len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("sealing: Ed25519 seed must be %d bytes", ed25519.SeedSize)
	}
	hash := sha512.Sum512(seed)
	return ecdh.X25519().NewPrivateKey(hash[:32])
}

// PublicKeyFromEd25519 converts the registered node Ed25519 public key to the
// matching X25519 public key. This is the standard Edwards y to Montgomery u
// map, with no private material involved on the control plane.
func PublicKeyFromEd25519(public ed25519.PublicKey) ([]byte, error) {
	if len(public) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("sealing: Ed25519 public key must be %d bytes", ed25519.PublicKeySize)
	}
	yBytes := append([]byte(nil), public...)
	yBytes[31] &= 0x7f
	y := littleInt(yBytes)
	if y.Cmp(p25519) >= 0 {
		return nil, errors.New("sealing: invalid Ed25519 public key")
	}
	numerator := new(big.Int).Add(big.NewInt(1), y)
	numerator.Mod(numerator, p25519)
	denominator := new(big.Int).Sub(big.NewInt(1), y)
	denominator.Mod(denominator, p25519)
	inverse := new(big.Int).ModInverse(denominator, p25519)
	if inverse == nil {
		return nil, errors.New("sealing: Ed25519 public key has no Montgomery image")
	}
	u := new(big.Int).Mul(numerator, inverse)
	u.Mod(u, p25519)
	return littleBytes(u, publicKeySize), nil
}

// Seal encrypts plaintext to the recipient's X25519 public key. The returned
// bytes are safe to persist and relay as opaque data. AAD binds the envelope
// to the operation context (machine, target, plan hash, and operator id).
func Seal(recipientPublic, plaintext, aad []byte) ([]byte, error) {
	if len(recipientPublic) != publicKeySize {
		return nil, errors.New("sealing: recipient public key must be 32 bytes")
	}
	if len(plaintext) == 0 || len(plaintext) > maxPlaintext {
		return nil, errors.New("sealing: plaintext must be between 1 and 65536 bytes")
	}
	recipient, err := ecdh.X25519().NewPublicKey(recipientPublic)
	if err != nil {
		return nil, fmt.Errorf("sealing: recipient key: %w", err)
	}
	ephemeral, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("sealing: ephemeral key: %w", err)
	}
	shared, err := ephemeral.ECDH(recipient)
	if err != nil {
		return nil, fmt.Errorf("sealing: derive shared key: %w", err)
	}
	defer zero(shared)
	key := derive(shared, ephemeral.PublicKey().Bytes(), recipientPublic, aad)
	defer zero(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("sealing: cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("sealing: gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("sealing: nonce: %w", err)
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, aad)
	out := make([]byte, 0, len(version)+publicKeySize+len(nonce)+len(ciphertext))
	out = append(out, version...)
	out = append(out, ephemeral.PublicKey().Bytes()...)
	out = append(out, nonce...)
	out = append(out, ciphertext...)
	return out, nil
}

// Open decrypts an envelope using the node's derived private key and verifies
// the same context binding supplied to Seal.
func Open(private *ecdh.PrivateKey, envelope, aad []byte) ([]byte, error) {
	if private == nil || private.Curve() != ecdh.X25519() {
		return nil, errors.New("sealing: X25519 private key is required")
	}
	if len(envelope) < len(version)+publicKeySize+12+16 || string(envelope[:len(version)]) != version {
		return nil, errors.New("sealing: malformed envelope")
	}
	offset := len(version)
	ephemeralRaw := envelope[offset : offset+publicKeySize]
	offset += publicKeySize
	ephemeral, err := ecdh.X25519().NewPublicKey(ephemeralRaw)
	if err != nil {
		return nil, fmt.Errorf("sealing: ephemeral key: %w", err)
	}
	shared, err := private.ECDH(ephemeral)
	if err != nil {
		return nil, fmt.Errorf("sealing: derive shared key: %w", err)
	}
	defer zero(shared)
	key := derive(shared, ephemeralRaw, private.PublicKey().Bytes(), aad)
	defer zero(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("sealing: cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("sealing: gcm: %w", err)
	}
	if len(envelope) < offset+gcm.NonceSize()+gcm.Overhead() {
		return nil, errors.New("sealing: truncated envelope")
	}
	nonce := envelope[offset : offset+gcm.NonceSize()]
	ciphertext := envelope[offset+gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, errors.New("sealing: authentication failed")
	}
	if len(plaintext) == 0 || len(plaintext) > maxPlaintext {
		zero(plaintext)
		return nil, errors.New("sealing: invalid plaintext size")
	}
	return plaintext, nil
}

func derive(shared, ephemeral, recipient, aad []byte) []byte {
	salt := []byte("vrooli-cleanup-sealing-v1")
	extractor := hmac.New(sha256.New, salt)
	_, _ = extractor.Write(shared)
	prk := extractor.Sum(nil)
	defer zero(prk)
	expander := hmac.New(sha256.New, prk)
	_, _ = expander.Write([]byte("envelope"))
	_, _ = expander.Write(ephemeral)
	_, _ = expander.Write(recipient)
	_, _ = expander.Write(aad)
	_, _ = expander.Write([]byte{1})
	return expander.Sum(nil)
}

func littleInt(raw []byte) *big.Int {
	reversed := append([]byte(nil), raw...)
	reverse(reversed)
	return new(big.Int).SetBytes(reversed)
}

func littleBytes(value *big.Int, size int) []byte {
	raw := value.Bytes()
	out := make([]byte, size)
	for i := 0; i < len(raw) && i < size; i++ {
		out[i] = raw[len(raw)-1-i]
	}
	return out
}

func reverse(raw []byte) {
	for left, right := 0, len(raw)-1; left < right; left, right = left+1, right-1 {
		raw[left], raw[right] = raw[right], raw[left]
	}
}

func zero(raw []byte) {
	for i := range raw {
		raw[i] = 0
	}
}
