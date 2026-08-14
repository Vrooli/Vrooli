package androidjourney

import (
	"context"
	"errors"
	"testing"
	"time"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

type fakeDevice struct {
	leaseLost bool
	noOffsets bool
	actions   []map[string]string
	evidence  []deliveryramp.EvidenceReference
}

func (f *fakeDevice) Acquire(context.Context, string, string, time.Duration) (Lease, error) {
	return Lease{ID: "lease-1", Token: "token-1"}, nil
}

func (f *fakeDevice) ValidateLease(context.Context, Lease) error {
	if f.leaseLost {
		return errors.New("lease expired")
	}
	return nil
}

func (f *fakeDevice) Execute(_ context.Context, _ Lease, _ string, arguments map[string]string) (ActionResult, error) {
	f.actions = append(f.actions, arguments)
	return ActionResult{Evidence: f.evidence}, nil
}
func (f *fakeDevice) StartRecording(context.Context, Lease) error { return nil }
func (f *fakeDevice) StopRecording(context.Context, Lease) (RecordingArtifact, error) {
	return RecordingArtifact{Reference: deliveryramp.EvidenceReference{ID: "video-1", Checksum: "sha256:video", Redacted: true, Kind: "video"}, StartMs: 10, EndMs: 100, HasOffsets: !f.noOffsets}, nil
}
func (f *fakeDevice) Release(context.Context, Lease) error { return nil }

type fakeBAS struct{}

func (fakeBAS) Execute(context.Context, BASRequest) (BASResult, error) {
	return BASResult{Completed: true}, nil
}

func testRequest() deliveryramp.DriverRequest {
	target := deliveryramp.Target{ID: "android:emulator:local", Label: "hello-mobile", Platform: "android", Available: true, Capabilities: []string{"android-webview"}, Transport: deliveryramp.Transport{Kind: deliveryramp.TransportLocal, ID: "local", Available: true}}
	return deliveryramp.DriverRequest{RunID: "run-1", Cell: deliveryramp.Cell{ID: "cell-1", Target: target}, Artifact: deliveryramp.Artifact{ImmutableRef: "android-debug:abc"}, Plan: deliveryramp.JourneyPlan{ID: "android-smoke-v1", Capability: "android-webview", Profile: "release", Steps: []deliveryramp.JourneyStepSpec{{ID: "bas", Action: "bas-flow", Purpose: "run BAS flow"}}}}
}

func TestDriverRequiresRecordingOffsets(t *testing.T) {
	device := &fakeDevice{noOffsets: true}
	_, err := (Driver{Devices: device, BAS: fakeBAS{}}).Execute(context.Background(), testRequest())
	if err == nil {
		t.Fatal("expected missing offset failure")
	}
}

func TestDriverFailsWhenLeaseIsLost(t *testing.T) {
	device := &fakeDevice{leaseLost: true}
	_, err := (Driver{Devices: device, BAS: fakeBAS{}}).Execute(context.Background(), testRequest())
	if err == nil {
		t.Fatal("expected lease loss failure")
	}
}

func TestDriverCompletesBASStepWithReferenceOnlyVideo(t *testing.T) {
	result, err := (Driver{Devices: &fakeDevice{}, BAS: fakeBAS{}}).Execute(context.Background(), testRequest())
	if err != nil {
		t.Fatal(err)
	}
	if result.Disposition != deliveryramp.DispositionPass || result.Steps[0].VideoStartOffsetMs == nil || result.Steps[0].VideoEndOffsetMs == nil {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestDriverForwardsPackageStateExpectation(t *testing.T) {
	device := &fakeDevice{}
	request := testRequest()
	request.Artifact.Metadata = map[string]string{"package_name": "com.example.generated", "apk_path": "/tmp/app.apk"}
	request.Plan.Steps = []deliveryramp.JourneyStepSpec{{ID: "assert-absent", Action: "package-state", Arguments: map[string]string{"target": "absent"}}}
	if _, err := (Driver{Devices: device}).Execute(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if len(device.actions) != 1 || device.actions[0]["expected"] != "absent" {
		t.Fatalf("package-state expectation was not forwarded: %#v", device.actions)
	}
}

func TestDriverRetainsNativeDeviceEvidence(t *testing.T) {
	device := &fakeDevice{evidence: []deliveryramp.EvidenceReference{{ID: "frame-1", Kind: "screenshot", Checksum: "sha256:frame", Redacted: true}}}
	request := testRequest()
	request.Artifact.Metadata = map[string]string{"package_name": "com.example.generated", "apk_path": "/tmp/app.apk"}
	request.Plan.Steps = []deliveryramp.JourneyStepSpec{{ID: "observe", Action: "observe", Arguments: map[string]string{"target": ""}}}
	result, err := (Driver{Devices: device}).Execute(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Steps) != 1 || len(result.Steps[0].Evidence) != 2 || result.Steps[0].Evidence[0].ID != "frame-1" {
		t.Fatalf("native evidence was not retained: %#v", result.Steps)
	}
}
