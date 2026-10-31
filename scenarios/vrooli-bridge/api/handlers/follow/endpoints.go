package follow

import "vrooli-bridge/internal/module"

var ownerSetting = &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Owner-only node update setting with a dedicated CLI wrapper (`vrooli-bridge follow`).", ProtoPayloads: &module.RESTProtoPayloads{Request: module.RESTPayload{Transport: "json", Conformance: "external_shape"}, Response: module.RESTPayload{Transport: "json", Conformance: "external_shape"}, Error: module.RESTPayload{Transport: "json", Conformance: "external_shape"}}}

var policySchema = &module.Schema{Type: "BranchFollowPolicy", Properties: map[string]string{"node_id": "string", "branch": "string", "repo_url": "string", "last_seen_head": "string", "last_checked_at": "string", "last_result": "string", "last_op_id": "string"}}

// Endpoints describes the branch-follow REST surface.
var Endpoints = []module.EndpointDescriptor{{
	ID: "bridge-follow-list", Path: "/api/v1/follow", Method: "GET", Category: "system",
	Summary: "List branch-follow policies", Description: "Every node set to follow a git branch, with the outcome of its last check.",
	Response:      &module.Schema{Type: "BranchFollowPolicies", Properties: map[string]string{"policies": "array"}},
	RESTException: ownerSetting,
}, {
	ID: "bridge-follow-set", Path: "/api/v1/follow/{node}", Method: "PUT", Category: "system",
	Summary: "Set a node to follow a branch", Description: "The node is updated by provisioning whenever the branch receives a new commit; update_now also sends the current head.",
	Request:       &module.Schema{Type: "BranchFollowRequest", Properties: map[string]string{"branch": "string", "repo_url": "string", "update_now": "boolean"}},
	Response:      policySchema,
	RESTException: ownerSetting,
}, {
	ID: "bridge-follow-unset", Path: "/api/v1/follow/{node}", Method: "DELETE", Category: "system",
	Summary: "Stop following a branch", Description: "The node keeps its current revision; Bridge stops updating it from the branch.",
	Response:      &module.Schema{Type: "BranchFollowUnset", Properties: map[string]string{"status": "string"}},
	RESTException: ownerSetting,
}, {
	ID: "bridge-follow-check", Path: "/api/v1/follow/{node}/check", Method: "POST", Category: "system",
	Summary: "Check a followed node now", Description: "Reads the branch head and dispatches an update when it moved.",
	Response:      policySchema,
	RESTException: ownerSetting,
}}
