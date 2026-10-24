package runs

import (
	"vrooli-bridge/internal/module"

	runsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs/runs_v1connect"
)

// Endpoints is the machine-readable description of the runs module's public
// surface. Connect-RPC method paths reference the generated *Procedure
// constants, so renaming an RPC in runs.proto breaks this file at compile time.
// The global parity test (TestProtoConnectParity) asserts every rpc has exactly
// one entry here once runs is listed in AllProtoFiles().
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "runs_get_run",
		Path:        runsconnect.RunsServiceGetRunProcedure,
		Method:      "POST",
		Summary:     "Get a run by id with its event history",
		Description: "Returns one durable run and its full persisted event history (logs/status/exit/artifact refs). Re-attaching after a client disconnect is just calling this again. Owner-gated.",
		Category:    "runs",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"run": "Run", "events": "array<RunEvent>"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 404, Code: "not_found", Description: "No run with that id"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "Get run", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.runs.RunsService/GetRun -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{\"id\":\"run-123\"}'"},
		},
	},
	{
		ID:          "runs_list_runs",
		Path:        runsconnect.RunsServiceListRunsProcedure,
		Method:      "POST",
		Summary:     "List runs",
		Description: "Returns runs newest-first, optionally filtered by node. Owner-gated.",
		Category:    "runs",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"node_id": "string", "limit": "int32"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"runs": "array<Run>"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "List runs", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.runs.RunsService/ListRuns -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "runs_wait_run",
		Path:        runsconnect.RunsServiceWaitRunProcedure,
		Method:      "POST",
		Summary:     "Block-once wait for a run to finish",
		Description: "Blocks server-side and returns EXACTLY ONCE when the run reaches a terminal status (no polling). Returns timed_out=true if the wait deadline elapses first. Owner-gated.",
		Category:    "runs",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string (required)", "timeout_seconds": "int64"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"run": "Run", "timed_out": "bool"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 404, Code: "not_found", Description: "No run with that id"},
			{Status: 499, Code: "canceled", Description: "Client cancelled the wait"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "Wait for run", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.runs.RunsService/WaitRun -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{\"id\":\"run-123\",\"timeout_seconds\":600}'"},
		},
	},
	{
		ID:          "runs_abort_run",
		Path:        runsconnect.RunsServiceAbortRunProcedure,
		Method:      "POST",
		Summary:     "Abort a run",
		Description: "Requests cancellation of a non-terminal run and marks it ABORTED. Idempotent on an already-terminal run. Owner-gated.",
		Category:    "runs",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string (required)", "reason": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"run": "Run"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 404, Code: "not_found", Description: "No run with that id"},
			{Status: 500, Code: "internal", Description: "Repository write failure"},
		},
		Examples: []module.Example{
			{Name: "Abort run", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.runs.RunsService/AbortRun -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{\"id\":\"run-123\",\"reason\":\"superseded\"}'"},
		},
	},
	{
		ID:          "runs_stream_run_events",
		Path:        runsconnect.RunsServiceStreamRunEventsProcedure,
		Method:      "POST",
		Summary:     "Follow a run's live event stream",
		Description: "Server-streams the run's persisted event history then tails live events until the run is terminal or the client disconnects. The human follow verb; agents use WaitRun. Owner-gated.",
		Category:    "runs",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string (required)"}},
		Response:    &module.Schema{Type: "stream", Properties: map[string]string{"event": "RunEvent"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 404, Code: "not_found", Description: "No run with that id"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "Follow run", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.runs.RunsService/StreamRunEvents -H 'Authorization: Bearer <token>' -H 'Content-Type: application/connect+json' -d '{\"id\":\"run-123\"}'"},
		},
	},
	{
		ID:          "runs_report_run_event",
		Path:        runsconnect.RunsServiceReportRunEventProcedure,
		Method:      "POST",
		Summary:     "Ingest a run event from the node-agent (node-facing)",
		Description: "Node-facing: the node-agent streams RunEvents back here, signed with its per-node Ed25519 credential. A node may only report against its own runs. A terminal EXIT flips the run terminal and wakes block-once waiters. Not an operator verb — omitted from the CLI manifest.",
		Category:    "runs",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"event": "RunEvent"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"accepted": "bool"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing run_id"},
			{Status: 401, Code: "unauthenticated", Description: "Node mutual-auth signature required"},
			{Status: 403, Code: "permission_denied", Description: "A node may only report its own runs"},
			{Status: 500, Code: "internal", Description: "Repository write failure"},
		},
		Examples: []module.Example{
			{Name: "Report run event (node-agent)", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.runs.RunsService/ReportRunEvent -H 'X-Bridge-Node: <node>' -H 'X-Bridge-Timestamp: <ts>' -H 'X-Bridge-Signature: <sig>' -H 'Content-Type: application/json' -d '{\"event\":{\"run_id\":\"run-123\",\"kind\":\"RUN_EVENT_KIND_LOG\",\"log_chunk\":\"...\"}}'"},
		},
	},
}
