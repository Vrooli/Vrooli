package cloudtarget

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/credentialauthority"
)

// fakeAuthority is an in-memory node credential store that can be locked.
type fakeAuthority struct {
	values map[string]string
	locked bool
	absent bool
	puts   int
}

func newFakeAuthority() *fakeAuthority { return &fakeAuthority{values: map[string]string{}} }

func (f *fakeAuthority) key(identity credentialauthority.Identity, field string) string {
	return string(identity) + "/" + field
}

func (f *fakeAuthority) Availability() error {
	switch {
	case f.absent:
		return fmt.Errorf("%w: no store", credentialauthority.ErrProviderAbsent)
	case f.locked:
		return fmt.Errorf("%w: keyring locked", credentialauthority.ErrProviderUnavailable)
	}
	return nil
}

func (f *fakeAuthority) Put(identity credentialauthority.Identity, field, value string) error {
	if err := f.Availability(); err != nil {
		return err
	}
	f.puts++
	f.values[f.key(identity, field)] = value
	return nil
}

func (f *fakeAuthority) Delete(identity credentialauthority.Identity, field string) error {
	if err := f.Availability(); err != nil {
		return err
	}
	if _, ok := f.values[f.key(identity, field)]; !ok {
		return credentialauthority.ErrUnconfigured
	}
	delete(f.values, f.key(identity, field))
	return nil
}

func (f *fakeAuthority) Status(identity credentialauthority.Identity, field string) credentialauthority.Status {
	state := credentialauthority.ProviderAvailable
	if f.locked {
		state = credentialauthority.ProviderUnavailable
	}
	_, ok := f.values[f.key(identity, field)]
	return credentialauthority.Status{Identity: identity, Field: field, Configured: ok, ProviderState: state}
}

const (
	testBindingID = "dep-1/vrooli/app:db-password"
	testCanary    = "canary-secret-7f2c9d1e"
)

func credentialFixture(t *testing.T) (*Store, *fakeAuthority, CredentialDeps) {
	t.Helper()
	store := NewStore(filepath.Join(t.TempDir(), "deployments"))
	authority := newFakeAuthority()
	deps := CredentialDeps{Authority: func() (CredentialAuthority, error) { return authority, nil }}
	return store, authority, deps
}

func ingestPayload(version int64, contentRef, value string) string {
	raw, _ := json.Marshal(IngestPayload{SchemaVersion: IngestSchemaVersion, DeploymentID: "dep-1", BindingID: testBindingID, LogicalID: "vrooli/app", Field: "db-password", Version: version, ContentRef: contentRef, Value: value})
	return string(raw)
}

func effect(op, step string, fence uint64) EffectRequest {
	return EffectRequest{DeploymentID: "dep-1", OperationID: op, Step: step, Fence: fence}
}

// [REQ:STC-P0-032] SECRET-01: the value reaches the node authority only
// through the standard-input payload; the receipt, the ledger and the
// receipt input digest carry metadata only, and a replay of the same
// (operation, step) never rewrites the store.
func TestCredentialIngestStoresValueFromStdinAndReceiptsMetadataOnly(t *testing.T) {
	store, authority, deps := credentialFixture(t)
	deps.Stdin = strings.NewReader(ingestPayload(1, "cref-1", testCanary))
	result, err := store.CredentialIngest(context.Background(), CredentialIngestRequest{Effect: effect("op-1", "ingest-v1", 3), BindingID: testBindingID, Version: 1}, deps)
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if result.Receipt.Outcome != OutcomeSucceeded || result.Replayed {
		t.Fatalf("receipt = %+v", result.Receipt)
	}
	if got := authority.values["vrooli/app/db-password"]; got != testCanary {
		t.Fatalf("authority holds %q", got)
	}
	record, found, err := store.ReadCredentialRecord("dep-1", testBindingID)
	if err != nil || !found || record.Version != 1 || record.ContentRef != "cref-1" || record.LogicalID != "vrooli/app" {
		t.Fatalf("record = %+v found=%t err=%v", record, found, err)
	}
	// Everything the target retains must be free of the value.
	dir, _ := store.DeploymentDir("dep-1")
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, _ := os.ReadFile(path)
		if strings.Contains(string(data), testCanary) {
			t.Fatalf("retained file %s contains the canary", path)
		}
		return nil
	})
	raw, _ := json.Marshal(result)
	if strings.Contains(string(raw), testCanary) {
		t.Fatal("the printed receipt contains the canary")
	}
	// Replay with the same metadata returns the receipt without a second write.
	deps.Stdin = strings.NewReader(ingestPayload(1, "cref-1", "different-value-must-not-be-written"))
	replay, err := store.CredentialIngest(context.Background(), CredentialIngestRequest{Effect: effect("op-1", "ingest-v1", 3), BindingID: testBindingID, Version: 1}, deps)
	if err != nil || !replay.Replayed || authority.puts != 1 {
		t.Fatalf("replay = %+v err=%v puts=%d", replay, err, authority.puts)
	}
}

// [REQ:STC-P0-032] SECRET-06: a version below the held one is refused so a
// concurrent older deployment cannot regress the credential, and a version
// revoked on this target can never be ingested again.
func TestCredentialIngestRefusesStaleAndRevokedVersions(t *testing.T) {
	store, _, deps := credentialFixture(t)
	deps.Stdin = strings.NewReader(ingestPayload(2, "cref-2", testCanary))
	if _, err := store.CredentialIngest(context.Background(), CredentialIngestRequest{Effect: effect("op-2", "ingest-v2", 5), BindingID: testBindingID, Version: 2}, deps); err != nil {
		t.Fatalf("ingest v2: %v", err)
	}
	deps.Stdin = strings.NewReader(ingestPayload(1, "cref-1", "older"))
	result, err := store.CredentialIngest(context.Background(), CredentialIngestRequest{Effect: effect("op-3", "ingest-v1", 6), BindingID: testBindingID, Version: 1}, deps)
	if typed := AsError(err); typed == nil || typed.Code != CodeCredentialVersionStale || typed.ExitCode() != ExitRefused {
		t.Fatalf("stale ingest err=%v result=%+v", err, result)
	}
	if _, err := store.CredentialRevoke(context.Background(), CredentialRevokeRequest{Effect: effect("op-4", "revoke-v2", 7), BindingID: testBindingID, Version: 2}, deps); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	deps.Stdin = strings.NewReader(ingestPayload(2, "cref-2", testCanary))
	_, err = store.CredentialIngest(context.Background(), CredentialIngestRequest{Effect: effect("op-5", "ingest-v2-again", 8), BindingID: testBindingID, Version: 2}, deps)
	if typed := AsError(err); typed == nil || typed.Code != CodeCredentialVersionRevoked {
		t.Fatalf("re-ingest of revoked version err=%v", err)
	}
}

// [REQ:STC-P0-032] SECRET-07: a locked store fails closed with
// credential_store_locked before any write; a mismatched payload is refused
// before the store is consulted.
func TestCredentialIngestFailsClosedOnLockedStoreAndBadPayload(t *testing.T) {
	store, authority, deps := credentialFixture(t)
	authority.locked = true
	deps.Stdin = strings.NewReader(ingestPayload(1, "cref-1", testCanary))
	result, err := store.CredentialIngest(context.Background(), CredentialIngestRequest{Effect: effect("op-1", "ingest-v1", 1), BindingID: testBindingID, Version: 1}, deps)
	typed := AsError(err)
	if typed == nil || typed.Code != CodeCredentialStoreLocked || typed.ExitCode() != ExitRefused {
		t.Fatalf("locked err=%v", err)
	}
	if result.Receipt.Outcome != OutcomeFailed || authority.puts != 0 {
		t.Fatalf("locked store wrote something: %+v puts=%d", result.Receipt, authority.puts)
	}
	authority.locked = false
	deps.Stdin = strings.NewReader(ingestPayload(9, "cref-9", testCanary))
	_, err = store.CredentialIngest(context.Background(), CredentialIngestRequest{Effect: effect("op-2", "ingest-v1", 2), BindingID: testBindingID, Version: 1}, deps)
	if typed := AsError(err); typed == nil || typed.Code != CodeCredentialPayloadInvalid {
		t.Fatalf("mismatched payload err=%v", err)
	}
	deps.Stdin = strings.NewReader("")
	_, err = store.CredentialIngest(context.Background(), CredentialIngestRequest{Effect: effect("op-3", "ingest-v1", 2), BindingID: testBindingID, Version: 1}, deps)
	if typed := AsError(err); typed == nil || typed.Code != CodeCredentialPayloadInvalid {
		t.Fatalf("empty stdin err=%v", err)
	}
	if authority.puts != 0 {
		t.Fatalf("refusals wrote to the store: %d", authority.puts)
	}
}

// [REQ:STC-P0-032] SECRET-01 (bridge): the Bridge dispatch path names an
// environment variable in argv and the value arrives only through it.
func TestCredentialIngestFromEnvNeverReadsArgvOrStdin(t *testing.T) {
	store, authority, deps := credentialFixture(t)
	deps.Env = func(name string) string {
		if name == "VROOLI_CREDENTIAL_INGEST_VALUE" {
			return testCanary
		}
		return ""
	}
	deps.Stdin = strings.NewReader("this must never be read")
	result, err := store.CredentialIngest(context.Background(), CredentialIngestRequest{Effect: effect("op-1", "ingest-v1", 1), BindingID: testBindingID, LogicalID: "vrooli/app", Field: "db-password", Version: 1, ContentRef: "cref-1", GrantRef: "grant-1", FromEnv: "VROOLI_CREDENTIAL_INGEST_VALUE"}, deps)
	if err != nil {
		t.Fatalf("ingest from env: %v", err)
	}
	if authority.values["vrooli/app/db-password"] != testCanary || result.Receipt.Details["grant_ref"] != "grant-1" || result.Receipt.Details["value_channel"] != "env" {
		t.Fatalf("result = %+v values=%v", result.Receipt.Details, authority.values)
	}
	deps.Env = func(string) string { return "" }
	_, err = store.CredentialIngest(context.Background(), CredentialIngestRequest{Effect: effect("op-2", "ingest-v2", 2), BindingID: testBindingID, LogicalID: "vrooli/app", Field: "db-password", Version: 2, ContentRef: "cref-2", FromEnv: "VROOLI_CREDENTIAL_INGEST_VALUE"}, deps)
	if typed := AsError(err); typed == nil || typed.Code != CodeCredentialPayloadInvalid {
		t.Fatalf("missing injection err=%v", err)
	}
}

// [REQ:STC-P0-032] P13-A02: acknowledgement proves the held version matches
// and the authority still answers; it is refused for a version the node
// does not hold and after revocation.
func TestCredentialAcknowledgeProvesHeldVersion(t *testing.T) {
	store, authority, deps := credentialFixture(t)
	_, err := store.CredentialAcknowledge(context.Background(), CredentialAckRequest{Effect: effect("op-0", "ack", 1), BindingID: testBindingID, Version: 1, Consumer: "scenario:app"}, deps)
	if typed := AsError(err); typed == nil || typed.Code != CodeCredentialNotIngested {
		t.Fatalf("ack before ingest err=%v", err)
	}
	deps.Stdin = strings.NewReader(ingestPayload(1, "cref-1", testCanary))
	if _, err := store.CredentialIngest(context.Background(), CredentialIngestRequest{Effect: effect("op-1", "ingest-v1", 2), BindingID: testBindingID, Version: 1}, deps); err != nil {
		t.Fatal(err)
	}
	result, err := store.CredentialAcknowledge(context.Background(), CredentialAckRequest{Effect: effect("op-1", "ack-app-v1", 2), BindingID: testBindingID, Version: 1, Consumer: "scenario:app"}, deps)
	if err != nil || result.Receipt.Details["verified"] != true || result.Receipt.Details["consumer"] != "scenario:app" {
		t.Fatalf("ack = %+v err=%v", result.Receipt, err)
	}
	_, err = store.CredentialAcknowledge(context.Background(), CredentialAckRequest{Effect: effect("op-1", "ack-app-v2", 2), BindingID: testBindingID, Version: 2, Consumer: "scenario:app"}, deps)
	if typed := AsError(err); typed == nil || typed.Code != CodeCredentialVersionMismatch {
		t.Fatalf("ack wrong version err=%v", err)
	}
	delete(authority.values, "vrooli/app/db-password")
	_, err = store.CredentialAcknowledge(context.Background(), CredentialAckRequest{Effect: effect("op-2", "ack-app-v1", 3), BindingID: testBindingID, Version: 1, Consumer: "scenario:app"}, deps)
	if typed := AsError(err); typed == nil || typed.Code != CodeCredentialNotIngested {
		t.Fatalf("ack after store lost the value err=%v", err)
	}
}

// [REQ:STC-P0-032] SECRET-04: revoking the active version deletes it from
// the authority and the receipt states what was proven and what cannot be;
// revoking a predecessor only marks it, and a locked store is a refusal.
func TestCredentialRevokePurgesAndStatesLimitations(t *testing.T) {
	store, authority, deps := credentialFixture(t)
	deps.Stdin = strings.NewReader(ingestPayload(1, "cref-1", testCanary))
	if _, err := store.CredentialIngest(context.Background(), CredentialIngestRequest{Effect: effect("op-1", "ingest-v1", 1), BindingID: testBindingID, Version: 1}, deps); err != nil {
		t.Fatal(err)
	}
	deps.Stdin = strings.NewReader(ingestPayload(2, "cref-2", "canary-secret-second"))
	if _, err := store.CredentialIngest(context.Background(), CredentialIngestRequest{Effect: effect("op-2", "ingest-v2", 2), BindingID: testBindingID, Version: 2}, deps); err != nil {
		t.Fatal(err)
	}
	result, err := store.CredentialRevoke(context.Background(), CredentialRevokeRequest{Effect: effect("op-2", "revoke-v1", 2), BindingID: testBindingID, Version: 1}, deps)
	if err != nil || result.Receipt.Details["store_purged"] != false {
		t.Fatalf("predecessor revoke = %+v err=%v", result.Receipt.Details, err)
	}
	if _, ok := authority.values["vrooli/app/db-password"]; !ok {
		t.Fatal("revoking the predecessor removed the active version")
	}
	authority.locked = true
	_, err = store.CredentialRevoke(context.Background(), CredentialRevokeRequest{Effect: effect("op-3", "revoke-v2", 3), BindingID: testBindingID, Version: 2}, deps)
	if typed := AsError(err); typed == nil || typed.Code != CodeCredentialStoreLocked {
		t.Fatalf("locked revoke err=%v", err)
	}
	authority.locked = false
	result, err = store.CredentialRevoke(context.Background(), CredentialRevokeRequest{Effect: effect("op-4", "revoke-v2", 4), BindingID: testBindingID, Version: 2}, deps)
	if err != nil || result.Receipt.Details["store_purged"] != true || result.Receipt.Details["active_version_after"] != "none" {
		t.Fatalf("active revoke = %+v err=%v", result.Receipt.Details, err)
	}
	if _, ok := authority.values["vrooli/app/db-password"]; ok {
		t.Fatal("active version still in the authority after revoke")
	}
	limitations, _ := result.Receipt.Details["limitations"].([]string)
	if len(limitations) != 1 || !strings.Contains(limitations[0], "cannot prove") {
		t.Fatalf("limitations = %v", result.Receipt.Details["limitations"])
	}
	unproven, _ := result.Receipt.Details["unproven"].([]string)
	if len(unproven) == 0 {
		t.Fatalf("receipt does not state what revocation cannot prove: %v", result.Receipt.Details)
	}
	record, _, _ := store.ReadCredentialRecord("dep-1", testBindingID)
	if !record.Revoked || len(record.RevokedVersions) != 2 {
		t.Fatalf("record = %+v", record)
	}
}

// Binding ids carry `/` and `:`; the ledger must accept them and keep
// distinct ids distinct while refusing unusable ones.
func TestCredentialBindingIDsMapToDistinctLedgerFiles(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "deployments"))
	a, err := store.credentialPath("dep-1", "dep-1/vrooli/app:db-password")
	if err != nil {
		t.Fatal(err)
	}
	b, err := store.credentialPath("dep-1", "dep-1/vrooli/app:db_password")
	if err != nil {
		t.Fatal(err)
	}
	if a == b || filepath.Dir(a) != filepath.Dir(b) {
		t.Fatalf("paths a=%s b=%s", a, b)
	}
	if _, err := store.credentialPath("dep-1", "bad\nid"); err == nil {
		t.Fatal("newline in a binding id was accepted")
	}
	var typed *Error
	if _, err := store.credentialPath("dep-1", ""); !errors.As(err, &typed) || typed.Code != CodeInvalidArgument {
		t.Fatalf("empty id err=%v", err)
	}
}
