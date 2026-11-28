package agentaccess

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	browserclosure "github.com/vrooli/vrooli/packages/browser-closure"
)

func TestVerifiedBrowserRuntimeEvidenceUsesReceiptAndOmitsPaths(t *testing.T) {
	root := t.TempDir()
	browserRoot := filepath.Join(root, "resources", "playwright")
	driverRoot := filepath.Join(root, "playwright-driver")
	if err := os.MkdirAll(filepath.Join(browserRoot, "chromium-1169"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(driverRoot, 0755); err != nil {
		t.Fatal(err)
	}
	executable := []byte("temporary fake browser executable")
	if err := os.WriteFile(filepath.Join(browserRoot, "chromium-1169", "chrome"), executable, 0755); err != nil {
		t.Fatal(err)
	}
	execHash := sha256.Sum256(executable)
	if err := os.WriteFile(filepath.Join(driverRoot, "package.json"), []byte(`{"dependencies":{"rebrowser-playwright":"1.52.0"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	digest, err := browserclosure.ComputeRuntimeDigest(browserRoot)
	if err != nil {
		t.Fatal(err)
	}
	identity := browserclosure.NewRuntimeIdentity("chromium", "1169", "rebrowser-playwright-core", "1.52.0", digest)
	lifecycleStatus := "unqualified-no-node"
	if runtimeBrowserPlatform() == "linux-x64" {
		lifecycleStatus = "unqualified-no-receipt"
	}
	closure := runtimeClosureManifest{
		SchemaVersion: "v3", Platform: runtimeBrowserPlatform(), Identity: identity, Executable: "chromium-1169/chrome", ExecutableSHA: hex.EncodeToString(execHash[:]),
		SupportMatrix: "v3", TargetLifecycleStatus: map[string]string{"runtime_launch": lifecycleStatus, "browser_task": lifecycleStatus, "web_console_pane": lifecycleStatus, "update_rollback": lifecycleStatus, "uninstall": lifecycleStatus},
		LayoutStatus: "source-fixture-tested", ArtifactStatus: "unqualified-no-provisioned-artifact",
		CleanHostStatus: "unqualified-no-receipt", OnlineAcquisitionStatus: "unsupported-no-governed-browser-source",
		OfflineTransferStatus: "source-fixture-tested-linux-x64", OfflineTransferScope: "Deterministic archive and install fixtures run on linux-x64 only.",
		OfflineArtifactStatus: "unqualified-no-provisioned-artifact", OfflineCleanHostStatus: "unqualified-no-receipt",
	}
	writeJSONFixture(t, filepath.Join(browserRoot, "browser-closure.json"), closure)
	receipt := runtimeInstallReceipt{SchemaVersion: "v3", ArchiveSHA256: "0000000000000000000000000000000000000000000000000000000000000000", Identity: identity}
	writeJSONFixture(t, filepath.Join(browserRoot, ".offline-install-receipt.json"), receipt)
	t.Setenv("PLAYWRIGHT_BROWSERS_PATH", browserRoot)
	t.Setenv("BROWSER_EXECUTABLE_PATH", filepath.Join(browserRoot, closure.Executable))
	t.Setenv("BAS_SIDECAR_DRIVER_DIR", "playwright-driver")

	evidence, err := verifiedBrowserRuntimeEvidence()
	if err != nil {
		t.Fatal(err)
	}
	want := &BrowserRuntimeEvidence{BundleVersion: identity.BundleVersion, BrowserVersion: identity.BrowserVersion, ClosureSHA256: identity.SHA256, Driver: identity.Driver, DriverVersion: identity.DriverVersion, Source: "offline-install-receipt", Channel: "offline-transfer"}
	if evidence == nil || *evidence != *want {
		t.Fatalf("unexpected runtime evidence: %#v", evidence)
	}
	closure.TargetLifecycleStatus["browser_task"] = "supported"
	writeJSONFixture(t, filepath.Join(browserRoot, "browser-closure.json"), closure)
	if evidence, err := verifiedBrowserRuntimeEvidence(); err == nil || evidence != nil {
		t.Fatalf("forged browser task status must not pass packaged runtime validation: evidence=%#v err=%v", evidence, err)
	}
	closure.TargetLifecycleStatus["browser_task"] = lifecycleStatus
	writeJSONFixture(t, filepath.Join(browserRoot, "browser-closure.json"), closure)
	receipt.Identity = browserclosure.NewRuntimeIdentity("chromium", "1169", "rebrowser-playwright-core", "1.53.0", digest)
	writeJSONFixture(t, filepath.Join(browserRoot, ".offline-install-receipt.json"), receipt)
	if evidence, err := verifiedBrowserRuntimeEvidence(); err == nil || evidence != nil {
		t.Fatalf("mismatched receipt must not claim identity: evidence=%#v err=%v", evidence, err)
	}
	receipt.Identity = identity
	writeJSONFixture(t, filepath.Join(browserRoot, ".offline-install-receipt.json"), receipt)
	if err := os.WriteFile(filepath.Join(browserRoot, closure.Executable), append(executable, []byte(" changed after packaging")...), 0755); err != nil {
		t.Fatal(err)
	}
	if evidence, err := verifiedBrowserRuntimeEvidence(); err == nil || evidence != nil {
		t.Fatalf("changed browser bytes must invalidate the packaged closure: evidence=%#v err=%v", evidence, err)
	}
}

func TestVerifiedBrowserRuntimeEvidenceRejectsMissingOrMismatchedProvenance(t *testing.T) {
	t.Setenv("PLAYWRIGHT_BROWSERS_PATH", "")
	t.Setenv("BROWSER_EXECUTABLE_PATH", "")
	t.Setenv("BAS_SIDECAR_DRIVER_DIR", "")
	if evidence, err := verifiedBrowserRuntimeEvidence(); err != nil || evidence != nil {
		t.Fatalf("empty runtime contract should not claim identity: evidence=%#v err=%v", evidence, err)
	}
	t.Setenv("PLAYWRIGHT_BROWSERS_PATH", t.TempDir())
	if evidence, err := verifiedBrowserRuntimeEvidence(); err == nil || evidence != nil {
		t.Fatalf("partial runtime contract must fail closed: evidence=%#v err=%v", evidence, err)
	}
}

func writeJSONFixture(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}
