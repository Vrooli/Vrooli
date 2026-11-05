package jobs

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/jobs/jobs_v1connect"
	"nutrition-planner/internal/entitlements"
	"nutrition-planner/internal/jobs"
	"nutrition-planner/internal/module"
	"nutrition-planner/internal/workspace"
)

func Module(repo jobs.Repository, workspaces workspace.Service, logger *log.Logger, entitlementRepos ...entitlements.Repository) module.Module {
	path, handler := connect.NewJobsServiceHandler(NewConnectHandler(repo, workspaces, logger, entitlementRepos...))
	return module.Module{Name: "jobs", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

func Schema() string { return jobs.Schema() }
