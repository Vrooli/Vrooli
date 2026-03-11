// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md
// [REQ:REQ-P0-003] Prompt-Manager Skill Connection Store
package skill

import (
	"context"
	"errors"
	"testing"
	"time"
)

// mockRepo is a simple test mock for the Repository interface
type mockRepo struct {
	connections   map[string]*Connection
	refSkillIndex map[string]string

	connectErr                error
	getByIDErr                error
	getByReferenceAndSkillErr error
	disconnectErr             error
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		connections:   make(map[string]*Connection),
		refSkillIndex: make(map[string]string),
	}
}

func (m *mockRepo) Connect(_ context.Context, input ConnectInput) (*Connection, error) {
	if m.connectErr != nil {
		return nil, m.connectErr
	}
	conn := &Connection{
		ID:               "test-connection-id",
		ReferenceID:      input.ReferenceID,
		SkillID:          input.SkillID,
		SkillVersion:     input.SkillVersion,
		SkillContentHash: input.SkillContentHash,
		ConnectedAt:      time.Now(),
		UpdatedAt:        time.Now(),
	}
	m.connections[conn.ID] = conn
	m.refSkillIndex[input.ReferenceID+":"+input.SkillID] = conn.ID
	return conn, nil
}

func (m *mockRepo) GetByID(_ context.Context, id string) (*Connection, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	conn, ok := m.connections[id]
	if !ok {
		return nil, ErrNotFound
	}
	return conn, nil
}

func (m *mockRepo) GetByReferenceAndSkill(_ context.Context, refID, skillID string) (*Connection, error) {
	if m.getByReferenceAndSkillErr != nil {
		return nil, m.getByReferenceAndSkillErr
	}
	key := refID + ":" + skillID
	connID, ok := m.refSkillIndex[key]
	if !ok {
		return nil, ErrNotFound
	}
	return m.connections[connID], nil
}

func (m *mockRepo) List(_ context.Context, opts ListOptions) ([]*Connection, error) {
	var result []*Connection
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

func (m *mockRepo) Update(_ context.Context, id string, input UpdateInput) (*Connection, error) {
	conn, ok := m.connections[id]
	if !ok {
		return nil, ErrNotFound
	}
	if input.SkillVersion != nil {
		conn.SkillVersion = *input.SkillVersion
	}
	if input.SkillContentHash != nil {
		conn.SkillContentHash = *input.SkillContentHash
	}
	conn.UpdatedAt = time.Now()
	return conn, nil
}

func (m *mockRepo) Disconnect(_ context.Context, id string) error {
	if m.disconnectErr != nil {
		return m.disconnectErr
	}
	conn, ok := m.connections[id]
	if !ok {
		return ErrNotFound
	}
	delete(m.refSkillIndex, conn.ReferenceID+":"+conn.SkillID)
	delete(m.connections, id)
	return nil
}

func (m *mockRepo) DisconnectByReferenceAndSkill(_ context.Context, refID, skillID string) error {
	key := refID + ":" + skillID
	connID, ok := m.refSkillIndex[key]
	if !ok {
		return ErrNotFound
	}
	delete(m.connections, connID)
	delete(m.refSkillIndex, key)
	return nil
}

// Tests for Connect

func TestService_Connect_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	input := ConnectInput{
		ReferenceID:      "ref-123",
		SkillID:          "api-steer",
		SkillVersion:     "v1.0",
		SkillContentHash: "abc123",
	}

	conn, err := svc.Connect(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn.SkillID != "api-steer" {
		t.Errorf("expected skill ID 'api-steer', got '%s'", conn.SkillID)
	}
	if conn.ReferenceID != "ref-123" {
		t.Errorf("expected reference ID 'ref-123', got '%s'", conn.ReferenceID)
	}
}

func TestService_Connect_InvalidSkillID(t *testing.T) {
	tests := []struct {
		name    string
		skillID string
	}{
		{"empty", ""},
		{"single char", "a"},
		{"starts with number", "123-skill"},
		{"uppercase", "API-STEER"},
		{"special chars", "api_steer"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newMockRepo()
			svc := NewService(repo)

			input := ConnectInput{
				ReferenceID: "ref-123",
				SkillID:     tc.skillID,
			}

			_, err := svc.Connect(context.Background(), input)
			if !errors.Is(err, ErrInvalidSkillID) {
				t.Errorf("expected ErrInvalidSkillID, got %v", err)
			}
		})
	}
}

func TestService_Connect_InvalidReferenceID(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	input := ConnectInput{
		ReferenceID: "",
		SkillID:     "api-steer",
	}

	_, err := svc.Connect(context.Background(), input)
	if !errors.Is(err, ErrInvalidReferenceID) {
		t.Errorf("expected ErrInvalidReferenceID, got %v", err)
	}
}

func TestService_Connect_AlreadyExists(t *testing.T) {
	repo := newMockRepo()
	// Pre-populate with existing connection
	repo.connections["existing-id"] = &Connection{
		ID:          "existing-id",
		ReferenceID: "ref-123",
		SkillID:     "api-steer",
	}
	repo.refSkillIndex["ref-123:api-steer"] = "existing-id"

	svc := NewService(repo)

	input := ConnectInput{
		ReferenceID: "ref-123",
		SkillID:     "api-steer",
	}

	_, err := svc.Connect(context.Background(), input)
	if !errors.Is(err, ErrConnectionExists) {
		t.Errorf("expected ErrConnectionExists, got %v", err)
	}
}

// Tests for GetByID

func TestService_GetByID_Success(t *testing.T) {
	repo := newMockRepo()
	repo.connections["conn-123"] = &Connection{
		ID:          "conn-123",
		ReferenceID: "ref-123",
		SkillID:     "api-steer",
	}

	svc := NewService(repo)

	conn, err := svc.GetByID(context.Background(), "conn-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn.ID != "conn-123" {
		t.Errorf("expected ID 'conn-123', got '%s'", conn.ID)
	}
}

func TestService_GetByID_NotFound(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	_, err := svc.GetByID(context.Background(), "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// Tests for Disconnect

func TestService_Disconnect_Success(t *testing.T) {
	repo := newMockRepo()
	repo.connections["conn-123"] = &Connection{
		ID:          "conn-123",
		ReferenceID: "ref-123",
		SkillID:     "api-steer",
	}
	repo.refSkillIndex["ref-123:api-steer"] = "conn-123"

	svc := NewService(repo)

	err := svc.Disconnect(context.Background(), "conn-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify deleted
	if _, exists := repo.connections["conn-123"]; exists {
		t.Error("expected connection to be deleted")
	}
}

func TestService_Disconnect_NotFound(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	err := svc.Disconnect(context.Background(), "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// Tests for ValidateConnect (dry-run)

func TestService_ValidateConnect_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	input := ConnectInput{
		ReferenceID: "ref-123",
		SkillID:     "api-steer",
	}

	err := svc.ValidateConnect(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify no connection was created
	if len(repo.connections) != 0 {
		t.Error("expected no connections created during validation")
	}
}

func TestService_ValidateConnect_InvalidInput(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	input := ConnectInput{
		ReferenceID: "",
		SkillID:     "api-steer",
	}

	err := svc.ValidateConnect(context.Background(), input)
	if !errors.Is(err, ErrInvalidReferenceID) {
		t.Errorf("expected ErrInvalidReferenceID, got %v", err)
	}
}

// Tests for List

func TestService_List_FilterByReference(t *testing.T) {
	repo := newMockRepo()
	repo.connections["conn-1"] = &Connection{ID: "conn-1", ReferenceID: "ref-123", SkillID: "api-steer"}
	repo.connections["conn-2"] = &Connection{ID: "conn-2", ReferenceID: "ref-123", SkillID: "cli-steer"}
	repo.connections["conn-3"] = &Connection{ID: "conn-3", ReferenceID: "ref-456", SkillID: "api-steer"}

	svc := NewService(repo)

	conns, err := svc.List(context.Background(), ListOptions{ReferenceID: "ref-123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conns) != 2 {
		t.Errorf("expected 2 connections, got %d", len(conns))
	}
}

func TestService_List_FilterBySkill(t *testing.T) {
	repo := newMockRepo()
	repo.connections["conn-1"] = &Connection{ID: "conn-1", ReferenceID: "ref-123", SkillID: "api-steer"}
	repo.connections["conn-2"] = &Connection{ID: "conn-2", ReferenceID: "ref-456", SkillID: "api-steer"}
	repo.connections["conn-3"] = &Connection{ID: "conn-3", ReferenceID: "ref-123", SkillID: "cli-steer"}

	svc := NewService(repo)

	conns, err := svc.List(context.Background(), ListOptions{SkillID: "api-steer"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conns) != 2 {
		t.Errorf("expected 2 connections, got %d", len(conns))
	}
}

// Tests for Update

func TestService_Update_Success(t *testing.T) {
	repo := newMockRepo()
	repo.connections["conn-123"] = &Connection{
		ID:               "conn-123",
		ReferenceID:      "ref-123",
		SkillID:          "api-steer",
		SkillVersion:     "v1.0",
		SkillContentHash: "hash123",
	}

	svc := NewService(repo)

	newVersion := "v2.0"
	newHash := "hash456"
	input := UpdateInput{
		SkillVersion:     &newVersion,
		SkillContentHash: &newHash,
	}

	conn, err := svc.Update(context.Background(), "conn-123", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn.SkillVersion != "v2.0" {
		t.Errorf("expected version 'v2.0', got '%s'", conn.SkillVersion)
	}
	if conn.SkillContentHash != "hash456" {
		t.Errorf("expected hash 'hash456', got '%s'", conn.SkillContentHash)
	}
}

func TestService_Update_NotFound(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)

	newVersion := "v2.0"
	input := UpdateInput{
		SkillVersion: &newVersion,
	}

	_, err := svc.Update(context.Background(), "nonexistent", input)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// Replay safety tests - [REQ:REQ-P0-003]

func TestService_Connect_ReplayReturnsConflict(t *testing.T) {
	// Connect with same input twice should return ErrConnectionExists
	repo := newMockRepo()
	svc := NewService(repo)

	input := ConnectInput{
		ReferenceID: "ref-123",
		SkillID:     "api-steer",
	}

	// First connect succeeds
	_, err := svc.Connect(context.Background(), input)
	if err != nil {
		t.Fatalf("first connect failed: %v", err)
	}

	// Second connect (replay) returns conflict
	_, err = svc.Connect(context.Background(), input)
	if !errors.Is(err, ErrConnectionExists) {
		t.Errorf("expected ErrConnectionExists on replay, got %v", err)
	}
}

func TestService_Disconnect_ReplayReturnsSafe(t *testing.T) {
	// Disconnect twice should return ErrNotFound on second call, not panic
	repo := newMockRepo()
	repo.connections["conn-123"] = &Connection{
		ID:          "conn-123",
		ReferenceID: "ref-123",
		SkillID:     "api-steer",
	}
	repo.refSkillIndex["ref-123:api-steer"] = "conn-123"

	svc := NewService(repo)

	// First disconnect succeeds
	err := svc.Disconnect(context.Background(), "conn-123")
	if err != nil {
		t.Fatalf("first disconnect failed: %v", err)
	}

	// Second disconnect (replay) returns not found
	err = svc.Disconnect(context.Background(), "conn-123")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound on replay, got %v", err)
	}
}

func TestService_ValidateConnect_NoSideEffects(t *testing.T) {
	// ValidateConnect should not create any connections
	repo := newMockRepo()
	svc := NewService(repo)

	input := ConnectInput{
		ReferenceID: "ref-123",
		SkillID:     "api-steer",
	}

	// Validate multiple times
	for i := 0; i < 3; i++ {
		err := svc.ValidateConnect(context.Background(), input)
		if err != nil {
			t.Fatalf("validate failed on iteration %d: %v", i, err)
		}
	}

	// Verify no connections created
	if len(repo.connections) != 0 {
		t.Errorf("expected 0 connections, got %d", len(repo.connections))
	}
}
