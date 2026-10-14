package execution

import (
	"context"
	"database/sql"
	"errors"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/projectmeta"
	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing/routing_v1connect"
	_ "modernc.org/sqlite"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

type heartbeatRoutingClient struct {
	heartbeatErr error
}

func TestFilePersistenceClassificationSurvivesUnrelatedStorageFailure(t *testing.T) {
	detail, err := structpb.NewStruct(map[string]any{"file_persisting": true})
	if err != nil {
		t.Fatal(err)
	}
	native, err := anypb.New(detail)
	if err != nil {
		t.Fatal(err)
	}
	resp := &scenariovalidationv1.ValidateScenarioResponse{
		Status:       scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED,
		NativeDetail: native,
	}

	got, err := filePersistenceClassification(resp)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Fatal("file_persisting = false, want true despite unrelated storage findings")
	}
}

func (c heartbeatRoutingClient) InstallTestPool(context.Context, *connect.Request[routingv1.InstallTestPoolRequest]) (*connect.Response[routingv1.InstallTestPoolResponse], error) {
	return connect.NewResponse(&routingv1.InstallTestPoolResponse{}), nil
}

func (c heartbeatRoutingClient) ClearTestPool(context.Context, *connect.Request[routingv1.ClearTestPoolRequest]) (*connect.Response[routingv1.ClearTestPoolResponse], error) {
	return connect.NewResponse(&routingv1.ClearTestPoolResponse{}), nil
}

func (c heartbeatRoutingClient) HeartbeatTestPool(context.Context, *connect.Request[routingv1.HeartbeatTestPoolRequest]) (*connect.Response[routingv1.HeartbeatTestPoolResponse], error) {
	if c.heartbeatErr != nil {
		return nil, c.heartbeatErr
	}
	return connect.NewResponse(&routingv1.HeartbeatTestPoolResponse{}), nil
}

func TestRoutingLeaseHeartbeatFailureSignalsHealthAndEvidence(t *testing.T) {
	heartbeatErr := errors.New("routing service unavailable")
	lease := &routingLease{
		client:     heartbeatRoutingClient{heartbeatErr: heartbeatErr},
		leaseID:    "lease-1",
		healthDone: make(chan struct{}),
		evidence:   IsolationEvidence{Installed: true, LeaseID: "lease-1"},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go lease.heartbeat(ctx, 3*time.Millisecond)
	select {
	case <-lease.Done():
	case <-time.After(250 * time.Millisecond):
		t.Fatal("lease heartbeat failure did not signal health loss")
	}

	if !errors.Is(lease.Err(), heartbeatErr) {
		t.Fatalf("health error = %v, want %v", lease.Err(), heartbeatErr)
	}
	if got := lease.Evidence().HeartbeatError; got != heartbeatErr.Error() {
		t.Fatalf("heartbeat evidence = %q, want %q", got, heartbeatErr.Error())
	}
}

// In-process integration: real provider-owned SQL/file lease and HTTP target;
// BAS browser transport is substituted, so this does not accept Planner D6.
func TestSelectedExecutionOwnsRealHTTPLeaseHeartbeatAndZeroPrimaryWrites(t *testing.T) {
	ctx := context.Background()
	modeRoot := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(modeRoot, ".vrooli"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(modeRoot, ".vrooli", "service.json"), []byte(`{"mode":"development"}`), 0644))
	projectmeta.SetStartDirForTesting(modeRoot)
	defer projectmeta.SetStartDirForTesting("")
	routed, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(t.TempDir(), "primary.db")})
	require.NoError(t, err)
	defer routed.Close()
	_, err = routed.Primary().Exec(`CREATE TABLE observations (id TEXT)`)
	require.NoError(t, err)
	routed.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		_, err := pool.ExecContext(ctx, `CREATE TABLE observations (id TEXT)`)
		return err
	})
	root := t.TempDir()
	roots := filerouting.New(storage.Paths{ConfigDir: filepath.Join(root, "config"), DataDir: filepath.Join(root, "data"), CacheDir: filepath.Join(root, "cache"), LogsDir: filepath.Join(root, "logs"), StateDir: filepath.Join(root, "state")})
	mux := http.NewServeMux()
	require.True(t, devrouting.RegisterWithFileRoots(mux, routed, roots))
	mux.Handle("/probe", apihttp.TestModeMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !database.IsTestMode(r.Context()) {
			http.Error(w, "missing test mode", 400)
			return
		}
		if _, err := routed.ExecContext(r.Context(), `INSERT INTO observations VALUES ('selected-case')`); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		path, err := roots.Pick(r.Context(), storage.ClassData)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if err := os.WriteFile(filepath.Join(path, "observation.txt"), []byte("selected-case"), 0644); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		roots.RecordWrite(r.Context())
		w.WriteHeader(204)
	})))
	heartbeat := make(chan struct{}, 1)
	target := httptest.NewServer(mux)
	defer target.Close()
	isolation := NewRoutingIsolation()
	isolation.NewClient = func(base string) routingClient {
		return heartbeatAckClient{routingClient: routing_v1connect.NewRoutingServiceClient(http.DefaultClient, base), ack: heartbeat}
	}
	isolation.ResolveTarget = func(_ context.Context, scenario string) (string, error) {
		require.Equal(t, "leased-fixture", scenario)
		return target.URL, nil
	}
	isolation.RequiresFileRoots = func(context.Context, string) (bool, error) { return true, nil }
	isolation.TTL = 600 * time.Millisecond
	scenarioPath := t.TempDir()
	for _, name := range []string{"selected", "unselected"} {
		writeFixtureJSON(t, filepath.Join(scenarioPath, "bas", "cases", name+".json"), map[string]any{"metadata": map[string]any{"name": name, "execution_mode": "observer", "labels": map[string]any{"reset": "none", "routed_isolation": "true"}}, "nodes": []any{}})
	}
	client := &fakeBASClient{}
	client.onExecute = func() {
		req, err := http.NewRequest(http.MethodPost, target.URL+"/probe", nil)
		require.NoError(t, err)
		for k, v := range client.lastRequest.Parameters.ExtraHeaders {
			req.Header.Set(k, v)
		}
		resp, err := target.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, 204, resp.StatusCode)
		select {
		case <-heartbeat:
		case <-time.After(3 * time.Second):
			t.Fatal("provider heartbeat not observed")
		}
	}
	report, err := NewService(client).RunScenario(ctx, "leased-fixture", scenarioPath, Options{IncludeExecution: true, RunID: "selected-real-lease", Selector: AssetSelector{CasesProvided: true, CasePaths: []string{"bas/cases/selected.json"}}, Isolation: isolation})
	require.NoError(t, err)
	require.Equal(t, 1, client.executeCalls)
	require.Equal(t, "selected", client.lastRequest.Name)
	require.True(t, report.Isolation.Installed)
	require.Equal(t, "selected-real-lease", report.Isolation.LeaseID)
	require.Greater(t, report.Isolation.TestPoolRequests, int64(0))
	require.Equal(t, int64(1), report.Isolation.TestRootWrites)
	require.Zero(t, report.Isolation.PrimaryDuringTestModeRequests)
	require.Zero(t, report.Isolation.PrimaryRootWritesDuringTestMode)
	require.Empty(t, report.Isolation.HeartbeatError)
	require.Empty(t, report.Isolation.ClearError)
	require.False(t, routed.HasTestPool())
	require.False(t, roots.HasTestRoots())
	var primaryRows int
	require.NoError(t, routed.Primary().QueryRow(`SELECT COUNT(*) FROM observations`).Scan(&primaryRows))
	require.Zero(t, primaryRows)
	_, err = os.Stat(filepath.Join(root, "data", "observation.txt"))
	require.True(t, os.IsNotExist(err), "selected operation must not write the primary file root")
}

// Observe the production heartbeat after the real RPC has returned successfully,
// so cleanup cannot cancel an in-flight response before the assertion.
type heartbeatAckClient struct {
	routingClient
	ack chan struct{}
}

func (c heartbeatAckClient) HeartbeatTestPool(ctx context.Context, req *connect.Request[routingv1.HeartbeatTestPoolRequest]) (*connect.Response[routingv1.HeartbeatTestPoolResponse], error) {
	resp, err := c.routingClient.HeartbeatTestPool(ctx, req)
	if err == nil {
		select {
		case c.ack <- struct{}{}:
		default:
		}
	}
	return resp, err
}
