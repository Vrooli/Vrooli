package operatorstate_test

import (
	"context"
	"testing"
)

// [REQ:ONB-STATE-SINGLE-WRITER]
func TestSingleWriterEvidenceUsesTheSharedServiceBoundary(t *testing.T) {
	service := newService(t)
	if _, err := service.Apply(context.Background(), []byte(`{"scenarios":{"alpha":{"enabled":true}}}`)); err != nil {
		t.Fatal(err)
	}
	loaded, err := service.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Scenarios["alpha"].Enabled == nil || !*loaded.Scenarios["alpha"].Enabled {
		t.Fatal("shared writer did not persist the operator choice")
	}
}
