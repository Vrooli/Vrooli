package jobs

import (
	"log"

	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	jobsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs/jobs_v1connect"
)

// Module returns the jobs domain's contribution: the generated Connect-RPC
// JobsService handler over the server-owned durable job Manager. The Manager is
// constructed and Started in main.go (it needs the server-lifetime context so a
// client disconnect never destroys in-flight work); this module only exposes its
// lifecycle verbs.
func Module(mgr JobManager, logger *log.Logger) module.Module {
	connectPath, connectHandler := jobsconnect.NewJobsServiceHandler(NewConnectHandler(Deps{
		Manager: mgr,
		Logger:  logger,
	}))
	return module.Module{
		Name: "jobs",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports internaljobs.Schema so the modules registry collects
// endpoint descriptors and schema from one symbol per handler package.
func Schema() string { return internaljobs.Schema() }

// Endpoints describes the jobs module's public surface. Connect-RPC method paths
// reference the generated *Procedure constants, so renaming an RPC in jobs.proto
// breaks this file at compile time; the global parity test asserts one endpoint
// per rpc.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "jobs_get",
		Path:        jobsconnect.JobsServiceGetJobProcedure,
		Method:      "POST",
		Summary:     "Get a job by id",
		Description: "Returns the current durable-job record for one job id.",
		Category:    "jobs",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"id": "string"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"job": "Job"},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No job with that id exists"},
		},
		Examples: []module.Example{
			{Name: "Get a job", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.jobs.JobsService/GetJob -H 'Content-Type: application/json' -d '{\"id\":\"<job-id>\"}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "image-tools jobs get", Args: []string{"<id>"}},
	},
	{
		ID:          "jobs_wait",
		Path:        jobsconnect.JobsServiceWaitJobProcedure,
		Method:      "POST",
		Summary:     "Wait for a job (block-once)",
		Description: "Blocks server-side until the job reaches a terminal state, then returns it. A client disconnect does not affect the job. This is the canonical wait verb — never poll GetJob in a loop.",
		Category:    "jobs",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"id": "string"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"job": "Job (terminal)"},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No job with that id exists"},
		},
		Examples: []module.Example{
			{Name: "Wait for a job", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.jobs.JobsService/WaitJob -H 'Content-Type: application/json' -d '{\"id\":\"<job-id>\"}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "image-tools jobs wait", Args: []string{"<id>"}},
	},
	{
		ID:          "jobs_list",
		Path:        jobsconnect.JobsServiceListJobsProcedure,
		Method:      "POST",
		Summary:     "List recent jobs",
		Description: "Returns recent durable jobs, newest first.",
		Category:    "jobs",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"limit": "int32 (0 = server default)"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"jobs": "array<Job>"},
		},
		Examples: []module.Example{
			{Name: "List jobs", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.jobs.JobsService/ListJobs -H 'Content-Type: application/json' -d '{\"limit\":20}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "image-tools jobs list", Args: []string{"--limit", "<n>"}},
	},
	{
		ID:          "jobs_cancel",
		Path:        jobsconnect.JobsServiceCancelJobProcedure,
		Method:      "POST",
		Summary:     "Cancel a job",
		Description: "Aborts a job: a running job's context is canceled; a queued job is marked canceled immediately. Returns the post-cancel record.",
		Category:    "jobs",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"id": "string"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"job": "Job"},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No job with that id exists"},
		},
		Examples: []module.Example{
			{Name: "Cancel a job", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.jobs.JobsService/CancelJob -H 'Content-Type: application/json' -d '{\"id\":\"<job-id>\"}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "image-tools jobs cancel", Args: []string{"<id>"}},
	},
	{
		ID:          "jobs_watch",
		Path:        jobsconnect.JobsServiceWatchJobProcedure,
		Method:      "POST",
		Summary:     "Watch a job's progress (server stream)",
		Description: "Server-streams progress events for a job until it reaches a terminal state. The latest known event is replayed first. Backs SSE-style live progress in the UI.",
		Category:    "jobs",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"id": "string"},
		},
		Response: &module.Schema{
			Type:       "stream",
			Properties: map[string]string{"event": "ProgressEvent (stream)"},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No job with that id exists"},
		},
		Examples: []module.Example{
			{Name: "Watch a job", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.jobs.JobsService/WatchJob -H 'Content-Type: application/json' -d '{\"id\":\"<job-id>\"}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "image-tools jobs watch", Args: []string{"<id>"}},
	},
}
