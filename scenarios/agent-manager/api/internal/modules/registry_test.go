package modules

import (
	"regexp"
	"testing"
)

func TestAllSchemasDeclaresEveryDomainTableExactlyOnce(t *testing.T) {
	want := map[string]bool{
		"agent_profiles": true, "workflow_revisions": true, "workflow_executions": true, "workflow_node_attempts": true, "workflow_journal": true,
		"tasks": true, "runs": true, "run_events": true, "run_checkpoints": true, "idempotency_records": true,
		"policies": true, "permission_policy_reconcile_audit": true, "scope_locks": true,
		"model_pricing": true, "model_aliases": true, "manual_price_overrides": true, "pricing_settings": true,
		"investigation_settings": true, "model_health_audit": true, "runner_health_audit": true, "stats_checkpoint": true,
		"artifacts": true,
	}
	create := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+IF\s+NOT\s+EXISTS\s+([a-z_]+)`)
	seen := map[string]int{}
	for _, provider := range AllSchemas() {
		for _, match := range create.FindAllStringSubmatch(provider.Schema(), -1) {
			seen[match[1]]++
		}
	}
	for table := range want {
		if seen[table] != 1 {
			t.Fatalf("table %s declared %d times, want exactly once", table, seen[table])
		}
	}
	for table, count := range seen {
		if !want[table] {
			t.Fatalf("unexpected table %s declared %d times", table, count)
		}
	}
}
