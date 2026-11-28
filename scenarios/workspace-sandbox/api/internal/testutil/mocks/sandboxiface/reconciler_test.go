package sandboxiface

import (
	"context"
	"testing"

	"workspace-sandbox/internal/sandbox"
)

func TestFakeReconciler_DefaultRunReportsZero(t *testing.T) {
	r := NewFakeReconciler("alpha")
	if r.Name() != "alpha" {
		t.Errorf("Name = %q, want alpha", r.Name())
	}
	rep := r.Run(context.Background())
	if rep.ItemsProcessed != 0 || rep.ItemsFailed != 0 {
		t.Errorf("zero report expected, got %+v", rep)
	}
	if r.CallCount() != 1 {
		t.Errorf("CallCount = %d, want 1", r.CallCount())
	}
}

func TestFakeReconciler_RunFnOverride(t *testing.T) {
	r := NewFakeReconciler("beta")
	r.RunFn = func(ctx context.Context) sandbox.ReconcileReport {
		return sandbox.ReconcileReport{ItemsProcessed: 7}
	}
	rep := r.Run(context.Background())
	if rep.ItemsProcessed != 7 {
		t.Errorf("RunFn should override; got ItemsProcessed=%d", rep.ItemsProcessed)
	}
}

func TestFakeReconciler_ErrTextSurfacedInReport(t *testing.T) {
	r := NewFakeReconciler("gamma")
	r.ErrText = "boom"
	rep := r.Run(context.Background())
	if rep.LastError != "boom" {
		t.Errorf("LastError = %q, want boom", rep.LastError)
	}
}
