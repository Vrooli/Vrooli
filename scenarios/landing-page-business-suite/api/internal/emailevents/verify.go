// Package emailevents contains the provider-neutral sign-in delivery event
// model and the SendGrid signed webhook verifier.
package emailevents

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"
)

const DefaultMaxAge = 10 * time.Minute

var (
	ErrMissingSignature = errors.New("missing SendGrid webhook signature")
	ErrMissingTimestamp = errors.New("missing SendGrid webhook timestamp")
	ErrStaleSignature   = errors.New("stale SendGrid webhook signature")
	ErrInvalidSignature = errors.New("invalid SendGrid webhook signature")
)

type ecdsaSignature struct {
	R, S *big.Int
}

// Verify validates a SendGrid Signed Event Webhook request. publicKey is the
// base64-encoded DER SubjectPublicKeyInfo copied from SendGrid. payload must
// be the exact bytes received on the wire.
func Verify(publicKey, signature, timestamp string, payload []byte, now time.Time) error {
	return VerifyWithMaxAge(publicKey, signature, timestamp, payload, now, DefaultMaxAge)
}

func VerifyWithMaxAge(publicKey, signature, timestamp string, payload []byte, now time.Time, maxAge time.Duration) error {
	if strings.TrimSpace(signature) == "" {
		return ErrMissingSignature
	}
	if strings.TrimSpace(timestamp) == "" {
		return ErrMissingTimestamp
	}
	seconds, err := strconv.ParseInt(strings.TrimSpace(timestamp), 10, 64)
	if err != nil {
		return fmt.Errorf("%w: invalid timestamp", ErrInvalidSignature)
	}
	when := time.Unix(seconds, 0)
	if maxAge <= 0 {
		maxAge = DefaultMaxAge
	}
	if delta := now.Sub(when); delta < -maxAge || delta > maxAge {
		return ErrStaleSignature
	}

	derKey, err := base64.StdEncoding.DecodeString(strings.TrimSpace(publicKey))
	if err != nil {
		return fmt.Errorf("%w: public key encoding", ErrInvalidSignature)
	}
	parsed, err := x509.ParsePKIXPublicKey(derKey)
	if err != nil {
		return fmt.Errorf("%w: public key DER", ErrInvalidSignature)
	}
	key, ok := parsed.(*ecdsa.PublicKey)
	if !ok || key.Curve == nil {
		return fmt.Errorf("%w: public key is not ECDSA", ErrInvalidSignature)
	}

	signatureDER, err := base64.StdEncoding.DecodeString(strings.TrimSpace(signature))
	if err != nil {
		return fmt.Errorf("%w: signature encoding", ErrInvalidSignature)
	}
	var sig ecdsaSignature
	if rest, err := asn1.Unmarshal(signatureDER, &sig); err != nil || len(rest) != 0 || sig.R == nil || sig.S == nil {
		return fmt.Errorf("%w: signature DER", ErrInvalidSignature)
	}
	digest := sha256.Sum256(append([]byte(timestamp), payload...))
	if !ecdsa.Verify(key, digest[:], sig.R, sig.S) {
		return ErrInvalidSignature
	}
	return nil
}
