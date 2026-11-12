//go:build linux

package capacity

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDiskJobDockerResolverVerifiesDaemonPathAndHostFilesystem(t *testing.T) {
	dir := t.TempDir()
	protected := filepath.Join(dir, "locked", "volume")
	if err := os.MkdirAll(protected, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Dir(protected), 0000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(filepath.Dir(protected), 0700)
	probe := filepath.Join(dir, "probe")
	if err := os.Mkdir(probe, 0700); err != nil {
		t.Fatal(err)
	}
	device, _, err := filesystemFromMountInfo(protected)
	if err != nil {
		t.Fatal(err)
	}
	mount := struct {
		Source      string `json:"Source"`
		Destination string `json:"Destination"`
		RW          bool   `json:"RW"`
	}{protected, "/var/lib/data", true}
	record := map[string]any{"State": map[string]bool{"Running": true}, "Mounts": []any{mount}, "GraphDriver": map[string]any{"Data": map[string]string{"UpperDir": protected}}}
	raw, err := json.Marshal([]any{record})
	if err != nil {
		t.Fatal(err)
	}
	fakeBin := filepath.Join(dir, "bin")
	if err := os.Mkdir(fakeBin, 0700); err != nil {
		t.Fatal(err)
	}
	docker := filepath.Join(fakeBin, "docker")
	script := "#!/bin/sh\nprintf '%s' '" + string(raw) + "'\n"
	if err := os.WriteFile(docker, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	originalPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", fakeBin+string(os.PathListSeparator)+originalPath); err != nil {
		t.Fatal(err)
	}
	defer os.Setenv("PATH", originalPath)
	canonical, fsid, free, probePath, err := resolveDiskJobPath(context.Background(), DiskJobPath{Path: protected, Role: "container-volume", PeakBytes: 10, DockerContainer: "m5", DockerDestination: "/var/lib/data"})
	if err != nil {
		t.Fatal(err)
	}
	if canonical != protected || fsid != device || free <= 0 || probePath == protected {
		t.Fatalf("resolved docker path=(%s,%s,%d,%s), expected exact daemon path/device and accessible probe", canonical, fsid, free, probePath)
	}
	if upper, upperFS, _, _, err := resolveDiskJobPath(context.Background(), DiskJobPath{Path: protected, Role: "upper", PeakBytes: 10, DockerContainer: "m5", DockerUpperDir: true}); err != nil || upper != protected || upperFS != device {
		t.Fatalf("resolve Docker upperdir=(%s,%s), error=%v", upper, upperFS, err)
	}
	if _, _, _, _, err := resolveDiskJobPath(context.Background(), DiskJobPath{Path: filepath.Join(dir, "other"), Role: "container-volume", PeakBytes: 10, DockerContainer: "m5", DockerDestination: "/var/lib/data"}); err == nil {
		t.Fatal("accepted caller path that differs from Docker daemon mount source")
	}
	zeroFloor := int64(0)
	store, err := NewSQLiteStore(context.Background(), Config{DBPath: filepath.Join(dir, "capacity.db"), MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	lease, token, err := store.ReserveDiskJob(context.Background(), DiskJobRequest{OwnerID: "docker-owner", JobID: "docker-volume", Paths: []DiskJobPath{{Path: protected, Role: "volume", PeakBytes: 4096, DockerContainer: "m5", DockerDestination: "/var/lib/data"}}})
	if err != nil {
		t.Fatalf("reserve Docker-resolved inaccessible path: %v", err)
	}
	receipt, err := store.SampleDiskJobPath(context.Background(), lease.ID, token, lease.ProcessID, lease.ProcessStart, protected, "m5", "/var/lib/data", false, false, 1234, 0)
	if err != nil {
		t.Fatalf("sample Docker-resolved inaccessible path: %v", err)
	}
	if len(receipt.Paths) != 1 || receipt.Paths[0].FilesystemID != device || len(receipt.Paths[0].Samples) != 1 || receipt.Paths[0].Samples[0].BaselineBytes != 1234 {
		t.Fatalf("Docker path receipt=%+v", receipt.Paths)
	}
}

func TestDiskJobDockerRootResolverSupportsPreProvisionReservation(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "docker-root")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	device, _, err := filesystemFromMountInfo(root)
	if err != nil {
		t.Fatal(err)
	}
	fakeBin := filepath.Join(dir, "bin")
	if err := os.Mkdir(fakeBin, 0700); err != nil {
		t.Fatal(err)
	}
	docker := filepath.Join(fakeBin, "docker")
	script := "#!/bin/sh\n[ \"$1\" = info ] || exit 2\n[ \"$2\" = --format ] || exit 3\nprintf '%s\\n' '" + root + "'\n"
	if err := os.WriteFile(docker, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	originalPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", fakeBin+string(os.PathListSeparator)+originalPath); err != nil {
		t.Fatal(err)
	}
	defer os.Setenv("PATH", originalPath)

	canonical, fsid, free, probe, err := resolveDiskJobPath(context.Background(), DiskJobPath{Path: root, Role: "pre-provision-docker-root-envelope", PeakBytes: 4096, DockerRootDir: true})
	if err != nil {
		t.Fatalf("resolve Docker root: %v", err)
	}
	if canonical != root || fsid != device || free <= 0 || probe == root {
		t.Fatalf("resolved root=(%s,%s,%d,%s), want daemon root, mount-derived device, free space and accessible probe", canonical, fsid, free, probe)
	}
	if _, _, _, _, err := resolveDiskJobPath(context.Background(), DiskJobPath{Path: filepath.Join(dir, "wrong-root"), Role: "pre-provision", PeakBytes: 1, DockerRootDir: true}); err == nil {
		t.Fatal("accepted caller root that differs from Docker daemon root")
	}
	if _, _, _, _, err := resolveDiskJobPath(context.Background(), DiskJobPath{Path: root, DockerContainer: "not-yet-created", DockerRootDir: true, PeakBytes: 1}); err == nil {
		t.Fatal("accepted conflicting Docker root and container selectors")
	}
	rootAlias := filepath.Join(dir, "docker-root-alias")
	if err := os.Symlink(root, rootAlias); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(docker, []byte(strings.Replace(script, root, rootAlias, 1)), 0700); err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, err := resolveDiskJobPath(context.Background(), DiskJobPath{Path: rootAlias, Role: "pre-provision", PeakBytes: 1, DockerRootDir: true}); err == nil {
		t.Fatal("accepted a symlink as the Docker daemon storage root")
	}
	oldTimeout := dockerDiskIdentityTimeout
	defer func() { dockerDiskIdentityTimeout = oldTimeout }()
	dockerDiskIdentityTimeout = 20 * time.Millisecond
	if err := os.WriteFile(docker, []byte("#!/bin/sh\nexec sleep 2\n"), 0700); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	if _, _, _, _, err := resolveDiskJobPath(context.Background(), DiskJobPath{Path: root, Role: "pre-provision", PeakBytes: 1, DockerRootDir: true}); err == nil || time.Since(started) > time.Second {
		t.Fatalf("Docker daemon query did not honor bounded identity timeout: err=%v elapsed=%s", err, time.Since(started))
	}
	if err := os.WriteFile(docker, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	zeroFloor := int64(0)
	store, err := NewSQLiteStore(context.Background(), Config{DBPath: filepath.Join(dir, "capacity.db"), MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	lease, token, err := store.ReserveDiskJob(context.Background(), DiskJobRequest{OwnerID: "pre-provision-owner", JobID: "docker-root-envelope", Paths: []DiskJobPath{{Path: root, Role: "pre-provision-docker-root-envelope", PeakBytes: 4096, DockerRootDir: true}}})
	if err != nil {
		t.Fatalf("reserve Docker root before container creation: %v", err)
	}
	receipt, err := store.SampleDiskJobPath(context.Background(), lease.ID, token, lease.ProcessID, lease.ProcessStart, root, "", "", false, true, 1234, 0)
	if err != nil {
		t.Fatalf("sample pre-provision Docker root: %v", err)
	}
	if len(receipt.Paths) != 1 || receipt.Paths[0].Path != root || receipt.Paths[0].FilesystemID != device || len(receipt.Paths[0].Samples) != 1 || receipt.Paths[0].Samples[0].BaselineBytes != 1234 {
		t.Fatalf("pre-provision Docker root receipt=%+v", receipt.Paths)
	}
}
