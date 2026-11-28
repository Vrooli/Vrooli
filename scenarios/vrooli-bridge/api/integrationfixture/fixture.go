//go:build integration

// Package integrationfixture exposes a real Bridge dispatch stack to cross-
// module integration tests. It is available only with the integration tag.
package integrationfixture

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/nodereach"
	"github.com/vrooli/api-core/scheduletest"
	"github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/dispatch/dispatch_v1connect"
	_ "modernc.org/sqlite"

	"vrooli-bridge/handlers/dispatch"
	"vrooli-bridge/internal/audit"
	"vrooli-bridge/internal/auth"
	localdb "vrooli-bridge/internal/database"
	dispatchdomain "vrooli-bridge/internal/dispatch"
	"vrooli-bridge/internal/registry"
	"vrooli-bridge/internal/runs"
)

const NodeID = "br4-integration-node"

type Fixture struct {
	Server *httptest.Server
	DB     *sql.DB
	NodeID string
}

func Start(t *testing.T, path string, effects *atomic.Int32) *Fixture {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	clk := scheduletest.New(time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC))
	for _, schema := range []string{localdb.SystemSchema(), registry.Schema(), runs.Schema(), audit.Schema()} {
		if _, err := db.Exec(schema); err != nil {
			db.Close()
			t.Fatalf("Bridge schema: %v", err)
		}
	}
	repo := registry.NewSQLiteRepository(db, clk)
	reg := registry.NewService(repo, registry.WithGrantValidator(func([]string) error { return nil }))
	if _, err := reg.Get(context.Background(), NodeID); err != nil {
		_, err = repo.Create(context.Background(), registry.Node{ID: NodeID, Name: "BR4 integration fixture", Kind: registry.KindAgent, OS: "linux", Arch: "amd64", DeploymentTier: "vps", Scopes: []string{"vrooli-bridge:write", "vrooli:write"}})
		if err != nil {
			db.Close()
			t.Fatalf("seed fixture registry: %v", err)
		}
	}
	reader := registryTarget{service: reg}
	runSvc := runs.NewService(runs.NewSQLiteRepository(db, clk), clk)
	auditStore := audit.NewSQLiteStore(db, clk)
	service := dispatchdomain.NewService(reader, alwaysOnline{}, runController{runSvc}, auditAdapter{auditStore}, pusher{effects}, dispatchdomain.WithManifest([]string{"scenario status\x00vrooli-bridge:write\x00vrooli:write"}))
	connectHandler := dispatch.NewConnectHandler(dispatch.Deps{Service: service, Logger: log.New(testWriter{t}, "", 0)})
	_, handler := dispatch_v1connect.NewDispatchServiceHandler(connectHandler)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := auth.Identity{OwnerID: "lpbs-admin", Kind: identity.ActorService, Scopes: []string{auth.DispatchServiceScope}}
		handler.ServeHTTP(w, r.WithContext(auth.WithIdentity(r.Context(), id)))
	}))
	f := &Fixture{Server: server, DB: db, NodeID: NodeID}
	t.Cleanup(func() { server.Close(); _ = db.Close() })
	return f
}

func (f *Fixture) Client() *nodereach.Client {
	return nodereach.New(nodereach.Config{BridgeURL: f.Server.URL, Token: "integration-only"})
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Logf("Bridge fixture: %s", p)
	return len(p), nil
}

type registryTarget struct{ service registry.Service }

func (r registryTarget) GetTarget(ctx context.Context, id string) (dispatchdomain.TargetNode, error) {
	n, e := r.service.Get(ctx, id)
	if e != nil {
		return dispatchdomain.TargetNode{}, e
	}
	return dispatchdomain.TargetNode{ID: n.ID, Kind: n.Kind, OS: n.OS, Arch: n.Arch, Scopes: n.Scopes, Revoked: n.Revoked(), DeploymentTier: n.DeploymentTier}, nil
}

type alwaysOnline struct{}

func (alwaysOnline) IsOnline(string) bool     { return true }
func (alwaysOnline) Dispatchable(string) bool { return true }

type runController struct{ service runs.Service }

func (c runController) CreateRun(ctx context.Context, in dispatchdomain.CreateRunInput) (dispatchdomain.CreateRunResult, error) {
	r, e := c.service.Create(ctx, runs.CreateInput{OwnerID: in.OwnerID, RequestID: in.RequestID, RequestFingerprint: in.RequestFingerprint, NodeID: in.NodeID, Scenario: in.Scenario, Verb: in.Verb, Args: in.Args, TimeoutSeconds: in.TimeoutSeconds})
	return dispatchdomain.CreateRunResult{RunID: r.ID, Replayed: r.IdempotentReplay}, e
}
func (c runController) AbortRun(ctx context.Context, id, reason string) error {
	_, e := c.service.Abort(ctx, id, reason)
	return e
}

type auditAdapter struct{ store audit.Store }

func (a auditAdapter) Record(ctx context.Context, e dispatchdomain.Entry) error {
	out := audit.OutcomeRejected
	if e.Accepted {
		out = audit.OutcomeAccepted
	}
	_, err := a.store.Append(ctx, audit.Record{Action: audit.ActionDispatch, Actor: e.Actor, NodeID: e.NodeID, Scenario: e.Scenario, Verb: e.Verb, Args: e.Args, Outcome: out, Detail: e.Detail, RunID: e.RunID})
	return err
}

type pusher struct{ effects *atomic.Int32 }

func (p pusher) PushJob(context.Context, string, dispatchdomain.PushedJob) (int, error) {
	p.effects.Add(1)
	return 1, nil
}
