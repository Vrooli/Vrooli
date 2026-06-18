package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	internalaudit "vrooli-bridge/internal/audit"
	"vrooli-bridge/internal/auth"
	"vrooli-bridge/internal/clock"
	"vrooli-bridge/internal/cpkeys"
	"vrooli-bridge/internal/modules"
	"vrooli-bridge/internal/nodeauth"
	internalpairing "vrooli-bridge/internal/pairing"
	"vrooli-bridge/internal/presence"
	internalqueue "vrooli-bridge/internal/queue"
	internalregistry "vrooli-bridge/internal/registry"
	internalruns "vrooli-bridge/internal/runs"
	"vrooli-bridge/internal/server"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	artifactsH "vrooli-bridge/handlers/artifacts"
	auditH "vrooli-bridge/handlers/audit"
	channelH "vrooli-bridge/handlers/channel"
	dispatchH "vrooli-bridge/handlers/dispatch"
	fleetH "vrooli-bridge/handlers/fleet"
	gateH "vrooli-bridge/handlers/gate"
	healthH "vrooli-bridge/handlers/health"
	pairingH "vrooli-bridge/handlers/pairing"
	provisionH "vrooli-bridge/handlers/provision"
	queueH "vrooli-bridge/handlers/queue"
	registryH "vrooli-bridge/handlers/registry"
	runsH "vrooli-bridge/handlers/runs"
)

// registrarAdapter bridges the registry service to the pairing domain's
// NodeRegistrar seam so pairing can create durable node records on redeem/
// approve without importing registry's proto-facing handler.
type registrarAdapter struct {
	svc internalregistry.Service
}

func (a registrarAdapter) RegisterNode(ctx context.Context, facts internalpairing.NodeFacts) (string, error) {
	node, err := a.svc.Register(ctx, internalregistry.RegisterInput{
		Name:         facts.Name,
		OS:           facts.OS,
		Arch:         facts.Arch,
		Endpoint:     facts.Endpoint,
		Capabilities: facts.Capabilities,
		Scopes:       facts.Scopes,
	})
	if err != nil {
		return "", err
	}
	return node.ID, nil
}

// sqliteDSN resolves the SQLite database file path and wraps it in a DSN
// with the canonical pragma string. Resolution order:
//
//  1. SQLITE_PATH env — the canonical override.
//  2. SQLITE_DB env — alias accepted for symmetry with other scenarios.
//  3. storage.NewResolver(ProfileAuto) — the storage-steer-mandated
//     filesystem-safe-by-default location.
//
// The path scope is the variant-aware namespace (storage.ScenarioNamespace),
// not the bare slug: under a Baseline Modes shadow engagement the lifecycle
// injects VROOLI_STORAGE_NAMESPACE, so the shadow's SQLite file lands beside
// "<scenario>_shadow" and never shares live's database. Outside the lifecycle
// (local `go run`, tests) it falls back to the compile-time slug, so live paths
// are unchanged. This is why a generated scenario is shadow-safe with zero
// per-scenario work — see packages/api-core/storage/namespace.go.
//
// The pragmas mirror agent-inbox; tweak in lockstep with
// internal/testutil/db.NewSQLite so production and tests open files the
// same way.
func sqliteDSN() (string, error) {
	if path := strings.TrimSpace(os.Getenv("SQLITE_PATH")); path != "" {
		return sqliteFileDSN(path)
	}
	if path := strings.TrimSpace(os.Getenv("SQLITE_DB")); path != "" {
		return sqliteFileDSN(path)
	}

	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("vrooli-bridge")
	if err != nil {
		return "", fmt.Errorf("resolve vrooli-bridge storage namespace: %w", err)
	}
	path, err := resolver.Path(
		storage.Options{ScenarioID: scenarioID},
		storage.ClassData,
		"vrooli-bridge.db",
	)
	if err != nil {
		return "", fmt.Errorf("resolve vrooli-bridge db path: %w", err)
	}
	return sqliteFileDSN(path)
}

func sqliteFileDSN(path string) (string, error) {
	if strings.HasPrefix(path, "file:") {
		return path, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("prepare sqlite directory: %w", err)
	}
	return fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=cache_size(-2000)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)",
		path,
	), nil
}

// cpKeyDir resolves the directory the control plane's long-lived Ed25519
// identity key is persisted in (internal/cpkeys). It mirrors sqliteDSN's
// resolution so the key lands in the same variant-aware namespace as the DB
// (shadow-safe). A BRIDGE_CP_KEY_DIR env override wins for tests/ops.
func cpKeyDir() (string, error) {
	if dir := strings.TrimSpace(os.Getenv("BRIDGE_CP_KEY_DIR")); dir != "" {
		return dir, nil
	}
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("vrooli-bridge")
	if err != nil {
		return "", fmt.Errorf("resolve vrooli-bridge storage namespace: %w", err)
	}
	path, err := resolver.Path(
		storage.Options{ScenarioID: scenarioID},
		storage.ClassData,
		filepath.Join("control-plane-keys", ".keep"),
	)
	if err != nil {
		return "", fmt.Errorf("resolve control-plane key dir: %w", err)
	}
	return filepath.Dir(path), nil
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "vrooli-bridge"}) {
		return
	}

	dsn, err := sqliteDSN()
	if err != nil {
		log.Fatalf("sqlite configuration failed: %v", err)
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}

	clk := clock.System{}
	logger := log.Default()

	// Owner identity is resolved against scenario-authenticator (the "Owner →
	// control plane" boundary, SECURITY.md). The resolver finds the
	// authenticator's URL by name via api-core/discovery (no env var); the
	// client verifies owner JWTs offline against its published RS256 key.
	authResolver := discovery.NewResolver(discovery.ResolverConfig{})
	authClient := auth.NewClient(auth.Config{Resolver: authResolver})

	// The presence hub is the in-memory view of which nodes hold a dial-out
	// channel and their self-reported health. It is shared with the registry
	// read path (which overlays live online/offline onto stored nodes) and the
	// channel handler (which opens a Conn per dial-out connection).
	presenceHub := presence.NewHub(clk)

	// The channel handler persists last-seen onto the registry's nodes table via
	// the repository's TouchLastSeen seam. It shares the same db/table the
	// registry module reads, so a heartbeat's last-seen is visible immediately.
	nodeLastSeen := internalregistry.NewSQLiteRepository(db, clk)

	// Pairing (OT-P0-002): the pairing service mints/burns codes and stores node
	// credentials. It registers redeeming nodes through the registry service
	// (the NodeRegistrar seam), and its repository doubles as the nodeauth
	// credential store and the registry atomic-revoke's CredentialRevoker.
	cpKeyDir, err := cpKeyDir()
	if err != nil {
		log.Fatalf("resolve control-plane key dir: %v", err)
	}
	cpKeypair, err := cpkeys.LoadOrCreate(cpKeyDir)
	if err != nil {
		log.Fatalf("load control-plane identity key: %v", err)
	}
	pairingRepo := internalpairing.NewSQLiteRepository(db, clk)
	// One registry service instance is shared by the pairing registrar (creates
	// node records on redeem) and the dispatch handler (reads node scopes to
	// authorize a job). Both read/write the same `nodes` table.
	registrySvc := internalregistry.NewService(nodeLastSeen)
	registrar := registrarAdapter{svc: registrySvc}
	pairingSvc := internalpairing.NewService(pairingRepo, registrar, clk)

	// The node mutual-auth verifier reads node public keys from the pairing
	// repository (a revoked credential reads as absent). Threaded into the
	// channel + runs modules so every heartbeat, dial-out, and run-event report
	// is authenticated.
	nodeVerifier := nodeauth.NewVerifier(pairingRepo)

	// Runs (OT-P0-005): a single durable-run service instance is shared by the
	// runs handler (operator verbs + node-facing ReportRunEvent ingest) and the
	// dispatch handler (Create), so the in-memory block-once waiter and
	// live-event subscriber coordination is one coherent instance.
	//
	// queue (OT-P1-004): the per-node job scheduler sits on the dispatch → push
	// path (bounded concurrency + fair FIFO) and is shared with the runs terminal
	// hook (free a slot + promote the next queued job when a run finishes) and
	// AbortRun's node-cancel push. The hook closure captures the `scheduler` var,
	// assigned just below, so the runs service and the scheduler reference each
	// other without a construction cycle.
	var scheduler *internalqueue.Scheduler
	runsSvc := internalruns.NewService(
		internalruns.NewSQLiteRepository(db, clk), clk,
		internalruns.WithCanceller(queueH.NewChannelCanceller(presenceHub)),
		internalruns.WithTerminalHook(func(ctx context.Context, run internalruns.Run) {
			if scheduler != nil {
				scheduler.Complete(ctx, run.NodeID, run.ID)
			}
		}),
	)
	scheduler = internalqueue.NewScheduler(queueH.NewChannelPusher(presenceHub), queueH.NewAborter(runsSvc), clk, 0)

	// Audit (OT-P0-008): the append-only accountability substrate. The same
	// store is the dispatch handler's write Sink and the audit handler's read
	// Reader. (A workspace-sandbox-backed Sink is the documented alternative
	// behind the same seam; see docs/internal/SECURITY.md + PROBLEMS.md.)
	auditStore := internalaudit.NewSQLiteStore(db, clk)

	// provision (OT-P0-006): the PRIVILEGED tier. Built once here so the same
	// instance backs both the provision handler (operator verbs + node ingest)
	// and the fleet roll's provisioner adapter — the in-memory op coordination
	// stays coherent across both call sites.
	provisionSvc := provisionH.NewService(db, clk, registrySvc, presenceHub, auditStore)

	// dispatch (OT-P0-004): the allowlist gate, built once here so the SAME
	// instance backs both the dispatch handler and the gate domain's runner
	// adapter — every cross-OS gate validation run flows through the same
	// allowlist + per-node scopes + audit gate as any other job.
	dispatchSvc := dispatchH.NewService(registrySvc, runsSvc, auditStore, presenceHub, scheduler)

	srv := server.New(
		server.Deps{Clock: clk, Logger: logger},
		healthH.Module(db, "vrooli-bridge-api", "1.0.0"),
		// registry RevokeNode performs atomic revocation: durable revoke +
		// credential destruction (pairingSvc) + live-channel drop (presenceHub).
		registryH.Module(db, clk, presenceHub, pairingSvc, presenceHub, logger),
		channelH.Module(presenceHub, nodeLastSeen, nodeVerifier, logger),
		pairingH.Module(pairingSvc, cpKeypair.PublicKeyBase64(), logger),
		// dispatch (OT-P0-004): the allowlist gate. It reads node scopes
		// (registrySvc), checks presence + protocol compatibility, creates durable
		// runs (runsSvc), audits (auditStore), and submits typed jobs to the
		// per-node scheduler (bounded concurrency on the channel-push path).
		dispatchH.Module(dispatchSvc, logger),
		// runs (OT-P0-005): durable run lifecycle + node-facing event ingest.
		runsH.Module(runsSvc, nodeVerifier, logger),
		// queue (OT-P1-004): read-only control-plane view over the per-node
		// scheduler (which jobs are running vs queued, per node).
		queueH.Module(scheduler, logger),
		// provision (OT-P0-006): the PRIVILEGED tier. Owns its durable op tables;
		// reads node revocation (registrySvc), checks presence + pushes the
		// privileged ProvisionCommand (presenceHub), audits (auditStore), and
		// gates the node-facing ReportProvisionEvent on mutual auth (nodeVerifier).
		provisionH.Module(provisionSvc, nodeVerifier, logger),
		// fleet (OT-P1-001): fleet-wide version roll. Enumerates nodes
		// (registrySvc), gates on presence + protocol compatibility (presenceHub),
		// and dispatches a privileged provisioning op per eligible node by
		// delegating to the shared provision service (provisionSvc).
		fleetH.Module(db, clk, registrySvc, presenceHub, provisionSvc, logger),
		// gate (OT-P1-002): cross-OS deployment gate. Selects one eligible node
		// per target OS (registrySvc + presenceHub), dispatches a validation run to
		// each by delegating to the shared dispatch service (dispatchSvc) + runs
		// service (runsSvc), and aggregates per-OS verdicts into one cross-OS
		// result deployment-manager owns. Owns its durable gate tables.
		gateH.Module(db, clk, registrySvc, presenceHub, dispatchSvc, runsSvc, logger),
		// artifacts (OT-P1-003): non-git artifact distribution. Validates node
		// revocation (registrySvc) and delegates the byte move to device-sync-hub
		// directed delivery (bridge stores no blob).
		artifactsH.Module(db, clk, registrySvc, logger),
		// audit (OT-P0-008): owner-gated read of the append-only trail.
		auditH.Module(auditStore, logger),
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.Register(rootMux, db)

	// auth.Middleware best-effort-injects the owner Identity when a valid
	// bearer token is present; owner-gated RPCs fail closed via RequireOwner.
	rootMux.Handle("/", auth.Middleware(authClient, logger)(srv.Handler()))

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// request context so *database.RoutedDB routes the call to the
	// installed test pool. Self-disables in production mode.
	handler := apihttp.TestModeMiddleware(rootMux)

	if err := apiserver.Run(apiserver.Config{
		Handler: handler,
		Cleanup: func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
