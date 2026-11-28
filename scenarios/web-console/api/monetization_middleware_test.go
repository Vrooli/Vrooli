package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/consumeridentity"
	entitlementclient "github.com/vrooli/vrooli/packages/entitlementclient-go"
	monetization "github.com/vrooli/vrooli/packages/monetization-go"
)

type webConsoleTestIssuerKeys struct{ jwks []byte }

func (s webConsoleTestIssuerKeys) Load(context.Context, string) ([]byte, error) {
	return append([]byte(nil), s.jwks...), nil
}

func (s *webConsoleTestIssuerKeys) Save(_ context.Context, _ string, jwks []byte) error {
	s.jwks = append([]byte(nil), jwks...)
	return nil
}

func TestWebConsoleInstallsSharedMonetizationRequestContext(t *testing.T) {
	router := mux.NewRouter()
	installMonetizationRequestContext(router)
	router.HandleFunc("/probe", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(monetization.AccessTokenFromContext(r.Context())))
	})

	request := httptest.NewRequest(http.MethodGet, "/probe", nil)
	request.Header.Set("Authorization", "Bearer shared-session-token")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Body.String() != "shared-session-token" {
		t.Fatalf("middleware result = status %d body %q", recorder.Code, recorder.Body.String())
	}
}

func TestWebConsoleLeaseStoreSurvivesOfflineServerRestart(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("APP_DATA_DIR", dataDir)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keys := consumeridentity.NewKeySet(consumeridentity.PublicKey{ID: "web-console-test", Key: &key.PublicKey})
	publicKeys, err := keys.JWKS()
	if err != nil {
		t.Fatal(err)
	}
	lease, err := entitlementclient.Sign(entitlementclient.Payload{
		UserIdentity: "member@example.test", Status: "active", Features: []string{"ai"},
		NotAfter: time.Now().UTC().Add(time.Hour),
	}, "web-console-test", key)
	if err != nil {
		t.Fatal(err)
	}
	firstStore, err := newWebConsoleLeaseStore()
	if err != nil {
		t.Fatal(err)
	}
	if err := firstStore.Save("member@example.test", entitlementclient.LeaseRecord{Lease: lease}); err != nil {
		t.Fatal(err)
	}

	// Build a fresh API-side client and file store. LPBS is intentionally not
	// configured as a resolver; only the separately trusted issuer key survives.
	restartedStore, err := newWebConsoleLeaseStore()
	if err != nil {
		t.Fatal(err)
	}
	issuerKeys := &webConsoleTestIssuerKeys{jwks: publicKeys}
	client := entitlementclient.NewClient("https://lpbs.example.test", nil, nil)
	client.SetStore(restartedStore)
	client.SetIssuerKeyStore(issuerKeys)
	got, err := client.CachedAt("MEMBER@example.test", time.Now().UTC())
	if err != nil || got.Status != "active" || got.UserIdentity != "member@example.test" {
		t.Fatalf("offline lease after restart = %+v, %v", got, err)
	}
	// The persisted record is data only: it must not contain issuer keys.
	files, err := os.ReadDir(filepath.Join(dataDir, "monetization", "leases"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("persisted lease file count = %d, want 1", len(files))
	}
	body, err := os.ReadFile(filepath.Join(dataDir, "monetization", "leases", files[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	var persisted map[string]json.RawMessage
	if err := json.Unmarshal(body, &persisted); err != nil {
		t.Fatal(err)
	}
	if len(persisted) != 1 || persisted["lease"] == nil {
		t.Fatalf("persisted lease record fields = %v, want only signed lease data", persisted)
	}
}
