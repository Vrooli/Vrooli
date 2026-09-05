package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"

	"command-center/internal/trends"
	"command-center/upstream"
)

// Server is the command-center HTTP aggregator.
type Server struct {
	router              *mux.Router
	registry            *Registry
	cache               *Cache
	stats               *StatsBuffer
	integrationRegistry *capabilityregistry.Registry
	actionService       capabilityregistry.LifecycleActionService
	instrumentProvider  InstrumentProvider
	objectiveProvider   ObjectiveProvider
	promptManager       *promptManagerInstrumentProvider
	trendStore          trends.Store

	swarm     upstream.Client
	vrooli    upstream.Client
	lpbs      upstream.Client
	offer     upstream.Client
	deploy    upstream.Client
	providers map[UpstreamSource]upstreamProvider
}

type upstreamProvider struct {
	client      func() upstream.Client
	defaultPath string
}

// NewServer wires the router, cache, upstream clients, and registry.
func NewServer(reg *Registry) *Server {
	return NewServerWithTrendStore(reg, trends.NewMemoryStore())
}

// NewServerWithTrendStore is the production/test seam for durable metric history.
func NewServerWithTrendStore(reg *Registry, trendStore trends.Store) *Server {
	s := &Server{
		router:     mux.NewRouter(),
		registry:   reg,
		cache:      NewCache(),
		stats:      NewStatsBuffer(1024, time.Hour),
		trendStore: trendStore,

		swarm:  upstream.NewSwarmTypedResolved(resolveScenarioBaseURL("swarm-manager", "SWARM_MANAGER_BASE_URL", "SWARM_MANAGER_API_PORT"), declaredFeatureSet(reg, "swarm-manager", "")),
		vrooli: upstream.NewVrooliTypedResolved(resolveControlPlaneBaseURL, declaredFeatureSet(reg, "vrooli-core", "")),
		lpbs:   upstream.NewLPBSTypedResolved(resolveScenarioBaseURL("landing-page-business-suite", "LPBS_BASE_URL", "LPBS_API_PORT"), resolveLPBSServiceToken(), declaredFeatureSet(reg, "landing-page-business-suite", "lpbs")),
		offer:  upstream.NewJSONConnectResolved("offer-desk", resolveScenarioBaseURL("offer-desk", "OFFER_DESK_BASE_URL", "OFFER_DESK_API_PORT"), "/vrooli.offer_desk.v1.offers.ReleaseLadderService/GetReleaseLadder", declaredFeatureSet(reg, "offer-desk", "")),
		deploy: upstream.NewRESTResolved("deployment-manager", resolveScenarioBaseURL("deployment-manager", "DEPLOYMENT_MANAGER_BASE_URL", "DEPLOYMENT_MANAGER_API_PORT"), ""),
	}
	s.providers = map[UpstreamSource]upstreamProvider{
		SourceSwarm:  {client: func() upstream.Client { return s.swarm }, defaultPath: "/api/v1/stats"},
		SourceVrooli: {client: func() upstream.Client { return s.vrooli }, defaultPath: "/scenarios"},
		SourceLPBS:   {client: func() upstream.Client { return s.lpbs }, defaultPath: "/api/v1/admin/dashboard/summary"},
		SourceOffer:  {client: func() upstream.Client { return s.offer }, defaultPath: "/vrooli.offer_desk.v1.offers.ReleaseLadderService/GetReleaseLadder"},
		SourceDeploy: {client: func() upstream.Client { return s.deploy }, defaultPath: "/api/v1/readiness/state"},
	}
	transmitter := newPromptManagerInstrumentProvider(resolveScenarioBaseURL("prompt-manager", "PROMPT_MANAGER_BASE_URL", "PROMPT_MANAGER_API_PORT"))
	s.instrumentProvider = transmitter
	s.objectiveProvider = transmitter
	s.promptManager = transmitter
	s.integrationRegistry = commandCenterIntegrationRegistry(s)
	if len(reg.Metrics) > 0 {
		if err := validateOutcomeBindings(s.integrationRegistry.Definitions(), reg.Metrics); err != nil {
			panic("command-center outcome binding validation failed: " + err.Error())
		}
	}
	s.actionService = capabilityregistry.LifecycleActionService{Defs: s.integrationRegistry.Definitions(), CLIPath: os.Getenv("VROOLI_CLI_PATH"), Timeout: 180 * time.Second, Tracker: capabilityregistry.NewActionTracker()}
	s.setupRoutes()
	return s
}

func declaredFeatureSet(reg *Registry, integrationID, kind string) map[string]string {
	features := map[string]string{}
	for _, metric := range reg.Metrics {
		if metric.Source.IntegrationID != integrationID || metric.Source.FeatureID == "" {
			continue
		}
		featureKind := ""
		if kind == "lpbs" {
			if strings.Contains(metric.Source.Read, "/revenue") {
				featureKind = "revenue"
			} else {
				featureKind = "analytics"
			}
		}
		features[metric.Source.FeatureID] = featureKind
	}
	return features
}

func resolveLPBSServiceToken() string {
	authority, err := credentialauthority.Default()
	if err != nil {
		return ""
	}
	// LPBS mints and owns this credential. Consume the same authority entry
	// rather than declaring a second token that can drift from its verifier.
	token, err := authority.Require(credentialauthority.Identity("vrooli/landing-page-business-suite"), "service-secret")
	if err != nil {
		return ""
	}
	return token
}

// Handler returns the HTTP handler wrapped with recovery middleware.
func (s *Server) Handler() http.Handler {
	return handlers.RecoveryHandler()(s.router)
}

// Shutdown releases any resources held by the server.
func (s *Server) Shutdown(_ context.Context) error {
	if s.trendStore != nil {
		return s.trendStore.Close()
	}
	return nil
}

// loggingMiddleware emits a compact structured log line per request.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("request",
			"method", r.Method,
			"uri", r.RequestURI,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

func resolveControlPlaneBaseURL() string {
	if v := os.Getenv("VROOLI_API_BASE_URL"); v != "" {
		return v
	}
	if v := os.Getenv("VROOLI_API_PORT"); v != "" {
		return "http://127.0.0.1:" + v
	}
	return "http://127.0.0.1:8092"
}
