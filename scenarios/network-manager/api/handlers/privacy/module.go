package privacy

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	"network-manager/internal/module"
	domainprivacy "network-manager/internal/privacy"

	privacyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/privacy"
	privacyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/privacy/privacy_v1connect"
)

type handler struct {
	service *domainprivacy.Service
}

func Module(db domainprivacy.SQLExecutor) module.Module {
	service := domainprivacy.NewService(domainprivacy.Config{Repo: domainprivacy.NewSQLiteRepository(db)})
	path, h := privacyconnect.NewPrivacyServiceHandler(&handler{service: service})
	return module.Module{Name: "privacy", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h})
	}, Endpoints: Endpoints}
}

func Schema() string { return domainprivacy.Schema() }

func (h *handler) GetRetentionSettings(ctx context.Context, _ *connect.Request[privacyv1.GetRetentionSettingsRequest]) (*connect.Response[privacyv1.GetRetentionSettingsResponse], error) {
	settings, err := h.service.GetRetention(ctx)
	if err != nil {
		return nil, privacyError(err)
	}
	return connect.NewResponse(&privacyv1.GetRetentionSettingsResponse{Settings: toProtoRetention(settings)}), nil
}

func (h *handler) UpdateRetentionSettings(ctx context.Context, req *connect.Request[privacyv1.UpdateRetentionSettingsRequest]) (*connect.Response[privacyv1.UpdateRetentionSettingsResponse], error) {
	if req.Msg.GetSettings() == nil {
		settings, err := h.service.GetRetention(ctx)
		if err != nil {
			return nil, privacyError(err)
		}
		return connect.NewResponse(&privacyv1.UpdateRetentionSettingsResponse{Settings: toProtoRetention(settings)}), nil
	}
	settings, err := h.service.UpdateRetention(ctx, fromProtoRetention(req.Msg.GetSettings()))
	if err != nil {
		return nil, privacyError(err)
	}
	return connect.NewResponse(&privacyv1.UpdateRetentionSettingsResponse{Settings: toProtoRetention(settings)}), nil
}

func (h *handler) GetVisibilitySettings(ctx context.Context, _ *connect.Request[privacyv1.GetVisibilitySettingsRequest]) (*connect.Response[privacyv1.GetVisibilitySettingsResponse], error) {
	settings, err := h.service.GetVisibility(ctx)
	if err != nil {
		return nil, privacyError(err)
	}
	return connect.NewResponse(&privacyv1.GetVisibilitySettingsResponse{Settings: toProtoVisibility(settings)}), nil
}

func privacyError(err error) error {
	switch {
	case errors.Is(err, domainprivacy.ErrInvalidSettings):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

func fromProtoRetention(settings *privacyv1.RetentionSettings) domainprivacy.RetentionSettings {
	return domainprivacy.RetentionSettings{
		QueryLogDays:   settings.GetQueryLogDays(),
		SnapshotDays:   settings.GetSnapshotDays(),
		ExperimentDays: settings.GetExperimentDays(),
		Profile:        settings.GetProfile(),
	}
}

func toProtoRetention(settings domainprivacy.RetentionSettings) *privacyv1.RetentionSettings {
	return &privacyv1.RetentionSettings{
		QueryLogDays:   settings.QueryLogDays,
		SnapshotDays:   settings.SnapshotDays,
		ExperimentDays: settings.ExperimentDays,
		Profile:        settings.Profile,
	}
}

func toProtoVisibility(settings domainprivacy.VisibilitySettings) *privacyv1.VisibilitySettings {
	return &privacyv1.VisibilitySettings{
		ShowQueryDomains:  settings.ShowQueryDomains,
		ShowDeviceHistory: settings.ShowDeviceHistory,
		HouseholdMode:     settings.HouseholdMode,
		Notes:             settings.Notes,
	}
}

var Endpoints = []module.EndpointDescriptor{
	connectEndpoint("privacy_retention_get", privacyconnect.PrivacyServiceGetRetentionSettingsProcedure, "Get retention settings"),
	connectEndpoint("privacy_retention_update", privacyconnect.PrivacyServiceUpdateRetentionSettingsProcedure, "Update retention settings"),
	connectEndpoint("privacy_visibility_get", privacyconnect.PrivacyServiceGetVisibilitySettingsProcedure, "Get visibility settings"),
}

func connectEndpoint(id, path, summary string) module.EndpointDescriptor {
	return module.EndpointDescriptor{ID: id, Path: path, Method: "POST", Summary: summary, Category: "privacy", Request: &module.Schema{Type: "object", Properties: map[string]string{}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"settings": "privacy settings"}}}
}
