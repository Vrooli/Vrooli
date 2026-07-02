package evidence

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func fixedNow() time.Time {
	return time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)
}

func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

// [REQ:BH-EVD-003] Staleness edge cases: no artifacts, missing snapshot
// with runs, stale snapshot, fresh snapshot.
func TestSnapshotStaleness(t *testing.T) {
	t.Run("no artifacts at all", func(t *testing.T) {
		s := NewStore(t.TempDir(), fixedNow)
		st := s.SnapshotStaleness(SyncSnapshot{}, false)
		require.False(t, st.Stale)
		require.False(t, st.SnapshotPresent)
	})
	t.Run("runs exist but no snapshot", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "coverage", "runs", "20260702-100000-abc"), 0o755))
		s := NewStore(dir, fixedNow)
		st := s.SnapshotStaleness(SyncSnapshot{}, false)
		require.True(t, st.Stale)
	})
	t.Run("snapshot predates newest run", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "coverage", "runs", "20260702-110000-abc"), 0o755))
		s := NewStore(dir, fixedNow)
		snap := SyncSnapshot{GeneratedAt: time.Date(2026, 7, 2, 9, 0, 0, 0, time.UTC)}
		st := s.SnapshotStaleness(snap, true)
		require.True(t, st.Stale)
		require.Contains(t, st.Detail, "predates")
	})
	t.Run("fresh snapshot", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "coverage", "runs", "20260702-100000-abc"), 0o755))
		s := NewStore(dir, fixedNow)
		snap := SyncSnapshot{GeneratedAt: time.Date(2026, 7, 2, 10, 30, 0, 0, time.UTC)}
		st := s.SnapshotStaleness(snap, true)
		require.False(t, st.Stale)
	})
}

// [REQ:BH-EVD-001] The ledger append is the one write; read round-trips;
// expiry applies the validity window.
func TestLedgerRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir, fixedNow)

	_, err := s.AppendAttestation("x", "", "agent", "notes")
	require.Error(t, err, "requirement id required")
	_, err = s.AppendAttestation("x", "R-001", "", "notes")
	require.Error(t, err, "attester identity required")

	a, err := s.AppendAttestation("x", "R-001", "agent-1", "checked the flow")
	require.NoError(t, err)
	require.Equal(t, fixedNow(), a.AttestedAt)
	require.Equal(t, fixedNow().Add(DefaultManualValidityWindow), a.ExpiresAt)
	require.False(t, a.Expired(fixedNow()))
	require.True(t, a.Expired(fixedNow().Add(DefaultManualValidityWindow+time.Hour)))

	_, err = s.AppendAttestation("x", "R-001", "agent-2", "re-checked")
	require.NoError(t, err)

	all, err := s.ReadAttestations()
	require.NoError(t, err)
	require.Len(t, all, 2)

	latest, err := s.LatestAttestations()
	require.NoError(t, err)
	require.Len(t, latest, 1)
	require.Equal(t, "agent-2", latest["R-001"].AttestedBy)
}

// [REQ:BH-EVD-001] Snapshot reader tolerates absence and parses the real
// sync shape.
func TestReadSnapshot(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir, fixedNow)
	_, ok, err := s.ReadSnapshot()
	require.NoError(t, err)
	require.False(t, ok)

	writeFile(t, dir, "coverage/requirements-sync/latest.json", `{
		"version":"1.0.0","generated_at":"2026-07-02T10:00:00Z",
		"summary":{"total_requirements":2,"completion_rate":50},
		"operational_targets":[{"id":"OT-P0-001","title":"T","status":"complete","requirement_ids":["R-001"],"completion_rate":100}],
		"modules":[{"name":"01-x","file_path":"/abs/requirements/01-x/module.json","total":2,"complete":1}]
	}`)
	snap, ok, err := s.ReadSnapshot()
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 2, snap.Summary.TotalRequirements)
	require.Len(t, snap.OperationalTargets, 1)
	require.Equal(t, "OT-P0-001", snap.OperationalTargets[0].ID)
}
