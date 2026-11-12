package operatorapi

import (
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	"research-desk/internal/module"
	"research-desk/internal/wire"

	operatorconnect "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/operator/operator_v1connect"
)

// Module mounts the operator Connect service and the read-only file route.
func Module(deps Deps) module.Module {
	path, handler := operatorconnect.NewOperatorServiceHandler(NewHandler(deps), wire.OperatorOnly(OperatorProcedures...))
	return module.Module{
		Name: "operator",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
			r.PathPrefix(FileRoute).Methods("GET", "HEAD").Handler(fileHandler(deps.ResearchDir))
		},
		Endpoints: Endpoints,
	}
}

// OperatorProcedures act for the operator; a caller verified as an agent or
// a service is refused them (rulings F13). The reads stay open.
var OperatorProcedures = []string{
	operatorconnect.OperatorServiceAnswerProcedure,
	operatorconnect.OperatorServiceAddEntryProcedure,
	operatorconnect.OperatorServiceEnableSteeringProcedure,
	operatorconnect.OperatorServiceAddVerdictProcedure,
	operatorconnect.OperatorServiceSetCourierProcedure,
	operatorconnect.OperatorServiceTestCourierProcedure,
	operatorconnect.OperatorServiceMarkSeenProcedure,
}

var notOperator = module.ErrorDesc{Status: 403, Code: "permission_denied", Description: "The caller is an agent or a service: " + wire.OperatorAction}

const curlBase = "curl http://localhost:${API_PORT}/vrooli.research_desk.v1.operator.OperatorService/"

var (
	readOnly      = module.ErrorDesc{Status: 412, Code: "failed_precondition", Description: "Another instance holds the write lock, or the research home cannot be read; the message says which"}
	invalid       = module.ErrorDesc{Status: 400, Code: "invalid_argument", Description: "The request names an option, kind or text the file grammar refuses; the message says what is allowed"}
	noCommission  = module.ErrorDesc{Status: 404, Code: "not_found", Description: "No such commission, item, review or topic"}
	notAnswerable = module.ErrorDesc{Status: 412, Code: "failed_precondition", Description: "The item is a secret or has no commission to read an answer; the message says where to act"}
)

func example(name, rpc, body string) []module.Example {
	return []module.Example{{Name: name, Curl: curlBase + rpc + " -H 'Content-Type: application/json' -d '" + body + "'"}}
}

// Endpoints describes the operator surface. Paths come from the generated
// procedure constants, so a renamed RPC breaks this file at compile time.
var Endpoints = []module.EndpointDescriptor{
	{
		ID: "operator_needs_you", Path: operatorconnect.OperatorServiceListNeedsYouProcedure, Method: "POST", Category: "operator",
		Summary:     "What needs the operator",
		Description: "Needs-you items of every commission that has not ended and of ATTENTION.md, merged by commission and ID, with answers, whether a unit has run since, and what the courier would do for each (the dry run).",
		Errors:      []module.ErrorDesc{readOnly},
		Examples:    example("Feed", "ListNeedsYou", "{}"),
	},
	{
		ID: "operator_answer", Path: operatorconnect.OperatorServiceAnswerProcedure, Method: "POST", Category: "operator",
		Summary:     "Answer an item",
		Description: "Appends `answer <ID> · via desk` to the commission's OPERATOR.md with the chosen option, or `other` and the operator's words. dry_run returns the exact text without writing; every answer says when the supervisor will read it.",
		Errors:      []module.ErrorDesc{readOnly, invalid, noCommission, notAnswerable, notOperator},
		Examples:    example("Keep", "Answer", `{"commission_id":"programme-five-20261112","item_id":"AV4","option_key":"keep"}`),
	},
	{
		ID: "operator_entry", Path: operatorconnect.OperatorServiceAddEntryProcedure, Method: "POST", Category: "operator",
		Summary:     "Give direction or wind down",
		Description: "Appends a direction or wind-down entry, in the operator's words, to the commission's OPERATOR.md; nothing is typed into the supervisor's conversation. dry_run returns the exact text without writing; every answer says when the supervisor will read it.",
		Errors:      []module.ErrorDesc{readOnly, invalid, noCommission, notOperator},
		Examples:    example("Direction", "AddEntry", `{"commission_id":"programme-five-20261112","kind":"ENTRY_KIND_DIRECTION","text":"Put AV6 first."}`),
	},
	{
		ID: "operator_steering", Path: operatorconnect.OperatorServiceEnableSteeringProcedure, Method: "POST", Category: "operator",
		Summary:     "Enable steering on an older charter",
		Description: "Appends the Steering section to a charter that does not tell the scheduler to read OPERATOR.md; changes nothing when it already does.",
		Errors:      []module.ErrorDesc{readOnly, noCommission, notOperator},
		Examples:    example("Enable", "EnableSteering", `{"commission_id":"programme-four-20261110"}`),
	},
	{
		ID: "operator_reviews", Path: operatorconnect.OperatorServiceListReviewsProcedure, Method: "POST", Category: "operator",
		Summary:     "List reviews",
		Description: "Every reviews/<id>/ folder: REVIEW.md, REVIEW.html, proposals with file-route URLs, the topics reviewed and the verdicts recorded.",
		Errors:      []module.ErrorDesc{readOnly},
		Examples:    example("Reviews", "ListReviews", "{}"),
	},
	{
		ID: "operator_verdict", Path: operatorconnect.OperatorServiceAddVerdictProcedure, Method: "POST", Category: "operator",
		Summary:     "Record a verdict",
		Description: "Appends the operator's verdict on one topic, verbatim, to reviews/<id>/VERDICTS.md. dry_run returns the exact text without writing.",
		Errors:      []module.ErrorDesc{readOnly, invalid, noCommission, notOperator},
		Examples:    example("Verdict", "AddVerdict", `{"review_id":"ai-video-20261112","topic":"ai-video-production-and-funnels","text":"Films first."}`),
	},
	{
		ID: "operator_courier_set", Path: operatorconnect.OperatorServiceSetCourierProcedure, Method: "POST", Category: "operator",
		Summary:     "Turn the courier on or off",
		Description: "Records the operator's courier switch for one commission or for global ATTENTION.md items in the desk's state folder.",
		Errors:      []module.ErrorDesc{readOnly, invalid, noCommission, notOperator},
		Examples:    example("On", "SetCourier", `{"commission_id":"programme-five-20261112","on":true}`),
	},
	{
		ID: "operator_courier_test", Path: operatorconnect.OperatorServiceTestCourierProcedure, Method: "POST", Category: "operator",
		Summary:     "Test delivery",
		Description: "Raises one research-desk delivery-test ask through Vrooli Events and waits up to a minute for a device to receive it. Nothing is answered; the ask expires 15 minutes later.",
		Errors:      []module.ErrorDesc{readOnly, {Status: 503, Code: "unavailable", Description: "Notification Hub refused or could not be reached; the message quotes its answer"}, notOperator},
		Examples:    example("Test", "TestCourier", "{}"),
	},
	{
		ID: "operator_mark_seen", Path: operatorconnect.OperatorServiceMarkSeenProcedure, Method: "POST", Category: "operator",
		Summary:     "Record a look",
		Description: "Records the operator's look at a commission or topic (ledger sizes, checkpoint times, workstream statuses) in the desk state folder, or with all at every commission and topic (caught up), so GetChanges reports from now.",
		Errors:      []module.ErrorDesc{readOnly, noCommission, notOperator},
		Examples:    append(example("Look", "MarkSeen", `{"subject":"CHANGE_SUBJECT_COMMISSION","id":"programme-four-20261110"}`), example("Caught up", "MarkSeen", `{"all":true}`)...),
	},
	{
		ID: "operator_home_file", Path: FileRoute + "{path}", Method: "GET", Category: "operator",
		Summary:     "Read a research-home file",
		Description: "Serves one file under the research home read-only (range requests for media), so REVIEW.html and the media it links open in the browser. HTML is sandboxed: no scripts.",
		Errors:      []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "No such file, a hidden path, or a link outside the research home"}},
		Examples:    []module.Example{{Name: "A review page", Curl: "curl http://localhost:${API_PORT}" + FileRoute + "reviews/<id>/REVIEW.html"}},
		RESTException: &module.RESTException{
			Reason: module.RESTReasonBrowserSurface,
			Note:   "Raw file bytes for <video>, <img> and an iframe; a browser element cannot call a Connect procedure.",
			ProtoPayloads: &module.RESTProtoPayloads{
				Request:  module.RESTPayload{Transport: "none", Conformance: "none"},
				Response: module.RESTPayload{Transport: "binary", Conformance: "external_shape"},
				Error:    module.RESTPayload{ProtoFullName: "vrooli.research_desk.v1.shared.ErrorEnvelope", Transport: "json", Conformance: "protojson"},
			},
		},
	},
}
