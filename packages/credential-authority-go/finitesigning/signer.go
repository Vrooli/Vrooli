// Package finitesigning owns dedicated finite-proof custody over the governed
// credential authority. It cannot provision, rotate, resolve or export a key.
package finitesigning

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/vrooli/api-core/effortauthority"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

const Purpose = "vrooli.finite-effort-proof.v1"
const Field = "finite-purpose-v1"

// Descriptor contains only the exact pre-enrolled public purpose identity.
// Purpose eligibility is read from the credential owner on every operation.
type Descriptor struct {
	Identity  credentialauthority.Identity
	HandleID  string
	PublicKey []byte
	Policy    effortauthority.Policy
}
type Config struct {
	Authority  *credentialauthority.Authority
	Descriptor Descriptor
}
type purposeStore interface {
	Require(credentialauthority.Identity, string) (string, error)
}

// Signer is opaque and deliberately does not implement crypto.Signer or expose
// Public/Sign(raw bytes). Its only capability is exact typed finite proof signing.
type Signer struct {
	store      purposeStore
	descriptor Descriptor
	now        func() time.Time
}

// provisionedRecord is an owner-only schema for an already enrolled dedicated
// handle. It is not a receipt key-ring and is never returned to consumers.
// No writer/enrollment path is provided by this disabled source integration.
type provisionedRecord struct {
	Version      int    `json:"version"`
	Purpose      string `json:"purpose"`
	HandleID     string `json:"handleId"`
	Client       string `json:"client"`
	PolicyDigest string `json:"policyDigest"`
	PublicKey    string `json:"publicKey"`
	PrivateKey   string `json:"privateKey"`
	Revoked      bool   `json:"revoked"`
}

func New(c Config) (*Signer, error) {
	if c.Authority == nil {
		return nil, effortauthority.ErrRefused
	}
	return newSigner(c.Authority, c.Descriptor)
}
func newSigner(store purposeStore, d Descriptor) (*Signer, error) {
	id, e := credentialauthority.ParseIdentity(string(d.Identity))
	if e != nil || id != d.Identity || !strings.HasPrefix(string(id), "finite-effort/") || d.HandleID == "" || store == nil || d.Policy.Validate() != nil || !slices.Equal(d.PublicKey, d.Policy.ClientKey) {
		return nil, effortauthority.ErrRefused
	}
	raw, e := json.Marshal(d)
	if e != nil {
		return nil, effortauthority.ErrRefused
	}
	var frozen Descriptor
	if json.Unmarshal(raw, &frozen) != nil {
		return nil, effortauthority.ErrRefused
	}
	return &Signer{store: store, descriptor: frozen, now: time.Now}, nil
}
func (s *Signer) load(ctx context.Context, p effortauthority.Policy) (provisionedRecord, error) {
	if s == nil || ctx.Err() != nil || effortauthority.Digest(p) != effortauthority.Digest(s.descriptor.Policy) || p.Validate() != nil || s.now().Before(p.NotBefore) || !s.now().Before(p.Deadline) {
		return provisionedRecord{}, effortauthority.ErrRefused
	}
	raw, e := s.store.Require(s.descriptor.Identity, Field)
	if e != nil || len(raw) > 4096 {
		return provisionedRecord{}, effortauthority.ErrRefused
	}
	var r provisionedRecord
	dec := json.NewDecoder(bytes.NewBufferString(raw))
	dec.DisallowUnknownFields()
	if dec.Decode(&r) != nil || dec.Decode(new(any)) != io.EOF || r.Version != 1 || r.Revoked || r.Purpose != Purpose || r.HandleID != s.descriptor.HandleID || r.Client != p.Client || r.PolicyDigest != effortauthority.Digest(p) {
		return provisionedRecord{}, effortauthority.ErrRefused
	}
	pub, e := base64.RawStdEncoding.DecodeString(r.PublicKey)
	if e != nil || !slices.Equal(pub, s.descriptor.PublicKey) {
		return provisionedRecord{}, effortauthority.ErrRefused
	}
	// Validate both seed and cached public bytes before publishing eligibility.
	private, err := base64.RawStdEncoding.DecodeString(r.PrivateKey)
	if err != nil || len(private) != ed25519.PrivateKeySize {
		return provisionedRecord{}, effortauthority.ErrRefused
	}
	defer clear(private)
	derived := ed25519.NewKeyFromSeed(private[:ed25519.SeedSize])
	defer clear(derived)
	if !slices.Equal(derived, private) || !slices.Equal(derived.Public().(ed25519.PublicKey), pub) {
		return provisionedRecord{}, effortauthority.ErrRefused
	}
	return r, nil
}
func (s *Signer) CheckPolicy(ctx context.Context, p effortauthority.Policy) error {
	_, e := s.load(ctx, p)
	return e
}
func (s *Signer) SignPolicyProof(ctx context.Context, p effortauthority.Policy, offered effortauthority.Proof) (effortauthority.Proof, error) {
	// Refuse offered arbitrary/wider carriers before any credential read.
	if effortauthority.CheckFiniteSigningIntent(p, offered.Intent) != nil || ctx.Err() != nil {
		return effortauthority.Proof{}, effortauthority.ErrRefused
	}
	r, e := s.load(ctx, p)
	if e != nil {
		return effortauthority.Proof{}, e
	}
	raw, e := base64.RawStdEncoding.DecodeString(r.PrivateKey)
	if e != nil || len(raw) != ed25519.PrivateKeySize {
		return effortauthority.Proof{}, effortauthority.ErrRefused
	}
	defer clear(raw)
	key := ed25519.PrivateKey(raw)
	if !slices.Equal(key.Public().(ed25519.PublicKey), p.ClientKey) {
		return effortauthority.Proof{}, effortauthority.ErrRefused
	}
	nonce := make([]byte, 32)
	if _, e = rand.Read(nonce); e != nil || ctx.Err() != nil {
		return effortauthority.Proof{}, effortauthority.ErrRefused
	}
	// The signed protocol already carries the fixed finite audience. No receipt
	// envelope, arbitrary payload, offered timestamp/nonce/signature is used; those fields are ignored.
	proof := effortauthority.Proof{Intent: offered.Intent, Nonce: hex.EncodeToString(nonce), IssuedAt: s.now().UTC()}
	if proof.IssuedAt.Before(p.NotBefore) || !proof.IssuedAt.Before(p.Deadline) {
		return effortauthority.Proof{}, effortauthority.ErrRefused
	}
	return effortauthority.Sign(proof, key)
}

var _ effortauthority.FinitePurposeCustody = (*Signer)(nil)
