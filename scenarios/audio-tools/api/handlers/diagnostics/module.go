// Package diagnostics hosts the DiagnosticsService Connect-RPC handler:
// a one-click capability suite that exercises STT, TTS, Summarize and
// Transcode end-to-end against bundled fixtures and records the most
// recent run in memory.
package diagnostics

import (
	"log"

	"audio-tools/internal/diagnostics"
	"audio-tools/internal/modulekit"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	diagconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/diagnostics/diagnostics_v1connect"
)

// Deps wires the handler. Orchestrator is required.
type Deps struct {
	Orchestrator *diagnostics.Orchestrator
	Logger       *log.Logger
}

// Module returns the diagnostics module contribution.
func Module(orch *diagnostics.Orchestrator, logger *log.Logger) modulekit.Module {
	if logger == nil {
		logger = log.Default()
	}
	connectPath, h := diagconnect.NewDiagnosticsServiceHandler(NewConnectHandler(Deps{Orchestrator: orch, Logger: logger}))
	return modulekit.Module{
		Name: "diagnostics",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: h})
		},
		Endpoints: Endpoints,
	}
}

// Schema reports the diagnostics module's SQL contribution. None: the
// store is in-memory.
func Schema() string { return "" }

// Endpoints lists the Connect procedures for codegen / docs.
var Endpoints = []modulekit.EndpointDescriptor{
	{
		ID: "diagnostics.run_suite", Path: "/vrooli.audio_tools.v1.diagnostics.DiagnosticsService/RunSuite",
		Method: "POST", Summary: "Run the audio-tools capability suite end-to-end against bundled fixtures",
		Category:   "diagnostics",
		CLIMapping: &modulekit.CLIMapping{Command: "audio-tools diagnostics run", Args: []string{"[--capability stt,tts,summarize,transcode]", "[--json]"}},
	},
	{
		ID: "diagnostics.get_last_run", Path: "/vrooli.audio_tools.v1.diagnostics.DiagnosticsService/GetLastRun",
		Method: "POST", Summary: "Return the most recent RunSuite result, or an empty envelope when no run has executed",
		Category:   "diagnostics",
		CLIMapping: &modulekit.CLIMapping{Command: "audio-tools diagnostics last", Args: []string{"[--json]"}},
	},
	{
		ID: "diagnostics.list_fixtures", Path: "/vrooli.audio_tools.v1.diagnostics.DiagnosticsService/ListFixtures",
		Method: "POST", Summary: "Describe the bundled diagnostics fixtures",
		Category: "diagnostics",
	},
}
