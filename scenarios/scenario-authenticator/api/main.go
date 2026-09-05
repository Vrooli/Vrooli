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

	"scenario-authenticator/internal/modules"
	"scenario-authenticator/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

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

	"github.com/vrooli/api-core/trustposture"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
)

type breakGlassProvisioner struct {
	paths     trustposture.KeyPaths
	available bool
	ttl       time.Duration
	target    string
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
	token, err := trustposture.IssueFromProvisionForTarget(p.paths, p.target, requested, now, p.ttl)
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

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "scenario-authenticator",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	schemas := append(modules.AllSchemas(), database.SchemaProviderFunc(redisstate.Schema))
	if err := database.EnsureSchemas(context.Background(), db.Primary(), schemas...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}

	// --- Authentication stack ------------------------------------------------
	// The signing keypair persists under the storage seam (absolute path, fatal
	// on write failure — never silently regenerate, which would rotate the key
	// and break every relying party). Hot state (sessions, refresh-family
	// revocation, blacklist, rate-limit counters) is a set of security controls
	// rather than a cache, so its store is selected explicitly below.
	clk := schedule.System()
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
	breakGlassPaths, err := trustposture.ResolveAuthenticatorKeyPaths()
	if err != nil {
		log.Fatalf("resolve break-glass key paths: %v", err)
	}
	breakGlassTarget, err := os.Hostname()
	if err != nil || strings.TrimSpace(breakGlassTarget) == "" {
		log.Fatalf("resolve break-glass target: %v", err)
	}
	signer := authcrypto.NewSigner(keys, authcrypto.SignerConfig{
		Issuer: realm.Issuer,
		Expiry: authcrypto.ResolveExpiry(defaults.AccessTokenTTL),
	})

	// Selection is explicit, never a fallback on connection failure. Redis
	// configured but unreachable stays boot-fatal: degrading to a store that
	// shares nothing across replicas would let one replica keep honouring a
	// token another replica revoked. With no Redis configured, this is a
	// single-node deployment and the durable local store is the right answer —
	// it keeps the blacklist across restart, which the in-memory fake cannot.
	var hotState redisstate.Store
	closeHotState := func() error { return nil }
	if redisstate.RedisConfigured() {
		redisStore, redisErr := redisstate.NewRedisStore(context.Background())
		if redisErr != nil {
			log.Fatalf("redis is configured but unavailable: %v", redisErr)
		}
		hotState = redisStore
		closeHotState = redisStore.Close
	} else {
		durable, durableErr := redisstate.NewSQLiteStore(db)
		if durableErr != nil {
			log.Fatalf("durable hot-state store unavailable: %v", durableErr)
		}
		sweepCtx, cancelSweep := context.WithCancel(context.Background())
		go durable.RunSweeper(sweepCtx, time.Hour, func(err error) {
			log.Printf("hot-state sweep failed: %v", err)
		})
		hotState = durable
		closeHotState = func() error { cancelSweep(); return nil }
	}
	storageNamespace, err := storage.ResolveNamespace(storage.NamespaceConfig{FallbackScenario: "scenario-authenticator"})
	if err != nil {
		log.Fatalf("resolve hot-state storage namespace: %v", err)
	}
	authRedisStore, err := redisstate.NewNamespacedStore(hotState, storageNamespace, "auth")
	if err != nil {
		log.Fatalf("scope hot-state storage namespace: %v", err)
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
		BreakGlass:       breakGlassProvisioner{paths: breakGlassPaths, available: defaults.BreakGlassAvailable, ttl: defaults.BreakGlassTTL, target: breakGlassTarget},
		BreakGlassIssuer: breakGlassProvisioner{paths: breakGlassPaths, available: defaults.BreakGlassAvailable, ttl: defaults.BreakGlassTTL, target: breakGlassTarget},
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
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
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
			_ = closeHotState()
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
