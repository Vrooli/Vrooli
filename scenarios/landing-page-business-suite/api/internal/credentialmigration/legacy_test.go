package credentialmigration

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

type migrationStore struct {
	mu     sync.Mutex
	values map[string]string
}

func (s *migrationStore) Put(service, key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[service+"/"+key] = value
	return nil
}
func (s *migrationStore) Get(service, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.values[service+"/"+key]
	if !ok {
		return "", fmt.Errorf("%w: %s/%s", credentialauthority.ErrNotFound, service, key)
	}
	return v, nil
}
func (s *migrationStore) Delete(service, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.values, service+"/"+key)
	return nil
}

type failingAuthority struct {
	Authority
	failField           string
	cleanupPendingField string
	fail                bool
}

func (a *failingAuthority) ActivateCandidate(ref credentialauthority.CandidateRef) error {
	return a.ActivateCandidateIfVersion(ref, ref.ExpectedVersion)
}

func (a *failingAuthority) ActivateCandidateIfVersion(ref credentialauthority.CandidateRef, expected string) error {
	if a.fail && ref.Field == a.failField {
		return errors.New("synthetic activation outage")
	}
	err := a.Authority.ActivateCandidateIfVersion(ref, expected)
	if err == nil && ref.Field == a.cleanupPendingField {
		return &credentialauthority.CandidateCleanupPendingError{Version: ref.Version, Cause: errors.New("synthetic cleanup outage")}
	}
	return err
}

func newMigrationFixture(t *testing.T) (*sql.DB, *credentialauthority.Authority) {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE payment_settings(id INTEGER PRIMARY KEY, publishable_key TEXT, secret_key TEXT, webhook_secret TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE download_storage_settings(bundle_key TEXT PRIMARY KEY, access_key_id TEXT, secret_access_key TEXT, session_token TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO payment_settings VALUES(1,'pk_test_fixture','rk_test_fixture','whsec_fixture')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO download_storage_settings VALUES('business_suite','access_fixture','secret_fixture','session_fixture')`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	authority, err := credentialauthority.NewAuthority(&migrationStore{})
	if err != nil {
		t.Fatal(err)
	}
	return db, authority
}

func TestMigrationBackfillsVersionedFieldsIdempotentlyAndRetainsSource(t *testing.T) {
	db, authority := newMigrationFixture(t)
	first, err := Migrate(context.Background(), db, "test", authority)
	if err != nil {
		t.Fatal("first migration failed")
	}
	if len(first.Applied) != 6 {
		t.Fatalf("applied %d fields, want six", len(first.Applied))
	}
	for _, field := range first.Applied {
		status := authority.Status(identity, field)
		if !status.Configured || strings.TrimSpace(status.Version) == "" {
			t.Fatalf("%s lacks active version metadata", field)
		}
	}
	second, err := Migrate(context.Background(), db, "test", authority)
	if err != nil {
		t.Fatal("retry failed")
	}
	if len(second.Applied) != 0 || len(second.Retained) != 6 {
		t.Fatalf("retry result applied=%d retained=%d", len(second.Applied), len(second.Retained))
	}
	var retainedStripe, retainedS3 string
	if err := db.QueryRow(`SELECT secret_key FROM payment_settings WHERE id=1`).Scan(&retainedStripe); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT secret_access_key FROM download_storage_settings WHERE bundle_key='business_suite'`).Scan(&retainedS3); err != nil {
		t.Fatal(err)
	}
	if retainedStripe != "rk_test_fixture" || retainedS3 != "secret_fixture" {
		t.Fatal("legacy source values were changed; migration must retain rollback source")
	}
	serialized, err := json.Marshal(struct {
		Plan   *Plan  `json:"plan"`
		Result Result `json:"result"`
	}{&Plan{Items: []Item{{Field: "stripe-test-secret-key", Disposition: "backfill"}}}, first})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(serialized), "rk_test_fixture") || strings.Contains(string(serialized), "secret_fixture") {
		t.Fatal("migration summary emitted credential values")
	}
}

func TestMigrationReportsCommittedCandidateCleanupPendingWithoutRollback(t *testing.T) {
	db, authority := newMigrationFixture(t)
	plan, err := Prepare(context.Background(), db, "test", authority)
	if err != nil {
		t.Fatal(err)
	}
	defer plan.Clear()
	result, err := Apply(context.Background(), plan, &failingAuthority{
		Authority: authority, cleanupPendingField: "stripe-test-publishable-key",
	})
	if err != nil {
		t.Fatalf("migration treated a committed candidate as failed: %v", err)
	}
	if len(result.CleanupPending) != 1 || result.CleanupPending[0] != "stripe-test-publishable-key" || len(result.Applied) == 0 {
		t.Fatalf("migration omitted committed cleanup receipt: %+v", result)
	}
	if status := authority.Status(identity, "stripe-test-publishable-key"); !status.Configured || status.Version == "" {
		t.Fatal("cleanup-pending migration did not retain the active version")
	}
}

func TestMigrationConflictPreflightPreventsPartialWrites(t *testing.T) {
	db, authority := newMigrationFixture(t)
	seed(t, authority, "stripe-test-secret-key", "preexisting-different-fixture")
	if _, err := Prepare(context.Background(), db, "test", authority); err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatal("different active value was not rejected as a conflict")
	}
	if !errors.Is(statusErr(authority, "stripe-test-publishable-key"), credentialauthority.ErrUnconfigured) {
		t.Fatal("conflict preflight wrote another candidate before checking all targets")
	}
}

func TestPrepareIsPlanOnlyAndSummaryOmitsAllValues(t *testing.T) {
	db, authority := newMigrationFixture(t)
	plan, err := Prepare(context.Background(), db, "test", authority)
	if err != nil {
		t.Fatal("prepare failed")
	}
	if len(plan.Items) != 6 || !errors.Is(statusErr(authority, "stripe-test-secret-key"), credentialauthority.ErrUnconfigured) {
		t.Fatal("prepare wrote an authority credential instead of returning a plan")
	}
	encoded, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range []string{"pk_test_fixture", "rk_test_fixture", "whsec_fixture", "secret_fixture", "session_fixture"} {
		if strings.Contains(string(encoded), fixture) {
			t.Fatal("migration plan summary contained a credential value")
		}
	}
}

func TestCanceledApplyAndClearLeaveAuthorityUntouched(t *testing.T) {
	db, authority := newMigrationFixture(t)
	plan, err := Prepare(context.Background(), db, "test", authority)
	if err != nil {
		t.Fatal("prepare failed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Apply(ctx, plan, authority); err == nil {
		t.Fatal("canceled apply unexpectedly continued")
	}
	if !errors.Is(statusErr(authority, "stripe-test-secret-key"), credentialauthority.ErrUnconfigured) {
		t.Fatal("canceled apply wrote an authority credential")
	}
	plan.Clear()
	if len(plan.values) != 0 {
		t.Fatal("Clear retained private migration values")
	}
}

func TestMigrationPreconditionsRefuseWrongModeIncompletePairsAndMissingSchema(t *testing.T) {
	db, authority := newMigrationFixture(t)
	if _, err := Prepare(context.Background(), db, "live", authority); err == nil {
		t.Fatal("mode mismatch accepted")
	}
	if _, err := Prepare(context.Background(), db, "", authority); err == nil {
		t.Fatal("unset mode accepted")
	}
	if _, err := db.Exec(`UPDATE payment_settings SET publishable_key=NULL WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if _, err := Prepare(context.Background(), db, "test", authority); err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Fatal("incomplete Stripe pair accepted")
	}
	empty, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer empty.Close()
	if _, err := Prepare(context.Background(), empty, "test", authority); err == nil {
		t.Fatal("missing legacy schema accepted as an empty migration")
	}
}

func TestMigrationActivationFailureRollsBackAndRetrySucceeds(t *testing.T) {
	db, authority := newMigrationFixture(t)
	if err := authority.Put(identity, "stripe-test-publishable-key", "pk_test_fixture"); err != nil {
		t.Fatal(err)
	}
	broken := &failingAuthority{Authority: authority, failField: "stripe-test-secret-key", fail: true}
	if _, err := Migrate(context.Background(), db, "test", broken); err == nil || !strings.Contains(err.Error(), "source values were retained") {
		t.Fatal("synthetic mid-migration activation failure did not preserve source")
	}
	value, err := authority.Resolve(identity, "stripe-test-publishable-key")
	if err != nil || value != "pk_test_fixture" || strings.TrimSpace(authority.Status(identity, "stripe-test-publishable-key").Version) == "" {
		t.Fatal("prior active value was not restored as a versioned credential")
	}
	broken.fail = false
	result, err := Migrate(context.Background(), db, "test", broken)
	if err != nil || len(result.Applied) != 5 {
		t.Fatalf("recovery retry applied=%d err=%v, want five new fields after restoring the existing field", len(result.Applied), err)
	}
	var source string
	if err := db.QueryRow(`SELECT publishable_key FROM payment_settings WHERE id=1`).Scan(&source); err != nil || source != "pk_test_fixture" {
		t.Fatal("legacy source was not retained after recovery")
	}
}

func TestMigrationRefusesAuthorityOutageWithoutLeakingValues(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE payment_settings(id INTEGER PRIMARY KEY, publishable_key TEXT, secret_key TEXT, webhook_secret TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE download_storage_settings(bundle_key TEXT PRIMARY KEY, access_key_id TEXT, secret_access_key TEXT, session_token TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO payment_settings VALUES(1,'pk_test_value_hidden','rk_test_value_hidden','whsec_value_hidden')`); err != nil {
		t.Fatal(err)
	}
	unavailable, err := credentialauthority.Unavailable("synthetic unavailable provider")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Prepare(context.Background(), db, "test", unavailable)
	if err == nil || strings.Contains(err.Error(), "value_hidden") {
		t.Fatal("authority outage did not fail closed without exposing source values")
	}
}

func seed(t *testing.T, authority *credentialauthority.Authority, field, value string) {
	t.Helper()
	candidate, err := authority.PutCandidate(identity, field, value)
	if err != nil {
		t.Fatal(err)
	}
	if err := authority.ActivateCandidate(candidate); err != nil {
		t.Fatal(err)
	}
}
func statusErr(authority *credentialauthority.Authority, field string) error {
	status := authority.Status(identity, field)
	if string(status.ProviderState) != "available" {
		return credentialauthority.ErrProviderUnavailable
	}
	if !status.Configured {
		return credentialauthority.ErrUnconfigured
	}
	return nil
}
