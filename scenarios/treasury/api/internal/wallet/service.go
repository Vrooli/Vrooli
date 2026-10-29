// Package wallet owns receive-only address custody. It cannot authorize or send funds.
package wallet

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	secp "github.com/go-webauthn/x/crypto/secp256k1"
	"github.com/go-webauthn/x/crypto/secp256k1/ecdsa"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"golang.org/x/crypto/sha3"
	"treasury/internal/book"
)

var (
	ErrInvalid     = errors.New("invalid research wallet")
	ErrUnavailable = errors.New("research wallet custody unavailable")
	ErrMismatch    = errors.New("research wallet custody or scope mismatch")
	ErrNotFound    = errors.New("research wallet not found")
)

//go:embed schema.sql
var schemaSQL string

func Schema() string { return schemaSQL }

const (
	keyField      = "secp256k1-scalar"
	recoveryField = "recovery-passphrase"
)

var (
	validID    = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,80}$`)
	validNonce = regexp.MustCompile(`^[a-zA-Z0-9_-]{16,128}$`)
)

type Wallet struct {
	ID, BookID, OperatorIdentity, Network, Address, PublicKey, CustodyReference string
	CreatedAt                                                                   time.Time
	RecoveryStatus, RecoverySHA256                                              string
}
type (
	CreateInput struct{ ID, BookID, Network, OperatorIdentity string }
	Proof       struct {
		Wallet                     Wallet
		Message, Signature, Scheme string
		Verified                   bool
	}
)
type Books interface {
	Get(context.Context, string) (book.Book, error)
}
type Custody interface {
	Recheck()
	Resolve(credentialauthority.Identity, string) (string, error)
	ResolveOrMint(credentialauthority.Identity, string, credentialauthority.Witness, func() (string, error)) (string, error)
	ExportRecovery([]credentialauthority.RecoveryEntry, string) ([]byte, error)
	RestoreRecovery([]byte, string) error
}
type (
	RecoveryPath func(context.Context, string) (string, error)
	Service      struct {
		repository   Repository
		books        Books
		custody      Custody
		clock        schedule.Clock
		recoveryPath RecoveryPath
	}
)

func NewService(repository Repository, books Books, custody Custody, clock schedule.Clock, path RecoveryPath) *Service {
	return &Service{repository: repository, books: books, custody: custody, clock: clock, recoveryPath: path}
}

func (s *Service) available(ctx context.Context) error {
	if database.IsTestMode(ctx) {
		return fmt.Errorf("%w: test-mode live credential access is refused", ErrUnavailable)
	}
	if s == nil || s.repository == nil || s.books == nil || s.custody == nil || s.clock == nil {
		return ErrUnavailable
	}
	s.custody.Recheck()
	return nil
}
func supported(network string) bool { return network == "eip155:8453" || network == "eip155:84532" }
func reference(in CreateInput, namespace string) string {
	encoded, _ := json.Marshal([]string{"treasury-research-wallet-v1", in.ID, in.BookID, in.OperatorIdentity, in.Network})
	sum := sha256.Sum256(encoded)
	return fmt.Sprintf("vrooli/treasury/wallets/%s/%x", namespace, sum[:16])
}

func validReference(w Wallet) bool {
	parts := strings.Split(strings.TrimPrefix(w.CustodyReference, "vrooli/treasury/wallets/"), "/")
	if len(parts) != 2 || len(parts[0]) != 32 {
		return false
	}
	bytes, err := hex.DecodeString(parts[0])
	if err != nil || len(bytes) != 16 {
		return false
	}
	return w.CustodyReference == reference(CreateInput{ID: w.ID, BookID: w.BookID, Network: w.Network, OperatorIdentity: w.OperatorIdentity}, parts[0])
}

func (s *Service) Create(ctx context.Context, in CreateInput) (Wallet, error) {
	if err := s.available(ctx); err != nil {
		return Wallet{}, err
	}
	if !validID.MatchString(in.ID) || !supported(in.Network) || strings.TrimSpace(in.BookID) == "" || len(in.BookID) > 128 || strings.ContainsAny(in.BookID, "\r\n") || in.OperatorIdentity == "" || len(in.OperatorIdentity) > 256 || strings.ContainsAny(in.OperatorIdentity, "\r\n") {
		return Wallet{}, ErrInvalid
	}
	b, err := s.books.Get(ctx, in.BookID)
	if err != nil {
		return Wallet{}, err
	}
	if b.BeneficiaryIdentity != in.OperatorIdentity {
		return Wallet{}, ErrMismatch
	}
	var nonce [16]byte
	if _, err = rand.Read(nonce[:]); err != nil {
		return Wallet{}, err
	}
	candidate := Wallet{ID: in.ID, BookID: in.BookID, OperatorIdentity: in.OperatorIdentity, Network: in.Network, CustodyReference: reference(in, hex.EncodeToString(nonce[:])), CreatedAt: s.clock.Now().UTC()}
	_, err = s.repository.Prepare(ctx, candidate)
	if err != nil {
		return Wallet{}, err
	}
	err = s.repository.WithCustodyLock(ctx, in.ID, func(repository Repository) error {
		w, err := repository.Get(ctx, in.ID)
		if err != nil {
			return err
		}
		if w.BookID != in.BookID || w.Network != in.Network || w.OperatorIdentity != in.OperatorIdentity || !validReference(w) {
			return ErrMismatch
		}
		identity, err := credentialauthority.ParseIdentity(w.CustodyReference)
		if err != nil {
			return ErrMismatch
		}
		_, err = s.custody.ResolveOrMint(identity, keyField, witness{repository: repository, ctx: ctx, id: w.ID, reference: w.CustodyReference}, func() (string, error) {
			key, err := secp.GeneratePrivateKey()
			if err != nil {
				return "", err
			}
			defer key.Zero()
			bytes := key.Serialize()
			defer clear(bytes)
			return hex.EncodeToString(bytes), nil
		})
		if err != nil {
			return fmt.Errorf("%w: %w", ErrUnavailable, err)
		}
		key, err := s.key(w)
		if err != nil {
			return err
		}
		defer key.Zero()
		address := Address(key.PubKey())
		public := hex.EncodeToString(key.PubKey().SerializeUncompressed())
		if w.Address != "" {
			if w.Address != address || w.PublicKey != public {
				return ErrMismatch
			}
			return nil
		}
		return repository.SetPublic(ctx, w.ID, address, public)
	})
	if err != nil {
		return Wallet{}, err
	}
	return s.Get(ctx, in.ID, in.Network, in.OperatorIdentity)
}

func (s *Service) binding(ctx context.Context, id, network, operator string) (Wallet, error) {
	if err := s.available(ctx); err != nil {
		return Wallet{}, err
	}
	w, err := s.repository.Get(ctx, id)
	if err != nil {
		return Wallet{}, err
	}
	if !supported(network) || w.Network != network || w.OperatorIdentity != operator || !validReference(w) {
		return Wallet{}, ErrMismatch
	}
	b, err := s.books.Get(ctx, w.BookID)
	if err != nil {
		return Wallet{}, err
	}
	if b.BeneficiaryIdentity != operator {
		return Wallet{}, ErrMismatch
	}
	return w, nil
}

func (s *Service) Get(ctx context.Context, id, network, operator string) (Wallet, error) {
	w, err := s.binding(ctx, id, network, operator)
	if err != nil {
		return Wallet{}, err
	}
	key, err := s.key(w)
	if err != nil {
		return Wallet{}, err
	}
	defer key.Zero()
	if w.Address == "" || w.Address != Address(key.PubKey()) || w.PublicKey != hex.EncodeToString(key.PubKey().SerializeUncompressed()) {
		return Wallet{}, ErrMismatch
	}
	return w, nil
}

func (s *Service) key(w Wallet) (*secp.PrivateKey, error) {
	id, err := credentialauthority.ParseIdentity(w.CustodyReference)
	if err != nil {
		return nil, ErrMismatch
	}
	raw, err := s.custody.Resolve(id, keyField)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	bytes, err := hex.DecodeString(raw)
	if err != nil || len(bytes) != 32 {
		return nil, ErrMismatch
	}
	defer clear(bytes)
	var scalar secp.ModNScalar
	if scalar.SetByteSlice(bytes) || scalar.IsZero() {
		return nil, ErrMismatch
	}
	scalar.Zero()
	return secp.PrivKeyFromBytes(bytes), nil
}

func Address(public *secp.PublicKey) string {
	h := sha3.NewLegacyKeccak256()
	h.Write(public.SerializeUncompressed()[1:])
	return "0x" + hex.EncodeToString(h.Sum(nil)[12:])
}

func proofMessage(w Wallet, nonce string) string {
	return fmt.Sprintf("Vrooli research wallet address-control proof v1\nNo spending authorization.\nwallet:%s\nbook:%s\noperator:%s\nnetwork:%s\naddress:%s\nnonce:%s", w.ID, w.BookID, w.OperatorIdentity, w.Network, w.Address, nonce)
}

func personalHash(message string) []byte {
	h := sha3.NewLegacyKeccak256()
	fmt.Fprintf(h, "\x19Ethereum Signed Message:\n%d%s", len([]byte(message)), message)
	return h.Sum(nil)
}

func (s *Service) Prove(ctx context.Context, id, network, operator, nonce string) (Proof, error) {
	if !validNonce.MatchString(nonce) {
		return Proof{}, ErrInvalid
	}
	w, err := s.Get(ctx, id, network, operator)
	if err != nil {
		return Proof{}, err
	}
	key, err := s.key(w)
	if err != nil {
		return Proof{}, err
	}
	defer key.Zero()
	message := proofMessage(w, nonce)
	signature := ecdsa.SignCompact(key, personalHash(message), false)
	// Ethereum wire form is R || S || v; compact header is 27+recovery id.
	eth := append(append([]byte{}, signature[1:]...), signature[0])
	if eth[64] > 28 {
		return Proof{}, ErrMismatch
	}
	p := Proof{Wallet: w, Message: message, Signature: "0x" + hex.EncodeToString(eth), Scheme: "eip191-personal-sign-fixed-research-v1"}
	p.Verified = Verify(p, nonce)
	if !p.Verified {
		return Proof{}, ErrMismatch
	}
	return p, nil
}

func Verify(p Proof, nonce string) bool {
	if p.Scheme != "eip191-personal-sign-fixed-research-v1" || !validNonce.MatchString(nonce) || p.Message != proofMessage(p.Wallet, nonce) || !supported(p.Wallet.Network) {
		return false
	}
	bytes, err := hex.DecodeString(strings.TrimPrefix(p.Signature, "0x"))
	if err != nil || len(bytes) != 65 || bytes[64] < 27 || bytes[64] > 28 {
		return false
	}
	compact := append([]byte{bytes[64]}, bytes[:64]...)
	pub, _, err := ecdsa.RecoverCompact(compact, personalHash(p.Message))
	return err == nil && Address(pub) == p.Wallet.Address && hex.EncodeToString(pub.SerializeUncompressed()) == p.Wallet.PublicKey
}

// witness survives separately from credential storage, so key loss never silently mints a replacement.
type witness struct {
	repository    Repository
	ctx           context.Context
	id, reference string
	recovery      bool
}

func (w witness) valid(identity credentialauthority.Identity, field string) bool {
	expected := keyField
	if w.recovery {
		expected = recoveryField
	}
	return string(identity) == w.reference && field == expected
}

func (w witness) Minted(identity credentialauthority.Identity, field string) (bool, error) {
	if !w.valid(identity, field) {
		return false, ErrMismatch
	}
	return w.repository.Minted(w.ctx, w.id, w.recovery)
}

func (w witness) RecordMint(identity credentialauthority.Identity, field string) error {
	if !w.valid(identity, field) {
		return ErrMismatch
	}
	return w.repository.RecordMint(w.ctx, w.id, w.recovery)
}

func backupName(id string) string {
	sum := sha256.Sum256([]byte(id))
	return fmt.Sprintf("wallet-backups/%x.recovery", sum[:16])
}
