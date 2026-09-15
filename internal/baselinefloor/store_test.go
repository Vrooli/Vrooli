package baselinefloor

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/testenv"
)

func TestDefaultCacheRoot_HonorsXDG(t *testing.T) {
	testenv.SetIdentityEnv(t, map[string]string{"XDG_CACHE_HOME": "/custom/cache"})
	root, err := DefaultCacheRoot()
	if err != nil {
		t.Fatalf("DefaultCacheRoot: %v", err)
	}
	if root != filepath.Join("/custom/cache", "vrooli") {
		t.Errorf("root = %q, want /custom/cache/vrooli", root)
	}
}

func TestDefaultCacheRoot_IgnoresRelativeXDG(t *testing.T) {
	// A non-absolute XDG_CACHE_HOME is ignored in favor of the home-based path.
	testenv.SetIdentityEnv(t, map[string]string{"XDG_CACHE_HOME": "relative/path", "HOME": "/home/tester"})
	root, err := DefaultCacheRoot()
	if err != nil {
		t.Fatalf("DefaultCacheRoot: %v", err)
	}
	if root != filepath.Join("/home/tester", ".cache", "vrooli") {
		t.Errorf("root = %q, want /home/tester/.cache/vrooli", root)
	}
}

func TestStore_PathShapes(t *testing.T) {
	s := NewStore("/root")
	if got := s.EngagementDir("swarm-manager", "abc"); got != "/root/swarm-manager/baseline-abc" {
		t.Errorf("EngagementDir = %q", got)
	}
	if got := s.RestorePointPath("swarm-manager", "abc"); got != "/root/swarm-manager/baseline-abc/restore-point" {
		t.Errorf("RestorePointPath = %q", got)
	}
	if got := s.ManifestPath("swarm-manager", "abc"); got != "/root/swarm-manager/baseline-abc/engagement.json" {
		t.Errorf("ManifestPath = %q", got)
	}
}

func TestDefaultStore_RootsAtCacheRoot(t *testing.T) {
	testenv.SetIdentityEnv(t, map[string]string{"XDG_CACHE_HOME": "/custom/cache"})
	s, err := DefaultStore()
	if err != nil {
		t.Fatalf("DefaultStore: %v", err)
	}
	if s.Root() != filepath.Join("/custom/cache", "vrooli") {
		t.Errorf("Root = %q, want /custom/cache/vrooli", s.Root())
	}
}

func TestDuration_String(t *testing.T) {
	if got := Duration(90 * time.Minute).String(); got != "1h30m0s" {
		t.Errorf("String = %q, want 1h30m0s", got)
	}
	if got := Duration(0).String(); got != "0s" {
		t.Errorf("zero String = %q, want 0s", got)
	}
}

func TestDuration_JSONRoundTrip(t *testing.T) {
	// String form on the wire.
	d := Duration(90 * time.Minute)
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `"1h30m0s"` {
		t.Errorf("marshal = %s, want \"1h30m0s\"", b)
	}

	// Reads a string...
	var fromStr Duration
	if err := json.Unmarshal([]byte(`"2h"`), &fromStr); err != nil {
		t.Fatal(err)
	}
	if fromStr.AsDuration() != 2*time.Hour {
		t.Errorf("fromStr = %v, want 2h", fromStr.AsDuration())
	}

	// ...a raw nanosecond number...
	var fromNum Duration
	if err := json.Unmarshal([]byte("3600000000000"), &fromNum); err != nil {
		t.Fatal(err)
	}
	if fromNum.AsDuration() != time.Hour {
		t.Errorf("fromNum = %v, want 1h", fromNum.AsDuration())
	}

	// ...and null.
	var fromNull Duration
	if err := json.Unmarshal([]byte("null"), &fromNull); err != nil {
		t.Fatal(err)
	}
	if fromNull != 0 {
		t.Errorf("fromNull = %v, want 0", fromNull)
	}

	// Garbage is an error.
	var bad Duration
	if err := json.Unmarshal([]byte(`"not-a-duration"`), &bad); err == nil {
		t.Error("expected error on invalid duration string")
	}
}
