package admin

import (
	"landing-page-react-vite-api/internal/module"

	landingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1/landing_page_react_vite_v1connect"
)

// Endpoints describes the admin-auth module's Connect-RPC surface for codegen.
var Endpoints = []module.EndpointDescriptor{
	{
		ID: "admin_login", Path: landingconnect.AdminAuthServiceLoginProcedure, Method: "POST",
		Summary: "Admin login", Description: "Authenticates an admin (bcrypt) and sets a signed session cookie.", Category: "admin",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"email": "string", "password": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"email": "string", "authenticated": "bool", "reset_enabled": "bool"}},
	},
	{
		ID: "admin_logout", Path: landingconnect.AdminAuthServiceLogoutProcedure, Method: "POST",
		Summary: "Admin logout", Description: "Clears the admin session cookie.", Category: "admin",
		Response: &module.Schema{Type: "object", Properties: map[string]string{"success": "bool"}},
	},
	{
		ID: "admin_session", Path: landingconnect.AdminAuthServiceSessionProcedure, Method: "POST",
		Summary: "Admin session", Description: "Reports the current admin session state from the session cookie.", Category: "admin",
		Response: &module.Schema{Type: "object", Properties: map[string]string{"email": "string", "authenticated": "bool", "reset_enabled": "bool"}},
	},
}
