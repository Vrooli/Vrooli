//nolint:goconst // test data deliberately reuses stable protocol fixtures.
package vaultbootstrap

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"github.com/vrooli/vrooli/packages/resource-deployment/securestore"
)

// memoryStore honours the adapter contract: a clean "no value" is ErrNotFound,
// and anything else means the backend is broken.
type memoryStore struct {
	values  map[string]string
	failGet error
}

func (s *memoryStore) Put(service, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[service+"/"+key] = value
	return nil
}

func (s *memoryStore) Get(service, key string) (string, error) {
	if s.failGet != nil {
		return "", s.failGet
	}
	value, ok := s.values[service+"/"+key]
	if !ok {
		return "", fmt.Errorf("%w: %s/%s", securestore.ErrNotFound, service, key)
	}
	return value, nil
}

func (s *memoryStore) Delete(service, key string) error {
	delete(s.values, service+"/"+key)
	return nil
}

// This is the defect the consolidation exposed. The previous implementations
// treated ANY read failure as "nothing stored" and bootstrapped a fresh
// instance — so one transient store error would initialize over a live Vault
// and replace the key to data still sealed under the old one.
func TestLoadSeparatesAnEmptyStoreFromABrokenOne(t *testing.T) {
	t.Run("nothing stored is not an error", func(t *testing.T) {
		_, found, err := Load(&memoryStore{}, nil, "instance-1")
		if err != nil {
			t.Fatalf("Load() = %v, want a clean not-found", err)
		}
		if found {
			t.Fatal("Load() reported material in an empty store")
		}
	})

	t.Run("a broken store is an error, never an empty one", func(t *testing.T) {
		store := &memoryStore{failGet: errors.New("keyring unreachable")}
		_, found, err := Load(store, nil, "instance-1")
		if err == nil {
			t.Fatal("Load() swallowed a store failure; a caller would bootstrap over a live instance")
		}
		if found {
			t.Fatal("Load() reported material it never read")
		}
	})
}

// Incomplete material cannot restore an instance, so it must not be mistaken
// for a usable answer or written in the first place.
func TestIncompleteMaterialIsRejectedBothWays(t *testing.T) {
	store := &memoryStore{}
	if err := Save(store, nil, "instance-1", Material{RootToken: "root"}); err == nil {
		t.Fatal("Save() stored material with no unseal key")
	}
	if err := Save(store, nil, "instance-1", Material{UnsealKey: "key"}); err == nil {
		t.Fatal("Save() stored material with no root token")
	}
	// Present but half-written on disk, as a partial legacy record would be.
	_ = store.Put(Service, "instance-2", `{"root_token":"root"}`)
	if _, _, err := Load(store, nil, "instance-2"); err == nil {
		t.Fatal("Load() accepted material that cannot unseal anything")
	}
}

// A store that accepts a write it did not keep would leave a running Vault
// nothing can ever unseal again, so Save proves the readback.
func TestSaveVerifiesTheReadback(t *testing.T) {
	material := Material{RootToken: "root", UnsealKey: "key"}
	good := &memoryStore{}
	if err := Save(good, nil, "instance-1", material); err != nil {
		t.Fatalf("Save() = %v", err)
	}
	loaded, found, err := Load(good, nil, "instance-1")
	if err != nil || !found || loaded != material {
		t.Fatalf("round trip = %#v, %v, %v", loaded, found, err)
	}

	if err := Save(&discardingStore{}, nil, "instance-1", material); err == nil {
		t.Fatal("Save() reported success against a store that kept nothing")
	}
}

// discardingStore accepts writes and forgets them — the silent-loss backend.
type discardingStore struct{}

func (discardingStore) Put(string, string, string) error { return nil }
func (discardingStore) Get(service, key string) (string, error) {
	return "", fmt.Errorf("%w: %s/%s", securestore.ErrNotFound, service, key)
}
func (discardingStore) Delete(string, string) error { return nil }

func TestClassifyCoversEverySealStatus(t *testing.T) {
	for _, testCase := range []struct {
		initialized, sealed bool
		want                State
	}{
		{initialized: false, sealed: true, want: StateUninitialized},
		{initialized: false, sealed: false, want: StateUninitialized},
		{initialized: true, sealed: true, want: StateSealed},
		{initialized: true, sealed: false, want: StateUnsealed},
	} {
		if got := Classify(testCase.initialized, testCase.sealed); got != testCase.want {
			t.Fatalf("Classify(%t,%t) = %q, want %q",
				testCase.initialized, testCase.sealed, got, testCase.want)
		}
	}
}

// Vault answers a duplicate mount with 400, which is the normal state on every
// restart after the first. Treating it as a failure would make a healthy
// instance refuse to start.
func TestEnsureKVv2TreatsAnExistingMountAsSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()
	if err := (Client{Endpoint: server.URL}).EnsureKVv2(context.Background(), "root"); err != nil {
		t.Fatalf("EnsureKVv2() = %v, want an existing mount to be success", err)
	}

	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failing.Close()
	if err := (Client{Endpoint: failing.URL}).EnsureKVv2(context.Background(), "root"); err == nil {
		t.Fatal("EnsureKVv2() accepted a genuine mount failure")
	}
}

// Initialize must refuse a response it cannot use. Returning half-material here
// would persist a record that unseals nothing.
func TestInitializeRejectsAnIncompleteResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"keys":[],"root_token":""}`))
	}))
	defer server.Close()
	if _, err := (Client{Endpoint: server.URL}).Initialize(context.Background()); err == nil {
		t.Fatal("Initialize() accepted a response with no unseal key")
	}
}

func TestWaitReachableHonoursItsDeadline(t *testing.T) {
	err := (Client{Endpoint: "http://127.0.0.1:1"}).WaitReachable(context.Background(), 50*time.Millisecond)
	if err == nil {
		t.Fatal("WaitReachable() accepted an unreachable endpoint")
	}
}

// An error must not carry the response body: a Vault error is not a place a
// token should be able to appear in a log.
func TestRequestErrorCarriesNoResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"errors":["permission denied for token hvs.SECRETVALUE"]}`))
	}))
	defer server.Close()
	err := (Client{Endpoint: server.URL}).Request(context.Background(), http.MethodGet, "/v1/sys/seal-status", "hvs.SECRETVALUE", nil, nil)
	if err == nil {
		t.Fatal("Request() accepted a 403")
	}
	if got := err.Error(); got != "Vault returned HTTP 403" {
		t.Fatalf("error = %q, want only the status", got)
	}
}

// fakeUnsealKeyStore stands in for the credential authority.
type fakeUnsealKeyStore struct {
	values   map[string]string
	putErr   error
	getErr   error
	putCalls int
	corrupt  bool
}

func (f *fakeUnsealKeyStore) Put(identity credentialauthority.Identity, field, value string) error {
	f.putCalls++
	if f.putErr != nil {
		return f.putErr
	}
	if f.values == nil {
		f.values = map[string]string{}
	}
	if f.corrupt {
		value += "-mangled"
	}
	f.values[string(identity)+":"+field] = value
	return nil
}

func (f *fakeUnsealKeyStore) Resolve(identity credentialauthority.Identity, field string) (string, error) {
	if f.getErr != nil {
		return "", f.getErr
	}
	value, ok := f.values[string(identity)+":"+field]
	if !ok {
		return "", credentialauthority.ErrUnconfigured
	}
	return value, nil
}

func TestLoadBackfillsAnUnsealKeyFromPreSplitMaterial(t *testing.T) {
	store := &memoryStore{}
	keys := &fakeUnsealKeyStore{}
	material := Material{RootToken: "root", UnsealKey: "unseal"}
	if err := store.Put(Service, "instance-1", `{"root_token":"root","unseal_key":"unseal"}`); err != nil {
		t.Fatal(err)
	}

	got, found, err := Load(store, keys, "instance-1")
	if err != nil || !found || got != material {
		t.Fatalf("Load() = %#v, %v, %v; want pre-split material", got, found, err)
	}
	loaded, keyFound, err := LoadUnsealKey(keys, "instance-1")
	if err != nil || !keyFound || loaded != material.UnsealKey {
		t.Fatalf("backfilled key = %q, %v, %v; want %q", loaded, keyFound, err, material.UnsealKey)
	}
}

func TestLoadDoesNotRewriteAnExistingUnsealKey(t *testing.T) {
	store := &memoryStore{}
	keys := &fakeUnsealKeyStore{}
	if err := SaveUnsealKey(keys, "instance-1", "unseal"); err != nil {
		t.Fatal(err)
	}
	keys.putCalls = 0
	if err := store.Put(Service, "instance-1", `{"root_token":"root","unseal_key":"unseal"}`); err != nil {
		t.Fatal(err)
	}
	if _, found, err := Load(store, keys, "instance-1"); err != nil || !found {
		t.Fatalf("Load() = found %v, err %v", found, err)
	}
	if keys.putCalls != 0 {
		t.Fatalf("Load() rewrote an existing key %d times", keys.putCalls)
	}
}

func TestLoadStillReturnsMaterialWhenUnsealKeyBackfillFails(t *testing.T) {
	store := &memoryStore{}
	keys := &fakeUnsealKeyStore{putErr: errors.New("credential store is locked")}
	if err := store.Put(Service, "instance-1", `{"root_token":"root","unseal_key":"unseal"}`); err != nil {
		t.Fatal(err)
	}
	got, found, err := Load(store, keys, "instance-1")
	if err != nil || !found || !got.Valid() {
		t.Fatalf("Load() = %#v, %v, %v; want usable material despite backfill failure", got, found, err)
	}
}

func TestLoadRefusesToTreatAnUnavailableMaterialStoreAsEmpty(t *testing.T) {
	keys := &fakeUnsealKeyStore{}
	_, found, err := Load(securestore.Unavailable("locked"), keys, "instance-1")
	if err == nil || found {
		t.Fatalf("Load() = found %v, err %v; want an unavailable-store error", found, err)
	}
	if keys.putCalls != 0 {
		t.Fatalf("Load() attempted a key backfill %d times", keys.putCalls)
	}
}

// The unseal key must land in the credential authority, because that is the
// namespace `recovery export --all` covers. Storing it only beside the root
// token is what left Vault material with no backup path at all.
func TestSavePutsTheUnsealKeyWhereRecoveryCanReachIt(t *testing.T) {
	store := &memoryStore{}
	keys := &fakeUnsealKeyStore{}
	material := Material{RootToken: "root", UnsealKey: "unseal"}

	if err := Save(store, keys, "instance-1", material); err != nil {
		t.Fatalf("Save() = %v", err)
	}

	got, found, err := LoadUnsealKey(keys, "instance-1")
	if err != nil || !found || got != "unseal" {
		t.Fatalf("LoadUnsealKey() = %q, %v, %v; want the stored key", got, found, err)
	}

	identity, err := UnsealKeyIdentity("instance-1")
	if err != nil {
		t.Fatal(err)
	}
	if string(identity) != "vrooli/vault/instance-1" {
		t.Fatalf("identity = %q, want vrooli/vault/instance-1", identity)
	}

	// The root token stays in the resource-private namespace and is
	// deliberately absent from the authority: Vault regenerates one from the
	// unseal key, so backing it up widens the blast radius for nothing.
	for key, value := range keys.values {
		if value == "root" {
			t.Fatalf("the root token reached the credential authority at %q", key)
		}
	}
}

// A key the authority did not keep is a key no bundle will carry, and the
// operator would not find out until a restore.
func TestSaveUnsealKeyVerifiesTheReadback(t *testing.T) {
	if err := SaveUnsealKey(&fakeUnsealKeyStore{corrupt: true}, "instance-1", "unseal"); err == nil {
		t.Fatal("SaveUnsealKey() reported success after a mangled write")
	}
}

// The unseal key is written before the blob, so a partial failure leaves the
// half that cannot be regenerated rather than the half that can.
func TestSaveFailsBeforeWritingTheBlobWhenTheUnsealKeyCannotBeStored(t *testing.T) {
	store := &memoryStore{}
	keys := &fakeUnsealKeyStore{putErr: errors.New("authority unavailable")}
	if err := Save(store, keys, "instance-1", Material{RootToken: "root", UnsealKey: "unseal"}); err == nil {
		t.Fatal("Save() ignored an unseal-key write failure")
	}
	if _, found, _ := Load(store, nil, "instance-1"); found {
		t.Fatal("Save() wrote the blob after failing to store the irreplaceable half")
	}
}

// A nil sink is the degraded host: still bootstrap, but the key is not in any
// bundle. Refusing outright would break a host that otherwise works.
func TestSaveWithoutAnAuthorityStillPersistsTheBlob(t *testing.T) {
	store := &memoryStore{}
	if err := Save(store, nil, "instance-1", Material{RootToken: "root", UnsealKey: "unseal"}); err != nil {
		t.Fatalf("Save() = %v, want a nil sink to be tolerated", err)
	}
	if _, found, _ := Load(store, nil, "instance-1"); !found {
		t.Fatal("Save() with no authority did not persist the material")
	}
}

func TestLoadUnsealKeySeparatesUnconfiguredFromBroken(t *testing.T) {
	_, found, err := LoadUnsealKey(&fakeUnsealKeyStore{}, "instance-1")
	if err != nil || found {
		t.Fatalf("LoadUnsealKey() = %v, %v; want a clean not-found", found, err)
	}
}

// mask produces what Vault returns: the token XORed with the one-time pad and
// base64-encoded, so the value never crosses the wire in the clear.
func mask(t *testing.T, token, otp string) string {
	t.Helper()
	if len(token) != len(otp) {
		t.Fatalf("fixture error: token and pad must be the same length")
	}
	out := make([]byte, len(token))
	for i := range out {
		out[i] = token[i] ^ otp[i]
	}
	return base64.RawStdEncoding.EncodeToString(out)
}

// The whole point of backing up only the unseal key: a restored host must be
// able to mint a root token from it. Without this, a recovery yields an
// instance you can unseal and cannot administer.
func TestGenerateRootTokenUnmasksVaultsOneTimePad(t *testing.T) {
	const wantToken = "hvs.THEROOTTOKENVALUE"
	const otp = "0123456789abcdefghijk"

	var sawDelete, sawKey, sawNonce bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/sys/generate-root/attempt":
			sawDelete = true
			w.WriteHeader(http.StatusNoContent)
		case r.URL.Path == "/v1/sys/generate-root/attempt":
			_, _ = fmt.Fprintf(w, `{"nonce":"the-nonce","otp":%q}`, otp)
		case r.URL.Path == "/v1/sys/generate-root/update":
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			sawKey = body["key"] == "the-unseal-key"
			sawNonce = body["nonce"] == "the-nonce"
			_, _ = fmt.Fprintf(w, `{"complete":true,"encoded_token":%q}`, mask(t, wantToken, otp))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	got, err := (Client{Endpoint: server.URL}).GenerateRootToken(context.Background(), "the-unseal-key")
	if err != nil {
		t.Fatalf("GenerateRootToken() = %v", err)
	}
	if got != wantToken {
		t.Fatalf("token round-trip failed: got %d bytes, want %d", len(got), len(wantToken))
	}
	// A stale attempt from an interrupted recovery must not block this one.
	if !sawDelete {
		t.Fatal("no in-progress attempt was cancelled first")
	}
	if !sawKey || !sawNonce {
		t.Fatalf("update did not carry the unseal key and nonce (key=%t nonce=%t)", sawKey, sawNonce)
	}
}

// An incomplete result means Vault wants more shares than we hold, which on a
// threshold-of-one instance means the stored key is not one of its own.
func TestGenerateRootTokenRefusesAnIncompleteResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.URL.Path == "/v1/sys/generate-root/attempt" {
			_, _ = fmt.Fprint(w, `{"nonce":"n","otp":"0123456789"}`)
			return
		}
		_, _ = fmt.Fprint(w, `{"complete":false,"encoded_token":""}`)
	}))
	defer server.Close()
	if _, err := (Client{Endpoint: server.URL}).GenerateRootToken(context.Background(), "wrong-key"); err == nil {
		t.Fatal("GenerateRootToken() accepted an incomplete generation")
	}
}

// A pad that does not match the token would XOR into a plausible-looking string
// that is not a token, surfacing much later as a bare 403.
func TestGenerateRootTokenRejectsAMismatchedPad(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.URL.Path == "/v1/sys/generate-root/attempt" {
			_, _ = fmt.Fprint(w, `{"nonce":"n","otp":"short"}`)
			return
		}
		_, _ = fmt.Fprintf(w, `{"complete":true,"encoded_token":%q}`,
			base64.RawStdEncoding.EncodeToString([]byte("a-much-longer-token")))
	}))
	defer server.Close()
	_, err := (Client{Endpoint: server.URL}).GenerateRootToken(context.Background(), "key")
	if err == nil {
		t.Fatal("GenerateRootToken() accepted a pad of the wrong length")
	}
	if !strings.Contains(err.Error(), "one-time pad") {
		t.Fatalf("error = %v, want it to name the pad mismatch", err)
	}
}

func TestGenerateRootTokenRequiresAnUnsealKey(t *testing.T) {
	if _, err := (Client{Endpoint: "http://127.0.0.1:1"}).GenerateRootToken(context.Background(), "  "); err == nil {
		t.Fatal("GenerateRootToken() accepted an empty unseal key")
	}
}
