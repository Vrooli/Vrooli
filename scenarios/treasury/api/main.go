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

	"treasury/internal/approval"
	"treasury/internal/capabilities"
	"treasury/internal/identity"
	"treasury/internal/instrument"
	"treasury/internal/modules"
	"treasury/internal/operatorauth"
	"treasury/internal/rail"
	"treasury/internal/rail/manual"
	"treasury/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	_ "modernc.org/sqlite"

	agentspendH "treasury/handlers/agentspend"
	capsH "treasury/handlers/capabilities"
	healthH "treasury/handlers/health"
	treasuryadminH "treasury/handlers/treasuryadmin"
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
	scenarioID, err := storage.ScenarioNamespace("treasury")
	if err != nil {
		return "", fmt.Errorf("resolve treasury storage namespace: %w", err)
	}
	path, err := resolver.Path(
		storage.Options{ScenarioID: scenarioID},
		storage.ClassData,
		"treasury.db",
	)
	if err != nil {
		return "", fmt.Errorf("resolve treasury db path: %w", err)
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
	scenarioID, err := storage.ScenarioNamespace("treasury")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve treasury storage namespace: %w", err)
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
	if preflight.Run(preflight.Config{ScenarioName: "treasury"}) {
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
	var identityVerifier identity.Verifier
	identityVerifier, err = identity.NewHTTPVerifier(os.Getenv("AGENT_MANAGER_API_URL"), &http.Client{Timeout: 5 * time.Second})
	if err != nil {
		log.Printf("agent-manager identity verification unavailable; automated spend will fail closed: %v", err)
		identityVerifier = identity.UnavailableVerifier{Cause: err}
	}

	operatorToken := strings.TrimSpace(os.Getenv("TREASURY_OPERATOR_TOKEN"))
	if operatorToken == "" {
		operatorToken = strings.TrimSpace(os.Getenv("API_TOKEN"))
	}
	var operatorAuthorizer operatorauth.Authorizer
	operatorAuthorizer, err = operatorauth.NewStaticToken(operatorToken)
	if err != nil {
		log.Printf("operator realm unavailable; TreasuryAdmin will fail closed: %v", err)
		operatorAuthorizer = operatorauth.Unavailable{Cause: err}
	}
	var approvalRelay approval.Relay
	approvalRelay, err = approval.NewNotificationRelay(os.Getenv("NOTIFICATION_HUB_API_URL"), &http.Client{Timeout: 5 * time.Second})
	if err != nil {
		log.Printf("notification relay unavailable; approvals remain local and relay attempts will record failure: %v", err)
		approvalRelay = approval.UnavailableRelay{Cause: err}
	}
	railRegistry, err := rail.NewRegistry(manual.New())
	if err != nil {
		log.Fatalf("rail registry configuration failed: %v", err)
	}
	var credentialResolver instrument.CredentialResolver
	authority, authorityErr := credentialauthority.Default()
	if authorityErr == nil {
		var client credentialclient.Client
		client, authorityErr = credentialclient.NewInProcess(credentialclient.InProcessOptions{Authority: authority})
		if authorityErr == nil {
			credentialResolver, authorityErr = instrument.NewCredentialClientResolver(client)
		}
	}
	if authorityErr != nil {
		log.Printf("instrument credential resolution unavailable; instrument use will fail closed: %v", authorityErr)
		credentialResolver = instrument.UnavailableResolver{Cause: authorityErr}
	}

	srv := server.New(
		server.Deps{Clock: clock, Logger: log.Default()},
		healthH.Module(db, "treasury-api", "1.0.0"),
		capsH.Module(capabilities.NewRegistry()),
		agentspendH.Module(db, identityVerifier, clock, approvalRelay, railRegistry, credentialResolver),
		treasuryadminH.Module(db, operatorAuthorizer, clock, approvalRelay, railRegistry, credentialResolver),
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
