package planning

import "testing"

const manifestFixture = `{
  "name": "tech-tree-designer",
  "groups": [
    {
      "name": "plan",
      "description": "Planning commands.",
      "commands": [
        {
          "name": "create",
          "positionals": [{ "name": "slug", "required": true }],
          "flags": [
            { "name": "display-name" },
            { "name": "sector" },
            { "name": "tier" },
            { "name": "stability", "default": "experimental" }
          ],
          "binding": { "kind": "connect-rpc", "service": "PlanningService", "method": "CreatePlannedScenario" },
          "governance": { "effect": "write", "run_eligible": false }
        },
        {
          "name": "list",
          "binding": { "kind": "connect-rpc", "service": "PlanningService", "method": "ListPlannedScenarios" },
          "governance": { "effect": "read", "run_eligible": true }
        },
        {
          "name": "tree",
          "positionals": [{ "name": "slug", "required": true }],
          "binding": { "kind": "connect-rpc", "service": "PlanningService", "method": "GetPlannedScenario" },
          "governance": { "effect": "read", "run_eligible": true }
        },
        {
          "name": "add",
          "positionals": [{ "name": "slug", "required": true }, { "name": "path", "required": true }],
          "flags": [{ "name": "from-file", "default": "-" }],
          "binding": { "kind": "connect-rpc", "service": "PlanningService", "method": "PutPlannedProtoFile" },
          "governance": { "effect": "write", "run_eligible": false }
        },
        {
          "name": "rm",
          "positionals": [{ "name": "slug", "required": true }, { "name": "path", "required": true }],
          "binding": { "kind": "connect-rpc", "service": "PlanningService", "method": "DeletePlannedProtoFile" },
          "governance": { "effect": "destructive", "run_eligible": false }
        },
        {
          "name": "validate",
          "positionals": [{ "name": "slug", "required": true }],
          "binding": { "kind": "connect-rpc", "service": "PlanningService", "method": "ValidatePlannedScenario" },
          "governance": { "effect": "read", "run_eligible": true }
        },
        {
          "name": "materialize",
          "positionals": [{ "name": "slug", "required": true }],
          "binding": { "kind": "connect-rpc", "service": "PlanningService", "method": "MaterializePlannedScenario" },
          "governance": { "effect": "write", "run_eligible": false }
        }
      ]
    }
  ]
}`

func TestRegisterBuildsPlanningGroup(t *testing.T) {
	group, err := Register(nil, []byte(manifestFixture))
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if group.Name != GroupName {
		t.Fatalf("group.Name = %q, want %q", group.Name, GroupName)
	}
	if len(group.Subcommands) != 7 {
		t.Fatalf("len(group.Subcommands) = %d, want 7", len(group.Subcommands))
	}
}
