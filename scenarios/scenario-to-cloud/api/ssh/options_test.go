package ssh

import (
	"runtime"
	"strings"
	"testing"
)

func TestBuildControlPathStablePerTarget(t *testing.T) {
	t.Parallel()

	cfg := Config{Host: "example.com", Port: 22, User: "root"}
	p1 := buildControlPath(cfg)
	p2 := buildControlPath(cfg)
	if p1 != p2 {
		t.Fatalf("buildControlPath should be stable, got %q vs %q", p1, p2)
	}
	if !strings.Contains(p1, "vrooli-ssh-") {
		t.Fatalf("buildControlPath should include hashed prefix, got %q", p1)
	}
}

func TestBuildControlPathDiffersAcrossTargets(t *testing.T) {
	t.Parallel()

	a := buildControlPath(Config{Host: "one.example.com", Port: 22, User: "root"})
	b := buildControlPath(Config{Host: "two.example.com", Port: 22, User: "root"})
	if a == b {
		t.Fatalf("buildControlPath should differ for different hosts, both were %q", a)
	}
}

func TestDefaultRunOptionsControlMasterByPlatform(t *testing.T) {
	t.Parallel()

	opts := DefaultRunOptions()
	if runtime.GOOS == "windows" && opts.ControlMaster {
		t.Fatal("ControlMaster should be disabled on Windows defaults")
	}
	if runtime.GOOS != "windows" && !opts.ControlMaster {
		t.Fatal("ControlMaster should be enabled on non-Windows defaults")
	}
}
