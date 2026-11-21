package identity

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/consumeridentity"
	"github.com/vrooli/api-core/database"
	entitlementclient "github.com/vrooli/vrooli/packages/entitlementclient-go"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing/routing_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	_ "modernc.org/sqlite"

	inferenceHTTP "ai-gateway/handlers/inference"
	consentHTTP "ai-gateway/handlers/privacyconsent"
	routingHTTP "ai-gateway/handlers/routing"
	internalinference "ai-gateway/internal/inference"
	"ai-gateway/internal/privacyconsent"
	"ai-gateway/internal/providers"
	providerMocks "ai-gateway/internal/providers/mocks"
	"ai-gateway/internal/rolecatalog"
	internalrouting "ai-gateway/internal/routing"
)

// This crosses the same boundaries as the running Gateway: API Core verifies
// an LPBS-signed account token, the consent HTTP handler derives the owner from
// that principal, SQLite persists the opaque receipt digest, and Connect
// routing checks it immediately before dispatch to a fake provider.
func TestGatewayConsentLifecycleAndDispatchAcrossVerifiedHTTPBoundary(t *testing.T) {
	ctx := context.Background()
	clock := time.Now().UTC().Truncate(time.Second)
	issuer := "https://lpbs.consent-fixture.invalid"

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	signer, err := consumeridentity.NewSigner("consent-fixture-key", privateKey, issuer, time.Hour)
	require.NoError(t, err)
	jwks, err := consumeridentity.NewKeySet(consumeridentity.PublicKey{ID: signer.KeyID, Key: &privateKey.PublicKey}).JWKS()
	require.NoError(t, err)
	provider := NewLPBSConsumerProvider(issuer)
	provider.cache = consumeridentity.NewCache(issuer, 0, time.Hour, func(*http.Request) ([]byte, error) { return jwks, nil })
	identityConfig := authn.Config{Providers: []authn.Provider{provider}}

	newToken := func(subject, email string) string {
		token, _, signErr := signer.Sign(consumeridentity.Claims{UserID: subject, Subject: subject, Email: email})
		require.NoError(t, signErr)
		return token
	}
	aliceSubject := "fixture-account-alice-" + randomConsentTestID(t)
	bobSubject := "fixture-account-bob-" + randomConsentTestID(t)
	aliceToken := newToken(aliceSubject, "alice@fixture.invalid")
	bobToken := newToken(bobSubject, "bob@fixture.invalid")

	dbPath := filepath.Join(t.TempDir(), "ai-gateway-consent.sqlite")
	openDB := func() *sql.DB {
		db, openErr := sql.Open("sqlite", dbPath)
		require.NoError(t, openErr)
		db.SetMaxOpenConns(1)
		require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(internalrouting.Schema)))
		return db
	}
	db := openDB()
	consentStore, err := privacyconsent.New(db, func() time.Time { return clock })
	require.NoError(t, err)

	runner := &providerMocks.FakeRunner{
		Results: map[string]providers.Result{
			"fixture-openrouter policy roles --json": {
				Stdout: `{"roles":[{"role":"chat.default","model":"fixture/openrouter","required_capabilities":["generate","chat"],"locality":"remote"}]}`,
			},
			"fixture-openrouter policy resolve --role chat.default --json": {
				Stdout: `{"role":"chat.default","model":"fixture/openrouter"}`,
			},
		},
		DefaultOK: true,
		Default:   providers.Result{Stdout: `{"response":"fake OpenRouter response","eval_count":3}`},
	}
	service := internalrouting.NewService(
		[]providers.Adapter{{Provider: providers.ProviderOpenRouter, CommandName: "fixture-openrouter", Locality: "remote", Runner: runner}},
		internalrouting.NewSQLRepository(db),
		internalrouting.WithRemoteConsentVerifier(consentStore),
	)
	api := newConsentGatewayHTTPServer(t, db, consentStore, service, identityConfig)
	t.Cleanup(api.Close)

	grant := issueConsentGrant(t, api.Client(), api.URL, aliceToken, "own-key", 90)
	creditsGrant := issueConsentGrant(t, api.Client(), api.URL, aliceToken, "vrooli-credits", 90)
	revokedGrant := issueConsentGrant(t, api.Client(), api.URL, aliceToken, "own-key", 90)
	expiredReceipt, _, err := consentStore.Grant(ctx, aliceSubject, "own-key", time.Minute)
	require.NoError(t, err)
	revokeResponse := sendConsentRequest(t, api.Client(), http.MethodDelete, api.URL+"/api/v1/ai/privacy-consent/"+revokedGrant.Grant.ID, aliceToken, "", "")
	require.Equal(t, http.StatusNoContent, revokeResponse.StatusCode)
	_ = mustReadBody(t, revokeResponse)
	listedBeforeRestart := listConsentGrants(t, api.Client(), api.URL, aliceToken)
	require.Contains(t, grantIDs(listedBeforeRestart), grant.Grant.ID)
	require.Contains(t, grantIDs(listedBeforeRestart), revokedGrant.Grant.ID)
	require.NotContains(t, mustReadBody(t, sendConsentRequest(t, api.Client(), http.MethodGet, api.URL+"/api/v1/ai/privacy-consent", aliceToken, "", "")), grant.Receipt, "list must never return a bearer receipt")
	api.Close()
	require.NoError(t, db.Close())

	// Reopen the file-backed store and mount the production HTTP handlers again.
	// The grants and revocation state must survive this process-like boundary.
	clock = clock.Add(2 * time.Minute)
	db = openDB()
	consentStore, err = privacyconsent.New(db, func() time.Time { return clock })
	require.NoError(t, err)
	service = internalrouting.NewService(
		[]providers.Adapter{{Provider: providers.ProviderOpenRouter, CommandName: "fixture-openrouter", Locality: "remote", Runner: runner}},
		internalrouting.NewSQLRepository(db),
		internalrouting.WithRemoteConsentVerifier(consentStore),
	)
	api = newConsentGatewayHTTPServer(t, db, consentStore, service, identityConfig)
	t.Cleanup(func() { _ = db.Close() })
	t.Cleanup(api.Close)

	listedAfterRestart := listConsentGrants(t, api.Client(), api.URL, aliceToken)
	require.Contains(t, grantIDs(listedAfterRestart), grant.Grant.ID, "issued grant must persist across DB reopen")
	require.Contains(t, grantIDs(listedAfterRestart), revokedGrant.Grant.ID, "revoked grant record must persist")
	for _, item := range listedAfterRestart {
		if item.ID == revokedGrant.Grant.ID {
			require.NotNil(t, item.RevokedAt, "revocation must persist across DB reopen")
		}
	}

	client := routingconnect.NewRoutingServiceClient(api.Client(), api.URL)
	request := func(token, source, receipt string, privacy sharedv1.PrivacyClass) *connect.Request[routingv1.ExecuteRouteRequest] {
		r := connect.NewRequest(&routingv1.ExecuteRouteRequest{
			Request: &sharedv1.GatewayRequest{
				Kind: sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION, Role: "chat.default",
				Profile: sharedv1.Profile_PROFILE_REMOTE_ONLY, PrivacyClass: privacy,
				PrivacyProvenance: "consent_gateway_integration_test.signed_fixture",
				Scenario:          "consent-fixture", Operation: "confidential-test", RequestId: randomConsentTestID(t), MaxOutputTokens: 16,
			},
			InputText: "fixture confidential prompt",
		})
		r.Header().Set("Authorization", "Bearer "+token)
		r.Header().Set("X-Vrooli-AI-Source", source)
		if receipt != "" {
			r.Header().Set("X-Vrooli-AI-Consent-Grant", receipt)
		}
		return r
	}
	dispatches := func() int {
		count := 0
		for _, command := range runner.Commands {
			if strings.Contains(command.String(), " generate ") {
				count++
			}
		}
		return count
	}
	execute := func(token, source, receipt string, privacy sharedv1.PrivacyClass) *routingv1.ExecuteRouteResponse {
		resp, callErr := client.ExecuteRoute(ctx, request(token, source, receipt, privacy))
		require.NoError(t, callErr)
		return resp.Msg
	}
	assertRefusedBeforeDispatch := func(name, token, source, receipt string) {
		t.Run(name, func(t *testing.T) {
			before := dispatches()
			resp := execute(token, source, receipt, sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL)
			require.NotEqual(t, "succeeded", resp.GetEvidence().GetStatus())
			require.Equal(t, before, dispatches(), "invalid or mismatched consent must be refused before fake provider dispatch")
		})
	}

	assertRefusedBeforeDispatch("missing receipt", aliceToken, "own-key", "")
	assertRefusedBeforeDispatch("forged receipt", aliceToken, "own-key", strings.Repeat("x", 48))
	assertRefusedBeforeDispatch("wrong user", bobToken, "own-key", grant.Receipt)
	assertRefusedBeforeDispatch("wrong source", aliceToken, "own-key", creditsGrant.Receipt)
	assertRefusedBeforeDispatch("expired grant", aliceToken, "own-key", expiredReceipt)
	assertRefusedBeforeDispatch("revoked grant", aliceToken, "own-key", revokedGrant.Receipt)

	beforeValid := dispatches()
	valid := execute(aliceToken, "own-key", grant.Receipt, sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL)
	require.Equal(t, "succeeded", valid.GetEvidence().GetStatus())
	require.Equal(t, providers.ProviderOpenRouter, valid.GetEvidence().GetSelectedProvider())
	require.Equal(t, beforeValid+1, dispatches(), "valid consent may dispatch once to the selected fake source; commands: "+fmt.Sprint(runner.Commands))

	beforeSecret := dispatches()
	secret := execute(aliceToken, "own-key", grant.Receipt, sharedv1.PrivacyClass_PRIVACY_CLASS_SECRET)
	require.NotEqual(t, "succeeded", secret.GetEvidence().GetStatus())
	require.Equal(t, beforeSecret, dispatches(), "SECRET remains local-only even with a valid grant")
}

// TestGatewayConsentBrowserJourney is an opt-in browser integration because
// Chrome/Vite are intentionally not prerequisites of the normal Go unit phase.
// It executes the real consent panel against the real HTTP/Connect handlers,
// fixture-signed identities, file-backed SQLite authority, and fake adapter.
func TestGatewayConsentBrowserJourney(t *testing.T) {
	if os.Getenv("A15_CONSENT_BROWSER") != "1" {
		t.Skip("set A15_CONSENT_BROWSER=1 to run the local fixture-only browser journey")
	}
	ctx := context.Background()
	clockNanos := &atomic.Int64{}
	clockNanos.Store(time.Now().UTC().Truncate(time.Second).UnixNano())
	now := func() time.Time { return time.Unix(0, clockNanos.Load()).UTC() }
	issuer := "https://lpbs.browser-consent-fixture.invalid"
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	signer, err := consumeridentity.NewSigner("consent-browser-fixture-key", privateKey, issuer, time.Hour)
	require.NoError(t, err)
	jwks, err := consumeridentity.NewKeySet(consumeridentity.PublicKey{ID: signer.KeyID, Key: &privateKey.PublicKey}).JWKS()
	require.NoError(t, err)
	provider := NewLPBSConsumerProvider(issuer)
	provider.cache = consumeridentity.NewCache(issuer, 0, time.Hour, func(*http.Request) ([]byte, error) { return jwks, nil })
	identityConfig := authn.Config{Providers: []authn.Provider{provider}}
	makeToken := func(subject string) string {
		token, _, signErr := signer.Sign(consumeridentity.Claims{UserID: subject, Subject: subject, Email: subject + "@fixture.invalid"})
		require.NoError(t, signErr)
		return token
	}
	alice := "fixture-browser-alice-" + randomConsentTestID(t)
	aliceToken, bobToken := makeToken(alice), makeToken("fixture-browser-bob-"+randomConsentTestID(t))
	dbPath := filepath.Join(t.TempDir(), "gateway-consent-browser.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(internalrouting.Schema)))
	store, err := privacyconsent.New(db, now)
	require.NoError(t, err)
	runner := &providerMocks.FakeRunner{
		Results: map[string]providers.Result{
			"fixture-openrouter policy roles --json":                       {Stdout: `{"roles":[{"role":"chat.default","model":"fixture/openrouter","required_capabilities":["generate","chat"],"locality":"remote"}]}`},
			"fixture-openrouter policy resolve --role chat.default --json": {Stdout: `{"role":"chat.default","model":"fixture/openrouter"}`},
		},
		DefaultOK: true,
		Default:   providers.Result{Stdout: `{"response":"fixture browser response","eval_count":1}`},
	}
	inferenceRunner := &a15ConsentInferenceRunner{}
	dispatches := func() int {
		count := 0
		for _, command := range append(append([]providers.Command(nil), runner.Commands...), inferenceRunner.CommandSnapshot()...) {
			if strings.Contains(command.String(), " generate ") {
				count++
			}
		}
		return count
	}
	inferenceCatalog := rolecatalog.RoleCatalog{SchemaVersion: 1, Roles: map[string]rolecatalog.InferenceRole{
		"write.default":      {Candidates: []rolecatalog.Candidate{{Provider: providers.ProviderOpenRouter, ResourceRole: "write.default", Reason: "fixture remote provider"}}},
		"extract.structured": {Candidates: []rolecatalog.Candidate{{Provider: providers.ProviderOpenRouter, ResourceRole: "extract.structured", Reason: "fixture remote provider"}}},
	}}
	inferenceRepository, err := internalinference.NewResourceRepository(inferenceCatalog, []providers.Adapter{{Provider: providers.ProviderOpenRouter, CommandName: "fixture-inference-openrouter", Locality: "remote", Runner: inferenceRunner}})
	require.NoError(t, err)
	inferenceRepository.SetRemoteConsentVerifier(store)
	inferenceService := internalinference.NewService(inferenceRepository)
	service := internalrouting.NewService(
		[]providers.Adapter{{Provider: providers.ProviderOpenRouter, CommandName: "fixture-openrouter", Locality: "remote", Runner: runner}},
		internalrouting.NewSQLRepository(db), internalrouting.WithRemoteConsentVerifier(store),
	)
	router := mux.NewRouter()
	consentHandler := consentHTTP.New(store)
	router.Handle("/api/v1/ai/privacy-consent", consentHandler)
	router.PathPrefix("/api/v1/ai/privacy-consent/").Handler(consentHandler)
	routingHTTP.Module(routingHTTP.Deps{DB: db, Service: service}).Mount(router)
	inferenceHTTP.Module(inferenceHTTP.Deps{Service: inferenceService}).Mount(router)
	router.HandleFunc("/__fixture/stats", func(w http.ResponseWriter, _ *http.Request) {
		count := 0
		for _, command := range append(append([]providers.Command(nil), runner.Commands...), inferenceRunner.CommandSnapshot()...) {
			if strings.Contains(command.String(), " generate ") {
				count++
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]int{"dispatches": count})
	}).Methods(http.MethodGet)
	router.HandleFunc("/__fixture/expired", func(w http.ResponseWriter, r *http.Request) {
		clockNanos.Add(int64(2 * time.Minute))
		receipt, _, grantErr := store.Grant(r.Context(), alice, "own-key", time.Minute)
		clockNanos.Add(int64(2 * time.Minute))
		if grantErr != nil {
			http.Error(w, grantErr.Error(), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"receipt": receipt})
	}).Methods(http.MethodPost)
	server := httptest.NewServer(authn.Middleware(identityConfig)(router))
	t.Cleanup(server.Close)
	t.Cleanup(func() { _ = db.Close() })

	repoRoot := filepath.Clean(filepath.Join("..", "..", "..", "..", ".."))
	repoRoot, err = filepath.Abs(repoRoot)
	require.NoError(t, err)
	script := filepath.Join(repoRoot, "scenarios/browser-automation-studio/ui/tests/e2e/consent-gateway-browser.mjs")
	sidecarOnly := os.Getenv("A15_CONSENT_SIDECAR_ONLY") == "1"
	if !sidecarOnly {
		command := exec.CommandContext(ctx, "node", script)
		command.Dir = filepath.Dir(script)
		command.Env = append(os.Environ(), "CONSENT_GATEWAY_URL="+server.URL, "CONSENT_ALICE_TOKEN="+aliceToken, "CONSENT_BOB_TOKEN="+bobToken)
		output, err := command.CombinedOutput()
		require.NoError(t, err, "browser consent integration failed:\n%s", output)
		var report struct {
			Results []struct {
				Name     string `json:"name"`
				Text     string `json:"text"`
				Provider string `json:"provider"`
				Before   int    `json:"before"`
				After    int    `json:"after"`
			} `json:"results"`
			FinalStats struct {
				Dispatches int `json:"dispatches"`
			} `json:"finalStats"`
		}
		// The runner prints the compact result object after its evidence write.
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		var decoded bool
		for i := len(lines) - 1; i >= 0; i-- {
			if json.Unmarshal([]byte(strings.Join(lines[i:], "\n")), &report) == nil && len(report.Results) > 0 {
				decoded = true
				break
			}
		}
		require.True(t, decoded, "browser result JSON missing:\n%s", output)
		require.Len(t, report.Results, 8)
		for _, result := range report.Results {
			if result.Name == "valid-own-key" {
				require.Contains(t, result.Text, "Dispatched: succeeded")
				require.Equal(t, "openrouter", result.Provider)
				require.Equal(t, result.Before+1, result.After)
			} else {
				require.Contains(t, result.Text, "Rejected:")
				require.Equal(t, result.Before, result.After, "%s dispatched before refusal", result.Name)
			}
		}
		require.Equal(t, 1, report.FinalStats.Dispatches, "only the valid exact-source CONFIDENTIAL request may reach fake OpenRouter")

		// Keep the production-caller screenshot easy to read and ensure its grant
		// lifecycle is independent from the preceding direct-routing browser pass.
		for _, priorGrant := range listConsentGrants(t, server.Client(), server.URL, aliceToken) {
			if priorGrant.RevokedAt != nil {
				continue
			}
			response := sendConsentRequest(t, server.Client(), http.MethodDelete, server.URL+"/api/v1/ai/privacy-consent/"+priorGrant.ID, aliceToken, "", "")
			require.Equal(t, http.StatusNoContent, response.StatusCode)
			_ = mustReadBody(t, response)
		}
		basURL, stopBAS := startA15BASConsentCaller(t, server.URL)
		defer stopBAS()
		productionCommand := exec.CommandContext(ctx, "node", script)
		productionCommand.Dir = filepath.Dir(script)
		productionCommand.Env = append(os.Environ(), "CONSENT_GATEWAY_URL="+server.URL, "CONSENT_ALICE_TOKEN="+aliceToken, "CONSENT_BOB_TOKEN="+bobToken, "CONSENT_BAS_API_URL="+basURL)
		productionOutput, productionErr := productionCommand.CombinedOutput()
		require.NoError(t, productionErr, "BAS production caller browser journey failed:\n%s", productionOutput)
		var productionReport struct {
			Results []struct {
				Name   string `json:"name"`
				Text   string `json:"text"`
				Before int    `json:"before"`
				After  int    `json:"after"`
			} `json:"results"`
			FinalStats struct {
				Dispatches int `json:"dispatches"`
			} `json:"finalStats"`
		}
		productionLines := strings.Split(strings.TrimSpace(string(productionOutput)), "\n")
		productionDecoded := false
		for i := len(productionLines) - 1; i >= 0; i-- {
			if json.Unmarshal([]byte(strings.Join(productionLines[i:], "\n")), &productionReport) == nil && len(productionReport.Results) > 0 {
				productionDecoded = true
				break
			}
		}
		require.True(t, productionDecoded, "production caller browser JSON missing:\n%s", productionOutput)
		require.Len(t, productionReport.Results, 8)
		for _, result := range productionReport.Results {
			if result.Name == "valid-own-key" {
				require.Contains(t, result.Text, "Dispatched:")
				require.Equal(t, result.Before+1, result.After)
			} else {
				require.Contains(t, result.Text, "Rejected:")
				require.Equal(t, result.Before, result.After, "%s dispatched before refusal", result.Name)
			}
		}
		require.Equal(t, 2, productionReport.FinalStats.Dispatches, "only one direct-router and one production BAS request should reach fake OpenRouter")
	}
	if os.Getenv("A15_CONSENT_ELECTRON") == "1" {
		electronConsentJourney(t, repoRoot, server.URL, alice, aliceToken, dispatches)
		return
	}

	// Continue through the managed BAS Playwright path using an independently
	// started loopback driver. This crosses the TS AI route, owned session
	// broker, production navigator, callback and Gateway inference handler.
	driverURL, stopDriver := startA15PlaywrightDriver(t, server.URL, repoRoot)
	defer stopDriver()
	visionBASURL, stopVisionBAS := startA15BASConsentCaller(t, server.URL, driverURL)
	defer stopVisionBAS()
	visionCommand := exec.CommandContext(ctx, "node", script)
	visionCommand.Dir = filepath.Dir(script)
	visionCommand.Env = append(os.Environ(), "CONSENT_GATEWAY_URL="+server.URL, "CONSENT_ALICE_TOKEN="+aliceToken, "CONSENT_BOB_TOKEN="+bobToken, "CONSENT_BAS_API_URL="+visionBASURL, "CONSENT_BAS_CALLER=vision")
	visionOutput, visionErr := visionCommand.CombinedOutput()
	driverLogs := stopDriver()
	require.NoError(t, visionErr, "BAS Playwright sidecar browser journey failed:\n%s\nPlaywright driver logs:\n%s", visionOutput, driverLogs)
	var visionReport struct {
		Results []struct {
			Name   string `json:"name"`
			Text   string `json:"text"`
			Before int    `json:"before"`
			After  int    `json:"after"`
		} `json:"results"`
		FinalStats struct {
			Dispatches int `json:"dispatches"`
		} `json:"finalStats"`
	}
	visionLines := strings.Split(strings.TrimSpace(string(visionOutput)), "\n")
	visionDecoded := false
	for i := len(visionLines) - 1; i >= 0; i-- {
		if json.Unmarshal([]byte(strings.Join(visionLines[i:], "\n")), &visionReport) == nil && len(visionReport.Results) > 0 {
			visionDecoded = true
			break
		}
	}
	require.True(t, visionDecoded, "BAS vision browser JSON missing:\n%s", visionOutput)
	require.Len(t, visionReport.Results, 8)
	for _, result := range visionReport.Results {
		if result.Name == "valid-own-key" {
			require.Equal(t, result.Before+1, result.After, "the selected fake OpenRouter must receive one valid exact-source request")
		} else {
			require.Equal(t, result.Before, result.After, "%s dispatched before Gateway refusal", result.Name)
		}
	}
	wantVisionDispatches := 3
	if sidecarOnly {
		wantVisionDispatches = 1
	}
	require.Equal(t, wantVisionDispatches, visionReport.FinalStats.Dispatches, "only the valid exact-source vision request may dispatch in this slice")
}

func electronConsentJourney(t *testing.T, repoRoot, gatewayURL, aliceSubject, aliceToken string, dispatches func() int) {
	t.Helper()
	baseline := dispatches()
	lpbsURL, stopLPBS := startA15NativeAuthLPBS(t, aliceSubject, aliceToken)
	defer stopLPBS()
	basURL, stopBAS := startA15BASConsentCallerMode(t, gatewayURL, lpbsURL)
	defer stopBAS()
	script := filepath.Join(repoRoot, "scenarios/browser-automation-studio/ui/tests/e2e/consent-packaged-electron.mjs")
	appImage := strings.TrimSpace(os.Getenv("A15_APPIMAGE_PATH"))
	if appImage == "" {
		appImage = "/home/matthalloran8/.vrooli/cache/vrooli/scenario-to-desktop/staging/browser-automation-studio/759d33c9-c519-c30a-58f5-a1949059b1a3/platforms/electron/dist-electron/Vrooli Ascension-1.0.0.AppImage"
	}
	digest := strings.TrimSpace(os.Getenv("A15_ARTIFACT_DIGEST"))
	if digest == "" {
		digest = "sha256:fab3e3715788ab92333b9d03219c2c3ccd97fba56499b9d374c64e6135ff61ac"
	}
	evidencePath := strings.TrimSpace(os.Getenv("A15_EVIDENCE_PATH"))
	if evidencePath == "" {
		evidencePath = filepath.Join(repoRoot, "../.vrooli/research/topics/web-console-monetization/runs/models-monetization-music-20261115/packaged-bas-consent-results.json")
	}
	screenshotPath := strings.TrimSpace(os.Getenv("A15_SCREENSHOT_PATH"))
	command := exec.Command("node", script)
	command.Dir = filepath.Dir(script)
	command.Env = append(os.Environ(), "A15_APPIMAGE_PATH="+appImage, "A15_ARTIFACT_DIGEST="+digest,
		"A15_LPBS_URL="+lpbsURL, "A15_BAS_API_URL="+basURL, "CONSENT_GATEWAY_URL="+gatewayURL,
		"CONSENT_ALICE_TOKEN="+aliceToken, "CONSENT_FINAL_DISPATCHES="+fmt.Sprint(baseline),
		"A15_UI_DIST_DIR="+filepath.Join(repoRoot, "scenarios/browser-automation-studio/ui/dist"),
		"A15_EVIDENCE_PATH="+evidencePath)
	if screenshotPath != "" {
		command.Env = append(command.Env, "A15_SCREENSHOT_PATH="+screenshotPath)
	}
	output, err := command.CombinedOutput()
	require.NoError(t, err, "packaged Electron consent journey failed:\n%s", output)
	var result struct {
		ArtifactDigest string `json:"artifact_digest"`
		Auth           struct {
			PKCEVerified bool `json:"pkce_verified"`
			TokenFound   bool `json:"token_found"`
		} `json:"auth"`
		Entitlement struct {
			AppKey  string `json:"app_key"`
			Feature string `json:"feature"`
		} `json:"entitlement"`
		GrantDispatches   int  `json:"grant_dispatches"`
		RevokedDispatches int  `json:"revoked_dispatches"`
		ForgedDispatches  int  `json:"forged_dispatches"`
		RevokedRefused    bool `json:"revoked_refused"`
		ForgedRefused     bool `json:"forged_refused"`
	}
	require.NoError(t, json.Unmarshal(output, &result), "packaged runner must print one JSON result object:\n%s", output)
	require.Equal(t, digest, result.ArtifactDigest)
	require.True(t, result.Auth.PKCEVerified)
	require.True(t, result.Auth.TokenFound)
	require.Equal(t, "browser-automation-studio", result.Entitlement.AppKey)
	require.Equal(t, "ai", result.Entitlement.Feature)
	require.Equal(t, baseline+1, result.GrantDispatches)
	require.Equal(t, result.GrantDispatches, result.RevokedDispatches)
	require.Equal(t, result.RevokedDispatches, result.ForgedDispatches)
	require.True(t, result.RevokedRefused)
	require.True(t, result.ForgedRefused)
	require.Equal(t, baseline+1, dispatches(), "only granted packaged BAS AI request should dispatch")
}

func startA15NativeAuthLPBS(t *testing.T, aliceSubject, aliceToken string) (string, func()) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	const keyID = "a15-entitlement-fixture-key"
	leaseJWKS, err := consumeridentity.NewKeySet(consumeridentity.PublicKey{ID: keyID, Key: &key.PublicKey}).JWKS()
	require.NoError(t, err)
	type challenge struct{ redirectURI, value string }
	var mu sync.Mutex
	challenges := map[string]challenge{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/auth/login":
			redirectURI, state, challengeValue := r.URL.Query().Get("redirect_uri"), r.URL.Query().Get("state"), r.URL.Query().Get("code_challenge")
			if redirectURI == "" || state == "" || challengeValue == "" {
				http.Error(w, "missing authorization parameters", http.StatusBadRequest)
				return
			}
			code := "a15-code-" + state
			mu.Lock()
			challenges[code] = challenge{redirectURI, challengeValue}
			mu.Unlock()
			callback, parseErr := url.Parse(redirectURI)
			if parseErr != nil {
				http.Error(w, "invalid redirect", http.StatusBadRequest)
				return
			}
			query := callback.Query()
			query.Set("code", code)
			query.Set("state", state)
			callback.RawQuery = query.Encode()
			http.Redirect(w, r, callback.String(), http.StatusFound)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/auth/token":
			var body struct {
				Code        string `json:"code"`
				Verifier    string `json:"code_verifier"`
				RedirectURI string `json:"redirect_uri"`
			}
			if json.NewDecoder(r.Body).Decode(&body) != nil {
				http.Error(w, "invalid token request", http.StatusBadRequest)
				return
			}
			mu.Lock()
			expected, found := challenges[body.Code]
			delete(challenges, body.Code)
			mu.Unlock()
			hash := sha256.Sum256([]byte(body.Verifier))
			if !found || expected.redirectURI != body.RedirectURI || expected.value != base64.RawURLEncoding.EncodeToString(hash[:]) {
				http.Error(w, "PKCE rejected", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": aliceToken, "refresh_token": "fixture-refresh-never-used", "token_type": "Bearer", "expires_at": time.Now().UTC().Add(time.Hour).Format(time.RFC3339)})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/auth/me":
			if r.Header.Get("Authorization") != "Bearer "+aliceToken {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"user":{"email":%q}}`, aliceSubject+"@fixture.invalid")
		case r.Method == http.MethodGet && r.URL.Path == "/.well-known/jwks.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(leaseJWKS)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/entitlements":
			if r.Header.Get("Authorization") != "Bearer "+aliceToken || r.URL.Query().Get("user") != aliceSubject+"@fixture.invalid" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			lease, signErr := entitlementclient.Sign(entitlementclient.Payload{UserIdentity: aliceSubject + "@fixture.invalid", Status: "active", PlanTier: "fixture", PlanRank: 10, AppEntitlements: []entitlementclient.AppEntitlement{{AppKey: "browser-automation-studio", Status: "active", PlanRank: 10, Features: []string{"ai"}}}, NotAfter: time.Now().UTC().Add(time.Hour)}, keyID, key)
			if signErr != nil {
				http.Error(w, signErr.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"lease": lease})
		default:
			http.NotFound(w, r)
		}
	}))
	return server.URL, server.Close
}

func startA15PlaywrightDriver(t *testing.T, gatewayURL, repoRoot string) (string, func() string) {
	t.Helper()
	driverDir := filepath.Join(repoRoot, "scenarios", "browser-automation-studio", "playwright-driver")
	dist := filepath.Join(driverDir, "dist", "server.js")
	if _, err := os.Stat(dist); err != nil {
		require.FailNow(t, "Playwright driver dist is missing; refusing to mutate it for this fixture: %v", err)
	}
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := probe.Addr().(*net.TCPAddr).Port
	require.NoError(t, probe.Close())
	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	node, err := exec.LookPath("node")
	require.NoError(t, err)
	command := exec.Command(node, dist)
	command.Dir = driverDir
	command.Env = append(os.Environ(), "PLAYWRIGHT_DRIVER_HOST=127.0.0.1", fmt.Sprintf("PLAYWRIGHT_DRIVER_PORT=%d", port), "AI_GATEWAY_URL="+gatewayURL, "HEADLESS=true", "METRICS_ENABLED=false", "MAX_SESSIONS=2", "PLAYWRIGHT_DRIVER_FAULT_CONTROL=false", "BROWSER_EXECUTABLE_PATH=/usr/bin/google-chrome")
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var stdout, stderr strings.Builder
	command.Stdout = &stdout
	command.Stderr = &stderr
	require.NoError(t, command.Start())
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	var stopOnce sync.Once
	var logs string
	stop := func() string {
		stopOnce.Do(func() {
			_ = syscall.Kill(-command.Process.Pid, syscall.SIGTERM)
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				_ = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
				<-done
			}
			logs = "stdout:\n" + stdout.String() + "\nstderr:\n" + stderr.String()
		})
		return logs
	}
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		response, requestErr := client.Get(url + "/health")
		if requestErr == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return url, stop
			}
		}
		select {
		case exitErr := <-done:
			require.FailNow(t, "isolated Playwright driver exited before health: %v; %s", exitErr, stop())
		default:
		}
		<-time.After(150 * time.Millisecond)
	}
	stop()
	require.FailNow(t, "isolated Playwright driver failed health startup: %s", stop())
	return "", func() string { return "" }
}

func startA15BASConsentCaller(t *testing.T, gatewayURL string, driverURLs ...string) (string, func()) {
	return startA15BASConsentCallerMode(t, gatewayURL, "", driverURLs...)
}

func startA15BASConsentCallerMode(t *testing.T, gatewayURL, lpbsURL string, driverURLs ...string) (string, func()) {
	t.Helper()
	basAPI := filepath.Clean(filepath.Join("..", "..", "..", "..", "browser-automation-studio", "api"))
	command := exec.Command("go", "test", "-v", "./handlers/ai_service", "-run", "^TestA15ConsentCallerFixtureServer$", "-count=1")
	command.Dir = basAPI
	command.Env = append(os.Environ(), "A15_CONSENT_CALLER_SERVER=1", "AI_GATEWAY_URL="+gatewayURL)
	if lpbsURL != "" {
		command.Env = append(command.Env, "A15_CONSENT_ENTITLEMENT=1", "A15_CONSENT_LPBS_URL="+lpbsURL)
	}
	if len(driverURLs) > 0 {
		command.Env = append(command.Env, "A15_CONSENT_VISION=1", "PLAYWRIGHT_DRIVER_URL="+driverURLs[0])
	}
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdout, err := command.StdoutPipe()
	require.NoError(t, err)
	var stderr strings.Builder
	command.Stderr = &stderr
	require.NoError(t, command.Start())
	lines := make(chan string, 8)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		close(lines)
	}()
	timer := time.NewTimer(90 * time.Second)
	defer timer.Stop()
	url := ""
	for url == "" {
		select {
		case line, ok := <-lines:
			if !ok {
				_ = command.Wait()
				require.FailNow(t, "BAS fixture server exited before binding: %s", stderr.String())
			}
			if strings.HasPrefix(line, "A15_BAS_FIXTURE_URL=") {
				url = strings.TrimPrefix(line, "A15_BAS_FIXTURE_URL=")
			}
		case <-timer.C:
			_ = syscall.Kill(-command.Process.Pid, syscall.SIGTERM)
			_ = command.Wait()
			require.FailNow(t, "timed out waiting for BAS fixture server; stderr=%s", stderr.String())
		}
	}
	stop := func() {
		_ = syscall.Kill(-command.Process.Pid, syscall.SIGTERM)
		_ = command.Wait()
	}
	return url, stop
}

type consentGrantResponse struct {
	Grant   privacyconsent.Grant `json:"grant"`
	Receipt string               `json:"receipt"`
}

// a15ConsentInferenceRunner is a deterministic OpenRouter-only adapter. It
// returns one schema-valid vision action and a shape-compatible response for
// the separate AIService caller, without contacting any provider.
type a15ConsentInferenceRunner struct {
	mu       sync.Mutex
	Commands []providers.Command
}

func (r *a15ConsentInferenceRunner) Run(_ context.Context, command providers.Command) (providers.Result, error) {
	r.mu.Lock()
	r.Commands = append(r.Commands, command)
	r.mu.Unlock()
	commandText := command.String()
	if strings.Contains(commandText, " policy resolve --role ") {
		fields := strings.Fields(commandText)
		role := "extract.structured"
		for i := range fields {
			if fields[i] == "--role" && i+1 < len(fields) {
				role = fields[i+1]
				break
			}
		}
		return providers.Result{Stdout: fmt.Sprintf(`{"role":%q,"model":"fixture/openrouter"}`, role)}, nil
	}
	if strings.Contains(commandText, " generate --role extract.structured ") {
		return providers.Result{Stdout: `{"choices":[{"message":{"content":"{\"action\":{\"type\":\"done\"},\"reasoning\":\"fixture task complete\",\"goalAchieved\":true,\"confidence\":1}"}}],"usage":{"prompt_tokens":2,"completion_tokens":1}}`}, nil
	}
	return providers.Result{Stdout: `{"choices":[{"message":{"content":"\"[]\""}}],"usage":{"prompt_tokens":2,"completion_tokens":1}}`}, nil
}

func (r *a15ConsentInferenceRunner) CommandSnapshot() []providers.Command {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]providers.Command(nil), r.Commands...)
}

func newConsentGatewayHTTPServer(t *testing.T, db *sql.DB, store *privacyconsent.Store, service *internalrouting.Service, identityConfig authn.Config) *httptest.Server {
	t.Helper()
	router := mux.NewRouter()
	consentHandler := consentHTTP.New(store)
	router.Handle("/api/v1/ai/privacy-consent", consentHandler)
	router.PathPrefix("/api/v1/ai/privacy-consent/").Handler(consentHandler)
	routingHTTP.Module(routingHTTP.Deps{DB: db, Service: service}).Mount(router)
	return httptest.NewServer(authn.Middleware(identityConfig)(router))
}

func issueConsentGrant(t *testing.T, client *http.Client, baseURL, token, source string, lifetimeDays int) consentGrantResponse {
	t.Helper()
	body, err := json.Marshal(map[string]any{"source": source, "lifetime_days": lifetimeDays})
	require.NoError(t, err)
	response := sendConsentRequest(t, client, http.MethodPost, baseURL+"/api/v1/ai/privacy-consent", token, "application/json", string(body))
	responseBody := mustReadBody(t, response)
	require.Equal(t, http.StatusCreated, response.StatusCode, responseBody)
	var result consentGrantResponse
	require.NoError(t, json.Unmarshal([]byte(responseBody), &result))
	require.NotEmpty(t, result.Grant.ID)
	require.NotEmpty(t, result.Receipt)
	return result
}

func listConsentGrants(t *testing.T, client *http.Client, baseURL, token string) []privacyconsent.Grant {
	t.Helper()
	response := sendConsentRequest(t, client, http.MethodGet, baseURL+"/api/v1/ai/privacy-consent", token, "", "")
	responseBody := mustReadBody(t, response)
	require.Equal(t, http.StatusOK, response.StatusCode)
	var result struct {
		Grants []privacyconsent.Grant `json:"grants"`
	}
	require.NoError(t, json.Unmarshal([]byte(responseBody), &result))
	return result.Grants
}

func grantIDs(grants []privacyconsent.Grant) []string {
	ids := make([]string, 0, len(grants))
	for _, grant := range grants {
		ids = append(ids, grant.ID)
	}
	return ids
}

func sendConsentRequest(t *testing.T, client *http.Client, method, url, token, contentType, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	response, err := client.Do(req)
	require.NoError(t, err)
	return response
}

func mustReadBody(t *testing.T, response *http.Response) string {
	t.Helper()
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	return string(body)
}

func randomConsentTestID(t *testing.T) string {
	t.Helper()
	var value [12]byte
	_, err := rand.Read(value[:])
	require.NoError(t, err)
	return fmt.Sprintf("%x", value[:])
}
