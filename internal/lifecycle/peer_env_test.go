package lifecycle

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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
