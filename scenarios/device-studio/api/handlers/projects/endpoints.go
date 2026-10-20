package projects

import (
	"device-studio/internal/module"
	projectsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/projects/projects_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "projects_list", Path: projectsconnect.ProjectsServiceListProjectsProcedure, Method: "POST", Summary: "List projects", Description: "Returns a bounded page of local projects.", Category: "projects"},
	{ID: "projects_create", Path: projectsconnect.ProjectsServiceCreateProjectProcedure, Method: "POST", Summary: "Create project", Description: "Creates an empty local project.", Category: "projects"},
	{ID: "projects_get", Path: projectsconnect.ProjectsServiceGetProjectProcedure, Method: "POST", Summary: "Open project", Description: "Returns a local project by id.", Category: "projects"},
	{ID: "projects_rename", Path: projectsconnect.ProjectsServiceRenameProjectProcedure, Method: "POST", Summary: "Rename project", Description: "Renames a project only when expected_revision matches; conflicts return ProjectConflict detail.", Category: "projects"},
	{ID: "projects_media_list", Path: projectsconnect.ProjectsServiceListSourceMediaProcedure, Method: "POST", Summary: "List project source media", Description: "Returns a bounded page of imported source image metadata.", Category: "media"},
	{ID: "projects_media_get", Path: projectsconnect.ProjectsServiceGetSourceMediaProcedure, Method: "POST", Summary: "Get project source media", Description: "Returns typed metadata for one imported source image.", Category: "media"},
	{ID: "projects_media_import", Path: "/api/v1/projects/{project_id}/media", Method: "POST", Summary: "Import project source media", Description: "Imports one locally uploaded PNG or JPEG and returns typed metadata.", Category: "media", RESTException: &module.RESTException{Reason: module.RESTReasonMultipartUpload, Note: "Multipart carries the file bytes; the response is the proto-typed UploadSourceMediaResponse.", ProtoPayloads: &module.RESTProtoPayloads{Request: module.RESTPayload{Transport: "multipart/form-data", Conformance: "transport_only"}, Response: module.RESTPayload{ProtoFullName: "vrooli.device_studio.v1.projects.UploadSourceMediaResponse", Transport: "json", Conformance: "protojson"}, Error: module.RESTPayload{ProtoFullName: "vrooli.device_studio.v1.shared.ErrorEnvelope", Transport: "json", Conformance: "protojson"}}}},
	{ID: "projects_media_content", Path: "/api/v1/projects/{project_id}/media/{media_id}/content", Method: "GET", Summary: "Fetch source image bytes", Description: "Serves a verified source image by opaque project and media ids for browser display.", Category: "media", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Serves opaque image bytes by identifiers; no generated client applies and content is digest-verified before response.", ProtoPayloads: &module.RESTProtoPayloads{Request: module.RESTPayload{Transport: "none", Conformance: "none"}, Response: module.RESTPayload{Transport: "none", Conformance: "none"}, Error: module.RESTPayload{ProtoFullName: "vrooli.device_studio.v1.shared.ErrorEnvelope", Transport: "json", Conformance: "protojson"}}}},
}
