package journal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	journalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal"
)

type listPage struct {
	response *connect.Response[journalv1.ListEntriesResponse]
	err      error
}

type fakeJournalLister struct {
	pages    []listPage
	requests []*journalv1.ListEntriesRequest
}

func (f *fakeJournalLister) ListEntries(_ context.Context, request *connect.Request[journalv1.ListEntriesRequest]) (*connect.Response[journalv1.ListEntriesResponse], error) {
	f.requests = append(f.requests, request.Msg)
	page := f.pages[0]
	f.pages = f.pages[1:]
	return page.response, page.err
}

func (f *fakeJournalLister) GetEntry(context.Context, *connect.Request[journalv1.GetEntryRequest]) (*connect.Response[journalv1.GetEntryResponse], error) {
	return nil, errors.New("unexpected GetEntry call")
}

func (f *fakeJournalLister) AppendEntry(context.Context, *connect.Request[journalv1.AppendEntryRequest]) (*connect.Response[journalv1.AppendEntryResponse], error) {
	return nil, errors.New("unexpected AppendEntry call")
}

func TestExportExhaustsAscendingPages(t *testing.T) {
	client := &fakeJournalLister{pages: []listPage{
		{response: connect.NewResponse(&journalv1.ListEntriesResponse{Entries: []*journalv1.Entry{{Id: "a"}}, NextCursor: "cursor-1"})},
		{response: connect.NewResponse(&journalv1.ListEntriesResponse{Entries: []*journalv1.Entry{{Id: "b"}}})},
	}}
	h := &handlers{client: client}
	snapshot := time.Date(2026, 11, 10, 1, 23, 25, 0, time.FixedZone("test", -5*60*60))

	result := h.export(context.Background(), "ecosystem-hunt", exportPageSize, snapshot)

	require.True(t, result.Complete)
	require.False(t, result.Degraded)
	require.Equal(t, "source-ledger.journal.list", result.Source)
	require.Equal(t, "ecosystem-hunt", result.Scope)
	require.Equal(t, "2026-11-10T06:23:25Z", result.SnapshotAt)
	require.Equal(t, 2, result.PageCount)
	require.Equal(t, 2, result.EntryCount)
	require.Len(t, result.Entries, 2)
	require.Equal(t, []string{"", "cursor-1"}, []string{client.requests[0].GetCursor(), client.requests[1].GetCursor()})
	require.Equal(t, []int32{500, 500}, []int32{client.requests[0].GetLimit(), client.requests[1].GetLimit()})
	require.Equal(t, []string{"ecosystem-hunt", "ecosystem-hunt"}, []string{client.requests[0].GetScope(), client.requests[1].GetScope()})
}

func TestExportFailedPageStaysIncompleteWithPartialCounts(t *testing.T) {
	client := &fakeJournalLister{pages: []listPage{
		{response: connect.NewResponse(&journalv1.ListEntriesResponse{Entries: []*journalv1.Entry{{Id: "a"}}, NextCursor: "cursor-1"})},
		{err: errors.New("owner API unavailable")},
	}}
	result := (&handlers{client: client}).export(context.Background(), "ecosystem-hunt", exportPageSize, time.Now())

	require.False(t, result.Complete)
	require.True(t, result.Degraded)
	require.Contains(t, result.DegradationReason, "page 2 failed")
	require.Equal(t, 1, result.PageCount)
	require.Equal(t, 1, result.EntryCount)
	require.Len(t, result.Entries, 1)
}

func TestExportRejectsNonAdvancingCursor(t *testing.T) {
	client := &fakeJournalLister{pages: []listPage{{
		response: connect.NewResponse(&journalv1.ListEntriesResponse{NextCursor: "cursor-1"}),
	}, {
		response: connect.NewResponse(&journalv1.ListEntriesResponse{NextCursor: "cursor-1"}),
	}}}
	result := (&handlers{client: client}).export(context.Background(), "ecosystem-hunt", exportPageSize, time.Now())

	require.False(t, result.Complete)
	require.True(t, result.Degraded)
	require.Contains(t, result.DegradationReason, "non-advancing cursor")
	require.Equal(t, 2, result.PageCount)
}

func TestExportRejectsDuplicateEntryAcrossPages(t *testing.T) {
	client := &fakeJournalLister{pages: []listPage{
		{response: connect.NewResponse(&journalv1.ListEntriesResponse{Entries: []*journalv1.Entry{{Id: "same"}}, NextCursor: "cursor-1"})},
		{response: connect.NewResponse(&journalv1.ListEntriesResponse{Entries: []*journalv1.Entry{{Id: "same"}}})},
	}}
	result := (&handlers{client: client}).export(context.Background(), "ecosystem-hunt", exportPageSize, time.Now())

	require.False(t, result.Complete)
	require.True(t, result.Degraded)
	require.Contains(t, result.DegradationReason, "repeated entry ID")
	require.Equal(t, 1, result.EntryCount)
	require.Equal(t, 2, result.PageCount)
}

func TestExportRunEmitsIncompleteJSONAndReturnsErrorAfterPageFailure(t *testing.T) {
	client := &fakeJournalLister{pages: []listPage{{err: errors.New("owner API unavailable")}}}
	h := &handlers{client: client}
	var output bytes.Buffer
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "scope", Default: "ecosystem-hunt"}}},
		Stdout: &output,
	})

	err := h.exportRun(ctx)

	require.ErrorContains(t, err, "journal export incomplete")
	var emitted journalExport
	require.NoError(t, json.Unmarshal(output.Bytes(), &emitted))
	require.False(t, emitted.Complete)
	require.True(t, emitted.Degraded)
	require.Equal(t, "ecosystem-hunt", emitted.Scope)
	require.Contains(t, emitted.DegradationReason, "page 1 failed")
}
