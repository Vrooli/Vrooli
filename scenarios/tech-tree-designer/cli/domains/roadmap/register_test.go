package roadmap

import "testing"

const manifestFixture = `{
  "name": "tech-tree-designer",
  "groups": [
    {
      "name": "roadmap",
      "description": "Roadmap commands.",
      "commands": [
        {
          "name": "sectors",
          "binding": { "kind": "connect-rpc", "service": "RoadmapService", "method": "ListSectors" },
          "governance": { "effect": "read", "run_eligible": true }
        },
        {
          "name": "sector",
          "positionals": [{ "name": "slug", "required": true }],
          "flags": [{ "name": "name" }, { "name": "description" }],
          "binding": { "kind": "connect-rpc", "service": "RoadmapService", "method": "UpsertSector" },
          "governance": { "effect": "write", "run_eligible": false }
        },
        {
          "name": "milestones",
          "binding": { "kind": "connect-rpc", "service": "RoadmapService", "method": "ListMilestones" },
          "governance": { "effect": "read", "run_eligible": true }
        },
        {
          "name": "milestone",
          "positionals": [{ "name": "id", "required": true }],
          "flags": [{ "name": "name" }, { "name": "description" }, { "name": "required" }],
          "binding": { "kind": "connect-rpc", "service": "RoadmapService", "method": "UpsertMilestone" },
          "governance": { "effect": "write", "run_eligible": false }
        },
        {
          "name": "progress",
          "binding": { "kind": "connect-rpc", "service": "RoadmapService", "method": "GetProgress" },
          "governance": { "effect": "read", "run_eligible": true }
        }
      ]
    }
  ]
}`

func TestRegisterBuildsRoadmapGroup(t *testing.T) {
	// [REQ:TTD-ROADMAP-001] Roadmap CLI commands are bound from the manifest.
	group, err := Register(nil, []byte(manifestFixture))
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if group.Name != GroupName {
		t.Fatalf("group.Name = %q, want %q", group.Name, GroupName)
	}
	if len(group.Subcommands) != 5 {
		t.Fatalf("len(group.Subcommands) = %d, want 5", len(group.Subcommands))
	}
}
