package portcontract

import (
	"path/filepath"
	"testing"
)

func TestCollectionPagePortOracle(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	contract, err := Build(root, "templates.collection-page")
	if err != nil {
		t.Fatal(err)
	}
	if contract.ClosureCount < 2 {
		t.Fatalf("expected a non-trivial collection-page closure, got %d", contract.ClosureCount)
	}
	if contract.SelfContained || len(contract.UnmetPorts) == 0 {
		t.Fatalf("ports=%#v", contract.UnmetPorts)
	}
	var reducedMotion *Port
	for index := range contract.UnmetPorts {
		if contract.UnmetPorts[index].CapabilityID == "reduced-motion" {
			reducedMotion = &contract.UnmetPorts[index]
			break
		}
	}
	if reducedMotion == nil || len(reducedMotion.DemandingAssets) == 0 {
		t.Fatal("reduced-motion has no demanding assets")
	}
	found := false
	for _, node := range reducedMotion.CandidateSatisfiers {
		if node.ID == "foundations.ui-provider" {
			found = true
		}
	}
	if !found {
		t.Fatal("ui-provider should be a candidate satisfier")
	}
}
