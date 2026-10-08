package restores

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"

	restoresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/restores"
	restoresconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/restores/restores_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	testutil "github.com/vrooli/cli-core/cliapptest"
)

// stubRestoresService is an in-test RestoresServiceHandler that records the request
// it receives and returns canned responses.
type stubRestoresService struct {
	gotVerify *restoresv1.VerifyTargetRequest
}

func (s *stubRestoresService) RestoreTarget(_ context.Context, req *connect.Request[restoresv1.RestoreTargetRequest]) (*connect.Response[restoresv1.RestoreTargetResponse], error) {
	return connect.NewResponse(&restoresv1.RestoreTargetResponse{
		Restore: &restoresv1.Restore{
			Id:            "rst-1",
			TargetId:      req.Msg.TargetId,
			DestinationId: req.Msg.DestinationId,
			SnapshotId:    req.Msg.SnapshotId,
			Mode:          restoresv1.RestoreMode_RESTORE_MODE_RESTORE,
		},
	}), nil
}

func (s *stubRestoresService) VerifyTarget(_ context.Context, req *connect.Request[restoresv1.VerifyTargetRequest]) (*connect.Response[restoresv1.VerifyTargetResponse], error) {
	s.gotVerify = req.Msg
	return connect.NewResponse(&restoresv1.VerifyTargetResponse{
		Restore: &restoresv1.Restore{
			Id:            "rst-2",
			TargetId:      req.Msg.TargetId,
			DestinationId: req.Msg.DestinationId,
			SnapshotId:    req.Msg.SnapshotId,
			Mode:          restoresv1.RestoreMode_RESTORE_MODE_VERIFY,
		},
	}), nil
}

func (s *stubRestoresService) GetRestore(_ context.Context, req *connect.Request[restoresv1.GetRestoreRequest]) (*connect.Response[restoresv1.GetRestoreResponse], error) {
	// Return a terminal status so the CLI's poll-to-terminal loop completes; the
	// request RPCs are async, so restore/verify poll GetRestore for the result.
	return connect.NewResponse(&restoresv1.GetRestoreResponse{Restore: &restoresv1.Restore{
		Id:     req.Msg.Id,
		Status: restoresv1.RestoreStatus_RESTORE_STATUS_VERIFIED,
	}}), nil
}

func (s *stubRestoresService) ListRestores(_ context.Context, _ *connect.Request[restoresv1.ListRestoresRequest]) (*connect.Response[restoresv1.ListRestoresResponse], error) {
	return connect.NewResponse(&restoresv1.ListRestoresResponse{}), nil
}

// TestVerifyCommand is the per-domain check: the verify command parses its flags
// and calls the generated RestoresService client against a real Connect server.
func TestVerifyCommand(t *testing.T) {
	stub := &stubRestoresService{}
	mux := http.NewServeMux()
	path, h := restoresconnect.NewRestoresServiceHandler(stub)
	mux.Handle(path, h)
	app := testutil.NewTestApp(t, mux)

	hs := newHandlers(app)
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "target"}, {Name: "destination"}, {Name: "snapshot"},
	}}
	ctx, stdout := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{
			"target":      "tgt-1",
			"destination": "dst-1",
			"snapshot":    "snap-abc123",
		},
	})

	if err := hs.verify(ctx); err != nil {
		t.Fatalf("verify: %v", err)
	}
	out := stdout.String()

	if stub.gotVerify == nil {
		t.Fatal("server did not receive VerifyTarget")
	}
	if stub.gotVerify.TargetId != "tgt-1" || stub.gotVerify.SnapshotId != "snap-abc123" {
		t.Fatalf("request fields wrong: %+v", stub.gotVerify)
	}
	if !strings.Contains(out, "rst-2") {
		t.Fatalf("output missing restore id: %q", out)
	}
}

// TestRegisterRestoresLoadsFromManifest proves the manifest wiring produces the
// expected subcommands.
func TestRegisterRestoresLoadsFromManifest(t *testing.T) {
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
	for _, want := range []string{"restore", "verify", "get", "list"} {
		if !got[want] {
			t.Errorf("missing subcommand %q", want)
		}
	}
}
