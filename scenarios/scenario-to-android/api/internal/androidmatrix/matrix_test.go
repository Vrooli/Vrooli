package androidmatrix

import (
	"context"
	"testing"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	validationmatrix "github.com/vrooli/vrooli/packages/delivery-ramp-go/validationmatrix"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"scenario-to-android/internal/androidprobe"
)

func TestCatalogPublishesConformanceJourneyAndTargetContract(t *testing.T) {
	catalog, err := (Catalog{Probe: androidprobe.Prober{
		LookPath: func(string) (string, error) { return "/tool", nil },
		KVM:      func() (bool, bool, string) { return true, true, "" },
		Run:      func(context.Context, string, ...string) ([]byte, error) { return []byte("vrooli-api36\n"), nil },
	}}).Resolve(context.Background(), "hello-mobile")
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
	catalog, err := (Catalog{Probe: androidprobe.Prober{
		LookPath: func(string) (string, error) { return "/tool", nil },
		KVM:      func() (bool, bool, string) { return true, true, "" },
		Run:      func(context.Context, string, ...string) ([]byte, error) { return []byte("vrooli-api36\n"), nil },
		Devices: staticDevices{items: []androidprobe.DeviceObservation{{
			ID: "android-emulator", Serial: "emulator-5554", Label: "AVD", OS: "Android",
			Transport:    deliveryramp.Transport{Kind: deliveryramp.TransportLocal, ID: "emulator-5554", Available: true},
			Capabilities: []string{deliveryramp.CapabilityDeviceControl}, Available: true,
		}}},
	}}).Resolve(context.Background(), "hello-mobile")
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

type staticDevices struct {
	items []androidprobe.DeviceObservation
}

func (s staticDevices) List(context.Context) ([]androidprobe.DeviceObservation, error) {
	return s.items, nil
}
