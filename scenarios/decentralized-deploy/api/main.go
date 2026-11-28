package main

import (
	"context"
	"decentralized-deploy/internal/capabilities"
	"decentralized-deploy/internal/modules"
	"decentralized-deploy/internal/server"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	capsH "decentralized-deploy/handlers/capabilities"
	deployH "decentralized-deploy/handlers/deployment"
	healthH "decentralized-deploy/handlers/health"
	notesH "decentralized-deploy/handlers/notes" // EXAMPLE-DOMAIN:notes
	deploy "decentralized-deploy/internal/deployment"
)

// runtimeDeps names the external startup effects so the complete application
// bootstrap can be tested without opening a listener or relying on host state.
type runtimeDeps struct {
	preflightRun    func(preflight.Config) bool
	openDatabase    func(context.Context, database.Config) (*database.RoutedDB, error)
	ensureSchemas   func(context.Context, *database.RoutedDB) error
	storageRoots    func() (storage.Paths, error)
	measuresHandler func(*database.RoutedDB) (http.Handler, error)
	authConfig      func(func(string) string) (authn.Config, error)
	runServer       func(apiserver.Config) error
}

func productionRuntimeDeps() runtimeDeps {
	return runtimeDeps{
		preflightRun: preflight.Run,
		openDatabase: func(ctx context.Context, cfg database.Config) (*database.RoutedDB, error) {
			return database.Open(ctx, cfg)
		},
		ensureSchemas:   ensureScenarioSchemas,
		storageRoots:    scenarioStorageRoots,
		measuresHandler: notesMeasuresHandler,
		authConfig:      authn.FromEnvironment,
		runServer:       apiserver.Run,
	}
}

// scenarioStorageRoots resolves all filesystem storage classes once at
// startup. Domain file stores receive the routed roots and select the
// request-appropriate class at their own storage seam.
func scenarioStorageRoots() (storage.Paths, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return storage.Paths{}, fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("decentralized-deploy")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve decentralized-deploy storage namespace: %w", err)
	}
	return resolver.Resolve(storage.Options{ScenarioID: scenarioID})
}

func ensureScenarioSchemas(ctx context.Context, db *database.RoutedDB) error {
	return database.EnsureSchemas(ctx, db.Primary(), modules.AllSchemas()...)
}

func scenarioRoot() (string, error) {
	root, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for range 5 {
		if _, err := os.Stat(filepath.Join(root, ".vrooli", "service.json")); err == nil {
			return root, nil
		}
		parent := filepath.Dir(root)
		if parent == root {
			break
		}
		root = parent
	}
	return "", fmt.Errorf("could not locate scenario root from current directory")
}

func managedKuboBinary() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve host-tool owner home: %w", err)
	}
	ownerCLI := filepath.Join(home, ".vrooli", "bin", "vrooli")
	if info, statErr := os.Stat(ownerCLI); statErr != nil || info.IsDir() || info.Mode()&0o111 == 0 {
		ownerCLI, err = exec.LookPath("vrooli")
		if err != nil {
			return "", fmt.Errorf("managed Kubo owner CLI is unavailable: %w", err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, ownerCLI, "host", "install", "kubo", "--dry-run", "--json").Output()
	if err != nil {
		return "", fmt.Errorf("query managed Kubo owner selection: %w", err)
	}
	return selectedKuboBinary(output, os.Getenv("KUBO_BINARY"))
}

type hostToolSelection struct {
	Command   string `json:"command"`
	Installed bool   `json:"installed"`
	Version   string `json:"version"`
	OK        bool   `json:"ok"`
}

func selectedKuboBinary(ownerOutput []byte, override string) (string, error) {
	var selection hostToolSelection
	if err := json.Unmarshal(ownerOutput, &selection); err != nil {
		return "", fmt.Errorf("decode managed Kubo owner selection: %w", err)
	}
	if !selection.OK || !selection.Installed || selection.Version != "0.43.1" {
		return "", fmt.Errorf("managed Kubo selection is not installed and pinned at 0.43.1")
	}
	if !filepath.IsAbs(selection.Command) {
		return "", fmt.Errorf("managed Kubo owner returned a non-absolute executable path")
	}
	selected := filepath.Clean(selection.Command)
	if strings.TrimSpace(override) != "" && filepath.Clean(override) != selected {
		return "", fmt.Errorf("KUBO_BINARY override does not match the owner-selected managed installation")
	}
	info, err := os.Stat(selected)
	if err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
		return "", fmt.Errorf("managed Kubo CLI is not executable at %s", selected)
	}
	return selected, nil
}

func turboSandboxPathRecord(scenarioRoot string) (deploy.PathRecord, error) {
	path := filepath.Join(scenarioRoot, "data", "provider-path-catalog.json")
	file, err := os.Open(path)
	if err != nil {
		return deploy.PathRecord{}, fmt.Errorf("open provider/path catalogue: %w", err)
	}
	defer file.Close()
	catalog, err := deploy.LoadCatalog(file)
	if err != nil {
		return deploy.PathRecord{}, err
	}
	return catalog.Path("turbo", "ar-io-free-testnet-sandbox")
}

func notesMeasuresHandler(db *database.RoutedDB) (http.Handler, error) {
	return notesH.MeasuresHandler(db, schedule.System())
}

func main() {
	if preflight.Run(preflight.Config{ScenarioName: "decentralized-deploy"}) {
		return
	}

	deps := productionRuntimeDeps()
	// The calls remain visible in main for API lifecycle validation; tests
	// replace the runner dependency before exercising the startup branches.
	deps.preflightRun = func(preflight.Config) bool { return false }
	deps.runServer = func(config apiserver.Config) error {
		return apiserver.Run(config)
	}
	if err := run(deps); err != nil {
		log.Fatal(err)
	}
}

func run(deps runtimeDeps) error {
	const scenarioName = "decentralized-deploy"
	ctx := context.Background()

	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if deps.preflightRun(preflight.Config{ScenarioName: scenarioName}) {
		return nil
	}

	db, err := deps.openDatabase(ctx, database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     scenarioName,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}
	serverCleanupCalled := false
	defer func() {
		if !serverCleanupCalled {
			_ = db.Close()
		}
	}()

	if err := deps.ensureSchemas(ctx, db); err != nil {
		return fmt.Errorf("schema initialization failed: %w", err)
	}
	primaryFileRoots, err := deps.storageRoots()
	if err != nil {
		return fmt.Errorf("file storage configuration failed: %w", err)
	}
	fileRoots := filerouting.New(primaryFileRoots)
	deploymentRecords, err := deploy.NewSQLRecordRepository(db)
	if err != nil {
		return fmt.Errorf("deployment record repository: %w", err)
	}
	var deploymentService *deploy.Service
	kuboBinary, kuboErr := managedKuboBinary()
	if kuboErr != nil {
		log.Printf("deployment storage unavailable: %v", kuboErr)
	} else {
		root, rootErr := scenarioRoot()
		if rootErr != nil {
			log.Printf("deployment reference bundle unavailable: %v", rootErr)
		} else {
			referencePath := filepath.Join(root, "examples", "reference")
			if info, statErr := os.Stat(referencePath); statErr != nil || !info.IsDir() {
				log.Printf("deployment reference bundle unavailable at %s", referencePath)
			} else {
				turboPath, pathErr := turboSandboxPathRecord(root)
				if pathErr != nil {
					return fmt.Errorf("Turbo sandbox path catalogue: %w", pathErr)
				}
				gate := deploy.Gate{Mode: deploy.GateStub, AllowFreeSandbox: true}
				storageAdapters := map[string]deploy.StorageAdapter{
					"kubo": deploy.KuboAdapter{
						Binary: kuboBinary,
						Repo:   filepath.Join(primaryFileRoots.DataDir, "deployment", "kubo"),
						Entry:  "site/index.html",
						Gate:   gate,
					},
					"turbo-sandbox": deploy.TurboSandboxAdapter{Gate: gate, Policy: turboPath},
				}
				deploymentService, err = deploy.NewServiceWithRecords(
					gate,
					deploy.NewFaults(),
					deploy.StorageRouter{Default: "kubo", Adapters: storageAdapters},
					deploymentRecords,
				)
				if err != nil {
					return fmt.Errorf("deployment service: %w", err)
				}
			}
		}
	}
	scenarioPath, err := scenarioRoot()
	if err != nil {
		return fmt.Errorf("scenario path: %w", err)
	}

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, scenarioName+"-api", "1.0.0"),
		capsH.Module(capabilities.NewRegistry()),
		deployH.ConnectModule(deploymentService, deploymentRecords, filepath.Join(scenarioPath, "examples", "reference")),
		notesH.Module(db, schedule.System(), log.Default()), // EXAMPLE-DOMAIN:notes
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.RegisterWithFileRoots(rootMux, db, fileRoots)

	// EXAMPLE-DOMAIN:notes START
	// /measures is the measures-go serve substrate: the central measures
	// index (measures-health) harvests <prefix>/declarations and the
	// auto-execution path POSTs <prefix>/execute. The notes domain owns the
	// one reference measure (notes.count); a real multi-domain scenario
	// registers each domain's measures on one shared registry here.
	notesMeasures, err := deps.measuresHandler(db)
	if err != nil {
		return fmt.Errorf("measures registry: %w", err)
	}
	rootMux.Handle("/measures/", http.StripPrefix("/measures", notesMeasures))
	// EXAMPLE-DOMAIN:notes END

	rootMux.Handle("/", srv.Handler())
	authConfig, err := deps.authConfig(os.Getenv)
	if err != nil {
		return fmt.Errorf("authentication configuration failed: %w", err)
	}
	var sharedAuth *authn.Config
	if authConfig.Enabled() {
		sharedAuth = &authConfig
	}

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// request context so *database.RoutedDB routes the call to the
	// installed test pool. Self-disables in production mode.
	handler := apihttp.TestModeMiddleware(rootMux)

	if err := deps.runServer(apiserver.Config{
		Handler:        handler,
		Authentication: sharedAuth,
		Cleanup: func(ctx context.Context) error {
			serverCleanupCalled = true
			return db.Close()
		},
	}); err != nil {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}
