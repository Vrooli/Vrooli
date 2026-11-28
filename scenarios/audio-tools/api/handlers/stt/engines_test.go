package stt

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
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

type inventoryProvider struct {
	id                  string
	serviceable         bool
	availabilityCalls   int
	serviceabilityCalls int
}

func (p *inventoryProvider) Type() sttchain.ProviderTier { return sttchain.TierLocal }
func (p *inventoryProvider) IsAvailable(context.Context) bool {
	p.availabilityCalls++
	return p.serviceable
}

func (p *inventoryProvider) IsServiceable(context.Context) bool {
	p.serviceabilityCalls++
	return p.serviceable
}
func (p *inventoryProvider) Model() string                   { return p.id }
func (p *inventoryProvider) Traits() sttchain.ProviderTraits { return sttchain.ProviderTraits{} }
func (p *inventoryProvider) Transcribe(context.Context, sttchain.Request) (*sttchain.Result, error) {
	return nil, nil
}

func (p *inventoryProvider) TranscribeStreaming(context.Context, sttchain.StreamStart, <-chan sttchain.AudioChunk) (<-chan sttchain.StreamEvent, error) {
	return nil, nil
}

func TestListEnginesUsesReadOnlyProviderLiveness(t *testing.T) {
	registry := sttengine.Default()
	providers := make(map[string]sttchain.Provider, len(registry.Engines()))
	byID := make(map[string]*inventoryProvider, len(registry.Engines()))
	for _, engine := range registry.Engines() {
		provider := &inventoryProvider{id: engine.ID, serviceable: engine.ID == "sherpa-streaming"}
		providers[engine.ID] = provider
		byID[engine.ID] = provider
	}
	chain := sttchain.NewChain(sttchain.Options{
		LocalEngines: providers, EnableLocal: true,
	})
	var controlPlaneCalls int
	resolver := sttengine.NewLiveResolver(func(context.Context, ...string) ([]byte, error) {
		controlPlaneCalls++
		return nil, errors.New("inventory must not shell out to the control plane")
	}, time.Minute)

	client := newSTTRuntimeClient(t, Deps{Registry: registry, Chain: chain, EngineResolver: resolver})
	resp, err := client.ListEngines(context.Background(), connect.NewRequest(&sttv1.ListEnginesRequest{}))
	require.NoError(t, err)
	require.NotEmpty(t, resp.Msg.GetEngines())
	require.Zero(t, controlPlaneCalls, "inventory must use the bounded, read-only provider liveness seam")
	for _, engine := range resp.Msg.GetEngines() {
		if engine.GetId() == "whisper-local" {
			require.False(t, engine.GetAvailable())
			require.Contains(t, engine.GetReason(), "not_serviceable")
			require.NotContains(t, engine.GetReason(), "not_installed")
		}
	}
	for _, provider := range byID {
		require.Zero(t, provider.availabilityCalls, "read-only listing must never call lifecycle-reconciling IsAvailable")
		require.Equal(t, 1, provider.serviceabilityCalls, "each local engine should be probed once and the result reused for projection")
	}
}
