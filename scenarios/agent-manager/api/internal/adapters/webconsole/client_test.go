package webconsole

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"google.golang.org/protobuf/proto"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"
	sessionsv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions/sessions_v1connect"
	terminalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/terminal"
	terminalv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/terminal/terminal_v1connect"
)

type sessionTestServer struct {
	sessionsv1connect.UnimplementedSessionsServiceHandler
	created          *sessionsv1.CreateRequest
	archived         string
	deleteCalls      int
	missing          bool
	listed           []*sessionsv1.Session
	listErr          error
	getCreatedAt     string
	stable           bool
	creatorOwner     string
	createCalls      int
	createErr        error
	fingerprint      string
	wrongFingerprint bool
	wrongOwner       bool
}

func (s *sessionTestServer) List(_ context.Context, _ *connect.Request[sessionsv1.ListRequest]) (*connect.Response[sessionsv1.ListResponse], error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return connect.NewResponse(&sessionsv1.ListResponse{Sessions: s.listed, SupportsStableCreateIdentity: s.stable, StableCreateOwner: s.creatorOwner}), nil
}

func (s *sessionTestServer) Create(_ context.Context, req *connect.Request[sessionsv1.CreateRequest]) (*connect.Response[sessionsv1.CreateResponse], error) {
	s.created = req.Msg
	s.createCalls++
	if req.Msg.RequestedSessionId != "" {
		raw, _ := proto.MarshalOptions{Deterministic: true}.Marshal(req.Msg)
		digest := sha256.Sum256(raw)
		s.fingerprint = hex.EncodeToString(digest[:])
		if s.createErr != nil {
			return nil, s.createErr
		}
		owner := s.creatorOwner
		if s.wrongOwner {
			owner = "unrelated"
		}
		return connect.NewResponse(&sessionsv1.CreateResponse{Session: &sessionsv1.Session{Id: req.Msg.RequestedSessionId, Owner: owner, CreateFingerprint: s.fingerprint}}), nil
	}
	return connect.NewResponse(&sessionsv1.CreateResponse{Session: &sessionsv1.Session{Id: "session-1", Owner: OwnerAgentManager, Backend: req.Msg.Backend, Origin: sessionsv1.SessionOrigin_SESSION_ORIGIN_PROGRAMMATIC, DisplayLabel: req.Msg.DisplayLabel}}), nil
}

func (s *sessionTestServer) Get(_ context.Context, req *connect.Request[sessionsv1.GetRequest]) (*connect.Response[sessionsv1.GetResponse], error) {
	if s.missing {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	fingerprint, owner := s.fingerprint, s.creatorOwner
	if owner == "" {
		owner = OwnerAgentManager
	}
	if s.wrongFingerprint {
		fingerprint = "different"
	}
	if s.wrongOwner {
		owner = "unrelated"
	}
	return connect.NewResponse(&sessionsv1.GetResponse{Session: &sessionsv1.Session{Id: req.Msg.Id, Owner: owner, CreateFingerprint: fingerprint, Backend: "persistent", Origin: sessionsv1.SessionOrigin_SESSION_ORIGIN_PROGRAMMATIC, DisplayLabel: "drill", CreatedAt: s.getCreatedAt}}), nil
}

// Delete mirrors web-console's permanent-deletion guard. agent-manager must
// never reach it; it is recorded so a regression is visible.
func (s *sessionTestServer) Delete(_ context.Context, req *connect.Request[sessionsv1.DeleteRequest]) (*connect.Response[sessionsv1.DeleteResponse], error) {
	s.deleteCalls++
	if req.Msg.GetConfirmation() != "DELETE:"+req.Msg.GetId() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, nil)
	}
	return connect.NewResponse(&sessionsv1.DeleteResponse{}), nil
}

func (s *sessionTestServer) Archive(_ context.Context, req *connect.Request[sessionsv1.ArchiveRequest]) (*connect.Response[sessionsv1.ArchiveResponse], error) {
	s.archived = req.Msg.Id
	if s.missing {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	return connect.NewResponse(&sessionsv1.ArchiveResponse{Id: req.Msg.Id}), nil
}

type terminalTestServer struct {
	terminalv1connect.UnimplementedTerminalServiceHandler
	mu     sync.Mutex
	inputs []*terminalv1.SendInputRequest
	// composer, when set, renders GetScreen as an agent TUI reacting to text
	// input and Enter requests received so far.
	composer *scriptedComposer
	// enterErr fails every Enter key after the paste.
	enterErr error
}

func (s *terminalTestServer) SendInput(_ context.Context, req *connect.Request[terminalv1.SendInputRequest]) (*connect.Response[terminalv1.SendInputResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inputs = append(s.inputs, req.Msg)
	if req.Msg.GetKeys() != nil && s.enterErr != nil {
		return nil, s.enterErr
	}
	if c := s.composer; c != nil {
		if req.Msg.GetText() != "" {
			c.prompt = req.Msg.GetText()
		} else if req.Msg.GetKeys() != nil {
			c.enterAfterPolls = append(c.enterAfterPolls, c.screenReads)
		}
	}
	return connect.NewResponse(&terminalv1.SendInputResponse{}), nil
}

func (s *terminalTestServer) GetScreen(_ context.Context, req *connect.Request[terminalv1.GetScreenRequest]) (*connect.Response[terminalv1.GetScreenResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.composer == nil {
		return connect.NewResponse(&terminalv1.GetScreenResponse{PlainText: "agent ready"}), nil
	}
	return connect.NewResponse(s.composer.render()), nil
}

// scriptedComposer models an agent TUI's input box on a 40-column screen: text
// becomes visible only after ingestReads screen reads, the first
// swallowEnters Enter presses become newlines inside the composer, and later
// presses submit (unless neverSubmit). A submitted prompt is echoed into the
// history above a working line and an empty composer.
type scriptedComposer struct {
	ingestReads   int
	swallowEnters int
	neverSubmit   bool
	hideCursor    bool
	placeholder   string // shown instead of the prompt, as for a large paste

	prompt          string
	screenReads     int
	enterAfterPolls []int // screenReads observed when each Enter arrived
}

func (c *scriptedComposer) render() *terminalv1.GetScreenResponse {
	c.screenReads++
	rows := []string{"• earlier assistant output", ""}
	visible := c.prompt != "" && c.screenReads > c.ingestReads
	enters := len(c.enterAfterPolls)
	switch {
	case visible && enters > c.swallowEnters && !c.neverSubmit:
		rows = append(rows, wrapRows("› "+c.prompt)...)
		rows = append(rows, "", "• Working (2s • esc to interrupt)", "", "› ")
	case visible:
		body := c.prompt
		if c.placeholder != "" {
			body = c.placeholder
		}
		rows = append(rows, wrapRows("› "+body)...)
		for range enters {
			rows = append(rows, "  ")
		}
	default:
		rows = append(rows, "› ")
	}
	cursorY := len(rows) - 1
	rows = append(rows, "", "  ⏎ send   ⌃J newline")
	resp := &terminalv1.GetScreenResponse{PlainText: strings.Join(rows, "\n"), Cols: 40, Rows: int32(len(rows))}
	if !c.hideCursor {
		resp.Cursor = &terminalv1.Cursor{Y: int32(cursorY)}
	}
	return resp
}

// wrapRows wraps s at 40 columns with the two-space continuation indent TUIs use.
func wrapRows(s string) []string {
	var rows []string
	for len(s) > 40 {
		rows = append(rows, s[:40])
		s = "  " + s[40:]
	}
	return append(rows, s)
}

func newWebConsoleTestClient(t *testing.T, sessions *sessionTestServer, terminal *terminalTestServer) *Client {
	t.Helper()
	mux := http.NewServeMux()
	sessionsPath, sessionsHandler := sessionsv1connect.NewSessionsServiceHandler(sessions)
	terminalPath, terminalHandler := terminalv1connect.NewTerminalServiceHandler(terminal)
	mux.Handle(sessionsPath, sessionsHandler)
	mux.Handle(terminalPath, terminalHandler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return NewClient(server.URL, server.Client())
}

func TestClientMaintainsInteractiveSessionContract(t *testing.T) {
	sessions := &sessionTestServer{}
	terminal := &terminalTestServer{}
	client := newWebConsoleTestClient(t, sessions, terminal)
	ctx := context.Background()

	id, err := client.CreateSession(ctx, CreateSessionParams{LaunchCommand: "codex", Execute: true, DisplayLabel: "drill"})
	if err != nil || id != "session-1" || sessions.created.GetCols() != 120 || sessions.created.GetRows() != 40 || sessions.created.GetBackend() != "persistent" || !sessions.created.GetExecuteLaunchCommand() {
		t.Fatalf("create id=%q req=%+v err=%v", id, sessions.created, err)
	}
	info, err := client.GetSession(ctx, id)
	if err != nil || info.ID != id || info.Owner != OwnerAgentManager || info.Origin != "SESSION_ORIGIN_PROGRAMMATIC" {
		t.Fatalf("info=%+v err=%v", info, err)
	}
	if err := client.SendText(ctx, id, "continue", "agent-manager"); err != nil {
		t.Fatal(err)
	}
	if err := client.Interrupt(ctx, id, "agent-manager"); err != nil {
		t.Fatal(err)
	}
	if screen, err := client.Screen(ctx, id, true); err != nil || screen != "agent ready" {
		t.Fatalf("screen=%q err=%v", screen, err)
	}
	if len(terminal.inputs) != 2 || terminal.inputs[0].GetText() != "continue" || len(terminal.inputs[1].GetKeys().GetKeys()) != 2 {
		t.Fatalf("inputs=%+v", terminal.inputs)
	}
	if err := client.ArchiveSession(ctx, id); err != nil || sessions.archived != id {
		t.Fatalf("archive=%q err=%v", sessions.archived, err)
	}

	sessions.missing = true
	if _, err := client.GetSession(ctx, id); err != ErrSessionNotFound {
		t.Fatalf("missing get err=%v", err)
	}
	if err := client.ArchiveSession(ctx, id); err != nil {
		t.Fatalf("missing archive err=%v, want idempotent success", err)
	}
	if sessions.deleteCalls != 0 {
		t.Fatalf("client reached permanent Delete %d times; agent-manager only archives", sessions.deleteCalls)
	}
}

// newPromptTestClient returns a client whose screen polls never sleep: the
// scripted composer advances per screen read, not per wall-clock tick.
func newPromptTestClient(t *testing.T, terminal *terminalTestServer) *Client {
	t.Helper()
	client := newWebConsoleTestClient(t, &sessionTestServer{}, terminal)
	client.wait = func(ctx context.Context, _ time.Duration) error { return ctx.Err() }
	return client
}

const longDirective = "D1 is recorded in E17.md; stop broad validation and run the J02 journey only, then report"

func TestSendPromptWaitsForSlowIngestBeforeEnter(t *testing.T) {
	composer := &scriptedComposer{ingestReads: 4}
	terminal := &terminalTestServer{composer: composer}
	client := newPromptTestClient(t, terminal)

	got, err := client.SendPrompt(context.Background(), "session-1", longDirective, "agent-manager:run-1")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Verified || got.EnterPresses != 1 {
		t.Fatalf("submission = %+v, want one verified Enter", got)
	}
	if len(composer.enterAfterPolls) != 1 || composer.enterAfterPolls[0] <= composer.ingestReads {
		t.Fatalf("Enter sent after %v screen reads; the paste was only visible after %d", composer.enterAfterPolls, composer.ingestReads)
	}
	first := terminal.inputs[0]
	if first.GetText() != longDirective || !first.GetIsPaste() || terminal.inputs[1].GetKeys().GetKeys()[0].GetName() != "enter" {
		t.Fatalf("prompt was not pasted then submitted with a named Enter: %+v", terminal.inputs)
	}
}

func TestSendCommandSubmitsNativeGoalAsLiteralTextThenNamedEnter(t *testing.T) {
	const goalCommand = "/goal finish the verified task"
	composer := &scriptedComposer{ingestReads: 4}
	terminal := &terminalTestServer{composer: composer}
	client := newPromptTestClient(t, terminal)

	got, err := client.SendCommand(context.Background(), "session-1", goalCommand, "agent-manager:run-goal")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Verified || got.EnterPresses != 1 {
		t.Fatalf("goal submission = %+v, want one verified Enter", got)
	}
	if len(terminal.inputs) != 2 {
		t.Fatalf("SendInput calls = %d, want literal text followed by Enter", len(terminal.inputs))
	}
	command := terminal.inputs[0]
	if command.GetText() != goalCommand || strings.ContainsAny(command.GetText(), "\r\n") || command.GetIsPaste() || command.GetKeys() != nil {
		t.Fatalf("goal command payload = %+v; want exact single-line non-paste slash command", command)
	}
	if command.GetSource() != "agent-manager:run-goal" {
		t.Fatalf("goal command source = %q, want agent-manager:run-goal", command.GetSource())
	}
	enter := terminal.inputs[1]
	if enter.GetIsPaste() || enter.GetText() != "" || enter.GetKeys() == nil || len(enter.GetKeys().GetKeys()) != 1 || enter.GetKeys().GetKeys()[0].GetName() != "enter" {
		t.Fatalf("goal submit input = %+v; want separate named Enter", enter)
	}
	if len(composer.enterAfterPolls) != 1 || composer.enterAfterPolls[0] <= composer.ingestReads {
		t.Fatalf("goal Enter was sent before slow text ingest completed: reads=%v ingest=%d", composer.enterAfterPolls, composer.ingestReads)
	}
}

func TestSendCommandRejectsEmbeddedNewlineBeforeTransport(t *testing.T) {
	terminal := &terminalTestServer{composer: &scriptedComposer{}}
	client := newPromptTestClient(t, terminal)
	for _, command := range []string{"/goal first\nsecond", "/goal first\rsecond", "\n"} {
		if _, err := client.SendCommand(context.Background(), "session-1", command, "agent-manager:run-goal"); !errors.Is(err, ErrCommandMustBeSingleLine) {
			t.Fatalf("SendCommand(%q) err=%v, want ErrCommandMustBeSingleLine", command, err)
		}
	}
	if len(terminal.inputs) != 0 {
		t.Fatalf("invalid command reached transport: %+v", terminal.inputs)
	}
}

func TestSendCommandRetriesEnterSwallowedAsNewline(t *testing.T) {
	composer := &scriptedComposer{swallowEnters: 1}
	terminal := &terminalTestServer{composer: composer}
	client := newPromptTestClient(t, terminal)
	got, err := client.SendCommand(context.Background(), "session-1", "/goal finish the verified task", "agent-manager:run-goal")
	if err != nil || !got.Verified || got.EnterPresses != 2 {
		t.Fatalf("submission = %+v err=%v, want verified after second named Enter", got, err)
	}
	if len(terminal.inputs) != 3 || terminal.inputs[0].GetIsPaste() {
		t.Fatalf("command transport = %+v, want one literal command then two Enter keys", terminal.inputs)
	}
}

func TestSendCommandWithoutComposerSubmissionReturnsNotSubmitted(t *testing.T) {
	composer := &scriptedComposer{neverSubmit: true}
	terminal := &terminalTestServer{composer: composer}
	client := newPromptTestClient(t, terminal)

	got, err := client.SendCommand(context.Background(), "session-1", "/goal finish the verified task", "agent-manager:run-goal")
	if !errors.Is(err, ErrCommandNotSubmitted) {
		t.Fatalf("err = %v, want ErrCommandNotSubmitted when slash command remains in composer", err)
	}
	if got.Verified || got.EnterPresses != len(promptEnterWindows) {
		t.Fatalf("goal submission = %+v, want bounded unverified attempts", got)
	}
	if len(terminal.inputs) != 1+len(promptEnterWindows) || terminal.inputs[0].GetIsPaste() {
		t.Fatalf("goal transport inputs = %d, want one literal command plus bounded Enter attempts", len(terminal.inputs))
	}
}

func TestSendPromptRetriesEnterSwallowedAsNewline(t *testing.T) {
	composer := &scriptedComposer{swallowEnters: 1}
	client := newPromptTestClient(t, &terminalTestServer{composer: composer})

	got, err := client.SendPrompt(context.Background(), "session-1", longDirective, "agent-manager:run-1")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Verified || got.EnterPresses != 2 {
		t.Fatalf("submission = %+v, want verified after a second Enter", got)
	}
}

func TestSendPromptReportsComposerThatNeverSubmits(t *testing.T) {
	composer := &scriptedComposer{neverSubmit: true}
	client := newPromptTestClient(t, &terminalTestServer{composer: composer})

	got, err := client.SendPrompt(context.Background(), "session-1", longDirective, "agent-manager:run-1")
	if !errors.Is(err, ErrPromptNotSubmitted) {
		t.Fatalf("err = %v, want ErrPromptNotSubmitted", err)
	}
	if got.Verified || got.EnterPresses != len(promptEnterWindows) {
		t.Fatalf("submission = %+v, want %d unverified Enter presses", got, len(promptEnterWindows))
	}
}

func TestSendPromptConfirmsCollapsedLargePaste(t *testing.T) {
	composer := &scriptedComposer{placeholder: "[Pasted Content 2048 chars]", swallowEnters: 1}
	client := newPromptTestClient(t, &terminalTestServer{composer: composer})

	got, err := client.SendPrompt(context.Background(), "session-1", strings.Repeat("large handoff ", 150), "agent-manager:run-1")
	if err != nil || !got.Verified || got.EnterPresses != 2 {
		t.Fatalf("submission = %+v err=%v, want the placeholder tracked to a verified submit", got, err)
	}
}

func TestSendPromptWithoutObservableComposerIsUnverified(t *testing.T) {
	composer := &scriptedComposer{hideCursor: true}
	client := newPromptTestClient(t, &terminalTestServer{composer: composer})

	got, err := client.SendPrompt(context.Background(), "session-1", longDirective, "agent-manager:run-1")
	if err != nil || got.Verified || got.EnterPresses != 1 {
		t.Fatalf("submission = %+v err=%v, want one Enter reported unverified", got, err)
	}
}

func TestSendPromptEnterTransportFailureIsNotNotSubmitted(t *testing.T) {
	terminal := &terminalTestServer{composer: &scriptedComposer{}, enterErr: connect.NewError(connect.CodeUnavailable, nil)}
	client := newPromptTestClient(t, terminal)

	_, err := client.SendPrompt(context.Background(), "session-1", longDirective, "agent-manager:run-1")
	if err == nil || errors.Is(err, ErrPromptNotSubmitted) {
		t.Fatalf("err = %v, want a transport error distinct from ErrPromptNotSubmitted", err)
	}
}

func TestSessionReadsProjectOwnershipAndCreationTime(t *testing.T) {
	cases := []struct {
		name string
		wire string
		want time.Time
	}{
		{"whole-second UTC", "2026-10-01T08:30:00Z", time.Date(2026, 10, 1, 8, 30, 0, 0, time.UTC)},
		{"nanosecond UTC", "2026-08-25T12:34:56.000000007Z", time.Date(2026, 8, 25, 12, 34, 56, 7, time.UTC)},
		{"positive offset previous day", "2026-08-25T00:04:05.12+05:30", time.Date(2026, 8, 24, 18, 34, 5, 120000000, time.UTC)},
		{"negative offset next day", "2026-08-25T23:34:56.000000007-07:00", time.Date(2026, 8, 26, 6, 34, 56, 7, time.UTC)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sessions := &sessionTestServer{getCreatedAt: tc.wire, listed: []*sessionsv1.Session{
				{Id: "am-1", Owner: OwnerAgentManager, Origin: sessionsv1.SessionOrigin_SESSION_ORIGIN_PROGRAMMATIC, DisplayLabel: "nooch/E7", CreatedAt: tc.wire},
			}}
			client := newWebConsoleTestClient(t, sessions, &terminalTestServer{})
			got, err := client.GetSession(context.Background(), "am-1")
			if err != nil {
				t.Fatal(err)
			}
			if got.ID != "am-1" || got.Owner != OwnerAgentManager || got.Backend != "persistent" || got.Origin != OriginProgrammatic || got.DisplayLabel != "drill" || !got.CreatedAt.Equal(tc.want) {
				t.Fatalf("GetSession=%+v, want creation instant %s", got, tc.want)
			}
			listed, err := client.ListSessions(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if len(listed) != 1 {
				t.Fatalf("ListSessions=%+v, want one identified session", listed)
			}
			if listed[0].ID != "am-1" || listed[0].Owner != OwnerAgentManager || listed[0].Origin != OriginProgrammatic || listed[0].DisplayLabel != "nooch/E7" || !listed[0].CreatedAt.Equal(tc.want) {
				t.Fatalf("ListSessions=%+v, want creation instant %s", listed[0], tc.want)
			}
		})
	}

	t.Run("unknown time and list outage", func(t *testing.T) {
		sessions := &sessionTestServer{getCreatedAt: "not-a-time", listed: []*sessionsv1.Session{
			{Id: "operator-1", Origin: sessionsv1.SessionOrigin_SESSION_ORIGIN_UI, CreatedAt: "not-a-time"},
			{Id: ""},
		}}
		client := newWebConsoleTestClient(t, sessions, &terminalTestServer{})
		got, err := client.GetSession(context.Background(), "am-1")
		if err != nil || !got.CreatedAt.IsZero() {
			t.Fatalf("GetSession=%+v err=%v, want unknown creation time", got, err)
		}
		listed, err := client.ListSessions(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if len(listed) != 1 || listed[0].ID != "operator-1" || listed[0].Origin == OriginProgrammatic || !listed[0].CreatedAt.IsZero() {
			t.Fatalf("ListSessions=%+v, want one non-programmatic session with unknown creation time", listed)
		}
		sessions.listErr = connect.NewError(connect.CodeUnavailable, nil)
		if _, err := client.ListSessions(context.Background()); err == nil {
			t.Fatal("list outage was not reported")
		}
	})
}

// agent-manager must never permanently delete a web-console session: release
// and Stop archive, so conversation evidence survives (web-console's archive
// retention owns disposal). Guard the seam so permanent deletion cannot return.
func TestSessionSeamHasNoPermanentDelete(t *testing.T) {
	for _, typ := range []reflect.Type{reflect.TypeFor[SessionController](), reflect.TypeFor[*Client]()} {
		if _, ok := typ.MethodByName("DeleteSession"); ok {
			t.Fatalf("%s exposes DeleteSession; agent-manager must archive sessions instead", typ)
		}
	}
}

func TestReservedCreateOldPeerAndUncertainResponse(t *testing.T) {
	for _, tc := range []struct {
		name                                                     string
		stable, missing, wrongFingerprint, wrongOwner, noCreator bool
		createErr                                                error
		want                                                     bool
		writes                                                   int
	}{
		{name: "old peer zero writes", writes: 0},
		{name: "missing verified creator zero writes", stable: true, noCreator: true, writes: 0},
		{name: "acknowledged changed creator refuses", stable: true, wrongOwner: true, writes: 1},
		{name: "acknowledged", stable: true, want: true, writes: 1},
		{name: "lost response exact live identity", stable: true, createErr: connect.NewError(connect.CodeUnavailable, errors.New("lost acknowledgement")), want: true, writes: 1},
		{name: "lost response missing runtime", stable: true, missing: true, createErr: connect.NewError(connect.CodeUnavailable, errors.New("lost acknowledgement")), writes: 1},
		{name: "lost response changed fingerprint", stable: true, wrongFingerprint: true, createErr: connect.NewError(connect.CodeUnavailable, errors.New("lost acknowledgement")), writes: 1},
		{name: "lost response changed owner", stable: true, wrongOwner: true, createErr: connect.NewError(connect.CodeUnavailable, errors.New("lost acknowledgement")), writes: 1},
		{name: "definite collision", stable: true, createErr: connect.NewError(connect.CodeAlreadyExists, errors.New("collision")), writes: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			owner := &sessionTestServer{stable: tc.stable, creatorOwner: "verified-service-subject", missing: tc.missing, wrongFingerprint: tc.wrongFingerprint, wrongOwner: tc.wrongOwner, createErr: tc.createErr}
			if tc.noCreator {
				owner.creatorOwner = ""
			}
			c := newWebConsoleTestClient(t, owner, &terminalTestServer{})
			id, err := c.CreateSession(t.Context(), CreateSessionParams{RequestedSessionID: "c06dce6e-c663-4898-bf98-d66f13013197", LaunchCommand: "bounded"})
			if (err == nil) != tc.want || tc.want && id != "c06dce6e-c663-4898-bf98-d66f13013197" || owner.createCalls != tc.writes {
				t.Fatalf("id=%q err=%v writes=%d", id, err, owner.createCalls)
			}
		})
	}
}
