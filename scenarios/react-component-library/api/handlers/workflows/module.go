package workflows

import (
	"database/sql"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	workflowsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workflows/workflows_v1connect"

	"react-component-library/internal/clock"
	"react-component-library/internal/module"
	internal "react-component-library/internal/workflows"
)

func Module(db *sql.DB, clk clock.Clock, dispatcher internal.Dispatcher, logger *log.Logger) module.Module {
	svc := internal.NewService(internal.NewSQLiteRepository(db, clk), dispatcher)
	path, h := workflowsconnect.NewWorkflowsServiceHandler(NewConnectHandler(svc, logger))
	return module.Module{Name: "workflows", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h}) }, Endpoints: Endpoints}
}
func Schema() string { return internal.Schema() }

var Endpoints = []module.EndpointDescriptor{
	{ID: "workflows_start", Path: workflowsconnect.WorkflowsServiceStartWorkflowProcedure, Method: "POST", Summary: "Start assisted catalog work", Category: "workflows"},
	{ID: "workflows_list", Path: workflowsconnect.WorkflowsServiceListWorkflowsProcedure, Method: "POST", Summary: "List assisted catalog work", Category: "workflows"},
	{ID: "workflows_get", Path: workflowsconnect.WorkflowsServiceGetWorkflowProcedure, Method: "POST", Summary: "Get assisted catalog work", Category: "workflows"},
	{ID: "workflows_refresh", Path: workflowsconnect.WorkflowsServiceRefreshWorkflowProcedure, Method: "POST", Summary: "Refresh assisted catalog work", Category: "workflows"},
	{ID: "workflows_stop", Path: workflowsconnect.WorkflowsServiceStopWorkflowProcedure, Method: "POST", Summary: "Stop assisted catalog work", Category: "workflows"},
	{ID: "workflows_retry", Path: workflowsconnect.WorkflowsServiceRetryWorkflowProcedure, Method: "POST", Summary: "Retry assisted catalog work", Category: "workflows"},
	{ID: "workflows_promotion_readiness", Path: workflowsconnect.WorkflowsServiceGetPromotionReadinessProcedure, Method: "POST", Summary: "Read promotion readiness evidence", Category: "workflows"},
}

func ModuleWithReadiness(db *sql.DB, clk clock.Clock, dispatcher internal.Dispatcher, readiness internal.PromotionReadinessReader, logger *log.Logger) module.Module {
	svc := internal.NewService(internal.NewSQLiteRepository(db, clk), dispatcher, readiness)
	path, h := workflowsconnect.NewWorkflowsServiceHandler(NewConnectHandler(svc, logger))
	return module.Module{Name: "workflows", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h}) }, Endpoints: Endpoints}
}
