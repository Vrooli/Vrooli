package resolver

import (
	"context"
	"errors"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/fallback"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/policy"
)

// [REQ:DI-001] EmittingResolver decorator tests
// [REQ:DI-003] Sender-side policy cache tests

// mockResolver is a stub Resolver for testing.
type mockResolver struct {
	port int
	url  string
	err  error
}

func (m *mockResolver) ResolveScenarioPort(_ context.Context, _, _ string) (int, error) {
	return m.port, m.err
}

func (m *mockResolver) ResolveScenarioURL(_ context.Context, _, _ string) (string, error) {
	return m.url, m.err
}

func TestEmittingResolver_ResolvePort_Success(t *testing.T) {
	inner := &mockResolver{port: 8080}
	// Use nil emitter to avoid network calls in tests
	er := NewEmittingResolver(Config{
		Inner:          inner,
		SourceScenario: "test-src",
	})

	port, err := er.ResolveScenarioPort(context.Background(), "target-svc", "API_PORT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port != 8080 {
		t.Fatalf("expected port 8080, got %d", port)
	}
}

func TestEmittingResolver_ResolveURL_Success(t *testing.T) {
	inner := &mockResolver{url: "http://localhost:9090"}
	er := NewEmittingResolver(Config{
		Inner:          inner,
		SourceScenario: "test-src",
	})

	url, err := er.ResolveScenarioURL(context.Background(), "target-svc", "API_PORT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "http://localhost:9090" {
		t.Fatalf("expected http://localhost:9090, got %s", url)
	}
}

func TestEmittingResolver_PolicyDenied(t *testing.T) {
	inner := &mockResolver{port: 8080}
	er := NewEmittingResolver(Config{
		Inner:          inner,
		SourceScenario: "blocked-svc",
		PolicyRules: []policy.Rule{
			{
				ID:             1,
				RuleType:       policy.RuleTypeAccess,
				SourceScenario: "blocked-svc",
				TargetScenario: "target-svc",
				Effect:         policy.EffectDeny,
				Enabled:        true,
			},
		},
	})

	_, err := er.ResolveScenarioPort(context.Background(), "target-svc", "API_PORT")
	if err == nil {
		t.Fatal("expected policy denied error")
	}
	var pde *PolicyDeniedError
	if !errors.As(err, &pde) {
		t.Fatalf("expected PolicyDeniedError, got %T: %v", err, err)
	}
	if pde.RuleID != 1 {
		t.Fatalf("expected rule ID 1, got %d", pde.RuleID)
	}
}

func TestEmittingResolver_PolicyAllowWildcard(t *testing.T) {
	inner := &mockResolver{port: 8080}
	er := NewEmittingResolver(Config{
		Inner:          inner,
		SourceScenario: "any-svc",
		PolicyRules: []policy.Rule{
			{
				ID:             1,
				RuleType:       policy.RuleTypeAccess,
				SourceScenario: "*",
				TargetScenario: "*",
				Effect:         policy.EffectAllow,
				Enabled:        true,
			},
		},
	})

	port, err := er.ResolveScenarioPort(context.Background(), "target-svc", "API_PORT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port != 8080 {
		t.Fatalf("expected 8080, got %d", port)
	}
}

func TestEmittingResolver_DisabledRuleSkipped(t *testing.T) {
	inner := &mockResolver{port: 8080}
	er := NewEmittingResolver(Config{
		Inner:          inner,
		SourceScenario: "blocked-svc",
		PolicyRules: []policy.Rule{
			{
				ID:             1,
				RuleType:       policy.RuleTypeAccess,
				SourceScenario: "blocked-svc",
				TargetScenario: "target-svc",
				Effect:         policy.EffectDeny,
				Enabled:        false, // disabled
			},
		},
	})

	_, err := er.ResolveScenarioPort(context.Background(), "target-svc", "API_PORT")
	if err != nil {
		t.Fatalf("expected disabled rule to be skipped, got: %v", err)
	}
}

func TestEmittingResolver_InnerError_Propagated(t *testing.T) {
	inner := &mockResolver{err: errors.New("connection refused")}
	er := NewEmittingResolver(Config{
		Inner:          inner,
		SourceScenario: "test-src",
	})

	_, err := er.ResolveScenarioPort(context.Background(), "target-svc", "API_PORT")
	if err == nil || err.Error() != "connection refused" {
		t.Fatalf("expected inner error, got: %v", err)
	}
}

func TestEmittingResolver_UpdateRules(t *testing.T) {
	inner := &mockResolver{port: 8080}
	er := NewEmittingResolver(Config{
		Inner:          inner,
		SourceScenario: "test-src",
	})

	// Initially no rules → should succeed
	if _, err := er.ResolveScenarioPort(context.Background(), "target", "API_PORT"); err != nil {
		t.Fatalf("expected success with empty rules: %v", err)
	}

	// Add deny rule
	er.UpdateRules([]policy.Rule{
		{
			ID: 1, RuleType: policy.RuleTypeAccess,
			SourceScenario: "test-src", TargetScenario: "target",
			Effect: policy.EffectDeny, Enabled: true,
		},
	})

	_, err := er.ResolveScenarioPort(context.Background(), "target", "API_PORT")
	if err == nil {
		t.Fatal("expected denial after UpdateRules")
	}
}

func TestEmittingResolver_ApplyEvent_CRUD(t *testing.T) {
	inner := &mockResolver{port: 8080}
	er := NewEmittingResolver(Config{
		Inner:          inner,
		SourceScenario: "test-src",
	})

	// Create
	denyRule := policy.Rule{
		ID: 1, RuleType: policy.RuleTypeAccess,
		SourceScenario: "test-src", TargetScenario: "target",
		Effect: policy.EffectDeny, Enabled: true,
	}
	er.ApplyEvent(policy.PolicyEvent{Action: "created", RuleID: 1, Rule: &denyRule})
	if len(er.Rules()) != 1 {
		t.Fatalf("expected 1 rule after create, got %d", len(er.Rules()))
	}

	// Update to allow
	allowRule := denyRule
	allowRule.Effect = policy.EffectAllow
	er.ApplyEvent(policy.PolicyEvent{Action: "updated", RuleID: 1, Rule: &allowRule})
	rules := er.Rules()
	if rules[0].Effect != policy.EffectAllow {
		t.Fatalf("expected allow after update, got %s", rules[0].Effect)
	}

	// Delete
	er.ApplyEvent(policy.PolicyEvent{Action: "deleted", RuleID: 1})
	if len(er.Rules()) != 0 {
		t.Fatalf("expected 0 rules after delete, got %d", len(er.Rules()))
	}
}

func TestEmittingResolver_DefaultMode(t *testing.T) {
	inner := &mockResolver{port: 8080}
	er := NewEmittingResolver(Config{
		Inner:          inner,
		SourceScenario: "test-src",
		DefaultMode:    fallback.ModeFailClosed,
	})

	// Empty rules with fail-closed mode still allows (no matching rule = no denial)
	// Fail-closed applies when events are unreachable, not when cache is empty
	if _, err := er.ResolveScenarioPort(context.Background(), "target", "API_PORT"); err != nil {
		t.Fatalf("expected success with empty rules even in fail-closed: %v", err)
	}
}

func TestEmittingResolver_EmitDoesNotBlock(t *testing.T) {
	inner := &mockResolver{port: 8080}

	// Use nil emitter — emitResolve is a no-op, proving resolve still works.
	er := NewEmittingResolver(Config{
		Inner:          inner,
		SourceScenario: "test-src",
	})

	// Call many times — should not block
	for i := 0; i < 50; i++ {
		port, err := er.ResolveScenarioPort(context.Background(), "target", "API_PORT")
		if err != nil {
			t.Fatalf("unexpected error on call %d: %v", i, err)
		}
		if port != 8080 {
			t.Fatalf("expected 8080 on call %d, got %d", i, port)
		}
	}
}
