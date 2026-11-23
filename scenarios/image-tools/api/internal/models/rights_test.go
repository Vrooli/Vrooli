package models

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/ai-provider-go/policy"
	"image-tools/internal/capabilities"
)

// strictRightsContextPath copies the canonical operator context without the
// standing normal-use licensing answer, so the fail-closed evidence-gap
// behavior stays proven independently of the operator's answer.
func strictRightsContextPath(t *testing.T, repoRoot string) string {
	t.Helper()
	canonical := filepath.Join(repoRoot, "scenarios", "ai-gateway", "config", "model-rights-context.json")
	data, err := os.ReadFile(canonical)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	contextFields, _ := document["context"].(map[string]any)
	delete(contextFields, "normal_use_licensing")
	delete(contextFields, "normal_use_licensing_evidence")
	out, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "strict-model-rights-context.json")
	if err := os.WriteFile(path, out, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCanonicalOperatorContextAllowsUnverifiedSeedModelsForNormalUseOnly(t *testing.T) {
	repoRoot := repositoryRoot(t)
	t.Setenv("VROOLI_SOURCE_ROOT", repoRoot)
	t.Setenv(policy.RightsContextPathEnv, "")
	evaluator := AdaptRightsEvaluator(policy.NewDefaultRightsEvaluator(os.Getenv))
	registry, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	candidate, ok := registry.ByID("depth-anything-v2-small")
	if !ok {
		t.Fatal("depth-anything-v2-small missing from seed")
	}
	decision, err := EvaluateCurrentModelRights(context.Background(), evaluator, candidate, RightsUseLocalInference)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Status != string(policy.RightsAllowed) || decision.Reason != "operator_normal_use_any_license_class" || len(decision.Obligations) == 0 {
		t.Fatalf("canonical operator context local-use rights = %+v, want operator normal-use allow with obligations", decision)
	}
	for _, use := range []string{RightsUseMonetizedUse, RightsUseHostedInference, RightsUseRedistributeWeights} {
		if leaked, _ := EvaluateCurrentModelRights(context.Background(), evaluator, candidate, use); leaked.Status == string(policy.RightsAllowed) {
			t.Fatalf("operator normal-use answer leaked to %s: %+v", use, leaked)
		}
	}
}

func TestSharedAdapterUsesCanonicalContextAndAllowsVerifiedOpenRAILLocalInference(t *testing.T) {
	repoRoot := repositoryRoot(t)
	t.Setenv("VROOLI_SOURCE_ROOT", repoRoot)
	t.Setenv(policy.RightsContextPathEnv, strictRightsContextPath(t, repoRoot))
	evaluator := AdaptRightsEvaluator(policy.NewDefaultRightsEvaluator(os.Getenv))
	registry, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, operation := range registry.SortedOperations() {
		for _, candidate := range registry.ForOperation(operation) {
			if !candidate.Enabled || candidate.Backend == BackendOpenRouter {
				continue
			}
			decision, err := EvaluateCurrentModelRights(context.Background(), evaluator, candidate, RightsUseLocalInference)
			if err != nil {
				t.Fatalf("evaluate seed candidate %s: %v", candidate.ID, err)
			}
			if candidate.ID == "sd-1.5" {
				if decision.Status != string(policy.RightsAllowed) || decision.Reason != "behavioral_use_restrictions_apply" || len(decision.Obligations) == 0 {
					t.Errorf("verified OpenRAIL model local-use rights = %+v, want allowed with source obligations", decision)
				}
			} else if candidate.ID == "sam-2.1-tiny" || candidate.ID == "sam-2.1-base-plus" {
				if decision.Status != string(policy.RightsAllowed) || decision.Reason != "verified_unrestricted_grant" {
					t.Errorf("verified SAM 2.1 checkpoint local-use rights = %+v, want allowed from pinned Apache-2.0 evidence", decision)
				}
			} else if candidate.ID == "u2netp" {
				if decision.Status != string(policy.RightsAllowed) || decision.Reason != "verified_unrestricted_grant" {
					t.Errorf("verified U2-Net-p ONNX local-use rights = %+v, want allowed from pinned Apache-2.0 evidence", decision)
				}
				var artifact policy.ArtifactRights
				if err := json.Unmarshal(candidate.Rights, &artifact); err != nil {
					t.Fatalf("decode U2-Net-p rights evidence: %v", err)
				}
				if candidate.Source.Checksum.Status != "pinned" || candidate.Source.Checksum.Value != artifact.SHA256 {
					t.Errorf("U2-Net-p artifact checksum %q/%s does not match rights digest %s", candidate.Source.Checksum.Value, candidate.Source.Checksum.Status, artifact.SHA256)
				}
			} else if candidate.ID == "flux-2-klein-4b" {
				if decision.Status != string(policy.RightsAllowed) || decision.Reason != "verified_unrestricted_grant" {
					t.Errorf("verified FLUX.2 klein 4B local-use rights = %+v, want allowed from pinned Apache-2.0 evidence", decision)
				}
				var artifact policy.ArtifactRights
				if err := json.Unmarshal(candidate.Rights, &artifact); err != nil {
					t.Fatalf("decode FLUX.2 klein rights evidence: %v", err)
				}
				if candidate.Source.Repo.FileSHA256["transformer/diffusion_pytorch_model.safetensors"] != artifact.SHA256 {
					t.Errorf("FLUX.2 klein rights digest %s is not the pinned transformer weights digest", artifact.SHA256)
				}
			} else if candidate.CapabilityLabels.CommercialUse == CommercialUseNo {
				if decision.Status != string(policy.RightsDenied) || decision.Reason != "license_prohibits_use" {
					t.Errorf("non-commercial %s rights without the operator's normal-use answer = %+v, want denied by its license", candidate.ID, decision)
				}
			} else if decision.Status != string(policy.RightsUnknown) {
				t.Errorf("unverified seed candidate %s rights = %q (%s), want unknown from source facts", candidate.ID, decision.Status, decision.Reason)
			}
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("no enabled local seed candidates checked")
	}
	for _, operation := range registry.SortedOperations() {
		for _, candidate := range registry.ForOperation(operation) {
			if !candidate.Enabled || candidate.Backend == BackendOpenRouter || candidate.ID == "sd-1.5" || candidate.ID == "u2netp" || candidate.ID == "flux-2-klein-4b" {
				continue
			}
			_, err := registry.Select(SelectRequest{Operation: operation, OverrideID: candidate.ID, Host: capabilities.Host{OS: "linux", Arch: "amd64", Cores: 8}, RightsEvaluator: evaluator}, nil)
			if err == nil || !strings.Contains(err.Error(), "rights") {
				t.Fatalf("real seed selection for unverified model %s/%s = %v, want fail-closed denial", operation, candidate.ID, err)
			}
			return
		}
	}
	t.Fatal("no enabled local operation candidate found for selection refusal proof")
}

func TestSAM21BasePlusRightsUsePinnedCheckpointAndPublisherLicense(t *testing.T) {
	repoRoot := repositoryRoot(t)
	t.Setenv("VROOLI_SOURCE_ROOT", repoRoot)
	t.Setenv(policy.RightsContextPathEnv, "")
	evaluator := AdaptRightsEvaluator(policy.NewDefaultRightsEvaluator(os.Getenv))
	registry, err := Load()
	if err != nil {
		t.Fatalf("load seed: %v", err)
	}
	model, ok := registry.ByID("sam-2.1-base-plus")
	if !ok {
		t.Fatal("SAM 2.1 Base+ missing from seed")
	}
	decision, err := EvaluateCurrentModelRights(context.Background(), evaluator, model, RightsUseLocalInference)
	if err != nil || decision.Status != string(policy.RightsAllowed) || decision.Reason != "verified_unrestricted_grant" {
		t.Fatalf("Base+ rights = %+v, %v; want allowed from pinned publisher terms", decision, err)
	}
	var artifact policy.ArtifactRights
	if err := json.Unmarshal(model.Rights, &artifact); err != nil {
		t.Fatalf("decode Base+ rights: %v", err)
	}
	var checkpointSHA string
	for _, asset := range model.Source.Assets {
		if asset.Filename == "model.safetensors" {
			checkpointSHA = asset.SHA256
			if !strings.Contains(asset.URL, "/resolve/"+artifact.Revision+"/") {
				t.Fatalf("Base+ rights revision %s does not match checkpoint URL %s", artifact.Revision, asset.URL)
			}
		}
	}
	if checkpointSHA == "" || artifact.SHA256 != checkpointSHA || artifact.Revision == "" {
		t.Fatalf("Base+ rights revision/digest do not match pinned checkpoint asset: rights=%+v assets=%+v", artifact, model.Source.Assets)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	marker := filepath.Join("scenarios", "ai-gateway", "config", "model-rights-context.json")
	starts := []string{os.Getenv("VROOLI_SOURCE_ROOT")}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get test working directory: %v", err)
	}
	starts = append(starts, cwd)
	for _, start := range starts {
		if start == "" {
			continue
		}
		current := filepath.Clean(start)
		for {
			if info, statErr := os.Stat(filepath.Join(current, marker)); statErr == nil && !info.IsDir() {
				return current
			}
			parent := filepath.Dir(current)
			if parent == current {
				break
			}
			current = parent
		}
	}
	t.Fatalf("locate repository root containing %s", marker)
	return ""
}

// This synthetic selector fixture exercises a source-backed conditional grant;
// no image seed grant is changed or inferred. The unrestricted fallback and
// conditional preferred candidate both pass through the production adapter.
func TestSharedAdapterRefusesOnContextFlipAndRestoresPermittedCandidate(t *testing.T) {
	contextPath := filepath.Join(t.TempDir(), "rights-context.json")
	writeContext := func(version, revenueStatus string) {
		t.Helper()
		doc := map[string]any{"schema_version": "1", "context": map[string]any{
			"id": "test-operator", "version": version, "entity_scope": "operator-group",
			"revenue_status": revenueStatus, "as_of": "2026-11-14", "source": "operator_statement",
		}}
		data, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(contextPath, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv(policy.RightsContextPathEnv, contextPath)
	writeContext("below", "pre_revenue")
	evaluator := AdaptRightsEvaluator(policy.NewDefaultRightsEvaluator(os.Getenv))
	registry := testRegistry(t)
	conditional, _ := json.Marshal(policy.ArtifactRights{
		ArtifactID: "fixture/gpu-quality", Revision: "r1", SHA256: strings.Repeat("a", 64),
		LicenseID: "fixture-conditional", Classification: policy.RightsConditional,
		Evidence:          policy.RightsEvidence{URL: "https://example.invalid/license", Revision: "r1", SHA256: strings.Repeat("b", 64), ObservedAt: "2026-11-14"},
		ConditionalGrants: []policy.ConditionalGrant{{ID: "revenue-cap", Uses: []policy.RightsUse{policy.RightsUseLocalInference}, Conditions: []policy.RightsCondition{{ID: "under-cap", Kind: "annual_revenue", Comparator: "lt", Threshold: 10_000_000, Unit: "USD", EntityScope: "operator-group", Interpretation: "fixture condition"}}}},
	})
	unrestricted, _ := json.Marshal(policy.ArtifactRights{
		ArtifactID: "fixture/cpu-default", Revision: "r1", SHA256: strings.Repeat("c", 64),
		LicenseID: "fixture-unrestricted", Classification: policy.RightsUnrestricted,
		Evidence: policy.RightsEvidence{URL: "https://example.invalid/license", Revision: "r1", SHA256: strings.Repeat("d", 64), ObservedAt: "2026-11-14"},
	})
	for i := range registry.models {
		switch registry.models[i].ID {
		case "gpu-quality":
			registry.models[i].Rights = conditional
		case "cpu-default":
			registry.models[i].Rights = unrestricted
		}
		registry.byID[registry.models[i].ID] = registry.models[i]
	}
	for i := range registry.byOperation["upscale"] {
		registry.byOperation["upscale"][i] = registry.byID[registry.byOperation["upscale"][i].ID]
	}
	request := SelectRequest{Operation: "upscale", Host: gpuHost(12), RightsEvaluator: evaluator}
	selected, err := registry.Select(request, func(string) bool { return true })
	if err != nil || selected.Model.ID != "gpu-quality" || selected.Rights.Status != "allowed" {
		t.Fatalf("below-threshold candidate = %+v, %v; want permitted preferred candidate", selected, err)
	}
	writeContext("above", "at_or_above_threshold")
	selected, err = registry.Select(request, func(string) bool { return true })
	if err != nil || selected.Model.ID != "cpu-default" {
		t.Fatalf("context-flipped fallback = %+v, %v; want unrestricted fallback", selected, err)
	}
	decision, err := EvaluateCurrentModelRights(context.Background(), evaluator, registry.byID["gpu-quality"], RightsUseLocalInference)
	if err != nil || decision.Status != "denied" {
		t.Fatalf("flipped preferred candidate rights = %+v, %v; want denied", decision, err)
	}
	writeContext("restored", "pre_revenue")
	selected, err = registry.Select(request, func(string) bool { return true })
	if err != nil || selected.Model.ID != "gpu-quality" {
		t.Fatalf("restored selection = %+v, %v; want source-permitted preferred candidate", selected, err)
	}
}

// nonCommercialModel is a registry-shaped model whose license forbids
// commercial use: the rights record is classified prohibited, as the seed
// requires for every commercial_use=no entry.
func nonCommercialModel() Model {
	rights, _ := json.Marshal(policy.ArtifactRights{
		ArtifactID: "nc-model", Revision: "weights-rev-1", SHA256: strings.Repeat("a", 64),
		LicenseID: "LicenseRef-non-commercial", Classification: policy.RightsProhibited,
		Evidence:    policy.RightsEvidence{URL: "https://example.invalid/LICENSE", Revision: "license-rev-1", SHA256: strings.Repeat("b", 64), ObservedAt: "2026-11-21"},
		Obligations: []string{"non-commercial use only"}, ReviewTriggers: []string{"license_source_changed"},
	})
	return Model{ID: "nc-model", Rights: rights, CapabilityLabels: CapabilityLabels{CommercialUse: CommercialUseNo, CommercialUseNotes: "non-commercial license"}}
}

func TestNonCommercialModelIsAllowedForNormalUseButNotMonetizationOrRedistribution(t *testing.T) {
	repoRoot := repositoryRoot(t)
	t.Setenv("VROOLI_SOURCE_ROOT", repoRoot)
	t.Setenv(policy.RightsContextPathEnv, "")
	canonical := AdaptRightsEvaluator(policy.NewDefaultRightsEvaluator(os.Getenv))
	model := nonCommercialModel()
	if err := RequireLocalUsePermitted(context.Background(), canonical, model); err != nil {
		t.Fatalf("non-commercial model refused for normal use under the operator stance: %v", err)
	}
	for _, use := range []string{RightsUseMonetizedUse, RightsUseHostedInference, RightsUseRedistributeWeights} {
		if err := RequireUsePermitted(context.Background(), canonical, model, use); !errors.Is(err, ErrLocalUseNotPermitted) {
			t.Fatalf("non-commercial model use %s = %v, want refusal", use, err)
		}
	}
	// Without the standing answer even normal use stays license-bound.
	t.Setenv(policy.RightsContextPathEnv, strictRightsContextPath(t, repoRoot))
	strict := AdaptRightsEvaluator(policy.NewDefaultRightsEvaluator(os.Getenv))
	if err := RequireLocalUsePermitted(context.Background(), strict, model); !errors.Is(err, ErrLocalUseNotPermitted) {
		t.Fatalf("non-commercial model without the standing answer = %v, want refusal", err)
	}
	if err := RequireLocalUsePermitted(context.Background(), nil, model); !errors.Is(err, ErrLocalUseNotPermitted) {
		t.Fatalf("no rights evaluator = %v, want fail-closed refusal", err)
	}
}

func TestSelectionAllowsNonCommercialModelForNormalUse(t *testing.T) {
	repoRoot := repositoryRoot(t)
	t.Setenv("VROOLI_SOURCE_ROOT", repoRoot)
	t.Setenv(policy.RightsContextPathEnv, "")
	evaluator := AdaptRightsEvaluator(policy.NewDefaultRightsEvaluator(os.Getenv))
	model := nonCommercialModel()
	if !EffectiveEnabled(model, nil) && model.Enabled {
		t.Fatal("non-commercial label masked an enabled model")
	}
	decision, err := EvaluateCurrentModelRights(context.Background(), evaluator, model, RightsUseLocalInference)
	if err != nil || decision.Status != string(policy.RightsAllowed) || decision.Reason != "operator_normal_use_any_license_class" {
		t.Fatalf("selection-time rights for a non-commercial model = %+v, %v", decision, err)
	}
}

// Under the operator's standing normal-use answer every enabled local seed model
// is allowed for local inference whatever its license class, and none of them is
// allowed for monetized use, hosted inference or redistribution unless its own
// verified license says so.
func TestCanonicalContextAllowsEveryEnabledLocalSeedModelForNormalUse(t *testing.T) {
	repoRoot := repositoryRoot(t)
	t.Setenv("VROOLI_SOURCE_ROOT", repoRoot)
	t.Setenv(policy.RightsContextPathEnv, "")
	evaluator := AdaptRightsEvaluator(policy.NewDefaultRightsEvaluator(os.Getenv))
	registry, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, operation := range registry.SortedOperations() {
		for _, candidate := range registry.ForOperation(operation) {
			if !candidate.Enabled || candidate.Backend == BackendOpenRouter {
				continue
			}
			decision, err := EvaluateCurrentModelRights(context.Background(), evaluator, candidate, RightsUseLocalInference)
			if err != nil || decision.Status != string(policy.RightsAllowed) {
				t.Errorf("%s/%s normal-use rights = %+v, %v; want allowed", operation, candidate.ID, decision, err)
			}
			if candidate.CapabilityLabels.CommercialUse != CommercialUseNo {
				checked++
				continue
			}
			for _, use := range []string{RightsUseMonetizedUse, RightsUseHostedInference, RightsUseRedistributeWeights} {
				if leaked, _ := EvaluateCurrentModelRights(context.Background(), evaluator, candidate, use); leaked.Status == string(policy.RightsAllowed) {
					t.Errorf("non-commercial %s allowed for %s: %+v", candidate.ID, use, leaked)
				}
			}
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("no enabled local seed candidates checked")
	}
}
