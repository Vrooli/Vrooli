package bootstrap

import (
	"reflect"
	"testing"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/providercatalog"
	"audio-tools/internal/sttengine"
)

func TestSTTBackendResourcesProjectDeclaredLocalEngines(t *testing.T) {
	registry := sttengine.Default()
	want := make([]string, 0)
	for _, engine := range registry.Engines() {
		if engine.Kind == sttengine.KindLocalResource && engine.Resource != "" {
			want = append(want, engine.Resource)
		}
	}
	if got := sttBackendResources(registry); !reflect.DeepEqual(got, want) {
		t.Fatalf("STT recovery resources = %v, want manifest projection %v", got, want)
	}
	if got := sttBackendResources(nil); got != nil {
		t.Fatalf("nil registry resources = %v, want nil", got)
	}
}

func TestLocalSTTEngineAdaptersProjectCatalogEngineIDs(t *testing.T) {
	whisper := sttchain.NewVirtualReplayProvider()
	streaming := sttchain.NewVirtualReplayProvider()
	sherpa := sttchain.NewVirtualReplayProvider()
	got := declaredLocalSTTEngineProviders(whisper, streaming, sherpa)
	want := 0
	for _, provider := range providercatalog.All() {
		if provider.Status != providercatalog.StatusActive || provider.Tier != providercatalog.TierLocal {
			continue
		}
		for _, engineID := range provider.EngineIDs {
			want++
			adapter, ok := got[engineID]
			if !ok || adapter == nil {
				t.Errorf("declared engine %q for provider %q has no runtime adapter", engineID, provider.ID)
			}
		}
	}
	if len(got) != want {
		t.Fatalf("runtime engine map has %d entries, catalog declares %d local engines", len(got), want)
	}
}

func TestLocalSTTEngineAdapterCanBeAddedByDeclaration(t *testing.T) {
	adapter := sttchain.NewVirtualReplayProvider()
	declarations := []providercatalog.Provider{{
		ID: "new-provider", Status: providercatalog.StatusActive,
		Tier: providercatalog.TierLocal, STTEvalAdapterID: "sherpa-stream",
		EngineIDs: []string{"new-streaming-engine"},
	}}
	got := declaredLocalSTTEngineProvidersFor(declarations, map[string]sttchain.Provider{"sherpa-stream": adapter})
	if got["new-streaming-engine"] != adapter {
		t.Fatalf("new declaration resolved to %v, want its registered adapter", got["new-streaming-engine"])
	}
}
