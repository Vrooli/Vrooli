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
	pages          []listPage
	requests       []*journalv1.ListEntriesRequest
	getRequests    []*journalv1.GetEntryRequest
	appendRequest  *journalv1.AppendEntryRequest
	getResponse    *connect.Response[journalv1.GetEntryResponse]
	appendResponse *connect.Response[journalv1.AppendEntryResponse]
	getErr         error
	appendErr      error
}

func (f *fakeJournalLister) ListEntries(_ context.Context, request *connect.Request[journalv1.ListEntriesRequest]) (*connect.Response[journalv1.ListEntriesResponse], error) {
	f.requests = append(f.requests, request.Msg)
	page := f.pages[0]
	f.pages = f.pages[1:]
	return page.response, page.err
}

func (f *fakeJournalLister) GetEntry(_ context.Context, request *connect.Request[journalv1.GetEntryRequest]) (*connect.Response[journalv1.GetEntryResponse], error) {
	f.getRequests = append(f.getRequests, request.Msg)
	return f.getResponse, f.getErr
}

func (f *fakeJournalLister) AppendEntry(_ context.Context, request *connect.Request[journalv1.AppendEntryRequest]) (*connect.Response[journalv1.AppendEntryResponse], error) {
	f.appendRequest = request.Msg
	return f.appendResponse, f.appendErr
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
	result := (&handlers{client: client}).export(context.Background(), "ecosystem-hunt", exportPageSize, time.Unix(1, 0))

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
	result := (&handlers{client: client}).export(context.Background(), "ecosystem-hunt", exportPageSize, time.Unix(1, 0))

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
	result := (&handlers{client: client}).export(context.Background(), "ecosystem-hunt", exportPageSize, time.Unix(1, 0))

	require.False(t, result.Complete)
	require.True(t, result.Degraded)
	require.Contains(t, result.DegradationReason, "repeated entry ID")
	require.Equal(t, 1, result.EntryCount)
	require.Equal(t, 2, result.PageCount)
}

func TestExportRunEmitsIncompleteJSONAndReturnsErrorAfterPageFailure(t *testing.T) {
	client := &fakeJournalLister{pages: []listPage{{err: errors.New("owner API unavailable")}}}
	h := &handlers{client: client, Now: func() time.Time { return time.Unix(1, 0) }}
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

func TestJournalCommandsForwardOwnerRequestsAndRenderReturnedEntries(t *testing.T) {
	client := &fakeJournalLister{
		pages:          []listPage{{response: connect.NewResponse(&journalv1.ListEntriesResponse{Entries: []*journalv1.Entry{{Id: "entry-1", Kind: "finding", Body: "source-confirmed defect"}}})}},
		appendResponse: connect.NewResponse(&journalv1.AppendEntryResponse{Entry: &journalv1.Entry{Id: "entry-2", FacetId: "coverage", Body: "owner fix passed"}}),
		getResponse:    connect.NewResponse(&journalv1.GetEntryResponse{Entry: &journalv1.Entry{Id: "entry-1", Body: "source-confirmed defect"}}),
	}
	h := &handlers{client: client}

	appendCtx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema:      cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "scope"}, {Name: "kind"}, {Name: "trigger"}, {Name: "approach"}, {Name: "evidence"}, {Name: "outcome"}}, Positionals: []cliapp.Positional{{Name: "body"}}},
		Flags:       map[string]string{"scope": "ecosystem-hunt", "kind": "finding", "trigger": "issue-8", "approach": "inject clock", "evidence": "receipt.json", "outcome": "deterministic export"},
		Positionals: map[string]string{"body": "source-confirmed defect"},
	})
	appended, err := h.noteCall(appendCtx)
	require.NoError(t, err)
	require.Equal(t, &journalv1.AppendEntryRequest{Scope: "ecosystem-hunt", Kind: "finding", Trigger: "issue-8", Approach: "inject clock", Evidence: "receipt.json", Outcome: "deterministic export", Body: "source-confirmed defect"}, client.appendRequest)
	require.Equal(t, []string{"Recorded entry-2."}, h.noteReport(appendCtx, appended).Result)

	listCtx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "scope"}, {Name: "limit"}, {Name: "kind"}, {Name: "cursor"}, {Name: "newest-first", Bool: true}}},
		Flags:  map[string]string{"scope": "ecosystem-hunt", "limit": "25", "kind": "finding"}, BoolFlags: map[string]bool{"newest-first": true},
	})
	listed, err := h.listCall(listCtx)
	require.NoError(t, err)
	require.Equal(t, &journalv1.ListEntriesRequest{Scope: "ecosystem-hunt", Limit: 25, Kind: "finding", NewestFirst: true}, client.requests[0])
	require.Equal(t, []string{"entry-1 [finding] source-confirmed defect"}, h.listReport(listCtx, listed).Results)

	getCtx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "scope"}}, Positionals: []cliapp.Positional{{Name: "id"}}},
		Flags:  map[string]string{"scope": "ecosystem-hunt"}, Positionals: map[string]string{"id": "entry-1"},
	})
	got, err := h.getCall(getCtx)
	require.NoError(t, err)
	require.Equal(t, []*journalv1.GetEntryRequest{{Id: "entry-1", Scope: "ecosystem-hunt"}}, client.getRequests)
	require.Equal(t, []string{"source-confirmed defect"}, h.getReport(getCtx, got).Results)
}

func TestListRejectsIncompatibleFiltersBeforeRPC(t *testing.T) {
	client := &fakeJournalLister{}
	h := &handlers{client: client}
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "scope"}, {Name: "limit"}, {Name: "kind"}, {Name: "cursor"}, {Name: "newest-first", Bool: true}}}
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: schema, Flags: map[string]string{"limit": "10", "kind": "finding"}})
	_, err := h.listCall(ctx)
	require.ErrorContains(t, err, "kind requires --newest-first")
	require.Empty(t, client.requests)

	ctx = cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: schema, Flags: map[string]string{"limit": "10", "cursor": "page-2"}, BoolFlags: map[string]bool{"newest-first": true}})
	_, err = h.listCall(ctx)
	require.ErrorContains(t, err, "newest-first does not accept a cursor")
	require.Empty(t, client.requests)
}

func TestIncompleteJournalResponsesReturnErrors(t *testing.T) {
	client := &fakeJournalLister{pages: []listPage{{response: nil}}}
	h := &handlers{client: client}
	listCtx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "scope"}, {Name: "limit"}, {Name: "kind"}, {Name: "cursor"}, {Name: "newest-first", Bool: true}}}, Flags: map[string]string{"limit": "5"}})
	_, err := h.listCall(listCtx)
	require.ErrorContains(t, err, "server returned no journal response")

	getCtx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "scope"}}, Positionals: []cliapp.Positional{{Name: "id"}}}, Positionals: map[string]string{"id": "missing"}})
	_, err = h.getCall(getCtx)
	require.ErrorContains(t, err, "server returned no journal entry")
	client.getResponse = connect.NewResponse(&journalv1.GetEntryResponse{})
	_, err = h.getCall(getCtx)
	require.ErrorContains(t, err, "server returned no journal entry")
}

func TestExportRunUsesInjectedClock(t *testing.T) {
	client := &fakeJournalLister{pages: []listPage{{response: connect.NewResponse(&journalv1.ListEntriesResponse{Entries: []*journalv1.Entry{{Id: "entry-1"}}})}}}
	fixed := time.Date(2026, 11, 21, 13, 0, 0, 123, time.FixedZone("operator", -5*60*60))
	calls := 0
	h := &handlers{client: client, Now: func() time.Time { calls++; return fixed }}
	var output bytes.Buffer
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "scope"}}}, Flags: map[string]string{"scope": "ecosystem-hunt"}, Stdout: &output})
	require.NoError(t, h.exportRun(ctx))
	require.Equal(t, 1, calls)
	var exported journalExport
	require.NoError(t, json.Unmarshal(output.Bytes(), &exported))
	require.Equal(t, "2026-11-21T18:00:00.000000123Z", exported.SnapshotAt)
	require.True(t, exported.Complete)
}
