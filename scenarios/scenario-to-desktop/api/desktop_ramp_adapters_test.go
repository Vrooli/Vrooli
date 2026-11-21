package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	validationmatrix "github.com/vrooli/vrooli/packages/delivery-ramp-go/validationmatrix"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"scenario-to-desktop-api/livedesktop"
	"scenario-to-desktop-api/target"
	"scenario-to-desktop-api/validationprovider"
)

type artifactFinderStub struct{ path string }

func (s artifactFinderStub) FindArtifact(string) (string, error) { return s.path, nil }

type unavailableValidationBuilder struct{ calls int }

func (b *unavailableValidationBuilder) Build(context.Context, deliveryramp.BuildRequest) (deliveryramp.Artifact, error) {
	b.calls++
	return deliveryramp.Artifact{}, fmt.Errorf("source checkout has no desktop package")
}

type modeDesktopOwner struct{ launch target.ElectronLaunchOptions }

func (d *modeDesktopOwner) StartSession(context.Context, livedesktop.SessionConfig) (*livedesktop.Session, error) {
	return &livedesktop.Session{ID: "session"}, nil
}
func (*modeDesktopOwner) ExecuteAction(_ context.Context, _ string, action string, _ json.RawMessage) (*livedesktop.ActionResult, error) {
	if action == "start_recording" {
		return &livedesktop.ActionResult{Data: map[string]any{"capture_id": "capture"}}, nil
	}
	return &livedesktop.ActionResult{}, nil
}
func (d *modeDesktopOwner) LaunchElectronValidation(_ context.Context, _, _ string, launch target.ElectronLaunchOptions, _ target.RendererExpectation) (*domainv1.AppTarget, error) {
	d.launch = launch
	return &domainv1.AppTarget{TargetId: "target", CdpEndpoint: "http://127.0.0.1:9222", RendererId: "renderer", RendererUrl: "http://127.0.0.1:41235", CdpTransport: "http"}, nil
}
func (*modeDesktopOwner) WaitElectronValidationExit(context.Context, string) error    { return nil }
func (*modeDesktopOwner) RequestElectronValidationExit(context.Context, string) error { return nil }
func (*modeDesktopOwner) RestartElectronValidation(context.Context, string) (*domainv1.AppTarget, error) {
	return &domainv1.AppTarget{TargetId: "target-restarted", CdpEndpoint: "http://127.0.0.1:9223", RendererId: "renderer-restarted"}, nil
}
func (*modeDesktopOwner) StopSession(string) error { return nil }

type modeWorkflowOwner struct{ request validationprovider.Request }

func (w *modeWorkflowOwner) Execute(_ context.Context, request validationprovider.Request) validationprovider.Result {
	w.request = request
	return validationprovider.Result{Passed: true, ProviderRunID: "provider"}
}

func TestMatrixExecutorCarriesBundledModeToElectronAndProvider(t *testing.T) {
	desktop := &modeDesktopOwner{}
	workflow := &modeWorkflowOwner{}
	executor := validationMatrixLocalExecutor{desktop: desktop, workflow: workflow}
	request := validationmatrix.CellRequest{
		RunID: "matrix-run", ArtifactDigest: "sha256:artifact", ArtifactPath: "/tmp/app", DeploymentMode: "bundled", BrowserAutomationStudioVariant: "preview-2",
		Cell:    &domainv1.ValidationCell{CellId: "cell", ScenarioName: "demo", JourneyId: "journey", TargetId: "target", EnvironmentProfile: domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_NORMAL},
		Journey: validationmatrix.JourneySelection{JourneyID: "journey", SourcePath: "/tmp/journey.json"},
		Target:  &domainv1.ValidationTargetDescriptor{TargetId: "target", Available: true},
	}
	executor.executeProviderJourney(context.Background(), request, request.ArtifactPath)
	if !desktop.launch.BundledRuntime || !workflow.request.BundledRuntime || workflow.request.BrowserAutomationStudioVariant != "preview-2" {
		t.Fatalf("adapter binding lost: Electron bundled=%v provider bundled=%v BAS variant=%q", desktop.launch.BundledRuntime, workflow.request.BundledRuntime, workflow.request.BrowserAutomationStudioVariant)
	}
}

func TestValidationUsesSelectedStagedArtifactWithoutRediscovery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Device Studio.AppImage")
	data := []byte("exact staged package")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	digest := "sha256:" + hex.EncodeToString(sum[:])
	builder := &unavailableValidationBuilder{}
	executor := validationMatrixLocalExecutor{builder: builder}
	got, err := executor.findValidationArtifact(validationmatrix.CellRequest{ArtifactPath: path, ArtifactDigest: digest})
	if err != nil || got != path || builder.calls != 0 {
		t.Fatalf("selected staged artifact replaced: path=%q err=%v rediscoveries=%d", got, err, builder.calls)
	}
	if actual, err := fileDigest(got); err != nil || actual != digest {
		t.Fatalf("artifact digest=%q err=%v", actual, err)
	}
}

func TestValidationRejectsInvalidSelectedArtifactInsteadOfFallingBack(t *testing.T) {
	root := t.TempDir()
	regular := filepath.Join(root, "app.AppImage")
	if err := os.WriteFile(regular, []byte("package"), 0o600); err != nil {
		t.Fatal(err)
	}
	validDigest := "sha256:" + strings.Repeat("a", 64)
	for _, tc := range []struct{ name, path, digest string }{
		{"relative", "app.AppImage", validDigest},
		{"missing", filepath.Join(root, "missing.AppImage"), validDigest},
		{"directory", root, validDigest},
		{"unbound", regular, ""},
		{"malformed digest", regular, "sha256:xyz"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			builder := &unavailableValidationBuilder{}
			executor := validationMatrixLocalExecutor{builder: builder}
			if _, err := executor.findValidationArtifact(validationmatrix.CellRequest{ArtifactPath: tc.path, ArtifactDigest: tc.digest}); err == nil {
				t.Fatal("invalid selected artifact accepted")
			}
			if builder.calls != 0 {
				t.Fatal("invalid explicit selection fell back to source discovery")
			}
		})
	}
}

func TestDesktopRampBuilderCarriesMonetizationIdentityAndDigest(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "app.AppImage")
	if err := os.WriteFile(path, []byte("desktop artifact"), 0o600); err != nil {
		t.Fatal(err)
	}
	rootWithScenario := t.TempDir()
	if err := os.MkdirAll(filepath.Join(rootWithScenario, "scenarios", "paid-app", ".vrooli"), 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := `{"bundle_key":"business_suite","app_key":"paid-app"}`
	if err := os.WriteFile(filepath.Join(rootWithScenario, "scenarios", "paid-app", ".vrooli", "monetization.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_ROOT", rootWithScenario)
	artifact, err := (desktopRampBuilder{finder: artifactFinderStub{path: path}}).Build(context.Background(), deliveryramp.BuildRequest{SourceRef: "paid-app"})
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Metadata["bundle_key"] != "business_suite" || artifact.Metadata["app_key"] != "paid-app" || artifact.Checksum == "" {
		t.Fatalf("artifact metadata = %#v, checksum=%q", artifact.Metadata, artifact.Checksum)
	}
}

func TestDesktopRampDistributorRegistersImmutableArtifact(t *testing.T) {
	var gotPath string
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode registration: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"id":42}}`))
	}))
	defer server.Close()
	t.Setenv("S2D_LPBS_CATALOG_URL", server.URL)
	t.Setenv("S2D_DESKTOP_ARTIFACT_URL", "https://downloads.example.test/paid.AppImage")
	t.Setenv("S2D_LPBS_ADMIN_TOKEN", "operator-token")

	result, err := (desktopRampDistributor{client: server.Client()}).Distribute(context.Background(), deliveryramp.DistributionRequest{Artifact: deliveryramp.Artifact{
		ImmutableRef: "artifact:sha256",
		Checksum:     "sha256:abc",
		Metadata:     map[string]string{"bundle_key": "business_suite", "app_key": "paid-app", "platform": "linux"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Disposition != deliveryramp.DispositionDegraded || result.EffectReceipt == nil || result.EffectReceipt.ExternalReceipt != "lpbs-asset:42" || result.EffectReceipt.Outcome != "catalog_registered" || gotPath != "/api/v1/admin/download-apps/paid-app" {
		t.Fatalf("registration result=%+v path=%q", result, gotPath)
	}
	if result.Reason == "" {
		t.Fatal("catalog-only registration must explain why publication is still incomplete")
	}
	if got["app_key"] != "paid-app" {
		t.Fatalf("registration payload = %#v", got)
	}
}

func TestDesktopRampDistributorRefusesHTTPOnlySuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	t.Setenv("S2D_LPBS_CATALOG_URL", server.URL)
	t.Setenv("S2D_DESKTOP_ARTIFACT_URL", "https://downloads.example.test/paid.AppImage")
	t.Setenv("S2D_LPBS_ADMIN_TOKEN", "operator-token")

	result, err := (desktopRampDistributor{client: server.Client()}).Distribute(context.Background(), deliveryramp.DistributionRequest{Artifact: deliveryramp.Artifact{
		ImmutableRef: "artifact:sha256",
		Checksum:     "sha256:abc",
		Metadata:     map[string]string{"bundle_key": "business_suite", "app_key": "paid-app", "platform": "linux"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Disposition != deliveryramp.DispositionUnavailable || result.EffectReceipt != nil {
		t.Fatalf("HTTP-only success result = %+v", result)
	}
}
