package catalog

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
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
