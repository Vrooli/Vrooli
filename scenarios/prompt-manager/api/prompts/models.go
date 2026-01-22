// Package prompts provides the core domain types and operations for prompt management.
// This is the heart of the prompt-manager scenario - all prompt-related concepts live here.
package prompts

// Metadata represents a prompt entry in metadata.json.
// Prompts are stored as markdown files with metadata tracked in JSON.
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
	Prompts []Metadata `json:"prompts"`
}

// Response is the API response for a prompt, enriched with content and metrics.
type Response struct {
	ID                  string   `json:"id"`
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
	Prompts     []Response `json:"prompts"`
	LastUpdated string     `json:"lastUpdated"`
	Hash        string     `json:"hash"`
}

// CreateRequest is the request body for creating a prompt.
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

// UpdateRequest is the request body for updating a prompt.
type UpdateRequest struct {
	Name         *string  `json:"name,omitempty"`
	Description  *string  `json:"description,omitempty"`
	Content      *string  `json:"content,omitempty"`
	Modes        []string `json:"modes,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Icon         *string  `json:"icon,omitempty"`
	TargetToolID *string  `json:"targetToolId,omitempty"`
	Draft        *bool    `json:"draft,omitempty"`
}

// CombineRequest is the request body for combining multiple prompts.
type CombineRequest struct {
	PromptIDs []string `json:"promptIds"`
	Format    string   `json:"format,omitempty"` // "xml", "markdown", or "json"
}

// CombineResponse is the response for combined prompts.
type CombineResponse struct {
	Combined    string `json:"combined"`
	PromptCount int    `json:"promptCount"`
	TotalTokens int    `json:"totalTokens"`
	Format      string `json:"format"`
}

// Folders defines the valid folder names for prompt storage.
var Folders = []string{"core", "local", "drafts"}

// WritableFolders defines folders where prompts can be created/updated/deleted.
var WritableFolders = []string{"local", "drafts"}

// IsWritableFolder checks if a folder allows modifications.
func IsWritableFolder(folder string) bool {
	for _, f := range WritableFolders {
		if f == folder {
			return true
		}
	}
	return false
}
