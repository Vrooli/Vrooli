package programsvalidation

import (
	"path/filepath"
	"testing"

	"program-runtime/internal/bindings"
	"program-runtime/internal/contracts"
)

func TestTransitiveDependencyFindingsFollowDeclaredGrandchildContracts(t *testing.T) {
	root := contracts.Contract{ID: "parent.root", Scenario: "parent", Name: "root", Source: `print(lib.child.middle())`}
	child := contracts.Contract{ID: "child.middle", Scenario: "child", Name: "middle", Source: `print(lib.grandchild.leaf())`, Bindings: []contracts.BindingRef{{ID: "grandchild/ops/read", Effect: "read"}}}
	grandchild := contracts.Contract{ID: "grandchild.leaf", Scenario: "grandchild", Name: "leaf", Source: `print({"status": "ok"})`}
	all := []contracts.Contract{root, child, grandchild}
	details := []any{}

	findings := transitiveDependencyFindings("parent", "root", root, all, nil, map[string]struct{}{"child": {}}, &details)
	if len(findings) != 1 || findings[0] != "programs.dependency_undeclared" {
		t.Fatalf("findings = %v, want one missing transitive dependency", findings)
	}
	if len(details) < 1 {
		t.Fatalf("details = %v, want a transitive dependency diagnostic", details)
	}
	for _, value := range details {
		detail := value.(map[string]any)
		if detail["target_scenario"] != "grandchild" || detail["kind"] != "dependency" || detail["transitive"] != true {
			t.Fatalf("detail = %v, want grandchild target and transitive kind", detail)
		}
	}

	findings = transitiveDependencyFindings("parent", "root", root, all, nil, map[string]struct{}{"child": {}, "grandchild": {}}, nil)
	if len(findings) != 0 {
		t.Fatalf("findings = %v, want declared child and grandchild dependencies to pass", findings)
	}
}

func TestTransitiveDependencyFindingsTolerateLibraryCycles(t *testing.T) {
	root := contracts.Contract{ID: "alpha.root", Scenario: "alpha", Name: "root", Source: `print(lib.beta.middle())`}
	child := contracts.Contract{ID: "beta.middle", Scenario: "beta", Name: "middle", Source: `print(lib.alpha.root())`}
	findings := transitiveDependencyFindings("alpha", "root", root, []contracts.Contract{root, child}, nil, map[string]struct{}{"beta": {}}, nil)
	if len(findings) != 0 {
		t.Fatalf("findings = %v, want cycle-safe declared dependency tree", findings)
	}
}

func TestTransitiveBindingEffectFindingsRejectChildEffectMismatch(t *testing.T) {
	repoRoot, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	registry, err := bindings.Load(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	governed, ok := registry.Binding("vrooli/scenario/list")
	if !ok {
		t.Fatal("fixture governed binding is unavailable")
	}
	declaredEffect := "read"
	if governed.GetEffect() == declaredEffect {
		declaredEffect = "write"
	}
	root := contracts.Contract{ID: "parent.root", Scenario: "parent", Name: "root", Source: `print(lib.child.middle())`}
	child := contracts.Contract{ID: "child.middle", Scenario: "child", Name: "middle", Bindings: []contracts.BindingRef{{ID: "vrooli/scenario/list", Effect: declaredEffect}}}
	details := []any{}
	findings := transitiveBindingEffectFindings("root", root, []contracts.Contract{root, child}, registry, &details)
	if len(findings) != 1 || findings[0] != "programs.binding_effect_mismatch" {
		t.Fatalf("findings = %v, want child effect mismatch", findings)
	}
	if len(details) != 1 || details[0].(map[string]any)["governed_effect"] != governed.GetEffect() {
		t.Fatalf("details = %v, want governed effect evidence", details)
	}
}
