package launchapi

import (
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	"research-desk/internal/module"
	"research-desk/internal/wire"

	launchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/launch/launch_v1connect"
)

// Module mounts the launch Connect service.
func Module(deps Deps) module.Module {
	path, handler := launchconnect.NewLaunchServiceHandler(NewHandler(deps), wire.OperatorOnly(OperatorProcedures...))
	return module.Module{
		Name: "launch",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

// OperatorProcedures act for the operator; a caller verified as an agent or
// a service is refused them (rulings F13). Build and Preflight stay open.
var OperatorProcedures = []string{
	launchconnect.LaunchServiceLaunchProcedure,
	launchconnect.LaunchServiceControlProcedure,
	launchconnect.LaunchServiceTestLaunchProcedure,
}

var notOperator = module.ErrorDesc{Status: 403, Code: "permission_denied", Description: "The caller is an agent or a service: " + wire.OperatorAction}

const curlBase = "curl http://localhost:${API_PORT}/vrooli.research_desk.v1.launch.LaunchService/"

var (
	readOnly    = module.ErrorDesc{Status: 412, Code: "failed_precondition", Description: "Another instance holds the write lock, the research home cannot be read, or the commission's state refuses this; the message says which and what to do"}
	noSuch      = module.ErrorDesc{Status: 404, Code: "not_found", Description: "No such commission under cohorts/"}
	invalid     = module.ErrorDesc{Status: 400, Code: "invalid_argument", Description: "Neither a spec nor a commission was given, or the action is unknown"}
	unavailable = module.ErrorDesc{Status: 503, Code: "unavailable", Description: "packet.py, Web Console or a Codex file could not be reached or read; the message quotes the cause"}
)

func example(name, rpc, body string) []module.Example {
	return []module.Example{{Name: name, Curl: curlBase + rpc + " -H 'Content-Type: application/json' -d '" + body + "'"}}
}

// Endpoints describes the launch surface. Paths come from the generated
// procedure constants, so a renamed RPC breaks this file at compile time.
var Endpoints = []module.EndpointDescriptor{
	{
		ID: "launch_options", Path: launchconnect.LaunchServiceGetLaunchOptionsProcedure, Method: "POST", Category: "launch",
		Summary:     "Launch defaults and choices",
		Description: "Supervisor model, effort and owner tier defaults and allowed values, and the packet gates, from the skill's limits.json; whether Web Console is reachable for an automatic launch. For the topics named: suggested stance quotes with their source (the newest commission's PACKET.json, else the quoted lines of its STANCE-MESSAGE.txt) and each topic's workstreams with title and showcase line.",
		Errors:      []module.ErrorDesc{unavailable},
		Examples:    example("Options", "GetLaunchOptions", `{"topics":["niche-book-production"]}`),
	},
	{
		ID: "launch_preflight", Path: launchconnect.LaunchServicePreflightProcedure, Method: "POST", Category: "launch",
		Summary:     "Preflight a spec or a prepared packet",
		Description: "Runs packet.py preview on a spec (or check on a commission), and says for every earlier commission on the same topics whether it blocks the launch, the choices that clear it (supersede or run alongside) with the desk's recommendation, and which checkpoint to read first. Resolutions and the never-started assertion are judged as a launch would judge them.",
		Errors:      []module.ErrorDesc{invalid, noSuch, unavailable},
		Examples:    example("Spec", "Preflight", `{"spec_json":"{\"topics\":[\"niche-book-production\"],\"allowance_hours\":12,\"stance_quotes\":[\"Books first.\"]}"}`),
	},
	{
		ID: "launch_build", Path: launchconnect.LaunchServiceBuildProcedure, Method: "POST", Category: "launch",
		Summary:     "Build a packet",
		Description: "Runs packet.py build to write a new commission folder from a spec, or from an earlier commission's PACKET.json with supersedes and carry_baseline_from set (a relaunch is a new commission). resolutions are written into the spec first, so the charter names the commission it supersedes (one) and each it runs alongside.",
		Errors:      []module.ErrorDesc{readOnly, invalid, noSuch, unavailable},
		Examples:    example("Relaunch", "Build", `{"relaunch_of":"programme-five-20261112"}`),
	},
	{
		ID: "launch_start", Path: launchconnect.LaunchServiceLaunchProcedure, Method: "POST", Category: "launch",
		Summary:     "Launch a commission",
		Description: "Refuses unless the packet passes, every blocker is resolved (supersede or run alongside) and the commission is prepared, or unknown with the operator's assertion that it never started and no session or goal found for it. Writes LAUNCH.json with the choices, then in the background stops each superseded supervisor through its own session and records it, creates the Web Console session, sends the stance, waits for ready, sets the goal and verifies it.",
		Errors:      []module.ErrorDesc{readOnly, noSuch, unavailable, notOperator},
		Examples:    example("Supersede", "Launch", `{"commission_id":"programme-five-20261112","resolutions":[{"commission_id":"programme-four-20261110","choice":"RESOLUTION_SUPERSEDE"}]}`),
	},
	{
		ID: "launch_messages", Path: launchconnect.LaunchServiceGetLaunchMessagesProcedure, Method: "POST", Category: "launch",
		Summary:     "The two launch messages",
		Description: "A built packet's charter path, stance message and kernel, read from its folder, for a launch by hand; writes nothing.",
		Errors:      []module.ErrorDesc{noSuch},
		Examples:    example("Messages", "GetLaunchMessages", `{"commission_id":"programme-five-20261112"}`),
	},
	{
		ID: "launch_get", Path: launchconnect.LaunchServiceGetLaunchProcedure, Method: "POST", Category: "launch",
		Summary:     "Launch record",
		Description: "The commission's LAUNCH.json steps, failure, discard and controls, and whether an attempt is running in this instance.",
		Errors:      []module.ErrorDesc{noSuch},
		Examples:    example("Record", "GetLaunch", `{"commission_id":"programme-five-20261112"}`),
	},
	{
		ID: "launch_control", Path: launchconnect.LaunchServiceControlProcedure, Method: "POST", Category: "launch",
		Summary:     "Pause, resume, stop, discard, re-send stance",
		Description: "Types /goal pause or /goal resume, pastes the stance message again, or pauses the goal and quits Codex and its app-server; discard also records the operator's reason. The goal status is read back from goals_1.sqlite and the control is recorded in LAUNCH.json. Sessions are never deleted.",
		Errors:      []module.ErrorDesc{readOnly, invalid, noSuch, unavailable, notOperator},
		Examples:    example("Pause", "Control", `{"commission_id":"programme-five-20261112","action":"CONTROL_ACTION_PAUSE"}`),
	},
	{
		ID: "launch_test", Path: launchconnect.LaunchServiceTestLaunchProcedure, Method: "POST", Category: "launch",
		Summary:     "Start a launch test",
		Description: "Starts, in the background, one throwaway labelled session with a harmless stance and a kernel of about 600 characters that completes at once, then pauses, resumes and stops it. Writes nothing in the research home; the session row stays. Read the outcome with launch_test_get.",
		Errors:      []module.ErrorDesc{readOnly, unavailable, notOperator},
		Examples:    example("Test", "TestLaunch", "{}"),
	},
	{
		ID: "launch_test_get", Path: launchconnect.LaunchServiceGetLaunchTestProcedure, Method: "POST", Category: "launch",
		Summary:     "Newest launch test",
		Description: "Each step and the goal status read back after the goal, pause, resume and at the end; whether the session's Codex processes were stopped; where it failed.",
		Errors:      []module.ErrorDesc{unavailable},
		Examples:    example("Newest", "GetLaunchTest", "{}"),
	},
}
