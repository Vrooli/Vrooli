package operatorstate_test

import (
	"context"
	"testing"
)

// [REQ:ONB-STATE-PRESERVES-UNKNOWN]
func TestPreserveEvidenceRetainsFieldsOwnedByOtherWriters(t *testing.T) {
	service := newService(t)
	ctx := context.Background()
	first, err := service.Apply(ctx, []byte(`{"trust_posture":"shared","core":{"seed":["alpha"],"trusted_base":["alpha"]},"future_permission":{"enabled":true}}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Apply(ctx, []byte(`{"resources":{"ollama":{"enabled":true}}}`)); err != nil {
		t.Fatal(err)
	}
	current, err := service.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if current.TrustPosture != first.TrustPosture || current.Core == nil || current.Core.TrustedBase[0] != "alpha" {
		t.Fatalf("owned fields were not preserved: %#v", current)
	}
	if string(current.RawFields["future_permission"]) != `{"enabled":true}` {
		t.Fatalf("unknown field was not preserved: %s", current.RawFields["future_permission"])
	}
}
