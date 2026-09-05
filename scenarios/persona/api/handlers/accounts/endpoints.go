package accounts

import (
	"persona/internal/module"

	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/accounts/accounts_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "accounts_link", Path: accountsconnect.AccountsServiceLinkAccountProcedure, Method: "POST", Summary: "Link an account created as a persona", Category: "accounts"},
	{ID: "accounts_list", Path: accountsconnect.AccountsServiceListAccountsProcedure, Method: "POST", Summary: "List linked accounts", Category: "accounts"},
	{ID: "accounts_add_address", Path: accountsconnect.AccountsServiceAddAddressProcedure, Method: "POST", Summary: "Add a persona address", Category: "accounts"},
	{ID: "accounts_list_addresses", Path: accountsconnect.AccountsServiceListAddressesProcedure, Method: "POST", Summary: "List persona addresses", Category: "accounts"},
	{ID: "accounts_add_obligation", Path: accountsconnect.AccountsServiceAddObligationProcedure, Method: "POST", Summary: "Record an account obligation", Category: "accounts"},
	{ID: "accounts_list_obligations", Path: accountsconnect.AccountsServiceListObligationsProcedure, Method: "POST", Summary: "List persona obligations", Category: "accounts"},
	{ID: "accounts_cancel_obligation", Path: accountsconnect.AccountsServiceCancelObligationProcedure, Method: "POST", Summary: "Cancel an obligation", Category: "accounts"},
	{ID: "accounts_release_address", Path: accountsconnect.AccountsServiceReleaseAddressProcedure, Method: "POST", Summary: "Release an address into a named target", Category: "accounts"},
}
