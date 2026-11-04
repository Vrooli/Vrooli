package releases

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"scenario-to-android/internal/targets"

	"github.com/stretchr/testify/require"
	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	validationmatrix "github.com/vrooli/vrooli/packages/delivery-ramp-go/validationmatrix"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
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

func TestJourneyEvidenceUsesRedactedRuntimeAndVerifiedPackageState(t *testing.T) {
	result := deliveryramp.JourneyResult{
		ReviewRecording: &deliveryramp.EvidenceReference{ID: "raw-video", Kind: "recording", Checksum: "sha256:raw", Redacted: false},
		Steps: []deliveryramp.JourneyStep{
			{ID: "cold-bas", Evidence: []deliveryramp.EvidenceReference{{ID: "raw-screenshot", Kind: "screenshot", Checksum: "sha256:screenshot", Redacted: false}, {ID: "bas-summary-exec-1", Kind: "bas-runtime-summary", URI: "validation://bas/executions/exec-1/summary", MediaType: "application/json", Checksum: "sha256:summary", Redacted: true}}},
			{ID: "temporary-cleanup/verify-absent", Action: "package-state", Disposition: deliveryramp.StepPassed, ExpectedState: "package absent, or installed identity matches the exact temporary artifact before removal", ObservedState: "scenario package is absent", AssertionStatus: "observed"},
		},
		ClockOffsetStart: &deliveryramp.ClockOffsetSample{Evidence: deliveryramp.EvidenceReference{ID: "raw-clock", Kind: "clock", Checksum: "sha256:clock", Redacted: false}},
	}
	request := validationmatrix.CellRequest{
		RunID: "run-1", Cell: &domainv1.ValidationCell{CellId: "cell-1"},
		Target:   &domainv1.ValidationTargetDescriptor{TargetId: "android-phone"},
		Journey:  validationmatrix.JourneySelection{JourneyID: "android-scenario-app-temporary-test-v1", ExecutionMode: "platform"},
		Metadata: map[string]string{"package_name": "com.example.generated"},
	}
	evidence := journeyEvidence(result, request)
	matrix := &domainv1.ValidationMatrix{
		MatrixId:            "matrix-1",
		Journeys:            []*domainv1.JourneyCatalogItem{{JourneyId: request.Journey.JourneyID, ExecutionMode: "platform", Required: true}},
		Targets:             []*domainv1.ValidationTargetDescriptor{{TargetId: "android-phone", Available: true}},
		EnvironmentProfiles: []domainv1.ValidationEnvironmentProfile{domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_NORMAL},
		Cells:               []*domainv1.ValidationCell{{CellId: "cell-1", JourneyId: request.Journey.JourneyID, TargetId: "android-phone", EnvironmentProfile: domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_NORMAL, Required: true, Applicable: true, Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS, Evidence: evidence}},
	}
	gate := validationmatrix.EvaluateReleaseGate(matrix)
	if !gate.GetPassed() {
		t.Fatalf("verified redacted evidence did not satisfy the platform gate: %#v evidence=%#v", gate, evidence)
	}
	for _, item := range evidence {
		if !item.GetRedacted() {
			t.Fatalf("unredacted source evidence entered the matrix: %#v", item)
		}
		if item.GetKind() == domainv1.LayeredEvidence_KIND_TARGET {
			want := hashText("target=android-phone package=com.example.generated state=scenario package is absent step=temporary-cleanup/verify-absent")
			require.Equal(t, want, item.GetSha256(), "target evidence must bind the observed terminal absence, not the cleanup precondition")
		}
	}
}

func TestJourneyEvidenceAcceptsDurablePackageReadbackAssertion(t *testing.T) {
	result := deliveryramp.JourneyResult{Steps: []deliveryramp.JourneyStep{{
		ID: "verify-installed", Action: "package-state", Disposition: deliveryramp.StepPassed,
		ExpectedState: "the exact scenario package remains installed", AssertionStatus: "observed",
	}}}
	request := validationmatrix.CellRequest{
		Cell:     &domainv1.ValidationCell{CellId: "cell-durable"},
		Target:   &domainv1.ValidationTargetDescriptor{TargetId: "android-phone"},
		Journey:  validationmatrix.JourneySelection{JourneyID: "android-scenario-app-durable-v1"},
		Metadata: map[string]string{"package_name": "com.example.generated"},
	}
	evidence := verifiedScenarioPackageEvidence(result, request)
	if evidence == nil || evidence.GetKind() != domainv1.LayeredEvidence_KIND_TARGET || !evidence.GetRedacted() {
		t.Fatalf("verified durable package readback did not produce target evidence: %#v", evidence)
	}
}

func TestJourneyEvidenceAcceptsUpdatedPackageReadbackAssertion(t *testing.T) {
	result := deliveryramp.JourneyResult{Steps: []deliveryramp.JourneyStep{{
		ID: "verify-updated", Action: "package-state", Disposition: deliveryramp.StepPassed,
		ExpectedState: "the higher-version scenario package remains installed", AssertionStatus: "observed",
	}}}
	request := validationmatrix.CellRequest{
		Cell:     &domainv1.ValidationCell{CellId: "cell-update"},
		Target:   &domainv1.ValidationTargetDescriptor{TargetId: "android-phone"},
		Journey:  validationmatrix.JourneySelection{JourneyID: "android-scenario-app-update-v1"},
		Metadata: map[string]string{"package_name": "com.example.generated"},
	}
	evidence := verifiedScenarioPackageEvidence(result, request)
	require.NotNil(t, evidence)
	require.Equal(t, domainv1.LayeredEvidence_KIND_TARGET, evidence.GetKind())
	require.True(t, evidence.GetRedacted())
}

func TestJourneyEvidenceAcceptsVerifiedInstalledPackageReadback(t *testing.T) {
	result := deliveryramp.JourneyResult{Steps: []deliveryramp.JourneyStep{{
		ID: "verify-installed", Action: "package-state", Disposition: deliveryramp.StepPassed,
		ExpectedState: "the scenario package remains installed", AssertionStatus: "observed",
	}}}
	request := validationmatrix.CellRequest{
		Cell:     &domainv1.ValidationCell{CellId: "cell-verify"},
		Target:   &domainv1.ValidationTargetDescriptor{TargetId: "android-phone"},
		Journey:  validationmatrix.JourneySelection{JourneyID: "android-scenario-app-verify-v1"},
		Metadata: map[string]string{"package_name": "com.example.generated", "install_mode": "verify"},
	}
	evidence := verifiedScenarioPackageEvidence(result, request)
	require.NotNil(t, evidence)
	require.Equal(t, domainv1.LayeredEvidence_KIND_TARGET, evidence.GetKind())
	require.True(t, evidence.GetRedacted())
}

func TestJourneyEvidenceAddsConformanceTargetProofFromTerminalPackageReadback(t *testing.T) {
	result := deliveryramp.JourneyResult{Steps: []deliveryramp.JourneyStep{
		{
			ID: "clean_uninstall/assert-absent", Action: "package-state", Disposition: deliveryramp.StepPassed,
			ExpectedState: "the app package is absent", AssertionStatus: "observed",
		},
		{ID: "bas", Evidence: []deliveryramp.EvidenceReference{{ID: "bas-summary", Kind: "bas-runtime-summary", URI: "validation://bas/executions/execution/summary", MediaType: "application/json", Checksum: "sha256:bas", Redacted: true}}},
	}}
	request := validationmatrix.CellRequest{
		Cell:     &domainv1.ValidationCell{CellId: "cell-conformance"},
		Target:   &domainv1.ValidationTargetDescriptor{TargetId: "android-emulator"},
		Journey:  testJourneySelection(),
		Metadata: map[string]string{"package_name": "com.example.generated"},
	}
	evidence := journeyEvidence(result, request)
	var hasTargetProof bool
	for _, item := range evidence {
		if item.GetKind() == domainv1.LayeredEvidence_KIND_TARGET && item.GetEvidenceId() == "target-state-cell-conformance" {
			hasTargetProof = true
		}
	}
	if !hasTargetProof {
		t.Fatalf("terminal package readback did not produce target evidence: %#v", evidence)
	}
	matrix := &domainv1.ValidationMatrix{
		MatrixId:            "matrix-conformance",
		Journeys:            []*domainv1.JourneyCatalogItem{{JourneyId: request.Journey.JourneyID, ExecutionMode: "platform", Required: true}},
		Targets:             []*domainv1.ValidationTargetDescriptor{{TargetId: "android-emulator", Available: true}},
		EnvironmentProfiles: []domainv1.ValidationEnvironmentProfile{domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_NORMAL},
		Cells:               []*domainv1.ValidationCell{{CellId: "cell-conformance", JourneyId: request.Journey.JourneyID, TargetId: "android-emulator", EnvironmentProfile: domainv1.ValidationEnvironmentProfile_VALIDATION_ENVIRONMENT_PROFILE_NORMAL, Required: true, Applicable: true, Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS, Evidence: evidence}},
	}
	if gate := validationmatrix.EvaluateReleaseGate(matrix); !gate.GetPassed() {
		t.Fatalf("complete conformance evidence did not satisfy the platform gate: %#v evidence=%#v", gate, evidence)
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

func TestAndroidArtifactResolvesRetainedAPKAndAABPair(t *testing.T) {
	t.Setenv("ANDROID_ARTIFACT_PACKAGE", "")
	dir := t.TempDir()
	apkPath := filepath.Join(dir, "app.apk")
	aabPath := filepath.Join(dir, "app.aab")
	apk := []byte("retained apk")
	aab := []byte("retained aab")
	require.NoError(t, os.WriteFile(apkPath, apk, 0o600))
	require.NoError(t, os.WriteFile(aabPath, aab, 0o600))
	hash := sha256.New()
	_, _ = hash.Write(apk)
	_, _ = hash.Write([]byte("\x00"))
	_, _ = hash.Write(aab)
	digest := "sha256:" + hex.EncodeToString(hash.Sum(nil))

	artifact, err := androidArtifact(apkPath, digest)
	require.NoError(t, err)
	require.Equal(t, digest, artifact.Checksum)
	require.Equal(t, apkPath, artifact.LocalPath)
}

type staticDevices struct {
	items []targets.DeviceObservation
}

func (s staticDevices) List(context.Context) ([]targets.DeviceObservation, error) {
	return s.items, nil
}
