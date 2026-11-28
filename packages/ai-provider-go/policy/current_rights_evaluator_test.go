package policy

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCurrentRightsEvaluatorReloadsTrustedContextForExecutionRecheck(t *testing.T) {
	artifact := testConditionalRights()
	calls := 0
	evaluator := NewRightsEvaluator(func(context.Context) (RightsContext, error) {
		calls++
		if calls == 1 {
			return testRightsContext("pre_revenue", "context-v1"), nil
		}
		return testRightsContext("at_or_above_threshold", "context-v2"), nil
	})

	decision, err := evaluator.Evaluate(context.Background(), &artifact, RightsUseLocalInference)
	if err != nil || decision.Status != RightsAllowed {
		t.Fatalf("current pre-revenue evaluation = %+v, %v", decision, err)
	}
	if err := evaluator.RequireCurrent(context.Background(), decision, &artifact, RightsUseLocalInference); err == nil {
		t.Fatal("execution recheck reused a decision after the trusted context crossed its threshold")
	}
	if calls != 2 {
		t.Fatalf("trusted context loads = %d, want one per evaluation", calls)
	}
}

func TestCurrentRightsEvaluatorReturnsVerifiableUnknownWhenContextIsUnavailable(t *testing.T) {
	artifact := testConditionalRights()
	evaluator := NewRightsEvaluator(func(context.Context) (RightsContext, error) {
		return RightsContext{}, errors.New("context file unavailable")
	})
	decision, err := evaluator.Evaluate(context.Background(), &artifact, RightsUseLocalInference)
	if err == nil || decision.Status != RightsUnknown || decision.Reason != "trusted_model_rights_context_unavailable" {
		t.Fatalf("missing context evaluation = %+v, %v", decision, err)
	}
	if err := VerifyRightsDecision(decision); err != nil {
		t.Fatalf("unknown context decision is not verifiable: %v", err)
	}
}

func TestDefaultRightsEvaluatorUsesCanonicalContextPathOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), "operator-rights-context.json")
	document := `{"schema_version":"1","context":{"id":"operator-context","version":"context-v4","entity_scope":"operator-group","revenue_status":"pre_revenue","as_of":"2026-11-14","source":"operator_statement"}}`
	if err := os.WriteFile(path, []byte(document), 0o600); err != nil {
		t.Fatal(err)
	}
	evaluator := NewDefaultRightsEvaluator(func(key string) string {
		if key == RightsContextPathEnv {
			return path
		}
		return ""
	})
	artifact := testConditionalRights()
	decision, err := evaluator.Evaluate(context.Background(), &artifact, RightsUseLocalInference)
	if err != nil || decision.Status != RightsAllowed || decision.ContextID != "operator-context" || decision.ContextVersion != "context-v4" {
		t.Fatalf("canonical path context evaluation = %+v, %v", decision, err)
	}
}
