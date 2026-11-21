package sttengine

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func svc(name, display string, resources ...string) string {
	deps := ""
	for i, r := range resources {
		if i > 0 {
			deps += ","
		}
		deps += `"` + r + `":{"required":false}`
	}
	return `{"name":"` + name + `","display_name":"` + display + `","dependencies":{"resources":{` + deps + `}}}`
}

func TestScanResourceConsumers(t *testing.T) {
	fsys := fstest.MapFS{
		"audio-tools/.vrooli/service.json":    {Data: []byte(svc("audio-tools", "Audio Tools", "whisper", "kyutai-stt"))},
		"meeting-notes/.vrooli/service.json":  {Data: []byte(svc("meeting-notes", "Meeting Notes", "whisper"))},
		"podcast-studio/.vrooli/service.json": {Data: []byte(`{"name":"podcast-studio","display_name":"Podcast Studio","dependencies":{"resources":{"whisper":{"required":true}}}}`)},
		"chartmaker/.vrooli/service.json":     {Data: []byte(svc("chartmaker", "Chart Maker", "postgres"))},
		"not-a-scenario/readme.md":            {Data: []byte("hi")},
	}

	// whisper is used by meeting-notes + podcast-studio (audio-tools excluded).
	got, err := ScanResourceConsumers(fsys, "whisper", "audio-tools")
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, "meeting-notes", got[0].Scenario)
	require.Equal(t, "Meeting Notes", got[0].DisplayName)
	require.False(t, got[0].Required)
	require.Equal(t, "podcast-studio", got[1].Scenario)
	require.True(t, got[1].Required, "podcast-studio hard-requires whisper")

	// kyutai-stt is used only by audio-tools, which is excluded → no other consumers.
	got, err = ScanResourceConsumers(fsys, "kyutai-stt", "audio-tools")
	require.NoError(t, err)
	require.Empty(t, got)

	// Unknown resource → empty.
	got, err = ScanResourceConsumers(fsys, "nonexistent", "audio-tools")
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestScanResourceConsumers_NilOrEmpty(t *testing.T) {
	out, err := ScanResourceConsumers(nil, "whisper", "audio-tools")
	require.NoError(t, err)
	require.Nil(t, out)

	out, err = ScanResourceConsumers(fstest.MapFS{}, "", "audio-tools")
	require.NoError(t, err)
	require.Nil(t, out)
}
