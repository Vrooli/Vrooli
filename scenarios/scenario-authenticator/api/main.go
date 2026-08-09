package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"scenario-authenticator/internal/clock"
	"scenario-authenticator/internal/modules"
	"scenario-authenticator/internal/server"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	"github.com/vrooli/api-core/trustposture"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
	authH "scenario-authenticator/handlers/auth"
	healthH "scenario-authenticator/handlers/health"
	jwksH "scenario-authenticator/handlers/jwks"
	sessionsH "scenario-authenticator/handlers/sessions"
	"scenario-authenticator/internal/accounts"
	"scenario-authenticator/internal/audit"
	"scenario-authenticator/internal/authcrypto"
	"scenario-authenticator/internal/authorization"
	"scenario-authenticator/internal/localexchange"
	"scenario-authenticator/internal/ratelimit"
	"scenario-authenticator/internal/realm"
	"scenario-authenticator/internal/redisstate"
	"scenario-authenticator/internal/sessions"
)

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
	scenarioID, err := storage.ScenarioNamespace("scenario-authenticator")
	if err != nil {
		return "", fmt.Errorf("resolve scenario-authenticator storage namespace: %w", err)
	}
	path, err := resolver.Path(
		storage.Options{ScenarioID: scenarioID},
		storage.ClassData,
		"scenario-authenticator.db",
	)
	if err != nil {
		return "", fmt.Errorf("resolve scenario-authenticator db path: %w", err)
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

type breakGlassProvisioner struct {
	paths     trustposture.KeyPaths
	available bool
	ttl       time.Duration
}

func (p breakGlassProvisioner) Provision(_ context.Context, accountID, realmID string, scopes []string, linkedAt time.Time) error {
	if !p.available {
		return fmt.Errorf("break-glass is unavailable under the configured trust posture")
	}
	return trustposture.Provision(p.paths, accountID, realm.AudienceFor(realmID), scopes, linkedAt)
}

func (p breakGlassProvisioner) Issue(_ context.Context, accountID, realmID string, requested []string, now time.Time) (string, time.Time, error) {
	if !p.available || p.ttl <= 0 {
		return "", time.Time{}, fmt.Errorf("break-glass is unavailable under the configured trust posture")
	}
	token, err := trustposture.IssueFromProvision(p.paths, requested, now, p.ttl)
	if err != nil {
		return "", time.Time{}, err
	}
	return token, now.Add(p.ttl), nil
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "scenario-authenticator"}) {
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

	// --- Authentication stack ------------------------------------------------
	// The signing keypair persists under the storage seam (absolute path, fatal
	// on write failure — never silently regenerate, which would rotate the key
	// and break every relying party). Redis is REQUIRED hot state (sessions,
	// refresh-family revocation, blacklist are security controls); a failed
	// connection is boot-fatal, never a silent degrade.
	clk := clock.System{}
	keyDir, err := authcrypto.ResolveKeyDir()
	if err != nil {
		log.Fatalf("resolve signing key directory: %v", err)
	}
	keys, err := authcrypto.LoadOrGenerate(keyDir)
	if err != nil {
		log.Fatalf("load/generate signing key: %v", err)
	}
	posture, err := trustposture.LoadWorkingTree()
	if err != nil {
		log.Fatalf("load trust posture: %v", err)
	}
	defaults, err := trustposture.DefaultsFor(posture.Posture)
	if err != nil {
		log.Fatalf("resolve trust posture defaults: %v", err)
	}
	breakGlassPaths, err := trustposture.ResolveKeyPaths()
	if err != nil {
		log.Fatalf("resolve break-glass key paths: %v", err)
	}
	signer := authcrypto.NewSigner(keys, authcrypto.SignerConfig{
		Issuer: realm.Issuer,
		Expiry: authcrypto.ResolveExpiry(defaults.AccessTokenTTL),
	})

	redisStore, err := redisstate.NewRedisStore(context.Background())
	if err != nil {
		log.Fatalf("redis (required resource) unavailable: %v", err)
	}
	storageNamespace, err := storage.ResolveNamespace(storage.NamespaceConfig{FallbackScenario: "scenario-authenticator"})
	if err != nil {
		log.Fatalf("resolve Redis storage namespace: %v", err)
	}
	authRedisStore, err := redisstate.NewNamespacedStore(redisStore, storageNamespace, "auth")
	if err != nil {
		log.Fatalf("scope Redis storage namespace: %v", err)
	}
	sessionMgr := sessions.NewManager(authRedisStore, nil)
	repo := accounts.NewSQLiteRepository(db, clk)
	auditLogger := audit.NewSQLiteLogger(db, clk)
	authorizationService := authorization.NewService(repo.(authorization.ScopeStore), auditLogger)
	authService := accounts.NewService(accounts.ServiceConfig{
		Repo:             repo,
		Signer:           signer,
		Sessions:         sessionMgr,
		Audit:            auditLogger,
		Authorization:    authorizationService,
		MachineBindings:  repo.(accounts.MachineBindingStore),
		BreakGlass:       breakGlassProvisioner{paths: breakGlassPaths, available: defaults.BreakGlassAvailable, ttl: defaults.BreakGlassTTL},
		BreakGlassIssuer: breakGlassProvisioner{paths: breakGlassPaths, available: defaults.BreakGlassAvailable, ttl: defaults.BreakGlassTTL},
		Clock:            clk,
	})
	_, localHandler := accountsconnect.NewAccountsServiceHandler(authH.NewConnectHandler(authH.Deps{Service: authService, Logger: log.Default()}))
	exchangeLimiter := localexchange.NewRateLimiter(20, time.Minute)
	localSocketPath := strings.TrimSpace(os.Getenv("VROOLI_AUTH_SOCKET"))
	if localSocketPath == "" {
		socketName := "vrooli-scenario-authenticator"
		if namespace := strings.TrimSpace(os.Getenv("VROOLI_STORAGE_NAMESPACE")); namespace != "" {
			namespace = strings.Map(func(r rune) rune {
				if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
					return r
				}
				return '-'
			}, namespace)
			socketName += "-" + namespace
		}
		localSocketPath = filepath.Join(os.TempDir(), socketName+".sock")
	}
	localMux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != accountsconnect.AccountsServiceExchangeMachinePrincipalProcedure {
			http.NotFound(w, r)
			return
		}
		principal, ok := localexchange.PeerPrincipal(r.Context())
		if !ok {
			_ = auditLogger.Log(r.Context(), audit.Event{Action: "machine.exchange.refused", Success: false, Metadata: map[string]any{"reason": "peer_credential_unavailable"}})
			http.Error(w, "local peer credential unavailable", http.StatusUnauthorized)
			return
		}
		if !exchangeLimiter.Allow(principal.String(), time.Now().UTC()) {
			_ = auditLogger.Log(r.Context(), audit.Event{Action: "machine.exchange.refused", Success: false, Metadata: map[string]any{"reason": "rate_limited", "local_principal": principal.String()}})
			http.Error(w, "local exchange rate limited", http.StatusTooManyRequests)
			return
		}
		localHandler.ServeHTTP(w, r)
	})
	localStop, err := localexchange.Start(context.Background(), localSocketPath, localMux)
	if err != nil {
		log.Fatalf("start local identity exchange: %v", err)
	}

	srv := server.New(
		server.Deps{Clock: clock.System{}, Logger: log.Default()},
		healthH.Module(db, "scenario-authenticator-api", "1.0.0"),
		authH.Module(authService, log.Default()),
		sessionsH.Module(authService, log.Default()),
		jwksH.Module(keys),
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.Register(rootMux, db)

	rootMux.Handle("/", srv.Handler())

	// Backend-authoritative fixed-window rate limit on the brute-force surface
	// (login/register). Scoped by Connect service path so health/JWKS probes are
	// never throttled. Defense-in-depth on top of per-account lockout.
	limiter := ratelimit.New(authRedisStore, ratelimit.Config{
		Limit:  20,
		Window: time.Minute,
		PathPrefixes: []string{
			"/vrooli.scenario_authenticator.v1.accounts.AccountsService/Login",
			"/vrooli.scenario_authenticator.v1.accounts.AccountsService/Register",
		},
	}, nil)

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// request context so *database.RoutedDB routes the call to the
	// installed test pool. Self-disables in production mode.
	handler := apihttp.TestModeMiddleware(limiter.Middleware(rootMux))

	if err := apiserver.Run(apiserver.Config{
		Handler: handler,
		Cleanup: func(ctx context.Context) error {
			localStop()
			_ = redisStore.Close()
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
