package skills

import (
	"context"

	"prompt-manager/internal/store"
)

// ImportService is the narrow operator-facing seam for governed vendor skills.
// Keeping it separate from the legacy metadata adapter ensures imported bytes
// always use the pack store's quarantine rules.
type ImportService interface {
	ImportSkill(context.Context, store.ImportRequest) (*store.Skill, error)
	ReviewImportedSkill(ctx context.Context, id, reviewer, verdict string) error
	ImportedSkillStaleness(ctx context.Context, id, currentVersion string) (string, bool, error)
}
