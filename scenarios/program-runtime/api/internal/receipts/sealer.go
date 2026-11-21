// Package receipts protects live resume authority in persisted Runtime text.
package receipts

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"regexp"
	"strings"
)

var PlaintextPattern = regexp.MustCompile(`prt_resume_v1_[A-Za-z0-9_-]*`)
var ciphertext = regexp.MustCompile(`prt_sealed_v1_[A-Za-z0-9_-]+`)

// Sealer intentionally holds a process-local key. After restart callers get an
// explicit unavailable marker, not durable recovery of the original secret.
type Sealer struct {
	aead cipher.AEAD
	err  error
}

func New() Sealer {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return Sealer{err: err}
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return Sealer{err: err}
	}
	aead, err := cipher.NewGCM(block)
	return Sealer{aead: aead, err: err}
}

func (s Sealer) Seal(value string) (string, error) {
	if !PlaintextPattern.MatchString(value) {
		return value, nil
	}
	if s.err != nil {
		return "", s.err
	}
	var sealErr error
	out := PlaintextPattern.ReplaceAllStringFunc(value, func(token string) string {
		nonce := make([]byte, s.aead.NonceSize())
		if _, err := rand.Read(nonce); err != nil {
			sealErr = err
			return "[receipt unavailable]"
		}
		sealed := s.aead.Seal(nonce, nonce, []byte(token), nil)
		return "prt_sealed_v1_" + base64.RawURLEncoding.EncodeToString(sealed)
	})
	return out, sealErr
}

func (s Sealer) Open(value string) string {
	out, _ := s.OpenStatus(value)
	return out
}

// OpenStatus reports whether any protected receipt could not be opened.
func (s Sealer) OpenStatus(value string) (string, bool) {
	unavailable := false
	out := ciphertext.ReplaceAllStringFunc(value, func(token string) string {
		sealed, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(token, "prt_sealed_v1_"))
		if err != nil || s.aead == nil || len(sealed) < s.aead.NonceSize() {
			unavailable = true
			return "[resume receipt unavailable after restart]"
		}
		nonce := sealed[:s.aead.NonceSize()]
		plain, err := s.aead.Open(nil, nonce, sealed[s.aead.NonceSize():], nil)
		if err != nil {
			unavailable = true
			return "[resume receipt unavailable after restart]"
		}
		return string(plain)
	})
	return out, unavailable
}
