package walletrecovery

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/api-core/receiptsigning"
	_ "modernc.org/sqlite"
)

type evidenceStub struct{ e Evidence }

func (s evidenceStub) ReadWalletRecoveryEvidence(context.Context, string, string, string) (Evidence, error) {
	return s.e, nil
}

type signerStub struct{ calls int }

func (s *signerStub) Sign(_ context.Context, purpose receiptsigning.Purpose, b []byte) (receiptsigning.SignatureEnvelope, error) {
	s.calls++
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	return receiptsigning.SignatureEnvelope{Version: receiptsigning.EnvelopeVersionV1, Purpose: purpose, Algorithm: receiptsigning.AlgorithmCredentialAuthorityEd25519, KeyID: "test-ca-key", Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(priv, b)), Digest: receiptsigning.Digest(b)}, nil
}

type failingSigner struct{}

func (failingSigner) Sign(context.Context, receiptsigning.Purpose, []byte) (receiptsigning.SignatureEnvelope, error) {
	return receiptsigning.SignatureEnvelope{}, errors.New("fixture signer unavailable")
}

func TestAttestFailsClosedWithoutApprovedSigner(t *testing.T) {
	db := openDB(t)
	store := SQLiteReservations{DB: db}
	if err := store.EnsureSchema(context.Background()); err != nil {
		t.Fatal(err)
	}
	s := Service{Evidence: evidenceStub{validEvidence()}, Reservations: store}
	if _, err := s.Attest(context.Background(), testReservationID(), "target-1", "dest-1"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("want signer fail-closed, got %v", err)
	}
}

func TestAttestBindsEvidenceAndReservesExactSnapshot(t *testing.T) {
	db := openDB(t)
	store := SQLiteReservations{DB: db}
	if err := store.EnsureSchema(context.Background()); err != nil {
		t.Fatal(err)
	}
	signer := &signerStub{}
	now := time.Now().UTC().Truncate(time.Nanosecond)
	s := Service{Evidence: evidenceStub{validEvidence()}, Reservations: store, Signer: signer, Now: func() time.Time { return now }}
	got, err := s.Attest(context.Background(), testReservationID(), "target-1", "dest-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Receipt.SnapshotID != "snapshot-7" || got.Receipt.TombstoneSHA256 != validEvidence().RestoredTombstoneSHA256 || got.Attestation.KeyID != "test-ca-key" || got.Receipt.ReservationExpiresAt != now.Add(5*time.Minute).Format(time.RFC3339Nano) {
		t.Fatalf("unexpected receipt: %+v", got)
	}
	// Attest pins the snapshot before returning any signature.
	if _, err := store.BeginMutation(context.Background(), "target-1", "dest-1"); !errors.Is(err, ErrReserved) {
		t.Fatalf("expected snapshot mutation fence, got %v", err)
	}
	if err := store.Commit(context.Background(), testReservationID(), "target-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.BeginMutation(context.Background(), "target-1", "dest-1"); !errors.Is(err, ErrReserved) {
		t.Fatalf("commit-pending pin must survive until Treasury confirms, got %v", err)
	}
	if err := store.Commit(context.Background(), testReservationID(), "target-1"); err != nil {
		t.Fatalf("commit-pending retry must be idempotent: %v", err)
	}
	if err := store.Release(context.Background(), testReservationID(), "target-1"); !errors.Is(err, ErrCommitAcknowledgementRequired) {
		t.Fatalf("commit-pending pin must not be releasable without a verified Treasury ack: %v", err)
	}
	if _, err := store.BeginMutation(context.Background(), "target-1", "dest-1"); !errors.Is(err, ErrReserved) {
		t.Fatalf("missing Treasury ack must leave commit-pending pin intact: %v", err)
	}
}

func TestAttestPinsBeforeSigningAndRetriesSameArtifactAfterSignerFailure(t *testing.T) {
	ctx := context.Background()
	db := openDB(t)
	store := SQLiteReservations{DB: db}
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatal(err)
	}
	fixedNow := time.Now().UTC().Truncate(time.Nanosecond)
	s := Service{Evidence: evidenceStub{validEvidence()}, Reservations: store, Signer: failingSigner{}, Now: func() time.Time { return fixedNow }}
	if _, err := s.Attest(ctx, testReservationID(), "target-1", "dest-1"); err == nil {
		t.Fatal("expected signer failure")
	}
	if _, err := store.BeginMutation(ctx, "target-1", "dest-1"); !errors.Is(err, ErrReserved) {
		t.Fatalf("receipt signer failure must not unpin the newest snapshot: %v", err)
	}
	s.Signer = &signerStub{}
	if _, err := s.Attest(ctx, testReservationID(), "target-1", "dest-1"); err != nil {
		t.Fatalf("same exact evidence should resume the existing pin after restart/failure: %v", err)
	}
	if _, err := store.BeginMutation(ctx, "target-1", "dest-1"); !errors.Is(err, ErrReserved) {
		t.Fatalf("issued receipt must still pin the target×destination: %v", err)
	}
	stale := validEvidence()
	stale.SnapshotID = "snapshot-newer"
	stale.RestoreSnapshotID = "snapshot-newer"
	s.Evidence = evidenceStub{stale}
	if _, err := s.Attest(ctx, testReservationID(), "target-1", "dest-1"); !errors.Is(err, ErrReserved) {
		t.Fatalf("committed pin must refuse replacing the attested snapshot: %v", err)
	}
}

func TestReservationRejectsAlreadyInFlightMutationAndDifferentRetry(t *testing.T) {
	db := openDB(t)
	store := SQLiteReservations{DB: db}
	if err := store.EnsureSchema(context.Background()); err != nil {
		t.Fatal(err)
	}
	release, err := store.BeginMutation(context.Background(), "target-1", "dest-1")
	if err != nil {
		t.Fatal(err)
	}
	r := Reservation{RequestID: "req-1", TargetID: "target-1", DestinationID: "dest-1", SnapshotID: "snapshot-7", TombstoneSHA256: validEvidence().RestoredTombstoneSHA256, ExpiresAt: time.Now().Add(time.Minute)}
	if err := store.Reserve(context.Background(), r, time.Now()); err == nil {
		t.Fatal("expected in-flight mutation to prevent reservation")
	}
	release()
	r.SnapshotGeneration = 1
	if err := store.Reserve(context.Background(), r, time.Now()); err != nil {
		t.Fatal(err)
	}
	other := r
	other.RequestID = "req-2"
	other.SnapshotID = "snapshot-8"
	if err := store.Reserve(context.Background(), other, time.Now()); !errors.Is(err, ErrReserved) {
		t.Fatalf("expected conflicting reservation to fail, got %v", err)
	}
	changedArtifact := r
	changedArtifact.TombstoneSHA256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err := store.Reserve(context.Background(), changedArtifact, time.Now()); !errors.Is(err, ErrReserved) {
		t.Fatalf("same id cannot substitute another tombstone: %v", err)
	}
}

func TestCommitPendingReservationSurvivesRestartAndRejectsOtherRequest(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "persistent.db")
	open := func() *sql.DB {
		db, err := sql.Open("sqlite", path)
		if err != nil {
			t.Fatal(err)
		}
		db.SetMaxOpenConns(1)
		return db
	}
	db := open()
	store := SQLiteReservations{DB: db}
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	r := Reservation{RequestID: "stable", TargetID: "target", DestinationID: "dest", SnapshotID: "snapshot", TombstoneSHA256: validEvidence().RestoredTombstoneSHA256, ExpiresAt: now.Add(time.Minute)}
	if err := store.Reserve(ctx, r, now); err != nil {
		t.Fatal(err)
	}
	if err := store.Commit(ctx, "stable", "target"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db = open()
	defer db.Close()
	store = SQLiteReservations{DB: db}
	if _, err := db.Exec(`UPDATE dbm_wallet_recovery_reservations SET expires_at=? WHERE request_id=?`, time.Now().UTC().Add(-time.Second).Format(time.RFC3339Nano), "stable"); err != nil {
		t.Fatal(err)
	}
	if err := store.Commit(ctx, "stable", "target"); err != nil {
		t.Fatalf("commit-pending acknowledgment must survive restart idempotently: %v", err)
	}
	if _, err := store.BeginMutation(ctx, "target", "dest"); !errors.Is(err, ErrReserved) {
		t.Fatalf("commit-pending must remain pinned after restart and expiry, got %v", err)
	}
	other := r
	other.RequestID = "other"
	other.ExpiresAt = time.Now().Add(time.Minute)
	if err := store.Reserve(ctx, other, time.Now()); !errors.Is(err, ErrReserved) {
		t.Fatalf("different request must not replace commit-pending pin: %v", err)
	}
	// The same intent can renew the same pinned snapshot and later produce a
	// fresh unexpired signed receipt after a restart.
	r.ExpiresAt = time.Now().Add(time.Minute)
	if err := store.Reserve(ctx, r, time.Now()); err != nil {
		t.Fatalf("same intent should renew its pinned snapshot: %v", err)
	}
	if err := store.Release(ctx, "stable", "target"); !errors.Is(err, ErrCommitAcknowledgementRequired) {
		t.Fatalf("release must require a Treasury commit acknowledgment: %v", err)
	}
	if _, err := store.BeginMutation(ctx, "target", "dest"); !errors.Is(err, ErrReserved) {
		t.Fatalf("restart retry must preserve the pin until a verified Treasury ack: %v", err)
	}
}

func TestAttestRejectsIncompleteOrStaleEvidence(t *testing.T) {
	for name, mutate := range map[string]func(*Evidence){"scheduled": func(e *Evidence) { e.RunTrigger = "scheduled" }, "wrong pair": func(e *Evidence) { e.OutcomeDestinationID = "other" }, "newer snapshot": func(e *Evidence) { e.SnapshotNewest = false }, "incomplete restore": func(e *Evidence) { e.RestoreStatus = "running" }, "bundle remains": func(e *Evidence) { e.WalletBundlePresent = true }} {
		t.Run(name, func(t *testing.T) {
			db := openDB(t)
			store := SQLiteReservations{DB: db}
			_ = store.EnsureSchema(context.Background())
			ev := validEvidence()
			mutate(&ev)
			signer := &signerStub{}
			s := Service{Evidence: evidenceStub{ev}, Reservations: store, Signer: signer}
			if _, err := s.Attest(context.Background(), testReservationID(), "target-1", "dest-1"); err == nil {
				t.Fatal("expected evidence to be rejected")
			}
			if signer.calls != 0 {
				t.Fatal("invalid evidence reached signer")
			}
		})
	}
}

func validEvidence() Evidence {
	id := testReservationID()
	return Evidence{RequestID: id, TargetID: "target-1", DestinationID: "dest-1", WalletID: "wallet-1", Network: "base", Address: "0xabc", SuccessorID: "successor-1", DestinationBackend: "sftp", PlanID: "plan-1", TargetName: "treasury-wallet", RunID: "manual-run-1", RunTrigger: "manual", RunStatus: "succeeded", OutcomeTargetID: "target-1", OutcomeDestinationID: "dest-1", OutcomeStatus: "succeeded", SnapshotID: "snapshot-7", SnapshotNewest: true, SnapshotImmutable: true, SnapshotGeneration: 0, RestoreID: "restore-1", RestoreStatus: "verified", RestoreSnapshotID: "snapshot-7", RestoredTombstoneSHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", RestoredRevocationSHA256: "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789", DBMVerifyPassed: true, RestoredTombstoneMatches: true}
}
func testReservationID() string { return ReservationID("wallet-1", "successor-1", "dest-1") }
func openDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "wallet-recovery.db"))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}
