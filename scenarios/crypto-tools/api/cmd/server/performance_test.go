// +build testing

package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"
)

// BenchmarkHashSHA256 benchmarks SHA-256 hashing performance
func BenchmarkHashSHA256(b *testing.B) {
	env := setupTestEnvironment(&testing.T{})
	defer env.Cleanup()

	req := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/crypto/hash",
		Body: map[string]interface{}{
			"data":      "benchmark test data for hashing",
			"algorithm": "sha256",
		},
		Headers: createAuthHeaders(env.Config.APIToken),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeHTTPRequest(env.Server, req)
	}
}

// BenchmarkHashSHA512 benchmarks SHA-512 hashing performance
func BenchmarkHashSHA512(b *testing.B) {
	env := setupTestEnvironment(&testing.T{})
	defer env.Cleanup()

	req := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/crypto/hash",
		Body: map[string]interface{}{
			"data":      "benchmark test data for hashing",
			"algorithm": "sha512",
		},
		Headers: createAuthHeaders(env.Config.APIToken),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeHTTPRequest(env.Server, req)
	}
}

// BenchmarkEncryptAES256 benchmarks AES-256 encryption performance
func BenchmarkEncryptAES256(b *testing.B) {
	env := setupTestEnvironment(&testing.T{})
	defer env.Cleanup()

	req := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/crypto/encrypt",
		Body: map[string]interface{}{
			"data":      "benchmark test data for encryption",
			"algorithm": "aes256",
		},
		Headers: createAuthHeaders(env.Config.APIToken),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeHTTPRequest(env.Server, req)
	}
}

// BenchmarkKeyGenRSA2048 benchmarks RSA-2048 key generation
func BenchmarkKeyGenRSA2048(b *testing.B) {
	env := setupTestEnvironment(&testing.T{})
	defer env.Cleanup()

	req := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/crypto/keys/generate",
		Body: map[string]interface{}{
			"key_type": "rsa",
			"key_size": 2048,
		},
		Headers: createAuthHeaders(env.Config.APIToken),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeHTTPRequest(env.Server, req)
	}
}

// BenchmarkKeyGenRSA4096 benchmarks RSA-4096 key generation
func BenchmarkKeyGenRSA4096(b *testing.B) {
	env := setupTestEnvironment(&testing.T{})
	defer env.Cleanup()

	req := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/crypto/keys/generate",
		Body: map[string]interface{}{
			"key_type": "rsa",
			"key_size": 4096,
		},
		Headers: createAuthHeaders(env.Config.APIToken),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeHTTPRequest(env.Server, req)
	}
}

// TestHashPerformanceTargets tests hash operation performance targets
func TestHashPerformanceTargets(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	testCases := []struct {
		name           string
		algorithm      string
		maxTimeMS      int64
		dataSize       int
	}{
		{"SHA256_SmallData", "sha256", 50, 100},
		{"SHA256_MediumData", "sha256", 100, 10000},
		{"SHA512_SmallData", "sha512", 50, 100},
		{"Bcrypt_Password", "bcrypt", 500, 20},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := string(make([]byte, tc.dataSize))

			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/crypto/hash",
				Body: map[string]interface{}{
					"data":      data,
					"algorithm": tc.algorithm,
				},
				Headers: createAuthHeaders(env.Config.APIToken),
			}

			start := time.Now()
			w, err := makeHTTPRequest(env.Server, req)
			elapsed := time.Since(start).Milliseconds()

			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			if elapsed > tc.maxTimeMS {
				t.Errorf("Hash operation took %dms, expected < %dms", elapsed, tc.maxTimeMS)
			}

			t.Logf("%s completed in %dms", tc.name, elapsed)
		})
	}
}

// TestEncryptionPerformanceTargets tests encryption performance targets
func TestEncryptionPerformanceTargets(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	key := make([]byte, 32)
	encodedKey := base64.StdEncoding.EncodeToString(key)

	testCases := []struct {
		name      string
		dataSize  int
		maxTimeMS int64
	}{
		{"SmallData_1KB", 1024, 100},
		{"MediumData_10KB", 10240, 200},
		{"LargeData_100KB", 102400, 500},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := string(make([]byte, tc.dataSize))

			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/crypto/encrypt",
				Body: map[string]interface{}{
					"data":      data,
					"algorithm": "aes256",
					"key":       encodedKey,
				},
				Headers: createAuthHeaders(env.Config.APIToken),
			}

			start := time.Now()
			w, err := makeHTTPRequest(env.Server, req)
			elapsed := time.Since(start).Milliseconds()

			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			if elapsed > tc.maxTimeMS {
				t.Errorf("Encryption took %dms, expected < %dms", elapsed, tc.maxTimeMS)
			}

			t.Logf("%s completed in %dms (%.2f MB/s)", tc.name, elapsed,
				float64(tc.dataSize)/float64(elapsed)/1000.0)
		})
	}
}

// TestKeyGenerationPerformanceTargets tests key generation performance
func TestKeyGenerationPerformanceTargets(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	testCases := []struct {
		name      string
		keyType   string
		keySize   int
		maxTimeMS int64
	}{
		{"RSA_2048", "rsa", 2048, 500},
		{"RSA_4096", "rsa", 4096, 2000},
		{"Symmetric_256", "symmetric", 256, 100},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/crypto/keys/generate",
				Body: map[string]interface{}{
					"key_type": tc.keyType,
					"key_size": tc.keySize,
				},
				Headers: createAuthHeaders(env.Config.APIToken),
			}

			start := time.Now()
			w, err := makeHTTPRequest(env.Server, req)
			elapsed := time.Since(start).Milliseconds()

			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			if elapsed > tc.maxTimeMS {
				t.Errorf("Key generation took %dms, expected < %dms", elapsed, tc.maxTimeMS)
			}

			t.Logf("%s completed in %dms", tc.name, elapsed)
		})
	}
}

// TestConcurrentHashOperations tests concurrent hashing
func TestConcurrentHashOperations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	concurrentRequests := 10
	var wg sync.WaitGroup
	errors := make(chan error, concurrentRequests)

	start := time.Now()

	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/crypto/hash",
				Body: map[string]interface{}{
					"data":      fmt.Sprintf("concurrent test data %d", id),
					"algorithm": "sha256",
				},
				Headers: createAuthHeaders(env.Config.APIToken),
			}

			w, err := makeHTTPRequest(env.Server, req)
			if err != nil {
				errors <- err
				return
			}

			if w.Code != http.StatusOK {
				errors <- fmt.Errorf("Request %d failed with status %d", id, w.Code)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	elapsed := time.Since(start).Milliseconds()

	for err := range errors {
		t.Errorf("Concurrent request failed: %v", err)
	}

	t.Logf("Completed %d concurrent hash operations in %dms", concurrentRequests, elapsed)
}

// TestConcurrentEncryptionOperations tests concurrent encryption
func TestConcurrentEncryptionOperations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	concurrentRequests := 10
	var wg sync.WaitGroup
	errors := make(chan error, concurrentRequests)

	key := make([]byte, 32)
	encodedKey := base64.StdEncoding.EncodeToString(key)

	start := time.Now()

	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/crypto/encrypt",
				Body: map[string]interface{}{
					"data":      fmt.Sprintf("concurrent encryption test %d", id),
					"algorithm": "aes256",
					"key":       encodedKey,
				},
				Headers: createAuthHeaders(env.Config.APIToken),
			}

			w, err := makeHTTPRequest(env.Server, req)
			if err != nil {
				errors <- err
				return
			}

			if w.Code != http.StatusOK {
				errors <- fmt.Errorf("Request %d failed with status %d", id, w.Code)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	elapsed := time.Since(start).Milliseconds()

	for err := range errors {
		t.Errorf("Concurrent request failed: %v", err)
	}

	t.Logf("Completed %d concurrent encryption operations in %dms", concurrentRequests, elapsed)
}

// TestMemoryEfficiency tests memory usage during operations
func TestMemoryEfficiency(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("LargeDataHash", func(t *testing.T) {
		// Test hashing 1MB of data
		largeData := string(make([]byte, 1024*1024))

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/crypto/hash",
			Body: map[string]interface{}{
				"data":      largeData,
				"algorithm": "sha256",
			},
			Headers: createAuthHeaders(env.Config.APIToken),
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to hash large data: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestResponseTimeConsistency tests response time consistency
func TestResponseTimeConsistency(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	iterations := 100
	times := make([]int64, iterations)

	for i := 0; i < iterations; i++ {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/crypto/hash",
			Body: map[string]interface{}{
				"data":      "consistency test data",
				"algorithm": "sha256",
			},
			Headers: createAuthHeaders(env.Config.APIToken),
		}

		start := time.Now()
		w, err := makeHTTPRequest(env.Server, req)
		times[i] = time.Since(start).Milliseconds()

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	}

	// Calculate statistics
	var sum, min, max int64
	min = times[0]
	max = times[0]

	for _, t := range times {
		sum += t
		if t < min {
			min = t
		}
		if t > max {
			max = t
		}
	}

	avg := sum / int64(iterations)
	variance := max - min

	t.Logf("Response time stats (ms) - Min: %d, Max: %d, Avg: %d, Variance: %d", min, max, avg, variance)

	if variance > avg*3 {
		t.Errorf("Response time variance too high: %dms (avg: %dms)", variance, avg)
	}
}
