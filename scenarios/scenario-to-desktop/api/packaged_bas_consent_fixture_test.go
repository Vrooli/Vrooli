package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	validationmatrix "github.com/vrooli/vrooli/packages/delivery-ramp-go/validationmatrix"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"scenario-to-desktop-api/smoketest"
)

func TestPackagedBASConsentFixtureRefusesUnboundArtifactsAndScenario(t *testing.T) {
	cell := &domainv1.ValidationCell{ScenarioName: "browser-automation-studio", JourneyId: packagedBASConsentJourneyID}
	executor := validationMatrixLocalExecutor{}
	for _, tc := range []struct {
		name     string
		cell     *domainv1.ValidationCell
		digest   string
		contains string
	}{
		{name: "wrong digest", cell: cell, digest: "sha256:" + strings.Repeat("0", 64), contains: packagedBASConsentDigest},
		{name: "wrong scenario", cell: &domainv1.ValidationCell{ScenarioName: "web-console", JourneyId: packagedBASConsentJourneyID}, digest: packagedBASConsentDigest, contains: "outside its registered scenario"},
		{name: "missing durable capture owner", cell: cell, digest: packagedBASConsentDigest, contains: "no durable capture store"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := executor.executePackagedBASConsentFixture(context.Background(), validationmatrix.CellRequest{
				RunID: "matrix-run", ArtifactDigest: tc.digest, Cell: tc.cell,
			}, "/tmp/exact-bas.AppImage")
			if result.Disposition != domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED || !strings.Contains(result.Reason, tc.contains) {
				t.Fatalf("fixture refusal = disposition %s reason %q, want reason containing %q", result.Disposition, result.Reason, tc.contains)
			}
		})
	}
}

func TestPackagedConsentEnvironmentIsFixtureScoped(t *testing.T) {
	t.Setenv("LPBS_ADMIN_TOKEN", "must-not-cross-fixture-boundary")
	got := strings.Join(packagedConsentEnvironment("/tmp/app.AppImage", packagedBASConsentDigest, "/tmp/evidence.json", "/tmp/screenshot.png"), "\n")
	if strings.Contains(got, "LPBS_ADMIN_TOKEN") || strings.Contains(got, "must-not-cross-fixture-boundary") {
		t.Fatal("fixture subprocess inherited an unrelated credential")
	}
	for _, want := range []string{"A15_CONSENT_BROWSER=1", "A15_CONSENT_ELECTRON=1", "A15_CONSENT_SIDECAR_ONLY=1", "A15_APPIMAGE_PATH=/tmp/app.AppImage", "A15_ARTIFACT_DIGEST=" + packagedBASConsentDigest, "A15_EVIDENCE_PATH=/tmp/evidence.json", "A15_SCREENSHOT_PATH=/tmp/screenshot.png"} {
		if !strings.Contains(got, want) {
			t.Fatalf("fixture environment missing %q", want)
		}
	}
}

func TestValidationSmokeDeploymentModeUsesExplicitSelectionOrExactBundleManifest(t *testing.T) {
	root := t.TempDir()
	artifact := filepath.Join(root, "platforms", "electron", "dist-electron", "AppImage")
	if err := os.MkdirAll(filepath.Dir(artifact), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifact, []byte("selected artifact"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, "manifest", "platforms", "electron", "bundle", "bundle.json")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, []byte(`{"services":[{"id":"api"},{"id":"ui"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ selected, want string }{
		{"", "bundled"},
		{"bundled", "bundled"},
		{"proxy", "proxy"},
		{"external-server", "unknown"},
	} {
		if got := validationSmokeDeploymentMode(tc.selected, artifact); got != tc.want {
			t.Errorf("validationSmokeDeploymentMode(%q)=%q, want %q", tc.selected, got, tc.want)
		}
	}
	if got := validationSmokeDeploymentMode("", "/tmp/unpackaged/AppImage"); got != "unknown" {
		t.Fatalf("artifact without a nearby bundle manifest inferred mode %q", got)
	}
}

type deploymentModeSmokeService struct {
	store   smoketest.Store
	request smoketest.SmokeTestRequest
}

func (s *deploymentModeSmokeService) PerformSmokeTest(context.Context, string, string, string, string) {
}
func (*deploymentModeSmokeService) CurrentPlatform() string { return "linux" }
func (s *deploymentModeSmokeService) PerformSmokeTestRequest(_ context.Context, request smoketest.SmokeTestRequest) {
	s.request = request
	s.store.Update(request.SmokeTestID, func(status *smoketest.Status) {
		status.Status = "passed"
		status.DeploymentMode = request.DeploymentMode
	})
}

func TestMatrixPlatformSmokeCarriesDeclaredBundledModeAndSelectedArtifact(t *testing.T) {
	artifact := filepath.Join(t.TempDir(), "selected.AppImage")
	data := []byte("exact selected artifact")
	if err := os.WriteFile(artifact, data, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	store := smoketest.NewInMemoryStore()
	service := &deploymentModeSmokeService{store: store}
	executor := validationMatrixLocalExecutor{smokeService: service, smokeStore: store, findArtifact: artifactFinderStub{path: artifact}}
	result := executor.Execute(context.Background(), validationmatrix.CellRequest{
		RunID: "matrix-run", ArtifactPath: artifact, ArtifactDigest: "sha256:" + hex.EncodeToString(digest[:]), DeploymentMode: "bundled",
		Cell:    &domainv1.ValidationCell{CellId: "cell", ScenarioName: "web-console", TargetId: "local-linux-amd64"},
		Journey: validationmatrix.JourneySelection{JourneyID: "desktop-smoke", ExecutionMode: "platform"},
	})
	if result.Disposition != domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS {
		t.Fatalf("matrix platform smoke failed: %+v", result)
	}
	if service.request.DeploymentMode != "bundled" || service.request.ArtifactPath != artifact {
		t.Fatalf("smoke request lost exact selection: mode=%q artifact=%q", service.request.DeploymentMode, service.request.ArtifactPath)
	}
}

func TestPackagedJourneyEvidenceRequiresAndSatisfiesTargetLayer(t *testing.T) {
	cell := &domainv1.ValidationCell{
		CellId: "cell", ScenarioName: "browser-automation-studio", ArtifactDigest: packagedBASConsentDigest,
		TargetId: "local-linux-amd64", JourneyId: packagedBASConsentJourneyID,
		EnvironmentProfile: domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_NORMAL,
		Disposition:        domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS, Required: true, Applicable: true,
	}
	for _, kind := range []domainv1.LayeredEvidence_Kind{
		domainv1.LayeredEvidence_KIND_BAS_WORKFLOW,
		domainv1.LayeredEvidence_KIND_DESKTOP_RUNTIME,
		domainv1.LayeredEvidence_KIND_TARGET,
		domainv1.LayeredEvidence_KIND_MACHINE_ASSERTION,
	} {
		cell.Evidence = append(cell.Evidence, &domainv1.LayeredEvidence{Kind: kind, EvidenceId: kind.String(), Uri: "capture://" + kind.String(), Sha256: "sha256:evidence", Redacted: true})
	}
	matrix := &domainv1.ValidationMatrix{
		MatrixId: "matrix", ScenarioName: cell.GetScenarioName(), ArtifactDigest: cell.GetArtifactDigest(),
		Journeys:            []*domainv1.JourneyCatalogItem{{JourneyId: packagedBASConsentJourneyID, ExecutionMode: "s2d-packaged-fixture", Required: true}},
		Targets:             []*domainv1.ValidationTargetDescriptor{{TargetId: cell.GetTargetId(), Available: true}},
		EnvironmentProfiles: []domainv1.ValidationEnvironmentProfile{cell.GetEnvironmentProfile()}, Cells: []*domainv1.ValidationCell{cell},
	}
	if gate := validationmatrix.EvaluateReleaseGate(matrix); !gate.GetPassed() {
		t.Fatalf("complete packaged fixture evidence did not pass structured matrix gate: %s", gate.GetReason())
	}
	cell.Evidence = append(cell.Evidence[:2], cell.Evidence[3:]...)
	if gate := validationmatrix.EvaluateReleaseGate(matrix); gate.GetPassed() {
		t.Fatal("structured matrix gate passed without target evidence")
	}
}
