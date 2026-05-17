package byokstore

import (
	"context"
	"time"

	"audio-tools/internal/clock"
	"audio-tools/internal/store"
)

// Store wraps the persistent BYOK store with encryption and
// fingerprint redaction. Handlers and provider chains depend on this
// surface — never on the raw store.BYOKStore.
type Store struct {
	enc  *Encryptor
	repo *store.BYOKStore
	clk  clock.Clock
}

// New wires the encryptor + persistence into a single facade using the
// system clock.
func New(enc *Encryptor, repo *store.BYOKStore) *Store {
	return &Store{enc: enc, repo: repo, clk: clock.System{}}
}

// NewWithClock is the clock-injected constructor used by tests for
// deterministic CreatedAt / LastUsedAt timestamps.
func NewWithClock(enc *Encryptor, repo *store.BYOKStore, clk clock.Clock) *Store {
	if clk == nil {
		clk = clock.System{}
	}
	return &Store{enc: enc, repo: repo, clk: clk}
}

func (s *Store) now() time.Time {
	if s.clk == nil {
		return clock.System{}.Now().UTC()
	}
	return s.clk.Now().UTC()
}

// Credential is the redacted view returned by List/Upsert. The plaintext
// secret is intentionally absent; use Get to fetch it for chain dispatch.
type Credential struct {
	ProviderID  string
	Capability  string
	Fingerprint string
	CreatedAt   time.Time
	LastUsedAt  *time.Time
}

// Upsert encrypts the plaintext and persists it; returns the redacted view.
func (s *Store) Upsert(ctx context.Context, providerID, capability, secret string) (Credential, error) {
	ct, err := s.enc.Seal([]byte(secret))
	if err != nil {
		return Credential{}, err
	}
	now := s.now()
	if err := s.repo.Upsert(ctx, store.BYOKCredential{
		ProviderID: providerID, Capability: capability,
		Cipher: ct, Fingerprint: Fingerprint(secret), CreatedAt: now,
	}); err != nil {
		return Credential{}, err
	}
	return Credential{
		ProviderID:  providerID,
		Capability:  capability,
		Fingerprint: Fingerprint(secret),
		CreatedAt:   now,
	}, nil
}

// Delete removes the credential. Returns (false, nil) when not found.
func (s *Store) Delete(ctx context.Context, providerID, capability string) (bool, error) {
	return s.repo.Delete(ctx, providerID, capability)
}

// List returns redacted credential summaries.
func (s *Store) List(ctx context.Context) ([]Credential, error) {
	rows, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Credential, 0, len(rows))
	for _, r := range rows {
		out = append(out, Credential{
			ProviderID:  r.ProviderID,
			Capability:  r.Capability,
			Fingerprint: r.Fingerprint,
			CreatedAt:   r.CreatedAt,
			LastUsedAt:  r.LastUsedAt,
		})
	}
	return out, nil
}

// Get returns the decrypted plaintext secret for chain dispatch.
// (secret, true, nil) on hit; ("", false, nil) on miss.
func (s *Store) Get(ctx context.Context, providerID, capability string) (string, bool, error) {
	c, ok, err := s.repo.Get(ctx, providerID, capability)
	if err != nil || !ok {
		return "", ok, err
	}
	pt, err := s.enc.Open(c.Cipher)
	if err != nil {
		return "", false, err
	}
	// Best-effort touch; we don't block on it.
	_ = s.repo.MarkUsed(ctx, providerID, capability, s.now())
	return string(pt), true, nil
}
