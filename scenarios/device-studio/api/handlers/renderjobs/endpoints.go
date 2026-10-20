package renderjobs

import (
	"device-studio/internal/module"
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots/shots_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID:"render_submit",Path:connect.RenderJobsServiceSubmitRenderProcedure,Method:"POST",Summary:"Submit a still render",Description:"Submits an immutable still render to the durable local job ledger.",Category:"renders"},
	{ID:"render_get",Path:connect.RenderJobsServiceGetRenderProcedure,Method:"POST",Summary:"Get render job",Description:"Reads a project-owned render job without advancing execution.",Category:"renders"},
	{ID:"render_list",Path:connect.RenderJobsServiceListRendersProcedure,Method:"POST",Summary:"List render jobs",Description:"Returns a bounded project-scoped keyset page of render jobs.",Category:"renders"},
	{ID:"render_cancel",Path:connect.RenderJobsServiceCancelRenderProcedure,Method:"POST",Summary:"Cancel render job",Description:"Requests cancellation using an expected job version.",Category:"renders"},
	{ID:"render_output_content",Path:"/api/v1/projects/{project_id}/renders/{render_id}/content",Method:"GET",Summary:"Fetch rendered PNG",Description:"Serves digest-verified PNG bytes for a succeeded project-owned render.",Category:"renders",RESTException:&module.RESTException{Reason:module.RESTReasonOpsProbe,Note:"Serves verified opaque output bytes by project and job id; no filesystem path is exposed.",ProtoPayloads:&module.RESTProtoPayloads{Request:module.RESTPayload{Transport:"none",Conformance:"none"},Response:module.RESTPayload{Transport:"none",Conformance:"none"},Error:module.RESTPayload{ProtoFullName:"vrooli.device_studio.v1.shared.ErrorEnvelope",Transport:"json",Conformance:"protojson"}}}},
}
