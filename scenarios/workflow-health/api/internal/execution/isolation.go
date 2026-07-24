package execution

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing"
	"github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing/routing_v1connect"
)

const isolationLeaseTTL = 90 * time.Second

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
	ResolveTarget func(context.Context, string) (string, error)
	NewClient     func(string) routingClient
	NewDSN        func(string) (string, func(), error)
	TTL           time.Duration
}

func NewRoutingIsolation() *RoutingIsolation {
	return &RoutingIsolation{
		ResolveTarget: discovery.ResolveScenarioURLDefault,
		NewClient: func(base string) routingClient {
			return routing_v1connect.NewRoutingServiceClient(http.DefaultClient, base)
		},
		NewDSN: disposableSQLiteDSN,
		TTL:    isolationLeaseTTL,
	}
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
	installed, err := client.InstallTestPool(ctx, connect.NewRequest(&routingv1.InstallTestPoolRequest{Dsn: dsn, LeaseId: leaseID, LeaseTtlMs: ttl.Milliseconds()}))
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("install target test pool: %w", err)
	}
	if !installed.Msg.GetFileRootsInstalled() {
		_, _ = client.ClearTestPool(context.Background(), connect.NewRequest(&routingv1.ClearTestPoolRequest{LeaseId: leaseID}))
		cleanup()
		return nil, fmt.Errorf("target scenario did not install leased file roots; run storage-health validation and wire RoutedRoots")
	}
	leaseCtx, cancel := context.WithCancel(context.Background())
	lease := &routingLease{client: client, leaseID: leaseID, cleanup: cleanup, cancel: cancel, evidence: IsolationEvidence{Installed: true, LeaseID: leaseID}}
	go lease.heartbeat(leaseCtx, ttl)
	return lease, nil
}

type routingLease struct {
	client   routingClient
	leaseID  string
	cleanup  func()
	cancel   context.CancelFunc
	mu       sync.Mutex
	evidence IsolationEvidence
}

func (l *routingLease) Evidence() IsolationEvidence {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.evidence
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
				if l.evidence.ClearError == "" {
					l.evidence.ClearError = "lease heartbeat failed: " + err.Error()
				}
				l.mu.Unlock()
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
	dsn := "file:" + path + "?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)"
	return dsn, func() { _ = os.RemoveAll(dir) }, nil
}
