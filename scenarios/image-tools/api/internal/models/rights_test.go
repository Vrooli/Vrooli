package models

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/ai-provider-go/policy"
	"image-tools/internal/capabilities"
)

func TestSharedAdapterUsesCanonicalContextAndAllowsVerifiedOpenRAILLocalInference(t *testing.T) {
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
			if err != nil {
				t.Fatalf("evaluate seed candidate %s: %v", candidate.ID, err)
			}
			if candidate.ID == "sd-1.5" {
				if decision.Status != string(policy.RightsAllowed) || decision.Reason != "behavioral_use_restrictions_apply" || len(decision.Obligations) == 0 {
					t.Errorf("verified OpenRAIL model local-use rights = %+v, want allowed with source obligations", decision)
				}
			} else if candidate.ID == "sam-2.1-tiny" {
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
			if !candidate.Enabled || candidate.Backend == BackendOpenRouter || candidate.ID == "sd-1.5" || candidate.ID == "u2netp" {
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
