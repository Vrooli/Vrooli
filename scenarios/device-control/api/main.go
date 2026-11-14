package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"device-control/internal/androidartifact"
	"device-control/internal/capabilities"
	internalclock "device-control/internal/clock"
	"device-control/internal/environment"
	internalflows "device-control/internal/flows"
	"device-control/internal/modules"
	"device-control/internal/server"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	_ "modernc.org/sqlite"

	capsH "device-control/handlers/capabilities"
	controlH "device-control/handlers/control"
	desktopH "device-control/handlers/desktop"
	flowsH "device-control/handlers/flows"
	healthH "device-control/handlers/health"
	lifecycleH "device-control/handlers/lifecycle"
	"device-control/internal/control"
	"device-control/internal/desktophelper"
	"device-control/internal/desktopwebrtc"
	nativemacos "device-control/internal/native/macos"
	strategyregistry "device-control/strategy/registry"
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
	scenarioID, err := storage.ScenarioNamespace("device-control")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve device-control storage namespace: %w", err)
	}
	return resolver.Resolve(storage.Options{ScenarioID: scenarioID})
}

func main() {
	processEnv := environment.OSReader{}
	configPath := flag.String("config", "", "user-scoped companion configuration path")
	flag.Parse()
	if *configPath != "" {
		if err := validateCompanionConfig(*configPath); err != nil {
			log.Fatalf("desktop companion configuration failed: %v", err)
		}
	}
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "device-control"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "device-control",
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
	gateway := internalflows.NewGateway(http.DefaultClient, discovery.ResolveScenarioURLDefault)
	bridgeCredentials := control.BridgeCredentialConfig{
		OwnerTokenFile: environment.Value(processEnv, "VROOLI_BRIDGE_TOKEN_FILE"),
		AuthTokenFile:  environment.Value(processEnv, "VROOLI_AUTH_TOKEN_FILE"),
		AuthSocket:     environment.Value(processEnv, "VROOLI_AUTH_SOCKET"),
	}
	hostNodeID := strings.TrimSpace(environment.Value(processEnv, "VROOLI_NODE_ID"))
	if hostNodeID == "" {
		hostNodeID, _ = os.Hostname()
	}
	controlService, err := control.NewWithDBAndAttached(strategyregistry.DefaultWithEnvironment(processEnv), db, control.NewBridgeAttachedReader(http.DefaultClient, nil, bridgeCredentials), fileRoots)
	if err != nil {
		log.Fatalf("device-control state initialization failed: %v", err)
	}
	controlService.SetHostNodeID(hostNodeID)
	lifecycleProvider, err := lifecycleH.New(context.Background(), db.Primary())
	if err != nil {
		log.Fatalf("lifecycle provider initialization failed: %v", err)
	}
	controlService.SetAndroidArtifactResolver(androidartifact.Client{
		HTTP: &http.Client{
			Timeout: 5 * time.Minute,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		BaseURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "scenario-to-android")
		},
		Inspector:        androidartifact.AAPTInspector{ADBPath: "adb"},
		SigningInspector: androidartifact.AndroidSDKSigningInspector{ADBPath: "adb"},
	})
	controlService.SetEvidenceDir(environment.Value(processEnv, "DEVICE_CONTROL_EVIDENCE_DIR"))
	controlService.SetAgentPlanner(internalflows.NewGatewayPlanner(gateway))
	desktopOwnerBootstrap := environment.Value(processEnv, "DEVICE_CONTROL_DESKTOP_OWNER_CONFIG")
	desktopCtx, stopDesktop := context.WithCancel(context.Background())
	desktopDone := make(chan struct{})
	var desktopOwnerErr error // Read only after desktopDone closes.
	if desktopOwnerBootstrap != "" {
		go func() {
			defer close(desktopDone)
			desktopOwnerErr = controlService.RunDesktopOwner(desktopCtx, desktopOwnerBootstrap, func(err error) { log.Printf("desktop admission owner unavailable: %v", err) })
		}()
	} else {
		close(desktopDone)
	}
	defer stopDesktop()

	desktopProvider := desktopwebrtc.Provider(desktopwebrtc.UnavailableProvider{})
	if runtime.GOOS == "darwin" {
		if backend, backendErr := nativemacos.NewHostBackend(""); backendErr == nil {
			desktopProvider = nativemacos.Provider{Backend: backend}
		} else {
			log.Printf("desktop companion unavailable: %v", backendErr)
		}
	}
	desktopManager := desktopwebrtc.NewManager(desktopProvider)
	lifecycleProvider.SetOwnerInventory(func(ctx context.Context) ([]*commonv1.LifecycleWorkItem, []*commonv1.LifecycleBlocker, bool) {
		controlWork, controlBlockers, controlComplete := controlService.LifecycleInventory(ctx)
		desktopWork, desktopBlockers, desktopComplete := desktopManager.LifecycleInventory(ctx)
		work := append(controlWork, desktopWork...)
		blockers := append(controlBlockers, desktopBlockers...)
		select {
		case <-desktopDone:
			if desktopOwnerErr != nil {
				blockers = append(blockers, &commonv1.LifecycleBlocker{Code: "desktop_owner_cleanup_uncertain", Message: "API desktop owner cleanup did not finish successfully", Disposition: commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_UNKNOWN, Observed: true})
			}
		default:
			blockers = append(blockers, &commonv1.LifecycleBlocker{Code: "desktop_owner_active", Message: "API desktop owner has not joined its accepted RPC handlers", Disposition: commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_BLOCKING, Observed: true})
		}
		return work, blockers, controlComplete && desktopComplete
	})
	lifecycleProvider.SetComponentMaintenance(func(ctx context.Context, exits map[string]string) ([]*commonv1.LifecycleManagedComponent, []*commonv1.LifecycleWorkItem, []*commonv1.LifecycleBlocker, bool) {
		return desktopHelperManagedInventory(ctx, desktopOwnerBootstrap, exits)
	}, func(ctx context.Context) error {
		if err := controlService.QuiesceObservers(ctx); err != nil {
			return err
		}
		stopDesktop()
		select {
		case <-desktopDone:
			return desktopOwnerErr
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	srv := server.New(
		server.Deps{Clock: internalclock.System(), Logger: log.Default(), Admission: lifecycleProvider},
		healthH.Module(db, "device-control-api", "1.0.0"),
		lifecycleH.Module(lifecycleProvider),
		capsH.Module(capabilities.NewRegistry()),
		controlH.Module(controlService),
		desktopH.Module(desktopManager),
		flowsH.Module(internalflows.NewResolverWithAnchors(gateway, controlService.Anchors())),
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
	handler := lifecycleProvider.Middleware(apihttp.TestModeMiddleware(rootMux))

	if err := apiserver.Run(apiserver.Config{
		Handler: handler,
		// Device Control has a domain provider that also accounts for Android,
		// Cast and desktop helper generations; its own middleware closes ingress.
		LifecycleConfiguredExternally: true,
		ReadTimeout:                   30 * time.Second,
		// Flow execution allows up to five minutes, then still has to encode its
		// result and audit evidence. Keep the server write deadline above that
		// contract so bounded Android operations return their actual outcome.
		WriteTimeout: control.FlowHTTPWriteTimeout(),
		Cleanup: func(ctx context.Context) error {
			stopDesktop()
			select {
			case <-desktopDone:
				return db.Close()
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func desktopHelperManagedInventory(ctx context.Context, ownerBootstrapPath string, exits map[string]string) ([]*commonv1.LifecycleManagedComponent, []*commonv1.LifecycleWorkItem, []*commonv1.LifecycleBlocker, bool) {
	work, blockers, complete := desktopHelperLifecycleInventory(ctx, ownerBootstrapPath)
	if ownerBootstrapPath == "" {
		return nil, work, blockers, complete
	}
	owner, err := desktophelper.LoadOwnerBootstrap(ownerBootstrapPath)
	if err != nil {
		return nil, work, blockers, false
	}
	receipt, err := desktophelper.ReadLifecycleReceipt(ctx, owner.HelperConfigPath)
	if err != nil || receipt.ComponentGeneration == "" {
		return nil, work, blockers, false
	}
	components := []*commonv1.LifecycleManagedComponent{{Name: "desktop-helper", Generation: receipt.ComponentGeneration}}
	// A valid current registration enumerates the component without claiming
	// present liveness. Unknown exit remains a blocker until its owner waits.
	if exits["desktop-helper"] != receipt.ComponentGeneration {
		return components, work, blockers, true
	}
	for _, item := range work {
		if item.GetKind() == "desktop-helper-process-exit" {
			item.Status = "completed"
		}
	}
	kept := blockers[:0]
	for _, blocker := range blockers {
		if blocker.GetCode() != "desktop_helper_process_exit_unobserved" && blocker.GetCode() != "desktop_helper_liveness_unconfirmed" {
			kept = append(kept, blocker)
		}
	}
	// Exit is independent of controller cleanup. Missing success remains a
	// blocker even after the process owner has proved process termination.
	if receipt.Status != "controller-cleanup-completed" && len(kept) == 0 {
		kept = append(kept, &commonv1.LifecycleBlocker{Code: "desktop_helper_cleanup_uncertain", Message: "exited helper has no successful controller cleanup receipt", Disposition: commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_UNKNOWN, Observed: true})
	}
	return components, work, kept, true
}

func desktopHelperLifecycleInventory(ctx context.Context, ownerBootstrapPath string) ([]*commonv1.LifecycleWorkItem, []*commonv1.LifecycleBlocker, bool) {
	if ownerBootstrapPath == "" {
		return nil, nil, true
	}
	unknown := func(message string) ([]*commonv1.LifecycleWorkItem, []*commonv1.LifecycleBlocker, bool) {
		return []*commonv1.LifecycleWorkItem{{Id: "owner/desktop-helper-shutdown", Kind: "desktop-helper-shutdown", Status: "unknown"}, {Id: "owner/desktop-helper-process-exit", Kind: "desktop-helper-process-exit", Status: "unknown"}}, []*commonv1.LifecycleBlocker{{Code: "desktop_helper_receipt_unavailable", Message: message, Disposition: commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_UNKNOWN, Observed: true}}, false
	}
	owner, err := desktophelper.LoadOwnerBootstrap(ownerBootstrapPath)
	if err != nil {
		return unknown("desktop helper owner bootstrap could not be read for lifecycle inventory")
	}
	receipt, err := desktophelper.ReadLifecycleReceipt(ctx, owner.HelperConfigPath)
	if err != nil {
		return unknown("desktop helper shutdown receipt is missing or invalid")
	}
	status := receipt.Status
	blockers := []*commonv1.LifecycleBlocker{}
	// A recognized receipt completes owner-work enumeration even when it leaves
	// shutdown, liveness, cleanup, or process exit unresolved. Those conditions
	// remain explicit blockers; the control plane needs the complete component
	// list before it can quiesce and stop the exact generation safely.
	complete := true
	switch receipt.Status {
	case "controller-cleanup-completed":
		status = "completed"
	case "running":
		status = "unknown"
		blockers = append(blockers, &commonv1.LifecycleBlocker{Code: "desktop_helper_liveness_unconfirmed", Message: "last helper receipt reports running, but current process liveness is unconfirmed", Disposition: commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_UNKNOWN, Observed: true})
	case "stopping":
		blockers = append(blockers, &commonv1.LifecycleBlocker{Code: "desktop_helper_shutdown_pending", Message: fmt.Sprintf("desktop helper shutdown is pending with %d accepted RPC handler(s); receipt code %s", receipt.ActiveRequests, receipt.ErrorCode), Disposition: commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_BLOCKING, Observed: true})
	case "uncertain":
		blockers = append(blockers, &commonv1.LifecycleBlocker{Code: "desktop_helper_cleanup_uncertain", Message: "desktop helper cleanup did not produce a successful terminal receipt", Disposition: commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_UNKNOWN, Observed: true})
	default:
		return unknown("desktop helper shutdown receipt has an unsupported state")
	}
	// The helper receipt is written before RunBound's deferred native/SQLite
	// teardown. Only the external process owner can confirm process exit.
	work := []*commonv1.LifecycleWorkItem{{Id: "owner/desktop-helper-shutdown", Kind: "desktop-helper-shutdown", Status: status}, {Id: "owner/desktop-helper-process-exit", Kind: "desktop-helper-process-exit", Status: "unknown"}}
	blockers = append(blockers, &commonv1.LifecycleBlocker{Code: "desktop_helper_process_exit_unobserved", Message: "desktop helper cleanup receipt does not establish helper process exit", Disposition: commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_UNKNOWN, Observed: true})
	return work, blockers, complete
}

func validateCompanionConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %q: %w", path, err)
	}
	var config struct {
		DisplayID string `json:"display_id"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("decode %q: %w", path, err)
	}
	if len(config.DisplayID) > 256 {
		return fmt.Errorf("display_id exceeds 256 bytes")
	}
	return nil
}
