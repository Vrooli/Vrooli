package ontology

import "testing"

const manifestFixture = `{
  "name": "tech-tree-designer",
  "groups": [
    {
      "name": "ontology",
      "description": "Ontology commands.",
      "commands": [
        { "name": "capabilities", "binding": { "kind": "connect-rpc", "service": "OntologyService", "method": "ListCapabilities" }, "governance": { "effect": "read", "run_eligible": true } },
        { "name": "capability", "positionals": [{ "name": "slug", "required": true }], "binding": { "kind": "connect-rpc", "service": "OntologyService", "method": "GetCapability" }, "governance": { "effect": "read", "run_eligible": true } },
        { "name": "capability-upsert", "positionals": [{ "name": "slug", "required": true }], "binding": { "kind": "connect-rpc", "service": "OntologyService", "method": "UpsertCapability" }, "governance": { "effect": "write", "run_eligible": false } },
        { "name": "capability-rm", "positionals": [{ "name": "slug", "required": true }], "binding": { "kind": "connect-rpc", "service": "OntologyService", "method": "DeleteCapability" }, "governance": { "effect": "destructive", "run_eligible": false } },
        { "name": "edge-add", "positionals": [{ "name": "from", "required": true }, { "name": "to", "required": true }], "binding": { "kind": "connect-rpc", "service": "OntologyService", "method": "UpsertCapabilityEdge" }, "governance": { "effect": "write", "run_eligible": false } },
        { "name": "edge-rm", "positionals": [{ "name": "from", "required": true }, { "name": "to", "required": true }], "binding": { "kind": "connect-rpc", "service": "OntologyService", "method": "DeleteCapabilityEdge" }, "governance": { "effect": "destructive", "run_eligible": false } },
        { "name": "import", "binding": { "kind": "connect-rpc", "service": "OntologyService", "method": "ImportTopology" }, "governance": { "effect": "write", "run_eligible": false } },
        { "name": "fulfill", "positionals": [{ "name": "capability", "required": true }, { "name": "scenario", "required": true }], "binding": { "kind": "connect-rpc", "service": "OntologyService", "method": "LinkFulfillment" }, "governance": { "effect": "write", "run_eligible": false } },
        { "name": "unfulfill", "positionals": [{ "name": "capability", "required": true }, { "name": "scenario", "required": true }], "binding": { "kind": "connect-rpc", "service": "OntologyService", "method": "UnlinkFulfillment" }, "governance": { "effect": "destructive", "run_eligible": false } },
        { "name": "fulfillments", "binding": { "kind": "connect-rpc", "service": "OntologyService", "method": "ListFulfillments" }, "governance": { "effect": "read", "run_eligible": true } },
        { "name": "coverage", "binding": { "kind": "connect-rpc", "service": "OntologyService", "method": "GetCoverage" }, "governance": { "effect": "read", "run_eligible": true } },
        { "name": "focus", "binding": { "kind": "connect-rpc", "service": "OntologyService", "method": "ListFocus" }, "governance": { "effect": "read", "run_eligible": true } },
        { "name": "capability-scenarios", "positionals": [{ "name": "slug", "required": true }], "binding": { "kind": "connect-rpc", "service": "OntologyService", "method": "GetCapabilityScenarios" }, "governance": { "effect": "read", "run_eligible": true } },
        { "name": "scenario", "positionals": [{ "name": "slug", "required": true }], "binding": { "kind": "connect-rpc", "service": "OntologyService", "method": "GetScenarioCapabilities" }, "governance": { "effect": "read", "run_eligible": true } },
        { "name": "overlay", "binding": { "kind": "connect-rpc", "service": "OntologyService", "method": "DescribeOverlayGraph" }, "governance": { "effect": "read", "run_eligible": true } }
      ]
    }
  ]
}`

func TestRegisterBuildsOntologyGroup(t *testing.T) {
	group, err := Register(nil, []byte(manifestFixture))
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if group.Name != GroupName {
		t.Fatalf("group.Name = %q, want %q", group.Name, GroupName)
	}
	if len(group.Subcommands) != 15 {
		t.Fatalf("len(group.Subcommands) = %d, want 15", len(group.Subcommands))
	}
	if group.Subcommands[0].RunCtx == nil {
		t.Fatal("capabilities RunCtx = nil")
	}
}
