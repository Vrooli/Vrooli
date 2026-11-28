//go:build !windows

package agentinstall

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andybalholm/brotli"
	harnesscatalog "github.com/vrooli/cli-core/agentcatalog/harness"
)

func TestPrunePreviewMatchesDeletionAndRetainsCurrentPlusRollback(t *testing.T) {
	home := t.TempDir()
	bin := filepath.Join(home, ".local", "bin")
	foundDir := filepath.Join(home, "tools")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(foundDir, 0o755); err != nil {
		t.Fatal(err)
	}
	found := filepath.Join(foundDir, "codex")
	const foundBytes = "#!/bin/sh\necho 'codex 9.9.9'\n"
	if err := os.WriteFile(found, []byte(foundBytes), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("PATH", foundDir+string(os.PathListSeparator)+"/usr/bin:/bin")
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	h, err := LoadHarness(root, "codex")
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"1.0.0", "2.0.0", "3.0.0"} {
		writeReceiptOwnedInstall(t, h, home, version)
	}
	activePrefixForLink, err := NPMInstallPrefix(home, "codex", "3.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if err := LinkNPMBinary(activePrefixForLink, h.Binary, bin); err != nil {
		t.Fatal(err)
	}
	plan, err := PreviewPruneOldManaged(root, "codex", 2)
	if err != nil {
		t.Fatal(err)
	}
	oldPrefix, err := NPMInstallPrefix(home, "codex", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	rollbackPrefix, err := NPMInstallPrefix(home, "codex", "2.0.0")
	if err != nil {
		t.Fatal(err)
	}
	activePrefix, err := NPMInstallPrefix(home, "codex", "3.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Remove) != 1 || plan.Remove[0] != oldPrefix {
		t.Fatalf("remove preview=%v", plan.Remove)
	}
	if !pathIn(plan.Preserved, rollbackPrefix) || !pathIn(plan.Preserved, activePrefix) || !pathIn(plan.Preserved, found) {
		t.Fatalf("preserved preview=%v", plan.Preserved)
	}
	tampered := plan
	tampered.Remove = []string{activePrefix}
	if err := ApplyPruneOldManaged(context.Background(), root, tampered); err == nil {
		t.Fatal("tampered remove set was accepted")
	}
	if _, err := os.Stat(activePrefix); err != nil {
		t.Fatalf("tampered plan removed active prefix: %v", err)
	}
	writeReceiptOwnedInstall(t, h, home, "4.0.0")
	if err := ApplyPruneOldManaged(context.Background(), root, plan); err == nil {
		t.Fatal("stale preview was accepted after managed state changed")
	}
	if _, err := os.Stat(oldPrefix); err != nil {
		t.Fatalf("stale plan removed an install: %v", err)
	}
	plan, err = PreviewPruneOldManaged(root, "codex", 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyPruneOldManaged(context.Background(), root, plan); err != nil {
		t.Fatal(err)
	}
	newestPrefix, err := NPMInstallPrefix(home, "codex", "4.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(rollbackPrefix); !os.IsNotExist(err) {
		t.Fatalf("obsolete rollback prefix remains: %v", err)
	}
	if _, err := os.Stat(oldPrefix); !os.IsNotExist(err) {
		t.Fatalf("old prefix remains: %v", err)
	}
	for _, path := range []string{newestPrefix, activePrefix} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("retained prefix %s: %v", path, err)
		}
	}
	if data, err := os.ReadFile(found); err != nil || string(data) != foundBytes {
		t.Fatalf("found copy changed: %v", err)
	}
}

func writeReceiptOwnedInstall(t *testing.T, h Harness, home, version string) {
	t.Helper()
	prefix, err := NPMInstallPrefix(home, h.Resource, version)
	if err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(prefix, filepath.FromSlash(h.Executable))
	if err := os.MkdirAll(filepath.Dir(executable), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(executable, []byte("#!/bin/sh\necho '"+h.Binary+" "+version+"'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	digest := treeDigest(t, filepath.Join(prefix, "node_modules"))
	receipt, _ := json.Marshal(installationRecord{Resource: h.Resource, Package: h.Package, Version: version, TreeSHA256: digest})
	if err := os.WriteFile(filepath.Join(prefix, "installation.json"), receipt, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestCleanupPreviewMatchesRemovalAndLeavesFoundCopyUntouched(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	bin := filepath.Join(home, ".local", "bin")
	foundDir := filepath.Join(home, "tools")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(foundDir, 0o755); err != nil {
		t.Fatal(err)
	}
	found := filepath.Join(foundDir, "codex")
	const original = "#!/bin/sh\necho 'codex 9.9.9'\n"
	if err := os.WriteFile(found, []byte(original), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", foundDir+string(os.PathListSeparator)+"/usr/bin:/bin")
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	h, err := LoadHarness(root, "codex")
	if err != nil {
		t.Fatal(err)
	}
	prefix, err := NPMInstallPrefix(home, h.Resource, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(prefix, filepath.FromSlash(h.Executable))
	if err := os.MkdirAll(filepath.Dir(executable), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(executable, []byte("#!/bin/sh\necho 'codex 1.2.3'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	digest := treeDigest(t, filepath.Join(prefix, "node_modules"))
	receipt, _ := json.Marshal(installationRecord{Resource: h.Resource, Package: h.Package, Version: "1.2.3", TreeSHA256: digest})
	if err := os.WriteFile(filepath.Join(prefix, "installation.json"), receipt, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := LinkNPMBinary(prefix, h.Binary, bin); err != nil {
		t.Fatal(err)
	}

	inventory, err := InventoryHarness(context.Background(), root, "codex")
	if err != nil {
		t.Fatal(err)
	}
	if inventory.Doctor.State != StateFound || inventory.Doctor.InstallState != "found" {
		t.Fatalf("doctor=%+v", inventory.Doctor)
	}
	plan, err := PreviewCleanup(root, "codex")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Remove) != 2 || !pathIn(plan.Remove, prefix) || !pathIn(plan.Remove, filepath.Join(bin, "codex")) {
		t.Fatalf("preview removal=%v", plan.Remove)
	}
	if len(plan.Preserved) != 1 || plan.Preserved[0] != found {
		t.Fatalf("preview preservation=%v", plan.Preserved)
	}
	tampered := plan
	tampered.Remove = nil
	if err := ApplyCleanup(context.Background(), root, tampered); err == nil {
		t.Fatal("cleanup accepted caller-supplied removal fields that differ from preview")
	}
	if err := ApplyCleanup(context.Background(), root, plan); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(prefix); !os.IsNotExist(err) {
		t.Fatalf("managed prefix remains: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(bin, "codex")); !os.IsNotExist(err) {
		t.Fatalf("managed link remains: %v", err)
	}
	data, err := os.ReadFile(found)
	if err != nil || string(data) != original {
		t.Fatalf("found copy changed: %v", err)
	}
}

func TestManagementUninstallRequiresCurrentPreviewIdentity(t *testing.T) {
	f := newHarnessFixture(t, "codex", "codex")
	f.harness.Package = "@openai/codex"
	f.prefix("1.0.0")
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	preview, err := ManageHarness(t.Context(), root, "codex", "preview_cleanup", "operator", "")
	if err != nil || !preview.Success || preview.CleanupPlan == nil {
		t.Fatalf("preview: %+v %v", preview, err)
	}
	f.prefix("2.0.0")
	for _, identity := range []string{"", preview.CleanupPlan.ID} {
		result, err := ManageHarness(t.Context(), root, "codex", "uninstall", "operator", identity)
		if err != nil || result.Success {
			t.Fatalf("changed preview accepted: %+v %v", result, err)
		}
	}
	current, err := ManageHarness(t.Context(), root, "codex", "preview_cleanup", "operator", "")
	if err != nil {
		t.Fatal(err)
	}
	result, err := ManageHarness(t.Context(), root, "codex", "uninstall", "operator", current.CleanupPlan.ID)
	if err != nil || !result.Success {
		t.Fatalf("current preview refused: %+v %v", result, err)
	}
	for _, path := range current.CleanupPlan.Remove {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("previewed path remains: %s %v", path, err)
		}
	}
}

func TestDesktopInstallRejectsEscapingHomeSymlink(t *testing.T) {
	home, outside := t.TempDir(), t.TempDir()
	if err := os.Symlink(outside, filepath.Join(home, "linked")); err != nil {
		t.Fatal(err)
	}
	if err := desktopPath(home, filepath.Join(home, "linked", "future", "binary")); err == nil {
		t.Fatal("escaping install path accepted")
	}
	if err := desktopPath(home, filepath.Join(home, "new", "nested", "binary")); err != nil {
		t.Fatalf("new user-owned directory refused: %v", err)
	}
	for _, manifest := range []string{"", "https://invalid.example/manifest"} {
		h := Harness{Home: home, BinDir: filepath.Join(home, "linked"), Resource: "antigravity", ManifestBase: manifest}
		if err := installDesktopRelease(context.Background(), h, "1.0.0", desktopHarnessDeps()); err == nil || !strings.Contains(err.Error(), "escapes your home") {
			t.Fatalf("native and manifest routes must reject the escaped path before acquisition: %v", err)
		}
	}
	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 0 {
		t.Fatalf("outside path changed: %v %v", entries, err)
	}
}

func TestDesktopBrotliDecodeBoundsAndCleansFailedOutput(t *testing.T) {
	var compressed bytes.Buffer
	writer := brotli.NewWriter(&compressed)
	contents := bytes.Repeat([]byte("native executable bytes"), 100)
	if _, err := writer.Write(contents); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"valid", "too-large", "corrupt", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			source := filepath.Join(t.TempDir(), "grok.br")
			data := compressed.Bytes()
			if mode == "corrupt" {
				data = []byte("not a Brotli artifact")
			}
			if err := os.WriteFile(source, data, 0o600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "cancelled" {
				cancel()
			}
			limit := int64(len(contents))
			if mode == "too-large" {
				limit--
			}
			decoded, err := decodeNativeExecutable(ctx, source, limit)
			if mode == "valid" {
				if err != nil {
					t.Fatal(err)
				}
				actual, err := os.ReadFile(decoded)
				if err != nil || !bytes.Equal(actual, contents) {
					t.Fatalf("decoded artifact mismatch: %v", err)
				}
			} else if err == nil {
				t.Fatal("invalid artifact accepted")
			} else if _, err := os.Lstat(source + ".decoded"); !os.IsNotExist(err) {
				t.Fatalf("failed output retained: %v", err)
			}
		})
	}
}

func TestCompressedNativeCandidateReplacesOnlyItsOwnWrapper(t *testing.T) {
	for _, mode := range []string{"wrapper", "regular-entry", "outside-entry", "outside-native"} {
		t.Run(mode, func(t *testing.T) {
			candidate, outside := t.TempDir(), t.TempDir()
			native := filepath.Join(candidate, "node_modules", "native", "tool.br")
			entry := filepath.Join(candidate, "node_modules", ".bin", "tool")
			wrapper := filepath.Join(candidate, "wrapper")
			for _, dir := range []string{filepath.Dir(native), filepath.Dir(entry)} {
				if err := os.MkdirAll(dir, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			var compressed bytes.Buffer
			writer := brotli.NewWriter(&compressed)
			if _, err := writer.Write([]byte("native bytes")); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			if mode == "outside-native" {
				if err := os.WriteFile(filepath.Join(outside, "tool.br"), compressed.Bytes(), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(outside, "tool.br"), native); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(native, compressed.Bytes(), 0o600); err != nil {
				t.Fatal(err)
			}
			if mode == "outside-entry" {
				wrapper = filepath.Join(outside, "wrapper")
			}
			if err := os.WriteFile(wrapper, []byte("wrapper bytes"), 0o600); err != nil {
				t.Fatal(err)
			}
			if mode == "regular-entry" {
				if err := os.WriteFile(entry, []byte("preserve"), 0o600); err != nil {
					t.Fatal(err)
				}
			} else if err := os.Symlink(wrapper, entry); err != nil {
				t.Fatal(err)
			}
			h := Harness{Executable: "node_modules/.bin/tool", NativeArtifact: &harnesscatalog.NativeArtifact{Dependency: "native", Executable: "tool.br", Compression: "brotli"}}
			got, err := prepareNativeCandidate(t.Context(), h, candidate)
			if mode == "wrapper" {
				data, readErr := os.ReadFile(got)
				if err != nil || readErr != nil || string(data) != "native bytes" {
					t.Fatalf("native preparation: %q %v %v", data, err, readErr)
				}
			} else if err == nil {
				t.Fatal("unsafe candidate entry accepted")
			}
			if data, err := os.ReadFile(wrapper); err != nil || string(data) != "wrapper bytes" {
				t.Fatalf("original wrapper changed: %q %v", data, err)
			}
		})
	}
}

// The served J8 fixture asks the real owner to acquire an older upstream
// release, then operates Update through the UI. No fake version or receipt.
func TestHarnessOldGrokJourneyFixture(t *testing.T) {
	root := os.Getenv("HARNESS_OLD_GROK_JOURNEY_ROOT")
	if root == "" {
		t.Skip("called by the isolated J8 journey")
	}
	h, err := LoadHarness(root, "grok")
	if err != nil {
		t.Fatal(err)
	}
	if err := installDesktopRelease(context.Background(), h, "1.0.69", desktopHarnessDeps()); err != nil {
		t.Fatal(err)
	}
}
