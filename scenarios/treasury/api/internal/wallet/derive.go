package wallet

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	secp "github.com/go-webauthn/x/crypto/secp256k1"
)

// CreateDerived gives a budget a deterministic, domain-separated child key.
// The parent and child stay in the credential authority. The derivation label
// is public metadata, never a seed or an authority granted to an agent.
func (s *Service) CreateDerived(ctx context.Context, parentID, label string, in CreateInput) (Wallet, error) {
	if strings.TrimSpace(label) == "" || len(label) > 256 || strings.ContainsAny(label, "\r\n") || parentID == in.ID {
		return Wallet{}, ErrInvalid
	}
	parent, err := s.Get(ctx, parentID, in.Network, in.OperatorIdentity)
	if err != nil {
		return Wallet{}, err
	}
	if parent.BookID != in.BookID {
		return Wallet{}, ErrMismatch
	}
	key, err := s.key(parent)
	if err != nil {
		return Wallet{}, err
	}
	defer key.Zero()
	seed := key.Serialize()
	defer clear(seed)
	return s.create(ctx, in, func() (*secp.PrivateKey, error) {
		for counter := 0; counter < 256; counter++ {
			payload, _ := json.Marshal([]any{"vrooli-treasury-budget-key-v1", parent.ID, in.BookID, in.Network, label, counter})
			mac := hmac.New(sha256.New, seed)
			_, _ = mac.Write(payload)
			raw := mac.Sum(nil)
			var scalar secp.ModNScalar
			valid := !scalar.SetByteSlice(raw) && !scalar.IsZero()
			scalar.Zero()
			if valid {
				child := secp.PrivKeyFromBytes(raw)
				clear(raw)
				return child, nil
			}
			clear(raw)
		}
		return nil, fmt.Errorf("%w: child derivation exhausted", ErrUnavailable)
	})
}
