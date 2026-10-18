package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"scenario-authenticator/internal/modules"
	"scenario-authenticator/internal/server"

	"github.com/vrooli/api-core/schedule"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	coreidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	authH "scenario-authenticator/handlers/auth"
	healthH "scenario-authenticator/handlers/health"
	jwksH "scenario-authenticator/handlers/jwks"
	mfaH "scenario-authenticator/handlers/mfa"
	sessionsH "scenario-authenticator/handlers/sessions"
	"scenario-authenticator/internal/accounts"
	"scenario-authenticator/internal/audit"
	"scenario-authenticator/internal/authcrypto"
	"scenario-authenticator/internal/authorization"
	"scenario-authenticator/internal/localexchange"
	mfainternal "scenario-authenticator/internal/mfa"
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

// cloudflareHumanAssertion adapts the shared managed-Cloudflare provider to the
// accounts.HumanAssertionVerifier seam. Its authn.Config contains only this
// provider plus the shared owner bindings selected for agent-manager, so the
// browser assertion cannot acquire identity from other providers or ambient
// request context.
type cloudflareHumanAssertion struct {
	config authn.Config
}

func (c cloudflareHumanAssertion) VerifyHuman(ctx context.Context, rawAssertion string) (coreidentity.Principal, bool) {
	if strings.TrimSpace(rawAssertion) == "" {
		return coreidentity.Principal{}, false
	}
	// The managed provider verifies from a request header, so present the
	// assertion the same way Cloudflare delivers it. The request is synthetic and
	// carries only the assertion; no ambient identity leaks in.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://local/mint", nil)
	if err != nil {
		return coreidentity.Principal{}, false
	}
	req.Header.Set("Cf-Access-Jwt-Assertion", strings.TrimSpace(rawAssertion))
	principal, err := c.config.Authenticate(ctx, req)
	if err != nil || !principal.IsHuman() || !principal.IsVerified() {
		return coreidentity.Principal{}, false
	}
	return principal, true
}

// cloudflareOwnerAuthConfig reuses api-core's OwnerBinding parser and matcher
// while selecting bindings for the relying scenario (agent-manager), not this
// issuer process. Provider loading is disabled during parsing; the returned
// verifier config contains only the existing managed Web Console Cloudflare
// provider, preventing unrelated environment providers from joining identity.
func cloudflareOwnerAuthConfig(provider authn.Provider, getenv func(string) string) (authn.Config, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	bindingsConfig, err := authn.FromEnvironment(func(key string) string {
		switch key {
		case "VROOLI_AUTH_PROVIDERS":
			return ""
		case "VROOLI_SCENARIO", "SCENARIO_NAME":
			return "agent-manager"
		default:
			return getenv(key)
		}
	})
	if err != nil {
		return authn.Config{}, err
	}
	if provider == nil || provider.Source() != coreidentity.SourceCloudflareAccess {
		return authn.Config{}, fmt.Errorf("Cloudflare owner verifier requires the managed Cloudflare provider")
	}
	return authn.Config{Providers: []authn.Provider{provider}, OwnerBindings: bindingsConfig.OwnerBindings}, nil
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
	credentialAuthority, authorityErr := credentialauthority.Default()
	if authorityErr != nil {
		log.Printf("credential authority unavailable for MFA seed custody: %v", authorityErr)
		credentialAuthority, _ = credentialauthority.Unavailable(authorityErr.Error())
	}
	seedCustodian, err := newAuthoritySeedCustodian(credentialAuthority)
	if err != nil {
		log.Fatalf("initialize MFA seed custodian: %v", err)
	}
	mfaStore, err := mfainternal.NewStoreWithCustodian(db, seedCustodian, clk)
	if err != nil {
		log.Fatalf("initialize MFA store: %v", err)
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
	servicePrincipals, err := accounts.ParseServicePrincipalProfiles(os.Getenv("VROOLI_AUTH_SERVICE_PRINCIPALS"))
	if err != nil {
		log.Fatalf("service principal configuration: %v", err)
	}

	// Cf-Access binding reachability self-check. Human operators authenticate a
	// session-delegation mint through the web-console Cloudflare Access binding,
	// resolved from tunnel-manager. If that binding is misconfigured or
	// tunnel-manager is unreachable, the failure would otherwise surface only on
	// the first real operator login (a silent fleet human-login outage). We probe
	// it once at startup and surface it loudly. This is deliberately best-effort,
	// not boot-fatal: resolution depends on a live tunnel-manager, and a healthy
	// authenticator must still boot (to serve native logins/JWKS) when Cloudflare
	// is simply not configured in this deployment. The check primes the provider's
	// cache on success so the first operator login does not pay the resolution.
	cloudflareHumanProvider := authn.NewManagedCloudflareProvider("web-console")
	checkCtx, cancelCheck := context.WithTimeout(context.Background(), 10*time.Second)
	if bindingErr := cloudflareHumanProvider.CheckBindingReachable(checkCtx); bindingErr != nil {
		log.Printf("WARNING: cloudflare access binding self-check failed; operator human login via the tunnel may be unavailable until the tunnel-manager binding is reachable: %v", bindingErr)
	} else {
		log.Printf("cloudflare access binding self-check passed; web-console human-login binding is reachable")
	}
	cancelCheck()
	cloudflareOwnerConfig, err := cloudflareOwnerAuthConfig(cloudflareHumanProvider, os.Getenv)
	if err != nil {
		log.Fatalf("cloudflare owner binding configuration is invalid: %v", err)
	}

	authService := accounts.NewService(accounts.ServiceConfig{
		Repo:              repo,
		Signer:            signer,
		Sessions:          sessionMgr,
		Audit:             auditLogger,
		Authorization:     authorizationService,
		MachineBindings:   repo.(accounts.MachineBindingStore),
		BreakGlass:        breakGlassProvisioner{paths: breakGlassPaths, available: defaults.BreakGlassAvailable, ttl: defaults.BreakGlassTTL, target: breakGlassTarget},
		BreakGlassIssuer:  breakGlassProvisioner{paths: breakGlassPaths, available: defaults.BreakGlassAvailable, ttl: defaults.BreakGlassTTL, target: breakGlassTarget},
		MFA:               mfaStore,
		Clock:             clk,
		ResourceAudiences: parseResourceAudiences(os.Getenv("VROOLI_AUTH_RESOURCE_AUDIENCES")),
		ServicePrincipals: servicePrincipals,
		ServiceCustody:    authorityServiceCustodian{authority: credentialAuthority},
		// A tunnel operator authenticates via Cloudflare Access, not an account
		// bearer. Verify through the existing Web Console binding and shared
		// agent-manager OwnerBindings; an absent binding supplies no scopes.
		HumanAssertion: cloudflareHumanAssertion{config: cloudflareOwnerConfig},
	})
	_, localHandler := accountsconnect.NewAccountsServiceHandler(authH.NewConnectHandler(authH.Deps{Service: authService, Logger: log.Default()}))
	exchangeLimiter := localexchange.NewRateLimiter(20, time.Minute)
	localSocketPath := strings.TrimSpace(os.Getenv("VROOLI_AUTH_SOCKET"))
	if localSocketPath == "" {
		localSocketPath = authn.LocalAuthenticatorSocket(os.Getenv("VROOLI_STORAGE_NAMESPACE"))
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
		mfaH.Module(authService, mfaStore, auditLogger, log.Default()),
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
			"/vrooli.scenario_authenticator.v1.accounts.AccountsService/ExchangeServiceCredential",
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

// parseResourceAudiences loads the operator-owned resource registry. Each
// comma-separated entry is resource-id=audience. Empty or malformed entries
// are ignored so an unset variable preserves the default-realm contract.
func parseResourceAudiences(raw string) map[string]string {
	resources := make(map[string]string)
	for _, entry := range strings.Split(raw, ",") {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			continue
		}
		resource := strings.TrimSpace(parts[0])
		audience := strings.TrimSpace(parts[1])
		if resource != "" && audience != "" && !strings.ContainsAny(resource+audience, "|\n\r") {
			resources[resource] = audience
		}
	}
	return resources
}
