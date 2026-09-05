package main

import (
	"net/http"

	updatehttp "landing-page-business-suite-api/handlers/delivery"
)

func updateDependencies(bundles interface{ BundleKey() string }) updatehttp.UpdateDependencies {
	return updatehttp.UpdateDependencies{
		BundleKey: bundles.BundleKey,
		PathParam: getPathParam,
		WriteError: func(w http.ResponseWriter, status int, message, kind string) {
			writeJSONError(w, status, message, kind)
		},
		WriteData:  writeJSONSuccessData,
		DecodeJSON: decodeJSONBody,
	}
}

func registerUpdateRoutes(s *Server) {
	deps := updateDependencies(s.planService)
	updateAPIKeyMiddleware := updatehttp.RequireUpdateAPIKey(deps, s.downloadService)
	s.router.HandleFunc("/api/v1/updates/{app_key}/{channel}/{file}", updateAPIKeyMiddleware(updatehttp.UpdateFile(deps, s.downloadService, s.downloadHosting))).Methods("GET")
	s.router.HandleFunc("/api/v1/updates/{app_key}/channels", updateAPIKeyMiddleware(updatehttp.ChannelDiscovery(deps, s.downloadService))).Methods("GET")
	s.router.HandleFunc("/api/v1/updates/{app_key}/verify", updateAPIKeyMiddleware(updatehttp.VerifyUpdate(deps, s.downloadService, s.downloadHosting))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/download-apps/{app_key}/update-policy", s.requireAdmin(updatehttp.GetUpdatePolicy(deps, s.downloadService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/download-apps/{app_key}/update-policy", s.requireAdmin(updatehttp.PutUpdatePolicy(deps, s.downloadService))).Methods("PUT")
}
