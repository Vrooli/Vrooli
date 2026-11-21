package administration

import (
	"database/sql"
	"errors"
	"strings"
	"sync"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

type witnessAuthorityStore struct {
	mu    sync.Mutex
	value string
}

func (s *witnessAuthorityStore) Put(_, _ string, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.value = value
	return nil
}

func (s *witnessAuthorityStore) Get(_, _ string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.value == "" {
		return "", credentialauthority.ErrNotFound
	}
	return s.value, nil
}

func (s *witnessAuthorityStore) Delete(_, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.value = ""
	return nil
}

func newCredentialWitnessTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE credential_mint_witness (
		logical_id TEXT NOT NULL,
		field TEXT NOT NULL,
		minted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (logical_id, field)
	)`); err != nil {
		db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestCredentialWitnessRecordsAndReadsOnlyMintMetadata(t *testing.T) {
	db := newCredentialWitnessTestDB(t)
	witness := NewCredentialWitness(db)
	identity, err := credentialauthority.ParseIdentity("vrooli/landing-page-business-suite")
	if err != nil {
		t.Fatal(err)
	}

	minted, err := witness.Minted(identity, "service-secret")
	if err != nil || minted {
		t.Fatalf("initial Minted() = %v, %v; want false, nil", minted, err)
	}
	if err := witness.RecordMint(identity, "service-secret"); err != nil {
		t.Fatal(err)
	}
	minted, err = witness.Minted(identity, "service-secret")
	if err != nil || !minted {
		t.Fatalf("recorded Minted() = %v, %v; want true, nil", minted, err)
	}
	if err := witness.RecordMint(identity, "service-secret"); err != nil {
		t.Fatalf("duplicate RecordMint() must be idempotent: %v", err)
	}

	var columns string
	if err := db.QueryRow(`SELECT group_concat(name, ',') FROM pragma_table_info('credential_mint_witness')`).Scan(&columns); err != nil {
		t.Fatal(err)
	}
	if columns != "logical_id,field,minted_at" {
		t.Fatalf("witness table columns = %q; credential values must not be stored", columns)
	}
}

func TestCredentialWitnessPropagatesDatabaseErrors(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	witness := NewCredentialWitness(db)
	identity, err := credentialauthority.ParseIdentity("vrooli/landing-page-business-suite")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := witness.Minted(identity, "field"); err == nil {
		t.Fatal("Minted() must propagate a witness-store error")
	}
	if err := witness.RecordMint(identity, "field"); err == nil {
		t.Fatal("RecordMint() must propagate a witness-store error")
	}
}

func TestCredentialWitnessRefusesRemintAfterStoreLoss(t *testing.T) {
	db := newCredentialWitnessTestDB(t)
	witness := NewCredentialWitness(db)
	identity, err := credentialauthority.ParseIdentity("vrooli/landing-page-business-suite")
	if err != nil {
		t.Fatal(err)
	}
	store := &witnessAuthorityStore{}
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	mints := 0
	if _, err := authority.ResolveOrMint(identity, "service-secret", witness, func() (string, error) {
		mints++
		return "minted-once", nil
	}); err != nil {
		t.Fatalf("first resolve-or-mint: %v", err)
	}
	if mints != 1 {
		t.Fatalf("mint count after virgin start = %d, want 1", mints)
	}
	if err := store.Delete(string(identity), "service-secret"); err != nil {
		t.Fatal(err)
	}
	_, err = authority.ResolveOrMint(identity, "service-secret", witness, func() (string, error) {
		mints++
		return "replacement", nil
	})
	var refused *credentialauthority.MintRefusedError
	if !errors.As(err, &refused) {
		t.Fatalf("second resolve-or-mint error = %v, want MintRefusedError", err)
	}
	if mints != 1 {
		t.Fatalf("mint count after store loss = %d, want 1", mints)
	}
	if !strings.Contains(err.Error(), "vrooli credentials") || !strings.Contains(err.Error(), "--accept-credential-loss") {
		t.Fatalf("refusal error lacks recovery guidance: %v", err)
	}
}
