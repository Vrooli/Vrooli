package protoconv

import (
	"sort"

	"agent-manager/internal/domain"
	"agent-manager/internal/modelpolicy"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ModelPolicyStatusToProto converts activation state without dropping the last
// failed-attempt diagnostic that operators need for recovery.
func ModelPolicyStatusToProto(status modelpolicy.Status) *apipb.ModelPolicyStatus {
	result := &apipb.ModelPolicyStatus{
		Path: status.Path,
		Requirement: &apipb.ModelPolicyRequirement{
			Required: status.Requirement.Required,
			Reason:   status.Requirement.Reason,
		},
		Ready:        status.Ready,
		ActiveDigest: status.ActiveDigest,
	}
	if status.ActivatedAt != nil {
		result.ActivatedAt = timestamppb.New(*status.ActivatedAt)
	}
	if attempt := status.LastReloadAttempt; attempt != nil {
		result.LastReloadAttempt = &apipb.ModelPolicyReloadAttempt{
			AttemptedAt: timestamppb.New(attempt.AttemptedAt),
			Succeeded:   attempt.Succeeded,
			Digest:      attempt.Digest,
			Diagnostic:  ModelPolicyDiagnosticToProto(attempt.Diagnostic),
		}
	}
	return result
}

func ModelPolicyDiagnosticToProto(diagnostic *modelpolicy.Diagnostic) *apipb.ModelPolicyDiagnostic {
	if diagnostic == nil {
		return nil
	}
	return &apipb.ModelPolicyDiagnostic{
		Code:    diagnostic.Code,
		Message: diagnostic.Message,
		Cause:   diagnostic.Cause,
	}
}

// ModelPolicyCatalogToProto returns a stable ordered projection of map-backed
// declared state so human and JSON output remain diff-friendly.
func ModelPolicyCatalogToProto(catalog *modelpolicy.Catalog) *apipb.ModelPolicyCatalog {
	if catalog == nil {
		return nil
	}
	result := &apipb.ModelPolicyCatalog{
		SchemaVersion: int32(catalog.SchemaVersion),
		Metadata: &apipb.ModelPolicyCatalogMetadata{
			CatalogId: catalog.Metadata.CatalogID,
			UpdatedAt: catalog.Metadata.UpdatedAt,
		},
		DefaultPolicy: catalog.DefaultPolicy,
	}
	for _, source := range catalog.Metadata.Sources {
		result.Metadata.Sources = append(result.Metadata.Sources, &apipb.ModelPolicySource{
			Name: source.Name, Reference: source.Reference, VerifiedAt: source.VerifiedAt,
		})
	}

	runnerNames := make([]string, 0, len(catalog.Runners))
	for runnerType := range catalog.Runners {
		runnerNames = append(runnerNames, string(runnerType))
	}
	sort.Strings(runnerNames)
	for _, name := range runnerNames {
		runnerType := domain.RunnerType(name)
		inventory := catalog.Runners[runnerType]
		converted := &apipb.ModelPolicyRunnerInventory{
			RunnerType:            RunnerTypeToProto(runnerType),
			SupportsRunnerDefault: inventory.SupportsRunnerDefault,
			DynamicModelPrefixes:  append([]string(nil), inventory.DynamicModelPrefixes...),
		}
		for _, model := range inventory.Models {
			converted.Models = append(converted.Models, &apipb.ModelPolicyModel{Id: model.ID, Description: model.Description})
		}
		result.Runners = append(result.Runners, converted)
	}

	policyNames := make([]string, 0, len(catalog.Policies))
	for name := range catalog.Policies {
		policyNames = append(policyNames, name)
	}
	sort.Strings(policyNames)
	for _, name := range policyNames {
		policy := catalog.Policies[name]
		converted := &apipb.ModelPolicyDefinition{Name: name, Intent: string(policy.Intent)}
		for _, candidate := range policy.Candidates {
			selectionType := modelpolicySelectionTypeToProto(candidate.Selection.Type)
			converted.Candidates = append(converted.Candidates, &apipb.ModelPolicyCandidate{
				RunnerType: RunnerTypeToProto(candidate.Runner), SelectionType: selectionType, Model: candidate.Selection.Model,
			})
		}
		result.Policies = append(result.Policies, converted)
	}
	return result
}

func modelpolicySelectionTypeToProto(selection modelpolicy.SelectionType) domainpb.ModelSelectionType {
	switch selection {
	case modelpolicy.SelectionTypeModel:
		return domainpb.ModelSelectionType_MODEL_SELECTION_TYPE_MODEL
	case modelpolicy.SelectionTypeRunnerDefault:
		return domainpb.ModelSelectionType_MODEL_SELECTION_TYPE_RUNNER_DEFAULT
	default:
		return domainpb.ModelSelectionType_MODEL_SELECTION_TYPE_UNSPECIFIED
	}
}
