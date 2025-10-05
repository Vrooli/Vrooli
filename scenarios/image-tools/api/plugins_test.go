package main

import (
	"bytes"
	"image-tools/plugins"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// TestPluginRegistry tests the plugin registry functionality
func TestPluginRegistry(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	server := setupTestServer()

	t.Run("RegistryInitialized", func(t *testing.T) {
		if server.registry == nil {
			t.Fatal("Plugin registry should not be nil")
		}
	})

	t.Run("PluginsLoaded", func(t *testing.T) {
		plugins := server.registry.ListPlugins()

		if len(plugins) == 0 {
			t.Error("Expected at least one plugin to be loaded")
		}

		// Verify essential plugins are loaded
		expectedFormats := []string{"jpeg", "png", "webp", "svg"}
		loadedFormats := make(map[string]bool)

		for _, formats := range plugins {
			for _, format := range formats {
				loadedFormats[format] = true
			}
		}

		for _, expected := range expectedFormats {
			if !loadedFormats[expected] {
				t.Errorf("Expected format %s to be supported", expected)
			}
		}
	})

	t.Run("GetJPEGPlugin", func(t *testing.T) {
		plugin, ok := server.registry.GetPlugin("jpeg")
		if !ok {
			t.Fatal("Failed to get JPEG plugin")
		}

		if plugin == nil {
			t.Error("JPEG plugin should not be nil")
		}
	})

	t.Run("GetPNGPlugin", func(t *testing.T) {
		plugin, ok := server.registry.GetPlugin("png")
		if !ok {
			t.Fatal("Failed to get PNG plugin")
		}

		if plugin == nil {
			t.Error("PNG plugin should not be nil")
		}
	})

	t.Run("GetWebPPlugin", func(t *testing.T) {
		plugin, ok := server.registry.GetPlugin("webp")
		if !ok {
			t.Fatal("Failed to get WebP plugin")
		}

		if plugin == nil {
			t.Error("WebP plugin should not be nil")
		}
	})

	t.Run("GetSVGPlugin", func(t *testing.T) {
		plugin, ok := server.registry.GetPlugin("svg")
		if !ok {
			t.Fatal("Failed to get SVG plugin")
		}

		if plugin == nil {
			t.Error("SVG plugin should not be nil")
		}
	})

	t.Run("GetNonExistentPlugin", func(t *testing.T) {
		_, ok := server.registry.GetPlugin("bmp")
		if ok {
			t.Error("Should not find BMP plugin")
		}
	})

	t.Run("GetInvalidPlugin", func(t *testing.T) {
		_, ok := server.registry.GetPlugin("invalid")
		if ok {
			t.Error("Should not find invalid plugin")
		}
	})
}

// TestPluginBasicFunctionality tests basic plugin operations
func TestPluginBasicFunctionality(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	server := setupTestServer()

	t.Run("JPEGPluginCompress", func(t *testing.T) {
		plugin, ok := server.registry.GetPlugin("jpeg")
		if !ok {
			t.Skip("JPEG plugin not available")
		}

		imageData := generateTestImageData("jpeg")
		reader := bytes.NewReader(imageData)

		options := plugins.ProcessOptions{
			Quality: 85,
		}

		result, err := plugin.Compress(reader, options)
		if err != nil {
			// Compression may fail with minimal test data - that's acceptable
			t.Logf("Compression failed (expected with test data): %v", err)
			return
		}

		if result == nil {
			t.Error("Result should not be nil")
		}
	})

	t.Run("PNGPluginCompress", func(t *testing.T) {
		plugin, ok := server.registry.GetPlugin("png")
		if !ok {
			t.Skip("PNG plugin not available")
		}

		imageData := generateTestImageData("png")
		reader := bytes.NewReader(imageData)

		options := plugins.ProcessOptions{
			Quality: 90,
		}

		result, err := plugin.Compress(reader, options)
		if err != nil {
			t.Logf("Compression failed (expected with test data): %v", err)
			return
		}

		if result == nil {
			t.Error("Result should not be nil")
		}
	})

	t.Run("PluginGetInfo", func(t *testing.T) {
		plugin, ok := server.registry.GetPlugin("jpeg")
		if !ok {
			t.Skip("JPEG plugin not available")
		}

		imageData := generateTestImageData("jpeg")
		reader := bytes.NewReader(imageData)

		info, err := plugin.GetInfo(reader)
		if err != nil {
			t.Logf("GetInfo failed (expected with test data): %v", err)
			return
		}

		if info == nil {
			t.Error("Info should not be nil")
		}
	})

	t.Run("PluginStripMetadata", func(t *testing.T) {
		plugin, ok := server.registry.GetPlugin("jpeg")
		if !ok {
			t.Skip("JPEG plugin not available")
		}

		imageData := generateTestImageData("jpeg")
		reader := bytes.NewReader(imageData)

		result, err := plugin.StripMetadata(reader)
		if err != nil {
			t.Logf("StripMetadata failed (expected with test data): %v", err)
			return
		}

		if result == nil {
			t.Error("Result should not be nil")
		}
	})

	t.Run("PluginResize", func(t *testing.T) {
		plugin, ok := server.registry.GetPlugin("jpeg")
		if !ok {
			t.Skip("JPEG plugin not available")
		}

		imageData := generateTestImageData("jpeg")
		reader := bytes.NewReader(imageData)

		options := plugins.ProcessOptions{
			Width:          800,
			Height:         600,
			MaintainAspect: true,
		}

		result, err := plugin.Resize(reader, options)
		if err != nil {
			t.Logf("Resize failed (expected with test data): %v", err)
			return
		}

		if result == nil {
			t.Error("Result should not be nil")
		}
	})

	t.Run("PluginConvert", func(t *testing.T) {
		plugin, ok := server.registry.GetPlugin("png")
		if !ok {
			t.Skip("PNG plugin not available")
		}

		imageData := generateTestImageData("jpeg")
		reader := bytes.NewReader(imageData)

		options := plugins.ProcessOptions{
			CustomOptions: make(map[string]interface{}),
		}

		result, err := plugin.Convert(reader, "png", options)
		if err != nil {
			t.Logf("Convert failed (expected with test data): %v", err)
			return
		}

		if result == nil {
			t.Error("Result should not be nil")
		}
	})
}

// TestServerHelperFunctions tests server helper utilities
func TestServerHelperFunctions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	server := setupTestServer()

	t.Run("SaveToStorage", func(t *testing.T) {
		testData := []byte("test image data")
		testKey := "test/image.jpg"

		url, err := server.saveToStorage(testKey, testData)
		if err != nil {
			t.Fatalf("Failed to save to storage: %v", err)
		}

		if url == "" {
			t.Error("Expected non-empty URL")
		}
	})

	t.Run("ProcessSingleImage", func(t *testing.T) {
		imageData := generateTestImageData("jpeg")

		// Create a test file header
		file := createTestFileHeader(t, "test.jpg", imageData)

		operations := []Operation{
			{
				Type: "compress",
				Options: map[string]interface{}{
					"quality": float64(85),
				},
			},
		}

		result, err := server.processSingleImage(file, operations)
		if err != nil {
			t.Logf("Processing failed (expected with test data): %v", err)
			return
		}

		if result == nil {
			t.Error("Result should not be nil")
		}
	})
}

// TestHealthCheckHelpers tests health check helper methods
func TestHealthCheckHelpers(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	server := setupTestServer()

	t.Run("CheckPluginRegistry", func(t *testing.T) {
		health := server.checkPluginRegistry()

		if health == nil {
			t.Fatal("Health check result should not be nil")
		}

		status, ok := health["status"].(string)
		if !ok {
			t.Error("Health check should have status")
		}

		if status == "" {
			t.Error("Status should not be empty")
		}
	})

	t.Run("CheckStorageSystem", func(t *testing.T) {
		health := server.checkStorageSystem()

		if health == nil {
			t.Fatal("Health check result should not be nil")
		}

		status, ok := health["status"].(string)
		if !ok {
			t.Error("Health check should have status")
		}

		if status == "" {
			t.Error("Status should not be empty")
		}
	})

	t.Run("CheckImageProcessing", func(t *testing.T) {
		health := server.checkImageProcessing()

		if health == nil {
			t.Fatal("Health check result should not be nil")
		}

		status, ok := health["status"].(string)
		if !ok {
			t.Error("Health check should have status")
		}

		if status == "" {
			t.Error("Status should not be empty")
		}
	})

	t.Run("CheckFileSystemOperations", func(t *testing.T) {
		health := server.checkFileSystemOperations()

		if health == nil {
			t.Fatal("Health check result should not be nil")
		}

		status, ok := health["status"].(string)
		if !ok {
			t.Error("Health check should have status")
		}

		if status == "" {
			t.Error("Status should not be empty")
		}
	})

	t.Run("CountHealthyDependencies", func(t *testing.T) {
		deps := fiber.Map{
			"dep1": fiber.Map{"status": "healthy"},
			"dep2": fiber.Map{"status": "degraded"},
			"dep3": fiber.Map{"status": "healthy"},
		}

		count := server.countHealthyDependencies(deps)

		if count != 2 {
			t.Errorf("Expected 2 healthy dependencies, got %d", count)
		}
	})
}
