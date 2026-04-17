package persistence

import (
	"encoding/json"
	"errors"
	"io/fs"
	"testing"
	"time"
)

func TestFileRepository_Get(t *testing.T) {
	mockFS := NewMockFileSystem()
	repo := NewFileRepositoryWithConfig("/data", nil, FileRepositoryConfig{
		FileSystem: mockFS,
	})

	// Create a profile in the mock filesystem
	profile := &SessionProfile{
		ID:         "test-profile-1",
		Name:       "Test Profile",
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
		LastUsedAt: time.Now().UTC(),
	}
	data, _ := json.MarshalIndent(profile, "", "  ")
	mockFS.SetFile("/data/test-profile-1.json", data)

	// Test successful get
	t.Run("successful get", func(t *testing.T) {
		retrieved, err := repo.Get("test-profile-1")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if retrieved == nil {
			t.Fatal("expected profile to exist")
		}
		if retrieved.Name != "Test Profile" {
			t.Errorf("expected name 'Test Profile', got '%s'", retrieved.Name)
		}
	})

	// Test not found
	t.Run("not found returns nil", func(t *testing.T) {
		retrieved, err := repo.Get("nonexistent")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if retrieved != nil {
			t.Error("expected nil for nonexistent profile")
		}
	})

	// Test empty ID
	t.Run("empty ID returns error", func(t *testing.T) {
		_, err := repo.Get("")
		if err == nil {
			t.Error("expected error for empty ID")
		}
	})

	// Test corrupt JSON
	t.Run("corrupt JSON returns error", func(t *testing.T) {
		mockFS.SetFile("/data/corrupt.json", []byte("{invalid json"))
		_, err := repo.Get("corrupt")
		if err == nil {
			t.Error("expected error for corrupt JSON")
		}
	})
}

func TestFileRepository_List(t *testing.T) {
	mockFS := NewMockFileSystem()
	repo := NewFileRepositoryWithConfig("/data", nil, FileRepositoryConfig{
		FileSystem: mockFS,
	})

	// Create multiple profiles with different timestamps
	now := time.Now().UTC()
	profiles := []SessionProfile{
		{ID: "profile-1", Name: "Old Profile", CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now, LastUsedAt: now.Add(-2 * time.Hour)},
		{ID: "profile-2", Name: "Recent Profile", CreatedAt: now.Add(-time.Hour), UpdatedAt: now, LastUsedAt: now},
		{ID: "profile-3", Name: "Middle Profile", CreatedAt: now.Add(-90 * time.Minute), UpdatedAt: now, LastUsedAt: now.Add(-time.Hour)},
	}

	for _, p := range profiles {
		data, _ := json.MarshalIndent(&p, "", "  ")
		mockFS.SetFile("/data/"+string(p.ID)+".json", data)
	}

	// List should return profiles sorted by last_used_at desc
	listed, err := repo.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(listed) != 3 {
		t.Errorf("expected 3 profiles, got %d", len(listed))
	}

	// First should be most recently used
	if listed[0].ID != "profile-2" {
		t.Errorf("expected first profile to be 'profile-2', got '%s'", listed[0].ID)
	}
}

func TestFileRepository_Create(t *testing.T) {
	mockFS := NewMockFileSystem()
	repo := NewFileRepositoryWithConfig("/data", nil, FileRepositoryConfig{
		FileSystem: mockFS,
	})

	now := time.Now().UTC()
	profile := &SessionProfile{
		ID:         "new-profile",
		Name:       "New Profile",
		CreatedAt:  now,
		UpdatedAt:  now,
		LastUsedAt: now,
	}

	// Test successful create
	t.Run("successful create", func(t *testing.T) {
		err := repo.Create(profile)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		// Verify file was created
		if !mockFS.FileExists("/data/new-profile.json") {
			t.Error("expected file to be created")
		}
	})

	// Test duplicate create
	t.Run("duplicate create fails", func(t *testing.T) {
		err := repo.Create(profile)
		if err == nil {
			t.Error("expected error for duplicate create")
		}
	})

	// Test nil profile
	t.Run("nil profile fails", func(t *testing.T) {
		err := repo.Create(nil)
		if err == nil {
			t.Error("expected error for nil profile")
		}
	})

	// Test empty ID
	t.Run("empty ID fails", func(t *testing.T) {
		err := repo.Create(&SessionProfile{Name: "No ID"})
		if err == nil {
			t.Error("expected error for empty ID")
		}
	})
}

func TestFileRepository_Save_AtomicWrite(t *testing.T) {
	mockFS := NewMockFileSystem()
	repo := NewFileRepositoryWithConfig("/data", nil, FileRepositoryConfig{
		FileSystem: mockFS,
	})

	now := time.Now().UTC()
	profile := &SessionProfile{
		ID:         "atomic-test",
		Name:       "Atomic Test",
		CreatedAt:  now,
		UpdatedAt:  now,
		LastUsedAt: now,
	}

	// Verify atomic write pattern
	err := repo.Save(profile)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Final file should exist
	if !mockFS.FileExists("/data/atomic-test.json") {
		t.Error("expected final file to exist")
	}

	// Temp file should NOT exist (was renamed)
	if mockFS.FileExists("/data/atomic-test.json.tmp") {
		t.Error("temp file should be removed after rename")
	}
}

func TestFileRepository_Save_RenameFailure(t *testing.T) {
	mockFS := NewMockFileSystem()
	repo := NewFileRepositoryWithConfig("/data", nil, FileRepositoryConfig{
		FileSystem: mockFS,
	})

	now := time.Now().UTC()
	profile := &SessionProfile{
		ID:         "rename-fail",
		Name:       "Rename Fail Test",
		CreatedAt:  now,
		UpdatedAt:  now,
		LastUsedAt: now,
	}

	// Inject rename error
	mockFS.RenameErr = errors.New("rename failed")

	err := repo.Save(profile)
	if err == nil {
		t.Error("expected error when rename fails")
	}

	// Temp file should be cleaned up on failure
	// (Note: in mock, the file is created but rename fails)
	mockFS.RenameErr = nil
}

func TestFileRepository_Delete(t *testing.T) {
	mockFS := NewMockFileSystem()
	repo := NewFileRepositoryWithConfig("/data", nil, FileRepositoryConfig{
		FileSystem: mockFS,
	})

	// Create a profile first
	profile := &SessionProfile{
		ID:        "to-delete",
		Name:      "To Delete",
		CreatedAt: time.Now().UTC(),
	}
	data, _ := json.MarshalIndent(profile, "", "  ")
	mockFS.SetFile("/data/to-delete.json", data)

	// Test successful delete
	t.Run("successful delete", func(t *testing.T) {
		err := repo.Delete("to-delete")
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		if mockFS.FileExists("/data/to-delete.json") {
			t.Error("expected file to be deleted")
		}
	})

	// Test delete nonexistent
	t.Run("delete nonexistent fails", func(t *testing.T) {
		err := repo.Delete("nonexistent")
		if err == nil {
			t.Error("expected error for nonexistent profile")
		}
	})

	// Test empty ID
	t.Run("empty ID fails", func(t *testing.T) {
		err := repo.Delete("")
		if err == nil {
			t.Error("expected error for empty ID")
		}
	})
}

func TestFileRepository_ReadError(t *testing.T) {
	mockFS := NewMockFileSystem()
	repo := NewFileRepositoryWithConfig("/data", nil, FileRepositoryConfig{
		FileSystem: mockFS,
	})

	// Create a profile so stat passes but read fails
	mockFS.SetFile("/data/read-error.json", []byte(`{"id":"read-error"}`))

	// Inject read error
	mockFS.ReadFileErr = errors.New("permission denied")

	_, err := repo.Get("read-error")
	if err == nil {
		t.Error("expected error when read fails")
	}

	mockFS.ReadFileErr = nil
}

func TestFileRepository_WriteError(t *testing.T) {
	mockFS := NewMockFileSystem()
	repo := NewFileRepositoryWithConfig("/data", nil, FileRepositoryConfig{
		FileSystem: mockFS,
	})

	profile := &SessionProfile{
		ID:        "write-error",
		Name:      "Write Error Test",
		CreatedAt: time.Now().UTC(),
	}

	// Inject write error
	mockFS.WriteFileErr = errors.New("disk full")

	err := repo.Save(profile)
	if err == nil {
		t.Error("expected error when write fails")
	}

	mockFS.WriteFileErr = nil
}

func TestFileRepository_ListReadDirError(t *testing.T) {
	mockFS := NewMockFileSystem()
	repo := NewFileRepositoryWithConfig("/data", nil, FileRepositoryConfig{
		FileSystem: mockFS,
	})

	// Inject ReadDir error
	mockFS.ReadDirErr = errors.New("cannot read directory")

	_, err := repo.List()
	if err == nil {
		t.Error("expected error when ReadDir fails")
	}

	mockFS.ReadDirErr = nil
}

func TestFileRepository_ConcurrentWrites(t *testing.T) {
	mockFS := NewMockFileSystem()
	repo := NewFileRepositoryWithConfig("/data", nil, FileRepositoryConfig{
		FileSystem: mockFS,
	})

	// Create initial profile
	now := time.Now().UTC()
	profile := &SessionProfile{
		ID:         "concurrent",
		Name:       "Initial",
		CreatedAt:  now,
		UpdatedAt:  now,
		LastUsedAt: now,
	}

	// Multiple goroutines writing to the same profile
	done := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			p := *profile
			p.Name = "Update " + string(rune('A'+n))
			p.UpdatedAt = time.Now().UTC()
			done <- repo.Save(&p)
		}(i)
	}

	// Wait for all writes
	var errors []error
	for i := 0; i < 10; i++ {
		if err := <-done; err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		t.Errorf("got %d errors during concurrent writes", len(errors))
	}

	// Verify file exists and is valid JSON
	data, ok := mockFS.GetFile("/data/concurrent.json")
	if !ok {
		t.Fatal("expected file to exist after concurrent writes")
	}

	var finalProfile SessionProfile
	if err := json.Unmarshal(data, &finalProfile); err != nil {
		t.Errorf("final file is not valid JSON: %v", err)
	}
}

func TestFileRepository_SaveSetsDefaultTimestamps(t *testing.T) {
	mockFS := NewMockFileSystem()
	repo := NewFileRepositoryWithConfig("/data", nil, FileRepositoryConfig{
		FileSystem: mockFS,
	})

	// Profile with only UpdatedAt set
	profile := &SessionProfile{
		ID:        "timestamps",
		Name:      "Timestamps Test",
		UpdatedAt: time.Now().UTC(),
	}

	err := repo.Save(profile)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Read it back
	data, _ := mockFS.GetFile("/data/timestamps.json")
	var saved SessionProfile
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// CreatedAt and LastUsedAt should be set from UpdatedAt
	if saved.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if saved.LastUsedAt.IsZero() {
		t.Error("expected LastUsedAt to be set")
	}
}

func TestMockFileSystem_ReadDir(t *testing.T) {
	mockFS := NewMockFileSystem()

	// Add files at different paths
	mockFS.SetFile("/data/profiles/a.json", []byte(`{}`))
	mockFS.SetFile("/data/profiles/b.json", []byte(`{}`))
	mockFS.SetFile("/data/other/c.json", []byte(`{}`))

	entries, err := mockFS.ReadDir("/data/profiles")
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}

	// Check entry names
	names := make(map[string]bool)
	for _, e := range entries {
		names[e.Name()] = true
	}
	if !names["a.json"] || !names["b.json"] {
		t.Errorf("expected a.json and b.json, got %v", names)
	}
}

func TestMockFileSystem_Stat(t *testing.T) {
	mockFS := NewMockFileSystem()

	// Test file stat
	mockFS.SetFile("/data/file.json", []byte(`{"test": true}`))
	info, err := mockFS.Stat("/data/file.json")
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if info.IsDir() {
		t.Error("expected file, not directory")
	}
	if info.Size() != 14 {
		t.Errorf("expected size 14, got %d", info.Size())
	}

	// Test directory stat
	if err := mockFS.MkdirAll("/data/mydir", 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	info, err = mockFS.Stat("/data/mydir")
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory")
	}

	// Test not found
	_, err = mockFS.Stat("/data/nonexistent")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected ErrNotExist, got %v", err)
	}
}
