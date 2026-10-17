package download

import (
	"landing-page-react-vite-api/internal/module"

	landingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1/landing_page_react_vite_v1connect"
)

// Endpoints describes the downloads module's Connect-RPC surface for codegen.
var Endpoints = []module.EndpointDescriptor{
	{
		ID: "download_authorize", Path: landingconnect.DownloadServiceAuthorizeDownloadProcedure, Method: "POST",
		Summary: "Authorize download", Description: "Authorizes a platform artifact download (public, entitlement-gated; identity via X-User-Email).", Category: "download",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"app": "string", "platform": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"asset": "DownloadAsset"}},
	},
	{
		ID: "download_list_apps", Path: landingconnect.DownloadServiceListDownloadAppsProcedure, Method: "POST",
		Summary: "List download apps", Description: "Lists all download apps for the configured bundle (admin).", Category: "download",
		Response: &module.Schema{Type: "object", Properties: map[string]string{"apps": "DownloadApp[]"}},
	},
	{
		ID: "download_create_app", Path: landingconnect.DownloadServiceCreateDownloadAppProcedure, Method: "POST",
		Summary: "Create download app", Description: "Creates a download app with its platform installers (admin).", Category: "download",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"app": "DownloadApp"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"app": "DownloadApp"}},
	},
	{
		ID: "download_save_app", Path: landingconnect.DownloadServiceSaveDownloadAppProcedure, Method: "POST",
		Summary: "Save download app", Description: "Upserts a download app by key with its platform installers (admin).", Category: "download",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"app_key": "string", "app": "DownloadApp"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"app": "DownloadApp"}},
	},
}
