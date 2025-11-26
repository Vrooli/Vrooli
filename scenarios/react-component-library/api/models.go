package main

import (
	"time"
)

// Component represents a React component in the library
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

// ComponentVersion represents a historical snapshot of a component
type ComponentVersion struct {
	ID          string    `json:"id"`
	ComponentID string    `json:"componentId"`
	Version     string    `json:"version"`
	Content     string    `json:"content"`
	Changelog   *string   `json:"changelog,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

// AdoptionRecord tracks which scenarios use which components
type AdoptionRecord struct {
	ID                 string    `json:"id"`
	ComponentID        string    `json:"componentId"`
	ComponentLibraryID string    `json:"componentLibraryId"`
	ScenarioName       string    `json:"scenarioName"`
	AdoptedPath        string    `json:"adoptedPath"`
	Version            string    `json:"version"`
	Status             string    `json:"status"` // current, behind, modified, unknown
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// ComponentContent is used for content-only responses
type ComponentContent struct {
	Content string `json:"content"`
}

// AIRequest represents a request to the AI service
type AIRequest struct {
	Message string                 `json:"message"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// AIResponse represents a response from the AI service
type AIResponse struct {
	Response    string   `json:"response"`
	Suggestions []string `json:"suggestions,omitempty"`
}

// AIRefactorRequest represents a request to refactor code
type AIRefactorRequest struct {
	Code        string `json:"code"`
	Instruction string `json:"instruction"`
}

// AIRefactorResponse represents a refactored code response
type AIRefactorResponse struct {
	RefactoredCode string `json:"refactoredCode"`
	Diff           string `json:"diff"`
}
