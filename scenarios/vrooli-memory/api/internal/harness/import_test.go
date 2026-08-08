package harness

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	sourcejournal "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal"
	journalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal/journal_v1connect"
)

type fakeJournal struct {
	entries []*sourcejournal.Entry
	keys    map[string]string
}

func (f *fakeJournal) AppendEntry(_ context.Context, req *connect.Request[sourcejournal.AppendEntryRequest]) (*connect.Response[sourcejournal.AppendEntryResponse], error) {
	if f.keys == nil {
		f.keys = map[string]string{}
	}
	key := req.Msg.GetImportProvenance().GetRuntime() + ":" + req.Msg.GetImportProvenance().GetSourceLocator() + ":" + req.Msg.GetImportProvenance().GetContentHash()
	if id := f.keys[key]; id != "" {
		return connect.NewResponse(&sourcejournal.AppendEntryResponse{Entry: &sourcejournal.Entry{Id: id}, Existing: true}), nil
	}
	id := "entry-" + string(rune(len(f.entries)+'1'))
	f.keys[key] = id
	f.entries = append(f.entries, &sourcejournal.Entry{Id: id, Body: req.Msg.GetBody(), Kind: req.Msg.GetKind(), ImportProvenance: req.Msg.GetImportProvenance()})
	return connect.NewResponse(&sourcejournal.AppendEntryResponse{Entry: f.entries[len(f.entries)-1]}), nil
}
func (*fakeJournal) GetEntry(context.Context, *connect.Request[sourcejournal.GetEntryRequest]) (*connect.Response[sourcejournal.GetEntryResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (*fakeJournal) ListEntries(context.Context, *connect.Request[sourcejournal.ListEntriesRequest]) (*connect.Response[sourcejournal.ListEntriesResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (*fakeJournal) ProcessClassificationRetries(context.Context, *connect.Request[sourcejournal.ProcessClassificationRetriesRequest]) (*connect.Response[sourcejournal.ProcessClassificationRetriesResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (*fakeJournal) ProcessEmbeddingRetries(context.Context, *connect.Request[sourcejournal.ProcessEmbeddingRetriesRequest]) (*connect.Response[sourcejournal.ProcessEmbeddingRetriesResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

var _ journalconnect.JournalServiceClient = (*fakeJournal)(nil)

func TestImportAndCaptureAreRemoteAndContentAddressed(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "one.md"), []byte("first memory"), 0o600))
	client := &fakeJournal{}
	importer := NewImporter(client, dir, nil)
	first, err := importer.Import(context.Background(), "claude-code", false)
	require.NoError(t, err)
	require.Equal(t, 1, first.Imported)
	second, err := importer.Import(context.Background(), "claude-code", false)
	require.NoError(t, err)
	require.Equal(t, 1, second.Existing)
	_, err = importer.Capture(context.Background(), "claude-code", filepath.Join(dir, "one.md"), "first memory")
	require.NoError(t, err)
	require.Len(t, client.entries, 1)
}

func TestImportWorkerCountIsBoundedAndConfigurable(t *testing.T) {
	t.Setenv("VROOLI_MEMORY_IMPORT_CONCURRENCY", "8")
	require.Equal(t, 8, importWorkerCount())
	t.Setenv("VROOLI_MEMORY_IMPORT_CONCURRENCY", "99")
	require.Equal(t, 16, importWorkerCount())
	t.Setenv("VROOLI_MEMORY_IMPORT_CONCURRENCY", "0")
	require.Equal(t, 4, importWorkerCount())
}
