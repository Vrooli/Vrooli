//go:build audio_lab

package modelpromotion

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apiDatabase "github.com/vrooli/api-core/database"
	apiTestDB "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/scheduletest"

	"audio-tools/internal/ai/sttchain"
	intcorpus "audio-tools/internal/corpus"
	"audio-tools/internal/database"
	inteval "audio-tools/internal/eval"
	"audio-tools/internal/experiment"
	expreport "audio-tools/internal/experiment/report"
	"audio-tools/internal/modelcatalog"
	trustfloor "audio-tools/internal/qualification"
)

func TestExperimentResolverAcceptsOnlyOwnerProducedPassingQualityReceipt(t *testing.T) {
	ctx := context.Background()
	repo := testRepo(t)
	catalog := modelcatalog.Default()
	model, ok := catalog.ByID("whisper-small")
	require.True(t, ok)

	passed := persistQualityReceipt(t, ctx, repo, intcorpus.RequiredQualityClips)
	resolver := ExperimentResolver{Store: repo, Catalog: catalog}
	receipt, err := resolver.ResolvePromotionReceipt(ctx, FacetCorpusQuality, passed.ID, model)
	require.NoError(t, err)
	require.Equal(t, Receipt{
		Reference: passed.ID, Owner: "audio-tools.experiment.qualification_evidence",
		Facet: FacetCorpusQuality, ModelID: "whisper-small", Capability: modelcatalog.CapabilitySTT, Passed: true,
	}, receipt)

	failed := persistQualityReceipt(t, ctx, repo, 1)
	_, err = resolver.ResolvePromotionReceipt(ctx, FacetCorpusQuality, failed.ID, model)
	require.ErrorContains(t, err, "outcome is failed")
	_, err = resolver.ResolvePromotionReceipt(ctx, FacetCorpusQuality, "fabricated-ref", model)
	require.ErrorContains(t, err, "not found")

	wrongCapability := model
	wrongCapability.Capability = modelcatalog.CapabilityTTS
	_, err = resolver.ResolvePromotionReceipt(ctx, FacetCorpusQuality, passed.ID, wrongCapability)
	require.ErrorContains(t, err, "capability")
}

func TestExperimentResolverAcceptsOnlyMeasuredRealtimeLatencyReceipt(t *testing.T) {
	ctx := context.Background()
	repo := testRepo(t)
	catalog := modelcatalog.Default()
	model, ok := catalog.ByID("whisper-small")
	require.True(t, ok)
	resolver := ExperimentResolver{Store: repo, Catalog: catalog}

	good := persistLatencyReceipt(t, ctx, repo, model, 30, true)
	receipt, err := resolver.ResolvePromotionReceipt(ctx, FacetLatency, good.ID, model)
	require.NoError(t, err)
	require.Equal(t, Receipt{
		Reference: good.ID, Owner: "audio-tools.experiment.qualification_evidence", Facet: FacetLatency,
		ModelID: model.ID, Capability: model.Capability, Passed: true,
	}, receipt)

	tooFew := persistLatencyReceipt(t, ctx, repo, model, 29, true)
	_, err = resolver.ResolvePromotionReceipt(ctx, FacetLatency, tooFew.ID, model)
	require.ErrorContains(t, err, "identity-matched realtime interactive-latency")

	failed := persistLatencyReceipt(t, ctx, repo, model, 30, false)
	_, err = resolver.ResolvePromotionReceipt(ctx, FacetLatency, failed.ID, model)
	require.ErrorContains(t, err, "outcome is failed")
}

func TestExperimentResolverAcceptsOnlyCanonicalOwnerModelMemoryReceipt(t *testing.T) {
	ctx := context.Background()
	repo := testRepo(t)
	catalog := modelcatalog.Default()
	model, ok := catalog.ByID("sherpa-zipformer-en-2023-06-26")
	require.True(t, ok)
	receiptRow := persistMemoryReceipt(t, ctx, repo, model, nil)
	resolver := ExperimentResolver{Store: repo, Catalog: catalog}
	got, err := resolver.ResolvePromotionReceipt(ctx, FacetMemory, receiptRow.ID, model)
	require.NoError(t, err)
	require.Equal(t, Receipt{
		Reference: receiptRow.ID, Owner: "audio-tools.experiment.qualification_evidence", Facet: FacetMemory,
		ModelID: model.ID, Capability: model.Capability, Passed: true,
	}, got)
}

func TestExperimentResolverUsesCanonicalIDWhenRuntimeModelIDIsOmitted(t *testing.T) {
	ctx := context.Background()
	repo := testRepo(t)
	catalog := modelcatalog.Default()
	model, ok := catalog.ByID("whisper-small")
	require.True(t, ok)
	require.Empty(t, model.RuntimeModelID, "fixture exercises the registry's direct model-ID fallback")
	row := persistMemoryReceipt(t, ctx, repo, model, nil)
	resolver := ExperimentResolver{Store: repo, Catalog: catalog}
	got, err := resolver.ResolvePromotionReceipt(ctx, FacetMemory, row.ID, model)
	require.NoError(t, err)
	require.Equal(t, model.ID, got.ModelID)
	require.Equal(t, modelcatalog.CapabilitySTT, got.Capability)
}

func TestExperimentResolverRejectsInvalidModelMemoryEvidence(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*modelcatalog.Model, map[string]any, *experiment.QualificationEvidence)
		want   string
	}{
		{name: "missing peak", mutate: func(_ *modelcatalog.Model, machine map[string]any, _ *experiment.QualificationEvidence) {
			delete(machine["execution"].(map[string]any), "peak_memory_bytes")
		}, want: "positive budget and measured peak"},
		{name: "zero peak", mutate: func(_ *modelcatalog.Model, machine map[string]any, _ *experiment.QualificationEvidence) {
			machine["execution"].(map[string]any)["peak_memory_bytes"] = 0
		}, want: "positive budget and measured peak"},
		{name: "budget exceeded", mutate: func(_ *modelcatalog.Model, machine map[string]any, _ *experiment.QualificationEvidence) {
			machine["execution"].(map[string]any)["peak_memory_bytes"] = int64(3 * 1024 * 1024 * 1024)
		}, want: "in-budget measured peak"},
		{name: "wrong model", mutate: func(_ *modelcatalog.Model, machine map[string]any, _ *experiment.QualificationEvidence) {
			machine["provider_cell"].(map[string]any)["model_id"] = "other-model"
		}, want: "provider_cell identity"},
		{name: "wrong engine", mutate: func(_ *modelcatalog.Model, machine map[string]any, row *experiment.QualificationEvidence) {
			machine["provider_cell"].(map[string]any)["engine_id"] = "unknown-engine"
			row.EngineID = "unknown-engine"
		}, want: "not declared"},
		{name: "wrong runtime", mutate: func(_ *modelcatalog.Model, machine map[string]any, _ *experiment.QualificationEvidence) {
			machine["model"].(map[string]any)["runtime"] = "other-runtime"
		}, want: "runtime or artifact identity"},
		{name: "wrong runtime model id", mutate: func(_ *modelcatalog.Model, machine map[string]any, _ *experiment.QualificationEvidence) {
			machine["model"].(map[string]any)["runtime_model_id"] = "other-runtime-model"
		}, want: "runtime or artifact identity"},
		{name: "missing runtime model id", mutate: func(_ *modelcatalog.Model, machine map[string]any, _ *experiment.QualificationEvidence) {
			delete(machine["model"].(map[string]any), "runtime_model_id")
		}, want: "requires runtime, runtime_model_id"},
		{name: "wrong artifact", mutate: func(_ *modelcatalog.Model, machine map[string]any, _ *experiment.QualificationEvidence) {
			machine["model"].(map[string]any)["artifact_name"] = "other-artifact"
		}, want: "runtime or artifact identity"},
		{name: "wrong canonical ram budget", mutate: func(m *modelcatalog.Model, _ map[string]any, _ *experiment.QualificationEvidence) {
			m.Hardware.MinRAMGB++
		}, want: "requested model declaration does not match the canonical model catalog"},
		{name: "missing measurement integrity verdict", mutate: func(_ *modelcatalog.Model, machine map[string]any, _ *experiment.QualificationEvidence) {
			delete(machine["acceptance"].(map[string]any)["memory-budget"].(map[string]any), "measurement_integrity_passed")
		}, want: "passing integrity acceptance"},
		{name: "failed owner outcome", mutate: func(_ *modelcatalog.Model, _ map[string]any, row *experiment.QualificationEvidence) {
			row.Passed = false
		}, want: "outcome is failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			repo := testRepo(t)
			catalog := modelcatalog.Default()
			model, ok := catalog.ByID("sherpa-zipformer-en-2023-06-26")
			require.True(t, ok)
			var machine map[string]any
			require.NoError(t, json.Unmarshal([]byte(memoryMachineJSON(model)), &machine))
			row := experiment.QualificationEvidence{ID: "memory-test", EngineID: model.EngineIDs[0], ModelID: model.ID, Strategy: "streaming", Kind: "model_memory", Passed: true, ArtifactRef: "audio-tools:model-memory:test"}
			tc.mutate(&model, machine, &row)
			encoded, err := json.Marshal(machine)
			require.NoError(t, err)
			row.MachineJSON = encoded
			_, err = repo.CreateQualificationEvidence(ctx, row)
			require.NoError(t, err)
			resolver := ExperimentResolver{Store: repo, Catalog: catalog}
			_, err = resolver.ResolvePromotionReceipt(ctx, FacetMemory, row.ID, model)
			require.ErrorContains(t, err, tc.want)
		})
	}
}

func persistMemoryReceipt(t *testing.T, ctx context.Context, repo experiment.Repository, model modelcatalog.Model, mutate func(map[string]any)) experiment.QualificationEvidence {
	t.Helper()
	var machine map[string]any
	require.NoError(t, json.Unmarshal([]byte(memoryMachineJSON(model)), &machine))
	if mutate != nil {
		mutate(machine)
	}
	encoded, err := json.Marshal(machine)
	require.NoError(t, err)
	row, err := repo.CreateQualificationEvidence(ctx, experiment.QualificationEvidence{
		EngineID: model.EngineIDs[0], ModelID: model.ID, Strategy: "streaming", Kind: "model_memory",
		Passed: true, ArtifactRef: "audio-tools:model-memory:run-1", MachineJSON: encoded,
	})
	require.NoError(t, err)
	return row
}

func memoryMachineJSON(model modelcatalog.Model) string {
	runtimeModelID := model.RuntimeModelID
	if runtimeModelID == "" {
		runtimeModelID = model.ID
	}
	return fmt.Sprintf(`{"schema_version":1,"provider_cell":{"engine_id":%q,"model_id":%q,"capability":%q,"strategy":"streaming"},"model":{"runtime":%q,"runtime_model_id":%q,"artifact_resource":%q,"artifact_name":%q,"artifact_sha256":"%s","runtime_sha256":"%s"},"execution":{"lane":"cpu","os":"linux","arch":"amd64","accelerator":"none","memory_budget_bytes":%d,"peak_memory_bytes":123456789,"peak_memory_source":"cgroup-v2-memory.peak","inference_passed":true},"acceptance":{"memory-budget":{"in_band":true,"measured":true,"measurement_integrity_passed":true}}}`,
		model.EngineIDs[0], model.ID, model.Capability, model.Runtime, runtimeModelID, model.Artifact.Resource, model.Artifact.Name,
		strings.Repeat("a", 64), strings.Repeat("b", 64), int64(model.Hardware.MinRAMGB)*1024*1024*1024)
}

func persistLatencyReceipt(t *testing.T, ctx context.Context, repo experiment.Repository, model modelcatalog.Model, attempts int, claimPass bool) experiment.QualificationEvidence {
	t.Helper()
	payload := map[string]any{
		"lane": "realtime", "run_id": "soak-run-123", "browser_product_path": true,
		"provider_cell": map[string]string{"engine_id": model.EngineIDs[0], "model_id": model.ID, "strategy": "passthrough", "policy": "dictation"},
		"acceptance":    map[string]any{"interactive-latency": map[string]any{"in_band": claimPass, "negative_controls_passed": claimPass, "attempts": attempts}},
	}
	machine, err := json.Marshal(payload)
	require.NoError(t, err)
	evidence, err := repo.CreateQualificationEvidence(ctx, experiment.QualificationEvidence{
		EngineID: model.EngineIDs[0], ModelID: model.ID, Strategy: "passthrough", PolicyProfile: "dictation",
		Kind: "browser_product_path", Passed: claimPass, ArtifactRef: "bas:run-123", MachineJSON: machine,
	})
	require.NoError(t, err)
	return evidence
}

func TestPromotionVerificationReportsUnownedDimensionsAsBlockers(t *testing.T) {
	ctx := context.Background()
	repo := testRepo(t)
	catalog := modelcatalog.Default()
	model, ok := catalog.ByID("whisper-small")
	require.True(t, ok)
	receipt := persistQualityReceipt(t, ctx, repo, intcorpus.RequiredQualityClips)
	model.Promotion = modelcatalog.PromotionEvidence{
		CorpusQuality: modelcatalog.EvidenceFacet{Status: "qualified", Ref: receipt.ID},
		Latency:       modelcatalog.EvidenceFacet{Status: "qualified", Ref: "latency-not-real"},
		Memory:        modelcatalog.EvidenceFacet{Status: "qualified", Ref: "memory-not-real"},
		Platform:      modelcatalog.EvidenceFacet{Status: "qualified", Ref: "platform-not-real"},
		License:       modelcatalog.EvidenceFacet{Status: "qualified", Ref: "license-not-real"},
	}

	result := Verify(ctx, model, map[string]Resolver{
		FacetCorpusQuality: ExperimentResolver{Store: repo, Catalog: catalog},
	})
	require.False(t, result.Qualified)
	require.False(t, model.DefaultEligible(), "unowned facets keep a declaration ineligible for default selection")
	require.Equal(t, []string{FacetCorpusQuality}, result.VerifiedFacets)
	require.Len(t, result.Blockers, 4)
	require.False(t, result.VerifiedFor(model))
	require.Contains(t, result.Blockers[0], "latency has no integrated receipt owner")
	require.Contains(t, result.Blockers[1], "memory has no integrated receipt owner")
	require.Contains(t, result.Blockers[2], "platform has no integrated receipt owner")
	require.Contains(t, result.Blockers[3], "license has no integrated receipt owner")
}

func TestProductionReceiptOwnersMatchExistingDurableSources(t *testing.T) {
	for _, facet := range []string{FacetCorpusQuality, FacetLatency} {
		owner, ok := ownerForFacet(facet)
		require.True(t, ok, "expected the persisted experiment owner for %s", facet)
		require.Equal(t, "audio-tools.experiment.qualification_evidence", owner)
	}
	owner, ok := ownerForFacet(FacetMemory)
	require.True(t, ok)
	require.Equal(t, "audio-tools.experiment.qualification_evidence", owner)
	for _, facet := range []string{FacetPlatform, FacetLicense} {
		owner, ok := ownerForFacet(facet)
		require.False(t, ok, "%s has no model-promotion receipt owner", facet)
		require.Empty(t, owner)
	}
}

type passingResolver struct{}

func (passingResolver) ResolvePromotionReceipt(_ context.Context, facet, reference string, model modelcatalog.Model) (Receipt, error) {
	owner := "fixture." + facet
	return Receipt{Reference: reference, Owner: owner, Facet: facet, ModelID: model.ID, Capability: model.Capability, Passed: true}, nil
}

func TestVerifiedPromotionResultBridgesIntoCatalogSelection(t *testing.T) {
	// This declaration is intentionally independent of production model IDs,
	// providers, engines, and adapters. A newly declared model should qualify
	// through the same catalog and promotion joins without a model-specific
	// branch or a production catalog edit.
	model := modelcatalog.Model{
		ID: "fixture-new-stt-model", Name: "Synthetic streaming candidate",
		ProviderIDs: []string{"fixture-provider"}, EngineIDs: []string{"fixture-stream-engine"},
		Capability: modelcatalog.CapabilitySTT, Runtime: "fixture-runtime", RuntimeModelID: "fixture-runtime-model",
		Languages: []string{"en"}, Params: "1.7B", License: "fixture-license",
		Commercial: modelcatalog.CommercialYes, Readiness: modelcatalog.ReadinessCandidate,
		AllowDefault: true, Roles: []string{"default"}, Hardware: modelcatalog.Hardware{CPUCapable: true},
	}
	model.Promotion.CorpusQuality = modelcatalog.EvidenceFacet{Status: "qualified", Ref: "corpus-owner-ref"}
	model.Promotion.Latency = modelcatalog.EvidenceFacet{Status: "qualified", Ref: "latency-owner-ref"}
	model.Promotion.Memory = modelcatalog.EvidenceFacet{Status: "qualified", Ref: "memory-owner-ref"}
	model.Promotion.Platform = modelcatalog.EvidenceFacet{Status: "qualified", Ref: "platform-owner-ref"}
	model.Promotion.License = modelcatalog.EvidenceFacet{Status: "qualified", Ref: "license-owner-ref"}
	catalog := modelcatalog.New(modelcatalog.Seed{KnownProviderIDs: []string{"fixture-provider"}, Models: []modelcatalog.Model{model}})
	resolved, ok := catalog.ByID("fixture-new-stt-model")
	require.True(t, ok)
	require.Equal(t, "fixture-runtime-model", resolved.RuntimeModelID)
	require.Equal(t, "fixture-new-stt-model", catalog.ByProvider("fixture-provider")[0].ID)
	require.Equal(t, "fixture-new-stt-model", catalog.ByEngine("fixture-stream-engine")[0].ID)
	owners := map[string]string{
		FacetCorpusQuality: "fixture." + FacetCorpusQuality, FacetLatency: "fixture." + FacetLatency, FacetMemory: "fixture." + FacetMemory,
		FacetPlatform: "fixture.platform", FacetLicense: "fixture.license",
	}
	resolvers := map[string]Resolver{}
	for facet := range owners {
		resolvers[facet] = passingResolver{}
	}
	result := verifyWithOwners(context.Background(), resolved, resolvers, owners)
	require.True(t, result.Qualified)
	require.True(t, result.VerifiedFor(resolved))
	changedReceipt := resolved
	changedReceipt.Promotion.Latency.Ref = "different-receipt"
	require.False(t, result.VerifiedFor(changedReceipt))
	changedAdapter := resolved
	changedAdapter.EngineIDs = append([]string(nil), resolved.EngineIDs...)
	changedAdapter.EngineIDs[0] = "different-engine"
	require.False(t, result.VerifiedFor(changedAdapter))
	selected, err := catalog.DefaultForVerified(resolved.Capability, "en", modelcatalog.Platform{}, result.VerifiedFor)
	require.NoError(t, err)
	require.Equal(t, resolved.ID, selected.ID)

	fabricated := Result{
		ModelID: resolved.ID, Capability: resolved.Capability, Qualified: true,
		VerifiedFacets: append([]string(nil), requiredFacets...),
	}
	selected, err = catalog.DefaultForVerified(model.Capability, "en", modelcatalog.Platform{}, fabricated.VerifiedFor)
	require.ErrorIs(t, err, modelcatalog.ErrNoEligibleModel)
	require.Empty(t, selected.ID)
}

func testRepo(t *testing.T) experiment.Repository {
	t.Helper()
	db := apiTestDB.NewSQLite(t)
	require.NoError(t, apiDatabase.EnsureSchemas(context.Background(), db,
		apiDatabase.SchemaProviderFunc(database.SystemSchema),
		apiDatabase.SchemaProviderFunc(experiment.Schema),
		apiDatabase.SchemaProviderFunc(trustfloor.Schema),
	))
	return experiment.NewSQLiteRepository(db, scheduletest.New(time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)))
}

func persistQualityReceipt(t *testing.T, ctx context.Context, repo experiment.Repository, clipCount int) experiment.QualificationEvidence {
	t.Helper()
	clips := make([]intcorpus.Clip, 0, clipCount)
	results := make([]inteval.ClipResult, 0, clipCount)
	accents := []string{"uk", "us", "south_african"}
	for i := 0; i < clipCount; i++ {
		condition, snr := intcorpus.NoiseClean, float64(0)
		switch (i / len(accents)) % 3 {
		case 1:
			condition, snr = intcorpus.NoiseNoisy, 20
		case 2:
			condition, snr = intcorpus.NoiseNoisy, 10
		}
		text := strings.Repeat("one two three four five six seven eight nine ten ", 5)
		clips = append(clips, intcorpus.Clip{
			ID: fmt.Sprintf("clip-%d", i), ReferenceText: text, CorpusRevision: "voice-pilot-en-v1",
			LicenseReference: "license-ref", SpeakerPseudonym: fmt.Sprintf("speaker-%d", i%6),
			Split: intcorpus.SplitHeldOut, NoiseCondition: condition, SNRDB: snr,
			Tags:          []string{"accent_" + accents[i%len(accents)]},
			ContentSHA256: fmt.Sprintf("sha-%d", i),
			ReferenceReview: &intcorpus.ReferenceReview{
				Reviewer: "fixture-auditor", Protocol: "listen-and-correct-v1",
				ReviewedAt: "2026-06-29T12:00:00Z", ReferenceSHA256: intcorpus.ReferenceDigest(text),
			},
		})
		results = append(results, inteval.ClipResult{ClipID: fmt.Sprintf("clip-%d", i), Reference: text, Hypothesis: text, WER: inteval.WERResult{RefWords: 20}})
	}
	report := inteval.EvalReport{QualityMeasured: true, PerStrategy: []inteval.StrategyReport{{
		Strategy: sttchain.StrategyVADSegment, EngineID: "whisper-local", ModelID: "whisper-small",
		PolicyProfile: "default", ReplayLane: "batch", PerClip: results,
		Safety: inteval.SafetyGateReport{
			Passed: true, RetractionFree: true, DroppedSpanFree: true,
			DroppedSpanThresholdWords: inteval.DefaultDroppedSpanThresholdWords,
		},
	}}}
	evidence := expreport.QualityEvidenceForReport(clips, report, "test-build")
	require.Len(t, evidence, 1)
	if clipCount >= intcorpus.RequiredQualityClips {
		require.True(t, evidence[0].Passed, string(evidence[0].MachineJSON))
	} else {
		require.False(t, evidence[0].Passed)
	}
	var machine map[string]any
	require.NoError(t, json.Unmarshal(evidence[0].MachineJSON, &machine))
	returned, err := repo.CreateQualificationEvidence(ctx, evidence[0])
	require.NoError(t, err)
	require.NotEmpty(t, returned.ID)
	return returned
}
