package persistence

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/internal/testutil"
)

// [REQ:BAS-RH-J14] Caller-controlled IDs never address sibling files.
func TestFileRepositoryRejectsPathsBeforeIO(t *testing.T) {
	operations := map[string]func(*FileRepository, ProfileID) error{
		"get":    func(repo *FileRepository, id ProfileID) error { _, err := repo.Get(id); return err },
		"create": func(repo *FileRepository, id ProfileID) error { return repo.Create(&SessionProfile{ID: id}) },
		"update": func(repo *FileRepository, id ProfileID) error {
			_, err := repo.Update(id, func(*SessionProfile) error { t.Error("invalid ID reached mutation"); return nil })
			return err
		},
		"delete": (*FileRepository).Delete,
	}
	for name, operation := range operations {
		for _, id := range []ProfileID{"", ".", "..", "../outside", "nested/../../outside", `..\outside`, `/outside`, `C:\outside`, "inside:stream", "bad\x00id"} {
			t.Run(name+"/"+string(id), func(t *testing.T) {
				parent := t.TempDir()
				outside := filepath.Join(parent, "outside.json")
				require.NoError(t, os.WriteFile(outside, []byte(`{"unrelated":"preserve"}`), 0o600))
				root := filepath.Join(parent, "profiles")
				repo := NewFileRepositoryWithConfig(root, nil, FileRepositoryConfig{Authority: testutil.ProfileCredentialAuthority})
				err := operation(repo, id)
				require.ErrorContains(t, err, "profile id", "invalid identifier must fail at the identity boundary")
				contents, err := os.ReadFile(outside)
				require.NoError(t, err)
				require.Equal(t, `{"unrelated":"preserve"}`, string(contents))
				entries, err := os.ReadDir(root)
				require.NoError(t, err)
				require.Empty(t, entries, "invalid identity must not create a lock, key witness or profile")
			})
		}
	}
}

// [REQ:BAS-RH-J06] A rejected commit cannot damage the acknowledged snapshot.
func TestFileRepositoryCommitPreservesAcknowledgedSnapshot(t *testing.T) {
	for _, fault := range []string{"write", "rename"} {
		t.Run(fault, func(t *testing.T) {
			files := NewMockFileSystem()
			repo := NewFileRepositoryWithConfig("/data", nil, FileRepositoryConfig{Authority: testutil.ProfileCredentialAuthority, FileSystem: files})
			old := &SessionProfile{ID: "identity", Name: "old", StorageState: []byte(`{"cookies":[{"value":"old-secret"}]}`)}
			if err := repo.Create(old); err != nil {
				t.Fatal(err)
			}
			candidate := *old
			candidate.Name = "new"
			candidate.StorageState = []byte(`{"cookies":[{"value":"new-secret"}]}`)
			if fault == "write" {
				files.WriteFileErr = errors.New("disk full")
			} else {
				files.RenameErr = errors.New("commit rejected")
			}
			if _, err := repo.Update(candidate.ID, func(profile *SessionProfile) error { *profile = candidate; return nil }); err == nil {
				t.Fatal("failed commit acknowledged")
			}
			files.WriteFileErr, files.RenameErr = nil, nil
			fresh := NewFileRepositoryWithConfig("/data", nil, FileRepositoryConfig{Authority: testutil.ProfileCredentialAuthority, FileSystem: files})
			recovered, err := fresh.Get(old.ID)
			if err != nil {
				t.Fatal(err)
			}
			if recovered.Name != old.Name || !bytes.Equal(recovered.StorageState, old.StorageState) {
				t.Fatal("failed commit changed the acknowledged snapshot")
			}
		})
	}
}

// [REQ:BAS-RH-J14] One profile has one protected, atomically replaceable document.
func TestFileRepositoryCommitUsesOneDocument(t *testing.T) {
	root := t.TempDir()
	config := FileRepositoryConfig{Authority: testutil.ProfileCredentialAuthority}
	repo := NewFileRepositoryWithConfig(root, nil, config)
	profile := &SessionProfile{ID: "identity", Name: "Private name", StorageState: []byte(`{"cookies":[{"value":"synthetic-secret"}]}`)}
	require.NoError(t, repo.Create(profile))
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	require.Len(t, entries, 3, "one profile document plus credential-loss witness and stable write lock")
	data, err := os.ReadFile(filepath.Join(root, string(profile.ID)+".json"))
	require.NoError(t, err)
	require.NotContains(t, string(data), "synthetic-secret")
	require.NotContains(t, string(data), "Private name")
	fresh := NewFileRepositoryWithConfig(root, nil, config)
	got, err := fresh.Get(profile.ID)
	require.NoError(t, err)
	require.Equal(t, profile, got, "one complete snapshot must recover")
}

func TestFileRepository_Get(t *testing.T) {
	mockFS := NewMockFileSystem()
	repo := NewFileRepositoryWithConfig("/data", nil, FileRepositoryConfig{
		Authority:  testutil.ProfileCredentialAuthority,
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
	if err := repo.Create(profile); err != nil {
		t.Fatal(err)
	}

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

func TestFileRepositoryKeepsSensitiveStateOutOfProfileJSON(t *testing.T) {
	mockFS := NewMockFileSystem()
	repo := NewFileRepositoryWithConfig("/data", nil, FileRepositoryConfig{Authority: testutil.ProfileCredentialAuthority, FileSystem: mockFS})
	profile := &SessionProfile{
		ID:             "protected",
		Name:           "Protected",
		StorageState:   []byte(`{"cookies":[{"value":"cookie-secret"}]}`),
		BrowserProfile: &BrowserProfile{Proxy: &ProxySettings{Password: "proxy-secret"}},
		History:        []HistoryEntry{{ID: "history", URL: "https://private.example"}},
	}
	if err := repo.Create(profile); err != nil {
		t.Fatalf("create: %v", err)
	}
	metadata, ok := mockFS.GetFile("/data/protected.json")
	if !ok {
		t.Fatal("metadata file missing")
	}
	for _, secret := range []string{"cookie-secret", "proxy-secret", "private.example"} {
		if strings.Contains(string(metadata), secret) {
			t.Fatalf("metadata leaks %q", secret)
		}
	}
	var document profileDocument
	if err := json.Unmarshal(metadata, &document); err != nil || len(document.Sealed) == 0 {
		t.Fatal("encrypted profile payload missing")
	}
	loaded, err := repo.Get("protected")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if string(loaded.StorageState) != string(profile.StorageState) || loaded.BrowserProfile.Proxy.Password != "proxy-secret" {
		t.Fatal("protected state did not round trip")
	}
}

func TestFileRepository_List(t *testing.T) {
	mockFS := NewMockFileSystem()
	repo := NewFileRepositoryWithConfig("/data", nil, FileRepositoryConfig{
		Authority:  testutil.ProfileCredentialAuthority,
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
		if err := repo.Create(&p); err != nil {
			t.Fatal(err)
		}
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
		Authority:  testutil.ProfileCredentialAuthority,
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

func TestFileRepository_Delete(t *testing.T) {
	mockFS := NewMockFileSystem()
	repo := NewFileRepositoryWithConfig("/data", nil, FileRepositoryConfig{
		Authority:  testutil.ProfileCredentialAuthority,
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
		Authority:  testutil.ProfileCredentialAuthority,
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
		Authority:  testutil.ProfileCredentialAuthority,
		FileSystem: mockFS,
	})

	profile := &SessionProfile{
		ID:        "write-error",
		Name:      "Write Error Test",
		CreatedAt: time.Now().UTC(),
	}

	// Inject write error
	mockFS.WriteFileErr = errors.New("disk full")

	err := repo.Create(profile)
	if err == nil {
		t.Error("expected error when write fails")
	}

	mockFS.WriteFileErr = nil
}

func TestFileRepository_ListReadDirError(t *testing.T) {
	mockFS := NewMockFileSystem()
	repo := NewFileRepositoryWithConfig("/data", nil, FileRepositoryConfig{
		Authority:  testutil.ProfileCredentialAuthority,
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
		Authority:  testutil.ProfileCredentialAuthority,
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

	require.NoError(t, repo.Create(profile))

	// Multiple goroutines writing to the same profile
	done := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			p := *profile
			p.Name = "Update " + string(rune('A'+n))
			p.UpdatedAt = time.Now().UTC()
			_, err := repo.Update(p.ID, func(current *SessionProfile) error { *current = p; return nil })
			done <- err
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

	// The final committed document must decrypt as a coherent profile.
	finalProfile, err := repo.Get(profile.ID)
	if err != nil || finalProfile == nil || !strings.HasPrefix(finalProfile.Name, "Update ") {
		t.Fatalf("final profile did not recover: %v", err)
	}
}

func TestFileRepository_CreateSetsDefaultTimestamps(t *testing.T) {
	mockFS := NewMockFileSystem()
	repo := NewFileRepositoryWithConfig("/data", nil, FileRepositoryConfig{
		Authority:  testutil.ProfileCredentialAuthority,
		FileSystem: mockFS,
	})

	// Profile with only UpdatedAt set
	profile := &SessionProfile{
		ID:        "timestamps",
		Name:      "Timestamps Test",
		UpdatedAt: time.Now().UTC(),
	}

	err := repo.Create(profile)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Read it back
	saved, err := repo.Get(profile.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
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
	if err := mockFS.MkdirAll("/data/mydir", 0o755); err != nil {
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

// [REQ:BAS-RH-J14] Concurrent complete saves never expose mixed or partial documents.
func TestFileRepositoryConcurrentAtomicSnapshots(t *testing.T) {
	root := t.TempDir()
	repo := NewFileRepositoryWithConfig(root, nil, FileRepositoryConfig{Authority: testutil.ProfileCredentialAuthority})
	require.NoError(t, repo.Create(&SessionProfile{ID: "identity", Name: "initial", StorageState: []byte(`{"snapshot":"initial"}`)}))
	const writers = 24
	done := make(chan error, writers)
	for i := 0; i < writers; i++ {
		go func(n int) {
			name := fmt.Sprintf("snapshot-%d", n)
			state, _ := json.Marshal(map[string]string{"snapshot": name})
			profile := &SessionProfile{ID: "identity", Name: name, StorageState: state}
			if _, err := repo.Update(profile.ID, func(current *SessionProfile) error { *current = *profile; return nil }); err != nil {
				done <- err
				return
			}
			got, err := NewFileRepositoryWithConfig(root, nil, FileRepositoryConfig{Authority: testutil.ProfileCredentialAuthority}).Get(profile.ID)
			if err != nil {
				done <- err
				return
			}
			var decoded map[string]string
			if err := json.Unmarshal(got.StorageState, &decoded); err != nil {
				done <- err
				return
			}
			if got.Name != decoded["snapshot"] {
				done <- errors.New("mixed profile generations")
				return
			}
			done <- nil
		}(i)
	}
	for i := 0; i < writers; i++ {
		if err := <-done; err != nil {
			t.Error(err)
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 3 {
		t.Fatalf("temporary documents remain after concurrent writes: %d, %v", len(entries), err)
	}
}

func TestFileRepositoryRejectsForeignOrOldDocument(t *testing.T) {
	files := NewMockFileSystem()
	repo := NewFileRepositoryWithConfig("/data", nil, FileRepositoryConfig{Authority: testutil.ProfileCredentialAuthority, FileSystem: files})
	if err := repo.Create(&SessionProfile{ID: "one", Name: "One"}); err != nil {
		t.Fatal(err)
	}
	encoded, _ := files.GetFile("/data/one.json")
	files.SetFile("/data/two.json", encoded)
	if _, err := repo.Get("two"); err == nil {
		t.Fatal("profile was accepted under another identity")
	}
	files.SetFile("/data/old.json", []byte(`{"id":"old","name":"Original","storage_state":{"cookies":[]}}`))
	if _, err := repo.Get("old"); err == nil {
		t.Fatal("old profile was silently accepted or converted")
	}
	unchanged, _ := files.GetFile("/data/old.json")
	if !bytes.Equal(unchanged, []byte(`{"id":"old","name":"Original","storage_state":{"cookies":[]}}`)) {
		t.Fatal("reading old format mutated it")
	}
}

// [REQ:BAS-RH-J06] A failed transaction never acknowledges or leaks a partial edit.
func TestFileRepositoryUpdateFailurePreservesSnapshot(t *testing.T) {
	for _, fault := range []string{"callback", "commit", "lock", "identity"} {
		t.Run(fault, func(t *testing.T) {
			files := NewMockFileSystem()
			repo := NewFileRepositoryWithConfig("/data", nil, FileRepositoryConfig{FileSystem: files, Authority: testutil.ProfileCredentialAuthority})
			require.NoError(t, repo.Create(&SessionProfile{ID: "original", Name: "Acknowledged", StorageState: []byte(`{"cookies":[{"value":"original"}]}`)}))
			before, ok := files.GetFile("/data/original.json")
			require.True(t, ok)
			if fault == "commit" {
				files.WriteFileErr = errors.New("synthetic disk failure")
			}
			if fault == "lock" {
				files.LockErr = errors.New("synthetic writer deadline")
			}
			called := false
			result, err := repo.Update("original", func(p *SessionProfile) error {
				called = true
				p.Name = "Rejected"
				p.StorageState = []byte(`{"cookies":[]}`)
				if fault == "callback" {
					return errors.New("synthetic rejected edit")
				}
				if fault == "identity" {
					p.ID = "replacement"
				}
				return nil
			})
			require.Error(t, err)
			require.Nil(t, result)
			require.Equal(t, fault != "lock", called, "refused ownership must not run a mutation")
			after, ok := files.GetFile("/data/original.json")
			require.True(t, ok)
			require.Equal(t, before, after)
			require.False(t, files.FileExists("/data/replacement.json"))
		})
	}
}
