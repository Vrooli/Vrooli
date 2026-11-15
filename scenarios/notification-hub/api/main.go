package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/packages/credential-authority-go/eventpublishing"

	"notification-hub/internal/capabilities"
	"notification-hub/internal/hub"
	internalidentity "notification-hub/internal/identity"
	"notification-hub/internal/integrations"
	"notification-hub/internal/modules"
	"notification-hub/internal/preferences"
	"notification-hub/internal/push"
	"notification-hub/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/askreader"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/hubsender"
	apilifecycle "github.com/vrooli/api-core/lifecycle"
	"github.com/vrooli/api-core/owneridentity"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/api-core/trustposture"
	_ "modernc.org/sqlite"

	appDeliveryH "notification-hub/handlers/appdelivery"
	capsH "notification-hub/handlers/capabilities"
	conversationH "notification-hub/handlers/conversations"
	deliveryH "notification-hub/handlers/delivery"
	externalURLH "notification-hub/handlers/externalurl"
	healthH "notification-hub/handlers/health"
	identityH "notification-hub/handlers/identity"
	notificationH "notification-hub/handlers/notifications"
	preferencesH "notification-hub/handlers/preferences"
	recipientsH "notification-hub/handlers/recipients"
	routingH "notification-hub/handlers/routing"
)

// preferenceSeam adapts *preferences.Storage to hub.Preferences, bridging the
// preferences.Effective result into hub.PreferenceDecision so the hub's
// delivery seam carries no import of the preferences type set.
type preferenceSeam struct {
	store    *preferences.Storage
	repoRoot string
}

func (s preferenceSeam) ObserveCatalog(ctx context.Context, source, category, now string) error {
	return s.store.ObserveCatalog(ctx, source, category, now)
}

// InferObservedBranding resolves display branding for a newly-observed source
// from its service.json, when the repo root was resolvable at startup.
func (s preferenceSeam) InferObservedBranding(ctx context.Context, source, now string) {
	if s.repoRoot == "" {
		return
	}
	s.store.InferObservedBranding(ctx, s.repoRoot, source, now)
}

// ResolveBrand exposes the source's declared icon/deep-link branding to the
// hub's icon resolver, satisfying hub.BrandResolver.
func (s preferenceSeam) ResolveBrand(ctx context.Context, source string) (displayName, icon, deepLinkBase string, ok bool) {
	return s.store.ResolveBrand(ctx, source)
}

func (s preferenceSeam) GetPreference(ctx context.Context, recipientKey, source, category, scopeKey string) (hub.PreferenceDecision, error) {
	effective, err := s.store.GetPreference(ctx, recipientKey, source, category, scopeKey)
	if err != nil {
		return hub.PreferenceDecision{}, err
	}
	return hub.PreferenceDecision{Enabled: effective.Enabled, MinUrgency: effective.MinUrgency}, nil
}

func (s preferenceSeam) UrgencyAtLeast(have, min string) bool {
	return preferences.UrgencyAtLeast(have, min)
}

type pushAdapter struct{ sender *push.Sender }

func (a pushAdapter) Send(ctx context.Context, subscription hub.PushSubscription, message hub.PushMessage) (string, error) {
	sub := push.Subscription{Endpoint: subscription.Endpoint, P256DH: subscription.P256DH, Auth: subscription.Auth, Origin: subscription.Origin}
	msg := push.Message{Title: message.Title, Body: message.Body, ID: message.Tag, URL: message.URL, Icon: message.Icon, Awaiting: message.Awaiting}
	if message.Navigate != "" {
		return a.sender.SendPayload(ctx, sub, push.DeclarativePayload(msg, message.Navigate))
	}
	return a.sender.SendMessage(ctx, sub, msg)
}

// scenarioStorageRoots resolves all filesystem storage classes once at
// startup. File writers must select their class through fileRootPath so a
// test-mode request uses the lease-owned root instead of the live tree.
func scenarioStorageRoots() (storage.Paths, error) {
	return resolveScenarioStorageRoots(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
}

// resolveScenarioStorageRoots resolves the storage classes with the given
// resolver settings; scenarioStorageRoots passes the process's own.
//
// The data root is the directory the database lives in, taken from the same
// call database.Open makes, so a packet's rows and the files they point at can
// never end up in different roots. They used to: a lifecycle-launched live
// instance keeps its database in SCENARIO_DATA_DIR (<scenario>/data, see
// storage.SQLitePath) while the class resolver answers
// ~/.vrooli/data/vrooli/notification-hub, so attachments landed in the second
// and the database in the first. A shadow variant or a test lease resolves both
// to the same class root, and then this changes nothing.
func resolveScenarioStorageRoots(cfg storage.ResolverConfig) (storage.Paths, error) {
	resolver, err := storage.NewResolver(cfg)
	if err != nil {
		return storage.Paths{}, fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("notification-hub")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve notification-hub storage namespace: %w", err)
	}
	paths, err := resolver.Resolve(storage.Options{ScenarioID: scenarioID})
	if err != nil {
		return storage.Paths{}, err
	}
	databasePath, err := storage.SQLitePath(storage.SQLiteConfig{
		Scenario: "notification-hub", AppID: cfg.AppID, Profile: cfg.Profile, EnvGet: cfg.EnvGet, UserHomeDir: cfg.UserHomeDir,
	})
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve notification-hub database location: %w", err)
	}
	paths.DataDir = filepath.Dir(databasePath)
	return paths, nil
}

// fileRootPath is the template's mandatory file-store seam. Domain stores
// compose their relative paths from it rather than retaining startup root
// strings, so X-Vrooli-Test-Mode is honored independently per request.
func fileRootPath(ctx context.Context, roots *filerouting.RoutedRoots, class storage.Class, rel string) (string, error) {
	root, err := roots.Pick(ctx, class)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, rel), nil
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "notification-hub"}) {
		return
	}
	eventpublishing.Register()

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "notification-hub",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	db.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		if err := database.EnsureSchemas(ctx, pool, modules.AllSchemas()...); err != nil {
			return err
		}
		return seedTestRecipient(ctx, pool)
	})

	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}
	primaryFileRoots, err := scenarioStorageRoots()
	if err != nil {
		log.Fatalf("file storage configuration failed: %v", err)
	}
	fileRoots := filerouting.New(primaryFileRoots)
	service := hub.New(db, schedule.System(), log.Default())
	preferenceStore := preferences.New(db)
	// Resolve the repo root once for catalog ingestion and observed-branding
	// inference. A missing root is non-fatal: declared manifests are skipped and
	// branding falls back to the observed path.
	repoRoot, repoRootErr := preferences.ResolveRepoRoot()
	if repoRootErr != nil {
		slog.Warn("notification catalog: repo root unresolved; declared-manifest ingestion and branding inference disabled", "error", repoRootErr)
	} else if err := preferenceStore.IngestDeclaredManifests(context.Background(), repoRoot, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		slog.Warn("notification catalog: declared-manifest ingestion failed", "error", err)
	}
	service.SetPreferences(preferenceSeam{store: preferenceStore, repoRoot: repoRoot})
	service.BackfillAskSenders(context.Background()) // now that source names resolve
	service.SetDefaultRecipient(strings.TrimSpace(os.Getenv("VROOLI_NOTIFICATION_RECIPIENT")))
	service.SetRecipientResolver(integrations.OperatorStateRecipient())
	service.SetOwnerFallback(hub.OperatorSubject)
	configureWebPush(service, fileRoots)
	configureAttachmentStore(service, fileRoots)
	service.SetAppEligibility(hub.TunnelAppEligibility(func(ctx context.Context) (string, error) {
		return discovery.ResolveScenarioURLDefault(ctx, "tunnel-manager")
	}))
	posture := "personal"
	if state, postureErr := trustposture.LoadWorkingTree(); postureErr != nil {
		slog.Warn("trust posture unavailable; using fail-safe personal posture", "error", postureErr)
	} else {
		posture = string(state.Posture)
	}
	scenarioResolver := discovery.NewResolver(discovery.ResolverConfig{})
	service.SetNativeConsentReader(integrations.AgentManagerConsentReader{Resolver: scenarioResolver})
	ownerVerifier := owneridentity.NewClient(owneridentity.Config{Resolver: scenarioResolver})
	// Service principals the Hub admits besides the owner and agent runs: a scenario
	// reading back the asks it raised, and a scenario sending as itself.
	serviceVerifiers := internalidentity.ServiceVerifiers{AskReaders: askreader.NewVerifier(), Senders: hubsender.NewVerifier()}
	providerDecisionAudience := strings.TrimSpace(os.Getenv("VROOLI_AUTH_PROVIDER_DECISION_AUDIENCE"))
	var providerDecisionVerifier authn.TokenVerifier
	if providerDecisionAudience != "" {
		providerDecisionVerifier = authn.NewScenarioAuthenticatorProvider(authn.JWTConfig{Audience: providerDecisionAudience})
	}
	service.SetEmailSender(hub.NewSMTPSenderFromEnvironment(os.Getenv))
	service.SetDesktopSender(hub.NewDesktopSender())
	slog.Info("desktop notification transport selected", "platform", runtime.GOOS)
	service.SetRemoteDelivery(hub.NewBridgeRemoteFromEnvironment())
	if switchboardURL := strings.TrimSpace(os.Getenv("SWITCHBOARD_API_URL")); switchboardURL != "" {
		service.SetChannelDelivery(hub.NewSwitchboardDelivery(switchboardURL))
	} else if switchboardURL, resolveErr := discovery.ResolveScenarioURLDefault(context.Background(), "switchboard"); resolveErr == nil {
		service.SetChannelDelivery(hub.NewSwitchboardDelivery(switchboardURL))
	} else {
		slog.Warn("switchboard URL unavailable; conversational notification delivery will retry when configured", "error", resolveErr)
	}
	eventDefaults := integrations.LiveConfig{
		Pattern:               "incident.**",
		SensitivityBySeverity: map[string]string{"critical": "critical", "warning": "sensitive", "informational": "public"},
		Templates: map[string]integrations.EventTemplate{
			"incident.opened.v1":           {Title: "Incident opened: {{check_id}}", Body: "{{message}}"},
			"incident.severity_changed.v1": {Title: "Incident severity changed: {{check_id}}", Body: "{{message}}"},
			"incident.resolved.v1":         {Title: "Incident resolved: {{check_id}}", Body: "{{message}}"},
		},
	}
	if eventsBase := strings.TrimSpace(os.Getenv("VROOLI_EVENTS_API_BASE")); eventsBase != "" {
		eventDefaults.EventsAPIBase = eventsBase
	} else if eventsBase, resolveErr := discovery.ResolveScenarioURLDefault(context.Background(), "vrooli-events"); resolveErr == nil {
		eventDefaults.EventsAPIBase = eventsBase
	} else {
		slog.Error("vrooli-events URL could not be resolved", "error", resolveErr)
	}
	if apiPort := strings.TrimSpace(os.Getenv("API_PORT")); apiPort != "" {
		eventDefaults.WebhookURL = "http://127.0.0.1:" + apiPort + "/api/v1/integrations/events"
	}
	eventConfig, err := integrations.NewPersistentLiveConfigStore(context.Background(), db, eventDefaults)
	if err != nil {
		log.Fatalf("event integration config initialization failed: %v", err)
	}
	// A missing Events service at startup is transient. Rediscover an absent
	// default and maintain each subscription independently; never replace an
	// explicitly configured endpoint with an inferred one.
	eventsBase := func(ctx context.Context) string {
		if base := eventConfig.Get().EventsAPIBase; base != "" {
			return base
		}
		base, _ := discovery.ResolveScenarioURLDefault(ctx, "vrooli-events")
		return base
	}
	reconcileSubscription := func(ctx context.Context, kind string) error {
		config := eventConfig.Get()
		base := eventsBase(ctx)
		var err error
		switch kind {
		case "incident":
			err = integrations.EnsureEventSubscription(ctx, base, config.WebhookURL, config.Pattern)
		case "decision":
			err = integrations.EnsureDecisionSubscription(ctx, base, config.WebhookURL)
		case "withdrawal":
			err = integrations.EnsureDecisionWithdrawalSubscription(ctx, base, config.WebhookURL)
		case "notification":
			err = integrations.EnsureNotificationSubscription(ctx, base, config.WebhookURL)
		case "heartbeat":
			err = integrations.EnsureHeartbeatAttemptSubscription(ctx, base, config.WebhookURL)
		}
		if err != nil {
			slog.Warn("vrooli-events subscription reconciliation failed; will retry", "kind", kind, "error", err)
		}
		return err
	}
	var subscriptionRetryMu sync.Mutex
	var subscriptionRetryCancel context.CancelFunc
	var subscriptionRetryDone chan struct{}
	startSubscriptionRetry := func() {
		subscriptionRetryMu.Lock()
		defer subscriptionRetryMu.Unlock()
		if subscriptionRetryCancel != nil {
			return
		}
		workerCtx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		subscriptionRetryCancel, subscriptionRetryDone = cancel, done
		go func() {
			defer close(done)
			integrations.RetrySubscriptions(workerCtx, 30*time.Second, reconcileSubscription, "incident", "decision", "withdrawal", "notification", "heartbeat")
		}()
	}
	stopSubscriptionRetry := func(ctx context.Context) error {
		subscriptionRetryMu.Lock()
		cancel, done := subscriptionRetryCancel, subscriptionRetryDone
		if cancel == nil {
			subscriptionRetryMu.Unlock()
			return nil
		}
		cancel()
		subscriptionRetryMu.Unlock()
		select {
		case <-done:
			subscriptionRetryMu.Lock()
			if subscriptionRetryDone == done {
				subscriptionRetryCancel, subscriptionRetryDone = nil, nil
			}
			subscriptionRetryMu.Unlock()
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	startSubscriptionRetry()
	// The webhook verifies with the publisher's own signing secret: the
	// environment override, then the credential authority. The value is
	// never logged.
	webhookSecret := integrations.CachedSecret(func() (string, error) {
		return integrations.ResolveEventsWebhookSecret(os.Getenv)
	}, func(err error) {
		slog.Warn("vrooli-events webhook secret unavailable; signed event deliveries are rejected until it resolves", "error", err)
	})
	slog.Info("vrooli-events webhook secret", "resolved", webhookSecret() != "")
	service.SetResolutionPublisher(integrations.EventsPublisher{BaseURL: func() string { return eventsBase(context.Background()) }})
	configureAskDefaultFloor(service)
	configurePushBudget(service, os.Getenv)

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.ModuleWithIdentity(db, ownerVerifier, "notification-hub-api", "1.0.0", posture),
		capsH.Module(capabilities.NewRegistry()),
		externalURLH.Module(),
		notificationH.ModuleWithIdentities(service, ownerVerifier, serviceVerifiers),
		recipientsH.ModuleWithVerifier(service, ownerVerifier),
		appDeliveryH.ModuleWithVerifier(service, ownerVerifier),
		preferencesH.ModuleWithRecipientResolver(preferenceStore, service.ResolveRecipient, ownerVerifier),
		routingH.ModuleWithVerifier(service, ownerVerifier),
		deliveryH.ModuleWithVerifier(service, ownerVerifier),
		conversationH.ModuleWithIdentities(service, ownerVerifier, serviceVerifiers),
		identityH.Module(internalidentity.Forwarder{Resolver: scenarioResolver}, ownerVerifier),
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.RegisterWithFileRoots(rootMux, db, fileRoots)
	rootMux.Handle("/api/v1/integrations/events", integrations.EventWebhookFromSecretSourceWithDecisionDefaults(service, webhookSecret, func() map[string]integrations.EventTemplate {
		return eventConfig.Get().Templates
	}, func() map[string]string { return eventConfig.Get().SensitivityBySeverity }, func() integrations.DecisionDefaults {
		config := eventConfig.Get()
		return integrations.DecisionDefaults{Sensitivity: config.DecisionSensitivity, DefaultDeadline: time.Duration(config.DecisionDeadlineSecond) * time.Second}
	}))
	rootMux.Handle("/api/v1/integrations/deliveries", integrations.DeliveryProjectionHandler(service))
	rootMux.Handle(internalidentity.ProviderDecisionRoute, internalidentity.ProviderDecisionLookupHandler(service, providerDecisionVerifier))
	rootMux.Handle(integrations.ProviderDecisionPublishRoute, integrations.ProviderDecisionPublishHandler(service, providerDecisionVerifier))
	rootMux.Handle("/api/v1/config/event-integration", integrations.Handler(eventConfig, func(config integrations.LiveConfig) error {
		return integrations.EnsureEventSubscription(context.Background(), config.EventsAPIBase, config.WebhookURL, config.Pattern)
	}))

	// The owner's Cloudflare Access login is verified for the Connect API
	// only. The signed event webhook and the routes above keep their own
	// authentication. Reading an ask needs an owner, the run that sent it, or the
	// service that raised it.
	ownerAuth := internalidentity.OwnerAuthenticator{
		Providers:    internalidentity.OwnerProvidersFromEnvironment(os.Getenv, scenarioResolver, "notification-hub"),
		OwnerSubject: service.ResolveRecipient,
	}
	rootMux.Handle("/", ownerAuth.Middleware(srv.Handler()))

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// request context so *database.RoutedDB routes the call to the
	// installed test pool. Self-disables in production mode.
	handler := apihttp.TestModeMiddleware(requireRoutedTestLease(db, rootMux, service))

	if err := apiserver.Run(apiserver.Config{
		Handler: handler,
		LifecycleHooks: apilifecycle.Hooks{
			Quiesce: stopSubscriptionRetry,
			Resume: func(context.Context) error {
				startSubscriptionRetry()
				return nil
			},
		},
		Cleanup: func(ctx context.Context) error {
			if err := stopSubscriptionRetry(ctx); err != nil {
				return err
			}
			// The worker uses the database, so it stops first.
			service.Close()
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// requireRoutedTestLease prevents a test-mode request from reaching a handler
// unless it can be pinned to the currently installed lease. RoutedDB normally
// preserves production behavior by falling back to Primary when no test pool
// exists; browser test-mode requests must instead fail closed. Pinning closes
// the race between checking the lease and the handler's first database call.
func requireRoutedTestLease(db *database.RoutedDB, next http.Handler, fixtureServices ...*hub.Service) http.Handler {
	type pendingFixture struct {
		name       string
		selectedAt time.Time
	}
	var pendingMu sync.Mutex
	pendingByLease := make(map[string]pendingFixture)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !database.IsTestMode(r.Context()) {
			next.ServeHTTP(w, r)
			return
		}
		leaseID := db.ActiveLeaseID()
		if leaseID == "" {
			http.Error(w, "routed test lease is not active", http.StatusServiceUnavailable)
			return
		}
		ctx, err := db.PinTestPool(r.Context(), leaseID)
		if err != nil {
			http.Error(w, "routed test lease is not active", http.StatusServiceUnavailable)
			return
		}
		// A routed test lease is a disposable owner boundary as well as a
		// storage boundary. Give every request in that pinned lease the fixed
		// fixture recipient so scenario-to-scenario rehearsals can exercise the
		// real producer and readback handlers without inventing owner credentials.
		// Production identity remains untouched: this context exists only when
		// TestModeMiddleware has marked the request and the lease was pinned above.
		ctx = internalidentity.WithLeasedTestOwner(ctx, hub.TestRecipientSubject)
		r = r.WithContext(ctx)
		// E4 visual cases may select a disposable-pool fixture after the lease
		// has been pinned. These mutations cannot run on ordinary requests or
		// fall through to primary storage.
		fixture := ""
		if r.URL.Path == "/vrooli.notification_hub.v1.conversations.ConversationsService/ListAsks" {
			fixture = r.Header.Get("X-Notification-Hub-E4-Fixture")
			refererPath := ""
			refererFixture := ""
			if referer, parseErr := url.Parse(r.Referer()); parseErr == nil {
				refererPath = referer.Path
				refererFixture = referer.Query().Get("e4_fixture")
				if fixture == "" {
					fixture = refererFixture
				}
			}
			slog.Info("E4 test-mode ListAsks selector inspected", "lease_id", leaseID, "referer_path", refererPath, "referer_fixture", refererFixture, "header_fixture", r.Header.Get("X-Notification-Hub-E4-Fixture"))
			pendingMu.Lock()
			now := time.Now()
			for lease, pending := range pendingByLease {
				if now.Sub(pending.selectedAt) > time.Minute {
					delete(pendingByLease, lease)
				}
			}
			applyPath := "/asks"
			if fixture == "nh16.2-history" {
				applyPath = "/history"
			}
			if fixture != "" && refererPath != applyPath {
				pendingByLease[leaseID] = pendingFixture{name: fixture, selectedAt: now}
				fixture = ""
			} else {
				if fixture == "" {
					fixture = pendingByLease[leaseID].name
					if fixture == "nh16.2-history" {
						if refererPath == "/history" {
							delete(pendingByLease, leaseID)
						} else {
							fixture = ""
						}
					}
				}
			}
			pendingMu.Unlock()
		} else if r.Header.Get("X-Notification-Hub-E4-Fixture") == "decision-links-phone" && (r.URL.Path == "/vrooli.notification_hub.v1.conversations.ConversationsService/GetAsk" || strings.Contains(r.URL.Path, "/attachments/")) {
			// This bounded visual fixture is selected only by an explicit routed
			// test request. The decision and its content route share the same
			// short lived attachment seed so every read stays inside the lease.
			if referer, parseErr := url.Parse(r.Referer()); parseErr == nil && referer.Path == "/asks/"+hub.TestAskID {
				fixture = "decision-links-phone"
			}
		}
		// A selected Delivery analytics failure is carried from the Updates
		// page's initial routed list read to exactly one analytics request. The
		// fault is a real SQL read failure in the pinned disposable pool; the
		// table is restored before this request returns, so a retry succeeds and
		// timeline/detail reads remain independently available.
		injectDeliveryAnalyticsFailure := false
		if r.URL.Path == "/vrooli.notification_hub.v1.delivery.DeliveryService/GetAnalytics" {
			pendingMu.Lock()
			pending, ok := pendingByLease[leaseID]
			if ok && time.Since(pending.selectedAt) <= time.Minute && pending.name == "delivery-analytics-read-failure" {
				delete(pendingByLease, leaseID)
				injectDeliveryAnalyticsFailure = true
			}
			pendingMu.Unlock()
		}
		if r.URL.Path == "/vrooli.notification_hub.v1.delivery.DeliveryService/GetTimeline" {
			pendingMu.Lock()
			pending, ok := pendingByLease[leaseID]
			if ok && time.Since(pending.selectedAt) <= time.Minute && (pending.name == "delivery-unknown-state" || pending.name == "delivery-pending-state") {
				delete(pendingByLease, leaseID)
				fixture = pending.name
			}
			pendingMu.Unlock()
		}
		if injectDeliveryAnalyticsFailure {
			if _, err := db.ExecContext(ctx, `ALTER TABLE delivery_attempts RENAME TO delivery_attempts_e4_fixture_backup`); err != nil {
				http.Error(w, "test fixture setup failed", http.StatusInternalServerError)
				return
			}
			defer func() {
				if _, err := db.ExecContext(ctx, `ALTER TABLE delivery_attempts_e4_fixture_backup RENAME TO delivery_attempts`); err != nil {
					slog.Error("E4 delivery analytics fixture restore failed", "lease_id", leaseID, "error", err)
				}
			}()
		}
		if fixture != "" {
			slog.Info("E4 routed fixture selected", "fixture", fixture, "lease_id", leaseID, "path", r.URL.Path)
		}
		restoreFixture, err := applyE4TestFixture(ctx, db, fixture)
		if err != nil {
			http.Error(w, "test fixture setup failed", http.StatusInternalServerError)
			return
		}
		defer restoreFixture()
		if fixture == "decision-links-phone" && len(fixtureServices) > 0 && fixtureServices[0] != nil {
			restoreArtifact, err := attachE4DecisionLinkArtifact(ctx, db, fixtureServices[0])
			if err != nil {
				http.Error(w, "test artifact setup failed", http.StatusInternalServerError)
				return
			}
			defer restoreArtifact()
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// attachE4DecisionLinkArtifact stores a real disposable attachment for the
// phone decision-link fixture. The browser fetches these bytes through the
// Hub's attachment route; it never reads a requester-host path or Web Console.
func attachE4DecisionLinkArtifact(ctx context.Context, db *database.RoutedDB, service *hub.Service) (func(), error) {
	const name = "phone-ready-project-brief.md"
	const sourcePath = "/work/phone-ready-project-brief.md"
	const content = "# Phone-ready project brief\n\nThe short review is ready. Approving it lets the agent finish this one bounded step.\n"
	digest := sha256.Sum256([]byte(content))
	hash := hex.EncodeToString(digest[:])
	stored, err := service.PutAttachmentBlob(ctx, hash, "text/markdown", []byte(content))
	if err != nil {
		return nil, fmt.Errorf("store disposable decision artifact: %w", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO ask_attachments (ask_id, name, position, label, kind, role, source_path, sha256, size_bytes, media_type, preview_kind) VALUES (?, ?, 0, ?, 'file', 'evidence', ?, ?, ?, 'text/markdown', 'markdown')`, hub.TestAskID, name, "Phone-ready project brief", sourcePath, stored.SHA256, stored.SizeBytes); err != nil {
		return nil, fmt.Errorf("attach disposable decision artifact: %w", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE asks SET attachment_count = 1 WHERE id = ?`, hub.TestAskID); err != nil {
		_, _ = db.ExecContext(ctx, `DELETE FROM ask_attachments WHERE ask_id = ? AND name = ?`, hub.TestAskID, name)
		return nil, fmt.Errorf("count disposable decision artifact: %w", err)
	}
	return func() {
		if _, err := db.ExecContext(context.WithoutCancel(ctx), `DELETE FROM ask_attachments WHERE ask_id = ? AND name = ?`, hub.TestAskID, name); err != nil {
			slog.Error("restore disposable decision artifact", "error", err)
		}
		if _, err := db.ExecContext(context.WithoutCancel(ctx), `UPDATE asks SET attachment_count = 0 WHERE id = ?`, hub.TestAskID); err != nil {
			slog.Error("restore disposable decision artifact count", "error", err)
		}
	}, nil
}

func applyE4TestFixture(ctx context.Context, db *database.RoutedDB, fixture string) (func(), error) {
	switch fixture {
	case "":
		return func() {}, nil
	case "empty-open-asks":
		result, err := db.ExecContext(ctx, `DELETE FROM asks WHERE id = ? AND state IN ('pending', 'escalated')`, hub.TestAskID)
		if err != nil {
			return nil, err
		}
		deleted, err := result.RowsAffected()
		if err != nil {
			return nil, err
		}
		slog.Info("E4 empty ask fixture applied", "removed_open_asks", deleted)
		return func() {
			const stamp = "2026-01-01T00:00:00Z"
			_, _ = db.ExecContext(ctx, `INSERT INTO asks (id, notification_id, question, allowed_answers, deadline, state, created_at, updated_at, options_json) VALUES (?, ?, ?, ?, ?, 'pending', ?, ?, ?)`, hub.TestAskID, hub.TestNotificationID, "Choose a test-only option", `["continue","stop"]`, "2099-01-01T00:00:00Z", stamp, stamp, `["continue","stop"]`)
		}, nil
	case "decision-links-phone":
		var original struct {
			question, allowedAnswers, optionsJSON, recommended, recommendationReason, defaultAnswer string
		}
		if err := db.QueryRowContext(ctx, `SELECT question, allowed_answers, options_json, recommended, recommendation_reason, default_answer FROM asks WHERE id = ?`, hub.TestAskID).Scan(&original.question, &original.allowedAnswers, &original.optionsJSON, &original.recommended, &original.recommendationReason, &original.defaultAnswer); err != nil {
			return nil, fmt.Errorf("read disposable decision-link ask: %w", err)
		}
		question := "Review the [phone-ready project brief](/work/phone-ready-project-brief.md) before you choose."
		// Keep the seed ask's valid keys (continue/stop) so that answer submission
		// remains valid after this request-scoped visual fixture is restored.
		optionsJSON := `[{"key":"continue","label":"Approve the brief"},{"key":"stop","label":"Decide later"}]`
		if _, err := db.ExecContext(ctx, `UPDATE asks SET question = ?, allowed_answers = ?, options_json = ?, recommended = ?, recommendation_reason = ?, default_answer = ? WHERE id = ?`, question, `["continue","stop"]`, optionsJSON, "continue", "The brief has the supporting detail.", "continue", hub.TestAskID); err != nil {
			return nil, fmt.Errorf("set disposable decision-link ask: %w", err)
		}
		return func() {
			if _, err := db.ExecContext(context.WithoutCancel(ctx), `UPDATE asks SET question = ?, allowed_answers = ?, options_json = ?, recommended = ?, recommendation_reason = ?, default_answer = ? WHERE id = ?`, original.question, original.allowedAnswers, original.optionsJSON, original.recommended, original.recommendationReason, original.defaultAnswer, hub.TestAskID); err != nil {
				slog.Error("restore disposable decision-link ask", "error", err)
			}
		}, nil
	case "asks-read-failure":
		// Hiding the leased table for this ListAsks request makes the real
		// database query fail; no response is fabricated and other E4 cases
		// regain the ordinary disposable table when this request returns.
		_, err := db.ExecContext(ctx, `ALTER TABLE asks RENAME TO asks_e4_fixture_backup`)
		if err != nil {
			return nil, err
		}
		return func() {
			_, _ = db.ExecContext(ctx, `ALTER TABLE asks_e4_fixture_backup RENAME TO asks`)
		}, nil
	case "nh16.2-history":
		return func() {}, seedE4NH162HistoryFixture(ctx, db)
	case "delivery-pending-state", "delivery-unknown-state":
		// The selected atlas uses one record identity for both its Pending and
		// unknown-state examples. Transition that disposable record only after
		// an owner-routed Updates read selected this fixture for the active lease.
		//
		// The leased pool is shared by every case of one run, and the History
		// fixture's 229 synthetic answered records persist in it. Left in place
		// they fill the newest-first Delivery list ahead of the fixture record,
		// so a case that opens that record from the list would not find it.
		if err := removeE4NH162SyntheticRows(ctx, db); err != nil {
			return nil, err
		}
		state, reason := "pending", "E4 disposable Pending fixture"
		if fixture == "delivery-unknown-state" {
			state, reason = "future-state", "E4 disposable unknown-state fixture"
		}
		result, err := db.ExecContext(ctx, `UPDATE notification_events SET state = ?, reason = ? WHERE notification_id = 'DEMO-N043'`, state, reason)
		if err != nil {
			return nil, err
		}
		changed, err := result.RowsAffected()
		if err != nil {
			return nil, err
		}
		if changed != 1 {
			return nil, fmt.Errorf("expected one leased DEMO-N043 state row, updated %d", changed)
		}
		return func() {}, nil
	default:
		return nil, fmt.Errorf("unsupported E4 test fixture %q", fixture)
	}
}

// removeE4NH162SyntheticRows deletes the answered records the NH16.2 History
// fixture adds, so a fixture that needs a small, known record list starts from
// the common E4 seed. The base records the fixture only rewrites (the answered
// detail ask and its notification) are not synthetic and stay.
func removeE4NH162SyntheticRows(ctx context.Context, db *database.RoutedDB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin E4 NH16.2 cleanup: %w", err)
	}
	defer tx.Rollback()
	for _, statement := range []string{
		`DELETE FROM answers WHERE ask_id LIKE 'e4-test-nh162-ask-%'`,
		`DELETE FROM asks WHERE id LIKE 'e4-test-nh162-ask-%'`,
		`DELETE FROM notification_events WHERE notification_id LIKE 'e4-nh162-notification-%'`,
		`DELETE FROM notifications WHERE id LIKE 'e4-nh162-notification-%'`,
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("clear E4 NH16.2 synthetic rows: %w", err)
		}
	}
	return tx.Commit()
}

// seedE4NH162HistoryFixture replaces the common answered-history row with the
// selected 200-row NH16.2 inventory plus 30 older archive rows for pagination. The caller
// is pinned to the active Test Genie lease before this function can run.
func seedE4NH162HistoryFixture(ctx context.Context, db *database.RoutedDB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin E4 NH16.2 History fixture: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM answers WHERE ask_id IN (SELECT id FROM asks WHERE id = ? OR id LIKE 'e4-test-nh162-ask-%')`, "e4-test-answered-ask"); err != nil {
		return fmt.Errorf("clear E4 NH16.2 History answers: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM asks WHERE id = ? OR id LIKE 'e4-test-nh162-ask-%'`, "e4-test-answered-ask"); err != nil {
		return fmt.Errorf("clear E4 NH16.2 History asks: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM notification_events WHERE notification_id LIKE 'e4-nh162-notification-%'`); err != nil {
		return fmt.Errorf("clear E4 NH16.2 History events: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM notifications WHERE id LIKE 'e4-nh162-notification-%'`); err != nil {
		return fmt.Errorf("clear E4 NH16.2 History notifications: %w", err)
	}

	const recipient = hub.TestRecipientSubject
	for rank := 200; rank >= -29; rank-- {
		suffix := fmt.Sprintf("%03d", rank)
		notificationID := "e4-nh162-notification-" + suffix
		askID := "e4-test-nh162-ask-" + suffix
		source := fmt.Sprintf("DEMO-H%d", rank)
		question := fmt.Sprintf("Review capture check %d", rank)
		created := time.Date(2026, 10, 24, 5, 5, 0, 0, time.UTC).Add(-time.Duration(198-rank) * time.Second).Format(time.RFC3339)
		if rank <= 0 {
			archive := 1 - rank
			suffix = fmt.Sprintf("archive-%03d", archive)
			notificationID = "e4-nh162-notification-" + suffix
			askID = "e4-test-nh162-ask-" + suffix
			source = fmt.Sprintf("DEMO-ARCHIVE-%d", archive)
			question = fmt.Sprintf("Archived review %d", archive)
		}
		if rank == 200 {
			notificationID = "e4-test-history"
			askID = "e4-test-answered-ask"
			source = "DEMO-H200"
			question = "Validate the recording fix"
			created = "2026-10-25T05:05:00Z"
		} else if rank == 199 {
			question = "Confirm the release checklist"
			created = "2026-10-25T05:04:59Z"
		}
		if rank != 200 {
			if _, err := tx.ExecContext(ctx, `INSERT INTO notifications (id, requested_by, title, body, urgency, sensitivity_label, idempotency_key, created_at, updated_at) VALUES (?, ?, ?, ?, 'normal', 'public', ?, ?, ?)`, notificationID, recipient, question, "Synthetic answered E4 history record", notificationID, created, created); err != nil {
				return fmt.Errorf("seed E4 NH16.2 notification %s: %w", source, err)
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO notification_events (notification_id, state, reason, created_at) VALUES (?, 'answered', 'E4 NH16.2 synthetic history fixture', ?)`, notificationID, created); err != nil {
				return fmt.Errorf("seed E4 NH16.2 notification state %s: %w", source, err)
			}
		} else {
			if _, err := tx.ExecContext(ctx, `UPDATE notifications SET title = ?, body = ?, created_at = ?, updated_at = ? WHERE id = ?`, question, "Synthetic answered E4 history record", created, created, notificationID); err != nil {
				return fmt.Errorf("update E4 NH16.2 notification %s: %w", source, err)
			}
			if _, err := tx.ExecContext(ctx, `UPDATE notification_events SET state = 'answered', reason = 'E4 NH16.2 synthetic history fixture', created_at = ? WHERE notification_id = ?`, created, notificationID); err != nil {
				return fmt.Errorf("update E4 NH16.2 notification state %s: %w", source, err)
			}
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO asks (id, notification_id, question, allowed_answers, deadline, state, created_at, updated_at, options_json, source, resolved_at) VALUES (?, ?, ?, '["continue","stop"]', ?, 'answered', ?, ?, '["continue","stop"]', ?, ?)`, askID, notificationID, question, created, created, created, source, created); err != nil {
			return fmt.Errorf("seed E4 NH16.2 ask %s: %w", source, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO answers (id, ask_id, answer, answered_by, answered_at) VALUES (?, ?, 'continue', ?, ?)`, "e4-nh162-answer-"+suffix, askID, recipient, created); err != nil {
			return fmt.Errorf("seed E4 NH16.2 answer %s: %w", source, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit E4 NH16.2 History fixture: %w", err)
	}
	return nil
}

// seedTestRecipient creates a deterministic, unanswered fixture only in the
// disposable routed pool. It never runs against the primary database.
func seedTestRecipient(ctx context.Context, pool *sql.DB) error {
	const stamp = "2026-01-01T00:00:00Z"
	if _, err := pool.ExecContext(ctx, `INSERT INTO recipients (id, subject, trust_posture, created_at, updated_at) VALUES (?, ?, 'personal', ?, ?)`, hub.TestRecipientID, hub.TestRecipientSubject, stamp, stamp); err != nil {
		return fmt.Errorf("seed test recipient: %w", err)
	}
	if _, err := pool.ExecContext(ctx, `INSERT INTO notifications (id, requested_by, title, body, urgency, sensitivity_label, idempotency_key, created_at, updated_at) VALUES (?, ?, ?, ?, 'normal', 'public', ?, ?, ?)`, hub.TestNotificationID, hub.TestRecipientSubject, "Test fixture", "Disposable unanswered decision request", hub.TestNotificationID, stamp, stamp); err != nil {
		return fmt.Errorf("seed test notification: %w", err)
	}
	if _, err := pool.ExecContext(ctx, `INSERT INTO notification_events (notification_id, state, reason, created_at) VALUES (?, 'pending', 'test fixture accepted', ?)`, hub.TestNotificationID, stamp); err != nil {
		return fmt.Errorf("seed test notification state: %w", err)
	}
	if _, err := pool.ExecContext(ctx, `INSERT INTO asks (id, notification_id, question, allowed_answers, deadline, state, created_at, updated_at, options_json) VALUES (?, ?, ?, ?, ?, 'pending', ?, ?, ?)`, hub.TestAskID, hub.TestNotificationID, "Choose a test-only option", `["continue","stop"]`, "2099-01-01T00:00:00Z", stamp, stamp, `["continue","stop"]`); err != nil {
		return fmt.Errorf("seed test ask: %w", err)
	}
	if err := seedTestDeliveryAndHistory(ctx, pool, stamp); err != nil {
		return err
	}
	return seedTestPreferenceCatalog(ctx, pool, stamp)
}

// testPreferenceSource is the one declared source in the disposable pool's catalog.
const testPreferenceSource = "settings-test-source"

// seedTestPreferenceCatalog declares one source with two categories in the
// disposable pool, so the notification source settings have a catalog to show
// without depending on which scenarios are installed. The owner has no stored
// choices for it, so every category starts at its default.
func seedTestPreferenceCatalog(ctx context.Context, pool *sql.DB, stamp string) error {
	store := preferences.New(database.NewFromPrimary(pool))
	if err := store.UpsertDeclaredSource(ctx, testPreferenceSource, "Test source", "", "", stamp); err != nil {
		return fmt.Errorf("seed test preference source: %w", err)
	}
	for _, category := range []struct{ key, name, description, minUrgency string }{
		{"delivery", "Delivery updates", "Progress on things being delivered.", "normal"},
		{"critical", "Critical notices", "Things that cannot wait.", "critical"},
	} {
		if err := store.UpsertDeclaredCategory(ctx, testPreferenceSource, category.key, category.name, category.description, true, category.minUrgency, stamp); err != nil {
			return fmt.Errorf("seed test preference category %s: %w", category.key, err)
		}
	}
	return nil
}

// seedTestDeliveryAndHistory creates deterministic rows only inside Test Genie’s
// disposable routed pool. The notification events describe current state;
// receipts separately describe adapter acceptance.
func seedTestDeliveryAndHistory(ctx context.Context, pool *sql.DB, stamp string) error {
	const recipient = hub.TestRecipientSubject
	const deliveredID = "e4-test-delivered"
	states := []struct{ id, state string }{
		{deliveredID, "delivered"},
		{"DEMO-N043", "pending"},
		{"e4-test-held", "held"},
		{"e4-test-routed", "routed"},
		{"e4-test-failed", "failed"},
		{"e4-test-unroutable", "unroutable"},
		{"e4-test-suppressed", "suppressed"},
	}
	for index, fixture := range states {
		created := fmt.Sprintf("2026-01-%02dT00:00:00Z", 10-index)
		title := "E4 state fixture: " + fixture.state
		if fixture.id == deliveredID {
			title = "E4 delivery receipt fixture"
		}
		if _, err := pool.ExecContext(ctx, `INSERT INTO notifications (id, requested_by, title, body, urgency, sensitivity_label, idempotency_key, created_at, updated_at) VALUES (?, ?, ?, ?, 'normal', 'public', ?, ?, ?)`, fixture.id, recipient, title, "Disposable E4 delivery state", fixture.id, created, created); err != nil {
			return fmt.Errorf("seed E4 delivery notification %s: %w", fixture.id, err)
		}
		if _, err := pool.ExecContext(ctx, `INSERT INTO notification_events (notification_id, state, reason, created_at) VALUES (?, ?, ?, ?)`, fixture.id, fixture.state, "E4 disposable fixture", created); err != nil {
			return fmt.Errorf("seed E4 delivery state %s: %w", fixture.id, err)
		}
	}
	if _, err := pool.ExecContext(ctx, `INSERT INTO receipts (id, notification_id, channel, machine_id, provider_id, delivered_at) VALUES (?, ?, 'webpush', 'e4-test-machine', 'e4-test-provider', ?)`, "e4-test-receipt", deliveredID, stamp); err != nil {
		return fmt.Errorf("seed E4 delivery receipt: %w", err)
	}
	if _, err := pool.ExecContext(ctx, `INSERT INTO delivery_attempts (notification_id, channel, machine_id, attempt_number, outcome, reason, created_at) VALUES (?, 'webpush', 'e4-test-machine', 1, 'accepted', 'test adapter accepted', ?)`, deliveredID, stamp); err != nil {
		return fmt.Errorf("seed E4 delivery attempt: %w", err)
	}
	const historyID = "e4-test-history"
	if _, err := pool.ExecContext(ctx, `INSERT INTO notifications (id, requested_by, title, body, urgency, sensitivity_label, idempotency_key, created_at, updated_at) VALUES (?, ?, ?, ?, 'normal', 'public', ?, ?, ?)`, historyID, recipient, "E4 answered history fixture", "Disposable closed ask", historyID, stamp, stamp); err != nil {
		return fmt.Errorf("seed E4 history notification: %w", err)
	}
	if _, err := pool.ExecContext(ctx, `INSERT INTO notification_events (notification_id, state, reason, created_at) VALUES (?, 'answered', 'test fixture outcome', ?)`, historyID, stamp); err != nil {
		return fmt.Errorf("seed E4 history notification state: %w", err)
	}
	if _, err := pool.ExecContext(ctx, `INSERT INTO asks (id, notification_id, question, allowed_answers, deadline, state, created_at, updated_at, options_json, source, resolved_at) VALUES (?, ?, 'E4 history answer fixture', '[\"continue\",\"stop\"]', ?, 'answered', ?, ?, '[\"continue\",\"stop\"]', 'notification-hub-e4-test', ?)`, "e4-test-answered-ask", historyID, stamp, stamp, stamp, stamp); err != nil {
		return fmt.Errorf("seed E4 answered history ask: %w", err)
	}
	if _, err := pool.ExecContext(ctx, `INSERT INTO answers (id, ask_id, answer, answered_by, answered_at) VALUES (?, ?, 'continue', ?, ?)`, "e4-test-answer", "e4-test-answered-ask", recipient, stamp); err != nil {
		return fmt.Errorf("seed E4 history answer: %w", err)
	}
	return nil
}

// configureAskDefaultFloor reads the minimum time between the first
// delivered receipt and applying an ask's default (Go duration, default 12h).
// Lowering it is for fixtures such as the delivery canary.
func configureAskDefaultFloor(service *hub.Service) {
	value := strings.TrimSpace(os.Getenv("VROOLI_NOTIFICATION_ASK_DEFAULT_FLOOR"))
	if value == "" {
		return
	}
	floor, err := time.ParseDuration(value)
	if err != nil || floor <= 0 {
		slog.Warn("ignoring invalid VROOLI_NOTIFICATION_ASK_DEFAULT_FLOOR; using the 12h floor", "value", value)
		return
	}
	service.SetAskDefaultFloor(floor)
	slog.Info("ask default floor configured", "floor", floor.String())
}

type pushBudgetSetter interface {
	SetPushBudget(hub.PushBudgetConfig)
	PushBudget() hub.PushBudgetConfig
}

// configurePushBudget tunes the in-memory per-source device-push bucket.
// Notifications and asks remain accepted and stored when it is empty.
func configurePushBudget(service pushBudgetSetter, getenv func(string) string) {
	defaults := hub.DefaultPushBudget
	capacity, refill := strconv.Itoa(defaults.Capacity), defaults.RefillInterval.String()
	if value := strings.TrimSpace(getenv("VROOLI_NOTIFICATION_PUSH_BUCKET_CAPACITY")); value != "" {
		capacity = value
	}
	if value := strings.TrimSpace(getenv("VROOLI_NOTIFICATION_PUSH_BUCKET_REFILL_INTERVAL")); value != "" {
		refill = value
	}
	config, err := hub.ParsePushBudgetConfig(capacity, refill)
	if err != nil {
		slog.Warn("ignoring invalid notification push bucket settings; using defaults", "error", err)
		config = defaults
	}
	service.SetPushBudget(config)
	slog.Info("notification push bucket configured", "config", service.PushBudget().String())
}

func configureWebPush(service *hub.Service, fileRoots *filerouting.RoutedRoots) {
	if service == nil {
		return
	}
	subject := strings.TrimSpace(os.Getenv("VROOLI_WEBPUSH_VAPID_SUBJECT"))
	if subject == "" {
		subject = "mailto:notification-hub@localhost"
	}
	privateValue := strings.TrimSpace(os.Getenv("VROOLI_WEBPUSH_VAPID_PRIVATE_KEY"))
	publicValue := strings.TrimSpace(os.Getenv("VROOLI_WEBPUSH_VAPID_PUBLIC_KEY"))
	if privateValue == "" {
		keyPath, err := fileRootPath(context.Background(), fileRoots, storage.ClassState, "vapid-private-key")
		if err != nil {
			slog.Warn("web push transport disabled", "error", fmt.Errorf("resolve VAPID state path: %w", err))
			return
		}
		privateValue, err = push.LoadOrCreatePrivateKey(keyPath)
		if err != nil {
			slog.Warn("web push transport disabled", "error", err)
			return
		}
	}
	sender, err := push.NewFromValues(privateValue, publicValue, subject)
	if err != nil {
		slog.Warn("web push transport disabled", "error", err)
		return
	}
	service.SetPushSender(pushAdapter{sender: sender})
	service.SetWebPushPublicKey(sender.PublicKeyValue())
}
