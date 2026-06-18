package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"vrooli-bridge/internal/auth"
	"vrooli-bridge/internal/clock"
	"vrooli-bridge/internal/cpkeys"
	"vrooli-bridge/internal/modules"
	"vrooli-bridge/internal/nodeauth"
	internalpairing "vrooli-bridge/internal/pairing"
	"vrooli-bridge/internal/presence"
	internalregistry "vrooli-bridge/internal/registry"
	"vrooli-bridge/internal/server"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	channelH "vrooli-bridge/handlers/channel"
	healthH "vrooli-bridge/handlers/health"
	pairingH "vrooli-bridge/handlers/pairing"
	registryH "vrooli-bridge/handlers/registry"
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
	registrar := registrarAdapter{svc: internalregistry.NewService(nodeLastSeen)}
	pairingSvc := internalpairing.NewService(pairingRepo, registrar, clk)

	// The node mutual-auth verifier reads node public keys from the pairing
	// repository (a revoked credential reads as absent). Threaded into the
	// channel module so every heartbeat + dial-out is authenticated.
	nodeVerifier := nodeauth.NewVerifier(pairingRepo)

	srv := server.New(
		server.Deps{Clock: clk, Logger: logger},
		healthH.Module(db, "vrooli-bridge-api", "1.0.0"),
		// registry RevokeNode performs atomic revocation: durable revoke +
		// credential destruction (pairingSvc) + live-channel drop (presenceHub).
		registryH.Module(db, clk, presenceHub, pairingSvc, presenceHub, logger),
		channelH.Module(presenceHub, nodeLastSeen, nodeVerifier, logger),
		pairingH.Module(pairingSvc, cpKeypair.PublicKeyBase64(), logger),
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
