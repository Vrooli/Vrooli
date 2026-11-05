package providers

import (
	"context"
	"strings"
	"testing"

	"storage-manager/internal/cleanup"
	cleanupfakes "storage-manager/internal/testutil/cleanup"
)

// pruningRunner simulates a tool prune command: on success it empties the cache
// the way `go clean -cache` would, so the provider's reclaimed-bytes delta is
// measured against a filesystem that actually changed.
type pruningRunner struct {
	fsys   *cleanupfakes.FileSystem
	result cleanup.ProcessResult
	calls  []cleanup.ProcessCommand
}

func (r *pruningRunner) Run(_ context.Context, cmd cleanup.ProcessCommand) (cleanup.ProcessResult, error) {
	r.calls = append(r.calls, cmd)
	if r.result.ExitCode == 0 {
		r.fsys.Files = map[string]cleanup.FileInfo{}
	}
	return r.result, nil
}

func goBuildCacheToolPruneConfig(root string) ToolPruneProviderConfig {
	return ToolPruneProviderConfig{
		ID:        "go-build-cache",
		Name:      "Governed go-build-cache",
		Root:      root,
		Command:   cleanup.ProcessCommand{Name: "go", Args: []string{"clean", "-cache"}},
		Tier:      cleanup.SafetyTierConditional,
		Platforms: []string{"linux", "macos", "windows"},
		Proof:     cleanup.RegenerableProof{Derived: true, ToolRecreates: true, ExactRoot: true, NoLease: false},
	}
}

func cacheFS(root string, bytesEach int64, names ...string) *cleanupfakes.FileSystem {
	files := map[string]cleanup.FileInfo{}
	for _, name := range names {
		path := root + "/" + name
		files[path] = cleanup.FileInfo{Path: path, Size: bytesEach, IsDir: false}
	}
	return &cleanupfakes.FileSystem{Root: root, Files: files}
}

func TestToolPruneProviderGoCacheMetadataStaysOperatorGated(t *testing.T) {
	provider, err := NewToolPruneProvider(cacheFS("/fake/go-build", 0), &cleanupfakes.ProcessRunner{}, cleanupfakes.Clock{Time: now}, goBuildCacheToolPruneConfig("/fake/go-build"))
	if err != nil {
		t.Fatalf("NewToolPruneProvider: %v", err)
	}
	meta := provider.Metadata()
	if err := meta.Validate(); err != nil {
		t.Fatalf("metadata invalid: %v", err)
	}
	// The Go build cache must never run unattended: conditional tier, operator
	// approval, and no regenerable proof that would make it autonomous.
	if meta.SafetyTier != cleanup.SafetyTierConditional {
		t.Errorf("tier = %q, want conditional", meta.SafetyTier)
	}
	if meta.DefaultApproval != cleanup.ApprovalModeOperator {
		t.Errorf("approval = %q, want operator", meta.DefaultApproval)
	}
	if meta.DefaultMode != cleanup.ProviderModeDisabled {
		t.Errorf("mode = %q, want disabled", meta.DefaultMode)
	}
	if meta.NoLease || meta.RegenerableProof.NoLease {
		t.Errorf("Go cache must not carry NoLease proof: %#v", meta)
	}
}

func TestToolPruneProviderEstimateAndPreviewMeasureTheCache(t *testing.T) {
	fsys := cacheFS("/fake/go-build", 1000, "aa/out-a", "bb/out-b", "cc/out-c")
	provider, err := NewToolPruneProvider(fsys, &cleanupfakes.ProcessRunner{}, cleanupfakes.Clock{Time: now}, goBuildCacheToolPruneConfig("/fake/go-build"))
	if err != nil {
		t.Fatal(err)
	}
	policy := cleanup.ProviderPolicy{Enabled: true, ApprovalMode: cleanup.ApprovalModeOperator}

	estimate, err := provider.Estimate(context.Background(), cleanup.EstimateRequest{Policy: policy})
	if err != nil {
		t.Fatal(err)
	}
	if estimate.EstimatedBytes != 3000 || estimate.ItemCount != 1 || !estimate.RequiresApproval {
		t.Fatalf("estimate = %#v, want 3000 bytes, one item, approval required", estimate)
	}

	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Policy: policy, Estimate: estimate})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Items) != 1 {
		t.Fatalf("preview items = %d, want 1", len(preview.Items))
	}
	item := preview.Items[0]
	if item.Bytes != 3000 || item.Action != "tool-prune" || item.Path != "/fake/go-build" {
		t.Errorf("preview item = %#v, want whole-cache tool-prune item", item)
	}
	// One indivisible command over a cache larger than the batch cap: the item
	// must be allowed to overshoot or the controller would drop it.
	if !preview.AllowSingleOvershoot {
		t.Error("preview must set AllowSingleOvershoot for the indivisible prune")
	}
	if !strings.Contains(item.Description, "go clean -cache") {
		t.Errorf("preview description = %q, want the prune command named", item.Description)
	}
}

func TestToolPruneProviderApplyRunsCommandAndReportsMeasuredReclaim(t *testing.T) {
	fsys := cacheFS("/fake/go-build", 1000, "aa/out-a", "bb/out-b")
	runner := &pruningRunner{fsys: fsys, result: cleanup.ProcessResult{ExitCode: 0}}
	provider, err := NewToolPruneProvider(fsys, runner, cleanupfakes.Clock{Time: now}, goBuildCacheToolPruneConfig("/fake/go-build"))
	if err != nil {
		t.Fatal(err)
	}
	req := cleanup.ApplyRequest{
		ProviderVersion: "v1",
		IdempotencyKey:  "run-1",
		ApprovalMode:    cleanup.ApprovalModeOperator,
		Preview: cleanup.Preview{ProviderID: "go-build-cache", ProviderVersion: "v1", Items: []cleanup.PreviewItem{
			{ID: "go-build-cache:tool-prune", Path: "/fake/go-build", Bytes: 2000, Action: "tool-prune"},
		}},
	}
	result, err := provider.Apply(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(runner.calls) != 1 || runner.calls[0].Name != "go" || strings.Join(runner.calls[0].Args, " ") != "clean -cache" {
		t.Fatalf("expected one `go clean -cache` invocation, got %#v", runner.calls)
	}
	// Reclaimed bytes come from the measured before/after delta, not the echoed
	// estimate: the whole 2000-byte cache was emptied.
	if !result.Applied || result.ReclaimedBytes != 2000 || len(result.AppliedItems) != 1 {
		t.Fatalf("result = %#v, want applied with 2000 reclaimed", result)
	}
}

func TestToolPruneProviderApplyNonZeroExitReclaimsNothing(t *testing.T) {
	fsys := cacheFS("/fake/go-build", 1000, "aa/out-a")
	runner := &pruningRunner{fsys: fsys, result: cleanup.ProcessResult{ExitCode: 2, Stderr: "go: cannot clean cache"}}
	provider, err := NewToolPruneProvider(fsys, runner, cleanupfakes.Clock{Time: now}, goBuildCacheToolPruneConfig("/fake/go-build"))
	if err != nil {
		t.Fatal(err)
	}
	req := cleanup.ApplyRequest{
		ProviderVersion: "v1",
		IdempotencyKey:  "run-1",
		ApprovalMode:    cleanup.ApprovalModeOperator,
		Preview:         cleanup.Preview{ProviderID: "go-build-cache", ProviderVersion: "v1", Items: []cleanup.PreviewItem{{ID: "go-build-cache:tool-prune", Path: "/fake/go-build", Bytes: 1000}}},
	}
	result, err := provider.Apply(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if result.Applied || result.ReclaimedBytes != 0 || len(result.SkippedItems) != 1 {
		t.Fatalf("result = %#v, want not applied and nothing reclaimed on non-zero exit", result)
	}
	if len(result.Warnings) == 0 || !strings.Contains(strings.Join(result.Warnings, " "), "exited 2") {
		t.Errorf("warnings = %v, want the exit code surfaced", result.Warnings)
	}
	if len(fsys.Files) != 1 {
		t.Errorf("a failed prune must not change the cache; files = %v", fsys.Files)
	}
}

func TestToolPruneProviderDisabledPolicyBlocks(t *testing.T) {
	fsys := cacheFS("/fake/go-build", 1000, "aa/out-a")
	provider, err := NewToolPruneProvider(fsys, &cleanupfakes.ProcessRunner{}, cleanupfakes.Clock{Time: now}, goBuildCacheToolPruneConfig("/fake/go-build"))
	if err != nil {
		t.Fatal(err)
	}
	estimate, err := provider.Estimate(context.Background(), cleanup.EstimateRequest{Policy: cleanup.ProviderPolicy{Enabled: false}})
	if err != nil {
		t.Fatal(err)
	}
	if estimate.BlockedReason == "" || estimate.EstimatedBytes != 0 {
		t.Fatalf("disabled estimate = %#v, want blocked with no bytes", estimate)
	}
}

func TestToolPruneProviderApplyRequiresIdempotencyKeyAndVersion(t *testing.T) {
	fsys := cacheFS("/fake/go-build", 1000, "aa/out-a")
	provider, err := NewToolPruneProvider(fsys, &pruningRunner{fsys: fsys}, cleanupfakes.Clock{Time: now}, goBuildCacheToolPruneConfig("/fake/go-build"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Apply(context.Background(), cleanup.ApplyRequest{ProviderVersion: "v1"}); err == nil {
		t.Error("apply without idempotency key must fail")
	}
	if _, err := provider.Apply(context.Background(), cleanup.ApplyRequest{ProviderVersion: "v2", IdempotencyKey: "k"}); err == nil {
		t.Error("apply with wrong provider version must fail")
	}
}

// A regenerable tool-pruned cache (uv/pnpm style) may run unattended, so the
// provider grants ApprovalModeNone only when the complete regenerable proof is
// present, and refuses construction when it is not.
func TestToolPruneProviderRegenerableRequiresCompleteProof(t *testing.T) {
	root := "/fake/uv"
	valid := ToolPruneProviderConfig{
		ID: "uv-cache", Name: "Governed uv-cache", Root: root,
		Command:   cleanup.ProcessCommand{Name: "uv", Args: []string{"cache", "prune"}},
		Tier:      cleanup.SafetyTierRegenerable,
		Platforms: []string{"linux", "macos"},
		Proof:     cleanup.RegenerableProof{Derived: true, ToolRecreates: true, ExactRoot: true, NoLease: true},
	}
	provider, err := NewToolPruneProvider(cacheFS(root, 0), &cleanupfakes.ProcessRunner{}, cleanupfakes.Clock{}, valid)
	if err != nil {
		t.Fatalf("valid regenerable config rejected: %v", err)
	}
	if meta := provider.Metadata(); meta.DefaultApproval != cleanup.ApprovalModeNone || !meta.NoLease {
		t.Errorf("regenerable meta = %#v, want autonomous approval and NoLease proof", meta)
	}

	incomplete := valid
	incomplete.Proof.NoLease = false
	if _, err := NewToolPruneProvider(cacheFS(root, 0), &cleanupfakes.ProcessRunner{}, cleanupfakes.Clock{}, incomplete); err == nil {
		t.Error("regenerable config without complete proof must fail construction")
	}
}

// The prune clears the whole cache, so pressure recovery runs it only when the
// cache exceeds its declared ceiling; an operator plan is never gated.
func TestToolPruneProviderRecoveryRunsOnlyAboveDeclaredCeiling(t *testing.T) {
	for _, tc := range []struct {
		name     string
		ceiling  int64
		recovery bool
		want     int64
	}{
		{"recovery under ceiling", 5000, true, 0},
		{"recovery at ceiling", 3000, true, 0},
		{"recovery over ceiling", 2000, true, 3000},
		{"plan under ceiling", 5000, false, 3000},
		{"recovery without ceiling", 0, true, 3000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := goBuildCacheToolPruneConfig("/fake/go-build")
			cfg.MaxBytes = tc.ceiling
			provider, err := NewToolPruneProvider(cacheFS("/fake/go-build", 1000, "aa/a", "bb/b", "cc/c"), &cleanupfakes.ProcessRunner{}, cleanupfakes.Clock{Time: now}, cfg)
			if err != nil {
				t.Fatal(err)
			}
			scope := cleanup.ObservationScope{Now: now, Recovery: tc.recovery}
			policy := cleanup.ProviderPolicy{Enabled: true, ApprovalMode: cleanup.ApprovalModeNone}
			estimate, err := provider.Estimate(context.Background(), cleanup.EstimateRequest{Scope: scope, Policy: policy})
			if err != nil {
				t.Fatal(err)
			}
			preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Scope: scope, Policy: policy, Estimate: estimate})
			if err != nil {
				t.Fatal(err)
			}
			if estimate.EstimatedBytes != tc.want || sumPreviewBytes(preview.Items) != tc.want {
				t.Fatalf("estimate=%d preview=%d, want %d", estimate.EstimatedBytes, sumPreviewBytes(preview.Items), tc.want)
			}
			if tc.want > 0 && !preview.AllowSingleOvershoot {
				t.Fatal("indivisible prune lost its single-overshoot authority")
			}
		})
	}
}
