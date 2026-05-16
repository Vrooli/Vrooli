// Package settings hosts the SettingsService Connect-RPC handler.
//
// All methods are stubs returning Unimplemented until the BYOK credential
// store + provider-config persistence land (Phase C / Phase G).
package settings

import (
	"log"

	"audio-tools/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	settconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/settings/settings_v1connect"
)

type Deps struct {
	Logger *log.Logger
}

type connectHandler struct {
	settconnect.UnimplementedSettingsServiceHandler
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "settings.get_provider_config", Path: "/vrooli.audio_tools.v1.settings.SettingsService/GetProviderConfig", Method: "POST", Category: "settings"},
	{ID: "settings.update_provider_config", Path: "/vrooli.audio_tools.v1.settings.SettingsService/UpdateProviderConfig", Method: "POST", Category: "settings"},
	{ID: "settings.list_byok_credentials", Path: "/vrooli.audio_tools.v1.settings.SettingsService/ListBYOKCredentials", Method: "POST", Category: "settings"},
	{ID: "settings.upsert_byok_credential", Path: "/vrooli.audio_tools.v1.settings.SettingsService/UpsertBYOKCredential", Method: "POST", Category: "settings"},
	{ID: "settings.delete_byok_credential", Path: "/vrooli.audio_tools.v1.settings.SettingsService/DeleteBYOKCredential", Method: "POST", Category: "settings"},
	{ID: "settings.get_voice_overrides", Path: "/vrooli.audio_tools.v1.settings.SettingsService/GetVoiceOverrides", Method: "POST", Category: "settings"},
	{ID: "settings.set_voice_override", Path: "/vrooli.audio_tools.v1.settings.SettingsService/SetVoiceOverride", Method: "POST", Category: "settings"},
}

func Module(logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	connectPath, connectHandler := settconnect.NewSettingsServiceHandler(NewConnectHandler(Deps{Logger: logger}))
	return module.Module{
		Name: "settings",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }
