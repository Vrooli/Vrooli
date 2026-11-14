package activityapi

import (
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	"research-desk/internal/module"

	activityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/activity/activity_v1connect"
)

// Module mounts the activity Connect service.
func Module(deps Deps) module.Module {
	path, handler := activityconnect.NewActivityServiceHandler(NewHandler(deps))
	return module.Module{
		Name: "activity",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

const curlBase = "curl http://localhost:${API_PORT}/vrooli.research_desk.v1.activity.ActivityService/"

var unreadableHome = module.ErrorDesc{Status: 412, Code: "failed_precondition", Description: "The research home cannot be read; the message names the path"}

// Endpoints describes the activity surface. Paths come from the generated
// procedure constants, so a renamed RPC breaks this file at compile time.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "activity_overview",
		Path:        activityconnect.ActivityServiceGetOverviewProcedure,
		Method:      "POST",
		Summary:     "What is moving",
		Description: "State counts (live, paused and limited apart), commissions with a supervisor window open, overlapping commissions and the lenses read, derived from the research home, Codex session records and the process table.",
		Category:    "activity",
		Errors:      []module.ErrorDesc{unreadableHome},
		Examples:    []module.Example{{Name: "Overview", Curl: curlBase + "GetOverview -H 'Content-Type: application/json' -d '{}'"}},
	},
	{
		ID:          "activity_commissions",
		Path:        activityconnect.ActivityServiceListCommissionsProcedure,
		Method:      "POST",
		Summary:     "List commissions",
		Description: "Every commission folder with its derived state, topics, times and claim, newest last write first; optionally filtered by state.",
		Category:    "activity",
		Errors:      []module.ErrorDesc{unreadableHome},
		Examples:    []module.Example{{Name: "Running commissions", Curl: curlBase + "ListCommissions -H 'Content-Type: application/json' -d '{\"states\":[\"COMMISSION_STATE_LIVE\"]}'"}},
	},
	{
		ID:          "activity_commission",
		Path:        activityconnect.ActivityServiceGetCommissionProcedure,
		Method:      "POST",
		Summary:     "One commission",
		Description: "One commission with every session match, its files, run folders and the read sets of its scheduler and owners measured against limits.json.",
		Category:    "activity",
		Errors:      []module.ErrorDesc{unreadableHome, {Status: 404, Code: "not_found", Description: "No commission folder with that id"}},
		Examples:    []module.Example{{Name: "Programme three", Curl: curlBase + "GetCommission -H 'Content-Type: application/json' -d '{\"id\":\"programme-three-20261111\"}'"}},
	},
	{
		ID:          "activity_topics",
		Path:        activityconnect.ActivityServiceListTopicsProcedure,
		Method:      "POST",
		Summary:     "List topics",
		Description: "Every topic with its last activity, the commissions naming it and mission_check.py's workstream summary.",
		Category:    "activity",
		Errors:      []module.ErrorDesc{unreadableHome},
		Examples:    []module.Example{{Name: "Topics", Curl: curlBase + "ListTopics -H 'Content-Type: application/json' -d '{}'"}},
	},
	{
		ID:          "activity_changes",
		Path:        activityconnect.ActivityServiceGetChangesProcedure,
		Method:      "POST",
		Summary:     "Changes since the last look or a time",
		Description: "New ledger text, changed checkpoints, showcase lines, workstream status changes and files written since the operator's previous look (OperatorService.MarkSeen) at a commission or topic; with since, files and checkpoints written after that time. with neither, the last 24 hours. baseline and baseline_kind say which.",
		Category:    "activity",
		Errors:      []module.ErrorDesc{unreadableHome, {Status: 404, Code: "not_found", Description: "No commission or topic with that id"}},
		Examples:    []module.Example{{Name: "Commission changes", Curl: curlBase + "GetChanges -H 'Content-Type: application/json' -d '{\"subject\":\"CHANGE_SUBJECT_COMMISSION\",\"id\":\"programme-three-20261111\"}'"}},
	},
}
