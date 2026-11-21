package journeys

import (
	"strings"
	"testing"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

func TestValidateScenarioInstallArtifactRequiresExactImmutableIdentity(t *testing.T) {
	digest := strings.Repeat("a", 64)
	artifact := deliveryramp.Artifact{
		ImmutableRef: "android-debug:" + digest,
		Checksum:     "sha256:" + digest,
		Metadata: map[string]string{
			"package_name": "com.example.generated", "version_name": "1.2.3", "version_code": "12", "signing": "debug-only",
		},
	}
	packageName, err := validateScenarioInstallArtifact(artifact)
	if err != nil || packageName != "com.example.generated" {
		t.Fatalf("valid artifact identity rejected: package=%q err=%v", packageName, err)
	}

	artifact.ImmutableRef = "android-release:" + digest
	if _, err := validateScenarioInstallArtifact(artifact); err == nil || !strings.Contains(err.Error(), "signing identity") {
		t.Fatalf("release artifact without signing identity was accepted: %v", err)
	}
	artifact.Metadata["signing_identity"] = "vrooli/android/release"
	artifact.ImmutableRef = "android-debug:" + strings.Repeat("b", 64)
	if _, err := validateScenarioInstallArtifact(artifact); err == nil || !strings.Contains(err.Error(), "matching the APK/AAB digest") {
		t.Fatalf("mismatched immutable reference was accepted: %v", err)
	}
}

func TestValidAndroidPackageNameUsesAndroidIdentifierSyntax(t *testing.T) {
	for _, value := range []string{"com.example.app", "org.vrooli.test_app1"} {
		if !validAndroidPackageName(value) {
			t.Errorf("valid package %q rejected", value)
		}
	}
	for _, value := range []string{"example", "com..app", "com.1app", "com.exämple.app"} {
		if validAndroidPackageName(value) {
			t.Errorf("invalid package %q accepted", value)
		}
	}
}
