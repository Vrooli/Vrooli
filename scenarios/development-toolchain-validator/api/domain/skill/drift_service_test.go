// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md
// [REQ:REQ-P0-004] Skill Drift Detection
package skill

import (
	"context"
	"testing"
	"time"
)

// driftMockRepo is a test mock for drift detection tests
type driftMockRepo struct {
	connections map[string]*Connection
}

func newDriftMockRepo() *driftMockRepo {
	return &driftMockRepo{
		connections: make(map[string]*Connection),
	}
}

func (m *driftMockRepo) Connect(_ context.Context, input ConnectInput) (*Connection, error) {
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
	return conn, nil
}

func (m *driftMockRepo) GetByID(_ context.Context, id string) (*Connection, error) {
	conn, ok := m.connections[id]
	if !ok {
		return nil, ErrNotFound
	}
	return conn, nil
}

func (m *driftMockRepo) GetByReferenceAndSkill(_ context.Context, _, _ string) (*Connection, error) {
	return nil, ErrNotFound
}

func (m *driftMockRepo) List(_ context.Context, _ ListOptions) ([]*Connection, error) {
	return nil, nil
}

func (m *driftMockRepo) Update(_ context.Context, _ string, _ UpdateInput) (*Connection, error) {
	return nil, nil
}

func (m *driftMockRepo) Disconnect(_ context.Context, _ string) error {
	return nil
}

func (m *driftMockRepo) DisconnectByReferenceAndSkill(_ context.Context, _, _ string) error {
	return nil
}

// Tests for CheckDrift - [REQ:REQ-P0-004] Version/hash comparison

func TestService_CheckDrift_NoDrift(t *testing.T) {
	repo := newDriftMockRepo()
	repo.connections["conn-123"] = &Connection{
		ID:               "conn-123",
		ReferenceID:      "ref-123",
		SkillID:          "api-steer",
		SkillVersion:     "v1.0",
		SkillContentHash: "hash123",
	}

	svc := NewService(repo)

	status, err := svc.CheckDrift(context.Background(), "conn-123", "v1.0", "hash123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.HasDrifted {
		t.Error("expected no drift")
	}
	if status.VersionChanged {
		t.Error("expected version unchanged")
	}
	if status.ContentChanged {
		t.Error("expected content unchanged")
	}
}

func TestService_CheckDrift_VersionChanged(t *testing.T) {
	repo := newDriftMockRepo()
	repo.connections["conn-123"] = &Connection{
		ID:               "conn-123",
		ReferenceID:      "ref-123",
		SkillID:          "api-steer",
		SkillVersion:     "v1.0",
		SkillContentHash: "hash123",
	}

	svc := NewService(repo)

	status, err := svc.CheckDrift(context.Background(), "conn-123", "v2.0", "hash123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !status.HasDrifted {
		t.Error("expected drift detected")
	}
	if !status.VersionChanged {
		t.Error("expected version changed")
	}
	if status.ContentChanged {
		t.Error("expected content unchanged")
	}
}

func TestService_CheckDrift_ContentChanged(t *testing.T) {
	repo := newDriftMockRepo()
	repo.connections["conn-123"] = &Connection{
		ID:               "conn-123",
		ReferenceID:      "ref-123",
		SkillID:          "api-steer",
		SkillVersion:     "v1.0",
		SkillContentHash: "hash123",
	}

	svc := NewService(repo)

	status, err := svc.CheckDrift(context.Background(), "conn-123", "v1.0", "hash456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !status.HasDrifted {
		t.Error("expected drift detected")
	}
	if status.VersionChanged {
		t.Error("expected version unchanged")
	}
	if !status.ContentChanged {
		t.Error("expected content changed")
	}
}

func TestService_CheckDrift_BothChanged(t *testing.T) {
	repo := newDriftMockRepo()
	repo.connections["conn-123"] = &Connection{
		ID:               "conn-123",
		ReferenceID:      "ref-123",
		SkillID:          "api-steer",
		SkillVersion:     "v1.0",
		SkillContentHash: "hash123",
	}

	svc := NewService(repo)

	status, err := svc.CheckDrift(context.Background(), "conn-123", "v2.0", "hash456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !status.HasDrifted {
		t.Error("expected drift detected")
	}
	if !status.VersionChanged {
		t.Error("expected version changed")
	}
	if !status.ContentChanged {
		t.Error("expected content changed")
	}
}

func TestService_CheckDrift_NotFound(t *testing.T) {
	repo := newDriftMockRepo()
	svc := NewService(repo)

	_, err := svc.CheckDrift(context.Background(), "nonexistent", "v1.0", "hash123")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// Edge cases for drift detection

func TestService_CheckDrift_EmptyVersion(t *testing.T) {
	repo := newDriftMockRepo()
	repo.connections["conn-123"] = &Connection{
		ID:               "conn-123",
		ReferenceID:      "ref-123",
		SkillID:          "api-steer",
		SkillVersion:     "",
		SkillContentHash: "hash123",
	}

	svc := NewService(repo)

	// Comparing empty stored version against new version
	status, err := svc.CheckDrift(context.Background(), "conn-123", "v1.0", "hash123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !status.HasDrifted {
		t.Error("expected drift when version changes from empty")
	}
	if !status.VersionChanged {
		t.Error("expected version changed flag")
	}
}

func TestService_CheckDrift_EmptyHash(t *testing.T) {
	repo := newDriftMockRepo()
	repo.connections["conn-123"] = &Connection{
		ID:               "conn-123",
		ReferenceID:      "ref-123",
		SkillID:          "api-steer",
		SkillVersion:     "v1.0",
		SkillContentHash: "",
	}

	svc := NewService(repo)

	// Comparing empty stored hash against new hash
	status, err := svc.CheckDrift(context.Background(), "conn-123", "v1.0", "hash123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !status.HasDrifted {
		t.Error("expected drift when hash changes from empty")
	}
	if !status.ContentChanged {
		t.Error("expected content changed flag")
	}
}
