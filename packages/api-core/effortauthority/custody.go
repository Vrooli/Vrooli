package effortauthority

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
)

// CustodySigner receives already provisioned signing handles from the owning
// credential authority. It never generates, enrolls, exports or persists keys.
// Private handles must not be placed in JSON/configuration/queue fields.
type CustodySigner struct {
	Clients map[string]crypto.Signer
	Purpose FinitePurposeCustody
}

func (s *CustodySigner) SignEffort(ctx context.Context, client string, p Proof) (Proof, error) {
	if s == nil || ctx.Err() != nil || p.Intent.Client != client {
		return Proof{}, ErrRefused
	}
	key := s.Clients[client]
	if key == nil {
		return Proof{}, ErrRefused
	}
	if _, ok := key.Public().(ed25519.PublicKey); !ok {
		return Proof{}, ErrRefused
	}
	sig, e := key.Sign(rand.Reader, unsigned(p), crypto.Hash(0))
	if e != nil {
		return Proof{}, ErrRefused
	}
	p.Signature = sig
	return p, nil
}
