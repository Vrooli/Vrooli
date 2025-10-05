package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestCompressionPerformance tests compression performance with various file sizes
func TestCompressionPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	server := &Server{
		config: &Config{
			Port:        "8080",
			APIToken:    "test-token",
			DatabaseURL: "postgres://test",
		},
		db: nil, // No database connection in tests
	}

	testCases := []struct {
		name        string
		fileCount   int
		fileSize    int64
		maxDuration time.Duration
	}{
		{"SmallFiles", 10, 1024, 5 * time.Second},           // 10 files of 1KB
		{"MediumFiles", 5, 1024 * 100, 10 * time.Second},    // 5 files of 100KB
		{"LargeFile", 1, 1024 * 1024, 15 * time.Second},     // 1 file of 1MB
		{"ManySmallFiles", 100, 512, 20 * time.Second},      // 100 files of 512B
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test files
			var files []string
			content := generateTestContent(int(tc.fileSize))

			for i := 0; i < tc.fileCount; i++ {
				filename := fmt.Sprintf("perf_test_%d.txt", i)
				filePath := createTestFile(t, env.TempDir, filename, content)
				files = append(files, filePath)
			}

			outputPath := filepath.Join(env.TempDir, "perf_output.zip")

			compressReq := CompressRequest{
				Files:         files,
				ArchiveFormat: "zip",
				OutputPath:    outputPath,
				CompLevel:     5,
			}

			body, _ := json.Marshal(compressReq)

			start := time.Now()

			req := httptest.NewRequest("POST", "/api/v1/files/compress", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			server.handleCompress(w, req)

			duration := time.Since(start)

			if w.Code != 200 {
				t.Fatalf("Compression failed with status %d: %s", w.Code, w.Body.String())
			}

			if duration > tc.maxDuration {
				t.Errorf("Compression took %v, expected less than %v", duration, tc.maxDuration)
			}

			t.Logf("✓ %s: Compressed %d files (%d bytes each) in %v", tc.name, tc.fileCount, tc.fileSize, duration)

			// Cleanup
			for _, file := range files {
				os.Remove(file)
			}
			os.Remove(outputPath)
		})
	}
}

// TestFileOperationPerformance tests file operation performance
func TestFileOperationPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	server := &Server{
		config: &Config{
			Port:        "8080",
			APIToken:    "test-token",
			DatabaseURL: "postgres://test",
		},
		db: nil, // No database connection in tests
	}

	t.Run("BulkCopy", func(t *testing.T) {
		maxDuration := 10 * time.Second
		fileCount := 50
		content := generateTestContent(1024) // 1KB per file

		start := time.Now()

		for i := 0; i < fileCount; i++ {
			sourceFile := createTestFile(t, env.TempDir, fmt.Sprintf("source_%d.txt", i), content)
			targetFile := filepath.Join(env.TempDir, fmt.Sprintf("target_%d.txt", i))

			fileOp := FileOperation{
				Operation: "copy",
				Source:    sourceFile,
				Target:    targetFile,
			}

			body, _ := json.Marshal(fileOp)
			req := httptest.NewRequest("POST", "/api/v1/files/operation", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			server.handleFileOperation(w, req)

			if w.Code != 200 {
				t.Fatalf("Copy operation %d failed with status %d", i, w.Code)
			}
		}

		duration := time.Since(start)

		if duration > maxDuration {
			t.Errorf("Bulk copy took %v, expected less than %v", duration, maxDuration)
		}

		t.Logf("✓ BulkCopy: Copied %d files in %v (avg %v per file)", fileCount, duration, duration/time.Duration(fileCount))
	})

	t.Run("LargeFileCopy", func(t *testing.T) {
		maxDuration := 5 * time.Second
		largeContent := make([]byte, 5*1024*1024) // 5MB
		for i := range largeContent {
			largeContent[i] = byte(i % 256)
		}

		sourceFile := filepath.Join(env.TempDir, "large_source.bin")
		if err := os.WriteFile(sourceFile, largeContent, 0644); err != nil {
			t.Fatalf("Failed to create large file: %v", err)
		}

		targetFile := filepath.Join(env.TempDir, "large_target.bin")

		start := time.Now()

		fileOp := FileOperation{
			Operation: "copy",
			Source:    sourceFile,
			Target:    targetFile,
		}

		body, _ := json.Marshal(fileOp)
		req := httptest.NewRequest("POST", "/api/v1/files/operation", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		server.handleFileOperation(w, req)

		duration := time.Since(start)

		if w.Code != 200 {
			t.Fatalf("Large file copy failed with status %d", w.Code)
		}

		if duration > maxDuration {
			t.Errorf("Large file copy took %v, expected less than %v", duration, maxDuration)
		}

		t.Logf("✓ LargeFileCopy: Copied 5MB file in %v", duration)
	})
}

// TestChecksumPerformance tests checksum calculation performance
func TestChecksumPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	server := &Server{
		config: &Config{
			Port:        "8080",
			APIToken:    "test-token",
			DatabaseURL: "postgres://test",
		},
		db: nil, // No database connection in tests
	}

	testCases := []struct {
		name        string
		fileSize    int64
		algorithm   string
		maxDuration time.Duration
	}{
		{"MD5_Small", 1024 * 10, "md5", 1 * time.Second},         // 10KB
		{"MD5_Medium", 1024 * 1024, "md5", 2 * time.Second},      // 1MB
		{"SHA1_Small", 1024 * 10, "sha1", 1 * time.Second},       // 10KB
		{"SHA256_Small", 1024 * 10, "sha256", 1 * time.Second},   // 10KB
		{"SHA256_Medium", 1024 * 1024, "sha256", 3 * time.Second}, // 1MB
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			content := generateTestContent(int(tc.fileSize))
			testFile := createTestFile(t, env.TempDir, "checksum_test.bin", content)

			checksumReq := map[string]interface{}{
				"files":     []string{testFile},
				"algorithm": tc.algorithm,
			}

			body, _ := json.Marshal(checksumReq)

			start := time.Now()

			req := httptest.NewRequest("POST", "/api/v1/files/checksum", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			server.handleChecksum(w, req)

			duration := time.Since(start)

			if w.Code != 200 {
				t.Fatalf("Checksum calculation failed with status %d: %s", w.Code, w.Body.String())
			}

			if duration > tc.maxDuration {
				t.Errorf("Checksum calculation took %v, expected less than %v", duration, tc.maxDuration)
			}

			t.Logf("✓ %s: Calculated %s checksum for %d bytes in %v", tc.name, tc.algorithm, tc.fileSize, duration)

			os.Remove(testFile)
		})
	}
}

// TestMetadataExtractionPerformance tests metadata extraction performance
func TestMetadataExtractionPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	server := &Server{
		config: &Config{
			Port:        "8080",
			APIToken:    "test-token",
			DatabaseURL: "postgres://test",
		},
		db: nil, // No database connection in tests
	}

	t.Run("BulkMetadata", func(t *testing.T) {
		maxDuration := 5 * time.Second
		fileCount := 100

		start := time.Now()

		for i := 0; i < fileCount; i++ {
			content := generateTestContent(512)
			testFile := createTestFile(t, env.TempDir, fmt.Sprintf("metadata_%d.txt", i), content)

			req := httptest.NewRequest("GET", "/api/v1/files/metadata?path="+testFile, nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()

			server.handleGetMetadata(w, req)

			if w.Code != 200 {
				t.Fatalf("Metadata extraction %d failed with status %d", i, w.Code)
			}

			os.Remove(testFile)
		}

		duration := time.Since(start)

		if duration > maxDuration {
			t.Errorf("Bulk metadata extraction took %v, expected less than %v", duration, maxDuration)
		}

		t.Logf("✓ BulkMetadata: Extracted metadata for %d files in %v (avg %v per file)",
			fileCount, duration, duration/time.Duration(fileCount))
	})
}

// TestConcurrentOperations tests concurrent file operations
func TestConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	server := &Server{
		config: &Config{
			Port:        "8080",
			APIToken:    "test-token",
			DatabaseURL: "postgres://test",
		},
		db: nil, // No database connection in tests
	}

	t.Run("ConcurrentCopies", func(t *testing.T) {
		maxDuration := 10 * time.Second
		operationCount := 20

		// Pre-create source files
		var sourceFiles []string
		content := generateTestContent(1024)
		for i := 0; i < operationCount; i++ {
			sourceFile := createTestFile(t, env.TempDir, fmt.Sprintf("concurrent_src_%d.txt", i), content)
			sourceFiles = append(sourceFiles, sourceFile)
		}

		start := time.Now()

		done := make(chan bool, operationCount)

		for i := 0; i < operationCount; i++ {
			go func(index int) {
				targetFile := filepath.Join(env.TempDir, fmt.Sprintf("concurrent_dst_%d.txt", index))

				fileOp := FileOperation{
					Operation: "copy",
					Source:    sourceFiles[index],
					Target:    targetFile,
				}

				body, _ := json.Marshal(fileOp)
				req := httptest.NewRequest("POST", "/api/v1/files/operation", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer test-token")
				w := httptest.NewRecorder()

				server.handleFileOperation(w, req)

				if w.Code != 200 {
					t.Errorf("Concurrent copy %d failed with status %d", index, w.Code)
				}

				done <- true
			}(i)
		}

		// Wait for all operations to complete
		for i := 0; i < operationCount; i++ {
			<-done
		}

		duration := time.Since(start)

		if duration > maxDuration {
			t.Errorf("Concurrent operations took %v, expected less than %v", duration, maxDuration)
		}

		t.Logf("✓ ConcurrentCopies: Completed %d concurrent copies in %v (avg %v per operation)",
			operationCount, duration, duration/time.Duration(operationCount))
	})
}

// BenchmarkCompress benchmarks compression operations
func BenchmarkCompress(b *testing.B) {
	env := setupTestDirectory(&testing.T{})
	defer env.Cleanup()

	server := &Server{
		config: &Config{
			Port:        "8080",
			APIToken:    "test-token",
			DatabaseURL: "postgres://test",
		},
		db: nil, // No database connection in tests
	}

	// Create test file
	content := generateTestContent(1024)
	testFile := createTestFile(&testing.T{}, env.TempDir, "bench_test.txt", content)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		outputPath := filepath.Join(env.TempDir, fmt.Sprintf("bench_output_%d.zip", i))

		compressReq := CompressRequest{
			Files:         []string{testFile},
			ArchiveFormat: "zip",
			OutputPath:    outputPath,
			CompLevel:     5,
		}

		body, _ := json.Marshal(compressReq)
		req := httptest.NewRequest("POST", "/api/v1/files/compress", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		server.handleCompress(w, req)

		os.Remove(outputPath)
	}
}

// BenchmarkChecksum benchmarks checksum calculation
func BenchmarkChecksum(b *testing.B) {
	env := setupTestDirectory(&testing.T{})
	defer env.Cleanup()

	server := &Server{
		config: &Config{
			Port:        "8080",
			APIToken:    "test-token",
			DatabaseURL: "postgres://test",
		},
		db: nil, // No database connection in tests
	}

	content := generateTestContent(10240) // 10KB
	testFile := createTestFile(&testing.T{}, env.TempDir, "bench_checksum.txt", content)

	checksumReq := map[string]interface{}{
		"files":     []string{testFile},
		"algorithm": "md5",
	}

	body, _ := json.Marshal(checksumReq)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/v1/files/checksum", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()

		server.handleChecksum(w, req)
	}
}
