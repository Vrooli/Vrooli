package notes_test

import (
	"context"
	"errors"
	"log"
	"testing"
	"time"

	"network-manager/handlers/notes"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	notesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/notes"
	notesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/notes/notes_v1connect"

	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"

	internalnotes "network-manager/internal/notes"
	mocks "network-manager/internal/notes/mocks"
	clockmocks "network-manager/internal/testutil/mocks"
)

func newNotesClient(t *testing.T, fake *mocks.FakeService, logger *log.Logger) notesconnect.NotesServiceClient {
	t.Helper()
	return newNotesClientWithClock(t, fake, logger, nil)
}

func newNotesClientWithClock(t *testing.T, fake *mocks.FakeService, logger *log.Logger, clk *clockmocks.FakeClock) notesconnect.NotesServiceClient {
	t.Helper()
	if logger == nil {
		logger, _ = connectxtest.NewLogger(t)
	}
	deps := notes.Deps{Service: fake, Logger: logger}
	if clk != nil {
		deps.Clock = clk
	}
	path, handler := notesconnect.NewNotesServiceHandler(notes.NewConnectHandler(deps))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: handler})
	return notesconnect.NewNotesServiceClient(server.Client(), server.URL)
}

func TestConnectHandlerListReturnsItems(t *testing.T) {
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	client := newNotesClient(t, &mocks.FakeService{
		ListOut: []internalnotes.Note{
			{ID: "a", Title: "first", CreatedAt: now, UpdatedAt: now, AttachmentKeys: []string{"notes/a/one.txt"}},
			{ID: "b", Title: "second", CreatedAt: now, UpdatedAt: now},
		},
	}, nil)

	resp, err := client.ListNotes(context.Background(), connect.NewRequest(&notesv1.ListNotesRequest{}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Notes, 2)
	require.Equal(t, "first", resp.Msg.Notes[0].Title)
	require.Equal(t, []string{"notes/a/one.txt"}, resp.Msg.Notes[0].AttachmentKeys)
	require.Equal(t, now.Unix(), resp.Msg.Notes[0].CreatedAt.AsTime().Unix())
}

func TestConnectHandlerCreateSuccess(t *testing.T) {
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	fake := &mocks.FakeService{
		CreateOut: &internalnotes.Note{ID: "created", Title: "first", Body: "hello", CreatedAt: now, UpdatedAt: now},
	}
	client := newNotesClient(t, fake, nil)

	resp, err := client.CreateNote(context.Background(), connect.NewRequest(&notesv1.CreateNoteRequest{
		Title: "first",
		Body:  "hello",
	}))
	require.NoError(t, err)
	require.Equal(t, "created", resp.Msg.Note.Id)
	require.Equal(t, "first", fake.CreateInputs[0].Title)
	require.Equal(t, "hello", fake.CreateInputs[0].Body)
}

func TestConnectHandlerCreateInvalidArgument(t *testing.T) {
	client := newNotesClient(t, &mocks.FakeService{
		CreateErr: internalnotes.ErrInvalidNote{Field: "title", Reason: "required"},
	}, nil)

	_, err := client.CreateNote(context.Background(), connect.NewRequest(&notesv1.CreateNoteRequest{Title: ""}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	require.Contains(t, err.Error(), "title")
}

func TestConnectHandlerGetReturnsNotFound(t *testing.T) {
	client := newNotesClient(t, &mocks.FakeService{GetErr: internalnotes.ErrNoteNotFound{ID: "ghost"}}, nil)

	_, err := client.GetNote(context.Background(), connect.NewRequest(&notesv1.GetNoteRequest{Id: "ghost"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestConnectHandlerCountResolvesWindow(t *testing.T) {
	// Wednesday 2026-05-06 12:00 UTC; start of that ISO week is Monday 2026-05-04 00:00.
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	fake := &mocks.FakeService{CountOut: 7}
	client := newNotesClientWithClock(t, fake, nil, clockmocks.NewFakeClock(now))

	resp, err := client.CountNotes(context.Background(), connect.NewRequest(&notesv1.CountNotesRequest{
		Window: &measuresv1.TimeWindow{Window: &measuresv1.TimeWindow_Token{Token: measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_THIS_WEEK}},
	}))
	require.NoError(t, err)
	require.Equal(t, int64(7), resp.Msg.Count)
	require.Len(t, fake.CountWindows, 1)
	require.Equal(t, time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC), fake.CountWindows[0][0].UTC())
	require.Equal(t, now, fake.CountWindows[0][1].UTC())
}

func TestConnectHandlerCountDefaultsToThisWeekWhenUnset(t *testing.T) {
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	fake := &mocks.FakeService{CountOut: 3}
	client := newNotesClientWithClock(t, fake, nil, clockmocks.NewFakeClock(now))

	// No Window set: the handler defaults to this_week rather than erroring.
	resp, err := client.CountNotes(context.Background(), connect.NewRequest(&notesv1.CountNotesRequest{}))
	require.NoError(t, err)
	require.Equal(t, int64(3), resp.Msg.Count)
	require.Len(t, fake.CountWindows, 1)
	require.Equal(t, time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC), fake.CountWindows[0][0].UTC())
}

func TestConnectHandlerGetInternalErrorLogs(t *testing.T) {
	logger, logBuf := connectxtest.NewLogger(t)
	client := newNotesClient(t, &mocks.FakeService{GetErr: errors.New("boom")}, logger)

	_, err := client.GetNote(context.Background(), connect.NewRequest(&notesv1.GetNoteRequest{Id: "x"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	require.Contains(t, logBuf.String(), "boom")
}
