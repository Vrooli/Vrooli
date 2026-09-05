package graph

import "testing"

const manifestFixture = `{
  "name": "tech-tree-designer",
  "groups": [
    {
      "name": "graph",
      "description": "Scenario interface graph commands.",
      "commands": [
        {
          "name": "describe",
          "description": "Describe graph.",
          "flags": [
            { "name": "scenarios" },
            { "name": "stability" }
          ],
          "binding": { "kind": "connect-rpc", "service": "GraphService", "method": "DescribeTechTree" },
          "governance": { "effect": "read", "run_eligible": true }
        },
        {
          "name": "neighbors",
          "description": "Get neighbors.",
          "positionals": [{ "name": "scenario", "required": true }],
          "flags": [
            { "name": "depth" },
            { "name": "scenarios" }
          ],
          "binding": { "kind": "connect-rpc", "service": "GraphService", "method": "GetNeighborhood" },
          "governance": { "effect": "read", "run_eligible": true }
        },
        {
          "name": "path",
          "description": "Find path.",
          "positionals": [
            { "name": "from", "required": true },
            { "name": "to", "required": true }
          ],
          "flags": [{ "name": "scenarios" }],
          "binding": { "kind": "connect-rpc", "service": "GraphService", "method": "FindPath" },
          "governance": { "effect": "read", "run_eligible": true }
        },
        {
          "name": "ancestors",
          "description": "List ancestors.",
          "positionals": [{ "name": "scenario", "required": true }],
          "flags": [{ "name": "scenarios" }],
          "binding": { "kind": "connect-rpc", "service": "GraphService", "method": "ListAncestors" },
          "governance": { "effect": "read", "run_eligible": true }
        },
        {
          "name": "export",
          "description": "Export graph.",
          "flags": [
            { "name": "format", "default": "text" },
            { "name": "scenarios" },
            { "name": "stability" }
          ],
          "binding": { "kind": "connect-rpc", "service": "GraphService", "method": "ExportTechTree" },
          "governance": { "effect": "read", "run_eligible": true }
        }
      ]
    }
  ]
}`

func TestRegisterBuildsGraphGroup(t *testing.T) {
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
	if group.Subcommands[0].Name != "describe" {
		t.Fatalf("command name = %q, want describe", group.Subcommands[0].Name)
	}
}

func TestRegisterBuildsRunnableDescribeHandler(t *testing.T) {
	group, err := Register(nil, []byte(manifestFixture))
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if group.Subcommands[0].RunCtx == nil {
		t.Fatal("describe RunCtx = nil")
	}
}
