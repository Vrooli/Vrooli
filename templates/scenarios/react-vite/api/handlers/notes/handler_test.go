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

	"{{SCENARIO_ID}}/handlers/notes"
	"{{SCENARIO_ID}}/internal/clock"
	internalnotes "{{SCENARIO_ID}}/internal/notes"
	"{{SCENARIO_ID}}/internal/server"
	"{{SCENARIO_ID}}/internal/testutil/assertx"
	"{{SCENARIO_ID}}/internal/testutil/httpx"
	mocks "{{SCENARIO_ID}}/internal/notes/mocks"

	"github.com/gorilla/mux"
	"{{SCENARIO_ID}}/internal/module"
)

// newServer wires a server.Server with a single notes module backed by
// the supplied FakeService. The stub-module pattern matches what
// production does (a domain owns its mount), but injects the fake
// service in place of the real repo+service chain so the handler test
// stays focused on transport behaviour.
func newServer(t *testing.T, fake *mocks.FakeService, logger *log.Logger) *httpx.LiveServer {
	t.Helper()
	if logger == nil {
		logger = log.New(&bytes.Buffer{}, "", 0)
	}
	h := notes.NewHandler(notes.Deps{Service: fake, Logger: logger})
	mod := module.Module{
		Name:  "notes",
		Mount: func(r *mux.Router) { r.PathPrefix("/api/v1/notes").Handler(h) },
	}
	srv := server.New(server.Deps{Clock: clock.System{}, Logger: logger}, mod)
	return httpx.NewLiveServer(t, srv)
}

func TestNotesHandler_ListEmpty(t *testing.T) {
	live := newServer(t, &mocks.FakeService{}, nil)
	resp, body := live.Do(t, http.MethodGet, "/api/v1/notes", nil)
	assertx.AssertStatus(t, resp, http.StatusOK)

	got := assertx.MustUnmarshalProto[notesv1.ListNotesResponse](t, body)
	require.Empty(t, got.Notes, "empty service must return notes:[]")
}

func TestNotesHandler_ListReturnsItems(t *testing.T) {
	fake := &mocks.FakeService{
		ListOut: []internalnotes.Note{
			{ID: "a", Title: "first"},
			{ID: "b", Title: "second"},
		},
	}
	live := newServer(t, fake, nil)
	resp, body := live.Do(t, http.MethodGet, "/api/v1/notes", nil)
	assertx.AssertStatus(t, resp, http.StatusOK)

	got := assertx.MustUnmarshalProto[notesv1.ListNotesResponse](t, body)
	require.Len(t, got.Notes, 2)
	require.Equal(t, "first", got.Notes[0].Title)
	require.Equal(t, "second", got.Notes[1].Title)
	require.Equal(t, int64(1), fake.ListCalls.Load())
}

func TestNotesHandler_CreateSuccess(t *testing.T) {
	fake := &mocks.FakeService{}
	live := newServer(t, fake, nil)

	resp, body := live.Do(t, http.MethodPost, "/api/v1/notes",
		strings.NewReader(`{"title":"first","body":"hello"}`))
	assertx.AssertStatus(t, resp, http.StatusCreated)

	got := assertx.MustUnmarshalProto[notesv1.CreateNoteResponse](t, body)
	require.NotNil(t, got.Note)
	require.NotEmpty(t, got.Note.Id, "Create must return an ID")
	require.Equal(t, "first", got.Note.Title)
	require.Equal(t, "hello", got.Note.Body)
	require.Equal(t, int64(1), fake.CreateCalls.Load())
	require.Len(t, fake.CreateInputs, 1, "service must have recorded the input")
	require.Equal(t, "first", fake.CreateInputs[0].Title)
	require.Equal(t, "hello", fake.CreateInputs[0].Body)
}

// TestNotesHandler_CreateRejectsMissingTitle proves the handler
// translates ErrInvalidNote into the same wire shape the inline
// pre-service-layer validation used to emit. The validation source
// moved from handler.go to service.go; the wire contract is unchanged.
func TestNotesHandler_CreateRejectsMissingTitle(t *testing.T) {
	fake := &mocks.FakeService{
		CreateErr: internalnotes.ErrInvalidNote{Field: "title", Reason: "required"},
	}
	live := newServer(t, fake, nil)

	resp, body := live.Do(t, http.MethodPost, "/api/v1/notes",
		strings.NewReader(`{"title":"","body":"x"}`))
	assertx.AssertStatus(t, resp, http.StatusBadRequest)

	env := assertx.MustUnmarshalProto[errorsv1.ErrorEnvelope](t, body)
	require.Equal(t, "invalid_request", env.Code)
	require.Contains(t, env.Message, "title")
}

func TestNotesHandler_CreateRejectsMalformedJSON(t *testing.T) {
	live := newServer(t, &mocks.FakeService{}, nil)

	resp, body := live.Do(t, http.MethodPost, "/api/v1/notes",
		strings.NewReader(`{"title":`))
	assertx.AssertStatus(t, resp, http.StatusBadRequest)

	env := assertx.MustUnmarshalProto[errorsv1.ErrorEnvelope](t, body)
	require.Equal(t, "invalid_request", env.Code)
}

func TestNotesHandler_CreateRejectsUnknownFields(t *testing.T) {
	live := newServer(t, &mocks.FakeService{}, nil)

	resp, body := live.Do(t, http.MethodPost, "/api/v1/notes",
		strings.NewReader(`{"title":"x","extra":"y"}`))
	assertx.AssertStatus(t, resp, http.StatusBadRequest)

	env := assertx.MustUnmarshalProto[errorsv1.ErrorEnvelope](t, body)
	require.Equal(t, "invalid_request", env.Code)
	require.Contains(t, env.Message, "extra",
		"DisallowUnknownFields must surface the offending field name")
}

func TestNotesHandler_GetReturnsNote(t *testing.T) {
	fake := &mocks.FakeService{
		GetByID: map[string]internalnotes.Note{"abc": {ID: "abc", Title: "found"}},
	}
	live := newServer(t, fake, nil)

	resp, body := live.Do(t, http.MethodGet, "/api/v1/notes/abc", nil)
	assertx.AssertStatus(t, resp, http.StatusOK)

	got := assertx.MustUnmarshalProto[notesv1.GetNoteResponse](t, body)
	require.NotNil(t, got.Note)
	require.Equal(t, "abc", got.Note.Id)
	require.Equal(t, "found", got.Note.Title)
}

func TestNotesHandler_GetReturnsNotFound(t *testing.T) {
	fake := &mocks.FakeService{GetErr: internalnotes.ErrNoteNotFound{ID: "ghost"}}
	live := newServer(t, fake, nil)

	resp, body := live.Do(t, http.MethodGet, "/api/v1/notes/ghost", nil)
	assertx.AssertStatus(t, resp, http.StatusNotFound)

	env := assertx.MustUnmarshalProto[errorsv1.ErrorEnvelope](t, body)
	require.Equal(t, "not_found", env.Code)
	require.Contains(t, env.Message, "ghost")
}

func TestNotesHandler_GetInternalError(t *testing.T) {
	logBuf := &bytes.Buffer{}
	fake := &mocks.FakeService{GetErr: errors.New("boom")}
	live := newServer(t, fake, log.New(logBuf, "", 0))

	resp, body := live.Do(t, http.MethodGet, "/api/v1/notes/x", nil)
	assertx.AssertStatus(t, resp, http.StatusInternalServerError)

	env := assertx.MustUnmarshalProto[errorsv1.ErrorEnvelope](t, body)
	require.Equal(t, "internal", env.Code)
	require.Contains(t, fmt.Sprint(logBuf.String()), "boom",
		"underlying error must be logged for operator correlation")
}
