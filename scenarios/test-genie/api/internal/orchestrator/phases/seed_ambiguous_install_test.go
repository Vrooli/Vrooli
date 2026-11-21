package phases

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"
	routingpb "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing/routing_v1connect"
	seedspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/seeds"
	"test-genie/internal/orchestrator/phases/isolation"
	"test-genie/internal/orchestrator/workspace"
)

type uncertainSeedTarget struct {
	routingconnect.UnimplementedRoutingServiceHandler
	installed   atomic.Bool
	refuseClear atomic.Bool
}

func (f *uncertainSeedTarget) InstallTestPool(context.Context, *connect.Request[routingpb.InstallTestPoolRequest]) (*connect.Response[routingpb.InstallTestPoolResponse], error) {
	f.installed.Store(true)
	return nil, connect.NewError(connect.CodeUnavailable, errors.New("install acknowledgement lost after effect"))
}
func (f *uncertainSeedTarget) ClearTestPool(context.Context, *connect.Request[routingpb.ClearTestPoolRequest]) (*connect.Response[routingpb.ClearTestPoolResponse], error) {
	if f.refuseClear.Load() {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("clear unavailable"))
	}
	f.installed.Store(false)
	return connect.NewResponse(&routingpb.ClearTestPoolResponse{Stats: &routingpb.LeaseStats{TestPoolRequests: 3}}), nil
}

type preparedSeedIsolation struct{ result *isolation.Result }

func (p preparedSeedIsolation) Prepare(context.Context) (*isolation.Result, error) {
	return p.result, nil
}

func TestSeedUncertainInstallRetainsBackingUntilExactClearAcknowledgement(t *testing.T) {
	for _, unavailable := range []bool{false, true} {
		t.Run(map[bool]string{false: "clear acknowledged", true: "clear unavailable"}[unavailable], func(t *testing.T) {
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, "api"), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "api", "go.mod"), []byte("module fixture\n\ngo 1.26.0\nrequire modernc.org/sqlite v1.50.1\n"), 0600); err != nil {
				t.Fatal(err)
			}
			backing := filepath.Join(t.TempDir(), "retained.db")
			if err := os.WriteFile(backing, []byte("owner-backed fixture"), 0600); err != nil {
				t.Fatal(err)
			}
			target := &uncertainSeedTarget{}
			target.refuseClear.Store(unavailable)
			path, handler := routingconnect.NewRoutingServiceHandler(target)
			mux := http.NewServeMux()
			mux.Handle(path, handler)
			api := httptest.NewServer(mux)
			defer api.Close()
			var deletes atomic.Int32
			result := &isolation.Result{RunID: "exact-seed-lease", Env: map[string]string{"PLAYBOOKS_SQLITE_DSN": backing}, Cleanup: func(context.Context) error {
				if target.installed.Load() {
					t.Error("backing removed while target still owns the lease")
				}
				deletes.Add(1)
				return os.Remove(backing)
			}}
			previous := isolationManagerFactory
			isolationManagerFactory = func(isolation.Config) isolationProvider { return preparedSeedIsolation{result} }
			defer func() { isolationManagerFactory = previous }()
			session, err := ApplyPlaybooksSeed(t.Context(), workspace.Environment{ScenarioName: "fixture", ScenarioDir: root, AppRoot: root, APIURL: api.URL}, nil, false)
			if err == nil || session == nil {
				t.Fatalf("uncertain setup lost its retained cleanup owner: session=%v err=%v", session, err)
			}
			if session.SeedEvidence().GetState() != seedspb.SeedState_SEED_STATE_FAILED {
				t.Fatal("uncertain setup became successful evidence")
			}
			if unavailable {
				if _, err := os.Stat(backing); err != nil || deletes.Load() != 0 {
					t.Fatalf("unconfirmed clear deleted backing: deletes=%d err=%v", deletes.Load(), err)
				}
				target.refuseClear.Store(false)
				if err := session.Cleanup(t.Context()); err == nil {
					t.Fatal("retry erased the earlier ambiguous cleanup observation")
				}
			}
			if _, err := os.Stat(backing); !os.IsNotExist(err) || deletes.Load() != 1 {
				t.Fatalf("acknowledged clear did not retire backing once: deletes=%d err=%v", deletes.Load(), err)
			}
			if !session.CleanupComplete() || session.SeedEvidence().GetStats().GetTestPoolRequests() != 3 {
				t.Fatal("retained retirement/counters missing")
			}
			_ = session.Cleanup(t.Context())
			if deletes.Load() != 1 {
				t.Fatal("cleanup retry replayed backing deletion")
			}
		})
	}
}
