package modelpromotion

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"audio-tools/internal/modelcatalog"
	trustfloor "audio-tools/internal/qualification"
)

type receiptResolverFunc func(context.Context, string, string, modelcatalog.Model) (Receipt, error)

func (f receiptResolverFunc) ResolvePromotionReceipt(ctx context.Context, facet, reference string, model modelcatalog.Model) (Receipt, error) {
	return f(ctx, facet, reference, model)
}

func fullyDeclaredModel() modelcatalog.Model {
	facet := func(ref string) modelcatalog.EvidenceFacet {
		return modelcatalog.EvidenceFacet{Status: "qualified", Ref: ref}
	}
	return modelcatalog.Model{ID: "product-test-model", Capability: modelcatalog.CapabilitySTT, Promotion: modelcatalog.PromotionEvidence{CorpusQuality: facet("quality"), Latency: facet("latency"), Memory: facet("memory"), Platform: facet("platform"), License: facet("license")}}
}

func TestVerifyFailsClosedForMissingProductOwners(t *testing.T) {
	model := fullyDeclaredModel()
	resolver := receiptResolverFunc(func(_ context.Context, facet, ref string, m modelcatalog.Model) (Receipt, error) {
		owner, ok := ownerForFacet(facet)
		if !ok {
			return Receipt{}, errors.New("unexpected unowned facet")
		}
		return Receipt{Reference: ref, Owner: owner, Facet: facet, ModelID: m.ID, Capability: m.Capability, Passed: true}, nil
	})
	result := Verify(context.Background(), model, map[string]Resolver{FacetCorpusQuality: resolver, FacetLatency: resolver, FacetMemory: resolver})
	if result.Qualified || result.VerifiedFor(model) {
		t.Fatal("Verify accepted facets without integrated product owners")
	}
	if len(result.Blockers) != 2 {
		t.Fatalf("expected platform and license blockers, got %v", result.Blockers)
	}
}

func TestVerifiedForRequiresVerifierProofAndExactDeclaration(t *testing.T) {
	model := fullyDeclaredModel()
	resolver := receiptResolverFunc(func(_ context.Context, facet, ref string, m modelcatalog.Model) (Receipt, error) {
		return Receipt{Reference: ref, Owner: "test-owner", Facet: facet, ModelID: m.ID, Capability: m.Capability, Passed: true}, nil
	})
	owners, resolvers := map[string]string{}, map[string]Resolver{}
	for _, facet := range requiredFacets {
		owners[facet] = "test-owner"
		resolvers[facet] = resolver
	}
	result := verifyWithOwners(context.Background(), model, resolvers, owners)
	if !result.VerifiedFor(model) {
		t.Fatal("expected fully verified declaration to be accepted")
	}
	changed := model
	changed.EngineIDs = []string{"different-engine"}
	if result.VerifiedFor(changed) {
		t.Fatal("accepted declaration whose identity facts changed")
	}
	fabricated := Result{ModelID: model.ID, Capability: model.Capability, Qualified: true, VerifiedFacets: append([]string(nil), requiredFacets...)}
	if fabricated.VerifiedFor(model) {
		t.Fatal("accepted caller-assembled result without Verify proof")
	}
}

type evidenceStoreFunc func(context.Context, trustfloor.QualificationEvidenceFilter) ([]trustfloor.QualificationEvidence, error)

func (f evidenceStoreFunc) ListQualificationEvidence(ctx context.Context, filter trustfloor.QualificationEvidenceFilter) ([]trustfloor.QualificationEvidence, error) {
	return f(ctx, filter)
}

func TestExperimentResolverBindsReceiptToRequestedAndCanonicalIdentity(t *testing.T) {
	catalog := modelcatalog.Default()
	model, ok := catalog.ByID("whisper-small")
	if !ok {
		t.Fatal("default catalog lacks whisper-small")
	}
	store := evidenceStoreFunc(func(_ context.Context, filter trustfloor.QualificationEvidenceFilter) ([]trustfloor.QualificationEvidence, error) {
		if filter.ModelID != model.ID {
			t.Fatalf("filter model=%q, want %q", filter.ModelID, model.ID)
		}
		return []trustfloor.QualificationEvidence{{ID: "receipt-1", ModelID: model.ID, Kind: "wrong-kind", Passed: true}}, nil
	})
	resolver := ExperimentResolver{Store: store, Catalog: catalog}
	if _, err := resolver.ResolvePromotionReceipt(context.Background(), FacetCorpusQuality, "receipt-1", model); err == nil {
		t.Fatal("accepted receipt with mismatched kind")
	}
	if _, err := resolver.ResolvePromotionReceipt(context.Background(), FacetPlatform, "receipt-1", model); err == nil {
		t.Fatal("accepted facet outside experiment owner's scope")
	}
	if _, err := resolver.ResolvePromotionReceipt(context.Background(), FacetCorpusQuality, "missing", model); err == nil {
		t.Fatal("accepted missing receipt")
	}
}

func TestExperimentResolverPropagatesStoreFailure(t *testing.T) {
	catalog := modelcatalog.Default()
	model, ok := catalog.ByID("whisper-small")
	if !ok {
		t.Fatal("default catalog lacks whisper-small")
	}
	want := errors.New("store unavailable")
	resolver := ExperimentResolver{Store: evidenceStoreFunc(func(context.Context, trustfloor.QualificationEvidenceFilter) ([]trustfloor.QualificationEvidence, error) {
		return nil, want
	}), Catalog: catalog}
	if _, err := resolver.ResolvePromotionReceipt(context.Background(), FacetMemory, "receipt", model); !errors.Is(err, want) {
		t.Fatalf("error=%v, want %v", err, want)
	}
}

func TestExperimentResolverAcceptsValidQualityLatencyAndMemoryReceipts(t *testing.T) {
	catalog := modelcatalog.Default()
	model, ok := catalog.ByID("whisper-small")
	if !ok {
		t.Fatal("default catalog lacks whisper-small")
	}
	cell := map[string]any{
		"engine_id": "whisper-local", "model_id": model.ID,
		"capability": string(model.Capability), "strategy": "passthrough",
	}
	qualityPayload := map[string]any{
		"provider_cell":       cell,
		"acceptance":          map[string]any{"corpus-quality": map[string]any{"in_band": true, "negative_controls_passed": true}},
		"quality_measurement": map[string]any{"passed": true, "provider_cell": cell},
	}
	qualityJSON, err := json.Marshal(qualityPayload)
	if err != nil {
		t.Fatal(err)
	}
	latencyPayload := map[string]any{
		"lane": "realtime", "run_id": "browser-run-1", "browser_product_path": true,
		"acceptance": map[string]any{"interactive-latency": map[string]any{
			"in_band": true, "negative_controls_passed": true, "attempts": 30,
		}},
	}
	latencyJSON, err := json.Marshal(latencyPayload)
	if err != nil {
		t.Fatal(err)
	}
	budget := int64(model.Hardware.MinRAMGB) * 1024 * 1024 * 1024
	runtimeModelID := model.RuntimeModelID
	if runtimeModelID == "" {
		runtimeModelID = model.ID
	}
	memoryPayload := map[string]any{
		"schema_version": 1,
		"provider_cell": map[string]any{
			"engine_id": "whisper-local", "model_id": model.ID,
			"capability": string(model.Capability), "strategy": "passthrough",
		},
		"model": map[string]any{
			"runtime": model.Runtime, "runtime_model_id": runtimeModelID,
			"artifact_resource": model.Artifact.Resource, "artifact_name": model.Artifact.Name,
			"artifact_sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"runtime_sha256":  "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		},
		"execution": map[string]any{
			"lane": "cpu", "os": "linux", "arch": "amd64", "accelerator": "none",
			"memory_budget_bytes": budget, "peak_memory_bytes": budget - 1,
			"peak_memory_source": "cgroup-v2-memory.peak", "inference_passed": true,
		},
		"acceptance": map[string]any{"memory-budget": map[string]any{
			"in_band": true, "measured": true, "measurement_integrity_passed": true,
		}},
	}
	memoryJSON, err := json.Marshal(memoryPayload)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		facet, kind, reference string
		machineJSON            []byte
	}{
		{FacetCorpusQuality, "corpus_quality", "quality-1", qualityJSON},
		{FacetLatency, "browser_product_path", "latency-1", latencyJSON},
		{FacetMemory, trustfloor.QualificationModelMemory, "memory-1", memoryJSON},
	} {
		t.Run(tc.facet, func(t *testing.T) {
			item := trustfloor.QualificationEvidence{
				ID: tc.reference, EngineID: "whisper-local", ModelID: model.ID,
				Strategy: "passthrough", PolicyProfile: "dictation", Kind: tc.kind,
				Passed: true, ArtifactRef: "bas:run-1", MachineJSON: tc.machineJSON,
			}
			resolver := ExperimentResolver{Store: evidenceStoreFunc(func(context.Context, trustfloor.QualificationEvidenceFilter) ([]trustfloor.QualificationEvidence, error) {
				return []trustfloor.QualificationEvidence{item}, nil
			}), Catalog: catalog}
			got, err := resolver.ResolvePromotionReceipt(context.Background(), tc.facet, tc.reference, model)
			if err != nil {
				t.Fatalf("ResolvePromotionReceipt() error = %v", err)
			}
			if !got.Passed || got.Reference != tc.reference || got.Facet != tc.facet || got.ModelID != model.ID || got.Capability != model.Capability {
				t.Fatalf("receipt identity = %#v", got)
			}
		})
	}
}

func TestExperimentResolverRejectsInvalidOwnerMachineClaims(t *testing.T) {
	catalog := modelcatalog.Default()
	model, ok := catalog.ByID("whisper-small")
	if !ok {
		t.Fatal("default catalog lacks whisper-small")
	}
	validLatency := []byte(`{"lane":"realtime","run_id":"run-1","browser_product_path":true,"acceptance":{"interactive-latency":{"in_band":true,"negative_controls_passed":true,"attempts":29}}}`)
	validQuality := []byte(`{"provider_cell":{"engine_id":"whisper-local","model_id":"whisper-small","capability":"stt","strategy":"passthrough"},"acceptance":{"corpus-quality":{"in_band":true,"negative_controls_passed":true}},"quality_measurement":{"passed":true,"provider_cell":{"engine_id":"whisper-local","model_id":"whisper-small","capability":"stt","strategy":"passthrough"}}}`)
	for _, tc := range []struct {
		name, facet, kind string
		passed            bool
		machineJSON       []byte
	}{
		{"short latency cohort", FacetLatency, "browser_product_path", true, validLatency},
		{"failed owner outcome", FacetCorpusQuality, "corpus_quality", false, validQuality},
		{"malformed quality JSON", FacetCorpusQuality, "corpus_quality", true, []byte(`{`)},
		{"quality identity mismatch", FacetCorpusQuality, "corpus_quality", true, []byte(`{"provider_cell":{"engine_id":"other","model_id":"whisper-small","capability":"stt","strategy":"passthrough"},"quality_measurement":{"passed":true,"provider_cell":{"engine_id":"whisper-local","model_id":"whisper-small","capability":"stt","strategy":"passthrough"}},"acceptance":{"corpus-quality":{"in_band":true,"negative_controls_passed":true}}}`)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			item := trustfloor.QualificationEvidence{
				ID: "receipt", EngineID: "whisper-local", ModelID: model.ID,
				Strategy: "passthrough", PolicyProfile: "dictation", Kind: tc.kind,
				Passed: tc.passed, ArtifactRef: "browser:run-1", MachineJSON: tc.machineJSON,
			}
			resolver := ExperimentResolver{Store: evidenceStoreFunc(func(context.Context, trustfloor.QualificationEvidenceFilter) ([]trustfloor.QualificationEvidence, error) {
				return []trustfloor.QualificationEvidence{item}, nil
			}), Catalog: catalog}
			if _, err := resolver.ResolvePromotionReceipt(context.Background(), tc.facet, "receipt", model); err == nil {
				t.Fatal("ResolvePromotionReceipt() accepted an invalid owner claim")
			}
		})
	}
}

func TestExperimentResolverRejectsMemoryModelIdentityAndBudgetDrift(t *testing.T) {
	catalog := modelcatalog.Default()
	model, ok := catalog.ByID("whisper-small")
	if !ok {
		t.Fatal("default catalog lacks whisper-small")
	}
	budget := int64(model.Hardware.MinRAMGB) * 1024 * 1024 * 1024
	for _, tc := range []struct {
		name         string
		artifactName string
		memoryBudget int64
	}{
		{"wrong artifact", "other-model", budget},
		{"wrong memory budget", model.Artifact.Name, budget - 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runtimeModelID := model.RuntimeModelID
			if runtimeModelID == "" {
				runtimeModelID = model.ID
			}
			payload := map[string]any{
				"schema_version": 1,
				"provider_cell":  map[string]any{"engine_id": "whisper-local", "model_id": model.ID, "capability": "stt", "strategy": "passthrough"},
				"model": map[string]any{
					"runtime": model.Runtime, "runtime_model_id": runtimeModelID,
					"artifact_resource": model.Artifact.Resource, "artifact_name": tc.artifactName,
					"artifact_sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					"runtime_sha256":  "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				},
				"execution":  map[string]any{"lane": "cpu", "os": "linux", "arch": "amd64", "accelerator": "none", "memory_budget_bytes": tc.memoryBudget, "peak_memory_bytes": 1, "peak_memory_source": "cgroup-v2-memory.peak", "inference_passed": true},
				"acceptance": map[string]any{"memory-budget": map[string]any{"in_band": true, "measured": true, "measurement_integrity_passed": true}},
			}
			machineJSON, err := json.Marshal(payload)
			if err != nil {
				t.Fatal(err)
			}
			item := trustfloor.QualificationEvidence{ID: "memory", EngineID: "whisper-local", ModelID: model.ID, Strategy: "passthrough", Kind: trustfloor.QualificationModelMemory, Passed: true, MachineJSON: machineJSON}
			resolver := ExperimentResolver{Store: evidenceStoreFunc(func(context.Context, trustfloor.QualificationEvidenceFilter) ([]trustfloor.QualificationEvidence, error) {
				return []trustfloor.QualificationEvidence{item}, nil
			}), Catalog: catalog}
			if _, err := resolver.ResolvePromotionReceipt(context.Background(), FacetMemory, item.ID, model); err == nil {
				t.Fatal("ResolvePromotionReceipt() accepted memory identity or budget drift")
			}
		})
	}
}
