package aisearch

import (
	"fmt"
	"strings"
)

// ReindexMigrationRequest is the provider-neutral portion of the shared
// Search Hub reindex wire contract. Keeping this value type independent of the
// generated proto lets every owner use the same validation without coupling
// this shared search package to the proto module.
const ReindexMigrationProtocolVersion uint32 = 1

// NormalizeReindexMigrationAction canonicalizes the legacy promote alias so
// providers negotiate and execute one pointer-switch vocabulary.
func NormalizeReindexMigrationAction(action string) string {
	action = strings.ToLower(strings.TrimSpace(action))
	if action == "promote" {
		return "cutover"
	}
	return action
}

type ReindexMigrationRequest struct {
	Action                       string
	ShadowCollection             string
	RollbackCollection           string
	EmbeddingModel               string
	EmbeddingRole                string
	EmbeddingDimensions          int32
	EmbeddingPolicySchemaVersion string
	DryRun                       bool
}

// ValidateReindexMigrationRequest validates the additive migration portion of
// the shared Search Hub Reindex contract before a provider starts any work.
// An empty action remains the legacy ordinary-reindex request, but it cannot
// carry migration targets that a provider might otherwise silently ignore.
//
// allowedActions is the provider's actual capability set. A declared reindex
// endpoint alone does not imply that it can build or switch embedding indexes.
func ValidateReindexMigrationRequest(req *ReindexMigrationRequest, allowedActions ...string) error {
	if req == nil {
		return fmt.Errorf("reindex request is required")
	}
	action := NormalizeReindexMigrationAction(req.Action)
	allowed := make(map[string]struct{}, len(allowedActions))
	for _, value := range allowedActions {
		allowed[strings.ToLower(strings.TrimSpace(value))] = struct{}{}
	}
	if action == "" {
		if hasMigrationTarget(req) {
			return fmt.Errorf("migration target fields require an explicitly supported action")
		}
		return nil
	}
	if _, ok := allowed[action]; !ok {
		return fmt.Errorf("unsupported reindex action %q", action)
	}
	switch action {
	case "capabilities":
		if hasMigrationTarget(req) || req.DryRun {
			return fmt.Errorf("capabilities cannot include migration targets or dry_run")
		}
	case "shadow":
		if strings.TrimSpace(req.ShadowCollection) == "" || strings.TrimSpace(req.EmbeddingModel) == "" || strings.TrimSpace(req.EmbeddingRole) == "" || req.EmbeddingDimensions <= 0 || strings.TrimSpace(req.EmbeddingPolicySchemaVersion) == "" {
			return fmt.Errorf("shadow requires a collection and complete embedding target (model, role, positive dimensions, and policy schema version)")
		}
		if !validMigrationCollection(req.ShadowCollection) {
			return fmt.Errorf("shadow_collection has an invalid collection name")
		}
		if strings.TrimSpace(req.RollbackCollection) != "" {
			return fmt.Errorf("shadow cannot include a rollback collection")
		}
	case "cutover":
		if strings.TrimSpace(req.ShadowCollection) == "" {
			return fmt.Errorf("%s requires a shadow collection", action)
		}
		if !validMigrationCollection(req.ShadowCollection) {
			return fmt.Errorf("shadow_collection has an invalid collection name")
		}
		if hasEmbeddingRecipe(req) || strings.TrimSpace(req.RollbackCollection) != "" || req.DryRun {
			return fmt.Errorf("%s cannot include an embedding target, rollback collection, or dry_run", action)
		}
	case "rollback":
		if strings.TrimSpace(req.RollbackCollection) == "" {
			return fmt.Errorf("rollback requires a rollback collection")
		}
		if !validMigrationCollection(req.RollbackCollection) {
			return fmt.Errorf("rollback_collection has an invalid collection name")
		}
		if strings.TrimSpace(req.ShadowCollection) != "" || hasEmbeddingRecipe(req) || req.DryRun {
			return fmt.Errorf("rollback cannot include a shadow collection, embedding target, or dry_run")
		}
	default:
		if hasMigrationTarget(req) {
			return fmt.Errorf("action %q cannot include migration target fields", action)
		}
	}
	return nil
}

func validMigrationCollection(value string) bool {
	if len(value) == 0 || len(value) > 128 || strings.TrimSpace(value) != value {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			continue
		}
		return false
	}
	return value != "." && value != ".."
}

func hasMigrationTarget(req *ReindexMigrationRequest) bool {
	return strings.TrimSpace(req.ShadowCollection) != "" || strings.TrimSpace(req.RollbackCollection) != "" || hasEmbeddingRecipe(req)
}

func hasEmbeddingRecipe(req *ReindexMigrationRequest) bool {
	return strings.TrimSpace(req.EmbeddingModel) != "" || strings.TrimSpace(req.EmbeddingRole) != "" || req.EmbeddingDimensions != 0 || strings.TrimSpace(req.EmbeddingPolicySchemaVersion) != ""
}
