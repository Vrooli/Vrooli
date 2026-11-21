package services

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

// capturedAgentList is a real `vrooli agent list --json` (control-plane build
// with reap-planner fields) read on 2026-10-08: 42 rows, 5 orphaned
// web-console sessions 535-724 h old holding ~0.7 GiB memory+swap, every row
// measured, none reapable ("an operator terminal is never auto-reaped").
func capturedAgentList(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile("testdata/agent_list_2026-10-08.json")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// withPlannerFields returns the captured listing with reap-planner fields
// set on the rows matched by scope suffix, as the control plane will emit them.
func withPlannerFields(t *testing.T, fields map[string]map[string]interface{}) []byte {
	t.Helper()
	var doc map[string]interface{}
	if err := json.Unmarshal(capturedAgentList(t), &doc); err != nil {
		t.Fatal(err)
	}
	for _, row := range doc["sessions"].([]interface{}) {
		r := row.(map[string]interface{})
		for scopePrefix, extra := range fields {
			if strings.HasPrefix(r["scope"].(string), scopePrefix) {
				for k, v := range extra {
					r[k] = v
				}
			}
		}
	}
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

type staticInventory struct {
	inventory AgentInventory
	err       error
}

func (s staticInventory) ListAgentSessions(context.Context) (AgentInventory, error) {
	return s.inventory, s.err
}

func parseInventory(t *testing.T, raw []byte) AgentInventory {
	t.Helper()
	inventory, err := ParseAgentInventory(raw)
	if err != nil {
		t.Fatal(err)
	}
	return inventory
}

func TestParseAgentInventory_CapturedListing(t *testing.T) {
	inventory := parseInventory(t, capturedAgentList(t))
	if !inventory.ClassifiesOrphans || len(inventory.Sessions) != 42 {
		t.Fatalf("classifies=%v sessions=%d, want owner-classified 42 rows", inventory.ClassifiesOrphans, len(inventory.Sessions))
	}
	orphans := 0
	for _, s := range inventory.Sessions {
		if s.State != "orphaned" {
			continue
		}
		orphans++
		if s.Owner == nil || s.Owner.Kind != "web-console-session" || s.Owner.Ref == "" || s.KernelState != "agentless" {
			t.Fatalf("orphan row lost owner/kernel state: %+v", s)
		}
		if !s.MemoryKnown || !s.SwapKnown || s.MemoryStatus != "measured" || !s.ReapableKnown || s.Reapable || s.ReapReason == "" {
			t.Fatalf("planner fields not bound: %+v", s)
		}
	}
	if orphans != 5 {
		t.Fatalf("orphans = %d, want 5", orphans)
	}
}

func TestParseAgentInventory_NullMemoryIsUnknownNotZero(t *testing.T) {
	inventory := parseInventory(t, []byte(`{"sessions":[{"scope":"s","state":"orphaned","owner":{"kind":"terminal","ref":"t1"},"memory_bytes":null,"memory_status":"unavailable","swap_bytes":0,"swap_status":"measured"}],"summary":{"sessions":1,"memory_unknown":1}}`))
	s := inventory.Sessions[0]
	if s.MemoryKnown || s.MemoryStatus == "" {
		t.Fatalf("null memory = %+v, want unknown with status", s)
	}
	if !s.SwapKnown || s.SwapBytes != 0 {
		t.Fatalf("explicit zero swap = %+v, want known 0", s)
	}
}

func TestOrphanedAgentsRule_CapturedOldOrphansWarn(t *testing.T) {
	rule := NewOrphanedAgentsRule(staticInventory{inventory: parseInventory(t, capturedAgentList(t))}, NewStubClock(time.Unix(0, 0)))
	eval := rule.Evaluate(context.Background(), AlertSnapshot{})
	if eval.Status != RuleSupported || len(eval.Conditions) != 1 {
		t.Fatalf("evaluation = %+v", eval)
	}
	c := eval.Conditions[0]
	if c.Severity != SeverityWarning || c.Value != 5 {
		t.Fatalf("condition = %s value %v, want warning for 5 orphans", c.Severity, c.Value)
	}
	// Sum of the five orphaned rows' memory_bytes + swap_bytes in the capture.
	if c.Details["known_memory_swap_bytes"] != int64(732958720) || c.Details["memory_unknown_sessions"] != 0 {
		t.Fatalf("memory evidence = %v / %v, want 732958720 bytes, none unknown", c.Details["known_memory_swap_bytes"], c.Details["memory_unknown_sessions"])
	}
	if c.Details["stuck_reapable"] != 0 {
		t.Fatalf("stuck reapable = %v, want 0", c.Details["stuck_reapable"])
	}
	first := c.Details["sessions"].([]interface{})[0].(map[string]interface{})
	if first["owner_kind"] != "web-console-session" || first["owner_ref"] == "" || first["memory_status"] != "measured" || first["reap_reason"] == "" {
		t.Fatalf("session detail = %v", first)
	}
	if !strings.Contains(c.Message, "does not terminate") {
		t.Fatalf("message %q", c.Message)
	}
}

func TestOrphanedAgentsRule_MemoryAtTwoGiBIsCritical(t *testing.T) {
	// Grow one captured orphan so the five together reach exactly 2 GiB.
	captured := int64(732958720)
	const heavyScope = "vrooli-agent-claude-code-3279c393"
	inventory := parseInventory(t, capturedAgentList(t))
	var heavyMemory, heavySwap int64
	for _, s := range inventory.Sessions {
		if strings.HasPrefix(s.Scope, heavyScope) {
			heavyMemory, heavySwap = s.MemoryBytes, s.SwapBytes
		}
	}
	grown := heavyMemory + (2<<30 - captured)
	raw := withPlannerFields(t, map[string]map[string]interface{}{heavyScope: {"memory_bytes": grown, "swap_bytes": heavySwap}})
	eval := NewOrphanedAgentsRule(staticInventory{inventory: parseInventory(t, raw)}, NewStubClock(time.Unix(0, 0))).Evaluate(context.Background(), AlertSnapshot{})
	c := eval.Conditions[0]
	if c.Severity != SeverityCritical || c.Details["known_memory_swap_bytes"] != int64(2<<30) {
		t.Fatalf("condition = %s bytes %v, want critical at exactly 2 GiB", c.Severity, c.Details["known_memory_swap_bytes"])
	}

	below := withPlannerFields(t, map[string]map[string]interface{}{heavyScope: {"memory_bytes": grown - 1, "swap_bytes": heavySwap}})
	eval = NewOrphanedAgentsRule(staticInventory{inventory: parseInventory(t, below)}, NewStubClock(time.Unix(0, 0))).Evaluate(context.Background(), AlertSnapshot{})
	if eval.Conditions[0].Severity != SeverityWarning {
		t.Fatalf("one byte under 2 GiB = %s, want warning (old orphans only)", eval.Conditions[0].Severity)
	}
}

func TestOrphanedAgentsRule_UnknownMemoryIsNotCountedAsZero(t *testing.T) {
	raw := withPlannerFields(t, map[string]map[string]interface{}{
		"vrooli-agent-claude-code-3279c393": {"memory_bytes": nil, "memory_status": "unavailable", "swap_bytes": nil, "swap_status": "unsupported"},
	})
	eval := NewOrphanedAgentsRule(staticInventory{inventory: parseInventory(t, raw)}, NewStubClock(time.Unix(0, 0))).Evaluate(context.Background(), AlertSnapshot{})
	c := eval.Conditions[0]
	if c.Details["memory_unknown_sessions"] != 1 || !strings.Contains(c.Message, "memory unknown for 1") {
		t.Fatalf("details = %v message %q, want one session reported unknown", c.Details, c.Message)
	}
	for _, row := range c.Details["sessions"].([]interface{}) {
		session := row.(map[string]interface{})
		if strings.HasPrefix(session["scope"].(string), "vrooli-agent-claude-code-3279c393") && session["memory_bytes"] != "unknown: unavailable" {
			t.Fatalf("unknown row memory = %v, want unknown: unavailable", session["memory_bytes"])
		}
	}
}

func TestOrphanedAgentsRule_ReapableForAnHourIsCritical(t *testing.T) {
	young := []byte(`{"sessions":[{"session":"s1","scope":"scope-1","state":"orphaned","age":"20m0s","owner":{"kind":"terminal","ref":"t1"},"reapable":true,"reap_reason":"owner gone; idle 20m"}]}`)
	clock := NewStubClock(time.Unix(0, 0))
	rule := NewOrphanedAgentsRule(staticInventory{inventory: parseInventory(t, young)}, clock)

	if eval := rule.Evaluate(context.Background(), AlertSnapshot{}); len(eval.Conditions) != 0 {
		t.Fatalf("a freshly reapable young orphan alerted: %+v", eval.Conditions)
	}
	clock.Advance(59 * time.Minute)
	if eval := rule.Evaluate(context.Background(), AlertSnapshot{}); len(eval.Conditions) != 0 {
		t.Fatal("reapable for 59 minutes alerted")
	}
	clock.Advance(time.Minute)
	eval := rule.Evaluate(context.Background(), AlertSnapshot{})
	if len(eval.Conditions) != 1 || eval.Conditions[0].Severity != SeverityCritical {
		t.Fatalf("reapable for an hour = %+v, want critical (auto-reap off or failing)", eval.Conditions)
	}
	session := eval.Conditions[0].Details["sessions"].([]interface{})[0].(map[string]interface{})
	if session["reap_reason"] != "owner gone; idle 20m" || session["owner_kind"] != "terminal" {
		t.Fatalf("session detail = %v", session)
	}
}

func TestOrphanedAgentsRule_ReapedRowResetsPersistence(t *testing.T) {
	reapable := parseInventory(t, []byte(`{"sessions":[{"scope":"scope-1","state":"orphaned","age":"5m0s","owner":{"kind":"terminal","ref":"t1"},"reapable":true}]}`))
	gone := parseInventory(t, []byte(`{"sessions":[{"scope":"other","state":"live","owner":{"kind":"terminal","ref":"t2"}}]}`))
	clock := NewStubClock(time.Unix(0, 0))
	source := &switchingInventory{inventory: reapable}
	rule := NewOrphanedAgentsRule(source, clock)
	rule.Evaluate(context.Background(), AlertSnapshot{})
	clock.Advance(50 * time.Minute)
	source.inventory = gone
	rule.Evaluate(context.Background(), AlertSnapshot{})
	source.inventory = reapable
	clock.Advance(20 * time.Minute)
	if eval := rule.Evaluate(context.Background(), AlertSnapshot{}); len(eval.Conditions) != 0 {
		t.Fatal("persistence survived the row disappearing")
	}
}

type switchingInventory struct{ inventory AgentInventory }

func (s *switchingInventory) ListAgentSessions(context.Context) (AgentInventory, error) {
	return s.inventory, nil
}

func TestOrphanedAgentsRule_UnsupportedWithoutOwnerAttribution(t *testing.T) {
	legacy := parseInventory(t, []byte(`{"sessions":[{"session":"(no lease)","scope":"x","state":"orphan","tasks":3}]}`))
	eval := NewOrphanedAgentsRule(staticInventory{inventory: legacy}, nil).Evaluate(context.Background(), AlertSnapshot{})
	if eval.Status != RuleUnsupported || eval.Reason != "control plane does not classify orphans yet" {
		t.Fatalf("evaluation = %+v", eval)
	}
}

func TestOrphanedAgentsRule_InventoryFailureIsFailedNotClear(t *testing.T) {
	eval := NewOrphanedAgentsRule(staticInventory{err: errors.New("vrooli: command not found")}, nil).Evaluate(context.Background(), AlertSnapshot{})
	if eval.Status != RuleFailed {
		t.Fatalf("status = %s, want failed", eval.Status)
	}
}
