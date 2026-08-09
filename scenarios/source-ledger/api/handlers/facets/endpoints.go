package facets

import (
	facetsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/facets/facets_v1connect"
	"source-ledger/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "facets_list", Path: facetsconnect.FacetsServiceListFacetsProcedure, Method: "POST", Summary: "List seeded facets", Category: "facets"},
	{ID: "facets_set_policy", Path: facetsconnect.FacetsServiceSetFacetPolicyProcedure, Method: "POST", Summary: "Update scoped facet policy", Category: "facets"},
	{ID: "facets_assign", Path: facetsconnect.FacetsServiceAssignFacetProcedure, Method: "POST", Summary: "Append a facet assignment", Category: "facets"},
	{ID: "facets_set_pin", Path: facetsconnect.FacetsServiceSetPinProcedure, Method: "POST", Summary: "Set operator pin state", Category: "facets"},
	{ID: "facets_list_pin_proposals", Path: facetsconnect.FacetsServiceListPinProposalsProcedure, Method: "POST", Summary: "List pin proposals", Category: "facets"},
	{ID: "facets_list_pin_candidates", Path: facetsconnect.FacetsServiceListPinCandidatesProcedure, Method: "POST", Summary: "List ranked pin candidates", Category: "facets"},
	{ID: "facets_resolve_pin_proposal", Path: facetsconnect.FacetsServiceResolvePinProposalProcedure, Method: "POST", Summary: "Resolve pin proposal", Category: "facets"},
	{ID: "facets_mark_superseded", Path: facetsconnect.FacetsServiceMarkSupersededProcedure, Method: "POST", Summary: "Append supersession mark", Category: "facets"},
	{ID: "facets_resolve_thread", Path: facetsconnect.FacetsServiceResolveThreadProcedure, Method: "POST", Summary: "Resolve a thread", Category: "facets"},
}
