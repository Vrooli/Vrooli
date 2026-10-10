package effortauthority

import "context"

// FinitePurposeCustody is a credential-owner capability. It signs only typed
// finite initial-run proofs for an exact installed policy, never arbitrary
// bytes. CheckPolicy must read current purpose eligibility/revocation.
// Implementations retain private material entirely inside that owner.
type FinitePurposeCustody interface {
	CheckPolicy(context.Context, Policy) error
	SignPolicyProof(context.Context, Policy, Proof) (Proof, error)
}

// CheckFiniteSigningIntent lets the credential owner validate the exact shared
// carrier without copying the admission protocol or accepting a wider effect.
func CheckFiniteSigningIntent(p Policy, i Intent) error {
	if p.Validate() != nil || i.Effect != "run.create" {
		return ErrRefused
	}
	return checkIntent(p, i)
}

func (s *CustodySigner) CheckPolicy(ctx context.Context, p Policy) error {
	if s == nil || ctx.Err() != nil || p.Validate() != nil {
		return ErrRefused
	}
	if s.Purpose != nil {
		return s.Purpose.CheckPolicy(ctx, p)
	}
	// Legacy handles remain supported only by explicit lower-level fixtures;
	// prepared issuer startup requires the dedicated owner capability.
	key := s.Clients[p.Client]
	if key == nil || Digest(key.Public()) != Digest(p.ClientKey) {
		return ErrRefused
	}
	return nil
}
func (s *CustodySigner) SignPolicyProof(ctx context.Context, p Policy, proof Proof) (Proof, error) {
	if s.CheckPolicy(ctx, p) != nil || CheckFiniteSigningIntent(p, proof.Intent) != nil {
		return Proof{}, ErrRefused
	}
	if s.Purpose != nil {
		return s.Purpose.SignPolicyProof(ctx, p, proof)
	}
	return s.SignEffort(ctx, p.Client, proof)
}
