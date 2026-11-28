package control

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"device-control/internal/androidartifact"
	authdomain "device-control/internal/auth"
	devicedomain "device-control/internal/devices"
	internalflows "device-control/internal/flows"
	"device-control/strategy"
	"device-control/strategy/androidtvremote"
	strategyregistry "device-control/strategy/registry"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
)

type routedDB interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type Service struct {
	registry   *strategyregistry.Registry
	db         routedDB
	fileRoots  *filerouting.RoutedRoots
	attached   AttachedReader
	hostNodeID string
	mu         sync.Mutex
	// Inventory and identity mutations remain ordered while probes run unlocked.
	inventoryGate     chan struct{}
	inventoryGateOnce sync.Once
	// Lease admission and cancellation must not wait for transport inventory.
	sessionMu               sync.Mutex
	sessions                map[string]Session
	audits                  []Audit
	agents                  map[string]AgentRun
	devices                 *devicedomain.Store
	artifacts               map[string]string
	artifactKinds           map[string]string
	evidenceDir             string
	activeCancels           map[string]context.CancelFunc
	transportStrategies     map[string]strategy.Strategy
	transportStates         map[string]transportState
	transportProfiles       map[string]transportState
	anchors                 *internalflows.AnchorStore
	runs                    map[string]RunResult
	flowRuns                map[string]Flow
	runDevices              map[string]string
	library                 internalflows.Library
	desktopRuns             internalflows.DesktopRuns
	retainedRuns            internalflows.RetainedRuns
	externalRecordings      map[string]externalRecording
	auth                    *authdomain.Store
	stateEvents             *strategy.EventBus
	observedStates          map[string]any
	observerCancels         map[string]*observerJob
	observerAdmissionClosed bool
	actuationCauses         map[string]actuationCause
	auditAliases            map[string]map[string]struct{}
	agentPlanner            internalflows.AgentPlanner
	apkInspector            androidartifact.APKInspector
	androidArtifactResolver interface {
		Resolve(context.Context, string, string, string) (androidartifact.StagedAPK, error)
	}
	inventoryTimeout    time.Duration
	reconnects          *reconnectGate
	sessionQueryTimeout time.Duration
	pendingPairings     map[string]pendingPairing
	bridgeInventoryErr  string
}

// SetHostNodeID supplies the local host identity resolved by the application
// composition root before inventory begins.
func (s *Service) SetHostNodeID(id string) {
	s.mu.Lock()
	s.hostNodeID = strings.TrimSpace(id)
	s.mu.Unlock()
}

// SetEvidenceDir supplies the evidence root resolved by the application
// composition root before artifact operations begin.
func (s *Service) SetEvidenceDir(dir string) {
	if strings.TrimSpace(dir) == "" {
		return
	}
	s.mu.Lock()
	s.evidenceDir = strings.TrimSpace(dir)
	s.mu.Unlock()
}

// lockInventory serializes inventory mutations while allowing canceled callers
// to leave the queue promptly. A plain Mutex.Lock cannot honor the bounded
// contexts used by state recovery and request handlers.
func (s *Service) lockInventory(ctx context.Context) (func(), error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.inventoryGateOnce.Do(func() { s.inventoryGate = make(chan struct{}, 1) })
	select {
	case s.inventoryGate <- struct{}{}:
		if err := ctx.Err(); err != nil {
			<-s.inventoryGate
			return nil, err
		}
		return func() { <-s.inventoryGate }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

type externalRecording struct {
	DeviceID  string
	LeaseID   string
	Actor     string
	Recorder  strategy.SessionRecorder
	Handle    strategy.RecordingHandle
	State     closeState
	Recovered *ExternalRecordingResult
}

type interactivePairer interface {
	BeginPairing(context.Context) (androidtvremote.PairingSession, error)
	CompletePairing(context.Context, androidtvremote.PairingSession, []byte) (strategy.PairResult, error)
}

type pendingPairing struct {
	deviceID  string
	pairer    interactivePairer
	session   androidtvremote.PairingSession
	expiresAt time.Time
	close     closeState
}

func New(registry *strategyregistry.Registry) *Service {
	return newService(registry, internalflows.NewAnchorStore())
}

func newService(registry *strategyregistry.Registry, anchors *internalflows.AnchorStore) *Service {
	dir := filepath.Join(os.TempDir(), "device-control-evidence")
	if configDir, err := os.UserConfigDir(); err == nil {
		dir = filepath.Join(configDir, "vrooli", "device-control", "evidence")
	}
	authStore, _ := authdomain.NewStore(nil, nil)
	return &Service{registry: registry, sessions: map[string]Session{}, audits: []Audit{}, agents: map[string]AgentRun{}, devices: devicedomain.NewStore(), artifacts: map[string]string{}, artifactKinds: map[string]string{}, evidenceDir: dir, activeCancels: map[string]context.CancelFunc{}, observerCancels: map[string]*observerJob{}, actuationCauses: map[string]actuationCause{}, auditAliases: map[string]map[string]struct{}{}, transportStrategies: map[string]strategy.Strategy{}, transportStates: map[string]transportState{}, transportProfiles: map[string]transportState{}, anchors: anchors, runs: map[string]RunResult{}, flowRuns: map[string]Flow{}, runDevices: map[string]string{}, externalRecordings: map[string]externalRecording{}, auth: authStore, stateEvents: strategy.NewEventBus(), observedStates: map[string]any{}, inventoryTimeout: defaultInventoryTimeout, reconnects: &reconnectGate{}, sessionQueryTimeout: 750 * time.Millisecond, pendingPairings: map[string]pendingPairing{}}
}

func NewWithAttached(registry *strategyregistry.Registry, reader AttachedReader) *Service {
	s := New(registry)
	s.attached = reader
	return s
}

func (s *Service) SetAgentPlanner(planner internalflows.AgentPlanner) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.agentPlanner = planner
}

// SetAndroidArtifactResolver wires the producer-owned immutable artifact
// boundary before device installation requests are admitted.
func (s *Service) SetAndroidArtifactResolver(resolver interface {
	Resolve(context.Context, string, string, string) (androidartifact.StagedAPK, error)
}) {
	s.mu.Lock()
	s.androidArtifactResolver = resolver
	s.mu.Unlock()
}

// NewWithDB keeps the in-memory registry fast while making operator state
// durable. The API already owns the SQLite connection; passing it here avoids
// a second connection and keeps test-mode routing under api-core's control.
func NewWithDB(registry *strategyregistry.Registry, db routedDB, roots ...*filerouting.RoutedRoots) (*Service, error) {
	s := newService(registry, nil)
	s.db = db
	if len(roots) > 0 {
		s.fileRoots = roots[0]
	}
	if db == nil {
		s.anchors = internalflows.NewAnchorStore()
		return s, nil
	}
	anchors, err := internalflows.NewAnchorStoreWithDB(db)
	if err != nil {
		return nil, err
	}
	s.anchors = anchors
	library, err := internalflows.NewSQLiteLibrary(db)
	if err != nil {
		return nil, err
	}
	s.library = library
	s.desktopRuns, err = internalflows.NewSQLiteDesktopRuns(context.Background(), db)
	if err != nil {
		return nil, err
	}
	s.retainedRuns, err = internalflows.NewSQLiteRetainedRuns(context.Background(), db)
	if err != nil {
		return nil, err
	}
	if _, err := db.ExecContext(context.Background(), `
CREATE TABLE IF NOT EXISTS device_control_audits (
 id TEXT PRIMARY KEY, actor TEXT NOT NULL, device_id TEXT NOT NULL, transport TEXT NOT NULL DEFAULT '', causation_id TEXT NOT NULL DEFAULT '', lease_id TEXT NOT NULL,
 verb TEXT NOT NULL, outcome TEXT NOT NULL, created_at TEXT NOT NULL, redaction_verified INTEGER NOT NULL,
	operation_id TEXT NOT NULL DEFAULT '',
	details_json TEXT NOT NULL DEFAULT '',
	redaction_opted_out INTEGER NOT NULL DEFAULT 0, profile_id TEXT NOT NULL DEFAULT '',
	interactive INTEGER NOT NULL DEFAULT 0, evidence_backed INTEGER NOT NULL DEFAULT 1,
 method TEXT NOT NULL DEFAULT '', attempts INTEGER NOT NULL DEFAULT 0,
 provider_state TEXT NOT NULL DEFAULT '', before_lock_state TEXT NOT NULL DEFAULT '',
 after_lock_state TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS device_control_transports (
 device_id TEXT PRIMARY KEY, serial TEXT NOT NULL, strategy_id TEXT NOT NULL,
 transport TEXT NOT NULL, endpoint TEXT NOT NULL, updated_at TEXT NOT NULL

);
CREATE TABLE IF NOT EXISTS device_control_transport_profiles (
 device_id TEXT NOT NULL, serial TEXT NOT NULL, strategy_id TEXT NOT NULL,
 transport TEXT NOT NULL, endpoint TEXT NOT NULL, updated_at TEXT NOT NULL,
 profile_json TEXT NOT NULL DEFAULT '',
 PRIMARY KEY (device_id, strategy_id, transport)

);
CREATE TABLE IF NOT EXISTS device_control_identity_claims (
 device_id TEXT NOT NULL, kind TEXT NOT NULL, value TEXT NOT NULL,
 strategy_id TEXT NOT NULL, evidence TEXT NOT NULL,
 PRIMARY KEY (device_id, kind, value)
);
CREATE TABLE IF NOT EXISTS device_control_identity_merges (
 canonical_id TEXT NOT NULL, member_id TEXT NOT NULL,
 claim_kind TEXT NOT NULL, claim_value TEXT NOT NULL,
 claim_strategy_id TEXT NOT NULL, claim_evidence TEXT NOT NULL,
 canonical_snapshot TEXT NOT NULL, member_snapshot TEXT NOT NULL,
 merged_at TEXT NOT NULL,
 PRIMARY KEY (canonical_id, member_id)

);
CREATE TABLE IF NOT EXISTS device_control_identity_aliases (
 canonical_id TEXT NOT NULL, alias_id TEXT NOT NULL,
 created_at TEXT NOT NULL,
 PRIMARY KEY (canonical_id, alias_id)
);
CREATE TABLE IF NOT EXISTS device_control_command_receipts (
 device_id TEXT NOT NULL, transport TEXT NOT NULL, idempotency_key TEXT NOT NULL,
 step_kind TEXT NOT NULL, run_id TEXT NOT NULL, outcome TEXT NOT NULL,
 created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
 PRIMARY KEY (device_id, transport, idempotency_key)
);`); err != nil {
		return nil, fmt.Errorf("initialize device-control state: %w", err)
	}
	// Existing local databases predate the opt-out audit field. This additive
	// migration is intentionally best-effort because SQLite reports a duplicate
	// column when the database has already been upgraded.
	_, _ = db.ExecContext(context.Background(), `ALTER TABLE device_control_audits ADD COLUMN redaction_opted_out INTEGER NOT NULL DEFAULT 0`)
	_, _ = db.ExecContext(context.Background(), `ALTER TABLE device_control_audits ADD COLUMN interactive INTEGER NOT NULL DEFAULT 0`)
	_, _ = db.ExecContext(context.Background(), `ALTER TABLE device_control_audits ADD COLUMN evidence_backed INTEGER NOT NULL DEFAULT 1`)
	_, _ = db.ExecContext(context.Background(), `ALTER TABLE device_control_transport_profiles ADD COLUMN profile_json TEXT NOT NULL DEFAULT ''`)
	_, _ = db.ExecContext(context.Background(), `ALTER TABLE device_control_audits ADD COLUMN operation_id TEXT NOT NULL DEFAULT ''`)
	_, _ = db.ExecContext(context.Background(), `ALTER TABLE device_control_audits ADD COLUMN details_json TEXT NOT NULL DEFAULT ''`)
	for _, column := range []string{
		`transport TEXT NOT NULL DEFAULT ''`,
		`causation_id TEXT NOT NULL DEFAULT ''`,
		`profile_id TEXT NOT NULL DEFAULT ''`,
		`method TEXT NOT NULL DEFAULT ''`,
		`attempts INTEGER NOT NULL DEFAULT 0`,
		`provider_state TEXT NOT NULL DEFAULT ''`,
		`before_lock_state TEXT NOT NULL DEFAULT ''`,
		`after_lock_state TEXT NOT NULL DEFAULT ''`,
	} {
		_, _ = db.ExecContext(context.Background(), `ALTER TABLE device_control_audits ADD COLUMN `+column)
	}
	if err := database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(func() string {
		return `
CREATE TABLE IF NOT EXISTS device_control_sessions (
 id TEXT PRIMARY KEY, device_id TEXT NOT NULL, actor TEXT NOT NULL,
 state TEXT NOT NULL, lease_token TEXT NOT NULL DEFAULT '', kill_reason TEXT NOT NULL DEFAULT '',
 expires_at TEXT NOT NULL, created_at TEXT NOT NULL, observation_only INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS device_control_sessions_device ON device_control_sessions(device_id,state);`
	})); err != nil {
		return nil, fmt.Errorf("ensure session schema: %w", err)
	}
	authStore, err := authdomain.NewStore(db, nil)
	if err != nil {
		return nil, err
	}
	s.auth = authStore
	if remote, ok := s.registry.Get("android-tv-remote"); ok {
		if configurable, ok := remote.(interface {
			SetCertificateStore(androidtvremote.CertificateStore)
		}); ok {
			configurable.SetCertificateStore(authStore)
		}
	}
	if err := s.loadSessions(); err != nil {
		return nil, err
	}
	if err := s.loadTransportStates(); err != nil {
		return nil, err
	}
	if err := s.loadIdentityClaims(); err != nil {
		return nil, err
	}
	if err := s.loadIdentityMerges(); err != nil {
		return nil, err
	}
	if err := s.loadIdentityAliases(); err != nil {
		return nil, err
	}
	s.restoreTransportStrategies()
	return s, nil
}

func (s *Service) Anchors() *internalflows.AnchorStore { return s.anchors }

func NewWithDBAndAttached(registry *strategyregistry.Registry, db routedDB, reader AttachedReader, roots ...*filerouting.RoutedRoots) (*Service, error) {
	s, err := NewWithDB(registry, db, roots...)
	if err != nil {
		return nil, err
	}
	s.attached = reader
	return s, nil
}
