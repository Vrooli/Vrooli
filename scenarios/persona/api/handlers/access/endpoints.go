package access

import (
	accessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/access/access_v1connect"
	"persona/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "access_act_as", Path: accessconnect.AccessServiceActAsProcedure, Method: "POST", Summary: "Act as a named persona", Category: "access"},
	{ID: "access_resolve_persona", Path: accessconnect.AccessServiceResolvePersonaProcedure, Method: "POST", Summary: "Resolve entitled persona metadata", Category: "access"},
	{ID: "access_create_grant", Path: accessconnect.AccessServiceCreateGrantProcedure, Method: "POST", Summary: "Grant a human persona access", Category: "access"},
	{ID: "access_list_grants", Path: accessconnect.AccessServiceListGrantsProcedure, Method: "POST", Summary: "List persona access grants", Category: "access"},
	{ID: "access_remove_grant", Path: accessconnect.AccessServiceRemoveGrantProcedure, Method: "POST", Summary: "Remove a persona access grant", Category: "access"},
	{ID: "access_issue_attestation", Path: accessconnect.AccessServiceIssueAttestationProcedure, Method: "POST", Summary: "Issue a signed identity attestation", Category: "access"},
}
