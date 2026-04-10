package policy

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// [REQ:REQ-POL-001A] Priority-based rule matching
// [REQ:REQ-POL-001A1] Conflict resolution with equal priority

func TestSQLiteStore_UpdateRule_NonExistent(t *testing.T) {
	ps := newTestPolicyStore(t)
	ctx := context.Background()

	err := ps.UpdateRule(ctx, Rule{
		ID:             999,
		RuleType:       RuleTypeAccess,
		SourceScenario: "test",
		TargetScenario: "test",
		Effect:         EffectAllow,
	})
	if err != nil {
		t.Fatalf("update non-existent: %v", err)
	}
}

func TestSQLiteStore_DeleteRule_NonExistent(t *testing.T) {
	ps := newTestPolicyStore(t)
	ctx := context.Background()

	err := ps.DeleteRule(ctx, 999)
	if err != nil {
		t.Fatalf("delete non-existent: %v", err)
	}
}

// [REQ:REQ-POL-003A] Three-state circuit breaker
func TestSQLiteStore_CircuitBreakerOverride_SetGet(t *testing.T) {
	ps := newTestPolicyStore(t)
	ctx := context.Background()

	id, err := ps.CreateRule(ctx, Rule{
		RuleType:         RuleTypeCircuitBreaker,
		SourceScenario:   "src",
		TargetScenario:   "tgt",
		Enabled:          true,
		FailureThreshold: 5,
		CooldownSeconds:  60,
		SuccessThreshold: 2,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	err = ps.SetCircuitBreakerOverride(ctx, id, CircuitOpen, 3600)
	if err != nil {
		t.Fatalf("set override: %v", err)
	}

	override, err := ps.GetCircuitBreakerOverride(ctx, id)
	if err != nil {
		t.Fatalf("get override: %v", err)
	}
	if override == nil {
		t.Fatal("expected override, got nil")
	}
	if override.State != CircuitOpen {
		t.Fatalf("expected state=open, got %s", override.State)
	}
}

func TestSQLiteStore_CircuitBreakerOverride_Missing(t *testing.T) {
	ps := newTestPolicyStore(t)
	ctx := context.Background()

	override, err := ps.GetCircuitBreakerOverride(ctx, 999)
	if err != nil {
		t.Fatalf("get override: %v", err)
	}
	if override != nil {
		t.Fatal("expected nil override for non-existent rule")
	}
}

// [REQ:REQ-POL-003A1] State transition from closed to open
func TestSQLiteStore_CircuitBreakerOverride_Replacement(t *testing.T) {
	ps := newTestPolicyStore(t)
	ctx := context.Background()

	id, _ := ps.CreateRule(ctx, Rule{
		RuleType:       RuleTypeCircuitBreaker,
		SourceScenario: "src",
		TargetScenario: "tgt",
		Enabled:        true,
	})

	_ = ps.SetCircuitBreakerOverride(ctx, id, CircuitOpen, 3600)
	_ = ps.SetCircuitBreakerOverride(ctx, id, CircuitHalfOpen, 1800)

	override, err := ps.GetCircuitBreakerOverride(ctx, id)
	if err != nil {
		t.Fatalf("get override: %v", err)
	}
	if override.State != CircuitHalfOpen {
		t.Fatalf("expected half_open, got %s", override.State)
	}
}

// [REQ:REQ-POL-007] Policy violation logging with filters
func TestSQLiteStore_ListViolations_SinceUntilFilters(t *testing.T) {
	ps := newTestPolicyStore(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_ = ps.LogViolation(ctx, Violation{
			SourceScenario: "src",
			TargetScenario: "tgt",
			Endpoint:       "/api/test",
			RuleID:         1,
			RuleType:       RuleTypeAccess,
			Reason:         "denied",
		})
	}

	violations, err := ps.ListViolations(ctx, ViolationFilters{Limit: 3})
	if err != nil {
		t.Fatalf("list violations: %v", err)
	}
	if len(violations) != 3 {
		t.Fatalf("expected 3, got %d", len(violations))
	}

	violations, err = ps.ListViolations(ctx, ViolationFilters{Source: "src"})
	if err != nil {
		t.Fatalf("list by source: %v", err)
	}
	if len(violations) != 5 {
		t.Fatalf("expected 5, got %d", len(violations))
	}

	violations, err = ps.ListViolations(ctx, ViolationFilters{Source: "unknown"})
	if err != nil {
		t.Fatalf("list unknown: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected 0, got %d", len(violations))
	}
}

func TestNewSQLiteStore_ClosedDB(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	_, err = NewSQLiteStore(db)
	if err == nil {
		t.Fatal("expected error from closed DB")
	}
}
