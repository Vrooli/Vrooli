package rehearsals

import (
	"client-rehearsal/internal/logx"
	"strings"

	"client-rehearsal/internal/module"
	internal "client-rehearsal/internal/rehearsals"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/client-rehearsal/v1/rehearsals/rehearsals_v1connect"
)

func Module(db *database.RoutedDB, clock schedule.Clock, logger logx.Logger) module.Module {
	return ModuleWithWorkspaceSandbox(db, clock, logger, nil)
}

func ModuleWithWorkspaceSandbox(db *database.RoutedDB, clock schedule.Clock, logger logx.Logger, sandbox *internal.WorkspaceSandboxProfileClient) module.Module {
	return moduleWithRuntime(db, clock, logger, sandbox, nil)
}

// ModuleWithNativeN8N enables the managed n8n runtime through its owner CLI.
// The n8n resource remains responsible for image admission, runtime isolation,
// and task cleanup.
func ModuleWithNativeN8N(db *database.RoutedDB, clock schedule.Clock, logger logx.Logger, sandbox *internal.WorkspaceSandboxProfileClient, cliPath string, environment func() []string) module.Module {
	if strings.TrimSpace(cliPath) == "" || environment == nil {
		panic("rehearsals.ModuleWithNativeN8N requires an owner CLI and environment")
	}
	runtime := internal.N8NWorkflowRuntimeCLI{Binary: cliPath, Environment: environment}
	return moduleWithRuntime(db, clock, logger, sandbox, runtime)
}

func moduleWithRuntime(db *database.RoutedDB, clock schedule.Clock, logger logx.Logger, sandbox *internal.WorkspaceSandboxProfileClient, runtime internal.N8NWorkflowRuntime) module.Module {
	if logger == nil {
		panic("rehearsals.Module requires Logger")
	}
	if clock == nil {
		clock = schedule.System()
	}
	var service internal.Service
	if sandbox == nil {
		service = internal.NewService(internal.NewSQLiteRepository(db), internal.SyntheticEngine{}, clock.Now)
	} else if runtime != nil {
		service = internal.NewServiceWithNativeN8N(internal.NewSQLiteRepository(db), internal.SyntheticEngine{}, clock.Now, sandbox, runtime)
	} else {
		service = internal.NewServiceWithWorkspaceSandbox(internal.NewSQLiteRepository(db), internal.SyntheticEngine{}, clock.Now, sandbox)
	}
	path, handler := connect.NewRehearsalServiceHandler(NewConnectHandler(service, logger))
	return module.Module{Name: "rehearsals", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

func Schema() string { return internal.Schema() }
