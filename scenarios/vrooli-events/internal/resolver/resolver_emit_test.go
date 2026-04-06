package resolver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/emitter"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/policy"
)

// [REQ:DI-001] emitResolve fires event on successful port resolution
func TestEmitResolve_PortSuccess(t *testing.T) {
	var received atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	em := emitter.NewEmitter(emitter.Config{
		EventsURL:     srv.URL,
		BufferSize:    10,
		BatchSize:     1,
		FlushInterval: time.Millisecond,
		MaxRetries:    1,
		RetryBackoff:  time.Millisecond,
	})

	inner := &mockResolver{port: 8080}
	er := NewEmittingResolver(Config{
		Inner:          inner,
		Emitter:        em,
		SourceScenario: "test-src",
	})

	port, err := er.ResolveScenarioPort(context.Background(), "target-svc", "API_PORT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port != 8080 {
		t.Fatalf("expected port 8080, got %d", port)
	}

	em.Close()

	if got := received.Load(); got != 1 {
		t.Fatalf("expected 1 emitted event, got %d", got)
	}
}

// [REQ:DI-001] emitResolve fires event on URL resolution
func TestEmitResolve_URLSuccess(t *testing.T) {
	var received atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	em := emitter.NewEmitter(emitter.Config{
		EventsURL:     srv.URL,
		BufferSize:    10,
		BatchSize:     1,
		FlushInterval: time.Millisecond,
		MaxRetries:    1,
		RetryBackoff:  time.Millisecond,
	})

	inner := &mockResolver{url: "http://localhost:9090"}
	er := NewEmittingResolver(Config{
		Inner:          inner,
		Emitter:        em,
		SourceScenario: "test-src",
	})

	url, err := er.ResolveScenarioURL(context.Background(), "target-svc", "API_PORT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "http://localhost:9090" {
		t.Fatalf("expected http://localhost:9090, got %s", url)
	}

	em.Close()

	if got := received.Load(); got != 1 {
		t.Fatalf("expected 1 emitted event, got %d", got)
	}
}

// [REQ:DI-001] emitResolve fires event with error info on inner failure
func TestEmitResolve_InnerError(t *testing.T) {
	var received atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	em := emitter.NewEmitter(emitter.Config{
		EventsURL:     srv.URL,
		BufferSize:    10,
		BatchSize:     1,
		FlushInterval: time.Millisecond,
		MaxRetries:    1,
		RetryBackoff:  time.Millisecond,
	})

	inner := &mockResolver{err: errors.New("dial failed")}
	er := NewEmittingResolver(Config{
		Inner:          inner,
		Emitter:        em,
		SourceScenario: "test-src",
	})

	_, err := er.ResolveScenarioPort(context.Background(), "target-svc", "API_PORT")
	if err == nil {
		t.Fatal("expected error from inner resolver")
	}

	em.Close()

	// Event should still be emitted even on failure (with error info)
	if got := received.Load(); got != 1 {
		t.Fatalf("expected 1 emitted event on failure, got %d", got)
	}
}

// [REQ:DI-001] emitResolve skipped when policy denies (no inner call, no emit)
func TestEmitResolve_PolicyDenied_NoEmit(t *testing.T) {
	var received atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	em := emitter.NewEmitter(emitter.Config{
		EventsURL:     srv.URL,
		BufferSize:    10,
		BatchSize:     1,
		FlushInterval: time.Millisecond,
		MaxRetries:    1,
		RetryBackoff:  time.Millisecond,
	})

	inner := &mockResolver{port: 8080}
	er := NewEmittingResolver(Config{
		Inner:          inner,
		Emitter:        em,
		SourceScenario: "blocked-svc",
		PolicyRules: []policy.Rule{
			{
				ID: 1, RuleType: policy.RuleTypeAccess,
				SourceScenario: "blocked-svc", TargetScenario: "target-svc",
				Effect: policy.EffectDeny, Enabled: true,
			},
		},
	})

	_, err := er.ResolveScenarioPort(context.Background(), "target-svc", "API_PORT")
	if err == nil {
		t.Fatal("expected policy denied error")
	}

	em.Close()

	// No emit should happen since policy check fails before inner call
	if got := received.Load(); got != 0 {
		t.Fatalf("expected 0 emitted events when policy denies, got %d", got)
	}
}

// [REQ:DI-003] Non-access rule types are skipped during policy check
func TestCheckPolicy_NonAccessRuleSkipped(t *testing.T) {
	inner := &mockResolver{port: 8080}
	er := NewEmittingResolver(Config{
		Inner:          inner,
		SourceScenario: "test-src",
		PolicyRules: []policy.Rule{
			{
				ID: 1, RuleType: policy.RuleTypeRateLimit,
				SourceScenario: "test-src", TargetScenario: "target-svc",
				Effect: policy.EffectDeny, Enabled: true,
			},
		},
	})

	// Rate limit rule should be skipped, so resolve should succeed
	port, err := er.ResolveScenarioPort(context.Background(), "target-svc", "API_PORT")
	if err != nil {
		t.Fatalf("rate_limit rule should not block access, got: %v", err)
	}
	if port != 8080 {
		t.Fatalf("expected 8080, got %d", port)
	}
}

// [REQ:DI-003] Source mismatch allows request through
func TestCheckPolicy_SourceMismatch(t *testing.T) {
	inner := &mockResolver{port: 8080}
	er := NewEmittingResolver(Config{
		Inner:          inner,
		SourceScenario: "different-svc",
		PolicyRules: []policy.Rule{
			{
				ID: 1, RuleType: policy.RuleTypeAccess,
				SourceScenario: "blocked-svc", TargetScenario: "target-svc",
				Effect: policy.EffectDeny, Enabled: true,
			},
		},
	})

	_, err := er.ResolveScenarioPort(context.Background(), "target-svc", "API_PORT")
	if err != nil {
		t.Fatalf("expected access when source doesn't match deny rule, got: %v", err)
	}
}

// [REQ:DI-001] ApplyEvent update for non-existent rule adds it
func TestApplyEvent_UpdateNonExistent(t *testing.T) {
	inner := &mockResolver{port: 8080}
	er := NewEmittingResolver(Config{
		Inner:          inner,
		SourceScenario: "test-src",
	})

	newRule := policy.Rule{
		ID: 99, RuleType: policy.RuleTypeAccess,
		SourceScenario: "src", TargetScenario: "tgt",
		Effect: policy.EffectAllow, Enabled: true,
	}
	er.ApplyEvent(policy.PolicyEvent{Action: "updated", RuleID: 99, Rule: &newRule})

	rules := er.Rules()
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule after update of non-existent, got %d", len(rules))
	}
	if rules[0].ID != 99 {
		t.Fatalf("expected rule ID 99, got %d", rules[0].ID)
	}
}

// [REQ:DI-001] ApplyEvent with nil rule for create/update is a no-op
func TestApplyEvent_NilRule(t *testing.T) {
	inner := &mockResolver{port: 8080}
	er := NewEmittingResolver(Config{
		Inner:          inner,
		SourceScenario: "test-src",
	})

	er.ApplyEvent(policy.PolicyEvent{Action: "created", RuleID: 1, Rule: nil})
	if len(er.Rules()) != 0 {
		t.Fatal("expected no rules added when Rule is nil on create")
	}

	er.ApplyEvent(policy.PolicyEvent{Action: "updated", RuleID: 1, Rule: nil})
	if len(er.Rules()) != 0 {
		t.Fatal("expected no rules added when Rule is nil on update")
	}
}

// [REQ:DI-001] ApplyEvent delete for non-existent rule is a no-op
func TestApplyEvent_DeleteNonExistent(t *testing.T) {
	inner := &mockResolver{port: 8080}
	er := NewEmittingResolver(Config{
		Inner:          inner,
		SourceScenario: "test-src",
	})

	// Should not panic or error
	er.ApplyEvent(policy.PolicyEvent{Action: "deleted", RuleID: 999})
	if len(er.Rules()) != 0 {
		t.Fatal("expected 0 rules")
	}
}
