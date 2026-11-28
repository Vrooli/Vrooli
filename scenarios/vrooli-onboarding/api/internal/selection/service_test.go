package selection

import (
	"context"
	"errors"
	"testing"

	selectionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/selection"
)

func TestMissingSelectionDependenciesKeepCancellationSentinel(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		call func(Service) error
	}{
		{"scenarios", func(s Service) error { _, err := s.Scenarios(ctx); return err }},
		{"core set", func(s Service) error { _, err := s.CoreSet(ctx, nil); return err }},
		{"recommendation", func(s Service) error { _, err := s.Recommendation(ctx); return err }},
		{"accept", func(s Service) error { _, err := s.Accept(ctx, nil); return err }},
		{"closure", func(s Service) error { _, err := s.Closure(ctx); return err }},
		{"union", func(s Service) error { _, err := s.Union(ctx); return err }},
		{"handoff", func(s Service) error { _, err := s.Handoff(ctx, nil); return err }},
		{"resolve handoff", func(s Service) error { _, err := s.ResolveHandoff(ctx, nil); return err }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.call(Service{}); !errors.Is(err, context.Canceled) {
				t.Fatalf("missing dependency error = %v, want context.Canceled", err)
			}
		})
	}
}

func TestCoreSetForwardsSeedToOwner(t *testing.T) {
	ctx := context.Background()
	seed := []string{"control-plane", "writer"}
	called := false
	service := Service{GetCoreSet: func(gotCtx context.Context, gotSeed []string) (*selectionv1.GetCoreSetResponse, error) {
		called = true
		if gotCtx != ctx || len(gotSeed) != len(seed) || gotSeed[0] != seed[0] || gotSeed[1] != seed[1] {
			t.Fatalf("owner arguments = (%v, %v), want (%v, %v)", gotCtx, gotSeed, ctx, seed)
		}
		return &selectionv1.GetCoreSetResponse{}, nil
	}}
	if _, err := service.CoreSet(ctx, seed); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("core-set owner was not called")
	}
}
