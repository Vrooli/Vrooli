package releases

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	validationmatrix "github.com/vrooli/vrooli/packages/delivery-ramp-go/validationmatrix"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"scenario-to-android/internal/targets"
)

func TestCatalogPublishesConformanceJourneyAndTargetContract(t *testing.T) {
	catalog, err := (Catalog{Probe: targets.Prober{
		LookPath: func(string) (string, error) { return "/tool", nil },
		KVM:      func() (bool, bool, string) { return true, true, "" },
		Run:      func(context.Context, string, ...string) ([]byte, error) { return []byte("vrooli-api36\n"), nil },
	}, Journey: testJourneySelection()}).Resolve(context.Background(), "hello-mobile")
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Journeys) != 1 || catalog.Journeys[0].JourneyID == "" {
		t.Fatalf("catalog omitted Android journey: %#v", catalog.Journeys)
	}
	if len(catalog.Targets) < 1 || catalog.Targets[0].Descriptor.GetTargetId() != "android:emulator:local" {
		t.Fatalf("catalog omitted local target: %#v", catalog.Targets)
	}
}

func TestCatalogDoesNotCreateSyntheticDuplicateForPromotedEmulator(t *testing.T) {
	catalog, err := (Catalog{Probe: targets.Prober{
		LookPath: func(string) (string, error) { return "/tool", nil },
		KVM:      func() (bool, bool, string) { return true, true, "" },
		Run:      func(context.Context, string, ...string) ([]byte, error) { return []byte("vrooli-api36\n"), nil },
		Devices: staticDevices{items: []targets.DeviceObservation{{
			ID: "android-emulator", Serial: "emulator-5554", Label: "AVD", OS: "Android",
			Transport:    deliveryramp.Transport{Kind: deliveryramp.TransportLocal, ID: "emulator-5554", Available: true},
			Capabilities: []string{deliveryramp.CapabilityDeviceControl}, Available: true,
		}}},
	}, Journey: testJourneySelection()}).Resolve(context.Background(), "hello-mobile")
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Targets) != 1 || catalog.Targets[0].Descriptor.GetTargetId() != "android-emulator" {
		t.Fatalf("catalog retained a non-executable synthetic target: %#v", catalog.Targets)
	}
}

func TestTargetFromDescriptorPreservesPhysicalVerdictKind(t *testing.T) {
	physical := targetFromDescriptor(&domainv1.ValidationTargetDescriptor{TargetId: "android-024665203bca17fa", DisplayName: "Galaxy A03s", Available: true})
	if physical.DeviceKind != "physical" || physical.Mode != "physical" {
		t.Fatalf("physical descriptor was not preserved: %#v", physical)
	}
	emulator := targetFromDescriptor(&domainv1.ValidationTargetDescriptor{TargetId: "android:emulator:local", DisplayName: "Local emulator", Available: true})
	if emulator.DeviceKind != "emulator" || emulator.Mode != "emulator" {
		t.Fatalf("emulator descriptor was not classified: %#v", emulator)
	}
}

func testJourneySelection() validationmatrix.JourneySelection {
	return validationmatrix.JourneySelection{JourneyID: "android-generated-app-conformance-v1", DisplayName: "Android generated-app conformance", SourcePath: "internal/journeys/plan.go", ExecutionMode: "platform", Required: true, Category: "android"}
}

func TestJourneyEvidenceClassifiesClaimedVideoByMediaType(t *testing.T) {
	result := deliveryramp.JourneyResult{Steps: []deliveryramp.JourneyStep{{Evidence: []deliveryramp.EvidenceReference{{ID: "video", Kind: "native", MediaType: "video/mp4", URI: "device-control://evidence/video", Checksum: "sha256:video", Redacted: true}}}}}
	request := validationmatrix.CellRequest{RunID: "run-1", Cell: &domainv1.ValidationCell{CellId: "cell-1"}, Target: &domainv1.ValidationTargetDescriptor{TargetId: "android-phone"}, Journey: validationmatrix.JourneySelection{JourneyID: "android-generated-app-conformance-v1"}}
	evidence := journeyEvidence(result, request)
	if len(evidence) != 2 || evidence[0].GetKind() != domainv1.LayeredEvidence_KIND_DESKTOP_RUNTIME {
		t.Fatalf("claimed video was not classified as desktop runtime: %#v", evidence)
	}
}

func TestJourneyFailureReasonPrefersFailedStepDetail(t *testing.T) {
	result := deliveryramp.JourneyResult{Disposition: deliveryramp.DispositionFailed, Steps: []deliveryramp.JourneyStep{{Error: `visible surface unavailable: lock state is "locked" and screen state is "on"`}}}
	if got := journeyFailureReason(result); got != result.Steps[0].Error {
		t.Fatalf("failure detail was discarded: %q", got)
	}
}

func TestJourneyEvidenceIncludesClockCalibrationReferences(t *testing.T) {
	result := deliveryramp.JourneyResult{ClockOffsetStart: &deliveryramp.ClockOffsetSample{Evidence: deliveryramp.EvidenceReference{ID: "clock-start", Kind: "log", Checksum: "sha256:start", Redacted: true}}, ClockOffsetEnd: &deliveryramp.ClockOffsetSample{Evidence: deliveryramp.EvidenceReference{ID: "clock-end", Kind: "log", Checksum: "sha256:end", Redacted: true}}}
	request := validationmatrix.CellRequest{RunID: "run-1", Cell: &domainv1.ValidationCell{CellId: "cell-1"}, Target: &domainv1.ValidationTargetDescriptor{TargetId: "android-phone"}, Journey: validationmatrix.JourneySelection{JourneyID: "android-generated-app-conformance-v1"}}
	evidence := journeyEvidence(result, request)
	if len(evidence) != 3 || evidence[0].GetEvidenceId() != "clock-start" || evidence[1].GetEvidenceId() != "clock-end" {
		t.Fatalf("clock calibration references were not retained: %#v", evidence)
	}
}

func TestAndroidArtifactUsesBuilderPackageDefault(t *testing.T) {
	t.Setenv("ANDROID_ARTIFACT_PACKAGE", "")
	dir := t.TempDir()
	apkPath := filepath.Join(dir, "apk", "app-debug.apk")
	aabPath := filepath.Join(dir, "bundle", "app-debug.aab")
	require.NoError(t, os.MkdirAll(filepath.Dir(apkPath), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(aabPath), 0o755))
	apk := []byte("apk")
	aab := []byte("aab")
	require.NoError(t, os.WriteFile(apkPath, apk, 0o600))
	require.NoError(t, os.WriteFile(aabPath, aab, 0o600))
	hash := sha256.New()
	_, _ = hash.Write(apk)
	_, _ = hash.Write([]byte("\x00"))
	_, _ = hash.Write(aab)
	digest := "sha256:" + hex.EncodeToString(hash.Sum(nil))

	artifact, err := androidArtifact(apkPath, digest)
	require.NoError(t, err)
	require.Equal(t, defaultAndroidPackageName, artifact.Metadata["package_name"])
}

type staticDevices struct {
	items []targets.DeviceObservation
}

func (s staticDevices) List(context.Context) ([]targets.DeviceObservation, error) {
	return s.items, nil
}
