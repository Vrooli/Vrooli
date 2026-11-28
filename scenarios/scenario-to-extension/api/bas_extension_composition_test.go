package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/mux"
	basaccess "github.com/vrooli/browser-automation-studio/automation/agentaccess"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/automation/session"
	bastestaccess "github.com/vrooli/browser-automation-studio/testsupport/agentaccess"
)

const (
	extensionComposeIdentity = "signed-s2e-run-token"
	extensionComposeGrant    = "browser-grant-s2e-run"
	extensionComposeRun      = "00000000-0000-0000-0000-000000000019"
	extensionComposeSession  = "s2e-broker-session"
	extensionComposeProfile  = "profile-extension-test"
	extensionComposeURL      = "https://extension.fixture.test/start"
)

type composedExtensionController struct {
	mu         sync.Mutex
	calls      []string
	closed     int
	binding    basaccess.ExtensionSessionBinding
	artifact   basaccess.ExtensionArtifact
	failReload bool
}

func (c *composedExtensionController) Install(_ context.Context, request basaccess.ExtensionInstallRequest) (basaccess.ExtensionReceipt, error) {
	c.record("install")
	c.mu.Lock()
	c.binding, c.artifact = request.Binding, request.Artifact
	c.mu.Unlock()
	return composedReceipt(request.Binding, request.Artifact, "install", "installed"), nil
}

func (c *composedExtensionController) Load(_ context.Context, request basaccess.ExtensionLifecycleRequest) (basaccess.ExtensionReceipt, error) {
	c.record("load")
	return composedReceipt(request.Binding, request.Artifact, "load", "loaded"), nil
}

func (c *composedExtensionController) Reload(_ context.Context, request basaccess.ExtensionLifecycleRequest) (basaccess.ExtensionReceipt, error) {
	c.record("reload")
	c.mu.Lock()
	fail := c.failReload
	c.mu.Unlock()
	if fail {
		return basaccess.ExtensionReceipt{}, errors.New("fixture reload failure")
	}
	return composedReceipt(request.Binding, request.Artifact, "reload", "reloaded"), nil
}

func (c *composedExtensionController) Unload(_ context.Context, request basaccess.ExtensionLifecycleRequest) (basaccess.ExtensionReceipt, error) {
	c.record("unload")
	return composedReceipt(request.Binding, request.Artifact, "unload", "unloaded"), nil
}

func (c *composedExtensionController) Validate(_ context.Context, request basaccess.ExtensionLifecycleRequest) (basaccess.ExtensionValidation, error) {
	c.record("validate")
	return basaccess.ExtensionValidation{
		Receipt: composedReceipt(request.Binding, request.Artifact, "validate", "valid"),
		Valid:   true,
		Checks:  map[string]bool{"installed": true, "enabled": true, "id": true, "version": true},
	}, nil
}

func (c *composedExtensionController) Inventory(_ context.Context, binding basaccess.ExtensionSessionBinding) (basaccess.ExtensionInventoryReceipt, error) {
	c.record("inventory")
	now := time.Now().UTC()
	c.mu.Lock()
	artifact := c.artifact
	c.mu.Unlock()
	return bastestaccess.VerifiedExtensionInventoryReceipt(binding, artifact, now), nil
}

func (c *composedExtensionController) record(operation string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls = append(c.calls, operation)
}

func (c *composedExtensionController) snapshot() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.calls...)
}

func composedReceipt(binding basaccess.ExtensionSessionBinding, artifact basaccess.ExtensionArtifact, operation, state string) basaccess.ExtensionReceipt {
	return basaccess.ExtensionReceipt{
		SessionID: binding.SessionID, Backend: binding.Backend, ProfileID: binding.ProfileID,
		PermissionRef: binding.PermissionRef, OwnerID: binding.OwnerID, LeaseID: binding.LeaseID,
		CallerID: binding.CallerID, ArtifactRef: artifact.ArtifactRef, ExtensionID: artifact.ExtensionID,
		Version: artifact.Version, Digest: artifact.Digest, ManifestSHA256: artifact.ManifestSHA256,
		Operation: operation, State: state, OccurredAt: time.Now().UTC(),
	}
}

type extensionDriverFixture struct {
	mu        sync.Mutex
	current   string
	closes    int
	starts    int
	navigates int
}

func (f *extensionDriverFixture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch {
	case r.URL.Path == "/session/start":
		f.mu.Lock()
		f.starts++
		f.current = ""
		f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(driver.CreateSessionResponse{SessionID: extensionComposeSession, LeaseID: "driver-lease-1", ActivePageID: "page-1"})
	case strings.HasSuffix(r.URL.Path, "/record/navigate"):
		var request struct {
			URL string `json:"url"`
		}
		_ = json.NewDecoder(r.Body).Decode(&request)
		f.mu.Lock()
		f.current = request.URL
		f.navigates++
		f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(driver.NavigateResponse{DriverPageID: "page-1", URL: request.URL, Title: "Extension fixture"})
	case strings.HasSuffix(r.URL.Path, "/record/navigation-state"):
		f.mu.Lock()
		current := f.current
		f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(driver.NavigationStateResponse{SessionID: extensionComposeSession, URL: current, Title: "Extension fixture"})
	case strings.HasSuffix(r.URL.Path, "/close"):
		f.mu.Lock()
		f.closes++
		f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(driver.CloseSessionResponse{Success: true})
	case strings.HasSuffix(r.URL.Path, "/run"):
		_ = json.NewEncoder(w).Encode(contracts.StepOutcome{Success: true, StepType: "read"})
	default:
		http.NotFound(w, r)
	}
}

func TestProductionExtensionTestRouteUsesMountedBASBrokerAndSeparateLifecycleOptIns(t *testing.T) {
	for _, tc := range []struct {
		name            string
		install         bool
		load            bool
		wrongGrant      bool
		wrongDigest     bool
		wrongSession    bool
		wantCalls       []string
		wantSuccess     bool
		wantReasonHas   string
		wantCleanupFail bool
		failReload      bool
		wrongExpectedID bool
	}{
		{name: "install opt-in absent", install: false, load: true, wantCalls: nil, wantReasonHas: "extension installation is disabled"},
		{name: "load opt-in absent stops before install", install: true, load: false, wantCalls: nil, wantReasonHas: "lifecycle preflight failed before install"},
		{name: "wrong grant rejected before extension effect", install: true, load: true, wrongGrant: true, wantCalls: nil, wantReasonHas: "lifecycle preflight failed before install"},
		{name: "changed artifact digest rejected before extension effect", install: true, load: true, wrongDigest: true, wantCalls: nil, wantReasonHas: "lifecycle preflight failed before install"},
		{name: "foreign session rejected before extension effect", install: true, load: true, wrongSession: true, wantCalls: nil, wantReasonHas: "lifecycle preflight failed before install"},
		{name: "exact install and load opt-ins", install: true, load: true, wantCalls: []string{"install", "reload", "validate", "inventory"}, wantSuccess: true},
		{name: "expected extension mismatch denied before install", install: true, load: true, wrongExpectedID: true, wantCalls: nil, wantReasonHas: "lifecycle preflight failed before install"},
		{name: "reload failure unloads the installed extension", install: true, load: true, failReload: true, wantCalls: []string{"install", "reload", "unload"}, wantReasonHas: "installed extension unloaded after failure"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			build, derivedExtensionID := installableBuildFixture(t)
			previousBuildManager := buildManager
			buildManager = NewBuildManager()
			if err := buildManager.Add(build); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { buildManager = previousBuildManager })
			if tc.wrongDigest {
				build.ArtifactSHA256 = strings.Repeat("0", 64)
			}

			// These are the same production handlers and route paths registered by
			// scenario-to-extension's main.go. The metadata endpoint is called by
			// BAS's real artifact verifier during the install operation.
			s2eRouter := mux.NewRouter()
			s2eAPI := s2eRouter.PathPrefix("/api/v1").Subrouter()
			s2eAPI.HandleFunc("/extension/test", testExtensionHandler).Methods(http.MethodPost)
			s2eAPI.HandleFunc("/extension/artifact/{build_id}/metadata", browserExtensionArtifactMetadataHandler).Methods(http.MethodGet)
			s2eServer := httptest.NewServer(s2eRouter)
			defer s2eServer.Close()

			amServer := newExtensionRunAuthorityFixture(t, derivedExtensionID, tc.install, tc.load)
			defer amServer.Close()
			driverFixture := &extensionDriverFixture{}
			driverServer := httptest.NewServer(driverFixture)
			defer driverServer.Close()

			driverClient, err := driver.NewClientWithURL(driverServer.URL, driver.WithoutCircuitBreaker())
			if err != nil {
				t.Fatal(err)
			}
			manager := session.NewManagerWithClient(driverClient)
			runContext := &basaccess.AgentManagerRunContext{BaseURL: amServer.URL, Client: amServer.Client()}
			authorizer := &basaccess.AgentManagerOpenAuthorizer{BaseURL: amServer.URL, Client: amServer.Client(), Context: runContext}
			controller := &composedExtensionController{}
			controller.failReload = tc.failReload
			broker, err := basaccess.NewBroker(manager, &basaccess.AgentManagerProfileResolver{RunContext: runContext}, authorizer, runContext, controller)
			if err != nil {
				t.Fatal(err)
			}
			broker.SetTrustedOpenContextResolver(runContext)
			broker.SetEvidenceRecorder(bastestaccess.NoopEvidenceRecorder())
			broker.SetCapabilityReporter(bastestaccess.QualifiedManagedExtensionLifecycle())
			verifier, err := basaccess.NewScenarioToExtensionArtifactVerifier(s2eServer.URL, s2eServer.Client())
			if err != nil {
				t.Fatal(err)
			}
			broker.SetExtensionArtifactVerifier(verifier)

			basRouter := chi.NewRouter()
			basRouter.Route("/api/v1", (&basaccess.HTTPHandler{Broker: broker}).RegisterRoutes)
			basServer := httptest.NewServer(basRouter)
			defer basServer.Close()
			t.Setenv("BAS_AGENT_ACCESS_URL", basServer.URL)

			opened := extensionComposeREST(t, basServer.Client(), http.MethodPost, basServer.URL+"/api/v1/browser/sessions", `{"backend":"managed","start_url":"`+extensionComposeURL+`","reuse_mode":"fresh"}`, http.StatusOK)
			var openedBody struct {
				SessionID string `json:"session_id"`
			}
			if err := json.Unmarshal(opened, &openedBody); err != nil || openedBody.SessionID != extensionComposeSession {
				t.Fatalf("open session = %s, err=%v", opened, err)
			}
			closeSession := func() {
				t.Helper()
				extensionComposeREST(t, basServer.Client(), http.MethodPost, basServer.URL+"/api/v1/browser/sessions/"+openedBody.SessionID+"/close", `{}`, http.StatusNoContent)
			}
			defer closeSession()

			sessionID := openedBody.SessionID
			if tc.wrongSession {
				sessionID = "foreign-session"
			}
			grant := extensionComposeGrant
			if tc.wrongGrant {
				grant = "browser-grant:another-run"
			}
			expectedID := derivedExtensionID
			if tc.wrongExpectedID {
				expectedID = "different-extension-id"
			}
			resultBytes := extensionComposeRESTWithGrant(t, s2eServer.Client(), http.MethodPost, s2eServer.URL+"/api/v1/extension/test", fmt.Sprintf(`{"build_id":%q,"session_id":%q,"extension_id":%q}`, build.BuildID, sessionID, expectedID), http.StatusOK, grant)
			if strings.Contains(string(resultBytes), extensionComposeIdentity) || strings.Contains(string(resultBytes), extensionComposeGrant) {
				t.Fatalf("S2E lifecycle response leaked run authority: %s", resultBytes)
			}
			var result ExtensionTestResult
			if err := json.Unmarshal(resultBytes, &result); err != nil {
				t.Fatalf("decode S2E test result %s: %v", resultBytes, err)
			}
			if result.Success != tc.wantSuccess {
				t.Fatalf("result success=%t reason=%q, want success=%t", result.Success, result.Reason, tc.wantSuccess)
			}
			if tc.wantReasonHas != "" && !strings.Contains(result.Reason, tc.wantReasonHas) {
				t.Fatalf("result reason %q does not contain %q", result.Reason, tc.wantReasonHas)
			}
			if tc.wantCleanupFail && !strings.Contains(result.Reason, "extension cleanup failed") {
				t.Fatalf("missing load authority must be reported as failed cleanup: %q", result.Reason)
			}
			if got := controller.snapshot(); !equalStrings(got, tc.wantCalls) {
				t.Fatalf("BAS controller operations = %v, want %v", got, tc.wantCalls)
			}
			if tc.wantSuccess {
				if result.LifecycleReceipt == nil {
					t.Fatal("successful S2E route omitted the lifecycle receipt")
				}
				receipt := result.LifecycleReceipt
				if receipt.SessionID != openedBody.SessionID || receipt.RunBinding.CallerID != "run:"+extensionComposeRun || receipt.RunBinding.PermissionRef != extensionComposeProfile || receipt.BuildID != build.BuildID || receipt.ArtifactSHA256 != build.ArtifactSHA256 || receipt.ManifestSHA256 != build.ManifestSHA256 || receipt.Install.ExtensionID != derivedExtensionID || receipt.Reload.ExtensionID != derivedExtensionID || receipt.Validation.Receipt.ExtensionID != derivedExtensionID || receipt.Install.SessionID != openedBody.SessionID || receipt.Reload.SessionID != openedBody.SessionID || receipt.Validation.Receipt.SessionID != openedBody.SessionID {
					t.Fatalf("lifecycle receipt lost exact run/session/artifact/profile binding: %#v", receipt)
				}
				if receipt.Install.Operation != "install" || receipt.Reload.Operation != "reload" || receipt.Validation.Receipt.Operation != "validate" || !receipt.Install.OccurredAt.Before(receipt.Reload.OccurredAt) || !receipt.Reload.OccurredAt.Before(receipt.Validation.Receipt.OccurredAt) {
					t.Fatalf("lifecycle receipt is not fresh ordered install→reload→validate: %#v", receipt)
				}
				if receipt.InventoryObservedAt.IsZero() || result.PermissionEvidenceScope != "grant_set_only" || result.PerTabSiteAccess != "not_measured" {
					t.Fatalf("effective permission report is incomplete/overstated: %#v", result)
				}
			}
			driverFixture.mu.Lock()
			starts, navigates := driverFixture.starts, driverFixture.navigates
			driverFixture.mu.Unlock()
			if starts != 1 || navigates != 1 {
				t.Fatalf("BAS session owner lifecycle starts=%d navigates=%d, want exactly one each", starts, navigates)
			}
		})
	}
}

func newExtensionRunAuthorityFixture(t *testing.T, extensionID string, install, load bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Agent-Identity-Token") != extensionComposeIdentity || r.Header.Get("X-Vrooli-Browser-Grant-Receipt") != extensionComposeGrant {
			http.Error(w, "wrong run authority", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/browser/resolve-context":
			_, _ = fmt.Fprintf(w, `{"caller_id":%q,"run_id":%q,"grant_receipt":%q,"capability":{"enabled":true,"grantReceipt":%q,"mode":"managed","permissions":{"profileId":%q,"profileMode":"fresh"}}}`, "run:"+extensionComposeRun, extensionComposeRun, extensionComposeGrant, extensionComposeGrant, extensionComposeProfile)
		case "/api/v1/browser/resolve-caller":
			_, _ = fmt.Fprintf(w, `{"caller_id":%q,"grant_receipt":%q}`, "run:"+extensionComposeRun, extensionComposeGrant)
		case "/api/v1/browser/resolve-profile":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "active", "run_id": extensionComposeRun, "grant_receipt": extensionComposeGrant, "mode": "managed",
				"permission_profile": map[string]any{
					"profileId": extensionComposeProfile, "profileMode": "fresh", "level": "act", "view": true, "control": true,
					"allowedOrigins": []string{"https://extension.fixture.test"}, "extensionIds": []string{extensionID},
					"extensionInstall": install, "extensionLoad": load,
				},
				"session_permission_mapping": map[string]any{
					"view": true, "control": true, "profile_mode": "fresh", "extension_access": true,
					"extension_permissions": map[string][]string{extensionID: {"storage", "https://extension.fixture.test/*"}},
				},
				"budget":                               map[string]any{"sessionMinutes": 5, "actions": 20},
				"unsupported_sessionpermission_fields": []string{},
			})
		case "/api/v1/browser/authorize-open":
			var request struct {
				ProfileID  string `json:"profile_id"`
				InitialURL string `json:"initial_url"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.ProfileID != extensionComposeProfile || request.InitialURL != extensionComposeURL {
				http.Error(w, "open authorization mismatch", http.StatusForbidden)
				return
			}
			_, _ = fmt.Fprintf(w, `{"authorized":true,"grant_receipt":%q}`, extensionComposeGrant)
		default:
			http.NotFound(w, r)
		}
	}))
}

func extensionComposeREST(t *testing.T, client *http.Client, method, target, body string, wantStatus int) []byte {
	return extensionComposeRESTWithGrant(t, client, method, target, body, wantStatus, extensionComposeGrant)
}

func extensionComposeRESTWithGrant(t *testing.T, client *http.Client, method, target, body string, wantStatus int, grant string) []byte {
	t.Helper()
	request, err := http.NewRequest(method, target, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Agent-Identity-Token", extensionComposeIdentity)
	request.Header.Set("X-Vrooli-Browser-Grant-Receipt", grant)
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != wantStatus {
		t.Fatalf("%s %s status=%d body=%s, want %d", method, target, response.StatusCode, data, wantStatus)
	}
	return data
}

func installableBuildFixture(t *testing.T) (*ExtensionBuild, string) {
	t.Helper()
	root := t.TempDir()
	templatesPath, err := filepath.Abs("../templates")
	if err != nil {
		t.Fatal(err)
	}
	build := &ExtensionBuild{
		BuildID: "s2e-composition-build", ScenarioName: "s2e-composition-fixture", TemplateType: "full", Status: ExtensionBuildBuilding,
		Config: ExtensionConfig{
			AppName: "S2E composition fixture", Description: "Generated extension broker composition test",
			APIEndpoint: "http://localhost:3202", Version: "1.0.0", Permissions: []string{"storage"},
			HostPermissions: []string{"https://extension.fixture.test/*"},
		},
		ExtensionPath: filepath.Join(root, "platforms", "extension"),
		CreatedAt:     time.Now().UTC(), BuildLog: []string{}, ErrorLog: []string{},
	}
	generationConfig := &Config{TemplatesPath: templatesPath, OutputPath: root}
	generateExtensionWithConfig(build, generationConfig)
	if build.Status != ExtensionBuildReady || build.ArtifactPath == "" || build.ArtifactSHA256 == "" || build.ManifestSHA256 == "" {
		t.Fatalf("scenario-to-extension generator did not produce a ready artifact: status=%q errors=%v log=%v", build.Status, build.ErrorLog, build.BuildLog)
	}
	manifestBytes, err := os.ReadFile(filepath.Join(build.ExtensionPath, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	key, err := chromeExtensionIDFromManifest(manifestBytes)
	if err != nil {
		t.Fatal(err)
	}
	return build, key
}

func chromeExtensionIDFromManifest(manifest []byte) (string, error) {
	var parsed struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(manifest, &parsed); err != nil {
		return "", err
	}
	return chromeExtensionID(parsed.Key)
}

func equalStrings(first, second []string) bool {
	if len(first) != len(second) {
		return false
	}
	for index := range first {
		if first[index] != second[index] {
			return false
		}
	}
	return true
}
