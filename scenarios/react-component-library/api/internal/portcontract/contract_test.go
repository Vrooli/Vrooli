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
	if contract.ClosureCount != 47 {
		t.Fatalf("closure=%d", contract.ClosureCount)
	}
	if len(contract.UnmetPorts) != 1 || contract.UnmetPorts[0].CapabilityID != "reduced-motion" {
		t.Fatalf("ports=%#v", contract.UnmetPorts)
	}
	if len(contract.UnmetPorts[0].DemandingAssets) != 7 {
		t.Fatalf("demanders=%d", len(contract.UnmetPorts[0].DemandingAssets))
	}
	found := false
	for _, node := range contract.UnmetPorts[0].CandidateSatisfiers {
		if node.ID == "foundations.ui-provider" {
			found = true
		}
	}
	if !found {
		t.Fatal("ui-provider should be a candidate satisfier")
	}
}
