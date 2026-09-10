package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"deployment-manager/build"
	"deployment-manager/bundles"
	"deployment-manager/codesigning"
	"deployment-manager/dependencies"
	"deployment-manager/deployments"
	"deployment-manager/fitness"
	evidencehandler "deployment-manager/handlers/evidence"
	profileshandler "deployment-manager/handlers/profiles"
	readinesshandler "deployment-manager/handlers/readiness"
	"deployment-manager/health"
	"deployment-manager/internal/authz"
	internalEvidence "deployment-manager/internal/evidence"
	"deployment-manager/internal/modules"
	internalReadiness "deployment-manager/internal/readiness"
	internalReleases "deployment-manager/internal/releases"
	transport "deployment-manager/internal/transport"
	"deployment-manager/migrationtasks"
	"deployment-manager/profiles"
	"deployment-manager/readiness"
	"deployment-manager/releases"
	"deployment-manager/secrets"
	"deployment-manager/swaps"
	"deployment-manager/telemetry"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	evidenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/evidence/evidencev1connect"
	profilesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/profiles/profilesv1connect"
	readinessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/readiness/readinessv1connect"
	_ "modernc.org/sqlite"
)

// Server wires the HTTP router and database connection.
type Server struct {
	Config   *RuntimeConfig
	DB       interface{ Close() error }
	RoutedDB *database.RoutedDB
	Router   *mux.Router
	Logger   Logger

	// Domain handlers
	HealthHandler            *health.Handler
	FitnessHandler           *fitness.Handler
	TelemetryHandler         *telemetry.Handler
	SecretsHandler           *secrets.Handler
	DependenciesHandler      *dependencies.Handler
	SwapsHandler             *swaps.Handler
	DeploymentsHandler       *deployments.Handler
	BundlesHandler           *bundles.Handler
	ProfilesHandler          *profiles.Handler
	SigningHandler           *codesigning.Handler
	BuildHandler             *build.Handler
	ApprovalsHandler         *deployments.ApprovalsHandler
	PublishedVersionsHandler *deployments.PublishedVersionsHandler
	LPBSConfigHandler        *profiles.LPBSConfigHandler
	ReleasesHandler          *releases.Handler
	MigrationTasksHandler    *migrationtasks.Handler
	EvidencePath             string
	EvidenceHandler          http.Handler
	ProfilesConnectPath      string
	ProfilesConnectHandler   http.Handler
	ConnectRoutes            []transport.Route
	Orchestrator             *deployments.Orchestrator
	Handler                  http.Handler

	// Repositories
	ProfilesRepo      profiles.Repository
	SigningRepo       codesigning.Repository // Interface to allow SQL or Proxy implementation
	LPBSConfigRepo    profiles.LPBSReleaseConfigRepository
	ReleasesRepo      releases.Repository
	ReadinessRepo     readiness.ReviewRepository
	ReadinessPreparer *readiness.Preparer
}

type candidateProvenanceResolver struct {
	repository releases.IdentityRepository
}

type cloudObservabilityResolver struct {
	repository releases.IdentityRepository
	client     deployments.CloudHealthClient
}

func (r cloudObservabilityResolver) CheckObservability(ctx context.Context, identity readiness.ReviewIdentity) (readiness.ObservabilityReadiness, error) {
	if r.repository == nil {
		return readiness.ObservabilityReadiness{}, fmt.Errorf("release identity repository is not configured")
	}
	if r.client == nil {
		return readiness.ObservabilityReadiness{}, fmt.Errorf("cloud health owner is not configured")
	}
	if strings.TrimSpace(identity.DestinationRevisionID) == "" {
		return readiness.ObservabilityReadiness{}, fmt.Errorf("destination revision identity is required")
	}
	destination, err := r.repository.GetDestinationRevision(ctx, identity.DestinationRevisionID)
	if err != nil {
		return readiness.ObservabilityReadiness{}, err
	}
	if destination == nil || destination.ID != identity.DestinationRevisionID {
		return readiness.ObservabilityReadiness{}, fmt.Errorf("destination revision %q is unavailable", identity.DestinationRevisionID)
	}
	if !strings.EqualFold(destination.Revision.Kind, "cloud") {
		return readiness.ObservabilityReadiness{}, fmt.Errorf("observability owner does not support destination kind %q", destination.Revision.Kind)
	}
	deploymentID := strings.TrimSpace(destination.Revision.DestinationID)
	if deploymentID == "" {
		return readiness.ObservabilityReadiness{}, fmt.Errorf("cloud destination has no deployment identity")
	}
	result, err := r.client.CheckDeploymentHealth(ctx, deployments.DeploymentHealthRequest{DeploymentID: deploymentID})
	if err != nil {
		return readiness.ObservabilityReadiness{}, err
	}
	if result == nil || result.ObservedAt.IsZero() || strings.TrimSpace(result.DeploymentID) != deploymentID {
		return readiness.ObservabilityReadiness{}, fmt.Errorf("cloud health owner returned incomplete deployment observation")
	}
	return readiness.ObservabilityReadiness{
		Healthy: result.Healthy, Target: "cloud:" + deploymentID,
		Reference:  "scenario-to-cloud:health:" + deploymentID + ":" + result.ObservedAt.UTC().Format(time.RFC3339Nano),
		ObservedAt: result.ObservedAt, Detail: result.Explain(),
	}, nil
}

func (r candidateProvenanceResolver) ResolveCandidate(ctx context.Context, id string) (readiness.CandidateProvenance, error) {
	if r.repository == nil {
		return readiness.CandidateProvenance{}, fmt.Errorf("candidate identity repository is not configured")
	}
	record, err := r.repository.GetCandidate(ctx, id)
	if err != nil {
		return readiness.CandidateProvenance{}, err
	}
	if record == nil {
		return readiness.CandidateProvenance{}, fmt.Errorf("candidate %q was not found", id)
	}
	signed := 0
	targetIDs := make([]string, 0, len(record.Candidate.Artifacts))
	platforms := make([]string, 0, len(record.Candidate.Artifacts))
	refsComplete := true
	for _, artifact := range record.Candidate.Artifacts {
		targetIDs = append(targetIDs, artifact.Target.ID)
		platforms = append(platforms, artifact.Target.Platform)
		if strings.TrimSpace(artifact.ImmutableRef) == "" || strings.TrimSpace(artifact.Digest) == "" || artifact.SizeBytes < 0 {
			refsComplete = false
		}
		if strings.TrimSpace(artifact.SignatureDigest) != "" && strings.TrimSpace(artifact.SignerRef) != "" {
			signed++
		}
	}
	ownership := readiness.OperationsOwnership{}
	if declaration := record.Candidate.CapabilityDeclaration; declaration != nil {
		ownership = readiness.OperationsOwnership{
			SupportOwner: declaration.SupportOwner, IncidentOwner: declaration.IncidentOwner,
			CustomerContact: declaration.CustomerContact, ReleaseAuthority: declaration.ReleaseAuthority,
			RollbackAuthority: declaration.RollbackAuthority, DegradedModeAuthority: declaration.DegradedModeAuthority,
		}
	}
	return readiness.CandidateProvenance{
		ID: record.ID, SourceRevision: record.Candidate.SourceRevision,
		ArtifactManifestDigest: record.ArtifactManifestDigest,
		DependencyLockDigest:   record.Candidate.DependencyLockDigest,
		PolicyDigest:           record.Candidate.PolicyDigest,
		BuildInputsPresent:     len(record.Candidate.BuildInputs) > 0,
		ArtifactCount:          len(record.Candidate.Artifacts), SignedArtifactCount: signed,
		ArtifactTargetIDs: targetIDs, ArtifactPlatforms: platforms, ArtifactRefsComplete: refsComplete,
		CapabilityDeclaration: ownership,
	}, nil
}

type rampEvidenceResolver struct {
	repository internalEvidence.Repository
}

func (r rampEvidenceResolver) ListRampEvidence(ctx context.Context, profileID, commit string) ([]readiness.RampEvidence, error) {
	if r.repository == nil {
		return nil, fmt.Errorf("evidence repository is not configured")
	}
	verdicts, err := r.repository.List(ctx, profileID, commit, 1000)
	if err != nil {
		return nil, err
	}
	rows := make([]readiness.RampEvidence, 0, len(verdicts))
	for _, verdict := range verdicts {
		if verdict == nil || verdict.Target == nil {
			continue
		}
		disposition := "unknown"
		if verdict.Disposition == commonv1.Disposition_DISPOSITION_PASSED {
			disposition = "passed"
		} else if verdict.Disposition == commonv1.Disposition_DISPOSITION_FAILED {
			disposition = "failed"
		}
		rows = append(rows, readiness.RampEvidence{
			Target: verdict.Target.Platform, Platform: verdict.Target.Platform, OS: verdict.Target.Os,
			Ramp: verdict.Target.Ramp, Disposition: disposition, RunID: verdict.RunId,
			Reference: "run:" + verdict.RunId,
		})
	}
	return rows, nil
}

type recoveryReadinessResolver struct {
	executor                releases.RecoveryExecutor
	authorizationConfigured bool
}

func (r recoveryReadinessResolver) CheckRecovery(_ context.Context, identity readiness.ReviewIdentity) (readiness.RecoveryReadiness, error) {
	return readiness.RecoveryReadiness{
		ExecutorConfigured:      r.executor != nil,
		AuthorizationConfigured: r.authorizationConfigured,
		TargetIdentityBound:     strings.TrimSpace(identity.CandidateID) != "" && strings.TrimSpace(identity.DestinationRevisionID) != "" && identity.AuthorizationEpoch > 0,
	}, nil
}

// New initializes configuration, database, and routes.
func New() (*Server, error) {
	cfg := &RuntimeConfig{
		Port: RequireEnv("API_PORT"),
	}
	authConfig, err := authn.FromEnvironment(os.Getenv)
	if err != nil {
		return nil, fmt.Errorf("configure authentication: %w", err)
	}
	fileRoots, err := newFileRoots()
	if err != nil {
		return nil, fmt.Errorf("failed to configure file roots: %w", err)
	}
	// Connect to this scenario's own database. The path derives from the
	// scenario slug rather than from the environment, and the seam creates the
	// parent directory, so no pre-flight mkdir is needed here.
	routedDB, err := database.Open(context.Background(), database.Config{
		Driver:   database.DriverSQLite,
		Scenario: "deployment-manager",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Apply every domain schema once at boot. Schema failure is fatal: serving
	// against a partially initialized database would fabricate release state.
	if err := database.EnsureSchemas(context.Background(), routedDB.Primary(), modules.AllSchemas()...); err != nil {
		_ = routedDB.Close()
		return nil, fmt.Errorf("failed to ensure database schemas: %w", err)
	}
	if err := internalReleases.MigrateLegacyReleaseConstraints(context.Background(), routedDB); err != nil {
		_ = routedDB.Close()
		return nil, fmt.Errorf("failed to migrate release identity constraints: %w", err)
	}
	if err := database.ApplySchemas(context.Background(), routedDB.Primary(), database.SchemaProviderFunc(internalReleases.IndexesSchema)); err != nil {
		_ = routedDB.Close()
		return nil, fmt.Errorf("failed to ensure release indexes: %w", err)
	}

	// Create repositories through the RoutedDB seam.
	profilesRepo := profiles.NewSQLRepository(routedDB)

	// Signing is owned by scenario-to-desktop. deployment-manager retains only
	// the proxy repository seam and never persists signing material locally.
	signingRepo := codesigning.NewProxyRepository(profilesRepo)

	// Create domain handlers
	logFn := func(msg string, fields map[string]interface{}) {
		LogStructured(msg, fields)
	}

	// Create approvals repository and ensure schema
	approvalsRepo := deployments.NewSQLApprovalsRepository(routedDB)
	approvalsHandler := deployments.NewApprovalsHandler(approvalsRepo, logFn).WithActorResolver(func(ctx context.Context) (string, error) {
		if err := authz.RequireWrite(ctx); err != nil {
			return "", err
		}
		principal, err := authn.RequireHuman(ctx)
		if err != nil {
			return "", err
		}
		return principal.Subject, nil
	})

	// Create published versions repository and ensure schema
	publishedVersionsRepo := deployments.NewSQLPublishedVersionsRepository(routedDB)

	// LPBS release-config repository (1:1 child of profiles).
	lpbsConfigRepo := profiles.NewSQLLPBSReleaseConfigRepository(routedDB)

	// Releases repository (canonical release records + per-platform rows).
	releasesRepo := releases.NewSQLRepository(routedDB)

	// Evidence is a reference ledger owned by deployment-manager. Producers
	// retain their bytes; this service stores only target verdicts and references.
	evidenceRepo := internalEvidence.NewSQLRepository(routedDB, "sqlite")
	readinessRepo := internalReadiness.NewSQLRepository(routedDB, "sqlite")
	approvalsRepo.WithEvidenceRepository(evidenceRepo)
	evidenceDomainHandler := evidencehandler.NewConnectHandler(evidenceRepo).WithAuthorization(func(ctx context.Context) error {
		return authz.RequireReadinessEvidence(ctx)
	}).WithTargetAuthorization(func(ctx context.Context, target *commonv1.EvidenceTarget) error {
		return authz.RequireEvidenceForRamp(ctx, target.GetRamp())
	}).WithReadAuthorization(func(ctx context.Context) error {
		return authz.RequireRead(ctx)
	})
	evidencePath, evidenceHandler := evidenceconnect.NewEvidenceServiceHandler(evidenceDomainHandler)
	profilesConnectHandlerDomain := profileshandler.NewConnectHandler(profilesRepo).WithAuthorization(func(ctx context.Context) error {
		return authz.RequireWrite(ctx)
	}).WithReadAuthorization(func(ctx context.Context) error {
		return authz.RequireRead(ctx)
	})
	profilesConnectPath, profilesConnectHandler := profilesconnect.NewProfilesServiceHandler(profilesConnectHandlerDomain)

	// Best-effort inter-scenario clients; the orchestrator skips the matching
	// step if a client is nil, and logs a warning on construction failure.
	var cloudClient deployments.CloudHealthClient
	if c, err := deployments.NewHTTPCloudHealthClient(logFn); err == nil {
		cloudClient = c
	} else {
		LogStructured("cloud health client unavailable", map[string]interface{}{"error": err.Error()})
	}
	var lpbsClient deployments.LPBSReleaseClient
	if c, err := deployments.NewHTTPLPBSReleaseClient(deployments.LPBSClientConfig{Log: logFn}); err == nil {
		lpbsClient = c
	} else {
		LogStructured("lpbs release client unavailable", map[string]interface{}{"error": err.Error()})
	}

	srv := &Server{
		Config:              cfg,
		DB:                  routedDB.Primary(),
		RoutedDB:            routedDB,
		Router:              mux.NewRouter(),
		Logger:              NewProcessLogger(),
		ProfilesRepo:        profilesRepo,
		SigningRepo:         signingRepo,
		HealthHandler:       health.NewHandler(routedDB),
		FitnessHandler:      fitness.NewHandler(logFn),
		TelemetryHandler:    telemetry.NewHandler(logFn),
		SecretsHandler:      secrets.NewHandler(profilesRepo, logFn),
		DependenciesHandler: dependencies.NewHandler(logFn),
		SwapsHandler:        swaps.NewHandler(profilesRepo, logFn),
		DeploymentsHandler:  deployments.NewHandler(logFn),
		BundlesHandler: bundles.NewHandlerWithSigning(secrets.NewClient(), profilesRepo, signingRepo, logFn).WithExportAuthorization(func(ctx context.Context) error {
			return authz.RequireReleasePreparation(ctx)
		}),
		ProfilesHandler:          profiles.NewHandler(profilesRepo, logFn),
		SigningHandler:           codesigning.NewHandler(signingRepo, logFn),
		BuildHandler:             build.NewHandler(profilesRepo, logFn),
		ApprovalsHandler:         approvalsHandler,
		PublishedVersionsHandler: deployments.NewPublishedVersionsHandler(publishedVersionsRepo, logFn),
		LPBSConfigHandler:        profiles.NewLPBSConfigHandler(profilesRepo, lpbsConfigRepo, logFn),
		MigrationTasksHandler:    migrationtasks.NewHandler(logFn),
		EvidencePath:             evidencePath,
		EvidenceHandler:          evidenceHandler,
		ProfilesConnectPath:      profilesConnectPath,
		ProfilesConnectHandler:   profilesConnectHandler,
		LPBSConfigRepo:           lpbsConfigRepo,
		ReleasesRepo:             releasesRepo,
		ReadinessRepo:            readinessRepo,
		Orchestrator: deployments.NewOrchestratorFull(
			profilesRepo, approvalsRepo, publishedVersionsRepo,
			releasesRepo, lpbsConfigRepo, cloudClient, lpbsClient, logFn,
		),
	}
	goalClient := readiness.NewGoalClient()
	readinessPolicy := readiness.DefaultChecklist()
	policyDigest, err := readiness.PolicyDigest(readinessPolicy)
	if err != nil {
		return nil, fmt.Errorf("digest readiness policy: %w", err)
	}
	recoveryExecutor, _ := interface{}(srv.Orchestrator).(releases.RecoveryExecutor)
	srv.ReadinessPreparer = &readiness.Preparer{
		Policy: readinessPolicy, Repository: readinessRepo,
		Producers: map[string]readiness.EvidenceProducer{
			"deployment-manager.policy.update":        readiness.PolicyProducer{Policy: readinessPolicy},
			"deployment-manager.artifacts.provenance": readiness.CandidateProvenanceProducer{Resolver: candidateProvenanceResolver{repository: releasesRepo}, ExpectedPolicyDigest: policyDigest},
			"deployment-manager.evidence.coverage":    readiness.RampEvidenceProducer{Resolver: rampEvidenceResolver{repository: evidenceRepo}},
			"deployment-manager.assets.readiness":     readiness.PlatformAssetsProducer{Resolver: candidateProvenanceResolver{repository: releasesRepo}},
			"deployment-manager.recovery.preflight":   readiness.RecoveryProducer{Resolver: recoveryReadinessResolver{executor: recoveryExecutor, authorizationConfigured: true}},
			// These bindings are explicit owner seams. Keeping them in the producer
			// map prevents Prepare from reading a stale persisted observation and
			// presenting it as current Deployment Manager execution.
			"deployment-manager.compatibility.compare":   readiness.UnavailableProducer{Binding: "deployment-manager.compatibility.compare", Reason: "predecessor compatibility comparison owner is not configured"},
			"deployment-manager.observability.readiness": readiness.ObservabilityProducer{Resolver: cloudObservabilityResolver{repository: releasesRepo, client: cloudClient}},
			"deployment-manager.operations.readiness":    readiness.OperationsOwnershipProducer{Resolver: candidateProvenanceResolver{repository: releasesRepo}},
		},
		Goals: goalClient, Predecessor: readinessPredecessorResolver{releases: releasesRepo, reviews: readinessRepo},
	}
	srv.ReleasesHandler = releases.NewHandler(
		releasesRepo, lpbsConfigRepo, releasesVerifierAdapter{inner: lpbsClient}, srv.Orchestrator, logFn,
	).WithAuthorization(func(ctx context.Context) error {
		return authz.RequireWrite(ctx)
	}).WithRecoveryAuthorization(func(ctx context.Context) error {
		return authz.RequireDestructive(ctx)
	}).WithActorResolver(func(ctx context.Context) (string, error) {
		principal, err := authn.RequireHuman(ctx)
		if err != nil {
			return "", err
		}
		return principal.Subject, nil
	}).WithReadAuthorization(func(ctx context.Context) error {
		return authz.RequireRead(ctx)
	}).WithAlertPublisher(eventbus.NewDiscoveredClient(context.Background())).WithSigningExpiryLookup(signingExpiryLookup{repo: signingRepo}).WithDurableOperations(true).WithRecoveryExecutor(srv.Orchestrator).WithReadinessLookup(func(ctx context.Context, key string) (*releases.ReadinessApproval, error) {
		review, err := readinessRepo.Get(ctx, key)
		if err != nil {
			return nil, err
		}
		evidence, findings, err := readinessRepo.ListEvaluation(ctx, key)
		if err != nil {
			return nil, err
		}
		now := time.Now().UTC()
		activeWaivers, err := readinessRepo.ListActiveWaivers(ctx, key, now)
		if err != nil {
			return nil, err
		}
		humanChecks, err := readinessRepo.ListHumanChecks(ctx, key)
		if err != nil {
			return nil, err
		}
		if review.GoalRef != "" && review.GoalClosedAt == nil {
			return nil, fmt.Errorf("independent goal closure has not been synchronized")
		}
		if err := readiness.ValidateCurrentEvidence(readinessPolicy, *review, evidence, activeWaivers, humanChecks, now); err != nil {
			return nil, err
		}
		evidenceDigest, err := readiness.EvidenceSetDigest(evidence, findings)
		if err != nil {
			return nil, err
		}
		policyDigest, err := readiness.PolicyDigest(readinessPolicy)
		if err != nil {
			return nil, err
		}
		return &releases.ReadinessApproval{Key: review.Key, Scenario: review.Identity.Scenario, ProfileID: review.Identity.ProfileID, CandidateCommit: review.Identity.CandidateCommit, ArtifactDigest: review.Identity.ArtifactDigest, CandidateID: review.Identity.CandidateID, DestinationRevisionID: review.Identity.DestinationRevisionID, AuthorizationEpoch: review.Identity.AuthorizationEpoch, EvidenceSetDigest: evidenceDigest, PolicyDigest: policyDigest, Targets: review.Identity.Targets, Channel: review.Identity.Channel, PolicyVersion: review.Identity.PolicyVersion, Status: string(review.Status), ApprovedAt: review.ApprovedAt}, nil
	}).WithReadinessPromoter(readinessRepo.MarkPromoted)
	srv.ReleasesHandler.WithIdentityRepository(releasesRepo)
	connectTransport := transport.NewHandler(
		srv.DependenciesHandler, srv.FitnessHandler, srv.DeploymentsHandler, srv.Orchestrator,
		srv.SwapsHandler, srv.TelemetryHandler.List, srv.TelemetryHandler.Upload, srv.MigrationTasksHandler.Report, srv.MigrationTasksHandler.Status,
		srv.ApprovalsHandler, srv.LPBSConfigHandler.Get, srv.LPBSConfigHandler.Upsert,
		srv.ReleasesHandler.ListByProfile, srv.ReleasesHandler.Get, srv.ReleasesHandler.Verify, srv.ReleasesHandler.Start,
	).WithReleaseDossierHandler(srv.ReleasesHandler.Dossier).WithReleaseOperationHandler(srv.ReleasesHandler.GetOperation).WithReleaseReconcileHandler(srv.ReleasesHandler.Reconcile).WithReleaseRecoverHandler(srv.ReleasesHandler.Recover).WithAuthorization(func(ctx context.Context) error {
		return authz.RequireWrite(ctx)
	}).WithClientUpdateReceiptAuthorization(func(ctx context.Context) error {
		return authz.RequireClientUpdateReceipt(ctx)
	}).WithReleasePreparationAuthorization(func(ctx context.Context) error {
		return authz.RequireReleasePreparation(ctx)
	}).WithReadAuthorization(func(ctx context.Context) error {
		return authz.RequireRead(ctx)
	}).WithReleaseIdentityRepository(releasesRepo)
	srv.ConnectRoutes = transport.Routes(connectTransport)
	readinessDomainHandler := readinesshandler.NewConnectHandler(srv.ReadinessPreparer, readinessRepo, goalClient).WithEvidenceBindingAuthorization(func(ctx context.Context, binding string) error {
		return authz.RequireReadinessEvidenceForBinding(ctx, binding)
	})
	readinessPath, readinessHandler := readinessconnect.NewReadinessServiceHandler(readinessDomainHandler)
	srv.ConnectRoutes = append(srv.ConnectRoutes, transport.Route{Path: readinessPath, Handler: readinessHandler})

	srv.setupRoutes()
	rootMux := http.NewServeMux()
	devrouting.RegisterWithFileRoots(rootMux, routedDB, fileRoots)
	rootMux.Handle("/", srv.Router)
	srv.Handler = authn.Middleware(authConfig)(apihttp.TestModeMiddleware(SecurityHeadersMiddleware(rootMux)))
	if err := srv.ReleasesHandler.ResumeOperations(context.Background()); err != nil {
		_ = routedDB.Close()
		return nil, fmt.Errorf("resume release operations: %w", err)
	}
	return srv, nil
}

type signingExpiryLookup struct {
	repo *codesigning.ProxyRepository
}

func (l signingExpiryLookup) LookupSigningExpiry(ctx context.Context, _ string, platform string) (*releases.SigningExpiry, error) {
	if l.repo == nil {
		return nil, nil
	}
	platform = strings.ToLower(strings.TrimSpace(platform))
	switch {
	case strings.HasPrefix(platform, "windows") || strings.HasPrefix(platform, "win"):
		platform = codesigning.PlatformWindows
	case strings.HasPrefix(platform, "darwin") || strings.HasPrefix(platform, "mac") || strings.HasPrefix(platform, "osx"):
		platform = codesigning.PlatformMacOS
	case strings.HasPrefix(platform, "linux"):
		platform = codesigning.PlatformLinux
	default:
		return nil, nil
	}
	certificates, err := l.repo.DiscoverCertificates(ctx, platform)
	if err != nil {
		return nil, err
	}
	var earliest time.Time
	for _, certificate := range certificates {
		if !certificate.IsCodeSign || strings.TrimSpace(certificate.ExpiresAt) == "" {
			continue
		}
		expiresAt, parseErr := time.Parse(time.RFC3339, certificate.ExpiresAt)
		if parseErr != nil {
			continue
		}
		if earliest.IsZero() || expiresAt.Before(earliest) {
			earliest = expiresAt
		}
	}
	if earliest.IsZero() {
		return nil, nil
	}
	return &releases.SigningExpiry{Platform: platform, ExpiresAt: earliest.UTC()}, nil
}

func newFileRoots() (*filerouting.RoutedRoots, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return nil, fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("deployment-manager")
	if err != nil {
		return nil, fmt.Errorf("resolve storage namespace: %w", err)
	}
	roots, err := storage.EnsureAllDirs(resolver, storage.Options{ScenarioID: scenarioID}, 0o755)
	if err != nil {
		return nil, fmt.Errorf("resolve storage roots: %w", err)
	}
	return filerouting.New(roots), nil
}

// Start launches the HTTP server with graceful shutdown.
func (s *Server) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.startReleaseFactPublisher(ctx)

	LogStructured("starting server", map[string]interface{}{
		"service": "deployment-manager-api",
		"port":    s.Config.Port,
	})

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", s.Config.Port),
		Handler:      handlers.RecoveryHandler()(s.Handler),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			LogStructured("server startup failed", map[string]interface{}{"error": err.Error()})
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	LogStructured("server stopped", nil)
	return nil
}

// WriteJSON sends a JSON response with the given status code.
func (s *Server) WriteJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
