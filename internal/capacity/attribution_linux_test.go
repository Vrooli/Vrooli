//go:build linux

package capacity

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestParseCgroupContainerIDLinuxLayouts(t *testing.T) {
	id := "abc123abc123abc123abc123abc123abc123abc123abc123abc123abc1234567"
	cases := []string{
		"12:devices:/docker/" + id,
		"0::/system.slice/docker-" + id + ".scope",
		"0::/kubepods/besteffort/pod123/" + id,
	}
	for _, in := range cases {
		got, ok := parseCgroupContainerID(in)
		if !ok || got != id {
			t.Errorf("parseCgroupContainerID(%q) = %q,%v want %q,true", in, got, ok, id)
		}
	}
	if _, ok := parseCgroupContainerID("0::/system.slice/sshd.service"); ok {
		t.Error("non-container cgroup should not yield an id")
	}
}

func TestProcCgroupSourceReadsProc(t *testing.T) {
	id := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	root := t.TempDir()
	pidDir := filepath.Join(root, strconv.Itoa(4242))
	if err := os.MkdirAll(pidDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pidDir, "cgroup"), []byte("0::/system.slice/docker-"+id+".scope\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := procCgroupSource{procRoot: root}
	got, ok := src.ContainerID(4242)
	if !ok || got != id {
		t.Errorf("ContainerID(4242) = %q,%v want %q,true", got, ok, id)
	}
	if _, ok := src.ContainerID(9999); ok {
		t.Error("missing pid should not resolve a container")
	}
}
