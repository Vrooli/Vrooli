package skills

import (
	"context"
	"errors"
	"prompt-manager/store"
	"testing"
)

// MockSkillStore implements store.SkillStore for testing StoreAdapter.
// Uses builder pattern for clean test setup.
type MockSkillStore struct {
	skills       map[string]*store.Skill // id -> skill
	contentPaths map[string]string       // pack/id -> path
	listErr      error
	getErr       error
	createErr    error
	updateErr    error
	deleteErr    error
	// UpdateCalls tracks which skill IDs had Update() called on them.
	// This is the testing seam for verifying update behavior.
	UpdateCalls []string
}

func NewMockSkillStore() *MockSkillStore {
	return &MockSkillStore{
		skills:       make(map[string]*store.Skill),
		contentPaths: make(map[string]string),
	}
}

// WithSkill adds a skill to the mock store (builder pattern)
func (m *MockSkillStore) WithSkill(s *store.Skill) *MockSkillStore {
	m.skills[s.ID] = s
	return m
}

// WithGetError configures Get to return an error
func (m *MockSkillStore) WithGetError(err error) *MockSkillStore {
	m.getErr = err
	return m
}

func (m *MockSkillStore) List(ctx context.Context) ([]store.Skill, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var result []store.Skill
	for _, s := range m.skills {
		result = append(result, *s)
	}
	return result, nil
}

func (m *MockSkillStore) Get(ctx context.Context, id string) (*store.Skill, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if s, ok := m.skills[id]; ok {
		return s, nil
	}
	return nil, errors.New("skill not found: " + id)
}

func (m *MockSkillStore) GetWithContent(ctx context.Context, id string) (*store.Skill, string, error) {
	s, err := m.Get(ctx, id)
	if err != nil {
		return nil, "", err
	}
	// Return empty content - actual content comes from ContentIO
	return s, "", nil
}

func (m *MockSkillStore) Create(ctx context.Context, pack string, skill *store.Skill, content string) error {
	if m.createErr != nil {
		return m.createErr
	}
	skill.Pack = pack
	m.skills[skill.ID] = skill
	return nil
}

func (m *MockSkillStore) Update(ctx context.Context, id string, skill *store.Skill, content *string) error {
	// Track the update call for test assertions
	m.UpdateCalls = append(m.UpdateCalls, id)

	if m.updateErr != nil {
		return m.updateErr
	}
	if existing, ok := m.skills[id]; ok {
		// Update fields
		existing.Name = skill.Name
		existing.Description = skill.Description
		existing.Modes = skill.Modes
		existing.Tags = skill.Tags
		return nil
	}
	return errors.New("skill not found: " + id)
}

func (m *MockSkillStore) Delete(ctx context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.skills, id)
	return nil
}

func (m *MockSkillStore) GetVersionHistory(ctx context.Context, id string) ([]store.HistoryEntry, error) {
	return []store.HistoryEntry{}, nil
}

func (m *MockSkillStore) ContentPath(pack, skillID string) string {
	return "/test/store/skills/packs/" + pack + "/" + skillID + "/SKILL.md"
}

func (m *MockSkillStore) Rename(ctx context.Context, oldID, newID string) (*store.Skill, error) {
	if s, ok := m.skills[oldID]; ok {
		delete(m.skills, oldID)
		s.ID = newID
		m.skills[newID] = s
		return s, nil
	}
	return nil, errors.New("skill not found: " + oldID)
}

// MockContentIO implements store.ContentIO for testing.
// This is the testing seam for filesystem operations.
type MockContentIO struct {
	files    map[string]string // path -> content
	readErr  error
	writeErr error
}

func NewMockContentIO() *MockContentIO {
	return &MockContentIO{
		files: make(map[string]string),
	}
}

// WithFile pre-populates a file (builder pattern)
func (m *MockContentIO) WithFile(path, content string) *MockContentIO {
	m.files[path] = content
	return m
}

// WithReadError configures ReadContent to return an error
func (m *MockContentIO) WithReadError(err error) *MockContentIO {
	m.readErr = err
	return m
}

// WithWriteError configures WriteContent to return an error
func (m *MockContentIO) WithWriteError(err error) *MockContentIO {
	m.writeErr = err
	return m
}

func (m *MockContentIO) ReadContent(path string) (string, error) {
	if m.readErr != nil {
		return "", m.readErr
	}
	if content, ok := m.files[path]; ok {
		return content, nil
	}
	return "", errors.New("file not found: " + path)
}

func (m *MockContentIO) WriteContent(path, content string) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	m.files[path] = content
	return nil
}

// HasFile checks if a file exists in the mock
func (m *MockContentIO) HasFile(path string) bool {
	_, ok := m.files[path]
	return ok
}

// GetFile returns content at path (for assertions)
func (m *MockContentIO) GetFile(path string) string {
	return m.files[path]
}

// --- Tests ---

func TestStoreAdapter_SaveContent_SkillExists(t *testing.T) {
	// ARRANGE: Skill exists in store
	skillStore := NewMockSkillStore().WithSkill(&store.Skill{
		ID:   "existing-skill",
		Name: "Existing",
		Pack: "local",
	})
	contentIO := NewMockContentIO()
	adapter := NewStoreAdapter(skillStore, contentIO)

	// ACT: Save content for existing skill
	err := adapter.SaveContent("local", "existing-skill.md", "new content")
	// ASSERT: Update should be called (content passed to store.Update)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// Content is passed to store.Update, not written directly to filesystem
	if contentIO.HasFile("/test/store/skills/packs/local/existing-skill/SKILL.md") {
		t.Error("content should NOT be written to disk for existing skill")
	}
}

func TestStoreAdapter_SaveContent_SkillDoesNotExist_WritesToDisk(t *testing.T) {
	// ARRANGE: Skill does NOT exist in store (move operation scenario)
	skillStore := NewMockSkillStore() // Empty - no skills
	contentIO := NewMockContentIO()
	adapter := NewStoreAdapter(skillStore, contentIO)

	// ACT: Save content for non-existent skill (pre-write during move)
	err := adapter.SaveContent("core", "new-skill.md", "skill content here")
	// ASSERT: Content should be written directly to disk
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expectedPath := "/test/store/skills/packs/core/new-skill/SKILL.md"
	if !contentIO.HasFile(expectedPath) {
		t.Error("content should be written to disk when skill doesn't exist")
	}
	if contentIO.GetFile(expectedPath) != "skill content here" {
		t.Errorf("expected content 'skill content here', got %q", contentIO.GetFile(expectedPath))
	}
}

func TestStoreAdapter_GetContent_SkillExists(t *testing.T) {
	// ARRANGE: Skill exists in store
	skillStore := NewMockSkillStore().WithSkill(&store.Skill{
		ID:   "my-skill",
		Name: "My Skill",
		Pack: "local",
	})
	contentIO := NewMockContentIO()
	adapter := NewStoreAdapter(skillStore, contentIO)

	// ACT: Get content for existing skill
	content, err := adapter.GetContent("local", "my-skill.md")
	// ASSERT: Should return content from store (empty in this mock)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if content != "" {
		t.Errorf("expected empty content from mock, got %q", content)
	}
}

func TestStoreAdapter_GetContent_SkillDoesNotExist_ReadsFromDisk(t *testing.T) {
	// ARRANGE: Skill does NOT exist in store, but content was pre-written to disk
	skillStore := NewMockSkillStore() // Empty - no skills
	contentIO := NewMockContentIO().
		WithFile("/test/store/skills/packs/core/pending-skill/SKILL.md", "pre-written content")
	adapter := NewStoreAdapter(skillStore, contentIO)

	// ACT: Get content for skill that doesn't exist in store yet
	content, err := adapter.GetContent("core", "pending-skill.md")
	// ASSERT: Should fall back to reading from disk
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if content != "pre-written content" {
		t.Errorf("expected 'pre-written content', got %q", content)
	}
}

func TestStoreAdapter_GetContent_NeitherExistsReturnsError(t *testing.T) {
	// ARRANGE: Neither skill nor file exists
	skillStore := NewMockSkillStore()
	contentIO := NewMockContentIO()
	adapter := NewStoreAdapter(skillStore, contentIO)

	// ACT: Get content for non-existent skill
	_, err := adapter.GetContent("core", "missing.md")

	// ASSERT: Should return error
	if err == nil {
		t.Fatal("expected error when neither skill nor file exists")
	}
}

func TestStoreAdapter_MoveOperation_EndToEnd(t *testing.T) {
	// This tests the critical move workflow bug fix:
	// When a skill is being created (doesn't exist in store), SaveContent
	// must write to disk so SaveMetadata/Create can read it later.
	//
	// The bug was: SaveContent returned nil without writing when skill
	// didn't exist, causing empty SKILL.md files after moves.

	// ARRANGE: Empty store (simulating state after skill deleted from source pack)
	skillStore := NewMockSkillStore() // No skills - simulates post-delete state
	contentIO := NewMockContentIO()
	adapter := NewStoreAdapter(skillStore, contentIO)

	// ACT Step 1: Pre-write content to target location (during move)
	// This is called BEFORE SaveMetadata creates the skill
	err := adapter.SaveContent("core", "moving-skill.md", "the skill content")
	if err != nil {
		t.Fatalf("SaveContent failed: %v", err)
	}

	// ASSERT: Content should be written to disk
	targetPath := "/test/store/skills/packs/core/moving-skill/SKILL.md"
	if !contentIO.HasFile(targetPath) {
		t.Fatal("content should be pre-written to target pack")
	}
	if contentIO.GetFile(targetPath) != "the skill content" {
		t.Errorf("expected 'the skill content', got %q", contentIO.GetFile(targetPath))
	}

	// ACT Step 2: GetContent reads the pre-written content
	// This is called by SaveMetadata when creating the skill
	content, err := adapter.GetContent("core", "moving-skill.md")
	if err != nil {
		t.Fatalf("GetContent failed: %v", err)
	}
	if content != "the skill content" {
		t.Errorf("expected 'the skill content', got %q", content)
	}
}

func TestStoreAdapter_SaveContent_SkillExistsInDifferentPack_UsesUpdate(t *testing.T) {
	// When skill exists (even in different pack), SaveContent routes to Update.
	// This is correct - the store.Update handles content persistence.

	// ARRANGE: Skill exists in "local"
	skillStore := NewMockSkillStore().WithSkill(&store.Skill{
		ID:   "my-skill",
		Name: "My Skill",
		Pack: "local",
	})
	contentIO := NewMockContentIO()
	adapter := NewStoreAdapter(skillStore, contentIO)

	// ACT: Save content (even to different pack)
	err := adapter.SaveContent("core", "my-skill.md", "updated content")
	// ASSERT: No error, and content NOT written directly to disk
	// (it goes through store.Update instead)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// Content should NOT be in contentIO - it's handled by store.Update
	if contentIO.HasFile("/test/store/skills/packs/core/my-skill/SKILL.md") {
		t.Error("content should not be written to disk when skill exists")
	}
}

func TestStoreAdapter_SaveContent_WriteError(t *testing.T) {
	// ARRANGE: ContentIO returns write error
	skillStore := NewMockSkillStore()
	contentIO := NewMockContentIO().WithWriteError(errors.New("disk full"))
	adapter := NewStoreAdapter(skillStore, contentIO)

	// ACT: Try to save content
	err := adapter.SaveContent("core", "new-skill.md", "content")

	// ASSERT: Error should propagate
	if err == nil {
		t.Fatal("expected error when write fails")
	}
	if err.Error() != "disk full" {
		t.Errorf("expected 'disk full' error, got %q", err.Error())
	}
}

func TestStoreAdapter_GetContent_ExtractsIDFromFilename(t *testing.T) {
	tests := []struct {
		name           string
		filename       string
		expectedPath   string
		preWriteToPath string
	}{
		{
			name:           "simple filename",
			filename:       "my-skill.md",
			expectedPath:   "/test/store/skills/packs/local/my-skill/SKILL.md",
			preWriteToPath: "/test/store/skills/packs/local/my-skill/SKILL.md",
		},
		{
			name:           "filename with folder prefix",
			filename:       "core/my-skill.md",
			expectedPath:   "/test/store/skills/packs/local/my-skill/SKILL.md",
			preWriteToPath: "/test/store/skills/packs/local/my-skill/SKILL.md",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			skillStore := NewMockSkillStore()
			contentIO := NewMockContentIO().WithFile(tc.preWriteToPath, "test content")
			adapter := NewStoreAdapter(skillStore, contentIO)

			// ACT
			content, err := adapter.GetContent("local", tc.filename)
			// ASSERT
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if content != "test content" {
				t.Errorf("expected 'test content', got %q", content)
			}
		})
	}
}

func TestStoreAdapter_FindByID(t *testing.T) {
	// ARRANGE
	skillStore := NewMockSkillStore().WithSkill(&store.Skill{
		ID:          "test-skill",
		Name:        "Test Skill",
		Description: "A test skill",
		Pack:        "core",
		Modes:       []string{"steer"},
		Status:      store.StatusActive,
	})
	contentIO := NewMockContentIO()
	adapter := NewStoreAdapter(skillStore, contentIO)

	// ACT
	meta, folder, err := adapter.FindByID("test-skill")
	// ASSERT
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if meta.ID != "test-skill" {
		t.Errorf("expected ID 'test-skill', got %q", meta.ID)
	}
	if meta.Name != "Test Skill" {
		t.Errorf("expected Name 'Test Skill', got %q", meta.Name)
	}
	if folder != "core" {
		t.Errorf("expected folder 'core', got %q", folder)
	}
	if len(meta.Modes) != 1 || meta.Modes[0] != "steer" {
		t.Errorf("expected Modes ['steer'], got %v", meta.Modes)
	}
	if meta.Draft {
		t.Error("expected Draft=false for StatusActive")
	}
}

func TestStoreAdapter_FindByID_NotFound(t *testing.T) {
	// ARRANGE
	skillStore := NewMockSkillStore()
	contentIO := NewMockContentIO()
	adapter := NewStoreAdapter(skillStore, contentIO)

	// ACT
	_, _, err := adapter.FindByID("missing")

	// ASSERT
	if err == nil {
		t.Fatal("expected error for missing skill")
	}
}

func TestStoreAdapter_GetAll(t *testing.T) {
	// ARRANGE
	skillStore := NewMockSkillStore().
		WithSkill(&store.Skill{ID: "skill-1", Name: "Skill 1", Pack: "core"}).
		WithSkill(&store.Skill{ID: "skill-2", Name: "Skill 2", Pack: "local"})
	contentIO := NewMockContentIO()
	adapter := NewStoreAdapter(skillStore, contentIO)

	// ACT
	all, err := adapter.GetAll()
	// ASSERT
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 skills, got %d", len(all))
	}
}

func TestStoreAdapter_LoadMetadata_FiltersByPack(t *testing.T) {
	// ARRANGE
	skillStore := NewMockSkillStore().
		WithSkill(&store.Skill{ID: "core-skill", Name: "Core", Pack: "core"}).
		WithSkill(&store.Skill{ID: "local-skill", Name: "Local", Pack: "local"})
	contentIO := NewMockContentIO()
	adapter := NewStoreAdapter(skillStore, contentIO)

	// ACT
	coreSkills, err := adapter.LoadMetadata("core")
	// ASSERT
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(coreSkills) != 1 {
		t.Errorf("expected 1 skill, got %d", len(coreSkills))
	}
	if coreSkills[0].ID != "core-skill" {
		t.Errorf("expected ID 'core-skill', got %q", coreSkills[0].ID)
	}
}

func TestStoreAdapter_Rename(t *testing.T) {
	// ARRANGE
	skillStore := NewMockSkillStore().WithSkill(&store.Skill{
		ID:   "old-id",
		Name: "My Skill",
		Pack: "local",
	})
	contentIO := NewMockContentIO()
	adapter := NewStoreAdapter(skillStore, contentIO)

	// ACT
	meta, err := adapter.Rename("old-id", "new-id")
	// ASSERT
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if meta.ID != "new-id" {
		t.Errorf("expected ID 'new-id', got %q", meta.ID)
	}

	// Old ID should be gone
	_, _, err = adapter.FindByID("old-id")
	if err == nil {
		t.Error("expected error when finding old-id after rename")
	}

	// New ID should exist
	_, _, err = adapter.FindByID("new-id")
	if err != nil {
		t.Errorf("expected new-id to exist, got error: %v", err)
	}
}

func TestStoreAdapter_StatusDraftMapping(t *testing.T) {
	tests := []struct {
		name          string
		status        string
		expectedDraft bool
	}{
		{"active status", store.StatusActive, false},
		{"draft status", store.StatusDraft, true},
		{"archived status", store.StatusArchived, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			skillStore := NewMockSkillStore().WithSkill(&store.Skill{
				ID:     "skill",
				Name:   "Skill",
				Pack:   "local",
				Status: tc.status,
			})
			contentIO := NewMockContentIO()
			adapter := NewStoreAdapter(skillStore, contentIO)

			// ACT
			meta, _, err := adapter.FindByID("skill")
			// ASSERT
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if meta.Draft != tc.expectedDraft {
				t.Errorf("expected Draft=%v for status %q, got %v", tc.expectedDraft, tc.status, meta.Draft)
			}
		})
	}
}

// --- Bug Reproduction Tests ---
// These tests verify the fix for the revision increment bug where
// SaveMetadata was calling Update() on ALL skills, not just changed ones.

func TestStoreAdapter_SaveMetadata_OnlyUpdatesChangedSkills(t *testing.T) {
	// This test reproduces the bug: when SaveMetadata is called with all skills
	// in a folder, but only ONE skill has actually changed, ONLY that skill
	// should have Update() called on it.
	//
	// BUG: Previously, SaveMetadata called Update() on EVERY skill in the list,
	// causing all skills to get revision increments and history.jsonl entries.

	// ARRANGE: Two existing skills in the same pack
	skillStore := NewMockSkillStore().
		WithSkill(&store.Skill{
			ID:          "unchanged-skill",
			Name:        "Unchanged Skill",
			Description: "Original description",
			Pack:        "local",
			Modes:       []string{"steer"},
			Tags:        []string{"tag1"},
			Status:      store.StatusActive,
			Timestamps: store.Timestamps{
				Revision:  5,
				CreatedAt: "2024-01-01T00:00:00Z",
				UpdatedAt: "2024-01-15T00:00:00Z",
			},
		}).
		WithSkill(&store.Skill{
			ID:          "changed-skill",
			Name:        "Changed Skill",
			Description: "Original description",
			Pack:        "local",
			Modes:       []string{"practice"},
			Tags:        []string{"tag2"},
			Status:      store.StatusActive,
			Timestamps: store.Timestamps{
				Revision:  3,
				CreatedAt: "2024-01-01T00:00:00Z",
				UpdatedAt: "2024-01-10T00:00:00Z",
			},
		})
	contentIO := NewMockContentIO()
	adapter := NewStoreAdapter(skillStore, contentIO)

	// ACT: Save metadata with one changed skill, one unchanged
	// The unchanged-skill has identical metadata to what's in the store
	// The changed-skill has a new name
	err := adapter.SaveMetadata("local", []Metadata{
		{
			ID:          "unchanged-skill",
			Name:        "Unchanged Skill", // Same as store
			Description: "Original description",
			File:        "local/unchanged-skill.md",
			Modes:       []string{"steer"},
			Tags:        []string{"tag1"},
			Draft:       false,
			CreatedAt:   "2024-01-01T00:00:00Z",
			UpdatedAt:   "2024-01-15T00:00:00Z",
		},
		{
			ID:          "changed-skill",
			Name:        "UPDATED Skill Name", // CHANGED!
			Description: "Original description",
			File:        "local/changed-skill.md",
			Modes:       []string{"practice"},
			Tags:        []string{"tag2"},
			Draft:       false,
			CreatedAt:   "2024-01-01T00:00:00Z",
			UpdatedAt:   "2024-01-10T00:00:00Z",
		},
	})
	// ASSERT: Only the changed skill should have Update() called
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// This is the critical assertion - ONLY the changed skill should be updated
	if len(skillStore.UpdateCalls) != 1 {
		t.Errorf("expected 1 Update() call for changed skill only, got %d calls: %v",
			len(skillStore.UpdateCalls), skillStore.UpdateCalls)
	}
	if len(skillStore.UpdateCalls) == 1 && skillStore.UpdateCalls[0] != "changed-skill" {
		t.Errorf("expected Update() for 'changed-skill', got %q", skillStore.UpdateCalls[0])
	}
}

func TestStoreAdapter_SaveMetadata_NoUpdatesWhenNothingChanged(t *testing.T) {
	// When no skills have changed, no Update() calls should be made.

	// ARRANGE
	skillStore := NewMockSkillStore().
		WithSkill(&store.Skill{
			ID:          "skill-1",
			Name:        "Skill One",
			Description: "Description",
			Pack:        "local",
			Modes:       []string{"steer"},
			Status:      store.StatusActive,
		}).
		WithSkill(&store.Skill{
			ID:          "skill-2",
			Name:        "Skill Two",
			Description: "Description",
			Pack:        "local",
			Modes:       []string{"practice"},
			Status:      store.StatusDraft,
		})
	contentIO := NewMockContentIO()
	adapter := NewStoreAdapter(skillStore, contentIO)

	// ACT: Save the same metadata (no changes)
	err := adapter.SaveMetadata("local", []Metadata{
		{
			ID:          "skill-1",
			Name:        "Skill One",
			Description: "Description",
			File:        "local/skill-1.md",
			Modes:       []string{"steer"},
			Draft:       false,
		},
		{
			ID:          "skill-2",
			Name:        "Skill Two",
			Description: "Description",
			File:        "local/skill-2.md",
			Modes:       []string{"practice"},
			Draft:       true,
		},
	})
	// ASSERT: No Update() calls
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(skillStore.UpdateCalls) != 0 {
		t.Errorf("expected 0 Update() calls when nothing changed, got %d: %v",
			len(skillStore.UpdateCalls), skillStore.UpdateCalls)
	}
}

func TestStoreAdapter_SaveMetadata_AllChangedUpdatesAll(t *testing.T) {
	// When all skills have changed, all should be updated.

	// ARRANGE
	skillStore := NewMockSkillStore().
		WithSkill(&store.Skill{
			ID:   "skill-1",
			Name: "Old Name 1",
			Pack: "local",
		}).
		WithSkill(&store.Skill{
			ID:   "skill-2",
			Name: "Old Name 2",
			Pack: "local",
		})
	contentIO := NewMockContentIO()
	adapter := NewStoreAdapter(skillStore, contentIO)

	// ACT: Update both skills
	err := adapter.SaveMetadata("local", []Metadata{
		{
			ID:   "skill-1",
			Name: "New Name 1", // Changed
			File: "local/skill-1.md",
		},
		{
			ID:   "skill-2",
			Name: "New Name 2", // Changed
			File: "local/skill-2.md",
		},
	})
	// ASSERT: Both updated
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(skillStore.UpdateCalls) != 2 {
		t.Errorf("expected 2 Update() calls, got %d: %v",
			len(skillStore.UpdateCalls), skillStore.UpdateCalls)
	}
}

// --- metadataChanged Helper Tests ---
// These test the change detection logic directly for edge cases.

func TestMetadataChanged_EachFieldDetectsChanges(t *testing.T) {
	// Test that each field change is detected independently.
	base := Metadata{
		ID:           "test-skill",
		Name:         "Test Skill",
		Description:  "A description",
		Modes:        []string{"steer"},
		Tags:         []string{"tag1", "tag2"},
		Icon:         "🎯",
		Draft:        false,
		DefaultScope: "scenario",
		TargetToolID: ptrString("tool-1"),
	}

	tests := []struct {
		name       string
		modify     func(m *Metadata)
		wantChange bool
	}{
		{
			name:       "no_change",
			modify:     func(m *Metadata) {}, // No changes
			wantChange: false,
		},
		{
			name:       "name_changed",
			modify:     func(m *Metadata) { m.Name = "Different Name" },
			wantChange: true,
		},
		{
			name:       "description_changed",
			modify:     func(m *Metadata) { m.Description = "Different description" },
			wantChange: true,
		},
		{
			name:       "icon_changed",
			modify:     func(m *Metadata) { m.Icon = "🔧" },
			wantChange: true,
		},
		{
			name:       "draft_changed",
			modify:     func(m *Metadata) { m.Draft = true },
			wantChange: true,
		},
		{
			name:       "defaultScope_changed",
			modify:     func(m *Metadata) { m.DefaultScope = "project" },
			wantChange: true,
		},
		{
			name:       "modes_changed_content",
			modify:     func(m *Metadata) { m.Modes = []string{"practice"} },
			wantChange: true,
		},
		{
			name:       "modes_changed_order",
			modify:     func(m *Metadata) { m.Modes = []string{"practice", "steer"} },
			wantChange: true,
		},
		{
			name:       "modes_changed_length",
			modify:     func(m *Metadata) { m.Modes = []string{"steer", "practice", "tool"} },
			wantChange: true,
		},
		{
			name:       "tags_changed_content",
			modify:     func(m *Metadata) { m.Tags = []string{"tag1", "different"} },
			wantChange: true,
		},
		{
			name:       "tags_changed_length",
			modify:     func(m *Metadata) { m.Tags = []string{"tag1"} },
			wantChange: true,
		},
		{
			name:       "targetToolID_changed_value",
			modify:     func(m *Metadata) { m.TargetToolID = ptrString("tool-2") },
			wantChange: true,
		},
		{
			name:       "targetToolID_to_nil",
			modify:     func(m *Metadata) { m.TargetToolID = nil },
			wantChange: true,
		},
		// File, ID, CreatedAt, UpdatedAt should NOT trigger changes
		{
			name:       "file_changed_ignored",
			modify:     func(m *Metadata) { m.File = "different/path.md" },
			wantChange: false,
		},
		{
			name:       "createdAt_changed_ignored",
			modify:     func(m *Metadata) { m.CreatedAt = "2025-01-01T00:00:00Z" },
			wantChange: false,
		},
		{
			name:       "updatedAt_changed_ignored",
			modify:     func(m *Metadata) { m.UpdatedAt = "2025-01-01T00:00:00Z" },
			wantChange: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a copy and modify it
			modified := base
			modified.Modes = make([]string, len(base.Modes))
			copy(modified.Modes, base.Modes)
			modified.Tags = make([]string, len(base.Tags))
			copy(modified.Tags, base.Tags)
			if base.TargetToolID != nil {
				modified.TargetToolID = ptrString(*base.TargetToolID)
			}

			tc.modify(&modified)

			got := metadataChanged(base, modified)
			if got != tc.wantChange {
				t.Errorf("metadataChanged() = %v, want %v", got, tc.wantChange)
			}
		})
	}
}

func TestMetadataChanged_NilVsEmptySlices(t *testing.T) {
	// nil slices and empty slices should be treated as equal
	base := Metadata{
		ID:    "test",
		Name:  "Test",
		Modes: nil,
		Tags:  nil,
	}

	modified := Metadata{
		ID:    "test",
		Name:  "Test",
		Modes: []string{},
		Tags:  []string{},
	}

	// Both nil vs both empty should NOT be considered a change
	got := metadataChanged(base, modified)
	if got {
		t.Errorf("nil slices vs empty slices should NOT be considered a change")
	}
}

func TestMetadataChanged_TargetToolID_NilToNil(t *testing.T) {
	base := Metadata{ID: "test", Name: "Test", TargetToolID: nil}
	modified := Metadata{ID: "test", Name: "Test", TargetToolID: nil}

	got := metadataChanged(base, modified)
	if got {
		t.Errorf("both nil TargetToolID should NOT be a change")
	}
}

func TestMetadataChanged_TargetToolID_NilToValue(t *testing.T) {
	base := Metadata{ID: "test", Name: "Test", TargetToolID: nil}
	modified := Metadata{ID: "test", Name: "Test", TargetToolID: ptrString("tool-1")}

	got := metadataChanged(base, modified)
	if !got {
		t.Errorf("nil to value TargetToolID should be a change")
	}
}

func TestMetadataChanged_TargetToolID_ValueToNil(t *testing.T) {
	base := Metadata{ID: "test", Name: "Test", TargetToolID: ptrString("tool-1")}
	modified := Metadata{ID: "test", Name: "Test", TargetToolID: nil}

	got := metadataChanged(base, modified)
	if !got {
		t.Errorf("value to nil TargetToolID should be a change")
	}
}

func TestMetadataChanged_TargetToolID_SameValue(t *testing.T) {
	base := Metadata{ID: "test", Name: "Test", TargetToolID: ptrString("tool-1")}
	modified := Metadata{ID: "test", Name: "Test", TargetToolID: ptrString("tool-1")}

	got := metadataChanged(base, modified)
	if got {
		t.Errorf("same TargetToolID value should NOT be a change")
	}
}

// ptrString returns a pointer to the given string value.
// Helper for creating *string in test cases.
func ptrString(s string) *string {
	return &s
}
