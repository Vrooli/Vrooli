package buildinfo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/statecontract"
)

func TestPreparedBundleIntentV1ReaderAndWriter(t *testing.T) {
	home := t.TempDir()
	intent := PreparedBundleIntent{
		Version: 1, Manifest: strings.Repeat("a", 64), Root: strings.Repeat("b", 64),
		Prior:        map[string]string{"vrooli": "", "vrooli-api": strings.Repeat("c", 64)},
		SameContract: true, SourceRoot: filepath.Join(string(filepath.Separator), "checkout"),
	}
	if err := WritePreparedBundleIntent(home, intent); err != nil {
		t.Fatal(err)
	}
	got, err := ReadPreparedBundleIntent(home)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Version != intent.Version || got.Manifest != intent.Manifest || got.Root != intent.Root || !got.SameContract || got.SourceRoot != intent.SourceRoot || got.Prior["vrooli"] != "" || got.Prior["vrooli-api"] != intent.Prior["vrooli-api"] {
		t.Fatalf("intent round trip mismatch: %#v", got)
	}
	path, _, err := PreparedBundlePaths(home)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(data, &wire); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"version", "manifest_sha256", "root_sha256", "prior_sha256", "same_contract", "source_root"} {
		if _, ok := wire[field]; !ok {
			t.Errorf("v1 wire record omitted %q: %s", field, data)
		}
	}
}

func TestPreparedBundleIntentWriterRejectsInvalidRecords(t *testing.T) {
	valid := PreparedBundleIntent{Version: 1, Manifest: strings.Repeat("a", 64), Root: strings.Repeat("b", 64)}
	for _, tc := range []struct {
		name   string
		mutate func(*PreparedBundleIntent)
	}{
		{"unsupported version", func(intent *PreparedBundleIntent) { intent.Version = 2 }},
		{"malformed root digest", func(intent *PreparedBundleIntent) { intent.Root = "not-a-digest" }},
		{"noncanonical source root", func(intent *PreparedBundleIntent) { intent.SourceRoot = "relative/checkout" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			intent := valid
			tc.mutate(&intent)
			if err := WritePreparedBundleIntent(home, intent); err == nil {
				t.Fatal("invalid intent was accepted")
			}
			got, err := ReadPreparedBundleIntent(home)
			if err != nil || got != nil {
				t.Fatalf("invalid intent became durable: got=%#v err=%v", got, err)
			}
		})
	}
}

func TestPreparedBundleManifestFormatChangesOwner(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	const path = "internal/cliinstall/companions.go"
	data, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatal(err)
	}
	fixture := t.TempDir()
	writeTestFile(t, fixture, path, string(data))
	for _, extra := range []string{"internal/cliinstall/ordinary_bundle_transaction.go", "internal/cliinstall/rebaseline_bundle.go", "internal/cli/runtimehandlers/root_update_recovery.go", "internal/app/runtime/prepared_rebaseline.go", "internal/buildinfo/prepared_bundle_guard.go", "internal/buildinfo/root_activation_guard.go", "internal/buildinfo/source_root_pointer.go"} {
		content, err := os.ReadFile(filepath.Join(root, extra))
		if err != nil {
			t.Fatal(err)
		}
		writeTestFile(t, fixture, extra, string(content))
	}

	digest := func() string {
		t.Helper()
		d, err := wholeSourceDigest(fixture, statecontract.WholeSources["prepared-root-bundle"])
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	before := digest()
	changed := strings.Replace(string(data), `"SHA256SUMS"`, `"SHA256SUMS-v2"`, 1)
	if changed == string(data) {
		t.Fatal("manifest fixture no longer contains filename")
	}
	writeTestFile(t, fixture, path, changed)
	if digest() == before {
		t.Fatal("prepared bundle manifest change escaped owner")
	}
}

// The durable fence must detect changes to its wire fields, recovery path and
// source pointer reader, not merely changes to the bundle manifest.
func TestPreparedBundleRecoveryFormatChangesOwner(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	fixture := t.TempDir()
	paths := []string{"internal/cliinstall/companions.go", "internal/cliinstall/ordinary_bundle_transaction.go", "internal/cliinstall/rebaseline_bundle.go", "internal/cli/runtimehandlers/root_update_recovery.go", "internal/app/runtime/prepared_rebaseline.go", "internal/buildinfo/prepared_bundle_guard.go", "internal/buildinfo/root_activation_guard.go", "internal/buildinfo/source_root_pointer.go"}
	contents := map[string]string{}
	for _, path := range paths {
		data, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		contents[path] = string(data)
		writeTestFile(t, fixture, path, string(data))
	}
	digest := func() string {
		t.Helper()
		d, err := wholeSourceDigest(fixture, statecontract.WholeSources["prepared-root-bundle"])
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	before := digest()
	for _, tc := range []struct{ path, old, replacement string }{
		{"internal/buildinfo/prepared_bundle_guard.go", `json:"prior_sha256"`, `json:"prior_digests"`},
		{"internal/buildinfo/prepared_bundle_guard.go", `"ordinary-bundle"`, `"ordinary-bundle-v2"`},
		{"internal/buildinfo/prepared_bundle_guard.go", `json.Marshal(intent)`, `json.MarshalIndent(intent, "", "  ")`},
		{"internal/cliinstall/ordinary_bundle_transaction.go", `return finishOrdinaryBundle(home, binDir, manifest)`, `return clearOrdinaryBundleIntent(home)`},
		{"internal/buildinfo/source_root_pointer.go", `strings.TrimSpace(string(contents))`, `string(contents)`},
		{"internal/cliinstall/rebaseline_bundle.go", `"prepared-bundles"`, `"prepared-bundles-v2"`},
		{"internal/cliinstall/rebaseline_bundle.go", `filepath.Join(stage, "SHA256SUMS")`, `filepath.Join(stage, "manifest-v2")`},
		{"internal/app/runtime/prepared_rebaseline.go", `"progress.json"`, `"progress-v2.json"`},
		{"internal/app/runtime/prepared_rebaseline.go", `"watchdog-pending"`, `"watchdog-recovery"`},
		{"internal/app/runtime/prepared_rebaseline.go", `cliinstall.PreparedBundleWatchdogSHA(bundleDir, manifest)`, `cliinstall.PreparedBundleWatchdogSHA(manifest, bundleDir)`},
		{"internal/app/runtime/prepared_rebaseline.go", `progress.Candidate != candidateSHA`, `progress.Candidate == candidateSHA`},
		{"internal/app/runtime/prepared_rebaseline.go", `progress.Watchdog == watchdogSHA`, `progress.Watchdog != watchdogSHA`},
		{"internal/app/runtime/prepared_rebaseline.go", `e.retainBundle(home, filepath.Dir(originalCandidate), bundleSHA)`, `e.retainBundle(home, filepath.Dir(originalCandidate), "other-bundle")`},
		{"internal/app/runtime/prepared_rebaseline.go", `retainRebaselineFile(stagedWatchdog, pendingWatchdog)`, `retainRebaselineFile(pendingWatchdog, stagedWatchdog)`},
		{"internal/app/runtime/prepared_rebaseline.go", `rebaselineprogress.Write(progressPath, record)`, `rebaselineprogress.Write(record, progressPath)`},
		{"internal/app/runtime/prepared_rebaseline.go", `e.removeBundle(home, bundleSHA)`, `e.removeBundle(home, "other-bundle")`},
	} {
		t.Run(tc.old, func(t *testing.T) {
			changed := strings.Replace(contents[tc.path], tc.old, tc.replacement, 1)
			if changed == contents[tc.path] {
				t.Fatal("mutation target missing")
			}
			writeTestFile(t, fixture, tc.path, changed)
			defer writeTestFile(t, fixture, tc.path, contents[tc.path])
			if digest() == before {
				t.Fatal("recovery format change escaped owner")
			}
		})
	}
}

// Ordinary bundle coordination may evolve without changing the recovery
// format. The record owner remains stamped, so changing its wire field names
// still changes the prepared-root-bundle contract.
func TestOrdinaryBundleCoordinatorBehaviorDoesNotChangeOwner(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	fixture := t.TempDir()
	paths := []string{"internal/cliinstall/companions.go", "internal/cliinstall/ordinary_bundle.go", "internal/cliinstall/ordinary_bundle_transaction.go", "internal/cliinstall/rebaseline_bundle.go", "internal/cli/runtimehandlers/root_update_recovery.go", "internal/app/runtime/prepared_rebaseline.go", "internal/buildinfo/prepared_bundle_guard.go", "internal/buildinfo/root_activation_guard.go", "internal/buildinfo/source_root_pointer.go"}
	contents := map[string]string{}
	for _, path := range paths {
		data, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		contents[path] = string(data)
		writeTestFile(t, fixture, path, string(data))
	}
	digest := func() string {
		t.Helper()
		got, err := wholeSourceDigest(fixture, statecontract.WholeSources["prepared-root-bundle"])
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	before := digest()
	ordinaryPath := "internal/cliinstall/ordinary_bundle.go"
	changed := strings.Replace(contents[ordinaryPath], `"no ordinary bundle install needs recovery"`, `"ordinary recovery needs attention"`, 1)
	if changed == contents[ordinaryPath] {
		t.Fatal("ordinary bundle behavior mutation target missing")
	}
	writeTestFile(t, fixture, ordinaryPath, changed)
	if digest() != before {
		t.Fatal("ordinary bundle coordinator behavior churned the prepared-root-bundle owner")
	}
}
