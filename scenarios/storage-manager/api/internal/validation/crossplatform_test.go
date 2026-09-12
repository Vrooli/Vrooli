package validation

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	corestorage "github.com/vrooli/api-core/storage"
)

func TestCrossPlatformFlagsBareAbsolutePathAcrossPlatforms(t *testing.T) {
	owner, root := crossPlatformOwner(t, corestorage.OwnerResource, "demo", corestorage.OwnerManifest{}, corestorage.StorageEntry{Name: "data", Path: corestorage.PortablePath{Value: "/var/lib/demo"}})
	findings, err := (crossPlatform{}).Analyze(context.Background(), AnalyzerContext{RepoRoot: root, Owner: owner})
	if err != nil {
		t.Fatal(err)
	}
	assertFindingCodes(t, findings, "STORAGE_PATH_NOT_PORTABLE")

	owner, root = crossPlatformOwner(t, corestorage.OwnerResource, "demo-token", corestorage.OwnerManifest{}, corestorage.StorageEntry{Name: "data", Path: corestorage.PortablePath{Value: "$USER_DATA_DIR/demo"}})
	findings, err = (crossPlatform{}).Analyze(context.Background(), AnalyzerContext{RepoRoot: root, Owner: owner})
	if err != nil {
		t.Fatal(err)
	}
	assertNoFindingCode(t, findings, "STORAGE_PATH_NOT_PORTABLE")
}

func TestCrossPlatformFlagsPathPlatformMismatch(t *testing.T) {
	owner, root := crossPlatformOwner(t, corestorage.OwnerTool, "windows-only", corestorage.OwnerManifest{Platforms: []corestorage.Platform{corestorage.PlatformWindows}}, corestorage.StorageEntry{Name: "data", Path: corestorage.PortablePath{Value: "/var/lib/demo"}})
	findings, err := (crossPlatform{}).Analyze(context.Background(), AnalyzerContext{RepoRoot: root, Owner: owner})
	if err != nil {
		t.Fatal(err)
	}
	assertFindingCodes(t, findings, "STORAGE_PATH_PLATFORM_MISMATCH")

	owner, root = crossPlatformOwner(t, corestorage.OwnerTool, "windows-correct", corestorage.OwnerManifest{Platforms: []corestorage.Platform{corestorage.PlatformWindows}}, corestorage.StorageEntry{Name: "data", Path: corestorage.PortablePath{Value: "C:/ProgramData/demo"}})
	findings, err = (crossPlatform{}).Analyze(context.Background(), AnalyzerContext{RepoRoot: root, Owner: owner})
	if err != nil {
		t.Fatal(err)
	}
	assertNoFindingCode(t, findings, "STORAGE_PATH_PLATFORM_MISMATCH")
}

func TestCrossPlatformFlagsMissingBranchesButAllowsNull(t *testing.T) {
	linux := "/var/lib/demo"
	owner, root := crossPlatformOwner(t, corestorage.OwnerResource, "missing-branch", corestorage.OwnerManifest{}, corestorage.StorageEntry{Name: "data", Path: corestorage.PortablePath{ByOS: map[corestorage.Platform]*string{corestorage.PlatformLinux: &linux}}})
	findings, err := (crossPlatform{}).Analyze(context.Background(), AnalyzerContext{RepoRoot: root, Owner: owner})
	if err != nil {
		t.Fatal(err)
	}
	assertFindingCodes(t, findings, "STORAGE_PATH_BRANCH_MISSING")

	windows := "C:/ProgramData/demo"
	owner, root = crossPlatformOwner(t, corestorage.OwnerResource, "null-branch", corestorage.OwnerManifest{}, corestorage.StorageEntry{Name: "data", Path: corestorage.PortablePath{ByOS: map[corestorage.Platform]*string{corestorage.PlatformLinux: &linux, corestorage.PlatformMacOS: nil, corestorage.PlatformWindows: &windows}}})
	findings, err = (crossPlatform{}).Analyze(context.Background(), AnalyzerContext{RepoRoot: root, Owner: owner})
	if err != nil {
		t.Fatal(err)
	}
	assertNoFindingCode(t, findings, "STORAGE_PATH_BRANCH_MISSING")
}

func TestCrossPlatformFlagsSupersedableDataBranches(t *testing.T) {
	linux := "$USER_HOME/.local/share/vrooli/demo"
	macos := "$USER_HOME/Library/Application Support/vrooli/demo"
	windows := "$USER_HOME/AppData/Local/vrooli/demo"
	owner, root := crossPlatformOwner(t, corestorage.OwnerResource, "supersedable", corestorage.OwnerManifest{}, corestorage.StorageEntry{Name: "data", Path: corestorage.PortablePath{ByOS: map[corestorage.Platform]*string{corestorage.PlatformLinux: &linux, corestorage.PlatformMacOS: &macos, corestorage.PlatformWindows: &windows}}})
	findings, err := (crossPlatform{}).Analyze(context.Background(), AnalyzerContext{RepoRoot: root, Owner: owner})
	if err != nil {
		t.Fatal(err)
	}
	finding := findingByCode(findings, "STORAGE_TOKEN_SUPERSEDABLE")
	if finding == nil || finding.Remediation == "" {
		t.Fatalf("findings = %#v, want supersedable finding with remediation", findings)
	}
	if finding.Location != "resources/supersedable/resource.json" || finding.Message == "" {
		t.Fatalf("finding = %#v", finding)
	}

	owner, root = crossPlatformOwner(t, corestorage.OwnerResource, "divergent", corestorage.OwnerManifest{}, corestorage.StorageEntry{Name: "data", Path: corestorage.PortablePath{ByOS: map[corestorage.Platform]*string{corestorage.PlatformLinux: &linux, corestorage.PlatformMacOS: &macos, corestorage.PlatformWindows: &windows}}})
	owner.StorageEntries[0].Path.ByOS[corestorage.PlatformWindows] = &linux
	findings, err = (crossPlatform{}).Analyze(context.Background(), AnalyzerContext{RepoRoot: root, Owner: owner})
	if err != nil {
		t.Fatal(err)
	}
	assertNoFindingCode(t, findings, "STORAGE_TOKEN_SUPERSEDABLE")
}

func crossPlatformOwner(t *testing.T, kind corestorage.OwnerKind, id string, owner corestorage.OwnerManifest, entry corestorage.StorageEntry) (*corestorage.OwnerManifest, string) {
	t.Helper()
	root := t.TempDir()
	manifest := filepath.Join(root, "resources", id, "resource.json")
	if kind == corestorage.OwnerTool {
		manifest = filepath.Join(root, "internal", "tools", id, "tool.json")
	}
	if err := os.MkdirAll(filepath.Dir(manifest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifest, []byte(`{"name":"`+id+`"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	owner.Kind, owner.ID, owner.ManifestPath = kind, id, manifest
	owner.StorageEntries = []corestorage.StorageEntry{entry}
	return &owner, root
}

func assertFindingCodes(t *testing.T, findings []Finding, want ...string) {
	t.Helper()
	for _, code := range want {
		if findingByCode(findings, code) == nil {
			t.Fatalf("findings = %#v, want %s", findings, code)
		}
	}
}

func assertNoFindingCode(t *testing.T, findings []Finding, code string) {
	t.Helper()
	if findingByCode(findings, code) != nil {
		t.Fatalf("findings = %#v, do not want %s", findings, code)
	}
}

func findingByCode(findings []Finding, code string) *Finding {
	for i := range findings {
		if findings[i].Code == code {
			return &findings[i]
		}
	}
	return nil
}
