package stt

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"audio-tools/internal/sttengine"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

// writeScenario writes a fixture scenarios/<name>/.vrooli/service.json.
func writeScenario(t *testing.T, root, name string, body string) {
	t.Helper()
	dir := filepath.Join(root, name, ".vrooli")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "service.json"), []byte(body), 0o644))
}

func TestGetEngineSwitchImpact_ListsOtherConsumers(t *testing.T) {
	root := t.TempDir()
	writeScenario(t, root, "audio-tools", `{"name":"audio-tools","dependencies":{"resources":{"kyutai-stt":{"required":false},"whisper":{"required":false}}}}`)
	writeScenario(t, root, "meeting-notes", `{"name":"meeting-notes","display_name":"Meeting Notes","dependencies":{"resources":{"whisper":{"required":false}}}}`)
	writeScenario(t, root, "lecture-capture", `{"name":"lecture-capture","display_name":"Lecture Capture","dependencies":{"resources":{"whisper":{"required":true}}}}`)
	t.Setenv("VROOLI_SCENARIOS_DIR", root)
	t.Setenv("VROOLI_SCENARIO_DIR", filepath.Join(root, "audio-tools"))

	c := newSTTClient(t, Deps{Registry: sttengine.Default()})
	resp, err := c.GetEngineSwitchImpact(context.Background(), connect.NewRequest(&sttv1.GetEngineSwitchImpactRequest{
		FromEngineId: "whisper-local",
	}))
	require.NoError(t, err)
	msg := resp.Msg
	require.Equal(t, "whisper", msg.GetResource())
	require.Equal(t, "vrooli resource stop whisper", msg.GetStopCommand())
	require.True(t, msg.GetConsumersKnown())
	require.False(t, msg.GetSafeToStop(), "whisper is still used by two other scenarios")
	require.Len(t, msg.GetConsumers(), 2)
	require.Equal(t, "lecture-capture", msg.GetConsumers()[0].GetScenario())
	require.True(t, msg.GetConsumers()[0].GetRequired())
	require.Equal(t, "Meeting Notes", msg.GetConsumers()[1].GetDisplayName())
}

func TestGetEngineSwitchImpact_SafeWhenNoOtherConsumer(t *testing.T) {
	root := t.TempDir()
	writeScenario(t, root, "audio-tools", `{"name":"audio-tools","dependencies":{"resources":{"kyutai-stt":{"required":false}}}}`)
	t.Setenv("VROOLI_SCENARIOS_DIR", root)
	t.Setenv("VROOLI_SCENARIO_DIR", filepath.Join(root, "audio-tools"))

	c := newSTTClient(t, Deps{Registry: sttengine.Default()})
	resp, err := c.GetEngineSwitchImpact(context.Background(), connect.NewRequest(&sttv1.GetEngineSwitchImpactRequest{
		FromEngineId: "kyutai",
	}))
	require.NoError(t, err)
	require.Equal(t, "kyutai-stt", resp.Msg.GetResource())
	require.True(t, resp.Msg.GetConsumersKnown())
	require.True(t, resp.Msg.GetSafeToStop(), "no other scenario uses kyutai-stt")
	require.Empty(t, resp.Msg.GetConsumers())
}

func TestGetEngineSwitchImpact_UnknownEngineIsSafe(t *testing.T) {
	c := newSTTClient(t, Deps{Registry: sttengine.Default()})
	resp, err := c.GetEngineSwitchImpact(context.Background(), connect.NewRequest(&sttv1.GetEngineSwitchImpactRequest{
		FromEngineId: "does-not-exist",
	}))
	require.NoError(t, err)
	require.True(t, resp.Msg.GetSafeToStop())
	require.Empty(t, resp.Msg.GetResource())
}

func TestGetEngineSwitchImpact_ScenariosDirUnknown(t *testing.T) {
	t.Setenv("VROOLI_SCENARIOS_DIR", "")
	t.Setenv("VROOLI_SCENARIO_DIR", "")
	c := newSTTClient(t, Deps{Registry: sttengine.Default()})
	resp, err := c.GetEngineSwitchImpact(context.Background(), connect.NewRequest(&sttv1.GetEngineSwitchImpactRequest{
		FromEngineId: "kyutai",
	}))
	require.NoError(t, err)
	// Resource + command are known; consumer enumeration is not, so the
	// prompt conservatively reports not-safe-to-stop.
	require.Equal(t, "kyutai-stt", resp.Msg.GetResource())
	require.False(t, resp.Msg.GetConsumersKnown())
	require.False(t, resp.Msg.GetSafeToStop())
}

func TestListEngines_ProjectsManifestAndAvailability(t *testing.T) {
	c := newSTTRuntimeClient(t, Deps{Registry: sttengine.Default()})
	resp, err := c.ListEngines(context.Background(), connect.NewRequest(&sttv1.ListEnginesRequest{}))
	require.NoError(t, err)
	require.NotEmpty(t, resp.Msg.GetEngines())
	for _, engine := range resp.Msg.GetEngines() {
		require.NotEmpty(t, engine.GetId())
		require.NotEmpty(t, engine.GetDisplayName())
	}
}

func TestListEngines_EmptyWithoutRegistry(t *testing.T) {
	c := newSTTRuntimeClient(t, Deps{})
	resp, err := c.ListEngines(context.Background(), connect.NewRequest(&sttv1.ListEnginesRequest{}))
	require.NoError(t, err)
	require.Empty(t, resp.Msg.GetEngines())
}
