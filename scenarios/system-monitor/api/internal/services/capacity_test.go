package services

import (
	"context"
	"errors"
	"testing"

	capacityapp "github.com/vrooli/vrooli/internal/app/capacity"
	engine "github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/internal/hostinventory"
)

// fakeCapacityApp is an injectable stand-in for the platform capacity service.
type fakeCapacityApp struct {
	listOut      capacityapp.ListOutput
	listErr      error
	reconcileOut capacityapp.ReconcileOutput
	reconcileErr error
	policyOut    capacityapp.PolicyOutput
	setKey       string
	setValue     string
	setErr       error
}

func (f *fakeCapacityApp) List(context.Context, capacityapp.ListRequest) (capacityapp.ListOutput, error) {
	return f.listOut, f.listErr
}

func (f *fakeCapacityApp) Reconcile(context.Context) (capacityapp.ReconcileOutput, error) {
	return f.reconcileOut, f.reconcileErr
}

func (f *fakeCapacityApp) PolicyGet(context.Context, string) (capacityapp.PolicyOutput, error) {
	return f.policyOut, nil
}

func (f *fakeCapacityApp) PolicySet(_ context.Context, key, value string) (capacityapp.PolicyOutput, error) {
	f.setKey, f.setValue = key, value
	if f.setErr != nil {
		return capacityapp.PolicyOutput{}, f.setErr
	}
	return f.policyOut, nil
}

func intPtr(v int) *int { return &v }

func gb(n int64) int64 { return n * 1024 * 1024 * 1024 }

func TestOverview_ComputesPerGPUContention(t *testing.T) {
	app := &fakeCapacityApp{
		listOut: capacityapp.ListOutput{Claims: []capacityapp.ClaimView{
			{ClaimID: "a", OwnerID: "whisper", ResourceKind: engine.ResourceKindVRAM, GPUIndex: intPtr(0), AmountBytes: gb(7)},
			{ClaimID: "b", OwnerID: "reranker", ResourceKind: engine.ResourceKindVRAM, GPUIndex: intPtr(0), AmountBytes: gb(1)},
			// A claim with no GPU index must not be attributed to any GPU.
			{ClaimID: "c", OwnerID: "ram-user", ResourceKind: engine.ResourceKindRAM, AmountBytes: gb(4)},
		}},
	}
	collect := func(context.Context) (hostinventory.Snapshot, error) {
		return hostinventory.Snapshot{
			GPUs: []hostinventory.GPU{{
				Index:                    0,
				Name:                     "NVIDIA RTX",
				VRAMBytes:                uint64(gb(16)),
				VRAMUsedBytes:            uint64(gb(13)),
				MemoryUtilizationPercent: 81.0,
			}},
			RuntimeTools: map[string]hostinventory.Tool{"nvidia-smi": {Present: true}},
		}, nil
	}

	svc := NewCapacityServiceWith(app, collect)
	out, err := svc.Overview(context.Background())
	if err != nil {
		t.Fatalf("Overview: %v", err)
	}
	if !out.SensingAvailable {
		t.Errorf("expected sensing available")
	}
	if len(out.GPUs) != 1 {
		t.Fatalf("expected 1 GPU, got %d", len(out.GPUs))
	}
	g := out.GPUs[0]
	if g.FreeBytes != gb(3) {
		t.Errorf("free bytes = %d, want %d", g.FreeBytes, gb(3))
	}
	if g.ClaimedBytes != gb(8) {
		t.Errorf("claimed bytes = %d, want %d (RAM claim must be excluded)", g.ClaimedBytes, gb(8))
	}
	if len(out.Claims) != 3 {
		t.Errorf("expected 3 claims passed through, got %d", len(out.Claims))
	}
}

func TestOverview_SensingFailureIsWarnedNotFatal(t *testing.T) {
	app := &fakeCapacityApp{listOut: capacityapp.ListOutput{Claims: []capacityapp.ClaimView{{ClaimID: "a"}}}}
	collect := func(context.Context) (hostinventory.Snapshot, error) {
		return hostinventory.Snapshot{}, errors.New("nvidia-smi not found")
	}
	svc := NewCapacityServiceWith(app, collect)
	out, err := svc.Overview(context.Background())
	if err != nil {
		t.Fatalf("Overview should not fail on sensing error: %v", err)
	}
	if out.SensingAvailable {
		t.Errorf("expected sensing unavailable")
	}
	if len(out.Warnings) == 0 {
		t.Errorf("expected a sensing warning, got none")
	}
	if len(out.Claims) != 1 {
		t.Errorf("claim table should still render, got %d claims", len(out.Claims))
	}
}

func TestSetPolicy_PassesThroughKeyAndValue(t *testing.T) {
	app := &fakeCapacityApp{policyOut: capacityapp.PolicyOutput{Entries: []capacityapp.PolicyEntry{{Key: "enforce", Value: "advisory"}}}}
	svc := NewCapacityServiceWith(app, nil)
	entries, err := svc.SetPolicy(context.Background(), "enforce", "advisory")
	if err != nil {
		t.Fatalf("SetPolicy: %v", err)
	}
	if app.setKey != "enforce" || app.setValue != "advisory" {
		t.Errorf("key/value not forwarded: got %q=%q", app.setKey, app.setValue)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 lever, got %d", len(entries))
	}
}

func TestReconcile_PassesThroughFindings(t *testing.T) {
	app := &fakeCapacityApp{reconcileOut: capacityapp.ReconcileOutput{Findings: []engine.Finding{
		{Class: engine.FindingUnclaimed, OwnerID: "whisper", Severity: "warn"},
	}}}
	svc := NewCapacityServiceWith(app, nil)
	findings, err := svc.Reconcile(context.Background())
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if len(findings) != 1 || findings[0].Class != engine.FindingUnclaimed {
		t.Errorf("unexpected findings: %+v", findings)
	}
}
