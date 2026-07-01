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

func TestAdmitResourceIdempotentReuse(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writeResourceManifest(t, root, "whisper", `{"capacity":{"resource_kind":"vram","preferred_bytes":7516192768,"floor_bytes":1073741824,"priority":"service"}}`)
	open, clk := admitTestStore(t)
	opts := AdmitOptions{
		Root: root, ResourceName: "whisper",
		Source:    StaticSource{Inventory: snapshotWith(16, 0)},
		OpenStore: open, Clock: clk, EnforceEnv: EnforceAdvisory,
	}

	first, err := AdmitResource(ctx, opts)
	if err != nil || first.ClaimID == "" {
		t.Fatalf("first AdmitResource() = %+v, err %v", first, err)
	}
	// Re-admitting the same resident (restart) must reuse the existing active
	// claim, not stack a second one.
	second, err := AdmitResource(ctx, opts)
	if err != nil {
		t.Fatalf("second AdmitResource() error = %v", err)
	}
	if second.ClaimID != first.ClaimID {
		t.Errorf("second claim id = %q, want reuse of %q", second.ClaimID, first.ClaimID)
	}

	store, _ := open(ctx)
	defer store.Close()
	claims, _ := store.ListClaims(ctx, ClaimFilter{OwnerID: "whisper", Statuses: ActiveClaimStatuses()})
	if len(claims) != 1 {
		t.Fatalf("ledger has %d active whisper claims, want exactly 1 (idempotent)", len(claims))
	}
}

func TestAdmitResourceReplacesStaleManifestClaim(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writeResourceManifest(t, root, "whisper", `{"capacity":{"resource_kind":"vram","preferred_bytes":8589934592,"floor_bytes":2147483648,"priority":"interactive","yield_when_idle":true,"idle_grace_seconds":60}}`)
	open, clk := admitTestStore(t)
	opts := AdmitOptions{
		Root: root, ResourceName: "whisper",
		Source:    StaticSource{Inventory: snapshotWith(16, 0)},
		OpenStore: open, Clock: clk, EnforceEnv: EnforceAdvisory,
	}

	first, err := AdmitResource(ctx, opts)
	if err != nil || first.ClaimID == "" {
		t.Fatalf("first AdmitResource() = %+v, err %v", first, err)
	}
	writeResourceManifest(t, root, "whisper", `{"capacity":{"resource_kind":"vram","preferred_bytes":5368709120,"floor_bytes":2147483648,"priority":"interactive","yield_when_idle":true,"idle_grace_seconds":900}}`)
	second, err := AdmitResource(ctx, opts)
	if err != nil {
		t.Fatalf("second AdmitResource() error = %v", err)
	}
	if second.ClaimID == "" || second.ClaimID == first.ClaimID {
		t.Fatalf("second claim id = %q, want replacement of stale %q", second.ClaimID, first.ClaimID)
	}

	store, _ := open(ctx)
	defer store.Close()
	active, _ := store.ListClaims(ctx, ClaimFilter{OwnerID: "whisper", Statuses: ActiveClaimStatuses()})
	if len(active) != 1 {
		t.Fatalf("active whisper claims = %d, want exactly 1", len(active))
	}
	if active[0].PreferredBytes != 5368709120 || active[0].IdleGrace != 15*time.Minute {
		t.Fatalf("active claim = %+v, want refreshed manifest shape", active[0])
	}
	old, err := store.GetClaim(ctx, first.ClaimID)
	if err != nil {
		t.Fatalf("old claim lookup: %v", err)
	}
	if old.Status != StatusReleased {
		t.Fatalf("old claim status = %q, want released", old.Status)
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
	writeResourceManifest(t, root, "img", `{"capacity":{"resource_kind":"vram","preferred_bytes":100,"floor_bytes":0,"priority":"batch","idle_grace_seconds":900,"profile":{"steps":[{"label":"fp16","amount_bytes":100},{"label":"cpu","amount_bytes":0}],"upshift":true}}}`)
	spec, ok, err := LoadResourceClaimSpec(root, "img")
	if err != nil || !ok {
		t.Fatalf("LoadResourceClaimSpec() = ok %v err %v", ok, err)
	}
	if spec.IdleGraceSeconds != 900 {
		t.Fatalf("idle_grace_seconds = %d, want 900", spec.IdleGraceSeconds)
	}
	if spec.Profile == nil || len(spec.Profile.Steps) != 2 || spec.Profile.Steps[1].Label != "cpu" {
		t.Errorf("spec profile = %+v", spec.Profile)
	}
}

func TestRealWhisperCapacitySpecIsRightSized(t *testing.T) {
	spec, ok, err := LoadResourceClaimSpec(findAdmissionRepoRoot(t), "whisper")
	if err != nil || !ok {
		t.Fatalf("LoadResourceClaimSpec(whisper) = ok %v err %v", ok, err)
	}
	if spec.PreferredBytes != 5*gib {
		t.Fatalf("whisper preferred_bytes = %d, want 5GiB", spec.PreferredBytes)
	}
	if spec.FloorBytes != 2*gib {
		t.Fatalf("whisper floor_bytes = %d, want 2GiB", spec.FloorBytes)
	}
	if spec.Priority != "interactive" || !spec.YieldWhenIdle {
		t.Fatalf("whisper priority/yield = %q/%v, want interactive/true", spec.Priority, spec.YieldWhenIdle)
	}
	if spec.IdleGraceSeconds != 900 {
		t.Fatalf("whisper idle_grace_seconds = %d, want 900", spec.IdleGraceSeconds)
	}
	if spec.Profile == nil || len(spec.Profile.Steps) != 3 {
		t.Fatalf("whisper profile = %+v, want 3-step degradation ladder", spec.Profile)
	}
	if got := spec.Profile.Steps[0]; got.Label != "large-v3" || got.AmountBytes != 5*gib {
		t.Fatalf("whisper top step = %+v, want large-v3/5GiB", got)
	}
}

func TestRealKyutaiCapacitySpecYieldsWhenIdle(t *testing.T) {
	spec, ok, err := LoadResourceClaimSpec(findAdmissionRepoRoot(t), "kyutai-stt")
	if err != nil || !ok {
		t.Fatalf("LoadResourceClaimSpec(kyutai-stt) = ok %v err %v", ok, err)
	}
	if spec.Protected {
		t.Fatal("kyutai-stt must not be statically protected; activity state owns active protection")
	}
	if !spec.YieldWhenIdle {
		t.Fatal("kyutai-stt must yield when idle")
	}
	if spec.IdleUnloadTTLSeconds != 900 {
		t.Fatalf("kyutai-stt idle_unload_ttl_seconds = %d, want 900", spec.IdleUnloadTTLSeconds)
	}
	if spec.IdleGraceSeconds != 900 {
		t.Fatalf("kyutai-stt idle_grace_seconds = %d, want 900", spec.IdleGraceSeconds)
	}
	if spec.Profile == nil || len(spec.Profile.Steps) != 2 {
		t.Fatalf("kyutai-stt profile = %+v, want loaded/unloaded ladder", spec.Profile)
	}
	if got := spec.Profile.Steps[1]; got.Label != "unloaded" || got.AmountBytes != 0 {
		t.Fatalf("kyutai-stt floor step = %+v, want unloaded/0", got)
	}
	if spec.Profile.Apply.Verb != "stop" {
		t.Fatalf("kyutai-stt apply verb = %q, want standard lifecycle stop", spec.Profile.Apply.Verb)
	}
	if spec.Profile.Upshift {
		t.Fatal("kyutai-stt profile should not auto-upshift through stop; next STT request owns restart")
	}
}

func findAdmissionRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("could not locate repo root")
	return ""
}
