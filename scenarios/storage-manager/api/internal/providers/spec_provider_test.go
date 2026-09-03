package providers

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"storage-manager/hostfs"
	"storage-manager/internal/cleanup"
	cleanupfakes "storage-manager/internal/testutil/cleanup"
)

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

func TestResolveRootExpandsSupportedVariables(t *testing.T) {
	home := filepath.Join("/tmp", "root-spec-home")
	if got, want := ResolveRoot("$USER_HOME/cache", home), filepath.Join(home, "cache"); got != want {
		t.Fatalf("ResolveRoot home = %q, want %q", got, want)
	}
	if got, want := ResolveRoot("$TMPDIR/go-build*", home), filepath.Join("/tmp", "go-build*"); got != want {
		t.Fatalf("ResolveRoot tmp = %q, want %q", got, want)
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
