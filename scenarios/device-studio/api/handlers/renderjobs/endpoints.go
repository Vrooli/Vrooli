package renderjobs

import (
	"device-studio/internal/module"
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots/shots_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "render_submit", Path: connect.RenderJobsServiceSubmitRenderProcedure, Method: "POST", Summary: "Submit a render", Description: "Submits an immutable still render, video sample preview, or fixed-profile MP4 render to the shared durable local job ledger.", Category: "renders"},
	{ID: "render_sample_binding", Path: connect.RenderJobsServiceGetVideoSampleBindingProcedure, Method: "POST", Summary: "Get verified phone video sample claim", Description: "Returns a server-verified AUTO phone trim-start sample claim for saving into a new immutable shot revision; manual and browser previews do not produce saveable claims.", Category: "renders"},
	{ID: "render_get", Path: connect.RenderJobsServiceGetRenderProcedure, Method: "POST", Summary: "Get render job", Description: "Reads a project-owned render job without advancing execution.", Category: "renders"},
	{ID: "render_list", Path: connect.RenderJobsServiceListRendersProcedure, Method: "POST", Summary: "List render jobs", Description: "Returns a bounded project-scoped keyset page of render jobs.", Category: "renders"},
	{ID: "render_cancel", Path: connect.RenderJobsServiceCancelRenderProcedure, Method: "POST", Summary: "Cancel render job", Description: "Requests cancellation using an expected job version.", Category: "renders"},
	{ID: "render_output_content", Path: "/api/v1/projects/{project_id}/renders/{render_id}/content", Method: "GET", Summary: "Fetch rendered output", Description: "Streams digest-verified PNG, sample PNG, or MP4 bytes for a succeeded project-owned render.", Category: "renders", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Streams verified opaque output bytes by project and job id; no filesystem path is exposed.", ProtoPayloads: &module.RESTProtoPayloads{Request: module.RESTPayload{Transport: "none", Conformance: "none"}, Response: module.RESTPayload{Transport: "none", Conformance: "none"}, Error: module.RESTPayload{ProtoFullName: "vrooli.device_studio.v1.shared.ErrorEnvelope", Transport: "json", Conformance: "protojson"}}}},
}
