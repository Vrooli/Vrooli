package build

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestNewStore(t *testing.T) {
	store := NewStore()
	if store == nil {
		t.Fatalf("expected store to be created")
	}
	if store.statusMap == nil {
		t.Errorf("expected statusMap to be initialized")
	}
}

func TestInMemoryStore_SaveAndGet(t *testing.T) {
	store := NewStore()

	status := &Status{
		BuildID:      "build-123",
		ScenarioName: "test-scenario",
		Status:       "building",
		Platforms:    []string{"linux"},
	}

	store.Save(status)

	got, ok := store.Get("build-123")
	if !ok {
		t.Fatalf("expected to find saved status")
	}
	if got.ScenarioName != "test-scenario" {
		t.Errorf("expected scenario name 'test-scenario', got %q", got.ScenarioName)
	}
}

func TestInMemoryStore_GetNonExistent(t *testing.T) {
	store := NewStore()

	_, ok := store.Get("nonexistent")
	if ok {
		t.Errorf("expected ok=false for nonexistent build")
	}
}

func TestInMemoryStore_Update(t *testing.T) {
	store := NewStore()

	status := &Status{
		BuildID: "build-123",
		Status:  "building",
	}
	store.Save(status)

	t.Run("update existing", func(t *testing.T) {
		updated := store.Update("build-123", func(s *Status) {
			s.Status = "ready"
			now := time.Now()
			s.CompletedAt = &now
		})
		if !updated {
			t.Errorf("expected Update to return true")
		}

		got, _ := store.Get("build-123")
		if got.Status != "ready" {
			t.Errorf("expected status 'ready', got %q", got.Status)
		}
		if got.CompletedAt == nil {
			t.Errorf("expected CompletedAt to be set")
		}
	})

	t.Run("update nonexistent", func(t *testing.T) {
		updated := store.Update("nonexistent", func(s *Status) {
			s.Status = "failed"
		})
		if updated {
			t.Errorf("expected Update to return false for nonexistent")
		}
	})
}

func TestInMemoryStore_UpdatePlatform(t *testing.T) {
	store := NewStore()

	status := &Status{
		BuildID: "build-123",
		Status:  "building",
		PlatformResults: map[string]*PlatformResult{
			"linux": {Platform: "linux", Status: "building"},
		},
	}
	store.Save(status)

	t.Run("update existing platform", func(t *testing.T) {
		updated := store.UpdatePlatform("build-123", "linux", func(s *Status, r *PlatformResult) {
			r.Status = "ready"
			r.Artifact = "/path/to/artifact"
		})
		if !updated {
			t.Errorf("expected UpdatePlatform to return true")
		}

		got, _ := store.Get("build-123")
		if got.PlatformResults["linux"].Status != "ready" {
			t.Errorf("expected platform status 'ready'")
		}
		if got.PlatformResults["linux"].Artifact != "/path/to/artifact" {
			t.Errorf("expected artifact path to be set")
		}
	})

	t.Run("update nonexistent platform creates default", func(t *testing.T) {
		updated := store.UpdatePlatform("build-123", "darwin", func(s *Status, r *PlatformResult) {
			// Platform should be auto-created with failed status
		})
		if !updated {
			t.Errorf("expected UpdatePlatform to return true")
		}

		got, _ := store.Get("build-123")
		darwinResult := got.PlatformResults["darwin"]
		if darwinResult == nil {
			t.Fatalf("expected darwin platform to be created")
		}
		if darwinResult.Status != "failed" {
			t.Errorf("expected auto-created platform to have failed status")
		}
		if darwinResult.SkipReason != "platform not initialized" {
			t.Errorf("expected skip reason to be set")
		}
	})

	t.Run("update nonexistent build", func(t *testing.T) {
		updated := store.UpdatePlatform("nonexistent", "linux", func(s *Status, r *PlatformResult) {})
		if updated {
			t.Errorf("expected UpdatePlatform to return false for nonexistent build")
		}
	})
}

func TestInMemoryStore_Snapshot(t *testing.T) {
	store := NewStore()

	store.Save(&Status{BuildID: "build-1", Status: "ready"})
	store.Save(&Status{BuildID: "build-2", Status: "building"})

	snapshot := store.Snapshot()
	if len(snapshot) != 2 {
		t.Errorf("expected 2 items in snapshot, got %d", len(snapshot))
	}

	// Verify snapshot is independent
	delete(snapshot, "build-1")
	if store.Len() != 2 {
		t.Errorf("deleting from snapshot shouldn't affect store")
	}
}

func TestInMemoryStore_Len(t *testing.T) {
	store := NewStore()

	if store.Len() != 0 {
		t.Errorf("expected initial length 0")
	}

	store.Save(&Status{BuildID: "build-1"})
	store.Save(&Status{BuildID: "build-2"})

	if store.Len() != 2 {
		t.Errorf("expected length 2, got %d", store.Len())
	}
}

func TestInMemoryStore_Concurrency(t *testing.T) {
	store := NewStore()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			buildID := "build-" + string(rune('a'+i%26))
			store.Save(&Status{BuildID: buildID, Status: "building"})
			store.Get(buildID)
			store.Update(buildID, func(s *Status) {
				s.Status = "ready"
			})
			store.Snapshot()
		}(i)
	}
	wg.Wait()

	// Should not panic or corrupt data
	snapshot := store.Snapshot()
	for _, status := range snapshot {
		if status.BuildID == "" {
			t.Errorf("found corrupted status entry")
		}
	}
}

func TestStatus_Fields(t *testing.T) {
	now := time.Now()
	status := &Status{
		BuildID:            "build-123",
		ScenarioName:       "test-scenario",
		Status:             "building",
		Framework:          "electron",
		TemplateType:       "universal",
		Platforms:          []string{"linux", "darwin"},
		RequestedPlatforms: []string{"linux", "darwin", "win32"},
		PlatformResults: map[string]*PlatformResult{
			"linux": {
				Platform:  "linux",
				Status:    "ready",
				StartedAt: &now,
				Artifact:  "/path/to/artifact",
				FileSize:  1024,
			},
		},
		OutputPath: "/output/path",
		CreatedAt:  now,
		ErrorLog:   []string{"error1"},
		BuildLog:   []string{"log1", "log2"},
		Artifacts:  map[string]string{"linux": "/path/to/linux.tar.gz"},
		Metadata:   map[string]interface{}{"key": "value"},
	}

	if status.BuildID != "build-123" {
		t.Errorf("expected BuildID 'build-123'")
	}
	if status.Framework != "electron" {
		t.Errorf("expected Framework 'electron'")
	}
	if len(status.Platforms) != 2 {
		t.Errorf("expected 2 platforms")
	}
	if len(status.RequestedPlatforms) != 3 {
		t.Errorf("expected 3 requested platforms")
	}
	if status.PlatformResults["linux"].FileSize != 1024 {
		t.Errorf("expected file size 1024")
	}
}

func TestPlatformResult_Fields(t *testing.T) {
	now := time.Now()
	result := &PlatformResult{
		Platform:    "darwin",
		Status:      "skipped",
		StartedAt:   &now,
		CompletedAt: &now,
		ErrorLog:    []string{"wine not installed"},
		Artifact:    "",
		FileSize:    0,
		SkipReason:  "Wine not installed for Windows builds on macOS",
	}

	if result.Platform != "darwin" {
		t.Errorf("expected Platform 'darwin'")
	}
	if result.SkipReason == "" {
		t.Errorf("expected SkipReason to be set")
	}
}

// Service tests

type mockWineChecker struct {
	installed bool
}

func (m *mockWineChecker) IsWineInstalled() bool {
	return m.installed
}

type mockPackageFinder struct {
	path string
	err  error
}

func (m *mockPackageFinder) FindBuiltPackage(distPath, platform string) (string, error) {
	return m.path, m.err
}

type mockLogger struct {
	infoCalls  int
	warnCalls  int
	errorCalls int
}

func (m *mockLogger) Info(msg string, args ...interface{})  { m.infoCalls++ }
func (m *mockLogger) Warn(msg string, args ...interface{})  { m.warnCalls++ }
func (m *mockLogger) Error(msg string, args ...interface{}) { m.errorCalls++ }

type scriptedRun struct {
	command string
	output  string
	err     error
}

type mockCommandRunner struct {
	mu    sync.Mutex
	runs  []scriptedRun
	calls []string
	dirs  []string
}

func (m *mockCommandRunner) Run(dir, command string) ([]byte, error) {
	return m.RunWithEnv(dir, command, nil)
}

func (m *mockCommandRunner) RunWithEnv(dir, command string, _ map[string]string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, command)
	m.dirs = append(m.dirs, dir)

	if len(m.runs) == 0 {
		return nil, errors.New("unexpected command run: no scripted output available")
	}
	next := m.runs[0]
	m.runs = m.runs[1:]
	if next.command != "" && next.command != command {
		return nil, errors.New("unexpected command run: expected " + next.command + ", got " + command)
	}
	return []byte(next.output), next.err
}

func TestNewService(t *testing.T) {
	service := NewService()
	if service == nil {
		t.Fatalf("expected service to be created")
	}
}

func TestNewServiceWithOptions(t *testing.T) {
	store := NewStore()
	wineChecker := &mockWineChecker{installed: true}
	packageFinder := &mockPackageFinder{path: "/path/to/package"}
	logger := &mockLogger{}

	service := NewService(
		WithStore(store),
		WithWineChecker(wineChecker),
		WithPackageFinder(packageFinder),
		WithLogger(logger),
	)

	if service == nil {
		t.Fatalf("expected service to be created")
	}
	if service.store != store {
		t.Errorf("expected custom store to be used")
	}
	if service.wineChecker != wineChecker {
		t.Errorf("expected custom wine checker to be used")
	}
	if service.packageFinder != packageFinder {
		t.Errorf("expected custom package finder to be used")
	}
	if service.logger != logger {
		t.Errorf("expected custom logger to be used")
	}
}

// createTestDir creates a subdirectory under parent and returns its path.
func createTestDir(t *testing.T, parent, name string) string {
	t.Helper()
	dir := filepath.Join(parent, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create dir %s: %v", name, err)
	}
	return dir
}

// createTestFile creates a file with dummy content and returns its path.
func createTestFile(t *testing.T, dir, name string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("test"), 0o644); err != nil {
		t.Fatalf("failed to create test file %s: %v", name, err)
	}
	return p
}

// assertFindsPackage asserts that FindBuiltPackage returns the expected path without error.
func assertFindsPackage(t *testing.T, finder *defaultPackageFinder, distPath, platform, wantPath string) {
	t.Helper()
	path, err := finder.FindBuiltPackage(distPath, platform)
	if err != nil {
		t.Fatalf("FindBuiltPackage error: %v", err)
	}
	if path != wantPath {
		t.Errorf("expected %q, got %q", wantPath, path)
	}
}

// assertFindPackageError asserts that FindBuiltPackage returns an error.
func assertFindPackageError(t *testing.T, finder *defaultPackageFinder, distPath, platform string) {
	t.Helper()
	_, err := finder.FindBuiltPackage(distPath, platform)
	if err == nil {
		t.Errorf("expected error for FindBuiltPackage(%q, %q)", distPath, platform)
	}
}

func TestDefaultPackageFinder_FindBuiltPackage(t *testing.T) {
	finder := &defaultPackageFinder{}
	tmpDir := t.TempDir()

	t.Run("nonexistent dist dir", func(t *testing.T) {
		assertFindPackageError(t, finder, "/nonexistent", "linux")
	})

	t.Run("unknown platform", func(t *testing.T) {
		assertFindPackageError(t, finder, tmpDir, "unknown")
	})

	t.Run("finds linux AppImage", func(t *testing.T) {
		appImage := createTestFile(t, tmpDir, "MyApp.AppImage")
		assertFindsPackage(t, finder, tmpDir, "linux", appImage)
	})

	t.Run("finds linux deb", func(t *testing.T) {
		debDir := createTestDir(t, tmpDir, "deb-test")
		debFile := createTestFile(t, debDir, "myapp.deb")
		assertFindsPackage(t, finder, debDir, "linux", debFile)
	})

	t.Run("finds mac dmg", func(t *testing.T) {
		macDir := createTestDir(t, tmpDir, "mac-test")
		dmgFile := createTestFile(t, macDir, "MyApp.dmg")
		assertFindsPackage(t, finder, macDir, "mac", dmgFile)
	})

	t.Run("finds win exe", func(t *testing.T) {
		winDir := createTestDir(t, tmpDir, "win-test")
		exeFile := createTestFile(t, winDir, "MyApp Setup.exe")
		assertFindsPackage(t, finder, winDir, "win", exeFile)
	})

	t.Run("prefers msi over exe", func(t *testing.T) {
		msiDir := createTestDir(t, tmpDir, "msi-test")
		createTestFile(t, msiDir, "MyApp.exe")
		msiFile := createTestFile(t, msiDir, "MyApp.msi")
		assertFindsPackage(t, finder, msiDir, "win", msiFile)
	})

	t.Run("prefers pkg over dmg", func(t *testing.T) {
		pkgDir := createTestDir(t, tmpDir, "pkg-test")
		createTestFile(t, pkgDir, "MyApp.dmg")
		pkgFile := createTestFile(t, pkgDir, "MyApp.pkg")
		assertFindsPackage(t, finder, pkgDir, "mac", pkgFile)
	})

	t.Run("no packages found", func(t *testing.T) {
		emptyDir := createTestDir(t, tmpDir, "empty")
		assertFindPackageError(t, finder, emptyDir, "linux")
	})
}
