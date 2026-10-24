package investigation

import (
	"context"
	"log"

	internalexecution "plan-manager/internal/execution"
	internalbrief "plan-manager/internal/investigationbrief"
	internalpolicy "plan-manager/internal/investigationpolicy"
	"plan-manager/internal/module"
	planmodel "plan-manager/internal/planmodel"
	internalplans "plan-manager/internal/plans"
	internalvalidation "plan-manager/internal/validation"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	investigationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/investigation/investigationconnect"
)

func Module(db *database.RoutedDB, clock schedule.Clock, logger *log.Logger) module.Module {
	repository := internalpolicy.NewRepository(db, clock)
	plans := internalplans.NewService(internalplans.Deps{Repo: internalplans.NewSQLiteRepository(db, clock), Clock: clock})
	brief := internalbrief.NewProvider(
		planSourceAdapter{svc: plans},
		internalexecution.NewSQLiteRepository(db, clock),
		internalvalidation.NewSQLiteResultStore(db, clock),
		clock,
	)
	path, handler := investigationconnect.NewInvestigationPolicyServiceHandler(NewConnectHandlerWithRunnerAndBrief(repository, logger, NewDiscoveredProgramRunner(), brief))
	return module.Module{
		Name: "investigation",
		Mount: func(router *mux.Router) {
			connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return internalpolicy.Schema() }

var Endpoints = []module.EndpointDescriptor{
	{ID: "investigation_policy_get", Path: investigationconnect.InvestigationPolicyServiceGetPolicyProcedure, Method: "POST", Summary: "Get investigation trigger policy", Description: "Returns the active versioned Plan Manager investigation trigger policy.", Category: "investigation"},
	{ID: "investigation_policy_put", Path: investigationconnect.InvestigationPolicyServicePutPolicyProcedure, Method: "POST", Summary: "Store investigation trigger policy", Description: "Stores a reviewed versioned Plan Manager investigation trigger policy.", Category: "investigation"},
	{ID: "investigation_trigger_preview", Path: investigationconnect.InvestigationPolicyServicePreviewTriggerProcedure, Method: "POST", Summary: "Preview investigation trigger", Description: "Evaluates trigger eligibility from execution state without reading transcripts or dispatching an agent.", Category: "investigation"},
	{ID: "investigation_trigger_record", Path: investigationconnect.InvestigationPolicyServiceRecordTriggerProcedure, Method: "POST", Summary: "Record investigation trigger", Description: "Persists an eligible trigger incident for later Agent Manager linkage.", Category: "investigation"},
	{ID: "investigation_trigger_link", Path: investigationconnect.InvestigationPolicyServiceLinkTriggerProcedure, Method: "POST", Summary: "Link investigation trigger", Description: "Links an Agent Manager investigation to a persisted Plan Manager trigger incident.", Category: "investigation"},
	{ID: "investigation_incidents_list", Path: investigationconnect.InvestigationPolicyServiceListIncidentsProcedure, Method: "POST", Summary: "List investigation incidents", Description: "Lists bounded durable trigger incidents with occurrence counts and linked investigation state.", Category: "investigation"},
	{ID: "investigation_incident_get", Path: investigationconnect.InvestigationPolicyServiceGetIncidentProcedure, Method: "POST", Summary: "Get investigation incident", Description: "Reads one durable trigger incident and its bounded occurrence history.", Category: "investigation"},
	{ID: "investigation_occurrences_list", Path: investigationconnect.InvestigationPolicyServiceListOccurrencesProcedure, Method: "POST", Summary: "List investigation trigger occurrences", Description: "Lists bounded trigger evaluations, including suppressed evaluations that did not create an incident.", Category: "investigation"},
	{ID: "investigation_brief_get", Path: investigationconnect.InvestigationPolicyServiceGetBriefProcedure, Method: "POST", Summary: "Get authoritative investigation brief", Description: "Returns a bounded execution- and phase-scoped brief with authoritative identities, validation evidence, producer wait, and budget standing.", Category: "investigation"},
}

type planSourceAdapter struct{ svc internalplans.Service }

func (a planSourceAdapter) GetPlan(ctx context.Context, id string) (planmodel.Plan, error) {
	return a.svc.Get(ctx, id, internalplans.WorkspaceScope{})
}
