package main

import (
	"database/sql"
	"fmt"
	"time"
)

// --- Mock Implementations (implement service interfaces) ---

// mockSchemes implements SchemeStore for handler testing.
type mockSchemes struct {
	schemes map[string]*Scheme
	nextIdx int
	listErr error
}

func newMockSchemes() *mockSchemes {
	return &mockSchemes{schemes: make(map[string]*Scheme)}
}

func (m *mockSchemes) WithListError(err error) *mockSchemes {
	m.listErr = err
	return m
}

func (m *mockSchemes) seed(name string) *Scheme {
	m.nextIdx++
	id := fakeUUID(m.nextIdx)
	s := &Scheme{ID: id, Name: name, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	m.schemes[id] = s
	return s
}

func (m *mockSchemes) List() ([]Scheme, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	out := make([]Scheme, 0, len(m.schemes))
	for _, s := range m.schemes {
		out = append(out, *s)
	}
	return out, nil
}

func (m *mockSchemes) Create(input *CreateSchemeInput) (*Scheme, error) {
	name := input.Name
	if name == "" {
		name = "Untitled"
	}
	return m.seed(name), nil
}

func (m *mockSchemes) GetByID(id string) (*Scheme, error) {
	s, ok := m.schemes[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return s, nil
}

func (m *mockSchemes) Update(id string, input *UpdateSchemeInput) (*Scheme, error) {
	s, ok := m.schemes[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	s.Name = input.Name
	s.UpdatedAt = time.Now()
	return s, nil
}

func (m *mockSchemes) Delete(id string) error {
	if _, ok := m.schemes[id]; !ok {
		return sql.ErrNoRows
	}
	delete(m.schemes, id)
	return nil
}

// mockInfo implements InformationStore for handler testing.
type mockInfo struct {
	items   map[string]*Information
	nextIdx int
	listErr error
}

func newMockInfo() *mockInfo {
	return &mockInfo{items: make(map[string]*Information)}
}

func (m *mockInfo) WithListError(err error) *mockInfo {
	m.listErr = err
	return m
}

func (m *mockInfo) seed(schemeID, content string) *Information {
	m.nextIdx++
	id := fakeUUID(100 + m.nextIdx)
	info := &Information{ID: id, SchemeID: schemeID, Type: "text", Content: content, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	m.items[id] = info
	return info
}

func (m *mockInfo) ListByScheme(schemeID string) ([]Information, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	out := make([]Information, 0)
	for _, info := range m.items {
		if info.SchemeID == schemeID {
			out = append(out, *info)
		}
	}
	return out, nil
}

func (m *mockInfo) Create(schemeID string, input *CreateInformationInput) (*Information, error) {
	itemType := input.Type
	if itemType == "" {
		itemType = "text"
	}
	info := m.seed(schemeID, input.Content)
	info.Type = itemType
	info.CanvasX = input.CanvasX
	info.CanvasY = input.CanvasY
	return info, nil
}

func (m *mockInfo) Update(id string, input *UpdateInformationInput) (*Information, error) {
	info, ok := m.items[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	if input.Content != nil {
		info.Content = *input.Content
	}
	if input.Type != nil {
		info.Type = *input.Type
	}
	if input.CanvasX != nil {
		info.CanvasX = *input.CanvasX
	}
	if input.CanvasY != nil {
		info.CanvasY = *input.CanvasY
	}
	info.UpdatedAt = time.Now()
	return info, nil
}

func (m *mockInfo) Delete(id string) error {
	if _, ok := m.items[id]; !ok {
		return sql.ErrNoRows
	}
	delete(m.items, id)
	return nil
}

// mockThoughts implements ThoughtStore for handler testing.
type mockThoughts struct {
	thoughts map[string]*Thought
	edges    map[string]*ThoughtEdge
	nextIdx  int
	listErr  error
}

func newMockThoughts() *mockThoughts {
	return &mockThoughts{
		thoughts: make(map[string]*Thought),
		edges:    make(map[string]*ThoughtEdge),
	}
}

func (m *mockThoughts) WithListError(err error) *mockThoughts {
	m.listErr = err
	return m
}

func (m *mockThoughts) seedThought(title string, schemeID *string) *Thought {
	m.nextIdx++
	id := fakeUUID(200 + m.nextIdx)
	t := &Thought{ID: id, SchemeID: schemeID, Title: title, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	m.thoughts[id] = t
	return t
}

func (m *mockThoughts) seedEdge(sourceID, targetID, label string) *ThoughtEdge {
	m.nextIdx++
	id := fakeUUID(300 + m.nextIdx)
	e := &ThoughtEdge{ID: id, SourceID: sourceID, TargetID: targetID, Label: label, CreatedAt: time.Now()}
	m.edges[id] = e
	return e
}

func (m *mockThoughts) List(schemeID string) ([]Thought, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	out := make([]Thought, 0)
	for _, t := range m.thoughts {
		if schemeID == "" || (t.SchemeID != nil && *t.SchemeID == schemeID) {
			out = append(out, *t)
		}
	}
	return out, nil
}

func (m *mockThoughts) Create(input *CreateThoughtInput) (*Thought, error) {
	return m.seedThought(input.Title, input.SchemeID), nil
}

func (m *mockThoughts) GetByID(id string) (*Thought, error) {
	t, ok := m.thoughts[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return t, nil
}

func (m *mockThoughts) Update(id string, input *UpdateThoughtInput) (*Thought, error) {
	t, ok := m.thoughts[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	if input.Title != nil {
		t.Title = *input.Title
	}
	if input.Body != nil {
		t.Body = *input.Body
	}
	t.UpdatedAt = time.Now()
	return t, nil
}

func (m *mockThoughts) Delete(id string) error {
	if _, ok := m.thoughts[id]; !ok {
		return sql.ErrNoRows
	}
	delete(m.thoughts, id)
	return nil
}

func (m *mockThoughts) CreateEdge(sourceID string, input *CreateEdgeInput) (*ThoughtEdge, error) {
	return m.seedEdge(sourceID, input.TargetID, input.Label), nil
}

func (m *mockThoughts) ListEdges(thoughtID string) ([]ThoughtEdge, error) {
	out := make([]ThoughtEdge, 0)
	for _, e := range m.edges {
		if e.SourceID == thoughtID || e.TargetID == thoughtID {
			out = append(out, *e)
		}
	}
	return out, nil
}

func (m *mockThoughts) DeleteEdge(id string) error {
	if _, ok := m.edges[id]; !ok {
		return sql.ErrNoRows
	}
	delete(m.edges, id)
	return nil
}

// mockExport implements ExportStore for handler testing.
type mockExport struct {
	data    map[string]*ExportData
	findErr error
}

func newMockExport() *mockExport {
	return &mockExport{data: make(map[string]*ExportData)}
}

func (m *mockExport) seed(schemeID, name string) {
	m.data[schemeID] = &ExportData{
		Scheme:       Scheme{ID: schemeID, Name: name},
		Information:  []Information{},
		Thoughts:     []Thought{},
		Edges:        []ThoughtEdge{},
		ExportFormat: "vrooli-graph-v1",
	}
}

func (m *mockExport) ExportScheme(schemeID string) (*ExportData, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	d, ok := m.data[schemeID]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return d, nil
}

// mockSuggestions implements SuggestionProvider for handler testing.
type mockSuggestions struct {
	providers   []LLMProvider
	suggestions []Suggestion
	genErr      error
}

func newMockSuggestions() *mockSuggestions {
	return &mockSuggestions{
		providers: []LLMProvider{
			{Name: "ollama", URL: "http://localhost:11434", Active: true, Fallback: false},
			{Name: "openrouter", URL: "https://openrouter.ai/api/v1", Active: false, Fallback: true},
		},
	}
}

func (m *mockSuggestions) WithGenerateError(err error) *mockSuggestions {
	m.genErr = err
	return m
}

func (m *mockSuggestions) WithSuggestions(s []Suggestion) *mockSuggestions {
	m.suggestions = s
	return m
}

func (m *mockSuggestions) GetProviders() []LLMProvider {
	return m.providers
}

func (m *mockSuggestions) GetActiveProvider() (*LLMProvider, error) {
	for i := range m.providers {
		if m.providers[i].Active && !m.providers[i].Fallback {
			return &m.providers[i], nil
		}
	}
	for i := range m.providers {
		if m.providers[i].Active && m.providers[i].Fallback {
			return &m.providers[i], nil
		}
	}
	return nil, fmt.Errorf("no LLM provider available")
}

func (m *mockSuggestions) GenerateSuggestions(schemeID string) ([]Suggestion, *LLMProvider, error) {
	if m.genErr != nil {
		return nil, nil, m.genErr
	}
	provider, err := m.GetActiveProvider()
	if err != nil {
		return nil, nil, err
	}
	if m.suggestions != nil {
		return m.suggestions, provider, nil
	}
	return []Suggestion{}, provider, nil
}

// Compile-time checks that production services satisfy their interfaces
var (
	_ SchemeStore        = (*SchemeService)(nil)
	_ InformationStore   = (*InformationService)(nil)
	_ ThoughtStore       = (*ThoughtService)(nil)
	_ ExportStore        = (*ExportService)(nil)
	_ SuggestionProvider = (*SuggestionService)(nil)
)
