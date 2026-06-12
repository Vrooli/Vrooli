package notes

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	notesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/notes"
	notesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/notes/notes_v1connect"
	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "code-facts/cli/internal/testutil"
)

type notesService struct {
	mu           sync.Mutex
	listResp     *notesv1.ListNotesResponse
	createResp   *notesv1.CreateNoteResponse
	getResp      *notesv1.GetNoteResponse
	listErr      error
	createErr    error
	getErr       error
	createInputs []*notesv1.CreateNoteRequest
	getIDs       []string
	countResp    *notesv1.CountNotesResponse
	countErr     error
	countWindows []*measuresv1.TimeWindow
}

func (s *notesService) ListNotes(context.Context, *connect.Request[notesv1.ListNotesRequest]) (*connect.Response[notesv1.ListNotesResponse], error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	if s.listResp == nil {
		s.listResp = &notesv1.ListNotesResponse{}
	}
	return connect.NewResponse(s.listResp), nil
}

func (s *notesService) CreateNote(_ context.Context, req *connect.Request[notesv1.CreateNoteRequest]) (*connect.Response[notesv1.CreateNoteResponse], error) {
	s.mu.Lock()
	s.createInputs = append(s.createInputs, req.Msg)
	s.mu.Unlock()
	if s.createErr != nil {
		return nil, s.createErr
	}
	if s.createResp == nil {
		s.createResp = &notesv1.CreateNoteResponse{}
	}
	return connect.NewResponse(s.createResp), nil
}

func (s *notesService) GetNote(_ context.Context, req *connect.Request[notesv1.GetNoteRequest]) (*connect.Response[notesv1.GetNoteResponse], error) {
	s.mu.Lock()
	s.getIDs = append(s.getIDs, req.Msg.Id)
	s.mu.Unlock()
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.getResp == nil {
		s.getResp = &notesv1.GetNoteResponse{}
	}
	return connect.NewResponse(s.getResp), nil
}

func (s *notesService) CountNotes(_ context.Context, req *connect.Request[notesv1.CountNotesRequest]) (*connect.Response[notesv1.CountNotesResponse], error) {
	s.mu.Lock()
	s.countWindows = append(s.countWindows, req.Msg.Window)
	s.mu.Unlock()
	if s.countErr != nil {
		return nil, s.countErr
	}
	if s.countResp == nil {
		s.countResp = &notesv1.CountNotesResponse{}
	}
	return connect.NewResponse(s.countResp), nil
}

func connectAPI(t *testing.T, svc *notesService) http.Handler {
	t.Helper()
	path, handler := notesconnect.NewNotesServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

func note(id, title string) *notesv1.Note {
	ts := timestamppb.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	return &notesv1.Note{
		Id:             id,
		Title:          title,
		CreatedAt:      ts,
		UpdatedAt:      ts,
		AttachmentKeys: []string{"notes/" + id + "/attachments/a.txt"},
	}
}

func TestNotesList_RendersResults(t *testing.T) {
	svc := &notesService{listResp: &notesv1.ListNotesResponse{
		Notes: []*notesv1.Note{note("a", "first"), note("b", "second")},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})

	require.NoError(t, h.list(ctx))
	require.Contains(t, out.String(), "Found 2 note(s).")
	require.Contains(t, out.String(), "first")
	require.Contains(t, out.String(), "second")
	require.Contains(t, out.String(), "attachments=1")
}

// TestNotesList_JSONIsProtoWireShape pins the contract that --json output is
// the proto-typed ListNotesResponse wire shape (round-trips through
// protojson.Unmarshal), with no summary/retrieval_hints wrapper. Machine
// consumers parse the same JSON `cli notes list --json` and `curl /Notes/List`
// produce.
func TestNotesList_JSONIsProtoWireShape(t *testing.T) {
	svc := &notesService{listResp: &notesv1.ListNotesResponse{
		Notes: []*notesv1.Note{note("a", "first")},
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{JSON: true})

	require.NoError(t, h.list(ctx))

	body := out.String()
	require.NotContains(t, body, "summary",
		"--json output must be proto wire shape, not the human ListReport wrapper")
	require.NotContains(t, body, "retrieval_hints",
		"--json output must be proto wire shape, not the human ListReport wrapper")

	// Round-trip through the generated proto type — proves the wire format
	// matches what any Connect-RPC client would parse.
	var got notesv1.ListNotesResponse
	require.NoError(t, protojson.Unmarshal(out.Bytes(), &got))
	require.Len(t, got.Notes, 1)
	require.Equal(t, "a", got.Notes[0].Id)
	require.Equal(t, "first", got.Notes[0].Title)
	require.Equal(t, []string{"notes/a/attachments/a.txt"}, got.Notes[0].AttachmentKeys)
}

func TestNotesList_SurfacesConnectErrors(t *testing.T) {
	svc := &notesService{listErr: connect.NewError(connect.CodeInternal, io.ErrUnexpectedEOF)}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})

	err := h.list(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "internal")
	require.Contains(t, err.Error(), "unexpected EOF")
}

func TestNotesCreate_RequiresTitle(t *testing.T) {
	core := clitest.NewTestApp(t, connectAPI(t, &notesService{}))
	createCmd := findSubcommand(t, registerForTest(t, core), "create")
	_, err := cliapptest.NewTestRunContextFromArgs(createCmd.Args, []string{}, core, nil, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing required flag --title")
}

func TestNotesCreate_CallsConnectClient(t *testing.T) {
	svc := &notesService{createResp: &notesv1.CreateNoteResponse{Note: note("new", "hello")}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "title"}, {Name: "body"}},
	}, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"title": "hello", "body": "world"},
	})

	require.NoError(t, h.create(ctx))
	require.Len(t, svc.createInputs, 1)
	require.Equal(t, "hello", svc.createInputs[0].Title)
	require.Equal(t, "world", svc.createInputs[0].Body)
	require.Contains(t, out.String(), "Created note new.")
	require.Contains(t, out.String(), "hello")
}

// TestNotesCreate_JSONIsProtoWireShape pins the contract that --json output
// is the proto-typed CreateNoteResponse wire shape, not the human
// MutationReport wrapper.
func TestNotesCreate_JSONIsProtoWireShape(t *testing.T) {
	svc := &notesService{createResp: &notesv1.CreateNoteResponse{Note: note("new", "hello")}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "title"}, {Name: "body"}},
	}, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"title": "hello", "body": "world"},
		JSON:  true,
	})

	require.NoError(t, h.create(ctx))

	body := out.String()
	require.NotContains(t, body, "result",
		"--json output must be proto wire shape, not the human MutationReport wrapper")
	require.NotContains(t, body, "next_command",
		"--json output must be proto wire shape, not the human MutationReport wrapper")

	var got notesv1.CreateNoteResponse
	require.NoError(t, protojson.Unmarshal(out.Bytes(), &got))
	require.NotNil(t, got.Note)
	require.Equal(t, "new", got.Note.Id)
	require.Equal(t, "hello", got.Note.Title)
}

// TestNotesGet_JSONIsProtoWireShape pins the same contract for the get path,
// which routes through RenderProtoList.
func TestNotesGet_JSONIsProtoWireShape(t *testing.T) {
	svc := &notesService{getResp: &notesv1.GetNoteResponse{Note: note("abc", "found")}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "id", Required: true}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"id": "abc"},
		JSON:        true,
	})

	require.NoError(t, h.get(ctx))

	body := out.String()
	require.NotContains(t, body, "summary",
		"--json output must be proto wire shape, not the human ListReport wrapper")

	var got notesv1.GetNoteResponse
	require.NoError(t, protojson.Unmarshal(out.Bytes(), &got))
	require.NotNil(t, got.Note)
	require.Equal(t, "abc", got.Note.Id)
	require.Equal(t, "found", got.Note.Title)
}

// TestNotesCount_SendsResolvedWindow pins the measure CLI path: the --window
// token is mapped to the canonical TimeWindow proto and the count is rendered.
func TestNotesCount_SendsResolvedWindow(t *testing.T) {
	svc := &notesService{countResp: &notesv1.CountNotesResponse{Count: 4}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "window"}},
	}, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"window": "last_30d"},
	})

	require.NoError(t, h.count(ctx))
	require.Contains(t, out.String(), "4")

	require.Len(t, svc.countWindows, 1)
	require.Equal(t, measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_LAST_30D, svc.countWindows[0].GetToken())
}

// TestNotesCount_DefaultsWindow proves an omitted --window defaults to
// this_week (matching the manifest measure default) rather than erroring.
func TestNotesCount_DefaultsWindow(t *testing.T) {
	svc := &notesService{countResp: &notesv1.CountNotesResponse{Count: 1}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "window"}},
	}, cliapptest.TestRunContextOptions{})

	require.NoError(t, h.count(ctx))
	require.Len(t, svc.countWindows, 1)
	require.Equal(t, measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_THIS_WEEK, svc.countWindows[0].GetToken())
}

// TestNotesCount_RejectsUnknownWindow proves an unknown token is a usage error,
// never a silent wrong-question answer.
func TestNotesCount_RejectsUnknownWindow(t *testing.T) {
	svc := &notesService{countResp: &notesv1.CountNotesResponse{Count: 0}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "window"}},
	}, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"window": "yesterday"},
	})

	err := h.count(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown time window")
	require.Empty(t, svc.countWindows, "must not call the server on an unresolvable window")
}

func TestNotesGet_ReportsNotFound(t *testing.T) {
	svc := &notesService{getErr: connect.NewError(connect.CodeNotFound, io.ErrUnexpectedEOF)}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "id", Required: true}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"id": "ghost"},
	})

	err := h.get(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not_found")
}

func TestNotesGet_RendersNote(t *testing.T) {
	svc := &notesService{getResp: &notesv1.GetNoteResponse{Note: note("abc", "found")}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "id", Required: true}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"id": "abc"},
	})

	require.NoError(t, h.get(ctx))
	require.Equal(t, []string{"abc"}, svc.getIDs)
	require.Contains(t, out.String(), "Fetched note abc.")
	require.Contains(t, out.String(), "found")
}

func TestNotesGet_RequiresID(t *testing.T) {
	core := clitest.NewTestApp(t, connectAPI(t, &notesService{}))
	getCmd := findSubcommand(t, registerForTest(t, core), "get")
	_, err := cliapptest.NewTestRunContextFromArgs(getCmd.Args, []string{}, core, nil, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing required positional <id>")
}

func TestNotesAttach_UploadsMultipart(t *testing.T) {
	var gotPath, gotBody string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		require.Equal(t, http.MethodPost, r.Method)
		require.NoError(t, r.ParseMultipartForm(32<<20))
		file, header, err := r.FormFile("file")
		require.NoError(t, err)
		defer file.Close()
		require.Equal(t, "note.txt", header.Filename)
		body, err := io.ReadAll(file)
		require.NoError(t, err)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(cliapptest.MustMarshalProto(t, &notesv1.UploadAttachmentResponse{
			Attachment: &notesv1.Attachment{
				Key:       "notes/abc/attachments/note.txt",
				MimeType:  "text/plain",
				SizeBytes: 5,
				NoteId:    "abc",
			},
		}))
	})
	core := clitest.NewTestApp(t, handler)
	h := newHandlers(core)
	tmp := t.TempDir() + "/note.txt"
	require.NoError(t, os.WriteFile(tmp, []byte("hello"), 0o600))
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "id", Required: true}},
		Flags:       []cliapp.Flag{{Name: "file"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"id": "abc"},
		Flags:       map[string]string{"file": tmp},
	})

	require.NoError(t, h.attach(ctx))
	require.True(t, strings.HasSuffix(gotPath, "/api/v1/notes/abc/attachments"), gotPath)
	require.Equal(t, "hello", gotBody)
	require.Contains(t, out.String(), "Attached file to note abc.")
	require.Contains(t, out.String(), "notes/abc/attachments/note.txt")
}

func TestNotesAttach_RequiresFile(t *testing.T) {
	core := clitest.NewTestApp(t, connectAPI(t, &notesService{}))
	attachCmd := findSubcommand(t, registerForTest(t, core), "attach")
	_, err := cliapptest.NewTestRunContextFromArgs(attachCmd.Args, []string{"abc"}, core, nil, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing required flag --file")
}

func registerForTest(t *testing.T, core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	t.Helper()
	manifest := readNotesManifest(t)
	group, err := Register(core, manifest)
	require.NoError(t, err)
	return group
}

func findSubcommand(t *testing.T, group cliapp.SubcommandGroup, name string) cliapp.Command {
	t.Helper()
	for _, sc := range group.Subcommands {
		if sc.Name == name {
			return sc
		}
	}
	t.Fatalf("subcommand %q not registered", name)
	return cliapp.Command{}
}

func TestRegister_Wiring(t *testing.T) {
	core := clitest.NewTestApp(t, connectAPI(t, &notesService{}))
	group := registerForTest(t, core)

	require.Equal(t, "notes", group.Name)
	require.True(t, group.NeedsAPI)
	names := make([]string, 0, len(group.Subcommands))
	for _, sc := range group.Subcommands {
		names = append(names, sc.Name)
	}
	require.ElementsMatch(t, []string{"list", "create", "get", "count", "attach"}, names)
	for _, sc := range group.Subcommands {
		require.NotNil(t, sc.RunCtx, "subcommand %s should use RunCtx", sc.Name)
	}
}
