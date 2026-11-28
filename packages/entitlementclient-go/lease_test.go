package entitlementclient

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/consumeridentity"
)

type memoryIssuerKeyStore struct {
	jwks    []byte
	loadErr error
	saveErr error
}

func (s *memoryIssuerKeyStore) Load(context.Context, string) ([]byte, error) {
	return append([]byte(nil), s.jwks...), s.loadErr
}

func (s *memoryIssuerKeyStore) Save(_ context.Context, _ string, jwks []byte) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.jwks = append([]byte(nil), jwks...)
	return nil
}

func TestLeaseRoundTripAndTamperRejection(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keys := consumeridentity.NewKeySet(consumeridentity.PublicKey{ID: "lease-key", Key: &key.PublicKey})
	payload := Payload{UserIdentity: "user@example.com", Status: "active", PlanTier: "pro", PlanRank: 3, Features: []string{"ai"}, NotAfter: time.Now().Add(time.Hour)}
	token, err := Sign(payload, "lease-key", key)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Verify(token, keys, time.Now())
	if err != nil || got.PlanTier != "pro" || got.PlanRank != 3 {
		t.Fatalf("verify = %#v, %v", got, err)
	}
	parts := split(t, token)
	parts[1] += "x"
	if _, err := Verify(parts[0]+"."+parts[1]+"."+parts[2], keys, time.Now()); !errors.Is(err, ErrLeaseSignature) && !errors.Is(err, ErrLeaseMalformed) {
		t.Fatalf("tampered lease error = %v", err)
	}
}

func TestSignRejectsAmbiguousAppEntitlements(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	base := Payload{UserIdentity: "user@example.com", Status: "active", NotAfter: time.Now().Add(time.Hour)}
	for _, test := range []struct {
		name   string
		grants []AppEntitlement
	}{
		{name: "duplicate app", grants: []AppEntitlement{{AppKey: "audio-tools", Status: "active"}, {AppKey: "audio-tools", Status: "active"}}},
		{name: "missing status", grants: []AppEntitlement{{AppKey: "audio-tools"}}},
		{name: "duplicate feature", grants: []AppEntitlement{{AppKey: "audio-tools", Status: "active", Features: []string{"ai", "ai"}}}},
		{name: "duplicate meter", grants: []AppEntitlement{{AppKey: "audio-tools", Status: "active", Limits: []Limit{{Key: "ai_credits"}, {Key: "ai_credits"}}}}},
		{name: "invalid negative limit", grants: []AppEntitlement{{AppKey: "audio-tools", Status: "active", Limits: []Limit{{Key: "ai_credits", Value: -2}}}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			payload := base
			payload.AppEntitlements = test.grants
			if _, err := Sign(payload, "lease-key", key); err == nil {
				t.Fatal("ambiguous app grant was signed")
			}
		})
	}
}

func TestFileLeaseStoreSurvivesClientRestartAndRejectsTampering(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keys := consumeridentity.NewKeySet(consumeridentity.PublicKey{ID: "lease-key", Key: &key.PublicKey})
	jwks, err := keys.JWKS()
	if err != nil {
		t.Fatal(err)
	}
	lease, err := Sign(Payload{
		UserIdentity:   "user@example.com",
		Status:         "active",
		Features:       []string{"ai"},
		InstallationID: "install-1",
		Resource:       "browser-automation-studio",
		Audience:       "scenario:browser-automation-studio",
		Scopes:         []string{"workflow:execute"},
		NotAfter:       time.Now().Add(time.Hour),
	}, "lease-key", key)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	store, err := NewFileLeaseStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save("user@example.com", LeaseRecord{Lease: lease}); err != nil {
		t.Fatal(err)
	}

	client := NewClient("", nil, nil)
	client.SetStore(store)
	client.SetIssuerKeyStore(&memoryIssuerKeyStore{jwks: jwks})
	binding := LeaseBinding{InstallationID: "install-1", Resource: "browser-automation-studio", Audience: "scenario:browser-automation-studio", Scopes: []string{"workflow:execute"}}
	got, err := client.GetFor(context.Background(), "USER@example.com", binding)
	if err != nil || len(got.Features) != 1 || got.Features[0] != "ai" {
		t.Fatalf("restart restore = %#v, %v", got, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("stored record entries = %d, %v", len(entries), err)
	}
	info, err := entries[0].Info()
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("stored record mode = %v", info.Mode().Perm())
	}

	client = NewClient("", nil, nil)
	client.SetStore(store)
	client.SetIssuerKeyStore(&memoryIssuerKeyStore{jwks: jwks})
	if _, err := client.GetFor(context.Background(), "user@example.com", LeaseBinding{InstallationID: "install-2"}); !errors.Is(err, ErrLeaseBinding) {
		t.Fatalf("wrong binding on restart = %v", err)
	}
	client = NewClient("", nil, nil)
	client.SetStore(store)
	client.SetIssuerKeyStore(&memoryIssuerKeyStore{jwks: jwks})
	client.Now = func() time.Time { return time.Now().Add(2 * time.Hour) }
	if _, err := client.GetFor(context.Background(), "user@example.com", binding); !errors.Is(err, ErrLeaseExpired) {
		t.Fatalf("expired lease on restart = %v", err)
	}

	var record LeaseRecord
	path := filepath.Join(dir, entries[0].Name())
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, &record); err != nil {
		t.Fatal(err)
	}
	parts := split(t, record.Lease)
	parts[1] += "x"
	record.Lease = parts[0] + "." + parts[1] + "." + parts[2]
	if err := store.Save("user@example.com", record); err != nil {
		t.Fatal(err)
	}
	client = NewClient("", nil, nil)
	client.SetStore(store)
	client.SetIssuerKeyStore(&memoryIssuerKeyStore{jwks: jwks})
	if _, err := client.GetFor(context.Background(), "user@example.com", binding); !errors.Is(err, ErrLeaseMalformed) && !errors.Is(err, ErrLeaseSignature) {
		t.Fatalf("tampered persisted lease error = %v", err)
	}

	attackerKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	attackerKeys := consumeridentity.NewKeySet(consumeridentity.PublicKey{ID: "attacker-key", Key: &attackerKey.PublicKey})
	attackerJWKS, err := attackerKeys.JWKS()
	if err != nil {
		t.Fatal(err)
	}
	forgedLease, err := Sign(Payload{UserIdentity: "user@example.com", Status: "active", Features: []string{"ai"}, NotAfter: time.Now().Add(time.Hour)}, "attacker-key", attackerKey)
	if err != nil {
		t.Fatal(err)
	}
	forgedRecord, err := json.Marshal(map[string]any{"lease": forgedLease, "jwks": json.RawMessage(attackerJWKS)})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, forgedRecord, 0o600); err != nil {
		t.Fatal(err)
	}
	client = NewClient("", nil, nil)
	client.SetStore(store)
	client.SetIssuerKeyStore(&memoryIssuerKeyStore{jwks: jwks})
	if _, err := client.CachedAt("user@example.com", time.Now()); !errors.Is(err, ErrLeaseUnknownKey) && !errors.Is(err, ErrLeaseSignature) {
		t.Fatalf("record with replaced JWKS was not rejected against independent trust: %v", err)
	}
}

func TestFetchedLeaseIsPersistedForOfflineRestart(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keys := consumeridentity.NewKeySet(consumeridentity.PublicKey{ID: "lease-key", Key: &key.PublicKey})
	jwks, err := keys.JWKS()
	if err != nil {
		t.Fatal(err)
	}
	lease, err := Sign(Payload{
		UserIdentity: "user@example.com",
		Status:       "active",
		Features:     []string{"workflow:run"},
		Scopes:       []string{"workflow:execute"},
		Limits:       []Limit{{Key: "runs", Value: 4}},
		AppEntitlements: []AppEntitlement{{
			AppKey: "demo", Status: "active", Features: []string{"workflow:run"}, Limits: []Limit{{Key: "runs", Value: 4}},
		}},
		Credits:      map[string]any{"remaining": float64(4), "ledger": []any{map[string]any{"kind": "grant"}}},
		Subscription: map[string]any{"status": "active"},
		NotAfter:     time.Now().Add(time.Hour),
	}, "lease-key", key)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/jwks.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(jwks)
		case "/api/v1/entitlements":
			_, _ = w.Write([]byte(`{"lease":"` + lease + `"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	store, err := NewFileLeaseStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	client := NewClient(server.URL, func(context.Context, string) (string, error) { return "access", nil }, server.Client())
	client.SetStore(store)
	issuerKeys := &memoryIssuerKeyStore{}
	client.SetIssuerKeyStore(issuerKeys)
	got, err := client.Get(context.Background(), "user@example.com")
	if err != nil {
		t.Fatalf("initial fetch = %v", err)
	}
	got.Scopes[0] = "workflow:admin"
	got.Features[0] = "workflow:admin"
	got.Limits[0].Value = 99
	got.AppEntitlements[0].Features[0] = "workflow:admin"
	got.AppEntitlements[0].Limits[0].Value = 99
	got.Credits.(map[string]any)["remaining"] = float64(0)
	got.Credits.(map[string]any)["ledger"].([]any)[0].(map[string]any)["kind"] = "revoked"
	got.Subscription.(map[string]any)["status"] = "cancelled"
	cached, err := client.CachedAt("user@example.com", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	assertCachedGrant(t, cached, []string{"workflow:execute"}, map[string]any{
		"remaining": float64(4), "ledger": []any{map[string]any{"kind": "grant"}},
	}, map[string]any{"status": "active"})
	client = NewClient(server.URL, nil, nil)
	client.SetStore(store)
	client.SetIssuerKeyStore(issuerKeys)
	if _, err := client.CachedAt("user@example.com", time.Now().UTC()); err != nil {
		t.Fatalf("offline restart cached read = %v", err)
	}
	if err := client.Clear(); err != nil {
		t.Fatalf("clear signed-out cache = %v", err)
	}
	if _, err := client.Cached("user@example.com"); !errors.Is(err, ErrLeaseUnavailable) {
		t.Fatalf("cleared memory cache = %v", err)
	}
	if _, err := store.Load("user@example.com"); !errors.Is(err, ErrLeaseUnavailable) {
		t.Fatalf("cleared disk cache = %v", err)
	}
	if trusted, err := issuerKeys.Load(context.Background(), server.URL); err != nil || len(trusted) == 0 {
		t.Fatalf("sign-out unexpectedly cleared issuer trust: bytes=%d err=%v", len(trusted), err)
	}
}

func TestCredentialAuthorityOutageDoesNotBlockOnlineLeaseButSkipsOfflinePersistence(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keys := consumeridentity.NewKeySet(consumeridentity.PublicKey{ID: "lease-key", Key: &key.PublicKey})
	jwks, err := keys.JWKS()
	if err != nil {
		t.Fatal(err)
	}
	lease, err := Sign(Payload{UserIdentity: "user@example.com", Status: "active", NotAfter: time.Now().Add(time.Hour)}, "lease-key", key)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/jwks.json":
			_, _ = w.Write(jwks)
		case "/api/v1/entitlements":
			_, _ = w.Write([]byte(`{"lease":"` + lease + `"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	store, err := NewFileLeaseStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	client := NewClient(server.URL, func(context.Context, string) (string, error) { return "access", nil }, server.Client())
	client.SetStore(store)
	client.SetIssuerKeyStore(&memoryIssuerKeyStore{saveErr: errors.New("credential store locked")})
	if _, err := client.Get(context.Background(), "user@example.com"); err != nil {
		t.Fatalf("online authority result should still be usable: %v", err)
	}
	if _, err := store.Load("user@example.com"); !errors.Is(err, ErrLeaseUnavailable) {
		t.Fatalf("lease was persisted without independent issuer trust: %v", err)
	}
}

func TestClearCancelsInFlightLeasePersistence(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keys := consumeridentity.NewKeySet(consumeridentity.PublicKey{ID: "lease-key", Key: &key.PublicKey})
	jwks, err := keys.JWKS()
	if err != nil {
		t.Fatal(err)
	}
	lease, err := Sign(Payload{UserIdentity: "user@example.com", Status: "active", NotAfter: time.Now().Add(time.Hour)}, "lease-key", key)
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/jwks.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(jwks)
		case "/api/v1/entitlements":
			close(started)
			<-release
			_, _ = w.Write([]byte(`{"lease":"` + lease + `"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	store, err := NewFileLeaseStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	client := NewClient(server.URL, nil, server.Client())
	client.SetStore(store)
	done := make(chan error, 1)
	go func() {
		_, err := client.GetWithAccess(context.Background(), "user@example.com", "access")
		done <- err
	}()
	<-started
	if err := client.Clear(); err != nil {
		t.Fatalf("clear cache during fetch = %v", err)
	}
	releaseOnce.Do(func() { close(release) })
	if err := <-done; !errors.Is(err, ErrLeaseUnavailable) {
		t.Fatalf("in-flight request after sign-out = %v", err)
	}
	if _, err := store.Load("user@example.com"); !errors.Is(err, ErrLeaseUnavailable) {
		t.Fatalf("in-flight request persisted after sign-out: %v", err)
	}
}

func TestLeaseExpiryIsRejectedEvenWithValidSignature(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keys := consumeridentity.NewKeySet(consumeridentity.PublicKey{ID: "lease-key", Key: &key.PublicKey})
	issued := time.Now()
	token, err := Sign(Payload{UserIdentity: "user@example.com", Status: "active", NotAfter: issued.Add(time.Minute)}, "lease-key", key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(token, keys, issued.Add(2*time.Minute)); !errors.Is(err, ErrLeaseExpired) {
		t.Fatalf("expiry error = %v", err)
	}
}

func TestInsecureRemoteAuthorityIsRejectedBeforeAccessTokenResolution(t *testing.T) {
	client := NewClient("http://lpbs.example.test", func(context.Context, string) (string, error) {
		t.Fatal("insecure remote authority must be rejected before resolving credentials")
		return "", nil
	}, nil)
	if _, err := client.Get(context.Background(), "user@example.com"); !errors.Is(err, ErrLeaseInsecureAuthority) {
		t.Fatalf("insecure remote authority error = %v", err)
	}
}

func TestLeaseBindingRejectsWrongDesktopContext(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keys := consumeridentity.NewKeySet(consumeridentity.PublicKey{ID: "lease-key", Key: &key.PublicKey})
	payload := Payload{
		UserIdentity:      "user@example.com",
		BusinessAccountID: "business-1",
		LinkID:            "link-1",
		InstallationID:    "install-1",
		Resource:          "demo",
		Audience:          "scenario:demo",
		Scopes:            []string{"demo:read", "demo:write"},
		Status:            "active",
		NotAfter:          time.Now().Add(time.Hour),
	}
	token, err := Sign(payload, "lease-key", key)
	if err != nil {
		t.Fatal(err)
	}
	binding := LeaseBinding{
		BusinessAccountID: "business-1",
		LinkID:            "link-1",
		InstallationID:    "install-1",
		Resource:          "demo",
		Audience:          "scenario:demo",
		Scopes:            []string{"demo:write", "demo:read"},
	}
	if got, err := VerifyFor(token, keys, time.Now(), binding); err != nil || got.LinkID != "link-1" {
		t.Fatalf("matching binding = %#v, %v", got, err)
	}

	tests := []struct {
		name    string
		binding LeaseBinding
	}{
		{"business account", LeaseBinding{BusinessAccountID: "business-2"}},
		{"installation", LeaseBinding{InstallationID: "other-install"}},
		{"resource", LeaseBinding{Resource: "other-resource"}},
		{"audience", LeaseBinding{Audience: "scenario:other"}},
		{"link", LeaseBinding{LinkID: "link-2"}},
		{"scopes", LeaseBinding{Scopes: []string{"demo:read"}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := VerifyFor(token, keys, time.Now(), test.binding); !errors.Is(err, ErrLeaseBinding) {
				t.Fatalf("binding error = %v", err)
			}
		})
	}
}

func TestCachedLeaseHonorsSignedExpiryWithoutRefreshing(t *testing.T) {
	issued := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	client := NewClient("http://127.0.0.1:1", nil, nil)
	client.cache["user@example.com"] = Payload{UserIdentity: "user@example.com", Status: "active", NotAfter: issued.Add(time.Hour)}

	if _, err := client.CachedAt("USER@example.com", issued.Add(30*time.Minute)); err != nil {
		t.Fatalf("valid cached lease = %v", err)
	}
	if _, err := client.CachedAt("user@example.com", issued.Add(time.Hour)); !errors.Is(err, ErrLeaseExpired) {
		t.Fatalf("expired cached lease error = %v", err)
	}
}

func TestCachedLeaseBindingIsCheckedOffline(t *testing.T) {
	issued := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	client := NewClient("http://127.0.0.1:1", nil, nil)
	client.cache["user@example.com"] = Payload{
		UserIdentity:      "user@example.com",
		BusinessAccountID: "business-1",
		LinkID:            "link-1",
		InstallationID:    "install-1",
		Resource:          "demo",
		Audience:          "scenario:demo",
		Scopes:            []string{"demo:read"},
		NotAfter:          issued.Add(time.Hour),
	}

	binding := LeaseBinding{BusinessAccountID: "business-1", LinkID: "link-1", InstallationID: "install-1", Resource: "demo", Audience: "scenario:demo", Scopes: []string{"demo:read"}}
	if _, err := client.CachedForAt("user@example.com", binding, issued.Add(30*time.Minute)); err != nil {
		t.Fatalf("matching cached binding = %v", err)
	}
	if _, err := client.CachedForAt("user@example.com", LeaseBinding{InstallationID: "other-install"}, issued.Add(30*time.Minute)); !errors.Is(err, ErrLeaseBinding) {
		t.Fatalf("wrong cached binding error = %v", err)
	}
}

func TestCachedLeaseReadsReturnIndependentPayload(t *testing.T) {
	issued := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	client := NewClient("http://127.0.0.1:1", nil, nil)
	client.cache["user@example.com"] = Payload{
		UserIdentity: "user@example.com", Status: "active", NotAfter: issued.Add(time.Hour),
		Scopes: []string{"demo:read"}, Features: []string{"workflow:run"}, Limits: []Limit{{Key: "runs", Value: 4}},
		AppEntitlements: []AppEntitlement{{AppKey: "demo", Status: "active", Features: []string{"workflow:run"}, Limits: []Limit{{Key: "runs", Value: 4}}}},
		Credits:         map[string]any{"ledger": []any{map[string]any{"kind": "grant"}}},
	}
	first, err := client.CachedAt("user@example.com", issued.Add(30*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	first.Scopes[0] = "demo:admin"
	first.Features[0] = "workflow:admin"
	first.Limits[0].Value = 99
	first.AppEntitlements[0].Features[0] = "workflow:admin"
	first.AppEntitlements[0].Limits[0].Value = 99
	first.Credits.(map[string]any)["ledger"].([]any)[0].(map[string]any)["kind"] = "revoked"

	second, err := client.CachedAt("user@example.com", issued.Add(30*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	assertCachedGrant(t, second, []string{"demo:read"}, map[string]any{
		"ledger": []any{map[string]any{"kind": "grant"}},
	}, nil)
}

func assertCachedGrant(t *testing.T, got Payload, scopes []string, credits, subscription any) {
	t.Helper()
	if !reflect.DeepEqual(got.Scopes, scopes) {
		t.Errorf("cached scopes = %v, want %v", got.Scopes, scopes)
	}
	if !reflect.DeepEqual(got.Features, []string{"workflow:run"}) {
		t.Errorf("cached features = %v, want [workflow:run]", got.Features)
	}
	if !reflect.DeepEqual(got.Limits, []Limit{{Key: "runs", Value: 4}}) {
		t.Errorf("cached limits = %v, want runs=4", got.Limits)
	}
	wantEntitlement := []AppEntitlement{{
		AppKey: "demo", Status: "active", Features: []string{"workflow:run"}, Limits: []Limit{{Key: "runs", Value: 4}},
	}}
	if !reflect.DeepEqual(got.AppEntitlements, wantEntitlement) {
		t.Errorf("cached app entitlements = %v, want %v", got.AppEntitlements, wantEntitlement)
	}
	if !reflect.DeepEqual(got.Credits, credits) {
		t.Errorf("cached credits = %#v, want %#v", got.Credits, credits)
	}
	if !reflect.DeepEqual(got.Subscription, subscription) {
		t.Errorf("cached subscription = %#v, want %#v", got.Subscription, subscription)
	}
}

func split(t *testing.T, token string) []string {
	t.Helper()
	parts := make([]string, 0, 3)
	start := 0
	for i, char := range token {
		if char == '.' {
			parts = append(parts, token[start:i])
			start = i + 1
		}
	}
	parts = append(parts, token[start:])
	if len(parts) != 3 {
		t.Fatal("invalid token")
	}
	return parts
}
