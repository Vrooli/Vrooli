package settings

import (
	"testing"

	"audio-tools/internal/ai/chains"
	"audio-tools/internal/ai/ttschain"
	ttsmocks "audio-tools/internal/ai/ttschain/mocks"
	"audio-tools/internal/store"
)

func TestReconfigureProviderCoordinatorHonorsEnvironmentUpperBound(t *testing.T) {
	coordinator := &chains.Coordinator{TTS: ttschain.NewChain(ttschain.Options{
		BYOK: ttschain.NewBYOKProvider(map[string]ttschain.BYOKAdapter{
			"openai-tts": &ttsmocks.FakeBYOK{IDStr: "openai-tts", Available: true},
		}),
		EnableBYOK: true,
	})}
	config := store.ProviderConfig{BYOKEnabled: true}
	limits := &chains.Config{BYOKEnabled: false, VrooliEnabled: true, LocalEnabled: true}
	reconfigureProviderCoordinator(coordinator, config, limits)
	if got := coordinator.Probe(t.Context()).TTS.BYOK; got {
		t.Fatal("settings update enabled BYOK above the environment maximum")
	}

	config.BYOKEnabled = false
	*limits = chains.Config{BYOKEnabled: true, VrooliEnabled: true, LocalEnabled: true}
	reconfigureProviderCoordinator(coordinator, config, limits)
	if got := coordinator.Probe(t.Context()).TTS.BYOK; got {
		t.Fatal("settings update ignored the user's persisted BYOK disable")
	}
}
