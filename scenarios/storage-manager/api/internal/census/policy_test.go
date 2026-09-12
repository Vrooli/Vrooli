package census

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	corestorage "github.com/vrooli/api-core/storage"
)

// The census used to fall back to the device mount point when the policy
// left roots empty, which on a single-partition host meant walking every
// inode on the machine. An empty root list must now mean the attribution
// universe and nothing wider.
func TestDefaultPolicyNeverResolvesToTheDeviceRoot(t *testing.T) {
	repo := t.TempDir()
	resolved, err := resolvePolicy(repo, ScanPolicy{}, true, hostFileSystem{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved.roots) == 0 {
		t.Fatal("default policy resolved to no roots")
	}
	home := runtimeHome()
	for _, root := range resolved.roots {
		if root == resolved.deviceRoot {
			t.Fatalf("default policy resolved the device root %q as a scan root: %v", resolved.deviceRoot, resolved.roots)
		}
		if root != filepath.Clean(repo) && !isWithin(root, home) {
			t.Fatalf("default root %q is neither the repository nor under the runtime home %q", root, home)
		}
	}
	if !containsRoot(resolved.roots, repo) {
		t.Fatalf("default roots %v do not include the repository root %q", resolved.roots, repo)
	}
	optional := 0
	for _, root := range resolved.policy.Roots {
		if root.Optional {
			optional++
		}
	}
	if optional == 0 {
		t.Fatalf("runtime home class roots must be optional so an unused class is not a finding: %+v", resolved.policy.Roots)
	}
	if !resolved.policy.prunesName("node_modules") {
		t.Fatalf("default policy must prune node_modules by name: %+v", resolved.policy.Exclusions)
	}
}

func TestExplicitDeviceRootRemainsAnOperatorChoice(t *testing.T) {
	repo := t.TempDir()
	resolved, err := resolvePolicy(repo, ScanPolicy{Roots: []PolicyRoot{{Path: "$DEVICE_ROOT", Reason: "operator wants the whole device"}}}, true, hostFileSystem{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved.roots) != 1 || resolved.roots[0] != resolved.deviceRoot {
		t.Fatalf("explicit $DEVICE_ROOT resolved to %v, want [%s]", resolved.roots, resolved.deviceRoot)
	}
}

func TestOutermostRootsDropsNestedAndDuplicateRoots(t *testing.T) {
	got := outermostRoots([]string{"/a/b", "/a", "/c", "/a/b/c", "/c", "/cd"})
	want := []string{"/a", "/c", "/cd"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("outermostRoots = %v, want %v", got, want)
	}
}

func TestLoadPolicyRejectsAmbiguousExclusions(t *testing.T) {
	repo := t.TempDir()
	path := filepath.Join(repo, "scenarios", "storage-manager", "config", "storage-census-policy.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"both":       `{"exclusions":[{"path":"/x","name":"y","reason":"r"}]}`,
		"neither":    `{"exclusions":[{"reason":"r"}]}`,
		"no reason":  `{"exclusions":[{"name":"node_modules"}]}`,
		"name slash": `{"exclusions":[{"name":"a/b","reason":"r"}]}`,
	} {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadPolicy(repo); err == nil {
			t.Fatalf("%s: expected LoadPolicy to reject %s", name, body)
		}
	}
	if err := os.WriteFile(path, []byte(`{"roots":[],"exclusions":[{"name":"node_modules","reason":"regenerable"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	policy, err := LoadPolicy(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(policy.Roots) != 0 || !policy.prunesName("node_modules") {
		t.Fatalf("policy = %+v", policy)
	}
}

// recordingFileSystem tracks every path the walk visits so a test can prove
// the walker never descends into a tree the policy excluded.
type recordingFileSystem struct {
	treeFileSystem
	visited []string
}

func (r *recordingFileSystem) WalkDir(root string, fn fs.WalkDirFunc) error {
	return r.treeFileSystem.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		r.visited = append(r.visited, path)
		return fn(path, entry, err)
	})
}

func (r *recordingFileSystem) visitedAny(fragment string) bool {
	for _, path := range r.visited {
		if strings.Contains(path, fragment) {
			return true
		}
	}
	return false
}

func TestDeviceScopedWalkStaysInsideThePolicyRootsAndClosesAccounting(t *testing.T) {
	hostRoot, err := filepath.Abs("root")
	if err != nil {
		t.Fatal(err)
	}
	tree := fstest.MapFS{
		"root/repo/owned/a":        &fstest.MapFile{Data: []byte("1234")},
		"root/repo/loose/b":        &fstest.MapFile{Data: []byte("56")},
		"root/elsewhere/system/c":  &fstest.MapFile{Data: []byte("7890123")},
		"root/elsewhere/system/d":  &fstest.MapFile{Data: []byte("4")},
		"root/repo/owned/deeper/e": &fstest.MapFile{Data: []byte("9")},
	}
	filesystem := &recordingFileSystem{treeFileSystem: treeFileSystem{FS: tree, HostRoot: hostRoot}}
	repo := filepath.Join(hostRoot, "repo")
	report, err := scanWithPolicyUsing(scanRequest{
		displayRoot:  repo,
		policy:       ScanPolicy{Roots: []PolicyRoot{{Path: repo, Reason: "test"}}, FloorBytes: 1},
		declarations: []resolvedDeclaration{{owner: "owner", name: "data", path: filepath.Join(repo, "owned"), kind: "dir"}},
		ownerRoots:   []string{filepath.Join(repo, "owned")},
		deviceScoped: true,
	}, filesystem, fakeDeviceProbe{info: DeviceInfo{TotalBytes: 100, AvailableBytes: 40, Privilege: "least-privilege"}})
	if err != nil {
		t.Fatal(err)
	}
	if filesystem.visitedAny("elsewhere") {
		t.Fatalf("walk left the policy roots: %v", filesystem.visited)
	}
	if report.ScanCoverage.ScannedBytes != 7 || report.AttributedBytes != 5 || report.UnattributedBytes != 55 {
		t.Fatalf("scanned=%d attributed=%d unattributed=%d, want 7/5/55", report.ScanCoverage.ScannedBytes, report.AttributedBytes, report.UnattributedBytes)
	}
	if !report.Closed || report.MeasuredBytes != 60 {
		t.Fatalf("device-scoped scan must still close against statfs: %+v", report)
	}
	if len(report.ScanCoverage.ScannedRoots) != 1 || report.ScanCoverage.ScannedRoots[0] != repo {
		t.Fatalf("scanned roots = %v, want [%s]", report.ScanCoverage.ScannedRoots, repo)
	}
	// The 53 device bytes outside the roots are reported once, at the
	// accounting root, and the 2 loose repository bytes at their own bucket.
	var remainder, loose int64
	for _, root := range report.UnattributedRoots {
		switch root.Path {
		case report.Root:
			remainder = root.Bytes
		case filepath.Join(repo, "loose"):
			loose = root.Bytes
		}
	}
	if remainder != 53 || loose != 2 {
		t.Fatalf("unattributed roots = %+v, want 53 at %s and 2 at repo/loose", report.UnattributedRoots, report.Root)
	}
}

func TestNameExclusionPrunesSubtreesAndReportsThem(t *testing.T) {
	hostRoot, err := filepath.Abs("root")
	if err != nil {
		t.Fatal(err)
	}
	tree := fstest.MapFS{
		"root/app/src/main.go":                    &fstest.MapFile{Data: []byte("123")},
		"root/app/node_modules/left-pad/index.js": &fstest.MapFile{Data: []byte("1234567890")},
		"root/app/ui/node_modules/x/y.js":         &fstest.MapFile{Data: []byte("1234567890")},
	}
	filesystem := &recordingFileSystem{treeFileSystem: treeFileSystem{FS: tree, HostRoot: hostRoot}}
	report, err := scanWithPolicyUsing(scanRequest{
		displayRoot: hostRoot,
		policy: ScanPolicy{
			Roots:      []PolicyRoot{{Path: hostRoot}},
			FloorBytes: 1,
			Exclusions: []PolicyExclusion{{Name: "node_modules", Reason: "regenerable"}},
		},
	}, filesystem, fakeDeviceProbe{info: DeviceInfo{TotalBytes: 100, AvailableBytes: 40}})
	if err != nil {
		t.Fatal(err)
	}
	if filesystem.visitedAny("left-pad") || filesystem.visitedAny("y.js") {
		t.Fatalf("walk descended into a pruned tree: %v", filesystem.visited)
	}
	if report.ScanCoverage.ScannedBytes != 3 || report.ScanCoverage.FilesScanned != 1 {
		t.Fatalf("scanned = %d bytes / %d files, want 3 / 1", report.ScanCoverage.ScannedBytes, report.ScanCoverage.FilesScanned)
	}
	if report.ScanCoverage.PrunedDirectories["node_modules"] != 2 {
		t.Fatalf("pruned directories = %v, want node_modules:2", report.ScanCoverage.PrunedDirectories)
	}
}

func TestMissingOptionalRootIsNotAFindingButMissingRequiredRootIs(t *testing.T) {
	hostRoot, err := filepath.Abs("root")
	if err != nil {
		t.Fatal(err)
	}
	tree := fstest.MapFS{"root/present/a": &fstest.MapFile{Data: []byte("1")}}
	report, err := scanWithPolicyUsing(scanRequest{
		displayRoot: hostRoot,
		policy: ScanPolicy{FloorBytes: 1, Roots: []PolicyRoot{
			{Path: filepath.Join(hostRoot, "present")},
			{Path: filepath.Join(hostRoot, "unused-class"), Optional: true},
			{Path: filepath.Join(hostRoot, "declared-but-gone")},
		}},
	}, treeFileSystem{FS: tree, HostRoot: hostRoot}, fakeDeviceProbe{info: DeviceInfo{TotalBytes: 100, AvailableBytes: 40}})
	if err != nil {
		t.Fatal(err)
	}
	var missing []string
	for _, finding := range report.Findings {
		if finding.Code == "missing_scan_root" {
			missing = append(missing, finding.Path)
		}
	}
	if len(missing) != 1 || missing[0] != filepath.Join(hostRoot, "declared-but-gone") {
		t.Fatalf("missing_scan_root findings = %v, want only declared-but-gone", missing)
	}
}

func TestOrphanScenarioBytesComeFromTheMainWalk(t *testing.T) {
	root := t.TempDir()
	write := func(path, value string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(root, "scenarios", "demo", ".vrooli", "service.json"), `{"service":{"name":"demo"}}`)
	write(filepath.Join(root, "scenarios", "demo", "data", "keep.db"), "12")
	write(filepath.Join(root, "scenarios", "orphan", "data", "left.db"), "123456")
	write(filepath.Join(root, "scenarios", "orphan", "ui", "node_modules", "pkg", "index.js"), "1234567890")
	inventory := corestorage.OwnerInventory{RepoRoot: root, Owners: []corestorage.OwnerManifest{
		{Kind: corestorage.OwnerScenario, ID: "demo", ManifestPath: filepath.Join(root, "scenarios", "demo", ".vrooli", "service.json")},
	}}
	report, err := ScanInventoryWithPolicy(root, inventory, ScanPolicy{
		Roots:      []PolicyRoot{{Path: root}},
		FloorBytes: 1,
		Exclusions: []PolicyExclusion{{Name: "node_modules", Reason: "regenerable"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var orphan *Finding
	for index := range report.Findings {
		if report.Findings[index].Code == "STORAGE_PATH_ORPHANED" {
			orphan = &report.Findings[index]
		}
	}
	if orphan == nil {
		t.Fatalf("expected STORAGE_PATH_ORPHANED finding: %+v", report.Findings)
	}
	if orphan.Path != filepath.Join(root, "scenarios", "orphan") || !strings.Contains(orphan.Message, "6 bytes") {
		t.Fatalf("orphan finding = %+v, want scenarios/orphan with 6 bytes (pruned node_modules excluded)", orphan)
	}
	if report.ScanCoverage.PrunedDirectories["node_modules"] != 1 {
		t.Fatalf("pruned = %v", report.ScanCoverage.PrunedDirectories)
	}
}
