package models

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"audio-tools/internal/modelcatalog"
	trustfloor "audio-tools/internal/qualification"

	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/models"
)

func newCatalogHandler(t *testing.T) *connectHandler {
	t.Helper()
	return NewConnectHandler(Deps{Catalog: modelcatalog.Default()})
}

func listModels(t *testing.T, h *connectHandler, req *modelsv1.ListModelsRequest) []*modelsv1.ModelRecord {
	t.Helper()
	res, err := h.ListModels(context.Background(), connect.NewRequest(req))
	require.NoError(t, err)
	return res.Msg.GetModels()
}

func TestListModels_ReturnsWholeRegistryInCanonicalOrder(t *testing.T) {
	h := newCatalogHandler(t)
	records := listModels(t, h, &modelsv1.ListModelsRequest{})
	require.Len(t, records, len(modelcatalog.Default().All()))

	seen := make(map[string]struct{}, len(records))
	lastCapability := -1
	lastID := ""
	for _, record := range records {
		if _, dup := seen[record.GetId()]; dup {
			t.Fatalf("duplicate model id %q", record.GetId())
		}
		seen[record.GetId()] = struct{}{}

		order, ok := modelcatalog.CapabilityOrder(modelcatalog.Capability(record.GetCapability()))
		require.True(t, ok, "record %q has unknown capability %q", record.GetId(), record.GetCapability())
		if order < lastCapability {
			t.Fatalf("capability order regressed at %q", record.GetId())
		}
		if order > lastCapability {
			lastCapability = order
		} else if record.GetId() < lastID {
			t.Fatalf("ids not sorted within capability at %q", record.GetId())
		}
		lastID = record.GetId()

		require.NotEqual(t, modelsv1.Readiness_READINESS_UNSPECIFIED, record.GetReadiness(),
			"record %q has an unmapped readiness", record.GetId())
		require.NotEqual(t, modelsv1.CommercialUse_COMMERCIAL_USE_UNSPECIFIED, record.GetCommercialUse(),
			"record %q has an unmapped commercial use", record.GetId())
	}
}

func TestListModels_IsDeterministic(t *testing.T) {
	h := newCatalogHandler(t)
	first := listModels(t, h, &modelsv1.ListModelsRequest{})
	second := listModels(t, h, &modelsv1.ListModelsRequest{})
	require.Len(t, second, len(first))
	for i := range first {
		require.Equal(t, first[i].GetId(), second[i].GetId())
	}
}

func TestPlatformFromProtoCarriesCPUEnvelopeWithoutInferringQualification(t *testing.T) {
	platform := platformFromProto(&modelsv1.Platform{Os: "linux", Arch: "amd64", RamGb: 8, CpuLimit: 2})
	require.Equal(t, 2, platform.CPULimit)
	require.Equal(t, 8, platform.RAMGB)
	// No model CPU minimum is declared, so carrying a quota must not turn it
	// into an implicit qualification or selection rule.
	model, err := modelcatalog.Default().DefaultFor(modelcatalog.CapabilitySummarize, "", platform)
	require.NoError(t, err)
	require.Equal(t, "qwen3:1.7b", model.ID)
	moreCPU, err := modelcatalog.Default().DefaultFor(modelcatalog.CapabilitySummarize, "", modelcatalog.Platform{OS: "linux", Arch: "amd64", RAMGB: 8, CPULimit: 16})
	require.NoError(t, err)
	require.Equal(t, model.ID, moreCPU.ID)
}

func TestResolveModelRefusesSummarizeDefaultBelowDeclaredRAMFloor(t *testing.T) {
	h := newCatalogHandler(t)
	_, err := h.ResolveModel(context.Background(), connect.NewRequest(&modelsv1.ResolveModelRequest{
		Capability: "summarize",
		Commercial: true,
		Platform:   &modelsv1.Platform{Os: "linux", Arch: "amd64", RamGb: 4, CpuLimit: 2},
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	require.Contains(t, err.Error(), "requires at least 6 GB RAM; host reports 4 GB")
}

func TestListModelsExposesRegistryOwnedRuntimeArtifactIdentity(t *testing.T) {
	records := listModels(t, newCatalogHandler(t), &modelsv1.ListModelsRequest{})
	var sherpa *modelsv1.ModelRecord
	var external *modelsv1.ModelRecord
	for _, record := range records {
		switch record.GetId() {
		case "sherpa-zipformer-en-2023-06-26":
			sherpa = record
		case "qwen3:1.7b":
			external = record
		}
	}
	require.NotNil(t, sherpa)
	require.Equal(t, "sherpa-onnx/streaming-zipformer-en-2023-06-26", sherpa.GetRuntimeModelId())
	require.Equal(t, "sherpa-onnx", sherpa.GetArtifactResource())
	require.Equal(t, "streaming-zipformer-en-2023-06-26", sherpa.GetArtifactName())
	require.NotNil(t, external)
	require.Empty(t, external.GetRuntimeModelId())
	require.Empty(t, external.GetArtifactResource())
	require.Empty(t, external.GetArtifactName())
}

func TestListModels_CapabilityFilterAndUnknownValue(t *testing.T) {
	h := newCatalogHandler(t)
	records := listModels(t, h, &modelsv1.ListModelsRequest{Capability: "stt"})
	require.NotEmpty(t, records)
	for _, record := range records {
		require.Equal(t, "stt", record.GetCapability())
	}

	_, err := h.ListModels(context.Background(), connect.NewRequest(&modelsv1.ListModelsRequest{Capability: "speech-recognition"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestListModels_CommercialOnlyExcludesNonCommercialAndUnknown(t *testing.T) {
	h := newCatalogHandler(t)
	records := listModels(t, h, &modelsv1.ListModelsRequest{CommercialOnly: true})
	require.NotEmpty(t, records)
	defaults := make(map[string]int)
	for _, record := range records {
		require.Equal(t, modelsv1.CommercialUse_COMMERCIAL_USE_ALLOWED, record.GetCommercialUse(),
			"record %q leaked a non-commercial licence", record.GetId())
		if record.GetIsDefault() {
			defaults[record.GetCapability()]++
		}
	}
	for _, capability := range modelcatalog.DefaultCapabilities {
		require.Equal(t, 1, defaults[string(capability)], "commercial capability %s", capability)
	}
	require.NotContains(t, ids(records), "reverb-tts")
	require.NotContains(t, ids(records), "verba-tts-indic-zero")
}

func TestListModels_OnlyReadyExcludesUnreadyRecords(t *testing.T) {
	h := newCatalogHandler(t)
	records := listModels(t, h, &modelsv1.ListModelsRequest{OnlyReady: true})
	require.NotEmpty(t, records)
	for _, record := range records {
		require.Equal(t, modelsv1.Readiness_READINESS_AVAILABLE, record.GetReadiness(),
			"record %q is not available", record.GetId())
	}
	require.NotContains(t, ids(records), "whisper-large-v3")
}

func TestListModels_LanguageFilterUsesPrimarySubtag(t *testing.T) {
	h := newCatalogHandler(t)
	records := listModels(t, h, &modelsv1.ListModelsRequest{Language: "EN-us"})
	require.NotEmpty(t, records)
	for _, record := range records {
		matched := false
		for _, tag := range record.GetLanguages() {
			if tag == "en" || tag == "mul" || tag == "und" {
				matched = true
			}
		}
		require.True(t, matched, "record %q returned for en-US but declares %v", record.GetId(), record.GetLanguages())
	}
}

func TestListModels_MarksExactlyOneDefaultPerCapability(t *testing.T) {
	h := newCatalogHandler(t)
	records := listModels(t, h, &modelsv1.ListModelsRequest{})
	counts := map[string]int{}
	for _, record := range records {
		if record.GetIsDefault() {
			counts[record.GetCapability()]++
		}
	}
	for _, capability := range modelcatalog.DefaultCapabilities {
		require.Equal(t, 1, counts[string(capability)], "capability %s", capability)
	}
}

func TestResolveModel_SelectsPromotedDefault(t *testing.T) {
	h := newCatalogHandler(t)
	res, err := h.ResolveModel(context.Background(), connect.NewRequest(&modelsv1.ResolveModelRequest{Capability: "stt"}))
	require.NoError(t, err)
	require.Equal(t, "whisper-small", res.Msg.GetModel().GetId())
	require.True(t, res.Msg.GetModel().GetIsDefault())
	require.Contains(t, res.Msg.GetExplanation(), "whisper-small")

	res, err = h.ResolveModel(context.Background(), connect.NewRequest(&modelsv1.ResolveModelRequest{Capability: "tts"}))
	require.NoError(t, err)
	require.Equal(t, "kokoro-82m-v1.0", res.Msg.GetModel().GetId())
}

func TestResolveModelCanUseEngineScopedDefault(t *testing.T) {
	h := newCatalogHandler(t)
	res, err := h.ResolveModel(context.Background(), connect.NewRequest(&modelsv1.ResolveModelRequest{
		Capability: "stt", EngineId: "sherpa-streaming", Language: "en", Commercial: true,
	}))
	require.NoError(t, err)
	require.Equal(t, "sherpa-zipformer-en-2023-06-26", res.Msg.GetModel().GetId())
	require.Equal(t, []string{"sherpa-streaming"}, res.Msg.GetModel().GetEngineDefaults())
	require.True(t, res.Msg.GetModel().GetIsDefault())
	require.Contains(t, res.Msg.GetExplanation(), "engine=sherpa-streaming")
	require.Equal(t, modelsv1.CommercialUse_COMMERCIAL_USE_ALLOWED, res.Msg.GetModel().GetCommercialUse())

	_, err = h.ResolveModel(context.Background(), connect.NewRequest(&modelsv1.ResolveModelRequest{
		Capability: "tts", EngineId: "sherpa-streaming",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

type emptyQualificationStore struct{}

func (emptyQualificationStore) ListQualificationEvidence(context.Context, trustfloor.QualificationEvidenceFilter) ([]trustfloor.QualificationEvidence, error) {
	return nil, nil
}

func TestPromotionSelectionKeepsLegacyDefaultAndBlocksIncompleteDeclaration(t *testing.T) {
	legacy, ok := modelcatalog.Default().ByID("kokoro-82m-v1.0")
	require.True(t, ok)
	candidate := legacy
	candidate.ID = "aaa-new-tts"
	candidate.Name = "New TTS candidate"
	candidate.Promotion.LegacyException = ""
	candidate.Promotion.CorpusQuality = modelcatalog.EvidenceFacet{Status: "qualified", Ref: "fabricated-corpus-ref"}
	candidate.Promotion.Latency = modelcatalog.EvidenceFacet{Status: "qualified", Ref: "fabricated-latency-ref"}
	candidate.Promotion.Memory = modelcatalog.EvidenceFacet{Status: "qualified", Ref: "fabricated-memory-ref"}
	candidate.Promotion.Platform = modelcatalog.EvidenceFacet{Status: "qualified", Ref: "fabricated-platform-ref"}
	candidate.Promotion.License = modelcatalog.EvidenceFacet{Status: "qualified", Ref: "fabricated-license-ref"}
	catalog := modelcatalog.New(modelcatalog.Seed{Models: []modelcatalog.Model{candidate, legacy}})
	h := NewConnectHandler(Deps{Catalog: catalog, QualificationEvidence: emptyQualificationStore{}})

	resolved, err := h.ResolveModel(context.Background(), connect.NewRequest(&modelsv1.ResolveModelRequest{Capability: "tts"}))
	require.NoError(t, err)
	require.Equal(t, legacy.ID, resolved.Msg.GetModel().GetId())
	require.True(t, resolved.Msg.GetModel().GetIsDefault())

	records := listModels(t, h, &modelsv1.ListModelsRequest{Capability: "tts"})
	byID := make(map[string]*modelsv1.ModelRecord, len(records))
	for _, record := range records {
		byID[record.GetId()] = record
	}
	require.True(t, byID[legacy.ID].GetIsDefault())
	require.True(t, byID[legacy.ID].GetDefaultEligible())
	require.False(t, byID[candidate.ID].GetIsDefault())
	require.False(t, byID[candidate.ID].GetDefaultEligible())
}

func TestPromotionSelectionKeepsAllLegacyDefaults(t *testing.T) {
	h := NewConnectHandler(Deps{Catalog: modelcatalog.Default(), QualificationEvidence: emptyQualificationStore{}})
	want := map[modelcatalog.Capability]string{
		modelcatalog.CapabilitySTT:       "whisper-small",
		modelcatalog.CapabilityTTS:       "kokoro-82m-v1.0",
		modelcatalog.CapabilitySummarize: "qwen3:1.7b",
		modelcatalog.CapabilitySpeaker:   "sherpa-onnx-speaker-embedding",
	}
	for capability, id := range want {
		res, err := h.ResolveModel(context.Background(), connect.NewRequest(&modelsv1.ResolveModelRequest{Capability: string(capability)}))
		require.NoError(t, err, "capability %s", capability)
		require.Equal(t, id, res.Msg.GetModel().GetId())
		require.True(t, res.Msg.GetModel().GetIsDefault())
	}
}

func TestResolveModel_CommercialNeverReturnsNonCommercial(t *testing.T) {
	h := newCatalogHandler(t)
	for _, capability := range modelcatalog.DefaultCapabilities {
		res, err := h.ResolveModel(context.Background(), connect.NewRequest(&modelsv1.ResolveModelRequest{
			Capability: string(capability),
			Commercial: true,
		}))
		require.NoError(t, err, "capability %s", capability)
		require.Equal(t, modelsv1.CommercialUse_COMMERCIAL_USE_ALLOWED, res.Msg.GetModel().GetCommercialUse(),
			"capability %s resolved a non-commercial model", capability)
		require.True(t, res.Msg.GetModel().GetIsDefault(),
			"capability %s resolved its model but did not mark it as the commercial default", capability)
	}
}

func TestResolveModel_RejectsMissingOrUnknownCapability(t *testing.T) {
	h := newCatalogHandler(t)
	for _, capability := range []string{"", "video"} {
		_, err := h.ResolveModel(context.Background(), connect.NewRequest(&modelsv1.ResolveModelRequest{Capability: capability}))
		require.Error(t, err, "capability=%q", capability)
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	}
}

func TestResolveModel_NoEligibleModelIsNotFound(t *testing.T) {
	// A validated production seed always has a promoted default per
	// capability; this fixture deliberately does not, to prove the handler
	// reports "no eligible default" instead of inventing one.
	catalog := modelcatalog.New(modelcatalog.Seed{
		SelectionPolicy: modelcatalog.SelectionPolicy{
			ReadinessRank:    []string{"available", "candidate", "experimental", "watch"},
			RoleOrder:        []string{"default", "fast", "quality", "edge"},
			LanguageFallback: "none",
		},
		Models: []modelcatalog.Model{{
			ID: "only-tts", Name: "Only TTS", ProviderIDs: []string{"kokoro-tts"}, Capability: modelcatalog.CapabilityTTS,
			Runtime: "kokoro", Languages: []string{"en"}, License: "Apache-2.0", Commercial: modelcatalog.CommercialYes,
			Readiness: modelcatalog.ReadinessAvailable, Roles: []string{"fast"},
		}},
	})
	h := NewConnectHandler(Deps{Catalog: catalog})

	_, err := h.ResolveModel(context.Background(), connect.NewRequest(&modelsv1.ResolveModelRequest{Capability: "tts"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestNewConnectHandlerRequiresCatalog(t *testing.T) {
	require.Panics(t, func() { NewConnectHandler(Deps{}) })
}

func ids(records []*modelsv1.ModelRecord) []string {
	out := make([]string, 0, len(records))
	for _, record := range records {
		out = append(out, record.GetId())
	}
	return out
}

func TestListModels_PlatformConstrainsDefaultFlag(t *testing.T) {
	h := newCatalogHandler(t)
	tight := listModels(t, h, &modelsv1.ListModelsRequest{Platform: &modelsv1.Platform{RamGb: 1}})
	for _, record := range tight {
		require.False(t, record.GetIsDefault(),
			"record %q is marked default on a 1 GB host", record.GetId())
	}

	roomy := listModels(t, h, &modelsv1.ListModelsRequest{Platform: &modelsv1.Platform{RamGb: 32, Accelerated: true, FreeVramGb: 24}})
	require.NotEmpty(t, ids(roomy))
	counts := map[string]int{}
	for _, record := range roomy {
		if record.GetIsDefault() {
			counts[record.GetCapability()]++
		}
	}
	for _, capability := range modelcatalog.DefaultCapabilities {
		require.Equal(t, 1, counts[string(capability)], "capability %s", capability)
	}
}

func TestResolveModel_PlatformConstraintCanMakeDefaultIneligible(t *testing.T) {
	h := newCatalogHandler(t)
	_, err := h.ResolveModel(context.Background(), connect.NewRequest(&modelsv1.ResolveModelRequest{
		Capability: "summarize",
		Platform:   &modelsv1.Platform{RamGb: 1},
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	res, err := h.ResolveModel(context.Background(), connect.NewRequest(&modelsv1.ResolveModelRequest{
		Capability: "summarize",
		Platform:   &modelsv1.Platform{RamGb: 32},
	}))
	require.NoError(t, err)
	require.Equal(t, "qwen3:1.7b", res.Msg.GetModel().GetId())
}
