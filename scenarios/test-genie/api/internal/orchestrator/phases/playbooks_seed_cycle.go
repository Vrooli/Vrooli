package phases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"test-genie/internal/orchestrator/phases/dbdetect"
	"test-genie/internal/orchestrator/phases/isolation"
	"test-genie/internal/orchestrator/phases/seeds"
	"test-genie/internal/orchestrator/testconfig/config"
	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/shared"
	"test-genie/internal/storage/sqlfiles"

	"github.com/vrooli/api-core/database"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing/routing_v1connect"
	seedspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/seeds"
	"google.golang.org/protobuf/proto"
	// Register modernc.org/sqlite as the pure-Go "sqlite" driver.
	_ "modernc.org/sqlite"

	sharedartifacts "test-genie/internal/shared/artifacts"
)

const seedCleanupTimeout = 30 * time.Second
const seedAllocationAllowance = 2 * time.Minute
const maxSeedStateBytes = 1 << 20

// PlaybooksSeedSession holds state for a seed lifecycle run.
type PlaybooksSeedSession struct {
	RunID           string
	Env             map[string]string
	Resources       []isolation.ResourceInfo
	SeedState       map[string]any
	CleanupRef      string
	routing         routedLease
	cleanup         func(ctx context.Context) error
	cleanupMu       sync.Mutex
	cleanupErr      error
	cleanupComplete bool
	setupErr        error
	recovery        *seedCheckpoint
}

type routedLease struct {
	client           routingconnect.RoutingServiceClient
	leaseID          string
	heartbeatCancel  context.CancelFunc
	heartbeatStopped chan struct{}
	state            *routedLeaseState
}

type routedLeaseState struct {
	mu           sync.Mutex
	lastAck      time.Time
	heartbeatErr error
	clearErr     error
	stats        *routingv1.LeaseStats
	cleared      bool
	installed    bool
}

type resourceNeeds struct {
	RequirePostgres bool
	RequireRedis    bool
	RequireSQLite   bool
	// PrimaryDriver identifies the scenario's primary database driver
	// ("postgres" or "sqlite") when detection has strong evidence
	// (e.g. a Go driver import). Empty when no single driver dominates.
	// Used by the routed path to pick a DSN that matches the scenario's
	// actual driver, avoiding hangs from cross-driver DSN injection.
	PrimaryDriver string
	SQLiteEnvVars []string
}

// isolationProvider lets tests stub seed isolation without requiring Docker.
type isolationProvider interface {
	Prepare(ctx context.Context) (*isolation.Result, error)
}

var isolationManagerFactory = func(cfg isolation.Config) isolationProvider {
	return isolation.NewManager(cfg)
}

// Cleanup clears the live target routing lease and tears down the leased
// backing resources. The target remains on its primary configuration.
func (s *PlaybooksSeedSession) Cleanup(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if s.cleanup == nil {
		return errors.New("seed cleanup ownership is unavailable")
	}
	s.cleanupMu.Lock()
	defer s.cleanupMu.Unlock()
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), seedCleanupTimeout)
	defer cancel()
	if err := s.cleanup(cleanupCtx); err != nil && s.cleanupErr == nil {
		s.cleanupErr = err
	}
	if err := s.checkpoint(); err != nil && s.cleanupErr == nil {
		s.cleanupErr = err
	}
	return s.cleanupErr
}

// SeedEvidence contains no bearer token, DSN or filesystem path. It reports
// the retained owner outcome rather than deriving success from a session object.
func (s *PlaybooksSeedSession) SeedEvidence() *seedspb.SeedEvidence {
	s.cleanupMu.Lock()
	defer s.cleanupMu.Unlock()
	out := &seedspb.SeedEvidence{LeaseId: s.RunID}
	state := s.routing.state
	if state == nil {
		return out
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	out.PairedStorageInstalled = state.installed
	out.State = seedspb.SeedState_SEED_STATE_ACTIVE
	if state.cleared {
		out.State = seedspb.SeedState_SEED_STATE_CLEARED
	}
	if !state.cleared && state.heartbeatErr == nil && time.Since(state.lastAck) > 3*time.Minute {
		state.heartbeatErr = errors.New("seed lease heartbeat is stale")
	}
	if state.heartbeatErr != nil {
		out.HeartbeatError = "seed lease heartbeat failed; inspect retained owner evidence"
		out.State = seedspb.SeedState_SEED_STATE_FAILED
	}
	if err := errors.Join(s.setupErr, state.clearErr, s.cleanupErr); err != nil {
		out.CleanupError = "seed lifecycle failed; inspect retained owner evidence"
		out.State = seedspb.SeedState_SEED_STATE_FAILED
	}
	if state.stats != nil {
		out.Stats = proto.Clone(state.stats).(*routingv1.LeaseStats)
	}
	return out
}

// CleanupComplete reports resource retirement independently of validation success.
func (s *PlaybooksSeedSession) CleanupComplete() bool {
	s.cleanupMu.Lock()
	defer s.cleanupMu.Unlock()
	return s.cleanupComplete
}

// FailSeed revokes execution proof while retaining exact cleanup ownership.
func (s *PlaybooksSeedSession) FailSeed(cause error) {
	s.cleanupMu.Lock()
	defer s.cleanupMu.Unlock()
	if s.setupErr == nil {
		s.setupErr = cause
	}
	if s.routing.heartbeatCancel != nil {
		s.routing.heartbeatCancel()
	}
	if err := s.checkpoint(); err != nil && s.cleanupErr == nil {
		s.cleanupErr = err
	}
}

// ApplyPlaybooksSeed provisions isolated resources, installs a runtime routing
// lease on the live target, runs seed scripts, and returns seed state for BAS
// workflow execution. It refuses when the paired routing surface is absent.
func ApplyPlaybooksSeed(ctx context.Context, env workspace.Environment, logWriter io.Writer, retain bool) (*PlaybooksSeedSession, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	playbooksCfg, err := config.Load(env.ScenarioDir)
	if err != nil {
		return nil, fmt.Errorf("load seed configuration: %w", err)
	}
	if playbooksCfg != nil && !playbooksCfg.Seeds.Enabled {
		return nil, fmt.Errorf("playbooks seeds disabled via .vrooli/testing.json")
	}
	// timeout_ms remains the script's own budget. The allocation allowance also
	// bounds all preceding setup, including migrations, within one owner budget.
	ctx, cancelSetup := context.WithTimeout(ctx, seedAllocationAllowance+playbooksCfg.Seeds.SeedTimeout())
	defer cancelSetup()

	needs := resolveDBNeeds(ctx, env, logWriter)
	checkpoint := newSeedCheckpoint(ctx)
	checkpoint.Retain = retain
	if err := checkpoint.save(); err != nil {
		return nil, err
	}
	allocationRoot := ""
	if checkpoint.path != "" {
		allocationRoot = filepath.Join(filepath.Dir(checkpoint.path), "backing")
	}
	isoManager := isolationManagerFactory(isolation.Config{
		ScenarioName:    env.ScenarioName,
		RequirePostgres: needs.RequirePostgres,
		RequireRedis:    needs.RequireRedis,
		RequireSQLite:   needs.RequireSQLite,
		SQLiteEnvVars:   needs.SQLiteEnvVars,
		Retain:          retain,
		LogWriter:       logWriter,
		Timeout:         seedAllocationAllowance,
		RunID:           checkpoint.RunID,
		AllocationRoot:  allocationRoot,
		OnAllocation: func(a isolation.Allocation) error {
			checkpoint.Allocations = append(checkpoint.Allocations, a)
			return checkpoint.save()
		},
	})

	isoResult, err := isoManager.Prepare(ctx)
	if isoResult == nil {
		return nil, fmt.Errorf("failed to prepare playbooks isolation: %w", err)
	}
	// Ownership begins as soon as any allocation exists, including a partial
	// Prepare result. Early setup errors retain the same retryable cleanup owner.
	session := newSeedSession(isoResult, routedLease{state: &routedLeaseState{cleared: true}}, logWriter)
	session.recovery = checkpoint
	retireFailedSeed := func(cause error) (*PlaybooksSeedSession, error) {
		session.setupErr = cause
		return session, errors.Join(cause, session.Cleanup(ctx))
	}
	if err != nil {
		return retireFailedSeed(fmt.Errorf("failed to prepare playbooks isolation: %w", err))
	}
	if err := applyPlaybooksMigrations(ctx, env, needs, isoResult.Env, logWriter); err != nil {
		return retireFailedSeed(fmt.Errorf("failed to apply playbooks migrations: %w", err))
	}
	if err := ctx.Err(); err != nil {
		return retireFailedSeed(err)
	}
	checkpoint.TargetURL = env.APIURL
	checkpoint.RoutingAttempted = true
	checkpoint.Cleared = false
	if err := checkpoint.save(); err != nil {
		return retireFailedSeed(err)
	}
	routing, err := installRoutedLease(ctx, env, needs, isoResult, logWriter)
	if routing.client != nil {
		session = newSeedSession(isoResult, routing, logWriter)
		session.recovery = checkpoint
	}
	if err != nil {
		return retireFailedSeed(err)
	}
	if err := session.checkpoint(); err != nil {
		return retireFailedSeed(err)
	}

	seedCtx, cancel := context.WithTimeout(ctx, playbooksCfg.Seeds.SeedTimeout())
	defer cancel()
	if err := seedCtx.Err(); err != nil {
		return retireFailedSeed(err)
	}

	seedManager := seeds.NewManager(env.ScenarioDir, env.AppRoot, env.CoverageDir, logWriter).WithTargetAPIBase(env.APIURL).WithIsolationEnv(isoResult.Env)
	if !seedManager.HasSeeds() {
		return retireFailedSeed(errors.New("declared seed lifecycle has no supported seed entrypoint"))
	}
	if checkpoint.path != "" {
		seedManager.WithOperationRoot(filepath.Join(filepath.Dir(checkpoint.path), "gates", checkpoint.RunID))
	}
	seedManager.WithOperationRecorder(func(operation *seeds.OperationRecord) error {
		checkpoint.Operation = operation
		return session.checkpoint()
	})
	seedStatePath := sharedartifacts.SeedStatePath(env.ScenarioDir)
	if err := os.Remove(seedStatePath); err != nil && !os.IsNotExist(err) {
		return retireFailedSeed(fmt.Errorf("retire previous seed state: %w", err))
	}
	_, seedErr := seedManager.Apply(seedCtx)
	if seedErr != nil {
		return retireFailedSeed(fmt.Errorf("seed execution failed: %w", seedErr))
	}

	seedState, err := loadSeedState(env.ScenarioDir)
	if err != nil {
		return retireFailedSeed(err)
	}

	session.SeedState = seedState
	if err := session.checkpoint(); err != nil {
		return retireFailedSeed(err)
	}
	if session.SeedEvidence().GetState() != seedspb.SeedState_SEED_STATE_ACTIVE {
		return retireFailedSeed(errors.New("seed lease health failed during setup"))
	}

	return session, nil
}

func newSeedSession(isoResult *isolation.Result, routing routedLease, logWriter io.Writer) *PlaybooksSeedSession {
	session := &PlaybooksSeedSession{
		RunID:     isoResult.RunID,
		Env:       isoResult.Env,
		Resources: isoResult.Resources,

		routing: routing,
	}
	session.cleanup = func(cleanupCtx context.Context) error {
		// No routing or backing retirement may race a still-running seed child.
		if session.recovery != nil {
			if err := seeds.StopOperation(cleanupCtx, session.recovery.Operation); err != nil {
				return fmt.Errorf("seed descendants are not quiescent: %w", err)
			}
			if err := session.checkpoint(); err != nil {
				return err
			}
			if err := seeds.ReleaseOperation(cleanupCtx, session.recovery.Operation); err != nil {
				return fmt.Errorf("retire seed native operation evidence: %w", err)
			}
			if err := session.checkpoint(); err != nil {
				return err
			}
		}
		clearErr := clearRoutedLease(cleanupCtx, session.routing, logWriter)
		session.routing.state.mu.Lock()
		cleared := session.routing.state.cleared
		session.routing.state.mu.Unlock()
		// Never delete a database that the target might still have installed.
		if !cleared {
			return clearErr
		}
		// Persist exact clear counters before deleting their backing. A journal
		// failure retains resources and the cleanup handle for another attempt.
		if err := session.checkpoint(); err != nil {
			return errors.Join(clearErr, err)
		}
		if !session.cleanupComplete {
			if err := isoResult.Cleanup(cleanupCtx); err != nil {
				return errors.Join(clearErr, fmt.Errorf("clean up seed backing resources: %w", err))
			}
			session.cleanupComplete = true
		}
		return clearErr
	}

	return session
}

func cleanupSeedResources(parent context.Context, result *isolation.Result) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), seedCleanupTimeout)
	defer cancel()
	return result.Cleanup(ctx)
}

func installRoutedLease(ctx context.Context, env workspace.Environment, needs resourceNeeds, result *isolation.Result, logWriter io.Writer) (routedLease, error) {
	if result == nil || strings.TrimSpace(env.APIURL) == "" {
		return routedLease{}, fmt.Errorf("routed qualification requires a live target API URL")
	}
	dsn := routedDSN(needs, result.Env)
	if dsn == "" {
		return routedLease{}, fmt.Errorf("routed qualification requires a DSN matching the detected primary database driver")
	}
	client := routingconnect.NewRoutingServiceClient(&http.Client{Timeout: 15 * time.Second}, strings.TrimRight(env.APIURL, "/"))
	state := &routedLeaseState{lastAck: time.Now()}
	lease := routedLease{client: client, leaseID: result.RunID, state: state}
	response, err := client.InstallTestPool(ctx, connect.NewRequest(&routingv1.InstallTestPoolRequest{Dsn: dsn, LeaseId: result.RunID, LeaseTtlMs: int64((3 * time.Minute) / time.Millisecond)}))
	if err != nil {
		return lease, fmt.Errorf("install routed qualification lease: %w", err)
	}
	if response == nil || response.Msg == nil || response.Msg.GetActiveLeaseId() != result.RunID || !response.Msg.GetFileRootsInstalled() {
		return lease, fmt.Errorf("routed qualification lease did not prove both database and file roots are installed")
	}
	state.installed = true
	// Publish only the verified lease identity to seed/browser producers.
	result.Env["TEST_GENIE_ROUTING_LEASE_ID"] = result.RunID
	shared.LogInfo(logWriter, "installed routed qualification lease %s on %s", result.RunID, env.APIURL)
	heartbeatCtx, heartbeatCancel := context.WithCancel(context.Background())
	heartbeatStopped := make(chan struct{})
	go heartbeatRoutedLease(heartbeatCtx, client, result.RunID, logWriter, heartbeatStopped, state)
	lease.heartbeatCancel = heartbeatCancel
	lease.heartbeatStopped = heartbeatStopped
	return lease, nil
}

func clearRoutedLease(ctx context.Context, lease routedLease, logWriter io.Writer) error {
	if lease.client == nil || lease.leaseID == "" {
		return nil
	}
	if lease.heartbeatCancel != nil {
		lease.heartbeatCancel()
		if lease.heartbeatStopped != nil {
			<-lease.heartbeatStopped
		}
	}
	state := lease.state
	if state == nil {
		return errors.New("seed lease has no retained owner state")
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.cleared {
		return state.clearErr
	}
	response, err := lease.client.ClearTestPool(ctx, connect.NewRequest(&routingv1.ClearTestPoolRequest{LeaseId: lease.leaseID}))
	if err != nil {
		if state.clearErr == nil {
			state.clearErr = fmt.Errorf("clear routed qualification lease: %w", err)
		}
		return state.clearErr
	}
	if response == nil || response.Msg == nil || response.Msg.GetStats() == nil {
		if state.clearErr == nil {
			state.clearErr = errors.New("clear routed qualification lease returned no routing statistics")
		}
		return state.clearErr
	}
	stats := response.Msg.GetStats()
	state.stats = proto.Clone(stats).(*routingv1.LeaseStats)
	state.cleared = true
	shared.LogInfo(logWriter, "cleared routed qualification lease %s: test_pool_requests=%d primary_during_test_mode_requests=%d test_root_writes=%d primary_root_writes_during_test_mode=%d", lease.leaseID, stats.GetTestPoolRequests(), stats.GetPrimaryDuringTestModeRequests(), stats.GetTestRootWrites(), stats.GetPrimaryRootWritesDuringTestMode())
	if stats.GetPrimaryDuringTestModeRequests() != 0 || stats.GetPrimaryRootWritesDuringTestMode() != 0 {
		if state.clearErr == nil {
			state.clearErr = fmt.Errorf("routed qualification detected primary storage use during test mode: db=%d files=%d", stats.GetPrimaryDuringTestModeRequests(), stats.GetPrimaryRootWritesDuringTestMode())
		}
	}
	return state.clearErr
}

func heartbeatRoutedLease(ctx context.Context, client routingconnect.RoutingServiceClient, leaseID string, logWriter io.Writer, stopped chan<- struct{}, state *routedLeaseState) {
	defer close(stopped)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := client.HeartbeatTestPool(ctx, connect.NewRequest(&routingv1.HeartbeatTestPoolRequest{LeaseId: leaseID})); err != nil {
				if ctx.Err() != nil {
					return
				}
				shared.LogWarn(logWriter, "routed qualification lease heartbeat failed for %s: %v", leaseID, err)
				state.mu.Lock()
				state.heartbeatErr = errors.New("target seed lease heartbeat failed")
				state.mu.Unlock()
				return
			}
			state.mu.Lock()
			state.lastAck = time.Now()
			state.mu.Unlock()
		}
	}
}

func routedDSN(needs resourceNeeds, env map[string]string) string {
	if needs.PrimaryDriver == "postgres" {
		return firstNonEmpty(env["DATABASE_URL"], env["POSTGRES_URL"])
	}
	if needs.PrimaryDriver == "sqlite" {
		return firstNonEmpty(env["PLAYBOOKS_SQLITE_DSN"], env["PLAYBOOKS_SQLITE_PATH"])
	}
	return ""
}

// applyPlaybooksMigrations applies optional .sql files under bas/seeds/migrations
// against the isolated database backend. Files execute in lexicographic order.
func applyPlaybooksMigrations(ctx context.Context, env workspace.Environment, needs resourceNeeds, isolatedEnv map[string]string, logWriter io.Writer) error {
	if !needs.RequirePostgres && !needs.RequireSQLite {
		return nil
	}

	migrationsDir := filepath.Join(env.ScenarioDir, "bas", "seeds", "migrations")
	commonFiles, err := collectMigrationFiles(migrationsDir, "common")
	if err != nil {
		return err
	}
	postgresFiles, err := collectScopedMigrationFiles(migrationsDir, "postgres")
	if err != nil {
		return err
	}
	sqliteFiles, err := collectScopedMigrationFiles(migrationsDir, "sqlite")
	if err != nil {
		return err
	}
	if len(commonFiles) == 0 && len(postgresFiles) == 0 && len(sqliteFiles) == 0 {
		return nil
	}

	if needs.RequirePostgres {
		files := append([]string(nil), commonFiles...)
		files = append(files, postgresFiles...)
		if err := applyPostgresMigrations(ctx, env, isolatedEnv, files, logWriter); err != nil {
			return err
		}
	}
	if needs.RequireSQLite {
		files := append([]string(nil), commonFiles...)
		files = append(files, sqliteFiles...)
		if err := applySQLiteMigrations(ctx, env, isolatedEnv, files, logWriter); err != nil {
			return err
		}
	}
	return nil
}

func applyPostgresMigrations(ctx context.Context, env workspace.Environment, isolatedEnv map[string]string, files []string, logWriter io.Writer) error {
	if err := EnsureCommandAvailable("psql"); err != nil {
		return fmt.Errorf("psql not available for playbooks migrations: %w", err)
	}
	connURL := strings.TrimSpace(firstNonEmpty(isolatedEnv["DATABASE_URL"], isolatedEnv["POSTGRES_URL"]))
	if connURL == "" {
		return fmt.Errorf("DATABASE_URL is not set for playbooks migrations")
	}
	shared.LogStep(logWriter, "applying playbooks migrations (%d file(s))", len(files))
	for _, file := range files {
		shared.LogInfo(logWriter, "  psql -f %s", file)
		if err := phaseCommandExecutor(ctx, env.ScenarioDir, logWriter, "psql", "-d", connURL, "-v", "ON_ERROR_STOP=1", "-f", file); err != nil {
			return fmt.Errorf("psql apply %s: %w", file, err)
		}
	}
	return nil
}

func applySQLiteMigrations(ctx context.Context, env workspace.Environment, isolatedEnv map[string]string, files []string, logWriter io.Writer) error {
	// Only the playbooks-scoped variables are read. The generic pair used to be
	// accepted here as a fallback, which made this migration path a consumer of
	// whatever database path happened to be in the environment.
	sqliteDSN := strings.TrimSpace(firstNonEmpty(
		isolatedEnv["PLAYBOOKS_SQLITE_DSN"],
		isolatedEnv["PLAYBOOKS_SQLITE_PATH"],
	))
	if sqliteDSN == "" {
		return fmt.Errorf("PLAYBOOKS_SQLITE_DSN is not set for playbooks migrations; the isolation manager supplies it")
	}
	db, err := database.Connect(ctx, database.Config{
		Driver:       database.DriverSQLite,
		DSN:          sqliteDSN,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		return fmt.Errorf("open sqlite migrations database: %w", err)
	}
	defer db.Close()

	shared.LogStep(logWriter, "applying playbooks sqlite migrations (%d file(s))", len(files))
	for _, file := range files {
		shared.LogInfo(logWriter, "  sqlite migrate %s", file)
		if err := sqlfiles.ExecFileContext(ctx, db, file); err != nil {
			return fmt.Errorf("sqlite apply %s: %w", file, err)
		}
	}
	return nil
}

func collectMigrationFiles(root, scopedDir string) ([]string, error) {
	legacyFiles, err := collectScopedMigrationFiles(root, "")
	if err != nil {
		return nil, err
	}
	scopedFiles, err := collectScopedMigrationFiles(root, scopedDir)
	if err != nil {
		return nil, err
	}
	merged := append([]string(nil), legacyFiles...)
	merged = append(merged, scopedFiles...)
	return merged, nil
}

func collectScopedMigrationFiles(root, scopedDir string) ([]string, error) {
	targetDir := root
	if scopedDir != "" {
		targetDir = filepath.Join(root, scopedDir)
	}
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read migrations dir %s: %w", targetDir, err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		files = append(files, filepath.Join(targetDir, entry.Name()))
	}
	sort.Strings(files)
	return files, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// resolveDBNeeds runs the dbdetect resolver against the
// scenario, writes the evidence chain to logWriter, and projects the report
// to the booleans the isolation manager consumes. There is no fallback: a
// scenario with no evidence is provisioned with nothing, which is the
// correct signal that detection should be fixed at the source.
func resolveDBNeeds(ctx context.Context, env workspace.Environment, logWriter io.Writer) resourceNeeds {
	manifestPath := filepath.Join(env.ScenarioDir, ".vrooli", "service.json")
	rawManifest, err := workspace.LoadServiceManifest(manifestPath)
	if err != nil {
		shared.LogWarn(logWriter, "unable to read service manifest (%v); proceeding with file-based detection only", err)
	}
	manifest := dbdetect.WrapManifest(rawManifest)

	resolver, err := dbdetect.NewResolver(dbdetect.DefaultCollectors(), dbdetect.DefaultProfiles())
	if err != nil {
		shared.LogWarn(logWriter, "db-detect resolver construction failed: %v", err)
		return resourceNeeds{SQLiteEnvVars: manifest.SQLitePathEnvVars()}
	}

	report := resolver.Resolve(ctx, dbdetect.ScenarioInputs{
		ScenarioDir: env.ScenarioDir,
		Manifest:    manifest,
		Filesystem:  dbdetect.OSFilesystem{},
	})
	if logWriter != nil {
		_, _ = logWriter.Write([]byte(report.FormatHuman()))
	}
	needs := resourceNeeds{
		RequirePostgres: report.Required("postgres"),
		RequireRedis:    report.Required("redis"),
		RequireSQLite:   report.Required("sqlite"),
		PrimaryDriver:   primaryDriver(report),
		SQLiteEnvVars:   manifest.SQLitePathEnvVars(),
	}
	if needs.PrimaryDriver == "" {
		shared.LogWarn(logWriter, "db-detect did not pick a primary driver — routed path will not be used")
	} else {
		shared.LogInfo(logWriter, "db-detect primary driver = %s", needs.PrimaryDriver)
	}
	return needs
}

// primaryDriver returns "postgres" or "sqlite" when one has strictly
// stronger evidence than the other (per Evidence.Priority), and empty
// otherwise. The empty case means "no winner — fall back to the caller's
// default DSN order" (caller-defined behavior).
func primaryDriver(report dbdetect.DetectionReport) string {
	pgPriority := decisionPriority(report, "postgres")
	sqlitePriority := decisionPriority(report, "sqlite")
	if pgPriority > sqlitePriority {
		return "postgres"
	}
	if sqlitePriority > pgPriority {
		return "sqlite"
	}
	return ""
}

func decisionPriority(report dbdetect.DetectionReport, db string) dbdetect.Priority {
	res, ok := report.Results[db]
	if !ok || !res.Required || res.Decision == nil {
		return 0
	}
	return res.Decision.Priority
}

func loadSeedState(scenarioDir string) (map[string]any, error) {
	seedPath := sharedartifacts.SeedStatePath(scenarioDir)
	file, err := os.Open(seedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxSeedStateBytes+1))
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	if len(data) > maxSeedStateBytes {
		return nil, fmt.Errorf("seed state exceeds %d bytes", maxSeedStateBytes)
	}
	var state map[string]any
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse seed state JSON: %w", err)
	}
	// The journal and wire contract encode values again. Raw JSON containing
	// HTML characters can expand sixfold, so the source byte limit alone does
	// not establish a bounded value safe to publish into cleanup authority.
	encoded, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("encode seed state: %w", err)
	}
	if len(encoded) > maxSeedStateBytes {
		return nil, fmt.Errorf("encoded seed state exceeds %d bytes", maxSeedStateBytes)
	}
	return state, nil
}
