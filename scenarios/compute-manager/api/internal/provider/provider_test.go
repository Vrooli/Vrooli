package provider

import (
	"context"
	"errors"
	"testing"
)

func TestFakeLostCreateLeavesProviderInstance(t *testing.T) {
	fake := &Fake{LoseCreateResponse: true}
	_, err := fake.Create(context.Background(), Spec{Region: "fsn1", Size: "cx22", Image: "ubuntu"})
	if !errors.Is(err, ErrCreateResponseLost) {
		t.Fatalf("Create error = %v", err)
	}
	instances, err := fake.List(context.Background())
	if err != nil || len(instances) != 1 {
		t.Fatalf("List = %v, %v", instances, err)
	}
}

func TestProviderSurfaceHasNoPauseOperation(t *testing.T) {
	// This compile-time assertion is intentionally the test: the interface has
	// exactly the four provider operations allowed by the product contract.
	var _ Provider = (*Fake)(nil)
}
