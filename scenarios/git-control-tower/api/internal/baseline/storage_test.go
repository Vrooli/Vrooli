package baseline

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/storage"
)

func newTestStorage(t *testing.T) *Storage {
	t.Helper()
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli"})
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	return NewStorageAt(resolver, t.TempDir())
}

func sampleManifest(name, branch string) BaselineManifest {
	return BaselineManifest{
		Name:      name,
		Scenario:  "foo",
		Branch:    branch,
		CreatedAt: time.Now().UTC(),
		Run: RunAnchor{
			RunID: "run-1", CaptureProfile: CaptureProfile, TreeDigest: "td:abc", PhaseSetDigest: "ps:abc",
			DescriptorSnapshotRef: "test-genie-run:run-1#descriptor-snapshot", DescriptorSnapshotDigest: "ds:abc", DescriptorSnapshotSchemaVersion: 1,
		},
		SchemaVersion: SchemaVersion,
	}
}

func TestStorageSaveLoadRoundTrip(t *testing.T) {
	s := newTestStorage(t)
	m := sampleManifest("plan-1", "agi")
	if err := s.Save(1, m, CreateOnly); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load(1, "foo", "agi", "plan-1")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Name != "plan-1" || got.RunID() != "run-1" {
		t.Fatalf("unexpected loaded manifest: %+v", got)
	}
}

func writeRawManifest(t *testing.T, s *Storage, branch, name string, value any) string {
	t.Helper()
	dir, err := s.branchDir(1, "foo", branch)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func legacyFixture(name string, refs ...string) map[string]any {
	ids := []string{"structure", "rules", "tests", "visuals", "workflows"}
	surfaces := map[string]any{}
	for i, id := range ids {
		ref := "run-legacy"
		if i < len(refs) {
			ref = refs[i]
		}
		surfaces[id] = map[string]any{"surface_id": id, "kind": "test-genie-run", "ref": ref, "captured_at": "2026-07-10T12:00:00Z"}
	}
	return map[string]any{
		"name": name, "scenario": "foo", "branch": "agi", "created_at": "2026-07-10T12:00:00Z",
		"git": map[string]any{"sha": "abc", "branch": "agi"}, "surfaces": surfaces, "schema_version": 1,
	}
}

func TestStorageMigratesSameRunV1Once(t *testing.T) { // [REQ:GCT-BASELINE-V2-P0]
	s := newTestStorage(t)
	path := writeRawManifest(t, s, "agi", "legacy", legacyFixture("legacy"))
	first, err := s.Load(1, "foo", "agi", "legacy")
	if err != nil {
		t.Fatalf("Load legacy: %v", err)
	}
	if first.SchemaVersion != SchemaVersion || first.RunID() != "run-legacy" || first.Migration == nil {
		t.Fatalf("migration result = %+v", first)
	}
	dataAfterFirst, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(dataAfterFirst), `"surfaces"`) {
		t.Fatalf("V1 surface map survived rewrite: %s", dataAfterFirst)
	}
	second, err := s.Load(1, "foo", "agi", "legacy")
	if err != nil {
		t.Fatalf("second Load: %v", err)
	}
	dataAfterSecond, _ := os.ReadFile(path)
	if string(dataAfterFirst) != string(dataAfterSecond) {
		t.Fatal("second read changed the already-V2 manifest")
	}
	if !second.Migration.MigratedAt.Equal(first.Migration.MigratedAt) {
		t.Fatal("migration timestamp changed on second read")
	}
}

func TestStorageRejectsMixedAndIncompleteV1(t *testing.T) { // [REQ:GCT-BASELINE-V2-P0]
	t.Run("mixed", func(t *testing.T) {
		s := newTestStorage(t)
		writeRawManifest(t, s, "agi", "mixed", legacyFixture("mixed", "run-a", "run-b"))
		_, err := s.Load(1, "foo", "agi", "mixed")
		if !errors.Is(err, ErrLegacyMixedRuns) || !strings.Contains(err.Error(), "recapture") {
			t.Fatalf("mixed error = %v", err)
		}
	})
	t.Run("partial", func(t *testing.T) {
		s := newTestStorage(t)
		fixture := legacyFixture("partial")
		delete(fixture["surfaces"].(map[string]any), "visuals")
		writeRawManifest(t, s, "agi", "partial", fixture)
		_, err := s.Load(1, "foo", "agi", "partial")
		if !errors.Is(err, ErrLegacyIncomplete) || !strings.Contains(err.Error(), "recapture") {
			t.Fatalf("partial error = %v", err)
		}
	})
	t.Run("empty", func(t *testing.T) {
		s := newTestStorage(t)
		fixture := legacyFixture("empty")
		fixture["surfaces"] = map[string]any{}
		writeRawManifest(t, s, "agi", "empty", fixture)
		_, err := s.Load(1, "foo", "agi", "empty")
		if !errors.Is(err, ErrLegacyIncomplete) {
			t.Fatalf("empty error = %v", err)
		}
	})
	t.Run("skipped", func(t *testing.T) {
		s := newTestStorage(t)
		fixture := legacyFixture("skipped")
		fixture["skipped"] = map[string]string{"visuals": "provider unavailable"}
		writeRawManifest(t, s, "agi", "skipped", fixture)
		_, err := s.Load(1, "foo", "agi", "skipped")
		if !errors.Is(err, ErrLegacyIncomplete) {
			t.Fatalf("skipped error = %v", err)
		}
	})
}

func TestStorageRejectsCorruptAndAcceptsAlreadyV2(t *testing.T) { // [REQ:GCT-BASELINE-V2-P0]
	s := newTestStorage(t)
	dir, err := s.branchDir(1, "foo", "agi")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "corrupt.json"), []byte(`{"schema_version":`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load(1, "foo", "agi", "corrupt"); err == nil || !strings.Contains(err.Error(), "decode baseline schema") {
		t.Fatalf("corrupt error = %v", err)
	}

	want := sampleManifest("already-v2", "agi")
	path := writeRawManifest(t, s, "agi", "already-v2", want)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.Load(1, "foo", "agi", "already-v2")
	if err != nil || got.Migration != nil {
		t.Fatalf("already V2 load = %+v, %v", got, err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("reading an already-V2 manifest rewrote it")
	}
}

func TestStorageConcurrentLegacyMigrationAndDeleteLeavesNoPartialManifest(t *testing.T) { // [REQ:GCT-BASELINE-V2-P0]
	s := newTestStorage(t)
	writeRawManifest(t, s, "agi", "racing", legacyFixture("racing"))
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		_, err := s.Load(1, "foo", "agi", "racing")
		if err != nil && !errors.Is(err, ErrNotFound) {
			t.Errorf("concurrent migration: %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		if err := s.Delete(1, "foo", "agi", "racing"); err != nil && !errors.Is(err, ErrNotFound) {
			t.Errorf("concurrent delete: %v", err)
		}
	}()
	close(start)
	wg.Wait()
	if _, err := s.Load(1, "foo", "agi", "racing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("racing manifest survived delete or became corrupt: %v", err)
	}
}

func TestStorageBranchScopingIsolation(t *testing.T) {
	s := newTestStorage(t)
	if err := s.Save(1, sampleManifest("plan", "agi"), CreateOnly); err != nil {
		t.Fatalf("save agi: %v", err)
	}
	if err := s.Save(1, sampleManifest("plan", "master"), CreateOnly); err != nil {
		t.Fatalf("save master: %v", err)
	}
	if _, err := s.Load(1, "foo", "agi", "plan"); err != nil {
		t.Fatalf("load agi: %v", err)
	}
	if _, err := s.Load(1, "foo", "master", "plan"); err != nil {
		t.Fatalf("load master: %v", err)
	}
	all, err := s.List(1, "foo", "")
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 baselines across branches, got %d", len(all))
	}
}

func TestStorageSlashBranchSanitized(t *testing.T) {
	s := newTestStorage(t)
	m := sampleManifest("plan", "feature/login")
	if err := s.Save(1, m, CreateOnly); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := s.Load(1, "foo", "feature/login", "plan"); err != nil {
		t.Fatalf("load with slash branch: %v", err)
	}
}

func TestStorageCreateOnlyRejectsDuplicate(t *testing.T) {
	s := newTestStorage(t)
	if err := s.Save(1, sampleManifest("plan", "agi"), CreateOnly); err != nil {
		t.Fatalf("first save: %v", err)
	}
	err := s.Save(1, sampleManifest("plan", "agi"), CreateOnly)
	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
	// Overwrite mode replaces it.
	if err := s.Save(1, sampleManifest("plan", "agi"), Overwrite); err != nil {
		t.Fatalf("overwrite: %v", err)
	}
}

func TestStorageConcurrentSameNameOneWinner(t *testing.T) {
	s := newTestStorage(t)
	const n = 50
	var wg sync.WaitGroup
	var mu sync.Mutex
	wins, exists := 0, 0
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := s.Save(1, sampleManifest("plan-7c3", "agi"), CreateOnly)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				wins++
			case errors.Is(err, ErrAlreadyExists):
				exists++
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()
	if wins != 1 {
		t.Fatalf("expected exactly 1 winner, got %d (exists=%d)", wins, exists)
	}
	if exists != n-1 {
		t.Fatalf("expected %d ErrAlreadyExists, got %d", n-1, exists)
	}
}

func TestStorageConcurrentDistinctNames(t *testing.T) {
	s := newTestStorage(t)
	const n = 50
	var wg sync.WaitGroup
	errCh := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errCh <- s.Save(1, sampleManifest(fmt.Sprintf("plan-%d", i), "agi"), CreateOnly)
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("distinct save failed: %v", err)
		}
	}
	all, err := s.List(1, "foo", "agi")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != n {
		t.Fatalf("expected %d baselines, got %d", n, len(all))
	}
}

func TestStorageDeleteAndNotFound(t *testing.T) {
	s := newTestStorage(t)
	if err := s.Delete(1, "foo", "agi", "ghost"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete missing: expected ErrNotFound, got %v", err)
	}
	if _, err := s.Load(1, "foo", "agi", "ghost"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("load missing: expected ErrNotFound, got %v", err)
	}
	if err := s.Save(1, sampleManifest("plan", "agi"), CreateOnly); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := s.Delete(1, "foo", "agi", "plan"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.Load(1, "foo", "agi", "plan"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("load after delete: expected ErrNotFound, got %v", err)
	}
}
