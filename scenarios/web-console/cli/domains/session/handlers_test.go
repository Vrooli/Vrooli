package session

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"

	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"
	sessionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions/sessions_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/shared"

	"github.com/vrooli/cli-core/cliapp"
)

// createSchema mirrors the session create flags declared in cli/manifest.json so
// NewTestRunContext can drive create() with the same flag surface as production.
func createSchema() cliapp.ArgSchema {
	return cliapp.ArgSchema{
		Flags: []cliapp.Flag{
			{Name: "shell"},
			{Name: "cols"},
			{Name: "rows"},
			{Name: "backend"},
			{Name: "origin"},
			{Name: "owner"},
			{Name: "label"},
			{Name: "target"},
			{Name: "working-dir"},
			{Name: "launch-command"},
			{Name: "execute-launch-command", Bool: true},
			{Name: "idempotency-key"},
			{Name: "body-file"},
		},
	}
}

// stubClient captures the last CreateRequest and returns a canned session so we
// can assert the CLI maps flags onto the Connect request without a live server.
// Only Create is exercised; the other methods satisfy the interface.
type stubClient struct {
	sessionsconnect.SessionsServiceClient
	lastCreate *sessionsv1.CreateRequest
	lastHeader http.Header
	reply      *sessionsv1.Session
}

func (s *stubClient) Create(_ context.Context, req *connect.Request[sessionsv1.CreateRequest]) (*connect.Response[sessionsv1.CreateResponse], error) {
	s.lastCreate = req.Msg
	s.lastHeader = req.Header().Clone()
	reply := s.reply
	if reply == nil {
		reply = &sessionsv1.Session{Id: "sess-1"}
	}
	return connect.NewResponse(&sessionsv1.CreateResponse{Session: reply}), nil
}

func newCreateCtx(flags map[string]string, boolFlags map[string]bool) cliapp.RunContext {
	return cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema:    createSchema(),
		Flags:     flags,
		BoolFlags: boolFlags,
		Core:      nil,
		Stdout:    &bytes.Buffer{},
	})
}

func TestParseOrigin(t *testing.T) {
	cases := []struct {
		in   string
		want sessionsv1.SessionOrigin
	}{
		{"", sessionsv1.SessionOrigin_SESSION_ORIGIN_UNSPECIFIED},
		{"ui", sessionsv1.SessionOrigin_SESSION_ORIGIN_UI},
		{"programmatic", sessionsv1.SessionOrigin_SESSION_ORIGIN_PROGRAMMATIC},
		{"remote", sessionsv1.SessionOrigin_SESSION_ORIGIN_REMOTE},
		{"  Remote ", sessionsv1.SessionOrigin_SESSION_ORIGIN_REMOTE}, // trims + case-folds
	}
	for _, c := range cases {
		got, err := parseOrigin(c.in)
		if err != nil {
			t.Fatalf("parseOrigin(%q) unexpected error: %v", c.in, err)
		}
		if got != c.want {
			t.Fatalf("parseOrigin(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseOriginUnknown(t *testing.T) {
	_, err := parseOrigin("bogus")
	if err == nil || !strings.Contains(err.Error(), "invalid --origin") {
		t.Fatalf("expected invalid-origin error, got %v", err)
	}
}

func TestOriginString(t *testing.T) {
	cases := map[sessionsv1.SessionOrigin]string{
		sessionsv1.SessionOrigin_SESSION_ORIGIN_UNSPECIFIED:  "unspecified",
		sessionsv1.SessionOrigin_SESSION_ORIGIN_UI:           "ui",
		sessionsv1.SessionOrigin_SESSION_ORIGIN_PROGRAMMATIC: "programmatic",
		sessionsv1.SessionOrigin_SESSION_ORIGIN_REMOTE:       "remote",
	}
	for in, want := range cases {
		if got := originString(in); got != want {
			t.Fatalf("originString(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestCreateUnknownOriginFailsBeforeCall(t *testing.T) {
	stub := &stubClient{}
	h := &handlers{client: stub}
	err := h.create(newCreateCtx(map[string]string{"origin": "sideways"}, nil))
	if err == nil || !strings.Contains(err.Error(), "invalid --origin") {
		t.Fatalf("expected invalid-origin error, got %v", err)
	}
	if stub.lastCreate != nil {
		t.Fatalf("client.Create should not be reached on a bad origin")
	}
}

func TestCreateMapsProvenanceFlags(t *testing.T) {
	stub := &stubClient{}
	h := &handlers{client: stub}
	err := h.create(newCreateCtx(
		map[string]string{
			"origin":         "programmatic",
			"owner":          "agent-manager",
			"label":          "nightly build",
			"target":         "bridge-node:node-a",
			"working-dir":    "/workspaces/demo",
			"launch-command": "echo hi",
		},
		map[string]bool{"execute-launch-command": true},
	))
	if err != nil {
		t.Fatalf("create returned error: %v", err)
	}
	got := stub.lastCreate
	if got == nil {
		t.Fatal("client.Create was not called")
	}
	if got.GetOrigin() != sessionsv1.SessionOrigin_SESSION_ORIGIN_PROGRAMMATIC {
		t.Errorf("Origin = %v, want PROGRAMMATIC", got.GetOrigin())
	}
	if got.GetOwner() != "agent-manager" {
		t.Errorf("Owner = %q, want agent-manager", got.GetOwner())
	}
	if got.GetDisplayLabel() != "nightly build" {
		t.Errorf("DisplayLabel = %q, want %q", got.GetDisplayLabel(), "nightly build")
	}
	if got.GetLaunchCommand() != "echo hi" {
		t.Errorf("LaunchCommand = %q, want %q", got.GetLaunchCommand(), "echo hi")
	}
	if got.GetTargetId() != "bridge-node:node-a" {
		t.Errorf("TargetId = %q, want bridge-node:node-a", got.GetTargetId())
	}
	if got.GetWorkingDir() != "/workspaces/demo" {
		t.Errorf("WorkingDir = %q, want /workspaces/demo", got.GetWorkingDir())
	}
	if !got.GetExecuteLaunchCommand() {
		t.Errorf("ExecuteLaunchCommand = false, want true")
	}
}

func TestCreateMapsIdempotencyKeyToConnectHeader(t *testing.T) {
	stub := &stubClient{}
	h := &handlers{client: stub}
	if err := h.create(newCreateCtx(
		map[string]string{"idempotency-key": "remote-create-1"}, nil,
	)); err != nil {
		t.Fatalf("create returned error: %v", err)
	}
	if got := stub.lastCreate; got == nil {
		t.Fatal("client.Create was not called")
	} else if value := stub.lastHeader.Get("X-Idempotency-Key"); value != "remote-create-1" {
		t.Fatalf("idempotency header = %q, want remote-create-1", value)
	}
}

func TestCreateOmittedOriginSendsUnspecified(t *testing.T) {
	stub := &stubClient{}
	h := &handlers{client: stub}
	err := h.create(newCreateCtx(map[string]string{"shell": "bash"}, nil))
	if err != nil {
		t.Fatalf("create returned error: %v", err)
	}
	if got := stub.lastCreate.GetOrigin(); got != sessionsv1.SessionOrigin_SESSION_ORIGIN_UNSPECIFIED {
		t.Fatalf("omitted origin sent %v, want UNSPECIFIED (server normalizes to PROGRAMMATIC)", got)
	}
	if stub.lastCreate.GetExecuteLaunchCommand() {
		t.Fatalf("ExecuteLaunchCommand should default to false when flag omitted")
	}
}

func TestSessionRowsRenderProvenance(t *testing.T) {
	rows := sessionRows([]*sessionsv1.Session{
		{
			Id:      "abcdef123456",
			Shell:   "bash",
			Backend: "pty",
			Cols:    80,
			Rows:    24,
			Origin:  sessionsv1.SessionOrigin_SESSION_ORIGIN_PROGRAMMATIC,
			Owner:   "agent-manager",
		},
		{
			Id:      "ffffff000000",
			Shell:   "zsh",
			Backend: "tmux",
			Cols:    120,
			Rows:    40,
			Origin:  sessionsv1.SessionOrigin_SESSION_ORIGIN_UI,
		},
		{
			Id:      "remote-session",
			Shell:   "bash",
			Backend: "standard",
			Target:  &sharedv1.Target{Id: "bridge-node:node-a", Label: "Build host"},
		},
	})
	if !strings.Contains(rows[0], "origin=programmatic") || !strings.Contains(rows[0], "owner=agent-manager") {
		t.Fatalf("row 0 missing provenance: %q", rows[0])
	}
	if !strings.Contains(rows[1], "origin=ui") {
		t.Fatalf("row 1 missing origin: %q", rows[1])
	}
	if strings.Contains(rows[1], "owner=") {
		t.Fatalf("row 1 should omit empty owner: %q", rows[1])
	}
	if !strings.Contains(rows[2], "target=Build host (bridge-node:node-a)") {
		t.Fatalf("row 2 missing target: %q", rows[2])
	}
}
