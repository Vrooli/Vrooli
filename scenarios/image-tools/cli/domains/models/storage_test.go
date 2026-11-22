package models

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	testutil "github.com/vrooli/cli-core/cliapptest"
	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
	modelsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models/models_v1connect"
)

type storageService struct {
	modelsconnect.UnimplementedModelsServiceHandler
	offload *modelsv1.OffloadModelsRequest
	keep    *modelsv1.SetModelKeepRequest
}

func storageReport() *modelsv1.ModelStorageReport {
	return &modelsv1.ModelStorageReport{
		Policy:               &modelsv1.ModelStoragePolicy{FloorBytes: 120 << 30, ReserveBytes: 24 << 30, AmpleBytes: 96 << 30, AutoAcquire: true, AutoOffload: true, IntervalSeconds: 1800},
		FreeBytes:            170 << 30,
		HeadroomBytes:        50 << 30,
		Mode:                 "adaptive",
		IdleThresholdSeconds: 3 * 86400,
		SelectedBytes:        4 << 30,
		Candidates: []*modelsv1.ModelStorageCandidate{
			{ModelId: "sd-1.5", Bytes: 4 << 30, IdleSeconds: 10 * 86400, LastUseBasis: "successful_job", Selected: true, SelectionReason: "idle_beyond_threshold"},
			{ModelId: "imported-sd15", Bytes: 7 << 30, LastUseBasis: "install_receipt", Protected: []string{"not_auto_reacquirable"}},
		},
	}
}

func (s *storageService) GetModelStorage(context.Context, *connect.Request[modelsv1.GetModelStorageRequest]) (*connect.Response[modelsv1.GetModelStorageResponse], error) {
	return connect.NewResponse(&modelsv1.GetModelStorageResponse{Report: storageReport(), Receipts: []*modelsv1.ModelOffloadReceipt{{ModelId: "dncnn", Outcome: "removed", Bytes: 1 << 20, Reason: "operator_requested", Trigger: "operator"}}}), nil
}

func (s *storageService) OffloadModels(_ context.Context, req *connect.Request[modelsv1.OffloadModelsRequest]) (*connect.Response[modelsv1.OffloadModelsResponse], error) {
	s.offload = req.Msg
	resp := &modelsv1.OffloadModelsResponse{Report: storageReport(), Applied: req.Msg.GetApply()}
	if req.Msg.GetApply() {
		resp.Receipts = []*modelsv1.ModelOffloadReceipt{{ModelId: "sd-1.5", Outcome: "removed", Bytes: 4 << 30, Reason: "idle_beyond_threshold"}}
	}
	return connect.NewResponse(resp), nil
}

func (s *storageService) SetModelKeep(_ context.Context, req *connect.Request[modelsv1.SetModelKeepRequest]) (*connect.Response[modelsv1.SetModelKeepResponse], error) {
	s.keep = req.Msg
	return connect.NewResponse(&modelsv1.SetModelKeepResponse{Id: req.Msg.GetId(), Kept: req.Msg.GetKeep()}), nil
}

func runStorageCommand(t *testing.T, name string, flags, positionals map[string]string) (*storageService, string) {
	t.Helper()
	svc := &storageService{}
	mux := http.NewServeMux()
	path, handler := modelsconnect.NewModelsServiceHandler(svc)
	mux.Handle(path, handler)
	core := testutil.NewTestApp(t, mux)
	group, err := Register(core, readManifest(t))
	if err != nil {
		t.Fatal(err)
	}
	var cmd *cliapp.Command
	for i := range group.Subcommands {
		if group.Subcommands[i].Name == name {
			cmd = &group.Subcommands[i]
		}
	}
	if cmd == nil {
		t.Fatalf("command %q is not registered", name)
	}
	var out bytes.Buffer
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: cmd.Args, Flags: flags, Positionals: positionals, Core: core, Stdout: &out, Stderr: &out})
	if err := cmd.RunCtx(ctx); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return svc, out.String()
}

func TestStorageShowsThresholdCandidatesAndReceipts(t *testing.T) {
	_, out := runStorageCommand(t, "storage", nil, nil)
	for _, want := range []string{"mode adaptive", "Idle threshold: 3.0d", "sd-1.5", "SELECTED (idle_beyond_threshold)", "protected: not_auto_reacquirable", "receipt", "dncnn removed"} {
		if !strings.Contains(out, want) {
			t.Errorf("storage output lacks %q:\n%s", want, out)
		}
	}
}

func TestOffloadDefaultsToDryRun(t *testing.T) {
	svc, out := runStorageCommand(t, "offload", map[string]string{"model": "sd-1.5,dncnn"}, nil)
	if svc.offload.GetApply() || strings.Join(svc.offload.GetModelIds(), ",") != "sd-1.5,dncnn" {
		t.Fatalf("request=%v, want a dry-run for both models", svc.offload)
	}
	if !strings.Contains(out, "Dry-run (nothing removed)") || !strings.Contains(out, "sd-1.5") {
		t.Fatalf("dry-run output:\n%s", out)
	}
	svc, out = runStorageCommand(t, "offload", map[string]string{"apply": "true"}, nil)
	if !svc.offload.GetApply() || !strings.Contains(out, "Offloaded 4.0 GiB") {
		t.Fatalf("apply request=%v output:\n%s", svc.offload, out)
	}
}

func TestKeepAndUnkeepSendThePin(t *testing.T) {
	svc, out := runStorageCommand(t, "keep", map[string]string{"reason": "showcase"}, map[string]string{"id": "sd-1.5"})
	if !svc.keep.GetKeep() || svc.keep.GetReason() != "showcase" || !strings.Contains(out, "Kept sd-1.5") {
		t.Fatalf("keep request=%v output=%s", svc.keep, out)
	}
	svc, out = runStorageCommand(t, "unkeep", nil, map[string]string{"id": "sd-1.5"})
	if svc.keep.GetKeep() || !strings.Contains(out, "Unkept sd-1.5") {
		t.Fatalf("unkeep request=%v output=%s", svc.keep, out)
	}
}
