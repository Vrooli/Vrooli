package walletrecovery

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/receiptsigning"
)

type rejectAuth struct{}

func (rejectAuth) AuthorizeWalletRecovery(*http.Request) error { return errors.New("not allowed") }

func TestHTTPHandlerFailsClosedWithoutServiceIdentity(t *testing.T) {
	h := NewHTTPHandler(Service{}, nil)
	r := httptest.NewRequest("POST", "/api/wallet-recovery/attest", strings.NewReader(`{"reservation_id":"x","target_id":"t","destination_id":"d"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 503 {
		t.Fatalf("want fail-closed 503, got %d", w.Code)
	}
}

func TestHTTPHandlerRequiresAuthorizedCaller(t *testing.T) {
	h := NewHTTPHandler(Service{}, rejectAuth{})
	r := httptest.NewRequest("POST", "/api/wallet-recovery/attest", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatalf("want 403, got %d", w.Code)
	}
}

type testAckVerifier struct{}

func (testAckVerifier) Verify(_ context.Context, envelope receiptsigning.SignatureEnvelope, canonical []byte) error {
	if envelope.Purpose != receiptsigning.PurposeWalletRecoveryCommitAcknowledgement || envelope.Digest != receiptsigning.Digest(canonical) {
		return errors.New("wrong signature purpose or digest")
	}
	sum := sha256.Sum256(canonical)
	if envelope.Signature != base64.StdEncoding.EncodeToString(sum[:]) {
		return errors.New("forged test signature")
	}
	return nil
}

func TestReservationReleaseRequiresExactTreasuryCommitAcknowledgement(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name          string
		mutate        func(*receiptsigning.WalletRecoveryCommitAcknowledgement)
		requestTarget string
		forge         bool
		want          int
	}{
		{name: "forged", forge: true, want: http.StatusConflict},
		{name: "stale snapshot", mutate: func(a *receiptsigning.WalletRecoveryCommitAcknowledgement) { a.SnapshotID = "older-snapshot" }, want: http.StatusConflict},
		{name: "wrong target", mutate: func(a *receiptsigning.WalletRecoveryCommitAcknowledgement) { a.TargetID = "other-target" }, requestTarget: "other-target", want: http.StatusConflict},
		{name: "wrong successor", mutate: func(a *receiptsigning.WalletRecoveryCommitAcknowledgement) { a.SuccessorID = "other-successor" }, want: http.StatusConflict},
		{name: "exact commit", want: http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := openDB(t)
			store := SQLiteReservations{DB: db}
			if err := store.EnsureSchema(ctx); err != nil {
				t.Fatal(err)
			}
			r := Reservation{RequestID: "rotation-1", TargetID: "target-1", DestinationID: "drive-1", SnapshotID: "snapshot-7", SuccessorID: "successor-1", TombstoneSHA256: strings.Repeat("a", 64), ExpiresAt: time.Now().UTC().Add(time.Minute)}
			if err := store.Reserve(ctx, r, time.Now().UTC()); err != nil {
				t.Fatal(err)
			}
			if err := store.Commit(ctx, r.RequestID, r.TargetID); err != nil {
				t.Fatal(err)
			}
			ack := receiptsigning.WalletRecoveryCommitAcknowledgement{Version: 1, OperationID: r.RequestID, ReservationID: r.RequestID, TargetID: r.TargetID, DestinationID: r.DestinationID, SnapshotID: r.SnapshotID, TombstoneSHA256: r.TombstoneSHA256, SuccessorID: "successor-1", SuccessorAddress: "0x1234", SuccessorPublicKey: "pubkey-1"}
			if tc.mutate != nil {
				tc.mutate(&ack)
			}
			canonical, err := receiptsigning.CanonicalJSON(ack)
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(canonical)
			signature := base64.StdEncoding.EncodeToString(sum[:])
			if tc.forge {
				signature = "forged"
			}
			signed := struct {
				Acknowledgement receiptsigning.WalletRecoveryCommitAcknowledgement `json:"acknowledgement"`
				Signature       receiptsigning.SignatureEnvelope                   `json:"signature"`
			}{ack, receiptsigning.SignatureEnvelope{Version: receiptsigning.EnvelopeVersionV1, Purpose: receiptsigning.PurposeWalletRecoveryCommitAcknowledgement, Algorithm: receiptsigning.AlgorithmCredentialAuthorityEd25519, KeyID: "treasury-key-v1", Digest: receiptsigning.Digest(canonical), Signature: signature}}
			payload, err := json.Marshal(signed)
			if err != nil {
				t.Fatal(err)
			}
			requestTarget := tc.requestTarget
			if requestTarget == "" {
				requestTarget = r.TargetID
			}
			body, err := json.Marshal(FinishReservationRequest{ReservationID: r.RequestID, TargetID: requestTarget, State: "release", CommitAckJSON: string(payload)})
			if err != nil {
				t.Fatal(err)
			}
			h := NewReservationFinishHandler(store, nil, testAckVerifier{})
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/wallet-recovery/reservation/finish", strings.NewReader(string(body))))
			if w.Code != tc.want {
				t.Fatalf("release status=%d, want %d: %s", w.Code, tc.want, w.Body.String())
			}
			_, mutationErr := store.BeginMutation(ctx, r.TargetID, r.DestinationID)
			if tc.want == http.StatusOK {
				if mutationErr != nil {
					t.Fatalf("verified acknowledgement should release pin, got %v", mutationErr)
				}
				if err := store.ReleaseAcknowledged(ctx, ack, receiptsigning.Digest(canonical)); err != nil {
					t.Fatalf("identical acknowledgement retry must be idempotent: %v", err)
				}
				changed := ack
				changed.SuccessorAddress = "0xchanged"
				changedBytes, err := receiptsigning.CanonicalJSON(changed)
				if err != nil {
					t.Fatal(err)
				}
				if err := store.ReleaseAcknowledged(ctx, changed, receiptsigning.Digest(changedBytes)); err == nil {
					t.Fatal("changed replay must be refused")
				}
				return
			}
			if !errors.Is(mutationErr, ErrReserved) {
				t.Fatalf("refused acknowledgement must retain pin, got %v", mutationErr)
			}
		})
	}
}
