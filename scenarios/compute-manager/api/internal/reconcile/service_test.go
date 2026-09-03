package reconcile

import (
	"context"
	"testing"

	"compute-manager/internal/provider"
)

func TestSweepReportsBothDirectionsWithoutDestroying(t *testing.T) {
	fake := &provider.Fake{}
	created, err := fake.Create(context.Background(), provider.Spec{Region: "fsn1", Size: "small"})
	if err != nil {
		t.Fatal(err)
	}
	findings, err := (Service{Provider: fake}).Sweep(context.Background(), []Local{{ProviderID: "missing", State: "running"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 2 {
		t.Fatalf("findings = %#v, want provider-only and local-only", findings)
	}
	if _, err := fake.Describe(context.Background(), created.ID); err != nil {
		t.Fatalf("sweep destroyed provider instance: %v", err)
	}
	if fake.DestroyCalls != 0 {
		t.Fatalf("destroy calls = %d, want 0", fake.DestroyCalls)
	}
}
