// Package byok hosts the per-capability BYOK adapter registries. Provider
// declarations select reusable protocol factories through byok_adapter_id;
// provider IDs remain request and result identity.
package byok

import (
	"fmt"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/ai/ttschain"
	"audio-tools/internal/providercatalog"
)

type Registries struct {
	STT       map[string]sttchain.BYOKAdapter
	TTS       map[string]ttschain.BYOKAdapter
	Summarize map[string]summarizechain.BYOKAdapter
}

// protocolFactories are keyed by declared protocol identity, never by provider
// ID. A factory may therefore serve a newly declared provider without adding a
// provider-specific branch here.
type protocolFactories struct {
	stt       map[string]func(string) sttchain.BYOKAdapter
	tts       map[string]func(string) ttschain.BYOKAdapter
	summarize map[string]func(string) summarizechain.BYOKAdapter
}

func defaultProtocolFactories() protocolFactories {
	return protocolFactories{
		stt: map[string]func(string) sttchain.BYOKAdapter{
			"stt.openai-whisper-v1": func(id string) sttchain.BYOKAdapter { a := NewOpenAIWhisperSTT(); a.ProviderID = id; return a },
			"stt.deepgram-v1":       func(id string) sttchain.BYOKAdapter { a := NewDeepgramSTT(); a.ProviderID = id; return a },
		},
		tts: map[string]func(string) ttschain.BYOKAdapter{
			"tts.openai-speech-v1": func(id string) ttschain.BYOKAdapter { a := NewOpenAITTS(); a.ProviderID = id; return a },
			"tts.elevenlabs-v1":    func(id string) ttschain.BYOKAdapter { a := NewElevenLabsTTS(); a.ProviderID = id; return a },
		},
		summarize: map[string]func(string) summarizechain.BYOKAdapter{
			"summarize.openrouter-cli-v1": func(id string) summarizechain.BYOKAdapter { a := NewOpenRouterSummarize(); a.ProviderID = id; return a },
		},
	}
}

// NewRegistries builds adapters for active credential providers in the
// validated provider catalog. The catalog chooses protocol factories; factory
// outputs stamp their canonical catalog provider ID in result identity.
func NewRegistries() Registries {
	r, err := NewRegistriesForProviders(providercatalog.Default().Active())
	if err != nil {
		panic(err)
	}
	return r
}

// NewRegistriesForProviders is exported to support catalog conformance tests
// and alternate validated catalog snapshots.
func NewRegistriesForProviders(providers []providercatalog.Provider) (Registries, error) {
	return buildRegistries(providers, defaultProtocolFactories())
}

func buildRegistries(providers []providercatalog.Provider, factories protocolFactories) (Registries, error) {
	r := Registries{STT: map[string]sttchain.BYOKAdapter{}, TTS: map[string]ttschain.BYOKAdapter{}, Summarize: map[string]summarizechain.BYOKAdapter{}}
	for _, p := range providers {
		if p.Status != providercatalog.StatusActive || p.Tier != providercatalog.TierBYOK {
			continue
		}
		if p.BYOKAdapterID == "" {
			return Registries{}, fmt.Errorf("active BYOK provider %q has no byok_adapter_id", p.ID)
		}
		for _, capability := range p.Capabilities {
			switch capability {
			case providercatalog.CapabilitySTT:
				factory := factories.stt[p.BYOKAdapterID]
				if factory == nil {
					return Registries{}, fmt.Errorf("provider %q declares unsupported STT BYOK adapter %q", p.ID, p.BYOKAdapterID)
				}
				r.STT[p.ID] = factory(p.ID)
			case providercatalog.CapabilityTTS:
				factory := factories.tts[p.BYOKAdapterID]
				if factory == nil {
					return Registries{}, fmt.Errorf("provider %q declares unsupported TTS BYOK adapter %q", p.ID, p.BYOKAdapterID)
				}
				r.TTS[p.ID] = factory(p.ID)
			case providercatalog.CapabilitySummarize:
				factory := factories.summarize[p.BYOKAdapterID]
				if factory == nil {
					return Registries{}, fmt.Errorf("provider %q declares unsupported summarize BYOK adapter %q", p.ID, p.BYOKAdapterID)
				}
				r.Summarize[p.ID] = factory(p.ID)
			}
		}
	}
	return r, nil
}
