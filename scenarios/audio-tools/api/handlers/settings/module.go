// Package settings hosts the SettingsService Connect-RPC handler.
package settings

import (
	"audio-tools/internal/ai/chains"
	"audio-tools/internal/logx"
	"audio-tools/internal/modulekit"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	settconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/settings/settings_v1connect"
)

type Deps struct {
	Logger         logx.Logger
	ProviderConfig ProviderConfigRepository
	BYOK           BYOKRepository
	VoiceOverrides VoiceOverridesRepository
	Coordinator    *chains.Coordinator
}

var Endpoints = []modulekit.EndpointDescriptor{
	{ID: "settings.get_provider_config", Path: "/vrooli.audio_tools.v1.settings.SettingsService/GetProviderConfig", Method: "POST", Category: "settings", CLIMapping: &modulekit.CLIMapping{Command: "audio-tools settings provider"}},
	{ID: "settings.update_provider_config", Path: "/vrooli.audio_tools.v1.settings.SettingsService/UpdateProviderConfig", Method: "POST", Category: "settings"},
	{ID: "settings.list_byok_credentials", Path: "/vrooli.audio_tools.v1.settings.SettingsService/ListBYOKCredentials", Method: "POST", Category: "settings", CLIMapping: &modulekit.CLIMapping{Command: "audio-tools settings byok-list"}},
	{ID: "settings.upsert_byok_credential", Path: "/vrooli.audio_tools.v1.settings.SettingsService/UpsertBYOKCredential", Method: "POST", Category: "settings", CLIMapping: &modulekit.CLIMapping{Command: "audio-tools settings byok-upsert"}},
	{ID: "settings.delete_byok_credential", Path: "/vrooli.audio_tools.v1.settings.SettingsService/DeleteBYOKCredential", Method: "POST", Category: "settings", CLIMapping: &modulekit.CLIMapping{Command: "audio-tools settings byok-delete"}},
	{ID: "settings.get_voice_overrides", Path: "/vrooli.audio_tools.v1.settings.SettingsService/GetVoiceOverrides", Method: "POST", Category: "settings"},
	{ID: "settings.set_voice_override", Path: "/vrooli.audio_tools.v1.settings.SettingsService/SetVoiceOverride", Method: "POST", Category: "settings"},
}

func Module(d Deps) modulekit.Module {
	if d.Logger == nil {
		panic("settings.Module requires Deps.Logger")
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
