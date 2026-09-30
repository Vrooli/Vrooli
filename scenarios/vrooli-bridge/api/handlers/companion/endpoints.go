package companion

import (
	companionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/companion/companionv1connect"
	"vrooli-bridge/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "companion_install", Path: companionconnect.CompanionServiceInstallProcedure, Method: "POST", Summary: "Install Device Control companion", Category: "companion"},
	{ID: "companion_inspect", Path: companionconnect.CompanionServiceInspectProcedure, Method: "POST", Summary: "Inspect Device Control companion", Category: "companion"},
	{ID: "companion_upgrade", Path: companionconnect.CompanionServiceUpgradeProcedure, Method: "POST", Summary: "Upgrade Device Control companion", Category: "companion"},
	{ID: "companion_revoke", Path: companionconnect.CompanionServiceRevokeProcedure, Method: "POST", Summary: "Revoke Device Control companion", Category: "companion"},
	{ID: "companion_remove", Path: companionconnect.CompanionServiceRemoveProcedure, Method: "POST", Summary: "Remove Device Control companion", Category: "companion"},
}
