package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"scenario-to-android/internal/builds"
	"scenario-to-android/internal/capabilities"
	"scenario-to-android/internal/journeys"
	"scenario-to-android/internal/modules"
	"scenario-to-android/internal/releases"
	"scenario-to-android/internal/server"
	"scenario-to-android/internal/targets"

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
	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	validationmatrix "github.com/vrooli/vrooli/packages/delivery-ramp-go/validationmatrix"
	_ "modernc.org/sqlite"

	buildsh "scenario-to-android/handlers/builds"
	capsH "scenario-to-android/handlers/capabilities"
	healthH "scenario-to-android/handlers/health"
	rampH "scenario-to-android/handlers/releases"
)

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
	scenarioID, err := storage.ScenarioNamespace("scenario-to-android")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve scenario-to-android storage namespace: %w", err)
	}
	return resolver.Resolve(storage.Options{ScenarioID: scenarioID})
}

func repoRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	for dir := cwd; ; dir = filepath.Dir(dir) {
		if st, err := os.Stat(filepath.Join(dir, "VISION.md")); err == nil && !st.IsDir() {
			return dir
		}
		next := filepath.Dir(dir)
		if next == dir {
			return ""
		}
	}
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "scenario-to-android"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "scenario-to-android",
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
	matrixStore, err := validationmatrix.NewFileStore(filepath.Join(primaryFileRoots.DataDir, "validation-matrices"))
	if err != nil {
		log.Fatalf("validation matrix storage configuration failed: %v", err)
	}
	androidPlan := journeys.AndroidPlan()
	journeySelection := validationmatrix.JourneySelection{JourneyID: androidPlan.ID, DisplayName: "Android generated-app conformance", SourcePath: "internal/journeys/plan.go", ExecutionMode: "platform", Required: true, Category: "android", Requirements: []string{"hello-mobile APK", "device-control lease", "redacted recording", "BAS WebView flow"}, Safety: validationmatrix.JourneySafety{Mutating: true, RequiresIsolation: true, RequiresConfirmation: true}}
	journeyRunner := func(ctx context.Context, request deliveryramp.DriverRequest, deviceURL, basURL, deviceTransport string, client *http.Client) (deliveryramp.JourneyResult, error) {
		if client == nil {
			client = http.DefaultClient
		}
		driver := journeys.Driver{
			Devices: &journeys.HTTPDeviceClient{BaseURL: deviceURL, Actor: "scenario-to-android", DeviceTransport: deviceTransport, Client: client},
			BAS:     journeys.HTTPBASClient{BaseURL: basURL, FlowRoot: repoRoot(), HTTP: client},
			Actor:   "scenario-to-android",
		}
		return driver.Execute(ctx, request)
	}
	resolveTransport := func(ctx context.Context, deviceURL, targetID string, client *http.Client) string {
		observed, err := (targets.DeviceControlInventory{Resolve: func(context.Context) (string, error) { return deviceURL, nil }, Client: client}).List(ctx)
		if err == nil {
			for _, item := range observed {
				if item.ID == targetID && strings.TrimSpace(item.ADBTransport) != "" {
					return item.ADBTransport
				}
			}
		}
		return "usb"
	}
	executors := validationmatrix.Executors{Local: releases.Executor{JourneyPlan: androidPlan.JourneyPlan(), ResolveTransport: resolveTransport, RunJourney: journeyRunner}}
	if bridgeExecutor := validationmatrix.NewClientFromEnv(validationmatrix.WithPlatform("android")); bridgeExecutor != nil {
		executors.Bridge = bridgeExecutor
	}
	matrixOptions := []validationmatrix.ServiceOption{
		validationmatrix.WithCatalogResolver(releases.Catalog{Probe: targets.Prober{Devices: targets.NewDeviceControlInventory()}, Journey: journeySelection}),
	}
	if deploymentURL := strings.TrimSpace(os.Getenv("DEPLOYMENT_MANAGER_URL")); deploymentURL != "" {
		profileID := strings.TrimSpace(os.Getenv("DEPLOYMENT_MANAGER_PROFILE_ID"))
		gitCommit := strings.TrimSpace(os.Getenv("DEPLOYMENT_MANAGER_GIT_COMMIT"))
		if profileID != "" && gitCommit != "" {
			matrixOptions = append(matrixOptions, validationmatrix.WithReleaseReporter(
				validationmatrix.NewDeploymentReporterFromURL(
					deploymentURL,
					profileID,
					gitCommit,
					nil,
					validationmatrix.WithDeploymentIdentity("scenario-to-android", "scenario-to-android", "android", runtime.GOOS),
				),
			))
		}
	}
	matrixService := validationmatrix.NewService(matrixStore, executors, matrixOptions...)
	matrixService.RecoverStale()
	matrixHandler := validationmatrix.NewHandler(matrixService)
	var signingClient credentialclient.Client
	if authority, authorityErr := credentialauthority.Default(); authorityErr != nil {
		log.Printf("Android signing unavailable: credential authority unavailable: %v", authorityErr)
	} else if client, clientErr := credentialclient.NewClient(credentialclient.ClientOptions{
		Authority: authority,
		Descriptors: func() ([]credentialclient.CredentialRef, error) {
			return credentialclient.DiscoverDescriptors(repoRoot())
		},
	}); clientErr != nil {
		log.Printf("Android signing unavailable: credential client unavailable: %v", clientErr)
	} else {
		signingClient = client
	}
	builder := builds.Builder{Signing: signingClient}
	buildSurface := buildsh.Surface{
		Builder:         builder,
		SigningIdentity: builds.DefaultSigningIdentity,
		Generate:        builder.Generate,
		ProvisionSigning: func(ctx context.Context, identity string) error {
			if builder.Signing == nil {
				return fmt.Errorf("secrets-manager credential client is not configured")
			}
			provisioner, ok := builder.Signing.(builds.SigningProvisioner)
			if !ok {
				return fmt.Errorf("configured credential client cannot provision")
			}
			return builds.ProvisionSigningKey(ctx, provisioner, identity, "", builder.Run)
		},
	}

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "scenario-to-android-api", "1.0.0"),
		capsH.Module(capabilities.NewRegistry()),
		rampH.Module([]*validationmatrix.Handler{matrixHandler}, buildSurface),
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
