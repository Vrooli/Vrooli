package rolepolicy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestAgentManagerAndAIGatewayRoleCatalogsAreDisjoint(t *testing.T) {
	revision, err := Load(ResolvePath())
	if err != nil {
		t.Fatalf("load agent-manager catalog: %v", err)
	}
	gatewayPath := filepath.Join(filepath.Dir(ResolvePath()), "..", "..", "ai-gateway", "config", "inference-role-catalog.json")
	data, err := os.ReadFile(gatewayPath)
	if err != nil {
		t.Fatalf("read ai-gateway catalog: %v", err)
	}
	var gateway struct {
		Roles map[string]json.RawMessage `json:"roles"`
	}
	if err := json.Unmarshal(data, &gateway); err != nil {
		t.Fatalf("decode ai-gateway catalog: %v", err)
	}
	if duplicates := catalogRoleIntersection(revision.Catalog().Roles, gateway.Roles); len(duplicates) != 0 {
		t.Fatalf("role catalogs overlap: %v", duplicates)
	}
}

func TestCatalogRoleIntersectionDetectsDuplicate(t *testing.T) {
	duplicates := catalogRoleIntersection(map[string]Role{"extract.structured": {}}, map[string]json.RawMessage{"extract.structured": json.RawMessage(`{}`)})
	if len(duplicates) != 1 || duplicates[0] != "extract.structured" {
		t.Fatalf("duplicates = %v", duplicates)
	}
}

func catalogRoleIntersection(agent map[string]Role, gateway map[string]json.RawMessage) []string {
	duplicates := make([]string, 0)
	for name := range agent {
		if _, exists := gateway[name]; exists {
			duplicates = append(duplicates, name)
		}
	}
	sort.Strings(duplicates)
	return duplicates
}
