package androidadb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInspectionReadsHardwareProviderAndHomeWithoutWrites(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s tv shell am get-current-user":            []byte("0"),
		"adb -s tv shell getprop ro.product.cpu.abilist": []byte("armeabi-v7a,armeabi"),
		"adb -s tv shell cat /proc/meminfo":              []byte("MemTotal:  1500000 kB\n"),
		"adb -s tv shell df -k /data":                    []byte("Filesystem 1K-blocks Used Available Use% Mounted on\n/dev/block/dm-2 8000000 2000000 6000000 25% /mnt/userdata\n"),
		"adb -s tv shell dumpsys webviewupdate":          []byte("Current WebView package (name, version): (com.google.android.webview, 155.0.8059.43)\n"),
		"adb -s tv shell cmd package resolve-activity --brief --user 0 -a android.intent.action.MAIN -c android.intent.category.HOME": []byte("com.google.android.apps.tv.launcherx/.HomeActivity\n"),
		"adb -s tv shell getprop ro.config.low_ram": []byte("true"),
		"adb -s tv shell dumpsys media.player":      []byte("Encoder infos by media types:\nMedia type 'video/avc':\n  Encoder \"c2.android.avc.encoder\" supports\n    software-only: 1,\n    hw-accelerated: 0 ]\n"),
	}}
	profile, err := NewWithRunner(runner, "tv").Inspect(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"armeabi-v7a", "armeabi"}, profile.ABIs)
	require.Equal(t, uint64(1536000000), *profile.MemoryTotalBytes)
	require.Equal(t, uint64(6000000*1024), *profile.DataAvailableBytes)
	require.Equal(t, "com.google.android.webview", profile.WebViewPackage)
	require.Equal(t, "155.0.8059.43", profile.WebViewVersion)
	require.Equal(t, "com.google.android.apps.tv.launcherx/com.google.android.apps.tv.launcherx.HomeActivity", profile.HomeComponent)
	require.True(t, *profile.LowRAM)
	require.Len(t, profile.VideoEncoders, 1)
	require.True(t, *profile.VideoEncoders[0].SoftwareOnly)
	require.False(t, *profile.VideoEncoders[0].HardwareAccelerated)
	require.Empty(t, profile.Unavailable)
	require.Contains(t, runner.calls, "adb -s tv shell am get-current-user")
	for _, call := range runner.calls {
		require.NotContains(t, call, "settings put")
		require.NotContains(t, call, "am start")
	}
}

func TestVideoEncoderInventoryExcludesDecodersAndUnknownFlags(t *testing.T) {
	data := `Decoder infos by media types:
Media type 'video/avc':
  Encoder "not-an-encoder" supports
Encoder infos by media types:
Media type 'video/avc':
  Encoder "codec.hardware" supports
    software-only: 0,
    hw-accelerated: 1 ]
  Encoder "codec.unknown" supports
Media type 'audio/avc':
  Encoder "private-audio" supports
`
	encoders := parseVideoEncoders(data)
	require.Len(t, encoders, 2)
	require.Equal(t, "video/avc", encoders[0].MIMEType)
	require.False(t, *encoders[0].SoftwareOnly)
	require.True(t, *encoders[0].HardwareAccelerated)
	require.Equal(t, "codec.unknown", encoders[1].Name)
	require.Nil(t, encoders[1].HardwareAccelerated)
}

func TestInspectionPreservesUnknownRatherThanZero(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s tv shell am get-current-user":            []byte("0"),
		"adb -s tv shell getprop ro.product.cpu.abilist": []byte("armeabi-v7a"),
	}}
	profile, err := NewWithRunner(runner, "tv").Inspect(context.Background())
	require.NoError(t, err)
	require.Nil(t, profile.MemoryTotalBytes)
	require.Nil(t, profile.LowRAM)
	require.Nil(t, profile.DataAvailableBytes)
	require.NotEmpty(t, profile.Unavailable["webview"])
}
