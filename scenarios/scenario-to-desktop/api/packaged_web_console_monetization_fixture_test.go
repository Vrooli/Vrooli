package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	validationmatrix "github.com/vrooli/vrooli/packages/delivery-ramp-go/validationmatrix"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

func TestPackagedWebConsoleFixtureRefusesWrongScenarioModeAndMissingTargetOwner(t *testing.T) {
	root := t.TempDir()
	artifact := filepath.Join(root, "platforms", "electron", "dist-electron", "AppImage")
	if err := os.MkdirAll(filepath.Dir(artifact), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifact, []byte("exact artifact"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(root, "manifest", "platforms", "electron", "bundle", "bundle.json")
	if err := os.MkdirAll(filepath.Dir(manifest), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifest, []byte(`{"services":[{"id":"api"},{"id":"ui"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if !hasBundledAPIService(artifact) {
		t.Fatal("fixture test artifact did not resolve its exact bundled root API manifest")
	}

	executor := validationMatrixLocalExecutor{}
	base := validationmatrix.CellRequest{
		DeploymentMode: "bundled",
		Cell: &domainv1.ValidationCell{
			ScenarioName: "web-console", JourneyId: packagedWebConsoleMonetizationJourneyID,
			TargetId: "local-linux-amd64",
		},
		Target: &domainv1.ValidationTargetDescriptor{TargetId: "local-linux-amd64", Available: true},
	}
	for _, tc := range []struct {
		name     string
		request  validationmatrix.CellRequest
		artifact string
		contains string
	}{
		{name: "wrong scenario", request: func() validationmatrix.CellRequest {
			r := base
			r.Cell = &domainv1.ValidationCell{ScenarioName: "browser-automation-studio", JourneyId: packagedWebConsoleMonetizationJourneyID}
			return r
		}(), artifact: artifact, contains: "outside its registered scenario"},
		{name: "proxy artifact", request: func() validationmatrix.CellRequest { r := base; r.DeploymentMode = "proxy"; return r }(), artifact: artifact, contains: "bundled root API"},
		{name: "unavailable target", request: func() validationmatrix.CellRequest {
			r := base
			r.Target = &domainv1.ValidationTargetDescriptor{TargetId: "local-linux-amd64"}
			return r
		}(), artifact: artifact, contains: "available target identity"},
		{name: "no S2D target owner", request: base, artifact: artifact, contains: "target and durable evidence owners"},
		{name: "missing bundled root API", request: base, artifact: filepath.Join(t.TempDir(), "unmanifested.AppImage"), contains: "bundled root API"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := executor.executePackagedWebConsoleMonetizationFixture(context.Background(), tc.request, tc.artifact)
			if result.Disposition != domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED || !strings.Contains(result.Reason, tc.contains) {
				t.Fatalf("fixture refusal = %s %q; want reason containing %q", result.Disposition, result.Reason, tc.contains)
			}
		})
	}
}

func TestPackagedWebConsoleFixtureEnvironmentExcludesAmbientCredentials(t *testing.T) {
	t.Setenv("STRIPE_SECRET_KEY", "must-not-cross-fixture-boundary")
	got := strings.Join(packagedWebConsoleMonetizationEnvironment(packagedWebConsoleTestInputs{
		Phase: "signed-online", APIURL: "http://127.0.0.1:1234", ArtifactPath: "/tmp/app.AppImage",
		ArtifactDigest: "sha256:artifact", TargetID: "local-linux-amd64", EvidencePath: "/tmp/evidence.json",
		AppDataDir: "/tmp/private/app-data", JWKSPath: "/tmp/private/app-data/jwks.json", UpstreamListenURL: "http://127.0.0.1:4321",
	}), "\n")
	if strings.Contains(got, "STRIPE_SECRET_KEY") || strings.Contains(got, "must-not-cross-fixture-boundary") {
		t.Fatal("fixture subprocess inherited an unrelated credential")
	}
	for _, want := range []string{
		"WC_MONETIZATION_API_URL=http://127.0.0.1:1234",
		"WC_MONETIZATION_APPIMAGE_PATH=/tmp/app.AppImage",
		"WC_MONETIZATION_ARTIFACT_DIGEST=sha256:artifact",
		"WC_MONETIZATION_TARGET_ID=local-linux-amd64",
		"WC_MONETIZATION_EVIDENCE_PATH=/tmp/evidence.json",
		"WC_MONETIZATION_PHASE=signed-online",
		"WC_MONETIZATION_UPSTREAM_LISTEN_ADDR=127.0.0.1:4321",
		"WC_MONETIZATION_APP_DATA_DIR=/tmp/private/app-data",
		"WC_MONETIZATION_JWKS_PATH=/tmp/private/app-data/jwks.json",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("fixture environment missing %q", want)
		}
	}
}
