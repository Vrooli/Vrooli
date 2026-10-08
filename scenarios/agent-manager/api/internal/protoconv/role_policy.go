package protoconv

import (
	"sort"

	"agent-manager/internal/domain"
	"agent-manager/internal/rolepolicy"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// RolePolicyStatusToProto preserves activation and failed-reload diagnostics so
// operators can recover without reading server logs.
func RolePolicyStatusToProto(status rolepolicy.Status) *apipb.RolePolicyStatus {
	result := &apipb.RolePolicyStatus{
		Path: status.Path,
		Requirement: &apipb.RolePolicyRequirement{
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
		result.LastReloadAttempt = &apipb.RolePolicyReloadAttempt{
			AttemptedAt: timestamppb.New(attempt.AttemptedAt),
			Succeeded:   attempt.Succeeded,
			Digest:      attempt.Digest,
			Diagnostic:  RolePolicyDiagnosticToProto(attempt.Diagnostic),
		}
	}
	return result
}

func RolePolicyDiagnosticToProto(diagnostic *rolepolicy.Diagnostic) *apipb.RolePolicyDiagnostic {
	if diagnostic == nil {
		return nil
	}
	return &apipb.RolePolicyDiagnostic{
		Code:    diagnostic.Code,
		Message: diagnostic.Message,
		Cause:   diagnostic.Cause,
	}
}

// RolePolicyCatalogToProto exposes portable role intent only. Resources are
// the sole authority for concrete model selection, which appears only in a
// resolved run snapshot.
func RolePolicyCatalogToProto(catalog *rolepolicy.Catalog) *apipb.RolePolicyCatalog {
	if catalog == nil {
		return nil
	}
	result := &apipb.RolePolicyCatalog{
		SchemaVersion: int32(catalog.SchemaVersion),
		Metadata: &apipb.RolePolicyCatalogMetadata{
			CatalogId: catalog.Metadata.CatalogID,
			UpdatedAt: catalog.Metadata.UpdatedAt,
		},
		DefaultRole: catalog.DefaultRole,
	}

	roleRefs := make([]string, 0, len(catalog.Roles))
	for roleRef := range catalog.Roles {
		roleRefs = append(roleRefs, roleRef)
	}
	sort.Strings(roleRefs)
	for _, roleRef := range roleRefs {
		role := catalog.Roles[roleRef]
		converted := &apipb.RolePolicyDefinition{
			RoleRef:     roleRef,
			Intent:      role.Intent,
			Description: role.Description,
		}
		for _, candidate := range role.Candidates {
			converted.Candidates = append(converted.Candidates, &apipb.RolePolicyCandidate{
				RunnerType:   RunnerTypeToProto(domain.RunnerType(candidate.Runner)),
				ResourceRole: candidate.ResourceRole,
			})
		}
		result.Roles = append(result.Roles, converted)
	}
	return result
}
