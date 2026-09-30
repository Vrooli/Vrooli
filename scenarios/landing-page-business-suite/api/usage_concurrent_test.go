package main

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"landing-page-business-suite-api/internal/commerce"
)

// ============================================================================
// Concurrent Operation Tests
//
// LPBS is a shared web service handling concurrent requests from multiple
// BAS instances and users. These tests verify thread safety of the usage
// tracking system.
//
// Note: These tests use SQLite in-memory database for simplicity, but the
// actual UPSERT operations in PostgreSQL provide stronger atomicity guarantees.
// ============================================================================

func TestConcurrent_MultipleUsageReports_SameUser(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()
	const numGoroutines = 10
	const reportsPerGoroutine = 10

	var wg sync.WaitGroup
	var errors int32

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < reportsPerGoroutine; j++ {
				err := svc.RecordUsage(ctx, commerce.UsageReportRequest{
					UserIdentity: "concurrent-user@example.com",
					LimitKey:     "ai_credits",
					Amount:       1000,
					AppBundleKey: "test-app",
				})
				if err != nil {
					atomic.AddInt32(&errors, 1)
					t.Logf("Worker %d report %d failed: %v", workerID, j, err)
				}
			}
		}(i)
	}

	wg.Wait()

	if errors > 0 {
		t.Errorf("Expected 0 errors, got %d", errors)
	}

	// Verify total usage is correct
	usage, err := svc.GetUsage(ctx, "concurrent-user@example.com", "ai_credits", nil)
	if err != nil {
		t.Fatalf("GetUsage() returned error: %v", err)
	}

	expectedUsage := int64(numGoroutines * reportsPerGoroutine * 1000)
	if usage != expectedUsage {
		t.Errorf("Expected usage %d, got %d", expectedUsage, usage)
	}
}

func TestConcurrent_MultipleUsageReports_DifferentUsers(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()
	const numUsers = 5
	const reportsPerUser = 20

	var wg sync.WaitGroup
	var errors int32

	for i := 0; i < numUsers; i++ {
		wg.Add(1)
		userEmail := "user" + string(rune('0'+i)) + "@example.com"
		go func(email string, userID int) {
			defer wg.Done()
			for j := 0; j < reportsPerUser; j++ {
				err := svc.RecordUsage(ctx, commerce.UsageReportRequest{
					UserIdentity: email,
					LimitKey:     "ai_credits",
					Amount:       1000,
					AppBundleKey: "test-app",
				})
				if err != nil {
					atomic.AddInt32(&errors, 1)
					t.Logf("User %d report %d failed: %v", userID, j, err)
				}
			}
		}(userEmail, i)
	}

	wg.Wait()

	if errors > 0 {
		t.Errorf("Expected 0 errors, got %d", errors)
	}

	// Verify each user has correct usage
	for i := 0; i < numUsers; i++ {
		userEmail := "user" + string(rune('0'+i)) + "@example.com"
		usage, err := svc.GetUsage(ctx, userEmail, "ai_credits", nil)
		if err != nil {
			t.Fatalf("GetUsage() for %s returned error: %v", userEmail, err)
		}

		expectedUsage := int64(reportsPerUser * 1000)
		if usage != expectedUsage {
			t.Errorf("User %s: expected usage %d, got %d", userEmail, expectedUsage, usage)
		}
	}
}

func TestConcurrent_CheckLimit_RaceCondition(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	seedTestUsageTierLimits(t, db)

	ctx := context.Background()
	const numGoroutines = 10

	// Start background goroutines that continuously record usage
	stopRecording := make(chan struct{})
	var recordErrors int32

	go func() {
		for {
			select {
			case <-stopRecording:
				return
			default:
				err := svc.RecordUsage(ctx, commerce.UsageReportRequest{
					UserIdentity: "race-test@example.com",
					LimitKey:     "ai_credits",
					Amount:       1000,
					AppBundleKey: "test-app",
				})
				if err != nil {
					atomic.AddInt32(&recordErrors, 1)
				}
				time.Sleep(1 * time.Millisecond) // Small delay to allow interleaving
			}
		}
	}()

	// Concurrent limit checks
	var wg sync.WaitGroup
	var checkErrors int32

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_, _, err := svc.CheckLimit(ctx, "race-test@example.com", "solo", "ai_credits", 1000)
				if err != nil {
					atomic.AddInt32(&checkErrors, 1)
					t.Logf("Worker %d check %d failed: %v", workerID, j, err)
				}
			}
		}(i)
	}

	wg.Wait()
	close(stopRecording)

	if checkErrors > 0 {
		t.Errorf("Expected 0 check errors, got %d", checkErrors)
	}
	if recordErrors > 0 {
		t.Errorf("Expected 0 record errors, got %d", recordErrors)
	}
}

func TestConcurrent_RecordUsage_AtomicUpsert(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()
	const numGoroutines = 20
	const amountPerReport = int64(5000)

	var wg sync.WaitGroup
	var errors int32

	// All goroutines try to upsert the same record simultaneously
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			err := svc.RecordUsage(ctx, commerce.UsageReportRequest{
				UserIdentity: "atomic-test@example.com",
				LimitKey:     "ai_credits",
				Amount:       amountPerReport,
				AppBundleKey: "same-app",
			})
			if err != nil {
				atomic.AddInt32(&errors, 1)
				t.Logf("Worker %d failed: %v", workerID, err)
			}
		}(i)
	}

	wg.Wait()

	if errors > 0 {
		t.Errorf("Expected 0 errors from concurrent upserts, got %d", errors)
	}

	// Verify the final amount is correct (all upserts should have succeeded)
	usage, err := svc.GetUsage(ctx, "atomic-test@example.com", "ai_credits", nil)
	if err != nil {
		t.Fatalf("GetUsage() returned error: %v", err)
	}

	expectedUsage := int64(numGoroutines) * amountPerReport
	if usage != expectedUsage {
		t.Errorf("Expected usage %d, got %d (missing %d)", expectedUsage, usage, expectedUsage-usage)
	}

	// Verify only one record exists (not duplicates)
	var count int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM usage_records
		WHERE user_identity = ? AND limit_key = ?
	`, "atomic-test@example.com", "ai_credits").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count records: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 usage record, got %d (duplicates were created)", count)
	}
}

func TestConcurrent_GetUsageSummary_DuringRecords(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	seedTestUsageTierLimits(t, db)

	ctx := context.Background()
	const numReaders = 5
	const numWriters = 3
	const operationsPerWorker = 30

	stopChan := make(chan struct{})
	var readErrors, writeErrors int32

	// Writers: continuously record usage
	for i := 0; i < numWriters; i++ {
		go func(workerID int) {
			for j := 0; j < operationsPerWorker; j++ {
				select {
				case <-stopChan:
					return
				default:
					err := svc.RecordUsage(ctx, commerce.UsageReportRequest{
						UserIdentity: "summary-test@example.com",
						LimitKey:     "ai_credits",
						Amount:       1000,
						AppBundleKey: "test-app",
					})
					if err != nil {
						atomic.AddInt32(&writeErrors, 1)
					}
					time.Sleep(2 * time.Millisecond)
				}
			}
		}(i)
	}

	// Readers: continuously get usage summaries
	var wg sync.WaitGroup
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < operationsPerWorker; j++ {
				_, err := svc.GetUsageSummary(ctx, "summary-test@example.com", "solo")
				if err != nil {
					atomic.AddInt32(&readErrors, 1)
					t.Logf("Reader %d op %d failed: %v", workerID, j, err)
				}
				time.Sleep(1 * time.Millisecond)
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

func TestConcurrent_MultipleApps_SameUser(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()
	apps := []string{"browser-automation-studio", "other-app-1", "other-app-2"}
	const reportsPerApp = 15

	var wg sync.WaitGroup
	var errors int32

	for _, app := range apps {
		wg.Add(1)
		go func(appKey string) {
			defer wg.Done()
			for j := 0; j < reportsPerApp; j++ {
				err := svc.RecordUsage(ctx, commerce.UsageReportRequest{
					UserIdentity: "multiapp-user@example.com",
					LimitKey:     "ai_credits",
					Amount:       1000,
					AppBundleKey: appKey,
				})
				if err != nil {
					atomic.AddInt32(&errors, 1)
					t.Logf("App %s report %d failed: %v", appKey, j, err)
				}
			}
		}(app)
	}

	wg.Wait()

	if errors > 0 {
		t.Errorf("Expected 0 errors, got %d", errors)
	}

	// Verify total usage (summed across all apps)
	totalUsage, err := svc.GetUsage(ctx, "multiapp-user@example.com", "ai_credits", nil)
	if err != nil {
		t.Fatalf("GetUsage() returned error: %v", err)
	}

	expectedTotal := int64(len(apps) * reportsPerApp * 1000)
	if totalUsage != expectedTotal {
		t.Errorf("Expected total usage %d, got %d", expectedTotal, totalUsage)
	}

	// Verify per-app breakdown
	for _, app := range apps {
		appKey := app
		appUsage, err := svc.GetUsage(ctx, "multiapp-user@example.com", "ai_credits", &appKey)
		if err != nil {
			t.Fatalf("GetUsage() for app %s returned error: %v", app, err)
		}

		expectedAppUsage := int64(reportsPerApp * 1000)
		if appUsage != expectedAppUsage {
			t.Errorf("App %s: expected usage %d, got %d", app, expectedAppUsage, appUsage)
		}
	}
}

func TestConcurrent_BYOKAndNonBYOK_Interleaved(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()
	const numGoroutines = 10

	var wg sync.WaitGroup
	var errors int32

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		isBYOK := i%2 == 0 // Alternate between BYOK and non-BYOK
		go func(workerID int, byok bool) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				err := svc.RecordUsage(ctx, commerce.UsageReportRequest{
					UserIdentity: "byok-test@example.com",
					LimitKey:     "ai_credits",
					Amount:       1000,
					AppBundleKey: "test-app",
					IsBYOK:       byok,
				})
				if err != nil {
					atomic.AddInt32(&errors, 1)
					t.Logf("Worker %d (BYOK=%v) report %d failed: %v", workerID, byok, j, err)
				}
			}
		}(i, isBYOK)
	}

	wg.Wait()

	if errors > 0 {
		t.Errorf("Expected 0 errors, got %d", errors)
	}

	// Verify usage: only non-BYOK should be counted
	// Half the goroutines are BYOK (amount=0), half are non-BYOK (amount=1000)
	usage, err := svc.GetUsage(ctx, "byok-test@example.com", "ai_credits", nil)
	if err != nil {
		t.Fatalf("GetUsage() returned error: %v", err)
	}

	// 5 non-BYOK workers × 10 reports × 1000 = 50000
	expectedUsage := int64((numGoroutines / 2) * 10 * 1000)
	if usage != expectedUsage {
		t.Errorf("Expected usage %d, got %d", expectedUsage, usage)
	}
}

func TestConcurrent_HighContention_StressTest(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

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
				err := svc.RecordUsage(ctx, commerce.UsageReportRequest{
					UserIdentity: "stress-test@example.com",
					LimitKey:     "ai_credits",
					Amount:       1,
					AppBundleKey: "test-app",
				})
				if err != nil {
					atomic.AddInt32(&errors, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	if errors > 0 {
		t.Errorf("Expected 0 errors under high contention, got %d", errors)
	}

	// Verify final count
	usage, err := svc.GetUsage(ctx, "stress-test@example.com", "ai_credits", nil)
	if err != nil {
		t.Fatalf("GetUsage() returned error: %v", err)
	}

	expectedUsage := int64(numGoroutines * opsPerGoroutine)
	if usage != expectedUsage {
		t.Errorf("Expected usage %d, got %d (lost %d)", expectedUsage, usage, expectedUsage-usage)
	}
}

func TestConcurrent_DifferentLimitKeys(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()
	limitKeys := []string{"ai_credits", "api_calls", "storage_bytes", "compute_minutes"}
	const reportsPerKey = 20

	var wg sync.WaitGroup
	var errors int32

	for _, key := range limitKeys {
		wg.Add(1)
		go func(limitKey string) {
			defer wg.Done()
			for j := 0; j < reportsPerKey; j++ {
				err := svc.RecordUsage(ctx, commerce.UsageReportRequest{
					UserIdentity: "multikey-user@example.com",
					LimitKey:     limitKey,
					Amount:       1000,
					AppBundleKey: "test-app",
				})
				if err != nil {
					atomic.AddInt32(&errors, 1)
					t.Logf("Key %s report %d failed: %v", limitKey, j, err)
				}
			}
		}(key)
	}

	wg.Wait()

	if errors > 0 {
		t.Errorf("Expected 0 errors, got %d", errors)
	}

	// Verify each limit key has correct usage
	for _, key := range limitKeys {
		usage, err := svc.GetUsage(ctx, "multikey-user@example.com", key, nil)
		if err != nil {
			t.Fatalf("GetUsage() for %s returned error: %v", key, err)
		}

		expectedUsage := int64(reportsPerKey * 1000)
		if usage != expectedUsage {
			t.Errorf("Key %s: expected usage %d, got %d", key, expectedUsage, usage)
		}
	}
}
