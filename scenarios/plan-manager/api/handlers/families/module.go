// Package families exposes Plan Manager's reviewed multi-plan coordination graph.
package families

import (
	"plan-manager/internal/families"
	"plan-manager/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	familiesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/families/families_v1connect"
)

func Module(db *database.RoutedDB) module.Module {
	service := families.NewService(families.NewSQLiteRepository(db))
	path, handler := familiesconnect.NewFamiliesServiceHandler(NewConnectHandler(service))
	return module.Module{Name: "families", Mount: func(router *mux.Router) {
		connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
	}, Endpoints: Endpoints}
}

func Schema() string { return families.Schema() }

var Endpoints = []module.EndpointDescriptor{
	endpoint("families_create", familiesconnect.FamiliesServiceCreateFamilyProcedure, "Create family"),
	endpoint("families_get", familiesconnect.FamiliesServiceGetFamilyProcedure, "Get family"),
	endpoint("families_list", familiesconnect.FamiliesServiceListFamiliesProcedure, "List families"),
	endpoint("families_update", familiesconnect.FamiliesServiceUpdateFamilyProcedure, "Update family"),
	endpoint("families_member_put", familiesconnect.FamiliesServicePutMemberProcedure, "Put family member"),
	endpoint("families_member_remove", familiesconnect.FamiliesServiceRemoveMemberProcedure, "Remove family member"),
	endpoint("families_claim_put", familiesconnect.FamiliesServicePutClaimProcedure, "Put resource claim"),
	endpoint("families_claim_remove", familiesconnect.FamiliesServiceRemoveClaimProcedure, "Remove resource claim"),
	endpoint("families_graph_propose", familiesconnect.FamiliesServiceProposeGraphProcedure, "Propose family graph"),
	endpoint("families_graph_review", familiesconnect.FamiliesServiceReviewGraphProcedure, "Review family graph"),
	endpoint("families_frontier_get", familiesconnect.FamiliesServiceGetFrontierProcedure, "Get reviewed launch frontier"),
	endpoint("families_render", familiesconnect.FamiliesServiceRenderFamilyProcedure, "Render family"),
}

func endpoint(id, path, summary string) module.EndpointDescriptor {
	return module.EndpointDescriptor{ID: id, Path: path, Method: "POST", Summary: summary, Description: summary + " through the authoritative reviewed family aggregate.", Category: "families"}
}
