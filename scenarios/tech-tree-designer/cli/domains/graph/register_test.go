package graph

import (
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

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
          "binding": { "kind": "connect-rpc", "service": "GraphService", "method": "DescribeTechTree" },
          "governance": { "effect": "read", "run_eligible": false }
        },
        {
          "name": "neighbors",
          "description": "Get neighbors.",
          "binding": { "kind": "connect-rpc", "service": "GraphService", "method": "GetNeighborhood" },
          "governance": { "effect": "read", "run_eligible": false }
        },
        {
          "name": "path",
          "description": "Find path.",
          "binding": { "kind": "connect-rpc", "service": "GraphService", "method": "FindPath" },
          "governance": { "effect": "read", "run_eligible": false }
        },
        {
          "name": "ancestors",
          "description": "List ancestors.",
          "binding": { "kind": "connect-rpc", "service": "GraphService", "method": "ListAncestors" },
          "governance": { "effect": "read", "run_eligible": false }
        },
        {
          "name": "export",
          "description": "Export graph.",
          "binding": { "kind": "connect-rpc", "service": "GraphService", "method": "ExportTechTree" },
          "governance": { "effect": "read", "run_eligible": false }
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

func TestDescribeHandlerIsReserved(t *testing.T) {
	group, err := Register(nil, []byte(manifestFixture))
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	err = group.Subcommands[0].RunCtx(cliapp.NewTestRunContext(cliapp.TestRunContextOptions{}))
	if err == nil {
		t.Fatal("RunCtx() error = nil, want reserved-domain guidance")
	}
	if !strings.Contains(err.Error(), "graph domain is implemented") {
		t.Fatalf("RunCtx() error = %q, want reserved-domain guidance", err)
	}
}
