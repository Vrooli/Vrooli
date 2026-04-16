package support

import "time"

// Component mirrors the shape returned by /api/v1/components and /api/v1/components/{id}.
type Component struct {
	ID          string    `json:"id"`
	LibraryID   string    `json:"libraryId"`
	DisplayName string    `json:"displayName"`
	Description string    `json:"description"`
	Version     string    `json:"version"`
	FilePath    string    `json:"filePath"`
	SourcePath  string    `json:"sourcePath"`
	Tags        []string  `json:"tags"`
	Category    *string   `json:"category,omitempty"`
	TechStack   []string  `json:"techStack"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// ComponentVersion mirrors /api/v1/components/{id}/versions entries.
type ComponentVersion struct {
	ID          string    `json:"id"`
	ComponentID string    `json:"componentId"`
	Version     string    `json:"version"`
	Content     string    `json:"content"`
	Changelog   *string   `json:"changelog,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

// ComponentContent mirrors /api/v1/components/{id}/content responses.
type ComponentContent struct {
	Content string `json:"content"`
}

// AdoptionRecord mirrors /api/v1/adoptions entries.
type AdoptionRecord struct {
	ID                 string    `json:"id"`
	ComponentID        string    `json:"componentId"`
	ComponentLibraryID string    `json:"componentLibraryId"`
	ScenarioName       string    `json:"scenarioName"`
	AdoptedPath        string    `json:"adoptedPath"`
	Version            string    `json:"version"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// AIResponse mirrors the shape returned by /api/v1/ai/chat.
type AIResponse struct {
	Response    string   `json:"response"`
	Suggestions []string `json:"suggestions,omitempty"`
}

// AIRefactorResponse mirrors the shape returned by /api/v1/ai/refactor.
type AIRefactorResponse struct {
	RefactoredCode string `json:"refactoredCode"`
	Diff           string `json:"diff"`
}
