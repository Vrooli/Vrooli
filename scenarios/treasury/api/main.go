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
	"treasury/internal/ledger"
	"treasury/internal/mandate"
	"treasury/internal/modules"
	"treasury/internal/operatorauth"
	"treasury/internal/rail"
	"treasury/internal/rail/card"
	lithicrail "treasury/internal/rail/card/lithic"
	"treasury/internal/rail/manual"
	x402rail "treasury/internal/rail/x402"
	"treasury/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/discovery"
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
	x402gateH "treasury/handlers/x402gate"
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

type credentialReader interface {
	Resolve(context.Context, string, string) (string, error)
}

func runtimeCredential(ctx context.Context, reader credentialReader, envValue, identityName, field string) (string, error) {
	if value := strings.TrimSpace(envValue); value != "" {
		return value, nil
	}
	if reader == nil {
		return "", fmt.Errorf("credential authority unavailable")
	}
	value, err := reader.Resolve(ctx, identityName, field)
	if err != nil {
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("credential value is empty")
	}
	return value, nil
}

func scenarioURL(ctx context.Context, override, scenario string) string {
	if value := strings.TrimSpace(override); value != "" {
		return value
	}
	value, err := discovery.ResolveScenarioURLDefault(ctx, scenario)
	if err != nil {
		return ""
	}
	return value
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
	startupCtx := context.Background()
	var credentialClient credentialclient.Client
	authority, authorityErr := credentialauthority.Default()
	if authorityErr == nil {
		credentialClient, authorityErr = credentialclient.NewInProcess(credentialclient.InProcessOptions{Authority: authority})
	}
	var identityVerifier identity.Verifier
	identityVerifier, err = identity.NewHTTPVerifier(scenarioURL(startupCtx, os.Getenv("AGENT_MANAGER_API_URL"), "agent-manager"), &http.Client{Timeout: 5 * time.Second})
	if err != nil {
		log.Printf("agent-manager identity verification unavailable; automated spend will fail closed: %v", err)
		identityVerifier = identity.UnavailableVerifier{Cause: err}
	}

	operatorToken, operatorTokenErr := runtimeCredential(startupCtx, credentialClient, os.Getenv("TREASURY_OPERATOR_TOKEN"), "vrooli/treasury", "operator-token")
	if operatorTokenErr != nil {
		operatorToken = strings.TrimSpace(os.Getenv("API_TOKEN"))
	}
	var operatorAuthorizer operatorauth.Authorizer
	operatorAuthorizer, err = operatorauth.NewStaticToken(operatorToken)
	if err != nil {
		log.Printf("operator realm unavailable; TreasuryAdmin will fail closed: %v", err)
		operatorAuthorizer = operatorauth.Unavailable{Cause: err}
	}
	var approvalRelay approval.Relay
	approvalRelay, err = approval.NewNotificationRelay(scenarioURL(startupCtx, os.Getenv("NOTIFICATION_HUB_API_URL"), "notification-hub"), &http.Client{Timeout: 5 * time.Second})
	if err != nil {
		log.Printf("notification relay unavailable; approvals remain local and relay attempts will record failure: %v", err)
		approvalRelay = approval.UnavailableRelay{Cause: err}
	}
	var mandateSigner mandate.Signer
	mandateSigningKey, signingKeyErr := runtimeCredential(startupCtx, credentialClient, os.Getenv("TREASURY_MANDATE_SIGNING_KEY"), "vrooli/treasury", "mandate-signing-key")
	if signingKeyErr == nil {
		mandateSigner, err = mandate.NewHMACSigner([]byte(mandateSigningKey))
	} else {
		err = signingKeyErr
	}
	if err != nil {
		log.Printf("mandate issuance unavailable; TREASURY_MANDATE_SIGNING_KEY is required: %v", err)
		mandateSigner = nil
	}
	x402HTTPClient := &http.Client{Timeout: 30 * time.Second}
	x402Signer, err := x402rail.NewRPCSigner(x402HTTPClient)
	if err != nil {
		log.Fatalf("x402 signer configuration failed: %v", err)
	}
	x402Adapter, err := x402rail.New(x402HTTPClient, x402Signer)
	if err != nil {
		log.Fatalf("x402 rail configuration failed: %v", err)
	}
	cardHTTPClient := &http.Client{Timeout: 30 * time.Second}
	lithicURL := strings.TrimSpace(os.Getenv("LITHIC_SANDBOX_URL"))
	if lithicURL == "" {
		lithicURL = "https://sandbox.lithic.com"
	}
	lithicAdapter, err := lithicrail.New(lithicURL, cardHTTPClient)
	if err != nil {
		log.Fatalf("scoped card rail configuration failed: %v", err)
	}
	railRegistry, err := rail.NewRegistry(manual.New(), x402Adapter, lithicAdapter)
	if err != nil {
		log.Fatalf("rail registry configuration failed: %v", err)
	}
	cardRegistry, err := card.NewRegistry(lithicAdapter)
	if err != nil {
		log.Fatalf("card issuer registry configuration failed: %v", err)
	}
	facilitatorURL := strings.TrimSpace(os.Getenv("X402_FACILITATOR_URL"))
	if facilitatorURL == "" {
		facilitatorURL = "http://127.0.0.1:14020"
	}
	x402Facilitator, err := x402rail.NewHTTPFacilitator(facilitatorURL, x402HTTPClient)
	if err != nil {
		log.Fatalf("x402 facilitator configuration failed: %v", err)
	}
	x402Gate, err := x402rail.NewGate(x402rail.NewSQLiteInboundRepository(db), x402Facilitator)
	if err != nil {
		log.Fatalf("x402 inbound gate configuration failed: %v", err)
	}
	var credentialResolver instrument.CredentialResolver
	if authorityErr == nil {
		credentialResolver, authorityErr = instrument.NewCredentialClientResolver(credentialClient)
	}
	if authorityErr != nil {
		log.Printf("instrument credential resolution unavailable; instrument use will fail closed: %v", authorityErr)
		credentialResolver = instrument.UnavailableResolver{Cause: authorityErr}
	}
	var ledgerEmitter ledger.Emitter
	ledgerEmitter, err = ledger.NewMoneyLedgerEmitter(scenarioURL(startupCtx, os.Getenv("MONEY_LEDGER_API_URL"), "money-ledger"), os.Getenv("TREASURY_LEDGER_BOOK_ID"), os.Getenv("TREASURY_LEDGER_ACCOUNT_ID"), &http.Client{Timeout: 5 * time.Second})
	if err != nil {
		log.Printf("money-ledger emission unavailable; settlements will remain durably queued: %v", err)
		ledgerEmitter = ledger.UnavailableEmitter{Cause: err}
	}
	ledgerService := ledger.NewService(ledger.NewSQLiteRepository(db), ledgerEmitter, time.Now)
	ledgerCtx, stopLedger := context.WithCancel(context.Background())
	go ledgerService.Run(ledgerCtx, 5*time.Second, log.Default())

	srv := server.New(
		server.Deps{Clock: clock, Logger: log.Default()},
		healthH.Module(db, "treasury-api", "1.0.0"),
		capsH.Module(capabilities.NewRegistry()),
		agentspendH.Module(db, identityVerifier, clock, approvalRelay, railRegistry, cardRegistry, credentialResolver),
		treasuryadminH.Module(db, operatorAuthorizer, clock, approvalRelay, railRegistry, cardRegistry, credentialResolver, mandateSigner),
		x402gateH.Module(x402Gate, operatorAuthorizer),
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
		Cleanup: func(ctx context.Context) error {
			stopLedger()
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
