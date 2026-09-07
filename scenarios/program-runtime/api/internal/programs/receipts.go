package programs

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"regexp"
	"strings"
)

// Resume receipts are bearer authority. Seal their occurrences before writing
// source, streamed output, or failure text to the program corpus. The key lives
// only in this repository instance, so Wait/Get can return the original live
// receipt while a fresh process cannot recover authority from historical rows.
// Partial streamed tokens are sealed too, not just complete 43-byte secrets.
var resumeReceiptPattern = regexp.MustCompile(`prt_resume_v1_[A-Za-z0-9_-]*`)
var sealedReceiptPattern = regexp.MustCompile(`prt_sealed_v1_[A-Za-z0-9_-]+`)

type receiptSealer struct {
	aead cipher.AEAD
	err  error
}

func newReceiptSealer() receiptSealer {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return receiptSealer{err: err}
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return receiptSealer{err: err}
	}
	aead, err := cipher.NewGCM(block)
	return receiptSealer{aead: aead, err: err}
}

func (s receiptSealer) seal(value string) (string, error) {
	if !resumeReceiptPattern.MatchString(value) {
		return value, nil
	}
	if s.err != nil {
		return "", s.err
	}
	var sealErr error
	out := resumeReceiptPattern.ReplaceAllStringFunc(value, func(token string) string {
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

func (s receiptSealer) open(value string) string {
	return sealedReceiptPattern.ReplaceAllStringFunc(value, func(token string) string {
		sealed, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(token, "prt_sealed_v1_"))
		if err != nil || s.aead == nil || len(sealed) < s.aead.NonceSize() {
			return "[resume receipt unavailable after restart]"
		}
		nonce := sealed[:s.aead.NonceSize()]
		plain, err := s.aead.Open(nil, nonce, sealed[s.aead.NonceSize():], nil)
		if err != nil {
			return "[resume receipt unavailable after restart]"
		}
		return string(plain)
	})
}
