package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/consumeridentity"
	"github.com/vrooli/vrooli/packages/entitlementclient-go"
	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/ai"
	aiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/ai/ai_v1connect"
)

type packagedMonetizationEvidence struct {
	SchemaVersion         int      `json:"schema_version"`
	Scenario              string   `json:"scenario"`
	ArtifactPath          string   `json:"artifact_path"`
	ArtifactDigest        string   `json:"artifact_digest"`
	TargetID              string   `json:"target_id"`
	RootAPIOperation      string   `json:"root_api_operation"`
	DisabledSources       []string `json:"disabled_sources"`
	VrooliSourceEnabled   bool     `json:"vrooli_source_enabled"`
	IdentityHint          string   `json:"identity_hint"`
	HTTPStatus            int      `json:"http_status"`
	ErrorType             string   `json:"error_type"`
	UnauthenticatedMeter  bool     `json:"unauthenticated_meter_refused"`
	Phase                 string   `json:"phase,omitempty"`
	Identity              string   `json:"identity,omitempty"`
	LeaseStatus           string   `json:"lease_status,omitempty"`
	SignedSessionAccepted bool     `json:"signed_session_accepted,omitempty"`
	SessionConfigured     bool     `json:"session_configured,omitempty"`
	PersistedLeaseValid   bool     `json:"persisted_lease_valid,omitempty"`
	LPBSUnavailable       bool     `json:"lpbs_unavailable,omitempty"`
	OfflineSummaryStatus  int      `json:"offline_summary_status,omitempty"`
	OfflineSummaryError   string   `json:"offline_summary_error,omitempty"`
}

// TestWebConsolePackagedMeteringRefusal runs only inside the S2D semantic
// matrix. S2D starts the exact AppImage in an isolated offline runtime and
// passes its private root-API bridge; this test mutates only that app's
// disposable provider configuration and sends a non-dispatching attack.
func TestWebConsolePackagedMeteringRefusal(t *testing.T) {
	artifactPath := strings.TrimSpace(os.Getenv("WC_MONETIZATION_APPIMAGE_PATH"))
	if artifactPath == "" {
		t.Skip("packaged matrix supplies WC_MONETIZATION_APPIMAGE_PATH")
	}
	apiBase := strings.TrimRight(strings.TrimSpace(os.Getenv("WC_MONETIZATION_API_URL")), "/")
	artifactDigest := normalizePackagedDigest(os.Getenv("WC_MONETIZATION_ARTIFACT_DIGEST"))
	targetID := strings.TrimSpace(os.Getenv("WC_MONETIZATION_TARGET_ID"))
	evidencePath := strings.TrimSpace(os.Getenv("WC_MONETIZATION_EVIDENCE_PATH"))
	if apiBase == "" || artifactDigest == "" || targetID == "" || evidencePath == "" {
		t.Fatal("packaged matrix requires API URL, artifact digest, target ID, and evidence path")
	}
	parsed, err := url.Parse(apiBase)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || !isPackagedLoopback(parsed.Hostname()) {
		t.Fatalf("packaged API bridge must be loopback HTTP(S), got %q", apiBase)
	}
	artifactPath, err = filepath.Abs(artifactPath)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := os.Open(artifactPath)
	if err != nil {
		t.Fatalf("open selected artifact: %v", err)
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, artifact); err != nil {
		_ = artifact.Close()
		t.Fatalf("hash selected artifact: %v", err)
	}
	if err := artifact.Close(); err != nil {
		t.Fatal(err)
	}
	if got := "sha256:" + hex.EncodeToString(hash.Sum(nil)); got != artifactDigest {
		t.Fatalf("artifact digest mismatch: selected %s, expected %s", got, artifactDigest)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client := aiconnect.NewAIServiceClient(&http.Client{Timeout: 10 * time.Second}, apiBase)
	for _, source := range []string{"ollama", "openrouter"} {
		_, err := client.UpdateConfig(ctx, connect.NewRequest(&aiv1.UpdateConfigRequest{
			Name: source, Enabled: false, HasEnabled: true,
		}))
		if err != nil {
			t.Fatalf("disable isolated %s source: %v", source, err)
		}
	}
	configuration, err := client.GetConfig(ctx, connect.NewRequest(&aiv1.GetConfigRequest{}))
	if err != nil {
		t.Fatalf("read isolated provider configuration: %v", err)
	}
	for _, source := range []string{"ollama", "openrouter"} {
		foundDisabled := false
		for _, provider := range configuration.Msg.GetProviders() {
			if provider.GetName() == source && !provider.GetEnabled() {
				foundDisabled = true
			}
		}
		if !foundDisabled {
			t.Fatalf("isolated source %q was not confirmed disabled", source)
		}
	}
	vrooliEnabled := false
	for _, provider := range configuration.Msg.GetProviders() {
		if provider.GetName() == "vrooli" && provider.GetEnabled() {
			vrooliEnabled = true
		}
	}
	if !vrooliEnabled {
		t.Fatal("Vrooli credits provider is missing or disabled in the packaged app configuration")
	}

	phase := strings.ToLower(strings.TrimSpace(os.Getenv("WC_MONETIZATION_PHASE")))
	if phase == "signed-online" {
		writePackagedMonetizationEvidence(t, evidencePath, runPackagedSignedSessionOnline(t, ctx, apiBase, artifactPath, artifactDigest, targetID))
		return
	}
	if phase == "offline-restart" {
		previous := readPackagedOnlineEvidence(t, os.Getenv("WC_MONETIZATION_PREVIOUS_EVIDENCE_PATH"), artifactPath, artifactDigest, targetID)
		writePackagedMonetizationEvidence(t, evidencePath, runPackagedOfflineRestart(t, ctx, apiBase, artifactPath, artifactDigest, targetID, previous))
		return
	}

	body, _ := json.Marshal(map[string]string{"prompt": "list files"})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBase+"/api/v1/ai/generate", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-User-Identity", "forged@example.invalid")
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil {
		t.Fatalf("call bundled root API: %v", err)
	}
	defer response.Body.Close()
	var refusal struct {
		ErrorType string `json:"error_type"`
	}
	if err := json.NewDecoder(response.Body).Decode(&refusal); err != nil {
		t.Fatalf("decode bundled root API response: %v", err)
	}
	if response.StatusCode != http.StatusPaymentRequired || refusal.ErrorType != "credits_required" {
		t.Fatalf("forged identity hint outcome = HTTP %d %q, want HTTP 402 credits_required", response.StatusCode, refusal.ErrorType)
	}

	evidence := packagedMonetizationEvidence{
		SchemaVersion: 1, Scenario: "web-console", ArtifactPath: artifactPath,
		ArtifactDigest: artifactDigest, TargetID: targetID,
		RootAPIOperation: "POST /api/v1/ai/generate", DisabledSources: []string{"ollama", "openrouter"},
		VrooliSourceEnabled: true,
		IdentityHint:        "forged@example.invalid", HTTPStatus: response.StatusCode,
		ErrorType: refusal.ErrorType, UnauthenticatedMeter: true,
	}
	writePackagedMonetizationEvidence(t, evidencePath, evidence)
}

const packagedMonetizationIdentity = "sp1-web-console@example.invalid"

func runPackagedSignedSessionOnline(t *testing.T, ctx context.Context, apiBase, artifactPath, artifactDigest, targetID string) packagedMonetizationEvidence {
	t.Helper()
	startPackagedLPBSFixture(t)
	client := &http.Client{Timeout: 10 * time.Second}
	var provisioned map[string]any
	postPackagedJSON(t, ctx, client, apiBase+"/api/v1/auth/subscription/session", map[string]string{"refresh_token": "sp1-fixture-refresh-token"}, http.StatusCreated, &provisioned)
	if provisioned["configured"] != true {
		t.Fatalf("bundled API did not confirm subscription credential provisioning: %+v", provisioned)
	}
	var summary struct {
		Configured bool   `json:"configured"`
		Status     string `json:"status"`
		PlanTier   string `json:"plan_tier"`
	}
	getPackagedJSON(t, ctx, client, apiBase+"/api/v1/auth/subscription/summary", http.StatusOK, &summary)
	if !summary.Configured || summary.Status != "active" {
		t.Fatalf("bundled LPBS-signed account summary = %+v, want active signed entitlement", summary)
	}
	return packagedMonetizationEvidence{
		SchemaVersion: 1, Scenario: "web-console", ArtifactPath: artifactPath,
		ArtifactDigest: artifactDigest, TargetID: targetID,
		RootAPIOperation: "POST /api/v1/auth/subscription/session; GET /api/v1/auth/subscription/summary",
		DisabledSources:  []string{"ollama", "openrouter"}, VrooliSourceEnabled: true,
		Phase: "signed-online", Identity: packagedMonetizationIdentity, SessionConfigured: true,
		LeaseStatus: summary.Status, SignedSessionAccepted: true,
	}
}

func startPackagedLPBSFixture(t *testing.T) {
	t.Helper()
	rawListenAddress := strings.TrimSpace(os.Getenv("WC_MONETIZATION_UPSTREAM_LISTEN_ADDR"))
	appDataDir := strings.TrimSpace(os.Getenv("WC_MONETIZATION_APP_DATA_DIR"))
	jwksPath := strings.TrimSpace(os.Getenv("WC_MONETIZATION_JWKS_PATH"))
	if rawListenAddress == "" || appDataDir == "" || jwksPath == "" {
		t.Fatal("signed-online phase requires loopback upstream, bundled APP_DATA_DIR, and public JWKS paths")
	}
	listenURL, err := url.Parse(rawListenAddress)
	if err != nil || listenURL.Scheme != "http" || !isPackagedLoopback(listenURL.Hostname()) || listenURL.Port() == "" {
		t.Fatalf("LPBS fixture listen address must be a numeric loopback HTTP origin, got %q", rawListenAddress)
	}
	listener, err := net.Listen("tcp", listenURL.Host)
	if err != nil {
		t.Fatalf("listen for isolated LPBS fixture: %v", err)
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		_ = listener.Close()
		t.Fatalf("generate fixture lease key: %v", err)
	}
	keys := consumeridentity.NewKeySet(consumeridentity.PublicKey{ID: "sp1-web-console-fixture", Key: &key.PublicKey})
	jwks, err := keys.JWKS()
	if err != nil {
		_ = listener.Close()
		t.Fatalf("encode fixture issuer keys: %v", err)
	}
	lease, err := entitlementclient.Sign(entitlementclient.Payload{
		UserIdentity: packagedMonetizationIdentity, Status: "active", PlanTier: "business",
		Features: []string{"ai"}, Limits: []entitlementclient.Limit{{Key: "ai_credits", Value: 50}},
		Credits: map[string]any{"balance_credits": 50.0}, NotAfter: time.Now().UTC().Add(time.Hour),
	}, "sp1-web-console-fixture", key)
	if err != nil {
		_ = listener.Close()
		t.Fatalf("sign fixture entitlement lease: %v", err)
	}
	if err := os.MkdirAll(appDataDir, 0o700); err != nil {
		_ = listener.Close()
		t.Fatalf("create isolated bundled app data directory: %v", err)
	}
	if err := os.WriteFile(jwksPath, jwks, 0o600); err != nil {
		_ = listener.Close()
		t.Fatalf("persist public fixture JWKS for offline verification: %v", err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/.well-known/jwks.json":
			_, _ = w.Write(jwks)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/auth/refresh":
			var request struct {
				RefreshToken string `json:"refresh_token"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil || (request.RefreshToken != "sp1-fixture-refresh-token" && request.RefreshToken != "sp1-fixture-refresh-token-rotated") {
				http.Error(w, "fixture refresh token refused", http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{
				"access_token": "sp1-fixture-access-token", "refresh_token": "sp1-fixture-refresh-token-rotated",
				"expires_at": time.Now().UTC().Add(10 * time.Minute).Format(time.RFC3339),
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/auth/me":
			if r.Header.Get("Authorization") != "Bearer sp1-fixture-access-token" {
				http.Error(w, "fixture access token refused", http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"user": map[string]string{"email": packagedMonetizationIdentity}})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/entitlements":
			if r.Header.Get("Authorization") != "Bearer sp1-fixture-access-token" || r.URL.Query().Get("user") != packagedMonetizationIdentity {
				http.Error(w, "fixture entitlement request refused", http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"lease": lease})
		default:
			http.NotFound(w, r)
		}
	})}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() {
		if err := server.Close(); err != nil && err != http.ErrServerClosed {
			t.Errorf("close isolated LPBS fixture: %v", err)
		}
	})
	t.Logf("isolated LPBS fixture listening at %s", fmt.Sprintf("http://%s", listener.Addr()))
}

func runPackagedOfflineRestart(t *testing.T, ctx context.Context, apiBase, artifactPath, artifactDigest, targetID string, previous packagedMonetizationEvidence) packagedMonetizationEvidence {
	t.Helper()
	client := &http.Client{Timeout: 10 * time.Second}
	summaryResponse, err := client.Get(apiBase + "/api/v1/auth/subscription/summary")
	if err != nil {
		t.Fatalf("request offline account summary: %v", err)
	}
	defer summaryResponse.Body.Close()
	summaryBody, readErr := io.ReadAll(io.LimitReader(summaryResponse.Body, 8<<10))
	if readErr != nil {
		t.Fatalf("read offline account summary response: %v", readErr)
	}
	if summaryResponse.StatusCode != http.StatusUnauthorized {
		t.Fatalf("offline account summary status = %d, want HTTP 401 with LPBS unavailable: %s", summaryResponse.StatusCode, strings.TrimSpace(string(summaryBody)))
	}
	if got := strings.TrimSpace(string(summaryBody)); got != "subscription session is not configured" {
		t.Fatalf("offline account summary body = %q, want resolver failure while LPBS is unavailable", got)
	}
	var session struct {
		Configured bool `json:"configured"`
	}
	getPackagedJSON(t, ctx, client, apiBase+"/api/v1/auth/subscription/session", http.StatusOK, &session)
	if !session.Configured {
		t.Fatal("subscription credential did not survive the bundled API restart")
	}

	dataDir := strings.TrimSpace(os.Getenv("WC_MONETIZATION_APP_DATA_DIR"))
	keyPath := strings.TrimSpace(os.Getenv("WC_MONETIZATION_JWKS_PATH"))
	if dataDir == "" || keyPath == "" {
		t.Fatal("offline-restart phase requires the isolated bundled APP_DATA_DIR and fixture JWKS path")
	}
	keyDocument, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("read fixture issuer JWKS: %v", err)
	}
	keys, err := consumeridentity.ParseJWKS(keyDocument)
	if err != nil {
		t.Fatalf("parse fixture issuer JWKS: %v", err)
	}
	store, err := entitlementclient.NewFileLeaseStore(filepath.Join(dataDir, "monetization", "leases"))
	if err != nil {
		t.Fatalf("open isolated persisted lease directory: %v", err)
	}
	record, err := store.Load(packagedMonetizationIdentity)
	if err != nil {
		t.Fatalf("load signed lease after API restart: %v", err)
	}
	lease, err := entitlementclient.Verify(record.Lease, keys, time.Now().UTC())
	if err != nil || !strings.EqualFold(lease.UserIdentity, packagedMonetizationIdentity) || lease.Status != "active" {
		t.Fatalf("persisted offline lease = identity %q, status %q, verify error %v", lease.UserIdentity, lease.Status, err)
	}

	body, _ := json.Marshal(map[string]string{"prompt": "list files"})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBase+"/api/v1/ai/generate", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-User-Identity", packagedMonetizationIdentity)
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("call offline bundled root API: %v", err)
	}
	defer response.Body.Close()
	var refusal struct {
		ErrorType string `json:"error_type"`
	}
	if err := json.NewDecoder(response.Body).Decode(&refusal); err != nil {
		t.Fatalf("decode offline bundled root API response: %v", err)
	}
	if response.StatusCode != http.StatusPaymentRequired || refusal.ErrorType != "credits_required" {
		t.Fatalf("offline cached lease decision = HTTP %d %q, want HTTP 402 credits_required", response.StatusCode, refusal.ErrorType)
	}
	return packagedMonetizationEvidence{
		SchemaVersion: 1, Scenario: "web-console", ArtifactPath: artifactPath,
		ArtifactDigest: artifactDigest, TargetID: targetID,
		RootAPIOperation: "GET /api/v1/auth/subscription/session; POST /api/v1/ai/generate",
		DisabledSources:  []string{"ollama", "openrouter"}, VrooliSourceEnabled: true,
		Phase: "offline-restart", Identity: packagedMonetizationIdentity,
		LeaseStatus: lease.Status, SignedSessionAccepted: previous.SignedSessionAccepted, SessionConfigured: true,
		PersistedLeaseValid: true, HTTPStatus: response.StatusCode,
		ErrorType: refusal.ErrorType, UnauthenticatedMeter: true,
		LPBSUnavailable: true, OfflineSummaryStatus: summaryResponse.StatusCode,
		OfflineSummaryError: strings.TrimSpace(string(summaryBody)),
	}
}

func readPackagedOnlineEvidence(t *testing.T, path, artifactPath, artifactDigest, targetID string) packagedMonetizationEvidence {
	t.Helper()
	if strings.TrimSpace(path) == "" {
		t.Fatal("offline-restart phase requires the signed-online evidence path")
	}
	encoded, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read signed-online evidence: %v", err)
	}
	var previous packagedMonetizationEvidence
	if err := json.Unmarshal(encoded, &previous); err != nil {
		t.Fatalf("decode signed-online evidence: %v", err)
	}
	if previous.Phase != "signed-online" || !previous.SignedSessionAccepted || previous.ArtifactPath != artifactPath || previous.ArtifactDigest != artifactDigest || previous.TargetID != targetID || previous.Identity != packagedMonetizationIdentity {
		t.Fatalf("signed-online evidence does not bind this app restart: %+v", previous)
	}
	return previous
}

func getPackagedJSON(t *testing.T, ctx context.Context, client *http.Client, url string, wantStatus int, out any) {
	t.Helper()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer response.Body.Close()
	if response.StatusCode != wantStatus {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("GET %s status = %d, want %d: %s", url, response.StatusCode, wantStatus, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(response.Body).Decode(out); err != nil {
		t.Fatalf("decode GET %s: %v", url, err)
	}
}

func postPackagedJSON(t *testing.T, ctx context.Context, client *http.Client, url string, body any, wantStatus int, out any) {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	defer response.Body.Close()
	if response.StatusCode != wantStatus {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("POST %s status = %d, want %d: %s", url, response.StatusCode, wantStatus, strings.TrimSpace(string(body)))
	}
	if out != nil {
		if err := json.NewDecoder(response.Body).Decode(out); err != nil {
			t.Fatalf("decode POST %s: %v", url, err)
		}
	}
}

func writePackagedMonetizationEvidence(t *testing.T, path string, evidence packagedMonetizationEvidence) {
	t.Helper()
	encoded, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(encoded, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

func normalizePackagedDigest(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if !strings.HasPrefix(value, "sha256:") {
		value = "sha256:" + value
	}
	if len(value) != len("sha256:")+64 {
		return ""
	}
	if _, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:")); err != nil {
		return ""
	}
	return value
}

func isPackagedLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}
