package recovery

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"tunnel-manager/internal/authz"
	"tunnel-manager/internal/clock"
	"tunnel-manager/internal/cmdrunner"
	"tunnel-manager/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	recoveryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/recovery/recovery_v1connect"

	internalrecovery "tunnel-manager/internal/recovery"
)

// Module returns the recovery domain's contribution to the API: the
// generated Connect-RPC RecoveryService handler backed by the live
// recovery engine.
//
// The engine is a long-lived stateful singleton (it holds the circuit
// breaker / backoff state machine), so it is constructed once here and
// closed over by the handler. The readiness probe and the cloudflared
// restart go through the httpc / cmdrunner seams.
func Module(db *database.RoutedDB, clk clock.Clock, logger *log.Logger) module.Module {
	return ModuleWithService(NewProductionService(db, clk), logger)
}

// NewProductionService wires the recovery engine with the same production
// seams used by the Connect handler and the optional background scheduler.
func NewProductionService(db *database.RoutedDB, clk clock.Clock) internalrecovery.Service {
	repo := internalrecovery.NewSQLiteRepository(db, clk)
	readyURL := strings.TrimSpace(os.Getenv("TUNNEL_READY_URL"))
	if readyURL == "" {
		readyURL = internalrecovery.DefaultReadyURL
	}
	health := internalrecovery.NewHTTPHealthChecker(&http.Client{Timeout: 5 * time.Second}, readyURL)
	return internalrecovery.NewService(repo, health, cmdrunner.Default, clk, internalrecovery.Config{}, nil)
}

func ModuleWithService(svc internalrecovery.Service, logger *log.Logger) module.Module {
	connectPath, connectHandler := recoveryconnect.NewRecoveryServiceHandler(NewConnectHandler(Deps{
		Service:    svc,
		Logger:     logger,
		Authorizer: authz.FromEnv(),
	}))
	return module.Module{
		Name: "recovery",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports internalrecovery.Schema so the modules registry
// collects endpoint descriptors and schema from one symbol per handler
// package.
func Schema() string { return internalrecovery.Schema() }
