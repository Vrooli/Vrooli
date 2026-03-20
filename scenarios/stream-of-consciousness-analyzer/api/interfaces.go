// DOC: docs/internal/SEAMS.md#service-interfaces
package main

// SchemeStore defines the contract for scheme persistence operations.
// Implemented by SchemeService (production) and mock stores (testing).
type SchemeStore interface {
	List() ([]Scheme, error)
	Create(input *CreateSchemeInput) (*Scheme, error)
	GetByID(id string) (*Scheme, error)
	Update(id string, input *UpdateSchemeInput) (*Scheme, error)
	Delete(id string) error
}

// InformationStore defines the contract for information item persistence.
type InformationStore interface {
	ListByScheme(schemeID string) ([]Information, error)
	Create(schemeID string, input *CreateInformationInput) (*Information, error)
	Update(id string, input *UpdateInformationInput) (*Information, error)
	Delete(id string) error
}

// ThoughtStore defines the contract for thought and edge persistence.
type ThoughtStore interface {
	List(schemeID string) ([]Thought, error)
	Create(input *CreateThoughtInput) (*Thought, error)
	GetByID(id string) (*Thought, error)
	Update(id string, input *UpdateThoughtInput) (*Thought, error)
	Delete(id string) error
	CreateEdge(sourceID string, input *CreateEdgeInput) (*ThoughtEdge, error)
	ListEdges(thoughtID string) ([]ThoughtEdge, error)
	DeleteEdge(id string) error
}

// ExportStore defines the contract for scheme export operations.
type ExportStore interface {
	ExportScheme(schemeID string) (*ExportData, error)
}

// SuggestionProvider defines the contract for LLM suggestion operations.
type SuggestionProvider interface {
	GetProviders() []LLMProvider
	GetActiveProvider() (*LLMProvider, error)
	GenerateSuggestions(schemeID string) ([]Suggestion, *LLMProvider, error)
}
