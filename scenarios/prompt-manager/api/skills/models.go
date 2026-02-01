// Package skills provides the core domain types and operations for skill management.
// This is the heart of the prompt-manager scenario - all skill-related concepts live here.
package skills

// Metadata represents a skill entry in metadata.json.
// Skills are stored as markdown files with metadata tracked in JSON.
type Metadata struct {
	ID           string   `json:"id"`
	File         string   `json:"file"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Modes        []string `json:"modes"`
	Tags         []string `json:"tags"`
	Icon         string   `json:"icon,omitempty"`
	TargetToolID *string  `json:"targetToolId"`
	Draft        bool     `json:"draft"`
	CreatedAt    string   `json:"createdAt"`
	UpdatedAt    string   `json:"updatedAt"`
}

// MetadataFile represents the structure of metadata.json files in each folder.
type MetadataFile struct {
	Skills []Metadata `json:"skills"`
}

// Response is the API response for a skill, enriched with content and metrics.
type Response struct {
	ID                  string     `json:"id"`
	File                string     `json:"file"`
	Name                string     `json:"name"`
	Description         string     `json:"description"`
	Content             string     `json:"content"`
	Modes               []string   `json:"modes"`
	Tags                []string   `json:"tags"`
	Icon                string     `json:"icon,omitempty"`
	TargetToolID        *string    `json:"targetToolId,omitempty"`
	Draft               bool       `json:"draft"`
	Folder              string     `json:"folder"`
	SkillDir            string     `json:"skillDir,omitempty"`    // Absolute path to skill directory
	ContentPath         string     `json:"contentPath,omitempty"` // Absolute path to SKILL.md file
	CreatedAt           string     `json:"createdAt"`
	UpdatedAt           string     `json:"updatedAt"`
	UsageCount          int        `json:"usageCount"`
	LastUsed            *string    `json:"lastUsed,omitempty"`
	EffectivenessRating *int       `json:"effectivenessRating,omitempty"`
	Variables           []Variable `json:"variables,omitempty"`
}

// SyncResponse is returned by the sync endpoint for consumers like ecosystem-manager.
type SyncResponse struct {
	Skills      []Response `json:"skills"`
	LastUpdated string     `json:"lastUpdated"`
	Hash        string     `json:"hash"`
}

// CreateRequest is the request body for creating a skill.
type CreateRequest struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Content      string   `json:"content"`
	Modes        []string `json:"modes"`
	Tags         []string `json:"tags"`
	Icon         string   `json:"icon,omitempty"`
	TargetToolID *string  `json:"targetToolId,omitempty"`
	Draft        bool     `json:"draft"`
	Folder       string   `json:"folder"` // "local" or "drafts" only
}

// UpdateRequest is the request body for updating a skill.
type UpdateRequest struct {
	File         *string  `json:"file,omitempty"`
	Name         *string  `json:"name,omitempty"`
	Description  *string  `json:"description,omitempty"`
	Content      *string  `json:"content,omitempty"`
	Modes        []string `json:"modes,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Icon         *string  `json:"icon,omitempty"`
	TargetToolID *string  `json:"targetToolId,omitempty"`
	Draft        *bool    `json:"draft,omitempty"`
	Folder       *string  `json:"folder,omitempty"` // Move skill to different folder
}

// Folders defines the valid folder names for skill storage.
var Folders = []string{"core", "local", "drafts"}

// WritableFolders defines folders where skills can be created/updated/deleted.
var WritableFolders = []string{"core", "local", "drafts"}

// IsWritableFolder checks if a folder allows modifications.
func IsWritableFolder(folder string) bool {
	for _, f := range WritableFolders {
		if f == folder {
			return true
		}
	}
	return false
}

// SkillVersion represents a historical version of a skill.
type SkillVersion struct {
	Version   int    `json:"version"`
	Content   string `json:"content"`
	Name      string `json:"name"`
	UpdatedAt string `json:"updatedAt"`
	CreatedBy string `json:"createdBy,omitempty"`
}

// VersionFile represents the structure of versions.json files.
type VersionFile struct {
	SkillID  string         `json:"skillId"`
	Versions []SkillVersion `json:"versions"`
}

// VersionsResponse is the API response for version history.
type VersionsResponse struct {
	SkillID  string         `json:"skillId"`
	Current  int            `json:"current"`
	Versions []SkillVersion `json:"versions"`
}

// RevertResponse is the API response for reverting to a version.
type RevertResponse struct {
	SkillID    string `json:"skillId"`
	RevertedTo int    `json:"revertedTo"`
	NewVersion int    `json:"newVersion"`
	RestoredAt string `json:"restoredAt"`
}

// ReadRequest is the request body for reading multiple skills by identifier.
type ReadRequest struct {
	Identifiers  []string          `json:"identifiers"`
	Resolve      string            `json:"resolve,omitempty"`      // "auto", "id", "file", or "name"
	AllowMissing *bool             `json:"allowMissing,omitempty"` // default true
	Output       string            `json:"output,omitempty"`       // "skills", "combined", or "both"
	Format       string            `json:"format,omitempty"`       // "xml", "markdown", or "json" (for combined output)
	Variables    map[string]string `json:"variables,omitempty"`    // Values for {{VAR}} substitution
}

// ReadIssue captures missing identifiers.
type ReadIssue struct {
	Identifier string `json:"identifier"`
	Reason     string `json:"reason"`
}

// ReadCandidate is a minimal representation of a skill for ambiguity reporting.
type ReadCandidate struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	File   string `json:"file"`
	Folder string `json:"folder"`
}

// ReadAmbiguous captures ambiguous identifiers.
type ReadAmbiguous struct {
	Identifier string          `json:"identifier"`
	Candidates []ReadCandidate `json:"candidates"`
}

// ReadResponse is the response for reading multiple skills.
type ReadResponse struct {
	Skills      []Response      `json:"skills,omitempty"`
	Combined    string          `json:"combined,omitempty"`
	SkillCount  int             `json:"skillCount,omitempty"`
	TotalTokens int             `json:"totalTokens,omitempty"`
	Format      string          `json:"format,omitempty"`
	Missing     []ReadIssue     `json:"missing,omitempty"`
	Ambiguous   []ReadAmbiguous `json:"ambiguous,omitempty"`
	Resolve     string          `json:"resolve"`
	Output      string          `json:"output,omitempty"`
}
