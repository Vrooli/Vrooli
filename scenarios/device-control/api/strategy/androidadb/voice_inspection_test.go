package androidadb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVoiceInspectionReportsConfigurationWithoutRecordingOrMutation(t *testing.T) {
	r := &scriptedRunner{responses: map[string][]byte{
		"adb -s tv shell am get-current-user":                                                                          []byte("0"),
		"adb -s tv shell pm list features":                                                                             []byte("feature:android.hardware.microphone\nfeature:android.software.leanback"),
		"adb -s tv shell cmd role get-role-holders --user 0 android.app.role.ASSISTANT":                                []byte("com.google.android.katniss\n"),
		"adb -s tv shell settings --user 0 get secure assistant":                                                       []byte("com.google.android.katniss/.Assistant"),
		"adb -s tv shell settings --user 0 get secure voice_interaction_service":                                       []byte("com.google.android.katniss/.VoiceService"),
		"adb -s tv shell cmd package query-services --brief --user 0 -a android.service.voice.VoiceInteractionService": []byte("1 services found:\ncom.google.android.katniss/.VoiceService\n"),
		"adb -s tv shell dumpsys media.audio_policy":                                                                   []byte("AUDIO_DEVICE_IN_BUILTIN_MIC address=private-address AUDIO_DEVICE_IN_BLUETOOTH_SCO_HEADSET AUDIO_DEVICE_IN_BUILTIN_MIC"),
	}}
	v := NewWithRunner(r, "tv").inspectVoice(context.Background())
	require.True(t, *v.MicrophoneFeature)
	require.Equal(t, []string{"com.google.android.katniss"}, v.AssistantRoleHolders)
	require.Equal(t, "com.google.android.katniss/.Assistant", *v.AssistantComponent)
	require.Equal(t, []string{"AUDIO_DEVICE_IN_BLUETOOTH_SCO_HEADSET", "AUDIO_DEVICE_IN_BUILTIN_MIC"}, v.PolicyInputTypes)
	require.Empty(t, v.Unavailable)
	require.Len(t, r.calls, 7)
	for _, call := range r.calls {
		require.NotContains(t, call, "settings put")
		require.NotContains(t, call, "add-role-holder")
		require.NotContains(t, call, "am start")
		require.NotContains(t, call, "record")
	}
}

func TestVoiceInspectionPreservesUnknownAndExplicitUnset(t *testing.T) {
	r := &scriptedRunner{responses: map[string][]byte{
		"adb -s tv shell am get-current-user":                                           []byte("0"),
		"adb -s tv shell pm list features":                                              []byte("unknown"),
		"adb -s tv shell settings --user 0 get secure assistant":                        []byte("null"),
		"adb -s tv shell cmd role get-role-holders --user 0 android.app.role.ASSISTANT": []byte("Unknown command: get-role-holders"),
	}}
	v := NewWithRunner(r, "tv").inspectVoice(context.Background())
	require.Nil(t, v.MicrophoneFeature)
	require.NotNil(t, v.AssistantComponent)
	require.Empty(t, *v.AssistantComponent)
	require.NotEmpty(t, v.Unavailable["microphone_feature"])
	require.NotEmpty(t, v.Unavailable["assistant_role_holders"])
	require.Nil(t, v.VoiceInteractionComponent)
}

func TestVoiceInspectionDoesNotMixUsers(t *testing.T) {
	r := &scriptedRunner{responses: map[string][]byte{
		"adb -s tv shell am get-current-user":                                           []byte("10"),
		"adb -s tv shell cmd role get-role-holders --user 0 android.app.role.ASSISTANT": []byte("com.other.owner"),
	}}
	v := NewWithRunner(r, "tv").inspectVoice(context.Background())
	require.Empty(t, v.AssistantRoleHolders)
	require.NotEmpty(t, v.Unavailable["assistant_role_holders"])
	require.Nil(t, v.AssistantComponent)
	require.NotEmpty(t, v.Unavailable["voice_interaction_services"])
}

func TestVoiceInspectionReportsAbsentMicrophoneFeatureAsFalse(t *testing.T) {
	r := &scriptedRunner{responses: map[string][]byte{"adb -s tv shell pm list features": []byte("feature:android.software.leanback\n")}}
	v := NewWithRunner(r, "tv").inspectVoice(context.Background())
	require.NotNil(t, v.MicrophoneFeature)
	require.False(t, *v.MicrophoneFeature)
	require.Empty(t, v.Unavailable["microphone_feature"])
}
