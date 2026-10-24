package main

import (
	"context"
	"database/sql"
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
	return a.sender.SendMessage(ctx, push.Subscription{Endpoint: subscription.Endpoint, P256DH: subscription.P256DH, Auth: subscription.Auth, Origin: subscription.Origin}, push.Message{Title: message.Title, Body: message.Body, ID: message.Tag, URL: message.URL, Icon: message.Icon})
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
			if fixture != "" && refererPath != "/asks" {
				pendingByLease[leaseID] = pendingFixture{name: fixture, selectedAt: now}
				fixture = ""
			} else {
				if fixture == "" {
					fixture = pendingByLease[leaseID].name
				}
			}
			pendingMu.Unlock()
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
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
	default:
		return nil, fmt.Errorf("unsupported E4 test fixture %q", fixture)
	}
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
		{"e4-test-pending", "pending"},
		{"e4-test-held", "held"},
		{"e4-test-routed", "routed"},
		{"e4-test-failed", "failed"},
		{"e4-test-unroutable", "unroutable"},
		{"e4-test-suppressed", "suppressed"},
		{"e4-test-unknown", "future-state"},
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
