package mandate

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
)

// HMACSigner signs the canonical immutable mandate payload. Production passes
// key material resolved from secrets-manager; the key is never persisted.
type HMACSigner struct {
	key []byte
}

func NewHMACSigner(key []byte) (*HMACSigner, error) {
	if len(key) == 0 {
		return nil, fmt.Errorf("%w: signing key is required", ErrInvalid)
	}
	return &HMACSigner{key: append([]byte(nil), key...)}, nil
}

func (s *HMACSigner) Sign(_ context.Context, payload []byte) ([]byte, error) {
	mac := hmac.New(sha256.New, s.key)
	_, _ = mac.Write(payload)
	return mac.Sum(nil), nil
}

var _ Signer = (*HMACSigner)(nil)
