package models

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"

	"image-tools/internal/capabilities"
	internalmodels "image-tools/internal/models"
	"image-tools/internal/modelstorage"

	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
)

type fakeModelStorage struct {
	dryRun    bool
	requested []string
	keep      map[string]bool
}

func (f *fakeModelStorage) Config() modelstorage.Config {
	return modelstorage.Config{Policy: modelstorage.DefaultPolicy(), AutoAcquire: true, AutoOffload: true, Interval: 30 * time.Minute}
}

func (f *fakeModelStorage) report() modelstorage.Report {
	p := modelstorage.DefaultPolicy()
	return modelstorage.Report{Policy: p, Selection: p.Select([]modelstorage.Candidate{{ModelID: "sd-1.5", Bytes: 4 << 30, Idle: 400 * time.Hour, LastUseBasis: modelstorage.UseBasisInstall}}, p.Evaluate(p.FloorBytes+(50<<30)), 0, f.requested)}
}

func (f *fakeModelStorage) Plan(context.Context, string, int64, []string) (modelstorage.Report, error) {
	return f.report(), nil
}

func (f *fakeModelStorage) Offload(_ context.Context, _ string, _ int64, requested []string, dryRun bool) (modelstorage.Report, []modelstorage.Receipt, error) {
	f.dryRun, f.requested = dryRun, requested
	if dryRun {
		return f.report(), nil, nil
	}
	return f.report(), []modelstorage.Receipt{{ModelID: "sd-1.5", Outcome: modelstorage.OutcomeRemoved, Bytes: 4 << 30}}, nil
}

func (f *fakeModelStorage) Receipts(context.Context, int) ([]modelstorage.Receipt, error) {
	return nil, nil
}

func (f *fakeModelStorage) SetKeep(_ context.Context, id string, keep bool, _ string) error {
	if id == "missing" {
		return internalmodels.ErrModelNotFound
	}
	f.keep[id] = keep
	return nil
}

func TestStorageRPCsReportPlanAndHonourDryRun(t *testing.T) {
	h, _ := newTestHandler(t, capabilities.Host{OS: "linux", Arch: "amd64", Cores: 8})
	storage := &fakeModelStorage{keep: map[string]bool{}}
	h.deps.Storage = storage
	ctx := context.Background()

	status, err := h.GetModelStorage(ctx, connect.NewRequest(&modelsv1.GetModelStorageRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	report := status.Msg.GetReport()
	if report.GetMode() != modelstorage.ModeAdaptive || report.GetIdleThresholdSeconds() <= 0 || len(report.GetCandidates()) != 1 || !report.GetCandidates()[0].GetSelected() {
		t.Fatalf("status report=%v", report)
	}
	if report.GetPolicy().GetFloorBytes() != 120<<30 || !report.GetPolicy().GetAutoOffload() {
		t.Fatalf("policy=%v", report.GetPolicy())
	}

	resp, err := h.OffloadModels(ctx, connect.NewRequest(&modelsv1.OffloadModelsRequest{ModelIds: []string{"sd-1.5"}}))
	if err != nil || !storage.dryRun || resp.Msg.GetApplied() || len(resp.Msg.GetReceipts()) != 0 {
		t.Fatalf("unset apply must be a dry-run: dry=%t resp=%v err=%v", storage.dryRun, resp, err)
	}
	resp, err = h.OffloadModels(ctx, connect.NewRequest(&modelsv1.OffloadModelsRequest{Apply: true}))
	if err != nil || storage.dryRun || len(resp.Msg.GetReceipts()) != 1 {
		t.Fatalf("apply: dry=%t resp=%v err=%v", storage.dryRun, resp, err)
	}
	if _, err := h.OffloadModels(ctx, connect.NewRequest(&modelsv1.OffloadModelsRequest{ModelIds: []string{"no-such-model"}})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("unknown model err=%v, want not_found", err)
	}

	if _, err := h.SetModelKeep(ctx, connect.NewRequest(&modelsv1.SetModelKeepRequest{Id: "sd-1.5", Keep: true})); err != nil || !storage.keep["sd-1.5"] {
		t.Fatalf("keep err=%v pins=%v", err, storage.keep)
	}
	if _, err := h.SetModelKeep(ctx, connect.NewRequest(&modelsv1.SetModelKeepRequest{Id: "missing", Keep: true})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("missing keep err=%v", err)
	}
}

func TestStorageRPCsUnavailableWithoutOwner(t *testing.T) {
	h, _ := newTestHandler(t, capabilities.Host{OS: "linux", Arch: "amd64", Cores: 8})
	_, err := h.GetModelStorage(context.Background(), connect.NewRequest(&modelsv1.GetModelStorageRequest{}))
	var ce *connect.Error
	if !errors.As(err, &ce) || ce.Code() != connect.CodeUnimplemented {
		t.Fatalf("err=%v, want unimplemented", err)
	}
}
