package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

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
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/filerouting"
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
	msg := push.Message{Title: message.Title, Body: message.Body, ID: message.Tag, URL: message.URL, Icon: message.Icon}
	if message.Navigate != "" {
		return a.sender.SendPayload(ctx, sub, push.DeclarativePayload(msg, message.Navigate))
	}
	return a.sender.SendMessage(ctx, sub, msg)
}

// scenarioStorageRoots resolves all filesystem storage classes once at
// startup. File writers must select their class through fileRootPath so a
// test-mode request uses the lease-owned root instead of the live tree.
func scenarioStorageRoots() (storage.Paths, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return storage.Paths{}, fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("notification-hub")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve notification-hub storage namespace: %w", err)
	}
	return resolver.Resolve(storage.Options{ScenarioID: scenarioID})
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
	service.SetDefaultRecipient(strings.TrimSpace(os.Getenv("VROOLI_NOTIFICATION_RECIPIENT")))
	service.SetRecipientResolver(integrations.OperatorStateRecipient())
	service.SetOwnerFallback(hub.OperatorSubject)
	configureWebPush(service, fileRoots)
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
	ownerVerifier := owneridentity.NewClient(owneridentity.Config{Resolver: scenarioResolver})
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
	if eventsBase, resolveErr := discovery.ResolveScenarioURLDefault(context.Background(), "vrooli-events"); resolveErr == nil {
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
	if err := integrations.EnsureEventSubscription(context.Background(), eventConfig.Get().EventsAPIBase, eventConfig.Get().WebhookURL, eventConfig.Get().Pattern); err != nil {
		slog.Error("vrooli-events subscription reconciliation failed", "error", err)
	} else {
		slog.Info("vrooli-events subscription reconciled", "pattern", eventConfig.Get().Pattern)
	}
	if err := integrations.EnsureDecisionSubscription(context.Background(), eventConfig.Get().EventsAPIBase, eventConfig.Get().WebhookURL); err != nil {
		slog.Error("vrooli-events decision subscription reconciliation failed", "error", err)
	} else {
		slog.Info("vrooli-events decision subscription reconciled", "pattern", integrations.DecisionRequestPattern)
	}
	if err := integrations.EnsureDecisionWithdrawalSubscription(context.Background(), eventConfig.Get().EventsAPIBase, eventConfig.Get().WebhookURL); err != nil {
		slog.Error("vrooli-events decision-withdrawal subscription reconciliation failed", "error", err)
	} else {
		slog.Info("vrooli-events decision-withdrawal subscription reconciled", "pattern", integrations.DecisionWithdrawalPattern)
	}
	if err := integrations.EnsureNotificationSubscription(context.Background(), eventConfig.Get().EventsAPIBase, eventConfig.Get().WebhookURL); err != nil {
		slog.Error("vrooli-events notification subscription reconciliation failed", "error", err)
	} else {
		slog.Info("vrooli-events notification subscription reconciled", "pattern", integrations.NotificationRequestPattern)
	}
	if err := integrations.EnsureHeartbeatAttemptSubscription(context.Background(), eventConfig.Get().EventsAPIBase, eventConfig.Get().WebhookURL); err != nil {
		slog.Error("vrooli-events prompt-manager heartbeat subscription reconciliation failed", "error", err)
	} else {
		slog.Info("vrooli-events prompt-manager heartbeat subscription reconciled", "pattern", integrations.HeartbeatAttemptEventPattern)
	}
	// The webhook verifies with the publisher's own signing secret: the
	// environment override, then the credential authority. The value is
	// never logged.
	webhookSecret := integrations.CachedSecret(func() (string, error) {
		return integrations.ResolveEventsWebhookSecret(os.Getenv)
	}, func(err error) {
		slog.Warn("vrooli-events webhook secret unavailable; signed event deliveries are rejected until it resolves", "error", err)
	})
	slog.Info("vrooli-events webhook secret", "resolved", webhookSecret() != "")
	service.SetResolutionPublisher(integrations.EventsPublisher{BaseURL: func() string { return eventConfig.Get().EventsAPIBase }})
	configureAskDefaultFloor(service)

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.ModuleWithIdentity(db, ownerVerifier, "notification-hub-api", "1.0.0", posture),
		capsH.Module(capabilities.NewRegistry()),
		notificationH.ModuleWithVerifier(service, ownerVerifier),
		recipientsH.ModuleWithVerifier(service, ownerVerifier),
		appDeliveryH.ModuleWithVerifier(service, ownerVerifier),
		preferencesH.ModuleWithRecipientResolver(preferenceStore, service.ResolveRecipient, ownerVerifier),
		routingH.ModuleWithVerifier(service, ownerVerifier),
		deliveryH.ModuleWithVerifier(service, ownerVerifier),
		conversationH.ModuleWithVerifier(service, ownerVerifier),
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
	rootMux.Handle("/api/v1/config/event-integration", integrations.Handler(eventConfig, func(config integrations.LiveConfig) error {
		return integrations.EnsureEventSubscription(context.Background(), config.EventsAPIBase, config.WebhookURL, config.Pattern)
	}))

	// The owner's Cloudflare Access login is verified for the Connect API
	// only. The signed event webhook and the routes above keep their own
	// authentication, and Wait stays readable without an owner.
	ownerAuth := internalidentity.OwnerAuthenticator{
		Providers:    internalidentity.OwnerProvidersFromEnvironment(os.Getenv, scenarioResolver, "notification-hub"),
		OwnerSubject: service.ResolveRecipient,
	}
	rootMux.Handle("/", ownerAuth.Middleware(srv.Handler()))

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// request context so *database.RoutedDB routes the call to the
	// installed test pool. Self-disables in production mode.
	handler := apihttp.TestModeMiddleware(requireRoutedTestLease(db, rootMux))

	if err := apiserver.Run(apiserver.Config{
		Handler: handler,
		Cleanup: func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// requireRoutedTestLease prevents a test-mode request from reaching a handler
// unless it can be pinned to the currently installed lease. RoutedDB normally
// preserves production behavior by falling back to Primary when no test pool
// exists; browser test-mode requests must instead fail closed. Pinning closes
// the race between checking the lease and the handler's first database call.
func requireRoutedTestLease(db *database.RoutedDB, next http.Handler) http.Handler {
	preferenceStore := preferences.New(db)
	type pendingFixture struct {
		name       string
		selectedAt time.Time
	}
	type e22FixtureState struct {
		initialized                  bool
		initializedCases             map[string]bool
		canonicalOrderInstalled      bool
		singleOrderInstalled         bool
		partialFixtureInstalled      bool
		partialWriteCount            int
		partialReadFailuresRemaining int
		mixedQuietFixtureRestored    bool
		unknownReadArmed             bool
		unknownOrderReadArmed        bool
		unknownOrderFaultSpent       bool
		channelReadArmed             bool
		channelReadFaultSpent        bool
		deviceReadFailuresRemaining  int
		deviceReadFaultInitialized   bool
	}
	var pendingMu sync.Mutex
	pendingByLease := make(map[string]pendingFixture)
	var e22FixtureMu sync.Mutex
	e22FixtureByLease := make(map[string]*e22FixtureState)
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
		if e21ReachEmptyFixtureSelected(r) {
			// The routed test pool may retain rows from an earlier validation lease.
			// Clear only this disposable owner so the selected empty Reach state is
			// deterministic without touching primary storage.
			if err := clearE21ReachEmptyFixture(ctx, db); err != nil {
				http.Error(w, "E21 Reach empty fixture setup failed", http.StatusInternalServerError)
				return
			}
			ctx = internalidentity.WithLeasedTestOwner(ctx, hub.TestRecipientSubject)
		}
		if e22SettingsFixtureSelected(r) {
			e22FixtureMu.Lock()
			state := e22FixtureByLease[leaseID]
			if state == nil {
				state = &e22FixtureState{initializedCases: make(map[string]bool)}
				e22FixtureByLease[leaseID] = state
			}
			caseID := e22FixtureCase(r)
			needsSeed := !state.initialized
			if caseID != "" && !state.initializedCases[caseID] {
				needsSeed = true
			}
			if needsSeed {
				err = seedE22SettingsFixture(ctx, db)
				if err == nil && caseID == "e21-device-removal-review-readback" {
					err = seedE21DeviceRemovalReviewFixture(ctx, db)
				}
				if err == nil && e21WebPushContentAccessFixtureSelected(r) {
					err = seedE21WebPushContentAccessFixture(ctx, db)
					if err == nil && e21ChannelReadGapFixtureSelected(r) {
						_, err = db.ExecContext(ctx, `DELETE FROM channel_addresses WHERE device_id='device-demo-work' AND channel <> 'web_push'`)
					}
				}
				if err == nil && (caseID == "e22-quiet-timezone-change-dark-phone" || caseID == "e22-quiet-timezone-change-light-phone") {
					err = seedE22SelectedTimezoneChangeFixture(ctx, db)
				}
				if err == nil && caseID == "e43-preferences" {
					err = seedE43PreferencesFixture(ctx, db, preferenceStore)
				}
				if err == nil && (caseID == "e22-quiet-mixed-timezones-dark-phone" || caseID == "e22-quiet-mixed-timezones-light-phone") {
					err = seedE22SelectedMixedZoneFixture(ctx, db)
				}
				if err == nil && caseID == "e21-channel-create-unknown-readback" {
					_, err = db.ExecContext(ctx, `DELETE FROM channel_addresses WHERE id='e22-settings-email'`)
				}
				if err == nil && caseID == "e21-channel-address-validation-current" {
					_, err = db.ExecContext(ctx, `UPDATE channel_addresses SET approved_labels='["public"]' WHERE id='e22-settings-email'`)
					if err == nil {
						_, err = db.ExecContext(ctx, `UPDATE devices SET name='device-demo-email', machine_id='' WHERE id='e22-settings-device'`)
					}
				}
				if err == nil {
					state.initialized = true
					if caseID != "" {
						state.initializedCases[caseID] = true
						state.canonicalOrderInstalled = false
						state.singleOrderInstalled = false
						state.partialFixtureInstalled = false
						state.partialWriteCount = 0
						state.partialReadFailuresRemaining = 0
						state.mixedQuietFixtureRestored = false
						if caseID == "e22-quiet-timezone-change-dark-phone" || caseID == "e22-quiet-timezone-change-light-phone" {
							// Preserve the selected UTC baseline when the same case arms
							// unknown-readback; its generic recovery fixture is a different scene.
							state.mixedQuietFixtureRestored = true
						}
						state.unknownReadArmed = false
						state.unknownOrderReadArmed = false
						state.unknownOrderFaultSpent = false
						state.channelReadArmed = false
						state.channelReadFaultSpent = false
						state.deviceReadFailuresRemaining = 0
						state.deviceReadFaultInitialized = false
					}
				}
			}
			if err == nil && e22FixtureOrder(r) == "canonical" && !state.canonicalOrderInstalled {
				err = seedE22CanonicalOrder(ctx, db)
				if err == nil {
					state.canonicalOrderInstalled = true
				}
			}
			if err == nil && e22FixtureOrder(r) == "single" && !state.singleOrderInstalled {
				err = seedE22SingleChannelOrder(ctx, db)
				if err == nil {
					state.singleOrderInstalled = true
				}
			}
			e22FixtureMu.Unlock()
			if err != nil {
				http.Error(w, "E22 settings fixture setup failed", http.StatusInternalServerError)
				return
			}
			ctx = internalidentity.WithLeasedTestOwner(ctx, hub.TestRecipientSubject)
			fault := e22FixtureFault(r)
			if fault == "device-read-failure" && e21DeviceReadFailureFixtureSelected(r) && strings.HasSuffix(r.URL.Path, "/ListDevices") {
				e22FixtureMu.Lock()
				if !state.deviceReadFaultInitialized {
					// React Query's default policy makes five attempts. Fail each one
					// inside this leased fixture so the error state is observable;
					// the explicit user retry then reads the seeded disposable rows.
					state.deviceReadFaultInitialized = true
					state.deviceReadFailuresRemaining = 5
				}
				fail := state.deviceReadFailuresRemaining > 0
				if fail {
					state.deviceReadFailuresRemaining--
				}
				e22FixtureMu.Unlock()
				if fail {
					http.Error(w, "E21 routed fixture injected a device-list read failure", http.StatusServiceUnavailable)
					return
				}
			}
			if fault == "channel-address-read-failure" && e21ChannelReadGapFixtureSelected(r) && strings.HasSuffix(r.URL.Path, "/ListChannelAddresses") {
				http.Error(w, "E21 routed fixture injected a channel-address read failure", http.StatusServiceUnavailable)
				return
			}
			if fault == "partial" {
				e22FixtureMu.Lock()
				if !state.partialFixtureInstalled {
					err = seedE22PartialQuietFixture(ctx, db)
					if err == nil {
						state.partialFixtureInstalled = true
					}
				}
				e22FixtureMu.Unlock()
				if err != nil {
					http.Error(w, "E22 partial quiet-hours fixture setup failed", http.StatusInternalServerError)
					return
				}
			}
			if fault == "unknown-readback" {
				e22FixtureMu.Lock()
				if !state.mixedQuietFixtureRestored {
					err = seedE22MixedQuietFixture(ctx, db)
					if err == nil {
						state.mixedQuietFixtureRestored = true
					}
				}
				e22FixtureMu.Unlock()
				if err != nil {
					http.Error(w, "E22 mixed quiet-hours fixture restore failed", http.StatusInternalServerError)
					return
				}
			}
			if fault == "partial" && strings.HasSuffix(r.URL.Path, "/SetQuietWindow") {
				e22FixtureMu.Lock()
				state.partialWriteCount++
				if state.partialWriteCount == 3 {
					state.partialReadFailuresRemaining = 2
				}
				e22FixtureMu.Unlock()
			}
			if fault == "partial" && strings.HasSuffix(r.URL.Path, "/ListQuietWindows") {
				e22FixtureMu.Lock()
				fail := state.partialReadFailuresRemaining > 0
				if fail {
					state.partialReadFailuresRemaining--
				}
				e22FixtureMu.Unlock()
				if fail {
					http.Error(w, "E22 routed fixture injected one weekday confirming-read failure", http.StatusServiceUnavailable)
					return
				}
			}
			if fault == "unknown-readback" && strings.HasSuffix(r.URL.Path, "/SetQuietWindow") {
				e22FixtureMu.Lock()
				state.unknownReadArmed = true
				e22FixtureMu.Unlock()
			}
			if fault == "unknown-readback" && strings.HasSuffix(r.URL.Path, "/ListQuietWindows") {
				e22FixtureMu.Lock()
				fail := state.unknownReadArmed
				state.unknownReadArmed = false
				e22FixtureMu.Unlock()
				if fail {
					http.Error(w, "E22 routed fixture injected one confirming read failure", http.StatusServiceUnavailable)
					return
				}
			}
			if fault == "unknown-order-readback" && strings.HasSuffix(r.URL.Path, "/SetEscalationChain") {
				e22FixtureMu.Lock()
				if !state.unknownOrderFaultSpent {
					state.unknownOrderReadArmed = true
				}
				e22FixtureMu.Unlock()
			}
			if fault == "unknown-order-readback" && strings.HasSuffix(r.URL.Path, "/GetEscalationChain") {
				e22FixtureMu.Lock()
				fail := state.unknownOrderReadArmed
				state.unknownOrderReadArmed = false
				if fail {
					state.unknownOrderFaultSpent = true
				}
				e22FixtureMu.Unlock()
				if fail {
					http.Error(w, "E22 routed fixture injected one escalation-order confirming read failure", http.StatusServiceUnavailable)
					return
				}
			}
			if fault == "channel-unknown-readback" && (strings.HasSuffix(r.URL.Path, "/UpdateChannelAddress") || strings.HasSuffix(r.URL.Path, "/CreateChannelAddress")) {
				e22FixtureMu.Lock()
				if !state.channelReadFaultSpent {
					state.channelReadArmed = true
				}
				e22FixtureMu.Unlock()
			}
			if fault == "channel-unknown-readback" && strings.HasSuffix(r.URL.Path, "/ListChannelAddresses") {
				e22FixtureMu.Lock()
				fail := state.channelReadArmed
				if fail {
					state.channelReadArmed = false
					state.channelReadFaultSpent = true
				}
				e22FixtureMu.Unlock()
				if fail {
					http.Error(w, "E22 routed fixture injected one channel-address confirming read failure", http.StatusServiceUnavailable)
					return
				}
			}
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// e21ReachEmptyFixtureSelected selects the empty Reach owner state only from
// an explicit page query carried by a request inside a pinned test lease.
func e21ReachEmptyFixtureSelected(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.Header.Get("X-Notification-Hub-E21-Fixture") == "reach-empty" {
		return true
	}
	if r.URL.Query().Get("e21_fixture") == "reach-empty" {
		return true
	}
	page, err := url.Parse(r.Referer())
	return err == nil && page.Query().Get("e21_fixture") == "reach-empty"
}

// e21WebPushContentAccessFixtureSelected pins the selected Web Push editor
// baseline to a disposable owner run without changing the shared E22 fixture.
func e21WebPushContentAccessFixtureSelected(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.Header.Get("X-Notification-Hub-E21-Fixture") == "webpush-content-access" || r.URL.Query().Get("e21_fixture") == "webpush-content-access" {
		return true
	}
	page, err := url.Parse(r.Referer())
	return err == nil && page.Query().Get("e21_fixture") == "webpush-content-access"
}

func clearE21ReachEmptyFixture(ctx context.Context, db *database.RoutedDB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin E21 Reach empty fixture reset: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM channel_addresses WHERE device_id IN (SELECT d.id FROM devices d JOIN recipients r ON r.id=d.recipient_id WHERE r.subject=?)`, hub.TestRecipientSubject); err != nil {
		return fmt.Errorf("clear E21 Reach fixture channel addresses: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM devices WHERE recipient_id IN (SELECT id FROM recipients WHERE subject=?)`, hub.TestRecipientSubject); err != nil {
		return fmt.Errorf("clear E21 Reach fixture devices: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit E21 Reach empty fixture reset: %w", err)
	}
	return nil
}

// e22SettingsFixtureSelected honors an explicit fixture selector carried by
// the page URL. The caller reaches this helper only after TestModeMiddleware
// confirmed test mode and the active lease was pinned above; the browser's
// same-origin API requests carry the page URL in Referer.
func e22SettingsFixtureSelected(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.Header.Get("X-Notification-Hub-E22-Fixture") == "settings-owner" {
		return true
	}
	// The selected Reach empty state may preserve the Settings fixture query
	// while navigating between routes. It is a more specific readback fixture:
	// do not reseed Settings devices after its empty-owner reset.
	if e21ReachEmptyFixtureSelected(r) {
		return false
	}
	if r.URL.Query().Get("e22_fixture") == "settings-owner" {
		return true
	}
	page, err := url.Parse(r.Referer())
	return err == nil && page.Query().Get("e22_fixture") == "settings-owner"
}

func e22FixtureFault(r *http.Request) string {
	if r == nil {
		return ""
	}
	page, err := url.Parse(r.Referer())
	fault := ""
	if err == nil {
		fault = page.Query().Get("e22_fault")
		if fault == "device-read-failure" && page.Path != "/settings/reach" {
			fault = ""
		}
	}
	switch fault {
	case "partial", "unknown-readback", "unknown-order-readback", "channel-unknown-readback", "channel-address-read-failure", "device-read-failure":
		return fault
	}
	// Cross-origin browsers may reduce Referer to an origin. This one fault is
	// forwarded explicitly only in routed test mode and only for its allowlisted
	// E21 case; ordinary callers cannot request an injected response.
	if r.Header.Get("X-Notification-Hub-E22-Fault") == "device-read-failure" && e21DeviceReadFailureFixtureSelected(r) {
		return "device-read-failure"
	}
	return ""
}

func e21DeviceReadFailureFixtureSelected(r *http.Request) bool {
	if r == nil {
		return false
	}
	switch e22FixtureCase(r) {
	case "e21-device-read-failure-light-phone", "e21-device-read-failure-dark-phone":
		return true
	default:
		return false
	}
}

func e22FixtureOrder(r *http.Request) string {
	if r == nil {
		return ""
	}
	query := r.URL.Query()
	if query.Get("e22_order") != "" {
		return query.Get("e22_order")
	}
	page, err := url.Parse(r.Referer())
	if err != nil {
		return ""
	}
	if page.Query().Get("e22_fixture") != "settings-owner" {
		return ""
	}
	return page.Query().Get("e22_order")
}

// e22FixtureCase resets the selected disposable baseline once per allowlisted
// quiet-editor case. Workflow Health executes several cases under one lease,
// so lease-only initialization would let one case's save suppress the next
// case's review state. This selector is only honored after test mode and the
// pinned lease have been validated by the caller.
func e22FixtureCase(r *http.Request) string {
	if r == nil {
		return ""
	}
	query := r.URL.Query()
	if query.Get("e22_fixture") == "" && r.Header.Get("X-Notification-Hub-E22-Fixture") == "settings-owner" {
		query.Set("e22_fixture", "settings-owner")
	}
	if query.Get("e22_case") == "" {
		if selected := r.Header.Get("X-Notification-Hub-E22-Case"); selected != "" {
			query.Set("e22_case", selected)
		}
	}
	if query.Get("e22_case") == "" {
		page, err := url.Parse(r.Referer())
		if err == nil {
			query = page.Query()
		}
	}
	switch query.Get("e22_case") {
	case "e22-quiet-remove-monday-dark-phone", "e22-quiet-remove-monday-light-phone", "e22-quiet-monday-editor-dark-phone", "e22-quiet-monday-editor-dark-narrow", "e22-quiet-monday-editor-desktop", "e22-quiet-monday-editor-light-desktop", "e22-quiet-monday-editor-light-narrow", "e22-quiet-monday-editor-light-phone", "e22-quiet-japanese-editor-dark-phone", "e22-quiet-japanese-editor-light-phone", "e22-quiet-validation-states-dark-phone", "e22-quiet-validation-states-light-phone", "e22-quiet-mixed-timezones-dark-phone", "e22-quiet-mixed-timezones-light-phone", "e22-quiet-timezone-change-dark-phone", "e22-quiet-timezone-change-light-phone", "e22-order-add-channel-dark-phone", "e22-order-add-channel-light-phone", "e22-order-channel-editor-dark-desktop", "e22-order-channel-editor-dark-narrow-p1", "e22-order-channel-editor-dark-narrow-p2", "e22-order-channel-editor-dark-phone", "e22-order-channel-editor-light-desktop", "e22-order-channel-editor-light-narrow-p1", "e22-order-channel-editor-light-narrow-p2", "e22-order-channel-editor-light-phone", "e22-order-empty-draft-light-phone", "e22-settings-owner-dark-journey", "e22-settings-owner-light-journey", "e21-channel-address-readback", "e21-reach-channel-read-gap-light-phone", "e21-reach-channel-read-gap-dark-phone", "e21-reach-channel-read-gap-dark-narrow", "e21-reach-channel-read-gap-light-narrow", "e21-reach-channel-read-gap-dark-desktop", "e21-reach-channel-read-gap-light-desktop", "e21-webpush-content-access", "e21-channel-address-validation-current", "e21-channel-unknown-readback", "e21-channel-create-unknown-readback", "e21-device-removal-review-readback", "e21-device-read-failure-light-phone", "e21-device-read-failure-dark-phone", "e21-permission-denied-dark-phone", "e21-permission-denied-light-phone", "e43-preferences":
		if query.Get("e22_fixture") == "settings-owner" {
			return query.Get("e22_case")
		}
	}
	return ""
}

// e21ChannelReadGapFixtureSelected keeps the synthetic read failure scoped to
// the selected Reach read-gap visual family. It is only called inside the
// routed disposable test lease after e22FixtureCase has validated the owner
// fixture and exact case allowlist.
func e21ChannelReadGapFixtureSelected(r *http.Request) bool {
	switch e22FixtureCase(r) {
	case "e21-reach-channel-read-gap-light-phone", "e21-reach-channel-read-gap-dark-phone", "e21-reach-channel-read-gap-dark-narrow", "e21-reach-channel-read-gap-light-narrow", "e21-reach-channel-read-gap-dark-desktop", "e21-reach-channel-read-gap-light-desktop":
		return true
	default:
		return false
	}
}

// seedE22SettingsFixture installs deterministic owner settings only after the
// routed lease has been pinned. Its writes remain in that disposable pool for
// the journey, so later API reads and refreshes observe the actual mutations.
func seedE22SettingsFixture(ctx context.Context, db *database.RoutedDB) error {
	const stamp = "2026-01-01T00:00:00Z"
	if _, err := db.ExecContext(ctx, `DELETE FROM channel_addresses WHERE device_id='demo-phone'`); err != nil {
		return fmt.Errorf("clear prior E21 removal-result channel: %w", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM devices WHERE id='demo-phone'`); err != nil {
		return fmt.Errorf("clear prior E21 removal-result device: %w", err)
	}
	// A prior removal-review journey uses the atlas device ID; clear that
	// selected-only row before restoring the common Settings fixture.
	if _, err := db.ExecContext(ctx, `DELETE FROM channel_addresses WHERE device_id='device-demo-work'`); err != nil {
		return fmt.Errorf("clear prior E21 removal-review channel: %w", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM devices WHERE id='device-demo-work'`); err != nil {
		return fmt.Errorf("clear prior E21 removal-review device: %w", err)
	}
	var recipientID string
	if err := db.QueryRowContext(ctx, `SELECT id FROM recipients WHERE subject=?`, hub.TestRecipientSubject).Scan(&recipientID); err != nil {
		return fmt.Errorf("find E22 test recipient: %w", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO devices (id, recipient_id, name, machine_id, created_at) VALUES ('e22-settings-device', ?, 'E22 settings fixture device', 'e22-fixture-machine', ?) ON CONFLICT(id) DO UPDATE SET name=excluded.name, machine_id=excluded.machine_id`, recipientID, stamp); err != nil {
		return fmt.Errorf("seed E22 test device: %w", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM channel_addresses WHERE device_id='e22-settings-device'`); err != nil {
		return fmt.Errorf("reset E22 channel-address fixture: %w", err)
	}
	for _, channel := range []string{"email", "webhook", "browser", "web_push"} {
		address := "fixture://" + channel
		if channel == "email" {
			// A reserved example address lets the Reach editor validate normally
			// without referring to a real destination.
			address = "owner@example.test"
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO channel_addresses (id, device_id, channel, address, approved_labels, created_at) VALUES (?, 'e22-settings-device', ?, ?, '["public","private","sensitive","critical"]', ?)`, "e22-settings-"+channel, channel, address, stamp); err != nil {
			return fmt.Errorf("seed E22 test channel %s: %w", channel, err)
		}
	}
	// The same disposable pool can be reused across separate validation leases.
	// Restore the selected baseline once when each lease first chooses this
	// fixture, so a prior lease's quiet/order edits cannot make the next journey
	// appear unchanged. Requests within the current lease remain untouched.
	if err := replaceE22QuietWindows(ctx, db, []e22QuietFixtureRow{
		{id: "e22-quiet-monday", weekday: 1, start: "21:30", end: "23:00", timezone: "America/New_York"},
		{id: "e22-quiet-friday", weekday: 5, start: "20:00", end: "21:00", timezone: "Asia/Tokyo"},
	}); err != nil {
		return fmt.Errorf("reset E22 quiet-window fixture: %w", err)
	}
	if err := replaceE22EscalationOrder(ctx, db, []string{"web_push", "x-legacy-route"}); err != nil {
		return fmt.Errorf("reset E22 escalation-order fixture: %w", err)
	}
	return nil
}

// seedE21DeviceRemovalReviewFixture mirrors the selected Reach review data in
// the disposable owner pool, so its title, identity, channel and responsive
// composition can be compared without touching a live recipient.
func seedE21DeviceRemovalReviewFixture(ctx context.Context, db *database.RoutedDB) error {
	const stamp = "2026-01-01T00:00:00Z"
	if _, err := db.ExecContext(ctx, `DELETE FROM channel_addresses WHERE device_id='demo-phone'`); err != nil {
		return fmt.Errorf("reset E21 remaining-device channel fixture: %w", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM devices WHERE id='demo-phone'`); err != nil {
		return fmt.Errorf("reset E21 remaining-device fixture: %w", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM channel_addresses WHERE device_id='e22-settings-device'`); err != nil {
		return fmt.Errorf("reset E21 removal-review channel fixture: %w", err)
	}
	var recipientID string
	if err := db.QueryRowContext(ctx, `SELECT id FROM recipients WHERE subject=?`, hub.TestRecipientSubject).Scan(&recipientID); err != nil {
		return fmt.Errorf("find E21 test recipient: %w", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE devices SET id='device-demo-work', name='Work browser', machine_id='demo-workstation' WHERE id='e22-settings-device'`); err != nil {
		return fmt.Errorf("set selected E21 removal-review device identity: %w", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO channel_addresses (id, device_id, channel, address, approved_labels, created_at) VALUES ('e21-device-demo-work-web-push', 'device-demo-work', 'web_push', 'fixture://web_push', '["public","private","sensitive","critical"]', ?)`, stamp); err != nil {
		return fmt.Errorf("seed selected E21 removal-review channel: %w", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO devices (id, recipient_id, name, machine_id, created_at) VALUES ('demo-phone', ?, 'Travel phone', 'demo-mobile', ?)`, recipientID, stamp); err != nil {
		return fmt.Errorf("seed selected E21 remaining device: %w", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO channel_addresses (id, device_id, channel, address, approved_labels, created_at) VALUES ('e21-demo-phone-web-push', 'demo-phone', 'web_push', 'fixture://web_push', '["public","private"]', ?)`, stamp); err != nil {
		return fmt.Errorf("seed selected E21 remaining-device channel: %w", err)
	}
	return nil
}

// seedE21WebPushContentAccessFixture mirrors the selected Web Push editor's
// device identity and saved labels in the disposable owner pool. The unknown
// label is intentional: editing known labels must preserve it unchanged.
func seedE21WebPushContentAccessFixture(ctx context.Context, db *database.RoutedDB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin selected E21 Web Push fixture: %w", err)
	}
	defer tx.Rollback()
	type channelFixture struct {
		id, channel, address, labels, createdAt string
	}
	rows, err := tx.QueryContext(ctx, `SELECT id, channel, address, approved_labels, created_at FROM channel_addresses WHERE device_id=? ORDER BY id`, "e22-settings-device")
	if err != nil {
		return fmt.Errorf("read E22 channels before selected E21 Web Push identity change: %w", err)
	}
	var channels []channelFixture
	for rows.Next() {
		var channel channelFixture
		if err := rows.Scan(&channel.id, &channel.channel, &channel.address, &channel.labels, &channel.createdAt); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scan E22 channel before selected E21 Web Push identity change: %w", err)
		}
		channels = append(channels, channel)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("iterate E22 channels before selected E21 Web Push identity change: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close E22 channel read before selected E21 Web Push identity change: %w", err)
	}
	if len(channels) == 0 {
		return errors.New("E22 settings fixture has no channel addresses for the selected Work browser")
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM channel_addresses WHERE device_id=?`, "e22-settings-device"); err != nil {
		return fmt.Errorf("temporarily detach E22 channels for selected E21 Web Push identity change: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE devices SET id='device-demo-work', name='Work browser', machine_id='demo-workstation' WHERE id='e22-settings-device'`); err != nil {
		return fmt.Errorf("set selected E21 Web Push device identity: %w", err)
	}
	for _, channel := range channels {
		if channel.id == "e22-settings-web_push" {
			channel.labels = `["public","private","team-internal"]`
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO channel_addresses (id, device_id, channel, address, approved_labels, created_at) VALUES (?, 'device-demo-work', ?, ?, ?, ?)`, channel.id, channel.channel, channel.address, channel.labels, channel.createdAt); err != nil {
			return fmt.Errorf("restore selected E21 Web Push channel %s: %w", channel.channel, err)
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE push_subscriptions SET device_id='device-demo-work' WHERE device_id='e22-settings-device'`); err != nil {
		return fmt.Errorf("move selected E21 Web Push subscriptions with device identity: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit selected E21 Web Push fixture: %w", err)
	}
	return nil
}

func seedE43PreferencesFixture(ctx context.Context, db *database.RoutedDB, store *preferences.Storage) error {
	const source = "e43-fixture-source"
	const stamp = "2026-01-01T00:00:00Z"
	if err := store.UpsertDeclaredSource(ctx, source, "E43 Fixture Source", "", "", stamp); err != nil {
		return fmt.Errorf("seed E43 preference source: %w", err)
	}
	for _, category := range []struct {
		key, name, description, minUrgency string
	}{
		{key: "delivery", name: "Delivery updates", description: "A disposable source update.", minUrgency: "normal"},
		{key: "critical", name: "Critical notices", description: "A disposable critical notice.", minUrgency: "critical"},
	} {
		if err := store.UpsertDeclaredCategory(ctx, source, category.key, category.name, category.description, true, category.minUrgency, stamp); err != nil {
			return fmt.Errorf("seed E43 preference category %s: %w", category.key, err)
		}
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM notification_preferences WHERE recipient_key=? AND source_scenario=?`, hub.TestRecipientSubject, source); err != nil {
		return fmt.Errorf("reset E43 owner preference baseline: %w", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM notification_scope_preferences WHERE recipient_key=? AND source_scenario=?`, hub.TestRecipientSubject, source); err != nil {
		return fmt.Errorf("reset E43 scoped preference baseline: %w", err)
	}
	return nil
}

func replaceE22EscalationOrder(ctx context.Context, db *database.RoutedDB, channels []string) error {
	var recipientID string
	if err := db.QueryRowContext(ctx, `SELECT id FROM recipients WHERE subject=?`, hub.TestRecipientSubject).Scan(&recipientID); err != nil {
		return fmt.Errorf("find E22 test recipient for escalation-order reset: %w", err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin E22 escalation-order fixture reset: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM escalation_chains WHERE recipient_id=?`, recipientID); err != nil {
		return fmt.Errorf("clear E22 escalation-order fixture: %w", err)
	}
	for ordinal, channel := range channels {
		if _, err := tx.ExecContext(ctx, `INSERT INTO escalation_chains (id, recipient_id, ordinal, channel) VALUES (?, ?, ?, ?)`, fmt.Sprintf("e22-order-%d", ordinal+1), recipientID, ordinal+1, channel); err != nil {
			return fmt.Errorf("seed E22 escalation channel %s: %w", channel, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit E22 escalation-order fixture: %w", err)
	}
	return nil
}

type e22QuietFixtureRow struct {
	id, start, end, timezone string
	weekday                  int
}

// seedE22PartialQuietFixture replaces only the leased owner's quiet windows
// with a five-day baseline for the per-day uncertain-save journey.
func seedE22PartialQuietFixture(ctx context.Context, db *database.RoutedDB) error {
	rows := make([]e22QuietFixtureRow, 0, 5)
	for weekday := 1; weekday <= 5; weekday++ {
		rows = append(rows, e22QuietFixtureRow{
			id:       fmt.Sprintf("e22-partial-quiet-%d", weekday),
			weekday:  weekday,
			start:    "12:00",
			end:      "13:00",
			timezone: "Etc/UTC",
		})
	}
	return replaceE22QuietWindows(ctx, db, rows)
}

// seedE22MixedQuietFixture restores the supported mixed-zone baseline before
// later removal and readback journeys in the same disposable lease.
func seedE22MixedQuietFixture(ctx context.Context, db *database.RoutedDB) error {
	return replaceE22QuietWindows(ctx, db, []e22QuietFixtureRow{
		{id: "e22-quiet-monday", weekday: 1, start: "21:30", end: "23:00", timezone: "America/New_York"},
		{id: "e22-quiet-friday", weekday: 5, start: "20:00", end: "21:00", timezone: "Asia/Tokyo"},
	})
}

// seedE22SelectedMixedZoneFixture supplies the exact harmless synthetic values
// depicted by the selected mixed-timezone visual state. It is only selected
// through the allowlisted E22 owner fixture after the routed lease is pinned.
func seedE22SelectedMixedZoneFixture(ctx context.Context, db *database.RoutedDB) error {
	rows := make([]e22QuietFixtureRow, 0, 5)
	for weekday := 1; weekday <= 5; weekday++ {
		zone := "Etc/UTC"
		if weekday == 1 {
			zone = "Europe/London"
		}
		rows = append(rows, e22QuietFixtureRow{
			id:       fmt.Sprintf("e22-visual-mixed-%d", weekday),
			weekday:  weekday,
			start:    "13:00",
			end:      "14:00",
			timezone: zone,
		})
	}
	return replaceE22QuietWindows(ctx, db, rows)
}

// seedE22SelectedTimezoneChangeFixture preserves the source scene's exact
// pre-change Monday window while keeping another weekday in its own zone.
func seedE22SelectedTimezoneChangeFixture(ctx context.Context, db *database.RoutedDB) error {
	return replaceE22QuietWindows(ctx, db, []e22QuietFixtureRow{
		{id: "e22-timezone-monday", weekday: 1, start: "13:00", end: "14:00", timezone: "Etc/UTC"},
		{id: "e22-timezone-friday", weekday: 5, start: "20:00", end: "21:00", timezone: "Asia/Tokyo"},
	})
}

func replaceE22QuietWindows(ctx context.Context, db *database.RoutedDB, rows []e22QuietFixtureRow) error {
	var recipientID string
	if err := db.QueryRowContext(ctx, `SELECT id FROM recipients WHERE subject=?`, hub.TestRecipientSubject).Scan(&recipientID); err != nil {
		return fmt.Errorf("find E22 test recipient for quiet-window reset: %w", err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin E22 quiet-window fixture reset: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM quiet_windows WHERE recipient_id=?`, recipientID); err != nil {
		return fmt.Errorf("clear E22 quiet-window fixture: %w", err)
	}
	for _, row := range rows {
		if _, err := tx.ExecContext(ctx, `INSERT INTO quiet_windows (id, recipient_id, weekday, start_time, end_time, timezone, critical_override) VALUES (?, ?, ?, ?, ?, ?, 0)`, row.id, recipientID, row.weekday, row.start, row.end, row.timezone); err != nil {
			return fmt.Errorf("seed E22 quiet window %s: %w", row.id, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit E22 quiet-window fixture: %w", err)
	}
	return nil
}

// seedE22CanonicalOrder switches the leased fixture to the two known channels
// used in the selected order editor and unknown-save references. The selector
// is consumed once per lease after pinning, so later UI writes remain intact.
func seedE22CanonicalOrder(ctx context.Context, db *database.RoutedDB) error {
	var recipientID string
	if err := db.QueryRowContext(ctx, `SELECT id FROM recipients WHERE subject=?`, hub.TestRecipientSubject).Scan(&recipientID); err != nil {
		return fmt.Errorf("find E22 test recipient for canonical order: %w", err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin E22 canonical order fixture: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM escalation_chains WHERE recipient_id=?`, recipientID); err != nil {
		return fmt.Errorf("clear E22 escalation order for canonical fixture: %w", err)
	}
	for ordinal, channel := range []string{"web_push", "email"} {
		if _, err := tx.ExecContext(ctx, `INSERT INTO escalation_chains (id, recipient_id, ordinal, channel) VALUES (?, ?, ?, ?)`, fmt.Sprintf("e22-canonical-order-%d", ordinal+1), recipientID, ordinal+1, channel); err != nil {
			return fmt.Errorf("seed E22 canonical escalation channel %s: %w", channel, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit E22 canonical order fixture: %w", err)
	}
	return nil
}

// seedE22SingleChannelOrder installs the one-known-channel order used by the
// selected Add channel screen, where Email is the remaining available option.
// Like the canonical order fixture, this is consumed once per routed lease.
func seedE22SingleChannelOrder(ctx context.Context, db *database.RoutedDB) error {
	var recipientID string
	if err := db.QueryRowContext(ctx, `SELECT id FROM recipients WHERE subject=?`, hub.TestRecipientSubject).Scan(&recipientID); err != nil {
		return fmt.Errorf("find E22 test recipient for single-channel order: %w", err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin E22 single-channel order fixture: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM channel_addresses WHERE device_id='e22-settings-device' AND channel NOT IN ('web_push', 'email')`); err != nil {
		return fmt.Errorf("limit E22 single-channel fixture choices: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM escalation_chains WHERE recipient_id=?`, recipientID); err != nil {
		return fmt.Errorf("clear E22 escalation order for single-channel fixture: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO escalation_chains (id, recipient_id, ordinal, channel) VALUES (?, ?, ?, ?)`, "e22-single-order-1", recipientID, 1, "web_push"); err != nil {
		return fmt.Errorf("seed E22 single-channel escalation order: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit E22 single-channel order fixture: %w", err)
	}
	return nil
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
