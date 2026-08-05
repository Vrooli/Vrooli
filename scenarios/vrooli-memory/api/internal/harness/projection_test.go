package harness

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	"vrooli-memory/internal/recall"
	localdb "vrooli-memory/internal/testutil/db"
)

type projectionSource []recall.Node

func (s projectionSource) Nodes(context.Context) ([]recall.Node, error) { return s, nil }

func TestProjectionIsPinFirstDryRunAndDoesNotWrite(t *testing.T) { // [REQ:VMEM-P0-010]
	db := localdb.NewSQLite(t)
	_, err := db.Exec(Schema())
	require.NoError(t, err)
	now := time.Now()
	wake := recall.NewService(projectionSource{{ID: "frontier", Text: "later context", Frontier: true, CreatedAt: now.Add(time.Second)}, {ID: "pin", Text: "never forget", Pinned: true, CreatedAt: now}}, nil, recall.Config{WakeBudget: 40})
	p := NewProjector(db, wake)
	p.targets["claude-code"] = projectionTarget{Runtime: "claude-code", Path: t.TempDir() + "/MEMORY.md", Cap: 1024}
	r, err := p.Project(context.Background(), "claude-code", true)
	require.NoError(t, err)
	require.True(t, r.DryRun)
	require.Contains(t, r.Content, generatedHeader)
	require.Less(t, strings.Index(r.Content, "never forget"), strings.Index(r.Content, "later context"))
	var count int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM harness_projections`).Scan(&count))
	require.Zero(t, count)
}

func TestProjectionRefusesToDropPinnedMemory(t *testing.T) {
	db := localdb.NewSQLite(t)
	_, err := db.Exec(Schema())
	require.NoError(t, err)
	wake := recall.NewService(projectionSource{{ID: "pin", Text: "this pin cannot fit", Pinned: true, CreatedAt: time.Now()}}, nil, recall.Config{WakeBudget: 40})
	p := NewProjector(db, wake)
	p.targets["claude-code"] = projectionTarget{Runtime: "claude-code", Path: t.TempDir() + "/MEMORY.md", Cap: len(generatedHeader) + 10}
	_, err = p.Project(context.Background(), "claude-code", true)
	require.ErrorContains(t, err, "pinned memory exceeds")
}

func TestProjectionWritesToLeasedDataRootInTestMode(t *testing.T) {
	db := localdb.NewSQLite(t)
	_, err := db.Exec(Schema())
	require.NoError(t, err)

	routes := filerouting.New(storage.Paths{DataDir: filepath.Join(t.TempDir(), "primary-data")})
	leased, err := routes.InstallLeasedTestRoots("projection-test", time.Minute, true)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, routes.ClearTestRoots("projection-test")) })

	wake := recall.NewService(projectionSource{{ID: "entry", Text: "leased projection", Pinned: true, CreatedAt: time.Now()}}, nil, recall.Config{WakeBudget: 40})
	p := NewProjector(db, wake, routes)
	p.targets["claude-code"] = projectionTarget{Runtime: "claude-code", Path: filepath.Join(t.TempDir(), "MEMORY.md"), Cap: 1024}

	result, err := p.Project(database.WithTestMode(context.Background()), "claude-code", false)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(leased.DataDir, "harness-projections", "claude-code", "MEMORY.md"), result.Path)
	content, err := os.ReadFile(result.Path)
	require.NoError(t, err)
	require.Contains(t, string(content), "leased projection")
	require.Equal(t, int64(1), routes.LeaseStats().TestRootWrites)
	require.Zero(t, routes.LeaseStats().PrimaryWritesDuringTestMode)
}

func TestProjectionPreservesBytesOutsideManagedWakeBlock(t *testing.T) { // [REQ:VMEM-P1-007] [REQ:VMEM-P0-010]
	db := localdb.NewSQLite(t)
	_, err := db.Exec(Schema())
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "MEMORY.md")
	curated := "# Curated\n\nKeep this pointer exactly.\n"
	require.NoError(t, os.WriteFile(path, []byte(curated+wakeStart+"\nold\n"+wakeEnd+"\n## Tail\n"), 0o600))
	wake := recall.NewService(projectionSource{{ID: "entry", Text: "new generated memory", Frontier: true, CreatedAt: time.Now()}}, nil, recall.Config{WakeBudget: 40})
	p := NewProjector(db, wake)
	p.targets["claude-code"] = projectionTarget{Runtime: "claude-code", Path: path, Cap: 4096}
	_, err = p.Project(context.Background(), "claude-code", false)
	require.NoError(t, err)
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	text := string(got)
	require.Contains(t, text, curated)
	require.Contains(t, text, "## Tail")
	require.Contains(t, text, "new generated memory")
	require.NotContains(t, text, "old\n")
}

func TestProjectionRejectsMalformedManagedWakeBlockWithoutWriting(t *testing.T) {
	db := localdb.NewSQLite(t)
	_, err := db.Exec(Schema())
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "MEMORY.md")
	original := "curated\n" + wakeStart + "\nmissing end\n"
	require.NoError(t, os.WriteFile(path, []byte(original), 0o600))
	wake := recall.NewService(projectionSource{{ID: "entry", Text: "generated", Frontier: true, CreatedAt: time.Now()}}, nil, recall.Config{WakeBudget: 40})
	p := NewProjector(db, wake)
	p.targets["claude-code"] = projectionTarget{Runtime: "claude-code", Path: path, Cap: 4096}
	_, err = p.Project(context.Background(), "claude-code", false)
	var malformed ErrMalformedManagedBlock
	require.ErrorAs(t, err, &malformed)
	got, readErr := os.ReadFile(path)
	require.NoError(t, readErr)
	require.Equal(t, original, string(got))
}

func TestProjectionAddsManagedWakeBlockWithoutOverwritingExistingFile(t *testing.T) {
	db := localdb.NewSQLite(t)
	_, err := db.Exec(Schema())
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "MEMORY.md")
	original := "# Existing\n"
	require.NoError(t, os.WriteFile(path, []byte(original), 0o600))
	wake := recall.NewService(projectionSource{{ID: "entry", Text: "generated", Frontier: true, CreatedAt: time.Now()}}, nil, recall.Config{WakeBudget: 40})
	p := NewProjector(db, wake)
	p.targets["claude-code"] = projectionTarget{Runtime: "claude-code", Path: path, Cap: 4096}
	_, err = p.Project(context.Background(), "claude-code", false)
	require.NoError(t, err)
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(string(got), original))
	require.Contains(t, string(got), wakeStart)
	require.Contains(t, string(got), wakeEnd)
}
