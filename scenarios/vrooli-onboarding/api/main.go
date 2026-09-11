package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/nodereach"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	repocontract "github.com/vrooli/repo-contract-go"
	searchregister "github.com/vrooli/searchregister-go"
	"github.com/vrooli/vrooli/internal/operatorstate"
	applyH "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/handlers/apply"
	capabilitiesH "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/handlers/capabilities"
	credentialsH "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/handlers/credentials"
	glossaryH "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/handlers/glossary"
	healthH "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/handlers/health"
	hostH "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/handlers/host"
	operatorinputsH "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/handlers/operatorinputs"
	operatorstateH "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/handlers/operatorstate"
	profilesH "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/handlers/profiles"
	readinessH "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/handlers/readiness"
	resourcesH "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/handlers/resources"
	selectionH "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/handlers/selection"
	sessionH "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/handlers/session"
	applydomain "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/apply"
	capabilitiesdomain "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/capabilities"
	credentialsdomain "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/credentials"
	operatorstateapi "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/operatorstateapi"
	readinessdomain "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/readiness"
	resourcesdomain "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/resources"
	sessiondomain "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/session"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/targetproxy"
	_ "modernc.org/sqlite"
)

// Server wires the HTTP router.
type Server struct {
	router *mux.Router
	roots  Roots
	bridge *nodereach.Client
	auth   authn.Config
}

// routingMuxAdapter adapts gorilla/mux's fluent Handle signature to the small
// dev-routing interface without coupling api-core to this router package.
type routingMuxAdapter struct{ router *mux.Router }

func (m routingMuxAdapter) Handle(pattern string, handler http.Handler) {
	// Connect handlers own a procedure subtree. Gorilla's Handle is exact,
	// unlike net/http.ServeMux, so preserve devrouting's trailing-slash
	// subtree contract explicitly.
	m.router.PathPrefix(pattern).Handler(handler)
}

// NewServer initializes routes.
func NewServer() *Server {
	roots, _ := resolveRoots()
	srv := &Server{
		router: mux.NewRouter(),
		roots:  roots,
		bridge: nodereach.New(bridgeClientConfig()),
	}
	// NewServer is also the deterministic in-process test harness. Production
	// main replaces this with the environment-bound shared authentication config.
	// The harness uses an explicit in-memory session token so tests exercise the
	// same proof contract without reading the developer's OS identity.
	srv.auth = authn.Config{Providers: []authn.Provider{authn.NewPersonalLocalProviderWithSessionToken(
		"test-personal-local-session",
		"vrooli-onboarding:read", "vrooli-onboarding:write", "vrooli-onboarding:destructive",
	)}}
	srv.setupRoutes()
	return srv
}

func (s *Server) setupRoutes() {
	s.router.Use(securityHeadersMiddleware)
	s.router.Use(loggingMiddleware)
	// RESTException: ops_probe. Lifecycle probes and curl require these two
	// health endpoints without a generated client.
	s.router.HandleFunc("/health", healthH.Handler()).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthH.Handler()).Methods("GET")
	operatorinputsH.Module(controlPlaneExecutor{}, func(ctx context.Context) (string, error) {
		state, err := loadOperatorStateFor(ctx)
		if err != nil {
			return "", err
		}
		return operatorstate.Revision(state), nil
	}, targetproxy.Interceptor(s.bridge)).Mount(s.router)
	readinessH.Module(readinessdomain.Service{Evaluate: buildReadinessResponseForTarget, Acknowledge: acknowledgeDegradedReadiness}, targetproxy.Interceptor(s.bridge)).Mount(s.router)
	applyH.Module(applydomain.Service{Start: s.startApply, Cancel: s.cancelApply, Review: s.reviewApply, Get: s.getApplyRun, Plan: s.getApplyPlan}, targetproxy.Interceptor(s.bridge)).Mount(s.router)
	capabilitiesH.Module(capabilitiesdomain.Service{Executor: controlPlaneExecutor{}}, targetproxy.Interceptor(s.bridge)).Mount(s.router)
	credentialsH.Module(credentialsdomain.Service{ListFn: listCredentials, ProvisionFn: provisionCredential, DiagnoseFn: diagnoseCredentials}, targetproxy.Interceptor(s.bridge)).Mount(s.router)
	hostH.Module(hostService(s), targetproxy.Interceptor(s.bridge)).Mount(s.router)
	operatorstateH.Module(operatorstateapi.New(operatorStateService(), validateOperatorState), targetproxy.Interceptor(s.bridge)).Mount(s.router)
	profilesH.Module(onboardingProfilesService(), targetproxy.Interceptor(s.bridge)).Mount(s.router)
	resourcesH.Module(resourceService(), targetproxy.Interceptor(s.bridge)).Mount(s.router)
	glossaryH.Module().Mount(s.router)
	sessionH.Module(sessiondomain.Service{Get: s.getSession, Advance: s.advanceSession, Model: s.getStepModel, GetDraftFn: s.getDraft, SaveDraftFn: s.saveDraft, DiscardDraftFn: s.discardDraft, GetProfileSessionFn: s.getProfileSession, SaveProfileSessionFn: s.saveProfileSession}, targetproxy.Interceptor(s.bridge)).Mount(s.router)
	selectionH.Module(s.selectionService(), targetproxy.Interceptor(s.bridge)).Mount(s.router)
	s.router.Handle("/api/v2/handoff", selectionH.RESTHandoffHandler(s.selectionService())).Methods(http.MethodPost)
	mountOnboardingAuthHandler(s.router)
}

func resourceService() resourcesdomain.Service {
	return resourcesdomain.Service{
		Client: cliClient, Root: manifestRoot, LoadState: loadOperatorStateFor, Now: operatorStateNow,
		LoadScenarios: func(ctx context.Context) ([]resourcesdomain.Scenario, error) {
			models, err := loadScenarioReadModels()
			if err != nil {
				return nil, err
			}
			result := make([]resourcesdomain.Scenario, 0, len(models))
			for _, model := range models {
				result = append(result, resourcesdomain.Scenario{Name: model.Name, Enabled: model.Enabled})
			}
			return result, nil
		},
		ResolveClosure: func(root string, models []resourcesdomain.Scenario, state operatorstate.Document) ([]resourcesdomain.ClosureMember, error) {
			legacy := make([]ScenarioReadModel, 0, len(models))
			for _, model := range models {
				legacy = append(legacy, ScenarioReadModel{Name: model.Name, Enabled: model.Enabled})
			}
			closure, err := resolveClosureForState(root, legacy, state)
			if err != nil {
				return nil, err
			}
			result := make([]resourcesdomain.ClosureMember, 0, len(closure.Resources))
			for _, member := range closure.Resources {
				result = append(result, resourcesdomain.ClosureMember{Name: member.Name, Required: member.Required})
			}
			return result, nil
		},
	}
}

// securityHeadersMiddleware applies the baseline browser-facing API policy at
// one boundary so every handler, including errors and health probes, is safe.
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "0")
		next.ServeHTTP(w, r)
	})
}

// Handler returns the HTTP handler with recovery middleware
func (s *Server) Handler() http.Handler {
	return handlers.RecoveryHandler()(authn.Middleware(s.auth)(s.router))
}

func onboardingAuthenticationConfig() authn.Config {
	shared, err := authn.FromEnvironment(os.Getenv)
	if err != nil {
		// An invalid optional provider declaration must not silently create an
		// unauthenticated mutation path. The shared middleware remains disabled;
		// mutation handlers fail closed without a principal.
		return authn.Config{}
	}
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("VROOLI_AUTH_MODE")))
	// Shared provider configuration is the production default. The loopback
	// provider is an explicit bundled-desktop opt-in, never an implicit fallback
	// that turns an unconfigured API into a human-authorized endpoint.
	if mode == "personal_local" {
		local := newOnboardingPersonalLocalProvider()
		return authn.Config{Providers: []authn.Provider{local}, RecoveryURL: shared.RecoveryURL}
	}
	return authn.Config{Providers: onboardingAuthProviders(shared.Providers), RecoveryURL: shared.RecoveryURL}
}

// onboardingScenarioName is this API's own scenario. The apply executor needs
// it to recognise itself in a plan: starting this scenario from inside a
// request it is serving stops the process mid-run.
const onboardingScenarioName = "vrooli-onboarding"

// loggingMiddleware prints simple request logs
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "vrooli-onboarding",
	}) {
		return // Process was re-exec'd after rebuild
	}

	if err := configureOperatorStateRoots(); err != nil {
		panic("configure operator state roots: " + err.Error())
	}

	// Runner mode executes one accepted apply run and exits. It is deliberately
	// ahead of every server concern below: a runner opens no listener, claims
	// no port, and registers no routes, so an apply in flight never competes
	// with the API it was spawned from.
	if id, ok := applyRunnerRequest(os.Args[1:]); ok {
		if err := runApplyRunner(context.Background(), id); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "apply runner: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if err := prepareApplyRunnerExecutable(); err != nil {
		panic("prepare apply runner executable: " + err.Error())
	}
	// Keep the file-only API inside api-core's routed storage contract. The
	// operatorstate service performs the actual request-scoped selection; this
	// startup probe makes the seam explicit to storage-manager's static checker.
	ctx := context.Background()
	if _, err := operatorStateRoots.Pick(ctx, storage.ClassConfig); err != nil {
		panic("route operator state roots: " + err.Error())
	}
	if _, err := operatorStateService().PruneDrafts(ctx, operatorstate.DefaultDraftRetention); err != nil {
		slog.Default().Warn("onboarding draft retention cleanup unavailable", "error", err)
	}
	if err := cleanupOnboardingArtifacts(); err != nil {
		slog.Default().Warn("onboarding apply retention cleanup unavailable", "error", err)
	}
	// The onboarding API is file-authoritative, but test-genie still needs the
	// standard routed control surface to prove destructive workflows cannot use
	// a live pool. This private in-memory pool stores no onboarding state.
	routingDB, err := database.Open(ctx, database.Config{
		Driver:       database.DriverSQLite,
		DSN:          "file:vrooli-onboarding-routing?mode=memory&cache=shared",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		panic("configure onboarding test routing: " + err.Error())
	}
	if err := database.EnsureSchemas(ctx, routingDB.Primary()); err != nil {
		panic("configure onboarding test schemas: " + err.Error())
	}
	srv := NewServer()
	srv.auth = onboardingAuthenticationConfig()
	devrouting.RegisterWithFileRoots(routingMuxAdapter{router: srv.router}, routingDB, operatorStateRoots)
	handler := apihttp.TestModeMiddleware(srv.Handler())
	if repoRoot, err := repocontract.ResolveRepoRoot(); err == nil {
		go searchregister.Register(ctx, searchregister.Config{
			ScenarioID:     "vrooli-onboarding",
			SearchFilePath: filepath.Join(repoRoot, "scenarios", "vrooli-onboarding", ".vrooli", "search.json"),
			Logger:         log.Default(),
		})
	} else {
		slog.Default().Warn("configuration search registration skipped", "scenario", "vrooli-onboarding", "error", err)
	}

	// Start server with graceful shutdown (port from API_PORT env var)
	if err := server.Run(server.Config{
		Handler:        handler,
		Authentication: nil,
		Cleanup:        func(context.Context) error { return routingDB.Close() },
	}); err != nil {
		panic("onboarding server failed: " + err.Error())
	}
}
