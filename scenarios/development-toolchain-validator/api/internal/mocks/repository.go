// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md#mock-organization
// Package mocks provides mock implementations for testing.
package mocks

import (
	"context"
	"development-toolchain-validator/domain/reference"
	"development-toolchain-validator/domain/skill"
	"sync"
)

// MockRepository is a configurable mock implementation of reference.Repository.
// It uses a builder pattern to configure expected behaviors and return values.
type MockRepository struct {
	mu sync.RWMutex

	// Storage for in-memory state
	references map[string]*reference.Reference
	slugIndex  map[string]string // slug -> id mapping

	// Error injection
	createErr    error
	getByIDErr   error
	getBySlugErr error
	listErr      error
	updateErr    error
	deleteErr    error

	// Call tracking
	createCalls    []reference.CreateInput
	getByIDCalls   []string
	getBySlugCalls []string
	listCalls      []reference.ListOptions
	updateCalls    []updateCall
	deleteCalls    []string
}

type updateCall struct {
	ID    string
	Input reference.UpdateInput
}

// NewMockRepository creates a new mock repository with empty state.
func NewMockRepository() *MockRepository {
	return &MockRepository{
		references: make(map[string]*reference.Reference),
		slugIndex:  make(map[string]string),
	}
}

// Builder methods for configuring the mock

// WithReference adds a reference to the mock's internal storage.
func (m *MockRepository) WithReference(ref *reference.Reference) *MockRepository {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.references[ref.ID] = ref
	m.slugIndex[ref.Slug] = ref.ID
	return m
}

// WithCreateError configures Create to return an error.
func (m *MockRepository) WithCreateError(err error) *MockRepository {
	m.createErr = err
	return m
}

// WithGetByIDError configures GetByID to return an error.
func (m *MockRepository) WithGetByIDError(err error) *MockRepository {
	m.getByIDErr = err
	return m
}

// WithGetBySlugError configures GetBySlug to return an error.
func (m *MockRepository) WithGetBySlugError(err error) *MockRepository {
	m.getBySlugErr = err
	return m
}

// WithListError configures List to return an error.
func (m *MockRepository) WithListError(err error) *MockRepository {
	m.listErr = err
	return m
}

// WithUpdateError configures Update to return an error.
func (m *MockRepository) WithUpdateError(err error) *MockRepository {
	m.updateErr = err
	return m
}

// WithDeleteError configures Delete to return an error.
func (m *MockRepository) WithDeleteError(err error) *MockRepository {
	m.deleteErr = err
	return m
}

// Repository interface implementation

// Create stores a new reference scenario.
func (m *MockRepository) Create(_ context.Context, input reference.CreateInput) (*reference.Reference, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.createCalls = append(m.createCalls, input)

	if m.createErr != nil {
		return nil, m.createErr
	}

	// Create a new reference with generated ID
	ref := &reference.Reference{
		ID:          "mock-generated-id",
		Slug:        input.Slug,
		Name:        input.Name,
		Template:    input.Template,
		Path:        input.Path,
		Description: input.Description,
	}
	m.references[ref.ID] = ref
	m.slugIndex[ref.Slug] = ref.ID

	return ref, nil
}

// GetByID retrieves a reference by its UUID.
func (m *MockRepository) GetByID(_ context.Context, id string) (*reference.Reference, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.getByIDCalls = append(m.getByIDCalls, id)

	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}

	ref, exists := m.references[id]
	if !exists {
		return nil, reference.ErrNotFound
	}
	return ref, nil
}

// GetBySlug retrieves a reference by its unique slug.
func (m *MockRepository) GetBySlug(_ context.Context, slug string) (*reference.Reference, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.getBySlugCalls = append(m.getBySlugCalls, slug)

	if m.getBySlugErr != nil {
		return nil, m.getBySlugErr
	}

	id, exists := m.slugIndex[slug]
	if !exists {
		return nil, reference.ErrNotFound
	}
	return m.references[id], nil
}

// List retrieves references with optional filtering.
func (m *MockRepository) List(_ context.Context, opts reference.ListOptions) ([]*reference.Reference, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.listCalls = append(m.listCalls, opts)

	if m.listErr != nil {
		return nil, m.listErr
	}

	var result []*reference.Reference
	for _, ref := range m.references {
		if opts.Template != "" && ref.Template != opts.Template {
			continue
		}
		result = append(result, ref)
	}

	// Apply limit
	if opts.Limit > 0 && len(result) > opts.Limit {
		result = result[:opts.Limit]
	}

	return result, nil
}

// Update modifies an existing reference.
func (m *MockRepository) Update(_ context.Context, id string, input reference.UpdateInput) (*reference.Reference, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.updateCalls = append(m.updateCalls, updateCall{ID: id, Input: input})

	if m.updateErr != nil {
		return nil, m.updateErr
	}

	ref, exists := m.references[id]
	if !exists {
		return nil, reference.ErrNotFound
	}

	// Apply updates
	if input.Name != nil {
		ref.Name = *input.Name
	}
	if input.Template != nil {
		ref.Template = *input.Template
	}
	if input.Path != nil {
		ref.Path = *input.Path
	}
	if input.Description != nil {
		ref.Description = *input.Description
	}

	return ref, nil
}

// Delete removes a reference by ID.
func (m *MockRepository) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.deleteCalls = append(m.deleteCalls, id)

	if m.deleteErr != nil {
		return m.deleteErr
	}

	ref, exists := m.references[id]
	if !exists {
		return reference.ErrNotFound
	}

	delete(m.slugIndex, ref.Slug)
	delete(m.references, id)
	return nil
}

// Assertion methods for verifying mock interactions

// CreateCallCount returns the number of times Create was called.
func (m *MockRepository) CreateCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.createCalls)
}

// CreateCallArg returns the input argument for the nth Create call.
func (m *MockRepository) CreateCallArg(n int) reference.CreateInput {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.createCalls[n]
}

// GetByIDCallCount returns the number of times GetByID was called.
func (m *MockRepository) GetByIDCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.getByIDCalls)
}

// DeleteCallCount returns the number of times Delete was called.
func (m *MockRepository) DeleteCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.deleteCalls)
}

// UpdateCallCount returns the number of times Update was called.
func (m *MockRepository) UpdateCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.updateCalls)
}

// Reset clears all call tracking and internal state.
func (m *MockRepository) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.references = make(map[string]*reference.Reference)
	m.slugIndex = make(map[string]string)
	m.createCalls = nil
	m.getByIDCalls = nil
	m.getBySlugCalls = nil
	m.listCalls = nil
	m.updateCalls = nil
	m.deleteCalls = nil
}

// Ensure MockRepository implements Repository interface at compile time.
var _ reference.Repository = (*MockRepository)(nil)

// MockSkillRepository is a configurable mock implementation of skill.Repository.
type MockSkillRepository struct {
	mu sync.RWMutex

	// Storage for in-memory state
	connections   map[string]*skill.Connection
	refSkillIndex map[string]string // "refID:skillID" -> connection id mapping

	// Error injection
	connectErr                       error
	getByIDErr                       error
	getByReferenceAndSkillErr        error
	listErr                          error
	updateErr                        error
	disconnectErr                    error
	disconnectByReferenceAndSkillErr error

	// Call tracking
	connectCalls                       []skill.ConnectInput
	getByIDCalls                       []string
	getByReferenceAndSkillCalls        [][2]string
	listCalls                          []skill.ListOptions
	updateCalls                        []skillUpdateCall
	disconnectCalls                    []string
	disconnectByReferenceAndSkillCalls [][2]string
}

type skillUpdateCall struct {
	ID    string
	Input skill.UpdateInput
}

// NewMockSkillRepository creates a new mock skill repository with empty state.
func NewMockSkillRepository() *MockSkillRepository {
	return &MockSkillRepository{
		connections:   make(map[string]*skill.Connection),
		refSkillIndex: make(map[string]string),
	}
}

// Builder methods for configuring the mock

// WithConnection adds a connection to the mock's internal storage.
func (m *MockSkillRepository) WithConnection(conn *skill.Connection) *MockSkillRepository {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connections[conn.ID] = conn
	key := conn.ReferenceID + ":" + conn.SkillID
	m.refSkillIndex[key] = conn.ID
	return m
}

// WithConnectError configures Connect to return an error.
func (m *MockSkillRepository) WithConnectError(err error) *MockSkillRepository {
	m.connectErr = err
	return m
}

// WithGetByIDError configures GetByID to return an error.
func (m *MockSkillRepository) WithGetByIDError(err error) *MockSkillRepository {
	m.getByIDErr = err
	return m
}

// WithGetByReferenceAndSkillError configures GetByReferenceAndSkill to return an error.
func (m *MockSkillRepository) WithGetByReferenceAndSkillError(err error) *MockSkillRepository {
	m.getByReferenceAndSkillErr = err
	return m
}

// WithListError configures List to return an error.
func (m *MockSkillRepository) WithListError(err error) *MockSkillRepository {
	m.listErr = err
	return m
}

// WithUpdateError configures Update to return an error.
func (m *MockSkillRepository) WithUpdateError(err error) *MockSkillRepository {
	m.updateErr = err
	return m
}

// WithDisconnectError configures Disconnect to return an error.
func (m *MockSkillRepository) WithDisconnectError(err error) *MockSkillRepository {
	m.disconnectErr = err
	return m
}

// WithDisconnectByReferenceAndSkillError configures DisconnectByReferenceAndSkill to return an error.
func (m *MockSkillRepository) WithDisconnectByReferenceAndSkillError(err error) *MockSkillRepository {
	m.disconnectByReferenceAndSkillErr = err
	return m
}

// Repository interface implementation

// Connect creates a new skill-reference connection.
func (m *MockSkillRepository) Connect(_ context.Context, input skill.ConnectInput) (*skill.Connection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.connectCalls = append(m.connectCalls, input)

	if m.connectErr != nil {
		return nil, m.connectErr
	}

	conn := &skill.Connection{
		ID:               "mock-connection-id",
		ReferenceID:      input.ReferenceID,
		SkillID:          input.SkillID,
		SkillVersion:     input.SkillVersion,
		SkillContentHash: input.SkillContentHash,
	}
	m.connections[conn.ID] = conn
	key := conn.ReferenceID + ":" + conn.SkillID
	m.refSkillIndex[key] = conn.ID

	return conn, nil
}

// GetByID retrieves a connection by its UUID.
func (m *MockSkillRepository) GetByID(_ context.Context, id string) (*skill.Connection, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.getByIDCalls = append(m.getByIDCalls, id)

	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}

	conn, exists := m.connections[id]
	if !exists {
		return nil, skill.ErrNotFound
	}
	return conn, nil
}

// GetByReferenceAndSkill retrieves a connection by reference ID and skill ID.
func (m *MockSkillRepository) GetByReferenceAndSkill(_ context.Context, referenceID, skillID string) (*skill.Connection, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.getByReferenceAndSkillCalls = append(m.getByReferenceAndSkillCalls, [2]string{referenceID, skillID})

	if m.getByReferenceAndSkillErr != nil {
		return nil, m.getByReferenceAndSkillErr
	}

	key := referenceID + ":" + skillID
	id, exists := m.refSkillIndex[key]
	if !exists {
		return nil, skill.ErrNotFound
	}
	return m.connections[id], nil
}

// List retrieves connections with optional filtering.
func (m *MockSkillRepository) List(_ context.Context, opts skill.ListOptions) ([]*skill.Connection, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.listCalls = append(m.listCalls, opts)

	if m.listErr != nil {
		return nil, m.listErr
	}

	var result []*skill.Connection
	for _, conn := range m.connections {
		if opts.ReferenceID != "" && conn.ReferenceID != opts.ReferenceID {
			continue
		}
		if opts.SkillID != "" && conn.SkillID != opts.SkillID {
			continue
		}
		result = append(result, conn)
	}

	if opts.Limit > 0 && len(result) > opts.Limit {
		result = result[:opts.Limit]
	}

	return result, nil
}

// Update modifies an existing connection.
func (m *MockSkillRepository) Update(_ context.Context, id string, input skill.UpdateInput) (*skill.Connection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.updateCalls = append(m.updateCalls, skillUpdateCall{ID: id, Input: input})

	if m.updateErr != nil {
		return nil, m.updateErr
	}

	conn, exists := m.connections[id]
	if !exists {
		return nil, skill.ErrNotFound
	}

	if input.SkillVersion != nil {
		conn.SkillVersion = *input.SkillVersion
	}
	if input.SkillContentHash != nil {
		conn.SkillContentHash = *input.SkillContentHash
	}

	return conn, nil
}

// Disconnect removes a skill-reference connection by ID.
func (m *MockSkillRepository) Disconnect(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.disconnectCalls = append(m.disconnectCalls, id)

	if m.disconnectErr != nil {
		return m.disconnectErr
	}

	conn, exists := m.connections[id]
	if !exists {
		return skill.ErrNotFound
	}

	key := conn.ReferenceID + ":" + conn.SkillID
	delete(m.refSkillIndex, key)
	delete(m.connections, id)
	return nil
}

// DisconnectByReferenceAndSkill removes a connection by reference ID and skill ID.
func (m *MockSkillRepository) DisconnectByReferenceAndSkill(_ context.Context, referenceID, skillID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.disconnectByReferenceAndSkillCalls = append(m.disconnectByReferenceAndSkillCalls, [2]string{referenceID, skillID})

	if m.disconnectByReferenceAndSkillErr != nil {
		return m.disconnectByReferenceAndSkillErr
	}

	key := referenceID + ":" + skillID
	id, exists := m.refSkillIndex[key]
	if !exists {
		return skill.ErrNotFound
	}

	delete(m.refSkillIndex, key)
	delete(m.connections, id)
	return nil
}

// Assertion methods for verifying mock interactions

// ConnectCallCount returns the number of times Connect was called.
func (m *MockSkillRepository) ConnectCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.connectCalls)
}

// DisconnectCallCount returns the number of times Disconnect was called.
func (m *MockSkillRepository) DisconnectCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.disconnectCalls)
}

// UpdateCallCount returns the number of times Update was called.
func (m *MockSkillRepository) UpdateCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.updateCalls)
}

// ResetSkill clears all call tracking and internal state.
func (m *MockSkillRepository) ResetSkill() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connections = make(map[string]*skill.Connection)
	m.refSkillIndex = make(map[string]string)
	m.connectCalls = nil
	m.getByIDCalls = nil
	m.getByReferenceAndSkillCalls = nil
	m.listCalls = nil
	m.updateCalls = nil
	m.disconnectCalls = nil
	m.disconnectByReferenceAndSkillCalls = nil
}

// Ensure MockSkillRepository implements skill.Repository interface at compile time.
var _ skill.Repository = (*MockSkillRepository)(nil)
