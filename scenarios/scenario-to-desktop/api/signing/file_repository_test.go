package signing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"scenario-to-desktop-api/signing/mocks"
	"scenario-to-desktop-api/signing/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileRepository_Get_Success(t *testing.T) {
	fs := mocks.NewMockFileSystem()
	locator := mocks.NewMockScenarioLocator()
	locator.AddScenario("test-scenario", "/home/user/Vrooli/scenarios/test-scenario")

	config := &types.SigningConfig{
		SchemaVersion: "1.0",
		Enabled:       true,
		Windows: &types.WindowsSigningConfig{
			CertificateSource: types.CertSourceFile,
			CertificateFile:   "./cert.pfx",
		},
	}
	configBytes, _ := json.MarshalIndent(config, "", "  ")
	fs.AddFile("/home/user/Vrooli/scenarios/test-scenario/signing.json", configBytes)

	repo := NewFileRepository(WithFileSystemRepo(fs), WithScenarioLocator(locator))

	result, err := repo.Get(context.Background(), "test-scenario")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Enabled)
	assert.NotNil(t, result.Windows)
	assert.Equal(t, "./cert.pfx", result.Windows.CertificateFile)
}

func TestFileRepository_Get_NotFound(t *testing.T) {
	fs := mocks.NewMockFileSystem()
	locator := mocks.NewMockScenarioLocator()
	locator.AddScenario("test-scenario", "/home/user/Vrooli/scenarios/test-scenario")

	repo := NewFileRepository(WithFileSystemRepo(fs), WithScenarioLocator(locator))

	result, err := repo.Get(context.Background(), "test-scenario")

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestFileRepository_Get_InvalidJSON(t *testing.T) {
	fs := mocks.NewMockFileSystem()
	locator := mocks.NewMockScenarioLocator()
	locator.AddScenario("test-scenario", "/home/user/Vrooli/scenarios/test-scenario")
	fs.AddFile("/home/user/Vrooli/scenarios/test-scenario/signing.json", []byte("not valid json"))

	repo := NewFileRepository(WithFileSystemRepo(fs), WithScenarioLocator(locator))

	_, err := repo.Get(context.Background(), "test-scenario")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse signing config")
}

func TestFileRepository_Save_Success(t *testing.T) {
	fs := mocks.NewMockFileSystem()
	locator := mocks.NewMockScenarioLocator()
	locator.AddScenario("test-scenario", "/home/user/Vrooli/scenarios/test-scenario")
	fs.AddDirectory("/home/user/Vrooli/scenarios/test-scenario")

	repo := NewFileRepository(WithFileSystemRepo(fs), WithScenarioLocator(locator))

	config := &types.SigningConfig{
		Enabled: true,
		Windows: &types.WindowsSigningConfig{
			CertificateSource: types.CertSourceFile,
			CertificateFile:   "./cert.pfx",
		},
	}

	err := repo.Save(context.Background(), "test-scenario", config)

	require.NoError(t, err)

	// Verify file was written
	savedData, err := fs.ReadFile("/home/user/Vrooli/scenarios/test-scenario/signing.json")
	require.NoError(t, err)

	var savedConfig types.SigningConfig
	err = json.Unmarshal(savedData, &savedConfig)
	require.NoError(t, err)
	assert.True(t, savedConfig.Enabled)
	assert.Equal(t, types.SchemaVersion, savedConfig.SchemaVersion)
}

func TestFileRepository_Save_SetsSchemaVersion(t *testing.T) {
	fs := mocks.NewMockFileSystem()
	locator := mocks.NewMockScenarioLocator()
	locator.AddScenario("test-scenario", "/home/user/Vrooli/scenarios/test-scenario")

	repo := NewFileRepository(WithFileSystemRepo(fs), WithScenarioLocator(locator))

	config := &types.SigningConfig{
		Enabled: true,
		// SchemaVersion not set
	}

	err := repo.Save(context.Background(), "test-scenario", config)

	require.NoError(t, err)

	// Verify schema version was set
	savedData, _ := fs.ReadFile("/home/user/Vrooli/scenarios/test-scenario/signing.json")
	var savedConfig types.SigningConfig
	require.NoError(t, json.Unmarshal(savedData, &savedConfig))
	assert.Equal(t, types.SchemaVersion, savedConfig.SchemaVersion)
}

func TestFileRepository_Delete_Success(t *testing.T) {
	fs := mocks.NewMockFileSystem()
	locator := mocks.NewMockScenarioLocator()
	locator.AddScenario("test-scenario", "/home/user/Vrooli/scenarios/test-scenario")

	configBytes, _ := json.Marshal(&types.SigningConfig{Enabled: true})
	fs.AddFile("/home/user/Vrooli/scenarios/test-scenario/signing.json", configBytes)

	repo := NewFileRepository(WithFileSystemRepo(fs), WithScenarioLocator(locator))

	err := repo.Delete(context.Background(), "test-scenario")

	require.NoError(t, err)

	// Verify file was deleted
	assert.False(t, fs.Exists("/home/user/Vrooli/scenarios/test-scenario/signing.json"))
}

func TestFileRepository_Delete_NotExists(t *testing.T) {
	fs := mocks.NewMockFileSystem()
	locator := mocks.NewMockScenarioLocator()
	locator.AddScenario("test-scenario", "/home/user/Vrooli/scenarios/test-scenario")

	repo := NewFileRepository(WithFileSystemRepo(fs), WithScenarioLocator(locator))

	err := repo.Delete(context.Background(), "test-scenario")

	// Deleting non-existent config should not error
	require.NoError(t, err)
}

func TestFileRepository_Exists_True(t *testing.T) {
	fs := mocks.NewMockFileSystem()
	locator := mocks.NewMockScenarioLocator()
	locator.AddScenario("test-scenario", "/home/user/Vrooli/scenarios/test-scenario")

	configBytes, _ := json.Marshal(&types.SigningConfig{Enabled: true})
	fs.AddFile("/home/user/Vrooli/scenarios/test-scenario/signing.json", configBytes)

	repo := NewFileRepository(WithFileSystemRepo(fs), WithScenarioLocator(locator))

	exists, err := repo.Exists(context.Background(), "test-scenario")

	require.NoError(t, err)
	assert.True(t, exists)
}

func TestFileRepository_Exists_False(t *testing.T) {
	fs := mocks.NewMockFileSystem()
	locator := mocks.NewMockScenarioLocator()
	locator.AddScenario("test-scenario", "/home/user/Vrooli/scenarios/test-scenario")

	repo := NewFileRepository(WithFileSystemRepo(fs), WithScenarioLocator(locator))

	exists, err := repo.Exists(context.Background(), "test-scenario")

	require.NoError(t, err)
	assert.False(t, exists)
}

func TestFileRepository_GetPath(t *testing.T) {
	fs := mocks.NewMockFileSystem()
	locator := mocks.NewMockScenarioLocator()
	locator.AddScenario("test-scenario", "/home/user/Vrooli/scenarios/test-scenario")

	repo := NewFileRepository(WithFileSystemRepo(fs), WithScenarioLocator(locator))

	path := repo.GetPath("test-scenario")

	assert.Equal(t, "/home/user/Vrooli/scenarios/test-scenario/signing.json", path)
}

func TestFileRepository_GetPath_FallbackToRelative(t *testing.T) {
	fs := mocks.NewMockFileSystem()
	locator := mocks.NewMockScenarioLocator()
	locator.GetPathError = os.ErrNotExist

	repo := NewFileRepository(WithFileSystemRepo(fs), WithScenarioLocator(locator))

	path := repo.GetPath("test-scenario")

	assert.Equal(t, "scenarios/test-scenario/signing.json", path)
}

func TestFileRepositoryRejectsScenarioPathTraversal(t *testing.T) {
	root := filepath.Join(t.TempDir(), "repo")
	outside := filepath.Join(filepath.Dir(root), "outside")
	require.NoError(t, os.MkdirAll(outside, 0o700))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "scenarios"), 0o700))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "scenarios", "inside"), 0o700))
	require.NoError(t, os.Symlink(outside, filepath.Join(root, "scenarios", "linked-outside")))
	require.NoError(t, os.Symlink(filepath.Join(root, "scenarios", "inside"), filepath.Join(root, "scenarios", "linked-inside")))
	locator := &DefaultScenarioLocator{vrooliRoot: root}
	if path, err := locator.GetScenarioPath("../../outside"); err == nil {
		t.Fatalf("GetScenarioPath escaped to %q", path)
	}
	if path, err := locator.GetScenarioPath("linked-outside"); err == nil {
		t.Fatalf("GetScenarioPath followed an external scenario symlink to %q", path)
	}
	if path, err := locator.GetScenarioPath("linked-inside"); err == nil {
		t.Fatalf("GetScenarioPath changed scenario identity through an internal symlink to %q", path)
	}

	cases := []string{"../../outside", "../outside", "scenario/../../outside", "/tmp/outside", `..\outside`, ".", "..", "", "linked-outside", "linked-inside"}
	for _, scenario := range cases {
		t.Run(fmt.Sprintf("%q", scenario), func(t *testing.T) {
			for _, op := range []struct {
				name string
				run  func(*FileRepository) error
			}{
				{name: "get", run: func(repo *FileRepository) error { _, err := repo.Get(context.Background(), scenario); return err }},
				{name: "save", run: func(repo *FileRepository) error {
					return repo.Save(context.Background(), scenario, &types.SigningConfig{})
				}},
				{name: "delete", run: func(repo *FileRepository) error { return repo.Delete(context.Background(), scenario) }},
				{name: "exists", run: func(repo *FileRepository) error { _, err := repo.Exists(context.Background(), scenario); return err }},
			} {
				t.Run(op.name, func(t *testing.T) {
					fs := mocks.NewMockFileSystem()
					repo := NewFileRepository(WithFileSystemRepo(fs), WithScenarioLocator(locator))
					if path := repo.GetPath(scenario); path != "" {
						t.Fatalf("GetPath(%q) = %q, want empty for invalid scenario", scenario, path)
					}
					if err := op.run(repo); err == nil {
						t.Fatalf("operation accepted invalid scenario %q", scenario)
					}
					if len(fs.Files) != 0 || len(fs.Directories) != 0 {
						t.Fatalf("invalid scenario performed filesystem writes: files=%v directories=%v", fs.Files, fs.Directories)
					}
				})
			}
		})
	}
}

func TestFileRepository_GetForPlatform_Windows(t *testing.T) {
	fs := mocks.NewMockFileSystem()
	locator := mocks.NewMockScenarioLocator()
	locator.AddScenario("test-scenario", "/home/user/Vrooli/scenarios/test-scenario")

	config := &types.SigningConfig{
		Enabled: true,
		Windows: &types.WindowsSigningConfig{
			CertificateFile: "./cert.pfx",
		},
		MacOS: &types.MacOSSigningConfig{
			Identity: "Test",
		},
	}
	configBytes, _ := json.Marshal(config)
	fs.AddFile("/home/user/Vrooli/scenarios/test-scenario/signing.json", configBytes)

	repo := NewFileRepository(WithFileSystemRepo(fs), WithScenarioLocator(locator))

	result, err := repo.GetForPlatform(context.Background(), "test-scenario", types.PlatformWindows)

	require.NoError(t, err)
	winConfig, ok := result.(*types.WindowsSigningConfig)
	require.True(t, ok)
	assert.Equal(t, "./cert.pfx", winConfig.CertificateFile)
}

func TestFileRepository_GetForPlatform_NoConfig(t *testing.T) {
	fs := mocks.NewMockFileSystem()
	locator := mocks.NewMockScenarioLocator()
	locator.AddScenario("test-scenario", "/home/user/Vrooli/scenarios/test-scenario")

	repo := NewFileRepository(WithFileSystemRepo(fs), WithScenarioLocator(locator))

	result, err := repo.GetForPlatform(context.Background(), "test-scenario", types.PlatformWindows)

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestFileRepository_SaveForPlatform_Windows(t *testing.T) {
	fs := mocks.NewMockFileSystem()
	locator := mocks.NewMockScenarioLocator()
	locator.AddScenario("test-scenario", "/home/user/Vrooli/scenarios/test-scenario")

	repo := NewFileRepository(WithFileSystemRepo(fs), WithScenarioLocator(locator))

	winConfig := &types.WindowsSigningConfig{
		CertificateSource: types.CertSourceFile,
		CertificateFile:   "./cert.pfx",
	}

	err := repo.SaveForPlatform(context.Background(), "test-scenario", types.PlatformWindows, winConfig)

	require.NoError(t, err)

	// Verify Windows config was saved
	savedConfig, _ := repo.Get(context.Background(), "test-scenario")
	require.NotNil(t, savedConfig)
	require.NotNil(t, savedConfig.Windows)
	assert.Equal(t, "./cert.pfx", savedConfig.Windows.CertificateFile)
}

func TestFileRepository_SaveForPlatform_UpdatesExisting(t *testing.T) {
	fs := mocks.NewMockFileSystem()
	locator := mocks.NewMockScenarioLocator()
	locator.AddScenario("test-scenario", "/home/user/Vrooli/scenarios/test-scenario")

	// Pre-existing config
	existingConfig := &types.SigningConfig{
		Enabled: true,
		MacOS:   &types.MacOSSigningConfig{Identity: "Test"},
	}
	configBytes, _ := json.Marshal(existingConfig)
	fs.AddFile("/home/user/Vrooli/scenarios/test-scenario/signing.json", configBytes)

	repo := NewFileRepository(WithFileSystemRepo(fs), WithScenarioLocator(locator))

	winConfig := &types.WindowsSigningConfig{
		CertificateFile: "./cert.pfx",
	}

	err := repo.SaveForPlatform(context.Background(), "test-scenario", types.PlatformWindows, winConfig)

	require.NoError(t, err)

	// Verify both Windows and macOS configs exist
	savedConfig, _ := repo.Get(context.Background(), "test-scenario")
	require.NotNil(t, savedConfig)
	require.NotNil(t, savedConfig.Windows)
	require.NotNil(t, savedConfig.MacOS)
	assert.Equal(t, "Test", savedConfig.MacOS.Identity)
}

func TestFileRepository_DeleteForPlatform_RemovesOnlyPlatform(t *testing.T) {
	fs := mocks.NewMockFileSystem()
	locator := mocks.NewMockScenarioLocator()
	locator.AddScenario("test-scenario", "/home/user/Vrooli/scenarios/test-scenario")

	config := &types.SigningConfig{
		Enabled: true,
		Windows: &types.WindowsSigningConfig{CertificateFile: "./cert.pfx"},
		MacOS:   &types.MacOSSigningConfig{Identity: "Test"},
	}
	configBytes, _ := json.Marshal(config)
	fs.AddFile("/home/user/Vrooli/scenarios/test-scenario/signing.json", configBytes)

	repo := NewFileRepository(WithFileSystemRepo(fs), WithScenarioLocator(locator))

	err := repo.DeleteForPlatform(context.Background(), "test-scenario", types.PlatformWindows)

	require.NoError(t, err)

	// Windows should be gone, macOS should remain
	savedConfig, _ := repo.Get(context.Background(), "test-scenario")
	require.NotNil(t, savedConfig)
	assert.Nil(t, savedConfig.Windows)
	assert.NotNil(t, savedConfig.MacOS)
}

func TestFileRepository_DeleteForPlatform_DeletesEntireConfigIfLastPlatform(t *testing.T) {
	fs := mocks.NewMockFileSystem()
	locator := mocks.NewMockScenarioLocator()
	locator.AddScenario("test-scenario", "/home/user/Vrooli/scenarios/test-scenario")

	config := &types.SigningConfig{
		Enabled: true,
		Windows: &types.WindowsSigningConfig{CertificateFile: "./cert.pfx"},
	}
	configBytes, _ := json.Marshal(config)
	fs.AddFile("/home/user/Vrooli/scenarios/test-scenario/signing.json", configBytes)

	repo := NewFileRepository(WithFileSystemRepo(fs), WithScenarioLocator(locator))

	err := repo.DeleteForPlatform(context.Background(), "test-scenario", types.PlatformWindows)

	require.NoError(t, err)

	// Entire config should be deleted since no platforms remain
	assert.False(t, fs.Exists("/home/user/Vrooli/scenarios/test-scenario/signing.json"))
}

func TestDefaultScenarioLocator_GetScenarioPath(t *testing.T) {
	// This test requires VROOLI_ROOT to be set for the locator to work
	// Skip if running in an environment without it
	locator := NewDefaultScenarioLocator()
	if locator.vrooliRoot == "" {
		t.Skip("VROOLI_ROOT not set, skipping real filesystem test")
	}

	// Test with a scenario that should exist in the test environment
	// This will only pass if the scenario actually exists
	_, err := locator.GetScenarioPath("scenario-to-desktop")
	assert.NoError(t, err)
}

func TestDefaultScenarioLocator_ListScenarios(t *testing.T) {
	locator := NewDefaultScenarioLocator()
	if locator.vrooliRoot == "" {
		t.Skip("VROOLI_ROOT not set, skipping real filesystem test")
	}

	scenarios, err := locator.ListScenarios()

	require.NoError(t, err)
	assert.NotEmpty(t, scenarios)
	// scenario-to-desktop should be in the list
	assert.Contains(t, scenarios, "scenario-to-desktop")
}
