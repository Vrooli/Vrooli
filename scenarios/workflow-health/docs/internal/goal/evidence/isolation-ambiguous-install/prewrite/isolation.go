package execution

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/storage"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing"
	"github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing/routing_v1connect"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	isolationLeaseTTL       = 90 * time.Second
	isolationCleanupTimeout = 5 * time.Second
)

type routingClient interface {
	InstallTestPool(context.Context, *connect.Request[routingv1.InstallTestPoolRequest]) (*connect.Response[routingv1.InstallTestPoolResponse], error)
	ClearTestPool(context.Context, *connect.Request[routingv1.ClearTestPoolRequest]) (*connect.Response[routingv1.ClearTestPoolResponse], error)
	HeartbeatTestPool(context.Context, *connect.Request[routingv1.HeartbeatTestPoolRequest]) (*connect.Response[routingv1.HeartbeatTestPoolResponse], error)
}

// RoutingIsolation installs the two-engine dev-routing lease on a target.
// Test DSN creation is injectable because non-SQLite targets need their own
// provisioner; the default safely serves SQLite-based scenarios such as
// prompt-manager with a disposable local database.
type RoutingIsolation struct {
	ResolveTarget     func(context.Context, string) (string, error)
	NewClient         func(string) routingClient
	NewDSN            func(string) (string, func(), error)
	RequiresFileRoots func(context.Context, string) (bool, error)
	TTL               time.Duration
}

func NewRoutingIsolation() *RoutingIsolation {
	return &RoutingIsolation{
		ResolveTarget: discovery.ResolveScenarioURLDefault,
		NewClient: func(base string) routingClient {
			return routing_v1connect.NewRoutingServiceClient(http.DefaultClient, base)
		},
		NewDSN:            disposableSQLiteDSN,
		RequiresFileRoots: storageHealthRequiresFileRoots,
		TTL:               isolationLeaseTTL,
	}
}

// NewElectronRoutingIsolation targets the HTTP origin of the bound renderer.
// Its UI server must proxy routing RPCs to that package's private API. A
// desktop target must never silently install its lease on the fleet instance.
func NewElectronRoutingIsolation(target *ElectronTarget) (*RoutingIsolation, error) {
	if target == nil || target.TargetID == "" || target.RendererID == "" || target.ContextID == "" || target.ScenarioName == "" || target.ArtifactDigest == "" || target.CDPEndpoint == "" || target.CDPTransport == "" {
		return nil, fmt.Errorf("desktop routing isolation requires complete bound target identity")
	}
	u, err := url.Parse(target.RendererURL)
	if err != nil || u == nil || u.Scheme != "http" || u.User != nil || u.Opaque != "" {
		return nil, fmt.Errorf("desktop routing isolation requires a loopback HTTP renderer origin")
	}
	ip := net.ParseIP(u.Hostname())
	port, err := strconv.Atoi(u.Port())
	if ip == nil || !ip.IsLoopback() || err != nil || port < 1 || port > 65535 {
		return nil, fmt.Errorf("desktop routing isolation requires an explicit numeric loopback address and port")
	}
	base := (&url.URL{Scheme: u.Scheme, Host: u.Host}).String()
	scenario := target.ScenarioName
	isolation := NewRoutingIsolation()
	isolation.ResolveTarget = func(ctx context.Context, requested string) (string, error) {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if requested != scenario {
			return "", fmt.Errorf("bound desktop scenario %q does not match isolation target %q", scenario, requested)
		}
		return base, nil
	}
	return isolation, nil
}

func (r *RoutingIsolation) Acquire(ctx context.Context, scenario, leaseID string) (IsolationLease, error) {
	if r == nil {
		return nil, fmt.Errorf("routing isolation is nil")
	}
	base, err := r.ResolveTarget(ctx, scenario)
	if err != nil {
		return nil, fmt.Errorf("resolve target scenario %q: %w", scenario, err)
	}
	dsn, cleanup, err := r.NewDSN(leaseID)
	if err != nil {
		return nil, fmt.Errorf("create test database for %q: %w", scenario, err)
	}
	ttl := r.TTL
	if ttl <= 0 {
		ttl = isolationLeaseTTL
	}
	client := r.NewClient(base)
	requiresFileRoots := r.RequiresFileRoots
	if requiresFileRoots == nil {
		cleanup()
		return nil, fmt.Errorf("storage file-isolation classification is not configured")
	}
	filePersisting, err := requiresFileRoots(ctx, scenario)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("prove target file-isolation requirement: %w", err)
	}
	installed, err := client.InstallTestPool(ctx, connect.NewRequest(&routingv1.InstallTestPoolRequest{Dsn: dsn, LeaseId: leaseID, LeaseTtlMs: ttl.Milliseconds()}))
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("install target test pool: %w", err)
	}
	if installed == nil || installed.Msg == nil {
		return nil, cleanupFailedInstall(client, leaseID, cleanup, fmt.Errorf("target scenario did not confirm an active test-pool lease"))
	}
	if activeLeaseID := strings.TrimSpace(installed.Msg.GetActiveLeaseId()); activeLeaseID == "" || activeLeaseID != leaseID {
		return nil, cleanupFailedInstall(client, leaseID, cleanup, fmt.Errorf("target scenario active lease %q does not match requested lease %q", activeLeaseID, leaseID))
	}
	if filePersisting && !installed.Msg.GetFileRootsInstalled() {
		return nil, cleanupFailedInstall(client, leaseID, cleanup, fmt.Errorf("target scenario did not install leased file roots; run storage-manager validation and wire RoutedRoots"))
	}
	leaseCtx, cancel := context.WithCancel(context.Background())
	lease := &routingLease{
		client:     client,
		leaseID:    leaseID,
		cleanup:    cleanup,
		cancel:     cancel,
		healthDone: make(chan struct{}),
		evidence:   IsolationEvidence{Installed: true, LeaseID: leaseID},
	}
	go lease.heartbeat(leaseCtx, ttl)
	return lease, nil
}

func cleanupFailedInstall(client routingClient, leaseID string, cleanup func(), cause error) error {
	ctx, cancel := context.WithTimeout(context.Background(), isolationCleanupTimeout)
	defer cancel()
	_, clearErr := client.ClearTestPool(ctx, connect.NewRequest(&routingv1.ClearTestPoolRequest{LeaseId: leaseID}))
	cleanup()
	if clearErr != nil {
		return fmt.Errorf("%w (cleanup also failed clearing requested lease %q: %v)", cause, leaseID, clearErr)
	}
	return cause
}

func storageHealthRequiresFileRoots(ctx context.Context, scenario string) (bool, error) {
	base, err := discovery.ResolveScenarioURLDefault(ctx, "storage-manager")
	if err != nil {
		return false, fmt.Errorf("resolve storage-manager: %w", err)
	}
	client := scenariovalidationv1connect.NewScenarioValidationServiceClient(http.DefaultClient, base)
	resp, err := client.ValidateScenario(ctx, connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: scenario}))
	if err != nil {
		return false, fmt.Errorf("validate target through storage-manager: %w", err)
	}
	return filePersistenceClassification(resp.Msg)
}

// filePersistenceClassification answers only the isolation question consumed
// by workflow execution. Storage Manager's overall status also includes
// declaration, retention, and legacy-storage findings; those remain important
// storage debt but must not masquerade as evidence that routed seams are absent.
func filePersistenceClassification(resp *scenariovalidationv1.ValidateScenarioResponse) (bool, error) {
	detail := &structpb.Struct{}
	if native := resp.GetNativeDetail(); native == nil || native.UnmarshalTo(detail) != nil {
		return false, fmt.Errorf("storage-manager response omitted file-persistence classification")
	}
	field, ok := detail.Fields["file_persisting"]
	if !ok {
		return false, fmt.Errorf("storage-manager response has invalid file-persistence classification")
	}
	if _, ok := field.Kind.(*structpb.Value_BoolValue); !ok {
		return false, fmt.Errorf("storage-manager response has invalid file-persistence classification")
	}
	return field.GetBoolValue(), nil
}

type routingLease struct {
	client     routingClient
	leaseID    string
	cleanup    func()
	cancel     context.CancelFunc
	healthDone chan struct{}
	healthOnce sync.Once
	mu         sync.Mutex
	healthErr  error
	evidence   IsolationEvidence
}

func (l *routingLease) Evidence() IsolationEvidence {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.evidence
}

func (l *routingLease) Done() <-chan struct{} { return l.healthDone }

func (l *routingLease) Err() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.healthErr
}

func (l *routingLease) heartbeat(ctx context.Context, ttl time.Duration) {
	interval := ttl / 3
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := l.client.HeartbeatTestPool(ctx, connect.NewRequest(&routingv1.HeartbeatTestPoolRequest{LeaseId: l.leaseID})); err != nil {
				l.mu.Lock()
				l.healthErr = err
				l.evidence.HeartbeatError = err.Error()
				if l.evidence.ClearError == "" {
					l.evidence.ClearError = "lease heartbeat failed: " + err.Error()
				}
				l.mu.Unlock()
				l.healthOnce.Do(func() { close(l.healthDone) })
				return
			}
		}
	}
}

func (l *routingLease) Close(ctx context.Context) IsolationEvidence {
	l.cancel()
	defer l.cleanup()
	resp, err := l.client.ClearTestPool(ctx, connect.NewRequest(&routingv1.ClearTestPoolRequest{LeaseId: l.leaseID}))
	l.mu.Lock()
	defer l.mu.Unlock()
	if err != nil {
		l.evidence.ClearError = err.Error()
		return l.evidence
	}
	stats := resp.Msg.GetStats()
	l.evidence.TestPoolRequests = stats.GetTestPoolRequests()
	l.evidence.PrimaryDuringTestModeRequests = stats.GetPrimaryDuringTestModeRequests()
	l.evidence.TestRootWrites = stats.GetTestRootWrites()
	l.evidence.PrimaryRootWritesDuringTestMode = stats.GetPrimaryRootWritesDuringTestMode()
	return l.evidence
}

func disposableSQLiteDSN(leaseID string) (string, func(), error) {
	dir, err := os.MkdirTemp("", "vrooli-workflow-test-db-")
	if err != nil {
		return "", nil, err
	}
	name := strings.NewReplacer("/", "-", "\\", "-").Replace(leaseID)
	path := filepath.Join(dir, name+".db")
	dsn, err := storage.SQLiteDSNAt(path, storage.SQLiteTuning{})
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", nil, err
	}
	return dsn, func() { _ = os.RemoveAll(dir) }, nil
}
