package eventbus

import (
	"testing"
	"time"
)

func policy() CapturePolicy {
	var p CapturePolicy
	p.PolicyID = "plan-create"
	p.Enabled = true
	p.Version = "policy-v1"
	p.Selector.TargetScenario = "plan-manager"
	p.Selector.Operation = "POST /vrooli.plan_manager.v1.plans.PlansService/CreatePlan"
	p.Selector.Protocol = "connect"
	p.Selector.EventType = ReceiptEventType
	p.ResponseProjectionPaths = []string{"plan.id"}
	return p
}
func TestCacheProjectsOnlyDeclaredDescriptorPaths(t *testing.T) {
	c := NewCache()
	p := policy()
	c.Replace(PolicySnapshot{Version: "policy-v1", ReceiptCapturePolicies: []CapturePolicy{p}}, time.Now())
	projection, version, ok := c.ProjectReceipt("ignored", "plan-manager", p.Selector.Operation, map[string]any{"plan": map[string]any{"id": "p1", "secret": "no"}, "id": "implicit"})
	if !ok || version != "policy-v1" || projection["plan.id"] != "p1" || len(projection) != 1 {
		t.Fatalf("projection=%#v version=%q ok=%v", projection, version, ok)
	}
}
func TestCacheRejectsStaleOrUnmatchedPolicy(t *testing.T) {
	c := NewCacheWithMaxAge(time.Millisecond)
	p := policy()
	c.Replace(PolicySnapshot{Version: "policy-v1", ReceiptCapturePolicies: []CapturePolicy{p}}, time.Now().Add(-time.Second))
	if _, _, ok := c.ProjectReceipt("", "plan-manager", p.Selector.Operation, nil); ok {
		t.Fatal("stale policy emitted")
	}
}
