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
	ID                  string   `json:"id"`
	File                string   `json:"file"`
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	Content             string   `json:"content"`
	Modes               []string `json:"modes"`
	Tags                []string `json:"tags"`
	Icon                string   `json:"icon,omitempty"`
	TargetToolID        *string  `json:"targetToolId,omitempty"`
	Draft               bool     `json:"draft"`
	Folder              string   `json:"folder"`
	CreatedAt           string   `json:"createdAt"`
	UpdatedAt           string   `json:"updatedAt"`
	UsageCount          int      `json:"usageCount"`
	LastUsed            *string  `json:"lastUsed,omitempty"`
	EffectivenessRating *int     `json:"effectivenessRating,omitempty"`
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
}

// CombineRequest is the request body for combining multiple skills.
type CombineRequest struct {
	SkillIDs []string `json:"skillIds"`
	Format   string   `json:"format,omitempty"` // "xml", "markdown", or "json"
}

// CombineResponse is the response for combined skills.
type CombineResponse struct {
	Combined    string `json:"combined"`
	SkillCount  int    `json:"skillCount"`
	TotalTokens int    `json:"totalTokens"`
	Format      string `json:"format"`
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
