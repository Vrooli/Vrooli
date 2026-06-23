package privacy

import (
	"context"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	"network-manager/internal/module"

	privacyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/privacy"
	privacyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/privacy/privacy_v1connect"
)

type handler struct{}

func Module() module.Module {
	path, h := privacyconnect.NewPrivacyServiceHandler(&handler{})
	return module.Module{Name: "privacy", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h})
	}, Endpoints: Endpoints}
}

func Schema() string { return "" }

func (h *handler) GetRetentionSettings(context.Context, *connect.Request[privacyv1.GetRetentionSettingsRequest]) (*connect.Response[privacyv1.GetRetentionSettingsResponse], error) {
	return connect.NewResponse(&privacyv1.GetRetentionSettingsResponse{Settings: retention()}), nil
}

func (h *handler) UpdateRetentionSettings(_ context.Context, req *connect.Request[privacyv1.UpdateRetentionSettingsRequest]) (*connect.Response[privacyv1.UpdateRetentionSettingsResponse], error) {
	if req.Msg.GetSettings() != nil {
		return connect.NewResponse(&privacyv1.UpdateRetentionSettingsResponse{Settings: req.Msg.GetSettings()}), nil
	}
	return connect.NewResponse(&privacyv1.UpdateRetentionSettingsResponse{Settings: retention()}), nil
}

func (h *handler) GetVisibilitySettings(context.Context, *connect.Request[privacyv1.GetVisibilitySettingsRequest]) (*connect.Response[privacyv1.GetVisibilitySettingsResponse], error) {
	return connect.NewResponse(&privacyv1.GetVisibilitySettingsResponse{Settings: &privacyv1.VisibilitySettings{HouseholdMode: true, ShowQueryDomains: false, ShowDeviceHistory: false, Notes: []string{"Minimal visibility is the default until operator changes are implemented."}}}), nil
}

func retention() *privacyv1.RetentionSettings {
	return &privacyv1.RetentionSettings{QueryLogDays: 1, SnapshotDays: 30, ExperimentDays: 30, Profile: "home-minimal"}
}

var Endpoints = []module.EndpointDescriptor{
	connectEndpoint("privacy_retention_get", privacyconnect.PrivacyServiceGetRetentionSettingsProcedure, "Get retention settings"),
	connectEndpoint("privacy_retention_update", privacyconnect.PrivacyServiceUpdateRetentionSettingsProcedure, "Update retention settings"),
	connectEndpoint("privacy_visibility_get", privacyconnect.PrivacyServiceGetVisibilitySettingsProcedure, "Get visibility settings"),
}

func connectEndpoint(id, path, summary string) module.EndpointDescriptor {
	return module.EndpointDescriptor{ID: id, Path: path, Method: "POST", Summary: summary, Category: "privacy", Request: &module.Schema{Type: "object", Properties: map[string]string{}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"settings": "privacy settings"}}}
}
