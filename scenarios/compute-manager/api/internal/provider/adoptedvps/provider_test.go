package adoptedvps

import (
	"context"
	"errors"
	"testing"
	"time"

	"compute-manager/internal/provider"
)

func TestCreateRecordsOnlyExplicitOwnedTargetMetadata(t *testing.T) {
	now := time.Date(2026, 11, 23, 18, 0, 0, 0, time.UTC)
	adapter := &Adapter{Now: func() time.Time { return now }}
	instance, err := adapter.Create(context.Background(), provider.Spec{
		Region: "manual", Size: "declared-a1b2c3", SSHPort: 9922,
		AdoptedTarget: &provider.AdoptedTarget{Host: " vps.example.net ", User: " deploy ", DisplayName: "staging"},
		Tags:          map[string]string{"vrooli-intent-id": "intent-7"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if instance.ID != "adopted-vps:intent-7" || instance.Region != "manual" || instance.Size != "declared-a1b2c3" || instance.Address != "vps.example.net" || instance.SSHUser != "deploy" || instance.SSHPort != 22 || !instance.CreatedAt.Equal(now) {
		t.Fatalf("unexpected adopted target projection: %#v", instance)
	}
	if err := adapter.Destroy(context.Background(), instance.ID); !errors.Is(err, ErrOperatorOwned) {
		t.Fatalf("destroy error = %v, want ErrOperatorOwned", err)
	}
}

func TestCreateRejectsMissingOrUnsafeTargetAndMissingDurableIntent(t *testing.T) {
	adapter := &Adapter{}
	base := provider.Spec{Region: "manual", Size: "declared-a1b2c3", AdoptedTarget: &provider.AdoptedTarget{Host: "vps.example.net", User: "deploy"}}
	cases := []struct {
		name string
		spec provider.Spec
	}{
		{name: "missing intent", spec: base},
		{name: "host injection", spec: withTarget(base, "example.net;rm -rf /", "deploy", "intent-1")},
		{name: "user injection", spec: withTarget(base, "vps.example.net", "deploy;id", "intent-2")},
		{name: "non-manual region", spec: withRegion(base, "nbg1", "intent-3")},
		{name: "undeclared offer", spec: withSize(base, "cx23", "intent-4")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := adapter.Create(context.Background(), tc.spec); err == nil {
				t.Fatal("Create unexpectedly accepted invalid adopted target")
			}
		})
	}
}

func withTarget(spec provider.Spec, host, user, intentID string) provider.Spec {
	spec.AdoptedTarget = &provider.AdoptedTarget{Host: host, User: user}
	spec.Tags = map[string]string{"vrooli-intent-id": intentID}
	return spec
}

func withRegion(spec provider.Spec, region, intentID string) provider.Spec {
	spec.Region = region
	spec.Tags = map[string]string{"vrooli-intent-id": intentID}
	return spec
}

func withSize(spec provider.Spec, size, intentID string) provider.Spec {
	spec.Size = size
	spec.Tags = map[string]string{"vrooli-intent-id": intentID}
	return spec
}
