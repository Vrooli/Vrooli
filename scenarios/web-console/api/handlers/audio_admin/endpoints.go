package audio_admin

import (
	audioadminconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/audio_admin/audio_admin_v1connect"

	"web-console/internal/module"
)

// Endpoints lists every AudioAdminService RPC. Method paths come from the
// generated *Procedure constants so renames break this file at compile time.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:       "audio_admin_get_stream_config",
		Path:     audioadminconnect.AudioAdminServiceGetStreamConfigProcedure,
		Method:   "POST",
		Summary:  "Get STT stream config",
		Category: "audio_admin",
	},
	{
		ID:       "audio_admin_update_stream_config",
		Path:     audioadminconnect.AudioAdminServiceUpdateStreamConfigProcedure,
		Method:   "POST",
		Summary:  "Update STT stream config",
		Category: "audio_admin",
	},
	{
		ID:       "audio_admin_get_wake_word_config",
		Path:     audioadminconnect.AudioAdminServiceGetWakeWordConfigProcedure,
		Method:   "POST",
		Summary:  "Get wake-word config",
		Category: "audio_admin",
	},
	{
		ID:       "audio_admin_update_wake_word_template",
		Path:     audioadminconnect.AudioAdminServiceUpdateWakeWordTemplateProcedure,
		Method:   "POST",
		Summary:  "Update wake-word template",
		Category: "audio_admin",
	},
	{
		ID:       "audio_admin_delete_wake_word_template",
		Path:     audioadminconnect.AudioAdminServiceDeleteWakeWordTemplateProcedure,
		Method:   "POST",
		Summary:  "Delete wake-word template",
		Category: "audio_admin",
	},
	{
		ID:       "audio_admin_get_speaker_config",
		Path:     audioadminconnect.AudioAdminServiceGetSpeakerConfigProcedure,
		Method:   "POST",
		Summary:  "Get speaker-verification config",
		Category: "audio_admin",
	},
	{
		ID:       "audio_admin_update_speaker_config",
		Path:     audioadminconnect.AudioAdminServiceUpdateSpeakerConfigProcedure,
		Method:   "POST",
		Summary:  "Update speaker-verification config",
		Category: "audio_admin",
	},
	{
		ID:       "audio_admin_get_speaker_status",
		Path:     audioadminconnect.AudioAdminServiceGetSpeakerStatusProcedure,
		Method:   "POST",
		Summary:  "Get speaker-verification status",
		Category: "audio_admin",
	},
	{
		ID:       "audio_admin_list_speaker_profiles",
		Path:     audioadminconnect.AudioAdminServiceListSpeakerProfilesProcedure,
		Method:   "POST",
		Summary:  "List speaker profiles",
		Category: "audio_admin",
	},
	{
		ID:       "audio_admin_enroll_speaker_profile",
		Path:     audioadminconnect.AudioAdminServiceEnrollSpeakerProfileProcedure,
		Method:   "POST",
		Summary:  "Enroll a new speaker profile",
		Category: "audio_admin",
	},
	{
		ID:       "audio_admin_clear_speaker_profile_binding",
		Path:     audioadminconnect.AudioAdminServiceClearSpeakerProfileBindingProcedure,
		Method:   "POST",
		Summary:  "Clear all active speaker-profile bindings",
		Category: "audio_admin",
	},
	{
		ID:       "audio_admin_unbind_speaker_profile",
		Path:     audioadminconnect.AudioAdminServiceUnbindSpeakerProfileProcedure,
		Method:   "POST",
		Summary:  "Unbind a single speaker profile",
		Category: "audio_admin",
	},
	{
		ID:       "audio_admin_delete_speaker_profile",
		Path:     audioadminconnect.AudioAdminServiceDeleteSpeakerProfileProcedure,
		Method:   "POST",
		Summary:  "Delete a speaker profile permanently",
		Category: "audio_admin",
	},
	{
		ID:       "audio_admin_get_tts_config",
		Path:     audioadminconnect.AudioAdminServiceGetTTSConfigProcedure,
		Method:   "POST",
		Summary:  "Get TTS default config",
		Category: "audio_admin",
	},
	{
		ID:       "audio_admin_update_tts_config",
		Path:     audioadminconnect.AudioAdminServiceUpdateTTSConfigProcedure,
		Method:   "POST",
		Summary:  "Update TTS default config",
		Category: "audio_admin",
	},
	{
		ID:       "audio_admin_get_summarize_config",
		Path:     audioadminconnect.AudioAdminServiceGetSummarizeConfigProcedure,
		Method:   "POST",
		Summary:  "Get TTS summarize config",
		Category: "audio_admin",
	},
	{
		ID:       "audio_admin_update_summarize_config",
		Path:     audioadminconnect.AudioAdminServiceUpdateSummarizeConfigProcedure,
		Method:   "POST",
		Summary:  "Update TTS summarize config",
		Category: "audio_admin",
	},
}
