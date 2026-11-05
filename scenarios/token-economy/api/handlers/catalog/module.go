// Package catalog exposes redeemable declarations at the transport edge.
package catalog

import (
	"token-economy/internal/catalog"
	"token-economy/internal/module"

	accessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/access/accessv1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "catalog_create", Path: accessconnect.MinterServiceCreateCatalogEntryProcedure, Method: "POST", Summary: "Create catalog entry", Description: "Declares what a token type buys and its approval posture.", Category: "catalog"},
	{ID: "catalog_update", Path: accessconnect.MinterServiceUpdateCatalogEntryProcedure, Method: "POST", Summary: "Update catalog entry", Description: "Updates a retained minter-declared catalog entry.", Category: "catalog"},
	{ID: "catalog_get", Path: accessconnect.MinterServiceGetCatalogEntryProcedure, Method: "POST", Summary: "Get catalog entry", Description: "Returns one retained catalog declaration.", Category: "catalog"},
	{ID: "catalog_list", Path: accessconnect.MinterServiceListCatalogEntriesProcedure, Method: "POST", Summary: "List catalog entries", Description: "Lists catalog declarations, optionally including retired entries.", Category: "catalog"},
	{ID: "catalog_retire", Path: accessconnect.MinterServiceRetireCatalogEntryProcedure, Method: "POST", Summary: "Retire catalog entry", Description: "Retires an entry without deleting its identity.", Category: "catalog"},
	{ID: "catalog_browse", Path: accessconnect.HolderServiceBrowseCatalogProcedure, Method: "POST", Summary: "Browse available catalog", Description: "Returns only entries currently available under server-owned policy.", Category: "catalog"},
}

// Schema re-exports the domain-owned schema for the central boot registry.
func Schema() string { return catalog.Schema() }
