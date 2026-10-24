package lifecycle

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/peerrecord"
)

func TestPeerRecordPermissions(t *testing.T) {
	home := t.TempDir()
	record := PeerRecord{
		Scenario:  "authority",
		Instance:  "live",
		Tier:      1,
		OwnerPID:  os.Getpid(),
		StartedAt: time.Now().UTC(),
		Ports:     map[string]int{"api": 18444},
	}
	if err := writePeerRecord(home, record); err != nil {
		t.Fatalf("write peer record: %v", err)
	}
	info, err := os.Stat(filepath.Join(home, ".vrooli", "peers", "authority.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
	if err := removePeerRecord(home, "authority"); err != nil {
		t.Fatalf("remove peer record: %v", err)
	}
}

func TestRemovePeerRecordIsScopedToInstance(t *testing.T) {
	home := t.TempDir()
	for _, record := range []PeerRecord{
		{Scenario: "demo", Instance: "live", OwnerPID: os.Getpid(), Ports: map[string]int{"api": 18080}},
		{Scenario: "demo", Instance: "shadow", OwnerPID: os.Getpid(), Ports: map[string]int{"api": 18081}},
	} {
		if err := writePeerRecord(home, record); err != nil {
			t.Fatal(err)
		}
	}
	if err := removePeerRecord(home, "demo@shadow"); err != nil {
		t.Fatal(err)
	}
	if _, err := peerrecord.Read(home, "demo@shadow"); !os.IsNotExist(err) {
		t.Fatalf("shadow read after removal error = %v, want not-exist", err)
	}
	live, err := peerrecord.Read(home, "demo")
	if err != nil {
		t.Fatalf("live record after shadow removal: %v", err)
	}
	if live.Ports["api"] != 18080 {
		t.Fatalf("live port after shadow removal = %d, want 18080", live.Ports["api"])
	}
}
