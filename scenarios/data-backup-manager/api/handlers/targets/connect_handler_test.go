package targets

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"

	"data-backup-manager/internal/sources"
	"data-backup-manager/internal/targets"
	"data-backup-manager/internal/targets/mocks"

	sourcesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/sources"
	targetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/targets"
)

// TestTargetsService_Contract is the per-domain piece of DBM-API-001: it
// exercises every TargetsService RPC against the handler backed by a fake
// service and asserts the request→domain and domain→response translation,
// including the proto SourceKind enum mapping and typed-error codes.
func TestTargetsService_Contract(t *testing.T) {
	ctx := context.Background()

	t.Run("RegisterTarget maps enum and returns wire target", func(t *testing.T) {
		svc := &mocks.FakeService{RegisterOut: targets.Target{
			ID: "tgt-1", Owner: "prompt-manager", Name: "store",
			SourceKind: sources.KindFilesystem, Locator: "store/teams",
			CreatedAt: time.Unix(1700000000, 0).UTC(), UpdatedAt: time.Unix(1700000000, 0).UTC(),
		}}
		h := NewConnectHandler(Deps{Service: svc})

		resp, err := h.RegisterTarget(ctx, connect.NewRequest(&targetsv1.RegisterTargetRequest{
			Owner: "prompt-manager", Name: "store",
			SourceKind: sourcesv1.SourceKind_SOURCE_KIND_FILESYSTEM, Locator: "store/teams",
		}))
		if err != nil {
			t.Fatalf("RegisterTarget: %v", err)
		}
		if len(svc.RegisterInputs) != 1 || svc.RegisterInputs[0].SourceKind != sources.KindFilesystem {
			t.Fatalf("service got wrong input: %+v", svc.RegisterInputs)
		}
		got := resp.Msg.Target
		if got.Id != "tgt-1" || got.SourceKind != sourcesv1.SourceKind_SOURCE_KIND_FILESYSTEM {
			t.Fatalf("response target wrong: %+v", got)
		}
		if got.CreatedAt == nil || got.UpdatedAt == nil {
			t.Fatal("response target missing timestamps")
		}
	})

	t.Run("RegisterTarget surfaces invalid-argument", func(t *testing.T) {
		svc := &mocks.FakeService{RegisterErr: targets.ErrInvalidTarget{Field: "owner", Reason: "required"}}
		h := NewConnectHandler(Deps{Service: svc})
		_, err := h.RegisterTarget(ctx, connect.NewRequest(&targetsv1.RegisterTargetRequest{}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("code = %v, want invalid_argument", connect.CodeOf(err))
		}
	})

	t.Run("GetTarget surfaces not-found", func(t *testing.T) {
		svc := &mocks.FakeService{GetErr: targets.ErrTargetNotFound{ID: "missing"}}
		h := NewConnectHandler(Deps{Service: svc})
		_, err := h.GetTarget(ctx, connect.NewRequest(&targetsv1.GetTargetRequest{Id: "missing"}))
		if connect.CodeOf(err) != connect.CodeNotFound {
			t.Fatalf("code = %v, want not_found", connect.CodeOf(err))
		}
	})

	t.Run("DeregisterTarget returns removed flag", func(t *testing.T) {
		svc := &mocks.FakeService{DeregisterOut: true}
		h := NewConnectHandler(Deps{Service: svc})
		resp, err := h.DeregisterTarget(ctx, connect.NewRequest(&targetsv1.DeregisterTargetRequest{Owner: "o", Name: "n"}))
		if err != nil {
			t.Fatalf("DeregisterTarget: %v", err)
		}
		if !resp.Msg.Removed {
			t.Fatal("removed = false, want true")
		}
		if svc.DeregisterCalls[0] != [2]string{"o", "n"} {
			t.Fatalf("deregister call = %v", svc.DeregisterCalls)
		}
	})

	t.Run("ListTargets maps the collection", func(t *testing.T) {
		svc := &mocks.FakeService{ListOut: []targets.Target{
			{ID: "a", Owner: "o", Name: "1", SourceKind: sources.KindSQLite, Locator: "d.db"},
			{ID: "b", Owner: "o", Name: "2", SourceKind: sources.KindRedis, Locator: "pfx:"},
		}}
		h := NewConnectHandler(Deps{Service: svc})
		resp, err := h.ListTargets(ctx, connect.NewRequest(&targetsv1.ListTargetsRequest{Owner: "o"}))
		if err != nil {
			t.Fatalf("ListTargets: %v", err)
		}
		if svc.ListOwner != "o" {
			t.Fatalf("owner filter not passed: %q", svc.ListOwner)
		}
		if len(resp.Msg.Targets) != 2 || resp.Msg.Targets[1].SourceKind != sourcesv1.SourceKind_SOURCE_KIND_REDIS {
			t.Fatalf("list response wrong: %+v", resp.Msg.Targets)
		}
	})
}
