package baseline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/vrooli/api-core/storage"
)

// IssueSnapshot is a GCT-local snapshot of an auditor scan's findings, stored
// for the structure + rules surfaces (kind = gct-local-snapshot). It is small
// JSON (no artifacts), consistent with Decision 1.
type IssueSnapshot struct {
	Surface    string    `json:"surface"`
	Scenario   string    `json:"scenario"`
	CapturedAt time.Time `json:"captured_at"`
	Issues     []Issue   `json:"issues"`
}

// SnapshotStore persists IssueSnapshots under
// data/<repoID>/baseline-snapshots/<scenario>/<id>.json.
type SnapshotStore struct {
	resolver     *storage.Resolver
	scenarioD    string
	rootOverride string // test seam
}

// NewSnapshotStore builds a SnapshotStore over the given api-core resolver.
func NewSnapshotStore(resolver *storage.Resolver) *SnapshotStore {
	return &SnapshotStore{resolver: resolver, scenarioD: "git-control-tower"}
}

// NewSnapshotStoreAt builds a SnapshotStore with class roots forced under root
// (test seam).
func NewSnapshotStoreAt(resolver *storage.Resolver, root string) *SnapshotStore {
	return &SnapshotStore{resolver: resolver, scenarioD: "git-control-tower", rootOverride: root}
}

func (s *SnapshotStore) dir(repoID int64, scenario string) (string, error) {
	return s.resolver.Path(
		storage.Options{ScenarioID: s.scenarioD, RootOverride: s.rootOverride},
		storage.ClassData,
		fmt.Sprintf("%d/baseline-snapshots/%s", repoID, sanitizeSegment(scenario)),
	)
}

// Save writes a snapshot and returns its generated ID.
func (s *SnapshotStore) Save(repoID int64, snap IssueSnapshot) (string, error) {
	dir, err := s.dir(repoID, snap.Scenario)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create snapshot dir: %w", err)
	}
	id := NewArtifactID()
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal snapshot: %w", err)
	}
	path := filepath.Join(dir, sanitizeSegment(id)+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("write snapshot: %w", err)
	}
	return id, nil
}

// Load reads a snapshot by ID. Returns ErrNotFound if absent.
func (s *SnapshotStore) Load(repoID int64, scenario, id string) (IssueSnapshot, error) {
	dir, err := s.dir(repoID, scenario)
	if err != nil {
		return IssueSnapshot{}, err
	}
	data, rerr := os.ReadFile(filepath.Join(dir, sanitizeSegment(id)+".json"))
	if rerr != nil {
		if os.IsNotExist(rerr) {
			return IssueSnapshot{}, ErrNotFound
		}
		return IssueSnapshot{}, fmt.Errorf("read snapshot: %w", rerr)
	}
	var snap IssueSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return IssueSnapshot{}, fmt.Errorf("parse snapshot: %w", err)
	}
	return snap, nil
}

// Delete removes a snapshot by ID. Missing snapshots are a no-op.
func (s *SnapshotStore) Delete(repoID int64, scenario, id string) error {
	dir, err := s.dir(repoID, scenario)
	if err != nil {
		return err
	}
	err = os.Remove(filepath.Join(dir, sanitizeSegment(id)+".json"))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
