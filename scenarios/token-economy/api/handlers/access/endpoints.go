package access

import (
	"token-economy/internal/module"

	accessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/access/accessv1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "access_minter_create_token_type", Path: accessconnect.MinterServiceCreateTokenTypeProcedure, Method: "POST", Summary: "Create token type", Description: "Declares a token type, supply policy, and its sole named minter authority.", Category: "minter"},
	{ID: "access_minter_get_token_type", Path: accessconnect.MinterServiceGetTokenTypeProcedure, Method: "POST", Summary: "Get token type", Description: "Returns a declared token type, including retained retirement state.", Category: "minter"},
	{ID: "access_minter_list_token_types", Path: accessconnect.MinterServiceListTokenTypesProcedure, Method: "POST", Summary: "List token types", Description: "Lists active token types, optionally including retired declarations.", Category: "minter"},
	{ID: "access_minter_retire_token_type", Path: accessconnect.MinterServiceRetireTokenTypeProcedure, Method: "POST", Summary: "Retire token type", Description: "Retires a token type without deleting its identity or history.", Category: "minter"},
	{ID: "access_minter_mint_supply", Path: accessconnect.MinterServiceMintSupplyProcedure, Method: "POST", Summary: "Mint token supply", Description: "Increases lifetime issued supply while enforcing the declared cap; it does not alter a holder balance.", Category: "minter"},
	{ID: "access_minter_create_grant", Path: accessconnect.MinterServiceCreateGrantProcedure, Method: "POST", Summary: "Create grant", Description: "Admits tokens through a mandate-shaped grant and an atomic journal credit.", Category: "minter"},
	{ID: "access_minter_get_grant", Path: accessconnect.MinterServiceGetGrantProcedure, Method: "POST", Summary: "Get grant", Description: "Returns a retained grant and its server-evaluated rules.", Category: "minter"},
	{ID: "access_minter_list_grants", Path: accessconnect.MinterServiceListGrantsProcedure, Method: "POST", Summary: "List grants", Description: "Lists grants with optional holder and token-type filters.", Category: "minter"},
	{ID: "access_minter_revoke_grant", Path: accessconnect.MinterServiceRevokeGrantProcedure, Method: "POST", Summary: "Revoke grant", Description: "Revokes one grant idempotently while retaining its history.", Category: "minter"},
	{ID: "access_minter_update_grant_rule", Path: accessconnect.MinterServiceUpdateGrantRuleProcedure, Method: "POST", Summary: "Update grant rule", Description: "Changes a server-evaluated grant rule.", Category: "minter"},
	{ID: "access_minter_create_holder", Path: accessconnect.MinterServiceCreateHolderProcedure, Method: "POST", Summary: "Create holder", Description: "Binds a household holder to one authenticator subject idempotently.", Category: "minter"},
	{ID: "access_minter_get_holder", Path: accessconnect.MinterServiceGetHolderProcedure, Method: "POST", Summary: "Get holder", Description: "Returns one holder to the minter audience.", Category: "minter"},
	{ID: "access_minter_list_holders", Path: accessconnect.MinterServiceListHoldersProcedure, Method: "POST", Summary: "List holders", Description: "Lists household holders for the minter audience.", Category: "minter"},
	{ID: "access_holder_view_economy", Path: accessconnect.HolderServiceViewEconomyProcedure, Method: "POST", Summary: "View economy", Description: "Returns the authenticated holder's economy view.", Category: "holder"},
	{ID: "access_holder_submit_request", Path: accessconnect.HolderServiceSubmitRequestProcedure, Method: "POST", Summary: "Submit request", Description: "Submits a holder request without granting authority operations.", Category: "holder"},
}
