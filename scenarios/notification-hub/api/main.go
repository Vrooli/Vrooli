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

	"notification-hub/internal/capabilities"
	"notification-hub/internal/hub"
	"notification-hub/internal/integrations"
	"notification-hub/internal/modules"
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
	notificationH "notification-hub/handlers/notifications"
	recipientsH "notification-hub/handlers/recipients"
	routingH "notification-hub/handlers/routing"
)

type pushAdapter struct{ sender *push.Sender }

func (a pushAdapter) Send(ctx context.Context, subscription hub.PushSubscription, title, body string) (string, error) {
	return a.sender.Send(ctx, push.Subscription{Endpoint: subscription.Endpoint, P256DH: subscription.P256DH, Auth: subscription.Auth}, title, body)
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

	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}
	primaryFileRoots, err := scenarioStorageRoots()
	if err != nil {
		log.Fatalf("file storage configuration failed: %v", err)
	}
	fileRoots := filerouting.New(primaryFileRoots)
	service := hub.New(db, schedule.System(), log.Default())
	service.SetDefaultRecipient(strings.TrimSpace(os.Getenv("VROOLI_NOTIFICATION_RECIPIENT")))
	configureWebPush(service, fileRoots)
	posture := "personal"
	if state, postureErr := trustposture.LoadWorkingTree(); postureErr != nil {
		slog.Warn("trust posture unavailable; using fail-safe personal posture", "error", postureErr)
	} else {
		posture = string(state.Posture)
	}
	ownerVerifier := owneridentity.NewClient(owneridentity.Config{Resolver: discovery.NewResolver(discovery.ResolverConfig{})})
	service.SetEmailSender(hub.NewSMTPSenderFromEnvironment(os.Getenv))
	service.SetDesktopSender(hub.NewMacOSDesktopSender())
	service.SetRemoteDelivery(hub.NewBridgeRemoteFromEnvironment())

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.ModuleWithIdentity(db, ownerVerifier, "notification-hub-api", "1.0.0", posture),
		capsH.Module(capabilities.NewRegistry()),
		notificationH.ModuleWithVerifier(service, ownerVerifier),
		recipientsH.ModuleWithVerifier(service, ownerVerifier),
		routingH.ModuleWithVerifier(service, ownerVerifier),
		deliveryH.ModuleWithVerifier(service, ownerVerifier),
		conversationH.ModuleWithVerifier(service, ownerVerifier),
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.RegisterWithFileRoots(rootMux, db, fileRoots)
	rootMux.Handle("/api/v1/integrations/events", integrations.EventWebhook(service, os.Getenv("VROOLI_EVENTS_WEBHOOK_SECRET")))
	if err := integrations.EnsureEventSubscription(context.Background(), os.Getenv("VROOLI_EVENTS_API_BASE"), os.Getenv("VROOLI_NOTIFICATION_EVENTS_WEBHOOK_URL"), os.Getenv("VROOLI_NOTIFICATION_EVENTS_PATTERN")); err != nil {
		slog.Warn("event subscription reconciliation unavailable", "error", err)
	}

	rootMux.Handle("/", srv.Handler())

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
