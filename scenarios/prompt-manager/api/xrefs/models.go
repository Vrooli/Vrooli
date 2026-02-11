package xrefs

// ReferenceType classifies how a skill is referenced.
type ReferenceType string

const (
	RefCLIRead      ReferenceType = "cli-read"
	RefBoldListed   ReferenceType = "bold-listed"
	RefDefaultScope ReferenceType = "default-scope"
	RefPathRef      ReferenceType = "path-ref"
)

// ReferenceSource identifies where a reference was found.
type ReferenceSource struct {
	EntityType string `json:"entityType"` // "agent", "team", "skill"
	EntityID   string `json:"entityId"`
	EntityName string `json:"entityName"`
	FilePath   string `json:"filePath"`   // e.g. "AGENTS.md", "shared/TEAM.md"
	LineNumber int    `json:"lineNumber"` // 1-based; 0 for structured data
}

// Reference represents a single cross-reference to a skill.
type Reference struct {
	SkillID string          `json:"skillId"`
	RefType ReferenceType   `json:"refType"`
	Source  ReferenceSource `json:"source"`
}

// XRefsIndex is the persisted index of all cross-references.
type XRefsIndex struct {
	GeneratedAt string      `json:"generatedAt"`
	References  []Reference `json:"references"`
}

// SkillXRefsResponse is the API response for a single skill's references.
type SkillXRefsResponse struct {
	SkillID    string      `json:"skillId"`
	References []Reference `json:"references"`
	Total      int         `json:"total"`
}

// XRefInvalidator allows other packages to trigger index invalidation.
type XRefInvalidator interface {
	Invalidate()
}
