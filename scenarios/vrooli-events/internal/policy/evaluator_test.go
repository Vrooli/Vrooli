package policy

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func newTestPolicyStore(t *testing.T) *SQLiteStore {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	ps, err := NewSQLiteStore(db)
	if err != nil {
		t.Fatalf("new policy store: %v", err)
	}
	return ps
}

// [REQ:REQ-POL-006] Evaluator returns allow when no rules exist
func TestEvaluator_NoRules_DefaultAllow(t *testing.T) {
	ps := newTestPolicyStore(t)
	eval := NewEvaluator(ps)

	d := eval.Evaluate(context.Background(), EvalRequest{
		Source: "any", Target: "any",
	})
	if !d.Allowed {
		t.Fatal("expected default allow")
	}
}

// [REQ:REQ-POL-006] Evaluator matches deny rule
func TestEvaluator_DenyRule(t *testing.T) {
	ps := newTestPolicyStore(t)
	eval := NewEvaluator(ps)
	ctx := context.Background()

	ps.CreateRule(ctx, Rule{
		RuleType: RuleTypeAccess, SourceScenario: "bad", TargetScenario: "svc",
		Effect: EffectDeny, Priority: 10, Enabled: true,
	})

	d := eval.Evaluate(ctx, EvalRequest{Source: "bad", Target: "svc"})
	if d.Allowed {
		t.Fatal("expected deny")
	}
	if d.RuleType != RuleTypeAccess {
		t.Fatalf("expected access rule type, got %s", d.RuleType)
	}
}

// [REQ:REQ-POL-006] Evaluator respects priority ordering
func TestEvaluator_PriorityOrder(t *testing.T) {
	ps := newTestPolicyStore(t)
	eval := NewEvaluator(ps)
	ctx := context.Background()

	// Low priority deny
	ps.CreateRule(ctx, Rule{
		RuleType: RuleTypeAccess, SourceScenario: "*", TargetScenario: "svc",
		Effect: EffectDeny, Priority: 1, Enabled: true,
	})
	// High priority allow
	ps.CreateRule(ctx, Rule{
		RuleType: RuleTypeAccess, SourceScenario: "trusted", TargetScenario: "svc",
		Effect: EffectAllow, Priority: 10, Enabled: true,
	})

	d := eval.Evaluate(ctx, EvalRequest{Source: "trusted", Target: "svc"})
	if !d.Allowed {
		t.Fatal("expected allow from higher priority rule")
	}
}

// [REQ:REQ-POL-006] Evaluator skips disabled rules
func TestEvaluator_DisabledRule(t *testing.T) {
	ps := newTestPolicyStore(t)
	eval := NewEvaluator(ps)
	ctx := context.Background()

	ps.CreateRule(ctx, Rule{
		RuleType: RuleTypeAccess, SourceScenario: "src", TargetScenario: "tgt",
		Effect: EffectDeny, Priority: 10, Enabled: false,
	})

	d := eval.Evaluate(ctx, EvalRequest{Source: "src", Target: "tgt"})
	if !d.Allowed {
		t.Fatal("expected allow (disabled rule should be skipped)")
	}
}

// [REQ:REQ-POL-001] Evaluator handles glob patterns in rules
func TestEvaluator_GlobPattern(t *testing.T) {
	ps := newTestPolicyStore(t)
	eval := NewEvaluator(ps)
	ctx := context.Background()

	ps.CreateRule(ctx, Rule{
		RuleType: RuleTypeAccess, SourceScenario: "*", TargetScenario: "critical-svc",
		Effect: EffectDeny, Priority: 5, Enabled: true,
	})

	d := eval.Evaluate(ctx, EvalRequest{Source: "anything", Target: "critical-svc"})
	if d.Allowed {
		t.Fatal("expected deny with wildcard source")
	}
}

// [REQ:REQ-POL-001] Evaluator matches endpoint patterns
func TestEvaluator_EndpointPattern(t *testing.T) {
	ps := newTestPolicyStore(t)
	eval := NewEvaluator(ps)
	ctx := context.Background()

	ps.CreateRule(ctx, Rule{
		RuleType: RuleTypeAccess, SourceScenario: "src", TargetScenario: "svc",
		EndpointPattern: "/admin/**", Effect: EffectDeny, Priority: 10, Enabled: true,
	})

	// Should deny admin endpoint
	d := eval.Evaluate(ctx, EvalRequest{Source: "src", Target: "svc", Endpoint: "/admin/users"})
	if d.Allowed {
		t.Fatal("expected deny for admin endpoint")
	}

	// Should allow non-admin endpoint
	d = eval.Evaluate(ctx, EvalRequest{Source: "src", Target: "svc", Endpoint: "/api/public"})
	if !d.Allowed {
		t.Fatal("expected allow for non-admin endpoint")
	}
}
