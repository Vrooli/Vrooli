package walletrecovery

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"database/sql"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	_ "modernc.org/sqlite"
)

type snapshotCatalogStub struct {
	state SnapshotState
	err   error
}

func (s snapshotCatalogStub) NewestRecoverySnapshot(context.Context, string, string) (SnapshotState, error) {
	return s.state, s.err
}

func openLeaseDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "read-lease.db"))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestReadLeasePinsNewestGenerationAcrossTreasuryUse(t *testing.T) {
	ctx := context.Background()
	db := openLeaseDB(t)
	store := SQLiteReservations{DB: db}
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO dbm_wallet_snapshot_generation(target_id,destination_id,generation) VALUES('target','dest',7)`); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	service := ReadLeaseService{Catalog: snapshotCatalogStub{state: SnapshotState{ID: "snap-newest", TargetID: "target", Generation: 7}}, Store: store, Now: func() time.Time { return now }, TTL: time.Minute}
	lease, err := service.Acquire(ctx, "lease-1", "treasury-wallet-recovery-wallet", "dest", "snap-newest")
	if err != nil {
		t.Fatal(err)
	}
	if lease.SnapshotID != "snap-newest" || lease.Generation != 7 || !lease.ExpiresAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("lease = %+v", lease)
	}
	if _, err = store.BeginMutation(ctx, "target", "dest"); !errors.Is(err, ErrReserved) {
		t.Fatalf("DBM snapshot mutation crossed an active recovery lease: %v", err)
	}
	now = now.Add(30 * time.Second)
	renewed, err := service.Renew(ctx, lease.LeaseID, lease.TargetID)
	if err != nil || !renewed.ExpiresAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("renewed lease = %+v err=%v", renewed, err)
	}
	if err := service.Release(ctx, lease.LeaseID, lease.TargetID); err != nil {
		t.Fatal(err)
	}
	if err := service.Release(ctx, lease.LeaseID, lease.TargetID); err != nil {
		t.Fatalf("release should be idempotent: %v", err)
	}
	release, err := store.BeginMutation(ctx, "target", "dest")
	if err != nil {
		t.Fatalf("DBM mutation remained blocked after lease release: %v", err)
	}
	release()
}

func TestActiveReadLeaseExcludesRotationReservation(t *testing.T) {
	ctx := context.Background()
	db := openLeaseDB(t)
	store := SQLiteReservations{DB: db}
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO dbm_wallet_snapshot_generation(target_id,destination_id,generation) VALUES('target','dest',3)`); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	service := ReadLeaseService{Catalog: snapshotCatalogStub{state: SnapshotState{ID: "snap-current", TargetID: "target", Generation: 3}}, Store: store, Now: func() time.Time { return now }, TTL: time.Minute}
	lease, err := service.Acquire(ctx, "consumer-lease", "treasury-wallet-recovery-wallet", "dest", "snap-current")
	if err != nil {
		t.Fatal(err)
	}
	rotation := Reservation{
		RequestID: "rotation-request", TargetID: lease.TargetID, DestinationID: lease.DestinationID,
		SnapshotID: lease.SnapshotID, TombstoneSHA256: strings.Repeat("a", 64),
		SnapshotGeneration: lease.Generation, ExpiresAt: now.Add(time.Minute),
	}
	if err := store.Reserve(ctx, rotation, now); !errors.Is(err, ErrReserved) {
		t.Fatalf("rotation reservation crossed active read lease: %v", err)
	}
	if err := service.Release(ctx, lease.LeaseID, lease.TargetID); err != nil {
		t.Fatal(err)
	}
	if err := store.Reserve(ctx, rotation, now); err != nil {
		t.Fatalf("rotation reservation remained blocked after read lease release: %v", err)
	}
}

func TestReadLeaseAndTombstoneReservationHaveOneConcurrentWinner(t *testing.T) {
	ctx := context.Background()
	db := openLeaseDB(t)
	store := SQLiteReservations{DB: db}
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO dbm_wallet_snapshot_generation(target_id,destination_id,generation) VALUES('target','dest',5)`); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	service := ReadLeaseService{Catalog: snapshotCatalogStub{state: SnapshotState{ID: "snap-current", TargetID: "target", Generation: 5}}, Store: store, Now: func() time.Time { return now }}
	type result struct {
		name string
		err  error
	}
	start := make(chan struct{})
	results := make(chan result, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		_, err := service.Acquire(ctx, "consumer-lease", "treasury-wallet-recovery-wallet", "dest", "snap-current")
		results <- result{name: "read lease", err: err}
	}()
	go func() {
		defer wg.Done()
		<-start
		err := store.Reserve(ctx, Reservation{
			RequestID: "tombstone-attestation", TargetID: "target", DestinationID: "dest",
			SnapshotID: "snap-current", TombstoneSHA256: strings.Repeat("b", 64),
			SnapshotGeneration: 5, ExpiresAt: now.Add(time.Minute),
		}, now)
		results <- result{name: "tombstone attestation", err: err}
	}()
	close(start)
	wg.Wait()
	close(results)
	var succeeded int
	for got := range results {
		if got.err == nil {
			succeeded++
			continue
		}
		if !errors.Is(got.err, ErrReserved) {
			t.Fatalf("%s failed outside the reservation exclusion contract: %v", got.name, got.err)
		}
	}
	if succeeded != 1 {
		t.Fatalf("concurrent read lease/tombstone reservation winners = %d, want exactly one", succeeded)
	}
}

func TestReadLeaseRejectsSupersededSnapshotAndGenerationRace(t *testing.T) {
	ctx := context.Background()
	db := openLeaseDB(t)
	store := SQLiteReservations{DB: db}
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO dbm_wallet_snapshot_generation(target_id,destination_id,generation) VALUES('target','dest',9)`); err != nil {
		t.Fatal(err)
	}
	service := ReadLeaseService{Catalog: snapshotCatalogStub{state: SnapshotState{ID: "new-tombstone", TargetID: "target", Generation: 9}}, Store: store}
	if _, err := service.Acquire(ctx, "lease-old", "treasury-wallet-recovery-wallet", "dest", "old-active-bundle"); !errors.Is(err, ErrSnapshotSuperseded) {
		t.Fatalf("superseded snapshot accepted: %v", err)
	}
	if _, err := store.BeginMutation(ctx, "target", "dest"); err != nil {
		t.Fatalf("failed stale acquisition left a DBM mutation lock: %v", err)
	}

	_, err := db.Exec(`UPDATE dbm_wallet_snapshot_generation SET generation=10 WHERE target_id='target' AND destination_id='dest'`)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ReserveRead(ctx, Reservation{RequestID: "race", TargetID: "target", DestinationID: "dest", SnapshotID: "new-tombstone", SnapshotGeneration: 9, ExpiresAt: time.Now().Add(time.Minute)}, time.Now()); err == nil {
		t.Fatal("lease accepted after generation changed between catalog and reservation")
	}
}

func TestReadLeaseExpiresAndCannotBeRenewed(t *testing.T) {
	ctx := context.Background()
	db := openLeaseDB(t)
	store := SQLiteReservations{DB: db}
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO dbm_wallet_snapshot_generation(target_id,destination_id,generation) VALUES('target','dest',1)`); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	service := ReadLeaseService{Catalog: snapshotCatalogStub{state: SnapshotState{ID: "snap", TargetID: "target", Generation: 1}}, Store: store, Now: func() time.Time { return now }, TTL: time.Second}
	lease, err := service.Acquire(ctx, "lease-expiring", "treasury-wallet-recovery-wallet", "dest", "snap")
	if err != nil {
		t.Fatal(err)
	}
	now = lease.ExpiresAt.Add(time.Nanosecond)
	if _, err := service.Renew(ctx, lease.LeaseID, lease.TargetID); err == nil {
		t.Fatal("expired read lease was renewed")
	}
	if _, err := db.Exec(`UPDATE dbm_wallet_recovery_reservations SET expires_at=? WHERE request_id=?`, time.Now().UTC().Add(-time.Second).Format(time.RFC3339Nano), lease.LeaseID); err != nil {
		t.Fatal(err)
	}
	release, err := store.BeginMutation(ctx, "target", "dest")
	if err != nil {
		t.Fatalf("expired read lease blocked a new DBM mutation: %v", err)
	}
	release()
}

type readLeaseAuthStub struct{ err error }

func (a readLeaseAuthStub) AuthorizeWalletRecovery(*http.Request) error { return a.err }

type readLeaseVerifierStub struct {
	principal identity.Principal
	err       error
}

func (v readLeaseVerifierStub) VerifyRequest(context.Context, *http.Request) (identity.Principal, error) {
	return v.principal, v.err
}

func TestVerifiedHumanReadLeaseAuthorizerRequiresVerifiedHuman(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	cases := []struct {
		name     string
		verifier HumanSessionVerifier
		wantErr  bool
	}{
		{name: "missing verifier", wantErr: true},
		{name: "unavailable session", verifier: readLeaseVerifierStub{err: errors.New("JWKS unavailable")}, wantErr: true},
		{name: "unverified human", verifier: readLeaseVerifierStub{principal: identity.Principal{Kind: identity.ActorHuman, Subject: "operator"}}, wantErr: true},
		{name: "agent", verifier: readLeaseVerifierStub{principal: identity.Principal{Kind: identity.ActorAgent, Subject: "run-1", Verified: true}}, wantErr: true},
		{name: "service", verifier: readLeaseVerifierStub{principal: identity.Principal{Kind: identity.ActorService, Subject: "service:dbm", Verified: true}}, wantErr: true},
		{name: "verified operator", verifier: readLeaseVerifierStub{principal: identity.Principal{Kind: identity.ActorHuman, Subject: "operator", Verified: true}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := (VerifiedHumanAuthorizer{Verifier: tc.verifier}).AuthorizeWalletRecovery(req)
			if (err != nil) != tc.wantErr {
				t.Fatalf("AuthorizeWalletRecovery() error=%v, wantErr=%t", err, tc.wantErr)
			}
		})
	}
}

func TestReadLeaseHTTPScenarioAuthenticatorRejectsMissingAndMalformedBearer(t *testing.T) {
	verifier := NewDBMReadLeaseVerifier()
	handler := (ReadLeaseHTTP{Auth: VerifiedHumanAuthorizer{Verifier: verifier}}).Handler("acquire")
	for _, authorization := range []string{"", "Bearer not-a-jwt"} {
		req := httptest.NewRequest(http.MethodPost, "/wallet-recovery/read-lease/acquire", strings.NewReader(`{"lease_id":"lease","target_name":"treasury-wallet-recovery-wallet","destination_id":"dest","snapshot_id":"snapshot"}`))
		if authorization != "" {
			req.Header.Set("Authorization", authorization)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("authorization %q returned %d, want 403", authorization, rec.Code)
		}
	}
}

func TestDBMReadLeaseVerifierRejectsSecondAudience(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"keys":[{"kid":"dbm-test","kty":"RSA","alg":"RS256","n":%q,"e":%q}]}`,
			base64.RawURLEncoding.EncodeToString(key.N.Bytes()), base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}))
	}))
	defer jwks.Close()

	token := signReadLeaseJWT(t, key, map[string]any{
		"sub": "operator-1", "iss": "scenario-authenticator",
		"aud": []string{"scenario-authenticator:data-backup-manager", "scenario-authenticator:other"},
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	verifier := newDBMReadLeaseVerifier(authn.JWTConfig{JWKSURL: jwks.URL})
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"lease_id":"lease","target_name":"wallet","destination_id":"dest","snapshot_id":"snapshot"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	(ReadLeaseHTTP{Auth: VerifiedHumanAuthorizer{Verifier: verifier}}).Handler("acquire").ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("multi-audience DBM token returned %d, want 403", rec.Code)
	}

	validToken := signReadLeaseJWT(t, key, map[string]any{
		"sub": "operator-1", "iss": "scenario-authenticator",
		"aud": "scenario-authenticator:data-backup-manager", "exp": time.Now().Add(time.Hour).Unix(),
	})
	validReq := httptest.NewRequest(http.MethodPost, "/", nil)
	validReq.Header.Set("Authorization", "Bearer "+validToken)
	if err := (VerifiedHumanAuthorizer{Verifier: verifier}).AuthorizeWalletRecovery(validReq); err != nil {
		t.Fatalf("valid exclusive-audience human token refused: %v", err)
	}
}

func signReadLeaseJWT(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()
	header, err := json.Marshal(map[string]string{"alg": "RS256", "kid": "dbm-test", "typ": "JWT"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(input))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func TestReadLeaseHTTPRequiresAuthAndSupportsAcquireRenewRelease(t *testing.T) {
	ctx := context.Background()
	db := openLeaseDB(t)
	store := SQLiteReservations{DB: db}
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO dbm_wallet_snapshot_generation(target_id,destination_id,generation) VALUES('target','dest',2)`); err != nil {
		t.Fatal(err)
	}
	service := ReadLeaseService{Catalog: snapshotCatalogStub{state: SnapshotState{ID: "newest", TargetID: "target", Generation: 2}}, Store: store, TTL: time.Minute}
	body := `{"lease_id":"lease","target_name":"treasury-wallet-recovery-wallet","destination_id":"dest","snapshot_id":"newest"}`
	unauthorized := httptest.NewRecorder()
	ReadLeaseHTTP{Service: service}.Handler("acquire").ServeHTTP(unauthorized, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)))
	if unauthorized.Code != http.StatusServiceUnavailable {
		t.Fatalf("unconfigured authorizer status = %d, want 503", unauthorized.Code)
	}

	api := ReadLeaseHTTP{Service: service, Auth: readLeaseAuthStub{}}
	acquired := httptest.NewRecorder()
	api.Handler("acquire").ServeHTTP(acquired, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)))
	if acquired.Code != http.StatusOK || !strings.Contains(acquired.Body.String(), `"snapshot_id":"newest"`) {
		t.Fatalf("acquire response status=%d body=%s", acquired.Code, acquired.Body.String())
	}
	if _, err := store.BeginMutation(ctx, "target", "dest"); !errors.Is(err, ErrReserved) {
		t.Fatalf("DBM mutation not excluded by API lease: %v", err)
	}

	leaseBody := `{"lease_id":"lease","target_id":"target"}`
	renewed := httptest.NewRecorder()
	api.Handler("renew").ServeHTTP(renewed, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(leaseBody)))
	if renewed.Code != http.StatusOK {
		t.Fatalf("renew status=%d body=%s", renewed.Code, renewed.Body.String())
	}
	released := httptest.NewRecorder()
	api.Handler("release").ServeHTTP(released, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(leaseBody)))
	if released.Code != http.StatusOK {
		t.Fatalf("release status=%d body=%s", released.Code, released.Body.String())
	}
	if _, err := store.BeginMutation(ctx, "target", "dest"); err != nil {
		t.Fatalf("DBM mutation remains excluded after release: %v", err)
	}
}
