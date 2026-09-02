package evidence

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

type fakeRPC struct {
	listRequest    *domainv1.ListEvidenceCapturesRequest
	getRequest     *domainv1.GetEvidenceCaptureRequest
	summaryRequest *domainv1.ListEvidenceCapturesRequest
}

func (f *fakeRPC) ListEvidenceCaptures(_ context.Context, req *connect.Request[domainv1.ListEvidenceCapturesRequest]) (*connect.Response[domainv1.ListEvidenceCapturesResponse], error) {
	f.listRequest = req.Msg
	return connect.NewResponse(&domainv1.ListEvidenceCapturesResponse{Captures: []*domainv1.EvidenceCapture{{CaptureId: "capture-1", Kind: "journey"}}}), nil
}

func (f *fakeRPC) GetEvidenceCapture(_ context.Context, req *connect.Request[domainv1.GetEvidenceCaptureRequest]) (*connect.Response[domainv1.GetEvidenceCaptureResponse], error) {
	f.getRequest = req.Msg
	return connect.NewResponse(&domainv1.GetEvidenceCaptureResponse{Content: []byte(`{"disposition":"pass","steps":[{"name":"click","action":"pointer_click","disposition":"passed","before_capture_id":"before","after_capture_id":"after"}]}`)}), nil
}

func (f *fakeRPC) GetEvidenceCapturesSummary(_ context.Context, req *connect.Request[domainv1.ListEvidenceCapturesRequest]) (*connect.Response[domainv1.EvidenceCapturesSummary], error) {
	f.summaryRequest = req.Msg
	return connect.NewResponse(&domainv1.EvidenceCapturesSummary{Count: 1, TotalBytes: 42}), nil
}

func TestEvidencePrimitivesUseTypedScenarioRequests(t *testing.T) {
	fake := &fakeRPC{}
	commands := &Commands{rpc: fake}

	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true}}}
	listModes := cliapptest.RunPrimitiveHandlerModes(t, commands.listPrimitive(), schema, []string{"demo"}, nil)
	if listModes.HumanErr != nil || listModes.JSONErr != nil {
		t.Fatalf("list primitive errors: human=%v json=%v", listModes.HumanErr, listModes.JSONErr)
	}
	if fake.listRequest.GetScenarioName() != "demo" {
		t.Fatalf("list request scenario = %q, want demo", fake.listRequest.GetScenarioName())
	}

	summaryModes := cliapptest.RunPrimitiveHandlerModes(t, commands.summaryPrimitive(), schema, []string{"demo"}, nil)
	if summaryModes.HumanErr != nil || summaryModes.JSONErr != nil {
		t.Fatalf("summary primitive errors: human=%v json=%v", summaryModes.HumanErr, summaryModes.JSONErr)
	}
	if fake.summaryRequest.GetScenarioName() != "demo" {
		t.Fatalf("summary request scenario = %q, want demo", fake.summaryRequest.GetScenarioName())
	}

	journeyModes := cliapptest.RunPrimitiveHandlerModes(t, commands.journeyPrimitive(), schema, []string{"demo"}, nil)
	if journeyModes.HumanErr != nil || journeyModes.JSONErr != nil {
		t.Fatalf("journey primitive errors: human=%v json=%v", journeyModes.HumanErr, journeyModes.JSONErr)
	}
	if fake.getRequest.GetScenarioName() != "demo" || fake.getRequest.GetCaptureId() != "capture-1" {
		t.Fatalf("journey request = %#v, want demo/capture-1", fake.getRequest)
	}
}

func TestEvidenceCommandsRequireScenario(t *testing.T) {
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true}}}
	if _, err := cliapptest.NewTestRunContextFromArgs(schema, nil, nil, nil, nil); err == nil {
		t.Fatal("missing scenario should fail production argument parsing")
	}
}

func TestReadDesktopBuildFactUsesDurableRecordTimestamp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "desktop_records_v2.json")
	content := `[{"scenario_name":"web-console","updated_at":"2026-09-01T11:00:00Z"},{"scenario_name":"web-console","updated_at":"2026-09-01T12:00:00Z"}]`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	fact, err := readDesktopBuildFact(path, "web-console", 30)
	if err != nil {
		t.Fatal(err)
	}
	if fact.GetValue() != 1 || fact.GetDimension() != "producer:scenario-to-desktop" {
		t.Fatalf("fact = %v/%s", fact.GetValue(), fact.GetDimension())
	}
	if got := fact.GetObservedAt().AsTime(); !got.Equal(time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("observed_at = %s", got)
	}
}

func TestReadDesktopBuildFactRejectsUnknownScenario(t *testing.T) {
	path := filepath.Join(t.TempDir(), "desktop_records_v2.json")
	if err := os.WriteFile(path, []byte(`[{"scenario_name":"other","updated_at":"2026-09-01T12:00:00Z"}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readDesktopBuildFact(path, "web-console", 30); err == nil {
		t.Fatal("unknown scenario was accepted")
	}
}
