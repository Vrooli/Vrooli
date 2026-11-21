package pstoreobservability

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/testenv"
)

func TestApplyDryRunPreservesPstoreIdentity(t *testing.T) {
	testenv.SetIdentityEnv(t, map[string]string{"USER": "operator"})
	h := NewHandler(hostreqkit.SafeguardManifest{Name: "pstore_observability"})
	status := hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportSupported}
	got, err := h.Apply(hostreqkit.Host{}, status, hostreqkit.EnsureOptions{DryRun: true})
	if err != nil || got.ExecutionState != hostreqkit.ExecutionWouldApply {
		t.Fatalf("Apply dry-run = %#v, %v", got, err)
	}
	if len(got.Notes) != 1 || got.Notes[0] == "" {
		t.Fatalf("Apply notes = %v, want pstore-specific dry-run description", got.Notes)
	}
}

func TestInspectUnsupportedPlatform(t *testing.T) {
	h := NewHandler(hostreqkit.SafeguardManifest{Name: "pstore_observability"})
	req := hostreqspec.ResolvedRequirement{Name: "pstore_observability", Kind: hostreqspec.KindSafeguard}
	got := h.Inspect(hostreqkit.Host{OS: "windows"}, req)
	if got.SupportClass != hostreqkit.SupportUnsupported || got.ExecutionState != hostreqkit.ExecutionUnsupported {
		t.Fatalf("Inspect unsupported = %#v", got)
	}
}
