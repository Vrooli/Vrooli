package census

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

type fakeDeviceProbe struct {
	info DeviceInfo
	err  error
}

func (p fakeDeviceProbe) Probe(string) (DeviceInfo, error) { return p.info, p.err }

func TestScanWithFileSystemAttributesSyntheticTreeCompletely(t *testing.T) {
	tree := fstest.MapFS{
		"root/owned/a":   &fstest.MapFile{Data: []byte("1234")},
		"root/owned/b":   &fstest.MapFile{Data: []byte("56")},
		"root/unowned/c": &fstest.MapFile{Data: []byte("789")},
	}
	hostRoot, err := filepath.Abs("root")
	if err != nil {
		t.Fatal(err)
	}
	report, err := ScanWithFileSystem("root", map[string][]Declaration{
		"owner": {{Name: "data", Path: "root/owned"}},
	}, treeFileSystem{FS: tree, HostRoot: hostRoot}, fakeDeviceProbe{info: DeviceInfo{TotalBytes: 9, AvailableBytes: 0, Privilege: "privileged"}})
	if err != nil {
		t.Fatal(err)
	}
	if report.MeasuredBytes != 9 || report.AttributedBytes != 6 || report.UnattributedBytes != 3 || !report.Closed || !report.AccountingIdentity {
		t.Fatalf("synthetic report = %+v", report)
	}
}

func TestDeviceProbeCoverageHasNamedPlatformVerdicts(t *testing.T) {
	for _, tier := range []struct {
		name string
		info DeviceInfo
		err  error
		want string
	}{
		{name: "linux privileged", info: DeviceInfo{TotalBytes: 100, AvailableBytes: 40, Privilege: "privileged"}},
		{name: "linux least privilege", info: DeviceInfo{TotalBytes: 100, AvailableBytes: 40, Privilege: "least-privilege"}},
		{name: "macos", info: DeviceInfo{TotalBytes: 100, AvailableBytes: 40, Privilege: "least-privilege"}},
		{name: "windows degraded", err: fs.ErrNotExist, want: "device_probe_unavailable"},
	} {
		t.Run(tier.name, func(t *testing.T) {
			coverage := deviceCoverageWith(fakeDeviceProbe{info: tier.info, err: tier.err}, "/root", 60, true)
			if tier.err == nil {
				if !coverage.MeasuredByDevice || coverage.DeviceTotalBytes != 100 || coverage.PrivilegeLevel == "" {
					t.Fatalf("coverage = %+v", coverage)
				}
				return
			}
			if coverage.MeasuredByDevice || coverage.DegradedReason == "" || coverage.Complete {
				t.Fatalf("degraded coverage = %+v", coverage)
			}
			if !strings.HasPrefix(coverage.DegradedReason, tier.want) {
				t.Fatalf("reason = %q", coverage.DegradedReason)
			}
		})
	}
}

func TestCensusPlatformTiersUseTheSameSyntheticAccountingContract(t *testing.T) {
	for _, tier := range []struct {
		name      string
		probe     DeviceProbe
		wantFull  bool
		wantLevel string
	}{
		{name: "linux privileged", probe: fakeDeviceProbe{info: DeviceInfo{TotalBytes: 100, AvailableBytes: 40, Privilege: "privileged"}}, wantFull: true, wantLevel: "privileged"},
		{name: "linux least privilege", probe: fakeDeviceProbe{info: DeviceInfo{TotalBytes: 100, AvailableBytes: 40, Privilege: "least-privilege"}}, wantFull: true, wantLevel: "least-privilege"},
		{name: "macos least privilege", probe: fakeDeviceProbe{info: DeviceInfo{TotalBytes: 100, AvailableBytes: 40, Privilege: "least-privilege"}}, wantFull: true, wantLevel: "least-privilege"},
		{name: "windows degraded", probe: fakeDeviceProbe{err: fs.ErrNotExist}, wantFull: false},
	} {
		t.Run(tier.name, func(t *testing.T) {
			tree := fstest.MapFS{"root/owned/a": &fstest.MapFile{Data: []byte("1234")}, "root/unowned/b": &fstest.MapFile{Data: []byte("56")}}
			hostRoot, err := filepath.Abs("root")
			if err != nil {
				t.Fatal(err)
			}
			report, err := scanWithPolicyUsing(hostRoot, ScanPolicy{Roots: []PolicyRoot{{Path: hostRoot}}, FloorBytes: 1}, []resolvedDeclaration{{owner: "owner", name: "data", path: filepath.Join(hostRoot, "owned"), kind: "dir"}}, []string{filepath.Join(hostRoot, "owned")}, nil, true, treeFileSystem{FS: tree, HostRoot: hostRoot}, tier.probe)
			if err != nil {
				t.Fatal(err)
			}
			if report.Closed != tier.wantFull || report.ScanCoverage.PrivilegeLevel != tier.wantLevel {
				t.Fatalf("tier report = %+v", report)
			}
			if tier.wantFull && !report.ScanCoverage.MeasuredByDevice {
				t.Fatalf("tier did not use device denominator: %+v", report.ScanCoverage)
			}
			if !tier.wantFull && report.ScanCoverage.DegradedReason == "" {
				t.Fatalf("degraded tier has no named reason: %+v", report.ScanCoverage)
			}
		})
	}
}

func TestIncrementalScannerReusesUnchangedTreeAndInvalidates(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "item"), []byte("value"), 0o644); err != nil {
		t.Fatal(err)
	}
	filesystem := &countingFileSystem{}
	scanner := NewIncrementalScanner(filesystem, nil)
	manifests := map[string][]Declaration{"owner": {{Name: "data", Path: root}}}
	if _, err := scanner.Scan(root, manifests); err != nil {
		t.Fatal(err)
	}
	if _, err := scanner.Scan(root, manifests); err != nil {
		t.Fatal(err)
	}
	if filesystem.walks != 1 {
		t.Fatalf("unchanged tree walks = %d, want 1", filesystem.walks)
	}
	scanner.Invalidate()
	if _, err := scanner.Scan(root, manifests); err != nil {
		t.Fatal(err)
	}
	if filesystem.walks != 2 {
		t.Fatalf("invalidated tree walks = %d, want 2", filesystem.walks)
	}
}

// treeFileSystem adapts fstest.MapFS to the census FileSystem contract while
// preserving the test's guarantee that no os path is visited.
type treeFileSystem struct {
	FS       fstest.MapFS
	HostRoot string
}

type countingFileSystem struct {
	hostFileSystem
	walks int
}

func (f *countingFileSystem) WalkDir(root string, fn fs.WalkDirFunc) error {
	f.walks++
	return f.hostFileSystem.WalkDir(root, fn)
}

func (t treeFileSystem) WalkDir(root string, fn fs.WalkDirFunc) error {
	return fs.WalkDir(t.FS, t.virtual(root), func(path string, entry fs.DirEntry, err error) error {
		if path == "root" {
			path = t.HostRoot
		} else {
			path = filepath.Join(t.HostRoot, strings.TrimPrefix(path, "root/"))
		}
		return fn(path, entry, err)
	})
}

func (t treeFileSystem) Stat(path string) (os.FileInfo, error) {
	return fs.Stat(t.FS, t.virtual(path))
}

func (t treeFileSystem) Lstat(path string) (os.FileInfo, error) {
	return fs.Stat(t.FS, t.virtual(path))
}

func (t treeFileSystem) virtual(path string) string {
	if path == t.HostRoot {
		return "root"
	}
	rel, err := filepath.Rel(t.HostRoot, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return path
	}
	return filepath.ToSlash(filepath.Join("root", rel))
}
