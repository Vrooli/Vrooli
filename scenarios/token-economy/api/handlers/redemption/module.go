// Package redemption exposes redemption workflows at the transport edge.
package redemption

import (
	"token-economy/internal/module"
	"token-economy/internal/redemption"

	"github.com/gorilla/mux"
	accessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/access/accessv1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "redemption_request", Path: accessconnect.HolderServiceRequestRedemptionProcedure, Method: "POST", Summary: "Request redemption", Description: "Reserves or immediately spends the authenticated holder's balance.", Category: "redemption"},
	{ID: "redemption_pending_list", Path: accessconnect.MinterServiceListPendingRedemptionsProcedure, Method: "POST", Summary: "List pending redemptions", Description: "Returns the durable minter approval queue.", Category: "redemption"},
	{ID: "redemption_approve", Path: accessconnect.MinterServiceApproveRedemptionProcedure, Method: "POST", Summary: "Approve redemption", Description: "Settles one pending redemption exactly once.", Category: "redemption"},
	{ID: "redemption_deny", Path: accessconnect.MinterServiceDenyRedemptionProcedure, Method: "POST", Summary: "Deny redemption", Description: "Denies one pending redemption and releases its reservations.", Category: "redemption"},
}

// Module registers the domain skeleton without inventing transport behavior.
func Module() module.Module {
	return module.Module{Name: "redemption", Mount: func(*mux.Router) {}, Endpoints: Endpoints}
}

// Schema re-exports the domain-owned schema for the central boot registry.
func Schema() string { return redemption.Schema() }
