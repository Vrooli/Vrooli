package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"persona/internal/access"
	"persona/internal/accounts"
	"persona/internal/capabilities"
	"persona/internal/channels"
	"persona/internal/documents"
	"persona/internal/handoffs"
	"persona/internal/journal"
	"persona/internal/modules"
	"persona/internal/personas"
	"persona/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	accessH "persona/handlers/access"
	accountsH "persona/handlers/accounts"
	capsH "persona/handlers/capabilities"
	channelsH "persona/handlers/channels"
	documentsH "persona/handlers/documents"
	handoffsH "persona/handlers/handoffs"
	healthH "persona/handlers/health"
	journalH "persona/handlers/journal"
	personasH "persona/handlers/personas"
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
	scenarioID, err := storage.ScenarioNamespace("persona")
	if err != nil {
		return "", fmt.Errorf("resolve persona storage namespace: %w", err)
	}
	path, err := resolver.Path(
		storage.Options{ScenarioID: scenarioID},
		storage.ClassData,
		"persona.db",
	)
	if err != nil {
		return "", fmt.Errorf("resolve persona db path: %w", err)
	}
	return sqliteFileDSN(path)
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
	scenarioID, err := storage.ScenarioNamespace("persona")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve persona storage namespace: %w", err)
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

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "persona"}) {
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
	primaryFileRoots, err := scenarioStorageRoots()
	if err != nil {
		log.Fatalf("file storage configuration failed: %v", err)
	}
	fileRoots := filerouting.New(primaryFileRoots)
	clock := schedule.System()
	var channelService channels.Service
	var documentService documents.Service
	healthProvider := personas.HealthProviderFunc(func(ctx context.Context, persona personas.Persona) ([]personas.HealthFinding, error) {
		findings := make([]personas.HealthFinding, 0)
		if channelService != nil {
			channelFindings, err := channelService.CheckHealth(ctx, persona.ID)
			if err != nil {
				return nil, fmt.Errorf("channel health: %w", err)
			}
			findings = append(findings, channelFindings...)
		}
		if documentService != nil {
			documentFindings, err := documentService.CheckHealth(ctx, persona.ID)
			if err != nil {
				return nil, fmt.Errorf("document health: %w", err)
			}
			findings = append(findings, documentFindings...)
		}
		return findings, nil
	})
	personaService := personas.NewServiceWithHealth(personas.NewSQLiteRepository(db, clock), healthProvider)
	journalService := journal.NewService(journal.NewSQLiteRepository(db, clock))
	handoffService := handoffs.NewServiceWithRelay(handoffs.NewSQLiteRepository(db, clock), personaService, journalService, notificationRelay(), clock)
	channelService = channels.NewService(channels.NewSQLiteRepository(db, clock), personaService, defaultChannelAdapters(), journalService, clock)
	authority := documents.NewUnavailableAuthority()
	if base := strings.TrimSpace(os.Getenv("DOCUMENT_MANAGER_API_BASE")); base != "" {
		authority = documents.HTTPAuthority{BaseURL: base}
	}
	documentService = documents.NewService(documents.NewSQLiteRepository(db, clock), personaService, handoffService, authority, journalService, clock)
	accountService := accounts.NewService(accounts.NewSQLiteRepository(db, clock), personaService, handoffService, journalService, clock)
	accessService := access.NewService(access.NewSQLiteRepository(db, clock), personaService, journalService, access.LiveVerifier{}, access.ServiceOptions{Clock: clock, Secret: []byte(os.Getenv("PERSONA_ATTESTATION_SECRET")), KeyID: "persona-local"})

	srv := server.New(
		server.Deps{Clock: clock, Logger: log.Default()},
		healthH.Module(db, "persona-api", "1.0.0"),
		capsH.Module(capabilities.NewRegistry()),
		accessH.ModuleWithService(accessService),
		accountsH.ModuleWithService(accountService),
		channelsH.ModuleWithService(channelService),
		documentsH.ModuleWithService(documentService),
		handoffsH.ModuleWithService(handoffService),
		personasH.ModuleWithService(personaService),
		journalH.ModuleWithService(journalService),
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.RegisterWithFileRoots(rootMux, db, fileRoots)

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

func defaultChannelAdapters() channels.AdapterRegistry {
	return channels.Registry{
		"email":  channels.EmailAdapter{Source: channelSourceFromEnv("PERSONA_EMAIL_ADAPTER_URL", "email")},
		"sms":    channels.SMSAdapter{Source: channelSourceFromEnv("PERSONA_SMS_ADAPTER_URL", "sms")},
		"device": channels.DeviceAdapter{Source: channelSourceFromEnv("PERSONA_DEVICE_ADAPTER_URL", "device")},
	}
}

func channelSourceFromEnv(key, name string) channels.CodeSource {
	if baseURL := strings.TrimSpace(os.Getenv(key)); baseURL != "" {
		return channels.HTTPSource{BaseURL: baseURL}
	}
	return channels.NewUnavailableSource(name)
}

func notificationRelay() handoffs.Relay {
	if baseURL := strings.TrimSpace(os.Getenv("PERSONA_NOTIFICATION_HUB_API_BASE")); baseURL != "" {
		return handoffs.HTTPRelay{BaseURL: baseURL}
	}
	return nil
}
