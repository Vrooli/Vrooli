package harness

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	sourcerecall "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/recall"
	recallconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/recall/recall_v1connect"
	"vrooli-memory/internal/ledgerclient"
	localdb "vrooli-memory/internal/testutil/db"
)

type fakeRecall struct {
	hits []*sourcerecall.RecallHit
	err  error
}

func (f *fakeRecall) Wake(context.Context, *connect.Request[sourcerecall.WakeRequest]) (*connect.Response[sourcerecall.WakeResponse], error) {
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(&sourcerecall.WakeResponse{Hits: f.hits}), nil
}
func (*fakeRecall) Recall(context.Context, *connect.Request[sourcerecall.RecallRequest]) (*connect.Response[sourcerecall.RecallResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (*fakeRecall) Zoom(context.Context, *connect.Request[sourcerecall.ZoomRequest]) (*connect.Response[sourcerecall.ZoomResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (*fakeRecall) ListSiblingEvents(context.Context, *connect.Request[sourcerecall.ListSiblingEventsRequest]) (*connect.Response[sourcerecall.ListSiblingEventsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

var _ recallconnect.RecallServiceClient = (*fakeRecall)(nil)

func newProjectionTest(t *testing.T, wake recallconnect.RecallServiceClient, path string) (*Projector, *sql.DB) {
	t.Helper()
	db := localdb.NewSQLite(t)
	_, err := db.Exec(Schema())
	require.NoError(t, err)
	p := NewProjector(db, wake)
	p.targets["claude-code"] = projectionTarget{Runtime: "claude-code", Path: path, Cap: 4096}
	return p, db
}

func TestProjectionWritesManagedBlockFromSourceWake(t *testing.T) {
	path := filepath.Join(t.TempDir(), "MEMORY.md")
	p, db := newProjectionTest(t, &fakeRecall{hits: []*sourcerecall.RecallHit{{Text: "remote durable memory"}}}, path)
	defer db.Close()
	result, err := p.Project(context.Background(), "claude-code", false)
	require.NoError(t, err)
	require.Contains(t, result.Content, "remote durable memory")
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(got), wakeStart)
}

func TestProjectionLeavesLastSuccessfulBlockUntouchedDuringOutage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "MEMORY.md")
	original := generatedHeader + "# Unified Vrooli Memory\n\n" + wakeStart + "\n- last successful wake\n\n" + wakeEnd + "\n"
	require.NoError(t, os.WriteFile(path, []byte(original), 0o600))
	p, db := newProjectionTest(t, &fakeRecall{err: &ledgerclient.UnavailableError{Operation: "wake", Err: errors.New("source stopped")}}, path)
	defer db.Close()
	_, err := p.Project(context.Background(), "claude-code", false)
	require.Error(t, err)
	got, readErr := os.ReadFile(path)
	require.NoError(t, readErr)
	require.Equal(t, original, string(got))
}

func TestProjectionPreservesCuratedBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "MEMORY.md")
	original := "# Curated\n\n" + wakeStart + "\nold\n" + wakeEnd + "\n## Tail\n"
	require.NoError(t, os.WriteFile(path, []byte(original), 0o600))
	p, db := newProjectionTest(t, &fakeRecall{hits: []*sourcerecall.RecallHit{{Text: "new generated memory"}}}, path)
	defer db.Close()
	_, err := p.Project(context.Background(), "claude-code", false)
	require.NoError(t, err)
	got, _ := os.ReadFile(path)
	require.Contains(t, string(got), "# Curated")
	require.Contains(t, string(got), "## Tail")
	require.Contains(t, string(got), "new generated memory")
	require.NotContains(t, string(got), "old\n")
}
