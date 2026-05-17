// Package settings hosts the SettingsService Connect-RPC handler.
package settings

import (
	"log"

	"audio-tools/internal/ai/chains"
	"audio-tools/internal/byokstore"
	"audio-tools/internal/modulekit"
	"audio-tools/internal/store"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	settconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/settings/settings_v1connect"
)

type Deps struct {
	Logger         *log.Logger
	ProviderConfig *store.ProviderConfigStore
	BYOK           *byokstore.Store
	VoiceOverrides *store.VoiceOverrideStore
	Coordinator    *chains.Coordinator
}

type connectHandler struct{ deps Deps }

// NewConnectHandler returns the live Connect handler. Caller is
// responsible for wiring the dependencies.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

var Endpoints = []modulekit.EndpointDescriptor{
	{ID: "settings.get_provider_config", Path: "/vrooli.audio_tools.v1.settings.SettingsService/GetProviderConfig", Method: "POST", Category: "settings"},
	{ID: "settings.update_provider_config", Path: "/vrooli.audio_tools.v1.settings.SettingsService/UpdateProviderConfig", Method: "POST", Category: "settings"},
	{ID: "settings.list_byok_credentials", Path: "/vrooli.audio_tools.v1.settings.SettingsService/ListBYOKCredentials", Method: "POST", Category: "settings"},
	{ID: "settings.upsert_byok_credential", Path: "/vrooli.audio_tools.v1.settings.SettingsService/UpsertBYOKCredential", Method: "POST", Category: "settings"},
	{ID: "settings.delete_byok_credential", Path: "/vrooli.audio_tools.v1.settings.SettingsService/DeleteBYOKCredential", Method: "POST", Category: "settings"},
	{ID: "settings.get_voice_overrides", Path: "/vrooli.audio_tools.v1.settings.SettingsService/GetVoiceOverrides", Method: "POST", Category: "settings"},
	{ID: "settings.set_voice_override", Path: "/vrooli.audio_tools.v1.settings.SettingsService/SetVoiceOverride", Method: "POST", Category: "settings"},
}

func Module(d Deps) modulekit.Module {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	connectPath, h := settconnect.NewSettingsServiceHandler(NewConnectHandler(d))
	return modulekit.Module{
		Name: "settings",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: h})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }
