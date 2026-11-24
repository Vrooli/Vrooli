package assets

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	browserclosure "github.com/vrooli/vrooli/packages/browser-closure"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/infra"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/testutil"
)

func TestServiceUsesPlaywright(t *testing.T) {
	if !ServiceUsesPlaywright(manifest.Service{ID: "PLAYWRIGHT-driver"}) {
		t.Fatal("missed service id")
	}
	if !ServiceUsesPlaywright(manifest.Service{ID: "api", Env: map[string]string{"PLAYWRIGHT_BROWSERS_PATH": "x"}}) {
		t.Fatal("missed env")
	}
	if ServiceUsesPlaywright(manifest.Service{ID: "api", Env: map[string]string{"PORT": "1"}}) {
		t.Fatal("false positive")
	}
}

func TestCompatiblePlaywrightHostPlatform(t *testing.T) {
	for _, tc := range []struct {
		platform, host string
		want           bool
	}{
		{"linux-x64", "ubuntu24.04-x64", true}, {"linux-arm64", "debian12-arm64", true},
		{"darwin-x64", "mac12", true}, {"darwin-arm64", "mac12-arm64", true}, {"windows-x64", "win64", true},
		{"linux-x64", "ubuntu24.04-arm64", false}, {"darwin-x64", "mac12-arm64", false}, {"windows-x64", "win32", false},
	} {
		if got := compatiblePlaywrightHost(tc.platform, tc.host); got != tc.want {
			t.Errorf("compatiblePlaywrightHost(%q,%q)=%t want %t", tc.platform, tc.host, got, tc.want)
		}
	}
}

func closureFixture(t *testing.T) (string, string, string) {
	t.Helper()
	bundle := t.TempDir()
	driverDir := filepath.Join(bundle, "playwright-driver")
	if err := os.MkdirAll(driverDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(driverDir, "package.json"), []byte(`{"dependencies":{"rebrowser-playwright":"1.52.0"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(bundle, "resources", "playwright")
	if err := os.MkdirAll(filepath.Join(dir, "chromium-1169", "chrome-linux"), 0755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "chromium-1169", "chrome-linux", "chrome")
	data := []byte("fixture executable; not launched")
	if err := os.WriteFile(exe, data, 0755); err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(data)
	digest, err := browserclosure.ComputeRuntimeDigest(dir)
	if err != nil {
		t.Fatal(err)
	}
	lifecycleStatus := "unqualified-no-node"
	if browserPlatform() == "linux-x64" {
		lifecycleStatus = "unqualified-no-receipt"
	}
	c := browserClosure{SchemaVersion: "v3", Platform: browserPlatform(), HostPlatform: "ubuntu24.04-x64", Identity: browserclosure.NewRuntimeIdentity("chromium", "1169", "rebrowser-playwright-core", "1.52.0", digest), HeadlessRev: "1169", FFmpegRev: "1011", Executable: "chromium-1169/chrome-linux/chrome", ExecutableSHA: hex.EncodeToString(h[:]), SupportMatrix: "v3", LayoutStatus: "source-fixture-tested", ArtifactStatus: "unqualified-no-provisioned-artifact", CleanHostStatus: "unqualified-no-receipt", OnlineAcquisitionStatus: "unsupported-no-governed-browser-source", OfflineTransferStatus: "source-fixture-tested-linux-x64", OfflineTransferScope: "Deterministic archive and install fixtures run on linux-x64 only.", OfflineArtifactStatus: "unqualified-no-provisioned-artifact", OfflineCleanHostStatus: "unqualified-no-receipt", TargetLifecycleStatus: map[string]string{"runtime_launch": lifecycleStatus, "browser_task": lifecycleStatus, "web_console_pane": lifecycleStatus, "update_rollback": lifecycleStatus, "uninstall": lifecycleStatus}}
	b, _ := json.Marshal(c)
	if err := os.WriteFile(filepath.Join(dir, "browser-closure.json"), b, 0644); err != nil {
		t.Fatal(err)
	}
	receipt, _ := json.Marshal(offlineBrowserInstallReceipt{SchemaVersion: "v3", ArchiveSHA256: strings.Repeat("a", 64), Identity: c.Identity})
	if err := os.WriteFile(filepath.Join(dir, ".offline-install-receipt.json"), receipt, 0644); err != nil {
		t.Fatal(err)
	}
	return bundle, dir, exe
}

func testConfig(bundle string) PlaywrightConfig {
	return PlaywrightConfig{BundlePath: bundle, FS: infra.RealFileSystem{}, EnvReader: infra.RealEnvReader{}, Ports: &testutil.MockPortAllocator{Ports: map[string]map[string]int{"bas": {"http": 1234}}}}
}

func packagedService() manifest.Service {
	return manifest.Service{ID: "bas", Env: map[string]string{"PLAYWRIGHT_BROWSERS_PATH": "resources/playwright", "BAS_SIDECAR_DRIVER_DIR": "playwright-driver"}}
}

func TestPackagedClosureBindsExactExecutableAndHash(t *testing.T) {
	bundle, dir, exe := closureFixture(t)
	env := map[string]string{"PLAYWRIGHT_BROWSERS_PATH": "resources/playwright"}
	svc := packagedService()
	if err := ApplyPlaywrightConventions(testConfig(bundle), svc, env); err != nil {
		t.Fatal(err)
	}
	if env["PLAYWRIGHT_BROWSERS_PATH"] != dir || env["BROWSER_EXECUTABLE_PATH"] != exe {
		t.Fatalf("resolved environment = %#v", env)
	}
}

func TestOfflineInstalledClosureFeedsSingleDesktopExecutableContract(t *testing.T) {
	_, source, _ := closureFixture(t)
	root := t.TempDir()
	bundle := filepath.Join(root, "bundle")
	driver := filepath.Join(bundle, "playwright-driver")
	if err := os.MkdirAll(driver, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(driver, "package.json"), []byte(`{"dependencies":{"rebrowser-playwright":"1.52.0"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	archive, manifest := filepath.Join(root, "closure.tar.gz"), filepath.Join(root, "closure.json")
	expected := browserclosure.Expected{Platform: browserPlatform(), HostPlatform: "ubuntu24.04-x64", Package: "rebrowser-playwright-core", Version: "1.52.0", ChromiumRev: "1169", HeadlessRev: "1169", FFmpegRev: "1011"}
	if _, err := browserclosure.Create(source, archive, manifest, expected); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(bundle, "resources", "playwright")
	if err := browserclosure.Install(archive, manifest, destination, expected); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"PLAYWRIGHT_BROWSERS_PATH": "resources/playwright"}
	if err := ApplyPlaywrightConventions(testConfig(bundle), packagedService(), env); err != nil {
		t.Fatalf("desktop runtime rejected offline-installed closure: %v", err)
	}
	want := filepath.Join(destination, "chromium-1169", "chrome-linux", "chrome")
	if env["BROWSER_EXECUTABLE_PATH"] != want || env["PLAYWRIGHT_BROWSERS_PATH"] != destination {
		t.Fatalf("offline install did not preserve sole executable contract: %#v", env)
	}
}

// TestMB6OfflineClosureDurableFixture is selected by scenario-to-desktop's
// durable validation matrix. It writes a small evidence record only when the
// matrix supplies an isolated output path.
func TestMB6OfflineClosureDurableFixture(t *testing.T) {
	expectedLifecycleStatus := "unqualified-no-node"
	if browserPlatform() == "linux-x64" {
		expectedLifecycleStatus = "unqualified-no-receipt"
	}
	evidencePath := os.Getenv("MB6_BROWSER_CLOSURE_EVIDENCE_PATH")
	if evidencePath == "" {
		t.Skip("durable fixture output path was not supplied")
	}
	_, source, _ := closureFixture(t)
	root := t.TempDir()
	bundle := filepath.Join(root, "fresh-bundle")
	driver := filepath.Join(bundle, "playwright-driver")
	if err := os.MkdirAll(driver, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(driver, "package.json"), []byte(`{"dependencies":{"rebrowser-playwright":"1.52.0"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	archive, manifest := filepath.Join(root, "closure.tar.gz"), filepath.Join(root, "closure.json")
	expected := browserclosure.Expected{Platform: browserPlatform(), HostPlatform: "ubuntu24.04-x64", Package: "rebrowser-playwright-core", Version: "1.52.0", ChromiumRev: "1169", HeadlessRev: "1169", FFmpegRev: "1011"}
	created, err := browserclosure.Create(source, archive, manifest, expected)
	if err != nil {
		t.Fatalf("create offline bundle fixture: %v", err)
	}
	destination := filepath.Join(bundle, "resources", "playwright")
	if err := browserclosure.Install(archive, manifest, destination, expected); err != nil {
		t.Fatalf("install into empty bundle root: %v", err)
	}
	var receipt offlineBrowserInstallReceipt
	receiptBytes, err := os.ReadFile(filepath.Join(destination, ".offline-install-receipt.json"))
	if err != nil || json.Unmarshal(receiptBytes, &receipt) != nil {
		t.Fatalf("read installed receipt: %v", err)
	}
	if receipt.Identity != created.Identity {
		t.Fatal("installed generation identity differs from detached manifest")
	}
	env := map[string]string{"PLAYWRIGHT_BROWSERS_PATH": "resources/playwright"}
	if err := ApplyPlaywrightConventions(testConfig(bundle), packagedService(), env); err != nil {
		t.Fatalf("fresh offline bundle rejected before launch: %v", err)
	}
	wantExecutable := filepath.Join(destination, filepath.FromSlash(created.Executable))
	if env["BROWSER_EXECUTABLE_PATH"] != wantExecutable {
		t.Fatalf("runtime executable=%q want %q", env["BROWSER_EXECUTABLE_PATH"], wantExecutable)
	}
	checks := map[string]bool{"offline_manifest_and_install_receipt_identity_match": true, "fresh_bundle_binds_exact_browser_executable_path": true}
	rejectBeforeBinding := func(name string) {
		t.Helper()
		probe := map[string]string{"PLAYWRIGHT_BROWSERS_PATH": "resources/playwright"}
		if err := ApplyPlaywrightConventions(testConfig(bundle), packagedService(), probe); err == nil {
			t.Fatalf("%s was accepted", name)
		}
		if probe["BROWSER_EXECUTABLE_PATH"] != "" {
			t.Fatalf("%s failed after executable binding: %#v", name, probe)
		}
		checks[name] = true
	}
	receiptPath := filepath.Join(destination, ".offline-install-receipt.json")
	closurePath := filepath.Join(destination, "browser-closure.json")
	originalReceipt := append([]byte(nil), receiptBytes...)
	if err := os.Remove(receiptPath); err != nil {
		t.Fatal(err)
	}
	rejectBeforeBinding("missing_receipt_rejected_before_binding")
	if err := os.WriteFile(receiptPath, originalReceipt, 0644); err != nil {
		t.Fatal(err)
	}
	mutatedReceipt := receipt
	mutatedReceipt.ArchiveSHA256 = "not-a-digest"
	badReceipt, _ := json.Marshal(mutatedReceipt)
	if err := os.WriteFile(receiptPath, badReceipt, 0644); err != nil {
		t.Fatal(err)
	}
	rejectBeforeBinding("tampered_receipt_rejected_before_binding")
	mutatedReceipt = receipt
	mutatedReceipt.Identity.DriverVersion = "1.51.0"
	staleReceipt, _ := json.Marshal(mutatedReceipt)
	if err := os.WriteFile(receiptPath, staleReceipt, 0644); err != nil {
		t.Fatal(err)
	}
	rejectBeforeBinding("stale_receipt_rejected_before_binding")
	if err := os.WriteFile(receiptPath, originalReceipt, 0644); err != nil {
		t.Fatal(err)
	}
	var closure browserClosure
	closureBytes, err := os.ReadFile(closurePath)
	if err != nil || json.Unmarshal(closureBytes, &closure) != nil {
		t.Fatalf("read installed closure manifest: %v", err)
	}
	closure.OnlineAcquisitionStatus = "supported"
	forgedOnline, _ := json.Marshal(closure)
	if err := os.WriteFile(closurePath, forgedOnline, 0644); err != nil {
		t.Fatal(err)
	}
	rejectBeforeBinding("unsupported_online_acquisition_rejected_before_binding")
	closure.OnlineAcquisitionStatus = "unsupported-no-governed-browser-source"
	validClosure, _ := json.Marshal(closure)
	if err := os.WriteFile(closurePath, validClosure, 0644); err != nil {
		t.Fatal(err)
	}
	closure.TargetLifecycleStatus["browser_task"] = "supported"
	forgedLifecycle, _ := json.Marshal(closure)
	if err := os.WriteFile(closurePath, forgedLifecycle, 0644); err != nil {
		t.Fatal(err)
	}
	rejectBeforeBinding("forged_browser_task_status_rejected_before_binding")
	closure.TargetLifecycleStatus["browser_task"] = expectedLifecycleStatus
	validClosure, _ = json.Marshal(closure)
	if err := os.WriteFile(closurePath, validClosure, 0644); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(destination, filepath.FromSlash(created.Executable))
	originalExecutable, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(executable, []byte("tampered clean-host payload"), 0755); err != nil {
		t.Fatal(err)
	}
	rejectBeforeBinding("tampered_bundle_payload_rejected_before_binding")
	if err := os.WriteFile(executable, originalExecutable, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(evidencePath), 0755); err != nil {
		t.Fatal(err)
	}
	evidence := map[string]any{"schema_version": 1, "fixture": "s2d-mb6-offline-closure", "online_acquisition_status": closure.OnlineAcquisitionStatus, "offline_transfer_status": created.OfflineTransferStatus, "installed_runtime_identity": receipt.Identity, "receipt_archive_sha256": receipt.ArchiveSHA256, "browser_executable_path": wantExecutable, "checks": checks, "process_launched": false}
	data, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(evidencePath, append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestBrowserSupportMatrixContractMatchesRuntimeAdmission(t *testing.T) {
	type supportCell struct {
		BundleLayout string            `json:"bundle_layout"`
		Artifact     string            `json:"browser_artifact"`
		CleanHost    string            `json:"clean_host"`
		Lifecycle    map[string]string `json:"lifecycle_contract"`
	}
	var matrix struct {
		SchemaVersion string                 `json:"schema_version"`
		Targets       map[string]supportCell `json:"targets"`
		Remote        struct {
			Status            string            `json:"status"`
			OwnerGap          string            `json:"owner_gap"`
			LifecycleContract map[string]string `json:"lifecycle_contract"`
		} `json:"private-hosted-worker"`
	}
	path := filepath.Join("..", "..", "..", "browser-automation-studio", "bundle", "browser-support-matrix.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &matrix); err != nil {
		t.Fatal(err)
	}
	if matrix.SchemaVersion != "v3" || matrix.Remote.Status != "unsupported" || strings.TrimSpace(matrix.Remote.OwnerGap) == "" {
		t.Fatalf("support matrix lost its fail-closed remote contract: %+v", matrix)
	}
	wantLifecycleCells := []string{"workspace_isolation", "browser_closure_mount", "run_grant_session_binding", "network_effect_policy", "resource_timeout_limits", "artifact_integrity_receipt", "completion_cancel_crash_teardown"}
	if len(matrix.Remote.LifecycleContract) != len(wantLifecycleCells) {
		t.Fatalf("remote worker lifecycle inventory changed: %#v", matrix.Remote.LifecycleContract)
	}
	for _, cell := range wantLifecycleCells {
		if matrix.Remote.LifecycleContract[cell] != "unsupported-no-browser-task-owner" {
			t.Fatalf("remote worker lifecycle cell %q = %q, want explicit unsupported owner gap", cell, matrix.Remote.LifecycleContract[cell])
		}
	}
	cell, ok := matrix.Targets[browserPlatform()]
	if !ok {
		t.Fatalf("support matrix has no runtime platform cell %q", browserPlatform())
	}
	if cell.BundleLayout != "source-fixture-tested" || cell.Artifact != "unqualified-no-provisioned-artifact" || cell.CleanHost != "unqualified-no-receipt" {
		t.Fatalf("runtime platform support state is qualified or malformed: %+v", cell)
	}
	wantLifecycle := "unqualified-no-node"
	if browserPlatform() == "linux-x64" {
		wantLifecycle = "unqualified-no-receipt"
	}
	if len(cell.Lifecycle) != 5 {
		t.Fatalf("runtime lifecycle inventory changed: %#v", cell.Lifecycle)
	}
	for name, status := range cell.Lifecycle {
		if status != wantLifecycle {
			t.Fatalf("runtime lifecycle %s=%s, want %s", name, status, wantLifecycle)
		}
	}
	bundle, dir, _ := closureFixture(t)
	manifestPath := filepath.Join(dir, "browser-closure.json")
	contents, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var closure browserClosure
	if err := json.Unmarshal(contents, &closure); err != nil {
		t.Fatal(err)
	}
	closure.LayoutStatus, closure.ArtifactStatus, closure.CleanHostStatus = cell.BundleLayout, cell.Artifact, cell.CleanHost
	contents, _ = json.Marshal(closure)
	if err := os.WriteFile(manifestPath, contents, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ApplyPlaywrightConventions(testConfig(bundle), packagedService(), map[string]string{"PLAYWRIGHT_BROWSERS_PATH": "resources/playwright"}); err != nil {
		t.Fatalf("runtime rejected the source matrix's non-qualified state: %v", err)
	}
}

func TestPackagedClosureFailsClosed(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*browserClosure)
	}{
		{name: "floating browser version", edit: func(c *browserClosure) {
			c.Identity = browserclosure.NewRuntimeIdentity("chromium", "latest", "rebrowser-playwright-core", "1.52.0", c.Identity.SHA256)
		}},
		{name: "stale bundle version", edit: func(c *browserClosure) { c.Identity.BundleVersion += "-stale" }},
		{name: "driver version mismatch", edit: func(c *browserClosure) {
			c.Identity = browserclosure.NewRuntimeIdentity("chromium", "1169", "rebrowser-playwright-core", "1.53.0", c.Identity.SHA256)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bundle, dir, _ := closureFixture(t)
			manifestPath := filepath.Join(dir, "browser-closure.json")
			data, _ := os.ReadFile(manifestPath)
			var closure browserClosure
			_ = json.Unmarshal(data, &closure)
			tc.edit(&closure)
			data, _ = json.Marshal(closure)
			_ = os.WriteFile(manifestPath, data, 0644)
			err := ApplyPlaywrightConventions(testConfig(bundle), packagedService(), map[string]string{"PLAYWRIGHT_BROWSERS_PATH": "resources/playwright"})
			if err == nil || !strings.Contains(err.Error(), "metadata mismatch") {
				t.Fatalf("mismatched runtime identity accepted: %v", err)
			}
		})
	}
	t.Run("delivery status overclaim", func(t *testing.T) {
		bundle, dir, _ := closureFixture(t)
		manifestPath := filepath.Join(dir, "browser-closure.json")
		var closure browserClosure
		contents, err := os.ReadFile(manifestPath)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(contents, &closure); err != nil {
			t.Fatal(err)
		}
		closure.OnlineAcquisitionStatus = "supported"
		contents, _ = json.Marshal(closure)
		if err := os.WriteFile(manifestPath, contents, 0o644); err != nil {
			t.Fatal(err)
		}
		err = ApplyPlaywrightConventions(testConfig(bundle), packagedService(), map[string]string{"PLAYWRIGHT_BROWSERS_PATH": "resources/playwright"})
		if err == nil || !strings.Contains(err.Error(), "metadata mismatch") {
			t.Fatalf("delivery channel overclaim accepted: %v", err)
		}
	})
	t.Run("offline receipt fences staged identity", func(t *testing.T) {
		bundle, dir, _ := closureFixture(t)
		manifestPath := filepath.Join(dir, "browser-closure.json")
		data, _ := os.ReadFile(manifestPath)
		var closure browserClosure
		_ = json.Unmarshal(data, &closure)
		receiptPath := filepath.Join(dir, ".offline-install-receipt.json")
		receipt := offlineBrowserInstallReceipt{SchemaVersion: "v3", ArchiveSHA256: strings.Repeat("a", 64), Identity: closure.Identity}
		data, _ = json.Marshal(receipt)
		if err := os.WriteFile(receiptPath, data, 0644); err != nil {
			t.Fatal(err)
		}
		if err := ApplyPlaywrightConventions(testConfig(bundle), packagedService(), map[string]string{"PLAYWRIGHT_BROWSERS_PATH": "resources/playwright"}); err != nil {
			t.Fatalf("matching install receipt rejected: %v", err)
		}
		receipt.Identity = browserclosure.NewRuntimeIdentity("chromium", "1169", "rebrowser-playwright-core", "1.53.0", closure.Identity.SHA256)
		data, _ = json.Marshal(receipt)
		if err := os.WriteFile(receiptPath, data, 0644); err != nil {
			t.Fatal(err)
		}
		err := ApplyPlaywrightConventions(testConfig(bundle), packagedService(), map[string]string{"PLAYWRIGHT_BROWSERS_PATH": "resources/playwright"})
		if err == nil || !strings.Contains(err.Error(), "receipt mismatch") {
			t.Fatalf("post-install driver identity change accepted: %v", err)
		}
	})
	t.Run("missing install receipt rejected before executable binding", func(t *testing.T) {
		bundle, dir, _ := closureFixture(t)
		if err := os.Remove(filepath.Join(dir, ".offline-install-receipt.json")); err != nil {
			t.Fatal(err)
		}
		env := map[string]string{"PLAYWRIGHT_BROWSERS_PATH": "resources/playwright"}
		err := ApplyPlaywrightConventions(testConfig(bundle), packagedService(), env)
		if err == nil || !strings.Contains(err.Error(), "receipt mismatch") {
			t.Fatalf("missing receipt accepted: %v", err)
		}
		if env["BROWSER_EXECUTABLE_PATH"] != "" {
			t.Fatalf("executable binding happened before receipt validation: %#v", env)
		}
	})
	t.Run("bad checksum", func(t *testing.T) {
		bundle, dir, exe := closureFixture(t)
		_ = os.WriteFile(exe, []byte("changed"), 0755)
		manifestPath := filepath.Join(dir, "browser-closure.json")
		data, _ := os.ReadFile(manifestPath)
		var closure browserClosure
		_ = json.Unmarshal(data, &closure)
		closure.Identity.SHA256, _ = browserclosure.ComputeRuntimeDigest(dir)
		data, _ = json.Marshal(closure)
		_ = os.WriteFile(manifestPath, data, 0644)
		receipt := offlineBrowserInstallReceipt{SchemaVersion: "v3", ArchiveSHA256: strings.Repeat("a", 64), Identity: closure.Identity}
		receiptBytes, _ := json.Marshal(receipt)
		_ = os.WriteFile(filepath.Join(dir, ".offline-install-receipt.json"), receiptBytes, 0644)
		env := map[string]string{"PLAYWRIGHT_BROWSERS_PATH": "resources/playwright"}
		if err := ApplyPlaywrightConventions(testConfig(bundle), packagedService(), env); err == nil || !strings.Contains(err.Error(), "checksum") {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("whole bundle digest catches non executable payload mutation", func(t *testing.T) {
		bundle, _, exe := closureFixture(t)
		_ = os.WriteFile(exe, []byte("tampered payload"), 0755)
		err := ApplyPlaywrightConventions(testConfig(bundle), packagedService(), map[string]string{"PLAYWRIGHT_BROWSERS_PATH": "resources/playwright"})
		if err == nil || !strings.Contains(err.Error(), "bundle digest") {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("caller override conflict", func(t *testing.T) {
		bundle, _, _ := closureFixture(t)
		env := map[string]string{"PLAYWRIGHT_BROWSERS_PATH": "resources/playwright", "BROWSER_EXECUTABLE_PATH": "elsewhere/chrome"}
		if err := ApplyPlaywrightConventions(testConfig(bundle), packagedService(), env); err == nil || !strings.Contains(err.Error(), "conflicts") {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("traversal", func(t *testing.T) {
		bundle, dir, _ := closureFixture(t)
		b, _ := os.ReadFile(filepath.Join(dir, "browser-closure.json"))
		var c browserClosure
		_ = json.Unmarshal(b, &c)
		c.Executable = "../../outside"
		b, _ = json.Marshal(c)
		_ = os.WriteFile(filepath.Join(dir, "browser-closure.json"), b, 0644)
		env := map[string]string{"PLAYWRIGHT_BROWSERS_PATH": "resources/playwright"}
		if err := ApplyPlaywrightConventions(testConfig(bundle), packagedService(), env); err == nil {
			t.Fatal("accepted traversal")
		}
	})
	t.Run("missing manifest", func(t *testing.T) {
		bundle := t.TempDir()
		_ = os.MkdirAll(filepath.Join(bundle, "resources", "playwright"), 0755)
		env := map[string]string{"PLAYWRIGHT_BROWSERS_PATH": "resources/playwright"}
		if err := ApplyPlaywrightConventions(testConfig(bundle), packagedService(), env); err == nil {
			t.Fatal("accepted missing manifest")
		}
	})
	t.Run("platform mismatch", func(t *testing.T) {
		bundle, dir, _ := closureFixture(t)
		b, _ := os.ReadFile(filepath.Join(dir, "browser-closure.json"))
		var c browserClosure
		_ = json.Unmarshal(b, &c)
		c.Platform = "darwin-arm64"
		b, _ = json.Marshal(c)
		_ = os.WriteFile(filepath.Join(dir, "browser-closure.json"), b, 0o644)
		env := map[string]string{"PLAYWRIGHT_BROWSERS_PATH": "resources/playwright"}
		if err := ApplyPlaywrightConventions(testConfig(bundle), packagedService(), env); err == nil || !strings.Contains(err.Error(), "metadata mismatch") {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("forged supported status", func(t *testing.T) {
		bundle, dir, _ := closureFixture(t)
		b, _ := os.ReadFile(filepath.Join(dir, "browser-closure.json"))
		var c browserClosure
		_ = json.Unmarshal(b, &c)
		c.ArtifactStatus = "supported"
		b, _ = json.Marshal(c)
		_ = os.WriteFile(filepath.Join(dir, "browser-closure.json"), b, 0644)
		env := map[string]string{"PLAYWRIGHT_BROWSERS_PATH": "resources/playwright"}
		if err := ApplyPlaywrightConventions(testConfig(bundle), packagedService(), env); err == nil || !strings.Contains(err.Error(), "metadata mismatch") {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestNoAmbientElectronFallback(t *testing.T) {
	bundle := t.TempDir()
	electron := filepath.Join(bundle, "electron-chrome")
	_ = os.WriteFile(electron, []byte("not launched"), 0755)
	t.Setenv("ELECTRON_CHROMIUM_PATH", electron)
	env := map[string]string{"BROWSER_EXECUTABLE_PATH": "missing/chrome"}
	if err := ApplyPlaywrightConventions(testConfig(bundle), manifest.Service{ID: "playwright-driver"}, env); err == nil {
		t.Fatal("accepted absent explicit executable using Electron fallback")
	}
	if env["BROWSER_EXECUTABLE_PATH"] == electron {
		t.Fatal("used Electron fallback")
	}
}

func TestPlaywrightServiceRequiresAnExplicitBrowserPath(t *testing.T) {
	env := map[string]string{}
	err := ApplyPlaywrightConventions(testConfig(t.TempDir()), manifest.Service{ID: "playwright-driver"}, env)
	if err == nil || !strings.Contains(err.Error(), "requires PLAYWRIGHT_BROWSERS_PATH or an explicit BROWSER_EXECUTABLE_PATH") {
		t.Fatalf("missing packaged/explicit browser path error = %v", err)
	}
	if env["BROWSER_EXECUTABLE_PATH"] != "" {
		t.Fatalf("missing browser path was populated unexpectedly: %#v", env)
	}
}

func TestExplicitBrowserExecutableMustBeRegularAndRunnable(t *testing.T) {
	bundle := t.TempDir()
	regular := filepath.Join(bundle, "chrome")
	if err := os.WriteFile(regular, []byte("fixture"), 0644); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(bundle, "directory")
	if err := os.Mkdir(directory, 0755); err != nil {
		t.Fatal(err)
	}
	for name, path := range map[string]string{"non-executable file": "chrome", "directory": "directory"} {
		t.Run(name, func(t *testing.T) {
			env := map[string]string{"BROWSER_EXECUTABLE_PATH": path}
			if err := ApplyPlaywrightConventions(testConfig(bundle), manifest.Service{ID: "playwright-driver"}, env); err == nil {
				t.Fatalf("accepted %s as a browser executable", name)
			}
			if env["BROWSER_EXECUTABLE_PATH"] != path {
				t.Fatalf("rejected path was mutated: %#v", env)
			}
		})
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(regular, 0755); err != nil {
			t.Fatal(err)
		}
		env := map[string]string{"BROWSER_EXECUTABLE_PATH": "chrome"}
		if err := ApplyPlaywrightConventions(testConfig(bundle), manifest.Service{ID: "playwright-driver"}, env); err != nil {
			t.Fatalf("rejected runnable explicit executable: %v", err)
		}
		if env["BROWSER_EXECUTABLE_PATH"] != regular {
			t.Fatalf("resolved executable=%q want %q", env["BROWSER_EXECUTABLE_PATH"], regular)
		}
	}
}
