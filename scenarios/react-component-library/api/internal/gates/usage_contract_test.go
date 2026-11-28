package gates

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUsageContractGateFailsClosedOnIncompleteSuccessfulReport(t *testing.T) {
	root := t.TempDir()
	script := filepath.Join(root, "scenarios", "react-component-library", "ui", "scripts", "check-catalog-usage-contracts.mjs")
	if err := os.MkdirAll(filepath.Dir(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(script, []byte("// fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	nodeDir := t.TempDir()
	node := filepath.Join(nodeDir, "node")
	if err := os.WriteFile(node, []byte("#!/bin/sh\nprintf '%s\\n' '{\"coverage\":{\"eligible\":2}}'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", nodeDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	result, err := ValidateUsageContracts(Scope{Context: context.Background(), Root: root})
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range result.Findings {
		if finding.Code == "catalog.usage_contract.coverage_incomplete" {
			return
		}
	}
	t.Fatalf("successful but incomplete checker report passed: %+v", result)
}

func TestUsageContractGateRejectsContractedCountAboveEligible(t *testing.T) {
	root := t.TempDir()
	script := filepath.Join(root, "scenarios", "react-component-library", "ui", "scripts", "check-catalog-usage-contracts.mjs")
	if err := os.MkdirAll(filepath.Dir(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(script, []byte("// fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	nodeDir := t.TempDir()
	node := filepath.Join(nodeDir, "node")
	if err := os.WriteFile(node, []byte("#!/bin/sh\nprintf '%s\\n' '{\"coverage\":{\"eligible\":2,\"contracted\":3},\"errors\":[]}'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", nodeDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	result, err := ValidateUsageContracts(Scope{Context: context.Background(), Root: root})
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range result.Findings {
		if strings.Contains(finding.Code, "coverage_incomplete") {
			return
		}
	}
	t.Fatalf("impossible coverage counts passed: %+v", result)
}
