//go:build linux

package hostwireless

import (
	"context"
	"encoding/json"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/vrooli/internal/hostinventory"
)

// This opt-in native fixture uses explicitly supplied prepared bundles and
// synthetic boot files. kmod always operates on private copied generations.
func TestNativeSignedIndexedActivation(t *testing.T) {
	input := os.Getenv("VROOLI_WIRELESS_NATIVE_BUNDLES")
	if input == "" {
		t.Skip("set VROOLI_WIRELESS_NATIVE_BUNDLES to a JSON map of profiles to verified bundle directories")
	}
	var bundles map[string]string
	if err := json.Unmarshal([]byte(input), &bundles); err != nil {
		t.Fatal(err)
	}
	if len(bundles) == 0 {
		t.Fatal("native fixture requires an explicit prepared bundle")
	}
	for profile, directory := range bundles {
		t.Run(profile, func(t *testing.T) {
			ctx := context.Background()
			spec, ok := preparationProfiles[profile]
			if !ok {
				t.Fatal("unknown native fixture profile")
			}
			data, err := os.ReadFile(filepath.Join(directory, "receipt.json"))
			if err != nil {
				t.Fatal(err)
			}
			var receipt Receipt
			if err := json.Unmarshal(data, &receipt); err != nil {
				t.Fatal(err)
			}
			if receipt.Profile != profile || receipt.KernelRelease != nobleABI || receipt.State != "prepared" || !receipt.CopiedTreeRollback {
				t.Fatal("native fixture preparation identity mismatch")
			}
			expected, err := Prepare(ctx, profile, false)
			if err != nil || expected.State != "eligible" || receipt.ArtifactDirectory != filepath.Clean(directory) {
				t.Fatalf("native fixture host or canonical directory mismatch: %v", err)
			}
			if _, hash, err := verifyPreparedBundle(ctx, expected, spec, filepath.Base(directory)); err != nil || hash != digest(data) {
				t.Fatalf("native fixture canonical preparation verification: %v", err)
			}
			syntheticPlan, synthetic, _ := copiedActivationFixture(t)
			plan := ActivationPlan{
				Profile: profile, Bundle: filepath.Base(directory), BundleReceiptSHA256: digest(data),
				KernelRelease: nobleABI, OverlayDirectory: overlayDirectory(profile), Modaliases: receipt.Modaliases,
				RecoveryCandidates: syntheticPlan.RecoveryCandidates, RecoveryRelease: syntheticPlan.RecoveryRelease,
			}
			plan.KernelSymbolsSHA256, err = preparedKernelSymbols(directory)
			if err != nil {
				t.Fatal(err)
			}
			for _, module := range spec.modules {
				name := module + ".ko"
				hash := receipt.ArtifactSHA256[name]
				if err := verifyFile(ctx, os.DirFS(directory), name, hash); err != nil {
					t.Fatal(err)
				}
				out, err := privateKmod(ctx, "/usr/sbin/modinfo", "-k", nobleABI, "-F", "filename", module).Output()
				if err != nil {
					t.Fatal(err)
				}
				original := observeFile(ctx, strings.TrimSpace(string(out)))
				if original.Issue != "" {
					t.Fatal(original.Issue)
				}
				plan.Originals = append(plan.Originals, original)
				plan.Candidates = append(plan.Candidates, FileIdentity{Path: name, SHA256: hash})
			}
			for _, file := range syntheticPlan.Preserved {
				if strings.HasPrefix(file.Path, "/boot/") || file.Path == "/etc/default/grub" {
					plan.Preserved = append(plan.Preserved, file)
				}
			}
			// Native provider and index identities are observed, while the boot
			// recovery identities above remain visibly synthetic fixture data.
			native := ActivationPlan{KernelRelease: nobleABI}
			inventory, err := hostinventory.NewIntegrityCollector(hostinventory.IntegrityCollectorOptions{}).Collect(ctx)
			if err != nil {
				t.Fatal(err)
			}
			observeActivationHost(ctx, &native, spec, inventory)
			plan.WiredInterfaces, plan.WiredDrivers, plan.NVIDIALoadedModules = native.WiredInterfaces, native.WiredDrivers, native.NVIDIALoadedModules
			for _, file := range native.Preserved {
				if strings.HasPrefix(file.Path, "/lib/modules/") || file.Path == "/etc/depmod.d/ubuntu.conf" || file.Path == "/proc/net/route" {
					if file.Issue != "" {
						t.Fatal(file.Issue)
					}
					plan.Preserved = append(plan.Preserved, file)
				}
			}
			route, err := os.ReadFile("/proc/net/route")
			if err != nil {
				t.Fatal(err)
			}
			synthetic["proc/net/route"] = &fstest.MapFile{Data: route}
			baseline := nativeFixtureFiles{FS: os.DirFS("/"), synthetic: synthetic}
			temp := t.TempDir()
			certFile, key := testCertificate(t, temp, "native-index")
			cert, err := readPublicCertificate(certFile)
			if err != nil {
				t.Fatal(err)
			}
			signedDir := filepath.Join(temp, "signed")
			if err := os.Mkdir(signedDir, 0700); err != nil {
				t.Fatal(err)
			}
			for _, candidate := range plan.Candidates {
				payload, err := os.ReadFile(filepath.Join(directory, candidate.Path))
				if err != nil {
					t.Fatal(err)
				}
				target := filepath.Join(signedDir, candidate.Path)
				writeFixture(t, target, string(payload))
				cmd := exec.CommandContext(ctx, "/lib/modules/"+nobleABI+"/build/scripts/sign-file", "sha256", key, certFile, target)
				cmd.Env = []string{"LC_ALL=C", "OPENSSL_CONF=/dev/null"}
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("sign native fixture copy: %v %s", err, out)
				}
			}
			tx, err := newIndexedActivation(ctx, baseline, plan, nativePrivateIndexes)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = tx.close() })
			stock := maps.Clone(tx.stockTree)
			modules, err := tx.stage(ctx, os.DirFS(signedDir), cert)
			if err != nil {
				t.Fatal(err)
			}
			if err := tx.verifySelected(ctx); err != nil {
				t.Fatal(err)
			}
			if os.Getenv("VROOLI_WIRELESS_DURABLE_NATIVE_TEST") == "1" {
				rehearseNativeDurableWireless(t, tx, cert)
			}
			stockJSON, _ := json.Marshal(stock)
			candidateJSON, _ := json.Marshal(tx.candidateTree)
			stockIndexes, candidateIndexes := map[string]string{}, map[string]string{}
			for _, name := range depmodIndexes {
				file := filepath.ToSlash(filepath.Join(abiTree, name))
				stockIndexes[name], candidateIndexes[name] = stock[file].SHA256, tx.candidateTree[file].SHA256
			}
			evidence, err := json.Marshal(struct {
				Profile          string                 `json:"profile"`
				Bundle           string                 `json:"bundle"`
				ReceiptSHA       string                 `json:"bundle_receipt_sha256"`
				CertSHA          string                 `json:"certificate_der_sha256"`
				Modules          []SignedModuleEvidence `json:"signed_modules"`
				StockTreeSHA     string                 `json:"stock_tree_manifest_sha256"`
				CandidateTreeSHA string                 `json:"candidate_tree_manifest_sha256"`
				StockIndexes     map[string]string      `json:"stock_indexes"`
				CandidateIndexes map[string]string      `json:"candidate_indexes"`
				Wired            map[string]string      `json:"wired_drivers"`
				NVIDIA           []string               `json:"nvidia_modules"`
				SyntheticBoot    bool                   `json:"synthetic_boot_fixture"`
			}{
				Profile: profile, Bundle: plan.Bundle, ReceiptSHA: plan.BundleReceiptSHA256, CertSHA: tx.certSHA,
				Modules: modules, StockTreeSHA: digest(stockJSON), CandidateTreeSHA: digest(candidateJSON),
				StockIndexes: stockIndexes, CandidateIndexes: candidateIndexes, Wired: plan.WiredDrivers,
				NVIDIA: plan.NVIDIALoadedModules, SyntheticBoot: true,
			})
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("proof=%s", evidence)
			t.Logf("profile=%s signed_modules=%d stock_entries=%d candidate_entries=%d", profile, len(modules), len(stock), len(tx.candidateTree))
			if err := tx.rollback(ctx); err != nil {
				t.Fatal(err)
			}
			after, err := snapshotPrivateTree(ctx, tx.stock.root)
			if err != nil || !maps.Equal(stock, after) {
				t.Fatalf("native indexed rollback differs: %v", err)
			}
			for _, module := range plan.Candidates {
				actual, err := binaryfetch.FileDigestContext(ctx, filepath.Join(directory, module.Path))
				if err != nil || actual != module.SHA256 {
					t.Fatalf("prepared module changed: %v", err)
				}
			}
			t.Logf("profile=%s all_stock_bytes_restored=true kernel_trust_verified=false activation_eligible=false", profile)
		})
	}
}

type nativeFixtureFiles struct {
	fs.FS
	synthetic fstest.MapFS
}

func (f nativeFixtureFiles) Open(name string) (fs.File, error) {
	if _, ok := f.synthetic[name]; ok && (strings.HasPrefix(name, "boot/") || name == "etc/default/grub" || name == "proc/net/route") {
		return f.synthetic.Open(name)
	}
	return f.FS.Open(name)
}

func (f nativeFixtureFiles) ReadDir(name string) ([]fs.DirEntry, error) {
	return fs.ReadDir(f.FS, name)
}
