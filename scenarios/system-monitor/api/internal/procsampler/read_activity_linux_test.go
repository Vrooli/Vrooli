//go:build linux

package procsampler

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

type deniedIOFS struct{ *fakeProcFS }

type reusedDuringDetailsFS struct {
	*fakeProcFS
	statReads   int
	replacement []byte
}

func (f *reusedDuringDetailsFS) ReadFile(pid int, name string) ([]byte, error) {
	if name == "stat" {
		f.statReads++
		if f.statReads == 3 {
			return f.replacement, nil
		}
	}
	return f.fakeProcFS.ReadFile(pid, name)
}

func (f deniedIOFS) ReadFile(pid int, name string) ([]byte, error) {
	if name == "io" {
		return nil, os.ErrPermission
	}
	return f.fakeProcFS.ReadFile(pid, name)
}

func TestReadActivityLinuxSnapshotRetainsIdentityAndCoverage(t *testing.T) {
	stat := statLine("worker", 1, 0, 0, 1, 1)
	close := strings.LastIndexByte(stat, ')')
	fields := strings.Fields(stat[close+1:])
	fields[19] = "123"
	stat = stat[:close+1] + " " + strings.Join(fields, " ")
	fs := &fakeProcFS{pids: []int{1, 2}, files: map[string][]byte{
		"1/stat": []byte(stat), "1/io": []byte("rchar: 1234\nread_bytes: 5678\n"), "1/cmdline": []byte("worker\x00"),
	}, links: map[string]string{}}
	snapshot, err := sampleReadActivity(context.Background(), fs, func() time.Time { return time.Unix(1, 0) }, 10, true)
	if err != nil || snapshot.enumerated != 2 || snapshot.unreadable != 1 || len(snapshot.points) != 1 {
		t.Fatalf("snapshot=%+v err=%v", snapshot, err)
	}
	point := snapshot.points[1]
	if point.process.StartTimeTicks != 123 || point.requested != 1234 || point.storage != 5678 {
		t.Fatalf("point=%+v", point)
	}
}

func TestReadActivityLinuxRejectsMissingCounters(t *testing.T) {
	if _, _, err := parseReadCounters("rchar: 100\n"); err == nil {
		t.Fatal("missing physical counter accepted")
	}
	if _, _, err := parseReadCounters("rchar: nope\nread_bytes: 4\n"); err == nil {
		t.Fatal("invalid counter accepted")
	}
}

func TestReadActivityLinuxReportsPermissionAndBound(t *testing.T) {
	stat := statLine("worker", 1, 0, 0, 1, 1)
	close := strings.LastIndexByte(stat, ')')
	fields := strings.Fields(stat[close+1:])
	fields[19] = "123"
	stat = stat[:close+1] + " " + strings.Join(fields, " ")
	fs := deniedIOFS{&fakeProcFS{pids: []int{1, 2}, files: map[string][]byte{"1/stat": []byte(stat)}, links: map[string]string{}}}
	snapshot, err := sampleReadActivity(context.Background(), fs, time.Now, 1, false)
	if err != nil || snapshot.enumerated != 2 || !snapshot.truncated || snapshot.unreadable != 1 || snapshot.permissionDenied != 1 {
		t.Fatalf("snapshot=%+v err=%v", snapshot, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := sampleReadActivity(ctx, fs, time.Now, 1, false); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel = %v", err)
	}
}

func TestReadActivityLinuxRejectsIdentityChangeDuringAttribution(t *testing.T) {
	stat := statLine("worker", 1, 0, 0, 1, 1)
	close := strings.LastIndexByte(stat, ')')
	fields := strings.Fields(stat[close+1:])
	fields[19] = "123"
	before := stat[:close+1] + " " + strings.Join(fields, " ")
	fields[19] = "124"
	after := stat[:close+1] + " " + strings.Join(fields, " ")
	fs := &reusedDuringDetailsFS{fakeProcFS: &fakeProcFS{pids: []int{1}, files: map[string][]byte{"1/stat": []byte(before), "1/io": []byte("rchar: 10\nread_bytes: 1\n")}, links: map[string]string{}}, replacement: []byte(after)}
	snapshot, err := sampleReadActivity(context.Background(), fs, time.Now, 10, true)
	if err != nil || snapshot.unreadable != 1 || len(snapshot.points) != 0 {
		t.Fatalf("reused identity accepted: %+v %v", snapshot, err)
	}
}
