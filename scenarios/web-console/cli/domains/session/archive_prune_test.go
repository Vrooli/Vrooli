package session

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"
	sessionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions/sessions_v1connect"
)

type pruneStub struct {
	sessionsconnect.SessionsServiceClient
	last *sessionsv1.PruneArchiveRequest
}

func (s *pruneStub) PruneArchive(_ context.Context, req *connect.Request[sessionsv1.PruneArchiveRequest]) (*connect.Response[sessionsv1.PruneArchiveResponse], error) {
	s.last = req.Msg
	return connect.NewResponse(&sessionsv1.PruneArchiveResponse{DryRun: !req.Msg.GetApply()}), nil
}

// --apply is a bool flag; the request must carry Apply only when it is set.
func TestArchivePruneApplyFlagReachesRequest(t *testing.T) {
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "apply", Bool: true}}}
	for _, apply := range []bool{false, true} {
		stub := &pruneStub{}
		h := &handlers{client: stub}
		opts := cliapp.TestRunContextOptions{Schema: schema, JSON: true}
		if apply {
			opts.BoolFlags = map[string]bool{"apply": true}
		}
		if err := h.archivePrune(cliapp.NewTestRunContext(opts)); err != nil {
			t.Fatal(err)
		}
		if stub.last == nil || stub.last.GetApply() != apply {
			t.Fatalf("--apply=%v sent Apply=%v", apply, stub.last.GetApply())
		}
	}
}
