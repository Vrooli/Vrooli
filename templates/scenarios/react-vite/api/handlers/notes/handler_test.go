package notes_test

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	errorsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/{{SCENARIO_ID}}/v1/errors"
	notesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/{{SCENARIO_ID}}/v1/notes"

	"{{SCENARIO_ID}}/internal/clock"
	"{{SCENARIO_ID}}/internal/server"
	"{{SCENARIO_ID}}/internal/store"
	"{{SCENARIO_ID}}/internal/testutil/assertx"
	"{{SCENARIO_ID}}/internal/testutil/httpx"
	"{{SCENARIO_ID}}/internal/testutil/mocks"
)

// newServer wires a server.Server backed by the supplied FakeNoteStore.
// Health-side deps are populated with safe defaults so the test's
// surface stays focused on notes.
func newServer(t *testing.T, fake *mocks.FakeNoteStore) *httpx.LiveServer {
	t.Helper()
	srv := server.New(server.Deps{
		Pinger:    &mocks.FakePinger{},
		Clock:     clock.System{},
		Logger:    log.New(&bytes.Buffer{}, "", 0),
		NoteStore: fake,
		Service:   "react-vite-test",
		Version:   "1.0.0",
	})
	return httpx.NewLiveServer(t, srv)
}

func TestNotesHandler_ListEmpty(t *testing.T) {
	live := newServer(t, &mocks.FakeNoteStore{})
	resp, body := live.Do(t, http.MethodGet, "/api/v1/notes", nil)
	assertx.AssertStatus(t, resp, http.StatusOK)

	got := assertx.MustUnmarshalProto[notesv1.ListNotesResponse](t, body)
	require.Empty(t, got.Notes, "empty store must return notes:[]")
}

func TestNotesHandler_ListReturnsItems(t *testing.T) {
	fake := &mocks.FakeNoteStore{
		Notes: []store.Note{
			{ID: "a", Title: "first"},
			{ID: "b", Title: "second"},
		},
	}
	live := newServer(t, fake)
	resp, body := live.Do(t, http.MethodGet, "/api/v1/notes", nil)
	assertx.AssertStatus(t, resp, http.StatusOK)

	got := assertx.MustUnmarshalProto[notesv1.ListNotesResponse](t, body)
	require.Len(t, got.Notes, 2)
	require.Equal(t, "first", got.Notes[0].Title)
	require.Equal(t, "second", got.Notes[1].Title)
	require.Equal(t, int64(1), fake.ListCalls.Load())
}

func TestNotesHandler_CreateSuccess(t *testing.T) {
	fake := &mocks.FakeNoteStore{}
	live := newServer(t, fake)

	resp, body := live.Do(t, http.MethodPost, "/api/v1/notes",
		strings.NewReader(`{"title":"first","body":"hello"}`))
	assertx.AssertStatus(t, resp, http.StatusCreated)

	got := assertx.MustUnmarshalProto[notesv1.CreateNoteResponse](t, body)
	require.NotNil(t, got.Note)
	require.NotEmpty(t, got.Note.Id, "Create must return an ID")
	require.Equal(t, "first", got.Note.Title)
	require.Equal(t, "hello", got.Note.Body)
	require.Equal(t, int64(1), fake.CreateCalls.Load())
	require.Len(t, fake.Notes, 1, "store must have one note")
}

func TestNotesHandler_CreateRejectsMissingTitle(t *testing.T) {
	fake := &mocks.FakeNoteStore{}
	live := newServer(t, fake)

	resp, body := live.Do(t, http.MethodPost, "/api/v1/notes",
		strings.NewReader(`{"title":"","body":"x"}`))
	assertx.AssertStatus(t, resp, http.StatusBadRequest)

	env := assertx.MustUnmarshalProto[errorsv1.ErrorEnvelope](t, body)
	require.Equal(t, "invalid_request", env.Code)
	require.Contains(t, env.Message, "title")
	require.Equal(t, int64(0), fake.CreateCalls.Load(), "validation rejects before store call")
}

func TestNotesHandler_CreateRejectsMalformedJSON(t *testing.T) {
	live := newServer(t, &mocks.FakeNoteStore{})

	resp, body := live.Do(t, http.MethodPost, "/api/v1/notes",
		strings.NewReader(`{"title":`))
	assertx.AssertStatus(t, resp, http.StatusBadRequest)

	env := assertx.MustUnmarshalProto[errorsv1.ErrorEnvelope](t, body)
	require.Equal(t, "invalid_request", env.Code)
}

func TestNotesHandler_CreateRejectsUnknownFields(t *testing.T) {
	live := newServer(t, &mocks.FakeNoteStore{})

	resp, body := live.Do(t, http.MethodPost, "/api/v1/notes",
		strings.NewReader(`{"title":"x","extra":"y"}`))
	assertx.AssertStatus(t, resp, http.StatusBadRequest)

	env := assertx.MustUnmarshalProto[errorsv1.ErrorEnvelope](t, body)
	require.Equal(t, "invalid_request", env.Code)
	require.Contains(t, env.Message, "extra",
		"DisallowUnknownFields must surface the offending field name")
}

func TestNotesHandler_GetReturnsNote(t *testing.T) {
	fake := &mocks.FakeNoteStore{
		Notes: []store.Note{{ID: "abc", Title: "found"}},
	}
	live := newServer(t, fake)

	resp, body := live.Do(t, http.MethodGet, "/api/v1/notes/abc", nil)
	assertx.AssertStatus(t, resp, http.StatusOK)

	got := assertx.MustUnmarshalProto[notesv1.GetNoteResponse](t, body)
	require.NotNil(t, got.Note)
	require.Equal(t, "abc", got.Note.Id)
	require.Equal(t, "found", got.Note.Title)
}

func TestNotesHandler_GetReturnsNotFound(t *testing.T) {
	fake := &mocks.FakeNoteStore{GetErr: store.ErrNoteNotFound{ID: "ghost"}}
	live := newServer(t, fake)

	resp, body := live.Do(t, http.MethodGet, "/api/v1/notes/ghost", nil)
	assertx.AssertStatus(t, resp, http.StatusNotFound)

	env := assertx.MustUnmarshalProto[errorsv1.ErrorEnvelope](t, body)
	require.Equal(t, "not_found", env.Code)
	require.Contains(t, env.Message, "ghost")
}

func TestNotesHandler_GetInternalError(t *testing.T) {
	logBuf := &bytes.Buffer{}
	fake := &mocks.FakeNoteStore{GetErr: errors.New("boom")}
	srv := server.New(server.Deps{
		Pinger:    &mocks.FakePinger{},
		Clock:     clock.System{},
		Logger:    log.New(logBuf, "", 0),
		NoteStore: fake,
		Service:   "react-vite-test",
		Version:   "1.0.0",
	})
	live := httpx.NewLiveServer(t, srv)

	resp, body := live.Do(t, http.MethodGet, "/api/v1/notes/x", nil)
	assertx.AssertStatus(t, resp, http.StatusInternalServerError)

	env := assertx.MustUnmarshalProto[errorsv1.ErrorEnvelope](t, body)
	require.Equal(t, "internal", env.Code)
	require.Contains(t, fmt.Sprint(logBuf.String()), "boom",
		"underlying error must be logged for operator correlation")
}
