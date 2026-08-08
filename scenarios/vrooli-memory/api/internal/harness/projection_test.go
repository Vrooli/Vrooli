package harness

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	sourcerecall "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/recall"
	recallconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/recall/recall_v1connect"
	"vrooli-memory/internal/ledgerclient"
	localdb "vrooli-memory/internal/testutil/db"
)

type fakeRecall struct {
	hits    []*sourcerecall.RecallHit
	err     error
	request *sourcerecall.WakeRequest
}

func (f *fakeRecall) Wake(_ context.Context, request *connect.Request[sourcerecall.WakeRequest]) (*connect.Response[sourcerecall.WakeResponse], error) {
	f.request = request.Msg
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
	p.targets["claude-code"] = projectionTarget{Runtime: "claude-code", Path: path, Cap: 4096, LineCap: 12}
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

func TestProjectionSplicedFileStaysWithinConsumerLineCeiling(t *testing.T) {
	path := filepath.Join(t.TempDir(), "MEMORY.md")
	curated := "# Curated operator index\n\n- Keep this byte-for-byte\n"
	require.NoError(t, os.WriteFile(path, []byte(curated), 0o600))
	hits := make([]*sourcerecall.RecallHit, 0, 20)
	for i := 0; i < 20; i++ {
		hits = append(hits, &sourcerecall.RecallHit{Text: "generated memory entry"})
	}
	p, db := newProjectionTest(t, &fakeRecall{hits: hits}, path)
	defer db.Close()

	const consumerLineCeiling = 12
	p.targets["claude-code"] = projectionTarget{Runtime: "claude-code", Path: path, Cap: 4096, LineCap: consumerLineCeiling}
	result, err := p.Project(context.Background(), "claude-code", true)
	require.NoError(t, err)
	actualLines := strings.Count(result.Content, "\n")
	require.LessOrEqualf(t, actualLines, consumerLineCeiling, "spliced file rendered %d lines, ceiling is %d", actualLines, consumerLineCeiling)
}

func TestProjectionZeroHitWakeStillSplicesManagedBlock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "MEMORY.md")
	p, db := newProjectionTest(t, &fakeRecall{}, path)
	defer db.Close()

	result, err := p.Project(context.Background(), "claude-code", true)
	require.NoError(t, err)
	require.Contains(t, result.Content, wakeStart)
	require.Contains(t, result.Content, wakeEnd)
	require.False(t, result.Overflow)
	require.LessOrEqual(t, result.SizeLines, int64(12))
}

func TestProjectionCountsOneAndTwoLineEntriesAsRenderedLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "MEMORY.md")
	p, db := newProjectionTest(t, &fakeRecall{hits: []*sourcerecall.RecallHit{{Text: "one line"}, {Text: "first\nsecond"}}}, path)
	defer db.Close()

	result, err := p.Project(context.Background(), "claude-code", true)
	require.NoError(t, err)
	require.Contains(t, result.Content, "- one line\n\n")
	require.Contains(t, result.Content, "- first\nsecond\n\n")
	require.LessOrEqual(t, result.SizeLines, int64(12))
}

func TestProjectionLeavesCuratedRegionWhenItAlreadyExceedsCeiling(t *testing.T) {
	path := filepath.Join(t.TempDir(), "MEMORY.md")
	curated := "# Curated\nline one\nline two\nline three\nline four\n"
	require.NoError(t, os.WriteFile(path, []byte(curated), 0o600))
	p, db := newProjectionTest(t, &fakeRecall{hits: []*sourcerecall.RecallHit{{Text: "must not fit"}}}, path)
	defer db.Close()
	p.targets["claude-code"] = projectionTarget{Runtime: "claude-code", Path: path, Cap: 4096, LineCap: 3}

	result, err := p.Project(context.Background(), "claude-code", false)
	require.NoError(t, err)
	require.True(t, result.Overflow)
	require.Equal(t, curated, strings.Split(result.Content, wakeStart)[0])
	require.NotContains(t, result.Content, "must not fit")
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, curated, strings.Split(string(got), wakeStart)[0])
}

func TestProjectionPassesRemainingRenderedLineBudgetToWake(t *testing.T) {
	path := filepath.Join(t.TempDir(), "MEMORY.md")
	curated := "# Curated\nline one\nline two\n"
	require.NoError(t, os.WriteFile(path, []byte(curated), 0o600))
	wake := &fakeRecall{hits: []*sourcerecall.RecallHit{{Text: "one"}}}
	p, db := newProjectionTest(t, wake, path)
	defer db.Close()

	_, err := p.Project(context.Background(), "claude-code", true)
	require.NoError(t, err)
	require.NotNil(t, wake.request)
	require.Equal(t, int32(9), wake.request.GetLineBudget())
}

func TestProjectionRecordsSplicedFileMetrics(t *testing.T) {
	path := filepath.Join(t.TempDir(), "MEMORY.md")
	p, db := newProjectionTest(t, &fakeRecall{hits: []*sourcerecall.RecallHit{{Text: "durable"}}}, path)
	result, err := p.Project(context.Background(), "claude-code", false)
	require.NoError(t, err)
	defer db.Close()

	var bytes, lines int64
	err = db.QueryRow(`SELECT size_bytes,size_lines FROM harness_projections WHERE runtime=?`, "claude-code").Scan(&bytes, &lines)
	require.NoError(t, err)
	require.Equal(t, int64(len(result.Content)), bytes)
	require.Equal(t, result.SizeLines, lines)
	require.Equal(t, int64(len(result.Content)), result.SizeBytes)
}
