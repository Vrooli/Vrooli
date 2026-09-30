package eventbus

import (
	"testing"
	"time"

	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
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
	p.WorkReferenceProjections = []WorkReferenceProjection{{KindPath: "plan.kind", IDPath: "plan.id", Relationship: "created", RevisionPath: "plan.revision"}}
	return p
}

func TestCacheProjectsOnlyDeclaredDescriptorPaths(t *testing.T) {
	c := NewCache()
	p := policy()
	c.Replace(PolicySnapshot{Version: "policy-v1", ReceiptCapturePolicies: []CapturePolicy{p}}, time.Now())
	projection, refs, version, ok := c.ProjectReceipt("ignored", "plan-manager", p.Selector.Operation, "connect", map[string]any{"plan": map[string]any{"id": "p1", "secret": "no"}, "id": "implicit"})
	if !ok || version != "policy-v1" || projection["plan.id"] != "p1" || len(projection) != 1 {
		t.Fatalf("projection=%#v version=%q ok=%v", projection, version, ok)
	}
	if len(refs) != 1 || refs[0].Kind != "" || refs[0].Id != "p1" || refs[0].State != domain.WorkReferenceState_WORK_REFERENCE_STATE_PROJECTION_MISMATCH {
		t.Fatalf("refs=%#v", refs)
	}
}

func TestCacheProjectsGenericWorkReference(t *testing.T) {
	c := NewCache()
	p := policy()
	c.Replace(PolicySnapshot{Version: "policy-v1", ReceiptCapturePolicies: []CapturePolicy{p}}, time.Now())
	_, refs, _, ok := c.ProjectReceipt("ignored", "plan-manager", p.Selector.Operation, "connect", map[string]any{"plan": map[string]any{"kind": "plan", "id": "p1", "revision": "r7"}})
	if !ok || len(refs) != 1 || refs[0].Kind != "plan" || refs[0].Id != "p1" || refs[0].Revision != "r7" || refs[0].Relationship != "created" || refs[0].State != domain.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE {
		t.Fatalf("refs=%#v ok=%v", refs, ok)
	}
}

func TestCacheRejectsStaleOrUnmatchedPolicy(t *testing.T) {
	c := NewCacheWithMaxAge(time.Millisecond)
	p := policy()
	c.Replace(PolicySnapshot{Version: "policy-v1", ReceiptCapturePolicies: []CapturePolicy{p}}, time.Now().Add(-time.Second))
	if _, _, _, ok := c.ProjectReceipt("", "plan-manager", p.Selector.Operation, "connect", nil); ok {
		t.Fatal("stale policy emitted")
	}
}

func TestCacheRefreshesAgeWhenSnapshotVersionIsUnchanged(t *testing.T) {
	c := NewCacheWithMaxAge(time.Second)
	first := time.Now().Add(-2 * time.Second)
	c.Replace(PolicySnapshot{Version: "policy-v1"}, first)
	if _, _, usable := c.Health(time.Now()); usable {
		t.Fatal("old snapshot unexpectedly usable")
	}
	refreshed := time.Now()
	if changed := c.Replace(PolicySnapshot{Version: "policy-v1"}, refreshed); changed {
		t.Fatal("same snapshot should not be reported as a policy change")
	}
	_, age, usable := c.Health(refreshed)
	if !usable || age > time.Millisecond {
		t.Fatalf("same-version refresh did not renew cache age: age=%s usable=%v", age, usable)
	}
}
