package execution

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

type restartRoutingClient interface {
	PrepareTestPoolRestart(context.Context, *connect.Request[routingv1.PrepareTestPoolRestartRequest]) (*connect.Response[routingv1.PrepareTestPoolRestartResponse], error)
	ResumeTestPool(context.Context, *connect.Request[routingv1.ResumeTestPoolRequest]) (*connect.Response[routingv1.ResumeTestPoolResponse], error)
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
	PrepareRestart    bool
	ResumeRestart     bool
	HandoffPath       string
	HandoffIdentity   restartHandoffIdentity
	TargetDBRoot      string
	DownloadRoot      string
	// WaitForLease is the service-owned contention delay. Durable validation
	// keeps its run active while another owner holds the target pool.
	WaitForLease func(context.Context) error
}

type restartHandoff struct {
	LeaseID   string `json:"lease_id"`
	TargetID  string `json:"target_id"`
	ContextID string `json:"context_id"`
	Artifact  string `json:"artifact_digest"`
	Ticket    string `json:"restart_ticket"`
}

type restartHandoffIdentity struct {
	TargetID  string
	ContextID string
	Artifact  string
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

// NewElectronRoutingIsolation targets the API origin owned by the bound
// desktop package. Bundled targets must provide the target-discovered API
// endpoint; legacy targets retain their renderer-origin routing behavior.
// A desktop target must never silently install its lease on the fleet instance.
func NewElectronRoutingIsolation(target *ElectronTarget) (*RoutingIsolation, error) {
	if target == nil || target.TargetID == "" || target.RendererID == "" || target.ContextID == "" || target.ScenarioName == "" || target.ArtifactDigest == "" || target.CDPEndpoint == "" || target.CDPTransport == "" {
		return nil, fmt.Errorf("desktop routing isolation requires complete bound target identity")
	}
	endpoint := target.RendererURL
	if target.BundledRuntime {
		endpoint = target.APIURL
		if strings.TrimSpace(endpoint) == "" {
			return nil, fmt.Errorf("bundled desktop routing isolation requires the target-owned API endpoint")
		}
	}
	u, err := url.Parse(endpoint)
	if err != nil || u == nil || u.Scheme != "http" || u.User != nil || u.Opaque != "" {
		return nil, fmt.Errorf("desktop routing isolation requires a loopback HTTP API origin")
	}
	ip := net.ParseIP(u.Hostname())
	port, err := strconv.Atoi(u.Port())
	if ip == nil || !ip.IsLoopback() || err != nil || port < 1 || port > 65535 {
		return nil, fmt.Errorf("desktop routing isolation requires an explicit numeric loopback API address and port")
	}
	base := (&url.URL{Scheme: u.Scheme, Host: u.Host}).String()
	scenario := target.ScenarioName
	isolation := NewRoutingIsolation()
	if target.BundledRuntime {
		root, err := validateTargetTestDatabaseRoot(target.TestDatabaseRoot)
		if err != nil {
			return nil, err
		}
		isolation.NewDSN = func(leaseID string) (string, func(), error) {
			return disposableSQLiteDSNAt(root, leaseID)
		}
		isolation.TargetDBRoot = root
		downloadRoot, err := prepareTargetDownloadRoot(root)
		if err != nil {
			return nil, err
		}
		isolation.DownloadRoot = downloadRoot
		if target.PrepareIsolationRestart || target.ResumeIsolationRestart {
			if target.IsolationLeaseID == "" || (target.PrepareIsolationRestart && target.ResumeIsolationRestart) {
				return nil, fmt.Errorf("restart handoff requires a stable lease identity and one phase")
			}
			isolation.PrepareRestart = target.PrepareIsolationRestart
			isolation.ResumeRestart = target.ResumeIsolationRestart
			isolation.HandoffPath = filepath.Join(root, "workflow-health-restart-handoff.json")
			isolation.HandoffIdentity = restartHandoffIdentity{TargetID: target.TargetID, ContextID: target.ContextID, Artifact: target.ArtifactDigest}
		}
	}
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
	ttl := r.TTL
	if ttl <= 0 {
		ttl = isolationLeaseTTL
	}
	client := r.NewClient(base)
	if r.ResumeRestart {
		restartClient, ok := client.(restartRoutingClient)
		if !ok {
			return nil, fmt.Errorf("target routing owner does not support paired restart resume")
		}
		checkpoint, err := readRestartHandoff(r.HandoffPath, leaseID, r.HandoffIdentity)
		if err != nil {
			return nil, fmt.Errorf("read private restart handoff: %w", err)
		}
		resumed, err := restartClient.ResumeTestPool(ctx, connect.NewRequest(&routingv1.ResumeTestPoolRequest{LeaseId: leaseID, RestartTicket: checkpoint.Ticket}))
		if err != nil {
			return nil, fmt.Errorf("resume paired target routing lease: %w", err)
		}
		if resumed == nil || resumed.Msg == nil || resumed.Msg.GetActiveLeaseId() != leaseID || !resumed.Msg.GetFileRootsResumed() {
			return nil, fmt.Errorf("target routing owner did not confirm the paired restart lease")
		}
		leaseCtx, cancel := context.WithCancel(context.Background())
		lease := &routingLease{client: client, leaseID: leaseID, cleanup: func() { removeDisposableSQLiteDSNAt(r.TargetDBRoot, leaseID) }, cancel: cancel, healthDone: make(chan struct{}), restartTicket: checkpoint.Ticket, handoffPath: r.HandoffPath}
		lease.evidence = IsolationEvidence{Installed: true, LeaseID: leaseID, RestartResumed: true}
		go lease.heartbeat(leaseCtx, ttl)
		return lease, nil
	}
	dsn, cleanup, err := r.NewDSN(leaseID)
	if err != nil {
		return nil, fmt.Errorf("create test database for %q: %w", scenario, err)
	}
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
	var installed *connect.Response[routingv1.InstallTestPoolResponse]
	waits := 0
	for {
		if err := ctx.Err(); err != nil {
			cleanup()
			return nil, fmt.Errorf("waiting for target test pool: %w", err)
		}
		installed, err = client.InstallTestPool(ctx, connect.NewRequest(&routingv1.InstallTestPoolRequest{Dsn: dsn, LeaseId: leaseID, LeaseTtlMs: ttl.Milliseconds()}))
		if connect.CodeOf(err) != connect.CodeAlreadyExists {
			break
		}
		// This typed refusal proves our install was not applied. Never clear
		// the active owner's lease, nor classify ordinary contention as failure.
		waits++
		wait := r.WaitForLease
		if wait == nil {
			wait = waitForTestPool
		}
		if err := wait(ctx); err != nil {
			cleanup()
			return nil, fmt.Errorf("waiting for target test pool: %w", err)
		}
	}
	if err != nil {
		return nil, cleanupFailedInstall(client, leaseID, cleanup, fmt.Errorf("install target test pool: %w", err))
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
		client:          client,
		leaseID:         leaseID,
		cleanup:         cleanup,
		cancel:          cancel,
		healthDone:      make(chan struct{}),
		evidence:        IsolationEvidence{Installed: true, LeaseID: leaseID, LeaseWaits: waits},
		prepareRestart:  r.PrepareRestart,
		handoffPath:     r.HandoffPath,
		handoffIdentity: r.HandoffIdentity,
	}
	go lease.heartbeat(leaseCtx, ttl)
	return lease, nil
}

func waitForTestPool(ctx context.Context) error {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func cleanupFailedInstall(client routingClient, leaseID string, cleanup func(), cause error) error {
	ctx, cancel := context.WithTimeout(context.Background(), isolationCleanupTimeout)
	defer cancel()
	cleared, clearErr := client.ClearTestPool(ctx, connect.NewRequest(&routingv1.ClearTestPoolRequest{LeaseId: leaseID}))
	if clearErr != nil {
		return fmt.Errorf("%w (cleanup failed clearing requested lease %q; local test database retained for reconciliation: %w)", cause, leaseID, clearErr)
	}
	if cleared == nil || cleared.Msg == nil {
		return fmt.Errorf("%w (cleanup did not confirm clearing requested lease %q; local test database retained for reconciliation)", cause, leaseID)
	}
	cleanup()
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
	client          routingClient
	leaseID         string
	cleanup         func()
	cancel          context.CancelFunc
	healthDone      chan struct{}
	healthOnce      sync.Once
	mu              sync.Mutex
	healthErr       error
	evidence        IsolationEvidence
	prepareRestart  bool
	restartTicket   string
	handoffPath     string
	handoffIdentity restartHandoffIdentity
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
	if l.prepareRestart {
		restartClient, ok := l.client.(restartRoutingClient)
		if !ok {
			l.mu.Lock()
			l.evidence.ClearError = "target routing owner does not support paired restart preparation"
			l.mu.Unlock()
			return l.Evidence()
		}
		prepared, err := restartClient.PrepareTestPoolRestart(ctx, connect.NewRequest(&routingv1.PrepareTestPoolRestartRequest{LeaseId: l.leaseID}))
		if err == nil && prepared != nil && prepared.Msg != nil && prepared.Msg.GetRestartTicket() != "" {
			l.restartTicket = prepared.Msg.GetRestartTicket()
			handoff := restartHandoff{LeaseID: l.leaseID, TargetID: l.handoffIdentity.TargetID, ContextID: l.handoffIdentity.ContextID, Artifact: l.handoffIdentity.Artifact, Ticket: prepared.Msg.GetRestartTicket()}
			err = writeRestartHandoff(l.handoffPath, handoff)
		}
		if err == nil && prepared != nil && prepared.Msg != nil && prepared.Msg.GetRestartTicket() != "" {
			l.mu.Lock()
			l.evidence.RestartPrepared = true
			l.mu.Unlock()
			return l.Evidence()
		}
		if err == nil {
			err = fmt.Errorf("target routing owner returned no restart ticket")
		}
		l.mu.Lock()
		l.evidence.ClearError = fmt.Sprintf("prepare paired target restart: %v", err)
		l.mu.Unlock()
	}
	request := &routingv1.ClearTestPoolRequest{LeaseId: l.leaseID, RestartTicket: l.restartTicket}
	resp, err := l.client.ClearTestPool(ctx, connect.NewRequest(request))
	l.mu.Lock()
	defer l.mu.Unlock()
	if err != nil {
		l.evidence.ClearError = err.Error()
		return l.evidence
	}
	if resp == nil || resp.Msg == nil {
		l.evidence.ClearError = "target routing owner did not confirm clearing the paired lease"
		return l.evidence
	}
	stats := resp.Msg.GetStats()
	l.evidence.TestPoolRequests = stats.GetTestPoolRequests()
	l.evidence.PrimaryDuringTestModeRequests = stats.GetPrimaryDuringTestModeRequests()
	l.evidence.TestRootWrites = stats.GetTestRootWrites()
	l.evidence.PrimaryRootWritesDuringTestMode = stats.GetPrimaryRootWritesDuringTestMode()
	if l.handoffPath != "" {
		if err := os.Remove(l.handoffPath); err != nil && !os.IsNotExist(err) {
			l.evidence.ClearError = fmt.Sprintf("clear private restart handoff: %v", err)
		}
	}
	if l.cleanup != nil {
		l.cleanup()
	}
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

func validateTargetTestDatabaseRoot(root string) (string, error) {
	root = strings.TrimSpace(root)
	if root == "" || !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return "", fmt.Errorf("bundled desktop routing isolation requires an absolute target-owned test database root")
	}
	info, err := os.Lstat(root)
	if err != nil {
		return "", fmt.Errorf("inspect target-owned test database root: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o700 {
		return "", fmt.Errorf("target-owned test database root must be a real private directory with mode 0700")
	}
	return root, nil
}

func prepareTargetDownloadRoot(databaseRoot string) (string, error) {
	root, err := validateTargetTestDatabaseRoot(databaseRoot)
	if err != nil {
		return "", err
	}
	downloadRoot := filepath.Join(root, "downloads")
	if err := os.Mkdir(downloadRoot, 0o700); err != nil && !os.IsExist(err) {
		return "", fmt.Errorf("create target-owned browser download root: %w", err)
	} else if err == nil {
		if err := os.Chmod(downloadRoot, 0o700); err != nil {
			return "", fmt.Errorf("secure target-owned browser download root: %w", err)
		}
	}
	info, err := os.Lstat(downloadRoot)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o700 {
		return "", fmt.Errorf("target-owned browser download root must be a real private directory with mode 0700")
	}
	return downloadRoot, nil
}

func writeRestartHandoff(path string, handoff restartHandoff) error {
	if handoff.LeaseID == "" || handoff.TargetID == "" || handoff.ContextID == "" || handoff.Artifact == "" || handoff.Ticket == "" {
		return fmt.Errorf("restart handoff identity is incomplete")
	}
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return fmt.Errorf("restart handoff path must be absolute and clean")
	}
	parent := filepath.Dir(path)
	info, err := os.Lstat(parent)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o700 {
		return fmt.Errorf("restart handoff parent must be a private target-owned directory")
	}
	data, err := json.Marshal(handoff)
	if err != nil {
		return fmt.Errorf("encode restart handoff")
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create exclusive private restart handoff: %w", err)
	}
	_, writeErr := file.Write(data)
	syncErr := file.Sync()
	closeErr := file.Close()
	if writeErr != nil || syncErr != nil || closeErr != nil {
		_ = os.Remove(path)
		return fmt.Errorf("persist private restart handoff")
	}
	return nil
}

func readRestartHandoff(path, leaseID string, identity restartHandoffIdentity) (restartHandoff, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return restartHandoff{}, fmt.Errorf("restart handoff path must be absolute and clean")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o600 || info.Size() > 4096 {
		return restartHandoff{}, fmt.Errorf("restart handoff must be a small private regular file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return restartHandoff{}, fmt.Errorf("read restart handoff")
	}
	var handoff restartHandoff
	if err := json.Unmarshal(data, &handoff); err != nil || handoff.LeaseID != leaseID || handoff.TargetID != identity.TargetID || handoff.ContextID != identity.ContextID || handoff.Artifact != identity.Artifact || handoff.Ticket == "" {
		return restartHandoff{}, fmt.Errorf("restart handoff does not match the current target binding")
	}
	return handoff, nil
}

func disposableSQLiteDSNAt(root, leaseID string) (string, func(), error) {
	validated, err := validateTargetTestDatabaseRoot(root)
	if err != nil {
		return "", nil, err
	}
	if strings.TrimSpace(leaseID) == "" {
		return "", nil, fmt.Errorf("test database lease identity is required")
	}
	dir := disposableSQLiteDirectory(validated, leaseID)
	if err := os.Mkdir(dir, 0o700); err != nil {
		return "", nil, err
	}
	path := filepath.Join(dir, "routed-test.db")
	dsn, err := storage.SQLiteDSNAt(path, storage.SQLiteTuning{})
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", nil, err
	}
	return dsn, func() { _ = os.RemoveAll(dir) }, nil
}

func disposableSQLiteDirectory(root, leaseID string) string {
	digest := sha256.Sum256([]byte(leaseID))
	return filepath.Join(root, "workflow-test-db-"+hex.EncodeToString(digest[:]))
}

func removeDisposableSQLiteDSNAt(root, leaseID string) {
	validated, err := validateTargetTestDatabaseRoot(root)
	if err != nil || strings.TrimSpace(leaseID) == "" {
		return
	}
	_ = os.RemoveAll(disposableSQLiteDirectory(validated, leaseID))
}
