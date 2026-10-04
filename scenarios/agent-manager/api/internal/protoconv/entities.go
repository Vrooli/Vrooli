package protoconv

import (
	"agent-manager/internal/domain"
	"github.com/google/uuid"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"strings"
)

// validUTF8 protects the protobuf boundary from historical/imported data that
// may contain arbitrary bytes. Protobuf string fields must contain UTF-8;
// replacing malformed sequences keeps read surfaces available without
// discarding otherwise valid content.
func validUTF8(value string) string {
	return strings.ToValidUTF8(value, "\uFFFD")
}

func validUTF8Slice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = validUTF8(value)
	}
	return result
}

// =============================================================================
// AGENT PROFILE
// =============================================================================

// AgentProfileToProto converts a domain AgentProfile to proto AgentProfile.
func AgentProfileToProto(p *domain.AgentProfile) *pb.AgentProfile {
	if p == nil {
		return nil
	}
	return &pb.AgentProfile{
		Id:                    UUIDToString(p.ID),
		Name:                  p.Name,
		ProfileKey:            p.ProfileKey,
		Description:           p.Description,
		RoleRef:               p.RoleRef,
		MaxTurns:              int32(p.MaxTurns),
		Timeout:               DurationToProto(p.Timeout),
		Effort:                string(p.Effort),
		AllowedTools:          p.AllowedTools,
		DeniedTools:           p.DeniedTools,
		ToolRestrictionPolicy: string(p.ToolRestrictionPolicy.Effective()),
		SkipPermissionPrompt:  p.SkipPermissionPrompt,
		Features:              FeatureFlagsToProto(p.Features),
		ExtraFlags:            RunnerExtraFlagsToProto(p.ExtraFlags),
		NetworkAccess:         NetworkAccessToProto(p.NetworkAccess),
		OwnerScenario:         p.OwnerScenario,
		SourcePath:            p.SourcePath,
		SourceHash:            p.SourceHash,
		LastAppliedHash:       p.LastAppliedHash,
		SourceUpdatedAt:       TimestampToProto(p.SourceUpdatedAt),
		LocalOverride:         p.LocalOverride,
		SandboxConfig:         SandboxConfigToProto(p.SandboxConfig),
		AllowedPaths:          p.AllowedPaths,
		DeniedPaths:           p.DeniedPaths,
		DeclaredScopes:        p.DeclaredScopes,
		CreatedBy:             p.CreatedBy,
		CreatedAt:             TimestampToProto(p.CreatedAt),
		UpdatedAt:             TimestampToProto(p.UpdatedAt),
		SkillPack:             p.SkillPack,
		SkillExperimentId:     p.SkillExperimentID,
	}
}

// AgentProfileFromProto converts a proto AgentProfile to domain AgentProfile.
func AgentProfileFromProto(p *pb.AgentProfile) *domain.AgentProfile {
	if p == nil {
		return nil
	}
	return &domain.AgentProfile{
		ID:                    UUIDFromString(p.Id),
		Name:                  p.Name,
		ProfileKey:            p.ProfileKey,
		Description:           p.Description,
		RoleRef:               p.RoleRef,
		MaxTurns:              int(p.MaxTurns),
		Timeout:               DurationFromProto(p.Timeout),
		Effort:                domain.Effort(p.Effort),
		AllowedTools:          p.AllowedTools,
		DeniedTools:           p.DeniedTools,
		ToolRestrictionPolicy: domain.ToolRestrictionPolicy(p.ToolRestrictionPolicy),
		SkipPermissionPrompt:  p.SkipPermissionPrompt,
		Features:              FeatureFlagsFromProto(p.Features),
		ExtraFlags:            RunnerExtraFlagsFromProto(p.ExtraFlags),
		NetworkAccess:         NetworkAccessFromProto(p.NetworkAccess),
		OwnerScenario:         p.OwnerScenario,
		SourcePath:            p.SourcePath,
		SourceHash:            p.SourceHash,
		LastAppliedHash:       p.LastAppliedHash,
		SourceUpdatedAt:       TimestampFromProto(p.SourceUpdatedAt),
		LocalOverride:         p.LocalOverride,
		SandboxConfig:         SandboxConfigFromProto(p.SandboxConfig),
		AllowedPaths:          p.AllowedPaths,
		DeniedPaths:           p.DeniedPaths,
		DeclaredScopes:        p.DeclaredScopes,
		CreatedBy:             p.CreatedBy,
		CreatedAt:             TimestampFromProto(p.CreatedAt),
		UpdatedAt:             TimestampFromProto(p.UpdatedAt),
		SkillPack:             p.SkillPack,
		SkillExperimentID:     p.SkillExperimentId,
	}
}

// AgentProfilesToProto converts a slice of domain AgentProfile to proto.
func AgentProfilesToProto(profiles []*domain.AgentProfile) []*pb.AgentProfile {
	result := make([]*pb.AgentProfile, len(profiles))
	for i, p := range profiles {
		result[i] = AgentProfileToProto(p)
	}
	return result
}

// =============================================================================
// TASK
// =============================================================================

// TaskToProto converts a domain Task to proto Task.
func TaskToProto(t *domain.Task) *pb.Task {
	if t == nil {
		return nil
	}

	phasePromptIDs := make([]string, len(t.PhasePromptIDs))
	for i, id := range t.PhasePromptIDs {
		phasePromptIDs[i] = id.String()
	}

	attachments := make([]*pb.ContextAttachment, len(t.ContextAttachments))
	for i, a := range t.ContextAttachments {
		attachments[i] = &pb.ContextAttachment{
			Type:         a.Type,
			Key:          a.Key,
			Tags:         a.Tags,
			Path:         a.Path,
			Url:          a.URL,
			Content:      a.Content,
			Label:        a.Label,
			Summary:      a.Summary,
			Format:       a.Format,
			Priority:     a.Priority,
			AttachmentId: a.AttachmentID,
		}
	}

	return &pb.Task{
		Id:                 UUIDToString(t.ID),
		Title:              t.Title,
		Description:        t.Description,
		ScopePath:          t.ScopePath,
		ProjectRoot:        t.ProjectRoot,
		PhasePromptIds:     phasePromptIDs,
		ContextAttachments: attachments,
		Status:             TaskStatusToProto(t.Status),
		CreatedBy:          t.CreatedBy,
		CreatedAt:          TimestampToProto(t.CreatedAt),
		UpdatedAt:          TimestampToProto(t.UpdatedAt),
	}
}

// TaskFromProto converts a proto Task to domain Task.
func TaskFromProto(t *pb.Task) *domain.Task {
	if t == nil {
		return nil
	}

	phasePromptIDs := make([]uuid.UUID, len(t.PhasePromptIds))
	for i, id := range t.PhasePromptIds {
		phasePromptIDs[i] = UUIDFromString(id)
	}

	attachments := make([]domain.ContextAttachment, len(t.ContextAttachments))
	for i, a := range t.ContextAttachments {
		attachments[i] = domain.ContextAttachment{
			Type:         a.Type,
			Key:          a.Key,
			Tags:         a.Tags,
			Path:         a.Path,
			URL:          a.Url,
			Content:      a.Content,
			Label:        a.Label,
			Summary:      a.Summary,
			Format:       a.Format,
			Priority:     a.Priority,
			AttachmentID: a.AttachmentId,
		}
	}

	return &domain.Task{
		ID:                 UUIDFromString(t.Id),
		Title:              t.Title,
		Description:        t.Description,
		ScopePath:          t.ScopePath,
		ProjectRoot:        t.ProjectRoot,
		PhasePromptIDs:     phasePromptIDs,
		ContextAttachments: attachments,
		Status:             TaskStatusFromProto(t.Status),
		CreatedBy:          t.CreatedBy,
		CreatedAt:          TimestampFromProto(t.CreatedAt),
		UpdatedAt:          TimestampFromProto(t.UpdatedAt),
	}
}

// TasksToProto converts a slice of domain Task to proto.
func TasksToProto(tasks []*domain.Task) []*pb.Task {
	result := make([]*pb.Task, len(tasks))
	for i, t := range tasks {
		result[i] = TaskToProto(t)
	}
	return result
}

// =============================================================================
// RUN
// =============================================================================

// FeatureFlagsToProto converts domain FeatureFlags to proto FeatureFlags.
func FeatureFlagsToProto(f domain.FeatureFlags) *pb.FeatureFlags {
	if f.IsZero() {
		return nil
	}
	return &pb.FeatureFlags{EnableBrowser: f.EnableBrowser}
}

// FeatureFlagsFromProto converts proto FeatureFlags to domain FeatureFlags.
func FeatureFlagsFromProto(f *pb.FeatureFlags) domain.FeatureFlags {
	if f == nil {
		return domain.FeatureFlags{}
	}
	return domain.FeatureFlags{EnableBrowser: f.EnableBrowser}
}

// RunnerExtraFlagsToProto converts domain RunnerExtraFlags to proto map.
func RunnerExtraFlagsToProto(flags domain.RunnerExtraFlags) map[string]*pb.ExtraFlagList {
	if len(flags) == 0 {
		return nil
	}
	result := make(map[string]*pb.ExtraFlagList, len(flags))
	for rt, flagList := range flags {
		result[string(rt)] = &pb.ExtraFlagList{Flags: flagList}
	}
	return result
}

// RunnerExtraFlagsFromProto converts proto map to domain RunnerExtraFlags.
func RunnerExtraFlagsFromProto(flags map[string]*pb.ExtraFlagList) domain.RunnerExtraFlags {
	if len(flags) == 0 {
		return nil
	}
	result := make(domain.RunnerExtraFlags, len(flags))
	for rt, flagList := range flags {
		if flagList != nil && len(flagList.Flags) > 0 {
			result[domain.RunnerType(rt)] = flagList.Flags
		}
	}
	return result
}

// =============================================================================
// RUN EVENT
// =============================================================================
