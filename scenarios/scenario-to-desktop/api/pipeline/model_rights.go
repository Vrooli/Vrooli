package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	sharedpolicy "github.com/vrooli/ai-provider-go/policy"
)

type ModelRightsUse string

const (
	ModelRightsUseRedistributeWeights ModelRightsUse = "redistribute_weights"
	ModelRightsUseDistributeRuntime   ModelRightsUse = "distribute_runtime"
)

// ModelRightsEvaluationRequest is the typed owner-boundary input to the shared
// rights evaluator adapter. Resource manifests retain the source facts as JSON;
// the adapter decodes them into the shared policy type before evaluation.
type ModelRightsEvaluationRequest struct {
	ModelID       string          `json:"model_id"`
	Resource      string          `json:"resource"`
	ArtifactKind  string          `json:"artifact_kind"`
	Use           ModelRightsUse  `json:"use"`
	ArtifactFacts json.RawMessage `json:"artifact_rights"`
}

type ModelRightsDecision struct {
	Status         string `json:"status"`
	Reason         string `json:"reason"`
	DecisionSHA256 string `json:"decision_sha256,omitempty"`
}

// ModelRightsEvaluator is implemented by the shared ai-provider-go policy
// adapter. A missing evaluator is fail-closed whenever a resource declares a
// model artifact for desktop bundling.
type ModelRightsEvaluator interface {
	EvaluateModelRights(context.Context, ModelRightsEvaluationRequest) (ModelRightsDecision, error)
}

// NewSharedModelRightsEvaluator adapts the shared current-context evaluator to
// the desktop deployment-plan boundary. It reads the trusted operator context
// for each artifact evaluation; package callers cannot supply that context.
func NewSharedModelRightsEvaluator() ModelRightsEvaluator {
	return &sharedModelRightsEvaluator{evaluator: sharedpolicy.NewDefaultRightsEvaluator(nil)}
}

type sharedModelRightsEvaluator struct {
	evaluator sharedpolicy.RightsEvaluator
}

func (e *sharedModelRightsEvaluator) EvaluateModelRights(ctx context.Context, request ModelRightsEvaluationRequest) (ModelRightsDecision, error) {
	if e == nil || e.evaluator == nil {
		return ModelRightsDecision{Status: string(sharedpolicy.RightsUnknown), Reason: "shared_model_rights_evaluator_unavailable"}, fmt.Errorf("shared model rights evaluator is unavailable")
	}
	var artifact sharedpolicy.ArtifactRights
	if err := json.Unmarshal(request.ArtifactFacts, &artifact); err != nil {
		return ModelRightsDecision{Status: string(sharedpolicy.RightsUnknown), Reason: "model_artifact_rights_facts_invalid"}, fmt.Errorf("decode rights facts for model %q: %w", request.ModelID, err)
	}
	if strings.TrimSpace(artifact.ArtifactID) == "" {
		return ModelRightsDecision{Status: string(sharedpolicy.RightsUnknown), Reason: "model_artifact_identity_missing"}, fmt.Errorf("rights facts for model %q have no artifact identity", request.ModelID)
	}
	use := sharedpolicy.RightsUse(request.Use)
	decision, err := e.evaluator.Evaluate(ctx, &artifact, use)
	return ModelRightsDecision{Status: string(decision.Status), Reason: decision.Reason, DecisionSHA256: decision.DecisionSHA256}, err
}

func applyBundledModelRights(ctx context.Context, root string, item *ResourceDeploymentPlanItem, evaluator ModelRightsEvaluator) error {
	if item == nil || !strings.HasPrefix(item.Mode, "bundled-") {
		return nil
	}
	manifest, err := loadResourceArtifactManifest(filepath.Join(root, "resources", item.Resource, "resource.json"))
	if err != nil {
		return fmt.Errorf("load bundled model rights declarations for resource %q: %w", item.Resource, err)
	}
	for _, artifact := range manifest.ModelArtifacts {
		modelID := strings.TrimSpace(artifact.ModelID)
		use := ModelRightsUse("")
		switch strings.TrimSpace(artifact.Kind) {
		case "weights":
			use = ModelRightsUseRedistributeWeights
		case "runtime":
			use = ModelRightsUseDistributeRuntime
		}
		decision := ModelRightsDecision{Status: "unknown", Reason: "shared_model_rights_evaluator_unavailable"}
		if modelID == "" {
			decision.Reason = "model_artifact_model_id_missing"
		} else if use == "" {
			decision.Reason = "model_artifact_kind_must_be_weights_or_runtime"
		} else if len(artifact.Rights) == 0 {
			decision.Reason = "model_artifact_rights_facts_missing"
		} else if evaluator != nil {
			decision, err = evaluator.EvaluateModelRights(ctx, ModelRightsEvaluationRequest{
				ModelID: modelID, Resource: item.Resource, ArtifactKind: artifact.Kind, Use: use,
				ArtifactFacts: append(json.RawMessage(nil), artifact.Rights...),
			})
			if err != nil {
				decision = ModelRightsDecision{Status: "unknown", Reason: "shared_model_rights_evaluation_failed: " + err.Error()}
			}
		}
		if decision.Status == "allowed" {
			continue
		}
		if decision.Reason == "" {
			decision.Reason = "shared_model_rights_evaluator_returned_no_reason"
		}
		refusal := fmt.Sprintf("model %q %s at %s/%s-%s refused: rights %s (%s)", modelID, artifact.Kind, item.Resource, item.OS, item.Architecture, decision.Status, decision.Reason)
		item.Eligibility = "ineligible"
		item.EligibilityReason = appendReason(item.EligibilityReason, refusal)
		item.Evidence = append(item.Evidence, refusal)
	}
	return nil
}

func appendReason(existing, next string) string {
	if strings.TrimSpace(existing) == "" {
		return next
	}
	return existing + "; " + next
}
