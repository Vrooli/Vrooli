package execution

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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

func TestElectronIsolationInstallsOnBoundPackageAPIOrigin(t *testing.T) {
	dbRoot := t.TempDir()
	require.NoError(t, os.Chmod(dbRoot, 0o700))
	target := &ElectronTarget{TargetID: "native-target", RendererID: "native-renderer", ContextID: "native-cell", ScenarioName: "device-studio", ArtifactDigest: "sha256:artifact", CDPEndpoint: "http://127.0.0.1:9222", CDPTransport: "http", RendererURL: "http://127.0.0.1:41235/projects?tab=recent#saved", APIURL: "http://127.0.0.1:41236", TestDatabaseRoot: dbRoot, BundledRuntime: true}
	isolation, err := NewElectronRoutingIsolation(target)
	require.NoError(t, err)
	// Freeze the owner binding, independent of later caller mutations.
	target.APIURL = "http://127.0.0.1:19349/"
	target.ScenarioName = "other"
	var installedAt string
	client := &installAckRoutingClient{response: connect.NewResponse(&routingv1.InstallTestPoolResponse{ActiveLeaseId: "native-lease", FileRootsInstalled: true})}
	isolation.NewClient = func(base string) routingClient { installedAt = base; return client }
	created, cleaned := false, false
	isolation.NewDSN = func(string) (string, func(), error) {
		created = true
		return "file:private.db", func() { cleaned = true }, nil
	}
	isolation.RequiresFileRoots = func(_ context.Context, scenario string) (bool, error) {
		require.Equal(t, "device-studio", scenario)
		return true, nil
	}
	isolation.TTL = time.Hour
	lease, err := isolation.Acquire(context.Background(), "device-studio", "native-lease")
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, "http://127.0.0.1:41236", installedAt)
	require.True(t, lease.Evidence().Installed)
	lease.Close(context.Background())
	require.True(t, cleaned)
	require.Equal(t, 1, client.clearCalls)
	created = false
	_, err = isolation.Acquire(context.Background(), "other", "wrong-target")
	require.ErrorContains(t, err, "does not match")
	require.False(t, created, "scenario mismatch must fail before creating storage")
}

func TestElectronIsolationRequiresBoundAPIForBundledRuntime(t *testing.T) {
	target := &ElectronTarget{TargetID: "target", RendererID: "renderer", ContextID: "cell", ScenarioName: "sample", ArtifactDigest: "sha256:artifact", CDPEndpoint: "http://127.0.0.1:9222", CDPTransport: "http", RendererURL: "http://127.0.0.1:41235/", BundledRuntime: true}
	_, err := NewElectronRoutingIsolation(target)
	require.ErrorContains(t, err, "target-owned API endpoint")
	target.BundledRuntime = false
	isolation, err := NewElectronRoutingIsolation(target)
	require.NoError(t, err)
	resolved, err := isolation.ResolveTarget(context.Background(), "sample")
	require.NoError(t, err)
	require.Equal(t, "http://127.0.0.1:41235", resolved, "legacy external deployments retain their existing renderer routing")
}

func TestBundledElectronSQLiteUsesTargetPrivateRootAndCleansUp(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Chmod(root, 0o700))
	target := &ElectronTarget{TargetID: "target", RendererID: "renderer", ContextID: "cell", ScenarioName: "device-studio", ArtifactDigest: "sha256:artifact", CDPEndpoint: "http://127.0.0.1:9222", CDPTransport: "http", RendererURL: "http://127.0.0.1:41235/", APIURL: "http://127.0.0.1:41236", TestDatabaseRoot: root, BundledRuntime: true}
	isolation, err := NewElectronRoutingIsolation(target)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, "downloads"), isolation.DownloadRoot)
	downloadInfo, err := os.Lstat(isolation.DownloadRoot)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o700), downloadInfo.Mode().Perm())
	client := &installAckRoutingClient{response: connect.NewResponse(&routingv1.InstallTestPoolResponse{ActiveLeaseId: "offline-lease"})}
	isolation.NewClient = func(string) routingClient { return client }
	isolation.RequiresFileRoots = func(context.Context, string) (bool, error) { return false, nil }
	lease, err := isolation.Acquire(context.Background(), "device-studio", "offline-lease")
	require.NoError(t, err)
	require.Contains(t, client.installDSN, filepath.Join(root, "workflow-test-db-"), "the disposable DB must live under the target-owned root")
	db, err := sql.Open("sqlite", client.installDSN)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE routed_write (value TEXT); INSERT INTO routed_write VALUES ('private-root')`)
	require.NoError(t, err)
	require.NoError(t, db.Close())
	lease.Close(context.Background())
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "downloads", entries[0].Name(), "confirmed lease cleanup removes its exact DB but retains the target-owned download directory")
}

func TestBundledElectronRejectsUnownedOrUnsafeDatabaseRoots(t *testing.T) {
	for name, root := range map[string]string{"missing": "", "relative": "tmp/test-db", "nonexistent": filepath.Join(t.TempDir(), "missing")} {
		t.Run(name, func(t *testing.T) {
			_, err := NewElectronRoutingIsolation(&ElectronTarget{TargetID: "target", RendererID: "renderer", ContextID: "cell", ScenarioName: "sample", ArtifactDigest: "sha256:artifact", CDPEndpoint: "http://127.0.0.1:9222", CDPTransport: "http", RendererURL: "http://127.0.0.1:41235/", APIURL: "http://127.0.0.1:41236", TestDatabaseRoot: root, BundledRuntime: true})
			require.Error(t, err)
		})
	}
	unsafe := t.TempDir()
	require.NoError(t, os.Chmod(unsafe, 0o755))
	_, err := NewElectronRoutingIsolation(&ElectronTarget{TargetID: "target", RendererID: "renderer", ContextID: "cell", ScenarioName: "sample", ArtifactDigest: "sha256:artifact", CDPEndpoint: "http://127.0.0.1:9222", CDPTransport: "http", RendererURL: "http://127.0.0.1:41235/", APIURL: "http://127.0.0.1:41236", TestDatabaseRoot: unsafe, BundledRuntime: true})
	require.ErrorContains(t, err, "mode 0700")
}

func TestElectronIsolationRejectsUnboundOrNonlocalOrigins(t *testing.T) {
	_, err := NewElectronRoutingIsolation(nil)
	require.Error(t, err)
	for _, origin := range []string{"", "app://device-studio", "https://127.0.0.1:4000", "http://localhost:4000", "http://192.0.2.1:4000", "http://127.0.0.1", "http://127.0.0.1:0", "http://127.0.0.1:65536", "http://user@127.0.0.1:4000"} {
		t.Run(origin, func(t *testing.T) {
			_, err := NewElectronRoutingIsolation(&ElectronTarget{TargetID: "target", RendererID: "renderer", ContextID: "cell", ScenarioName: "sample", ArtifactDigest: "sha256:artifact", CDPEndpoint: "http://127.0.0.1:9222", CDPTransport: "http", RendererURL: origin})
			require.Error(t, err)
		})
	}
	_, err = NewElectronRoutingIsolation(&ElectronTarget{TargetID: "target", RendererID: "renderer", ContextID: "cell", ScenarioName: "sample", ArtifactDigest: "sha256:artifact", CDPEndpoint: "http://127.0.0.1:9222", CDPTransport: "http", RendererURL: "http://[::1]:4000/"})
	require.NoError(t, err)
}

type installAckRoutingClient struct {
	response       *connect.Response[routingv1.InstallTestPoolResponse]
	installErr     error
	clearErr       error
	clearResponse  *connect.Response[routingv1.ClearTestPoolResponse]
	installCalls   int
	clearCalls     int
	installLeaseID string
	installDSN     string
	clearLeaseID   string
	installApplied bool
}

type restartAckRoutingClient struct {
	*installAckRoutingClient
	prepareCalls int
	resumeCalls  int
	clearTicket  string
	resumeTicket string
}

func (c *restartAckRoutingClient) PrepareTestPoolRestart(context.Context, *connect.Request[routingv1.PrepareTestPoolRestartRequest]) (*connect.Response[routingv1.PrepareTestPoolRestartResponse], error) {
	c.prepareCalls++
	return connect.NewResponse(&routingv1.PrepareTestPoolRestartResponse{RestartTicket: "private-restart-ticket"}), nil
}

func (c *restartAckRoutingClient) ResumeTestPool(_ context.Context, req *connect.Request[routingv1.ResumeTestPoolRequest]) (*connect.Response[routingv1.ResumeTestPoolResponse], error) {
	c.resumeCalls++
	c.resumeTicket = req.Msg.GetRestartTicket()
	return connect.NewResponse(&routingv1.ResumeTestPoolResponse{ActiveLeaseId: req.Msg.GetLeaseId(), FileRootsResumed: true}), nil
}

func (c *restartAckRoutingClient) ClearTestPool(_ context.Context, req *connect.Request[routingv1.ClearTestPoolRequest]) (*connect.Response[routingv1.ClearTestPoolResponse], error) {
	c.clearTicket = req.Msg.GetRestartTicket()
	return c.installAckRoutingClient.ClearTestPool(context.Background(), req)
}

func TestBundledElectronIsolationPreparesAndResumesPrivatePairedLease(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Chmod(root, 0o700))
	identity := restartHandoffIdentity{TargetID: "target", ContextID: "context", Artifact: "sha256:artifact"}
	target := &ElectronTarget{TargetID: identity.TargetID, RendererID: "renderer", ContextID: identity.ContextID, ScenarioName: "device-studio", ArtifactDigest: identity.Artifact, CDPEndpoint: "http://127.0.0.1:9222", CDPTransport: "http", RendererURL: "http://127.0.0.1:41235/", APIURL: "http://127.0.0.1:41236", TestDatabaseRoot: root, BundledRuntime: true, IsolationLeaseID: "matrix:cell", PrepareIsolationRestart: true}
	first, err := NewElectronRoutingIsolation(target)
	require.NoError(t, err)
	prepareClient := &restartAckRoutingClient{installAckRoutingClient: &installAckRoutingClient{response: connect.NewResponse(&routingv1.InstallTestPoolResponse{ActiveLeaseId: "matrix:cell", FileRootsInstalled: true})}}
	first.NewClient = func(string) routingClient { return prepareClient }
	first.RequiresFileRoots = func(context.Context, string) (bool, error) { return true, nil }
	first.TTL = time.Hour
	lease, err := first.Acquire(context.Background(), "device-studio", "matrix:cell")
	require.NoError(t, err)
	prepared := lease.Close(context.Background())
	require.True(t, prepared.RestartPrepared)
	require.Empty(t, prepared.ClearError)
	require.Equal(t, 1, prepareClient.prepareCalls)
	require.Zero(t, prepareClient.clearCalls, "preparation keeps the paired roots live for owner-led restart")
	require.FileExists(t, first.HandoffPath)
	handoffInfo, err := os.Lstat(first.HandoffPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), handoffInfo.Mode().Perm())
	require.DirExists(t, disposableSQLiteDirectory(root, "matrix:cell"), "the exact routed database remains available across restart")
	encodedEvidence, err := json.Marshal(prepared)
	require.NoError(t, err)
	require.NotContains(t, string(encodedEvidence), "private-restart-ticket", "bearer ticket must not enter ordinary execution evidence")

	target.PrepareIsolationRestart = false
	target.ResumeIsolationRestart = true
	second, err := NewElectronRoutingIsolation(target)
	require.NoError(t, err)
	resumeClient := &restartAckRoutingClient{installAckRoutingClient: &installAckRoutingClient{}}
	second.NewClient = func(string) routingClient { return resumeClient }
	second.RequiresFileRoots = func(context.Context, string) (bool, error) { return true, nil }
	second.TTL = time.Hour
	resumed, err := second.Acquire(context.Background(), "device-studio", "matrix:cell")
	require.NoError(t, err)
	require.True(t, resumed.Evidence().RestartResumed)
	require.Equal(t, "private-restart-ticket", resumeClient.resumeTicket)
	cleared := resumed.Close(context.Background())
	require.Empty(t, cleared.ClearError)
	require.Equal(t, "private-restart-ticket", resumeClient.clearTicket)
	require.NoFileExists(t, second.HandoffPath)
	require.NoDirExists(t, disposableSQLiteDirectory(root, "matrix:cell"), "only a confirmed exact paired clear removes the retained DB")
}

func TestRestartHandoffRejectsIdentityMismatchAndSymlink(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Chmod(root, 0o700))
	path := filepath.Join(root, "handoff.json")
	handoff := restartHandoff{LeaseID: "matrix:cell", TargetID: "target", ContextID: "context", Artifact: "sha256:artifact", Ticket: "secret"}
	require.NoError(t, writeRestartHandoff(path, handoff))
	_, err := readRestartHandoff(path, handoff.LeaseID, restartHandoffIdentity{TargetID: "other", ContextID: handoff.ContextID, Artifact: handoff.Artifact})
	require.ErrorContains(t, err, "does not match")
	require.NoError(t, os.Remove(path))
	targetPath := filepath.Join(root, "outside")
	require.NoError(t, os.WriteFile(targetPath, []byte(`{"ticket":"secret"}`), 0o600))
	require.NoError(t, os.Symlink(targetPath, path))
	_, err = readRestartHandoff(path, handoff.LeaseID, restartHandoffIdentity{TargetID: handoff.TargetID, ContextID: handoff.ContextID, Artifact: handoff.Artifact})
	require.ErrorContains(t, err, "private regular file")
}

func (c *installAckRoutingClient) InstallTestPool(_ context.Context, req *connect.Request[routingv1.InstallTestPoolRequest]) (*connect.Response[routingv1.InstallTestPoolResponse], error) {
	c.installCalls++
	c.installLeaseID = req.Msg.GetLeaseId()
	c.installDSN = req.Msg.GetDsn()
	c.installApplied = true // model a timeout after the target may have applied the request
	return c.response, c.installErr
}

func (c *installAckRoutingClient) ClearTestPool(_ context.Context, req *connect.Request[routingv1.ClearTestPoolRequest]) (*connect.Response[routingv1.ClearTestPoolResponse], error) {
	c.clearCalls++
	c.clearLeaseID = req.Msg.GetLeaseId()
	if c.clearErr != nil {
		return nil, c.clearErr
	}
	if c.clearResponse != nil {
		return c.clearResponse, nil
	}
	return connect.NewResponse(&routingv1.ClearTestPoolResponse{}), nil
}

func (c *installAckRoutingClient) HeartbeatTestPool(context.Context, *connect.Request[routingv1.HeartbeatTestPoolRequest]) (*connect.Response[routingv1.HeartbeatTestPoolResponse], error) {
	return connect.NewResponse(&routingv1.HeartbeatTestPoolResponse{}), nil
}

func TestRoutingIsolationRequiresExactActiveLeaseAcknowledgement(t *testing.T) {
	for _, tc := range []struct {
		name      string
		response  *connect.Response[routingv1.InstallTestPoolResponse]
		wantError bool
	}{
		{name: "exact lease", response: connect.NewResponse(&routingv1.InstallTestPoolResponse{ActiveLeaseId: "lease-1"})},
		{name: "missing response", wantError: true},
		{name: "empty active lease", response: connect.NewResponse(&routingv1.InstallTestPoolResponse{}), wantError: true},
		{name: "different active lease", response: connect.NewResponse(&routingv1.InstallTestPoolResponse{ActiveLeaseId: "other-lease"}), wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &installAckRoutingClient{response: tc.response}
			cleaned := false
			isolation := &RoutingIsolation{
				ResolveTarget: func(context.Context, string) (string, error) { return "http://target", nil },
				NewClient:     func(string) routingClient { return client },
				NewDSN: func(string) (string, func(), error) {
					return "file:test.db", func() { cleaned = true }, nil
				},
				RequiresFileRoots: func(context.Context, string) (bool, error) { return false, nil },
				TTL:               time.Hour,
			}

			lease, err := isolation.Acquire(context.Background(), "sample", "lease-1")
			if tc.wantError {
				require.Error(t, err)
				require.Nil(t, lease)
				require.True(t, cleaned)
				require.Equal(t, 1, client.clearCalls, "unconfirmed acquisition should attempt exact-lease cleanup")
				return
			}
			require.NoError(t, err)
			require.NotNil(t, lease)
			require.True(t, lease.Evidence().Installed)
			require.Equal(t, "lease-1", lease.Evidence().LeaseID)
			lease.Close(context.Background())
			require.True(t, cleaned)
		})
	}
}

func TestRoutingIsolationReportsFailedCleanupAfterInvalidLeaseAcknowledgement(t *testing.T) {
	client := &installAckRoutingClient{
		response: connect.NewResponse(&routingv1.InstallTestPoolResponse{ActiveLeaseId: "other-lease"}),
		clearErr: errors.New("provider unavailable during clear"),
	}
	cleaned := false
	isolation := &RoutingIsolation{
		ResolveTarget: func(context.Context, string) (string, error) { return "http://target", nil },
		NewClient:     func(string) routingClient { return client },
		NewDSN: func(string) (string, func(), error) {
			return "file:test.db", func() { cleaned = true }, nil
		},
		RequiresFileRoots: func(context.Context, string) (bool, error) { return false, nil },
		TTL:               time.Hour,
	}

	lease, err := isolation.Acquire(context.Background(), "sample", "lease-1")
	require.ErrorContains(t, err, `active lease "other-lease" does not match requested lease "lease-1"`)
	require.ErrorContains(t, err, `cleanup failed clearing requested lease "lease-1"; local test database retained for reconciliation: provider unavailable during clear`)
	require.Nil(t, lease)
	require.False(t, cleaned, "keep the local DB available while remote lease cleanup is unconfirmed")
	require.Equal(t, 1, client.clearCalls)
	require.Equal(t, "lease-1", client.clearLeaseID)
}

func TestRoutingIsolationAmbiguousInstallErrorClearsExactLeaseBeforeLocalCleanup(t *testing.T) {
	installErr := errors.New("connection lost after request write")
	client := &installAckRoutingClient{installErr: installErr}
	cleaned := false
	isolation := &RoutingIsolation{
		ResolveTarget: func(context.Context, string) (string, error) { return "http://target", nil },
		NewClient:     func(string) routingClient { return client },
		NewDSN: func(string) (string, func(), error) {
			return "file:test.db", func() { cleaned = true }, nil
		},
		RequiresFileRoots: func(context.Context, string) (bool, error) { return false, nil },
	}

	lease, err := isolation.Acquire(context.Background(), "sample", "lease-ambiguous")
	require.ErrorIs(t, err, installErr)
	require.Nil(t, lease)
	require.True(t, client.installApplied, "the simulated timeout occurs after the target may have applied the request")
	require.Equal(t, 1, client.installCalls)
	require.Equal(t, 1, client.clearCalls, "ambiguous install must attempt remote cleanup")
	require.Equal(t, "lease-ambiguous", client.clearLeaseID, "cleanup must be fenced to the exact requested lease")
	require.True(t, cleaned, "confirmed remote cleanup permits local DSN removal")
}

func TestRoutingIsolationAmbiguousInstallCleanupFailureRetainsLocalDatabase(t *testing.T) {
	installErr := errors.New("connection lost after request write")
	clearErr := errors.New("target unavailable during clear")
	client := &installAckRoutingClient{installErr: installErr, clearErr: clearErr}
	cleaned := false
	isolation := &RoutingIsolation{
		ResolveTarget: func(context.Context, string) (string, error) { return "http://target", nil },
		NewClient:     func(string) routingClient { return client },
		NewDSN: func(string) (string, func(), error) {
			return "file:test.db", func() { cleaned = true }, nil
		},
		RequiresFileRoots: func(context.Context, string) (bool, error) { return false, nil },
	}

	lease, err := isolation.Acquire(context.Background(), "sample", "lease-ambiguous")
	require.ErrorIs(t, err, installErr)
	require.ErrorIs(t, err, clearErr)
	require.ErrorContains(t, err, "local test database retained for reconciliation")
	require.Nil(t, lease)
	require.Equal(t, 1, client.clearCalls)
	require.Equal(t, "lease-ambiguous", client.clearLeaseID)
	require.False(t, cleaned, "never unlink a DB that an ambiguously installed remote lease may still use")
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

type contendedRoutingClient struct {
	installAckRoutingClient
	busy bool
}

func (c *contendedRoutingClient) InstallTestPool(ctx context.Context, req *connect.Request[routingv1.InstallTestPoolRequest]) (*connect.Response[routingv1.InstallTestPoolResponse], error) {
	if c.busy {
		c.installCalls++
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("another owner holds the pool"))
	}
	return c.installAckRoutingClient.InstallTestPool(ctx, req)
}

func TestRoutingIsolationWaitsForBusyOwnerWithoutClearingItsLease(t *testing.T) {
	for _, cancelWait := range []bool{false, true} {
		t.Run(fmt.Sprint(cancelWait), func(t *testing.T) {
			client := &contendedRoutingClient{busy: true}
			client.response = connect.NewResponse(&routingv1.InstallTestPoolResponse{ActiveLeaseId: "waiting-worker"})
			cleaned := false
			isolation := &RoutingIsolation{
				ResolveTarget:     func(context.Context, string) (string, error) { return "http://target", nil },
				NewClient:         func(string) routingClient { return client },
				NewDSN:            func(string) (string, func(), error) { return "file:waiting.db", func() { cleaned = true }, nil },
				RequiresFileRoots: func(context.Context, string) (bool, error) { return false, nil }, TTL: time.Hour,
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			isolation.WaitForLease = func(ctx context.Context) error {
				require.Zero(t, client.clearCalls, "waiting must never clear the active owner's lease")
				require.False(t, cleaned, "keep this worker's uninstalled database while waiting")
				if cancelWait {
					cancel()
					return ctx.Err()
				}
				client.busy = false
				return nil
			}
			lease, err := isolation.Acquire(ctx, "sample", "waiting-worker")
			if cancelWait {
				require.ErrorIs(t, err, context.Canceled)
				require.Nil(t, lease)
				require.True(t, cleaned)
				require.Zero(t, client.clearCalls)
				require.Equal(t, 1, client.installCalls)
			} else {
				require.NoError(t, err)
				require.Equal(t, 2, client.installCalls)
				require.Equal(t, 1, lease.Evidence().LeaseWaits)
				require.Zero(t, client.clearCalls)
				lease.Close(context.Background())
				require.Equal(t, "waiting-worker", client.clearLeaseID)
				require.True(t, cleaned)
			}
		})
	}
}
