package supervision

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
)

func TestResourceProcessSourceProjectsCompanionPIDRecords(t *testing.T) {
	home := t.TempDir()
	processRoot, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyProcesses)
	if err != nil {
		t.Fatal(err)
	}
	resourceDir := filepath.Join(processRoot, "resources", "whisper")
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	pidPath := filepath.Join(resourceDir, "activity-edge.pid")
	if err := os.WriteFile(pidPath, []byte("4242\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writtenAt := time.Date(2026, 8, 29, 3, 0, 0, 0, time.UTC)
	if err := os.Chtimes(pidPath, writtenAt, writtenAt); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(resourceDir, "malformed.pid"), []byte("not-a-pid"), 0o644); err != nil {
		t.Fatal(err)
	}

	owners, err := NewResourceProcessSource(home).Owners()
	if err != nil {
		t.Fatal(err)
	}
	if len(owners) != 1 {
		t.Fatalf("owners = %#v, want one valid companion owner", owners)
	}
	owner := owners[0]
	if owner.Kind != OwnerKindResource || owner.Name != "whisper/activity-edge" || owner.PID != 4242 || !owner.StartedAt.Equal(writtenAt) {
		t.Fatalf("owner = %#v", owner)
	}
}

func TestBuildHostIndexIncludesResourceProcessSource(t *testing.T) {
	// BuildHostIndex's composition is safety-critical: a resource companion
	// record omitted here is later eligible for destructive orphan cleanup.
	sources := hostOwnershipSources(t.TempDir())
	if len(sources) != 3 {
		t.Fatalf("host ownership source count = %d, want managed service, resource process, and scenario sources", len(sources))
	}
}
