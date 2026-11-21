package models

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"audio-tools/internal/modelcatalog"

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
			lastID = ""
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
	for _, record := range records {
		require.Equal(t, modelsv1.CommercialUse_COMMERCIAL_USE_ALLOWED, record.GetCommercialUse(),
			"record %q leaked a non-commercial licence", record.GetId())
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
	for _, capability := range modelcatalog.Capabilities {
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

func TestResolveModel_CommercialNeverReturnsNonCommercial(t *testing.T) {
	h := newCatalogHandler(t)
	for _, capability := range modelcatalog.Capabilities {
		res, err := h.ResolveModel(context.Background(), connect.NewRequest(&modelsv1.ResolveModelRequest{
			Capability: string(capability),
			Commercial: true,
		}))
		require.NoError(t, err, "capability %s", capability)
		require.Equal(t, modelsv1.CommercialUse_COMMERCIAL_USE_ALLOWED, res.Msg.GetModel().GetCommercialUse(),
			"capability %s resolved a non-commercial model", capability)
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
	for _, capability := range modelcatalog.Capabilities {
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
