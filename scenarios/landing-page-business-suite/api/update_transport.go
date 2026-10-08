package main

import (
	"context"
	"net/http"

	updatehttp "landing-page-business-suite-api/handlers/delivery"
)

func updateDependencies(bundles interface{ BundleKey() string }, db StartupStore) updatehttp.UpdateDependencies {
	return updatehttp.UpdateDependencies{
		BundleKey: bundles.BundleKey,
		PathParam: getPathParam,
		WriteError: func(w http.ResponseWriter, status int, message, kind string) {
			writeJSONError(w, status, message, kind)
		},
		WriteData:  writeJSONSuccessData,
		DecodeJSON: decodeJSONBody,
		RecordEvent: func(ctx context.Context, bundle, app, platform, event string) error {
			if db == nil {
				return nil
			}
			_, err := db.ExecContext(ctx, `INSERT INTO delivery_events (event_type,bundle_key,app_key,platform,variant_key) VALUES ($1,$2,$3,$4,$5)`, event, bundle, app, platform, "default")
			return err
		},
	}
}

func registerUpdateRoutes(s *Server) {
	deps := updateDependencies(s.planService, s.db)
	updateAPIKeyMiddleware := updatehttp.RequireUpdateAPIKey(deps, s.downloadService)
	s.router.HandleFunc("/api/v1/updates/{app_key}/{channel}/{file}", updateAPIKeyMiddleware(updatehttp.UpdateFile(deps, s.downloadService, s.downloadHosting))).Methods("GET")
	s.router.HandleFunc("/api/v1/updates/{app_key}/channels", updateAPIKeyMiddleware(updatehttp.ChannelDiscovery(deps, s.downloadService))).Methods("GET")
	s.router.HandleFunc("/api/v1/updates/{app_key}/verify", updateAPIKeyMiddleware(updatehttp.VerifyUpdate(deps, s.downloadService, s.downloadHosting))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/download-apps/{app_key}/update-policy", s.requireAdmin(updatehttp.GetUpdatePolicy(deps, s.downloadService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/download-apps/{app_key}/update-policy", s.requireAdmin(updatehttp.PutUpdatePolicy(deps, s.downloadService))).Methods("PUT")
}
