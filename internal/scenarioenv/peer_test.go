package scenarioenv

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPeerRecordPermissionsAndStaleness(t *testing.T) {
	home := t.TempDir()
	record := PeerRecord{
		Scenario:  "authority",
		Instance:  "live",
		Tier:      1,
		OwnerPID:  os.Getpid(),
		StartedAt: time.Now().UTC(),
		Ports:     map[string]int{"api": 18444},
	}
	if err := Write(home, record); err != nil {
		t.Fatalf("Write: %v", err)
	}
	info, err := os.Stat(filepath.Join(home, ".vrooli", "peers", "authority.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
	if _, err := Read(home, "authority"); err != nil {
		t.Fatalf("Read live: %v", err)
	}
	record.OwnerPID = 1 << 30
	if err := Write(home, record); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(home, "authority"); !os.IsNotExist(err) {
		t.Fatalf("Read stale error = %v", err)
	}
	if err := Remove(home, "authority"); err != nil {
		t.Fatal(err)
	}
}
