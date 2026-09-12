package skills

import "prompt-manager/internal/store"

// ImportService is the narrow operator-facing seam for governed vendor skills.
// Keeping it separate from the legacy metadata adapter ensures imported bytes
// always use the pack store's quarantine rules.
type ImportService interface {
	ImportSkill(store.ImportRequest) (*store.Skill, error)
	ReviewImportedSkill(id, reviewer, verdict string) error
	ImportedSkillStaleness(id, currentVersion string) (string, bool, error)
}
