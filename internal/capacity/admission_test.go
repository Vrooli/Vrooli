package capacity

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

func writeResourceManifest(t *testing.T, root, name, body string) {
	t.Helper()
	dir := filepath.Join(root, "resources", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "resource.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func admitTestStore(t *testing.T) (func(context.Context) (AdmitStore, error), func() time.Time) {
	dbPath := filepath.Join(t.TempDir(), "capacity.db")
	clk := func() time.Time { return time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC) }
	open := func(ctx context.Context) (AdmitStore, error) {
		return NewSQLiteStore(ctx, Config{DBPath: dbPath, Clock: clockAdapter(clk)})
	}
	return open, clk
}

func TestAdmitResourceRecordsClaim(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writeResourceManifest(t, root, "whisper", `{"name":"whisper","capacity":{"resource_kind":"vram","preferred_bytes":7516192768,"floor_bytes":1073741824,"priority":"service"}}`)
	open, clk := admitTestStore(t)

	res, err := AdmitResource(ctx, AdmitOptions{
		Root: root, ResourceName: "whisper",
		Source:    StaticSource{Inventory: snapshotWith(16, 4)},
		OpenStore: open, Clock: clk, EnforceEnv: EnforceAdvisory,
	})
	if err != nil {
		t.Fatalf("AdmitResource() error = %v", err)
	}
	if res.Skipped {
		t.Fatalf("expected a recorded claim, got skipped: %s", res.Reason)
	}
	if res.Verdict.Kind != VerdictGrant || res.ClaimID == "" {
		t.Errorf("result = %+v, want grant + claim id", res)
	}

	store, _ := open(ctx)
	defer store.Close()
	claims, _ := store.ListClaims(ctx, ClaimFilter{OwnerID: "whisper"})
	if len(claims) != 1 || claims[0].Priority != PriorityService {
		t.Fatalf("ledger = %+v, want one service claim", claims)
	}
}

func TestAdmitResourceFlagOffIsByteIdenticalNoop(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	// Declares a profile, but enforcement is OFF -> no claim recorded at all.
	writeResourceManifest(t, root, "whisper", `{"capacity":{"resource_kind":"vram","preferred_bytes":1,"floor_bytes":1,"priority":"service"}}`)
	open, clk := admitTestStore(t)

	res, err := AdmitResource(ctx, AdmitOptions{
		Root: root, ResourceName: "whisper",
		Source:    StaticSource{Inventory: snapshotWith(16, 0)},
		OpenStore: open, Clock: clk, EnforceEnv: EnforceOff,
	})
	if err != nil {
		t.Fatalf("AdmitResource() error = %v", err)
	}
	if !res.Skipped {
		t.Fatalf("flag-off admission must skip; got %+v", res)
	}
	store, _ := open(ctx)
	defer store.Close()
	claims, _ := store.ListClaims(ctx, ClaimFilter{})
	if len(claims) != 0 {
		t.Errorf("flag-off must record no claim; ledger = %+v", claims)
	}
}

func TestAdmitResourceNoProfileSkips(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writeResourceManifest(t, root, "plain", `{"name":"plain"}`) // no capacity block
	open, clk := admitTestStore(t)

	res, err := AdmitResource(ctx, AdmitOptions{
		Root: root, ResourceName: "plain",
		Source:    StaticSource{Inventory: snapshotWith(16, 0)},
		OpenStore: open, Clock: clk, EnforceEnv: EnforceAdvisory,
	})
	if err != nil {
		t.Fatalf("AdmitResource() error = %v", err)
	}
	if !res.Skipped {
		t.Errorf("no-profile admission must skip; got %+v", res)
	}
}

func TestAdmitResourceMissingManifestSkips(t *testing.T) {
	ctx := context.Background()
	open, clk := admitTestStore(t)
	res, err := AdmitResource(ctx, AdmitOptions{
		Root: t.TempDir(), ResourceName: "ghost",
		Source: StaticSource{Inventory: hostinventory.Snapshot{}}, OpenStore: open, Clock: clk, EnforceEnv: EnforceAdvisory,
	})
	if err != nil {
		t.Fatalf("AdmitResource() error = %v", err)
	}
	if !res.Skipped {
		t.Errorf("missing manifest must skip cleanly; got %+v", res)
	}
}

func TestLoadResourceClaimSpec(t *testing.T) {
	root := t.TempDir()
	writeResourceManifest(t, root, "img", `{"capacity":{"resource_kind":"vram","preferred_bytes":100,"floor_bytes":0,"priority":"batch","profile":{"steps":[{"label":"fp16","amount_bytes":100},{"label":"cpu","amount_bytes":0}],"upshift":true}}}`)
	spec, ok, err := LoadResourceClaimSpec(root, "img")
	if err != nil || !ok {
		t.Fatalf("LoadResourceClaimSpec() = ok %v err %v", ok, err)
	}
	if spec.Profile == nil || len(spec.Profile.Steps) != 2 || spec.Profile.Steps[1].Label != "cpu" {
		t.Errorf("spec profile = %+v", spec.Profile)
	}
}
