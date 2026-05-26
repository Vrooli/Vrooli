package targets

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"

	sourcesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/sources"
	targetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/targets"
	targetsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/targets/targets_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	"data-backup-manager/cli/internal/testutil"
)

// stubTargetsService is an in-test TargetsServiceHandler that records the
// request it receives and returns a canned target.
type stubTargetsService struct {
	gotRegister *targetsv1.RegisterTargetRequest
}

func (s *stubTargetsService) RegisterTarget(_ context.Context, req *connect.Request[targetsv1.RegisterTargetRequest]) (*connect.Response[targetsv1.RegisterTargetResponse], error) {
	s.gotRegister = req.Msg
	return connect.NewResponse(&targetsv1.RegisterTargetResponse{Target: &targetsv1.Target{
		Id: "tgt-1", Owner: req.Msg.Owner, Name: req.Msg.Name,
		SourceKind: req.Msg.SourceKind, Locator: req.Msg.Locator,
	}}), nil
}

func (s *stubTargetsService) DeregisterTarget(_ context.Context, _ *connect.Request[targetsv1.DeregisterTargetRequest]) (*connect.Response[targetsv1.DeregisterTargetResponse], error) {
	return connect.NewResponse(&targetsv1.DeregisterTargetResponse{Removed: true}), nil
}

func (s *stubTargetsService) GetTarget(_ context.Context, req *connect.Request[targetsv1.GetTargetRequest]) (*connect.Response[targetsv1.GetTargetResponse], error) {
	return connect.NewResponse(&targetsv1.GetTargetResponse{Target: &targetsv1.Target{Id: req.Msg.Id}}), nil
}

func (s *stubTargetsService) ListTargets(_ context.Context, _ *connect.Request[targetsv1.ListTargetsRequest]) (*connect.Response[targetsv1.ListTargetsResponse], error) {
	return connect.NewResponse(&targetsv1.ListTargetsResponse{}), nil
}

// TestRegisterCommand is the per-domain DBM-CLI-001 check: the register
// command parses its flags (including --kind → proto enum) and calls the
// generated TargetsService client against a real Connect server.
func TestRegisterCommand(t *testing.T) {
	stub := &stubTargetsService{}
	mux := http.NewServeMux()
	path, h := targetsconnect.NewTargetsServiceHandler(stub)
	mux.Handle(path, h)
	app := testutil.NewTestApp(t, mux)

	hs := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "owner"}, {Name: "name"}, {Name: "kind"}, {Name: "locator"},
	}}
	ctx, stdout := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{
			"owner":   "prompt-manager",
			"name":    "store",
			"kind":    "filesystem",
			"locator": "store/teams",
		},
	})

	if err := hs.register(ctx); err != nil {
		t.Fatalf("register: %v", err)
	}
	out := stdout.String()

	if stub.gotRegister == nil {
		t.Fatal("server did not receive RegisterTarget")
	}
	if stub.gotRegister.Owner != "prompt-manager" || stub.gotRegister.Locator != "store/teams" {
		t.Fatalf("request fields wrong: %+v", stub.gotRegister)
	}
	if stub.gotRegister.SourceKind != sourcesv1.SourceKind_SOURCE_KIND_FILESYSTEM {
		t.Fatalf("kind not mapped to enum: %v", stub.gotRegister.SourceKind)
	}
	if !strings.Contains(out, "tgt-1") {
		t.Fatalf("output missing target id: %q", out)
	}
}

// TestParseKind pins the --kind flag → proto enum mapping and its rejection of
// unknown kinds.
func TestParseKind(t *testing.T) {
	cases := map[string]sourcesv1.SourceKind{
		"filesystem":     sourcesv1.SourceKind_SOURCE_KIND_FILESYSTEM,
		"sqlite":         sourcesv1.SourceKind_SOURCE_KIND_SQLITE,
		"postgres":       sourcesv1.SourceKind_SOURCE_KIND_POSTGRES,
		"redis":          sourcesv1.SourceKind_SOURCE_KIND_REDIS,
		"qdrant":         sourcesv1.SourceKind_SOURCE_KIND_QDRANT,
		"object-storage": sourcesv1.SourceKind_SOURCE_KIND_OBJECT_STORAGE,
	}
	for in, want := range cases {
		got, err := parseKind(in)
		if err != nil || got != want {
			t.Errorf("parseKind(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	if _, err := parseKind("nope"); err == nil {
		t.Error("parseKind(\"nope\") should error")
	}
}

// TestRegisterLoadsFromManifest proves the manifest wiring produces the four
// expected subcommands.
func TestRegisterLoadsFromManifest(t *testing.T) {
	manifest := readManifest(t)
	app := &cliapp.ScenarioApp{}
	group, err := Register(app, manifest)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	got := map[string]bool{}
	for _, c := range group.Subcommands {
		got[c.Name] = true
	}
	for _, want := range []string{"register", "deregister", "get", "list"} {
		if !got[want] {
			t.Errorf("missing subcommand %q", want)
		}
	}
}
