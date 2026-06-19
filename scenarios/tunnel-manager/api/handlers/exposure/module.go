package exposure

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"

	"tunnel-manager/internal/clock"
	"tunnel-manager/internal/cmdrunner"
	"tunnel-manager/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/coreset"
	"github.com/vrooli/api-core/database"

	exposureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/exposure/exposure_v1connect"

	internalconfig "tunnel-manager/internal/config"
	internalexposure "tunnel-manager/internal/exposure"
	internalroutes "tunnel-manager/internal/routes"
)

// Module returns the exposure domain's contribution to the API: the
// generated Connect-RPC ExposureService handler backed by the tiered
// exposure broker.
//
// Exposure composes three sibling domains through narrow seams: the routes
// manifest (CRUD), the config domain's ingress Sync (wrapped as the
// Ingress seam), and process lifecycle (a cmdrunner-backed Runner). The
// CORE set comes from api-core/coreset; UI ports come from each scenario's
// service.json via the FilePortResolver.
func Module(db *database.RoutedDB, clk clock.Clock, logger *log.Logger) module.Module {
	return ModuleWithService(NewProductionService(db, clk), logger)
}

// NewProductionService wires the exposure broker with the same production
// seams used by the Connect handler and the background scheduler.
func NewProductionService(db *database.RoutedDB, clk clock.Clock) internalexposure.Service {
	repo := internalexposure.NewSQLiteRepository(db, clk)
	manifest := internalroutes.NewService(internalroutes.NewSQLiteRepository(db, clk))

	// Reuse the config service's Sync as the ingress reconciler so exposure
	// never owns Cloudflare calls or local cloudflared restart behavior.
	configSvc := internalconfig.NewProductionService(db, clk, internalconfig.ProductionOptions{})

	scenariosRoot := resolveScenariosRoot()
	return internalexposure.NewService(
		repo,
		manifest,
		ingressAdapter{cfg: configSvc},
		internalexposure.NewCLIRunner(cmdrunner.Default),
		internalexposure.NewFilePortResolver(scenariosRoot),
		coreset.CoreSeedScenarios,
		clk,
	)
}

func ModuleWithService(svc internalexposure.Service, logger *log.Logger) module.Module {
	connectPath, connectHandler := exposureconnect.NewExposureServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "exposure",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports internalexposure.Schema so the modules registry
// collects endpoint descriptors and schema from one symbol per handler.
func Schema() string { return internalexposure.Schema() }

// ingressAdapter wraps the config service's Sync as the exposure Ingress
// seam: a full reconcile of live ingress against the routes manifest.
type ingressAdapter struct {
	cfg internalconfig.Service
}

func (a ingressAdapter) Reconcile(ctx context.Context) error {
	_, err := a.cfg.Sync(ctx, false)
	return err
}

// resolveScenariosRoot finds the scenarios directory. VROOLI_SCENARIOS_ROOT
// wins; otherwise walk up from the working directory looking for a
// "scenarios" dir; failing that, fall back to "scenarios" relative to cwd.
func resolveScenariosRoot() string {
	if v := strings.TrimSpace(os.Getenv("VROOLI_SCENARIOS_ROOT")); v != "" {
		return v
	}
	cwd, err := os.Getwd()
	if err == nil {
		dir := cwd
		for {
			candidate := filepath.Join(dir, "scenarios")
			if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
				return candidate
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return "scenarios"
}
