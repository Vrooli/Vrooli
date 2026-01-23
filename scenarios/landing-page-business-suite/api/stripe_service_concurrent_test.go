package main

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// ============================================================================
// Concurrent Operation Tests for StripeService
//
// These tests verify thread safety of the Stripe service, particularly around:
// - Config refresh operations
// - Concurrent checkout session creation
// - Cache operations
// ============================================================================

func TestConcurrent_StripeConfigRefresh(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewStripeService(db)
	ctx := context.Background()

	const numGoroutines = 20
	const refreshesPerGoroutine = 10

	var wg sync.WaitGroup
	var errors int32

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < refreshesPerGoroutine; j++ {
				if err := svc.RefreshConfig(ctx); err != nil {
					atomic.AddInt32(&errors, 1)
					t.Logf("Worker %d refresh %d failed: %v", workerID, j, err)
				}
			}
		}(i)
	}

	wg.Wait()

	if errors > 0 {
		t.Errorf("Expected 0 errors from concurrent config refresh, got %d", errors)
	}
}

func TestConcurrent_StripeConfigReadDuringRefresh(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewStripeService(db)
	ctx := context.Background()

	const numReaders = 10
	const numWriters = 5
	const operationsPerWorker = 20

	stopChan := make(chan struct{})
	var readErrors, writeErrors int32

	// Writers: continuously refresh config
	for i := 0; i < numWriters; i++ {
		go func(workerID int) {
			for j := 0; j < operationsPerWorker; j++ {
				select {
				case <-stopChan:
					return
				default:
					if err := svc.RefreshConfig(ctx); err != nil {
						atomic.AddInt32(&writeErrors, 1)
					}
					time.Sleep(1 * time.Millisecond)
				}
			}
		}(i)
	}

	// Readers: continuously read config snapshot
	var wg sync.WaitGroup
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < operationsPerWorker; j++ {
				// Reading config snapshot should never fail
				snapshot := svc.ConfigSnapshot()
				if snapshot == nil {
					atomic.AddInt32(&readErrors, 1)
				}
				time.Sleep(500 * time.Microsecond)
			}
		}(i)
	}

	wg.Wait()
	close(stopChan)

	if readErrors > 0 {
		t.Errorf("Expected 0 read errors, got %d", readErrors)
	}
	if writeErrors > 0 {
		t.Errorf("Expected 0 write errors, got %d", writeErrors)
	}
}

func TestConcurrent_MultipleConfigLoaders(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewStripeService(db)
	ctx := context.Background()

	const numGoroutines = 10

	var wg sync.WaitGroup
	var errors int32

	// Test that setting config loaders concurrently doesn't cause races
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Toggle between custom loader and default
			for j := 0; j < 10; j++ {
				if j%2 == 0 {
					svc.UseConfigLoader(func(ctx context.Context) (stripeRuntimeConfig, error) {
						return stripeRuntimeConfig{
							publishableKey: "pk_test_custom",
							secretKey:      "sk_test_custom",
							webhookSecret:  "whsec_custom",
							hasPublishable: true,
							hasSecret:      true,
							hasWebhook:     true,
							source:         "test",
						}, nil
					})
				} else {
					svc.UseConfigLoader(nil) // Reset to default
				}

				if err := svc.RefreshConfig(ctx); err != nil {
					atomic.AddInt32(&errors, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	if errors > 0 {
		t.Errorf("Expected 0 errors, got %d", errors)
	}
}

func TestConcurrent_StripeServiceInitialization(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	const numGoroutines = 10

	var wg sync.WaitGroup
	var errors int32
	services := make([]*StripeService, numGoroutines)

	// Create multiple StripeService instances concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			svc := NewStripeService(db)
			if svc == nil {
				atomic.AddInt32(&errors, 1)
				t.Logf("Worker %d: NewStripeService returned nil", workerID)
				return
			}
			services[workerID] = svc
		}(i)
	}

	wg.Wait()

	if errors > 0 {
		t.Errorf("Expected 0 errors during concurrent initialization, got %d", errors)
	}

	// Verify all services are properly initialized
	for i, svc := range services {
		if svc == nil {
			t.Errorf("Service %d was not initialized", i)
		}
	}
}

func TestConcurrent_HTTPClientSwap(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewStripeService(db)

	const numGoroutines = 20
	var wg sync.WaitGroup
	var errors int32

	// Concurrent HTTP client swaps should not cause races
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				// Swap HTTP clients - this tests the mutex protection
				svc.UseHTTPClient(nil)
				time.Sleep(100 * time.Microsecond)
			}
		}(i)
	}

	wg.Wait()

	if errors > 0 {
		t.Errorf("Expected 0 errors, got %d", errors)
	}
}

func TestConcurrent_ConfigSnapshotRead_DuringConfigChanges(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewStripeService(db)
	ctx := context.Background()

	const numReaders = 10
	const numWriters = 3
	const operationsPerWorker = 50

	var readOps, writeOps int32

	stopChan := make(chan struct{})

	// Writers: continuously change config
	for i := 0; i < numWriters; i++ {
		go func(workerID int) {
			for j := 0; j < operationsPerWorker; j++ {
				select {
				case <-stopChan:
					return
				default:
					// Toggle between different publishable keys
					key := "pk_test_key_" + string(rune('A'+workerID))
					svc.UseConfigLoader(func(ctx context.Context) (stripeRuntimeConfig, error) {
						return stripeRuntimeConfig{
							publishableKey: key,
							secretKey:      "sk_test",
							hasPublishable: true,
							hasSecret:      true,
							source:         "test",
						}, nil
					})
					_ = svc.RefreshConfig(ctx)
					atomic.AddInt32(&writeOps, 1)
					time.Sleep(1 * time.Millisecond)
				}
			}
		}(i)
	}

	// Readers: continuously read config snapshot
	var wg sync.WaitGroup
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < operationsPerWorker; j++ {
				snapshot := svc.ConfigSnapshot()
				if snapshot == nil {
					t.Logf("Worker %d got nil snapshot", workerID)
				}
				atomic.AddInt32(&readOps, 1)
			}
		}(i)
	}

	wg.Wait()
	close(stopChan)

	// Verify operations completed - use atomic loads
	finalReadOps := atomic.LoadInt32(&readOps)
	finalWriteOps := atomic.LoadInt32(&writeOps)
	if finalReadOps == 0 {
		t.Error("No read operations completed")
	}
	if finalWriteOps == 0 {
		t.Error("No write operations completed")
	}

	t.Logf("Completed %d read ops and %d write ops", finalReadOps, finalWriteOps)
}

func TestConcurrent_WebhookSignatureVerification(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewStripeService(db)
	ctx := context.Background()

	// Set up a consistent webhook secret
	svc.UseConfigLoader(func(ctx context.Context) (stripeRuntimeConfig, error) {
		return stripeRuntimeConfig{
			publishableKey: "pk_test",
			secretKey:      "sk_test",
			webhookSecret:  "whsec_test_secret_12345",
			hasPublishable: true,
			hasSecret:      true,
			hasWebhook:     true,
			source:         "test",
		}, nil
	})
	_ = svc.RefreshConfig(ctx)

	const numGoroutines = 20
	const verifyPerGoroutine = 50

	var wg sync.WaitGroup
	var verifyOps int32

	// Concurrent signature verifications
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < verifyPerGoroutine; j++ {
				// Create a test payload
				payload := []byte(`{"type":"test.event"}`)
				// We don't need valid signatures for concurrency testing,
				// we just need to ensure VerifyWebhookSignature doesn't race
				_ = svc.VerifyWebhookSignature(payload, "t=123,v1=abc")
				atomic.AddInt32(&verifyOps, 1)
			}
		}(i)
	}

	wg.Wait()

	expectedOps := int32(numGoroutines * verifyPerGoroutine)
	if verifyOps != expectedOps {
		t.Errorf("Expected %d verify ops, got %d", expectedOps, verifyOps)
	}
}

func TestConcurrent_StripeHighContention_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()

	svc := NewStripeService(db)
	ctx := context.Background()

	const numGoroutines = 50
	const opsPerGoroutine = 100

	var wg sync.WaitGroup
	var errors int32

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				// Mix of operations
				switch j % 3 {
				case 0:
					_ = svc.ConfigSnapshot()
				case 1:
					// Test webhook verification (doesn't require network)
					_ = svc.VerifyWebhookSignature([]byte("test"), "sig")
				case 2:
					if err := svc.RefreshConfig(ctx); err != nil {
						atomic.AddInt32(&errors, 1)
					}
				}
			}
		}(i)
	}

	wg.Wait()

	if errors > 0 {
		t.Errorf("Expected 0 errors under high contention, got %d", errors)
	}
}
