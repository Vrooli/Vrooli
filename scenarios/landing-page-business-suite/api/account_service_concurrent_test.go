package main

import (
	"sync"
	"sync/atomic"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// ============================================================================
// Concurrent Operation Tests for AccountService
//
// These tests verify thread safety of the account service, particularly around:
// - Concurrent subscription lookups
// - User tier resolution
// - Cache coherence during updates
// ============================================================================

func TestConcurrent_GetSubscription_SameUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	planSvc := NewPlanService(db)
	svc := NewAccountService(db, planSvc)

	const numGoroutines = 20
	const lookupsPerGoroutine = 50

	var wg sync.WaitGroup
	var errors int32
	var found int32

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < lookupsPerGoroutine; j++ {
				sub, err := svc.GetSubscription("concurrent-sub@example.com")
				if err != nil {
					atomic.AddInt32(&errors, 1)
					t.Logf("Worker %d lookup %d failed: %v", workerID, j, err)
					continue
				}
				if sub != nil {
					atomic.AddInt32(&found, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	if errors > 0 {
		t.Errorf("Expected 0 errors, got %d", errors)
	}
}

func TestConcurrent_GetSubscription_DifferentUsers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	planSvc := NewPlanService(db)
	svc := NewAccountService(db, planSvc)

	const numUsers = 10
	const lookupsPerUser = 30

	var wg sync.WaitGroup
	var errors int32

	for i := 0; i < numUsers; i++ {
		wg.Add(1)
		go func(userNum int) {
			defer wg.Done()
			email := "user" + string(rune('0'+userNum)) + "@example.com"
			for j := 0; j < lookupsPerUser; j++ {
				_, err := svc.GetSubscription(email)
				if err != nil {
					atomic.AddInt32(&errors, 1)
					t.Logf("User %d lookup %d failed: %v", userNum, j, err)
				}
			}
		}(i)
	}

	wg.Wait()

	if errors > 0 {
		t.Errorf("Expected 0 errors, got %d", errors)
	}
}

func TestConcurrent_GetEntitlements_MultipleUsers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	planSvc := NewPlanService(db)
	svc := NewAccountService(db, planSvc)

	const numUsers = 5
	const lookupsPerUser = 40

	var wg sync.WaitGroup
	var errors int32

	for i := 0; i < numUsers; i++ {
		wg.Add(1)
		go func(userNum int) {
			defer wg.Done()
			email := "entitle" + string(rune('0'+userNum)) + "@example.com"
			for j := 0; j < lookupsPerUser; j++ {
				entitlements, err := svc.GetEntitlements(email)
				if err != nil {
					atomic.AddInt32(&errors, 1)
					t.Logf("User %d lookup %d failed: %v", userNum, j, err)
					continue
				}
				if entitlements == nil {
					t.Logf("User %d got nil entitlements", userNum)
				}
			}
		}(i)
	}

	wg.Wait()

	if errors > 0 {
		t.Errorf("Expected 0 errors, got %d", errors)
	}
}

func TestConcurrent_ServiceInitialization(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	planSvc := NewPlanService(db)

	const numGoroutines = 20

	var wg sync.WaitGroup
	var errors int32
	services := make([]*AccountService, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			svc := NewAccountService(db, planSvc)
			if svc == nil {
				atomic.AddInt32(&errors, 1)
				t.Logf("Worker %d: NewAccountService returned nil", workerID)
				return
			}
			services[workerID] = svc
		}(i)
	}

	wg.Wait()

	if errors > 0 {
		t.Errorf("Expected 0 errors during concurrent initialization, got %d", errors)
	}
}

func TestConcurrent_GetSubscriptionAndEntitlements_Mixed(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	planSvc := NewPlanService(db)
	svc := NewAccountService(db, planSvc)

	const numGoroutines = 20
	const opsPerGoroutine = 30

	var wg sync.WaitGroup
	var subErrors, entErrors int32

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			email := "mixed-ops@example.com"
			for j := 0; j < opsPerGoroutine; j++ {
				// Alternate between subscription and entitlement lookups
				if j%2 == 0 {
					_, err := svc.GetSubscription(email)
					if err != nil {
						atomic.AddInt32(&subErrors, 1)
					}
				} else {
					_, err := svc.GetEntitlements(email)
					if err != nil {
						atomic.AddInt32(&entErrors, 1)
					}
				}
			}
		}(i)
	}

	wg.Wait()

	if subErrors > 0 {
		t.Errorf("Expected 0 subscription errors, got %d", subErrors)
	}
	if entErrors > 0 {
		t.Errorf("Expected 0 entitlement errors, got %d", entErrors)
	}
}

func TestConcurrent_GetCredits_MultipleUsers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	planSvc := NewPlanService(db)
	svc := NewAccountService(db, planSvc)

	const numUsers = 5
	const lookupsPerUser = 40

	var wg sync.WaitGroup
	var errors int32

	for i := 0; i < numUsers; i++ {
		wg.Add(1)
		go func(userNum int) {
			defer wg.Done()
			email := "credits" + string(rune('0'+userNum)) + "@example.com"
			for j := 0; j < lookupsPerUser; j++ {
				credits, err := svc.GetCredits(email)
				if err != nil {
					atomic.AddInt32(&errors, 1)
					t.Logf("User %d lookup %d failed: %v", userNum, j, err)
					continue
				}
				if credits == nil {
					t.Logf("User %d got nil credits", userNum)
				}
			}
		}(i)
	}

	wg.Wait()

	if errors > 0 {
		t.Errorf("Expected 0 errors, got %d", errors)
	}
}

func TestConcurrent_AccountService_HighContention_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	db := setupTestDB(t)
	defer db.Close()

	planSvc := NewPlanService(db)
	svc := NewAccountService(db, planSvc)

	const numGoroutines = 50
	const opsPerGoroutine = 100

	var wg sync.WaitGroup
	var errors int32

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				email := "stress" + string(rune('0'+(j%10))) + "@example.com"
				switch j % 3 {
				case 0:
					_, err := svc.GetSubscription(email)
					if err != nil {
						atomic.AddInt32(&errors, 1)
					}
				case 1:
					_, err := svc.GetCredits(email)
					if err != nil {
						atomic.AddInt32(&errors, 1)
					}
				case 2:
					_, err := svc.GetEntitlements(email)
					if err != nil {
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
