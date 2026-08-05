package evidence

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	evidencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/evidence"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"scenario-to-desktop-api/captures"
	"scenario-to-desktop-api/smoketest"
)

type fakeEvidenceClient struct {
	request *evidencev1.ReportTargetVerdictRequest
	err     error
}

func (f *fakeEvidenceClient) ReportTargetVerdict(_ context.Context, req *connect.Request[evidencev1.ReportTargetVerdictRequest]) (*connect.Response[evidencev1.ReportTargetVerdictResponse], error) {
	f.request = req.Msg
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(&evidencev1.ReportTargetVerdictResponse{Verdict: req.Msg.Verdict}), nil
}

func TestProducerMappersRoundTripAllContractFields(t *testing.T) {
	created := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	capture := captures.Capture{ID: "journey-1", Type: captures.CaptureJourney, FileSizeBytes: 123, Checksum: "sha256:abc", CreatedAt: created}
	ref := CaptureToEvidenceRef(capture)
	if ref.Producer != "scenario-to-desktop" || ref.ArtifactId != capture.ID || ref.Kind != string(capture.Type) || ref.SizeBytes != 123 || ref.Checksum != capture.Checksum || !ref.CreatedAt.AsTime().Equal(created) {
		t.Fatalf("capture mapper dropped fields: %#v", ref)
	}
	node, job := "node-1", "job-1"
	target := TargetToEvidenceTarget(&domainv1.EvidenceTarget{Kind: domainv1.EvidenceTarget_KIND_BRIDGE_NODE, BridgeNodeId: &node, BridgeJobId: &job}, "Windows")
	if target.Ramp != "scenario-to-desktop" || target.Platform != "windows" || target.Os != "windows" || target.DeviceKind != commonv1.DeviceKind_DEVICE_KIND_PHYSICAL || target.GetBridgeNodeId() != node || target.GetBridgeJobId() != job {
		t.Fatalf("target mapper dropped fields: %#v", target)
	}
}

func TestReporterSendsReferencesAndMapsDegradedToFailed(t *testing.T) {
	fake := &fakeEvidenceClient{}
	reporter := NewConnectReporter(fake)
	err := reporter.ReportJourney(context.Background(), smoketest.EvidenceReportInput{
		ProfileID: "profile-1", GitCommit: "deadbeef", Platform: "linux", RunID: "run-1",
		Disposition: "degraded", Target: &domainv1.EvidenceTarget{Kind: domainv1.EvidenceTarget_KIND_LOCAL},
		Captures: []captures.Capture{{ID: "journey-1", Type: captures.CaptureJourney, FileSizeBytes: 10, Checksum: "sha256:j", CreatedAt: time.Now()}, {ID: "recording-1", Type: captures.CaptureRecording, FileSizeBytes: 20, Checksum: "sha256:r", CreatedAt: time.Now()}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if fake.request == nil || fake.request.Verdict.Disposition != commonv1.Disposition_DISPOSITION_FAILED || len(fake.request.Verdict.Refs) != 2 {
		t.Fatalf("unexpected reported verdict: %#v", fake.request)
	}
	if len(fake.request.Verdict.Refs[0].ProtoReflect().GetUnknown()) != 0 {
		t.Fatal("unexpected unknown artifact bytes in evidence reference")
	}
}

func TestReporterUnreachableFailsClosed(t *testing.T) {
	reporter := NewConnectReporter(&fakeEvidenceClient{err: errors.New("connection refused")})
	err := reporter.ReportJourney(context.Background(), smoketest.EvidenceReportInput{ProfileID: "profile", GitCommit: "commit", Platform: "linux", RunID: "run", Disposition: "pass", Captures: []captures.Capture{{ID: "j", Type: captures.CaptureJourney, CreatedAt: time.Now()}}})
	if err == nil {
		t.Fatal("unreachable deployment-manager must return an error")
	}
}

func TestReporterRequiresJourneyAndRecording(t *testing.T) {
	for name, items := range map[string][]captures.Capture{
		"journey only": {{ID: "j", Type: captures.CaptureJourney}},
		"recording only": {{ID: "r", Type: captures.CaptureRecording}},
	} {
		t.Run(name, func(t *testing.T) {
			fake := &fakeEvidenceClient{}
			err := NewConnectReporter(fake).ReportJourney(context.Background(), smoketest.EvidenceReportInput{
				ProfileID: "profile", GitCommit: "commit", Platform: "linux", RunID: "run",
				Disposition: "pass", Captures: items,
			})
			if err == nil {
				t.Fatal("incomplete evidence set must fail closed")
			}
			if fake.request != nil {
				t.Fatal("incomplete evidence must not be sent")
			}
		})
	}
}
