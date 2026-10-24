package providers

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"storage-manager/hostfs"
	"storage-manager/internal/cleanup"
	cleanupfakes "storage-manager/internal/testutil/cleanup"
)

func TestGoBuildCacheDeclarationRequiresOwnerProof(t *testing.T) {
	specs, err := LoadRootSpecs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, spec := range specs {
		if spec.ID != "go-build-cache" {
			continue
		}
		found++
		if err := ValidateRootSpec(spec); err != nil {
			t.Error(err)
		}
		if spec.Tier != cleanup.SafetyTierConditional || spec.LeaseCheck != "owner" || spec.Proof.NoLease {
			t.Errorf("Go cache must require owner build-use proof, got %#v", spec)
		}
		if len(spec.ToolPruneCommand) != 0 {
			t.Errorf("Go cache must not advertise an uncoordinated prune command: %v", spec.ToolPruneCommand)
		}
	}
	if found != 1 {
		t.Fatalf("Go cache declarations = %d, want exactly one", found)
	}
}

func TestGoBuildCacheRefusesReclaimWithoutBuildUseProof(t *testing.T) {
	for _, id := range []string{"go-build-cache", "spec-go-build-cache"} {
		for _, declaration := range []string{"legacy-regenerable", "owner-required"} {
			t.Run(id+"/"+declaration, func(t *testing.T) {
				root := t.TempDir()
				path := filepath.Join(root, "02", "output-d")
				fresh := filepath.Join(root, "0e", "fresh-d")
				// No open handle or lock marker is required for an output to be
				// needed by a later link/vet step. Even an old output is not proof
				// of quiescence; the fake permits removal so safety is the oracle.
				fsys := &cleanupfakes.FileSystem{Root: root, AllowRemove: true, Files: map[string]cleanup.FileInfo{
					path: file(path, 100, now.Add(-30*24*time.Hour)), fresh: file(fresh, 100, now),
				}}
				spec := RootSpec{
					ID: id, Root: root, Class: "cache", Tier: cleanup.SafetyTierRegenerable,
					MaxAge: "14d", MaxBytes: "100B", LeaseCheck: "none", Platforms: []string{"linux", "macos", "windows"},
					Proof: RootSpecProof{Derived: true, ToolRecreates: true, ExactRoot: true, NoLease: true}, Rationale: "test Go cache",
				}
				if declaration == "owner-required" {
					spec.Tier, spec.LeaseCheck, spec.Proof.NoLease = cleanup.SafetyTierConditional, "owner", false
				}
				provider, err := NewSpecProvider(fsys, cleanupfakes.Clock{Time: now}, spec, FileProviderConfig{})
				if err != nil {
					t.Fatal(err)
				}
				meta := provider.Metadata()
				if err := meta.Validate(); err != nil {
					t.Error(err)
				}
				if meta.SafetyTier != cleanup.SafetyTierConditional || meta.NoLease || meta.RegenerableProof.NoLease || meta.DefaultApproval != cleanup.ApprovalModeOperator {
					t.Errorf("Go cache metadata still authorizes autonomous recovery: %#v", meta)
				}
				for _, recovery := range []bool{false, true} {
					scope := cleanup.ObservationScope{Now: now, RootPaths: []string{root}, Recovery: recovery, CompleteCensus: true}
					policy := cleanup.ProviderPolicy{Enabled: true, ApprovalMode: cleanup.ApprovalModeNone, AllowFreshReclaim: true}
					estimate, err := provider.Estimate(context.Background(), cleanup.EstimateRequest{Scope: scope, Policy: policy})
					if err != nil {
						t.Fatal(err)
					}
					if estimate.EstimatedBytes != 0 || estimate.ItemCount != 0 || !strings.Contains(estimate.BlockedReason, "build-use") {
						t.Errorf("recovery=%v estimate must explain missing build-use proof: %#v", recovery, estimate)
					}
					preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Scope: scope, Policy: policy})
					if err != nil {
						t.Fatal(err)
					}
					if len(preview.Items) != 0 || !strings.Contains(preview.BlockedReason, "build-use") {
						t.Errorf("recovery=%v preview must refuse even fresh-reclaim override: %#v", recovery, preview)
					}
				}
			})
		}
	}
}

func TestGoBuildCacheRefusesPreviouslyApprovedPreview(t *testing.T) {
	for _, id := range []string{"go-build-cache", "spec-go-build-cache"} {
		for _, approval := range []cleanup.ApprovalMode{cleanup.ApprovalModeNone, cleanup.ApprovalModeOperator, cleanup.ApprovalModeOwner} {
			t.Run(id+"/"+string(approval), func(t *testing.T) {
				root := t.TempDir()
				shard, path := filepath.Join(root, "02"), filepath.Join(root, "0e", "output-d")
				fsys := &cleanupfakes.FileSystem{Root: root, AllowRemove: true, Files: map[string]cleanup.FileInfo{
					shard: dir(shard, now), path: file(path, 100, now),
				}}
				provider := NewCacheProvider(fsys, cleanupfakes.Clock{Time: now}, FileProviderConfig{
					ID: id, Roots: []string{root}, Tier: cleanup.SafetyTierRegenerable, RetentionMaxBytes: 1,
				})
				// A persisted v1 plan may select either whole shards (recovery)
				// or individual outputs (ordinary cleanup). Never trust that plan
				// or its operator/standing approval as live build-use evidence.
				req := cleanup.ApplyRequest{ProviderVersion: "v1", IdempotencyKey: "old-plan", ApprovalMode: approval,
					Preview: cleanup.Preview{ProviderID: id, ProviderVersion: "v1", Items: []cleanup.PreviewItem{
						{ID: "shard", Path: shard, Bytes: 4096}, {ID: "output", Path: path, Bytes: 100},
					}},
				}
				for attempt := 0; attempt < 2; attempt++ {
					result, err := provider.Apply(context.Background(), req)
					if err != nil {
						t.Fatal(err)
					}
					if result.Applied || result.ReclaimedBytes != 0 || len(result.AppliedItems) != 0 || len(result.SkippedItems) != 2 || !strings.Contains(strings.Join(result.Warnings, " "), "build-use") {
						t.Errorf("attempt=%d old plan must be refused explicitly: %#v", attempt, result)
					}
					if len(fsys.Removed) != 0 {
						t.Fatalf("Go cache removal reached filesystem: %v", fsys.Removed)
					}
				}
			})
		}
	}
}

func TestValidateRootSpecRequiresRegenerableProof(t *testing.T) {
	spec := RootSpec{ID: "cache", Root: "/tmp/cache", Class: "cache", Tier: cleanup.SafetyTierRegenerable, Platforms: []string{"linux"}, Rationale: "derived cache"}
	if err := ValidateRootSpec(spec); err == nil {
		t.Fatal("ValidateRootSpec accepted regenerable root without proof")
	}
	spec.Proof = RootSpecProof{Derived: true, ToolRecreates: true, ExactRoot: true, NoLease: true}
	if err := ValidateRootSpec(spec); err != nil {
		t.Fatalf("ValidateRootSpec rejected complete proof: %v", err)
	}
}

func TestNewSpecProviderUsesDeclaredRootAndLimits(t *testing.T) {
	spec := RootSpec{
		ID: "spec-cache", Root: "/tmp/spec-cache", Class: "cache", Tier: cleanup.SafetyTierRegenerable,
		MaxAge: "30d", MaxBytes: "8GiB", Platforms: []string{"linux", "macos", "windows"},
		Proof: RootSpecProof{Derived: true, ToolRecreates: true, ExactRoot: true, NoLease: true}, Rationale: "test cache",
	}
	provider, err := NewSpecProvider(hostfs.New(hostfs.Options{AllowForeignOwnership: true}), cleanupfakes.Clock{}, spec, FileProviderConfig{})
	if err != nil {
		t.Fatalf("NewSpecProvider: %v", err)
	}
	if provider.Metadata().ID != spec.ID || provider.retentionMaxAge <= 0 || provider.retentionMaxBytes != 8*1024*1024*1024 {
		t.Fatalf("provider = %#v, want declared identity and limits", provider)
	}
	if runtime.GOOS == "linux" && len(provider.roots) != 1 {
		t.Fatalf("provider roots = %v, want declared root", provider.roots)
	}
}

func TestNewSpecProviderExpandsOnlyLastSegmentGlob(t *testing.T) {
	base := t.TempDir()
	if err := os.Mkdir(filepath.Join(base, "go-build123"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(base, "go-builder-notes"), 0o755); err != nil {
		t.Fatal(err)
	}
	spec := RootSpec{ID: "work", Root: filepath.Join(base, "go-build*"), Class: "temp", Tier: cleanup.SafetyTierSafe, Platforms: []string{"linux", "macos", "windows"}, Rationale: "test work dirs"}
	provider, err := NewSpecProvider(hostfs.New(hostfs.Options{AllowForeignOwnership: true}), cleanupfakes.Clock{}, spec, FileProviderConfig{})
	if err != nil {
		t.Fatalf("NewSpecProvider: %v", err)
	}
	if len(provider.roots) != 1 || filepath.Base(provider.roots[0]) != "go-build123" {
		t.Fatalf("expanded roots = %v, want only go-build123", provider.roots)
	}
}

func TestSpecProviderLeaseFileProtectsBuildingBundle(t *testing.T) {
	root := t.TempDir()
	bundle := filepath.Join(root, "bundle")
	if err := os.MkdirAll(bundle, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundle, "payload"), []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundle, ".building"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	spec := RootSpec{ID: "staging", Root: root, Class: "cache", Tier: cleanup.SafetyTierRegenerable, MaxAge: "1h", LeaseCheck: "lease_file", Platforms: []string{"linux", "macos", "windows"}, Proof: RootSpecProof{Derived: true, ToolRecreates: true, ExactRoot: true, NoLease: true}, Rationale: "staging"}
	provider, err := NewSpecProvider(&cleanupfakes.FileSystem{}, cleanupfakes.Clock{}, spec, FileProviderConfig{})
	if err != nil {
		t.Fatal(err)
	}
	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Scope: cleanup.ObservationScope{Now: time.Now()}, Policy: cleanup.ProviderPolicy{Enabled: true, MinAge: time.Hour}})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Items) != 0 {
		t.Fatalf("preview = %#v, want lease-protected bundle omitted", preview.Items)
	}
}

func TestSpecProviderOpenHandleProtectsHeldFile(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("open-handle enumeration is only implemented on Linux")
	}
	root := t.TempDir()
	path := filepath.Join(root, "payload")
	if err := os.WriteFile(path, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	held, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()

	spec := RootSpec{ID: "held-cache", Root: root, Class: "cache", Tier: cleanup.SafetyTierRegenerable, LeaseCheck: "open_handle", Platforms: []string{"linux"}, Proof: RootSpecProof{Derived: true, ToolRecreates: true, ExactRoot: true, NoLease: true}, Rationale: "held test cache"}
	provider, err := NewSpecProvider(hostfs.New(hostfs.Options{AllowForeignOwnership: true}), cleanupfakes.Clock{}, spec, FileProviderConfig{})
	if err != nil {
		t.Fatal(err)
	}
	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Scope: cleanup.ObservationScope{Now: time.Now()}, Policy: cleanup.ProviderPolicy{Enabled: true, MinAge: time.Hour}})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Items) != 0 {
		t.Fatalf("preview = %#v, want held file omitted", preview.Items)
	}
}
