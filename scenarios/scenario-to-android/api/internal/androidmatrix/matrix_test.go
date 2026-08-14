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

func TestJourneyEvidenceClassifiesClaimedVideoByMediaType(t *testing.T) {
	result := deliveryramp.JourneyResult{Steps: []deliveryramp.JourneyStep{{Evidence: []deliveryramp.EvidenceReference{{ID: "video", Kind: "native", MediaType: "video/mp4", URI: "device-control://evidence/video", Checksum: "sha256:video", Redacted: true}}}}}
	request := validationmatrix.CellRequest{RunID: "run-1", Cell: &domainv1.ValidationCell{CellId: "cell-1"}, Target: &domainv1.ValidationTargetDescriptor{TargetId: "android-phone"}, Journey: validationmatrix.JourneySelection{JourneyID: "android-generated-app-conformance-v1"}}
	evidence := journeyEvidence(result, request)
	if len(evidence) != 2 || evidence[0].GetKind() != domainv1.LayeredEvidence_KIND_DESKTOP_RUNTIME {
		t.Fatalf("claimed video was not classified as desktop runtime: %#v", evidence)
	}
}

type staticDevices struct {
	items []androidprobe.DeviceObservation
}

func (s staticDevices) List(context.Context) ([]androidprobe.DeviceObservation, error) {
	return s.items, nil
}
