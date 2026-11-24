package catalog

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"
	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog"
	catalogconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog/catalog_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
)

type gateClientSpy struct {
	catalogconnect.CatalogServiceClient
	runCalls    int
	recordCalls int
}

type readinessClientSpy struct {
	catalogconnect.CatalogServiceClient
	request  *catalogv1.GetReadinessRequest
	err      error
	response *connect.Response[catalogv1.GetReadinessResponse]
}

func (s *readinessClientSpy) GetReadiness(_ context.Context, req *connect.Request[catalogv1.GetReadinessRequest]) (*connect.Response[catalogv1.GetReadinessResponse], error) {
	s.request = req.Msg
	return s.response, s.err
}

func TestReadinessCallPassesFloorAndRejectsMissingResponse(t *testing.T) {
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "floor"}}}, Flags: map[string]string{"floor": "verified"}})
	client := &readinessClientSpy{response: connect.NewResponse(&catalogv1.GetReadinessResponse{Verdict: "ready"})}
	h := &handlers{client: client}
	got, err := h.readinessCall(ctx)
	if err != nil || got.GetVerdict() != "ready" || client.request.GetFloor() != "verified" {
		t.Fatalf("readinessCall = (%v, %v), floor request=%v", got, err, client.request)
	}
	client.response = nil
	if _, err := h.readinessCall(ctx); err == nil || err.Error() != "server returned no catalog readiness" {
		t.Fatalf("missing response error = %v", err)
	}
	client.err = errors.New("offline")
	if _, err := h.readinessCall(ctx); err == nil || !strings.Contains(err.Error(), "get catalog readiness") {
		t.Fatalf("API error = %v", err)
	}
}

func TestReadinessReportGroupsAndSortsTriage(t *testing.T) {
	msg := &catalogv1.GetReadinessResponse{
		Verdict: "blocked", Run: &catalogv1.ReadinessRun{RunId: "r1", Completed: true, CompletedAt: "now"},
		Config:             &catalogv1.ReadinessConfig{DeclaredFloor: "verified", AchievedRung: "draft", RungGap: 2},
		TriageOmittedCount: 3, NextSteps: []string{"rerun"},
		Triage: []*catalogv1.ReadinessTriageRow{
			{Gate: "zeta", AssetId: "a2", Message: "second", BlocksDownstream: 2, Weight: .5},
			{Gate: "alpha", AssetId: "a1", Message: "first", BlocksDownstream: 1, Weight: 1},
			{Gate: "zeta", AssetId: "a3", Message: "third"},
		},
	}
	report := (&handlers{}).readinessReport(cliapp.NewTestRunContext(cliapp.TestRunContextOptions{}), msg)
	if !strings.Contains(strings.Join(report.Status, "\n"), "3 omitted") || len(report.Triage) != 2 || report.Triage[0].Heading != "alpha" || report.Triage[1].Heading != "zeta" || len(report.Triage[1].Items) != 2 || len(report.NextSteps) != 1 {
		t.Fatalf("readiness report lost triage/status/next steps: %#v", report)
	}
}

func TestGraphReportIncludesDirectAndTransitiveRelationships(t *testing.T) {
	msg := &catalogv1.GetAssetRelationshipsResponse{Relationships: &catalogv1.AssetRelationships{
		Root: &catalogv1.AssetNode{AssetId: "button"}, DirectDependencies: []*catalogv1.AssetNode{{AssetId: "token"}},
		Closure:          []*catalogv1.AssetNode{{AssetId: "token"}, {AssetId: "color"}},
		DirectDependents: []*catalogv1.AssetNode{{AssetId: "form"}}, TransitiveDependents: []*catalogv1.AssetNode{{AssetId: "form"}, {AssetId: "page"}},
		ClosureBands: []*catalogv1.RungBand{{Rung: 2, RungName: "verified", Count: 2}},
	}}
	report := (&handlers{}).graphReport(cliapp.NewTestRunContext(cliapp.TestRunContextOptions{}), msg)
	joined := strings.Join(append(report.Summary, report.Results...), "\n")
	for _, expected := range []string{"button; blast radius 2", "depends-on: 1 direct, 2 in closure", "used-by: 1 direct, 2 transitive", "rung 2 (verified): 2"} {
		if !strings.Contains(joined, expected) {
			t.Errorf("graph report missing %q: %s", expected, joined)
		}
	}
}

func TestBuildListsStagesWithoutStartingGenerator(t *testing.T) {
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "list-stages", Bool: true}}},
		Flags:  map[string]string{"list-stages": "true"},
	})
	if err := (&handlers{}).build(ctx); err != nil {
		t.Fatalf("list stages: %v", err)
	}
}

func TestGateCallsRequireGateOrAll(t *testing.T) {
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "gate"}},
		Flags:       []cliapp.Flag{{Name: "all", Bool: true}},
	}})
	for _, tc := range []struct {
		name string
		call func(cliapp.OperationContext) (*catalogv1.RunGateResponse, error)
		want string
	}{
		{"inspection", (&handlers{}).gateCall, "catalog gates requires a gate name or --all"},
		{"record", (&handlers{}).recordGateCall, "catalog record-gate requires a gate name or --all"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := tc.call(ctx); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func (s *gateClientSpy) RunGate(_ context.Context, _ *connect.Request[catalogv1.RunGateRequest]) (*connect.Response[catalogv1.RunGateResponse], error) {
	s.runCalls++
	return connect.NewResponse(&catalogv1.RunGateResponse{Gate: "release-provenance"}), nil
}

func (s *gateClientSpy) RecordGate(_ context.Context, _ *connect.Request[catalogv1.RunGateRequest]) (*connect.Response[catalogv1.RunGateResponse], error) {
	s.recordCalls++
	return connect.NewResponse(&catalogv1.RunGateResponse{Gate: "release-provenance", EvidenceRowsWritten: 1}), nil
}

func TestReadinessFloorAllowsLegacyHealthAlias(t *testing.T) {
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{})
	if got := readinessFloor(ctx); got != "" {
		t.Fatalf("floor = %q, want empty for a command without --floor", got)
	}
}

func TestReadinessFloorReadsDeclaredFlag(t *testing.T) {
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "floor"}}},
		Flags:  map[string]string{"floor": "verified"},
	})
	if got := readinessFloor(ctx); got != "verified" {
		t.Fatalf("floor = %q, want verified", got)
	}
}

func TestCatalogInspectionAndRecordingUseSeparateAPIPaths(t *testing.T) {
	client := &gateClientSpy{}
	h := &handlers{client: client}
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cliapp.ArgSchema{
			Positionals: []cliapp.Positional{{Name: "gate"}},
			Flags: []cliapp.Flag{
				{Name: "all", Bool: true},
				{Name: "asset-id"},
				{Name: "calibration-only", Bool: true},
				{Name: "include-advisory", Bool: true},
			},
		},
		Positionals: map[string]string{"gate": "release-provenance"},
	})

	if _, err := h.gateCall(ctx); err != nil {
		t.Fatal(err)
	}
	if client.runCalls != 1 || client.recordCalls != 0 {
		t.Fatalf("inspection called RunGate=%d RecordGate=%d, want 1 and 0", client.runCalls, client.recordCalls)
	}
	if _, err := h.recordGateCall(ctx); err != nil {
		t.Fatal(err)
	}
	if client.runCalls != 1 || client.recordCalls != 1 {
		t.Fatalf("record command called RunGate=%d RecordGate=%d, want 1 and 1", client.runCalls, client.recordCalls)
	}
}

type catalogGateServiceStub struct {
	catalogconnect.UnimplementedCatalogServiceHandler
	runCalls    int
	recordCalls int
}

func (s *catalogGateServiceStub) RunGate(_ context.Context, _ *connect.Request[catalogv1.RunGateRequest]) (*connect.Response[catalogv1.RunGateResponse], error) {
	s.runCalls++
	return connect.NewResponse(&catalogv1.RunGateResponse{Gate: "release-provenance"}), nil
}

func (s *catalogGateServiceStub) RecordGate(_ context.Context, _ *connect.Request[catalogv1.RunGateRequest]) (*connect.Response[catalogv1.RunGateResponse], error) {
	s.recordCalls++
	return connect.NewResponse(&catalogv1.RunGateResponse{Gate: "release-provenance", EvidenceRowsWritten: 1}), nil
}

func TestCatalogManifestRoutesReadAndWriteCommandsToSeparateRPCs(t *testing.T) {
	server := &catalogGateServiceStub{}
	path, api := catalogconnect.NewCatalogServiceHandler(server)
	mux := http.NewServeMux()
	mux.Handle(path, api)
	core := cliapptest.NewTestApp(t, mux)
	manifest, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	group, err := Register(core, manifest)
	if err != nil {
		t.Fatal(err)
	}
	groups := []cliapp.SubcommandGroup{group}
	readOutput, err := cliapptest.RunCommand(t, cliapptest.FindCommand(t, groups, "catalog", "gates"), core, "release-provenance", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if server.runCalls != 1 || server.recordCalls != 0 {
		t.Fatalf("read command called RunGate=%d RecordGate=%d", server.runCalls, server.recordCalls)
	}
	var readResponse catalogv1.RunGateResponse
	if err := protojson.Unmarshal([]byte(readOutput), &readResponse); err != nil {
		t.Fatalf("decode catalog gates JSON: %v (%s)", err, readOutput)
	}
	if readResponse.GetEvidenceRowsWritten() != 0 {
		t.Fatalf("catalog gates reported %d persisted rows", readResponse.GetEvidenceRowsWritten())
	}

	writeOutput, err := cliapptest.RunCommand(t, cliapptest.FindCommand(t, groups, "catalog", "record-gate"), core, "release-provenance", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if server.runCalls != 1 || server.recordCalls != 1 {
		t.Fatalf("write command called RunGate=%d RecordGate=%d", server.runCalls, server.recordCalls)
	}
	var writeResponse catalogv1.RunGateResponse
	if err := protojson.Unmarshal([]byte(writeOutput), &writeResponse); err != nil {
		t.Fatalf("decode catalog record-gate JSON: %v (%s)", err, writeOutput)
	}
	if writeResponse.GetEvidenceRowsWritten() != 1 {
		t.Fatalf("catalog record-gate reported %d persisted rows, want 1", writeResponse.GetEvidenceRowsWritten())
	}
}
