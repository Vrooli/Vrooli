package policy

import (
	"context"
	"testing"
)

// [REQ:REQ-POL-004] CRUD operations on policy store
func TestSQLiteStore_CRUD(t *testing.T) {
	ps := newTestPolicyStore(t)
	ctx := context.Background()

	// Create
	id, err := ps.CreateRule(ctx, Rule{
		RuleType:       RuleTypeAccess,
		SourceScenario: "src",
		TargetScenario: "tgt",
		Effect:         EffectAllow,
		Priority:       5,
		Enabled:        true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected positive id, got %d", id)
	}

	// Get
	rule, err := ps.GetRule(ctx, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if rule.SourceScenario != "src" {
		t.Fatalf("expected src, got %s", rule.SourceScenario)
	}
	// Auto-computed specificity: exact(src)=3 + exact(tgt)=3 + empty(endpoint)=1 = 7
	if rule.Priority != 7 {
		t.Fatalf("expected priority=7 (auto-computed specificity), got %d", rule.Priority)
	}
	if !rule.Enabled {
		t.Fatal("expected enabled=true")
	}

	// Update
	rule.SourceScenario = "new-src"
	rule.Effect = EffectDeny
	if err := ps.UpdateRule(ctx, rule); err != nil {
		t.Fatalf("update: %v", err)
	}

	updated, _ := ps.GetRule(ctx, id)
	if updated.SourceScenario != "new-src" {
		t.Fatalf("expected new-src, got %s", updated.SourceScenario)
	}
	if updated.Effect != EffectDeny {
		t.Fatalf("expected deny, got %s", updated.Effect)
	}

	// Delete
	if err := ps.DeleteRule(ctx, id); err != nil {
		t.Fatalf("delete: %v", err)
	}

	rules, _ := ps.ListRules(ctx, ListFilters{})
	if len(rules) != 0 {
		t.Fatalf("expected 0 rules after delete, got %d", len(rules))
	}
}

// [REQ:REQ-POL-004] List rules with filters
func TestSQLiteStore_ListFilters(t *testing.T) {
	ps := newTestPolicyStore(t)
	ctx := context.Background()

	ps.CreateRule(ctx, Rule{RuleType: RuleTypeAccess, SourceScenario: "a", TargetScenario: "x", Effect: EffectAllow, Enabled: true})
	ps.CreateRule(ctx, Rule{RuleType: RuleTypeRateLimit, SourceScenario: "b", TargetScenario: "y", MaxRequests: 10, WindowSeconds: 60, Enabled: true})
	ps.CreateRule(ctx, Rule{RuleType: RuleTypeAccess, SourceScenario: "c", TargetScenario: "x", Effect: EffectDeny, Enabled: false})

	// By type
	rules, _ := ps.ListRules(ctx, ListFilters{RuleType: RuleTypeAccess})
	if len(rules) != 2 {
		t.Fatalf("expected 2 access rules, got %d", len(rules))
	}

	// By enabled
	enabled := true
	rules, _ = ps.ListRules(ctx, ListFilters{Enabled: &enabled})
	if len(rules) != 2 {
		t.Fatalf("expected 2 enabled rules, got %d", len(rules))
	}

	// By target
	rules, _ = ps.ListRules(ctx, ListFilters{Target: "x"})
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules targeting x, got %d", len(rules))
	}
}

// [REQ:REQ-POL-002] Rate limit rule storage
func TestSQLiteStore_RateLimitRule(t *testing.T) {
	ps := newTestPolicyStore(t)
	ctx := context.Background()

	id, err := ps.CreateRule(ctx, Rule{
		RuleType:       RuleTypeRateLimit,
		SourceScenario: "src",
		TargetScenario: "tgt",
		MaxRequests:    100,
		WindowSeconds:  60,
		BurstAllowance: 10,
		Enabled:        true,
	})
	if err != nil {
		t.Fatalf("create rate_limit: %v", err)
	}

	rule, _ := ps.GetRule(ctx, id)
	if rule.MaxRequests != 100 {
		t.Fatalf("expected max_requests=100, got %d", rule.MaxRequests)
	}
	if rule.WindowSeconds != 60 {
		t.Fatalf("expected window_seconds=60, got %d", rule.WindowSeconds)
	}
	if rule.BurstAllowance != 10 {
		t.Fatalf("expected burst_allowance=10, got %d", rule.BurstAllowance)
	}
}

// [REQ:REQ-POL-003] Circuit breaker rule storage
func TestSQLiteStore_CircuitBreakerRule(t *testing.T) {
	ps := newTestPolicyStore(t)
	ctx := context.Background()

	id, err := ps.CreateRule(ctx, Rule{
		RuleType:         RuleTypeCircuitBreaker,
		SourceScenario:   "src",
		TargetScenario:   "tgt",
		FailureThreshold: 5,
		CooldownSeconds:  30,
		SuccessThreshold: 2,
		Enabled:          true,
	})
	if err != nil {
		t.Fatalf("create circuit_breaker: %v", err)
	}

	rule, _ := ps.GetRule(ctx, id)
	if rule.FailureThreshold != 5 {
		t.Fatalf("expected failure_threshold=5, got %d", rule.FailureThreshold)
	}
	if rule.CooldownSeconds != 30 {
		t.Fatalf("expected cooldown_seconds=30, got %d", rule.CooldownSeconds)
	}
	if rule.SuccessThreshold != 2 {
		t.Fatalf("expected success_threshold=2, got %d", rule.SuccessThreshold)
	}
}

// [REQ:REQ-POL-008] Circuit breaker override set and get
func TestSQLiteStore_CircuitBreakerOverride(t *testing.T) {
	ps := newTestPolicyStore(t)
	ctx := context.Background()

	// Create a circuit breaker rule first
	ruleID, err := ps.CreateRule(ctx, Rule{
		RuleType:         RuleTypeCircuitBreaker,
		SourceScenario:   "src",
		TargetScenario:   "tgt",
		FailureThreshold: 5,
		CooldownSeconds:  30,
		SuccessThreshold: 2,
		Enabled:          true,
	})
	if err != nil {
		t.Fatalf("create cb rule: %v", err)
	}

	// Set override
	err = ps.SetCircuitBreakerOverride(ctx, ruleID, CircuitOpen, 3600)
	if err != nil {
		t.Fatalf("set override: %v", err)
	}

	// Get override
	override, err := ps.GetCircuitBreakerOverride(ctx, ruleID)
	if err != nil {
		t.Fatalf("get override: %v", err)
	}
	if override == nil {
		t.Fatal("expected non-nil override")
	}
	if override.RuleID != ruleID {
		t.Errorf("rule_id: want %d, got %d", ruleID, override.RuleID)
	}
	if override.State != CircuitOpen {
		t.Errorf("state: want open, got %s", override.State)
	}
	if override.ExpiresAt.IsZero() {
		t.Error("expires_at should be set")
	}
}

// [REQ:REQ-POL-008] GetCircuitBreakerOverride returns nil for non-existent override
func TestSQLiteStore_CircuitBreakerOverride_NotFound(t *testing.T) {
	ps := newTestPolicyStore(t)
	ctx := context.Background()

	override, err := ps.GetCircuitBreakerOverride(ctx, 99999)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if override != nil {
		t.Errorf("expected nil override, got: %+v", override)
	}
}

// [REQ:REQ-POL-008] SetCircuitBreakerOverride replaces existing override
func TestSQLiteStore_CircuitBreakerOverride_Replace(t *testing.T) {
	ps := newTestPolicyStore(t)
	ctx := context.Background()

	ruleID, _ := ps.CreateRule(ctx, Rule{
		RuleType: RuleTypeCircuitBreaker, SourceScenario: "s", TargetScenario: "t",
		FailureThreshold: 3, CooldownSeconds: 10, SuccessThreshold: 1, Enabled: true,
	})

	// Set to open
	ps.SetCircuitBreakerOverride(ctx, ruleID, CircuitOpen, 3600)
	// Replace with half_open
	ps.SetCircuitBreakerOverride(ctx, ruleID, CircuitHalfOpen, 7200)

	override, _ := ps.GetCircuitBreakerOverride(ctx, ruleID)
	if override == nil {
		t.Fatal("expected override")
	}
	if override.State != CircuitHalfOpen {
		t.Errorf("state: want half_open, got %s", override.State)
	}
}

// [REQ:REQ-POL-003] Close is a no-op (DB managed externally)
func TestSQLiteStore_Close(t *testing.T) {
	ps := newTestPolicyStore(t)
	if err := ps.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}
