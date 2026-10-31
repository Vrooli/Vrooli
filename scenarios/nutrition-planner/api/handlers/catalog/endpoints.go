package catalog

import (
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/catalog/catalog_v1connect"
	"nutrition-planner/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "catalog_list", Path: connect.CatalogServiceListCatalogProcedure, Method: "POST", Summary: "List food and product revisions", Category: "catalog"},
	{ID: "catalog_create_revision", Path: connect.CatalogServiceCreateRevisionProcedure, Method: "POST", Summary: "Create a food or product revision", Category: "catalog"},
	{ID: "catalog_get_revision", Path: connect.CatalogServiceGetRevisionProcedure, Method: "POST", Summary: "Get a food or product revision", Category: "catalog"},
}
